import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";
import { isDeepStrictEqual } from "node:util";
import { QueryAppliedPlanRequest, QueryAppliedPlanResponse, QueryModuleVersionsResponse } from "cosmjs-types/cosmos/upgrade/v1beta1/query";
import { buildNodeGuideProfile, PRIVATE_NETWORK_TOOL_PATH } from "./node-guide-profile";
import { buildObserverRelease, type ObserverPublication } from "./observer-release-profile";
import { buildDevelopmentProfile, validateDevelopmentPublication, validateDevelopmentRelease, type DevelopmentPublication } from "./development-profile";

function parseObserverPublication(raw: Buffer): ObserverPublication {
  if (raw.length > 16 * 1024) throw new Error("Observer publication metadata exceeds 16 KiB");
  const text = raw.toString("utf8");
  const value = JSON.parse(text) as ObserverPublication;
  // This contract is flat and scalar. Validate that shape first, then scan
  // complete JSON string tokens followed by a colon, including escaped keys.
  // JSON.parse alone would silently replace earlier duplicate fields.
  buildObserverRelease(value);
  const seen = new Set<string>();
  for (const match of text.matchAll(/"(?:\\[\s\S]|[^"\\])*"/gu)) {
    if (!/^\s*:/u.test(text.slice(match.index! + match[0].length))) continue;
    const key = JSON.parse(match[0]) as string;
    if (seen.has(key)) throw new Error("Observer publication contains duplicate JSON keys");
    seen.add(key);
  }
  return value;
}

// An install recipe must point to bytes that actually exist at its source pin.
// The deployment workflow additionally requires a clean, merged checkout.
export function loadNodeGuideProfile(repositoryRoot: string) {
  const git = (...args: string[]) => execFileSync("git", args, {
    cwd: repositoryRoot, stdio: ["ignore", "pipe", "pipe"],
  });
  const sourceCommit = git("rev-parse", "HEAD").toString("utf8").trim();
  const helperPath = "scripts/local-node.py";
  let committedHelper: Buffer;
  try {
    committedHelper = git("show", `${sourceCommit}:${helperPath}`);
  } catch {
    throw new Error("Node guide requires scripts/local-node.py committed at HEAD; commit the reviewed helper before building the guide.");
  }
  if (!committedHelper.equals(readFileSync(resolve(repositoryRoot, helperPath)))) {
    throw new Error("Node guide helper differs from HEAD; commit the reviewed helper before building the guide.");
  }
  // The closed-beta lane describes this exact private-network tool.
  let committedPrivateNetworkTool: Buffer;
  try {
    committedPrivateNetworkTool = git("show", `${sourceCommit}:${PRIVATE_NETWORK_TOOL_PATH}`);
  } catch {
    throw new Error(`Node guide requires ${PRIVATE_NETWORK_TOOL_PATH} committed at HEAD; commit the reviewed tool before building the guide.`);
  }
  if (!committedPrivateNetworkTool.equals(readFileSync(resolve(repositoryRoot, PRIVATE_NETWORK_TOOL_PATH)))) {
    throw new Error("Node guide private-network tool differs from HEAD; commit the reviewed tool before building the guide.");
  }
  const publicationPath = "dashboard/observer-release.json";
  let observerPublication: ObserverPublication | undefined;
  const committedPublication = git("ls-tree", "--name-only", sourceCommit, "--", publicationPath).length > 0;
  const workingPublication = existsSync(resolve(repositoryRoot, publicationPath));
  if (committedPublication !== workingPublication) {
    throw new Error("Observer publication metadata must match its presence at HEAD; commit the reviewed publication change before building.");
  }
  if (committedPublication) {
    const committed = git("show", `${sourceCommit}:${publicationPath}`);
    if (!committed.equals(readFileSync(resolve(repositoryRoot, publicationPath)))) {
      throw new Error("Observer publication metadata differs from HEAD; commit the verified release pins before building.");
    }
    observerPublication = parseObserverPublication(committed);
  }
  // New participant instructions must name source bytes present at the pin too.
  for (const path of ["scripts/shared-claims.py", "docs/SHARED-DEVELOPMENT.md"]) {
    let committed: Buffer;
    try { committed = git("show", `${sourceCommit}:${path}`); }
    catch { throw new Error(`Development guide requires ${path} committed at HEAD`); }
    if (!committed.equals(readFileSync(resolve(repositoryRoot, path)))) throw new Error(`Development source ${path} differs from HEAD`);
  }
  const developmentPath = "dashboard/development-publication.json";
  const committedDevelopment = git("ls-tree", "--name-only", sourceCommit, "--", developmentPath).length > 0;
  if (committedDevelopment !== existsSync(resolve(repositoryRoot, developmentPath))) throw new Error("Development publication presence differs from HEAD");
  let developmentPublication: DevelopmentPublication | undefined;
  if (committedDevelopment) {
    const bytes = git("show", `${sourceCommit}:${developmentPath}`);
    if (bytes.length > 16384 || !bytes.equals(readFileSync(resolve(repositoryRoot, developmentPath)))) throw new Error("Development publication bytes differ from HEAD or exceed the bound");
    const text = bytes.toString("utf8");
    developmentPublication = JSON.parse(text) as DevelopmentPublication;
    validateDevelopmentPublication(developmentPublication);
    const seen = new Set<string>();
    for (const match of text.matchAll(/"(?:\\[\s\S]|[^"\\])*"/gu)) {
      if (!/^\s*:/u.test(text.slice(match.index! + match[0].length))) continue;
      const key = JSON.parse(match[0]) as string;
      if (seen.has(key)) throw new Error("Duplicate development publication key");
      seen.add(key);
    }
    const committedArtifact = (path: string, maximum: number): Buffer => {
      let committed: Buffer;
      try { committed = git("show", `${sourceCommit}:${path}`); }
      catch { throw new Error(`Development publication requires committed ${path}`); }
      if (!existsSync(resolve(repositoryRoot, path)) || committed.length > maximum || !committed.equals(readFileSync(resolve(repositoryRoot, path)))) throw new Error(`Development packet ${path.split("/").at(-1)} differs from HEAD or exceeds the bound`);
      return committed;
    };
    const packet = (name: string, maximum: number): Buffer => committedArtifact(`deploy/networks/zerone-dev-1/${name}`, maximum);
    const descriptor = packet("network.json", 65536);
    const genesis = packet("genesis.json", 16 * 1024 * 1024);
    if (createHash("sha256").update(descriptor).digest("hex") !== developmentPublication.descriptor_sha256 || createHash("sha256").update(genesis).digest("hex") !== developmentPublication.genesis_sha256) throw new Error("Development public packet hash mismatch");
    const network = JSON.parse(descriptor.toString("utf8"));
    const checkDescriptor = (value: typeof network, publication: DevelopmentPublication, version?: number): void => {
      if (!value || value.schema !== "zerone-shared-development/v1" || value.chain_id !== publication.chain_id || value.source_commit !== publication.runtime_source_commit || value.runtime_binary_sha256 !== publication.binary_sha256 || value.genesis_sha256 !== publication.genesis_sha256 || value.rpc_genesis_sha256 !== publication.rpc_genesis_sha256 || value.rpc_url !== publication.gateway || value.genesis_url !== `${publication.gateway}/genesis.json` || value.faucet_url !== `${publication.gateway}/faucet` || value.local_test !== false || value.bootstrap_consensus !== "single-operator" || value.reset_policy !== "new-chain-id" || ![10, 11].includes(value.knowledge_version) || (version !== undefined && value.knowledge_version !== version) || value.commitment_scheme !== 2 || value.review_policy_version !== 1 || value.denom !== "uzrn" || value.gas_limit !== 2000000 || value.tx_fee_uzrn !== "2000000") throw new Error("Development descriptor publication mismatch");
    };
    checkDescriptor(network, developmentPublication);
    if (JSON.parse(genesis.toString("utf8")).chain_id !== developmentPublication.chain_id) throw new Error("Development genesis chain mismatch");
    const release = JSON.parse(packet("release.json", 65536).toString("utf8"));
    validateDevelopmentRelease(release, developmentPublication);
    const applied = buildDevelopmentProfile(sourceCommit, developmentPublication).upgradeApplied;
    if (applied) {
      checkDescriptor(network, developmentPublication, applied.knowledge_version);
      if (createHash("sha256").update(packet("participant-package.py", 128 * 1024)).digest("hex") !== release.packaging_recipe_sha256) throw new Error("Participant packaging recipe hash mismatch");
      const predecessorBytes = packet("network-knowledge-10.json", 65536);
      const predecessor = JSON.parse(predecessorBytes.toString("utf8"));
      const predecessorPublication: DevelopmentPublication = {
        ...developmentPublication,
        runtime_source_commit: applied.predecessor_source_commit,
        descriptor_sha256: createHash("sha256").update(predecessorBytes).digest("hex"),
        binary_sha256: predecessor?.runtime_binary_sha256,
      };
      // The retained release has its own source/binary/descriptor, but the same
      // exact genesis and RPC serialization as the observed successor.
      checkDescriptor(predecessor, predecessorPublication, 10);
      validateDevelopmentRelease(JSON.parse(packet("release-knowledge-10.json", 65536).toString("utf8")), predecessorPublication, null);

      const reportPath = "docs/reports/development-publication-2026-09-23.json";
      const report = JSON.parse(committedArtifact(reportPath, 128 * 1024).toString("utf8"));
      const observations = report?.public_observations;
      const appliedReply = observations?.applied?.response?.result?.response;
      const modulesReply = observations?.["module-versions"]?.response?.result?.response;
      const queryBase = `${developmentPublication.gateway}/abci_query?path=%22%2Fcosmos.upgrade.v1beta1.Query%2F`;
      const appliedData = Buffer.from(QueryAppliedPlanRequest.encode({ name: applied.name }).finish()).toString("hex");
      if (report?.schema !== "zerone.development-publication-observations/v1" || report.chain_id !== developmentPublication.chain_id || report.observed_at !== applied.recorded_at || report.decoded_applied_plan?.name !== applied.name || String(report.decoded_applied_plan?.applied_height) !== applied.height || report.decoded_applied_plan?.queried_height !== applied.observed_height || observations?.applied?.observed_at !== applied.recorded_at || observations?.["module-versions"]?.observed_at !== applied.recorded_at || observations?.applied?.url !== `${queryBase}AppliedPlan%22&data=0x${appliedData}` || observations?.["module-versions"]?.url !== `${queryBase}ModuleVersions%22&data=0x` || appliedReply?.code !== 0 || modulesReply?.code !== 0 || appliedReply?.height !== applied.observed_height || modulesReply?.height !== applied.observed_height) throw new Error("Development applied observation does not match the published upgrade");
      if (observations?.network?.url !== `${developmentPublication.gateway}/network.json` || observations.network.response_sha256 !== developmentPublication.descriptor_sha256 || !isDeepStrictEqual(observations.network.response, network) || report.genesis_verification?.genesis_sha256 !== developmentPublication.genesis_sha256 || report.genesis_verification?.rpc_genesis_sha256 !== developmentPublication.rpc_genesis_sha256 || report.genesis_verification?.unchanged_from_predecessor !== true) throw new Error("Development observation network pins or genesis continuity mismatch");
      const host = report.host_verification;
      if (!host || host.knowledge !== applied.knowledge_version || host.applied_height !== applied.height || host.runtime_binary !== developmentPublication.binary_sha256 || host.successor_descriptor !== developmentPublication.descriptor_sha256 || host.baseline_manifest_controls_identities_preserved !== true || host.signing_height_monotone !== true || !Number.isSafeInteger(host.height) || host.height <= 0 || BigInt(host.height) < BigInt(applied.observed_height) || typeof host.observed_at !== "string" || !/^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ$/u.test(host.observed_at) || new Date(host.observed_at).toISOString() !== host.observed_at.replace("Z", ".000Z") || host.observed_at < applied.recorded_at) throw new Error("Development host observation does not support the verified participant release");
      const responseBytes = (value: unknown): Buffer => {
        if (typeof value !== "string" || value.length > 65536 || !/^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$/u.test(value)) throw new Error("Invalid development observation response bytes");
        const bytes = Buffer.from(value, "base64");
        if (!bytes.length || bytes.toString("base64") !== value) throw new Error("Invalid development observation response bytes");
        return bytes;
      };
      const decodedApplied = QueryAppliedPlanResponse.decode(responseBytes(appliedReply.value));
      const decodedModules = QueryModuleVersionsResponse.decode(responseBytes(modulesReply.value)).moduleVersions;
      const knowledge = decodedModules.filter(({ name }) => name === "knowledge");
      if (decodedApplied.height.toString() !== applied.height || decodedModules.length > 100 || new Set(decodedModules.map(({ name }) => name)).size !== decodedModules.length || knowledge.length !== 1 || knowledge[0]!.version !== BigInt(applied.knowledge_version) || !Array.isArray(report.decoded_module_versions) || !isDeepStrictEqual(report.decoded_module_versions.map((item: { name: unknown; version: unknown }) => ({ name: item?.name, version: String(item?.version) })), decodedModules.map(({ name, version }) => ({ name, version: version.toString() })))) throw new Error("Development observation decoded upgrade or module versions mismatch");
    }
  }
  return buildNodeGuideProfile({
    sourceCommit,
    helperSha256: createHash("sha256").update(committedHelper).digest("hex"),
    privateNetworkToolSha256: createHash("sha256").update(committedPrivateNetworkTool).digest("hex"),
    observerPublication,
    developmentPublication,
  });
}

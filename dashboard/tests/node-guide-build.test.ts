import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { mkdtempSync, mkdirSync, readFileSync, rmSync, unlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { it } from "node:test";
import { loadNodeGuideProfile } from "../node-guide-build";
import { nodeGuidePage } from "../node-guide-page";
import { buildNodeGuideProfile } from "../node-guide-profile";
import { buildDevelopmentProfile, type DevelopmentPublication } from "../development-profile";

function developmentSources(root: string): void {
  mkdirSync(join(root, "docs"), { recursive: true });
  writeFileSync(join(root, "scripts/shared-claims.py"), "# Shared client fixture; never executed\n");
  writeFileSync(join(root, "docs/SHARED-DEVELOPMENT.md"), "# Shared development fixture\n");
}

it("keeps public-network links on zerone.ai when the guide is hosted by an observer", () => {
  const profile = buildNodeGuideProfile({ sourceCommit: "a1".repeat(20), helperSha256: "b2".repeat(32) });
  const html = nodeGuidePage(profile);
  const observerLocation = "https://observer.example.org/nodes/";
  const hrefs = [...html.matchAll(/href="([^"]+)"/gu)].map((match) => match[1]!);
  for (const fragment of ["wallet", "participate", "activity"]) {
    const matches = hrefs.filter((href) => new URL(href, observerLocation).hash === `#${fragment}`);
    assert.equal(matches.length, 1, `missing or duplicate public ${fragment} link`);
    assert.equal(new URL(matches[0]!, observerLocation).href, `https://zerone.ai/#${fragment}`);
  }
  assert.ok(hrefs.includes("#local"), "local setup remains on the current guide");
  assert.ok(hrefs.includes("/"), "site navigation remains on the current host");
  for (const read of profile.live.reads) {
    assert.ok(hrefs.includes(read.url));
    assert.equal(new URL(read.url, observerLocation).origin, "https://zerone.ai");
  }
});

it("publishes committed helper bytes and refuses missing or modified install instructions", () => {
  const root = mkdtempSync(join(tmpdir(), "zerone-node-guide-test-"));
  const git = (...args: string[]) => execFileSync("git", args, { cwd: root, stdio: ["ignore", "pipe", "pipe"] }).toString().trim();
  try {
    git("init", "--quiet");
    git("-c", "user.name=Guide test", "-c", "user.email=guide@example.invalid", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-m", "Fixture");
    assert.throws(() => loadNodeGuideProfile(root), /committed at HEAD/);
    mkdirSync(join(root, "scripts"));
    const helper = "# A local test fixture, never a runnable installer.\n";
    writeFileSync(join(root, "scripts/local-node.py"), helper);
    developmentSources(root);
    git("add", ".");
    git("-c", "user.name=Guide test", "-c", "user.email=guide@example.invalid", "-c", "commit.gpgsign=false", "commit", "-m", "Helper fixture");
    const profile = loadNodeGuideProfile(root);
    assert.equal(profile.source.commit, git("rev-parse", "HEAD"));
    assert.equal(profile.source.localNodeHelper.sha256, createHash("sha256").update(helper).digest("hex"));

    const html = nodeGuidePage(profile);
    const unescape = (value: string) => value.replace(/&amp;/gu, "&").replace(/&lt;/gu, "<").replace(/&gt;/gu, ">").replace(/&quot;/gu, '"').replace(/&#39;/gu, "'");
    const commands = [...html.matchAll(/<code id="command-[^"]+">([\s\S]*?)<\/code>/gu)].map((match) => unescape(match[1]!));
    for (const step of profile.local.steps) assert.ok(commands.includes(step.command), `static HTML omits ${step.id}`);
    assert.ok(html.includes(profile.source.commit));
    assert.ok(html.includes(profile.source.localNodeHelper.sha256));
    assert.match(html, /href="\/nodes\/guide\.json"/u);
    assert.doesNotMatch(html, /wallet-connect|<form\b|\/src\/main\.ts|\sonclick=/u);
    writeFileSync(join(root, "scripts/local-node.py"), `${helper}# Uncommitted drift.\n`);
    assert.throws(() => loadNodeGuideProfile(root), /differs from HEAD/);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

it("requires an exact committed and bounded observer publication without duplicate or missing fields", () => {
  const root = mkdtempSync(join(tmpdir(), "zerone-observer-publication-test-"));
  const git = (...args: string[]) => execFileSync("git", args, { cwd: root, stdio: ["ignore", "pipe", "pipe"] }).toString().trim();
  const commit = () => {
    git("add", ".");
    git("-c", "user.name=Guide test", "-c", "user.email=guide@example.invalid", "-c", "commit.gpgsign=false", "commit", "-m", "Public fixture");
  };
  const publication = {
    schema: "zerone.observer-publication/v1", releaseId: "zerone-1-observer-test",
    toolingCommit: "a1".repeat(20), executableBaseCommit: "b2".repeat(20), executableSha256: "c3".repeat(32),
    manifestSha256: "d4".repeat(32), archiveSha256: "e5".repeat(32), receiptSha256: "f6".repeat(32),
    expiresAt: "2026-09-16T19:19:13Z", checkpointHeight: 1262000, checkpointHash: "A7".repeat(32),
  };
  const path = join(root, "dashboard/observer-release.json");
  const raw = JSON.stringify(publication, null, 2) + "\n";
  try {
    git("init", "--quiet");
    mkdirSync(join(root, "scripts"));
    mkdirSync(join(root, "dashboard"));
    writeFileSync(join(root, "scripts/local-node.py"), "# Fixture; never executed\n");
    developmentSources(root);
    commit();
    assert.equal(loadNodeGuideProfile(root).live.replicaInstallation.availability, "not-published");
    writeFileSync(path, raw);
    assert.throws(() => loadNodeGuideProfile(root), /presence at HEAD/u);
    commit();
    assert.equal(loadNodeGuideProfile(root).live.replicaInstallation.release?.receiptSha256, publication.receiptSha256);
    writeFileSync(path, raw + " ");
    assert.throws(() => loadNodeGuideProfile(root), /differs from HEAD/u);
    unlinkSync(path);
    assert.throws(() => loadNodeGuideProfile(root), /presence at HEAD/u);
    // Both literal and decoded duplicate keys must fail despite valid final values.
    for (const extra of ['"releaseId":"older",', '"release\\u0049d":"older",']) {
      writeFileSync(path, raw.replace("{", `{${extra}`));
      commit();
      assert.throws(() => loadNodeGuideProfile(root), /duplicate JSON keys/u);
    }
    writeFileSync(path, "null\n");
    commit();
    assert.throws(() => loadNodeGuideProfile(root), /Invalid observer publication/u);
    writeFileSync(path, " ".repeat(16 * 1024) + raw);
    commit();
    assert.throws(() => loadNodeGuideProfile(root), /exceeds 16 KiB/u);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

it("binds optional development publication and client docs to committed bytes", () => {
  const root = mkdtempSync(join(tmpdir(), "zerone-development-publication-test-"));
  const git = (...args: string[]) => execFileSync("git", args, { cwd: root, stdio: ["ignore", "pipe", "pipe"] }).toString().trim();
  const commit = () => { git("add", "."); git("-c", "user.name=Guide test", "-c", "user.email=guide@example.invalid", "-c", "commit.gpgsign=false", "commit", "-m", "Fixture"); };
  const publication = { schema: "zerone.development-publication/v1", chain_id: "zerone-dev-1", gateway: "https://zerone-dev-1.fly.dev", descriptor_sha256: "12".repeat(32), genesis_sha256: "23".repeat(32), rpc_genesis_sha256: "34".repeat(32), runtime_source_commit: "ab".repeat(20), binary_sha256: "45".repeat(32), verified_at: "2026-09-11T22:00:00Z" };
  let raw = JSON.stringify(publication); const path = join(root, "dashboard/development-publication.json");
  try {
    git("init", "--quiet"); mkdirSync(join(root, "scripts")); mkdirSync(join(root, "dashboard"));
    writeFileSync(join(root, "scripts/local-node.py"), "# Fixture\n"); developmentSources(root); commit();
    assert.equal(loadNodeGuideProfile(root).development.availability, "verification-pending");
    writeFileSync(path, raw); assert.throws(() => loadNodeGuideProfile(root), /presence differs from HEAD/u); commit();
    assert.throws(() => loadNodeGuideProfile(root), /requires committed deploy\/networks/u);
    const directory = join(root, "deploy/networks/zerone-dev-1"); mkdirSync(directory, {recursive:true});
    const genesis = JSON.stringify({chain_id:"zerone-dev-1"});
    publication.genesis_sha256 = createHash("sha256").update(genesis).digest("hex");
    const descriptor = JSON.stringify({schema:"zerone-shared-development/v1",chain_id:"zerone-dev-1",source_commit:publication.runtime_source_commit,runtime_binary_sha256:publication.binary_sha256,genesis_sha256:publication.genesis_sha256,rpc_genesis_sha256:publication.rpc_genesis_sha256,rpc_url:publication.gateway,genesis_url:`${publication.gateway}/genesis.json`,faucet_url:`${publication.gateway}/faucet`,local_test:false,bootstrap_consensus:"single-operator",reset_policy:"new-chain-id",knowledge_version:10,commitment_scheme:2,review_policy_version:1,denom:"uzrn",gas_limit:2000000,tx_fee_uzrn:"2000000"});
    publication.descriptor_sha256 = createHash("sha256").update(descriptor).digest("hex");
    const tag = `zerone-dev-1-${publication.runtime_source_commit.slice(0,12)}`;
    const release = {schema:"zerone-development-release/v1",chain_id:"zerone-dev-1",source_commit:publication.runtime_source_commit,source_tree:"bc".repeat(20),release_tag:tag,release_url:`https://github.com/cambridgetcg/zerone-core/releases/tag/${tag}`,runtime_image:`registry.fly.io/zerone-dev-1@sha256:${"cd".repeat(32)}`,descriptor_sha256:publication.descriptor_sha256,genesis_sha256:publication.genesis_sha256,rpc_genesis_sha256:publication.rpc_genesis_sha256,runtime_binary_sha256:publication.binary_sha256,artifacts:["linux-amd64","darwin-arm64"].map(platform=>({name:`${tag}-${platform}.tar.gz`,platform,url:`https://github.com/cambridgetcg/zerone-core/releases/download/${tag}/${tag}-${platform}.tar.gz`,sha256:"ef".repeat(32),bytes:1024,binary_sha256:publication.binary_sha256})),bootstrap_consensus:"single-operator",funds:"valueless-development-only",private_keys_in_packages:false};
    writeFileSync(join(directory,"genesis.json"),genesis); writeFileSync(join(directory,"network.json"),descriptor); writeFileSync(join(directory,"release.json"),JSON.stringify(release));
    raw = JSON.stringify(publication); writeFileSync(path,raw); commit();
    assert.equal(loadNodeGuideProfile(root).development.publication?.descriptor_sha256, publication.descriptor_sha256);
    writeFileSync(join(directory,"genesis.json"),genesis+"\n"); assert.throws(()=>loadNodeGuideProfile(root),/packet genesis.json differs/u);
    commit(); assert.throws(()=>loadNodeGuideProfile(root),/public packet hash mismatch/u);
    writeFileSync(join(directory,"genesis.json"),genesis); commit();
    writeFileSync(path, `${raw}\n`); assert.throws(() => loadNodeGuideProfile(root), /bytes differ/u);
    for (const key of ["chain_id", "chain\\u005fid"]) { writeFileSync(path, raw.replace("{", `{"${key}":"zerone-1",`)); commit(); assert.throws(() => loadNodeGuideProfile(root), /Duplicate development publication key/u); }
    writeFileSync(path, raw); commit();
    writeFileSync(join(root, "docs/SHARED-DEVELOPMENT.md"), "Uncommitted drift"); assert.throws(() => loadNodeGuideProfile(root), /Development source .* differs from HEAD/u);
  } finally { rmSync(root, { recursive: true, force: true }); }
});

it("binds applied publication to committed replay artifacts and the recorded endpoint evidence", () => {
  const root = mkdtempSync(join(tmpdir(), "zerone-applied-publication-test-"));
  const git = (...args: string[]) => execFileSync("git", args, { cwd: root, stdio: ["ignore", "pipe", "pipe"] }).toString().trim();
  const commit = () => { git("add", "."); git("-c", "user.name=Guide test", "-c", "user.email=guide@example.invalid", "-c", "commit.gpgsign=false", "commit", "-m", "Applied publication fixture"); };
  const directory = "deploy/networks/zerone-dev-1";
  const reportPath = "docs/reports/development-publication-2026-09-23.json";
  const fixturePaths = ["dashboard/development-publication.json", `${directory}/network.json`, `${directory}/genesis.json`, `${directory}/network-knowledge-10.json`, `${directory}/release-knowledge-10.json`, `${directory}/participant-package.py`, reportPath];
  const fixtures = new Map(fixturePaths.map((path) => [path, readFileSync(new URL(`../../${path}`, import.meta.url))]));
  const publication = JSON.parse(fixtures.get("dashboard/development-publication.json")!.toString()) as DevelopmentPublication;
  const development = buildDevelopmentProfile("ab".repeat(20), publication);
  const packages = development.packages!;
  // Archive bytes are synthetic in this isolated Git fixture; runtime and
  // descriptor pins come from the committed-observation fixture being tested.
  const release = { schema: "zerone-development-participant-release/v1", chain_id: publication.chain_id, source_commit: publication.runtime_source_commit, source_tree: "bc".repeat(20), release_tag: packages.tag, release_url: packages.releaseUrl, runtime_image: `registry.fly.io/zerone-dev-1@sha256:${"cd".repeat(32)}`, descriptor_sha256: publication.descriptor_sha256, genesis_sha256: publication.genesis_sha256, rpc_genesis_sha256: publication.rpc_genesis_sha256, runtime_binary_sha256: publication.binary_sha256, artifacts: packages.artifacts.map(({name,platform,url}) => ({name,platform,url,sha256:"ef".repeat(32),bytes:1024,binary_sha256:publication.binary_sha256})), bootstrap_consensus:"single-operator", funds:"valueless-development-only", private_keys_in_packages:false, knowledge_version:11, packaging_recipe_sha256:createHash("sha256").update(fixtures.get(`${directory}/participant-package.py`)!).digest("hex"), upstream_release_sha256:development.upgradeApplied!.release_record_sha256, upgrade_packet_sha256:development.upgradeApplied!.packet_sha256, runtime_image_basis:"Upstream build-image provenance; no new hosted-machine observation.", publication_status:"applied-runtime-verified" };
  const report = JSON.parse(fixtures.get(reportPath)!.toString());
  const writeJSON = (path: string, value: unknown) => writeFileSync(join(root, path), JSON.stringify(value));
  try {
    git("init", "--quiet");
    for (const path of ["scripts", "dashboard", "docs/reports", directory]) mkdirSync(join(root, path), { recursive: true });
    writeFileSync(join(root, "scripts/local-node.py"), "# Fixture; never executed\n");
    developmentSources(root);
    for (const [path, bytes] of fixtures) writeFileSync(join(root, path), bytes);
    writeJSON(`${directory}/release.json`, release);
    commit();
    const profile = loadNodeGuideProfile(root).development;
    assert.equal(profile.upgradeApplied?.observed_height, report.decoded_applied_plan.queried_height);
    assert.equal(profile.packages?.tag, packages.tag);
    assert.match(profile.predecessorPackages!.recordUrl, /release-knowledge-10\.json$/u);

    for (const path of [reportPath, `${directory}/network-knowledge-10.json`, `${directory}/release-knowledge-10.json`, `${directory}/participant-package.py`]) {
      writeFileSync(join(root, path), Buffer.concat([fixtures.get(path)!, Buffer.from(" ")]));
      assert.throws(() => loadNodeGuideProfile(root), /differs from HEAD/u);
      unlinkSync(join(root, path));
      commit();
      assert.throws(() => loadNodeGuideProfile(root), /requires committed/u);
      writeFileSync(join(root, path), fixtures.get(path)!);
      commit();
    }
    writeFileSync(join(root, `${directory}/participant-package.py`), Buffer.concat([fixtures.get(`${directory}/participant-package.py`)!, Buffer.from("\n# changed recipe\n")]));
    commit();
    assert.throws(() => loadNodeGuideProfile(root), /packaging recipe hash mismatch/u);
    writeFileSync(join(root, `${directory}/participant-package.py`), fixtures.get(`${directory}/participant-package.py`)!);
    commit();

    const badReports = [
      { ...report, observed_at: "2026-09-22T10:01:17Z" },
      { ...report, chain_id: "zerone-1" },
      { ...report, decoded_applied_plan: { ...report.decoded_applied_plan, name: "other-upgrade" } },
      { ...report, decoded_applied_plan: { ...report.decoded_applied_plan, applied_height: 299999 } },
      { ...report, decoded_applied_plan: { ...report.decoded_applied_plan, queried_height: "925041" } },
      { ...report, decoded_module_versions: report.decoded_module_versions.map((item: {name: string; version: number}) => ({ ...item, version: item.name === "knowledge" ? 10 : item.version })) },
      { ...report, genesis_verification: { ...report.genesis_verification, unchanged_from_predecessor: false } },
      { ...report, host_verification: undefined },
    ];
    for (const change of [{ knowledge: 10 }, { applied_height: "299999" }, { runtime_binary: "ab".repeat(32) }, { successor_descriptor: "ab".repeat(32) }, { baseline_manifest_controls_identities_preserved: false }, { signing_height_monotone: false }, { height: 299999 }, { observed_at: "2026-09-22T10:11:31Z" }]) badReports.push({ ...report, host_verification: { ...report.host_verification, ...change } });
    for (const key of ["applied", "module-versions"]) {
      for (const change of [{ code: 1 }, { height: "925041" }, { value: "CAI=" }]) {
        const value = structuredClone(report);
        Object.assign(value.public_observations[key].response.result.response, change);
        badReports.push(value);
      }
      const value = structuredClone(report);
      value.public_observations[key].url = "https://unrelated.invalid/abci_query";
      badReports.push(value);
    }
    for (const change of [{ source_commit: "ab".repeat(20) }, { runtime_binary_sha256: "ab".repeat(32) }, { knowledge_version: 10 }]) {
      const value = structuredClone(report);
      Object.assign(value.public_observations.network.response, change);
      badReports.push(value);
    }
    const badDescriptorHash = structuredClone(report);
    badDescriptorHash.public_observations.network.response_sha256 = "ab".repeat(32);
    badReports.push(badDescriptorHash);
    for (const value of badReports) {
      writeJSON(reportPath, value); commit();
      assert.throws(() => loadNodeGuideProfile(root));
    }
    writeFileSync(join(root, reportPath), Buffer.concat([fixtures.get(reportPath)!, Buffer.alloc(128 * 1024, " ")]));
    commit();
    assert.throws(() => loadNodeGuideProfile(root), /exceeds the bound/u);
    writeFileSync(join(root, reportPath), fixtures.get(reportPath)!); commit();

    const predecessor = JSON.parse(fixtures.get(`${directory}/network-knowledge-10.json`)!.toString());
    const predecessorRelease = JSON.parse(fixtures.get(`${directory}/release-knowledge-10.json`)!.toString());
    for (const change of [{ knowledge_version: 11 }, { source_commit: publication.runtime_source_commit }, { genesis_sha256: "ab".repeat(32) }]) {
      const value = JSON.stringify({ ...predecessor, ...change });
      writeFileSync(join(root, `${directory}/network-knowledge-10.json`), value);
      writeJSON(`${directory}/release-knowledge-10.json`, { ...predecessorRelease, descriptor_sha256: createHash("sha256").update(value).digest("hex") });
      commit();
      assert.throws(() => loadNodeGuideProfile(root), /descriptor publication mismatch/u);
    }
    writeFileSync(join(root, `${directory}/network-knowledge-10.json`), fixtures.get(`${directory}/network-knowledge-10.json`)!);
    writeJSON(`${directory}/release-knowledge-10.json`, { ...predecessorRelease, runtime_binary_sha256: publication.binary_sha256 });
    commit();
    assert.throws(() => loadNodeGuideProfile(root), /runtime binary mismatch/u);
  } finally { rmSync(root, { recursive: true, force: true }); }
});

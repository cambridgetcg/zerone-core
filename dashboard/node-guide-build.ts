import { createHash } from "node:crypto";
import { execFileSync } from "node:child_process";
import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";
import { buildNodeGuideProfile } from "./node-guide-profile";
import { buildObserverRelease, type ObserverPublication } from "./observer-release-profile";
import { validateDevelopmentPublication, validateDevelopmentRelease, type DevelopmentPublication } from "./development-profile";

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
    const packet = (name: string, maximum: number): Buffer => {
      const path = `deploy/networks/zerone-dev-1/${name}`;
      let committed: Buffer;
      try { committed = git("show", `${sourceCommit}:${path}`); }
      catch { throw new Error(`Development publication requires committed ${path}`); }
      if (committed.length > maximum || !committed.equals(readFileSync(resolve(repositoryRoot, path)))) throw new Error(`Development packet ${name} differs from HEAD or exceeds the bound`);
      return committed;
    };
    const descriptor = packet("network.json", 65536);
    const genesis = packet("genesis.json", 16 * 1024 * 1024);
    if (createHash("sha256").update(descriptor).digest("hex") !== developmentPublication.descriptor_sha256 || createHash("sha256").update(genesis).digest("hex") !== developmentPublication.genesis_sha256) throw new Error("Development public packet hash mismatch");
    const network = JSON.parse(descriptor.toString("utf8"));
    if (network.schema !== "zerone-shared-development/v1" || network.chain_id !== developmentPublication.chain_id || network.source_commit !== developmentPublication.runtime_source_commit || network.runtime_binary_sha256 !== developmentPublication.binary_sha256 || network.genesis_sha256 !== developmentPublication.genesis_sha256 || network.rpc_genesis_sha256 !== developmentPublication.rpc_genesis_sha256 || network.rpc_url !== developmentPublication.gateway || network.local_test !== false) throw new Error("Development descriptor publication mismatch");
    if (JSON.parse(genesis.toString("utf8")).chain_id !== developmentPublication.chain_id) throw new Error("Development genesis chain mismatch");
    validateDevelopmentRelease(JSON.parse(packet("release.json", 65536).toString("utf8")), developmentPublication);
  }
  return buildNodeGuideProfile({
    sourceCommit,
    helperSha256: createHash("sha256").update(committedHelper).digest("hex"),
    observerPublication,
    developmentPublication,
  });
}

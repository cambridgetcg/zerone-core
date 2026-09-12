import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { mkdtempSync, mkdirSync, rmSync, unlinkSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { it } from "node:test";
import { loadNodeGuideProfile } from "../node-guide-build";
import { nodeGuidePage } from "../node-guide-page";
import { buildNodeGuideProfile } from "../node-guide-profile";

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
    const descriptor = JSON.stringify({schema:"zerone-shared-development/v1",chain_id:"zerone-dev-1",source_commit:publication.runtime_source_commit,runtime_binary_sha256:publication.binary_sha256,genesis_sha256:publication.genesis_sha256,rpc_genesis_sha256:publication.rpc_genesis_sha256,rpc_url:publication.gateway,local_test:false});
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

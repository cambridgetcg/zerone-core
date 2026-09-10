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
    git("add", "scripts/local-node.py");
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

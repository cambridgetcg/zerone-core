import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { it } from "node:test";
import { loadNodeGuideProfile } from "../node-guide-build";
import { nodeGuidePage } from "../node-guide-page";

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

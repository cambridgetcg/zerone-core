import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";

const root = new URL("../public/", import.meta.url);
const headers = readFileSync(new URL("_headers", root), "utf8");
const prefix = "/research/checkpoints/2026-09-14-demimetric-v1/packets/";
const documents = [
  { path: "v0.1/REPORT", sha: "33162b47ff0e5d90de6b3ace393b72649ee1002b8d439f6d78208634de0c9762", attributes: 13 },
  { path: "followups/translation-v0.1/NOTE", sha: "96b1963d318799cef6a879240d2078c13072f4792f3c338cf8ae5065baf59a9d", attributes: 18 },
  { path: "followups/restricted-repair-v0.1/NOTE", sha: "4dab7883c423fc08ca02308cbb246b0f7f0a7e91477ef887b03370f06eed8717", attributes: 0 },
];
const hash = (value: string) => `'sha256-${createHash("sha256").update(value, "utf8").digest("base64")}'`;

describe("Frozen research reading-copy CSP", () => {
  it("preserves the global app policy and confines header replacement to six exact document routes", () => {
    const global = headers.split("\n\n")[0]!;
    assert.ok(global.includes("  Content-Security-Policy: default-src 'self'; script-src 'self' 'sha256-FdszTUgRXyXO9dcmwGS4VdY8An7lQXTPUIS+Sfar9Cc='; style-src 'self' 'sha256-fpcmZX/xppFTzqMnGjIVcv/CIhI79REG6tv04UkzKjc='; img-src 'self' data:; connect-src 'self' https://zerone-dev-1.fly.dev; font-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'; object-src 'none'"));
    assert.ok(!global.includes("unsafe-hashes") && !headers.includes("unsafe-inline"));
    const replacements = headers.split("\n\n").filter(rule => rule.includes("! Content-Security-Policy"));
    assert.equal(replacements.length, 6);
    assert.deepEqual(replacements.map(rule => rule.split("\n")[0]).sort(), documents.flatMap(d => [prefix + d.path, prefix + d.path + ".html"]).sort());
    assert.ok(headers.split("\n").every(line => line.length <= 2000));
  });

  it("allows exactly the frozen style blocks and attribute values while refusing scripts and connections", () => {
    for (const document of documents) {
      const raw = readFileSync(new URL(prefix.slice(1) + document.path + ".html", root));
      assert.equal(createHash("sha256").update(raw).digest("hex"), document.sha);
      const html = raw.toString("utf8");
      const blocks = [...html.matchAll(/<style\b[^>]*>([\s\S]*?)<\/style>/giu)].map(m => m[1]!);
      const attributes = [...html.matchAll(/\sstyle="([^"]*)"/giu)].map(m => m[1]!);
      assert.equal(blocks.length, 2); assert.equal(attributes.length, document.attributes);
      // These frozen attributes contain no entities or line-ending normalization.
      assert.ok(attributes.every(a => ["text-align: left", "text-align: right;", "width: 33%"].includes(a)));
      const blockHashes = new Set(blocks.map(hash)), attributeHashes = new Set(attributes.map(hash));
      for (const suffix of [".html", ""]) {
        const rule = headers.split("\n\n").find(r => r.startsWith(prefix + document.path + suffix + "\n"))!;
        assert.ok(rule.includes("  ! Content-Security-Policy\n  Content-Security-Policy: "));
        const policy = rule.split("  Content-Security-Policy: ")[1]!;
        const directives = new Map(policy.trim().split("; ").map(d => { const [name, ...values] = d.split(" "); return [name, values]; }));
        assert.deepEqual(new Set(directives.get("style-src-elem")), blockHashes);
        assert.deepEqual(new Set(directives.get("style-src-attr")), attributeHashes.size ? new Set(["'unsafe-hashes'", ...attributeHashes]) : new Set(["'none'"]));
        for (const directive of ["default-src", "script-src", "style-src", "connect-src", "font-src", "frame-ancestors", "base-uri", "form-action", "object-src"]) assert.deepEqual(directives.get(directive), ["'none'"], directive);
        assert.deepEqual(directives.get("img-src"), ["'self'", "data:"]);
      }
    }
  });
});

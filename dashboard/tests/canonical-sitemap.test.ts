import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";

// Search engines were indexing zerone-ai.pages.dev, which serves the same bytes as zerone.ai.
// Every public page names its own absolute zerone.ai URL, the one that answers 200 (directory pages
// keep their slash; Pages 308s .html to the extensionless URL), and sitemap.xml lists exactly those.
const read = (path: string) => readFileSync(new URL(`../${path}`, import.meta.url), "utf8");

// [source that writes the page's <head>, canonical URL]. node-guide-page.ts and understand-guide.ts take
// theirs from the profile's documentation.humanGuide, checked below.
const LITERAL_HEADS: Array<[string, string]> = [
  ["index.html", "https://zerone.ai/"],
  ["compositions.ts", "https://zerone.ai/compositions/"],
  ["development-page.ts", "https://zerone.ai/development/"],
  ["research/index.html", "https://zerone.ai/research/"],
  ["research/adagrad/index.html", "https://zerone.ai/research/adagrad/"],
  ["research/review/index.html", "https://zerone.ai/research/review/"],
  ["public/research/checkpoints/index.html", "https://zerone.ai/research/checkpoints/"],
];
const PROFILE_HEADS: Array<[string, string]> = [
  ["node-guide-profile.ts", "https://zerone.ai/nodes/"],
  ["understand-guide.ts", "https://zerone.ai/understand/"],
];
// Frozen reading copies: SHA256SUMS pins their bytes, so _headers carries their canonical.
const READING_PREFIX = "/research/checkpoints/2026-09-14-demimetric-v1/packets/";
const READING_COPIES = ["v0.1/REPORT", "followups/translation-v0.1/NOTE", "followups/restricted-repair-v0.1/NOTE"];

const canonicals = (html: string) =>
  [...html.matchAll(/<link\b[^>]*\brel="canonical"[^>]*>/gu)].map(([tag]) => tag.match(/\bhref="([^"]*)"/u)?.[1]);
const sitemap = [...read("public/sitemap.xml").matchAll(/<loc>([^<]+)<\/loc>/gu)].map(([, loc]) => loc!);

describe("canonical links point every page at its zerone.ai URL", () => {
  for (const [source, url] of LITERAL_HEADS) {
    it(`${source} names ${url} once`, () => {
      assert.deepEqual(canonicals(read(source)), [url]);
    });
  }

  for (const [source, url] of PROFILE_HEADS) {
    it(`${source} gives its page the humanGuide ${url}`, () => {
      assert.ok(read(source).includes(`humanGuide: "${url}"`), source);
    });
  }

  it("the frozen reading copies keep their bytes and name their canonical in _headers", () => {
    const headers = read("public/_headers").split("\n\n");
    for (const document of READING_COPIES) {
      const html = read(`public${READING_PREFIX}${document}.html`);
      assert.deepEqual(canonicals(html), [], `${document}.html stays byte-pinned`);
      for (const suffix of ["", ".html"]) {
        const rule = headers.find((r) => r.startsWith(`${READING_PREFIX}${document}${suffix}\n`));
        assert.ok(rule, `${document}${suffix} has its own rule`);
        assert.ok(rule.includes(`\n  Link: <https://zerone.ai${READING_PREFIX}${document}>; rel="canonical"\n`), document);
      }
    }
  });
});

describe("sitemap.xml", () => {
  it("lists every canonical page once, and only absolute zerone.ai URLs", () => {
    const expected = [
      ...LITERAL_HEADS.map(([, url]) => url),
      ...PROFILE_HEADS.map(([, url]) => url),
      ...READING_COPIES.map((document) => `https://zerone.ai${READING_PREFIX}${document}`),
    ];
    assert.deepEqual([...sitemap].sort(), [...expected].sort());
    assert.equal(new Set(sitemap).size, sitemap.length);
    assert.ok(sitemap.every((loc) => loc.startsWith("https://zerone.ai/") && !loc.endsWith(".html")));
  });

  it("is a real XML file the Quiet Doors route passes through, and robots.txt names it", () => {
    const xml = read("public/sitemap.xml");
    assert.ok(xml.startsWith('<?xml version="1.0" encoding="UTF-8"?>\n'));
    assert.ok(xml.includes('<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">'));
    assert.ok(read("public/robots.txt").endsWith("\nSitemap: https://zerone.ai/sitemap.xml\n"));
  });
});

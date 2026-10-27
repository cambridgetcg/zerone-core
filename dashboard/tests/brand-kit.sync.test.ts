import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";

// The site serves the brand kit as built, and every page head says what the kit says. Adapted from the
// artcraft-brand sync test (its references/wiring.md §6). When a check fails, copy the kit's web/ files
// again or rebuild the kit from brand/brand.json; never hand-edit a kit file.
const bytes = (path: string) => readFileSync(new URL(`../${path}`, import.meta.url));
const read = (path: string) => bytes(path).toString("utf8");

const KIT_WEB = "brand/kit/web";
const SERVED = [
  "favicon.ico",
  "favicon.svg",
  "apple-touch-icon.png",
  "icon-192.png",
  "icon-512.png",
  "icon-maskable-512.png",
  "site.webmanifest",
  "og.jpg",
];
// Every source that writes a served page's <head>. observer-page.ts serves only non-legacy profiles, and
// /pi/callback/ keeps img-src 'none' with no icon.
const HEAD_SOURCES = [
  "index.html",
  "node-guide-page.ts",
  "understand-guide.ts",
  "development-page.ts",
  "compositions.ts",
  "research/index.html",
  "research/review/index.html",
  "research/adagrad/index.html",
];
const ICON_LINKS = [
  '<link rel="icon" href="/favicon.ico" sizes="32x32" />',
  '<link rel="icon" href="/favicon.svg" type="image/svg+xml" />',
  '<link rel="apple-touch-icon" href="/apple-touch-icon.png" />',
  '<link rel="manifest" href="/site.webmanifest" />',
];
// The home page's share card and site name, as the kit's head snippet writes them.
const HOME_SOCIAL = ["og:site_name", "og:image", "og:image:type", "og:image:width", "og:image:height", "og:image:alt", "twitter:card"];

const kitHead = read(`${KIT_WEB}/head-snippet.html`);

function head(source: string): string {
  const html = read(source);
  assert.equal(html.split("<head>").length, 2, `${source} writes one <head>`);
  return html.slice(html.indexOf("<head>"), html.indexOf("</head>"));
}

function metas(html: string, name: string): string[] {
  const escaped = name.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
  const pattern = new RegExp(`<meta\\s+(?:name|property)="${escaped}"\\s+content="([^"]*)"`, "gu");
  return [...html.matchAll(pattern)].map((match) => match[1]!);
}

function jsonLd(html: string): unknown[] {
  return [...html.matchAll(/<script type="application\/ld\+json">([\s\S]*?)<\/script>/gu)].map((match) => JSON.parse(match[1]!));
}

describe("brand kit served from public/", () => {
  for (const name of SERVED) {
    it(`public/${name} matches the kit byte for byte`, () => {
      assert.ok(bytes(`public/${name}`).equals(bytes(`${KIT_WEB}/${name}`)));
    });
  }

  it("serves every icon the manifest names, in the site's theme colour", () => {
    const manifest = JSON.parse(read("public/site.webmanifest")) as { icons: { src: string }[]; theme_color: string };
    assert.ok(manifest.icons.length > 0);
    for (const icon of manifest.icons) assert.ok(SERVED.includes(icon.src.replace(/^\//u, "")), icon.src);
    assert.equal(manifest.theme_color.toLowerCase(), "#09090c");
  });
});

describe("page heads from the kit", () => {
  const [scheme] = metas(kitHead, "color-scheme");
  const [themeColour] = metas(kitHead, "theme-color");

  it("reads a one-scheme kit: dark, with one theme colour", () => {
    assert.equal(scheme, "dark");
    assert.deepEqual(metas(kitHead, "theme-color"), ["#09090C"]);
  });

  for (const source of HEAD_SOURCES) {
    it(`${source} links the kit's icons and declares the dark scheme once`, () => {
      const html = head(source);
      for (const link of ICON_LINKS) assert.equal(html.split(link).length, 2, `${source}: ${link}`);
      assert.equal(html.match(/rel="(?:shortcut )?icon"/gu)?.length, 2, `${source}: only the two kit icons`);
      assert.deepEqual(metas(html, "color-scheme"), [scheme], `${source}: one color-scheme`);
      // The site's own single theme-color, which is the kit's dark ground.
      assert.deepEqual(metas(html, "theme-color").map((value) => value.toUpperCase()), [themeColour], `${source}: one theme-color`);
    });
  }

  it("gives the home page the kit's share card and JSON-LD, and no other page the JSON-LD", () => {
    const home = head("index.html");
    for (const name of HOME_SOCIAL) assert.deepEqual(metas(home, name), metas(kitHead, name), name);
    assert.deepEqual(metas(home, "og:image"), ["https://zerone.ai/og.jpg"]);
    assert.deepEqual(jsonLd(home), jsonLd(read(`${KIT_WEB}/head-home.html`)));
    for (const source of HEAD_SOURCES.slice(1)) assert.ok(!read(source).includes("application/ld+json"), source);
  });
});

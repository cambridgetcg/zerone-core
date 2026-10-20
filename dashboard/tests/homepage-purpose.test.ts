import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";

const html = readFileSync(new URL("../index.html", import.meta.url), "utf8");
const main = readFileSync(new URL("../src/main.ts", import.meta.url), "utf8");

const mainStart = html.indexOf('<main id="main-content">');
const piStart = html.indexOf('id="contribute"');
const mainEnd = html.indexOf("</main>");
// The hidden Pi pilot keeps its own reviewed copy and links; everything a
// visitor can see sits between <main> and that section.
const visible = html.slice(mainStart, html.lastIndexOf("<section", piStart));
const footer = html.slice(html.indexOf('<footer class="footer">'), html.indexOf("</footer>"));

function sectionById(id: string): string {
  const start = html.lastIndexOf("<section", html.indexOf(`id="${id}"`));
  let depth = 0;
  const pattern = /<section\b|<\/section>/gu;
  pattern.lastIndex = start;
  for (let match = pattern.exec(html); match; match = pattern.exec(html)) {
    depth += match[0] === "</section>" ? -1 : 1;
    if (depth === 0) return html.slice(start, match.index + match[0].length);
  }
  throw new Error(`unterminated section #${id}`);
}

describe("purpose-first homepage", () => {
  it("says why Zerone exists before any live numbers, tools or wallet", () => {
    const purpose = html.indexOf('id="purpose"');
    assert.ok(mainStart >= 0 && purpose > mainStart);
    for (const later of ['id="supply-value"', 'id="honest-state"', 'id="onboarding"', 'id="wallet"', 'id="look-closer"']) {
      assert.ok(purpose < html.indexOf(later), `${later} must follow the purpose`);
    }
    const section = sectionById("purpose");
    assert.match(section, /Why Zerone exists/u);
    assert.match(section, /How it works/u);
    assert.match(section, /does not prove the claim is true/u);
    assert.doesNotMatch(section, /<script\b|<button\b|<form\b|\son[a-z]+\s*=|\/api\//u);
  });

  it("states the purpose in the hero and the page metadata", () => {
    assert.match(html, /<h1 id="hero-title">Keep the reasoning,<br \/><em>not just the answer\.<\/em><\/h1>/u);
    assert.match(html, /<meta\s+name="description"\s+content="Zerone keeps a shared record of reasoning/u);
    assert.match(html, /<link rel="alternate" type="application\/json" href="\/understand\/guide\.json"/u);
    assert.match(html, /<link rel="alternate" type="text\/plain" href="\/llms\.txt"/u);
  });

  it("carries the trust model on the page instead of linking to a private repository", () => {
    const honest = html.slice(html.indexOf('id="honest-state"'), html.indexOf("</aside>"));
    assert.match(honest, /stop\s+the chain, roll it back, or restart it from scratch/u);
    assert.match(honest, /about 28 hours in July 2026/u);
    assert.match(honest, /about 54 hours in August 2026/u);
    assert.match(honest, /not an independent proof/u);
    assert.match(honest, /no date is promised/u);
    // Only in-page links: the trust model itself is here, not behind a link.
    assert.doesNotMatch(honest, /<a\b(?![^>]*href="#)/u);
  });

  it("links visitors only to places they can open", () => {
    for (const region of [visible, footer]) {
      assert.doesNotMatch(region, /github\.com|zerone\.money/u);
    }
    assert.ok(piStart > mainStart && piStart < mainEnd);
    // The folded views render their own links once opened; each must stay on
    // this site (an in-page fragment or a same-origin endpoint constant).
    for (const file of ["authority-geometry.ts", "knowledge-geometry.ts", "frontier-participation.ts"]) {
      const source = readFileSync(new URL(`../src/${file}`, import.meta.url), "utf8");
      for (const [, target] of source.matchAll(/\.href\s*=(?!=)\s*([^;]+);/gu)) {
        assert.match(target!.trim(), /^(?:"#[\w-]+"|[A-Z_]+_ENDPOINT)$/u, `${file}: ${target!.trim()}`);
      }
    }
  });

  it("offers the raw data behind the page through the read-only gateway", () => {
    const network = sectionById("network");
    for (const path of [
      "/api/rpc/block?height=1",
      "/api/rest/cosmos/bank/v1beta1/supply/by_denom?denom=uzrn",
      "/api/rpc/validators?page=1&amp;per_page=100",
      "/api/rest/cosmos/bank/v1beta1/balances/zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf/by_denom?denom=uzrn",
      "/api/rest/cosmos/bank/v1beta1/balances/zrn17h5scv3zu7xa8ep9kaqy47ae08h9x6c5fanwkh/by_denom?denom=uzrn",
      "/api/knowledge",
    ]) {
      assert.ok(network.includes(`href="${path}"`), path);
    }
    // Status responses carry the node's own listen address; the homepage does
    // not link them.
    assert.doesNotMatch(html, /href="\/api\/rpc\/(?:status|net_info)"/u);
  });

  it("folds each deeper view and loads it only when opened or deep-linked", () => {
    const folds: ReadonlyArray<readonly [string, string]> = [
      ["authority", "authority-geometry-root"],
      ["understanding", "knowledge-geometry-root"],
      ["participate", "frontier-participation-root"],
      ["liquidity", "pool-content"],
    ];
    for (const [view, root] of folds) {
      const section = sectionById(view);
      const details = section.indexOf(`<details class="closer-details" data-closer="${view}">`);
      assert.ok(details >= 0, `#${view} must be folded`);
      assert.ok(section.indexOf(`id="${root}"`) > details, `#${root} must sit inside its fold`);
      assert.match(html, new RegExp(`href="#${view}"`, "u"));
    }
    assert.match(main, /authority: \(\) => initialiseAuthorityGeometry\(authorityGeometryRoot\)/u);
    assert.match(main, /understanding: \(\) => initialiseKnowledgeGeometry\(knowledgeGeometryRoot\)/u);
    assert.match(main, /participate: \(\) => initialiseFrontierParticipation\(frontierParticipationRoot\)/u);
    assert.equal(main.match(/initialise(?:AuthorityGeometry|KnowledgeGeometry|FrontierParticipation)\(/gu)?.length, 3);
    assert.match(main, /details\.addEventListener\("toggle"/u);
    assert.match(main, /window\.addEventListener\("hashchange", \(\) => void openCloserForHash\(\)\)/u);
  });

  it("says what is closed on zerone-1, and points to the closed beta without offering it", () => {
    const honest = html.slice(html.indexOf('id="honest-state"'), html.indexOf("</aside>"));
    assert.match(honest, /no new accounts on zerone-1 and no way to join it as a validator/u);
    assert.match(honest, /Votes on changes to the chain are weighted by staked ZRN/u);
    assert.match(honest, /the last ran on 25 July 2026/u);
    const onboarding = sectionById("onboarding");
    assert.match(onboarding, /<a href="\/nodes\/#beta">closed beta<\/a>: for now, only\s+the household&rsquo;s own Macs, each invited, can run/u);
    assert.match(onboarding, /no safe way yet to create new zerone-1 accounts/u);
    assert.match(sectionById("more-pages"), /the closed beta on the household&rsquo;s own Macs/u);
  });

  it("labels genesis entries in the record map as written in, not reviewed", () => {
    const geometry = readFileSync(new URL("../src/knowledge-geometry.ts", import.meta.url), "utf8");
    assert.match(geometry, /"block 0 · written in when the chain began, not reviewed"/u);
    assert.doesNotMatch(geometry, /legacy metadata absent/u);
  });

  it("keeps the being-first line beside the reminder that claims need evidence", () => {
    assert.match(footer, /Every being is a truth: no one has to prove they belong\./u);
    assert.match(sectionById("purpose"), /A claim still needs its evidence\./u);
  });
});

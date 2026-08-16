import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";

import { CONSTRUCTIVE_TREE_SHA256 } from "../src/constructive-tree";
import { CORRESPONDENCE_GEOMETRY_SHA256 } from "../src/correspondence-geometry";
import { EXPLICIT_INVARIANT_DISCIPLINE_SHA256 } from "../src/explicit-invariant-discipline";

const html = readFileSync(new URL("../index.html", import.meta.url), "utf8");
const css = readFileSync(new URL("../src/styles.css", import.meta.url), "utf8");
const main = readFileSync(new URL("../src/main.ts", import.meta.url), "utf8");

const REVIEWED_ARTIFACTS = [
  {
    profile: "CG-0",
    sourceKind: "sealed-static-artifact",
    path: "../public/standards/correspondence-geometry.v0.json",
    reviewedSha256:
      "f8cfeebf7404ab7e2e86b80362471cdd64015a108c47e98147e80ba7bb9e9a90",
    runtimeSha256: CORRESPONDENCE_GEOMETRY_SHA256,
  },
  {
    profile: "EID-1",
    sourceKind: "sealed-static-artifact",
    path: "../public/standards/explicit-invariant-discipline.v1.json",
    reviewedSha256:
      "e60b89cbed8eb26d3fad0ee45ef8c433391341f3abb4865af2755595815354df",
    runtimeSha256: EXPLICIT_INVARIANT_DISCIPLINE_SHA256,
  },
  {
    profile: "zerone.constructive-intelligence-tree/v1",
    sourceKind: "sealed-static-artifact",
    path: "../public/standards/constructive-intelligence-tree.v1.json",
    reviewedSha256:
      "8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf",
    runtimeSha256: CONSTRUCTIVE_TREE_SHA256,
  },
] as const;

type ReadingStep = {
  href: string;
  profile: string;
  sourceKind: string;
  stage: string;
  step: string;
  text: string;
};

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/gu, "\\$&");
}

function elementById(source: string, tagName: string, id: string): string {
  const idIndex = source.indexOf(`id="${id}"`);
  assert.ok(idIndex >= 0, `missing #${id}`);
  const start = source.lastIndexOf(`<${tagName}`, idIndex);
  const close = `</${tagName}>`;
  const end = source.indexOf(close, idIndex);
  assert.ok(start >= 0 && end > idIndex, `missing complete <${tagName}>#${id}`);
  return source.slice(start, end + close.length);
}

function attribute(openingTag: string, name: string): string {
  const match = openingTag.match(
    new RegExp(
      `\\b${escapeRegExp(name)}\\s*=\\s*(?:"([^"]*)"|'([^']*)')`,
      "iu",
    ),
  );
  assert.ok(match, `missing ${name} on ${openingTag}`);
  return match[1] ?? match[2] ?? "";
}

function visibleText(fragment: string): string {
  return fragment
    .replace(/<!--[\s\S]*?-->/gu, " ")
    .replace(/<[^>]+>/gu, " ")
    .replace(/&amp;/gu, "&")
    .replace(/&lt;/gu, "<")
    .replace(/&gt;/gu, ">")
    .replace(/&quot;/gu, '"')
    .replace(/(?:&#39;|&apos;)/gu, "'")
    .replace(/&nbsp;/gu, " ")
    .replace(/\s+/gu, " ")
    .replace(/\s+([,.;:!?])/gu, "$1")
    .trim();
}

function countId(source: string, id: string): number {
  return Array.from(
    source.matchAll(
      new RegExp(`\\bid\\s*=\\s*(?:"${escapeRegExp(id)}"|'${escapeRegExp(id)}')`, "giu"),
    ),
  ).length;
}

function enclosedBlock(source: string, marker: string, fromIndex = 0): string {
  const markerIndex = source.indexOf(marker, fromIndex);
  assert.ok(markerIndex >= 0, `missing CSS marker ${marker}`);
  const openingBrace = source.indexOf("{", markerIndex);
  assert.ok(openingBrace >= 0, `missing CSS block for ${marker}`);
  let depth = 1;
  for (let index = openingBrace + 1; index < source.length; index += 1) {
    if (source[index] === "{") depth += 1;
    if (source[index] === "}") depth -= 1;
    if (depth === 0) return source.slice(openingBrace + 1, index);
  }
  throw new Error(`unterminated CSS block for ${marker}`);
}

function mediaBlockContaining(query: string, selector: string): string {
  let cursor = 0;
  const matches: string[] = [];
  while (cursor < css.length) {
    const index = css.indexOf(query, cursor);
    if (index < 0) break;
    const block = enclosedBlock(css, query, index);
    if (block.includes(selector)) matches.push(block);
    cursor = index + query.length;
  }
  assert.equal(
    matches.length,
    1,
    `expected one ${query} block containing ${selector}`,
  );
  return matches[0] as string;
}

function assertPassiveStaticRegion(fragment: string): void {
  assert.doesNotMatch(
    fragment,
    /<(?:script|link|style|img|picture|source|audio|video|track|iframe|embed|object)\b/iu,
  );
  assert.doesNotMatch(
    fragment,
    /<(?:button|form|input|select|textarea|details|summary|dialog)\b/iu,
  );
  assert.doesNotMatch(
    fragment,
    /\s(?:src|srcset|poster|action|formaction|ping|contenteditable|draggable|tabindex)\s*=/iu,
  );
  assert.doesNotMatch(fragment, /\son[a-z]+\s*=/iu);
  assert.doesNotMatch(
    fragment,
    /\s(?:aria-busy|aria-live|aria-atomic|autofocus|hidden)\s*=/iu,
  );
  assert.doesNotMatch(
    fragment,
    /\b(?:data-action|data-endpoint|data-fetch|data-url|data-wallet)\s*=/iu,
  );
  assert.doesNotMatch(
    fragment,
    /\b(?:loading|spinner|skeleton|wallet-connect|aria-current)\b/iu,
  );

  const hrefs = Array.from(
    fragment.matchAll(/\bhref\s*=\s*(?:"([^"]*)"|'([^']*)')/giu),
    (match) => match[1] ?? match[2] ?? "",
  );
  assert.equal(hrefs.length, 4);
  assert.ok(hrefs.every((href) => /^#[a-z][a-z0-9-]*$/u.test(href)));
}

const readingPath = elementById(html, "nav", "reading-path");
const readingPathStart = html.indexOf(readingPath);
const readingPathEnd = readingPathStart + readingPath.length;
const linkFragments = Array.from(
  readingPath.matchAll(
    /<a\b(?=[^>]*\bdata-reading-step\s*=)[^>]*>[\s\S]*?<\/a>/giu,
  ),
  (match) => match[0],
);
const stages = Array.from(
  readingPath.matchAll(/<li\b[^>]*\bdata-stage="([^"]+)"/gu),
  (match) => match[1] as string,
);
const steps: ReadingStep[] = linkFragments.map((fragment, index) => {
  const openingTag = fragment.slice(0, fragment.indexOf(">") + 1);
  return {
    href: attribute(openingTag, "href"),
    profile: attribute(openingTag, "data-profile"),
    sourceKind: attribute(openingTag, "data-source-kind"),
    stage: stages[index] ?? "",
    step: attribute(openingTag, "data-reading-step"),
    text: visibleText(fragment),
  };
});

describe("static dashboard reading path", () => {
  it("sits exactly between Honest state and Wallet without bypassing the disclosure", () => {
    const truthTitle = html.indexOf('id="truth-banner-title"');
    const truthStart = html.lastIndexOf("<aside", truthTitle);
    const truthEnd = html.indexOf("</aside>", truthTitle) + "</aside>".length;
    const walletId = html.indexOf('id="wallet"');
    const walletStart = html.lastIndexOf("<section", walletId);

    assert.ok(truthStart >= 0 && truthEnd > truthTitle);
    assert.ok(truthEnd < readingPathStart);
    assert.ok(readingPathEnd < walletStart);
    assert.match(html.slice(truthEnd, readingPathStart), /^\s*$/u);
    assert.match(html.slice(readingPathEnd, walletStart), /^\s*$/u);
    assert.equal(countId(html, "honest-state"), 1);
    assert.equal(countId(html, "reading-path"), 1);
    assert.equal(html.match(/href="#reading-path"/gu)?.length ?? 0, 0);
    assert.match(
      html,
      /<a class="button button-ghost" href="#honest-state">\s*Begin with honest state\s*<span aria-hidden="true">↓<\/span>\s*<\/a>/u,
    );
    assert.match(
      html,
      /<meta\s+name="description"\s+content="The live Zerone dashboard: hold your keys; inspect bounded knowledge; read proposed mappings and invariant constraints; trace a static curriculum\."\s*\/>/u,
    );
  });

  it("locks the ordered stages, destinations, profiles, source modes, and card copy", () => {
    assert.deepEqual(steps, [
      {
        step: "knowledge",
        stage: "observe",
        href: "#understanding",
        profile: "KG-0",
        sourceKind: "live-bounded-read",
        text:
          "01 · Observe Live · read-only Knowledge KG-0 Inspect the bounded response from zerone-1 and only the typed relations it returns. Completeness is not claimed. Establishes neither truth nor anyone’s understanding.",
      },
      {
        step: "correspondence",
        stage: "map",
        href: "#correspondence",
        profile: "CG-0",
        sourceKind: "sealed-static-artifact",
        text:
          "02 · Map Sealed · static Correspondence CG-0 Read a proposed analogy, translation, or projection with its preserved structure, losses, counterexamples, and non-transfers. CG-0 claims zero dualities or equivalences.",
      },
      {
        step: "explicit-invariants",
        stage: "bound",
        href: "#explicit-invariants",
        profile: "EID-1",
        sourceKind: "sealed-static-artifact",
        text:
          "03 · Bound Sealed · static Explicit invariants EID-1 Read a named candidate class and regime with explicit assumptions, witnesses, falsifiers, boundary terms, and a scoped result. A proposed local Zerone test—even if it passes—verifies only its named local invariant; it does not prove the cited physics.",
      },
      {
        step: "tree",
        stage: "trace",
        href: "#skills",
        profile: "zerone.constructive-intelligence-tree/v1",
        sourceKind: "sealed-static-artifact",
        text:
          "04 · Trace Versioned · static Constructive Tree v1 Follow the digest-pinned curriculum and its prerequisite graph. A visible path records no attainment. Grants no capability, qualification, authority, or reward.",
      },
    ]);
  });

  it("keeps the complete editorial and zero-effect account in static HTML", () => {
    assert.equal(
      visibleText(readingPath),
      "··· Reading path · editorial navigation only Carry the question. Keep every boundary. This optional route links four existing views. These anchors move only the reader: they do not pass a Fact or result between panels, create a shared workflow, import authority across sources, or write to zerone-1. 01 · Observe Live · read-only Knowledge KG-0 Inspect the bounded response from zerone-1 and only the typed relations it returns. Completeness is not claimed. Establishes neither truth nor anyone’s understanding. 02 · Map Sealed · static Correspondence CG-0 Read a proposed analogy, translation, or projection with its preserved structure, losses, counterexamples, and non-transfers. CG-0 claims zero dualities or equivalences. 03 · Bound Sealed · static Explicit invariants EID-1 Read a named candidate class and regime with explicit assumptions, witnesses, falsifiers, boundary terms, and a scoped result. A proposed local Zerone test—even if it passes—verifies only its named local invariant; it does not prove the cited physics. 04 · Trace Versioned · static Constructive Tree v1 Follow the digest-pinned curriculum and its prerequisite graph. A visible path records no attainment. Grants no capability, qualification, authority, or reward. Zero-effect boundary. Following this route changes no Fact, confidence, consensus, identity, person score, KARMA, qualification, reward, governance, money, consent, or chain state. It proves no physics or string theory and adjudicates no theology. Energy registers remain distinct: physical energy, compute, protocol accounting, economic value, lived experience, and spiritual language do not convert here. This route establishes no mind, understanding, consciousness, or personhood.",
    );
    assert.match(
      readingPath,
      /<nav[\s\S]*?aria-labelledby="reading-path-title"/u,
    );
    assert.doesNotMatch(readingPath, /aria-describedby=/u);
    assert.match(readingPath, /<ol class="reading-path-list" role="list">[\s\S]*<\/ol>/u);
  });

  it("resolves every labelled node and destination to one unique static id", () => {
    for (const id of [
      "reading-path",
      "reading-path-title",
      "reading-path-intro",
      "reading-path-boundary",
      "understanding",
      "correspondence",
      "explicit-invariants",
      "skills",
    ]) {
      assert.equal(countId(html, id), 1, `expected one #${id}`);
    }
    for (const { href } of steps) {
      const target = href.slice(1);
      assert.ok(html.indexOf(`id="${target}"`) > readingPathEnd, `${href} must resolve`);
    }
  });

  it("remains fully available without JavaScript and exposes no active surface", () => {
    const prefix = html.slice(0, readingPathStart);
    assert.equal(prefix.match(/<noscript\b/gu)?.length ?? 0, prefix.match(/<\/noscript>/gu)?.length ?? 0);
    assert.ok(readingPathEnd < html.indexOf('<script type="module" src="/src/main.ts"></script>'));
    assert.doesNotMatch(readingPath, /<noscript\b/iu);
    assert.doesNotMatch(readingPath, /\brole\s*=\s*["'](?:status|alert|progressbar)["']/iu);
    assert.doesNotThrow(() => assertPassiveStaticRegion(readingPath));

    for (const hostileAddition of [
      '<img src="https://attacker.example/pixel">',
      '<a href="https://attacker.example/leave">leave</a>',
      '<button type="button">mutate</button>',
      '<span aria-live="polite">dynamic</span>',
      '<span data-endpoint="/api/knowledge">load</span>',
      '<span onclick="fetch(`/api/knowledge`)">load</span>',
    ]) {
      assert.throws(() => assertPassiveStaticRegion(`${readingPath}${hostileAddition}`));
    }
  });

  it("has no feature initializer, fetch, or listener and re-aligns only its cold hash", () => {
    assert.doesNotMatch(
      main,
      /(?:initialise|initialize|fetch|load|render)[A-Za-z0-9_]*ReadingPath|readingPathReady/iu,
    );
    assert.equal(main.match(/"#reading-path"/gu)?.length, 1);
    assert.match(
      main,
      /const alignReadingPathHash = \(\): void => \{[\s\S]*window\.location\.hash !== "#reading-path"[\s\S]*getElementById\("reading-path"\)[\s\S]*scrollIntoView\(\{ block: "start", behavior: "instant" \}\);[\s\S]*\};\s*alignReadingPathHash\(\);\s*let initialHashInputsSettled/u,
    );
    assert.doesNotMatch(main, /#reading-path[\s\S]{0,160}(?:allSettled|\.then\()/iu);
  });

  it("pins the three sealed source-byte sets and leaves Knowledge live and unsealed", () => {
    assert.equal(steps[0]?.profile, "KG-0");
    assert.equal(steps[0]?.sourceKind, "live-bounded-read");
    assert.match(steps[0]?.text ?? "", /Live · read-only/u);
    assert.doesNotMatch(
      linkFragments[0] ?? "",
      /(?:sealed|digest-pinned|data-(?:sha|seal|digest))/iu,
    );

    for (const [index, artifact] of REVIEWED_ARTIFACTS.entries()) {
      const raw = readFileSync(new URL(artifact.path, import.meta.url));
      assert.equal(
        createHash("sha256").update(raw).digest("hex"),
        artifact.reviewedSha256,
        artifact.profile,
      );
      assert.equal(artifact.runtimeSha256, artifact.reviewedSha256);
      assert.equal(steps[index + 1]?.profile, artifact.profile);
      assert.equal(steps[index + 1]?.sourceKind, artifact.sourceKind);
    }
  });

  it("keeps the four-to-two-to-one layout, wrapping, focus, and reduced-motion hooks", () => {
    const baseList = enclosedBlock(css, ".reading-path-list {");
    const listItem = enclosedBlock(css, ".reading-path-list > li {");
    const link = enclosedBlock(css, ".reading-path-link {");
    const focus = enclosedBlock(css, ".reading-path-link:focus-visible {");
    const tablet = mediaBlockContaining(
      "@media (max-width: 1040px)",
      ".reading-path-list",
    );
    const narrow = mediaBlockContaining(
      "@media (max-width: 820px)",
      ".reading-path-list",
    );
    const mobile = mediaBlockContaining(
      "@media (max-width: 560px)",
      ".reading-path-link",
    );
    const reducedMotion = enclosedBlock(
      css,
      "@media (prefers-reduced-motion: reduce)",
    );

    assert.match(baseList, /grid-template-columns:\s*repeat\(4,\s*minmax\(0,\s*1fr\)\)/u);
    assert.match(tablet, /\.reading-path-list\s*\{[^}]*repeat\(2,\s*minmax\(0,\s*1fr\)\)/su);
    assert.match(narrow, /\.reading-path-list,\s*\.reading-path-boundary\s*\{[^}]*grid-template-columns:\s*1fr/su);
    assert.match(listItem, /display:\s*flex/u);
    assert.match(listItem, /min-width:\s*0/u);
    assert.match(link, /flex:\s*1/u);
    assert.match(link, /height:\s*100%/u);
    assert.match(link, /min-width:\s*0/u);
    assert.match(link, /overflow-wrap:\s*anywhere/u);
    assert.match(focus, /z-index:\s*1/u);
    assert.match(css, /:focus-visible\s*\{[^}]*outline:\s*2px solid var\(--acid\)/su);
    assert.match(mobile, /\.reading-path-link\s*\{[^}]*min-height:\s*0/su);
    assert.match(reducedMotion, /animation-duration:\s*0\.01ms\s*!important/u);
    assert.match(reducedMotion, /transition-duration:\s*0\.01ms\s*!important/u);

    const featureCssStart = css.indexOf(".reading-path {");
    const featureCssEnd = css.indexOf("\n  .section-heading {", featureCssStart);
    assert.ok(featureCssStart >= 0 && featureCssEnd > featureCssStart);
    assert.doesNotMatch(css.slice(featureCssStart, featureCssEnd), /@import|url\s*\(/iu);
  });
});

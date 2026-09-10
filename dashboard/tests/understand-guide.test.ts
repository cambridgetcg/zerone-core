import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import { buildNodeGuideProfile } from "../node-guide-profile";
import type { ObserverPublication } from "../observer-release-profile";
import { buildUnderstandGuide, understandPage } from "../understand-guide";

const sourceCommit = "a1".repeat(20);
const helperSha256 = "b2".repeat(32);
const publication = JSON.parse(readFileSync(new URL("../observer-release.json", import.meta.url), "utf8")) as ObserverPublication;
const build = (observerPublication?: ObserverPublication) => {
  const node = buildNodeGuideProfile({ sourceCommit, helperSha256, observerPublication });
  return { node, guide: buildUnderstandGuide(node) };
};

describe("shared explanatory guide", () => {
  it("keeps source identity, fictional work and effects explicit for agents and readers", () => {
    const { guide } = build();
    assert.equal(guide.source.commit, sourceCommit);
    assert.equal(guide.documentation.agentProtocolEndpoint, false);
    assert.equal(guide.documentation.documentationOnly, true);
    assert.ok(Object.values(guide.effects).every((effect) => effect === false));
    const example = guide.sections[1];
    assert.equal(example.illustrative, true);
    for (const field of ["actualTransaction", "actualReview", "actualReward", "actualReuse"] as const) {
      assert.equal(example[field], false);
    }
    const html = understandPage(guide);
    assert.match(html, /Illustrative example/u);
    assert.doesNotMatch(html, /<(?:script|form|input|textarea|iframe)\b|\son[a-z]+\s*=/iu);
    assert.ok(html.includes(sourceCommit));
    assert.throws(() => buildUnderstandGuide({ ...build().node, source: { ...build().node.source, commit: "main" } }));
  });

  it("uses the node guide's closed and open paths without inventing an observer release", () => {
    const { node, guide } = build();
    const today = guide.sections[5];
    assert.equal(today.rows[0].availability, node.live.availability);
    assert.equal(today.rows[1].availability, node.live.replicaInstallation.availability);
    assert.equal(today.rows[2].availability, node.local.availability);
    assert.equal(today.rows[3].availability, node.live.newAccountAdmission.availability);
    assert.equal(today.freshNetworkCheck, false);
    assert.equal(today.availabilityKind, "build-time-declaration");
    assert.doesNotMatch(understandPage(guide), /href="https:\/\/zerone\.ai\/nodes\/#observer"/u);
  });

  it("carries a published bootstrap deadline without turning publication into a fresh readiness check", () => {
    const expired = { ...publication, expiresAt: "2020-01-01T00:00:00Z" };
    const { node, guide } = build(expired);
    const today = guide.sections[5];
    assert.equal(today.rows[1].availability, node.live.replicaInstallation.availability);
    assert.equal(today.freshNetworkCheck, false);
    const html = understandPage(guide);
    assert.ok(html.includes(expired.expiresAt));
    assert.ok(html.includes("https://zerone.ai/nodes/#observer"));
    assert.match(html, /not a live health check/u);
    assert.equal(node.live.replicaInstallation.release?.validatorPower, 0);
  });

  it("renders shared prose as inert text and gives every local link a unique target", () => {
    const { guide } = build(publication);
    const prose = '<img src=x onerror="alert(1)"> & a correction';
    const html = understandPage({ ...guide, intro: { ...guide.intro, lede: prose as typeof guide.intro.lede } });
    assert.ok(html.includes("&lt;img src=x onerror=&quot;alert(1)&quot;&gt; &amp; a correction"));
    assert.ok(!html.includes(prose));
    const ids = [...html.matchAll(/\bid="([^"]+)"/gu)].map((match) => match[1]);
    assert.equal(new Set(ids).size, ids.length);
    for (const match of html.matchAll(/href="#([^"]+)"/gu)) assert.ok(ids.includes(match[1]), `Missing ${match[1]}`);
    const homepage = readFileSync(new URL("../index.html", import.meta.url), "utf8");
    for (const match of homepage.matchAll(/href="\/understand\/#([^"]+)"/gu)) assert.ok(ids.includes(match[1]), `Missing homepage destination ${match[1]}`);
  });

  it("preserves all research destinations through the shared library and their existing page anchors", () => {
    const { guide } = build();
    const homepage = readFileSync(new URL("../index.html", import.meta.url), "utf8");
    const html = understandPage(guide);
    for (const id of ["relations", "correspondence", "explicit-invariants", "skills", "life", "frontier-commons"]) {
      const url = `https://zerone.ai/#${id}`;
      assert.ok(guide.research.links.some((item) => item.url === url));
      assert.ok(html.includes(`href="${url}"`));
      assert.ok(homepage.includes(`id="${id}"`));
    }
  });
});

import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";

const html = readFileSync(new URL("../index.html", import.meta.url), "utf8");
const standardRaw = readFileSync(
  new URL(
    "../public/standards/frontier-commons-participation.v0.json",
    import.meta.url,
  ),
  "utf8",
);
const standard = JSON.parse(standardRaw) as {
  milestone: { id: string; title: string; state: string };
  participationModes: Array<{ id: string; state: string }>;
  reasoningLadder: Array<{ id: string }>;
  constituencies: Array<{ id: string }>;
  objectionRegister: Array<{ id: string }>;
  completionGates: Array<{ id: string; passed: boolean }>;
  nextMilestoneGates: Array<{ id: string; passed: boolean }>;
};
const digest = createHash("sha256").update(standardRaw).digest("hex");
const SECTION_SHA256 =
  "d2ee43c9b4a54b2a121836de849d58e0bf7829e52aaf48e3630ff837115942c9";
const start = html.indexOf('id="frontier-commons"');
const sectionStart = html.lastIndexOf("<section", start);
const sectionEnd = html.indexOf('<section\n          class="section split-section pi-section"', start);
assert.ok(start >= 0, "missing #frontier-commons");
assert.ok(sectionStart >= 0, "missing FC-0 section start");
assert.ok(sectionEnd > start, "missing FC-0 section end");
const section = html.slice(sectionStart, sectionEnd);

describe("Frontier Commons FC-0 page", () => {
  it("makes the read-only milestone visible in primary navigation", () => {
    assert.equal(html.match(/id="frontier-commons"/g)?.length, 1);
    assert.match(html, /<a href="#frontier-commons">Commons<\/a>/);
    assert.match(section, /The Reversible Hello/);
    assert.match(section, /An invitation, not enrollment/);
    assert.match(section, /FC-0 is set—and honestly not yet met/);
    assert.equal(standard.milestone.id, "FC-0");
    assert.equal(standard.milestone.state, "SET_NOT_MET");
  });

  it("renders the exact zero-effect boundary and no enrollment action", () => {
    for (const label of ["Membership", "Money", "Authority"]) {
      assert.match(
        section,
        new RegExp(`<span>${label}<\\/span><strong>0<\\/strong>`),
      );
    }
    assert.match(section, /<span>FC-0 checks passed<\/span><strong>0 \/ 4<\/strong>/);
    assert.doesNotMatch(section, /<form\b|<input\b|<button\b|wallet-connect/i);
    assert.doesNotMatch(section, />\s*join(?:\s|<)/i);
  });

  it("keeps progressive participation states honest", () => {
    assert.equal(section.match(/data-state="available"/g)?.length, 2);
    assert.equal(section.match(/data-state="review"/g)?.length, 2);
    assert.equal(section.match(/data-state="inactive"/g)?.length, 2);
    assert.deepEqual(
      standard.participationModes.map(({ id, state }) => ({ id, state })),
      [
        { id: "observe", state: "AVAILABLE_NOW" },
        { id: "verify-and-fork", state: "AVAILABLE_NOW" },
        {
          id: "public-source-contribution",
          state: "REQUIRES_SEPARATE_DUE_DILIGENCE",
        },
        {
          id: "reproduce-or-challenge",
          state: "STANDARD_ONLY_NO_INTAKE",
        },
        { id: "sponsor", state: "INACTIVE" },
        { id: "govern", state: "INACTIVE" },
      ],
    );
    assert.deepEqual(
      Array.from(
        section.matchAll(/data-mode-id="([^"]+)" data-state="([^"]+)"/g),
        (match) => ({ id: match[1], state: match[2] }),
      ),
      [
        { id: "observe", state: "available" },
        { id: "verify-and-fork", state: "available" },
        { id: "public-source-contribution", state: "review" },
        { id: "reproduce-or-challenge", state: "review" },
        { id: "sponsor", state: "inactive" },
        { id: "govern", state: "inactive" },
      ],
    );
  });

  it("renders all eight reasoning layers in canonical order", () => {
    const ids = Array.from(
      section.matchAll(/data-reasoning-id="([^"]+)"/g),
      (match) => match[1],
    );
    assert.deepEqual(ids, standard.reasoningLadder.map(({ id }) => id));
    assert.equal(ids.length, 8);
  });

  it("renders every non-ranking constituency lens and objection", () => {
    const constituencies = Array.from(
      section.matchAll(/data-constituency-id="([^"]+)"/g),
      (match) => match[1],
    );
    const objections = Array.from(
      section.matchAll(/data-objection-id="([^"]+)"/g),
      (match) => match[1],
    );
    assert.deepEqual(
      constituencies,
      standard.constituencies.map(({ id }) => id),
    );
    assert.deepEqual(
      objections,
      standard.objectionRegister.map(({ id }) => id),
    );
    assert.equal(constituencies.length, 15);
    assert.equal(objections.length, 11);
    assert.match(section, /Rights do not depend on role/);
    assert.match(section, /never rhetorically erased/);
  });

  it("publishes the exact raw standard and reviewed SHA-256", () => {
    assert.match(
      section,
      /href="\/standards\/frontier-commons-participation\.v0\.json"/,
    );
    assert.match(section, new RegExp(`sha256:${digest}`));
    assert.equal(
      createHash("sha256").update(section).digest("hex"),
      SECTION_SHA256,
    );
    assert.ok(standard.completionGates.every(({ passed }) => !passed));
    assert.equal(standard.completionGates.length, 4);
    assert.ok(standard.nextMilestoneGates.every(({ passed }) => !passed));
    assert.equal(standard.nextMilestoneGates.length, 9);
  });
});

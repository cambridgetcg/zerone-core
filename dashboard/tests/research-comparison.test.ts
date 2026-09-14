import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import { canonicalJSON, compareJournals, sha256, validateExport, selectSnapshot, type Journal } from "../src/research-review-journal";

const raw = readFileSync(new URL("../public/research/review/fictional-journal.v1.json", import.meta.url));
const fixture = JSON.parse(raw.toString("utf8")) as Journal;
const encode = (value: unknown) => new TextEncoder().encode(JSON.stringify(value));
async function changed(change: (j: Journal) => void): Promise<Journal> {
  const j = structuredClone(fixture); change(j); let previous = await sha256(canonicalJSON(j.header));
  for (const row of j.entries) {
    row.previous_sha256 = previous;
    const { sha256: _hash, ...unsigned } = row; void _hash;
    row.sha256 = await sha256(canonicalJSON(unsigned)); previous = row.sha256;
  }
  return validateExport(encode(j));
}

describe("Complete research journal comparison", () => {
  it("requires both complete validated immutable exports, including after an apparent shared prefix", async () => {
    const good = await validateExport(raw), forged = structuredClone(good);
    forged.entries.at(-1)!.record.summary += " altered";
    await assert.rejects(validateExport(encode(forged)), /SHA256 mismatch/u);
    for (const [left, right] of [[good, forged], [forged, good], [good, structuredClone(good)]]) await assert.rejects(compareJournals(left!, right!), /Validate both complete exports/u);
  });
  it("separates the same history from byte-identical serialization and returns only the declared contract", async () => {
    const left = await validateExport(raw), compact = JSON.stringify(fixture), right = await validateExport(encode(fixture));
    assert.notEqual(await sha256(raw.toString("utf8")), await sha256(compact));
    const summary = { collection_id: left.header.collection_id, entry_count: 27, head_sha256: left.entries[26]!.sha256 };
    assert.deepEqual(await compareJournals(left, right), { schema: "zerone-research-comparison/v1", relationship: "same-history", left: summary, right: summary, common_prefix: { entry_count: 27, head_sha256: summary.head_sha256 } });
  });
  it("reports strict prefixes in either direction, preserving the exact common head", async () => {
    const full = await validateExport(raw), prefix = await changed(j => { j.entries = j.entries.slice(0, 15); });
    const forward = await compareJournals(prefix, full), reverse = await compareJournals(full, prefix);
    assert.equal(forward.relationship, "left-prefix"); assert.equal(reverse.relationship, "right-prefix");
    assert.deepEqual(forward.common_prefix, { entry_count: 15, head_sha256: fixture.entries[14]!.sha256 });
    assert.deepEqual(forward.common_prefix, reverse.common_prefix);
    assert.deepEqual(forward.left, reverse.right); assert.deepEqual(forward.right, reverse.left);
  });
  it("uses the header hash as the empty history head and empty common prefix", async () => {
    const empty = await changed(j => { j.entries = []; }), full = await validateExport(raw);
    const root = await sha256(canonicalJSON(empty.header));
    const same = await compareJournals(empty, empty);
    assert.equal(same.relationship, "same-history"); assert.equal(same.left.head_sha256, root);
    assert.deepEqual(same.common_prefix, { entry_count: 0, head_sha256: root });
    const next = await compareJournals(empty, full); assert.equal(next.relationship, "left-prefix"); assert.deepEqual(next.common_prefix, same.common_prefix);
    assert.equal((await compareJournals(full, empty)).relationship, "right-prefix");
  });
  it("reports rehashed disagreement or acquisition-time changes as divergence, without preferring either", async () => {
    const left = await validateExport(raw);
    for (const change of [
      (j: Journal) => { j.entries[26]!.record.summary += " A different scoped assessment."; },
      (j: Journal) => { j.entries[26]!.recorded_at = "2025-09-28T12:00:00.000000Z"; },
    ]) {
      const right = await changed(change), result = await compareJournals(left, right);
      assert.equal(result.relationship, "diverged"); assert.deepEqual(result.common_prefix, { entry_count: 26, head_sha256: left.entries[25]!.sha256 });
      assert.notEqual(result.left.head_sha256, result.right.head_sha256);
      const reverse = await compareJournals(right, left); assert.equal(reverse.relationship, "diverged"); assert.deepEqual(reverse.common_prefix, result.common_prefix);
    }
    const first = await changed(j => { j.entries[0]!.record.summary += " Different first record."; });
    assert.deepEqual((await compareJournals(left, first)).common_prefix, { entry_count: 0, head_sha256: await sha256(canonicalJSON(left.header)) });
  });
  it("treats any header difference as a different root, including the same collection UUID", async () => {
    const left = await validateExport(raw);
    for (const right of [
      await changed(j => { j.header.created_at = "2025-08-31T00:00:00.000000Z"; }),
      await changed(j => { j.header.collection_id = "00000000-0000-0000-0000-000000000000"; }),
    ]) {
      const result = await compareJournals(left, right);
      assert.equal(result.relationship, "different-root"); assert.equal(result.common_prefix, null);
      assert.equal(result.left.entry_count, 27); assert.equal(result.right.entry_count, 27);
    }
  });
  it("compares complete files regardless of a separately selected historical snapshot", async () => {
    const left = await validateExport(raw), right = await changed(j => { j.entries[26]!.record.summary += " Later change."; });
    const expected = await compareJournals(left, right);
    for (const cutoff of [0, 15, 26, 27]) { selectSnapshot(left, cutoff); assert.deepEqual(await compareJournals(left, right), expected); }
    assert.equal(expected.common_prefix!.entry_count, 26);
  });
  it("compares the published fictional branches as differing assessments after their exact 27-row prefix", async () => {
    const [left, right] = await Promise.all(["a", "b"].map(async branch => validateExport(new Uint8Array(readFileSync(new URL(`../public/research/review/fictional-branch-${branch}.v1.json`, import.meta.url))))));
    const result = await compareJournals(left!, right!);
    assert.equal(result.relationship, "diverged");
    assert.deepEqual(result.common_prefix, { entry_count: 27, head_sha256: fixture.entries[26]!.sha256 });
    assert.equal(result.left.entry_count, 28); assert.equal(result.right.entry_count, 28);
    assert.equal(left!.entries[27]!.record.kind, "assessment"); assert.equal(right!.entries[27]!.record.kind, "assessment");
    assert.deepEqual(left!.entries.slice(0, 27), (await validateExport(raw)).entries);
  });
  it("offers comparison as a disabled local-file control with explicit scope and fictional downloads", () => {
    const page = readFileSync(new URL("../research/review/index.html", import.meta.url), "utf8");
    assert.match(page, /id="comparison-file"[^>]+disabled/u);
    assert.match(page, /independent of the timeline cutoff/u);
    assert.match(page, /fictional-branch-a\.v1\.json/u); assert.match(page, /fictional-branch-b\.v1\.json/u);
    assert.match(page, /Neither depicts an actual external reviewer/u);
  });
});

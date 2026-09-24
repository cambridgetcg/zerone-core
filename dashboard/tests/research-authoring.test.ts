import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { describe, it } from "node:test";
import {
  appendJournalRecords, canonicalJSON, compareJournals, createJournal, ENTRY_LIMIT, journalBytes,
  selectSnapshot, validateExport, type Contribution, type Journal, type JournalRecord, type Relation,
} from "../src/research-review-journal";

const pythonRoot = fileURLToPath(new URL("../../tools/research-review/", import.meta.url));
const decode = (bytes: Uint8Array): string => new TextDecoder().decode(bytes);
const contribution = (id: string): Contribution => ({
  id, kind: "contribution", contribution_type: "question", doi: null,
  title: "Fictional local authoring exercise", summary: "A synthetic test record, not a reported research result.",
  attributed_to: "Fictional practice author; no authenticated identity", occurred_on: null, evidence: [],
});
function pythonValidate(journal: Journal): void {
  const encoded = journalBytes(journal);
  const output = execFileSync("python3", ["-B", "-c",
    "import sys,journal\nvalue=journal.validate_export(journal.parse_json(sys.stdin.buffer.read()))\nsys.stdout.buffer.write(journal.canonical(value)+b'\\n')"],
  { cwd: pythonRoot, input: encoded, encoding: "utf8", maxBuffer: 16 * 1024 * 1024 });
  assert.equal(output, decode(encoded), "Existing Python validator accepts the exact browser-authored export");
}

describe("Local research journal authoring", () => {
  it("creates a verified empty collection with a fresh UUID and actual current UTC acquisition time", async () => {
    const before = Date.now(), first = await createJournal(), second = await createJournal(), after = Date.now();
    assert.equal(first.entries.length, 0);
    assert.match(first.header.collection_id, /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/u);
    assert.notEqual(first.header.collection_id, second.header.collection_id);
    assert.match(first.header.created_at, /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}000Z$/u);
    assert.ok(Date.parse(first.header.created_at) >= before && Date.parse(first.header.created_at) <= after);
    assert.equal(selectSnapshot(first).through, 0);
    assert.ok(Object.isFrozen(first) && Object.isFrozen(first.header) && Object.isFrozen(first.entries));
    pythonValidate(first);
  });

  it("appends contributions, a correction and a same-batch relation accepted by Python without altering the prefix", async () => {
    const empty = await createJournal(), original = await appendJournalRecords(empty, [contribution("practice:original")]);
    const before = journalBytes(original), correction: Contribution = {
      ...contribution("practice:correction"), supersedes: "practice:original", contribution_type: "finding",
      title: "Fictional correction — Ω😀", summary: "A narrower invented statement; the earlier record remains unchanged.",
    };
    const relation: Relation = {
      id: "practice:relation", kind: "relation", title: "Explicit fictional refinement", summary: "No native challenge is asserted.",
      attributed_to: correction.attributed_to, occurred_on: null, evidence: [],
      from_id: correction.id, to_id: "practice:original", relation_type: "refines",
    };
    const result = await appendJournalRecords(original, [correction, relation]);
    assert.deepEqual(result.header, original.header);
    assert.deepEqual(result.entries.slice(0, original.entries.length), original.entries);
    assert.deepEqual(journalBytes(original), before);
    assert.equal(empty.entries.length, 0);
    assert.equal(result.entries[1]!.record.supersedes, "practice:original");
    assert.equal(result.entries[1]!.recorded_at, result.entries[2]!.recorded_at);
    assert.equal((await compareJournals(original, result)).relationship, "left-prefix");
    pythonValidate(original); pythonValidate(result);
    const comparison = JSON.parse(execFileSync("python3", ["-B", "-c",
      "import sys,json,journal\na,b=json.load(sys.stdin)\nprint(json.dumps(journal.compare_exports(a,b)))"],
    { cwd: pythonRoot, input: JSON.stringify([original, result]), encoding: "utf8" }));
    assert.equal(comparison.relationship, "left-prefix");
    assert.equal(comparison.common_prefix.head_sha256, original.entries[0]!.sha256);
  });

  it("captures nested incoming records before any await without freezing or mutating caller objects", async () => {
    const original = await createJournal(), item = contribution("practice:captured");
    item.evidence.push({ label: "Synthetic byte commitment", url: null, sha256: "a".repeat(64) });
    const records = [item], expected = structuredClone(item);
    const pending = appendJournalRecords(original, records);
    item.title = "Changed during hashing"; item.evidence[0]!.sha256 = "b".repeat(64);
    records.push(contribution("practice:late-addition"));
    const result = await pending;
    assert.equal(result.entries.length, 1);
    assert.equal(canonicalJSON(result.entries[0]!.record), canonicalJSON(expected));
    assert.ok(!Object.isFrozen(item) && !Object.isFrozen(records));
    assert.ok(Object.isFrozen(result.entries[0]!.record.evidence[0]));
    assert.equal(original.entries.length, 0);
  });

  it("refuses complete invalid batches atomically, including duplicate IDs and missing or forward references", async () => {
    const original = await appendJournalRecords(await createJournal(), [contribution("practice:existing")]);
    const before = journalBytes(original);
    const cases: JournalRecord[][] = [
      [],
      [contribution("practice:new"), contribution("practice:existing")],
      [contribution("practice:new"), contribution("practice:new")],
      [contribution("practice:new"), { ...contribution("practice:bad"), supersedes: "practice:absent" }],
      [{ ...contribution("practice:bad"), supersedes: "practice:future" }, contribution("practice:future")],
      [{ ...contribution("practice:bad"), occurred_on: "2025-02-29" }],
      [{ ...contribution("practice:bad"), summary: "😀".repeat(2049) }],
      [{ ...contribution("practice:bad"), unexpected: true } as Contribution],
    ];
    for (const records of cases) {
      await assert.rejects(appendJournalRecords(original, records));
      assert.deepEqual(journalBytes(original), before);
      assert.equal(selectSnapshot(original).through, 1);
    }
  });

  it("refuses record, entry and complete export size limits without truncating the original", async () => {
    const original = await createJournal(), before = journalBytes(original), oversized = contribution("practice:oversized");
    oversized.evidence = Array.from({ length: 16 }, () => ({ label: "Synthetic", url: `https://example.invalid/${"😀".repeat(2000)}`, sha256: null }));
    await assert.rejects(appendJournalRecords(original, [oversized]), /64 KiB/u);
    await assert.rejects(appendJournalRecords(original, Array.from({ length: ENTRY_LIMIT + 1 }, (_, i) => contribution(`practice:${i}`))), /1000/u);
    const large = Array.from({ length: 250 }, (_, i) => ({
      ...contribution(`practice:large-${i}`), summary: "s".repeat(8192),
      evidence: Array.from({ length: 16 }, () => ({ label: "Synthetic", url: `https://example.invalid/${"a".repeat(1900)}`, sha256: null })),
    }));
    await assert.rejects(appendJournalRecords(original, large), /8 MiB/u);
    assert.deepEqual(journalBytes(original), before);
    const full = await appendJournalRecords(original, Array.from({ length: ENTRY_LIMIT }, (_, i) => contribution(`practice:${i}`)));
    const fullBytes = journalBytes(full);
    await assert.rejects(appendJournalRecords(full, [contribution("practice:one-more")]), /1000/u);
    assert.equal(full.entries.length, ENTRY_LIMIT);
    assert.deepEqual(journalBytes(full), fullBytes);
  });

  it("refuses a regressing clock relative to either creation or the last append (controlled synthetic clock)", async t => {
    const at = new Date("2030-01-02T03:04:05.123Z");
    t.mock.timers.enable({ apis: ["Date"], now: at });
    const original = await createJournal(), before = journalBytes(original);
    t.mock.timers.setTime(at.getTime() - 1);
    await assert.rejects(appendJournalRecords(original, [contribution("practice:backwards")]), /clock moved backwards/u);
    assert.deepEqual(journalBytes(original), before);
    t.mock.timers.setTime(at.getTime() + 10);
    const appended = await appendJournalRecords(original, [contribution("practice:later")]);
    const appendedBytes = journalBytes(appended);
    t.mock.timers.setTime(at.getTime() + 9);
    await assert.rejects(appendJournalRecords(appended, [contribution("practice:regressed")]), /clock moved backwards/u);
    assert.deepEqual(journalBytes(appended), appendedBytes);
  });

  it("requires verified journals and returns independent canonical bytes that can be re-imported", async () => {
    const original = await appendJournalRecords(await createJournal(), [contribution("practice:bytes")]);
    const fabricated = structuredClone(original);
    await assert.rejects(appendJournalRecords(fabricated, [contribution("practice:forged")]), /Validate the complete export/u);
    assert.throws(() => journalBytes(fabricated), /Validate the complete export/u);
    const first = journalBytes(original), second = journalBytes(original);
    assert.notEqual(first, second); assert.deepEqual(first, second);
    assert.equal(decode(first), canonicalJSON(original) + "\n");
    first.fill(0);
    assert.deepEqual(journalBytes(original), second);
    const reimported = await validateExport(second);
    assert.equal((await compareJournals(original, reimported)).relationship, "same-history");
    assert.throws(() => { original.entries[0]!.record.title = "Cannot edit a verified record"; }, TypeError);
  });
});

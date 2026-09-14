import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import { canonicalJSON, sha256, parseStrictJSON, validateExport, selectSnapshot, orderedContributions, impact, EXPORT_LIMIT, type Journal, type Concern, type Relation } from "../src/research-review-journal";

const fixtureBytes = new Uint8Array(readFileSync(new URL("../public/research/review/fictional-journal.v1.json", import.meta.url)));
const fixture = JSON.parse(new TextDecoder().decode(fixtureBytes)) as Journal;
const bytes = (value: unknown): Uint8Array => new TextEncoder().encode(JSON.stringify(value));
async function sealed(change: (value: Journal) => void): Promise<Uint8Array> {
  const value = structuredClone(fixture); change(value); let previous = await sha256(canonicalJSON(value.header));
  for (const row of value.entries) {
    row.previous_sha256 = previous;
    const { sha256: omitted, ...unsigned } = row; void omitted;
    row.sha256 = await sha256(canonicalJSON(unsigned)); previous = row.sha256;
  }
  return bytes(value);
}

describe("Research Review journal interoperability and historical projection", () => {
  it("validates the exact Python-produced fictional journal and canonical Unicode hashes", async () => {
    const value = await validateExport(fixtureBytes);
    assert.equal(value.entries.length, 27);
    assert.equal(value.entries.at(-1)!.sha256, "ba12850f11038e4d017aab44a08698384237a66b0ec621110625f63411fe31d6");
    assert.equal(value.entries[3]!.record.attributed_to, "Fictional researcher Ω");
    assert.equal(canonicalJSON({ z: "Ω😀\n", a: [1, null, "é"] }), '{"a":[1,null,"é"],"z":"Ω😀\\n"}');
    assert.ok(Object.isFrozen(value.entries[0]!.record));
    assert.throws(() => selectSnapshot(structuredClone(value), 1), /Validate the complete export/u);
  });
  it("refuses duplicate or unsupported keys, non-integer JSON forms, invalid UTF-8 and unpaired surrogates", async () => {
    for (const source of ['{"x":1,"x":2}', '{"x":1,"\\u0078":2}', '{"sequence":1.0}', '{"sequence":1e0}', '"\\ud800"', '"\\udc00"', '{"x":01}', '{"x":true,}', '{} trailing']) assert.throws(() => parseStrictJSON(source));
    await assert.rejects(validateExport(Uint8Array.from([0xff, 0xfe])));
    await assert.rejects(validateExport(new TextEncoder().encode("\ufeff" + JSON.stringify(fixture))));
    await assert.rejects(validateExport(bytes({ ...fixture, confidence: "trusted" })), /unsupported fields/u);
    await assert.rejects(validateExport(await sealed(j => { Object.assign(j.entries[0]!.record, { authenticated: true }); })), /unsupported fields/u);
  });
  it("refuses tampering anywhere in a full export, including later than a desired cutoff", async () => {
    const changed = structuredClone(fixture); changed.entries.at(-1)!.record.summary += " changed";
    await assert.rejects(validateExport(bytes(changed)), /SHA256 mismatch/u);
    const header = structuredClone(fixture); header.header.created_at = "2025-08-01T00:00:00.000000Z";
    await assert.rejects(validateExport(bytes(header)), /hash linkage/u);
    const predecessor = structuredClone(fixture); predecessor.entries[3]!.previous_sha256 = "0".repeat(64);
    await assert.rejects(validateExport(bytes(predecessor)), /hash linkage/u);
    assert.throws(() => selectSnapshot(changed, 1), /Validate the complete export/u);
  });
  it("enforces Gregorian dates, exact timestamps, strict IDs and allowed fields even when hashes are recomputed", async () => {
    const invalid: Array<(j: Journal) => void> = [
      j => { j.entries[0]!.record.id = "source\n"; },
      j => { j.header.collection_id += "\n"; },
      j => { j.entries[0]!.record.occurred_on = "2025-02-29"; },
      j => { j.entries[0]!.record.occurred_on = "0000-01-01"; },
      j => { j.entries[0]!.record.occurred_on = "2025-01-01\n"; },
      j => { j.entries[0]!.record.supersedes = null as unknown as string; },
      j => { j.entries[0]!.record.supersedes = "finding-v1"; },
      j => { j.entries[1]!.record.id = "source"; },
      j => { j.entries[1]!.record.summary = "😀".repeat(2049); },
      j => { j.entries[1]!.record.title = "😀".repeat(301); },
      j => { (j.entries[18]!.record as Concern).notice_type = ""; },
      j => { j.entries[1]!.record.evidence = [{ label: "absent", url: null, sha256: null }]; },
      j => { j.entries[1]!.record.evidence = [{ label: "hash", url: null, sha256: "a".repeat(64) + "\n" }]; },
      j => { j.entries[1]!.record.evidence = Array.from({ length: 17 }, () => ({ label: "hash", url: null, sha256: "a".repeat(64) })); },
      j => { j.entries[1]!.recorded_at = "2025-09-02T12:00:00Z"; },
      j => { j.entries[1]!.recorded_at = "2025-09-01T11:00:00.000000Z"; },
      j => { j.entries[1]!.recorded_at = "2025-09-02T24:00:00.000000Z"; },
      j => { j.entries[1]!.sequence = 3; },
      j => { (j.entries[9]!.record as Relation).from_id = "early-source"; },
      j => { (j.entries[9]!.record as Relation).to_id = "experiment"; },
    ];
    for (const change of invalid) await assert.rejects(validateExport(await sealed(change)));
    const valid = await validateExport(await sealed(j => { j.entries[1]!.record.title = "😀".repeat(300); j.entries[1]!.record.occurred_on = "2000-02-29"; }));
    assert.equal(Array.from(valid.entries[1]!.record.title).length, 300);
  });
  it("matches the backend HTTPS evidence URL scope without fetching or normalizing", async () => {
    const valid = ["https://example.org/a?q=x#f", "HTTPS://example.org/é", "https://[::1]/", "https://[2001:DB8::1]:443/x", "https://[::ffff:192.0.2.1]/", "https://example.org:0/"];
    const invalid = ["http://example.org", "https://a@b/", "https://example.org:/", "https://example.org:65536/", "https://example.org/\u0085", "https://example.org/\n", "https://example.org\\x", "https://[::1]garbage/", "https://[v1.name]/", "https://[fe80::1%25zone]/", "https://[1.2.3.4::]/", "https://[12345::]/"];
    for (const url of [...valid, ...invalid]) {
      const encoded = await sealed(j => { j.entries[0]!.record.evidence[0]!.url = url; });
      if (valid.includes(url)) assert.equal((await validateExport(encoded)).entries[0]!.record.evidence[0]!.url, url);
      else await assert.rejects(validateExport(encoded), Error, url);
    }
  });
  it("bounds total bytes, entry count and record size before display", async () => {
    await assert.rejects(validateExport(new Uint8Array(EXPORT_LIMIT + 1)), /8 MiB/u);
    await assert.rejects(validateExport(bytes({ ...fixture, entries: Array(1001).fill(fixture.entries[0]) })), /1000/u);
    await assert.rejects(validateExport(await sealed(j => {
      j.entries[0]!.record.summary = "😀".repeat(2048);
      j.entries[0]!.record.evidence = Array.from({ length: 16 }, () => ({ label: "😀".repeat(200), url: `https://example.org/${"😀".repeat(2000)}`, sha256: "a".repeat(64) }));
    })), /64 KiB/u);
  });
  it("applies every cutoff before occurrence sorting, version links, concerns and assessments", async () => {
    const j = await validateExport(fixtureBytes);
    for (let n = 0; n <= j.entries.length; n++) {
      const s = selectSnapshot(j, n); assert.equal(s.entries.length, n);
      assert.ok([...s.contributions, ...s.relations, ...s.concerns, ...s.assessments].every(e => e.sequence <= n));
      assert.equal(s.historical, n < j.entries.length);
    }
    const before = selectSnapshot(j, 15); assert.ok(!before.entries.some(e => e.record.id === "early-source"));
    assert.equal(before.concerns.length, 0); assert.equal(before.assessments.length, 0);
    const later = selectSnapshot(j, 25); assert.equal(orderedContributions(later, "occurred").dated[0]!.record.id, "early-source");
    assert.deepEqual(orderedContributions(later, "occurred").unknown.map(e => e.record.id), ["method"]);
    assert.equal(orderedContributions(later, "recorded").unknown.length, 0);
    assert.ok(later.contributions.some(e => e.record.id === "finding-v1"));
    assert.ok(later.contributions.some(e => e.record.id === "finding-v2" && e.record.supersedes === "finding-v1"));
    assert.deepEqual(selectSnapshot(j, 20).assessments.map(e => e.record.id), ["link-assessment", "assessment-open"]);
    assert.deepEqual(later.assessments.map(e => e.record.id), ["link-assessment", "assessment-open", "assessment-addressed", "assessment-disputed"]);
    assert.throws(() => selectSnapshot(j, -1)); assert.throws(() => selectSnapshot(j, 28));
  });
  it("traverses support and reverse requirements/shared-input only, preserving edge types and excluding citations", async () => {
    const j = await validateExport(fixtureBytes), s = selectSnapshot(j);
    const fromInput = impact(s, "input");
    assert.deepEqual([...fromInput.nodes].sort(), ["input", "experiment", "finding-v1", "finding-v2", "downstream"].sort());
    assert.equal(fromInput.steps.find(p => p.relation.record.id === "edge-input")!.mode, "shared-input");
    assert.ok(fromInput.steps.every(p => !["edge-citation", "edge-inspiration", "edge-refinement"].includes(p.relation.record.id)));
    assert.deepEqual([...impact(s, "method").nodes].sort(), ["method", "finding-v1", "downstream"].sort());
    assert.equal(impact(selectSnapshot(j, 9), "input").nodes.size, 1);
    assert.throws(() => impact(selectSnapshot(j, 15), "early-source"));
    const cyclic = await validateExport(await sealed(value => {
      const row = structuredClone(value.entries[9]!); row.sequence = 28; row.recorded_at = "2025-09-28T12:00:00.000000Z"; row.record.id = "cycle";
      Object.assign(row.record, { from_id: "downstream", to_id: "method" }); value.entries.push(row);
    }));
    const path = impact(selectSnapshot(cyclic), "method"); assert.equal(path.nodes.size, 3); assert.equal(path.steps.length, 3);
  });
  it("keeps the dedicated route passive and the fictional fixture separate from AdaGrad", () => {
    const page = readFileSync(new URL("../research/review/index.html", import.meta.url), "utf8");
    const controller = readFileSync(new URL("../src/research-review.ts", import.meta.url), "utf8");
    assert.match(page, /<noscript>/u); assert.match(page, /id="journal-file"[^>]+disabled/u);
    assert.match(page, /fictional example depicts no real research/u);
    assert.doesNotMatch(controller, /innerHTML|outerHTML|insertAdjacentHTML|localStorage|sessionStorage|indexedDB|sendBeacon|XMLHttpRequest|WebSocket|eval\(/u);
    assert.equal((controller.match(/\bfetch\(/gu) ?? []).length, 1);
    assert.match(controller, /fetch\("\/research\/review\/fictional-journal\.v1\.json", \{ method: "GET", credentials: "omit", redirect: "error" \}\)/u);
    assert.doesNotMatch(controller, /from ["'][^"']*(?:api|wallet|pi-|stargate)/u);
  });
});

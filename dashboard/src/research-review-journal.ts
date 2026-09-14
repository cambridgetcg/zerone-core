/** Local journal interoperability. No fetching, persistence, identities or chain authority. */
export const EXPORT_LIMIT = 8 * 1024 * 1024;
export const RECORD_LIMIT = 64 * 1024;
export const ENTRY_LIMIT = 1000;
export const CONTRIBUTION_TYPES = ["source", "question", "claim", "method", "experiment", "finding", "review"] as const;
export const RELATION_TYPES = ["supports", "requires", "inspired-by", "uses-input", "refines", "cites"] as const;
export const CONCERN_TYPES = ["publisher-notice", "reported-concern", "metadata-conflict", "artifact-discrepancy"] as const;
export const DISPOSITIONS = ["needs-review", "notice-checked", "disputed", "addressed", "inconclusive"] as const;
export interface Evidence { label: string; url: string | null; sha256: string | null }
interface Common { id: string; kind: string; title: string; summary: string; attributed_to: string; occurred_on: string | null; evidence: Evidence[]; supersedes?: string }
export interface Contribution extends Common { kind: "contribution"; contribution_type: typeof CONTRIBUTION_TYPES[number]; doi: string | null }
export interface Relation extends Common { kind: "relation"; from_id: string; to_id: string; relation_type: typeof RELATION_TYPES[number] }
export interface Concern extends Common { kind: "concern"; target_id: string; category: typeof CONCERN_TYPES[number]; notice_type: string | null; provider: string }
export interface Assessment extends Common { kind: "assessment"; target_id: string; disposition: typeof DISPOSITIONS[number] }
export type JournalRecord = Contribution | Relation | Concern | Assessment;
export interface Header { schema: "zerone-research-journal/v1"; collection_id: string; created_at: string }
export interface Entry { sequence: number; recorded_at: string; previous_sha256: string; record: JournalRecord; sha256: string }
export interface Journal { schema: "zerone-research-export/v1"; header: Header; entries: Entry[] }
export interface JournalComparison {
  schema: "zerone-research-comparison/v1";
  relationship: "same-history" | "left-prefix" | "right-prefix" | "diverged" | "different-root";
  left: { collection_id: string; entry_count: number; head_sha256: string };
  right: { collection_id: string; entry_count: number; head_sha256: string };
  common_prefix: { entry_count: number; head_sha256: string } | null;
}
export interface Snapshot { journal: Journal; through: number; historical: boolean; entries: Entry[]; contributions: Entry[]; relations: Entry[]; concerns: Entry[]; assessments: Entry[] }
export interface ImpactStep { relation: Entry; from: string; to: string; mode: "support" | "dependency" | "shared-input" }
const verified = new WeakSet<object>();
const utf8 = new TextEncoder();
const HASH = /^[0-9a-f]{64}(?![\s\S])/u;
const ID = /^[a-z0-9][a-z0-9:._-]{0,127}(?![\s\S])/u;
const SPACE_OR_CONTROL = /[\p{White_Space}\u0000-\u001f\u007f]/u;
type ObjectValue = Record<string, unknown>;

function fail(message: string): never { throw new Error(message); }
function object(value: unknown, name: string): ObjectValue {
  if (!value || typeof value !== "object" || Array.isArray(value)) fail(`Malformed ${name}`);
  return value as ObjectValue;
}
function keys(value: ObjectValue, required: readonly string[], optional: readonly string[] = []): void {
  if (required.some(k => !Object.hasOwn(value, k)) || Object.keys(value).some(k => !required.includes(k) && !optional.includes(k))) fail("Missing or unsupported fields");
}
function text(value: unknown, min: number, max: number, name: string): string {
  if (typeof value !== "string" || Array.from(value).length < min || Array.from(value).length > max || /[\uD800-\uDBFF](?![\uDC00-\uDFFF])|(?<![\uD800-\uDBFF])[\uDC00-\uDFFF]/u.test(value)) fail(`Invalid ${name}`);
  return value;
}
function identifier(value: unknown): string { const id = text(value, 1, 128, "record ID"); if (!ID.test(id)) fail("Invalid record ID"); return id; }
function calendar(value: unknown): string {
  const date = text(value, 10, 10, "date"); const match = /^(\d{4})-(\d{2})-(\d{2})$/u.exec(date);
  if (!match) fail("Invalid calendar date");
  const year = Number(match[1]), month = Number(match[2]), day = Number(match[3]);
  const days = [31, year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0) ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
  if (year < 1 || month < 1 || month > 12 || day < 1 || day > days[month - 1]!) fail("Invalid calendar date");
  return date;
}
function timestamp(value: unknown): string {
  const t = text(value, 27, 27, "recorded timestamp");
  const m = /^(\d{4}-\d{2}-\d{2})T(\d{2}):(\d{2}):(\d{2})\.\d{6}Z$/u.exec(t);
  if (!m || Number(m[2]) > 23 || Number(m[3]) > 59 || Number(m[4]) > 59) fail("Invalid UTC timestamp");
  calendar(m[1]); return t;
}
function choice<T extends string>(value: unknown, choices: readonly T[], name: string): T {
  if (typeof value !== "string" || !choices.includes(value as T)) fail(`Unsupported ${name}`);
  return value as T;
}

// urllib's bracketed-host scope without DNS lookup or WHATWG URL normalization.
function ipv6(value: string): boolean {
  const address = value; const halves = address.split("::"); if (halves.length > 2) return false;
  const groups = halves.flatMap(v => v ? v.split(":") : []); let count = 0;
  for (let i = 0; i < groups.length; i++) {
    const group = groups[i]!;
    if (group.includes(".")) {
      if (i !== groups.length - 1 || !address.endsWith(group) || !/^(0|[1-9]\d{0,2})(\.(0|[1-9]\d{0,2})){3}$/u.test(group) || group.split(".").some(v => Number(v) > 255)) return false;
      count += 2;
    } else { if (!/^[0-9a-f]{1,4}$/iu.test(group)) return false; count++; }
  }
  return halves.length === 2 ? count < 8 : count === 8;
}
function evidenceURL(value: unknown): string {
  const url = text(value, 1, 2048, "HTTPS evidence URL");
  if (SPACE_OR_CONTROL.test(url) || url.includes("\\")) fail("Invalid HTTPS evidence URL");
  const match = /^https:\/\/([^/?#]+)(?:[/?#].*)?$/iu.exec(url); if (!match) fail("Evidence URL must use HTTPS and a hostname");
  const authority = match[1]!; if (authority.includes("@")) fail("Credentials in evidence URL");
  const normalized = authority.replace(/[@:#?]/gu, "").normalize("NFKC");
  if (normalized !== authority.replace(/[@:#?]/gu, "") && /[/?#@:]/u.test(normalized)) fail("Invalid URL authority");
  let port: string | undefined;
  if (authority.startsWith("[")) {
    const bracket = /^\[([0-9A-Fa-f:.]+)\](?::([0-9]+))?(?![\s\S])/u.exec(authority); if (!bracket) fail("Invalid bracketed hostname");
    const host = bracket[1]!;
    if (!ipv6(host)) fail("Invalid bracketed hostname");
    port = bracket[2];
  } else {
    if (/[\[\]]/u.test(authority)) fail("Invalid hostname");
    const parts = authority.split(":"); if (parts.length > 2 || !parts[0]) fail("Invalid hostname"); port = parts[1];
  }
  if (port !== undefined && (!/^\d+$/u.test(port) || Number(port) > 65535)) fail("Invalid URL port");
  return url;
}

/** Strict JSON grammar with duplicate keys and float/exponent tokens refused. */
export function parseStrictJSON(source: string): unknown {
  let at = 0;
  const whitespace = (): void => { while (/[\t\r\n ]/u.test(source[at] ?? "x")) at++; };
  const string = (): string => {
    const start = at++; let escaped = false;
    while (at < source.length) {
      const c = source[at++]!;
      if (c === '"' && !escaped) { const value: unknown = JSON.parse(source.slice(start, at)); return text(value, 0, EXPORT_LIMIT, "JSON string"); }
      if (c === "\\" && !escaped) escaped = true; else escaped = false;
    }
    return fail("Unterminated JSON string");
  };
  const value = (depth: number): unknown => {
    if (depth > 16) fail("JSON nesting exceeds the record format"); whitespace(); const c = source[at];
    if (c === '"') return string();
    if (c === "{") {
      at++; whitespace(); const result: ObjectValue = Object.create(null);
      if (source[at] === "}") { at++; return result; }
      while (true) {
        whitespace(); if (source[at] !== '"') fail("Expected JSON object key"); const key = string();
        if (Object.hasOwn(result, key)) fail("Duplicate JSON key"); whitespace(); if (source[at++] !== ":") fail("Expected colon"); result[key] = value(depth + 1); whitespace();
        const next = source[at++]; if (next === "}") return result; if (next !== ",") fail("Expected object separator");
      }
    }
    if (c === "[") {
      at++; whitespace(); const result: unknown[] = []; if (source[at] === "]") { at++; return result; }
      while (true) { result.push(value(depth + 1)); whitespace(); const next = source[at++]; if (next === "]") return result; if (next !== ",") fail("Expected array separator"); }
    }
    for (const [literal, result] of [["true", true], ["false", false], ["null", null]] as const) if (source.startsWith(literal, at)) { at += literal.length; return result; }
    const number = /^-?(0|[1-9][0-9]*)/u.exec(source.slice(at));
    if (number) { at += number[0].length; if (/[.eE]/u.test(source[at] ?? "x")) fail("Only integer sequence values are supported"); const n = Number(number[0]); if (!Number.isSafeInteger(n)) fail("Unsafe JSON integer"); return n; }
    return fail("Invalid JSON value");
  };
  const result = value(0); whitespace(); if (at !== source.length) fail("Unexpected data after JSON value"); return result;
}

export function canonicalJSON(value: unknown): string {
  if (value === null || typeof value === "string" || typeof value === "boolean" || typeof value === "number") return JSON.stringify(value);
  if (Array.isArray(value)) return `[${value.map(canonicalJSON).join(",")}]`;
  return `{${Object.keys(value as object).sort().map(k => `${JSON.stringify(k)}:${canonicalJSON((value as ObjectValue)[k])}`).join(",")}}`;
}
export async function sha256(value: string): Promise<string> {
  const hash = await crypto.subtle.digest("SHA-256", utf8.encode(value));
  return [...new Uint8Array(hash)].map(v => v.toString(16).padStart(2, "0")).join("");
}
function record(value: unknown, prior: Map<string, JournalRecord>): JournalRecord {
  const r = object(value, "record");
  const common = ["id", "kind", "title", "summary", "attributed_to", "occurred_on", "evidence"];
  const extras: Record<string, string[]> = { contribution: ["contribution_type", "doi"], relation: ["from_id", "to_id", "relation_type"], concern: ["target_id", "category", "notice_type", "provider"], assessment: ["target_id", "disposition"] };
  if (typeof r.kind !== "string" || !Object.hasOwn(extras, r.kind)) fail("Unknown record kind");
  keys(r, [...common, ...extras[r.kind]!], ["supersedes"]);
  const id = identifier(r.id); if (prior.has(id)) fail("Duplicate immutable record ID");
  text(r.title, 1, 300, "title"); text(r.attributed_to, 1, 200, "attribution"); text(r.summary, 1, 8192, "summary");
  if (utf8.encode(r.summary as string).length > 8192) fail("Summary exceeds UTF-8 byte limit");
  if (r.occurred_on !== null) calendar(r.occurred_on);
  if (!Array.isArray(r.evidence) || r.evidence.length > 16) fail("Invalid evidence list");
  for (const raw of r.evidence) {
    const e = object(raw, "evidence"); keys(e, ["label", "url", "sha256"]); text(e.label, 1, 200, "evidence label");
    if (e.url !== null) evidenceURL(e.url);
    if (e.sha256 !== null && (typeof e.sha256 !== "string" || !HASH.test(e.sha256))) fail("Invalid evidence SHA256");
    if (e.url === null && e.sha256 === null) fail("Evidence needs a URL or SHA256");
  }
  const target = (value: unknown, kinds: string[]): JournalRecord => { const id = identifier(value); const p = prior.get(id); if (!p || !kinds.includes(p.kind)) fail("Reference must name an earlier record of the required kind"); return p; };
  if (Object.hasOwn(r, "supersedes")) target(r.supersedes, [r.kind]);
  if (r.kind === "contribution") {
    choice(r.contribution_type, CONTRIBUTION_TYPES, "contribution type");
    if (r.doi !== null) { const doi = text(r.doi, 1, RECORD_LIMIT, "DOI"); if (!/^10\.[0-9]{4,9}\/.+$/u.test(doi) || SPACE_OR_CONTROL.test(doi) || doi !== doi.toLowerCase()) fail("DOI must be normalized lowercase"); }
  } else if (r.kind === "relation") {
    target(r.from_id, ["contribution"]); target(r.to_id, ["contribution"]); if (r.from_id === r.to_id) fail("Self relation refused"); choice(r.relation_type, RELATION_TYPES, "relation type");
  } else if (r.kind === "concern") {
    target(r.target_id, ["contribution"]); choice(r.category, CONCERN_TYPES, "concern category"); text(r.provider, 1, 200, "provider"); if (r.notice_type !== null) text(r.notice_type, 1, 100, "notice type");
  } else { target(r.target_id, ["concern", "relation"]); choice(r.disposition, DISPOSITIONS, "assessment disposition"); }
  if (utf8.encode(canonicalJSON(r)).length > RECORD_LIMIT) fail("Record exceeds 64 KiB");
  return r as unknown as JournalRecord;
}
function freeze(value: unknown): void { if (value && typeof value === "object") { Object.values(value).forEach(freeze); Object.freeze(value); } }

export async function validateExport(bytes: Uint8Array): Promise<Journal> {
  if (!bytes.length || bytes.length > EXPORT_LIMIT) fail("Export must be nonempty and at most 8 MiB");
  const parsed = object(parseStrictJSON(new TextDecoder("utf-8", { fatal: true, ignoreBOM: true }).decode(bytes)), "export");
  keys(parsed, ["schema", "header", "entries"]); if (parsed.schema !== "zerone-research-export/v1") fail("Unsupported export schema");
  const header = object(parsed.header, "header"); keys(header, ["schema", "collection_id", "created_at"]);
  if (header.schema !== "zerone-research-journal/v1" || typeof header.collection_id !== "string" || !/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}(?![\s\S])/u.test(header.collection_id)) fail("Invalid collection identity or schema");
  let recorded = timestamp(header.created_at), previous = await sha256(canonicalJSON(header));
  if (!Array.isArray(parsed.entries) || parsed.entries.length > ENTRY_LIMIT) fail("Export exceeds 1000 entries or is malformed");
  const prior = new Map<string, JournalRecord>();
  for (let i = 0; i < parsed.entries.length; i++) {
    const row = object(parsed.entries[i], "entry"); keys(row, ["sequence", "recorded_at", "previous_sha256", "record", "sha256"]);
    if (row.sequence !== i + 1 || !Number.isSafeInteger(row.sequence)) fail("Sequence must be contiguous from one");
    const time = timestamp(row.recorded_at); if (time < recorded) fail("Recorded timestamps must not decrease");
    if (typeof row.sha256 !== "string" || !HASH.test(row.sha256) || row.previous_sha256 !== previous) fail("Broken journal hash linkage");
    const r = record(row.record, prior);
    const expected = await sha256(canonicalJSON({ sequence: row.sequence, recorded_at: time, previous_sha256: previous, record: r }));
    if (expected !== row.sha256) fail("Journal entry SHA256 mismatch");
    prior.set(r.id, r); previous = expected; recorded = time;
  }
  const journal = parsed as unknown as Journal; freeze(journal); verified.add(journal); return journal;
}
/** Compare complete validated histories, never selecting an authoritative branch. */
export async function compareJournals(left: Journal, right: Journal): Promise<JournalComparison> {
  if (!verified.has(left) || !verified.has(right)) fail("Validate both complete exports before comparing histories");
  const leftHeader = canonicalJSON(left.header), rightHeader = canonicalJSON(right.header);
  const [leftRoot, rightRoot] = await Promise.all([sha256(leftHeader), sha256(rightHeader)]);
  const summary = (journal: Journal, root: string) => ({ collection_id: journal.header.collection_id, entry_count: journal.entries.length, head_sha256: journal.entries.at(-1)?.sha256 ?? root });
  const result: JournalComparison = { schema: "zerone-research-comparison/v1", relationship: "different-root", left: summary(left, leftRoot), right: summary(right, rightRoot), common_prefix: null };
  if (leftHeader !== rightHeader) return result;
  let shared = 0;
  while (shared < Math.min(left.entries.length, right.entries.length) && canonicalJSON(left.entries[shared]) === canonicalJSON(right.entries[shared])) shared++;
  result.common_prefix = { entry_count: shared, head_sha256: shared ? left.entries[shared - 1]!.sha256 : leftRoot };
  result.relationship = shared === left.entries.length
    ? (shared === right.entries.length ? "same-history" : "left-prefix")
    : shared === right.entries.length ? "right-prefix" : "diverged";
  return result;
}
export function selectSnapshot(journal: Journal, through = journal.entries.length): Snapshot {
  if (!verified.has(journal)) fail("Validate the complete export before selecting a snapshot");
  if (!Number.isInteger(through) || through < 0 || through > journal.entries.length) fail("Invalid sequence cutoff");
  const entries = journal.entries.slice(0, through);
  return { journal, through, historical: through < journal.entries.length, entries,
    contributions: entries.filter(e => e.record.kind === "contribution"), relations: entries.filter(e => e.record.kind === "relation"),
    concerns: entries.filter(e => e.record.kind === "concern"), assessments: entries.filter(e => e.record.kind === "assessment") };
}
export function orderedContributions(snapshot: Snapshot, clock: "occurred" | "recorded"): { dated: Entry[]; unknown: Entry[] } {
  const known = snapshot.contributions.filter(e => clock === "recorded" || e.record.occurred_on !== null);
  known.sort((a, b) => (clock === "recorded" ? a.recorded_at.localeCompare(b.recorded_at) : a.record.occurred_on!.localeCompare(b.record.occurred_on!)) || a.sequence - b.sequence);
  return { dated: known, unknown: clock === "occurred" ? snapshot.contributions.filter(e => e.record.occurred_on === null) : [] };
}
/** Possible downstream reassessment, preserving native edge meaning. No falsity propagation. */
export function impact(snapshot: Snapshot, target: string): { nodes: Set<string>; steps: ImpactStep[] } {
  if (!snapshot.contributions.some(e => e.record.id === target)) fail("Impact target is absent from this snapshot");
  const nodes = new Set([target]), queue = [target], steps: ImpactStep[] = [], seen = new Set<string>();
  while (queue.length) {
    const current = queue.shift()!;
    for (const entry of snapshot.relations) {
      const r = entry.record as Relation; let from: string, to: string, mode: ImpactStep["mode"];
      if (r.relation_type === "supports") { from = r.from_id; to = r.to_id; mode = "support"; }
      else if (r.relation_type === "requires") { from = r.to_id; to = r.from_id; mode = "dependency"; }
      else if (r.relation_type === "uses-input") { from = r.to_id; to = r.from_id; mode = "shared-input"; }
      else continue;
      if (from !== current || seen.has(r.id)) continue; seen.add(r.id); steps.push({ relation: entry, from, to, mode });
      if (!nodes.has(to)) { nodes.add(to); queue.push(to); }
    }
  }
  return { nodes, steps };
}

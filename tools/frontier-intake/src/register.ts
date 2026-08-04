import { readFileSync } from "node:fs";

export const REGISTER_SCHEMA = "zerone.math-breakthrough-register/v0" as const;

export type EntryStatus =
  | "formalized"
  | "published_peer_reviewed"
  | "preprint_widely_accepted"
  | "preprint_under_review"
  | "announced_unverified"
  | "refuted"
  | "open_conjecture";

export type ChainIntent = "assert" | "conjecture" | "withhold";

export interface RegisterEntry {
  id: string;
  name: string;
  statement: string;
  authors: string;
  date: string;
  sources: string[];
  status: EntryStatus;
  machineVerifiable: boolean;
  significance: string;
  caveats: string;
  formalization?: string;
  chain: {
    intent: ChainIntent;
    domain: string;
    category: string;
    subject: string;
    predicate: string;
    tags: string;
    firstBatch: number | null;
    relations?: Array<{ relation: string; target: string; justification: string }>;
  };
}

export interface Register {
  schema: typeof REGISTER_SCHEMA;
  version: string;
  snapshotDate: string;
  authoritative: boolean;
  chainAdmission: {
    assertEligibleStatuses: EntryStatus[];
    conjectureEligibleStatuses: EntryStatus[];
    withheldStatuses: EntryStatus[];
  };
  entries: RegisterEntry[];
}

// Live chain params (MinClaimTextLength / MaxClaimTextLength).
export const MIN_STATEMENT = 20;
export const MAX_STATEMENT = 1000;
export const MAX_TAGS = 10;
export const MAX_TAG_LENGTH = 50;

export const CATEGORY_VOCABULARY: Record<ChainIntent, string[]> = {
  assert: ["theorem", "refutation", "formalization"],
  conjecture: ["open-question"],
  withhold: ["theorem", "refutation", "formalization", "open-question"],
};

// Press/aggregator hosts may evidence significance only (spec P11); every
// assert entry needs at least one source that is NOT on this list.
const AGGREGATOR_HOSTS = [
  "theconversation.com", "singularityhub.com", "quantamagazine.org",
  "kingy.ai", "biggo.com", "medium.com", "newscientist.com", "nytimes.com",
];

function isAggregatorOnly(sources: string[]): boolean {
  return sources.every(source => AGGREGATOR_HOSTS.some(host => source.includes(host)));
}

// contradicts excluded on purpose: it instantly flips the target to CONTESTED.
export const RELATION_VOCABULARY = ["supports", "requires", "refines", "generalizes", "cites", "reformulates", "supersedes"];

const SUB_PEER_REVIEW: EntryStatus[] = ["preprint_widely_accepted", "preprint_under_review", "announced_unverified"];
const DISCLOSURE_PATTERN = /preprint|not yet (?:a )?(?:journal|refereed|peer)|announced|awaiting (?:journal|peer)|under review|social.media/i;

export function loadRegister(path: string): Register {
  const parsed = JSON.parse(readFileSync(path, "utf8")) as Register;
  if (parsed?.schema !== REGISTER_SCHEMA) {
    throw new Error(`${path} is not a ${REGISTER_SCHEMA} register`);
  }
  return parsed;
}

export interface ValidationIssue {
  entry: string;
  problem: string;
}

/** Fail-closed register validation: schema, lengths, admission consistency. */
export function validateRegister(register: Register): ValidationIssue[] {
  const issues: ValidationIssue[] = [];
  const seenIds = new Set<string>();
  const seenSubjects = new Map<string, string>();
  const admission = register.chainAdmission;
  for (const entry of register.entries) {
    const flag = (problem: string) => issues.push({ entry: entry.id ?? "<missing id>", problem });
    if (!entry.id || seenIds.has(entry.id)) flag("missing or duplicate id");
    seenIds.add(entry.id);
    if (!entry.statement || entry.statement.length < MIN_STATEMENT || entry.statement.length > MAX_STATEMENT) {
      flag(`statement length ${entry.statement?.length ?? 0} outside ${MIN_STATEMENT}..${MAX_STATEMENT}`);
    }
    if (!entry.sources || entry.sources.length === 0) flag("no sources pinned");
    if (!entry.authors) flag("no authors");
    if (!/^\d{4}-\d{2}$/.test(entry.date ?? "")) flag(`date "${entry.date}" is not YYYY-MM`);
    const chain = entry.chain;
    if (!chain) { flag("no chain block"); continue; }
    if (chain.domain !== "mathematics") flag(`domain "${chain.domain}" is not mathematics`);
    if (!chain.subject) flag("no chain.subject (needed for novelty/dedup/bounty indexing)");
    const tags = chain.tags ? chain.tags.split(",") : [];
    if (tags.length > MAX_TAGS) flag(`${tags.length} tags exceeds ${MAX_TAGS}`);
    for (const tag of tags) {
      if (tag.length > MAX_TAG_LENGTH) flag(`tag "${tag}" exceeds ${MAX_TAG_LENGTH} chars`);
    }
    if (chain.intent === "assert" && !admission.assertEligibleStatuses.includes(entry.status)) {
      flag(`intent assert but status ${entry.status} is not assert-eligible`);
    }
    if (chain.intent === "conjecture" && !admission.conjectureEligibleStatuses.includes(entry.status)) {
      flag(`intent conjecture but status ${entry.status} is not conjecture-eligible`);
    }
    if (entry.status === "formalized" && !entry.machineVerifiable) {
      flag("status formalized but machineVerifiable is false");
    }
    if (!CATEGORY_VOCABULARY[chain.intent]?.includes(chain.category)) {
      flag(`category "${chain.category}" not in vocabulary for intent ${chain.intent}`);
    }
    if (chain.intent === "assert") {
      if (entry.status === "refuted") {
        if (chain.category !== "refutation") flag("refuted assert entries must use category refutation");
        if (!/false|refut|disproof|disprove|counterexample/i.test(entry.statement)) {
          flag("refuted assert statement must state the refutation, never the refuted proposition");
        }
      }
      if (SUB_PEER_REVIEW.includes(entry.status) && !DISCLOSURE_PATTERN.test(entry.statement)) {
        flag("sub-peer-review assert statement must disclose its own evidence level in the claim text (spec §2)");
      }
      if (isAggregatorOnly(entry.sources)) {
        flag("assert entry has aggregator-only sources; statement/status must terminate in a primary artifact (P11)");
      }
    }
    if (chain.firstBatch !== null && chain.intent !== "assert") {
      flag("firstBatch set but intent is not assert");
    }
    if (chain.intent === "assert" && chain.subject) {
      const existing = seenSubjects.get(chain.subject);
      if (existing) flag(`duplicate assert subject with ${existing}`);
      seenSubjects.set(chain.subject, entry.id);
    }
    for (const relation of chain.relations ?? []) {
      if (!RELATION_VOCABULARY.includes(relation.relation)) {
        flag(`relation type "${relation.relation}" not in vocabulary (contradicts is deliberately excluded: it flips targets to CONTESTED)`);
      }
      const isFactId = /^[0-9a-f]{32}$/.test(relation.target);
      const isEntryRef = relation.target.startsWith("entry:");
      if (!isFactId && !isEntryRef) {
        flag(`relation target "${relation.target}" is neither a 32-hex fact id nor an entry:<id> reference`);
      }
      if (isEntryRef && !register.entries.some(other => other.id === relation.target.slice(6))) {
        flag(`relation target ${relation.target} names an unknown register entry`);
      }
      if (!relation.justification) flag(`relation to ${relation.target} has no justification — every edge is a truth claim`);
      if ((chain.relations?.length ?? 0) > 3) flag("more than 3 relations on one entry; keep the graph conservative");
    }
  }
  return issues;
}

export function admissibleEntries(register: Register): RegisterEntry[] {
  return register.entries.filter(entry => entry.chain.intent === "assert");
}

export function firstBatch(register: Register): RegisterEntry[] {
  return register.entries
    .filter(entry => entry.chain.firstBatch !== null && entry.chain.intent === "assert")
    .sort((a, b) => (a.chain.firstBatch as number) - (b.chain.firstBatch as number));
}

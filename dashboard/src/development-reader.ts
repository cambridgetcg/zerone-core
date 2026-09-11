import { DEVELOPMENT_CHAIN, DEVELOPMENT_GATEWAY } from "../development-profile";

type Row = Record<string, unknown>;
export function object(value: unknown, label = "record"): Row {
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error(`Missing or malformed ${label}`);
  return value as Row;
}
export function rows(value: unknown): unknown[] {
  if (value === undefined || value === null) return [];
  if (!Array.isArray(value)) throw new Error("Malformed record list");
  return value;
}
export function claimId(value: unknown): string {
  if (typeof value !== "string" || !value.length || new TextEncoder().encode(value).length > 256 || new TextDecoder("utf-8", { fatal: true }).decode(new TextEncoder().encode(value)) !== value) throw new Error("Claim ID must contain 1–256 valid UTF-8 bytes");
  return value;
}
export function queryClaimId(value: unknown): string {
  if (typeof value !== "string" || !/^[a-f0-9]{32}$/u.test(value)) throw new Error("This development gateway accepts generated claim IDs: 32 lowercase hexadecimal characters");
  return value;
}
export function height(value: unknown): string {
  if (typeof value === "number" && Number.isSafeInteger(value) && value >= 0) return String(value);
  if (typeof value === "string" && /^(0|[1-9][0-9]*)$/u.test(value) && BigInt(value) <= 18446744073709551615n) return value;
  throw new Error("Malformed observed height");
}

/** Refuse identity mismatches and oversized view scopes rather than show partial history. */
export function validateHistory(value: unknown, requestedId: string): Row {
  const response = object(value, "history response");
  if (response.chain_id !== DEVELOPMENT_CHAIN || height(response.block_height) === "0") throw new Error("History response has the wrong chain or height");
  let count = 0;
  const bounded = (value: unknown): unknown[] => { const result = rows(value); count += result.length; if (count > 2000) throw new Error("History exceeds the readable record budget; use the bounded CLI query"); return result; };
  function checkRecord(value: unknown, expected?: string): void {
    const record = object(value, "claim history record");
    const id = claimId(record.claim_id);
    if (expected !== undefined && id !== expected) throw new Error("History claim ID mismatch");
    if (record.claim != null && object(record.claim, "claim").id !== id) throw new Error("Stored claim identity mismatch");
    for (const item of bounded(record.rounds)) {
      const round = object(item, "round");
      if (round.claim_id !== id) throw new Error("Round claim identity mismatch");
      claimId(round.id);
      bounded(round.commits).forEach((v) => object(v, "commit"));
      bounded(round.reveals).forEach((v) => { const reveal = object(v, "reveal"); if (reveal.attestation != null) object(reveal.attestation, "attestation"); });
    }
    for (const item of bounded(record.facts)) {
      const entry = object(item, "derived fact"); const fact = object(entry.fact, "fact");
      if (fact.claim_id !== id) throw new Error("Fact claim identity mismatch");
      claimId(fact.id);
      for (const direction of ["outgoing_relations", "incoming_relations"]) for (const relation of bounded(entry[direction])) {
        if (object(relation)[direction === "outgoing_relations" ? "source_fact_id" : "target_fact_id"] !== fact.id) throw new Error("Relation fact identity mismatch");
      }
      for (const transition of bounded(entry.status_transitions)) if (object(transition).fact_id !== fact.id) throw new Error("Status history identity mismatch");
    }
    bounded(record.missing_round_ids).forEach(claimId);
  }
  checkRecord(response.record, claimId(requestedId));
  const seen = new Set([requestedId]);
  for (const item of bounded(response.related_claims)) {
    const related = object(item); checkRecord(related.record); const id = claimId(object(related.record).claim_id);
    if (seen.has(id)) throw new Error("Duplicate related claim"); seen.add(id);
    for (const link of bounded(related.links)) {
      const row = object(link);
      if (!["provisional_fact_id", "challenged_claim_id", "relations.contradicts"].includes(String(row.field))) throw new Error("Unknown relationship field");
      claimId(row.target_id);
    }
  }
  return response;
}

export interface DescriptorPins { descriptor: string; genesis: string; rpcGenesis: string; source: string; binary: string }
export async function verifyDescriptor(bytes: Uint8Array, pins: DescriptorPins): Promise<Row> {
  const digest = Array.from(new Uint8Array(await crypto.subtle.digest("SHA-256", Uint8Array.from(bytes)))).map((v) => v.toString(16).padStart(2, "0")).join("");
  if (digest !== pins.descriptor) throw new Error("Network descriptor differs from the published SHA-256; stop and check the network notice");
  const value = object(JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)), "network descriptor");
  if (value.schema !== "zerone-shared-development/v1" || value.chain_id !== DEVELOPMENT_CHAIN || value.rpc_url !== DEVELOPMENT_GATEWAY || value.genesis_url !== `${DEVELOPMENT_GATEWAY}/genesis.json` || value.faucet_url !== `${DEVELOPMENT_GATEWAY}/faucet` || value.genesis_sha256 !== pins.genesis || value.rpc_genesis_sha256 !== pins.rpcGenesis || value.source_commit !== pins.source || value.runtime_binary_sha256 !== pins.binary || value.bootstrap_consensus !== "single-operator" || value.reset_policy !== "new-chain-id" || value.knowledge_version !== 10 || value.commitment_scheme !== 2 || value.review_policy_version !== 1 || value.local_test !== false || value.denom !== "uzrn" || value.gas_limit !== 2000000 || value.tx_fee_uzrn !== "2000000") throw new Error("Network descriptor identity or policy mismatch");
  return value;
}

export async function getBytes(url: string, limit: number, signal: AbortSignal, fetcher: typeof fetch = fetch): Promise<Uint8Array> {
  const target = new URL(url);
  if (target.origin !== DEVELOPMENT_GATEWAY || target.username || target.password || target.search || target.hash || !(target.pathname === "/network.json" || /^\/claims\/[^/]+$/u.test(target.pathname))) throw new Error("Unexpected development read endpoint");
  const response = await fetcher(url, { method: "GET", credentials: "omit", redirect: "error", cache: "no-store", referrerPolicy: "no-referrer", signal, headers: { Accept: "application/json" } });
  if (!response.ok) throw new Error(`Development endpoint returned HTTP ${response.status}; no history is shown`);
  if (!/^application\/json(?:\s*;|$)/iu.test(response.headers.get("Content-Type") ?? "")) throw new Error("Expected a JSON response");
  const declared = response.headers.get("Content-Length");
  if (declared !== null && (!/^\d+$/u.test(declared) || Number(declared) > limit)) throw new Error("Development response exceeds the byte limit");
  if (!response.body) throw new Error("Empty development response");
  const reader = response.body.getReader(); const chunks: Uint8Array[] = []; let size = 0;
  try {
    while (true) { const next = await reader.read(); if (next.done) break; size += next.value.length; if (size > limit) throw new Error("Development response exceeds the byte limit"); chunks.push(next.value); }
  } catch (error) { await reader.cancel(); throw error; } finally { reader.releaseLock(); }
  const result = new Uint8Array(size); let offset = 0; for (const chunk of chunks) { result.set(chunk, offset); offset += chunk.length; } return result;
}

const ENUMS: Record<string, readonly string[]> = {
  CLAIM_STATUS: ["UNSPECIFIED", "PENDING", "PENDING_EVALUATION", "EVALUATED", "PROVISIONAL", "IN_VERIFICATION", "ACCEPTED", "REJECTED", "CHALLENGED", "EXPIRED", "INSUFFICIENT", "CONTESTED", "MALFORMED"],
  VERIFICATION_PHASE: ["UNSPECIFIED", "COMMIT", "REVEAL", "AGGREGATION", "COMPLETE", "EXPIRED"],
  VERDICT: ["UNSPECIFIED", "ACCEPT", "REJECT", "INCONCLUSIVE", "MALFORMED"],
  FACT_STATUS: ["UNSPECIFIED", "PENDING", "PROVISIONAL", "VERIFIED", "ACTIVE", "CONTESTED", "CHALLENGED", "SUPERSEDED", "EXPIRED", "DISPROVEN", "REVOKED", "AT_RISK", "PRUNED"],
  RELATION_TYPE: ["UNSPECIFIED", "SUPPORTS", "CONTRADICTS", "REQUIRES", "REFINES", "GENERALIZES", "SUPERSEDES", "CITES", "REFORMULATES"],
  INFERENCE_TYPE: ["UNSPECIFIED", "DEDUCTIVE", "INDUCTIVE", "ABDUCTIVE", "EMPIRICAL", "ANALOGICAL", "CITATION"],
};
export function enumLabel(field: string, value: unknown): string {
  const choices = ENUMS[field]; if (!choices) throw new Error("Unknown enum field");
  if (value === undefined || value === null) return "Not recorded (default unspecified)";
  if ((typeof value === "number" && Number.isInteger(value)) || (typeof value === "string" && /^\d+$/u.test(value))) return choices[Number(value)]?.replaceAll("_", " ") ?? `Unknown ${field}: ${String(value)}`;
  if (typeof value === "string") { const stripped = value.replace(new RegExp(`^${field}_`, "u"), ""); if (choices.includes(stripped)) return stripped.replaceAll("_", " "); }
  return `Unknown ${field}: ${String(value)}`;
}

const node = (tag: string, text?: string, className?: string): HTMLElement => { const element = document.createElement(tag); if (text !== undefined) element.textContent = text; if (className) element.className = className; return element; };
const display = (value: unknown): string => value === undefined || value === null || value === "" ? "Not recorded" : typeof value === "string" || typeof value === "number" || typeof value === "boolean" ? String(value) : JSON.stringify(value);
function field(parent: HTMLElement, label: string, value: unknown): void { parent.append(node("p", label, "record-label"), node("p", display(value), "record-value")); }
function details(parent: HTMLElement, title: string): HTMLElement { const element = node("details"); element.append(node("summary", title)); parent.append(element); return element; }

export function renderHistory(response: Row): DocumentFragment {
  const fragment = document.createDocumentFragment();
  function recordView(record: Row, title: string): void {
    const card = node("article", undefined, "history-card"); card.append(node("h3", title)); field(card, "Claim ID", record.claim_id);
    if (record.claim == null) card.append(node("p", "The original claim record is missing from retained state. Surviving rounds and facts do not reconstruct it."));
    else {
      const claim = object(record.claim); field(card, "Author / submitter", claim.submitter); field(card, "Claim", claim.fact_content); field(card, "Claim status", enumLabel("CLAIM_STATUS", claim.status));
      for (const [key, label] of [["method_id", "Method"], ["reasoning_trace", "Reasoning"], ["argument_text", "Argument"], ["rebuttal_text", "Rebuttal"], ["evidence_ids", "Evidence references"], ["references", "References"], ["challenged_claim_id", "Challenged claim"], ["provisional_fact_id", "Provisional / challenged fact"]]) field(card, label!, claim[key!]);
      field(card, "Submitted at block", claim.submitted_at_block);
    }
    const rounds = rows(record.rounds);
    if (!rounds.length) card.append(node("p", "No retained rounds returned for this claim."));
    for (const value of rounds) {
      const round = object(value); const panel = details(card, `Review round ${display(round.id)} · ${enumLabel("VERIFICATION_PHASE", round.phase)}`);
      field(panel, "Recorded verdict", enumLabel("VERDICT", round.verdict)); field(panel, "Commit deadline (block)", round.commit_deadline); field(panel, "Reveal deadline (block)", round.reveal_deadline); field(panel, "Commitment scheme / review policy", `${display(round.commitment_scheme)} / ${display(round.review_policy_version)}`);
      field(panel, "Retained commits / revealed reviews", `${rows(round.commits).length} / ${rows(round.reveals).length}`);
      if (!rows(round.reveals).length) panel.append(node("p", "No revealed reviews returned. A commitment alone does not disclose a vote or a reason."));
      for (const item of rows(round.reveals)) {
        const review = object(item); const reviewPanel = details(panel, `Reviewer ${display(review.verifier)} · ${display(review.vote)}`);
        field(reviewPanel, "Revealed at block", review.revealed_at_block); field(reviewPanel, "Declared confidence (parts per million, not measured accuracy)", round.commitment_scheme === 2 || round.commitment_scheme === "2" ? review.confidence ?? 0 : review.confidence ?? "Not recorded for this legacy review");
        if (review.attestation == null) reviewPanel.append(node("p", "No reasoned attestation retained for this review."));
        else for (const [key, label] of [["reason", "Reason / check performed"], ["scope", "Scope and limits"], ["method_id", "Method"], ["evidence_ids", "Evidence references"]]) field(reviewPanel, label!, object(review.attestation)[key!]);
      }
      if (round.verifier_reward_settlement != null) { const plan = object(round.verifier_reward_settlement); field(panel, "Verifier payment record", plan.paid_at_block && plan.paid_at_block !== "0" ? `Recorded paid at block ${display(plan.paid_at_block)}` : "Pending recorded settlement"); panel.append(node("p", "This is a retained settlement record, not an independently verified bank transfer or evidence that the verdict is true.")); }
    }
    const facts = rows(record.facts); if (!facts.length) card.append(node("p", "No derived facts returned for this claim."));
    for (const item of facts) {
      const entry = object(item); const fact = object(entry.fact); const panel = details(card, `Derived fact ${display(fact.id)} · ${enumLabel("FACT_STATUS", fact.status)}`);
      field(panel, "Content", fact.content); field(panel, "Submitter", fact.submitter);
      for (const direction of ["outgoing_relations", "incoming_relations"]) {
        panel.append(node("h4", direction === "outgoing_relations" ? "Outgoing canonical relations" : "Incoming canonical relations"));
        const relations = rows(entry[direction]); if (!relations.length) panel.append(node("p", "None returned."));
        for (const item of relations) { const rel = object(item); field(panel, "Relation", `${display(rel.source_fact_id)} → ${enumLabel("RELATION_TYPE", rel.relation)} → ${display(rel.target_fact_id)}`); field(panel, "Method / inference", `${display(rel.method_id)} / ${enumLabel("INFERENCE_TYPE", rel.inference)}`); }
      }
      const transitions = rows(entry.status_transitions); panel.append(node("h4", "Retained status transitions"));
      if (!transitions.length) panel.append(node("p", "No transitions returned. This does not prove that the status never changed."));
      for (const item of transitions) { const transition = object(item); field(panel, `Block ${display(transition.block_height)}`, `${enumLabel("FACT_STATUS", transition.prior_status)} → ${enumLabel("FACT_STATUS", transition.new_status)} · ${display(transition.cause_id)}`); }
    }
    if (rows(record.missing_round_ids).length) field(card, "Referenced rounds missing from retained state", record.missing_round_ids);
    fragment.append(card);
  }
  recordView(object(response.record), "Requested claim");
  for (const item of rows(response.related_claims)) { const related = object(item); recordView(object(related.record), "Directly related claim / challenge"); const last = fragment.lastElementChild as HTMLElement; for (const value of rows(related.links)) { const link = object(value); field(last, "Literal relationship", `${display(link.field)} → ${display(link.target_id)}`); } }
  const raw = node("details", undefined, "history-card"); raw.append(node("summary", "Raw endpoint response")); raw.addEventListener("toggle", () => { if (raw.hasAttribute("open") && !raw.querySelector("pre")) raw.append(node("pre", JSON.stringify(response, null, 2))); }); fragment.append(raw);
  return fragment;
}

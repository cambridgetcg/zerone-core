/** Local research/native handoff. Consistency checks do not authenticate a chain observation. */
import { DEVELOPMENT_CHAIN } from "../development-profile";
import { height, object, queryClaimId, validateHistory } from "./development-reader";
import { canonicalJSON, EXPORT_LIMIT, parseStrictJSON, sha256, validateExport, type Entry, type Journal } from "./research-review-journal";

export const WORKFLOW_LIMIT = 16 * 1024 * 1024;
export const MAX_OBSERVATIONS = 32;
const utf8 = new TextEncoder();
const HASH = /^[a-f0-9]{64}$/u;
const CLIENTS = ["shared-claims.py", "claim-workflow.py", "local-node.py"] as const;
type Row = Record<string, unknown>;
export interface WorkflowIntent { content: string; reasoning: string; domain: string; category: string; method: string }
export interface ResearchWorkflow {
  schema: "zerone-research-workflow/v1";
  journal: { export_utf8: string; export_sha256: string; collection_id: string; head_sha256: string; record_id: string; entry_sha256: string };
  intent: WorkflowIntent;
  intent_sha256: string;
  participant: null | { chain_id: string; address: string; descriptor_sha256: string; genesis_sha256: string; binary_sha256: string; client_files: Record<string, string>; local_test: boolean };
  submission: null | { state: "started" | "observed"; receipt: Row | null };
  observations: { observed_at: string; history_utf8: string; history_sha256: string }[];
}
export interface ValidatedResearchWorkflow { packet: ResearchWorkflow; journal: Journal; entry: Entry; history?: Row }

function require(condition: unknown, message: string): asserts condition { if (!condition) throw new Error(message); }
function keys(value: Row, names: readonly string[], optional: readonly string[] = []): void {
  require(names.every(name => Object.hasOwn(value, name)) && Object.keys(value).every(name => names.includes(name) || optional.includes(name)), "Missing or unsupported workflow fields");
}
function text(value: unknown, min: number, max: number, name: string): string {
  require(typeof value === "string", `Invalid ${name}`);
  const bytes = utf8.encode(value);
  require(bytes.length >= min && bytes.length <= max && !value.includes("\0") && new TextDecoder("utf-8", { fatal: true }).decode(bytes) === value && (min === 0 || value.trim().length > 0), `Invalid ${name}`);
  return value;
}
function hash(value: unknown): string { require(typeof value === "string" && HASH.test(value), "Invalid workflow SHA-256"); return value; }
function intent(value: unknown): WorkflowIntent {
  const row = object(value, "claim intent"); keys(row, ["content", "reasoning", "domain", "category", "method"]);
  return { content: text(row.content, 20, 1000, "claim content"), reasoning: text(row.reasoning, 1, 8192, "reasoning"), domain: text(row.domain, 1, 128, "domain"), category: text(row.category, 1, 128, "category"), method: text(row.method, 0, 128, "method") };
}
async function intentHash(packet: Pick<ResearchWorkflow, "journal" | "intent">): Promise<string> {
  const { export_sha256, record_id, entry_sha256 } = packet.journal;
  return sha256(canonicalJSON({ export_sha256, record_id, entry_sha256, intent: packet.intent }));
}
export function boundReasoning(packet: ResearchWorkflow): string {
  const { export_sha256, record_id, entry_sha256 } = packet.journal;
  return packet.intent.reasoning + "\n\nZerone research binding: " + canonicalJSON({ schema: "zerone-research-binding/v1", intent_sha256: packet.intent_sha256, export_sha256, record_id, entry_sha256 });
}
function publicReceipt(value: unknown): Row {
  const receipt = object(value, "public submission receipt");
  keys(receipt, ["id", "action", "actor", "status", "txhash"], ["height", "code", "gas_wanted", "gas_used", "claim_id", "round_id", "tx_bytes", "review_fee_uzrn", "collateral_uzrn"]);
  queryClaimId(receipt.id);
  require(receipt.action === "submit" && receipt.actor === "participant", "Receipt is not a participant claim submission");
  require(typeof receipt.txhash === "string" && /^[A-F0-9]{64}$/u.test(receipt.txhash), "Invalid transaction hash");
  require(["prepared", "broadcasting", "pending", "rejected", "committed", "committed-unresolved", "failed"].includes(String(receipt.status)), "Unknown submission status");
  for (const name of ["height", "code", "gas_wanted", "gas_used", "tx_bytes", "review_fee_uzrn", "collateral_uzrn"]) if (receipt[name] !== undefined) height(receipt[name]);
  for (const name of ["review_fee_uzrn", "collateral_uzrn"]) if (receipt[name] !== undefined) require(typeof receipt[name] === "string", "Receipt amounts must be exact decimal strings");
  if (receipt.claim_id !== undefined) queryClaimId(receipt.claim_id);
  if (receipt.round_id !== undefined) queryClaimId(receipt.round_id);
  if (receipt.status === "committed") require(height(receipt.height) !== "0" && height(receipt.code) === "0" && receipt.claim_id !== undefined, "Successful receipt lacks committed inclusion or a claim ID");
  return receipt;
}
export function workflowClaimId(packet: ResearchWorkflow): string | null {
  const receipt = packet.submission?.receipt;
  if (packet.submission?.state !== "observed" || !receipt || receipt.status !== "committed") return null;
  publicReceipt(receipt);
  return queryClaimId(receipt.claim_id);
}
/** Describe the retained submission record without inferring a live outcome or payment. */
export function workflowSubmissionSummary(packet: ResearchWorkflow): string {
  const id = workflowClaimId(packet);
  if (id) return `Submission receipt identifies claim ${id}. The receipt is a local record; read native history to compare the retained claim.`;
  const submission = packet.submission;
  if (submission === null) return "Prepared claim intent; no submission is recorded in this packet. Use the explicit workflow adapter when ready to submit.";
  if (submission.state === "started" && submission.receipt === null) return "Submission was started, but this packet has no client receipt. Refresh through the workflow adapter first; use resume only if the refreshed packet still has a started submission without a receipt.";
  require(submission.state === "observed", "Invalid submission state");
  const receipt = publicReceipt(submission.receipt);
  switch (receipt.status) {
    case "prepared": return "Client receipt reports prepared. Refresh through the workflow adapter and inspect the existing client receipt before taking another submission action.";
    case "broadcasting": return "Client receipt reports broadcasting. Refresh through the workflow adapter to check the recorded outcome before taking another submission action.";
    case "pending": return "Client receipt reports pending. Refresh through the workflow adapter to check for an updated transaction result.";
    case "rejected": return "Client receipt reports rejected. Inspect the retained client receipt before deciding whether a new submission is appropriate.";
    case "failed": return "Client receipt reports failed. Inspect the recorded transaction result before deciding whether a new submission is appropriate.";
    case "committed-unresolved": return "Client receipt reports committed-unresolved; this packet does not yet identify a successful claim. Refresh through the workflow adapter to resolve the receipt before reading native claim history.";
    default: throw new Error("Unknown submission status");
  }
}
export function validateBoundHistory(packet: ResearchWorkflow, value: unknown): Row {
  const id = workflowClaimId(packet);
  require(id && packet.participant, "A successful bound submission is required before recording native history");
  const history = validateHistory(value, id);
  const claim = object(object(history.record).claim, "submitted claim");
  require(claim.submitter === packet.participant.address, "History submitter differs from the bound participant");
  require(claim.fact_content === packet.intent.content && claim.domain === packet.intent.domain && claim.category === packet.intent.category && (claim.method_id ?? "") === packet.intent.method && (claim.reasoning_trace ?? "") === boundReasoning(packet), "Native claim differs from the prepared research intent or journal binding");
  const receipt = packet.submission!.receipt!;
  require(height(claim.submitted_at_block) === height(receipt.height) && BigInt(height(history.block_height)) >= BigInt(height(receipt.height)), "Native history predates or differs from the submission height");
  if (receipt.round_id !== undefined) require(claim.verification_round_id === receipt.round_id, "Submission round differs from native claim");
  const block = BigInt(height(history.block_height));
  const records = [object(history.record), ...(history.related_claims as Row[] ?? []).map(row => object(row.record))];
  for (const record of records) for (const round of (record.rounds as Row[] ?? [])) {
    for (const name of ["verifier_reward_settlement", "claim_refund_settlement"]) if (round[name] != null) {
      const plan = object(round[name]);
      require(BigInt(height(plan.created_at_block)) <= block && BigInt(height(plan.paid_at_block ?? "0")) <= block, "Settlement observation has a future height");
    }
  }
  return history;
}
export async function validateWorkflow(raw: string): Promise<ValidatedResearchWorkflow> {
  require(typeof raw === "string" && utf8.encode(raw).length > 0 && utf8.encode(raw).length <= WORKFLOW_LIMIT, "Research workflow must be nonempty and at most 16 MiB");
  const value = object(parseStrictJSON(raw), "research workflow");
  keys(value, ["schema", "journal", "intent", "intent_sha256", "participant", "submission", "observations"]);
  require(value.schema === "zerone-research-workflow/v1", "Unsupported research workflow schema");
  const source = object(value.journal, "journal binding");
  keys(source, ["export_utf8", "export_sha256", "collection_id", "head_sha256", "record_id", "entry_sha256"]);
  const sourceText = text(source.export_utf8, 1, EXPORT_LIMIT, "journal export");
  require(await sha256(sourceText) === hash(source.export_sha256), "Exact journal export SHA-256 mismatch");
  const journal = await validateExport(utf8.encode(sourceText));
  const entry = journal.entries.find(row => row.record.id === source.record_id);
  require(entry?.record.kind === "contribution", "Select an existing contribution before preparing native admission");
  require(source.collection_id === journal.header.collection_id && source.head_sha256 === journal.entries.at(-1)?.sha256 && source.entry_sha256 === entry.sha256, "Journal record or head binding mismatch");
  intent(value.intent);
  const packet = value as unknown as ResearchWorkflow;
  require(await intentHash(packet) === hash(value.intent_sha256), "Prepared intent SHA-256 mismatch");
  text(boundReasoning(packet), 1, 8192, "bound reasoning including journal binding");
  if (value.participant !== null) {
    const participant = object(value.participant, "participant binding");
    keys(participant, ["chain_id", "address", "descriptor_sha256", "genesis_sha256", "binary_sha256", "client_files", "local_test"]);
    require(participant.chain_id === DEVELOPMENT_CHAIN && typeof participant.address === "string" && /^zrn1[023456789acdefghjklmnpqrstuvwxyz]{38}$/u.test(participant.address) && typeof participant.local_test === "boolean", "Invalid participant or network binding");
    for (const name of ["descriptor_sha256", "genesis_sha256", "binary_sha256"]) hash(participant[name]);
    const clients = object(participant.client_files, "pinned client files"); keys(clients, CLIENTS); CLIENTS.forEach(name => hash(clients[name]));
  }
  if (value.submission !== null) {
    require(value.participant !== null, "Submission is missing its participant binding");
    const submission = object(value.submission, "submission state"); keys(submission, ["state", "receipt"]);
    require(submission.state === "started" || submission.state === "observed", "Unknown submission state");
    if (submission.state === "started") require(submission.receipt === null, "An unresolved start cannot claim an observed receipt");
    else publicReceipt(submission.receipt);
  } else require(value.participant === null, "An unsubmitted draft cannot carry a participant binding");
  require(Array.isArray(value.observations) && value.observations.length <= MAX_OBSERVATIONS, "Workflow exceeds its 32 retained observations or is malformed");
  let latest: Row | undefined;
  let lastTime = "";
  const seen = new Set<string>();
  for (const item of value.observations) {
    const observation = object(item, "native observation"); keys(observation, ["observed_at", "history_utf8", "history_sha256"]);
    const date = text(observation.observed_at, 24, 24, "observation timestamp");
    require(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$/u.test(date) && !date.startsWith("0000") && Number.isFinite(Date.parse(date)) && new Date(date).toISOString() === date, "Invalid observation timestamp");
    require(date >= lastTime, "Retained observation timestamp regressed"); lastTime = date;
    const historyText = text(observation.history_utf8, 1, EXPORT_LIMIT, "retained history");
    require(await sha256(historyText) === hash(observation.history_sha256), "Retained history SHA-256 mismatch");
    require(!seen.has(observation.history_sha256 as string), "Duplicate retained native observation"); seen.add(observation.history_sha256 as string);
    const observed = validateBoundHistory(packet, parseStrictJSON(historyText));
    require(!latest || BigInt(height(observed.block_height)) >= BigInt(height(latest.block_height)), "Retained native observation height regressed");
    latest = observed;
  }
  return { packet, journal, entry, ...(latest ? { history: latest } : {}) };
}
/** Open either format while keeping the complete journal's exact bytes and hash. */
export async function validateResearchFile(bytes: Uint8Array): Promise<{ journal: Journal; bytes: Uint8Array; hash: string; workflow?: ValidatedResearchWorkflow }> {
  require(bytes.length > 0 && bytes.length <= WORKFLOW_LIMIT, "Research file must be nonempty and at most 16 MiB");
  const raw = new TextDecoder("utf-8", { fatal: true, ignoreBOM: true }).decode(bytes);
  const value = object(parseStrictJSON(raw), "research file");
  if (value.schema === "zerone-research-export/v1") {
    const [journal, hash] = await Promise.all([validateExport(bytes), sha256(raw)]);
    return { journal, bytes, hash };
  }
  if (value.schema === "zerone-research-workflow/v1") {
    const workflow = await validateWorkflow(raw);
    return { journal: workflow.journal, bytes: utf8.encode(workflow.packet.journal.export_utf8), hash: workflow.packet.journal.export_sha256, workflow };
  }
  throw new Error("Unsupported research file schema");
}
export async function prepareWorkflow(journalUtf8: string, recordId: string, claim: WorkflowIntent): Promise<ValidatedResearchWorkflow> {
  const journal = await validateExport(utf8.encode(journalUtf8));
  const entry = journal.entries.find(row => row.record.id === recordId);
  require(entry?.record.kind === "contribution", "Select an existing contribution");
  const packet: ResearchWorkflow = {
    schema: "zerone-research-workflow/v1",
    journal: { export_utf8: journalUtf8, export_sha256: await sha256(journalUtf8), collection_id: journal.header.collection_id, head_sha256: journal.entries.at(-1)!.sha256, record_id: recordId, entry_sha256: entry.sha256 },
    intent: intent(claim), intent_sha256: "", participant: null, submission: null, observations: [],
  };
  packet.intent_sha256 = await intentHash(packet);
  return validateWorkflow(JSON.stringify(packet));
}
export async function appendWorkflowObservation(packet: ResearchWorkflow, historyUtf8: string, observedAt = new Date().toISOString()): Promise<ValidatedResearchWorkflow> {
  const checked = await validateWorkflow(JSON.stringify(packet));
  const current = checked.packet;
  const historySha = await sha256(historyUtf8);
  if (current.observations.some(row => row.history_sha256 === historySha)) return checked;
  require(current.observations.length < MAX_OBSERVATIONS, "All 32 observations are retained; export this packet before starting another observation record");
  current.observations.push({ observed_at: observedAt, history_utf8: historyUtf8, history_sha256: historySha });
  return validateWorkflow(JSON.stringify(current));
}

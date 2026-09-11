import {
  QueryClaimHistoryRequest,
  QueryClaimHistoryResponse,
  type ClaimHistoryRecord,
} from "./generated/zerone/knowledge/v1/claim_history";

export type { QueryClaimHistoryResponse, ClaimHistoryRecord } from "./generated/zerone/knowledge/v1/claim_history";

/** Compatible with CosmJS ProtobufRpcClient; the caller owns endpoint and context selection. */
export interface ClaimHistoryRpc {
  request(service: string, method: string, data: Uint8Array): Promise<Uint8Array>;
}

export interface ClaimHistoryOptions {
  expectedChainId?: string;
  /** Must match the selected SDK query context; this option does not load historical state. */
  expectedHeight?: bigint;
  /** At most the server's 8 MiB protobuf output ceiling. Transport limits remain the caller's responsibility. */
  maximumResponseBytes?: number;
}

const maximumBytes = 8 * 1024 * 1024;
const maximumHeight = (1n << 64n) - 1n;
const utf8 = new TextEncoder();
const strictUtf8 = new TextDecoder("utf-8", { fatal: true });

function requireValue(condition: unknown, message: string): asserts condition {
  if (!condition) throw new Error(`ClaimHistory: ${message}`);
}

function identifier(value: string, field: string): void {
  requireValue(typeof value === "string", `${field} must be a string`);
  const bytes = utf8.encode(value);
  requireValue(bytes.length > 0 && bytes.length <= 256 && strictUtf8.decode(bytes) === value,
    `${field} must be nonempty valid UTF-8 of at most 256 bytes`);
}

function validateRecord(record: ClaimHistoryRecord | undefined): asserts record is ClaimHistoryRecord {
  requireValue(record && record.claimId !== "", "missing record identity");
  if (record.claim) requireValue(record.claim.id === record.claimId, "embedded claim identity mismatch");
  const rounds = new Set<string>();
  for (const round of record.rounds) {
    requireValue(round.id !== "" && !rounds.has(round.id) && round.claimId === record.claimId,
      "duplicate or inconsistent round identity");
    rounds.add(round.id);
  }
  const missing = new Set<string>();
  for (const id of record.missingRoundIds) {
    requireValue(id !== "" && !missing.has(id) && !rounds.has(id), "inconsistent missing round identity");
    missing.add(id);
  }
  const facts = new Set<string>();
  for (const row of record.facts) {
    const fact = row.fact;
    requireValue(fact && fact.id !== "" && !facts.has(fact.id) && fact.claimId === record.claimId,
      "duplicate or inconsistent fact identity");
    facts.add(fact.id);
    requireValue(row.outgoingRelations.every(edge => edge.sourceFactId === fact.id),
      "outgoing relation identity mismatch");
    requireValue(row.incomingRelations.every(edge => edge.targetFactId === fact.id),
      "incoming relation identity mismatch");
    requireValue(row.statusTransitions.every(transition => transition.factId === fact.id),
      "status transition identity mismatch");
  }
}

/**
 * Reads one source-compatible retained-record observation. Chain/height labels
 * and matching record IDs are consistency checks, not authenticated state,
 * transaction inclusion, complete historical retention, or scientific truth.
 * The caller supplies transport timeouts/byte limits and any proof verification.
 * Unknown or noncanonical protobuf bytes are refused rather than silently lost.
 */
export async function queryClaimHistory(
  rpc: ClaimHistoryRpc,
  claimId: string,
  options: ClaimHistoryOptions = {},
): Promise<QueryClaimHistoryResponse> {
  identifier(claimId, "claim ID");
  requireValue(rpc && typeof rpc.request === "function", "a protobuf RPC transport is required");
  const expectedChainId = options.expectedChainId;
  if (expectedChainId !== undefined) identifier(expectedChainId, "expected chain ID");
  const height = options.expectedHeight;
  requireValue(height === undefined || (typeof height === "bigint" && height > 0n && height <= maximumHeight),
    "expected height must be a positive uint64 bigint");
  const limit = options.maximumResponseBytes ?? maximumBytes;
  requireValue(Number.isInteger(limit) && limit >= 1 && limit <= maximumBytes,
    "maximum response bytes must be between 1 and 8388608");
  const request = QueryClaimHistoryRequest.fromPartial({ id: claimId, atBlockHeight: height ?? 0n });
  const bytes = await rpc.request("zerone.knowledge.v1.Query", "ClaimHistory",
    QueryClaimHistoryRequest.encode(request).finish());
  requireValue(bytes instanceof Uint8Array && bytes.byteLength > 0 && bytes.byteLength <= limit,
    "response is empty, invalid, or exceeds the byte limit");
  const response = QueryClaimHistoryResponse.decode(bytes);
  const encoded = QueryClaimHistoryResponse.encode(response).finish();
  requireValue(encoded.length === bytes.length && encoded.every((byte, index) => byte === bytes[index]),
    "unsupported or noncanonical protobuf response");
  identifier(response.chainId, "response chain ID");
  requireValue(response.blockHeight >= 0n && response.blockHeight <= maximumHeight, "invalid response height");
  if (height !== undefined) requireValue(response.blockHeight === height, "response height mismatch");
  if (expectedChainId !== undefined) {
    requireValue(response.chainId === expectedChainId, "response chain ID mismatch");
  }
  validateRecord(response.record);
  requireValue(response.record.claimId === claimId, "requested root claim identity mismatch");
  const relatedIds = new Set([claimId]);
  const linkFields = new Set(["provisional_fact_id", "challenged_claim_id", "relations.contradicts"]);
  for (const related of response.relatedClaims) {
    validateRecord(related.record);
    requireValue(!relatedIds.has(related.record.claimId), "duplicate related claim identity");
    relatedIds.add(related.record.claimId);
    requireValue(related.links.length > 0 && related.links.every(link => linkFields.has(link.field) && link.targetId !== ""),
      "missing or unsupported related-claim link");
  }
  return response;
}

import { QueryClaimHistoryResponse } from "./generated/zerone/knowledge/v1/claim_history.js";
export type { QueryClaimHistoryResponse, ClaimHistoryRecord } from "./generated/zerone/knowledge/v1/claim_history.js";
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
/**
 * Reads one source-compatible retained-record observation. Chain/height labels
 * and matching record IDs are consistency checks, not authenticated state,
 * transaction inclusion, complete historical retention, or scientific truth.
 * The caller supplies transport timeouts/byte limits and any proof verification.
 * Unknown or noncanonical protobuf bytes are refused rather than silently lost.
 */
export declare function queryClaimHistory(rpc: ClaimHistoryRpc, claimId: string, options?: ClaimHistoryOptions): Promise<QueryClaimHistoryResponse>;

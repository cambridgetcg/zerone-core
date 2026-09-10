import type { ReviewAttestation } from "./generated/zerone/knowledge/v1/types.js";
import { MsgSubmitReveal } from "./generated/zerone/knowledge/v1/tx.js";
export interface ReviewCommitmentV2 {
    /** The immutable commitment_chain_id returned by the round, including after import. */
    chainId: string;
    roundId: string;
    verifier: string;
    vote: "accept" | "reject" | "malformed";
    confidence: bigint;
    salt: Uint8Array;
    attestation: ReviewAttestation;
}
/**
 * Exact scheme-2 hash shared with x/knowledge/types.ComputeReviewCommitmentV2.
 * This binds a signer's assertion about a review; it proves neither independence
 * nor that the described work was performed. Never use it for scheme-0 rounds.
 */
export declare function computeReviewCommitmentV2(review: ReviewCommitmentV2): Uint8Array;
/** Construct exactly the signed reveal fields whose scheme-2 hash was checked. */
export declare function makeReviewRevealV2(review: ReviewCommitmentV2): MsgSubmitReveal;

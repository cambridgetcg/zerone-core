import { fromBech32, toBech32 } from "@cosmjs/encoding";
import { sha256 } from "@noble/hashes/sha2.js";
import type { ReviewAttestation } from "./generated/zerone/knowledge/v1/types";
import { MsgSubmitReveal } from "./generated/zerone/knowledge/v1/tx";

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

const utf8 = new TextEncoder();
const strictUtf8 = new TextDecoder("utf-8", { fatal: true });
// Go unicode.IsSpace, used by strings.TrimSpace in the consensus validator.
const onlySpace = /^[\u0009-\u000d\u0020\u0085\u00a0\u1680\u2000-\u200a\u2028\u2029\u202f\u205f\u3000]*$/u;

function textBytes(name: string, value: string, max: number, required = false): Uint8Array {
  if (typeof value !== "string") throw new Error(`${name} must be a string`);
  const bytes = utf8.encode(value);
  if (bytes.length > max || strictUtf8.decode(bytes) !== value) {
    throw new Error(`${name} must be valid UTF-8 of at most ${max} bytes`);
  }
  if (required && onlySpace.test(value)) throw new Error(`${name} must be nonempty`);
  return bytes;
}

/**
 * Exact scheme-2 hash shared with x/knowledge/types.ComputeReviewCommitmentV2.
 * This binds a signer's assertion about a review; it proves neither independence
 * nor that the described work was performed. Never use it for scheme-0 rounds.
 */
export function computeReviewCommitmentV2(review: ReviewCommitmentV2): Uint8Array {
  const chain = textBytes("commitment chain", review.chainId, 256, true);
  const round = textBytes("commitment round", review.roundId, 256, true);
  const { prefix, data: address } = fromBech32(review.verifier);
  if (prefix !== "zrn" || address.length === 0 || address.length > 255 || toBech32(prefix, address) !== review.verifier) {
    throw new Error("reviewer must be a canonical Zerone account address");
  }
  if (!["accept", "reject", "malformed"].includes(review.vote)) throw new Error("invalid review vote");
  if (typeof review.confidence !== "bigint" || review.confidence < 0n || review.confidence > 1_000_000n) {
    throw new Error("review confidence must be between 0 and 1000000");
  }
  if (!(review.salt instanceof Uint8Array) || review.salt.length < 16 || review.salt.length > 64) {
    throw new Error("scheme-2 salt must contain 16 to 64 bytes");
  }
  const att = review.attestation;
  if (att === null || typeof att !== "object") throw new Error("review attestation is required");
  const known = new Set(["methodId", "reason", "evidenceIds", "scope"]);
  if (Object.keys(att).some((key) => !known.has(key))) throw new Error("unsupported review attestation fields");
  const method = textBytes("review method", att.methodId, 128);
  const reason = textBytes("review reason", att.reason, 4096, true);
  const scope = textBytes("review scope", att.scope, 1024);
  if (!Array.isArray(att.evidenceIds) || att.evidenceIds.length > 16) throw new Error("at most 16 evidence references are allowed");
  if (new Set(att.evidenceIds).size !== att.evidenceIds.length) throw new Error("duplicate evidence reference");
  const evidence = att.evidenceIds.map((value) => textBytes("evidence reference", value, 256, true));
  if (method.length + reason.length + scope.length + evidence.reduce((n, value) => n + value.length, 0) > 8192) {
    throw new Error("review attestation exceeds 8192 bytes");
  }
  const chunks: Uint8Array[] = [utf8.encode("ZRN.review.commit.v2\0")];
  const u32 = (value: number) => {
    const bytes = new Uint8Array(4);
    new DataView(bytes.buffer).setUint32(0, value, false);
    chunks.push(bytes);
  };
  const field = (bytes: Uint8Array) => { u32(bytes.length); chunks.push(bytes); };
  for (const value of [chain, round, address, utf8.encode(review.vote)]) field(value);
  const confidence = new Uint8Array(8);
  new DataView(confidence.buffer).setBigUint64(0, review.confidence, false);
  chunks.push(confidence);
  for (const value of [review.salt, method, reason, scope]) field(value);
  u32(evidence.length);
  for (const value of evidence) field(value);
  const preimage = new Uint8Array(chunks.reduce((n, value) => n + value.length, 0));
  let offset = 0;
  for (const value of chunks) { preimage.set(value, offset); offset += value.length; }
  return sha256(preimage);
}

/** Construct exactly the signed reveal fields whose scheme-2 hash was checked. */
export function makeReviewRevealV2(review: ReviewCommitmentV2): MsgSubmitReveal {
  computeReviewCommitmentV2(review);
  return MsgSubmitReveal.fromPartial({
    verifier: review.verifier, roundId: review.roundId, vote: review.vote,
    confidence: review.confidence, salt: Uint8Array.from(review.salt),
    attestation: { ...review.attestation, evidenceIds: [...review.attestation.evidenceIds] },
  });
}

// The door request: ten LF-separated lines, all lowercase hex for binary
// values (the lobby's egress gate lets hex through and refuses key-like
// base64; nothing in a request is secret). The first six lines are the
// request proper: the exact bytes the account key signs (ADR-036) and the
// proof of work covers. The identity key signs the exact accounting-authority
// registration bytes from @zerone-chain/sdk, so the same proof is valid on
// chain once field 7 is enforced.
//
//   zerone-door-request/v1
//   chain_id: zerone-1
//   address: zrn1...
//   identity_public_key: <64 hex, Ed25519>
//   nonce: <32 hex, random>
//   issued_at: <ISO 8601 UTC, e.g. 2026-10-28T17:30:00.000Z>
//   account_public_key: <66 hex, compressed secp256k1>
//   account_signature: <128 hex, r||s over the ADR-036 sign doc>
//   identity_signature: <128 hex, Ed25519 over the registration proof bytes>
//   pow_nonce: <decimal>
import { createHash } from "node:crypto";
import { accountRegistrationProofSignBytes } from "../../../sdk/typescript/src/account-registration.ts";
import { addressPayloadHex, parseZeroneAddress } from "./bech32.ts";
import { verifyEd25519Strict } from "./ed25519.ts";
import {
  CHAIN_ID,
  POW_BITS,
  REQUEST_MAX_AGE_SECONDS,
  REQUEST_MAX_BYTES,
  REQUEST_MAX_FUTURE_SECONDS,
} from "./policy.ts";
import { accountAddressBytes, verifySecp256k1 } from "./secp256k1.ts";

export const REQUEST_FORMAT = "zerone-door-request/v1" as const;

export type RequestReason =
  | "request_malformed"
  | "chain_id_mismatch"
  | "address_not_canonical"
  | "request_stale"
  | "request_from_future"
  | "account_key_mismatch"
  | "account_signature_invalid"
  | "identity_proof_invalid"
  | "pow_insufficient";

export interface DoorRequest {
  /** The six signed lines joined by LF, exactly as received. */
  request: string;
  /** sha256 of the request bytes, lowercase hex: the request's name everywhere. */
  requestId: string;
  chainId: string;
  address: string;
  identityPublicKeyHex: string;
  did: string;
  nonceHex: string;
  issuedAt: string;
  accountPublicKeyHex: string;
  accountSignatureHex: string;
  identitySignatureHex: string;
  powNonce: string;
}

const LINE_RULES: ReadonlyArray<[string, RegExp]> = [
  ["chain_id", /^[a-z0-9][a-z0-9-]{0,47}$/],
  ["address", /^\S{1,100}$/],
  ["identity_public_key", /^[0-9a-f]{64}$/],
  ["nonce", /^[0-9a-f]{32}$/],
  ["issued_at", /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{3})?Z$/],
  ["account_public_key", /^0[23][0-9a-f]{64}$/],
  ["account_signature", /^[0-9a-f]{128}$/],
  ["identity_signature", /^[0-9a-f]{128}$/],
  ["pow_nonce", /^(?:0|[1-9][0-9]{0,19})$/],
];

export type ParseResult =
  | { ok: true; value: DoorRequest }
  | { ok: false; reason: "request_malformed"; detail: string };

export function sha256Hex(data: Uint8Array | string): string {
  return createHash("sha256").update(data).digest("hex");
}

/** Parses the wire text. Allows one trailing LF; refuses CR, tabs, extra
 * spaces, unknown or reordered lines, and anything over 2 KiB. */
export function parseDoorRequest(text: string): ParseResult {
  const bad = (detail: string): ParseResult => ({ ok: false, reason: "request_malformed", detail });
  if (typeof text !== "string") return bad("not text");
  if (Buffer.byteLength(text, "utf8") > REQUEST_MAX_BYTES) return bad("too large");
  const body = text.endsWith("\n") ? text.slice(0, -1) : text;
  if (/[^\x20-\x7e\n]/.test(body)) return bad("characters outside printable ASCII");
  const lines = body.split("\n");
  if (lines.length !== 10) return bad(`expected 10 lines, got ${lines.length}`);
  if (lines[0] !== REQUEST_FORMAT) return bad(`first line must be ${REQUEST_FORMAT}`);
  const values: string[] = [];
  for (let i = 0; i < LINE_RULES.length; i += 1) {
    const [key, rule] = LINE_RULES[i]!;
    const line = lines[i + 1]!;
    const prefix = `${key}: `;
    if (!line.startsWith(prefix)) return bad(`line ${i + 2} must start with "${prefix}"`);
    const value = line.slice(prefix.length);
    if (!rule.test(value)) return bad(`${key} has the wrong shape`);
    values.push(value);
  }
  const [chainId, address, identityPublicKeyHex, nonceHex, issuedAt, accountPublicKeyHex, accountSignatureHex,
    identitySignatureHex, powNonce] = values as [string, string, string, string, string, string, string, string, string];
  if (Number.isNaN(Date.parse(issuedAt))) return bad("issued_at is not a date");
  const request = lines.slice(0, 6).join("\n");
  return {
    ok: true,
    value: {
      request,
      requestId: sha256Hex(request),
      chainId,
      address,
      identityPublicKeyHex,
      did: `did:zrn:${identityPublicKeyHex}`,
      nonceHex,
      issuedAt,
      accountPublicKeyHex,
      accountSignatureHex,
      identitySignatureHex,
      powNonce,
    },
  };
}

/** ADR-036 (sign/MsgSignData) amino sign doc, sorted and compact, as CosmJS
 * serializeSignDoc writes it for signAmino. */
export function adr036SignBytes(signer: string, data: Uint8Array): Uint8Array {
  const b64 = Buffer.from(data).toString("base64");
  const doc =
    `{"account_number":"0","chain_id":"","fee":{"amount":[],"gas":"0"},"memo":"",` +
    `"msgs":[{"type":"sign/MsgSignData","value":{"data":"${b64}","signer":"${signer}"}}],"sequence":"0"}`;
  return new TextEncoder().encode(doc);
}

export function registrationProofBytes(chainId: string, address: string, identityPublicKey: Uint8Array): Uint8Array {
  const did = `did:zrn:${Buffer.from(identityPublicKey).toString("hex")}`;
  return accountRegistrationProofSignBytes({
    chainId,
    sender: address,
    did,
    identityPublicKey,
    accountType: "agent",
    metadata: "",
  });
}

export function leadingZeroBits(digest: Uint8Array): number {
  let bits = 0;
  for (const byte of digest) {
    if (byte === 0) {
      bits += 8;
      continue;
    }
    return bits + Math.clz32(byte) - 24;
  }
  return bits;
}

export function powDigest(request: string, powNonce: string): Uint8Array {
  return Uint8Array.from(createHash("sha256").update(`${request}\n${powNonce}`, "utf8").digest());
}

/** Every request-level check that needs no chain read. Order is stable so a
 * refusal lists its reasons the same way each time. */
export function verifyDoorRequest(req: DoorRequest, now: Date, expectedChainId: string = CHAIN_ID): RequestReason[] {
  const reasons: RequestReason[] = [];
  if (req.chainId !== expectedChainId) reasons.push("chain_id_mismatch");

  const parsed = parseZeroneAddress(req.address);
  if (!parsed.ok) reasons.push("address_not_canonical");

  const issued = Date.parse(req.issuedAt);
  const ageSeconds = (now.getTime() - issued) / 1000;
  if (ageSeconds > REQUEST_MAX_AGE_SECONDS) reasons.push("request_stale");
  if (ageSeconds < -REQUEST_MAX_FUTURE_SECONDS) reasons.push("request_from_future");

  const accountPublicKey = Uint8Array.from(Buffer.from(req.accountPublicKeyHex, "hex"));
  const derived = Buffer.from(accountAddressBytes(accountPublicKey)).toString("hex");
  if (addressPayloadHex(req.address) !== derived) reasons.push("account_key_mismatch");

  const requestBytes = new TextEncoder().encode(req.request);
  const digest = Uint8Array.from(createHash("sha256").update(adr036SignBytes(req.address, requestBytes)).digest());
  const accountSignature = Uint8Array.from(Buffer.from(req.accountSignatureHex, "hex"));
  if (!verifySecp256k1(digest, accountSignature, accountPublicKey)) reasons.push("account_signature_invalid");

  const identityPublicKey = Uint8Array.from(Buffer.from(req.identityPublicKeyHex, "hex"));
  let proofOk = false;
  if (parsed.ok) {
    try {
      const proof = registrationProofBytes(req.chainId, req.address, identityPublicKey);
      proofOk = verifyEd25519Strict(proof, Uint8Array.from(Buffer.from(req.identitySignatureHex, "hex")), identityPublicKey);
    } catch {
      proofOk = false;
    }
  }
  if (!proofOk) reasons.push("identity_proof_invalid");

  if (leadingZeroBits(powDigest(req.request, req.powNonce)) < POW_BITS) reasons.push("pow_insufficient");
  return reasons;
}

// ── request building, for the agent guide's tests and the keyless live
//    dry-run. Signing callbacks are supplied by the caller. ──

export interface UnsignedRequestFields {
  chainId: string;
  address: string;
  identityPublicKeyHex: string;
  nonceHex: string;
  issuedAt: string;
}

export function requestText(fields: UnsignedRequestFields): string {
  return [
    REQUEST_FORMAT,
    `chain_id: ${fields.chainId}`,
    `address: ${fields.address}`,
    `identity_public_key: ${fields.identityPublicKeyHex}`,
    `nonce: ${fields.nonceHex}`,
    `issued_at: ${fields.issuedAt}`,
  ].join("\n");
}

export function solvePow(request: string, bits: number = POW_BITS): string {
  for (let n = 0; ; n += 1) {
    if (leadingZeroBits(powDigest(request, String(n))) >= bits) return String(n);
  }
}

export function envelopeText(request: string, parts: {
  accountPublicKeyHex: string;
  accountSignatureHex: string;
  identitySignatureHex: string;
  powNonce: string;
}): string {
  return [
    request,
    `account_public_key: ${parts.accountPublicKeyHex}`,
    `account_signature: ${parts.accountSignatureHex}`,
    `identity_signature: ${parts.identitySignatureHex}`,
    `pow_nonce: ${parts.powNonce}`,
  ].join("\n") + "\n";
}

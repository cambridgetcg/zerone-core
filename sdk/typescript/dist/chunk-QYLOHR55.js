import {
  MsgSubmitReveal
} from "./chunk-L63AY2KW.js";

// src/auth-signing-common.ts
import { fromBech32 } from "@cosmjs/encoding";
var U32_MAX = 4294967295;
var BIP173_MAX_LENGTH = 90;
function encodeChainId(chainId) {
  const encoded = encodeText(chainId, "chain ID");
  if (chainId.length === 0 || chainId.trim() !== chainId || chainId.startsWith("\x85") || chainId.endsWith("\x85")) {
    throw new RangeError(
      "chain ID must be non-empty without surrounding whitespace"
    );
  }
  return encoded;
}
function encodeText(value, label) {
  if (hasIllFormedUtf16(value)) {
    throw new RangeError(`${label} must contain well-formed Unicode`);
  }
  const encoded = new TextEncoder().encode(value);
  if (encoded.length > U32_MAX) {
    throw new RangeError(`${label} exceeds the uint32 length bound`);
  }
  return encoded;
}
function validateZeroneAddress(address) {
  if (address !== address.toLowerCase()) {
    throw new RangeError("sender must be a canonical lowercase Zerone address");
  }
  let decoded;
  try {
    decoded = fromBech32(address, BIP173_MAX_LENGTH);
  } catch {
    throw new RangeError("sender must be a valid Zerone Bech32 address");
  }
  if (decoded.prefix !== "zrn" || decoded.data.length !== 20) {
    throw new RangeError("sender must be a 20-byte zrn account address");
  }
}
function writeUint32(output, offset, value) {
  new DataView(output.buffer, output.byteOffset, output.byteLength).setUint32(
    offset,
    value,
    false
  );
  return offset + 4;
}
function hasIllFormedUtf16(value) {
  for (let index = 0; index < value.length; index += 1) {
    const codeUnit = value.charCodeAt(index);
    if (codeUnit >= 55296 && codeUnit <= 56319) {
      const next = value.charCodeAt(index + 1);
      if (!(next >= 56320 && next <= 57343)) return true;
      index += 1;
    } else if (codeUnit >= 56320 && codeUnit <= 57343) {
      return true;
    }
  }
  return false;
}

// src/account-registration.ts
var ACCOUNT_REGISTRATION_PROOF_DOMAIN = "zerone.auth/register-account/v1";
function accountRegistrationProofSignBytes(proof) {
  const chainId = encodeChainId(proof.chainId);
  validateZeroneAddress(proof.sender);
  const sender = encodeText(proof.sender, "sender");
  if (proof.identityPublicKey.length !== 32) {
    throw new RangeError("identity public key must be 32 bytes");
  }
  const publicKeyHex = bytesToHex(proof.identityPublicKey);
  if (proof.did !== `did:zrn:${publicKeyHex}`) {
    throw new RangeError(
      "DID must be did:zrn: followed by the identity public key as 64 lowercase hex characters"
    );
  }
  const did = encodeText(proof.did, "DID");
  if (!isZeroneAccountType(proof.accountType)) {
    throw new RangeError(
      "account type must be agent, human, contract, or system"
    );
  }
  const accountType = encodeText(proof.accountType, "account type");
  const metadata = encodeText(proof.metadata, "metadata");
  const domain = new TextEncoder().encode(ACCOUNT_REGISTRATION_PROOF_DOMAIN);
  const result = new Uint8Array(
    domain.length + 1 + 4 + chainId.length + 4 + sender.length + 4 + did.length + proof.identityPublicKey.length + 4 + accountType.length + 4 + metadata.length
  );
  let offset = 0;
  result.set(domain, offset);
  offset += domain.length;
  result[offset] = 0;
  offset += 1;
  offset = writeLengthPrefixed(result, offset, chainId);
  offset = writeLengthPrefixed(result, offset, sender);
  offset = writeLengthPrefixed(result, offset, did);
  result.set(proof.identityPublicKey, offset);
  offset += proof.identityPublicKey.length;
  offset = writeLengthPrefixed(result, offset, accountType);
  writeLengthPrefixed(result, offset, metadata);
  return result;
}
function writeLengthPrefixed(output, offset, value) {
  offset = writeUint32(output, offset, value.length);
  output.set(value, offset);
  return offset + value.length;
}
function isZeroneAccountType(value) {
  return value === "agent" || value === "human" || value === "contract" || value === "system";
}
function bytesToHex(value) {
  let output = "";
  for (const byte of value) output += byte.toString(16).padStart(2, "0");
  return output;
}

// src/key-rotation.ts
var KEY_ROTATION_AUTHORIZATION_DOMAIN = "zerone.auth/rotate-key/v1";
var KEY_ROTATION_ACCEPTANCE_DOMAIN = "zerone.auth/accept-key/v1";
var KEY_ROTATION_AUTHORIZATION_MAX_TTL_SECONDS = 600n;
var I64_MAX = 0x7fffffffffffffffn;
function keyRotationAuthorizationSignBytes(authorization) {
  return keyRotationSignBytes(
    KEY_ROTATION_AUTHORIZATION_DOMAIN,
    authorization
  );
}
function keyRotationAcceptanceSignBytes(authorization) {
  return keyRotationSignBytes(KEY_ROTATION_ACCEPTANCE_DOMAIN, authorization);
}
function keyRotationSignBytes(domainName, authorization) {
  const chainId = encodeChainId(authorization.chainId);
  validateZeroneAddress(authorization.sender);
  const sender = encodeText(authorization.sender, "sender");
  if (!Number.isInteger(authorization.currentKeyVersion) || authorization.currentKeyVersion <= 0 || authorization.currentKeyVersion > U32_MAX) {
    throw new RangeError("current key version must be a positive uint32");
  }
  if (authorization.newOperationalKey.length !== 32) {
    throw new RangeError("new operational key must be 32 bytes");
  }
  if (authorization.authorizationExpiresAtUnix <= 0n || authorization.authorizationExpiresAtUnix > I64_MAX) {
    throw new RangeError(
      "authorization expiry must be a positive signed int64 Unix timestamp"
    );
  }
  const domain = new TextEncoder().encode(domainName);
  const result = new Uint8Array(
    domain.length + 1 + 4 + chainId.length + 4 + sender.length + 4 + 8 + 32
  );
  let offset = 0;
  result.set(domain, offset);
  offset += domain.length;
  result[offset] = 0;
  offset += 1;
  offset = writeUint32(result, offset, chainId.length);
  result.set(chainId, offset);
  offset += chainId.length;
  offset = writeUint32(result, offset, sender.length);
  result.set(sender, offset);
  offset += sender.length;
  offset = writeUint32(result, offset, authorization.currentKeyVersion);
  new DataView(result.buffer, result.byteOffset, result.byteLength).setBigInt64(
    offset,
    authorization.authorizationExpiresAtUnix,
    false
  );
  offset += 8;
  result.set(authorization.newOperationalKey, offset);
  return result;
}

// src/review-commitment.ts
import { fromBech32 as fromBech322, toBech32 } from "@cosmjs/encoding";
import { sha256 } from "@noble/hashes/sha2.js";
var utf8 = new TextEncoder();
var strictUtf8 = new TextDecoder("utf-8", { fatal: true });
var onlySpace = /^[\u0009-\u000d\u0020\u0085\u00a0\u1680\u2000-\u200a\u2028\u2029\u202f\u205f\u3000]*$/u;
function textBytes(name, value, max, required = false) {
  if (typeof value !== "string") throw new Error(`${name} must be a string`);
  const bytes = utf8.encode(value);
  if (bytes.length > max || strictUtf8.decode(bytes) !== value) {
    throw new Error(`${name} must be valid UTF-8 of at most ${max} bytes`);
  }
  if (required && onlySpace.test(value)) throw new Error(`${name} must be nonempty`);
  return bytes;
}
function computeReviewCommitmentV2(review) {
  const chain = textBytes("commitment chain", review.chainId, 256, true);
  const round = textBytes("commitment round", review.roundId, 256, true);
  const { prefix, data: address } = fromBech322(review.verifier);
  if (prefix !== "zrn" || address.length === 0 || address.length > 255 || toBech32(prefix, address) !== review.verifier) {
    throw new Error("reviewer must be a canonical Zerone account address");
  }
  if (!["accept", "reject", "malformed"].includes(review.vote)) throw new Error("invalid review vote");
  if (typeof review.confidence !== "bigint" || review.confidence < 0n || review.confidence > 1000000n) {
    throw new Error("review confidence must be between 0 and 1000000");
  }
  if (!(review.salt instanceof Uint8Array) || review.salt.length < 16 || review.salt.length > 64) {
    throw new Error("scheme-2 salt must contain 16 to 64 bytes");
  }
  const att = review.attestation;
  if (att === null || typeof att !== "object") throw new Error("review attestation is required");
  const known = /* @__PURE__ */ new Set(["methodId", "reason", "evidenceIds", "scope"]);
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
  const chunks = [utf8.encode("ZRN.review.commit.v2\0")];
  const u32 = (value) => {
    const bytes = new Uint8Array(4);
    new DataView(bytes.buffer).setUint32(0, value, false);
    chunks.push(bytes);
  };
  const field = (bytes) => {
    u32(bytes.length);
    chunks.push(bytes);
  };
  for (const value of [chain, round, address, utf8.encode(review.vote)]) field(value);
  const confidence = new Uint8Array(8);
  new DataView(confidence.buffer).setBigUint64(0, review.confidence, false);
  chunks.push(confidence);
  for (const value of [review.salt, method, reason, scope]) field(value);
  u32(evidence.length);
  for (const value of evidence) field(value);
  const preimage = new Uint8Array(chunks.reduce((n, value) => n + value.length, 0));
  let offset = 0;
  for (const value of chunks) {
    preimage.set(value, offset);
    offset += value.length;
  }
  return sha256(preimage);
}
function makeReviewRevealV2(review) {
  computeReviewCommitmentV2(review);
  return MsgSubmitReveal.fromPartial({
    verifier: review.verifier,
    roundId: review.roundId,
    vote: review.vote,
    confidence: review.confidence,
    salt: Uint8Array.from(review.salt),
    attestation: { ...review.attestation, evidenceIds: [...review.attestation.evidenceIds] }
  });
}

export {
  ACCOUNT_REGISTRATION_PROOF_DOMAIN,
  accountRegistrationProofSignBytes,
  KEY_ROTATION_AUTHORIZATION_DOMAIN,
  KEY_ROTATION_ACCEPTANCE_DOMAIN,
  KEY_ROTATION_AUTHORIZATION_MAX_TTL_SECONDS,
  keyRotationAuthorizationSignBytes,
  keyRotationAcceptanceSignBytes,
  computeReviewCommitmentV2,
  makeReviewRevealV2
};

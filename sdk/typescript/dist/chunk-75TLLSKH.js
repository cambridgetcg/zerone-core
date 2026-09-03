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

export {
  ACCOUNT_REGISTRATION_PROOF_DOMAIN,
  accountRegistrationProofSignBytes,
  KEY_ROTATION_AUTHORIZATION_DOMAIN,
  KEY_ROTATION_ACCEPTANCE_DOMAIN,
  KEY_ROTATION_AUTHORIZATION_MAX_TTL_SECONDS,
  keyRotationAuthorizationSignBytes,
  keyRotationAcceptanceSignBytes
};

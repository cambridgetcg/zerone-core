// src/caip.ts
import { fromBech32, toHex } from "@cosmjs/encoding";
import { sha256 } from "@noble/hashes/sha2.js";
var BIP173_MAX_LENGTH = 90;
var CaipError = class extends Error {
  code;
  constructor(code, message) {
    super(message);
    this.name = "CaipError";
    this.code = code;
  }
};
var CAIP2_PATTERN = /^([-a-z0-9]{3,8}):([-_a-zA-Z0-9]{1,32})$/;
var CAIP10_PATTERN = /^([-a-z0-9]{3,8}):([-_a-zA-Z0-9]{1,32}):([-.%a-zA-Z0-9]{1,128})$/;
var COSMOS_DIRECT_REFERENCE = /^(?!hashed-)[-a-zA-Z0-9]{1,32}$/;
var COSMOS_HASHED_REFERENCE = /^hashed-[0-9a-f]{16}$/;
var EXISTING_ZERONE_DID = /^did:zrn:[0-9a-f]{64}$/;
function fullMatch(pattern, value) {
  const match = pattern.exec(value);
  return match?.[0] === value ? match : null;
}
function hasIllFormedUtf16(value) {
  for (let index = 0; index < value.length; index += 1) {
    const codeUnit = value.charCodeAt(index);
    if (codeUnit >= 55296 && codeUnit <= 56319) {
      const next = value.charCodeAt(index + 1);
      if (!(next >= 56320 && next <= 57343)) {
        return true;
      }
      index += 1;
    } else if (codeUnit >= 56320 && codeUnit <= 57343) {
      return true;
    }
  }
  return false;
}
function parseCaip2(value) {
  const match = fullMatch(CAIP2_PATTERN, value);
  if (!match) {
    throw new CaipError("INVALID_CAIP2", `Invalid CAIP-2 chain ID: ${value}`);
  }
  const namespace = match[1];
  const reference = match[2];
  if (namespace === void 0 || reference === void 0) {
    throw new CaipError("INVALID_CAIP2", `Invalid CAIP-2 chain ID: ${value}`);
  }
  return { id: value, namespace, reference };
}
function formatCaip2(namespace, reference) {
  return parseCaip2(`${namespace}:${reference}`).id;
}
function parseCaip10(value) {
  const match = fullMatch(CAIP10_PATTERN, value);
  if (!match) {
    throw new CaipError("INVALID_CAIP10", `Invalid CAIP-10 account ID: ${value}`);
  }
  const namespace = match[1];
  const reference = match[2];
  const accountAddress = match[3];
  if (namespace === void 0 || reference === void 0 || accountAddress === void 0) {
    throw new CaipError("INVALID_CAIP10", `Invalid CAIP-10 account ID: ${value}`);
  }
  return {
    id: value,
    chainId: formatCaip2(namespace, reference),
    namespace,
    reference,
    accountAddress
  };
}
function formatCaip10(chainId, accountAddress) {
  parseCaip2(chainId);
  return parseCaip10(`${chainId}:${accountAddress}`).id;
}
function cosmosChainId(tendermintChainId) {
  if (tendermintChainId.length === 0) {
    throw new CaipError("INVALID_COSMOS_REFERENCE", "Cosmos chain ID cannot be empty");
  }
  if (hasIllFormedUtf16(tendermintChainId)) {
    throw new CaipError(
      "INVALID_COSMOS_REFERENCE",
      "Cosmos chain ID must contain well-formed Unicode"
    );
  }
  const reference = fullMatch(COSMOS_DIRECT_REFERENCE, tendermintChainId) ? tendermintChainId : `hashed-${toHex(sha256(new TextEncoder().encode(tendermintChainId))).slice(0, 16)}`;
  return formatCaip2("cosmos", reference);
}
function parseCosmosChainId(value) {
  const parsed = parseCaip2(value);
  if (parsed.namespace !== "cosmos") {
    throw new CaipError("NOT_COSMOS", `Expected cosmos namespace, got ${parsed.namespace}`);
  }
  if (!fullMatch(COSMOS_DIRECT_REFERENCE, parsed.reference) && !fullMatch(COSMOS_HASHED_REFERENCE, parsed.reference)) {
    throw new CaipError(
      "INVALID_COSMOS_REFERENCE",
      `Invalid Cosmos chain reference: ${parsed.reference}`
    );
  }
  return parsed.id;
}
function defineZeroneNetwork(rawChainId) {
  const chainId = cosmosChainId(rawChainId);
  return Object.freeze({
    rawChainId,
    chainId,
    accountPrefix: "zrn",
    accountBytes: 20
  });
}
function zeroneAccountId(network, address) {
  const expectedChainId = cosmosChainId(network.rawChainId);
  if (network.chainId !== expectedChainId || network.accountPrefix !== "zrn" || network.accountBytes !== 20) {
    throw new CaipError(
      "INVALID_COSMOS_REFERENCE",
      "Invalid or inconsistent Zerone network descriptor"
    );
  }
  if (address !== address.toLowerCase()) {
    throw new CaipError(
      "NON_CANONICAL_ADDRESS",
      "Zerone account address must be lowercase"
    );
  }
  let decoded;
  try {
    decoded = fromBech32(address, BIP173_MAX_LENGTH);
  } catch {
    throw new CaipError("INVALID_BECH32", `Invalid Zerone Bech32 address: ${address}`);
  }
  if (decoded.prefix !== network.accountPrefix) {
    throw new CaipError(
      "WRONG_HRP",
      `Expected ${network.accountPrefix} account prefix, got ${decoded.prefix}`
    );
  }
  if (decoded.data.length !== network.accountBytes) {
    throw new CaipError(
      "WRONG_ACCOUNT_LENGTH",
      `Expected a ${network.accountBytes}-byte Zerone account, got ${decoded.data.length}`
    );
  }
  return formatCaip10(network.chainId, address);
}
function asExistingZeroneDid(value) {
  if (!fullMatch(EXISTING_ZERONE_DID, value)) {
    throw new CaipError("INVALID_DID_ZRN", `Invalid on-chain did:zrn identifier: ${value}`);
  }
  return value;
}

export {
  CaipError,
  parseCaip2,
  formatCaip2,
  parseCaip10,
  formatCaip10,
  cosmosChainId,
  parseCosmosChainId,
  defineZeroneNetwork,
  zeroneAccountId,
  asExistingZeroneDid
};

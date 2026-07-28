import { fromBech32, toHex } from "@cosmjs/encoding";
import { sha256 } from "@noble/hashes/sha2.js";

declare const caip2Brand: unique symbol;
declare const caip10Brand: unique symbol;
declare const cosmosChainBrand: unique symbol;
declare const zeroneChainBrand: unique symbol;
declare const zeroneAccountBrand: unique symbol;
declare const zeroneNetworkBrand: unique symbol;
declare const zeroneDidBrand: unique symbol;

export type Caip2Id = string & { readonly [caip2Brand]: true };
export type Caip10Id = string & { readonly [caip10Brand]: true };
export type CosmosChainId = Caip2Id & { readonly [cosmosChainBrand]: true };
export type ZeroneChainId = CosmosChainId & {
  readonly [zeroneChainBrand]: true;
};
export type ZeroneAccountId = Caip10Id & {
  readonly [zeroneAccountBrand]: true;
};
export type ZeroneDidRef = string & { readonly [zeroneDidBrand]: true };

/**
 * An application-owned declaration that a Tendermint chain is a Zerone
 * network. Construct it once from trusted network configuration rather than
 * inferring chain semantics from an account's Bech32 prefix.
 */
export interface ZeroneNetwork {
  readonly rawChainId: string;
  readonly chainId: ZeroneChainId;
  readonly accountPrefix: "zrn";
  readonly accountBytes: 20;
  readonly [zeroneNetworkBrand]: true;
}

export interface Caip2Parts {
  readonly id: Caip2Id;
  readonly namespace: string;
  readonly reference: string;
}

export interface Caip10Parts {
  readonly id: Caip10Id;
  readonly chainId: Caip2Id;
  readonly namespace: string;
  readonly reference: string;
  readonly accountAddress: string;
}

export interface ZeroneIdentityRef {
  readonly accountId: ZeroneAccountId;
  readonly did?: ZeroneDidRef;
}

export type CaipErrorCode =
  | "INVALID_CAIP2"
  | "INVALID_CAIP10"
  | "NOT_COSMOS"
  | "INVALID_COSMOS_REFERENCE"
  | "INVALID_BECH32"
  | "NON_CANONICAL_ADDRESS"
  | "WRONG_HRP"
  | "WRONG_ACCOUNT_LENGTH"
  | "INVALID_DID_ZRN";

export class CaipError extends Error {
  readonly code: CaipErrorCode;

  constructor(code: CaipErrorCode, message: string) {
    super(message);
    this.name = "CaipError";
    this.code = code;
  }
}

const CAIP2_PATTERN = /^([-a-z0-9]{3,8}):([-_a-zA-Z0-9]{1,32})$/;
const CAIP10_PATTERN =
  /^([-a-z0-9]{3,8}):([-_a-zA-Z0-9]{1,32}):([-.%a-zA-Z0-9]{1,128})$/;
const COSMOS_DIRECT_REFERENCE = /^(?!hashed-)[-a-zA-Z0-9]{1,32}$/;
const COSMOS_HASHED_REFERENCE = /^hashed-[0-9a-f]{16}$/;
const EXISTING_ZERONE_DID =
  /^did:zrn:(?:[0-9A-Fa-f]{32}|[0-9A-Fa-f]{64})$/;

function fullMatch(pattern: RegExp, value: string): RegExpExecArray | null {
  const match = pattern.exec(value);
  return match?.[0] === value ? match : null;
}

function hasIllFormedUtf16(value: string): boolean {
  for (let index = 0; index < value.length; index += 1) {
    const codeUnit = value.charCodeAt(index);
    if (codeUnit >= 0xd800 && codeUnit <= 0xdbff) {
      const next = value.charCodeAt(index + 1);
      if (!(next >= 0xdc00 && next <= 0xdfff)) {
        return true;
      }
      index += 1;
    } else if (codeUnit >= 0xdc00 && codeUnit <= 0xdfff) {
      return true;
    }
  }
  return false;
}

export function parseCaip2(value: string): Caip2Parts {
  const match = fullMatch(CAIP2_PATTERN, value);
  if (!match) {
    throw new CaipError("INVALID_CAIP2", `Invalid CAIP-2 chain ID: ${value}`);
  }
  const namespace = match[1];
  const reference = match[2];
  if (namespace === undefined || reference === undefined) {
    throw new CaipError("INVALID_CAIP2", `Invalid CAIP-2 chain ID: ${value}`);
  }
  return { id: value as Caip2Id, namespace, reference };
}

export function formatCaip2(namespace: string, reference: string): Caip2Id {
  return parseCaip2(`${namespace}:${reference}`).id;
}

export function parseCaip10(value: string): Caip10Parts {
  const match = fullMatch(CAIP10_PATTERN, value);
  if (!match) {
    throw new CaipError("INVALID_CAIP10", `Invalid CAIP-10 account ID: ${value}`);
  }
  const namespace = match[1];
  const reference = match[2];
  const accountAddress = match[3];
  if (
    namespace === undefined ||
    reference === undefined ||
    accountAddress === undefined
  ) {
    throw new CaipError("INVALID_CAIP10", `Invalid CAIP-10 account ID: ${value}`);
  }
  return {
    id: value as Caip10Id,
    chainId: formatCaip2(namespace, reference),
    namespace,
    reference,
    accountAddress,
  };
}

export function formatCaip10(chainId: Caip2Id, accountAddress: string): Caip10Id {
  parseCaip2(chainId);
  return parseCaip10(`${chainId}:${accountAddress}`).id;
}

export function cosmosChainId(tendermintChainId: string): CosmosChainId {
  if (tendermintChainId.length === 0) {
    throw new CaipError("INVALID_COSMOS_REFERENCE", "Cosmos chain ID cannot be empty");
  }
  if (hasIllFormedUtf16(tendermintChainId)) {
    throw new CaipError(
      "INVALID_COSMOS_REFERENCE",
      "Cosmos chain ID must contain well-formed Unicode",
    );
  }

  const reference = fullMatch(COSMOS_DIRECT_REFERENCE, tendermintChainId)
    ? tendermintChainId
    : `hashed-${toHex(sha256(new TextEncoder().encode(tendermintChainId))).slice(0, 16)}`;
  return formatCaip2("cosmos", reference) as CosmosChainId;
}

export function parseCosmosChainId(value: string): CosmosChainId {
  const parsed = parseCaip2(value);
  if (parsed.namespace !== "cosmos") {
    throw new CaipError("NOT_COSMOS", `Expected cosmos namespace, got ${parsed.namespace}`);
  }
  if (
    !fullMatch(COSMOS_DIRECT_REFERENCE, parsed.reference) &&
    !fullMatch(COSMOS_HASHED_REFERENCE, parsed.reference)
  ) {
    throw new CaipError(
      "INVALID_COSMOS_REFERENCE",
      `Invalid Cosmos chain reference: ${parsed.reference}`,
    );
  }
  return parsed.id as CosmosChainId;
}

export function defineZeroneNetwork(rawChainId: string): ZeroneNetwork {
  const chainId = cosmosChainId(rawChainId) as ZeroneChainId;
  return Object.freeze({
    rawChainId,
    chainId,
    accountPrefix: "zrn",
    accountBytes: 20,
  }) as ZeroneNetwork;
}

export function zeroneAccountId(
  network: ZeroneNetwork,
  address: string,
): ZeroneAccountId {
  const expectedChainId = cosmosChainId(network.rawChainId);
  if (
    network.chainId !== expectedChainId ||
    network.accountPrefix !== "zrn" ||
    network.accountBytes !== 20
  ) {
    throw new CaipError(
      "INVALID_COSMOS_REFERENCE",
      "Invalid or inconsistent Zerone network descriptor",
    );
  }
  if (address !== address.toLowerCase()) {
    throw new CaipError(
      "NON_CANONICAL_ADDRESS",
      "Zerone account address must be lowercase",
    );
  }

  let decoded: ReturnType<typeof fromBech32>;
  try {
    decoded = fromBech32(address);
  } catch {
    throw new CaipError("INVALID_BECH32", `Invalid Zerone Bech32 address: ${address}`);
  }
  if (decoded.prefix !== network.accountPrefix) {
    throw new CaipError(
      "WRONG_HRP",
      `Expected ${network.accountPrefix} account prefix, got ${decoded.prefix}`,
    );
  }
  if (decoded.data.length !== network.accountBytes) {
    throw new CaipError(
      "WRONG_ACCOUNT_LENGTH",
      `Expected a ${network.accountBytes}-byte Zerone account, got ${decoded.data.length}`,
    );
  }
  return formatCaip10(network.chainId, address) as ZeroneAccountId;
}

/**
 * Validates the method-specific identifier accepted by Zerone x/auth today.
 * This does not claim that did:zrn is a published W3C DID method.
 */
export function asExistingZeroneDid(value: string): ZeroneDidRef {
  if (!fullMatch(EXISTING_ZERONE_DID, value)) {
    throw new CaipError("INVALID_DID_ZRN", `Invalid on-chain did:zrn identifier: ${value}`);
  }
  return value as ZeroneDidRef;
}

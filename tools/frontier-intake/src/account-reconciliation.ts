import { createHash } from "node:crypto";
import {
  assertPersistedIdentityKeyCustody,
  validateCanonicalZeroneAddress,
  type PersistedIdentityKey,
} from "./account-registration.ts";

const LOWER_HEX_ED25519_KEY = /^[0-9a-f]{64}$/;
const LOWER_HEX_SHA256 = /^[0-9a-f]{64}$/;
const MAX_UINT32 = 0xffff_ffff;
const CONFIRMED_NOT_FOUND =
  /^Error: failed to query account: rpc error: code = (?:Unknown|NotFound) desc = account not found(?:: (?:unknown request|key not found))?$/;

export interface RegisteredZeroneAccount {
  readonly address: string;
  readonly did: string;
  readonly public_key: string;
  readonly account_type: "agent" | "human" | "contract" | "system";
  readonly operational_key_hash: string;
  readonly operational_public_key: string;
  readonly operational_key_version: number;
}

export interface AccountQueryProcessResult {
  readonly exit_code: number;
  readonly stdout: string;
  readonly stderr: string;
}

export type ClassifiedAccountQuery =
  | { readonly kind: "found"; readonly account: RegisteredZeroneAccount }
  | { readonly kind: "not_found" };

/**
 * Interpret one `zeroned query zerone_auth account ... --output json` result.
 * Only the module's exact not-found diagnostic authorizes first registration;
 * transport, RPC, CLI, and malformed-output failures remain fatal/ambiguous.
 */
export function classifyRegisteredAccountQuery(
  result: AccountQueryProcessResult,
): ClassifiedAccountQuery {
  if (result.exit_code === 0) {
    if (result.stderr.trim() !== "") {
      throw new Error("successful registered-account query wrote unexpected stderr");
    }
    return {
      kind: "found",
      account: parseRegisteredAccountQueryOutput(result.stdout),
    };
  }

  if (result.stdout.trim() !== "") {
    throw new Error(
      `registered-account query failed ambiguously with stdout (exit ${result.exit_code})`,
    );
  }
  const firstDiagnostic = result.stderr
    .split(/\r?\n/, 1)[0]
    ?.trim() ?? "";
  if (CONFIRMED_NOT_FOUND.test(firstDiagnostic)) {
    return { kind: "not_found" };
  }
  throw new Error(
    `registered-account query failed ambiguously (exit ${result.exit_code}): ${firstDiagnostic || "no diagnostic"}`,
  );
}

/** Parse the exact JSON envelope emitted by the Zerone auth account query. */
export function parseRegisteredAccountQueryOutput(
  output: string,
): RegisteredZeroneAccount {
  let parsed: unknown;
  try {
    parsed = JSON.parse(output.trim());
  } catch {
    throw new Error("registered-account query did not return one valid JSON object");
  }
  const envelope = requireObject(parsed, "registered-account query response");
  const account = requireObject(envelope.account, "registered-account query response.account");

  const address = requireString(account.address, "account.address");
  validateCanonicalZeroneAddress(address);
  const did = requireString(account.did, "account.did");
  const publicKey = requireLowerHexKey(account.public_key, "account.public_key");
  if (did !== `did:zrn:${publicKey}`) {
    throw new Error("account.did must be the full DID derived from account.public_key");
  }
  const accountType = requireAccountType(account.account_type);
  const operationalPublicKey = requireLowerHexKey(
    account.operational_public_key,
    "account.operational_public_key",
  );
  const operationalKeyHash = requireString(
    account.operational_key_hash,
    "account.operational_key_hash",
  );
  if (!LOWER_HEX_SHA256.test(operationalKeyHash)) {
    throw new Error("account.operational_key_hash must be 64 lowercase hex characters");
  }
  const computedOperationalHash = sha256PublicKey(operationalPublicKey);
  if (operationalKeyHash !== computedOperationalHash) {
    throw new Error(
      "account.operational_key_hash is not the SHA-256 of account.operational_public_key",
    );
  }
  const operationalKeyVersion = requireOperationalKeyVersion(
    account.operational_key_version,
  );

  return {
    address,
    did,
    public_key: publicKey,
    account_type: accountType,
    operational_key_hash: operationalKeyHash,
    operational_public_key: operationalPublicKey,
    operational_key_version: operationalKeyVersion,
  };
}

/**
 * Reconcile an existing on-chain record with the locally held identity key.
 * Frontier currently persists one keypair per operator, so a rotated-away
 * operational key is unavailable to this client and must never be reported
 * green merely because an account query succeeded.
 */
export function reconcileRegisteredAccount(
  account: RegisteredZeroneAccount,
  expected: {
    readonly key: string;
    readonly address: string;
    readonly accountType: "agent";
    readonly identity: PersistedIdentityKey;
  },
): void {
  assertPersistedIdentityKeyCustody(expected.identity);
  validateCanonicalZeroneAddress(expected.address);

  const expectedDid = `did:zrn:${expected.identity.public_hex}`;
  const expectedHash = sha256PublicKey(expected.identity.public_hex);
  const mismatches: string[] = [];
  if (account.address !== expected.address) {
    mismatches.push(`address is ${account.address}, expected ${expected.address}`);
  }
  if (account.did !== expectedDid) {
    mismatches.push(`DID is ${account.did}, expected ${expectedDid}`);
  }
  if (account.public_key !== expected.identity.public_hex) {
    mismatches.push("immutable identity public key does not match local custody");
  }
  if (account.account_type !== expected.accountType) {
    mismatches.push(
      `account role is ${account.account_type}, expected ${expected.accountType}`,
    );
  }
  if (account.operational_public_key !== expected.identity.public_hex) {
    mismatches.push(
      `current operational key at version ${account.operational_key_version} is unavailable locally`,
    );
  }
  if (account.operational_key_hash !== expectedHash) {
    mismatches.push("current operational key hash does not match local custody");
  }
  if (mismatches.length > 0) {
    throw new Error(
      `registered account reconciliation failed for ${expected.key}: ${mismatches.join("; ")}`,
    );
  }
}

function requireObject(value: unknown, label: string): Record<string, unknown> {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    throw new Error(`${label} must be an object`);
  }
  return value as Record<string, unknown>;
}

function requireString(value: unknown, label: string): string {
  if (typeof value !== "string" || value.length === 0) {
    throw new Error(`${label} must be a non-empty string`);
  }
  return value;
}

function requireLowerHexKey(value: unknown, label: string): string {
  const key = requireString(value, label);
  if (!LOWER_HEX_ED25519_KEY.test(key)) {
    throw new Error(`${label} must be 64 lowercase hex characters`);
  }
  return key;
}

function requireAccountType(
  value: unknown,
): RegisteredZeroneAccount["account_type"] {
  if (
    typeof value !== "string" ||
    !["agent", "human", "contract", "system"].includes(value)
  ) {
    throw new Error("account.account_type is not a supported Zerone account role");
  }
  return value as RegisteredZeroneAccount["account_type"];
}

function requireOperationalKeyVersion(value: unknown): number {
  if (
    typeof value !== "number" ||
    !Number.isInteger(value) ||
    value < 1 ||
    value > MAX_UINT32
  ) {
    throw new Error("account.operational_key_version must be an integer from 1 through uint32 max");
  }
  return value;
}

function sha256PublicKey(publicHex: string): string {
  if (!LOWER_HEX_ED25519_KEY.test(publicHex)) {
    throw new Error("public key must be 64 lowercase hex characters before hashing");
  }
  return createHash("sha256").update(Buffer.from(publicHex, "hex")).digest("hex");
}

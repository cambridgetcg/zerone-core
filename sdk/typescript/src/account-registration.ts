import {
  encodeChainId,
  encodeText,
  validateZeroneAddress,
  writeUint32,
} from "./auth-signing-common";

export const ACCOUNT_REGISTRATION_PROOF_DOMAIN =
  "zerone.auth/register-account/v1" as const;

export type ZeroneAccountType = "agent" | "human" | "contract" | "system";

export interface AccountRegistrationProof {
  readonly chainId: string;
  readonly sender: string;
  readonly did: string;
  readonly identityPublicKey: Uint8Array;
  readonly accountType: ZeroneAccountType;
  readonly metadata: string;
}

/**
 * Returns the exact domain-separated bytes signed by the Ed25519 identity key
 * for MsgRegisterAccount. The independently signed Cosmos transaction does not
 * replace this proof of possession.
 *
 * The operational-key hash is deliberately absent: the chain derives that
 * SHA-256 commitment from identityPublicKey.
 */
export function accountRegistrationProofSignBytes(
  proof: AccountRegistrationProof,
): Uint8Array {
  const chainId = encodeChainId(proof.chainId);
  validateZeroneAddress(proof.sender);
  const sender = encodeText(proof.sender, "sender");
  if (proof.identityPublicKey.length !== 32) {
    throw new RangeError("identity public key must be 32 bytes");
  }

  const publicKeyHex = bytesToHex(proof.identityPublicKey);
  if (proof.did !== `did:zrn:${publicKeyHex}`) {
    throw new RangeError(
      "DID must be did:zrn: followed by the identity public key as 64 lowercase hex characters",
    );
  }
  const did = encodeText(proof.did, "DID");

  if (!isZeroneAccountType(proof.accountType)) {
    throw new RangeError(
      "account type must be agent, human, contract, or system",
    );
  }
  const accountType = encodeText(proof.accountType, "account type");
  const metadata = encodeText(proof.metadata, "metadata");
  const domain = new TextEncoder().encode(ACCOUNT_REGISTRATION_PROOF_DOMAIN);

  const result = new Uint8Array(
    domain.length +
      1 +
      4 +
      chainId.length +
      4 +
      sender.length +
      4 +
      did.length +
      proof.identityPublicKey.length +
      4 +
      accountType.length +
      4 +
      metadata.length,
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

function writeLengthPrefixed(
  output: Uint8Array,
  offset: number,
  value: Uint8Array,
): number {
  offset = writeUint32(output, offset, value.length);
  output.set(value, offset);
  return offset + value.length;
}

function isZeroneAccountType(value: string): value is ZeroneAccountType {
  return (
    value === "agent" ||
    value === "human" ||
    value === "contract" ||
    value === "system"
  );
}

function bytesToHex(value: Uint8Array): string {
  let output = "";
  for (const byte of value) output += byte.toString(16).padStart(2, "0");
  return output;
}

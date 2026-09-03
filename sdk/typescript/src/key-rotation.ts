import {
  U32_MAX,
  encodeChainId,
  encodeText,
  validateZeroneAddress,
  writeUint32,
} from "./auth-signing-common";

export const KEY_ROTATION_AUTHORIZATION_DOMAIN =
  "zerone.auth/rotate-key/v1" as const;
export const KEY_ROTATION_ACCEPTANCE_DOMAIN =
  "zerone.auth/accept-key/v1" as const;
export const KEY_ROTATION_AUTHORIZATION_MAX_TTL_SECONDS = 600n;

const I64_MAX = 0x7fff_ffff_ffff_ffffn;

export interface KeyRotationAuthorization {
  readonly chainId: string;
  readonly sender: string;
  readonly currentKeyVersion: number;
  readonly newOperationalKey: Uint8Array;
  readonly authorizationExpiresAtUnix: bigint;
}

/**
 * Returns the exact domain-separated bytes signed by the current Ed25519
 * operational key for MsgRotateKey. The chain verifies the expiry against
 * consensus block time; caller wall-clock time is not authoritative.
 */
export function keyRotationAuthorizationSignBytes(
  authorization: KeyRotationAuthorization,
): Uint8Array {
  return keyRotationSignBytes(
    KEY_ROTATION_AUTHORIZATION_DOMAIN,
    authorization,
  );
}

/**
 * Returns the exact domain-separated bytes signed by the proposed new
 * Ed25519 operational key. This proof is distinct from the current key's
 * authorization while binding the same transition fields.
 */
export function keyRotationAcceptanceSignBytes(
  authorization: KeyRotationAuthorization,
): Uint8Array {
  return keyRotationSignBytes(KEY_ROTATION_ACCEPTANCE_DOMAIN, authorization);
}

function keyRotationSignBytes(
  domainName: string,
  authorization: KeyRotationAuthorization,
): Uint8Array {
  const chainId = encodeChainId(authorization.chainId);
  validateZeroneAddress(authorization.sender);
  const sender = encodeText(authorization.sender, "sender");
  if (
    !Number.isInteger(authorization.currentKeyVersion) ||
    authorization.currentKeyVersion <= 0 ||
    authorization.currentKeyVersion > U32_MAX
  ) {
    throw new RangeError("current key version must be a positive uint32");
  }
  if (authorization.newOperationalKey.length !== 32) {
    throw new RangeError("new operational key must be 32 bytes");
  }
  if (
    authorization.authorizationExpiresAtUnix <= 0n ||
    authorization.authorizationExpiresAtUnix > I64_MAX
  ) {
    throw new RangeError(
      "authorization expiry must be a positive signed int64 Unix timestamp",
    );
  }

  const domain = new TextEncoder().encode(domainName);
  const result = new Uint8Array(
    domain.length + 1 + 4 + chainId.length + 4 + sender.length + 4 + 8 + 32,
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
    false,
  );
  offset += 8;
  result.set(authorization.newOperationalKey, offset);
  return result;
}

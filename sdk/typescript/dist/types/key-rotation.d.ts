export declare const KEY_ROTATION_AUTHORIZATION_DOMAIN: "zerone.auth/rotate-key/v1";
export declare const KEY_ROTATION_ACCEPTANCE_DOMAIN: "zerone.auth/accept-key/v1";
export declare const KEY_ROTATION_AUTHORIZATION_MAX_TTL_SECONDS = 600n;
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
export declare function keyRotationAuthorizationSignBytes(authorization: KeyRotationAuthorization): Uint8Array;
/**
 * Returns the exact domain-separated bytes signed by the proposed new
 * Ed25519 operational key. This proof is distinct from the current key's
 * authorization while binding the same transition fields.
 */
export declare function keyRotationAcceptanceSignBytes(authorization: KeyRotationAuthorization): Uint8Array;

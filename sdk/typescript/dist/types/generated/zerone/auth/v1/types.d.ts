import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * Account is a Zerone account with DID identity anchoring.
 *
 * Field 8 (session_key_count) was removed with session keys in the
 * 2026-07 slim cut; its number is reserved and must not be reused.
 * @name Account
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.Account
 */
export interface Account {
    /**
     * Bech32 address (store key for account lookups)
     */
    address: string;
    /**
     * Canonical DID derived from the full identity key: did:zrn:{64-lower-hex}
     */
    did: string;
    /**
     * Hex-encoded Ed25519 identity public key (immutable)
     */
    publicKey: string;
    /**
     * Account type: agent, human, contract, system
     */
    accountType: string;
    /**
     * Lowercase SHA-256 of the current operational public-key bytes
     */
    operationalKeyHash: string;
    /**
     * Hex-encoded Ed25519 operational public key. This is independent of the
     * Cosmos BaseAccount transaction key and must never replace it.
     */
    operationalPublicKey: string;
    operationalKeyVersion: number;
    /**
     * Cached reputation score (0-1000000)
     */
    reputationScore: number;
    createdAtBlock: bigint;
    lastActiveBlock: bigint;
    flags?: AccountFlags;
    /**
     * Optional metadata (JSON string, max length governed by params)
     */
    metadata: string;
}
/**
 * AccountFlags packed flags for account capabilities.
 *
 * Field 4 (in_recovery) was removed with social recovery in the 2026-07
 * slim cut; its number is reserved and must not be reused.
 * @name AccountFlags
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.AccountFlags
 */
export interface AccountFlags {
    isValidator: boolean;
    canSubmitClaims: boolean;
    canChallenge: boolean;
    frozen: boolean;
    freezeReason: string;
}
/**
 * DIDMapping stores the bidirectional DID <-> bech32 mapping.
 * @name DIDMapping
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.DIDMapping
 */
export interface DIDMapping {
    did: string;
    bech32: string;
    pubKey: string;
}
/**
 * Account is a Zerone account with DID identity anchoring.
 *
 * Field 8 (session_key_count) was removed with session keys in the
 * 2026-07 slim cut; its number is reserved and must not be reused.
 * @name Account
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.Account
 */
export declare const Account: {
    typeUrl: string;
    encode(message: Account, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Account;
    fromPartial(object: DeepPartial<Account>): Account;
};
/**
 * AccountFlags packed flags for account capabilities.
 *
 * Field 4 (in_recovery) was removed with social recovery in the 2026-07
 * slim cut; its number is reserved and must not be reused.
 * @name AccountFlags
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.AccountFlags
 */
export declare const AccountFlags: {
    typeUrl: string;
    encode(message: AccountFlags, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): AccountFlags;
    fromPartial(object: DeepPartial<AccountFlags>): AccountFlags;
};
/**
 * DIDMapping stores the bidirectional DID <-> bech32 mapping.
 * @name DIDMapping
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.DIDMapping
 */
export declare const DIDMapping: {
    typeUrl: string;
    encode(message: DIDMapping, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): DIDMapping;
    fromPartial(object: DeepPartial<DIDMapping>): DIDMapping;
};

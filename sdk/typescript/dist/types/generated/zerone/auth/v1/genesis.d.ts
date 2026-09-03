import { Account, DIDMapping } from "./types.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * GenesisState defines the auth module's genesis state.
 *
 * Fields 4 (session_keys) and 5 (recovery_configs) were removed with
 * sessions and social recovery in the 2026-07 slim cut; their numbers
 * are reserved and must not be reused.
 * @name GenesisState
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.GenesisState
 */
export interface GenesisState {
    params?: Params;
    accounts: Account[];
    didMappings: DIDMapping[];
    /**
     * Last successful operational-key rotation per account. Exporting this
     * preserves cooldown semantics across export/import and fork recovery.
     */
    lastKeyRotations: KeyRotationRecord[];
}
/**
 * KeyRotationRecord preserves the cooldown anchor for one account.
 * @name KeyRotationRecord
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.KeyRotationRecord
 */
export interface KeyRotationRecord {
    address: string;
    height: bigint;
}
/**
 * Params defines the auth module parameters.
 *
 * Session params (1, 2), recovery params (4, 5, 10, 11, 12) and the
 * dormant bootstrap auto-claim params (6, 7 — the real bootstrap path
 * is x/claiming_pot through MintWithCap) were removed in the 2026-07
 * slim cut; their numbers are reserved and must not be reused.
 * @name Params
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.Params
 */
export interface Params {
    keyRotationCooldown: bigint;
    /**
     * Max metadata length in bytes (default 1024)
     */
    maxMetadataLength: number;
    /**
     * Whether DID is required for registration
     */
    requireDid: boolean;
}
/**
 * GenesisState defines the auth module's genesis state.
 *
 * Fields 4 (session_keys) and 5 (recovery_configs) were removed with
 * sessions and social recovery in the 2026-07 slim cut; their numbers
 * are reserved and must not be reused.
 * @name GenesisState
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.GenesisState
 */
export declare const GenesisState: {
    typeUrl: string;
    encode(message: GenesisState, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GenesisState;
    fromPartial(object: DeepPartial<GenesisState>): GenesisState;
};
/**
 * KeyRotationRecord preserves the cooldown anchor for one account.
 * @name KeyRotationRecord
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.KeyRotationRecord
 */
export declare const KeyRotationRecord: {
    typeUrl: string;
    encode(message: KeyRotationRecord, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): KeyRotationRecord;
    fromPartial(object: DeepPartial<KeyRotationRecord>): KeyRotationRecord;
};
/**
 * Params defines the auth module parameters.
 *
 * Session params (1, 2), recovery params (4, 5, 10, 11, 12) and the
 * dormant bootstrap auto-claim params (6, 7 — the real bootstrap path
 * is x/claiming_pot through MintWithCap) were removed in the 2026-07
 * slim cut; their numbers are reserved and must not be reused.
 * @name Params
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.Params
 */
export declare const Params: {
    typeUrl: string;
    encode(message: Params, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params;
    fromPartial(object: DeepPartial<Params>): Params;
};

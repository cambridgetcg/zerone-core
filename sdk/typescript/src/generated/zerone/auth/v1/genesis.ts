//@ts-nocheck
import { Account, DIDMapping } from "./types";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
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
function createBaseGenesisState(): GenesisState {
  return {
    params: undefined,
    accounts: [],
    didMappings: [],
    lastKeyRotations: []
  };
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
export const GenesisState = {
  typeUrl: "/zerone.auth.v1.GenesisState",
  encode(message: GenesisState, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.accounts) {
      Account.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.didMappings) {
      DIDMapping.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.lastKeyRotations) {
      KeyRotationRecord.encode(v!, writer.uint32(50).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGenesisState();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.params = Params.decode(reader, reader.uint32());
          break;
        case 2:
          message.accounts.push(Account.decode(reader, reader.uint32()));
          break;
        case 3:
          message.didMappings.push(DIDMapping.decode(reader, reader.uint32()));
          break;
        case 6:
          message.lastKeyRotations.push(KeyRotationRecord.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<GenesisState>): GenesisState {
    const message = createBaseGenesisState();
    message.params = object.params !== undefined && object.params !== null ? Params.fromPartial(object.params) : undefined;
    message.accounts = object.accounts?.map(e => Account.fromPartial(e)) || [];
    message.didMappings = object.didMappings?.map(e => DIDMapping.fromPartial(e)) || [];
    message.lastKeyRotations = object.lastKeyRotations?.map(e => KeyRotationRecord.fromPartial(e)) || [];
    return message;
  }
};
function createBaseKeyRotationRecord(): KeyRotationRecord {
  return {
    address: "",
    height: BigInt(0)
  };
}
/**
 * KeyRotationRecord preserves the cooldown anchor for one account.
 * @name KeyRotationRecord
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.KeyRotationRecord
 */
export const KeyRotationRecord = {
  typeUrl: "/zerone.auth.v1.KeyRotationRecord",
  encode(message: KeyRotationRecord, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.address !== "") {
      writer.uint32(10).string(message.address);
    }
    if (message.height !== BigInt(0)) {
      writer.uint32(16).uint64(message.height);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): KeyRotationRecord {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseKeyRotationRecord();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.address = reader.string();
          break;
        case 2:
          message.height = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<KeyRotationRecord>): KeyRotationRecord {
    const message = createBaseKeyRotationRecord();
    message.address = object.address ?? "";
    message.height = object.height !== undefined && object.height !== null ? BigInt(object.height.toString()) : BigInt(0);
    return message;
  }
};
function createBaseParams(): Params {
  return {
    keyRotationCooldown: BigInt(0),
    maxMetadataLength: 0,
    requireDid: false
  };
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
export const Params = {
  typeUrl: "/zerone.auth.v1.Params",
  encode(message: Params, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.keyRotationCooldown !== BigInt(0)) {
      writer.uint32(24).uint64(message.keyRotationCooldown);
    }
    if (message.maxMetadataLength !== 0) {
      writer.uint32(64).uint32(message.maxMetadataLength);
    }
    if (message.requireDid === true) {
      writer.uint32(72).bool(message.requireDid);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Params {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseParams();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 3:
          message.keyRotationCooldown = reader.uint64();
          break;
        case 8:
          message.maxMetadataLength = reader.uint32();
          break;
        case 9:
          message.requireDid = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Params>): Params {
    const message = createBaseParams();
    message.keyRotationCooldown = object.keyRotationCooldown !== undefined && object.keyRotationCooldown !== null ? BigInt(object.keyRotationCooldown.toString()) : BigInt(0);
    message.maxMetadataLength = object.maxMetadataLength ?? 0;
    message.requireDid = object.requireDid ?? false;
    return message;
  }
};
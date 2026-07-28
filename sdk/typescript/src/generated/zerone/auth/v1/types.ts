//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
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
   * DID derived from identity key: did:zrn:{32-hex}
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
   * Hash of current operational key
   */
  operationalKeyHash: string;
  /**
   * Hex-encoded Ed25519 operational public key (synced to Cosmos BaseAccount)
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
function createBaseAccount(): Account {
  return {
    address: "",
    did: "",
    publicKey: "",
    accountType: "",
    operationalKeyHash: "",
    operationalPublicKey: "",
    operationalKeyVersion: 0,
    reputationScore: 0,
    createdAtBlock: BigInt(0),
    lastActiveBlock: BigInt(0),
    flags: undefined,
    metadata: ""
  };
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
export const Account = {
  typeUrl: "/zerone.auth.v1.Account",
  encode(message: Account, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.address !== "") {
      writer.uint32(10).string(message.address);
    }
    if (message.did !== "") {
      writer.uint32(18).string(message.did);
    }
    if (message.publicKey !== "") {
      writer.uint32(26).string(message.publicKey);
    }
    if (message.accountType !== "") {
      writer.uint32(34).string(message.accountType);
    }
    if (message.operationalKeyHash !== "") {
      writer.uint32(42).string(message.operationalKeyHash);
    }
    if (message.operationalPublicKey !== "") {
      writer.uint32(50).string(message.operationalPublicKey);
    }
    if (message.operationalKeyVersion !== 0) {
      writer.uint32(56).uint32(message.operationalKeyVersion);
    }
    if (message.reputationScore !== 0) {
      writer.uint32(72).uint32(message.reputationScore);
    }
    if (message.createdAtBlock !== BigInt(0)) {
      writer.uint32(80).uint64(message.createdAtBlock);
    }
    if (message.lastActiveBlock !== BigInt(0)) {
      writer.uint32(88).uint64(message.lastActiveBlock);
    }
    if (message.flags !== undefined) {
      AccountFlags.encode(message.flags, writer.uint32(98).fork()).ldelim();
    }
    if (message.metadata !== "") {
      writer.uint32(106).string(message.metadata);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Account {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAccount();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.address = reader.string();
          break;
        case 2:
          message.did = reader.string();
          break;
        case 3:
          message.publicKey = reader.string();
          break;
        case 4:
          message.accountType = reader.string();
          break;
        case 5:
          message.operationalKeyHash = reader.string();
          break;
        case 6:
          message.operationalPublicKey = reader.string();
          break;
        case 7:
          message.operationalKeyVersion = reader.uint32();
          break;
        case 9:
          message.reputationScore = reader.uint32();
          break;
        case 10:
          message.createdAtBlock = reader.uint64();
          break;
        case 11:
          message.lastActiveBlock = reader.uint64();
          break;
        case 12:
          message.flags = AccountFlags.decode(reader, reader.uint32());
          break;
        case 13:
          message.metadata = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Account>): Account {
    const message = createBaseAccount();
    message.address = object.address ?? "";
    message.did = object.did ?? "";
    message.publicKey = object.publicKey ?? "";
    message.accountType = object.accountType ?? "";
    message.operationalKeyHash = object.operationalKeyHash ?? "";
    message.operationalPublicKey = object.operationalPublicKey ?? "";
    message.operationalKeyVersion = object.operationalKeyVersion ?? 0;
    message.reputationScore = object.reputationScore ?? 0;
    message.createdAtBlock = object.createdAtBlock !== undefined && object.createdAtBlock !== null ? BigInt(object.createdAtBlock.toString()) : BigInt(0);
    message.lastActiveBlock = object.lastActiveBlock !== undefined && object.lastActiveBlock !== null ? BigInt(object.lastActiveBlock.toString()) : BigInt(0);
    message.flags = object.flags !== undefined && object.flags !== null ? AccountFlags.fromPartial(object.flags) : undefined;
    message.metadata = object.metadata ?? "";
    return message;
  }
};
function createBaseAccountFlags(): AccountFlags {
  return {
    isValidator: false,
    canSubmitClaims: false,
    canChallenge: false,
    frozen: false,
    freezeReason: ""
  };
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
export const AccountFlags = {
  typeUrl: "/zerone.auth.v1.AccountFlags",
  encode(message: AccountFlags, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.isValidator === true) {
      writer.uint32(8).bool(message.isValidator);
    }
    if (message.canSubmitClaims === true) {
      writer.uint32(16).bool(message.canSubmitClaims);
    }
    if (message.canChallenge === true) {
      writer.uint32(24).bool(message.canChallenge);
    }
    if (message.frozen === true) {
      writer.uint32(40).bool(message.frozen);
    }
    if (message.freezeReason !== "") {
      writer.uint32(50).string(message.freezeReason);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): AccountFlags {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseAccountFlags();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.isValidator = reader.bool();
          break;
        case 2:
          message.canSubmitClaims = reader.bool();
          break;
        case 3:
          message.canChallenge = reader.bool();
          break;
        case 5:
          message.frozen = reader.bool();
          break;
        case 6:
          message.freezeReason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<AccountFlags>): AccountFlags {
    const message = createBaseAccountFlags();
    message.isValidator = object.isValidator ?? false;
    message.canSubmitClaims = object.canSubmitClaims ?? false;
    message.canChallenge = object.canChallenge ?? false;
    message.frozen = object.frozen ?? false;
    message.freezeReason = object.freezeReason ?? "";
    return message;
  }
};
function createBaseDIDMapping(): DIDMapping {
  return {
    did: "",
    bech32: "",
    pubKey: ""
  };
}
/**
 * DIDMapping stores the bidirectional DID <-> bech32 mapping.
 * @name DIDMapping
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.DIDMapping
 */
export const DIDMapping = {
  typeUrl: "/zerone.auth.v1.DIDMapping",
  encode(message: DIDMapping, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.did !== "") {
      writer.uint32(10).string(message.did);
    }
    if (message.bech32 !== "") {
      writer.uint32(18).string(message.bech32);
    }
    if (message.pubKey !== "") {
      writer.uint32(26).string(message.pubKey);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): DIDMapping {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDIDMapping();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.did = reader.string();
          break;
        case 2:
          message.bech32 = reader.string();
          break;
        case 3:
          message.pubKey = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<DIDMapping>): DIDMapping {
    const message = createBaseDIDMapping();
    message.did = object.did ?? "";
    message.bech32 = object.bech32 ?? "";
    message.pubKey = object.pubKey ?? "";
    return message;
  }
};
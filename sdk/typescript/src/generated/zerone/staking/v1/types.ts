//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/** ValidatorTier enumerates the 4-tier graduated validator system. */
export enum ValidatorTier {
  VALIDATOR_TIER_UNSPECIFIED = 0,
  VALIDATOR_TIER_APPRENTICE = 1,
  VALIDATOR_TIER_VERIFIED = 2,
  VALIDATOR_TIER_SCHOLAR = 3,
  VALIDATOR_TIER_GUARDIAN = 4,
  UNRECOGNIZED = -1,
}
export function validatorTierFromJSON(object: any): ValidatorTier {
  switch (object) {
    case 0:
    case "VALIDATOR_TIER_UNSPECIFIED":
      return ValidatorTier.VALIDATOR_TIER_UNSPECIFIED;
    case 1:
    case "VALIDATOR_TIER_APPRENTICE":
      return ValidatorTier.VALIDATOR_TIER_APPRENTICE;
    case 2:
    case "VALIDATOR_TIER_VERIFIED":
      return ValidatorTier.VALIDATOR_TIER_VERIFIED;
    case 3:
    case "VALIDATOR_TIER_SCHOLAR":
      return ValidatorTier.VALIDATOR_TIER_SCHOLAR;
    case 4:
    case "VALIDATOR_TIER_GUARDIAN":
      return ValidatorTier.VALIDATOR_TIER_GUARDIAN;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ValidatorTier.UNRECOGNIZED;
  }
}
export function validatorTierToJSON(object: ValidatorTier): string {
  switch (object) {
    case ValidatorTier.VALIDATOR_TIER_UNSPECIFIED:
      return "VALIDATOR_TIER_UNSPECIFIED";
    case ValidatorTier.VALIDATOR_TIER_APPRENTICE:
      return "VALIDATOR_TIER_APPRENTICE";
    case ValidatorTier.VALIDATOR_TIER_VERIFIED:
      return "VALIDATOR_TIER_VERIFIED";
    case ValidatorTier.VALIDATOR_TIER_SCHOLAR:
      return "VALIDATOR_TIER_SCHOLAR";
    case ValidatorTier.VALIDATOR_TIER_GUARDIAN:
      return "VALIDATOR_TIER_GUARDIAN";
    case ValidatorTier.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/**
 * Validator represents a Zerone validator with tier and reputation.
 * @name Validator
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.Validator
 */
export interface Validator {
  operatorAddress: string;
  consensusPubkey: string;
  did: string;
  moniker: string;
  tier: ValidatorTier;
  /**
   * uzrn amount (string for big int)
   */
  selfDelegation: string;
  /**
   * uzrn delegated by others
   */
  delegatedStake: string;
  /**
   * self_delegation + delegated_stake
   */
  totalStake: string;
  /**
   * 0–1,000,000 BPS
   */
  reputationScore: bigint;
  correctVerifications: bigint;
  totalVerifications: bigint;
  /**
   * must increment! (Guardian fix)
   */
  contestedCount: bigint;
  contestedVerificationsCorrect: bigint;
  joinedAtBlock: bigint;
  isActive: boolean;
  jailed: boolean;
  jailReason: string;
  unjailAfterBlock: bigint;
  slashCount: bigint;
  slashesThisEpoch: bigint;
  lastSlashHeight: bigint;
  website: string;
  details: string;
  /**
   * up to 10,000 (100%)
   */
  commissionBps: bigint;
}
/**
 * Delegation records a delegation from delegator to validator.
 * @name Delegation
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.Delegation
 */
export interface Delegation {
  delegatorAddress: string;
  validatorAddress: string;
  /**
   * uzrn
   */
  amount: string;
  createdAtBlock: bigint;
}
/**
 * UnbondingEntry records a pending unbonding.
 * @name UnbondingEntry
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.UnbondingEntry
 */
export interface UnbondingEntry {
  id: string;
  delegatorAddress: string;
  validatorAddress: string;
  /**
   * uzrn
   */
  amount: string;
  createdAtHeight: bigint;
  completesAtHeight: bigint;
  /**
   * "pending" or "completed"
   */
  status: string;
}
/**
 * TierConfig defines requirements and rewards for each validator tier.
 * @name TierConfig
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.TierConfig
 */
export interface TierConfig {
  tier: ValidatorTier;
  /**
   * "Apprentice", "Verified", "Scholar", "Guardian"
   */
  name: string;
  /**
   * uzrn (MUST be enforced — audit fix)
   */
  minStake: string;
  /**
   * BPS
   */
  minReputation: bigint;
  minVerifications: bigint;
  /**
   * BPS (1,000,000 = 100%)
   */
  minAccuracy: bigint;
  /**
   * -1 = unlimited
   */
  maxSlashCount: bigint;
  allowedCategories: string[];
  /**
   * 1,000 = 1.0x
   */
  rewardMultiplierBps: bigint;
  /**
   * VRF selection weight
   */
  selectionWeightBps: bigint;
  slashWindowEpochs: bigint;
  minContestedVerifications: bigint;
  contestedVerificationMultiplier: bigint;
  /**
   * 1,000 = 1.0x
   */
  slashMultiplierBps: bigint;
}
function createBaseValidator(): Validator {
  return {
    operatorAddress: "",
    consensusPubkey: "",
    did: "",
    moniker: "",
    tier: 0,
    selfDelegation: "",
    delegatedStake: "",
    totalStake: "",
    reputationScore: BigInt(0),
    correctVerifications: BigInt(0),
    totalVerifications: BigInt(0),
    contestedCount: BigInt(0),
    contestedVerificationsCorrect: BigInt(0),
    joinedAtBlock: BigInt(0),
    isActive: false,
    jailed: false,
    jailReason: "",
    unjailAfterBlock: BigInt(0),
    slashCount: BigInt(0),
    slashesThisEpoch: BigInt(0),
    lastSlashHeight: BigInt(0),
    website: "",
    details: "",
    commissionBps: BigInt(0)
  };
}
/**
 * Validator represents a Zerone validator with tier and reputation.
 * @name Validator
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.Validator
 */
export const Validator = {
  typeUrl: "/zerone.staking.v1.Validator",
  encode(message: Validator, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.operatorAddress !== "") {
      writer.uint32(10).string(message.operatorAddress);
    }
    if (message.consensusPubkey !== "") {
      writer.uint32(18).string(message.consensusPubkey);
    }
    if (message.did !== "") {
      writer.uint32(26).string(message.did);
    }
    if (message.moniker !== "") {
      writer.uint32(34).string(message.moniker);
    }
    if (message.tier !== 0) {
      writer.uint32(40).int32(message.tier);
    }
    if (message.selfDelegation !== "") {
      writer.uint32(50).string(message.selfDelegation);
    }
    if (message.delegatedStake !== "") {
      writer.uint32(58).string(message.delegatedStake);
    }
    if (message.totalStake !== "") {
      writer.uint32(66).string(message.totalStake);
    }
    if (message.reputationScore !== BigInt(0)) {
      writer.uint32(72).uint64(message.reputationScore);
    }
    if (message.correctVerifications !== BigInt(0)) {
      writer.uint32(80).uint64(message.correctVerifications);
    }
    if (message.totalVerifications !== BigInt(0)) {
      writer.uint32(88).uint64(message.totalVerifications);
    }
    if (message.contestedCount !== BigInt(0)) {
      writer.uint32(96).uint64(message.contestedCount);
    }
    if (message.contestedVerificationsCorrect !== BigInt(0)) {
      writer.uint32(104).uint64(message.contestedVerificationsCorrect);
    }
    if (message.joinedAtBlock !== BigInt(0)) {
      writer.uint32(112).uint64(message.joinedAtBlock);
    }
    if (message.isActive === true) {
      writer.uint32(120).bool(message.isActive);
    }
    if (message.jailed === true) {
      writer.uint32(128).bool(message.jailed);
    }
    if (message.jailReason !== "") {
      writer.uint32(138).string(message.jailReason);
    }
    if (message.unjailAfterBlock !== BigInt(0)) {
      writer.uint32(144).uint64(message.unjailAfterBlock);
    }
    if (message.slashCount !== BigInt(0)) {
      writer.uint32(152).uint64(message.slashCount);
    }
    if (message.slashesThisEpoch !== BigInt(0)) {
      writer.uint32(160).uint64(message.slashesThisEpoch);
    }
    if (message.lastSlashHeight !== BigInt(0)) {
      writer.uint32(168).uint64(message.lastSlashHeight);
    }
    if (message.website !== "") {
      writer.uint32(178).string(message.website);
    }
    if (message.details !== "") {
      writer.uint32(186).string(message.details);
    }
    if (message.commissionBps !== BigInt(0)) {
      writer.uint32(192).uint64(message.commissionBps);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Validator {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseValidator();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.operatorAddress = reader.string();
          break;
        case 2:
          message.consensusPubkey = reader.string();
          break;
        case 3:
          message.did = reader.string();
          break;
        case 4:
          message.moniker = reader.string();
          break;
        case 5:
          message.tier = reader.int32() as any;
          break;
        case 6:
          message.selfDelegation = reader.string();
          break;
        case 7:
          message.delegatedStake = reader.string();
          break;
        case 8:
          message.totalStake = reader.string();
          break;
        case 9:
          message.reputationScore = reader.uint64();
          break;
        case 10:
          message.correctVerifications = reader.uint64();
          break;
        case 11:
          message.totalVerifications = reader.uint64();
          break;
        case 12:
          message.contestedCount = reader.uint64();
          break;
        case 13:
          message.contestedVerificationsCorrect = reader.uint64();
          break;
        case 14:
          message.joinedAtBlock = reader.uint64();
          break;
        case 15:
          message.isActive = reader.bool();
          break;
        case 16:
          message.jailed = reader.bool();
          break;
        case 17:
          message.jailReason = reader.string();
          break;
        case 18:
          message.unjailAfterBlock = reader.uint64();
          break;
        case 19:
          message.slashCount = reader.uint64();
          break;
        case 20:
          message.slashesThisEpoch = reader.uint64();
          break;
        case 21:
          message.lastSlashHeight = reader.uint64();
          break;
        case 22:
          message.website = reader.string();
          break;
        case 23:
          message.details = reader.string();
          break;
        case 24:
          message.commissionBps = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Validator>): Validator {
    const message = createBaseValidator();
    message.operatorAddress = object.operatorAddress ?? "";
    message.consensusPubkey = object.consensusPubkey ?? "";
    message.did = object.did ?? "";
    message.moniker = object.moniker ?? "";
    message.tier = object.tier ?? 0;
    message.selfDelegation = object.selfDelegation ?? "";
    message.delegatedStake = object.delegatedStake ?? "";
    message.totalStake = object.totalStake ?? "";
    message.reputationScore = object.reputationScore !== undefined && object.reputationScore !== null ? BigInt(object.reputationScore.toString()) : BigInt(0);
    message.correctVerifications = object.correctVerifications !== undefined && object.correctVerifications !== null ? BigInt(object.correctVerifications.toString()) : BigInt(0);
    message.totalVerifications = object.totalVerifications !== undefined && object.totalVerifications !== null ? BigInt(object.totalVerifications.toString()) : BigInt(0);
    message.contestedCount = object.contestedCount !== undefined && object.contestedCount !== null ? BigInt(object.contestedCount.toString()) : BigInt(0);
    message.contestedVerificationsCorrect = object.contestedVerificationsCorrect !== undefined && object.contestedVerificationsCorrect !== null ? BigInt(object.contestedVerificationsCorrect.toString()) : BigInt(0);
    message.joinedAtBlock = object.joinedAtBlock !== undefined && object.joinedAtBlock !== null ? BigInt(object.joinedAtBlock.toString()) : BigInt(0);
    message.isActive = object.isActive ?? false;
    message.jailed = object.jailed ?? false;
    message.jailReason = object.jailReason ?? "";
    message.unjailAfterBlock = object.unjailAfterBlock !== undefined && object.unjailAfterBlock !== null ? BigInt(object.unjailAfterBlock.toString()) : BigInt(0);
    message.slashCount = object.slashCount !== undefined && object.slashCount !== null ? BigInt(object.slashCount.toString()) : BigInt(0);
    message.slashesThisEpoch = object.slashesThisEpoch !== undefined && object.slashesThisEpoch !== null ? BigInt(object.slashesThisEpoch.toString()) : BigInt(0);
    message.lastSlashHeight = object.lastSlashHeight !== undefined && object.lastSlashHeight !== null ? BigInt(object.lastSlashHeight.toString()) : BigInt(0);
    message.website = object.website ?? "";
    message.details = object.details ?? "";
    message.commissionBps = object.commissionBps !== undefined && object.commissionBps !== null ? BigInt(object.commissionBps.toString()) : BigInt(0);
    return message;
  }
};
function createBaseDelegation(): Delegation {
  return {
    delegatorAddress: "",
    validatorAddress: "",
    amount: "",
    createdAtBlock: BigInt(0)
  };
}
/**
 * Delegation records a delegation from delegator to validator.
 * @name Delegation
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.Delegation
 */
export const Delegation = {
  typeUrl: "/zerone.staking.v1.Delegation",
  encode(message: Delegation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.delegatorAddress !== "") {
      writer.uint32(10).string(message.delegatorAddress);
    }
    if (message.validatorAddress !== "") {
      writer.uint32(18).string(message.validatorAddress);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    if (message.createdAtBlock !== BigInt(0)) {
      writer.uint32(32).uint64(message.createdAtBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Delegation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDelegation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.delegatorAddress = reader.string();
          break;
        case 2:
          message.validatorAddress = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        case 4:
          message.createdAtBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Delegation>): Delegation {
    const message = createBaseDelegation();
    message.delegatorAddress = object.delegatorAddress ?? "";
    message.validatorAddress = object.validatorAddress ?? "";
    message.amount = object.amount ?? "";
    message.createdAtBlock = object.createdAtBlock !== undefined && object.createdAtBlock !== null ? BigInt(object.createdAtBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseUnbondingEntry(): UnbondingEntry {
  return {
    id: "",
    delegatorAddress: "",
    validatorAddress: "",
    amount: "",
    createdAtHeight: BigInt(0),
    completesAtHeight: BigInt(0),
    status: ""
  };
}
/**
 * UnbondingEntry records a pending unbonding.
 * @name UnbondingEntry
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.UnbondingEntry
 */
export const UnbondingEntry = {
  typeUrl: "/zerone.staking.v1.UnbondingEntry",
  encode(message: UnbondingEntry, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.delegatorAddress !== "") {
      writer.uint32(18).string(message.delegatorAddress);
    }
    if (message.validatorAddress !== "") {
      writer.uint32(26).string(message.validatorAddress);
    }
    if (message.amount !== "") {
      writer.uint32(34).string(message.amount);
    }
    if (message.createdAtHeight !== BigInt(0)) {
      writer.uint32(40).uint64(message.createdAtHeight);
    }
    if (message.completesAtHeight !== BigInt(0)) {
      writer.uint32(48).uint64(message.completesAtHeight);
    }
    if (message.status !== "") {
      writer.uint32(58).string(message.status);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): UnbondingEntry {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUnbondingEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.delegatorAddress = reader.string();
          break;
        case 3:
          message.validatorAddress = reader.string();
          break;
        case 4:
          message.amount = reader.string();
          break;
        case 5:
          message.createdAtHeight = reader.uint64();
          break;
        case 6:
          message.completesAtHeight = reader.uint64();
          break;
        case 7:
          message.status = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<UnbondingEntry>): UnbondingEntry {
    const message = createBaseUnbondingEntry();
    message.id = object.id ?? "";
    message.delegatorAddress = object.delegatorAddress ?? "";
    message.validatorAddress = object.validatorAddress ?? "";
    message.amount = object.amount ?? "";
    message.createdAtHeight = object.createdAtHeight !== undefined && object.createdAtHeight !== null ? BigInt(object.createdAtHeight.toString()) : BigInt(0);
    message.completesAtHeight = object.completesAtHeight !== undefined && object.completesAtHeight !== null ? BigInt(object.completesAtHeight.toString()) : BigInt(0);
    message.status = object.status ?? "";
    return message;
  }
};
function createBaseTierConfig(): TierConfig {
  return {
    tier: 0,
    name: "",
    minStake: "",
    minReputation: BigInt(0),
    minVerifications: BigInt(0),
    minAccuracy: BigInt(0),
    maxSlashCount: BigInt(0),
    allowedCategories: [],
    rewardMultiplierBps: BigInt(0),
    selectionWeightBps: BigInt(0),
    slashWindowEpochs: BigInt(0),
    minContestedVerifications: BigInt(0),
    contestedVerificationMultiplier: BigInt(0),
    slashMultiplierBps: BigInt(0)
  };
}
/**
 * TierConfig defines requirements and rewards for each validator tier.
 * @name TierConfig
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.TierConfig
 */
export const TierConfig = {
  typeUrl: "/zerone.staking.v1.TierConfig",
  encode(message: TierConfig, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.tier !== 0) {
      writer.uint32(8).int32(message.tier);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.minStake !== "") {
      writer.uint32(26).string(message.minStake);
    }
    if (message.minReputation !== BigInt(0)) {
      writer.uint32(32).uint64(message.minReputation);
    }
    if (message.minVerifications !== BigInt(0)) {
      writer.uint32(40).uint64(message.minVerifications);
    }
    if (message.minAccuracy !== BigInt(0)) {
      writer.uint32(48).uint64(message.minAccuracy);
    }
    if (message.maxSlashCount !== BigInt(0)) {
      writer.uint32(56).int64(message.maxSlashCount);
    }
    for (const v of message.allowedCategories) {
      writer.uint32(66).string(v!);
    }
    if (message.rewardMultiplierBps !== BigInt(0)) {
      writer.uint32(72).uint64(message.rewardMultiplierBps);
    }
    if (message.selectionWeightBps !== BigInt(0)) {
      writer.uint32(80).uint64(message.selectionWeightBps);
    }
    if (message.slashWindowEpochs !== BigInt(0)) {
      writer.uint32(88).uint64(message.slashWindowEpochs);
    }
    if (message.minContestedVerifications !== BigInt(0)) {
      writer.uint32(96).uint64(message.minContestedVerifications);
    }
    if (message.contestedVerificationMultiplier !== BigInt(0)) {
      writer.uint32(104).uint64(message.contestedVerificationMultiplier);
    }
    if (message.slashMultiplierBps !== BigInt(0)) {
      writer.uint32(112).uint64(message.slashMultiplierBps);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): TierConfig {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTierConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.tier = reader.int32() as any;
          break;
        case 2:
          message.name = reader.string();
          break;
        case 3:
          message.minStake = reader.string();
          break;
        case 4:
          message.minReputation = reader.uint64();
          break;
        case 5:
          message.minVerifications = reader.uint64();
          break;
        case 6:
          message.minAccuracy = reader.uint64();
          break;
        case 7:
          message.maxSlashCount = reader.int64();
          break;
        case 8:
          message.allowedCategories.push(reader.string());
          break;
        case 9:
          message.rewardMultiplierBps = reader.uint64();
          break;
        case 10:
          message.selectionWeightBps = reader.uint64();
          break;
        case 11:
          message.slashWindowEpochs = reader.uint64();
          break;
        case 12:
          message.minContestedVerifications = reader.uint64();
          break;
        case 13:
          message.contestedVerificationMultiplier = reader.uint64();
          break;
        case 14:
          message.slashMultiplierBps = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<TierConfig>): TierConfig {
    const message = createBaseTierConfig();
    message.tier = object.tier ?? 0;
    message.name = object.name ?? "";
    message.minStake = object.minStake ?? "";
    message.minReputation = object.minReputation !== undefined && object.minReputation !== null ? BigInt(object.minReputation.toString()) : BigInt(0);
    message.minVerifications = object.minVerifications !== undefined && object.minVerifications !== null ? BigInt(object.minVerifications.toString()) : BigInt(0);
    message.minAccuracy = object.minAccuracy !== undefined && object.minAccuracy !== null ? BigInt(object.minAccuracy.toString()) : BigInt(0);
    message.maxSlashCount = object.maxSlashCount !== undefined && object.maxSlashCount !== null ? BigInt(object.maxSlashCount.toString()) : BigInt(0);
    message.allowedCategories = object.allowedCategories?.map(e => e) || [];
    message.rewardMultiplierBps = object.rewardMultiplierBps !== undefined && object.rewardMultiplierBps !== null ? BigInt(object.rewardMultiplierBps.toString()) : BigInt(0);
    message.selectionWeightBps = object.selectionWeightBps !== undefined && object.selectionWeightBps !== null ? BigInt(object.selectionWeightBps.toString()) : BigInt(0);
    message.slashWindowEpochs = object.slashWindowEpochs !== undefined && object.slashWindowEpochs !== null ? BigInt(object.slashWindowEpochs.toString()) : BigInt(0);
    message.minContestedVerifications = object.minContestedVerifications !== undefined && object.minContestedVerifications !== null ? BigInt(object.minContestedVerifications.toString()) : BigInt(0);
    message.contestedVerificationMultiplier = object.contestedVerificationMultiplier !== undefined && object.contestedVerificationMultiplier !== null ? BigInt(object.contestedVerificationMultiplier.toString()) : BigInt(0);
    message.slashMultiplierBps = object.slashMultiplierBps !== undefined && object.slashMultiplierBps !== null ? BigInt(object.slashMultiplierBps.toString()) : BigInt(0);
    return message;
  }
};
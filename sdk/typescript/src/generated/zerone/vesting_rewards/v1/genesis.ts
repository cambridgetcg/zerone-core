//@ts-nocheck
import { CategoryConfig, VestingSchedule, ClawbackRecord, BlockRewardDistribution, ClaimScheduleIndex } from "./state";
import { RevenueSplit, ProtocolSubSplit } from "../../common/v1/common";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * GenesisState defines the vesting_rewards module's genesis state.
 * @name GenesisState
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.GenesisState
 */
export interface GenesisState {
  params?: Params;
  categoryConfigs: CategoryConfig[];
  vestingSchedules: VestingSchedule[];
  clawbackRecords: ClawbackRecord[];
  blockRewardDistributions: BlockRewardDistribution[];
  claimScheduleIndexes: ClaimScheduleIndex[];
}
/**
 * Params defines the vesting_rewards module parameters.
 * @name Params
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.Params
 */
export interface Params {
  /**
   * Block rewards
   */
  blockReward: string;
  /**
   * per-epoch decay (default: 994,478 = 0.994478x on 1M scale)
   */
  rewardDecayBps: bigint;
  /**
   * blocks per decay epoch (default: 100,000)
   */
  blocksPerRewardEpoch: bigint;
  /**
   * Revenue split (governance-adjustable)
   */
  revenueSplit?: RevenueSplit;
  protocolSubSplit?: ProtocolSubSplit;
  /**
   * Compatibility-only founder fields. The tap is constitutionally renounced:
   * source validation requires 0/empty and governance cannot restore it.
   */
  founderShareBps: bigint;
  founderAddress: string;
  /**
   * DEPRECATED — retained for wire compatibility; not an activation gate
   */
  governanceActivationHeight: bigint;
  /**
   * Compatibility-only declarations. Current reward execution does not read
   * these entries; runtime updates must preserve them.
   */
  categoryRewardConfigs: CategoryRewardConfig[];
  /**
   * Compatibility-only. Runtime routes to the compiled research_fund module
   * account and rejects changing this field.
   */
  researchFundModuleAccount: string;
  /**
   * Vesting parameters
   * Compatibility-only: not an execution gate; runtime updates must preserve it.
   */
  vestingEnabled: boolean;
  /**
   * 10,000 scale: default 3300 = 33% of released value
   */
  releasedClawbackRate: bigint;
  /**
   * target validator count for full block reward (default: 22)
   */
  minValidatorsForFullReward: number;
  /**
   * 10,000 scale: 10,000 = 100%; default 0
   */
  emptyBlockRewardRate: bigint;
  /**
   * minimum reward per block in uzrn (default: "100000" = 0.1 ZRN)
   */
  floorReward: string;
  /**
   * Genesis/import-export TotalMinted bookkeeping only; runtime updates must
   * preserve it and it is not a spendable fund balance.
   */
  initialFundBalance: string;
  /**
   * Survived-challenge-coupled block reward.
   * When enabled, block reward is multiplied by clamp(rate/target, floor, 1.0).
   * Rate = survived / (survived + disproven); unchallenged facts are excluded.
   * Below target → reward is reduced; at target → full reward.
   * 0 target = disabled (backward compat).
   */
  knowledgeCouplingTargetBps: bigint;
  /**
   * minimum reward multiplier in BPS (default: 500,000 = 50%)
   */
  knowledgeCouplingFloorBps: bigint;
}
/**
 * CategoryRewardConfig defines a per-category block reward multiplier.
 * @name CategoryRewardConfig
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.CategoryRewardConfig
 */
export interface CategoryRewardConfig {
  /**
   * epistemic category name
   */
  category: string;
  /**
   * 1,000,000 = 1.0x
   */
  multiplierBps: bigint;
}
function createBaseGenesisState(): GenesisState {
  return {
    params: undefined,
    categoryConfigs: [],
    vestingSchedules: [],
    clawbackRecords: [],
    blockRewardDistributions: [],
    claimScheduleIndexes: []
  };
}
/**
 * GenesisState defines the vesting_rewards module's genesis state.
 * @name GenesisState
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.GenesisState
 */
export const GenesisState = {
  typeUrl: "/zerone.vesting_rewards.v1.GenesisState",
  encode(message: GenesisState, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.categoryConfigs) {
      CategoryConfig.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.vestingSchedules) {
      VestingSchedule.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.clawbackRecords) {
      ClawbackRecord.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.blockRewardDistributions) {
      BlockRewardDistribution.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    for (const v of message.claimScheduleIndexes) {
      ClaimScheduleIndex.encode(v!, writer.uint32(50).fork()).ldelim();
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
          message.categoryConfigs.push(CategoryConfig.decode(reader, reader.uint32()));
          break;
        case 3:
          message.vestingSchedules.push(VestingSchedule.decode(reader, reader.uint32()));
          break;
        case 4:
          message.clawbackRecords.push(ClawbackRecord.decode(reader, reader.uint32()));
          break;
        case 5:
          message.blockRewardDistributions.push(BlockRewardDistribution.decode(reader, reader.uint32()));
          break;
        case 6:
          message.claimScheduleIndexes.push(ClaimScheduleIndex.decode(reader, reader.uint32()));
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
    message.categoryConfigs = object.categoryConfigs?.map(e => CategoryConfig.fromPartial(e)) || [];
    message.vestingSchedules = object.vestingSchedules?.map(e => VestingSchedule.fromPartial(e)) || [];
    message.clawbackRecords = object.clawbackRecords?.map(e => ClawbackRecord.fromPartial(e)) || [];
    message.blockRewardDistributions = object.blockRewardDistributions?.map(e => BlockRewardDistribution.fromPartial(e)) || [];
    message.claimScheduleIndexes = object.claimScheduleIndexes?.map(e => ClaimScheduleIndex.fromPartial(e)) || [];
    return message;
  }
};
function createBaseParams(): Params {
  return {
    blockReward: "",
    rewardDecayBps: BigInt(0),
    blocksPerRewardEpoch: BigInt(0),
    revenueSplit: undefined,
    protocolSubSplit: undefined,
    founderShareBps: BigInt(0),
    founderAddress: "",
    governanceActivationHeight: BigInt(0),
    categoryRewardConfigs: [],
    researchFundModuleAccount: "",
    vestingEnabled: false,
    releasedClawbackRate: BigInt(0),
    minValidatorsForFullReward: 0,
    emptyBlockRewardRate: BigInt(0),
    floorReward: "",
    initialFundBalance: "",
    knowledgeCouplingTargetBps: BigInt(0),
    knowledgeCouplingFloorBps: BigInt(0)
  };
}
/**
 * Params defines the vesting_rewards module parameters.
 * @name Params
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.Params
 */
export const Params = {
  typeUrl: "/zerone.vesting_rewards.v1.Params",
  encode(message: Params, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.blockReward !== "") {
      writer.uint32(10).string(message.blockReward);
    }
    if (message.rewardDecayBps !== BigInt(0)) {
      writer.uint32(16).uint64(message.rewardDecayBps);
    }
    if (message.blocksPerRewardEpoch !== BigInt(0)) {
      writer.uint32(24).uint64(message.blocksPerRewardEpoch);
    }
    if (message.revenueSplit !== undefined) {
      RevenueSplit.encode(message.revenueSplit, writer.uint32(34).fork()).ldelim();
    }
    if (message.protocolSubSplit !== undefined) {
      ProtocolSubSplit.encode(message.protocolSubSplit, writer.uint32(42).fork()).ldelim();
    }
    if (message.founderShareBps !== BigInt(0)) {
      writer.uint32(48).uint64(message.founderShareBps);
    }
    if (message.founderAddress !== "") {
      writer.uint32(58).string(message.founderAddress);
    }
    if (message.governanceActivationHeight !== BigInt(0)) {
      writer.uint32(64).uint64(message.governanceActivationHeight);
    }
    for (const v of message.categoryRewardConfigs) {
      CategoryRewardConfig.encode(v!, writer.uint32(74).fork()).ldelim();
    }
    if (message.researchFundModuleAccount !== "") {
      writer.uint32(82).string(message.researchFundModuleAccount);
    }
    if (message.vestingEnabled === true) {
      writer.uint32(88).bool(message.vestingEnabled);
    }
    if (message.releasedClawbackRate !== BigInt(0)) {
      writer.uint32(96).uint64(message.releasedClawbackRate);
    }
    if (message.minValidatorsForFullReward !== 0) {
      writer.uint32(104).uint32(message.minValidatorsForFullReward);
    }
    if (message.emptyBlockRewardRate !== BigInt(0)) {
      writer.uint32(112).uint64(message.emptyBlockRewardRate);
    }
    if (message.floorReward !== "") {
      writer.uint32(122).string(message.floorReward);
    }
    if (message.initialFundBalance !== "") {
      writer.uint32(130).string(message.initialFundBalance);
    }
    if (message.knowledgeCouplingTargetBps !== BigInt(0)) {
      writer.uint32(136).uint64(message.knowledgeCouplingTargetBps);
    }
    if (message.knowledgeCouplingFloorBps !== BigInt(0)) {
      writer.uint32(144).uint64(message.knowledgeCouplingFloorBps);
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
        case 1:
          message.blockReward = reader.string();
          break;
        case 2:
          message.rewardDecayBps = reader.uint64();
          break;
        case 3:
          message.blocksPerRewardEpoch = reader.uint64();
          break;
        case 4:
          message.revenueSplit = RevenueSplit.decode(reader, reader.uint32());
          break;
        case 5:
          message.protocolSubSplit = ProtocolSubSplit.decode(reader, reader.uint32());
          break;
        case 6:
          message.founderShareBps = reader.uint64();
          break;
        case 7:
          message.founderAddress = reader.string();
          break;
        case 8:
          message.governanceActivationHeight = reader.uint64();
          break;
        case 9:
          message.categoryRewardConfigs.push(CategoryRewardConfig.decode(reader, reader.uint32()));
          break;
        case 10:
          message.researchFundModuleAccount = reader.string();
          break;
        case 11:
          message.vestingEnabled = reader.bool();
          break;
        case 12:
          message.releasedClawbackRate = reader.uint64();
          break;
        case 13:
          message.minValidatorsForFullReward = reader.uint32();
          break;
        case 14:
          message.emptyBlockRewardRate = reader.uint64();
          break;
        case 15:
          message.floorReward = reader.string();
          break;
        case 16:
          message.initialFundBalance = reader.string();
          break;
        case 17:
          message.knowledgeCouplingTargetBps = reader.uint64();
          break;
        case 18:
          message.knowledgeCouplingFloorBps = reader.uint64();
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
    message.blockReward = object.blockReward ?? "";
    message.rewardDecayBps = object.rewardDecayBps !== undefined && object.rewardDecayBps !== null ? BigInt(object.rewardDecayBps.toString()) : BigInt(0);
    message.blocksPerRewardEpoch = object.blocksPerRewardEpoch !== undefined && object.blocksPerRewardEpoch !== null ? BigInt(object.blocksPerRewardEpoch.toString()) : BigInt(0);
    message.revenueSplit = object.revenueSplit !== undefined && object.revenueSplit !== null ? RevenueSplit.fromPartial(object.revenueSplit) : undefined;
    message.protocolSubSplit = object.protocolSubSplit !== undefined && object.protocolSubSplit !== null ? ProtocolSubSplit.fromPartial(object.protocolSubSplit) : undefined;
    message.founderShareBps = object.founderShareBps !== undefined && object.founderShareBps !== null ? BigInt(object.founderShareBps.toString()) : BigInt(0);
    message.founderAddress = object.founderAddress ?? "";
    message.governanceActivationHeight = object.governanceActivationHeight !== undefined && object.governanceActivationHeight !== null ? BigInt(object.governanceActivationHeight.toString()) : BigInt(0);
    message.categoryRewardConfigs = object.categoryRewardConfigs?.map(e => CategoryRewardConfig.fromPartial(e)) || [];
    message.researchFundModuleAccount = object.researchFundModuleAccount ?? "";
    message.vestingEnabled = object.vestingEnabled ?? false;
    message.releasedClawbackRate = object.releasedClawbackRate !== undefined && object.releasedClawbackRate !== null ? BigInt(object.releasedClawbackRate.toString()) : BigInt(0);
    message.minValidatorsForFullReward = object.minValidatorsForFullReward ?? 0;
    message.emptyBlockRewardRate = object.emptyBlockRewardRate !== undefined && object.emptyBlockRewardRate !== null ? BigInt(object.emptyBlockRewardRate.toString()) : BigInt(0);
    message.floorReward = object.floorReward ?? "";
    message.initialFundBalance = object.initialFundBalance ?? "";
    message.knowledgeCouplingTargetBps = object.knowledgeCouplingTargetBps !== undefined && object.knowledgeCouplingTargetBps !== null ? BigInt(object.knowledgeCouplingTargetBps.toString()) : BigInt(0);
    message.knowledgeCouplingFloorBps = object.knowledgeCouplingFloorBps !== undefined && object.knowledgeCouplingFloorBps !== null ? BigInt(object.knowledgeCouplingFloorBps.toString()) : BigInt(0);
    return message;
  }
};
function createBaseCategoryRewardConfig(): CategoryRewardConfig {
  return {
    category: "",
    multiplierBps: BigInt(0)
  };
}
/**
 * CategoryRewardConfig defines a per-category block reward multiplier.
 * @name CategoryRewardConfig
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.CategoryRewardConfig
 */
export const CategoryRewardConfig = {
  typeUrl: "/zerone.vesting_rewards.v1.CategoryRewardConfig",
  encode(message: CategoryRewardConfig, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.category !== "") {
      writer.uint32(10).string(message.category);
    }
    if (message.multiplierBps !== BigInt(0)) {
      writer.uint32(16).uint64(message.multiplierBps);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): CategoryRewardConfig {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCategoryRewardConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.category = reader.string();
          break;
        case 2:
          message.multiplierBps = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<CategoryRewardConfig>): CategoryRewardConfig {
    const message = createBaseCategoryRewardConfig();
    message.category = object.category ?? "";
    message.multiplierBps = object.multiplierBps !== undefined && object.multiplierBps !== null ? BigInt(object.multiplierBps.toString()) : BigInt(0);
    return message;
  }
};
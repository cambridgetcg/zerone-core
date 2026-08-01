import { CategoryConfig, VestingSchedule, ClawbackRecord, BlockRewardDistribution, ClaimScheduleIndex } from "./state.js";
import { RevenueSplit, ProtocolSubSplit } from "../../common/v1/common.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
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
/**
 * GenesisState defines the vesting_rewards module's genesis state.
 * @name GenesisState
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.GenesisState
 */
export declare const GenesisState: {
    typeUrl: string;
    encode(message: GenesisState, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GenesisState;
    fromPartial(object: DeepPartial<GenesisState>): GenesisState;
};
/**
 * Params defines the vesting_rewards module parameters.
 * @name Params
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.Params
 */
export declare const Params: {
    typeUrl: string;
    encode(message: Params, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params;
    fromPartial(object: DeepPartial<Params>): Params;
};
/**
 * CategoryRewardConfig defines a per-category block reward multiplier.
 * @name CategoryRewardConfig
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.CategoryRewardConfig
 */
export declare const CategoryRewardConfig: {
    typeUrl: string;
    encode(message: CategoryRewardConfig, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): CategoryRewardConfig;
    fromPartial(object: DeepPartial<CategoryRewardConfig>): CategoryRewardConfig;
};

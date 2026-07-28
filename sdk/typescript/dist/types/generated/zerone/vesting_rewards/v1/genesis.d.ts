import { CategoryConfig, VestingSchedule } from "./state.js";
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
     * per-epoch decay (default: 850,000 = 0.85x on 1M scale)
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
     * Founder share
     */
    founderShareBps: bigint;
    /**
     * bech32, empty = disabled
     */
    founderAddress: string;
    /**
     * DEPRECATED — founder share is governance-immune
     */
    governanceActivationHeight: bigint;
    /**
     * Category reward multipliers
     */
    categoryRewardConfigs: CategoryRewardConfig[];
    /**
     * Research fund
     */
    researchFundModuleAccount: string;
    /**
     * Vesting parameters
     */
    vestingEnabled: boolean;
    /**
     * bps of released clawed back on falsification (default: 3300 = 33%)
     */
    releasedClawbackRate: bigint;
    /**
     * target validator count for full block reward (default: 22)
     */
    minValidatorsForFullReward: number;
    /**
     * bps of reward for empty blocks (default: 0)
     */
    emptyBlockRewardRate: bigint;
    /**
     * minimum reward per block in uzrn (default: "100000" = 0.1 ZRN)
     */
    floorReward: string;
    /**
     * uzrn total fund at genesis (default: "0" = pure PoT)
     */
    initialFundBalance: string;
    /**
     * Knowledge-coupled block reward (T9 / thesis claim 1).
     * When enabled, block reward is multiplied by clamp(rate/target, floor, 1.0).
     * Below target verification rate → reward decays faster; at target → full reward.
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

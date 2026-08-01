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
     * Retired transaction-presence block-reward fields. Consensus v2 requires
     * block_reward and floor_reward to be "0" and empty_block_reward_rate to be
     * zero. The remaining schedule fields are inert compatibility metadata.
     */
    blockReward: string;
    rewardDecayBps: bigint;
    blocksPerRewardEpoch: bigint;
    /**
     * Revenue split (governance-adjustable)
     */
    revenueSplit?: RevenueSplit;
    protocolSubSplit?: ProtocolSubSplit;
    /**
     * Retired compatibility fields. Consensus v2 requires zero and empty;
     * governance cannot recreate an identity-based founder revenue tap.
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
     * inert compatibility metadata
     */
    minValidatorsForFullReward: number;
    /**
     * retired; must be zero
     */
    emptyBlockRewardRate: bigint;
    /**
     * retired; must be "0"
     */
    floorReward: string;
    /**
     * Genesis/import-export TotalMinted bookkeeping only; runtime updates must
     * preserve it and it is not a spendable fund balance.
     */
    initialFundBalance: string;
    /**
     * Inert compatibility metadata from the retired block-reward path.
     */
    knowledgeCouplingTargetBps: bigint;
    /**
     * minimum reward multiplier in BPS (default: 500,000 = 50%)
     */
    knowledgeCouplingFloorBps: bigint;
}
/**
 * CategoryRewardConfig preserves a retired per-category reward multiplier.
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
 * CategoryRewardConfig preserves a retired per-category reward multiplier.
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

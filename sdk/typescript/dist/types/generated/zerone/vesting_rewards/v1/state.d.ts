import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * VestingSchedule represents a truth-linked reward vesting record.
 * @name VestingSchedule
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.VestingSchedule
 */
export interface VestingSchedule {
    /**
     * unique vesting ID
     */
    id: string;
    /**
     * the claim this reward is for
     */
    claimId: string;
    /**
     * the fact (once claim is accepted)
     */
    factId: string;
    /**
     * bech32 address of reward recipient
     */
    recipient: string;
    /**
     * total reward amount in uzrn (bigint as string)
     */
    totalAmount: string;
    /**
     * amount already released (bigint as string)
     */
    releasedAmount: string;
    /**
     * amount currently claimable (bigint as string)
     */
    claimableAmount: string;
    /**
     * permanent challenge reserve (bigint as string)
     */
    reserveAmount: string;
    /**
     * epistemic category: "axiomatic", "formal_proof", "on_chain", etc.
     */
    category: string;
    /**
     * reward origin: "block_production", "verification", "api_call", etc.
     */
    source: string;
    /**
     * "vesting", "paused", "completed", "falsified", "abandoned"
     */
    status: string;
    /**
     * when claim was accepted
     */
    acceptedAtBlock: bigint;
    /**
     * when cliff period ends
     */
    cliffEndsAtBlock: bigint;
    /**
     * last block rewards were claimed
     */
    lastClaimBlock: bigint;
    /**
     * total blocks spent paused
     */
    totalPausedBlocks: bigint;
    /**
     * block when paused (0 if not paused)
     */
    pausedAtBlock: bigint;
    /**
     * successful defenses (accelerates)
     */
    defenseCount: number;
    /**
     * independent replications (accelerates)
     */
    replicationCount: number;
    /**
     * block height when created
     */
    createdAt: bigint;
    /**
     * block height when last updated
     */
    updatedAt: bigint;
    /**
     * cross-domain corroborations (accelerates)
     */
    corroborationCount: number;
    /**
     * citations by other accepted facts (accelerates)
     */
    citationCount: number;
}
/**
 * ClaimScheduleIndex preserves the current claim_id -> vesting_id lookup.
 * Multiple legacy schedules may share a claim ID, so schedule replay order
 * alone cannot reconstruct which schedule the live index selected.
 * @name ClaimScheduleIndex
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.ClaimScheduleIndex
 */
export interface ClaimScheduleIndex {
    claimId: string;
    vestingId: string;
}
/**
 * CategoryConfig defines release curve parameters for a vesting category.
 * @name CategoryConfig
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.CategoryConfig
 */
export interface CategoryConfig {
    /**
     * epistemic category name
     */
    category: string;
    /**
     * blocks until half of reward is released
     */
    halfLifeBlocks: bigint;
    /**
     * blocks before any release begins
     */
    cliffBlocks: bigint;
    /**
     * basis points (950000 = 95%, rest is permanent reserve)
     */
    maxRelease: bigint;
}
/**
 * RewardRouting represents a routed reward with full revenue split applied.
 * @name RewardRouting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.RewardRouting
 */
export interface RewardRouting {
    /**
     * reward source
     */
    source: string;
    /**
     * original reward amount (bigint as string)
     */
    originalAmount: string;
    /**
     * contributor's share (bigint as string)
     */
    contributorShare: string;
    /**
     * protocol share (bigint as string)
     */
    protocolShare: string;
    /**
     * research fund share (bigint as string)
     */
    researchShare: string;
    /**
     * development fund amount (bigint as string)
     */
    developmentAmount: string;
    /**
     * bech32 recipient address
     */
    recipient: string;
    /**
     * related fact
     */
    factId: string;
    /**
     * block height
     */
    blockNumber: bigint;
    /**
     * Compatibility field: zero in v2+ outputs; legacy records may be nonzero.
     */
    founderShare: string;
    /**
     * protocol sub-split: citations (bigint as string)
     */
    citationPool: string;
    /**
     * protocol sub-split: verification (bigint as string)
     */
    verificationPool: string;
    /**
     * protocol sub-split: treasury (bigint as string)
     */
    treasuryShare: string;
}
/**
 * BlockRewardDistribution records block reward distribution for a specific block.
 * @name BlockRewardDistribution
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.BlockRewardDistribution
 */
export interface BlockRewardDistribution {
    /**
     * block number
     */
    blockHeight: bigint;
    /**
     * reward to block producer (bigint as string)
     */
    producerReward: string;
    /**
     * research fund share (bigint as string)
     */
    researchShare: string;
    /**
     * total new tokens (bigint as string)
     */
    totalMinted: string;
    /**
     * active validators
     */
    validatorCount: number;
    /**
     * Legacy field name: cumulative shared MintWithCap accounting ledger after
     * this distribution, not a spendable fund balance or full supply history.
     */
    fundBalanceAfter: string;
    /**
     * Compatibility field: zero in v2+ records; preserved legacy records may be nonzero.
     */
    founderShare: string;
    /**
     * development fund amount (bigint as string)
     */
    developmentAmount: string;
    /**
     * protocol share (bigint as string)
     */
    protocolShare: string;
}
/**
 * ClawbackRecord records a reward clawback due to falsification.
 * @name ClawbackRecord
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.ClawbackRecord
 */
export interface ClawbackRecord {
    /**
     * unique record ID
     */
    id: string;
    /**
     * vesting schedule being clawed back
     */
    vestingId: string;
    /**
     * amount clawed from already-released (bigint as string)
     */
    releasedClawback: string;
    /**
     * unvested amount forfeited (bigint as string)
     */
    unvestedForfeited: string;
    /**
     * reserve forfeited (bigint as string)
     */
    reserveForfeited: string;
    /**
     * amount given to challenger (bigint as string)
     */
    challengerReward: string;
    /**
     * when clawback occurred
     */
    blockHeight: bigint;
}
/**
 * VestingSchedule represents a truth-linked reward vesting record.
 * @name VestingSchedule
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.VestingSchedule
 */
export declare const VestingSchedule: {
    typeUrl: string;
    encode(message: VestingSchedule, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): VestingSchedule;
    fromPartial(object: DeepPartial<VestingSchedule>): VestingSchedule;
};
/**
 * ClaimScheduleIndex preserves the current claim_id -> vesting_id lookup.
 * Multiple legacy schedules may share a claim ID, so schedule replay order
 * alone cannot reconstruct which schedule the live index selected.
 * @name ClaimScheduleIndex
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.ClaimScheduleIndex
 */
export declare const ClaimScheduleIndex: {
    typeUrl: string;
    encode(message: ClaimScheduleIndex, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ClaimScheduleIndex;
    fromPartial(object: DeepPartial<ClaimScheduleIndex>): ClaimScheduleIndex;
};
/**
 * CategoryConfig defines release curve parameters for a vesting category.
 * @name CategoryConfig
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.CategoryConfig
 */
export declare const CategoryConfig: {
    typeUrl: string;
    encode(message: CategoryConfig, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): CategoryConfig;
    fromPartial(object: DeepPartial<CategoryConfig>): CategoryConfig;
};
/**
 * RewardRouting represents a routed reward with full revenue split applied.
 * @name RewardRouting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.RewardRouting
 */
export declare const RewardRouting: {
    typeUrl: string;
    encode(message: RewardRouting, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): RewardRouting;
    fromPartial(object: DeepPartial<RewardRouting>): RewardRouting;
};
/**
 * BlockRewardDistribution records block reward distribution for a specific block.
 * @name BlockRewardDistribution
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.BlockRewardDistribution
 */
export declare const BlockRewardDistribution: {
    typeUrl: string;
    encode(message: BlockRewardDistribution, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): BlockRewardDistribution;
    fromPartial(object: DeepPartial<BlockRewardDistribution>): BlockRewardDistribution;
};
/**
 * ClawbackRecord records a reward clawback due to falsification.
 * @name ClawbackRecord
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.ClawbackRecord
 */
export declare const ClawbackRecord: {
    typeUrl: string;
    encode(message: ClawbackRecord, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ClawbackRecord;
    fromPartial(object: DeepPartial<ClawbackRecord>): ClawbackRecord;
};

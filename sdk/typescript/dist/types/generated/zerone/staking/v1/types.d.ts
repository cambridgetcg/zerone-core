import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/** ValidatorTier enumerates the 4-tier graduated validator system. */
export declare enum ValidatorTier {
    VALIDATOR_TIER_UNSPECIFIED = 0,
    VALIDATOR_TIER_APPRENTICE = 1,
    VALIDATOR_TIER_VERIFIED = 2,
    VALIDATOR_TIER_SCHOLAR = 3,
    VALIDATOR_TIER_GUARDIAN = 4,
    UNRECOGNIZED = -1
}
export declare function validatorTierFromJSON(object: any): ValidatorTier;
export declare function validatorTierToJSON(object: ValidatorTier): string;
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
/**
 * Validator represents a Zerone validator with tier and reputation.
 * @name Validator
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.Validator
 */
export declare const Validator: {
    typeUrl: string;
    encode(message: Validator, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Validator;
    fromPartial(object: DeepPartial<Validator>): Validator;
};
/**
 * Delegation records a delegation from delegator to validator.
 * @name Delegation
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.Delegation
 */
export declare const Delegation: {
    typeUrl: string;
    encode(message: Delegation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Delegation;
    fromPartial(object: DeepPartial<Delegation>): Delegation;
};
/**
 * UnbondingEntry records a pending unbonding.
 * @name UnbondingEntry
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.UnbondingEntry
 */
export declare const UnbondingEntry: {
    typeUrl: string;
    encode(message: UnbondingEntry, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): UnbondingEntry;
    fromPartial(object: DeepPartial<UnbondingEntry>): UnbondingEntry;
};
/**
 * TierConfig defines requirements and rewards for each validator tier.
 * @name TierConfig
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.TierConfig
 */
export declare const TierConfig: {
    typeUrl: string;
    encode(message: TierConfig, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): TierConfig;
    fromPartial(object: DeepPartial<TierConfig>): TierConfig;
};

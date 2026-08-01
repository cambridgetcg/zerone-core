import { ResearchFundVoters, UpgradePlan, LIP, Vote, ResearchFundGovernanceState, SeatElectionProposal, SeatElectionVote } from "./types.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * Params defines the governance module parameters.
 * @name Params
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.Params
 */
export interface Params {
    /**
     * blocks for voting stage (default 102816)
     */
    votingPeriodBlocks: bigint;
    /**
     * blocks for last_call stage (default 68544)
     */
    discussionPeriodBlocks: bigint;
    /**
     * on 1,000,000 scale (default 334000 = 33.4%)
     */
    quorumThresholdBps: bigint;
    /**
     * on 1,000,000 scale (default 500000 = 50%)
     */
    supportThresholdBps: bigint;
    /**
     * uzrn minimum to submit a LIP
     */
    minLipStake: string;
    /**
     * uzrn minimum bonded stake to vote
     */
    minVoteStake: string;
    categoryConfigs: CategoryConfig[];
    researchFundVoters?: ResearchFundVoters;
    /**
     * discussion period for research proposals
     */
    researchDiscussionBlocks: bigint;
    /**
     * voting period for research proposals
     */
    researchVotingBlocks: bigint;
}
/**
 * CategoryConfig defines per-category stake and review requirements.
 * @name CategoryConfig
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.CategoryConfig
 */
export interface CategoryConfig {
    /**
     * "parameter", "upgrade", "text", "research_spend"
     */
    category: string;
    /**
     * uzrn required to advance from draft to review
     */
    requiredStakeUzrn: string;
    /**
     * blocks required in review stage before last_call
     */
    reviewBlocks: bigint;
}
/**
 * GenesisUpgradePlan pairs a LIP ID with its attached upgrade plan for genesis export/import.
 * @name GenesisUpgradePlan
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.GenesisUpgradePlan
 */
export interface GenesisUpgradePlan {
    lipId: string;
    plan?: UpgradePlan;
}
/**
 * GenesisCreedAmendmentPin pairs a LIP ID with its attached
 * creed-amendment payload for genesis export/import.
 * @name GenesisCreedAmendmentPin
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.GenesisCreedAmendmentPin
 */
export interface GenesisCreedAmendmentPin {
    lipId: string;
    canonicalHash: Uint8Array;
    commitmentsJson: Uint8Array;
}
/**
 * EmergencyTransitionHold is the durable post-incident review gate for every
 * automatic custom-governance transition. It is created when transaction
 * quarantine is observed and is not cleared by an ordinary resume ceremony.
 * @name EmergencyTransitionHold
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.EmergencyTransitionHold
 */
export interface EmergencyTransitionHold {
    incidentId: string;
    activatedAtBlock: bigint;
    /**
     * Most recently observed quarantine incident. incident_id remains the first
     * incident for compact API compatibility.
     */
    latestIncidentId: string;
    /**
     * Number of chronological incident observations committed by this hold.
     */
    incidentCount: bigint;
    /**
     * Domain-separated rolling SHA-256 commitment to the complete chronological
     * incident lineage. This keeps consensus state bounded even if a review hold
     * spans many independently finalized quarantine incidents.
     */
    incidentLineageSha256: Uint8Array;
}
/**
 * GenesisState defines the governance module's genesis state.
 * @name GenesisState
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.GenesisState
 */
export interface GenesisState {
    params?: Params;
    lips: LIP[];
    votes: Vote[];
    nextLipNumber: bigint;
    upgradePlans: GenesisUpgradePlan[];
    researchFundGovernance?: ResearchFundGovernanceState;
    seatElections: SeatElectionProposal[];
    seatElectionVotes: SeatElectionVote[];
    nextSeatElectionNumber: bigint;
    creedAmendmentPins: GenesisCreedAmendmentPin[];
    emergencyTransitionHold?: EmergencyTransitionHold;
}
/**
 * Params defines the governance module parameters.
 * @name Params
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.Params
 */
export declare const Params: {
    typeUrl: string;
    encode(message: Params, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params;
    fromPartial(object: DeepPartial<Params>): Params;
};
/**
 * CategoryConfig defines per-category stake and review requirements.
 * @name CategoryConfig
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.CategoryConfig
 */
export declare const CategoryConfig: {
    typeUrl: string;
    encode(message: CategoryConfig, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): CategoryConfig;
    fromPartial(object: DeepPartial<CategoryConfig>): CategoryConfig;
};
/**
 * GenesisUpgradePlan pairs a LIP ID with its attached upgrade plan for genesis export/import.
 * @name GenesisUpgradePlan
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.GenesisUpgradePlan
 */
export declare const GenesisUpgradePlan: {
    typeUrl: string;
    encode(message: GenesisUpgradePlan, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GenesisUpgradePlan;
    fromPartial(object: DeepPartial<GenesisUpgradePlan>): GenesisUpgradePlan;
};
/**
 * GenesisCreedAmendmentPin pairs a LIP ID with its attached
 * creed-amendment payload for genesis export/import.
 * @name GenesisCreedAmendmentPin
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.GenesisCreedAmendmentPin
 */
export declare const GenesisCreedAmendmentPin: {
    typeUrl: string;
    encode(message: GenesisCreedAmendmentPin, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GenesisCreedAmendmentPin;
    fromPartial(object: DeepPartial<GenesisCreedAmendmentPin>): GenesisCreedAmendmentPin;
};
/**
 * EmergencyTransitionHold is the durable post-incident review gate for every
 * automatic custom-governance transition. It is created when transaction
 * quarantine is observed and is not cleared by an ordinary resume ceremony.
 * @name EmergencyTransitionHold
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.EmergencyTransitionHold
 */
export declare const EmergencyTransitionHold: {
    typeUrl: string;
    encode(message: EmergencyTransitionHold, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): EmergencyTransitionHold;
    fromPartial(object: DeepPartial<EmergencyTransitionHold>): EmergencyTransitionHold;
};
/**
 * GenesisState defines the governance module's genesis state.
 * @name GenesisState
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.GenesisState
 */
export declare const GenesisState: {
    typeUrl: string;
    encode(message: GenesisState, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GenesisState;
    fromPartial(object: DeepPartial<GenesisState>): GenesisState;
};

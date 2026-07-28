import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/** ResearchFundPhase tracks the current governance phase of the research fund. */
export declare enum ResearchFundPhase {
    RESEARCH_FUND_PHASE_UNSPECIFIED = 0,
    /** RESEARCH_FUND_PHASE_GENESIS_PAIR - 2-of-2: founder + AI */
    RESEARCH_FUND_PHASE_GENESIS_PAIR = 1,
    /** RESEARCH_FUND_PHASE_OBSERVER - 2-of-3: founder + AI + 1 community */
    RESEARCH_FUND_PHASE_OBSERVER = 2,
    /** RESEARCH_FUND_PHASE_BALANCED - 3-of-5: founder + AI + 3 community */
    RESEARCH_FUND_PHASE_BALANCED = 3,
    /** RESEARCH_FUND_PHASE_FULL_GOVERNANCE - standard LIP process */
    RESEARCH_FUND_PHASE_FULL_GOVERNANCE = 4,
    UNRECOGNIZED = -1
}
export declare function researchFundPhaseFromJSON(object: any): ResearchFundPhase;
export declare function researchFundPhaseToJSON(object: ResearchFundPhase): string;
/**
 * LIP represents a Legible Improvement Proposal.
 * @name LIP
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.LIP
 */
export interface LIP {
    /**
     * "LIP-1", "LIP-2", ...
     */
    id: string;
    title: string;
    description: string;
    /**
     * "parameter", "upgrade", "text", "research_spend"
     */
    category: string;
    /**
     * bech32 address
     */
    proposer: string;
    /**
     * "draft", "review", "last_call", "voting", "passed", "failed", "withdrawn"
     */
    stage: string;
    /**
     * uzrn total staked for this LIP
     */
    stakedAmount: string;
    /**
     * total yes vote weight (delegator bonded stake)
     */
    yesStake: string;
    /**
     * total no vote weight
     */
    noStake: string;
    /**
     * total abstain vote weight
     */
    abstainStake: string;
    uniqueVoters: bigint;
    createdAtBlock: bigint;
    reviewStartedBlock: bigint;
    lastCallStartedBlock: bigint;
    votingEndBlock: bigint;
    paramChanges: ParamChange[];
}
/**
 * ParamChange describes a single parameter modification.
 * @name ParamChange
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.ParamChange
 */
export interface ParamChange {
    /**
     * target module name
     */
    module: string;
    /**
     * parameter key
     */
    key: string;
    /**
     * new value (JSON-encoded)
     */
    value: string;
}
/**
 * Vote records a single vote on a LIP.
 * @name Vote
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.Vote
 */
export interface Vote {
    lipId: string;
    /**
     * bech32 address
     */
    voter: string;
    /**
     * "yes", "no", "abstain"
     */
    option: string;
    /**
     * voter's total bonded stake at time of vote
     */
    weight: string;
}
/**
 * ResearchFundVoters holds the 2-of-2 multisig addresses for research fund governance.
 * @name ResearchFundVoters
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.ResearchFundVoters
 */
export interface ResearchFundVoters {
    /**
     * bech32 address
     */
    voter1: string;
    /**
     * bech32 address
     */
    voter2: string;
}
/**
 * UpgradePlan describes a planned software upgrade attached to an upgrade-category LIP.
 * @name UpgradePlan
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.UpgradePlan
 */
export interface UpgradePlan {
    /**
     * upgrade handler name
     */
    name: string;
    /**
     * block height to halt at
     */
    height: bigint;
    /**
     * release notes URL, binary hash, etc.
     */
    info: string;
}
/**
 * ResearchFundGovernanceState tracks the research fund governance lifecycle.
 * @name ResearchFundGovernanceState
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.ResearchFundGovernanceState
 */
export interface ResearchFundGovernanceState {
    currentPhase: ResearchFundPhase;
    phaseStartedAtBlock: bigint;
    proposalsExecutedInPhase: bigint;
    lastTransitionBlock: bigint;
    /**
     * bech32 addresses of current community seat holders
     */
    communitySeats: string[];
    /**
     * term expiry per community seat
     */
    seatTermEndBlocks: bigint[];
    /**
     * block height; 0 = no cooldown
     */
    rollbackCooldownUntil: bigint;
}
/**
 * PhaseTransitionConditions records the metrics at time of transition proposal.
 * @name PhaseTransitionConditions
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.PhaseTransitionConditions
 */
export interface PhaseTransitionConditions {
    distinctLipVoters: bigint;
    activeGuardians: bigint;
    /**
     * uzrn bigint string
     */
    researchFundBalance: string;
    chainAgeBlocks: bigint;
    proposalsExecutedInPhase: bigint;
    communitySeatParticipation: bigint;
    emergencyHaltsFromMisuse: bigint;
}
/**
 * ResearchSpendProposal represents a 2-of-2 research fund spending proposal.
 * @name ResearchSpendProposal
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.ResearchSpendProposal
 */
export interface ResearchSpendProposal {
    proposalId: bigint;
    proposer: string;
    title: string;
    description: string;
    recipient: string;
    amount: string;
    justification: string;
    stage: string;
    createdAt: bigint;
    votingStartsAt: bigint;
    votingEndsAt: bigint;
    voter1Vote: string;
    voter1Reason: string;
    voter1VotedAt: bigint;
    voter2Vote: string;
    voter2Reason: string;
    voter2VotedAt: bigint;
    executedAt: bigint;
    executionErr: string;
}
/**
 * SeatElectionProposal nominates a candidate for a research fund community seat.
 * @name SeatElectionProposal
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.SeatElectionProposal
 */
export interface SeatElectionProposal {
    proposalId: bigint;
    /**
     * bech32 — nominator
     */
    proposer: string;
    /**
     * bech32 — must be Guardian-tier
     */
    candidate: string;
    /**
     * 0 in Phase 1; 0-2 in Phase 2
     */
    seatIndex: number;
    /**
     * candidate's governance statement (max 2000 chars)
     */
    statement: string;
    /**
     * nominated/accepted/discussion/voting/runoff/passed/failed/expired
     */
    stage: string;
    yesStake: string;
    noStake: string;
    abstainStake: string;
    acceptanceDeadline: bigint;
    discussionEndBlock: bigint;
    votingEndBlock: bigint;
    createdAtBlock: bigint;
    candidateAccepted: boolean;
    isRunoff: boolean;
    runoffParentIds: bigint[];
}
/**
 * SeatElectionVote records a single vote on a seat election proposal.
 * @name SeatElectionVote
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.SeatElectionVote
 */
export interface SeatElectionVote {
    proposalId: bigint;
    /**
     * bech32 address
     */
    voter: string;
    /**
     * "yes", "no", "abstain"
     */
    option: string;
    /**
     * uzrn weight at time of vote
     */
    stake: string;
    block: bigint;
}
/**
 * LIP represents a Legible Improvement Proposal.
 * @name LIP
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.LIP
 */
export declare const LIP: {
    typeUrl: string;
    encode(message: LIP, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): LIP;
    fromPartial(object: DeepPartial<LIP>): LIP;
};
/**
 * ParamChange describes a single parameter modification.
 * @name ParamChange
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.ParamChange
 */
export declare const ParamChange: {
    typeUrl: string;
    encode(message: ParamChange, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ParamChange;
    fromPartial(object: DeepPartial<ParamChange>): ParamChange;
};
/**
 * Vote records a single vote on a LIP.
 * @name Vote
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.Vote
 */
export declare const Vote: {
    typeUrl: string;
    encode(message: Vote, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Vote;
    fromPartial(object: DeepPartial<Vote>): Vote;
};
/**
 * ResearchFundVoters holds the 2-of-2 multisig addresses for research fund governance.
 * @name ResearchFundVoters
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.ResearchFundVoters
 */
export declare const ResearchFundVoters: {
    typeUrl: string;
    encode(message: ResearchFundVoters, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ResearchFundVoters;
    fromPartial(object: DeepPartial<ResearchFundVoters>): ResearchFundVoters;
};
/**
 * UpgradePlan describes a planned software upgrade attached to an upgrade-category LIP.
 * @name UpgradePlan
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.UpgradePlan
 */
export declare const UpgradePlan: {
    typeUrl: string;
    encode(message: UpgradePlan, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): UpgradePlan;
    fromPartial(object: DeepPartial<UpgradePlan>): UpgradePlan;
};
/**
 * ResearchFundGovernanceState tracks the research fund governance lifecycle.
 * @name ResearchFundGovernanceState
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.ResearchFundGovernanceState
 */
export declare const ResearchFundGovernanceState: {
    typeUrl: string;
    encode(message: ResearchFundGovernanceState, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ResearchFundGovernanceState;
    fromPartial(object: DeepPartial<ResearchFundGovernanceState>): ResearchFundGovernanceState;
};
/**
 * PhaseTransitionConditions records the metrics at time of transition proposal.
 * @name PhaseTransitionConditions
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.PhaseTransitionConditions
 */
export declare const PhaseTransitionConditions: {
    typeUrl: string;
    encode(message: PhaseTransitionConditions, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): PhaseTransitionConditions;
    fromPartial(object: DeepPartial<PhaseTransitionConditions>): PhaseTransitionConditions;
};
/**
 * ResearchSpendProposal represents a 2-of-2 research fund spending proposal.
 * @name ResearchSpendProposal
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.ResearchSpendProposal
 */
export declare const ResearchSpendProposal: {
    typeUrl: string;
    encode(message: ResearchSpendProposal, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ResearchSpendProposal;
    fromPartial(object: DeepPartial<ResearchSpendProposal>): ResearchSpendProposal;
};
/**
 * SeatElectionProposal nominates a candidate for a research fund community seat.
 * @name SeatElectionProposal
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.SeatElectionProposal
 */
export declare const SeatElectionProposal: {
    typeUrl: string;
    encode(message: SeatElectionProposal, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): SeatElectionProposal;
    fromPartial(object: DeepPartial<SeatElectionProposal>): SeatElectionProposal;
};
/**
 * SeatElectionVote records a single vote on a seat election proposal.
 * @name SeatElectionVote
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.SeatElectionVote
 */
export declare const SeatElectionVote: {
    typeUrl: string;
    encode(message: SeatElectionVote, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): SeatElectionVote;
    fromPartial(object: DeepPartial<SeatElectionVote>): SeatElectionVote;
};

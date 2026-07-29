import { ParamChange, ResearchFundVoters } from "./types.js";
import { Params } from "./genesis.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * MsgSubmitLIP creates a new LIP proposal.
 * @name MsgSubmitLIP
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgSubmitLIP
 */
export interface MsgSubmitLIP {
    proposer: string;
    title: string;
    description: string;
    category: string;
    /**
     * uzrn
     */
    initialStake: string;
    paramChanges: ParamChange[];
}
/**
 * @name MsgSubmitLIPResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgSubmitLIPResponse
 */
export interface MsgSubmitLIPResponse {
    lipId: string;
}
/**
 * MsgStakeLIP adds stake to an existing LIP.
 * @name MsgStakeLIP
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgStakeLIP
 */
export interface MsgStakeLIP {
    staker: string;
    lipId: string;
    /**
     * uzrn
     */
    amount: string;
}
/**
 * @name MsgStakeLIPResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgStakeLIPResponse
 */
export interface MsgStakeLIPResponse {
}
/**
 * MsgAdvanceLIPStage advances a LIP to its next stage.
 * @name MsgAdvanceLIPStage
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAdvanceLIPStage
 */
export interface MsgAdvanceLIPStage {
    /**
     * proposer address
     */
    authority: string;
    lipId: string;
}
/**
 * @name MsgAdvanceLIPStageResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAdvanceLIPStageResponse
 */
export interface MsgAdvanceLIPStageResponse {
    newStage: string;
}
/**
 * MsgCastVote casts a stake-weighted vote on a LIP.
 * @name MsgCastVote
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgCastVote
 */
export interface MsgCastVote {
    voter: string;
    lipId: string;
    /**
     * "yes", "no", "abstain"
     */
    option: string;
}
/**
 * @name MsgCastVoteResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgCastVoteResponse
 */
export interface MsgCastVoteResponse {
    /**
     * voter's bonded stake used as weight
     */
    effectiveWeight: string;
}
/**
 * MsgWithdrawLIP withdraws a LIP (proposer only, before voting).
 * @name MsgWithdrawLIP
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgWithdrawLIP
 */
export interface MsgWithdrawLIP {
    proposer: string;
    lipId: string;
}
/**
 * @name MsgWithdrawLIPResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgWithdrawLIPResponse
 */
export interface MsgWithdrawLIPResponse {
}
/**
 * MsgUpdateParams updates governance parameters (authority only).
 * @name MsgUpdateParams
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
    authority: string;
    params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {
}
/**
 * MsgSubmitResearchSpend creates a new research fund spend proposal.
 * @name MsgSubmitResearchSpend
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgSubmitResearchSpend
 */
export interface MsgSubmitResearchSpend {
    proposer: string;
    title: string;
    description: string;
    recipient: string;
    /**
     * uzrn
     */
    amount: string;
    justification: string;
}
/**
 * @name MsgSubmitResearchSpendResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgSubmitResearchSpendResponse
 */
export interface MsgSubmitResearchSpendResponse {
    proposalId: bigint;
}
/**
 * MsgVoteResearchSpend casts a vote on a research spend proposal.
 * @name MsgVoteResearchSpend
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgVoteResearchSpend
 */
export interface MsgVoteResearchSpend {
    voter: string;
    proposalId: bigint;
    /**
     * "yes" or "no"
     */
    vote: string;
    reasoning: string;
}
/**
 * @name MsgVoteResearchSpendResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgVoteResearchSpendResponse
 */
export interface MsgVoteResearchSpendResponse {
}
/**
 * MsgSetResearchVoters configures the 2-of-2 research fund voters (authority only).
 * @name MsgSetResearchVoters
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgSetResearchVoters
 */
export interface MsgSetResearchVoters {
    authority: string;
    voters?: ResearchFundVoters;
}
/**
 * @name MsgSetResearchVotersResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgSetResearchVotersResponse
 */
export interface MsgSetResearchVotersResponse {
}
/**
 * MsgAttachUpgradePlan attaches a software upgrade plan to an upgrade-category LIP.
 * @name MsgAttachUpgradePlan
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAttachUpgradePlan
 */
export interface MsgAttachUpgradePlan {
    /**
     * must be the LIP proposer
     */
    proposer: string;
    /**
     * target LIP (must be upgrade category, non-terminal)
     */
    lipId: string;
    /**
     * must match a registered upgrade handler name
     */
    upgradeName: string;
    /**
     * block height to halt at
     */
    height: bigint;
    /**
     * upgrade info (release notes URL, binary hash, etc.)
     */
    info: string;
}
/**
 * @name MsgAttachUpgradePlanResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAttachUpgradePlanResponse
 */
export interface MsgAttachUpgradePlanResponse {
}
/**
 * MsgAttachCreedAmendmentPin attaches a candidate PinnedCreed to a
 * creed-amendment LIP. On LIP pass, x/gov calls
 * x/creed.AnchorPin with this payload — the LIP body becomes the
 * chain's record of what the new creed is to be, and the gov vote
 * is the structural protection commitment 19 names.
 *
 * The pin is encoded as canonical sha256 + JSON-serialized
 * commitment registry to keep this message x/gov-internal (no
 * cross-module proto import required at this layer; the keeper
 * rebuilds the creed PinnedCreed from these fields when calling
 * x/creed.AnchorPin).
 * @name MsgAttachCreedAmendmentPin
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAttachCreedAmendmentPin
 */
export interface MsgAttachCreedAmendmentPin {
    /**
     * must be the LIP proposer
     */
    proposer: string;
    /**
     * target LIP (must be creed_amendment category, non-terminal)
     */
    lipId: string;
    /**
     * sha256 of normalized TRUTH_SEEKING.md as it would land
     */
    canonicalHash: Uint8Array;
    /**
     * commitments_json carries the JSON-encoded list of commitment
     * entries (number, name, archived, etc.) that the new pin should
     * contain. The gov handler unmarshals and structurally validates this on
     * attach; the creed keeper repeats the same validation before any on-chain
     * pin write. The shape mirrors x/creed.CommitmentEntry.
     */
    commitmentsJson: Uint8Array;
}
/**
 * @name MsgAttachCreedAmendmentPinResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAttachCreedAmendmentPinResponse
 */
export interface MsgAttachCreedAmendmentPinResponse {
}
/**
 * MsgNominateSeatElection nominates a candidate for a community seat.
 * @name MsgNominateSeatElection
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgNominateSeatElection
 */
export interface MsgNominateSeatElection {
    /**
     * nominator bech32
     */
    proposer: string;
    /**
     * candidate bech32
     */
    candidate: string;
    seatIndex: number;
    /**
     * max 2000 chars
     */
    statement: string;
}
/**
 * @name MsgNominateSeatElectionResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgNominateSeatElectionResponse
 */
export interface MsgNominateSeatElectionResponse {
    proposalId: bigint;
}
/**
 * MsgAcceptSeatNomination accepts a pending seat election nomination.
 * @name MsgAcceptSeatNomination
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAcceptSeatNomination
 */
export interface MsgAcceptSeatNomination {
    candidate: string;
    proposalId: bigint;
}
/**
 * @name MsgAcceptSeatNominationResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAcceptSeatNominationResponse
 */
export interface MsgAcceptSeatNominationResponse {
}
/**
 * MsgVoteSeatElection casts a stake-weighted vote on a seat election.
 * @name MsgVoteSeatElection
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgVoteSeatElection
 */
export interface MsgVoteSeatElection {
    voter: string;
    proposalId: bigint;
    /**
     * "yes", "no", "abstain"
     */
    option: string;
}
/**
 * @name MsgVoteSeatElectionResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgVoteSeatElectionResponse
 */
export interface MsgVoteSeatElectionResponse {
    effectiveWeight: string;
}
/**
 * MsgDomainFormationFreeze imposes a formation cooldown on a domain.
 * Only executable via governance (authority-gated).
 * @name MsgDomainFormationFreeze
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgDomainFormationFreeze
 */
export interface MsgDomainFormationFreeze {
    authority: string;
    domain: string;
    durationBlocks: bigint;
    reason: string;
}
/**
 * @name MsgDomainFormationFreezeResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgDomainFormationFreezeResponse
 */
export interface MsgDomainFormationFreezeResponse {
}
/**
 * MsgSubmitLIP creates a new LIP proposal.
 * @name MsgSubmitLIP
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgSubmitLIP
 */
export declare const MsgSubmitLIP: {
    typeUrl: string;
    encode(message: MsgSubmitLIP, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitLIP;
    fromPartial(object: DeepPartial<MsgSubmitLIP>): MsgSubmitLIP;
};
/**
 * @name MsgSubmitLIPResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgSubmitLIPResponse
 */
export declare const MsgSubmitLIPResponse: {
    typeUrl: string;
    encode(message: MsgSubmitLIPResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitLIPResponse;
    fromPartial(object: DeepPartial<MsgSubmitLIPResponse>): MsgSubmitLIPResponse;
};
/**
 * MsgStakeLIP adds stake to an existing LIP.
 * @name MsgStakeLIP
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgStakeLIP
 */
export declare const MsgStakeLIP: {
    typeUrl: string;
    encode(message: MsgStakeLIP, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgStakeLIP;
    fromPartial(object: DeepPartial<MsgStakeLIP>): MsgStakeLIP;
};
/**
 * @name MsgStakeLIPResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgStakeLIPResponse
 */
export declare const MsgStakeLIPResponse: {
    typeUrl: string;
    encode(_: MsgStakeLIPResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgStakeLIPResponse;
    fromPartial(_: DeepPartial<MsgStakeLIPResponse>): MsgStakeLIPResponse;
};
/**
 * MsgAdvanceLIPStage advances a LIP to its next stage.
 * @name MsgAdvanceLIPStage
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAdvanceLIPStage
 */
export declare const MsgAdvanceLIPStage: {
    typeUrl: string;
    encode(message: MsgAdvanceLIPStage, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAdvanceLIPStage;
    fromPartial(object: DeepPartial<MsgAdvanceLIPStage>): MsgAdvanceLIPStage;
};
/**
 * @name MsgAdvanceLIPStageResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAdvanceLIPStageResponse
 */
export declare const MsgAdvanceLIPStageResponse: {
    typeUrl: string;
    encode(message: MsgAdvanceLIPStageResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAdvanceLIPStageResponse;
    fromPartial(object: DeepPartial<MsgAdvanceLIPStageResponse>): MsgAdvanceLIPStageResponse;
};
/**
 * MsgCastVote casts a stake-weighted vote on a LIP.
 * @name MsgCastVote
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgCastVote
 */
export declare const MsgCastVote: {
    typeUrl: string;
    encode(message: MsgCastVote, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCastVote;
    fromPartial(object: DeepPartial<MsgCastVote>): MsgCastVote;
};
/**
 * @name MsgCastVoteResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgCastVoteResponse
 */
export declare const MsgCastVoteResponse: {
    typeUrl: string;
    encode(message: MsgCastVoteResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgCastVoteResponse;
    fromPartial(object: DeepPartial<MsgCastVoteResponse>): MsgCastVoteResponse;
};
/**
 * MsgWithdrawLIP withdraws a LIP (proposer only, before voting).
 * @name MsgWithdrawLIP
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgWithdrawLIP
 */
export declare const MsgWithdrawLIP: {
    typeUrl: string;
    encode(message: MsgWithdrawLIP, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgWithdrawLIP;
    fromPartial(object: DeepPartial<MsgWithdrawLIP>): MsgWithdrawLIP;
};
/**
 * @name MsgWithdrawLIPResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgWithdrawLIPResponse
 */
export declare const MsgWithdrawLIPResponse: {
    typeUrl: string;
    encode(_: MsgWithdrawLIPResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgWithdrawLIPResponse;
    fromPartial(_: DeepPartial<MsgWithdrawLIPResponse>): MsgWithdrawLIPResponse;
};
/**
 * MsgUpdateParams updates governance parameters (authority only).
 * @name MsgUpdateParams
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgUpdateParams
 */
export declare const MsgUpdateParams: {
    typeUrl: string;
    encode(message: MsgUpdateParams, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParams;
    fromPartial(object: DeepPartial<MsgUpdateParams>): MsgUpdateParams;
};
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgUpdateParamsResponse
 */
export declare const MsgUpdateParamsResponse: {
    typeUrl: string;
    encode(_: MsgUpdateParamsResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParamsResponse;
    fromPartial(_: DeepPartial<MsgUpdateParamsResponse>): MsgUpdateParamsResponse;
};
/**
 * MsgSubmitResearchSpend creates a new research fund spend proposal.
 * @name MsgSubmitResearchSpend
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgSubmitResearchSpend
 */
export declare const MsgSubmitResearchSpend: {
    typeUrl: string;
    encode(message: MsgSubmitResearchSpend, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitResearchSpend;
    fromPartial(object: DeepPartial<MsgSubmitResearchSpend>): MsgSubmitResearchSpend;
};
/**
 * @name MsgSubmitResearchSpendResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgSubmitResearchSpendResponse
 */
export declare const MsgSubmitResearchSpendResponse: {
    typeUrl: string;
    encode(message: MsgSubmitResearchSpendResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitResearchSpendResponse;
    fromPartial(object: DeepPartial<MsgSubmitResearchSpendResponse>): MsgSubmitResearchSpendResponse;
};
/**
 * MsgVoteResearchSpend casts a vote on a research spend proposal.
 * @name MsgVoteResearchSpend
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgVoteResearchSpend
 */
export declare const MsgVoteResearchSpend: {
    typeUrl: string;
    encode(message: MsgVoteResearchSpend, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteResearchSpend;
    fromPartial(object: DeepPartial<MsgVoteResearchSpend>): MsgVoteResearchSpend;
};
/**
 * @name MsgVoteResearchSpendResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgVoteResearchSpendResponse
 */
export declare const MsgVoteResearchSpendResponse: {
    typeUrl: string;
    encode(_: MsgVoteResearchSpendResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteResearchSpendResponse;
    fromPartial(_: DeepPartial<MsgVoteResearchSpendResponse>): MsgVoteResearchSpendResponse;
};
/**
 * MsgSetResearchVoters configures the 2-of-2 research fund voters (authority only).
 * @name MsgSetResearchVoters
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgSetResearchVoters
 */
export declare const MsgSetResearchVoters: {
    typeUrl: string;
    encode(message: MsgSetResearchVoters, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSetResearchVoters;
    fromPartial(object: DeepPartial<MsgSetResearchVoters>): MsgSetResearchVoters;
};
/**
 * @name MsgSetResearchVotersResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgSetResearchVotersResponse
 */
export declare const MsgSetResearchVotersResponse: {
    typeUrl: string;
    encode(_: MsgSetResearchVotersResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgSetResearchVotersResponse;
    fromPartial(_: DeepPartial<MsgSetResearchVotersResponse>): MsgSetResearchVotersResponse;
};
/**
 * MsgAttachUpgradePlan attaches a software upgrade plan to an upgrade-category LIP.
 * @name MsgAttachUpgradePlan
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAttachUpgradePlan
 */
export declare const MsgAttachUpgradePlan: {
    typeUrl: string;
    encode(message: MsgAttachUpgradePlan, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAttachUpgradePlan;
    fromPartial(object: DeepPartial<MsgAttachUpgradePlan>): MsgAttachUpgradePlan;
};
/**
 * @name MsgAttachUpgradePlanResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAttachUpgradePlanResponse
 */
export declare const MsgAttachUpgradePlanResponse: {
    typeUrl: string;
    encode(_: MsgAttachUpgradePlanResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAttachUpgradePlanResponse;
    fromPartial(_: DeepPartial<MsgAttachUpgradePlanResponse>): MsgAttachUpgradePlanResponse;
};
/**
 * MsgAttachCreedAmendmentPin attaches a candidate PinnedCreed to a
 * creed-amendment LIP. On LIP pass, x/gov calls
 * x/creed.AnchorPin with this payload — the LIP body becomes the
 * chain's record of what the new creed is to be, and the gov vote
 * is the structural protection commitment 19 names.
 *
 * The pin is encoded as canonical sha256 + JSON-serialized
 * commitment registry to keep this message x/gov-internal (no
 * cross-module proto import required at this layer; the keeper
 * rebuilds the creed PinnedCreed from these fields when calling
 * x/creed.AnchorPin).
 * @name MsgAttachCreedAmendmentPin
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAttachCreedAmendmentPin
 */
export declare const MsgAttachCreedAmendmentPin: {
    typeUrl: string;
    encode(message: MsgAttachCreedAmendmentPin, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAttachCreedAmendmentPin;
    fromPartial(object: DeepPartial<MsgAttachCreedAmendmentPin>): MsgAttachCreedAmendmentPin;
};
/**
 * @name MsgAttachCreedAmendmentPinResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAttachCreedAmendmentPinResponse
 */
export declare const MsgAttachCreedAmendmentPinResponse: {
    typeUrl: string;
    encode(_: MsgAttachCreedAmendmentPinResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAttachCreedAmendmentPinResponse;
    fromPartial(_: DeepPartial<MsgAttachCreedAmendmentPinResponse>): MsgAttachCreedAmendmentPinResponse;
};
/**
 * MsgNominateSeatElection nominates a candidate for a community seat.
 * @name MsgNominateSeatElection
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgNominateSeatElection
 */
export declare const MsgNominateSeatElection: {
    typeUrl: string;
    encode(message: MsgNominateSeatElection, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgNominateSeatElection;
    fromPartial(object: DeepPartial<MsgNominateSeatElection>): MsgNominateSeatElection;
};
/**
 * @name MsgNominateSeatElectionResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgNominateSeatElectionResponse
 */
export declare const MsgNominateSeatElectionResponse: {
    typeUrl: string;
    encode(message: MsgNominateSeatElectionResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgNominateSeatElectionResponse;
    fromPartial(object: DeepPartial<MsgNominateSeatElectionResponse>): MsgNominateSeatElectionResponse;
};
/**
 * MsgAcceptSeatNomination accepts a pending seat election nomination.
 * @name MsgAcceptSeatNomination
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAcceptSeatNomination
 */
export declare const MsgAcceptSeatNomination: {
    typeUrl: string;
    encode(message: MsgAcceptSeatNomination, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAcceptSeatNomination;
    fromPartial(object: DeepPartial<MsgAcceptSeatNomination>): MsgAcceptSeatNomination;
};
/**
 * @name MsgAcceptSeatNominationResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAcceptSeatNominationResponse
 */
export declare const MsgAcceptSeatNominationResponse: {
    typeUrl: string;
    encode(_: MsgAcceptSeatNominationResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgAcceptSeatNominationResponse;
    fromPartial(_: DeepPartial<MsgAcceptSeatNominationResponse>): MsgAcceptSeatNominationResponse;
};
/**
 * MsgVoteSeatElection casts a stake-weighted vote on a seat election.
 * @name MsgVoteSeatElection
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgVoteSeatElection
 */
export declare const MsgVoteSeatElection: {
    typeUrl: string;
    encode(message: MsgVoteSeatElection, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteSeatElection;
    fromPartial(object: DeepPartial<MsgVoteSeatElection>): MsgVoteSeatElection;
};
/**
 * @name MsgVoteSeatElectionResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgVoteSeatElectionResponse
 */
export declare const MsgVoteSeatElectionResponse: {
    typeUrl: string;
    encode(message: MsgVoteSeatElectionResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteSeatElectionResponse;
    fromPartial(object: DeepPartial<MsgVoteSeatElectionResponse>): MsgVoteSeatElectionResponse;
};
/**
 * MsgDomainFormationFreeze imposes a formation cooldown on a domain.
 * Only executable via governance (authority-gated).
 * @name MsgDomainFormationFreeze
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgDomainFormationFreeze
 */
export declare const MsgDomainFormationFreeze: {
    typeUrl: string;
    encode(message: MsgDomainFormationFreeze, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgDomainFormationFreeze;
    fromPartial(object: DeepPartial<MsgDomainFormationFreeze>): MsgDomainFormationFreeze;
};
/**
 * @name MsgDomainFormationFreezeResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgDomainFormationFreezeResponse
 */
export declare const MsgDomainFormationFreezeResponse: {
    typeUrl: string;
    encode(_: MsgDomainFormationFreezeResponse, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): MsgDomainFormationFreezeResponse;
    fromPartial(_: DeepPartial<MsgDomainFormationFreezeResponse>): MsgDomainFormationFreezeResponse;
};

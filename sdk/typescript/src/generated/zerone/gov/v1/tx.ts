//@ts-nocheck
import { ParamChange, ResearchFundVoters } from "./types";
import { Params } from "./genesis";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
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
export interface MsgStakeLIPResponse {}
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
export interface MsgWithdrawLIPResponse {}
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
export interface MsgUpdateParamsResponse {}
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
export interface MsgVoteResearchSpendResponse {}
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
export interface MsgSetResearchVotersResponse {}
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
export interface MsgAttachUpgradePlanResponse {}
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
export interface MsgAttachCreedAmendmentPinResponse {}
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
export interface MsgAcceptSeatNominationResponse {}
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
export interface MsgDomainFormationFreezeResponse {}
function createBaseMsgSubmitLIP(): MsgSubmitLIP {
  return {
    proposer: "",
    title: "",
    description: "",
    category: "",
    initialStake: "",
    paramChanges: []
  };
}
/**
 * MsgSubmitLIP creates a new LIP proposal.
 * @name MsgSubmitLIP
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgSubmitLIP
 */
export const MsgSubmitLIP = {
  typeUrl: "/zerone.gov.v1.MsgSubmitLIP",
  encode(message: MsgSubmitLIP, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    if (message.category !== "") {
      writer.uint32(34).string(message.category);
    }
    if (message.initialStake !== "") {
      writer.uint32(42).string(message.initialStake);
    }
    for (const v of message.paramChanges) {
      ParamChange.encode(v!, writer.uint32(50).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitLIP {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitLIP();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.title = reader.string();
          break;
        case 3:
          message.description = reader.string();
          break;
        case 4:
          message.category = reader.string();
          break;
        case 5:
          message.initialStake = reader.string();
          break;
        case 6:
          message.paramChanges.push(ParamChange.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgSubmitLIP>): MsgSubmitLIP {
    const message = createBaseMsgSubmitLIP();
    message.proposer = object.proposer ?? "";
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.category = object.category ?? "";
    message.initialStake = object.initialStake ?? "";
    message.paramChanges = object.paramChanges?.map(e => ParamChange.fromPartial(e)) || [];
    return message;
  }
};
function createBaseMsgSubmitLIPResponse(): MsgSubmitLIPResponse {
  return {
    lipId: ""
  };
}
/**
 * @name MsgSubmitLIPResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgSubmitLIPResponse
 */
export const MsgSubmitLIPResponse = {
  typeUrl: "/zerone.gov.v1.MsgSubmitLIPResponse",
  encode(message: MsgSubmitLIPResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.lipId !== "") {
      writer.uint32(10).string(message.lipId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitLIPResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitLIPResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.lipId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgSubmitLIPResponse>): MsgSubmitLIPResponse {
    const message = createBaseMsgSubmitLIPResponse();
    message.lipId = object.lipId ?? "";
    return message;
  }
};
function createBaseMsgStakeLIP(): MsgStakeLIP {
  return {
    staker: "",
    lipId: "",
    amount: ""
  };
}
/**
 * MsgStakeLIP adds stake to an existing LIP.
 * @name MsgStakeLIP
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgStakeLIP
 */
export const MsgStakeLIP = {
  typeUrl: "/zerone.gov.v1.MsgStakeLIP",
  encode(message: MsgStakeLIP, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.staker !== "") {
      writer.uint32(10).string(message.staker);
    }
    if (message.lipId !== "") {
      writer.uint32(18).string(message.lipId);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgStakeLIP {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgStakeLIP();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.staker = reader.string();
          break;
        case 2:
          message.lipId = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgStakeLIP>): MsgStakeLIP {
    const message = createBaseMsgStakeLIP();
    message.staker = object.staker ?? "";
    message.lipId = object.lipId ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgStakeLIPResponse(): MsgStakeLIPResponse {
  return {};
}
/**
 * @name MsgStakeLIPResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgStakeLIPResponse
 */
export const MsgStakeLIPResponse = {
  typeUrl: "/zerone.gov.v1.MsgStakeLIPResponse",
  encode(_: MsgStakeLIPResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgStakeLIPResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgStakeLIPResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgStakeLIPResponse>): MsgStakeLIPResponse {
    const message = createBaseMsgStakeLIPResponse();
    return message;
  }
};
function createBaseMsgAdvanceLIPStage(): MsgAdvanceLIPStage {
  return {
    authority: "",
    lipId: ""
  };
}
/**
 * MsgAdvanceLIPStage advances a LIP to its next stage.
 * @name MsgAdvanceLIPStage
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAdvanceLIPStage
 */
export const MsgAdvanceLIPStage = {
  typeUrl: "/zerone.gov.v1.MsgAdvanceLIPStage",
  encode(message: MsgAdvanceLIPStage, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.lipId !== "") {
      writer.uint32(18).string(message.lipId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAdvanceLIPStage {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAdvanceLIPStage();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.lipId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAdvanceLIPStage>): MsgAdvanceLIPStage {
    const message = createBaseMsgAdvanceLIPStage();
    message.authority = object.authority ?? "";
    message.lipId = object.lipId ?? "";
    return message;
  }
};
function createBaseMsgAdvanceLIPStageResponse(): MsgAdvanceLIPStageResponse {
  return {
    newStage: ""
  };
}
/**
 * @name MsgAdvanceLIPStageResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAdvanceLIPStageResponse
 */
export const MsgAdvanceLIPStageResponse = {
  typeUrl: "/zerone.gov.v1.MsgAdvanceLIPStageResponse",
  encode(message: MsgAdvanceLIPStageResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.newStage !== "") {
      writer.uint32(10).string(message.newStage);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAdvanceLIPStageResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAdvanceLIPStageResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.newStage = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAdvanceLIPStageResponse>): MsgAdvanceLIPStageResponse {
    const message = createBaseMsgAdvanceLIPStageResponse();
    message.newStage = object.newStage ?? "";
    return message;
  }
};
function createBaseMsgCastVote(): MsgCastVote {
  return {
    voter: "",
    lipId: "",
    option: ""
  };
}
/**
 * MsgCastVote casts a stake-weighted vote on a LIP.
 * @name MsgCastVote
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgCastVote
 */
export const MsgCastVote = {
  typeUrl: "/zerone.gov.v1.MsgCastVote",
  encode(message: MsgCastVote, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.voter !== "") {
      writer.uint32(10).string(message.voter);
    }
    if (message.lipId !== "") {
      writer.uint32(18).string(message.lipId);
    }
    if (message.option !== "") {
      writer.uint32(26).string(message.option);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCastVote {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCastVote();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.voter = reader.string();
          break;
        case 2:
          message.lipId = reader.string();
          break;
        case 3:
          message.option = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCastVote>): MsgCastVote {
    const message = createBaseMsgCastVote();
    message.voter = object.voter ?? "";
    message.lipId = object.lipId ?? "";
    message.option = object.option ?? "";
    return message;
  }
};
function createBaseMsgCastVoteResponse(): MsgCastVoteResponse {
  return {
    effectiveWeight: ""
  };
}
/**
 * @name MsgCastVoteResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgCastVoteResponse
 */
export const MsgCastVoteResponse = {
  typeUrl: "/zerone.gov.v1.MsgCastVoteResponse",
  encode(message: MsgCastVoteResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.effectiveWeight !== "") {
      writer.uint32(10).string(message.effectiveWeight);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCastVoteResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCastVoteResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.effectiveWeight = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCastVoteResponse>): MsgCastVoteResponse {
    const message = createBaseMsgCastVoteResponse();
    message.effectiveWeight = object.effectiveWeight ?? "";
    return message;
  }
};
function createBaseMsgWithdrawLIP(): MsgWithdrawLIP {
  return {
    proposer: "",
    lipId: ""
  };
}
/**
 * MsgWithdrawLIP withdraws a LIP (proposer only, before voting).
 * @name MsgWithdrawLIP
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgWithdrawLIP
 */
export const MsgWithdrawLIP = {
  typeUrl: "/zerone.gov.v1.MsgWithdrawLIP",
  encode(message: MsgWithdrawLIP, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.lipId !== "") {
      writer.uint32(18).string(message.lipId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgWithdrawLIP {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgWithdrawLIP();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.lipId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgWithdrawLIP>): MsgWithdrawLIP {
    const message = createBaseMsgWithdrawLIP();
    message.proposer = object.proposer ?? "";
    message.lipId = object.lipId ?? "";
    return message;
  }
};
function createBaseMsgWithdrawLIPResponse(): MsgWithdrawLIPResponse {
  return {};
}
/**
 * @name MsgWithdrawLIPResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgWithdrawLIPResponse
 */
export const MsgWithdrawLIPResponse = {
  typeUrl: "/zerone.gov.v1.MsgWithdrawLIPResponse",
  encode(_: MsgWithdrawLIPResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgWithdrawLIPResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgWithdrawLIPResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgWithdrawLIPResponse>): MsgWithdrawLIPResponse {
    const message = createBaseMsgWithdrawLIPResponse();
    return message;
  }
};
function createBaseMsgUpdateParams(): MsgUpdateParams {
  return {
    authority: "",
    params: undefined
  };
}
/**
 * MsgUpdateParams updates governance parameters (authority only).
 * @name MsgUpdateParams
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgUpdateParams
 */
export const MsgUpdateParams = {
  typeUrl: "/zerone.gov.v1.MsgUpdateParams",
  encode(message: MsgUpdateParams, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParams {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParams();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.params = Params.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgUpdateParams>): MsgUpdateParams {
    const message = createBaseMsgUpdateParams();
    message.authority = object.authority ?? "";
    message.params = object.params !== undefined && object.params !== null ? Params.fromPartial(object.params) : undefined;
    return message;
  }
};
function createBaseMsgUpdateParamsResponse(): MsgUpdateParamsResponse {
  return {};
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgUpdateParamsResponse
 */
export const MsgUpdateParamsResponse = {
  typeUrl: "/zerone.gov.v1.MsgUpdateParamsResponse",
  encode(_: MsgUpdateParamsResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateParamsResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateParamsResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgUpdateParamsResponse>): MsgUpdateParamsResponse {
    const message = createBaseMsgUpdateParamsResponse();
    return message;
  }
};
function createBaseMsgSubmitResearchSpend(): MsgSubmitResearchSpend {
  return {
    proposer: "",
    title: "",
    description: "",
    recipient: "",
    amount: "",
    justification: ""
  };
}
/**
 * MsgSubmitResearchSpend creates a new research fund spend proposal.
 * @name MsgSubmitResearchSpend
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgSubmitResearchSpend
 */
export const MsgSubmitResearchSpend = {
  typeUrl: "/zerone.gov.v1.MsgSubmitResearchSpend",
  encode(message: MsgSubmitResearchSpend, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.title !== "") {
      writer.uint32(18).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(26).string(message.description);
    }
    if (message.recipient !== "") {
      writer.uint32(34).string(message.recipient);
    }
    if (message.amount !== "") {
      writer.uint32(42).string(message.amount);
    }
    if (message.justification !== "") {
      writer.uint32(50).string(message.justification);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitResearchSpend {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitResearchSpend();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.title = reader.string();
          break;
        case 3:
          message.description = reader.string();
          break;
        case 4:
          message.recipient = reader.string();
          break;
        case 5:
          message.amount = reader.string();
          break;
        case 6:
          message.justification = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgSubmitResearchSpend>): MsgSubmitResearchSpend {
    const message = createBaseMsgSubmitResearchSpend();
    message.proposer = object.proposer ?? "";
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.recipient = object.recipient ?? "";
    message.amount = object.amount ?? "";
    message.justification = object.justification ?? "";
    return message;
  }
};
function createBaseMsgSubmitResearchSpendResponse(): MsgSubmitResearchSpendResponse {
  return {
    proposalId: BigInt(0)
  };
}
/**
 * @name MsgSubmitResearchSpendResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgSubmitResearchSpendResponse
 */
export const MsgSubmitResearchSpendResponse = {
  typeUrl: "/zerone.gov.v1.MsgSubmitResearchSpendResponse",
  encode(message: MsgSubmitResearchSpendResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposalId !== BigInt(0)) {
      writer.uint32(8).uint64(message.proposalId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitResearchSpendResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitResearchSpendResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposalId = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgSubmitResearchSpendResponse>): MsgSubmitResearchSpendResponse {
    const message = createBaseMsgSubmitResearchSpendResponse();
    message.proposalId = object.proposalId !== undefined && object.proposalId !== null ? BigInt(object.proposalId.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgVoteResearchSpend(): MsgVoteResearchSpend {
  return {
    voter: "",
    proposalId: BigInt(0),
    vote: "",
    reasoning: ""
  };
}
/**
 * MsgVoteResearchSpend casts a vote on a research spend proposal.
 * @name MsgVoteResearchSpend
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgVoteResearchSpend
 */
export const MsgVoteResearchSpend = {
  typeUrl: "/zerone.gov.v1.MsgVoteResearchSpend",
  encode(message: MsgVoteResearchSpend, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.voter !== "") {
      writer.uint32(10).string(message.voter);
    }
    if (message.proposalId !== BigInt(0)) {
      writer.uint32(16).uint64(message.proposalId);
    }
    if (message.vote !== "") {
      writer.uint32(26).string(message.vote);
    }
    if (message.reasoning !== "") {
      writer.uint32(34).string(message.reasoning);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteResearchSpend {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteResearchSpend();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.voter = reader.string();
          break;
        case 2:
          message.proposalId = reader.uint64();
          break;
        case 3:
          message.vote = reader.string();
          break;
        case 4:
          message.reasoning = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgVoteResearchSpend>): MsgVoteResearchSpend {
    const message = createBaseMsgVoteResearchSpend();
    message.voter = object.voter ?? "";
    message.proposalId = object.proposalId !== undefined && object.proposalId !== null ? BigInt(object.proposalId.toString()) : BigInt(0);
    message.vote = object.vote ?? "";
    message.reasoning = object.reasoning ?? "";
    return message;
  }
};
function createBaseMsgVoteResearchSpendResponse(): MsgVoteResearchSpendResponse {
  return {};
}
/**
 * @name MsgVoteResearchSpendResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgVoteResearchSpendResponse
 */
export const MsgVoteResearchSpendResponse = {
  typeUrl: "/zerone.gov.v1.MsgVoteResearchSpendResponse",
  encode(_: MsgVoteResearchSpendResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteResearchSpendResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteResearchSpendResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgVoteResearchSpendResponse>): MsgVoteResearchSpendResponse {
    const message = createBaseMsgVoteResearchSpendResponse();
    return message;
  }
};
function createBaseMsgSetResearchVoters(): MsgSetResearchVoters {
  return {
    authority: "",
    voters: undefined
  };
}
/**
 * MsgSetResearchVoters configures the 2-of-2 research fund voters (authority only).
 * @name MsgSetResearchVoters
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgSetResearchVoters
 */
export const MsgSetResearchVoters = {
  typeUrl: "/zerone.gov.v1.MsgSetResearchVoters",
  encode(message: MsgSetResearchVoters, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.voters !== undefined) {
      ResearchFundVoters.encode(message.voters, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSetResearchVoters {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSetResearchVoters();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.voters = ResearchFundVoters.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgSetResearchVoters>): MsgSetResearchVoters {
    const message = createBaseMsgSetResearchVoters();
    message.authority = object.authority ?? "";
    message.voters = object.voters !== undefined && object.voters !== null ? ResearchFundVoters.fromPartial(object.voters) : undefined;
    return message;
  }
};
function createBaseMsgSetResearchVotersResponse(): MsgSetResearchVotersResponse {
  return {};
}
/**
 * @name MsgSetResearchVotersResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgSetResearchVotersResponse
 */
export const MsgSetResearchVotersResponse = {
  typeUrl: "/zerone.gov.v1.MsgSetResearchVotersResponse",
  encode(_: MsgSetResearchVotersResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSetResearchVotersResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSetResearchVotersResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgSetResearchVotersResponse>): MsgSetResearchVotersResponse {
    const message = createBaseMsgSetResearchVotersResponse();
    return message;
  }
};
function createBaseMsgAttachUpgradePlan(): MsgAttachUpgradePlan {
  return {
    proposer: "",
    lipId: "",
    upgradeName: "",
    height: BigInt(0),
    info: ""
  };
}
/**
 * MsgAttachUpgradePlan attaches a software upgrade plan to an upgrade-category LIP.
 * @name MsgAttachUpgradePlan
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAttachUpgradePlan
 */
export const MsgAttachUpgradePlan = {
  typeUrl: "/zerone.gov.v1.MsgAttachUpgradePlan",
  encode(message: MsgAttachUpgradePlan, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.lipId !== "") {
      writer.uint32(18).string(message.lipId);
    }
    if (message.upgradeName !== "") {
      writer.uint32(26).string(message.upgradeName);
    }
    if (message.height !== BigInt(0)) {
      writer.uint32(32).int64(message.height);
    }
    if (message.info !== "") {
      writer.uint32(42).string(message.info);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAttachUpgradePlan {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAttachUpgradePlan();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.lipId = reader.string();
          break;
        case 3:
          message.upgradeName = reader.string();
          break;
        case 4:
          message.height = reader.int64();
          break;
        case 5:
          message.info = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAttachUpgradePlan>): MsgAttachUpgradePlan {
    const message = createBaseMsgAttachUpgradePlan();
    message.proposer = object.proposer ?? "";
    message.lipId = object.lipId ?? "";
    message.upgradeName = object.upgradeName ?? "";
    message.height = object.height !== undefined && object.height !== null ? BigInt(object.height.toString()) : BigInt(0);
    message.info = object.info ?? "";
    return message;
  }
};
function createBaseMsgAttachUpgradePlanResponse(): MsgAttachUpgradePlanResponse {
  return {};
}
/**
 * @name MsgAttachUpgradePlanResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAttachUpgradePlanResponse
 */
export const MsgAttachUpgradePlanResponse = {
  typeUrl: "/zerone.gov.v1.MsgAttachUpgradePlanResponse",
  encode(_: MsgAttachUpgradePlanResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAttachUpgradePlanResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAttachUpgradePlanResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgAttachUpgradePlanResponse>): MsgAttachUpgradePlanResponse {
    const message = createBaseMsgAttachUpgradePlanResponse();
    return message;
  }
};
function createBaseMsgAttachCreedAmendmentPin(): MsgAttachCreedAmendmentPin {
  return {
    proposer: "",
    lipId: "",
    canonicalHash: new Uint8Array(),
    commitmentsJson: new Uint8Array()
  };
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
export const MsgAttachCreedAmendmentPin = {
  typeUrl: "/zerone.gov.v1.MsgAttachCreedAmendmentPin",
  encode(message: MsgAttachCreedAmendmentPin, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.lipId !== "") {
      writer.uint32(18).string(message.lipId);
    }
    if (message.canonicalHash.length !== 0) {
      writer.uint32(26).bytes(message.canonicalHash);
    }
    if (message.commitmentsJson.length !== 0) {
      writer.uint32(34).bytes(message.commitmentsJson);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAttachCreedAmendmentPin {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAttachCreedAmendmentPin();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.lipId = reader.string();
          break;
        case 3:
          message.canonicalHash = reader.bytes();
          break;
        case 4:
          message.commitmentsJson = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAttachCreedAmendmentPin>): MsgAttachCreedAmendmentPin {
    const message = createBaseMsgAttachCreedAmendmentPin();
    message.proposer = object.proposer ?? "";
    message.lipId = object.lipId ?? "";
    message.canonicalHash = object.canonicalHash ?? new Uint8Array();
    message.commitmentsJson = object.commitmentsJson ?? new Uint8Array();
    return message;
  }
};
function createBaseMsgAttachCreedAmendmentPinResponse(): MsgAttachCreedAmendmentPinResponse {
  return {};
}
/**
 * @name MsgAttachCreedAmendmentPinResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAttachCreedAmendmentPinResponse
 */
export const MsgAttachCreedAmendmentPinResponse = {
  typeUrl: "/zerone.gov.v1.MsgAttachCreedAmendmentPinResponse",
  encode(_: MsgAttachCreedAmendmentPinResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAttachCreedAmendmentPinResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAttachCreedAmendmentPinResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgAttachCreedAmendmentPinResponse>): MsgAttachCreedAmendmentPinResponse {
    const message = createBaseMsgAttachCreedAmendmentPinResponse();
    return message;
  }
};
function createBaseMsgNominateSeatElection(): MsgNominateSeatElection {
  return {
    proposer: "",
    candidate: "",
    seatIndex: 0,
    statement: ""
  };
}
/**
 * MsgNominateSeatElection nominates a candidate for a community seat.
 * @name MsgNominateSeatElection
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgNominateSeatElection
 */
export const MsgNominateSeatElection = {
  typeUrl: "/zerone.gov.v1.MsgNominateSeatElection",
  encode(message: MsgNominateSeatElection, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.candidate !== "") {
      writer.uint32(18).string(message.candidate);
    }
    if (message.seatIndex !== 0) {
      writer.uint32(24).uint32(message.seatIndex);
    }
    if (message.statement !== "") {
      writer.uint32(34).string(message.statement);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgNominateSeatElection {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgNominateSeatElection();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.candidate = reader.string();
          break;
        case 3:
          message.seatIndex = reader.uint32();
          break;
        case 4:
          message.statement = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgNominateSeatElection>): MsgNominateSeatElection {
    const message = createBaseMsgNominateSeatElection();
    message.proposer = object.proposer ?? "";
    message.candidate = object.candidate ?? "";
    message.seatIndex = object.seatIndex ?? 0;
    message.statement = object.statement ?? "";
    return message;
  }
};
function createBaseMsgNominateSeatElectionResponse(): MsgNominateSeatElectionResponse {
  return {
    proposalId: BigInt(0)
  };
}
/**
 * @name MsgNominateSeatElectionResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgNominateSeatElectionResponse
 */
export const MsgNominateSeatElectionResponse = {
  typeUrl: "/zerone.gov.v1.MsgNominateSeatElectionResponse",
  encode(message: MsgNominateSeatElectionResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposalId !== BigInt(0)) {
      writer.uint32(8).uint64(message.proposalId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgNominateSeatElectionResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgNominateSeatElectionResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposalId = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgNominateSeatElectionResponse>): MsgNominateSeatElectionResponse {
    const message = createBaseMsgNominateSeatElectionResponse();
    message.proposalId = object.proposalId !== undefined && object.proposalId !== null ? BigInt(object.proposalId.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgAcceptSeatNomination(): MsgAcceptSeatNomination {
  return {
    candidate: "",
    proposalId: BigInt(0)
  };
}
/**
 * MsgAcceptSeatNomination accepts a pending seat election nomination.
 * @name MsgAcceptSeatNomination
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAcceptSeatNomination
 */
export const MsgAcceptSeatNomination = {
  typeUrl: "/zerone.gov.v1.MsgAcceptSeatNomination",
  encode(message: MsgAcceptSeatNomination, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.candidate !== "") {
      writer.uint32(10).string(message.candidate);
    }
    if (message.proposalId !== BigInt(0)) {
      writer.uint32(16).uint64(message.proposalId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAcceptSeatNomination {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAcceptSeatNomination();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.candidate = reader.string();
          break;
        case 2:
          message.proposalId = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAcceptSeatNomination>): MsgAcceptSeatNomination {
    const message = createBaseMsgAcceptSeatNomination();
    message.candidate = object.candidate ?? "";
    message.proposalId = object.proposalId !== undefined && object.proposalId !== null ? BigInt(object.proposalId.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgAcceptSeatNominationResponse(): MsgAcceptSeatNominationResponse {
  return {};
}
/**
 * @name MsgAcceptSeatNominationResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgAcceptSeatNominationResponse
 */
export const MsgAcceptSeatNominationResponse = {
  typeUrl: "/zerone.gov.v1.MsgAcceptSeatNominationResponse",
  encode(_: MsgAcceptSeatNominationResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAcceptSeatNominationResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAcceptSeatNominationResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgAcceptSeatNominationResponse>): MsgAcceptSeatNominationResponse {
    const message = createBaseMsgAcceptSeatNominationResponse();
    return message;
  }
};
function createBaseMsgVoteSeatElection(): MsgVoteSeatElection {
  return {
    voter: "",
    proposalId: BigInt(0),
    option: ""
  };
}
/**
 * MsgVoteSeatElection casts a stake-weighted vote on a seat election.
 * @name MsgVoteSeatElection
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgVoteSeatElection
 */
export const MsgVoteSeatElection = {
  typeUrl: "/zerone.gov.v1.MsgVoteSeatElection",
  encode(message: MsgVoteSeatElection, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.voter !== "") {
      writer.uint32(10).string(message.voter);
    }
    if (message.proposalId !== BigInt(0)) {
      writer.uint32(16).uint64(message.proposalId);
    }
    if (message.option !== "") {
      writer.uint32(26).string(message.option);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteSeatElection {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteSeatElection();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.voter = reader.string();
          break;
        case 2:
          message.proposalId = reader.uint64();
          break;
        case 3:
          message.option = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgVoteSeatElection>): MsgVoteSeatElection {
    const message = createBaseMsgVoteSeatElection();
    message.voter = object.voter ?? "";
    message.proposalId = object.proposalId !== undefined && object.proposalId !== null ? BigInt(object.proposalId.toString()) : BigInt(0);
    message.option = object.option ?? "";
    return message;
  }
};
function createBaseMsgVoteSeatElectionResponse(): MsgVoteSeatElectionResponse {
  return {
    effectiveWeight: ""
  };
}
/**
 * @name MsgVoteSeatElectionResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgVoteSeatElectionResponse
 */
export const MsgVoteSeatElectionResponse = {
  typeUrl: "/zerone.gov.v1.MsgVoteSeatElectionResponse",
  encode(message: MsgVoteSeatElectionResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.effectiveWeight !== "") {
      writer.uint32(10).string(message.effectiveWeight);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteSeatElectionResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteSeatElectionResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.effectiveWeight = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgVoteSeatElectionResponse>): MsgVoteSeatElectionResponse {
    const message = createBaseMsgVoteSeatElectionResponse();
    message.effectiveWeight = object.effectiveWeight ?? "";
    return message;
  }
};
function createBaseMsgDomainFormationFreeze(): MsgDomainFormationFreeze {
  return {
    authority: "",
    domain: "",
    durationBlocks: BigInt(0),
    reason: ""
  };
}
/**
 * MsgDomainFormationFreeze imposes a formation cooldown on a domain.
 * Only executable via governance (authority-gated).
 * @name MsgDomainFormationFreeze
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgDomainFormationFreeze
 */
export const MsgDomainFormationFreeze = {
  typeUrl: "/zerone.gov.v1.MsgDomainFormationFreeze",
  encode(message: MsgDomainFormationFreeze, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    if (message.durationBlocks !== BigInt(0)) {
      writer.uint32(24).uint64(message.durationBlocks);
    }
    if (message.reason !== "") {
      writer.uint32(34).string(message.reason);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgDomainFormationFreeze {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgDomainFormationFreeze();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.domain = reader.string();
          break;
        case 3:
          message.durationBlocks = reader.uint64();
          break;
        case 4:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgDomainFormationFreeze>): MsgDomainFormationFreeze {
    const message = createBaseMsgDomainFormationFreeze();
    message.authority = object.authority ?? "";
    message.domain = object.domain ?? "";
    message.durationBlocks = object.durationBlocks !== undefined && object.durationBlocks !== null ? BigInt(object.durationBlocks.toString()) : BigInt(0);
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgDomainFormationFreezeResponse(): MsgDomainFormationFreezeResponse {
  return {};
}
/**
 * @name MsgDomainFormationFreezeResponse
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.MsgDomainFormationFreezeResponse
 */
export const MsgDomainFormationFreezeResponse = {
  typeUrl: "/zerone.gov.v1.MsgDomainFormationFreezeResponse",
  encode(_: MsgDomainFormationFreezeResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgDomainFormationFreezeResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgDomainFormationFreezeResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(_: DeepPartial<MsgDomainFormationFreezeResponse>): MsgDomainFormationFreezeResponse {
    const message = createBaseMsgDomainFormationFreezeResponse();
    return message;
  }
};
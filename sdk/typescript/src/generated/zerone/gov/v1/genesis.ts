//@ts-nocheck
import { ResearchFundVoters, UpgradePlan, LIP, Vote, ResearchFundGovernanceState, SeatElectionProposal, SeatElectionVote } from "./types";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
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
function createBaseParams(): Params {
  return {
    votingPeriodBlocks: BigInt(0),
    discussionPeriodBlocks: BigInt(0),
    quorumThresholdBps: BigInt(0),
    supportThresholdBps: BigInt(0),
    minLipStake: "",
    minVoteStake: "",
    categoryConfigs: [],
    researchFundVoters: undefined,
    researchDiscussionBlocks: BigInt(0),
    researchVotingBlocks: BigInt(0)
  };
}
/**
 * Params defines the governance module parameters.
 * @name Params
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.Params
 */
export const Params = {
  typeUrl: "/zerone.gov.v1.Params",
  encode(message: Params, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.votingPeriodBlocks !== BigInt(0)) {
      writer.uint32(8).uint64(message.votingPeriodBlocks);
    }
    if (message.discussionPeriodBlocks !== BigInt(0)) {
      writer.uint32(16).uint64(message.discussionPeriodBlocks);
    }
    if (message.quorumThresholdBps !== BigInt(0)) {
      writer.uint32(24).uint64(message.quorumThresholdBps);
    }
    if (message.supportThresholdBps !== BigInt(0)) {
      writer.uint32(32).uint64(message.supportThresholdBps);
    }
    if (message.minLipStake !== "") {
      writer.uint32(42).string(message.minLipStake);
    }
    if (message.minVoteStake !== "") {
      writer.uint32(50).string(message.minVoteStake);
    }
    for (const v of message.categoryConfigs) {
      CategoryConfig.encode(v!, writer.uint32(58).fork()).ldelim();
    }
    if (message.researchFundVoters !== undefined) {
      ResearchFundVoters.encode(message.researchFundVoters, writer.uint32(66).fork()).ldelim();
    }
    if (message.researchDiscussionBlocks !== BigInt(0)) {
      writer.uint32(72).uint64(message.researchDiscussionBlocks);
    }
    if (message.researchVotingBlocks !== BigInt(0)) {
      writer.uint32(80).uint64(message.researchVotingBlocks);
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
          message.votingPeriodBlocks = reader.uint64();
          break;
        case 2:
          message.discussionPeriodBlocks = reader.uint64();
          break;
        case 3:
          message.quorumThresholdBps = reader.uint64();
          break;
        case 4:
          message.supportThresholdBps = reader.uint64();
          break;
        case 5:
          message.minLipStake = reader.string();
          break;
        case 6:
          message.minVoteStake = reader.string();
          break;
        case 7:
          message.categoryConfigs.push(CategoryConfig.decode(reader, reader.uint32()));
          break;
        case 8:
          message.researchFundVoters = ResearchFundVoters.decode(reader, reader.uint32());
          break;
        case 9:
          message.researchDiscussionBlocks = reader.uint64();
          break;
        case 10:
          message.researchVotingBlocks = reader.uint64();
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
    message.votingPeriodBlocks = object.votingPeriodBlocks !== undefined && object.votingPeriodBlocks !== null ? BigInt(object.votingPeriodBlocks.toString()) : BigInt(0);
    message.discussionPeriodBlocks = object.discussionPeriodBlocks !== undefined && object.discussionPeriodBlocks !== null ? BigInt(object.discussionPeriodBlocks.toString()) : BigInt(0);
    message.quorumThresholdBps = object.quorumThresholdBps !== undefined && object.quorumThresholdBps !== null ? BigInt(object.quorumThresholdBps.toString()) : BigInt(0);
    message.supportThresholdBps = object.supportThresholdBps !== undefined && object.supportThresholdBps !== null ? BigInt(object.supportThresholdBps.toString()) : BigInt(0);
    message.minLipStake = object.minLipStake ?? "";
    message.minVoteStake = object.minVoteStake ?? "";
    message.categoryConfigs = object.categoryConfigs?.map(e => CategoryConfig.fromPartial(e)) || [];
    message.researchFundVoters = object.researchFundVoters !== undefined && object.researchFundVoters !== null ? ResearchFundVoters.fromPartial(object.researchFundVoters) : undefined;
    message.researchDiscussionBlocks = object.researchDiscussionBlocks !== undefined && object.researchDiscussionBlocks !== null ? BigInt(object.researchDiscussionBlocks.toString()) : BigInt(0);
    message.researchVotingBlocks = object.researchVotingBlocks !== undefined && object.researchVotingBlocks !== null ? BigInt(object.researchVotingBlocks.toString()) : BigInt(0);
    return message;
  }
};
function createBaseCategoryConfig(): CategoryConfig {
  return {
    category: "",
    requiredStakeUzrn: "",
    reviewBlocks: BigInt(0)
  };
}
/**
 * CategoryConfig defines per-category stake and review requirements.
 * @name CategoryConfig
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.CategoryConfig
 */
export const CategoryConfig = {
  typeUrl: "/zerone.gov.v1.CategoryConfig",
  encode(message: CategoryConfig, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.category !== "") {
      writer.uint32(10).string(message.category);
    }
    if (message.requiredStakeUzrn !== "") {
      writer.uint32(18).string(message.requiredStakeUzrn);
    }
    if (message.reviewBlocks !== BigInt(0)) {
      writer.uint32(24).uint64(message.reviewBlocks);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): CategoryConfig {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCategoryConfig();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.category = reader.string();
          break;
        case 2:
          message.requiredStakeUzrn = reader.string();
          break;
        case 3:
          message.reviewBlocks = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<CategoryConfig>): CategoryConfig {
    const message = createBaseCategoryConfig();
    message.category = object.category ?? "";
    message.requiredStakeUzrn = object.requiredStakeUzrn ?? "";
    message.reviewBlocks = object.reviewBlocks !== undefined && object.reviewBlocks !== null ? BigInt(object.reviewBlocks.toString()) : BigInt(0);
    return message;
  }
};
function createBaseGenesisUpgradePlan(): GenesisUpgradePlan {
  return {
    lipId: "",
    plan: undefined
  };
}
/**
 * GenesisUpgradePlan pairs a LIP ID with its attached upgrade plan for genesis export/import.
 * @name GenesisUpgradePlan
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.GenesisUpgradePlan
 */
export const GenesisUpgradePlan = {
  typeUrl: "/zerone.gov.v1.GenesisUpgradePlan",
  encode(message: GenesisUpgradePlan, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.lipId !== "") {
      writer.uint32(10).string(message.lipId);
    }
    if (message.plan !== undefined) {
      UpgradePlan.encode(message.plan, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): GenesisUpgradePlan {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGenesisUpgradePlan();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.lipId = reader.string();
          break;
        case 2:
          message.plan = UpgradePlan.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<GenesisUpgradePlan>): GenesisUpgradePlan {
    const message = createBaseGenesisUpgradePlan();
    message.lipId = object.lipId ?? "";
    message.plan = object.plan !== undefined && object.plan !== null ? UpgradePlan.fromPartial(object.plan) : undefined;
    return message;
  }
};
function createBaseGenesisCreedAmendmentPin(): GenesisCreedAmendmentPin {
  return {
    lipId: "",
    canonicalHash: new Uint8Array(),
    commitmentsJson: new Uint8Array()
  };
}
/**
 * GenesisCreedAmendmentPin pairs a LIP ID with its attached
 * creed-amendment payload for genesis export/import.
 * @name GenesisCreedAmendmentPin
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.GenesisCreedAmendmentPin
 */
export const GenesisCreedAmendmentPin = {
  typeUrl: "/zerone.gov.v1.GenesisCreedAmendmentPin",
  encode(message: GenesisCreedAmendmentPin, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.lipId !== "") {
      writer.uint32(10).string(message.lipId);
    }
    if (message.canonicalHash.length !== 0) {
      writer.uint32(18).bytes(message.canonicalHash);
    }
    if (message.commitmentsJson.length !== 0) {
      writer.uint32(26).bytes(message.commitmentsJson);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): GenesisCreedAmendmentPin {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGenesisCreedAmendmentPin();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.lipId = reader.string();
          break;
        case 2:
          message.canonicalHash = reader.bytes();
          break;
        case 3:
          message.commitmentsJson = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<GenesisCreedAmendmentPin>): GenesisCreedAmendmentPin {
    const message = createBaseGenesisCreedAmendmentPin();
    message.lipId = object.lipId ?? "";
    message.canonicalHash = object.canonicalHash ?? new Uint8Array();
    message.commitmentsJson = object.commitmentsJson ?? new Uint8Array();
    return message;
  }
};
function createBaseEmergencyTransitionHold(): EmergencyTransitionHold {
  return {
    incidentId: "",
    activatedAtBlock: BigInt(0),
    latestIncidentId: "",
    incidentCount: BigInt(0),
    incidentLineageSha256: new Uint8Array()
  };
}
/**
 * EmergencyTransitionHold is the durable post-incident review gate for every
 * automatic custom-governance transition. It is created when transaction
 * quarantine is observed and is not cleared by an ordinary resume ceremony.
 * @name EmergencyTransitionHold
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.EmergencyTransitionHold
 */
export const EmergencyTransitionHold = {
  typeUrl: "/zerone.gov.v1.EmergencyTransitionHold",
  encode(message: EmergencyTransitionHold, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.incidentId !== "") {
      writer.uint32(10).string(message.incidentId);
    }
    if (message.activatedAtBlock !== BigInt(0)) {
      writer.uint32(16).uint64(message.activatedAtBlock);
    }
    if (message.latestIncidentId !== "") {
      writer.uint32(26).string(message.latestIncidentId);
    }
    if (message.incidentCount !== BigInt(0)) {
      writer.uint32(32).uint64(message.incidentCount);
    }
    if (message.incidentLineageSha256.length !== 0) {
      writer.uint32(42).bytes(message.incidentLineageSha256);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): EmergencyTransitionHold {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEmergencyTransitionHold();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.incidentId = reader.string();
          break;
        case 2:
          message.activatedAtBlock = reader.uint64();
          break;
        case 3:
          message.latestIncidentId = reader.string();
          break;
        case 4:
          message.incidentCount = reader.uint64();
          break;
        case 5:
          message.incidentLineageSha256 = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<EmergencyTransitionHold>): EmergencyTransitionHold {
    const message = createBaseEmergencyTransitionHold();
    message.incidentId = object.incidentId ?? "";
    message.activatedAtBlock = object.activatedAtBlock !== undefined && object.activatedAtBlock !== null ? BigInt(object.activatedAtBlock.toString()) : BigInt(0);
    message.latestIncidentId = object.latestIncidentId ?? "";
    message.incidentCount = object.incidentCount !== undefined && object.incidentCount !== null ? BigInt(object.incidentCount.toString()) : BigInt(0);
    message.incidentLineageSha256 = object.incidentLineageSha256 ?? new Uint8Array();
    return message;
  }
};
function createBaseGenesisState(): GenesisState {
  return {
    params: undefined,
    lips: [],
    votes: [],
    nextLipNumber: BigInt(0),
    upgradePlans: [],
    researchFundGovernance: undefined,
    seatElections: [],
    seatElectionVotes: [],
    nextSeatElectionNumber: BigInt(0),
    creedAmendmentPins: [],
    emergencyTransitionHold: undefined
  };
}
/**
 * GenesisState defines the governance module's genesis state.
 * @name GenesisState
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.GenesisState
 */
export const GenesisState = {
  typeUrl: "/zerone.gov.v1.GenesisState",
  encode(message: GenesisState, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.lips) {
      LIP.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.votes) {
      Vote.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    if (message.nextLipNumber !== BigInt(0)) {
      writer.uint32(32).uint64(message.nextLipNumber);
    }
    for (const v of message.upgradePlans) {
      GenesisUpgradePlan.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    if (message.researchFundGovernance !== undefined) {
      ResearchFundGovernanceState.encode(message.researchFundGovernance, writer.uint32(50).fork()).ldelim();
    }
    for (const v of message.seatElections) {
      SeatElectionProposal.encode(v!, writer.uint32(58).fork()).ldelim();
    }
    for (const v of message.seatElectionVotes) {
      SeatElectionVote.encode(v!, writer.uint32(66).fork()).ldelim();
    }
    if (message.nextSeatElectionNumber !== BigInt(0)) {
      writer.uint32(72).uint64(message.nextSeatElectionNumber);
    }
    for (const v of message.creedAmendmentPins) {
      GenesisCreedAmendmentPin.encode(v!, writer.uint32(82).fork()).ldelim();
    }
    if (message.emergencyTransitionHold !== undefined) {
      EmergencyTransitionHold.encode(message.emergencyTransitionHold, writer.uint32(90).fork()).ldelim();
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
          message.lips.push(LIP.decode(reader, reader.uint32()));
          break;
        case 3:
          message.votes.push(Vote.decode(reader, reader.uint32()));
          break;
        case 4:
          message.nextLipNumber = reader.uint64();
          break;
        case 5:
          message.upgradePlans.push(GenesisUpgradePlan.decode(reader, reader.uint32()));
          break;
        case 6:
          message.researchFundGovernance = ResearchFundGovernanceState.decode(reader, reader.uint32());
          break;
        case 7:
          message.seatElections.push(SeatElectionProposal.decode(reader, reader.uint32()));
          break;
        case 8:
          message.seatElectionVotes.push(SeatElectionVote.decode(reader, reader.uint32()));
          break;
        case 9:
          message.nextSeatElectionNumber = reader.uint64();
          break;
        case 10:
          message.creedAmendmentPins.push(GenesisCreedAmendmentPin.decode(reader, reader.uint32()));
          break;
        case 11:
          message.emergencyTransitionHold = EmergencyTransitionHold.decode(reader, reader.uint32());
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
    message.lips = object.lips?.map(e => LIP.fromPartial(e)) || [];
    message.votes = object.votes?.map(e => Vote.fromPartial(e)) || [];
    message.nextLipNumber = object.nextLipNumber !== undefined && object.nextLipNumber !== null ? BigInt(object.nextLipNumber.toString()) : BigInt(0);
    message.upgradePlans = object.upgradePlans?.map(e => GenesisUpgradePlan.fromPartial(e)) || [];
    message.researchFundGovernance = object.researchFundGovernance !== undefined && object.researchFundGovernance !== null ? ResearchFundGovernanceState.fromPartial(object.researchFundGovernance) : undefined;
    message.seatElections = object.seatElections?.map(e => SeatElectionProposal.fromPartial(e)) || [];
    message.seatElectionVotes = object.seatElectionVotes?.map(e => SeatElectionVote.fromPartial(e)) || [];
    message.nextSeatElectionNumber = object.nextSeatElectionNumber !== undefined && object.nextSeatElectionNumber !== null ? BigInt(object.nextSeatElectionNumber.toString()) : BigInt(0);
    message.creedAmendmentPins = object.creedAmendmentPins?.map(e => GenesisCreedAmendmentPin.fromPartial(e)) || [];
    message.emergencyTransitionHold = object.emergencyTransitionHold !== undefined && object.emergencyTransitionHold !== null ? EmergencyTransitionHold.fromPartial(object.emergencyTransitionHold) : undefined;
    return message;
  }
};
//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/** ResearchFundPhase tracks the current governance phase of the research fund. */
export enum ResearchFundPhase {
  RESEARCH_FUND_PHASE_UNSPECIFIED = 0,
  /** RESEARCH_FUND_PHASE_GENESIS_PAIR - 2-of-2: founder + AI */
  RESEARCH_FUND_PHASE_GENESIS_PAIR = 1,
  /** RESEARCH_FUND_PHASE_OBSERVER - 2-of-3: founder + AI + 1 community */
  RESEARCH_FUND_PHASE_OBSERVER = 2,
  /** RESEARCH_FUND_PHASE_BALANCED - 3-of-5: founder + AI + 3 community */
  RESEARCH_FUND_PHASE_BALANCED = 3,
  /** RESEARCH_FUND_PHASE_FULL_GOVERNANCE - standard LIP process */
  RESEARCH_FUND_PHASE_FULL_GOVERNANCE = 4,
  UNRECOGNIZED = -1,
}
export function researchFundPhaseFromJSON(object: any): ResearchFundPhase {
  switch (object) {
    case 0:
    case "RESEARCH_FUND_PHASE_UNSPECIFIED":
      return ResearchFundPhase.RESEARCH_FUND_PHASE_UNSPECIFIED;
    case 1:
    case "RESEARCH_FUND_PHASE_GENESIS_PAIR":
      return ResearchFundPhase.RESEARCH_FUND_PHASE_GENESIS_PAIR;
    case 2:
    case "RESEARCH_FUND_PHASE_OBSERVER":
      return ResearchFundPhase.RESEARCH_FUND_PHASE_OBSERVER;
    case 3:
    case "RESEARCH_FUND_PHASE_BALANCED":
      return ResearchFundPhase.RESEARCH_FUND_PHASE_BALANCED;
    case 4:
    case "RESEARCH_FUND_PHASE_FULL_GOVERNANCE":
      return ResearchFundPhase.RESEARCH_FUND_PHASE_FULL_GOVERNANCE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ResearchFundPhase.UNRECOGNIZED;
  }
}
export function researchFundPhaseToJSON(object: ResearchFundPhase): string {
  switch (object) {
    case ResearchFundPhase.RESEARCH_FUND_PHASE_UNSPECIFIED:
      return "RESEARCH_FUND_PHASE_UNSPECIFIED";
    case ResearchFundPhase.RESEARCH_FUND_PHASE_GENESIS_PAIR:
      return "RESEARCH_FUND_PHASE_GENESIS_PAIR";
    case ResearchFundPhase.RESEARCH_FUND_PHASE_OBSERVER:
      return "RESEARCH_FUND_PHASE_OBSERVER";
    case ResearchFundPhase.RESEARCH_FUND_PHASE_BALANCED:
      return "RESEARCH_FUND_PHASE_BALANCED";
    case ResearchFundPhase.RESEARCH_FUND_PHASE_FULL_GOVERNANCE:
      return "RESEARCH_FUND_PHASE_FULL_GOVERNANCE";
    case ResearchFundPhase.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
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
function createBaseLIP(): LIP {
  return {
    id: "",
    title: "",
    description: "",
    category: "",
    proposer: "",
    stage: "",
    stakedAmount: "",
    yesStake: "",
    noStake: "",
    abstainStake: "",
    uniqueVoters: BigInt(0),
    createdAtBlock: BigInt(0),
    reviewStartedBlock: BigInt(0),
    lastCallStartedBlock: BigInt(0),
    votingEndBlock: BigInt(0),
    paramChanges: []
  };
}
/**
 * LIP represents a Legible Improvement Proposal.
 * @name LIP
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.LIP
 */
export const LIP = {
  typeUrl: "/zerone.gov.v1.LIP",
  encode(message: LIP, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
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
    if (message.proposer !== "") {
      writer.uint32(42).string(message.proposer);
    }
    if (message.stage !== "") {
      writer.uint32(50).string(message.stage);
    }
    if (message.stakedAmount !== "") {
      writer.uint32(58).string(message.stakedAmount);
    }
    if (message.yesStake !== "") {
      writer.uint32(66).string(message.yesStake);
    }
    if (message.noStake !== "") {
      writer.uint32(74).string(message.noStake);
    }
    if (message.abstainStake !== "") {
      writer.uint32(82).string(message.abstainStake);
    }
    if (message.uniqueVoters !== BigInt(0)) {
      writer.uint32(88).uint64(message.uniqueVoters);
    }
    if (message.createdAtBlock !== BigInt(0)) {
      writer.uint32(96).uint64(message.createdAtBlock);
    }
    if (message.reviewStartedBlock !== BigInt(0)) {
      writer.uint32(104).uint64(message.reviewStartedBlock);
    }
    if (message.lastCallStartedBlock !== BigInt(0)) {
      writer.uint32(112).uint64(message.lastCallStartedBlock);
    }
    if (message.votingEndBlock !== BigInt(0)) {
      writer.uint32(120).uint64(message.votingEndBlock);
    }
    for (const v of message.paramChanges) {
      ParamChange.encode(v!, writer.uint32(130).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): LIP {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseLIP();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
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
          message.proposer = reader.string();
          break;
        case 6:
          message.stage = reader.string();
          break;
        case 7:
          message.stakedAmount = reader.string();
          break;
        case 8:
          message.yesStake = reader.string();
          break;
        case 9:
          message.noStake = reader.string();
          break;
        case 10:
          message.abstainStake = reader.string();
          break;
        case 11:
          message.uniqueVoters = reader.uint64();
          break;
        case 12:
          message.createdAtBlock = reader.uint64();
          break;
        case 13:
          message.reviewStartedBlock = reader.uint64();
          break;
        case 14:
          message.lastCallStartedBlock = reader.uint64();
          break;
        case 15:
          message.votingEndBlock = reader.uint64();
          break;
        case 16:
          message.paramChanges.push(ParamChange.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<LIP>): LIP {
    const message = createBaseLIP();
    message.id = object.id ?? "";
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.category = object.category ?? "";
    message.proposer = object.proposer ?? "";
    message.stage = object.stage ?? "";
    message.stakedAmount = object.stakedAmount ?? "";
    message.yesStake = object.yesStake ?? "";
    message.noStake = object.noStake ?? "";
    message.abstainStake = object.abstainStake ?? "";
    message.uniqueVoters = object.uniqueVoters !== undefined && object.uniqueVoters !== null ? BigInt(object.uniqueVoters.toString()) : BigInt(0);
    message.createdAtBlock = object.createdAtBlock !== undefined && object.createdAtBlock !== null ? BigInt(object.createdAtBlock.toString()) : BigInt(0);
    message.reviewStartedBlock = object.reviewStartedBlock !== undefined && object.reviewStartedBlock !== null ? BigInt(object.reviewStartedBlock.toString()) : BigInt(0);
    message.lastCallStartedBlock = object.lastCallStartedBlock !== undefined && object.lastCallStartedBlock !== null ? BigInt(object.lastCallStartedBlock.toString()) : BigInt(0);
    message.votingEndBlock = object.votingEndBlock !== undefined && object.votingEndBlock !== null ? BigInt(object.votingEndBlock.toString()) : BigInt(0);
    message.paramChanges = object.paramChanges?.map(e => ParamChange.fromPartial(e)) || [];
    return message;
  }
};
function createBaseParamChange(): ParamChange {
  return {
    module: "",
    key: "",
    value: ""
  };
}
/**
 * ParamChange describes a single parameter modification.
 * @name ParamChange
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.ParamChange
 */
export const ParamChange = {
  typeUrl: "/zerone.gov.v1.ParamChange",
  encode(message: ParamChange, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.module !== "") {
      writer.uint32(10).string(message.module);
    }
    if (message.key !== "") {
      writer.uint32(18).string(message.key);
    }
    if (message.value !== "") {
      writer.uint32(26).string(message.value);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ParamChange {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseParamChange();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.module = reader.string();
          break;
        case 2:
          message.key = reader.string();
          break;
        case 3:
          message.value = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ParamChange>): ParamChange {
    const message = createBaseParamChange();
    message.module = object.module ?? "";
    message.key = object.key ?? "";
    message.value = object.value ?? "";
    return message;
  }
};
function createBaseVote(): Vote {
  return {
    lipId: "",
    voter: "",
    option: "",
    weight: ""
  };
}
/**
 * Vote records a single vote on a LIP.
 * @name Vote
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.Vote
 */
export const Vote = {
  typeUrl: "/zerone.gov.v1.Vote",
  encode(message: Vote, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.lipId !== "") {
      writer.uint32(10).string(message.lipId);
    }
    if (message.voter !== "") {
      writer.uint32(18).string(message.voter);
    }
    if (message.option !== "") {
      writer.uint32(26).string(message.option);
    }
    if (message.weight !== "") {
      writer.uint32(34).string(message.weight);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Vote {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseVote();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.lipId = reader.string();
          break;
        case 2:
          message.voter = reader.string();
          break;
        case 3:
          message.option = reader.string();
          break;
        case 4:
          message.weight = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Vote>): Vote {
    const message = createBaseVote();
    message.lipId = object.lipId ?? "";
    message.voter = object.voter ?? "";
    message.option = object.option ?? "";
    message.weight = object.weight ?? "";
    return message;
  }
};
function createBaseResearchFundVoters(): ResearchFundVoters {
  return {
    voter1: "",
    voter2: ""
  };
}
/**
 * ResearchFundVoters holds the 2-of-2 multisig addresses for research fund governance.
 * @name ResearchFundVoters
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.ResearchFundVoters
 */
export const ResearchFundVoters = {
  typeUrl: "/zerone.gov.v1.ResearchFundVoters",
  encode(message: ResearchFundVoters, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.voter1 !== "") {
      writer.uint32(10).string(message.voter1);
    }
    if (message.voter2 !== "") {
      writer.uint32(18).string(message.voter2);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ResearchFundVoters {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseResearchFundVoters();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.voter1 = reader.string();
          break;
        case 2:
          message.voter2 = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ResearchFundVoters>): ResearchFundVoters {
    const message = createBaseResearchFundVoters();
    message.voter1 = object.voter1 ?? "";
    message.voter2 = object.voter2 ?? "";
    return message;
  }
};
function createBaseUpgradePlan(): UpgradePlan {
  return {
    name: "",
    height: BigInt(0),
    info: ""
  };
}
/**
 * UpgradePlan describes a planned software upgrade attached to an upgrade-category LIP.
 * @name UpgradePlan
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.UpgradePlan
 */
export const UpgradePlan = {
  typeUrl: "/zerone.gov.v1.UpgradePlan",
  encode(message: UpgradePlan, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.name !== "") {
      writer.uint32(10).string(message.name);
    }
    if (message.height !== BigInt(0)) {
      writer.uint32(16).int64(message.height);
    }
    if (message.info !== "") {
      writer.uint32(26).string(message.info);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): UpgradePlan {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseUpgradePlan();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.name = reader.string();
          break;
        case 2:
          message.height = reader.int64();
          break;
        case 3:
          message.info = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<UpgradePlan>): UpgradePlan {
    const message = createBaseUpgradePlan();
    message.name = object.name ?? "";
    message.height = object.height !== undefined && object.height !== null ? BigInt(object.height.toString()) : BigInt(0);
    message.info = object.info ?? "";
    return message;
  }
};
function createBaseResearchFundGovernanceState(): ResearchFundGovernanceState {
  return {
    currentPhase: 0,
    phaseStartedAtBlock: BigInt(0),
    proposalsExecutedInPhase: BigInt(0),
    lastTransitionBlock: BigInt(0),
    communitySeats: [],
    seatTermEndBlocks: [],
    rollbackCooldownUntil: BigInt(0)
  };
}
/**
 * ResearchFundGovernanceState tracks the research fund governance lifecycle.
 * @name ResearchFundGovernanceState
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.ResearchFundGovernanceState
 */
export const ResearchFundGovernanceState = {
  typeUrl: "/zerone.gov.v1.ResearchFundGovernanceState",
  encode(message: ResearchFundGovernanceState, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.currentPhase !== 0) {
      writer.uint32(8).int32(message.currentPhase);
    }
    if (message.phaseStartedAtBlock !== BigInt(0)) {
      writer.uint32(16).uint64(message.phaseStartedAtBlock);
    }
    if (message.proposalsExecutedInPhase !== BigInt(0)) {
      writer.uint32(24).uint64(message.proposalsExecutedInPhase);
    }
    if (message.lastTransitionBlock !== BigInt(0)) {
      writer.uint32(32).uint64(message.lastTransitionBlock);
    }
    for (const v of message.communitySeats) {
      writer.uint32(42).string(v!);
    }
    writer.uint32(50).fork();
    for (const v of message.seatTermEndBlocks) {
      writer.uint64(v);
    }
    writer.ldelim();
    if (message.rollbackCooldownUntil !== BigInt(0)) {
      writer.uint32(56).uint64(message.rollbackCooldownUntil);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ResearchFundGovernanceState {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseResearchFundGovernanceState();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.currentPhase = reader.int32() as any;
          break;
        case 2:
          message.phaseStartedAtBlock = reader.uint64();
          break;
        case 3:
          message.proposalsExecutedInPhase = reader.uint64();
          break;
        case 4:
          message.lastTransitionBlock = reader.uint64();
          break;
        case 5:
          message.communitySeats.push(reader.string());
          break;
        case 6:
          if ((tag & 7) === 2) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.seatTermEndBlocks.push(reader.uint64());
            }
          } else {
            message.seatTermEndBlocks.push(reader.uint64());
          }
          break;
        case 7:
          message.rollbackCooldownUntil = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ResearchFundGovernanceState>): ResearchFundGovernanceState {
    const message = createBaseResearchFundGovernanceState();
    message.currentPhase = object.currentPhase ?? 0;
    message.phaseStartedAtBlock = object.phaseStartedAtBlock !== undefined && object.phaseStartedAtBlock !== null ? BigInt(object.phaseStartedAtBlock.toString()) : BigInt(0);
    message.proposalsExecutedInPhase = object.proposalsExecutedInPhase !== undefined && object.proposalsExecutedInPhase !== null ? BigInt(object.proposalsExecutedInPhase.toString()) : BigInt(0);
    message.lastTransitionBlock = object.lastTransitionBlock !== undefined && object.lastTransitionBlock !== null ? BigInt(object.lastTransitionBlock.toString()) : BigInt(0);
    message.communitySeats = object.communitySeats?.map(e => e) || [];
    message.seatTermEndBlocks = object.seatTermEndBlocks?.map(e => BigInt(e.toString())) || [];
    message.rollbackCooldownUntil = object.rollbackCooldownUntil !== undefined && object.rollbackCooldownUntil !== null ? BigInt(object.rollbackCooldownUntil.toString()) : BigInt(0);
    return message;
  }
};
function createBasePhaseTransitionConditions(): PhaseTransitionConditions {
  return {
    distinctLipVoters: BigInt(0),
    activeGuardians: BigInt(0),
    researchFundBalance: "",
    chainAgeBlocks: BigInt(0),
    proposalsExecutedInPhase: BigInt(0),
    communitySeatParticipation: BigInt(0),
    emergencyHaltsFromMisuse: BigInt(0)
  };
}
/**
 * PhaseTransitionConditions records the metrics at time of transition proposal.
 * @name PhaseTransitionConditions
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.PhaseTransitionConditions
 */
export const PhaseTransitionConditions = {
  typeUrl: "/zerone.gov.v1.PhaseTransitionConditions",
  encode(message: PhaseTransitionConditions, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.distinctLipVoters !== BigInt(0)) {
      writer.uint32(8).uint64(message.distinctLipVoters);
    }
    if (message.activeGuardians !== BigInt(0)) {
      writer.uint32(16).uint64(message.activeGuardians);
    }
    if (message.researchFundBalance !== "") {
      writer.uint32(26).string(message.researchFundBalance);
    }
    if (message.chainAgeBlocks !== BigInt(0)) {
      writer.uint32(32).uint64(message.chainAgeBlocks);
    }
    if (message.proposalsExecutedInPhase !== BigInt(0)) {
      writer.uint32(40).uint64(message.proposalsExecutedInPhase);
    }
    if (message.communitySeatParticipation !== BigInt(0)) {
      writer.uint32(48).uint64(message.communitySeatParticipation);
    }
    if (message.emergencyHaltsFromMisuse !== BigInt(0)) {
      writer.uint32(56).uint64(message.emergencyHaltsFromMisuse);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): PhaseTransitionConditions {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBasePhaseTransitionConditions();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.distinctLipVoters = reader.uint64();
          break;
        case 2:
          message.activeGuardians = reader.uint64();
          break;
        case 3:
          message.researchFundBalance = reader.string();
          break;
        case 4:
          message.chainAgeBlocks = reader.uint64();
          break;
        case 5:
          message.proposalsExecutedInPhase = reader.uint64();
          break;
        case 6:
          message.communitySeatParticipation = reader.uint64();
          break;
        case 7:
          message.emergencyHaltsFromMisuse = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<PhaseTransitionConditions>): PhaseTransitionConditions {
    const message = createBasePhaseTransitionConditions();
    message.distinctLipVoters = object.distinctLipVoters !== undefined && object.distinctLipVoters !== null ? BigInt(object.distinctLipVoters.toString()) : BigInt(0);
    message.activeGuardians = object.activeGuardians !== undefined && object.activeGuardians !== null ? BigInt(object.activeGuardians.toString()) : BigInt(0);
    message.researchFundBalance = object.researchFundBalance ?? "";
    message.chainAgeBlocks = object.chainAgeBlocks !== undefined && object.chainAgeBlocks !== null ? BigInt(object.chainAgeBlocks.toString()) : BigInt(0);
    message.proposalsExecutedInPhase = object.proposalsExecutedInPhase !== undefined && object.proposalsExecutedInPhase !== null ? BigInt(object.proposalsExecutedInPhase.toString()) : BigInt(0);
    message.communitySeatParticipation = object.communitySeatParticipation !== undefined && object.communitySeatParticipation !== null ? BigInt(object.communitySeatParticipation.toString()) : BigInt(0);
    message.emergencyHaltsFromMisuse = object.emergencyHaltsFromMisuse !== undefined && object.emergencyHaltsFromMisuse !== null ? BigInt(object.emergencyHaltsFromMisuse.toString()) : BigInt(0);
    return message;
  }
};
function createBaseResearchSpendProposal(): ResearchSpendProposal {
  return {
    proposalId: BigInt(0),
    proposer: "",
    title: "",
    description: "",
    recipient: "",
    amount: "",
    justification: "",
    stage: "",
    createdAt: BigInt(0),
    votingStartsAt: BigInt(0),
    votingEndsAt: BigInt(0),
    voter1Vote: "",
    voter1Reason: "",
    voter1VotedAt: BigInt(0),
    voter2Vote: "",
    voter2Reason: "",
    voter2VotedAt: BigInt(0),
    executedAt: BigInt(0),
    executionErr: ""
  };
}
/**
 * ResearchSpendProposal represents a 2-of-2 research fund spending proposal.
 * @name ResearchSpendProposal
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.ResearchSpendProposal
 */
export const ResearchSpendProposal = {
  typeUrl: "/zerone.gov.v1.ResearchSpendProposal",
  encode(message: ResearchSpendProposal, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposalId !== BigInt(0)) {
      writer.uint32(8).uint64(message.proposalId);
    }
    if (message.proposer !== "") {
      writer.uint32(18).string(message.proposer);
    }
    if (message.title !== "") {
      writer.uint32(26).string(message.title);
    }
    if (message.description !== "") {
      writer.uint32(34).string(message.description);
    }
    if (message.recipient !== "") {
      writer.uint32(42).string(message.recipient);
    }
    if (message.amount !== "") {
      writer.uint32(50).string(message.amount);
    }
    if (message.justification !== "") {
      writer.uint32(58).string(message.justification);
    }
    if (message.stage !== "") {
      writer.uint32(66).string(message.stage);
    }
    if (message.createdAt !== BigInt(0)) {
      writer.uint32(72).uint64(message.createdAt);
    }
    if (message.votingStartsAt !== BigInt(0)) {
      writer.uint32(80).uint64(message.votingStartsAt);
    }
    if (message.votingEndsAt !== BigInt(0)) {
      writer.uint32(88).uint64(message.votingEndsAt);
    }
    if (message.voter1Vote !== "") {
      writer.uint32(98).string(message.voter1Vote);
    }
    if (message.voter1Reason !== "") {
      writer.uint32(106).string(message.voter1Reason);
    }
    if (message.voter1VotedAt !== BigInt(0)) {
      writer.uint32(112).uint64(message.voter1VotedAt);
    }
    if (message.voter2Vote !== "") {
      writer.uint32(122).string(message.voter2Vote);
    }
    if (message.voter2Reason !== "") {
      writer.uint32(130).string(message.voter2Reason);
    }
    if (message.voter2VotedAt !== BigInt(0)) {
      writer.uint32(136).uint64(message.voter2VotedAt);
    }
    if (message.executedAt !== BigInt(0)) {
      writer.uint32(144).uint64(message.executedAt);
    }
    if (message.executionErr !== "") {
      writer.uint32(154).string(message.executionErr);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ResearchSpendProposal {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseResearchSpendProposal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposalId = reader.uint64();
          break;
        case 2:
          message.proposer = reader.string();
          break;
        case 3:
          message.title = reader.string();
          break;
        case 4:
          message.description = reader.string();
          break;
        case 5:
          message.recipient = reader.string();
          break;
        case 6:
          message.amount = reader.string();
          break;
        case 7:
          message.justification = reader.string();
          break;
        case 8:
          message.stage = reader.string();
          break;
        case 9:
          message.createdAt = reader.uint64();
          break;
        case 10:
          message.votingStartsAt = reader.uint64();
          break;
        case 11:
          message.votingEndsAt = reader.uint64();
          break;
        case 12:
          message.voter1Vote = reader.string();
          break;
        case 13:
          message.voter1Reason = reader.string();
          break;
        case 14:
          message.voter1VotedAt = reader.uint64();
          break;
        case 15:
          message.voter2Vote = reader.string();
          break;
        case 16:
          message.voter2Reason = reader.string();
          break;
        case 17:
          message.voter2VotedAt = reader.uint64();
          break;
        case 18:
          message.executedAt = reader.uint64();
          break;
        case 19:
          message.executionErr = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ResearchSpendProposal>): ResearchSpendProposal {
    const message = createBaseResearchSpendProposal();
    message.proposalId = object.proposalId !== undefined && object.proposalId !== null ? BigInt(object.proposalId.toString()) : BigInt(0);
    message.proposer = object.proposer ?? "";
    message.title = object.title ?? "";
    message.description = object.description ?? "";
    message.recipient = object.recipient ?? "";
    message.amount = object.amount ?? "";
    message.justification = object.justification ?? "";
    message.stage = object.stage ?? "";
    message.createdAt = object.createdAt !== undefined && object.createdAt !== null ? BigInt(object.createdAt.toString()) : BigInt(0);
    message.votingStartsAt = object.votingStartsAt !== undefined && object.votingStartsAt !== null ? BigInt(object.votingStartsAt.toString()) : BigInt(0);
    message.votingEndsAt = object.votingEndsAt !== undefined && object.votingEndsAt !== null ? BigInt(object.votingEndsAt.toString()) : BigInt(0);
    message.voter1Vote = object.voter1Vote ?? "";
    message.voter1Reason = object.voter1Reason ?? "";
    message.voter1VotedAt = object.voter1VotedAt !== undefined && object.voter1VotedAt !== null ? BigInt(object.voter1VotedAt.toString()) : BigInt(0);
    message.voter2Vote = object.voter2Vote ?? "";
    message.voter2Reason = object.voter2Reason ?? "";
    message.voter2VotedAt = object.voter2VotedAt !== undefined && object.voter2VotedAt !== null ? BigInt(object.voter2VotedAt.toString()) : BigInt(0);
    message.executedAt = object.executedAt !== undefined && object.executedAt !== null ? BigInt(object.executedAt.toString()) : BigInt(0);
    message.executionErr = object.executionErr ?? "";
    return message;
  }
};
function createBaseSeatElectionProposal(): SeatElectionProposal {
  return {
    proposalId: BigInt(0),
    proposer: "",
    candidate: "",
    seatIndex: 0,
    statement: "",
    stage: "",
    yesStake: "",
    noStake: "",
    abstainStake: "",
    acceptanceDeadline: BigInt(0),
    discussionEndBlock: BigInt(0),
    votingEndBlock: BigInt(0),
    createdAtBlock: BigInt(0),
    candidateAccepted: false,
    isRunoff: false,
    runoffParentIds: []
  };
}
/**
 * SeatElectionProposal nominates a candidate for a research fund community seat.
 * @name SeatElectionProposal
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.SeatElectionProposal
 */
export const SeatElectionProposal = {
  typeUrl: "/zerone.gov.v1.SeatElectionProposal",
  encode(message: SeatElectionProposal, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposalId !== BigInt(0)) {
      writer.uint32(8).uint64(message.proposalId);
    }
    if (message.proposer !== "") {
      writer.uint32(18).string(message.proposer);
    }
    if (message.candidate !== "") {
      writer.uint32(26).string(message.candidate);
    }
    if (message.seatIndex !== 0) {
      writer.uint32(32).uint32(message.seatIndex);
    }
    if (message.statement !== "") {
      writer.uint32(42).string(message.statement);
    }
    if (message.stage !== "") {
      writer.uint32(50).string(message.stage);
    }
    if (message.yesStake !== "") {
      writer.uint32(58).string(message.yesStake);
    }
    if (message.noStake !== "") {
      writer.uint32(66).string(message.noStake);
    }
    if (message.abstainStake !== "") {
      writer.uint32(74).string(message.abstainStake);
    }
    if (message.acceptanceDeadline !== BigInt(0)) {
      writer.uint32(80).uint64(message.acceptanceDeadline);
    }
    if (message.discussionEndBlock !== BigInt(0)) {
      writer.uint32(88).uint64(message.discussionEndBlock);
    }
    if (message.votingEndBlock !== BigInt(0)) {
      writer.uint32(96).uint64(message.votingEndBlock);
    }
    if (message.createdAtBlock !== BigInt(0)) {
      writer.uint32(104).uint64(message.createdAtBlock);
    }
    if (message.candidateAccepted === true) {
      writer.uint32(112).bool(message.candidateAccepted);
    }
    if (message.isRunoff === true) {
      writer.uint32(120).bool(message.isRunoff);
    }
    writer.uint32(130).fork();
    for (const v of message.runoffParentIds) {
      writer.uint64(v);
    }
    writer.ldelim();
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): SeatElectionProposal {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSeatElectionProposal();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposalId = reader.uint64();
          break;
        case 2:
          message.proposer = reader.string();
          break;
        case 3:
          message.candidate = reader.string();
          break;
        case 4:
          message.seatIndex = reader.uint32();
          break;
        case 5:
          message.statement = reader.string();
          break;
        case 6:
          message.stage = reader.string();
          break;
        case 7:
          message.yesStake = reader.string();
          break;
        case 8:
          message.noStake = reader.string();
          break;
        case 9:
          message.abstainStake = reader.string();
          break;
        case 10:
          message.acceptanceDeadline = reader.uint64();
          break;
        case 11:
          message.discussionEndBlock = reader.uint64();
          break;
        case 12:
          message.votingEndBlock = reader.uint64();
          break;
        case 13:
          message.createdAtBlock = reader.uint64();
          break;
        case 14:
          message.candidateAccepted = reader.bool();
          break;
        case 15:
          message.isRunoff = reader.bool();
          break;
        case 16:
          if ((tag & 7) === 2) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.runoffParentIds.push(reader.uint64());
            }
          } else {
            message.runoffParentIds.push(reader.uint64());
          }
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<SeatElectionProposal>): SeatElectionProposal {
    const message = createBaseSeatElectionProposal();
    message.proposalId = object.proposalId !== undefined && object.proposalId !== null ? BigInt(object.proposalId.toString()) : BigInt(0);
    message.proposer = object.proposer ?? "";
    message.candidate = object.candidate ?? "";
    message.seatIndex = object.seatIndex ?? 0;
    message.statement = object.statement ?? "";
    message.stage = object.stage ?? "";
    message.yesStake = object.yesStake ?? "";
    message.noStake = object.noStake ?? "";
    message.abstainStake = object.abstainStake ?? "";
    message.acceptanceDeadline = object.acceptanceDeadline !== undefined && object.acceptanceDeadline !== null ? BigInt(object.acceptanceDeadline.toString()) : BigInt(0);
    message.discussionEndBlock = object.discussionEndBlock !== undefined && object.discussionEndBlock !== null ? BigInt(object.discussionEndBlock.toString()) : BigInt(0);
    message.votingEndBlock = object.votingEndBlock !== undefined && object.votingEndBlock !== null ? BigInt(object.votingEndBlock.toString()) : BigInt(0);
    message.createdAtBlock = object.createdAtBlock !== undefined && object.createdAtBlock !== null ? BigInt(object.createdAtBlock.toString()) : BigInt(0);
    message.candidateAccepted = object.candidateAccepted ?? false;
    message.isRunoff = object.isRunoff ?? false;
    message.runoffParentIds = object.runoffParentIds?.map(e => BigInt(e.toString())) || [];
    return message;
  }
};
function createBaseSeatElectionVote(): SeatElectionVote {
  return {
    proposalId: BigInt(0),
    voter: "",
    option: "",
    stake: "",
    block: BigInt(0)
  };
}
/**
 * SeatElectionVote records a single vote on a seat election proposal.
 * @name SeatElectionVote
 * @package zerone.gov.v1
 * @see proto type: zerone.gov.v1.SeatElectionVote
 */
export const SeatElectionVote = {
  typeUrl: "/zerone.gov.v1.SeatElectionVote",
  encode(message: SeatElectionVote, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposalId !== BigInt(0)) {
      writer.uint32(8).uint64(message.proposalId);
    }
    if (message.voter !== "") {
      writer.uint32(18).string(message.voter);
    }
    if (message.option !== "") {
      writer.uint32(26).string(message.option);
    }
    if (message.stake !== "") {
      writer.uint32(34).string(message.stake);
    }
    if (message.block !== BigInt(0)) {
      writer.uint32(40).uint64(message.block);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): SeatElectionVote {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseSeatElectionVote();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposalId = reader.uint64();
          break;
        case 2:
          message.voter = reader.string();
          break;
        case 3:
          message.option = reader.string();
          break;
        case 4:
          message.stake = reader.string();
          break;
        case 5:
          message.block = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<SeatElectionVote>): SeatElectionVote {
    const message = createBaseSeatElectionVote();
    message.proposalId = object.proposalId !== undefined && object.proposalId !== null ? BigInt(object.proposalId.toString()) : BigInt(0);
    message.voter = object.voter ?? "";
    message.option = object.option ?? "";
    message.stake = object.stake ?? "";
    message.block = object.block !== undefined && object.block !== null ? BigInt(object.block.toString()) : BigInt(0);
    return message;
  }
};
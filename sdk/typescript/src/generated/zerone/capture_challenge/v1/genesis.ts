//@ts-nocheck
import { CaptureChallenge, DomainBountyPool } from "./types";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * @name Params
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.Params
 */
export interface Params {
  /**
   * minimum uzrn to stake a challenge
   */
  minChallengeStake: string;
  /**
   * blocks allowed for evidence submission
   */
  evidencePeriodBlocks: bigint;
  /**
   * blocks allowed for governance review
   */
  reviewPeriodBlocks: bigint;
  /**
   * blocks domain is paused on challenge
   */
  domainPauseBlocks: bigint;
  /**
   * reward from bounty pool (BPS of pool)
   */
  rewardRateBps: bigint;
  /**
   * slash of accused validators (BPS of stake)
   */
  slashRateBps: bigint;
  /**
   * uzrn per fact creation auto-funded
   */
  bountyContributionPerFact: string;
  /**
   * blocks between auto risk analysis
   */
  riskAnalysisInterval: bigint;
}
/**
 * @name GenesisState
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.GenesisState
 */
export interface GenesisState {
  params?: Params;
  challenges: CaptureChallenge[];
  bountyPools: DomainBountyPool[];
}
function createBaseParams(): Params {
  return {
    minChallengeStake: "",
    evidencePeriodBlocks: BigInt(0),
    reviewPeriodBlocks: BigInt(0),
    domainPauseBlocks: BigInt(0),
    rewardRateBps: BigInt(0),
    slashRateBps: BigInt(0),
    bountyContributionPerFact: "",
    riskAnalysisInterval: BigInt(0)
  };
}
/**
 * @name Params
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.Params
 */
export const Params = {
  typeUrl: "/zerone.capture_challenge.v1.Params",
  encode(message: Params, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.minChallengeStake !== "") {
      writer.uint32(10).string(message.minChallengeStake);
    }
    if (message.evidencePeriodBlocks !== BigInt(0)) {
      writer.uint32(16).uint64(message.evidencePeriodBlocks);
    }
    if (message.reviewPeriodBlocks !== BigInt(0)) {
      writer.uint32(24).uint64(message.reviewPeriodBlocks);
    }
    if (message.domainPauseBlocks !== BigInt(0)) {
      writer.uint32(32).uint64(message.domainPauseBlocks);
    }
    if (message.rewardRateBps !== BigInt(0)) {
      writer.uint32(40).uint64(message.rewardRateBps);
    }
    if (message.slashRateBps !== BigInt(0)) {
      writer.uint32(48).uint64(message.slashRateBps);
    }
    if (message.bountyContributionPerFact !== "") {
      writer.uint32(58).string(message.bountyContributionPerFact);
    }
    if (message.riskAnalysisInterval !== BigInt(0)) {
      writer.uint32(64).uint64(message.riskAnalysisInterval);
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
          message.minChallengeStake = reader.string();
          break;
        case 2:
          message.evidencePeriodBlocks = reader.uint64();
          break;
        case 3:
          message.reviewPeriodBlocks = reader.uint64();
          break;
        case 4:
          message.domainPauseBlocks = reader.uint64();
          break;
        case 5:
          message.rewardRateBps = reader.uint64();
          break;
        case 6:
          message.slashRateBps = reader.uint64();
          break;
        case 7:
          message.bountyContributionPerFact = reader.string();
          break;
        case 8:
          message.riskAnalysisInterval = reader.uint64();
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
    message.minChallengeStake = object.minChallengeStake ?? "";
    message.evidencePeriodBlocks = object.evidencePeriodBlocks !== undefined && object.evidencePeriodBlocks !== null ? BigInt(object.evidencePeriodBlocks.toString()) : BigInt(0);
    message.reviewPeriodBlocks = object.reviewPeriodBlocks !== undefined && object.reviewPeriodBlocks !== null ? BigInt(object.reviewPeriodBlocks.toString()) : BigInt(0);
    message.domainPauseBlocks = object.domainPauseBlocks !== undefined && object.domainPauseBlocks !== null ? BigInt(object.domainPauseBlocks.toString()) : BigInt(0);
    message.rewardRateBps = object.rewardRateBps !== undefined && object.rewardRateBps !== null ? BigInt(object.rewardRateBps.toString()) : BigInt(0);
    message.slashRateBps = object.slashRateBps !== undefined && object.slashRateBps !== null ? BigInt(object.slashRateBps.toString()) : BigInt(0);
    message.bountyContributionPerFact = object.bountyContributionPerFact ?? "";
    message.riskAnalysisInterval = object.riskAnalysisInterval !== undefined && object.riskAnalysisInterval !== null ? BigInt(object.riskAnalysisInterval.toString()) : BigInt(0);
    return message;
  }
};
function createBaseGenesisState(): GenesisState {
  return {
    params: undefined,
    challenges: [],
    bountyPools: []
  };
}
/**
 * @name GenesisState
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.GenesisState
 */
export const GenesisState = {
  typeUrl: "/zerone.capture_challenge.v1.GenesisState",
  encode(message: GenesisState, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.challenges) {
      CaptureChallenge.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.bountyPools) {
      DomainBountyPool.encode(v!, writer.uint32(26).fork()).ldelim();
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
          message.challenges.push(CaptureChallenge.decode(reader, reader.uint32()));
          break;
        case 3:
          message.bountyPools.push(DomainBountyPool.decode(reader, reader.uint32()));
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
    message.challenges = object.challenges?.map(e => CaptureChallenge.fromPartial(e)) || [];
    message.bountyPools = object.bountyPools?.map(e => DomainBountyPool.fromPartial(e)) || [];
    return message;
  }
};
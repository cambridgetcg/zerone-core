//@ts-nocheck
import { GlobalReputation, StratumReputation, DomainReputation, CaptureMetrics, CrossStratumRequirement } from "./types";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * @name Params
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.Params
 */
export interface Params {
  /**
   * half-life for reputation decay
   */
  decayEpochBlocks: bigint;
  /**
   * min verifications before domain rep is used
   */
  minVerificationsForScore: bigint;
  /**
   * HHI above which domain is flagged
   */
  hhiThreshold: bigint;
  /**
   * blocks between auto-analysis
   */
  riskAnalysisInterval: bigint;
  /**
   * how long verification history is kept
   */
  historyRetentionBlocks: bigint;
  /**
   * floor for decayed scores
   */
  baseReputationScore: bigint;
  /**
   * max rounds kept per domain
   */
  maxHistoryPerDomain: bigint;
  /**
   * base recovery rate per decay epoch (default: 50,000 = 5%)
   */
  baseReputationRecoveryBps: bigint;
  /**
   * max acceleration from verification activity (default: 500,000 = 50%)
   */
  activityRecoveryBonusMaxBps: bigint;
}
/**
 * @name GenesisState
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.GenesisState
 */
export interface GenesisState {
  params?: Params;
  globalReputations: GlobalReputation[];
  stratumReputations: StratumReputation[];
  domainReputations: DomainReputation[];
  captureMetrics: CaptureMetrics[];
  crossStratumRequirements: CrossStratumRequirement[];
}
function createBaseParams(): Params {
  return {
    decayEpochBlocks: BigInt(0),
    minVerificationsForScore: BigInt(0),
    hhiThreshold: BigInt(0),
    riskAnalysisInterval: BigInt(0),
    historyRetentionBlocks: BigInt(0),
    baseReputationScore: BigInt(0),
    maxHistoryPerDomain: BigInt(0),
    baseReputationRecoveryBps: BigInt(0),
    activityRecoveryBonusMaxBps: BigInt(0)
  };
}
/**
 * @name Params
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.Params
 */
export const Params = {
  typeUrl: "/zerone.capture_defense.v1.Params",
  encode(message: Params, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.decayEpochBlocks !== BigInt(0)) {
      writer.uint32(8).uint64(message.decayEpochBlocks);
    }
    if (message.minVerificationsForScore !== BigInt(0)) {
      writer.uint32(16).uint64(message.minVerificationsForScore);
    }
    if (message.hhiThreshold !== BigInt(0)) {
      writer.uint32(24).uint64(message.hhiThreshold);
    }
    if (message.riskAnalysisInterval !== BigInt(0)) {
      writer.uint32(32).uint64(message.riskAnalysisInterval);
    }
    if (message.historyRetentionBlocks !== BigInt(0)) {
      writer.uint32(40).uint64(message.historyRetentionBlocks);
    }
    if (message.baseReputationScore !== BigInt(0)) {
      writer.uint32(48).uint64(message.baseReputationScore);
    }
    if (message.maxHistoryPerDomain !== BigInt(0)) {
      writer.uint32(56).uint64(message.maxHistoryPerDomain);
    }
    if (message.baseReputationRecoveryBps !== BigInt(0)) {
      writer.uint32(64).uint64(message.baseReputationRecoveryBps);
    }
    if (message.activityRecoveryBonusMaxBps !== BigInt(0)) {
      writer.uint32(72).uint64(message.activityRecoveryBonusMaxBps);
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
          message.decayEpochBlocks = reader.uint64();
          break;
        case 2:
          message.minVerificationsForScore = reader.uint64();
          break;
        case 3:
          message.hhiThreshold = reader.uint64();
          break;
        case 4:
          message.riskAnalysisInterval = reader.uint64();
          break;
        case 5:
          message.historyRetentionBlocks = reader.uint64();
          break;
        case 6:
          message.baseReputationScore = reader.uint64();
          break;
        case 7:
          message.maxHistoryPerDomain = reader.uint64();
          break;
        case 8:
          message.baseReputationRecoveryBps = reader.uint64();
          break;
        case 9:
          message.activityRecoveryBonusMaxBps = reader.uint64();
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
    message.decayEpochBlocks = object.decayEpochBlocks !== undefined && object.decayEpochBlocks !== null ? BigInt(object.decayEpochBlocks.toString()) : BigInt(0);
    message.minVerificationsForScore = object.minVerificationsForScore !== undefined && object.minVerificationsForScore !== null ? BigInt(object.minVerificationsForScore.toString()) : BigInt(0);
    message.hhiThreshold = object.hhiThreshold !== undefined && object.hhiThreshold !== null ? BigInt(object.hhiThreshold.toString()) : BigInt(0);
    message.riskAnalysisInterval = object.riskAnalysisInterval !== undefined && object.riskAnalysisInterval !== null ? BigInt(object.riskAnalysisInterval.toString()) : BigInt(0);
    message.historyRetentionBlocks = object.historyRetentionBlocks !== undefined && object.historyRetentionBlocks !== null ? BigInt(object.historyRetentionBlocks.toString()) : BigInt(0);
    message.baseReputationScore = object.baseReputationScore !== undefined && object.baseReputationScore !== null ? BigInt(object.baseReputationScore.toString()) : BigInt(0);
    message.maxHistoryPerDomain = object.maxHistoryPerDomain !== undefined && object.maxHistoryPerDomain !== null ? BigInt(object.maxHistoryPerDomain.toString()) : BigInt(0);
    message.baseReputationRecoveryBps = object.baseReputationRecoveryBps !== undefined && object.baseReputationRecoveryBps !== null ? BigInt(object.baseReputationRecoveryBps.toString()) : BigInt(0);
    message.activityRecoveryBonusMaxBps = object.activityRecoveryBonusMaxBps !== undefined && object.activityRecoveryBonusMaxBps !== null ? BigInt(object.activityRecoveryBonusMaxBps.toString()) : BigInt(0);
    return message;
  }
};
function createBaseGenesisState(): GenesisState {
  return {
    params: undefined,
    globalReputations: [],
    stratumReputations: [],
    domainReputations: [],
    captureMetrics: [],
    crossStratumRequirements: []
  };
}
/**
 * @name GenesisState
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.GenesisState
 */
export const GenesisState = {
  typeUrl: "/zerone.capture_defense.v1.GenesisState",
  encode(message: GenesisState, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.globalReputations) {
      GlobalReputation.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.stratumReputations) {
      StratumReputation.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.domainReputations) {
      DomainReputation.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.captureMetrics) {
      CaptureMetrics.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    for (const v of message.crossStratumRequirements) {
      CrossStratumRequirement.encode(v!, writer.uint32(50).fork()).ldelim();
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
          message.globalReputations.push(GlobalReputation.decode(reader, reader.uint32()));
          break;
        case 3:
          message.stratumReputations.push(StratumReputation.decode(reader, reader.uint32()));
          break;
        case 4:
          message.domainReputations.push(DomainReputation.decode(reader, reader.uint32()));
          break;
        case 5:
          message.captureMetrics.push(CaptureMetrics.decode(reader, reader.uint32()));
          break;
        case 6:
          message.crossStratumRequirements.push(CrossStratumRequirement.decode(reader, reader.uint32()));
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
    message.globalReputations = object.globalReputations?.map(e => GlobalReputation.fromPartial(e)) || [];
    message.stratumReputations = object.stratumReputations?.map(e => StratumReputation.fromPartial(e)) || [];
    message.domainReputations = object.domainReputations?.map(e => DomainReputation.fromPartial(e)) || [];
    message.captureMetrics = object.captureMetrics?.map(e => CaptureMetrics.fromPartial(e)) || [];
    message.crossStratumRequirements = object.crossStratumRequirements?.map(e => CrossStratumRequirement.fromPartial(e)) || [];
    return message;
  }
};
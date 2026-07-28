//@ts-nocheck
import { AlignmentState, AlignmentObservation, DimensionScores, AlignmentHealthIndex, CorrectionRecord } from "./types";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * GenesisState defines the alignment module's genesis state.
 * @name GenesisState
 * @package zerone.alignment.v1
 * @see proto type: zerone.alignment.v1.GenesisState
 */
export interface GenesisState {
  params?: Params;
  state?: AlignmentState;
  observations: AlignmentObservation[];
  scores: DimensionScores[];
  healthIndices: AlignmentHealthIndex[];
  corrections: CorrectionRecord[];
}
/**
 * Params defines the alignment module parameters.
 * @name Params
 * @package zerone.alignment.v1
 * @see proto type: zerone.alignment.v1.Params
 */
export interface Params {
  /**
   * How often to observe (in blocks).
   */
  observationIntervalBlocks: bigint;
  /**
   * Dimension weights (BPS, must sum to 1,000,000).
   */
  weightKnowledgeQuality: bigint;
  weightEconomicStability: bigint;
  weightGovernanceParticipation: bigint;
  weightNetworkSecurity: bigint;
  weightStakingRatio: bigint;
  /**
   * Health category thresholds (BPS).
   */
  criticalThreshold: bigint;
  degradedThreshold: bigint;
  healthyThreshold: bigint;
  /**
   * Module enabled flag.
   */
  enabled: boolean;
  /**
   * Maximum correction magnitude (BPS) for auto-apply. Above this requires governance.
   */
  maxAutoApplyMagnitudeBps: bigint;
  /**
   * Correction confidence params (R29-4).
   */
  correctionConfidenceWindowSize: bigint;
  correctionConfidenceMinSamples: bigint;
  minConfidenceForAutoApply: bigint;
  correctionBoundsMinMultiplierBps: bigint;
  correctionBoundsMaxMultiplierBps: bigint;
  /**
   * Correction banding (L7). Corrections smaller than this are *advisory only*:
   * logged + event emitted, but not forwarded to autopoiesis. Prevents the
   * module from chattering at every small deviation.
   */
  advisoryMagnitudeBps: bigint;
}
function createBaseGenesisState(): GenesisState {
  return {
    params: undefined,
    state: undefined,
    observations: [],
    scores: [],
    healthIndices: [],
    corrections: []
  };
}
/**
 * GenesisState defines the alignment module's genesis state.
 * @name GenesisState
 * @package zerone.alignment.v1
 * @see proto type: zerone.alignment.v1.GenesisState
 */
export const GenesisState = {
  typeUrl: "/zerone.alignment.v1.GenesisState",
  encode(message: GenesisState, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    if (message.state !== undefined) {
      AlignmentState.encode(message.state, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.observations) {
      AlignmentObservation.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.scores) {
      DimensionScores.encode(v!, writer.uint32(34).fork()).ldelim();
    }
    for (const v of message.healthIndices) {
      AlignmentHealthIndex.encode(v!, writer.uint32(42).fork()).ldelim();
    }
    for (const v of message.corrections) {
      CorrectionRecord.encode(v!, writer.uint32(50).fork()).ldelim();
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
          message.state = AlignmentState.decode(reader, reader.uint32());
          break;
        case 3:
          message.observations.push(AlignmentObservation.decode(reader, reader.uint32()));
          break;
        case 4:
          message.scores.push(DimensionScores.decode(reader, reader.uint32()));
          break;
        case 5:
          message.healthIndices.push(AlignmentHealthIndex.decode(reader, reader.uint32()));
          break;
        case 6:
          message.corrections.push(CorrectionRecord.decode(reader, reader.uint32()));
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
    message.state = object.state !== undefined && object.state !== null ? AlignmentState.fromPartial(object.state) : undefined;
    message.observations = object.observations?.map(e => AlignmentObservation.fromPartial(e)) || [];
    message.scores = object.scores?.map(e => DimensionScores.fromPartial(e)) || [];
    message.healthIndices = object.healthIndices?.map(e => AlignmentHealthIndex.fromPartial(e)) || [];
    message.corrections = object.corrections?.map(e => CorrectionRecord.fromPartial(e)) || [];
    return message;
  }
};
function createBaseParams(): Params {
  return {
    observationIntervalBlocks: BigInt(0),
    weightKnowledgeQuality: BigInt(0),
    weightEconomicStability: BigInt(0),
    weightGovernanceParticipation: BigInt(0),
    weightNetworkSecurity: BigInt(0),
    weightStakingRatio: BigInt(0),
    criticalThreshold: BigInt(0),
    degradedThreshold: BigInt(0),
    healthyThreshold: BigInt(0),
    enabled: false,
    maxAutoApplyMagnitudeBps: BigInt(0),
    correctionConfidenceWindowSize: BigInt(0),
    correctionConfidenceMinSamples: BigInt(0),
    minConfidenceForAutoApply: BigInt(0),
    correctionBoundsMinMultiplierBps: BigInt(0),
    correctionBoundsMaxMultiplierBps: BigInt(0),
    advisoryMagnitudeBps: BigInt(0)
  };
}
/**
 * Params defines the alignment module parameters.
 * @name Params
 * @package zerone.alignment.v1
 * @see proto type: zerone.alignment.v1.Params
 */
export const Params = {
  typeUrl: "/zerone.alignment.v1.Params",
  encode(message: Params, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.observationIntervalBlocks !== BigInt(0)) {
      writer.uint32(8).uint64(message.observationIntervalBlocks);
    }
    if (message.weightKnowledgeQuality !== BigInt(0)) {
      writer.uint32(16).uint64(message.weightKnowledgeQuality);
    }
    if (message.weightEconomicStability !== BigInt(0)) {
      writer.uint32(24).uint64(message.weightEconomicStability);
    }
    if (message.weightGovernanceParticipation !== BigInt(0)) {
      writer.uint32(32).uint64(message.weightGovernanceParticipation);
    }
    if (message.weightNetworkSecurity !== BigInt(0)) {
      writer.uint32(40).uint64(message.weightNetworkSecurity);
    }
    if (message.weightStakingRatio !== BigInt(0)) {
      writer.uint32(48).uint64(message.weightStakingRatio);
    }
    if (message.criticalThreshold !== BigInt(0)) {
      writer.uint32(56).uint64(message.criticalThreshold);
    }
    if (message.degradedThreshold !== BigInt(0)) {
      writer.uint32(64).uint64(message.degradedThreshold);
    }
    if (message.healthyThreshold !== BigInt(0)) {
      writer.uint32(72).uint64(message.healthyThreshold);
    }
    if (message.enabled === true) {
      writer.uint32(80).bool(message.enabled);
    }
    if (message.maxAutoApplyMagnitudeBps !== BigInt(0)) {
      writer.uint32(88).uint64(message.maxAutoApplyMagnitudeBps);
    }
    if (message.correctionConfidenceWindowSize !== BigInt(0)) {
      writer.uint32(96).uint64(message.correctionConfidenceWindowSize);
    }
    if (message.correctionConfidenceMinSamples !== BigInt(0)) {
      writer.uint32(104).uint64(message.correctionConfidenceMinSamples);
    }
    if (message.minConfidenceForAutoApply !== BigInt(0)) {
      writer.uint32(112).uint64(message.minConfidenceForAutoApply);
    }
    if (message.correctionBoundsMinMultiplierBps !== BigInt(0)) {
      writer.uint32(120).uint64(message.correctionBoundsMinMultiplierBps);
    }
    if (message.correctionBoundsMaxMultiplierBps !== BigInt(0)) {
      writer.uint32(128).uint64(message.correctionBoundsMaxMultiplierBps);
    }
    if (message.advisoryMagnitudeBps !== BigInt(0)) {
      writer.uint32(136).uint64(message.advisoryMagnitudeBps);
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
          message.observationIntervalBlocks = reader.uint64();
          break;
        case 2:
          message.weightKnowledgeQuality = reader.uint64();
          break;
        case 3:
          message.weightEconomicStability = reader.uint64();
          break;
        case 4:
          message.weightGovernanceParticipation = reader.uint64();
          break;
        case 5:
          message.weightNetworkSecurity = reader.uint64();
          break;
        case 6:
          message.weightStakingRatio = reader.uint64();
          break;
        case 7:
          message.criticalThreshold = reader.uint64();
          break;
        case 8:
          message.degradedThreshold = reader.uint64();
          break;
        case 9:
          message.healthyThreshold = reader.uint64();
          break;
        case 10:
          message.enabled = reader.bool();
          break;
        case 11:
          message.maxAutoApplyMagnitudeBps = reader.uint64();
          break;
        case 12:
          message.correctionConfidenceWindowSize = reader.uint64();
          break;
        case 13:
          message.correctionConfidenceMinSamples = reader.uint64();
          break;
        case 14:
          message.minConfidenceForAutoApply = reader.uint64();
          break;
        case 15:
          message.correctionBoundsMinMultiplierBps = reader.uint64();
          break;
        case 16:
          message.correctionBoundsMaxMultiplierBps = reader.uint64();
          break;
        case 17:
          message.advisoryMagnitudeBps = reader.uint64();
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
    message.observationIntervalBlocks = object.observationIntervalBlocks !== undefined && object.observationIntervalBlocks !== null ? BigInt(object.observationIntervalBlocks.toString()) : BigInt(0);
    message.weightKnowledgeQuality = object.weightKnowledgeQuality !== undefined && object.weightKnowledgeQuality !== null ? BigInt(object.weightKnowledgeQuality.toString()) : BigInt(0);
    message.weightEconomicStability = object.weightEconomicStability !== undefined && object.weightEconomicStability !== null ? BigInt(object.weightEconomicStability.toString()) : BigInt(0);
    message.weightGovernanceParticipation = object.weightGovernanceParticipation !== undefined && object.weightGovernanceParticipation !== null ? BigInt(object.weightGovernanceParticipation.toString()) : BigInt(0);
    message.weightNetworkSecurity = object.weightNetworkSecurity !== undefined && object.weightNetworkSecurity !== null ? BigInt(object.weightNetworkSecurity.toString()) : BigInt(0);
    message.weightStakingRatio = object.weightStakingRatio !== undefined && object.weightStakingRatio !== null ? BigInt(object.weightStakingRatio.toString()) : BigInt(0);
    message.criticalThreshold = object.criticalThreshold !== undefined && object.criticalThreshold !== null ? BigInt(object.criticalThreshold.toString()) : BigInt(0);
    message.degradedThreshold = object.degradedThreshold !== undefined && object.degradedThreshold !== null ? BigInt(object.degradedThreshold.toString()) : BigInt(0);
    message.healthyThreshold = object.healthyThreshold !== undefined && object.healthyThreshold !== null ? BigInt(object.healthyThreshold.toString()) : BigInt(0);
    message.enabled = object.enabled ?? false;
    message.maxAutoApplyMagnitudeBps = object.maxAutoApplyMagnitudeBps !== undefined && object.maxAutoApplyMagnitudeBps !== null ? BigInt(object.maxAutoApplyMagnitudeBps.toString()) : BigInt(0);
    message.correctionConfidenceWindowSize = object.correctionConfidenceWindowSize !== undefined && object.correctionConfidenceWindowSize !== null ? BigInt(object.correctionConfidenceWindowSize.toString()) : BigInt(0);
    message.correctionConfidenceMinSamples = object.correctionConfidenceMinSamples !== undefined && object.correctionConfidenceMinSamples !== null ? BigInt(object.correctionConfidenceMinSamples.toString()) : BigInt(0);
    message.minConfidenceForAutoApply = object.minConfidenceForAutoApply !== undefined && object.minConfidenceForAutoApply !== null ? BigInt(object.minConfidenceForAutoApply.toString()) : BigInt(0);
    message.correctionBoundsMinMultiplierBps = object.correctionBoundsMinMultiplierBps !== undefined && object.correctionBoundsMinMultiplierBps !== null ? BigInt(object.correctionBoundsMinMultiplierBps.toString()) : BigInt(0);
    message.correctionBoundsMaxMultiplierBps = object.correctionBoundsMaxMultiplierBps !== undefined && object.correctionBoundsMaxMultiplierBps !== null ? BigInt(object.correctionBoundsMaxMultiplierBps.toString()) : BigInt(0);
    message.advisoryMagnitudeBps = object.advisoryMagnitudeBps !== undefined && object.advisoryMagnitudeBps !== null ? BigInt(object.advisoryMagnitudeBps.toString()) : BigInt(0);
    return message;
  }
};
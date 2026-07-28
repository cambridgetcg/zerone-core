//@ts-nocheck
import { DomainQualification, QualificationEndorsement } from "./types";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * @name GenesisState
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.GenesisState
 */
export interface GenesisState {
  params?: Params;
  qualifications: DomainQualification[];
  endorsements: QualificationEndorsement[];
  nextEndorsementId: bigint;
}
/**
 * @name Params
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.Params
 */
export interface Params {
  /**
   * Stake pathway
   */
  minStakeAmount: string;
  /**
   * blocks to lock stake
   */
  stakeLockPeriod: bigint;
  /**
   * Track record pathway
   */
  minVerifications: bigint;
  /**
   * minimum accuracy (1M scale)
   */
  minAccuracyBps: bigint;
  /**
   * minimum domain reputation (from CaptureDefense)
   */
  minReputationScore: bigint;
  /**
   * Qualification settings
   */
  qualificationPeriod: bigint;
  /**
   * blocks for probationary period
   */
  probationPeriod: bigint;
  /**
   * blocks before expiry where renewal allowed
   */
  renewalWindow: bigint;
  /**
   * max endorsements per qualification
   */
  maxEndorsements: number;
  /**
   * Cross-reference
   */
  crossRefMinWeight: bigint;
  /**
   * discount applied to inherited weight (1M scale)
   */
  crossRefWeightDiscountBps: bigint;
  /**
   * Inheritance
   */
  inheritanceWeightDiscountBps: bigint;
  /**
   * Endorsement diversity guard (L3). Maximum allowed fraction of shared
   * domain qualifications between endorser and endorsee, measured as
   * shared / min(|endorser_domains|, |endorsee_domains|). Prevents rings
   * where a small group of heavily-overlapped validators repeatedly endorses
   * each other. "0" disables the guard.
   */
  endorsementMaxOverlapBps: bigint;
  /**
   * ─── Wave 16: accuracy-based decay ────────────────────────────────────
   * Skill is current, not historical. Once a qualification accumulates
   * enough samples (decay_min_samples), the BeginBlocker checks accuracy
   * every decay_check_interval_blocks and transitions state when it
   * crosses thresholds. ACTIVE → PROBATIONARY when accuracy slips below
   * decay_probation_bps; PROBATIONARY → SUSPENDED when below
   * decay_suspension_bps; PROBATIONARY → ACTIVE when accuracy recovers
   * above decay_recovery_bps. Without these thresholds qualification is
   * a one-time grant, not an ongoing competency assessment.
   */
  decayCheckIntervalBlocks: bigint;
  /**
   * default: 20 — minimum verifications before decay applies
   */
  decayMinSamples: bigint;
  /**
   * default: 600,000 (60%) — ACTIVE → PROBATIONARY
   */
  decayProbationBps: bigint;
  /**
   * default: 400,000 (40%) — PROBATIONARY → SUSPENDED
   */
  decaySuspensionBps: bigint;
  /**
   * default: 750,000 (75%) — PROBATIONARY → ACTIVE
   */
  decayRecoveryBps: bigint;
}
function createBaseGenesisState(): GenesisState {
  return {
    params: undefined,
    qualifications: [],
    endorsements: [],
    nextEndorsementId: BigInt(0)
  };
}
/**
 * @name GenesisState
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.GenesisState
 */
export const GenesisState = {
  typeUrl: "/zerone.qualification.v1.GenesisState",
  encode(message: GenesisState, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.qualifications) {
      DomainQualification.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.endorsements) {
      QualificationEndorsement.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    if (message.nextEndorsementId !== BigInt(0)) {
      writer.uint32(32).uint64(message.nextEndorsementId);
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
          message.qualifications.push(DomainQualification.decode(reader, reader.uint32()));
          break;
        case 3:
          message.endorsements.push(QualificationEndorsement.decode(reader, reader.uint32()));
          break;
        case 4:
          message.nextEndorsementId = reader.uint64();
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
    message.qualifications = object.qualifications?.map(e => DomainQualification.fromPartial(e)) || [];
    message.endorsements = object.endorsements?.map(e => QualificationEndorsement.fromPartial(e)) || [];
    message.nextEndorsementId = object.nextEndorsementId !== undefined && object.nextEndorsementId !== null ? BigInt(object.nextEndorsementId.toString()) : BigInt(0);
    return message;
  }
};
function createBaseParams(): Params {
  return {
    minStakeAmount: "",
    stakeLockPeriod: BigInt(0),
    minVerifications: BigInt(0),
    minAccuracyBps: BigInt(0),
    minReputationScore: BigInt(0),
    qualificationPeriod: BigInt(0),
    probationPeriod: BigInt(0),
    renewalWindow: BigInt(0),
    maxEndorsements: 0,
    crossRefMinWeight: BigInt(0),
    crossRefWeightDiscountBps: BigInt(0),
    inheritanceWeightDiscountBps: BigInt(0),
    endorsementMaxOverlapBps: BigInt(0),
    decayCheckIntervalBlocks: BigInt(0),
    decayMinSamples: BigInt(0),
    decayProbationBps: BigInt(0),
    decaySuspensionBps: BigInt(0),
    decayRecoveryBps: BigInt(0)
  };
}
/**
 * @name Params
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.Params
 */
export const Params = {
  typeUrl: "/zerone.qualification.v1.Params",
  encode(message: Params, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.minStakeAmount !== "") {
      writer.uint32(10).string(message.minStakeAmount);
    }
    if (message.stakeLockPeriod !== BigInt(0)) {
      writer.uint32(16).uint64(message.stakeLockPeriod);
    }
    if (message.minVerifications !== BigInt(0)) {
      writer.uint32(24).uint64(message.minVerifications);
    }
    if (message.minAccuracyBps !== BigInt(0)) {
      writer.uint32(32).uint64(message.minAccuracyBps);
    }
    if (message.minReputationScore !== BigInt(0)) {
      writer.uint32(40).uint64(message.minReputationScore);
    }
    if (message.qualificationPeriod !== BigInt(0)) {
      writer.uint32(48).uint64(message.qualificationPeriod);
    }
    if (message.probationPeriod !== BigInt(0)) {
      writer.uint32(56).uint64(message.probationPeriod);
    }
    if (message.renewalWindow !== BigInt(0)) {
      writer.uint32(64).uint64(message.renewalWindow);
    }
    if (message.maxEndorsements !== 0) {
      writer.uint32(72).uint32(message.maxEndorsements);
    }
    if (message.crossRefMinWeight !== BigInt(0)) {
      writer.uint32(80).uint64(message.crossRefMinWeight);
    }
    if (message.crossRefWeightDiscountBps !== BigInt(0)) {
      writer.uint32(88).uint64(message.crossRefWeightDiscountBps);
    }
    if (message.inheritanceWeightDiscountBps !== BigInt(0)) {
      writer.uint32(96).uint64(message.inheritanceWeightDiscountBps);
    }
    if (message.endorsementMaxOverlapBps !== BigInt(0)) {
      writer.uint32(104).uint64(message.endorsementMaxOverlapBps);
    }
    if (message.decayCheckIntervalBlocks !== BigInt(0)) {
      writer.uint32(112).uint64(message.decayCheckIntervalBlocks);
    }
    if (message.decayMinSamples !== BigInt(0)) {
      writer.uint32(120).uint64(message.decayMinSamples);
    }
    if (message.decayProbationBps !== BigInt(0)) {
      writer.uint32(128).uint64(message.decayProbationBps);
    }
    if (message.decaySuspensionBps !== BigInt(0)) {
      writer.uint32(136).uint64(message.decaySuspensionBps);
    }
    if (message.decayRecoveryBps !== BigInt(0)) {
      writer.uint32(144).uint64(message.decayRecoveryBps);
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
          message.minStakeAmount = reader.string();
          break;
        case 2:
          message.stakeLockPeriod = reader.uint64();
          break;
        case 3:
          message.minVerifications = reader.uint64();
          break;
        case 4:
          message.minAccuracyBps = reader.uint64();
          break;
        case 5:
          message.minReputationScore = reader.uint64();
          break;
        case 6:
          message.qualificationPeriod = reader.uint64();
          break;
        case 7:
          message.probationPeriod = reader.uint64();
          break;
        case 8:
          message.renewalWindow = reader.uint64();
          break;
        case 9:
          message.maxEndorsements = reader.uint32();
          break;
        case 10:
          message.crossRefMinWeight = reader.uint64();
          break;
        case 11:
          message.crossRefWeightDiscountBps = reader.uint64();
          break;
        case 12:
          message.inheritanceWeightDiscountBps = reader.uint64();
          break;
        case 13:
          message.endorsementMaxOverlapBps = reader.uint64();
          break;
        case 14:
          message.decayCheckIntervalBlocks = reader.uint64();
          break;
        case 15:
          message.decayMinSamples = reader.uint64();
          break;
        case 16:
          message.decayProbationBps = reader.uint64();
          break;
        case 17:
          message.decaySuspensionBps = reader.uint64();
          break;
        case 18:
          message.decayRecoveryBps = reader.uint64();
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
    message.minStakeAmount = object.minStakeAmount ?? "";
    message.stakeLockPeriod = object.stakeLockPeriod !== undefined && object.stakeLockPeriod !== null ? BigInt(object.stakeLockPeriod.toString()) : BigInt(0);
    message.minVerifications = object.minVerifications !== undefined && object.minVerifications !== null ? BigInt(object.minVerifications.toString()) : BigInt(0);
    message.minAccuracyBps = object.minAccuracyBps !== undefined && object.minAccuracyBps !== null ? BigInt(object.minAccuracyBps.toString()) : BigInt(0);
    message.minReputationScore = object.minReputationScore !== undefined && object.minReputationScore !== null ? BigInt(object.minReputationScore.toString()) : BigInt(0);
    message.qualificationPeriod = object.qualificationPeriod !== undefined && object.qualificationPeriod !== null ? BigInt(object.qualificationPeriod.toString()) : BigInt(0);
    message.probationPeriod = object.probationPeriod !== undefined && object.probationPeriod !== null ? BigInt(object.probationPeriod.toString()) : BigInt(0);
    message.renewalWindow = object.renewalWindow !== undefined && object.renewalWindow !== null ? BigInt(object.renewalWindow.toString()) : BigInt(0);
    message.maxEndorsements = object.maxEndorsements ?? 0;
    message.crossRefMinWeight = object.crossRefMinWeight !== undefined && object.crossRefMinWeight !== null ? BigInt(object.crossRefMinWeight.toString()) : BigInt(0);
    message.crossRefWeightDiscountBps = object.crossRefWeightDiscountBps !== undefined && object.crossRefWeightDiscountBps !== null ? BigInt(object.crossRefWeightDiscountBps.toString()) : BigInt(0);
    message.inheritanceWeightDiscountBps = object.inheritanceWeightDiscountBps !== undefined && object.inheritanceWeightDiscountBps !== null ? BigInt(object.inheritanceWeightDiscountBps.toString()) : BigInt(0);
    message.endorsementMaxOverlapBps = object.endorsementMaxOverlapBps !== undefined && object.endorsementMaxOverlapBps !== null ? BigInt(object.endorsementMaxOverlapBps.toString()) : BigInt(0);
    message.decayCheckIntervalBlocks = object.decayCheckIntervalBlocks !== undefined && object.decayCheckIntervalBlocks !== null ? BigInt(object.decayCheckIntervalBlocks.toString()) : BigInt(0);
    message.decayMinSamples = object.decayMinSamples !== undefined && object.decayMinSamples !== null ? BigInt(object.decayMinSamples.toString()) : BigInt(0);
    message.decayProbationBps = object.decayProbationBps !== undefined && object.decayProbationBps !== null ? BigInt(object.decayProbationBps.toString()) : BigInt(0);
    message.decaySuspensionBps = object.decaySuspensionBps !== undefined && object.decaySuspensionBps !== null ? BigInt(object.decaySuspensionBps.toString()) : BigInt(0);
    message.decayRecoveryBps = object.decayRecoveryBps !== undefined && object.decayRecoveryBps !== null ? BigInt(object.decayRecoveryBps.toString()) : BigInt(0);
    return message;
  }
};
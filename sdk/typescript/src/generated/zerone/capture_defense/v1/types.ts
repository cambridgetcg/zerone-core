//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * IsolatedReputation tracks a validator's reputation at a single scope level.
 * @name IsolatedReputation
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.IsolatedReputation
 */
export interface IsolatedReputation {
  validator: string;
  /**
   * "global", stratum name, or domain ID
   */
  scope: string;
  /**
   * 0–1_000_000 BPS
   */
  score: bigint;
  verifications: bigint;
  lastUpdatedBlock: bigint;
}
/**
 * GlobalReputation is a validator's chain-wide reputation.
 * @name GlobalReputation
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.GlobalReputation
 */
export interface GlobalReputation {
  validator: string;
  score: bigint;
  totalVerifications: bigint;
  lastUpdatedBlock: bigint;
}
/**
 * StratumReputation is a validator's reputation within a knowledge stratum.
 * @name StratumReputation
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.StratumReputation
 */
export interface StratumReputation {
  validator: string;
  stratum: string;
  score: bigint;
  verifications: bigint;
  lastUpdatedBlock: bigint;
}
/**
 * DomainReputation is a validator's reputation for a specific domain.
 * @name DomainReputation
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.DomainReputation
 */
export interface DomainReputation {
  validator: string;
  domain: string;
  score: bigint;
  verifications: bigint;
  lastUpdatedBlock: bigint;
}
/**
 * CaptureMetrics holds computed capture risk analysis for a domain.
 * @name CaptureMetrics
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.CaptureMetrics
 */
export interface CaptureMetrics {
  domain: string;
  /**
   * 0–1_000_000 BPS
   */
  herfindahlIndex: bigint;
  /**
   * 0–1_000_000 BPS
   */
  timingCorrelation: bigint;
  /**
   * 0–1_000_000 BPS
   */
  verdictCorrelation: bigint;
  /**
   * 0–1_000_000 BPS
   */
  top3Share: bigint;
  /**
   * composite 0–1_000_000
   */
  riskScore: bigint;
  totalParticipations: bigint;
  analyzedAtBlock: bigint;
  flagged: boolean;
}
/**
 * VerificationHistoryEntry records one verification round for detection.
 * @name VerificationHistoryEntry
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.VerificationHistoryEntry
 */
export interface VerificationHistoryEntry {
  domain: string;
  roundId: string;
  validators: string[];
  /**
   * true = approved, false = rejected
   */
  verdicts: boolean[];
  submitBlocks: bigint[];
  blockHeight: bigint;
}
/**
 * CrossStratumRequirement defines cross-stratum validation rules.
 * @name CrossStratumRequirement
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.CrossStratumRequirement
 */
export interface CrossStratumRequirement {
  targetStratum: string;
  requiredStrata: string[];
  minValidatorsPerStratum: bigint;
}
function createBaseIsolatedReputation(): IsolatedReputation {
  return {
    validator: "",
    scope: "",
    score: BigInt(0),
    verifications: BigInt(0),
    lastUpdatedBlock: BigInt(0)
  };
}
/**
 * IsolatedReputation tracks a validator's reputation at a single scope level.
 * @name IsolatedReputation
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.IsolatedReputation
 */
export const IsolatedReputation = {
  typeUrl: "/zerone.capture_defense.v1.IsolatedReputation",
  encode(message: IsolatedReputation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.validator !== "") {
      writer.uint32(10).string(message.validator);
    }
    if (message.scope !== "") {
      writer.uint32(18).string(message.scope);
    }
    if (message.score !== BigInt(0)) {
      writer.uint32(24).uint64(message.score);
    }
    if (message.verifications !== BigInt(0)) {
      writer.uint32(32).uint64(message.verifications);
    }
    if (message.lastUpdatedBlock !== BigInt(0)) {
      writer.uint32(40).uint64(message.lastUpdatedBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): IsolatedReputation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseIsolatedReputation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator = reader.string();
          break;
        case 2:
          message.scope = reader.string();
          break;
        case 3:
          message.score = reader.uint64();
          break;
        case 4:
          message.verifications = reader.uint64();
          break;
        case 5:
          message.lastUpdatedBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<IsolatedReputation>): IsolatedReputation {
    const message = createBaseIsolatedReputation();
    message.validator = object.validator ?? "";
    message.scope = object.scope ?? "";
    message.score = object.score !== undefined && object.score !== null ? BigInt(object.score.toString()) : BigInt(0);
    message.verifications = object.verifications !== undefined && object.verifications !== null ? BigInt(object.verifications.toString()) : BigInt(0);
    message.lastUpdatedBlock = object.lastUpdatedBlock !== undefined && object.lastUpdatedBlock !== null ? BigInt(object.lastUpdatedBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseGlobalReputation(): GlobalReputation {
  return {
    validator: "",
    score: BigInt(0),
    totalVerifications: BigInt(0),
    lastUpdatedBlock: BigInt(0)
  };
}
/**
 * GlobalReputation is a validator's chain-wide reputation.
 * @name GlobalReputation
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.GlobalReputation
 */
export const GlobalReputation = {
  typeUrl: "/zerone.capture_defense.v1.GlobalReputation",
  encode(message: GlobalReputation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.validator !== "") {
      writer.uint32(10).string(message.validator);
    }
    if (message.score !== BigInt(0)) {
      writer.uint32(16).uint64(message.score);
    }
    if (message.totalVerifications !== BigInt(0)) {
      writer.uint32(24).uint64(message.totalVerifications);
    }
    if (message.lastUpdatedBlock !== BigInt(0)) {
      writer.uint32(32).uint64(message.lastUpdatedBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): GlobalReputation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGlobalReputation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator = reader.string();
          break;
        case 2:
          message.score = reader.uint64();
          break;
        case 3:
          message.totalVerifications = reader.uint64();
          break;
        case 4:
          message.lastUpdatedBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<GlobalReputation>): GlobalReputation {
    const message = createBaseGlobalReputation();
    message.validator = object.validator ?? "";
    message.score = object.score !== undefined && object.score !== null ? BigInt(object.score.toString()) : BigInt(0);
    message.totalVerifications = object.totalVerifications !== undefined && object.totalVerifications !== null ? BigInt(object.totalVerifications.toString()) : BigInt(0);
    message.lastUpdatedBlock = object.lastUpdatedBlock !== undefined && object.lastUpdatedBlock !== null ? BigInt(object.lastUpdatedBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseStratumReputation(): StratumReputation {
  return {
    validator: "",
    stratum: "",
    score: BigInt(0),
    verifications: BigInt(0),
    lastUpdatedBlock: BigInt(0)
  };
}
/**
 * StratumReputation is a validator's reputation within a knowledge stratum.
 * @name StratumReputation
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.StratumReputation
 */
export const StratumReputation = {
  typeUrl: "/zerone.capture_defense.v1.StratumReputation",
  encode(message: StratumReputation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.validator !== "") {
      writer.uint32(10).string(message.validator);
    }
    if (message.stratum !== "") {
      writer.uint32(18).string(message.stratum);
    }
    if (message.score !== BigInt(0)) {
      writer.uint32(24).uint64(message.score);
    }
    if (message.verifications !== BigInt(0)) {
      writer.uint32(32).uint64(message.verifications);
    }
    if (message.lastUpdatedBlock !== BigInt(0)) {
      writer.uint32(40).uint64(message.lastUpdatedBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): StratumReputation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseStratumReputation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator = reader.string();
          break;
        case 2:
          message.stratum = reader.string();
          break;
        case 3:
          message.score = reader.uint64();
          break;
        case 4:
          message.verifications = reader.uint64();
          break;
        case 5:
          message.lastUpdatedBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<StratumReputation>): StratumReputation {
    const message = createBaseStratumReputation();
    message.validator = object.validator ?? "";
    message.stratum = object.stratum ?? "";
    message.score = object.score !== undefined && object.score !== null ? BigInt(object.score.toString()) : BigInt(0);
    message.verifications = object.verifications !== undefined && object.verifications !== null ? BigInt(object.verifications.toString()) : BigInt(0);
    message.lastUpdatedBlock = object.lastUpdatedBlock !== undefined && object.lastUpdatedBlock !== null ? BigInt(object.lastUpdatedBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseDomainReputation(): DomainReputation {
  return {
    validator: "",
    domain: "",
    score: BigInt(0),
    verifications: BigInt(0),
    lastUpdatedBlock: BigInt(0)
  };
}
/**
 * DomainReputation is a validator's reputation for a specific domain.
 * @name DomainReputation
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.DomainReputation
 */
export const DomainReputation = {
  typeUrl: "/zerone.capture_defense.v1.DomainReputation",
  encode(message: DomainReputation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.validator !== "") {
      writer.uint32(10).string(message.validator);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    if (message.score !== BigInt(0)) {
      writer.uint32(24).uint64(message.score);
    }
    if (message.verifications !== BigInt(0)) {
      writer.uint32(32).uint64(message.verifications);
    }
    if (message.lastUpdatedBlock !== BigInt(0)) {
      writer.uint32(40).uint64(message.lastUpdatedBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): DomainReputation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDomainReputation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator = reader.string();
          break;
        case 2:
          message.domain = reader.string();
          break;
        case 3:
          message.score = reader.uint64();
          break;
        case 4:
          message.verifications = reader.uint64();
          break;
        case 5:
          message.lastUpdatedBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<DomainReputation>): DomainReputation {
    const message = createBaseDomainReputation();
    message.validator = object.validator ?? "";
    message.domain = object.domain ?? "";
    message.score = object.score !== undefined && object.score !== null ? BigInt(object.score.toString()) : BigInt(0);
    message.verifications = object.verifications !== undefined && object.verifications !== null ? BigInt(object.verifications.toString()) : BigInt(0);
    message.lastUpdatedBlock = object.lastUpdatedBlock !== undefined && object.lastUpdatedBlock !== null ? BigInt(object.lastUpdatedBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseCaptureMetrics(): CaptureMetrics {
  return {
    domain: "",
    herfindahlIndex: BigInt(0),
    timingCorrelation: BigInt(0),
    verdictCorrelation: BigInt(0),
    top3Share: BigInt(0),
    riskScore: BigInt(0),
    totalParticipations: BigInt(0),
    analyzedAtBlock: BigInt(0),
    flagged: false
  };
}
/**
 * CaptureMetrics holds computed capture risk analysis for a domain.
 * @name CaptureMetrics
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.CaptureMetrics
 */
export const CaptureMetrics = {
  typeUrl: "/zerone.capture_defense.v1.CaptureMetrics",
  encode(message: CaptureMetrics, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.domain !== "") {
      writer.uint32(10).string(message.domain);
    }
    if (message.herfindahlIndex !== BigInt(0)) {
      writer.uint32(16).uint64(message.herfindahlIndex);
    }
    if (message.timingCorrelation !== BigInt(0)) {
      writer.uint32(24).uint64(message.timingCorrelation);
    }
    if (message.verdictCorrelation !== BigInt(0)) {
      writer.uint32(32).uint64(message.verdictCorrelation);
    }
    if (message.top3Share !== BigInt(0)) {
      writer.uint32(40).uint64(message.top3Share);
    }
    if (message.riskScore !== BigInt(0)) {
      writer.uint32(48).uint64(message.riskScore);
    }
    if (message.totalParticipations !== BigInt(0)) {
      writer.uint32(56).uint64(message.totalParticipations);
    }
    if (message.analyzedAtBlock !== BigInt(0)) {
      writer.uint32(64).uint64(message.analyzedAtBlock);
    }
    if (message.flagged === true) {
      writer.uint32(72).bool(message.flagged);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): CaptureMetrics {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCaptureMetrics();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.domain = reader.string();
          break;
        case 2:
          message.herfindahlIndex = reader.uint64();
          break;
        case 3:
          message.timingCorrelation = reader.uint64();
          break;
        case 4:
          message.verdictCorrelation = reader.uint64();
          break;
        case 5:
          message.top3Share = reader.uint64();
          break;
        case 6:
          message.riskScore = reader.uint64();
          break;
        case 7:
          message.totalParticipations = reader.uint64();
          break;
        case 8:
          message.analyzedAtBlock = reader.uint64();
          break;
        case 9:
          message.flagged = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<CaptureMetrics>): CaptureMetrics {
    const message = createBaseCaptureMetrics();
    message.domain = object.domain ?? "";
    message.herfindahlIndex = object.herfindahlIndex !== undefined && object.herfindahlIndex !== null ? BigInt(object.herfindahlIndex.toString()) : BigInt(0);
    message.timingCorrelation = object.timingCorrelation !== undefined && object.timingCorrelation !== null ? BigInt(object.timingCorrelation.toString()) : BigInt(0);
    message.verdictCorrelation = object.verdictCorrelation !== undefined && object.verdictCorrelation !== null ? BigInt(object.verdictCorrelation.toString()) : BigInt(0);
    message.top3Share = object.top3Share !== undefined && object.top3Share !== null ? BigInt(object.top3Share.toString()) : BigInt(0);
    message.riskScore = object.riskScore !== undefined && object.riskScore !== null ? BigInt(object.riskScore.toString()) : BigInt(0);
    message.totalParticipations = object.totalParticipations !== undefined && object.totalParticipations !== null ? BigInt(object.totalParticipations.toString()) : BigInt(0);
    message.analyzedAtBlock = object.analyzedAtBlock !== undefined && object.analyzedAtBlock !== null ? BigInt(object.analyzedAtBlock.toString()) : BigInt(0);
    message.flagged = object.flagged ?? false;
    return message;
  }
};
function createBaseVerificationHistoryEntry(): VerificationHistoryEntry {
  return {
    domain: "",
    roundId: "",
    validators: [],
    verdicts: [],
    submitBlocks: [],
    blockHeight: BigInt(0)
  };
}
/**
 * VerificationHistoryEntry records one verification round for detection.
 * @name VerificationHistoryEntry
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.VerificationHistoryEntry
 */
export const VerificationHistoryEntry = {
  typeUrl: "/zerone.capture_defense.v1.VerificationHistoryEntry",
  encode(message: VerificationHistoryEntry, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.domain !== "") {
      writer.uint32(10).string(message.domain);
    }
    if (message.roundId !== "") {
      writer.uint32(18).string(message.roundId);
    }
    for (const v of message.validators) {
      writer.uint32(26).string(v!);
    }
    writer.uint32(34).fork();
    for (const v of message.verdicts) {
      writer.bool(v);
    }
    writer.ldelim();
    writer.uint32(42).fork();
    for (const v of message.submitBlocks) {
      writer.uint64(v);
    }
    writer.ldelim();
    if (message.blockHeight !== BigInt(0)) {
      writer.uint32(48).uint64(message.blockHeight);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): VerificationHistoryEntry {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseVerificationHistoryEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.domain = reader.string();
          break;
        case 2:
          message.roundId = reader.string();
          break;
        case 3:
          message.validators.push(reader.string());
          break;
        case 4:
          if ((tag & 7) === 2) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.verdicts.push(reader.bool());
            }
          } else {
            message.verdicts.push(reader.bool());
          }
          break;
        case 5:
          if ((tag & 7) === 2) {
            const end2 = reader.uint32() + reader.pos;
            while (reader.pos < end2) {
              message.submitBlocks.push(reader.uint64());
            }
          } else {
            message.submitBlocks.push(reader.uint64());
          }
          break;
        case 6:
          message.blockHeight = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<VerificationHistoryEntry>): VerificationHistoryEntry {
    const message = createBaseVerificationHistoryEntry();
    message.domain = object.domain ?? "";
    message.roundId = object.roundId ?? "";
    message.validators = object.validators?.map(e => e) || [];
    message.verdicts = object.verdicts?.map(e => e) || [];
    message.submitBlocks = object.submitBlocks?.map(e => BigInt(e.toString())) || [];
    message.blockHeight = object.blockHeight !== undefined && object.blockHeight !== null ? BigInt(object.blockHeight.toString()) : BigInt(0);
    return message;
  }
};
function createBaseCrossStratumRequirement(): CrossStratumRequirement {
  return {
    targetStratum: "",
    requiredStrata: [],
    minValidatorsPerStratum: BigInt(0)
  };
}
/**
 * CrossStratumRequirement defines cross-stratum validation rules.
 * @name CrossStratumRequirement
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.CrossStratumRequirement
 */
export const CrossStratumRequirement = {
  typeUrl: "/zerone.capture_defense.v1.CrossStratumRequirement",
  encode(message: CrossStratumRequirement, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.targetStratum !== "") {
      writer.uint32(10).string(message.targetStratum);
    }
    for (const v of message.requiredStrata) {
      writer.uint32(18).string(v!);
    }
    if (message.minValidatorsPerStratum !== BigInt(0)) {
      writer.uint32(24).uint64(message.minValidatorsPerStratum);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): CrossStratumRequirement {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCrossStratumRequirement();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.targetStratum = reader.string();
          break;
        case 2:
          message.requiredStrata.push(reader.string());
          break;
        case 3:
          message.minValidatorsPerStratum = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<CrossStratumRequirement>): CrossStratumRequirement {
    const message = createBaseCrossStratumRequirement();
    message.targetStratum = object.targetStratum ?? "";
    message.requiredStrata = object.requiredStrata?.map(e => e) || [];
    message.minValidatorsPerStratum = object.minValidatorsPerStratum !== undefined && object.minValidatorsPerStratum !== null ? BigInt(object.minValidatorsPerStratum.toString()) : BigInt(0);
    return message;
  }
};
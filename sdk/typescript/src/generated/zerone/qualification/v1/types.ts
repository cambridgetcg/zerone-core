//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
export enum QualificationPathway {
  QUALIFICATION_PATHWAY_UNSPECIFIED = 0,
  QUALIFICATION_PATHWAY_STAKE = 1,
  QUALIFICATION_PATHWAY_TRACK_RECORD = 2,
  QUALIFICATION_PATHWAY_CROSS_REFERENCE = 3,
  QUALIFICATION_PATHWAY_INHERITANCE = 4,
  UNRECOGNIZED = -1,
}
export function qualificationPathwayFromJSON(object: any): QualificationPathway {
  switch (object) {
    case 0:
    case "QUALIFICATION_PATHWAY_UNSPECIFIED":
      return QualificationPathway.QUALIFICATION_PATHWAY_UNSPECIFIED;
    case 1:
    case "QUALIFICATION_PATHWAY_STAKE":
      return QualificationPathway.QUALIFICATION_PATHWAY_STAKE;
    case 2:
    case "QUALIFICATION_PATHWAY_TRACK_RECORD":
      return QualificationPathway.QUALIFICATION_PATHWAY_TRACK_RECORD;
    case 3:
    case "QUALIFICATION_PATHWAY_CROSS_REFERENCE":
      return QualificationPathway.QUALIFICATION_PATHWAY_CROSS_REFERENCE;
    case 4:
    case "QUALIFICATION_PATHWAY_INHERITANCE":
      return QualificationPathway.QUALIFICATION_PATHWAY_INHERITANCE;
    case -1:
    case "UNRECOGNIZED":
    default:
      return QualificationPathway.UNRECOGNIZED;
  }
}
export function qualificationPathwayToJSON(object: QualificationPathway): string {
  switch (object) {
    case QualificationPathway.QUALIFICATION_PATHWAY_UNSPECIFIED:
      return "QUALIFICATION_PATHWAY_UNSPECIFIED";
    case QualificationPathway.QUALIFICATION_PATHWAY_STAKE:
      return "QUALIFICATION_PATHWAY_STAKE";
    case QualificationPathway.QUALIFICATION_PATHWAY_TRACK_RECORD:
      return "QUALIFICATION_PATHWAY_TRACK_RECORD";
    case QualificationPathway.QUALIFICATION_PATHWAY_CROSS_REFERENCE:
      return "QUALIFICATION_PATHWAY_CROSS_REFERENCE";
    case QualificationPathway.QUALIFICATION_PATHWAY_INHERITANCE:
      return "QUALIFICATION_PATHWAY_INHERITANCE";
    case QualificationPathway.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
export enum QualificationStatus {
  QUALIFICATION_STATUS_UNSPECIFIED = 0,
  QUALIFICATION_STATUS_ACTIVE = 1,
  QUALIFICATION_STATUS_PROBATIONARY = 2,
  QUALIFICATION_STATUS_SUSPENDED = 3,
  QUALIFICATION_STATUS_REVOKED = 4,
  QUALIFICATION_STATUS_EXPIRED = 5,
  UNRECOGNIZED = -1,
}
export function qualificationStatusFromJSON(object: any): QualificationStatus {
  switch (object) {
    case 0:
    case "QUALIFICATION_STATUS_UNSPECIFIED":
      return QualificationStatus.QUALIFICATION_STATUS_UNSPECIFIED;
    case 1:
    case "QUALIFICATION_STATUS_ACTIVE":
      return QualificationStatus.QUALIFICATION_STATUS_ACTIVE;
    case 2:
    case "QUALIFICATION_STATUS_PROBATIONARY":
      return QualificationStatus.QUALIFICATION_STATUS_PROBATIONARY;
    case 3:
    case "QUALIFICATION_STATUS_SUSPENDED":
      return QualificationStatus.QUALIFICATION_STATUS_SUSPENDED;
    case 4:
    case "QUALIFICATION_STATUS_REVOKED":
      return QualificationStatus.QUALIFICATION_STATUS_REVOKED;
    case 5:
    case "QUALIFICATION_STATUS_EXPIRED":
      return QualificationStatus.QUALIFICATION_STATUS_EXPIRED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return QualificationStatus.UNRECOGNIZED;
  }
}
export function qualificationStatusToJSON(object: QualificationStatus): string {
  switch (object) {
    case QualificationStatus.QUALIFICATION_STATUS_UNSPECIFIED:
      return "QUALIFICATION_STATUS_UNSPECIFIED";
    case QualificationStatus.QUALIFICATION_STATUS_ACTIVE:
      return "QUALIFICATION_STATUS_ACTIVE";
    case QualificationStatus.QUALIFICATION_STATUS_PROBATIONARY:
      return "QUALIFICATION_STATUS_PROBATIONARY";
    case QualificationStatus.QUALIFICATION_STATUS_SUSPENDED:
      return "QUALIFICATION_STATUS_SUSPENDED";
    case QualificationStatus.QUALIFICATION_STATUS_REVOKED:
      return "QUALIFICATION_STATUS_REVOKED";
    case QualificationStatus.QUALIFICATION_STATUS_EXPIRED:
      return "QUALIFICATION_STATUS_EXPIRED";
    case QualificationStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/**
 * @name QualificationMetrics
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.QualificationMetrics
 */
export interface QualificationMetrics {
  totalVerifications: bigint;
  correctVerifications: bigint;
  /**
   * 1,000,000 = 100%
   */
  accuracyBps: bigint;
  lastVerificationBlock: bigint;
}
/**
 * @name DomainQualification
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.DomainQualification
 */
export interface DomainQualification {
  /**
   * bech32 validator address
   */
  validator: string;
  domain: string;
  pathway: QualificationPathway;
  status: QualificationStatus;
  /**
   * 1-100 qualification weight
   */
  weight: number;
  /**
   * domain stratum level (lower = more foundational)
   */
  stratum: number;
  /**
   * uzrn locked for stake-pathway
   */
  stakedAmount: string;
  /**
   * block height
   */
  grantedAt: bigint;
  /**
   * block height (0 = no expiry)
   */
  expiresAt: bigint;
  /**
   * last renewal block
   */
  renewedAt: bigint;
  metrics?: QualificationMetrics;
  /**
   * for inheritance pathway
   */
  parentDomain: string;
  /**
   * for cross-reference pathway
   */
  crossRefDomain: string;
  endorsementCount: number;
  /**
   * block height, 0 if not on probation
   */
  probationUntil: bigint;
}
/**
 * @name QualificationEndorsement
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.QualificationEndorsement
 */
export interface QualificationEndorsement {
  id: bigint;
  /**
   * validator being endorsed
   */
  qualificationValidator: string;
  qualificationDomain: string;
  /**
   * bech32
   */
  endorser: string;
  reason: string;
  /**
   * endorsement weight (1-100)
   */
  weight: number;
  createdAt: bigint;
  /**
   * 0 = no expiry
   */
  expiresAt: bigint;
}
function createBaseQualificationMetrics(): QualificationMetrics {
  return {
    totalVerifications: BigInt(0),
    correctVerifications: BigInt(0),
    accuracyBps: BigInt(0),
    lastVerificationBlock: BigInt(0)
  };
}
/**
 * @name QualificationMetrics
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.QualificationMetrics
 */
export const QualificationMetrics = {
  typeUrl: "/zerone.qualification.v1.QualificationMetrics",
  encode(message: QualificationMetrics, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.totalVerifications !== BigInt(0)) {
      writer.uint32(8).uint64(message.totalVerifications);
    }
    if (message.correctVerifications !== BigInt(0)) {
      writer.uint32(16).uint64(message.correctVerifications);
    }
    if (message.accuracyBps !== BigInt(0)) {
      writer.uint32(24).uint64(message.accuracyBps);
    }
    if (message.lastVerificationBlock !== BigInt(0)) {
      writer.uint32(32).uint64(message.lastVerificationBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): QualificationMetrics {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQualificationMetrics();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.totalVerifications = reader.uint64();
          break;
        case 2:
          message.correctVerifications = reader.uint64();
          break;
        case 3:
          message.accuracyBps = reader.uint64();
          break;
        case 4:
          message.lastVerificationBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<QualificationMetrics>): QualificationMetrics {
    const message = createBaseQualificationMetrics();
    message.totalVerifications = object.totalVerifications !== undefined && object.totalVerifications !== null ? BigInt(object.totalVerifications.toString()) : BigInt(0);
    message.correctVerifications = object.correctVerifications !== undefined && object.correctVerifications !== null ? BigInt(object.correctVerifications.toString()) : BigInt(0);
    message.accuracyBps = object.accuracyBps !== undefined && object.accuracyBps !== null ? BigInt(object.accuracyBps.toString()) : BigInt(0);
    message.lastVerificationBlock = object.lastVerificationBlock !== undefined && object.lastVerificationBlock !== null ? BigInt(object.lastVerificationBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseDomainQualification(): DomainQualification {
  return {
    validator: "",
    domain: "",
    pathway: 0,
    status: 0,
    weight: 0,
    stratum: 0,
    stakedAmount: "",
    grantedAt: BigInt(0),
    expiresAt: BigInt(0),
    renewedAt: BigInt(0),
    metrics: undefined,
    parentDomain: "",
    crossRefDomain: "",
    endorsementCount: 0,
    probationUntil: BigInt(0)
  };
}
/**
 * @name DomainQualification
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.DomainQualification
 */
export const DomainQualification = {
  typeUrl: "/zerone.qualification.v1.DomainQualification",
  encode(message: DomainQualification, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.validator !== "") {
      writer.uint32(10).string(message.validator);
    }
    if (message.domain !== "") {
      writer.uint32(18).string(message.domain);
    }
    if (message.pathway !== 0) {
      writer.uint32(24).int32(message.pathway);
    }
    if (message.status !== 0) {
      writer.uint32(32).int32(message.status);
    }
    if (message.weight !== 0) {
      writer.uint32(40).uint32(message.weight);
    }
    if (message.stratum !== 0) {
      writer.uint32(48).uint32(message.stratum);
    }
    if (message.stakedAmount !== "") {
      writer.uint32(58).string(message.stakedAmount);
    }
    if (message.grantedAt !== BigInt(0)) {
      writer.uint32(64).uint64(message.grantedAt);
    }
    if (message.expiresAt !== BigInt(0)) {
      writer.uint32(72).uint64(message.expiresAt);
    }
    if (message.renewedAt !== BigInt(0)) {
      writer.uint32(80).uint64(message.renewedAt);
    }
    if (message.metrics !== undefined) {
      QualificationMetrics.encode(message.metrics, writer.uint32(90).fork()).ldelim();
    }
    if (message.parentDomain !== "") {
      writer.uint32(98).string(message.parentDomain);
    }
    if (message.crossRefDomain !== "") {
      writer.uint32(106).string(message.crossRefDomain);
    }
    if (message.endorsementCount !== 0) {
      writer.uint32(112).uint32(message.endorsementCount);
    }
    if (message.probationUntil !== BigInt(0)) {
      writer.uint32(120).uint64(message.probationUntil);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): DomainQualification {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDomainQualification();
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
          message.pathway = reader.int32() as any;
          break;
        case 4:
          message.status = reader.int32() as any;
          break;
        case 5:
          message.weight = reader.uint32();
          break;
        case 6:
          message.stratum = reader.uint32();
          break;
        case 7:
          message.stakedAmount = reader.string();
          break;
        case 8:
          message.grantedAt = reader.uint64();
          break;
        case 9:
          message.expiresAt = reader.uint64();
          break;
        case 10:
          message.renewedAt = reader.uint64();
          break;
        case 11:
          message.metrics = QualificationMetrics.decode(reader, reader.uint32());
          break;
        case 12:
          message.parentDomain = reader.string();
          break;
        case 13:
          message.crossRefDomain = reader.string();
          break;
        case 14:
          message.endorsementCount = reader.uint32();
          break;
        case 15:
          message.probationUntil = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<DomainQualification>): DomainQualification {
    const message = createBaseDomainQualification();
    message.validator = object.validator ?? "";
    message.domain = object.domain ?? "";
    message.pathway = object.pathway ?? 0;
    message.status = object.status ?? 0;
    message.weight = object.weight ?? 0;
    message.stratum = object.stratum ?? 0;
    message.stakedAmount = object.stakedAmount ?? "";
    message.grantedAt = object.grantedAt !== undefined && object.grantedAt !== null ? BigInt(object.grantedAt.toString()) : BigInt(0);
    message.expiresAt = object.expiresAt !== undefined && object.expiresAt !== null ? BigInt(object.expiresAt.toString()) : BigInt(0);
    message.renewedAt = object.renewedAt !== undefined && object.renewedAt !== null ? BigInt(object.renewedAt.toString()) : BigInt(0);
    message.metrics = object.metrics !== undefined && object.metrics !== null ? QualificationMetrics.fromPartial(object.metrics) : undefined;
    message.parentDomain = object.parentDomain ?? "";
    message.crossRefDomain = object.crossRefDomain ?? "";
    message.endorsementCount = object.endorsementCount ?? 0;
    message.probationUntil = object.probationUntil !== undefined && object.probationUntil !== null ? BigInt(object.probationUntil.toString()) : BigInt(0);
    return message;
  }
};
function createBaseQualificationEndorsement(): QualificationEndorsement {
  return {
    id: BigInt(0),
    qualificationValidator: "",
    qualificationDomain: "",
    endorser: "",
    reason: "",
    weight: 0,
    createdAt: BigInt(0),
    expiresAt: BigInt(0)
  };
}
/**
 * @name QualificationEndorsement
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.QualificationEndorsement
 */
export const QualificationEndorsement = {
  typeUrl: "/zerone.qualification.v1.QualificationEndorsement",
  encode(message: QualificationEndorsement, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== BigInt(0)) {
      writer.uint32(8).uint64(message.id);
    }
    if (message.qualificationValidator !== "") {
      writer.uint32(18).string(message.qualificationValidator);
    }
    if (message.qualificationDomain !== "") {
      writer.uint32(26).string(message.qualificationDomain);
    }
    if (message.endorser !== "") {
      writer.uint32(34).string(message.endorser);
    }
    if (message.reason !== "") {
      writer.uint32(42).string(message.reason);
    }
    if (message.weight !== 0) {
      writer.uint32(48).uint32(message.weight);
    }
    if (message.createdAt !== BigInt(0)) {
      writer.uint32(56).uint64(message.createdAt);
    }
    if (message.expiresAt !== BigInt(0)) {
      writer.uint32(64).uint64(message.expiresAt);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): QualificationEndorsement {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseQualificationEndorsement();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.uint64();
          break;
        case 2:
          message.qualificationValidator = reader.string();
          break;
        case 3:
          message.qualificationDomain = reader.string();
          break;
        case 4:
          message.endorser = reader.string();
          break;
        case 5:
          message.reason = reader.string();
          break;
        case 6:
          message.weight = reader.uint32();
          break;
        case 7:
          message.createdAt = reader.uint64();
          break;
        case 8:
          message.expiresAt = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<QualificationEndorsement>): QualificationEndorsement {
    const message = createBaseQualificationEndorsement();
    message.id = object.id !== undefined && object.id !== null ? BigInt(object.id.toString()) : BigInt(0);
    message.qualificationValidator = object.qualificationValidator ?? "";
    message.qualificationDomain = object.qualificationDomain ?? "";
    message.endorser = object.endorser ?? "";
    message.reason = object.reason ?? "";
    message.weight = object.weight ?? 0;
    message.createdAt = object.createdAt !== undefined && object.createdAt !== null ? BigInt(object.createdAt.toString()) : BigInt(0);
    message.expiresAt = object.expiresAt !== undefined && object.expiresAt !== null ? BigInt(object.expiresAt.toString()) : BigInt(0);
    return message;
  }
};
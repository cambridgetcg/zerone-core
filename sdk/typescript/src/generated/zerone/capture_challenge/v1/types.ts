//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/** ChallengeStatus represents the lifecycle of a capture challenge. */
export enum ChallengeStatus {
  CHALLENGE_STATUS_UNSPECIFIED = 0,
  CHALLENGE_STATUS_OPEN = 1,
  CHALLENGE_STATUS_EVIDENCE = 2,
  CHALLENGE_STATUS_UNDER_REVIEW = 3,
  CHALLENGE_STATUS_RESOLVED = 4,
  CHALLENGE_STATUS_EXPIRED = 5,
  UNRECOGNIZED = -1,
}
export function challengeStatusFromJSON(object: any): ChallengeStatus {
  switch (object) {
    case 0:
    case "CHALLENGE_STATUS_UNSPECIFIED":
      return ChallengeStatus.CHALLENGE_STATUS_UNSPECIFIED;
    case 1:
    case "CHALLENGE_STATUS_OPEN":
      return ChallengeStatus.CHALLENGE_STATUS_OPEN;
    case 2:
    case "CHALLENGE_STATUS_EVIDENCE":
      return ChallengeStatus.CHALLENGE_STATUS_EVIDENCE;
    case 3:
    case "CHALLENGE_STATUS_UNDER_REVIEW":
      return ChallengeStatus.CHALLENGE_STATUS_UNDER_REVIEW;
    case 4:
    case "CHALLENGE_STATUS_RESOLVED":
      return ChallengeStatus.CHALLENGE_STATUS_RESOLVED;
    case 5:
    case "CHALLENGE_STATUS_EXPIRED":
      return ChallengeStatus.CHALLENGE_STATUS_EXPIRED;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ChallengeStatus.UNRECOGNIZED;
  }
}
export function challengeStatusToJSON(object: ChallengeStatus): string {
  switch (object) {
    case ChallengeStatus.CHALLENGE_STATUS_UNSPECIFIED:
      return "CHALLENGE_STATUS_UNSPECIFIED";
    case ChallengeStatus.CHALLENGE_STATUS_OPEN:
      return "CHALLENGE_STATUS_OPEN";
    case ChallengeStatus.CHALLENGE_STATUS_EVIDENCE:
      return "CHALLENGE_STATUS_EVIDENCE";
    case ChallengeStatus.CHALLENGE_STATUS_UNDER_REVIEW:
      return "CHALLENGE_STATUS_UNDER_REVIEW";
    case ChallengeStatus.CHALLENGE_STATUS_RESOLVED:
      return "CHALLENGE_STATUS_RESOLVED";
    case ChallengeStatus.CHALLENGE_STATUS_EXPIRED:
      return "CHALLENGE_STATUS_EXPIRED";
    case ChallengeStatus.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/** ChallengeOutcome represents the resolution of a challenge. */
export enum ChallengeOutcome {
  CHALLENGE_OUTCOME_UNSPECIFIED = 0,
  CHALLENGE_OUTCOME_UPHELD = 1,
  CHALLENGE_OUTCOME_REJECTED = 2,
  CHALLENGE_OUTCOME_PARTIAL = 3,
  UNRECOGNIZED = -1,
}
export function challengeOutcomeFromJSON(object: any): ChallengeOutcome {
  switch (object) {
    case 0:
    case "CHALLENGE_OUTCOME_UNSPECIFIED":
      return ChallengeOutcome.CHALLENGE_OUTCOME_UNSPECIFIED;
    case 1:
    case "CHALLENGE_OUTCOME_UPHELD":
      return ChallengeOutcome.CHALLENGE_OUTCOME_UPHELD;
    case 2:
    case "CHALLENGE_OUTCOME_REJECTED":
      return ChallengeOutcome.CHALLENGE_OUTCOME_REJECTED;
    case 3:
    case "CHALLENGE_OUTCOME_PARTIAL":
      return ChallengeOutcome.CHALLENGE_OUTCOME_PARTIAL;
    case -1:
    case "UNRECOGNIZED":
    default:
      return ChallengeOutcome.UNRECOGNIZED;
  }
}
export function challengeOutcomeToJSON(object: ChallengeOutcome): string {
  switch (object) {
    case ChallengeOutcome.CHALLENGE_OUTCOME_UNSPECIFIED:
      return "CHALLENGE_OUTCOME_UNSPECIFIED";
    case ChallengeOutcome.CHALLENGE_OUTCOME_UPHELD:
      return "CHALLENGE_OUTCOME_UPHELD";
    case ChallengeOutcome.CHALLENGE_OUTCOME_REJECTED:
      return "CHALLENGE_OUTCOME_REJECTED";
    case ChallengeOutcome.CHALLENGE_OUTCOME_PARTIAL:
      return "CHALLENGE_OUTCOME_PARTIAL";
    case ChallengeOutcome.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/**
 * CaptureEvidence is a piece of evidence attached to a challenge.
 * @name CaptureEvidence
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.CaptureEvidence
 */
export interface CaptureEvidence {
  description: string;
  dataHash: string;
  submittedBlock: bigint;
}
/**
 * CaptureResolution records how a challenge was resolved.
 * @name CaptureResolution
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.CaptureResolution
 */
export interface CaptureResolution {
  outcome: ChallengeOutcome;
  /**
   * authority address
   */
  resolver: string;
  reason: string;
  resolvedBlock: bigint;
  /**
   * uzrn
   */
  rewardAmount: string;
  /**
   * uzrn
   */
  slashAmount: string;
}
/**
 * ValidatorSlash records a slash applied to an accused validator.
 * @name ValidatorSlash
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.ValidatorSlash
 */
export interface ValidatorSlash {
  validator: string;
  /**
   * uzrn
   */
  slashAmount: string;
  reason: string;
}
/**
 * CaptureChallenge is the main challenge record.
 * @name CaptureChallenge
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.CaptureChallenge
 */
export interface CaptureChallenge {
  id: string;
  challenger: string;
  domain: string;
  accusedValidators: string[];
  /**
   * uzrn
   */
  stake: string;
  status: ChallengeStatus;
  evidence: CaptureEvidence[];
  createdBlock: bigint;
  evidenceDeadline: bigint;
  reviewDeadline: bigint;
  resolution?: CaptureResolution;
  slashes: ValidatorSlash[];
}
/**
 * DomainBountyPool holds the bounty fund for a domain.
 * @name DomainBountyPool
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.DomainBountyPool
 */
export interface DomainBountyPool {
  domain: string;
  /**
   * uzrn
   */
  balance: string;
}
function createBaseCaptureEvidence(): CaptureEvidence {
  return {
    description: "",
    dataHash: "",
    submittedBlock: BigInt(0)
  };
}
/**
 * CaptureEvidence is a piece of evidence attached to a challenge.
 * @name CaptureEvidence
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.CaptureEvidence
 */
export const CaptureEvidence = {
  typeUrl: "/zerone.capture_challenge.v1.CaptureEvidence",
  encode(message: CaptureEvidence, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.description !== "") {
      writer.uint32(10).string(message.description);
    }
    if (message.dataHash !== "") {
      writer.uint32(18).string(message.dataHash);
    }
    if (message.submittedBlock !== BigInt(0)) {
      writer.uint32(24).uint64(message.submittedBlock);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): CaptureEvidence {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCaptureEvidence();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.description = reader.string();
          break;
        case 2:
          message.dataHash = reader.string();
          break;
        case 3:
          message.submittedBlock = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<CaptureEvidence>): CaptureEvidence {
    const message = createBaseCaptureEvidence();
    message.description = object.description ?? "";
    message.dataHash = object.dataHash ?? "";
    message.submittedBlock = object.submittedBlock !== undefined && object.submittedBlock !== null ? BigInt(object.submittedBlock.toString()) : BigInt(0);
    return message;
  }
};
function createBaseCaptureResolution(): CaptureResolution {
  return {
    outcome: 0,
    resolver: "",
    reason: "",
    resolvedBlock: BigInt(0),
    rewardAmount: "",
    slashAmount: ""
  };
}
/**
 * CaptureResolution records how a challenge was resolved.
 * @name CaptureResolution
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.CaptureResolution
 */
export const CaptureResolution = {
  typeUrl: "/zerone.capture_challenge.v1.CaptureResolution",
  encode(message: CaptureResolution, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.outcome !== 0) {
      writer.uint32(8).int32(message.outcome);
    }
    if (message.resolver !== "") {
      writer.uint32(18).string(message.resolver);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    if (message.resolvedBlock !== BigInt(0)) {
      writer.uint32(32).uint64(message.resolvedBlock);
    }
    if (message.rewardAmount !== "") {
      writer.uint32(42).string(message.rewardAmount);
    }
    if (message.slashAmount !== "") {
      writer.uint32(50).string(message.slashAmount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): CaptureResolution {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCaptureResolution();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.outcome = reader.int32() as any;
          break;
        case 2:
          message.resolver = reader.string();
          break;
        case 3:
          message.reason = reader.string();
          break;
        case 4:
          message.resolvedBlock = reader.uint64();
          break;
        case 5:
          message.rewardAmount = reader.string();
          break;
        case 6:
          message.slashAmount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<CaptureResolution>): CaptureResolution {
    const message = createBaseCaptureResolution();
    message.outcome = object.outcome ?? 0;
    message.resolver = object.resolver ?? "";
    message.reason = object.reason ?? "";
    message.resolvedBlock = object.resolvedBlock !== undefined && object.resolvedBlock !== null ? BigInt(object.resolvedBlock.toString()) : BigInt(0);
    message.rewardAmount = object.rewardAmount ?? "";
    message.slashAmount = object.slashAmount ?? "";
    return message;
  }
};
function createBaseValidatorSlash(): ValidatorSlash {
  return {
    validator: "",
    slashAmount: "",
    reason: ""
  };
}
/**
 * ValidatorSlash records a slash applied to an accused validator.
 * @name ValidatorSlash
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.ValidatorSlash
 */
export const ValidatorSlash = {
  typeUrl: "/zerone.capture_challenge.v1.ValidatorSlash",
  encode(message: ValidatorSlash, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.validator !== "") {
      writer.uint32(10).string(message.validator);
    }
    if (message.slashAmount !== "") {
      writer.uint32(18).string(message.slashAmount);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): ValidatorSlash {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseValidatorSlash();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.validator = reader.string();
          break;
        case 2:
          message.slashAmount = reader.string();
          break;
        case 3:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<ValidatorSlash>): ValidatorSlash {
    const message = createBaseValidatorSlash();
    message.validator = object.validator ?? "";
    message.slashAmount = object.slashAmount ?? "";
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseCaptureChallenge(): CaptureChallenge {
  return {
    id: "",
    challenger: "",
    domain: "",
    accusedValidators: [],
    stake: "",
    status: 0,
    evidence: [],
    createdBlock: BigInt(0),
    evidenceDeadline: BigInt(0),
    reviewDeadline: BigInt(0),
    resolution: undefined,
    slashes: []
  };
}
/**
 * CaptureChallenge is the main challenge record.
 * @name CaptureChallenge
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.CaptureChallenge
 */
export const CaptureChallenge = {
  typeUrl: "/zerone.capture_challenge.v1.CaptureChallenge",
  encode(message: CaptureChallenge, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.challenger !== "") {
      writer.uint32(18).string(message.challenger);
    }
    if (message.domain !== "") {
      writer.uint32(26).string(message.domain);
    }
    for (const v of message.accusedValidators) {
      writer.uint32(34).string(v!);
    }
    if (message.stake !== "") {
      writer.uint32(42).string(message.stake);
    }
    if (message.status !== 0) {
      writer.uint32(48).int32(message.status);
    }
    for (const v of message.evidence) {
      CaptureEvidence.encode(v!, writer.uint32(58).fork()).ldelim();
    }
    if (message.createdBlock !== BigInt(0)) {
      writer.uint32(64).uint64(message.createdBlock);
    }
    if (message.evidenceDeadline !== BigInt(0)) {
      writer.uint32(72).uint64(message.evidenceDeadline);
    }
    if (message.reviewDeadline !== BigInt(0)) {
      writer.uint32(80).uint64(message.reviewDeadline);
    }
    if (message.resolution !== undefined) {
      CaptureResolution.encode(message.resolution, writer.uint32(90).fork()).ldelim();
    }
    for (const v of message.slashes) {
      ValidatorSlash.encode(v!, writer.uint32(98).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): CaptureChallenge {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseCaptureChallenge();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.challenger = reader.string();
          break;
        case 3:
          message.domain = reader.string();
          break;
        case 4:
          message.accusedValidators.push(reader.string());
          break;
        case 5:
          message.stake = reader.string();
          break;
        case 6:
          message.status = reader.int32() as any;
          break;
        case 7:
          message.evidence.push(CaptureEvidence.decode(reader, reader.uint32()));
          break;
        case 8:
          message.createdBlock = reader.uint64();
          break;
        case 9:
          message.evidenceDeadline = reader.uint64();
          break;
        case 10:
          message.reviewDeadline = reader.uint64();
          break;
        case 11:
          message.resolution = CaptureResolution.decode(reader, reader.uint32());
          break;
        case 12:
          message.slashes.push(ValidatorSlash.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<CaptureChallenge>): CaptureChallenge {
    const message = createBaseCaptureChallenge();
    message.id = object.id ?? "";
    message.challenger = object.challenger ?? "";
    message.domain = object.domain ?? "";
    message.accusedValidators = object.accusedValidators?.map(e => e) || [];
    message.stake = object.stake ?? "";
    message.status = object.status ?? 0;
    message.evidence = object.evidence?.map(e => CaptureEvidence.fromPartial(e)) || [];
    message.createdBlock = object.createdBlock !== undefined && object.createdBlock !== null ? BigInt(object.createdBlock.toString()) : BigInt(0);
    message.evidenceDeadline = object.evidenceDeadline !== undefined && object.evidenceDeadline !== null ? BigInt(object.evidenceDeadline.toString()) : BigInt(0);
    message.reviewDeadline = object.reviewDeadline !== undefined && object.reviewDeadline !== null ? BigInt(object.reviewDeadline.toString()) : BigInt(0);
    message.resolution = object.resolution !== undefined && object.resolution !== null ? CaptureResolution.fromPartial(object.resolution) : undefined;
    message.slashes = object.slashes?.map(e => ValidatorSlash.fromPartial(e)) || [];
    return message;
  }
};
function createBaseDomainBountyPool(): DomainBountyPool {
  return {
    domain: "",
    balance: ""
  };
}
/**
 * DomainBountyPool holds the bounty fund for a domain.
 * @name DomainBountyPool
 * @package zerone.capture_challenge.v1
 * @see proto type: zerone.capture_challenge.v1.DomainBountyPool
 */
export const DomainBountyPool = {
  typeUrl: "/zerone.capture_challenge.v1.DomainBountyPool",
  encode(message: DomainBountyPool, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.domain !== "") {
      writer.uint32(10).string(message.domain);
    }
    if (message.balance !== "") {
      writer.uint32(18).string(message.balance);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): DomainBountyPool {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseDomainBountyPool();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.domain = reader.string();
          break;
        case 2:
          message.balance = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<DomainBountyPool>): DomainBountyPool {
    const message = createBaseDomainBountyPool();
    message.domain = object.domain ?? "";
    message.balance = object.balance ?? "";
    return message;
  }
};
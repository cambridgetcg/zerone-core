//@ts-nocheck
import { Params } from "./genesis";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/** EmergencyCategory classifies the emergency reason. */
export enum EmergencyCategory {
  EMERGENCY_CATEGORY_UNSPECIFIED = 0,
  EMERGENCY_CATEGORY_SECURITY_BREACH = 1,
  EMERGENCY_CATEGORY_CONSENSUS_FAILURE = 2,
  EMERGENCY_CATEGORY_ECONOMIC_EXPLOIT = 3,
  EMERGENCY_CATEGORY_STATE_CORRUPTION = 4,
  UNRECOGNIZED = -1,
}
export function emergencyCategoryFromJSON(object: any): EmergencyCategory {
  switch (object) {
    case 0:
    case "EMERGENCY_CATEGORY_UNSPECIFIED":
      return EmergencyCategory.EMERGENCY_CATEGORY_UNSPECIFIED;
    case 1:
    case "EMERGENCY_CATEGORY_SECURITY_BREACH":
      return EmergencyCategory.EMERGENCY_CATEGORY_SECURITY_BREACH;
    case 2:
    case "EMERGENCY_CATEGORY_CONSENSUS_FAILURE":
      return EmergencyCategory.EMERGENCY_CATEGORY_CONSENSUS_FAILURE;
    case 3:
    case "EMERGENCY_CATEGORY_ECONOMIC_EXPLOIT":
      return EmergencyCategory.EMERGENCY_CATEGORY_ECONOMIC_EXPLOIT;
    case 4:
    case "EMERGENCY_CATEGORY_STATE_CORRUPTION":
      return EmergencyCategory.EMERGENCY_CATEGORY_STATE_CORRUPTION;
    case -1:
    case "UNRECOGNIZED":
    default:
      return EmergencyCategory.UNRECOGNIZED;
  }
}
export function emergencyCategoryToJSON(object: EmergencyCategory): string {
  switch (object) {
    case EmergencyCategory.EMERGENCY_CATEGORY_UNSPECIFIED:
      return "EMERGENCY_CATEGORY_UNSPECIFIED";
    case EmergencyCategory.EMERGENCY_CATEGORY_SECURITY_BREACH:
      return "EMERGENCY_CATEGORY_SECURITY_BREACH";
    case EmergencyCategory.EMERGENCY_CATEGORY_CONSENSUS_FAILURE:
      return "EMERGENCY_CATEGORY_CONSENSUS_FAILURE";
    case EmergencyCategory.EMERGENCY_CATEGORY_ECONOMIC_EXPLOIT:
      return "EMERGENCY_CATEGORY_ECONOMIC_EXPLOIT";
    case EmergencyCategory.EMERGENCY_CATEGORY_STATE_CORRUPTION:
      return "EMERGENCY_CATEGORY_STATE_CORRUPTION";
    case EmergencyCategory.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/**
 * @name MsgProposeHalt
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeHalt
 */
export interface MsgProposeHalt {
  proposer: string;
  reason: string;
  category: EmergencyCategory;
}
/**
 * @name MsgProposeHaltResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeHaltResponse
 */
export interface MsgProposeHaltResponse {
  proposalId: string;
}
/**
 * @name MsgVoteHalt
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteHalt
 */
export interface MsgVoteHalt {
  voter: string;
  proposalId: string;
  approve: boolean;
}
/**
 * @name MsgVoteHaltResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteHaltResponse
 */
export interface MsgVoteHaltResponse {
  quorumReached: boolean;
  chainHalted: boolean;
}
/**
 * @name MsgProposeRevert
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeRevert
 */
export interface MsgProposeRevert {
  proposer: string;
  revertToHeight: bigint;
  justification: string;
}
/**
 * @name MsgProposeRevertResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeRevertResponse
 */
export interface MsgProposeRevertResponse {
  proposalId: string;
}
/**
 * @name MsgVoteRevert
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteRevert
 */
export interface MsgVoteRevert {
  voter: string;
  proposalId: string;
  approve: boolean;
}
/**
 * @name MsgVoteRevertResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteRevertResponse
 */
export interface MsgVoteRevertResponse {
  quorumReached: boolean;
  revertExecuted: boolean;
}
/**
 * @name MsgProposeResume
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeResume
 */
export interface MsgProposeResume {
  proposer: string;
  justification: string;
  /**
   * Lowercase 64-character SHA-256 of the canonical recovery manifest.
   */
  recoveryManifestSha256: string;
}
/**
 * @name MsgProposeResumeResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeResumeResponse
 */
export interface MsgProposeResumeResponse {
  proposalId: string;
}
/**
 * @name MsgVoteResume
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteResume
 */
export interface MsgVoteResume {
  voter: string;
  proposalId: string;
  approve: boolean;
}
/**
 * @name MsgVoteResumeResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteResumeResponse
 */
export interface MsgVoteResumeResponse {
  quorumReached: boolean;
  chainResumed: boolean;
}
/**
 * @name MsgProposeRecoveryAuthorization
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeRecoveryAuthorization
 */
export interface MsgProposeRecoveryAuthorization {
  proposer: string;
  sdkGovProposalId: bigint;
  /**
   * Canonical lowercase SHA-256 of the proposal's sole recovery action.
   */
  actionSha256: string;
  /**
   * Canonical lowercase SHA-256 of the reviewed, signed recovery manifest.
   */
  recoveryManifestSha256: string;
  justification: string;
  upgradePlanSha256: string;
  authorizedSubmitter: string;
  /**
   * software_upgrade, cancel_upgrade, or revoke. A revoke ceremony repeats
   * the exact hashes, proposal ID, and submitter of the live authorization.
   */
  actionType: string;
}
/**
 * @name MsgProposeRecoveryAuthorizationResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeRecoveryAuthorizationResponse
 */
export interface MsgProposeRecoveryAuthorizationResponse {
  proposalId: string;
}
/**
 * @name MsgVoteRecoveryAuthorization
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteRecoveryAuthorization
 */
export interface MsgVoteRecoveryAuthorization {
  voter: string;
  proposalId: string;
  approve: boolean;
}
/**
 * @name MsgVoteRecoveryAuthorizationResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteRecoveryAuthorizationResponse
 */
export interface MsgVoteRecoveryAuthorizationResponse {
  quorumReached: boolean;
  recoveryAuthorized: boolean;
}
/**
 * @name MsgUpdateParams
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
  authority: string;
  params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {}
function createBaseMsgProposeHalt(): MsgProposeHalt {
  return {
    proposer: "",
    reason: "",
    category: 0
  };
}
/**
 * @name MsgProposeHalt
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeHalt
 */
export const MsgProposeHalt = {
  typeUrl: "/zerone.emergency.v1.MsgProposeHalt",
  encode(message: MsgProposeHalt, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.reason !== "") {
      writer.uint32(18).string(message.reason);
    }
    if (message.category !== 0) {
      writer.uint32(24).int32(message.category);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeHalt {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeHalt();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.reason = reader.string();
          break;
        case 3:
          message.category = reader.int32() as any;
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgProposeHalt>): MsgProposeHalt {
    const message = createBaseMsgProposeHalt();
    message.proposer = object.proposer ?? "";
    message.reason = object.reason ?? "";
    message.category = object.category ?? 0;
    return message;
  }
};
function createBaseMsgProposeHaltResponse(): MsgProposeHaltResponse {
  return {
    proposalId: ""
  };
}
/**
 * @name MsgProposeHaltResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeHaltResponse
 */
export const MsgProposeHaltResponse = {
  typeUrl: "/zerone.emergency.v1.MsgProposeHaltResponse",
  encode(message: MsgProposeHaltResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposalId !== "") {
      writer.uint32(10).string(message.proposalId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeHaltResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeHaltResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposalId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgProposeHaltResponse>): MsgProposeHaltResponse {
    const message = createBaseMsgProposeHaltResponse();
    message.proposalId = object.proposalId ?? "";
    return message;
  }
};
function createBaseMsgVoteHalt(): MsgVoteHalt {
  return {
    voter: "",
    proposalId: "",
    approve: false
  };
}
/**
 * @name MsgVoteHalt
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteHalt
 */
export const MsgVoteHalt = {
  typeUrl: "/zerone.emergency.v1.MsgVoteHalt",
  encode(message: MsgVoteHalt, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.voter !== "") {
      writer.uint32(10).string(message.voter);
    }
    if (message.proposalId !== "") {
      writer.uint32(18).string(message.proposalId);
    }
    if (message.approve === true) {
      writer.uint32(24).bool(message.approve);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteHalt {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteHalt();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.voter = reader.string();
          break;
        case 2:
          message.proposalId = reader.string();
          break;
        case 3:
          message.approve = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgVoteHalt>): MsgVoteHalt {
    const message = createBaseMsgVoteHalt();
    message.voter = object.voter ?? "";
    message.proposalId = object.proposalId ?? "";
    message.approve = object.approve ?? false;
    return message;
  }
};
function createBaseMsgVoteHaltResponse(): MsgVoteHaltResponse {
  return {
    quorumReached: false,
    chainHalted: false
  };
}
/**
 * @name MsgVoteHaltResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteHaltResponse
 */
export const MsgVoteHaltResponse = {
  typeUrl: "/zerone.emergency.v1.MsgVoteHaltResponse",
  encode(message: MsgVoteHaltResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.quorumReached === true) {
      writer.uint32(8).bool(message.quorumReached);
    }
    if (message.chainHalted === true) {
      writer.uint32(16).bool(message.chainHalted);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteHaltResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteHaltResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.quorumReached = reader.bool();
          break;
        case 2:
          message.chainHalted = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgVoteHaltResponse>): MsgVoteHaltResponse {
    const message = createBaseMsgVoteHaltResponse();
    message.quorumReached = object.quorumReached ?? false;
    message.chainHalted = object.chainHalted ?? false;
    return message;
  }
};
function createBaseMsgProposeRevert(): MsgProposeRevert {
  return {
    proposer: "",
    revertToHeight: BigInt(0),
    justification: ""
  };
}
/**
 * @name MsgProposeRevert
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeRevert
 */
export const MsgProposeRevert = {
  typeUrl: "/zerone.emergency.v1.MsgProposeRevert",
  encode(message: MsgProposeRevert, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.revertToHeight !== BigInt(0)) {
      writer.uint32(16).uint64(message.revertToHeight);
    }
    if (message.justification !== "") {
      writer.uint32(26).string(message.justification);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeRevert {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeRevert();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.revertToHeight = reader.uint64();
          break;
        case 3:
          message.justification = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgProposeRevert>): MsgProposeRevert {
    const message = createBaseMsgProposeRevert();
    message.proposer = object.proposer ?? "";
    message.revertToHeight = object.revertToHeight !== undefined && object.revertToHeight !== null ? BigInt(object.revertToHeight.toString()) : BigInt(0);
    message.justification = object.justification ?? "";
    return message;
  }
};
function createBaseMsgProposeRevertResponse(): MsgProposeRevertResponse {
  return {
    proposalId: ""
  };
}
/**
 * @name MsgProposeRevertResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeRevertResponse
 */
export const MsgProposeRevertResponse = {
  typeUrl: "/zerone.emergency.v1.MsgProposeRevertResponse",
  encode(message: MsgProposeRevertResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposalId !== "") {
      writer.uint32(10).string(message.proposalId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeRevertResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeRevertResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposalId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgProposeRevertResponse>): MsgProposeRevertResponse {
    const message = createBaseMsgProposeRevertResponse();
    message.proposalId = object.proposalId ?? "";
    return message;
  }
};
function createBaseMsgVoteRevert(): MsgVoteRevert {
  return {
    voter: "",
    proposalId: "",
    approve: false
  };
}
/**
 * @name MsgVoteRevert
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteRevert
 */
export const MsgVoteRevert = {
  typeUrl: "/zerone.emergency.v1.MsgVoteRevert",
  encode(message: MsgVoteRevert, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.voter !== "") {
      writer.uint32(10).string(message.voter);
    }
    if (message.proposalId !== "") {
      writer.uint32(18).string(message.proposalId);
    }
    if (message.approve === true) {
      writer.uint32(24).bool(message.approve);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteRevert {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteRevert();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.voter = reader.string();
          break;
        case 2:
          message.proposalId = reader.string();
          break;
        case 3:
          message.approve = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgVoteRevert>): MsgVoteRevert {
    const message = createBaseMsgVoteRevert();
    message.voter = object.voter ?? "";
    message.proposalId = object.proposalId ?? "";
    message.approve = object.approve ?? false;
    return message;
  }
};
function createBaseMsgVoteRevertResponse(): MsgVoteRevertResponse {
  return {
    quorumReached: false,
    revertExecuted: false
  };
}
/**
 * @name MsgVoteRevertResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteRevertResponse
 */
export const MsgVoteRevertResponse = {
  typeUrl: "/zerone.emergency.v1.MsgVoteRevertResponse",
  encode(message: MsgVoteRevertResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.quorumReached === true) {
      writer.uint32(8).bool(message.quorumReached);
    }
    if (message.revertExecuted === true) {
      writer.uint32(16).bool(message.revertExecuted);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteRevertResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteRevertResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.quorumReached = reader.bool();
          break;
        case 2:
          message.revertExecuted = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgVoteRevertResponse>): MsgVoteRevertResponse {
    const message = createBaseMsgVoteRevertResponse();
    message.quorumReached = object.quorumReached ?? false;
    message.revertExecuted = object.revertExecuted ?? false;
    return message;
  }
};
function createBaseMsgProposeResume(): MsgProposeResume {
  return {
    proposer: "",
    justification: "",
    recoveryManifestSha256: ""
  };
}
/**
 * @name MsgProposeResume
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeResume
 */
export const MsgProposeResume = {
  typeUrl: "/zerone.emergency.v1.MsgProposeResume",
  encode(message: MsgProposeResume, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.justification !== "") {
      writer.uint32(18).string(message.justification);
    }
    if (message.recoveryManifestSha256 !== "") {
      writer.uint32(26).string(message.recoveryManifestSha256);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeResume {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeResume();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.justification = reader.string();
          break;
        case 3:
          message.recoveryManifestSha256 = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgProposeResume>): MsgProposeResume {
    const message = createBaseMsgProposeResume();
    message.proposer = object.proposer ?? "";
    message.justification = object.justification ?? "";
    message.recoveryManifestSha256 = object.recoveryManifestSha256 ?? "";
    return message;
  }
};
function createBaseMsgProposeResumeResponse(): MsgProposeResumeResponse {
  return {
    proposalId: ""
  };
}
/**
 * @name MsgProposeResumeResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeResumeResponse
 */
export const MsgProposeResumeResponse = {
  typeUrl: "/zerone.emergency.v1.MsgProposeResumeResponse",
  encode(message: MsgProposeResumeResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposalId !== "") {
      writer.uint32(10).string(message.proposalId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeResumeResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeResumeResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposalId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgProposeResumeResponse>): MsgProposeResumeResponse {
    const message = createBaseMsgProposeResumeResponse();
    message.proposalId = object.proposalId ?? "";
    return message;
  }
};
function createBaseMsgVoteResume(): MsgVoteResume {
  return {
    voter: "",
    proposalId: "",
    approve: false
  };
}
/**
 * @name MsgVoteResume
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteResume
 */
export const MsgVoteResume = {
  typeUrl: "/zerone.emergency.v1.MsgVoteResume",
  encode(message: MsgVoteResume, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.voter !== "") {
      writer.uint32(10).string(message.voter);
    }
    if (message.proposalId !== "") {
      writer.uint32(18).string(message.proposalId);
    }
    if (message.approve === true) {
      writer.uint32(24).bool(message.approve);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteResume {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteResume();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.voter = reader.string();
          break;
        case 2:
          message.proposalId = reader.string();
          break;
        case 3:
          message.approve = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgVoteResume>): MsgVoteResume {
    const message = createBaseMsgVoteResume();
    message.voter = object.voter ?? "";
    message.proposalId = object.proposalId ?? "";
    message.approve = object.approve ?? false;
    return message;
  }
};
function createBaseMsgVoteResumeResponse(): MsgVoteResumeResponse {
  return {
    quorumReached: false,
    chainResumed: false
  };
}
/**
 * @name MsgVoteResumeResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteResumeResponse
 */
export const MsgVoteResumeResponse = {
  typeUrl: "/zerone.emergency.v1.MsgVoteResumeResponse",
  encode(message: MsgVoteResumeResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.quorumReached === true) {
      writer.uint32(8).bool(message.quorumReached);
    }
    if (message.chainResumed === true) {
      writer.uint32(16).bool(message.chainResumed);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteResumeResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteResumeResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.quorumReached = reader.bool();
          break;
        case 2:
          message.chainResumed = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgVoteResumeResponse>): MsgVoteResumeResponse {
    const message = createBaseMsgVoteResumeResponse();
    message.quorumReached = object.quorumReached ?? false;
    message.chainResumed = object.chainResumed ?? false;
    return message;
  }
};
function createBaseMsgProposeRecoveryAuthorization(): MsgProposeRecoveryAuthorization {
  return {
    proposer: "",
    sdkGovProposalId: BigInt(0),
    actionSha256: "",
    recoveryManifestSha256: "",
    justification: "",
    upgradePlanSha256: "",
    authorizedSubmitter: "",
    actionType: ""
  };
}
/**
 * @name MsgProposeRecoveryAuthorization
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeRecoveryAuthorization
 */
export const MsgProposeRecoveryAuthorization = {
  typeUrl: "/zerone.emergency.v1.MsgProposeRecoveryAuthorization",
  encode(message: MsgProposeRecoveryAuthorization, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.sdkGovProposalId !== BigInt(0)) {
      writer.uint32(16).uint64(message.sdkGovProposalId);
    }
    if (message.actionSha256 !== "") {
      writer.uint32(26).string(message.actionSha256);
    }
    if (message.recoveryManifestSha256 !== "") {
      writer.uint32(34).string(message.recoveryManifestSha256);
    }
    if (message.justification !== "") {
      writer.uint32(42).string(message.justification);
    }
    if (message.upgradePlanSha256 !== "") {
      writer.uint32(50).string(message.upgradePlanSha256);
    }
    if (message.authorizedSubmitter !== "") {
      writer.uint32(58).string(message.authorizedSubmitter);
    }
    if (message.actionType !== "") {
      writer.uint32(66).string(message.actionType);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeRecoveryAuthorization {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeRecoveryAuthorization();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.sdkGovProposalId = reader.uint64();
          break;
        case 3:
          message.actionSha256 = reader.string();
          break;
        case 4:
          message.recoveryManifestSha256 = reader.string();
          break;
        case 5:
          message.justification = reader.string();
          break;
        case 6:
          message.upgradePlanSha256 = reader.string();
          break;
        case 7:
          message.authorizedSubmitter = reader.string();
          break;
        case 8:
          message.actionType = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgProposeRecoveryAuthorization>): MsgProposeRecoveryAuthorization {
    const message = createBaseMsgProposeRecoveryAuthorization();
    message.proposer = object.proposer ?? "";
    message.sdkGovProposalId = object.sdkGovProposalId !== undefined && object.sdkGovProposalId !== null ? BigInt(object.sdkGovProposalId.toString()) : BigInt(0);
    message.actionSha256 = object.actionSha256 ?? "";
    message.recoveryManifestSha256 = object.recoveryManifestSha256 ?? "";
    message.justification = object.justification ?? "";
    message.upgradePlanSha256 = object.upgradePlanSha256 ?? "";
    message.authorizedSubmitter = object.authorizedSubmitter ?? "";
    message.actionType = object.actionType ?? "";
    return message;
  }
};
function createBaseMsgProposeRecoveryAuthorizationResponse(): MsgProposeRecoveryAuthorizationResponse {
  return {
    proposalId: ""
  };
}
/**
 * @name MsgProposeRecoveryAuthorizationResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgProposeRecoveryAuthorizationResponse
 */
export const MsgProposeRecoveryAuthorizationResponse = {
  typeUrl: "/zerone.emergency.v1.MsgProposeRecoveryAuthorizationResponse",
  encode(message: MsgProposeRecoveryAuthorizationResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposalId !== "") {
      writer.uint32(10).string(message.proposalId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeRecoveryAuthorizationResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeRecoveryAuthorizationResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposalId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgProposeRecoveryAuthorizationResponse>): MsgProposeRecoveryAuthorizationResponse {
    const message = createBaseMsgProposeRecoveryAuthorizationResponse();
    message.proposalId = object.proposalId ?? "";
    return message;
  }
};
function createBaseMsgVoteRecoveryAuthorization(): MsgVoteRecoveryAuthorization {
  return {
    voter: "",
    proposalId: "",
    approve: false
  };
}
/**
 * @name MsgVoteRecoveryAuthorization
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteRecoveryAuthorization
 */
export const MsgVoteRecoveryAuthorization = {
  typeUrl: "/zerone.emergency.v1.MsgVoteRecoveryAuthorization",
  encode(message: MsgVoteRecoveryAuthorization, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.voter !== "") {
      writer.uint32(10).string(message.voter);
    }
    if (message.proposalId !== "") {
      writer.uint32(18).string(message.proposalId);
    }
    if (message.approve === true) {
      writer.uint32(24).bool(message.approve);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteRecoveryAuthorization {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteRecoveryAuthorization();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.voter = reader.string();
          break;
        case 2:
          message.proposalId = reader.string();
          break;
        case 3:
          message.approve = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgVoteRecoveryAuthorization>): MsgVoteRecoveryAuthorization {
    const message = createBaseMsgVoteRecoveryAuthorization();
    message.voter = object.voter ?? "";
    message.proposalId = object.proposalId ?? "";
    message.approve = object.approve ?? false;
    return message;
  }
};
function createBaseMsgVoteRecoveryAuthorizationResponse(): MsgVoteRecoveryAuthorizationResponse {
  return {
    quorumReached: false,
    recoveryAuthorized: false
  };
}
/**
 * @name MsgVoteRecoveryAuthorizationResponse
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgVoteRecoveryAuthorizationResponse
 */
export const MsgVoteRecoveryAuthorizationResponse = {
  typeUrl: "/zerone.emergency.v1.MsgVoteRecoveryAuthorizationResponse",
  encode(message: MsgVoteRecoveryAuthorizationResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.quorumReached === true) {
      writer.uint32(8).bool(message.quorumReached);
    }
    if (message.recoveryAuthorized === true) {
      writer.uint32(16).bool(message.recoveryAuthorized);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteRecoveryAuthorizationResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteRecoveryAuthorizationResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.quorumReached = reader.bool();
          break;
        case 2:
          message.recoveryAuthorized = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgVoteRecoveryAuthorizationResponse>): MsgVoteRecoveryAuthorizationResponse {
    const message = createBaseMsgVoteRecoveryAuthorizationResponse();
    message.quorumReached = object.quorumReached ?? false;
    message.recoveryAuthorized = object.recoveryAuthorized ?? false;
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
 * @name MsgUpdateParams
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgUpdateParams
 */
export const MsgUpdateParams = {
  typeUrl: "/zerone.emergency.v1.MsgUpdateParams",
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
 * @package zerone.emergency.v1
 * @see proto type: zerone.emergency.v1.MsgUpdateParamsResponse
 */
export const MsgUpdateParamsResponse = {
  typeUrl: "/zerone.emergency.v1.MsgUpdateParamsResponse",
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
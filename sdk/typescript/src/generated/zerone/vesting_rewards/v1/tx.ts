//@ts-nocheck
import { Params } from "./genesis";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/** VestingCategory classifies the purpose of a vesting schedule. */
export enum VestingCategory {
  VESTING_CATEGORY_UNSPECIFIED = 0,
  VESTING_CATEGORY_VERIFICATION_REWARD = 1,
  VESTING_CATEGORY_BLOCK_REWARD = 2,
  VESTING_CATEGORY_BOUNTY_REWARD = 3,
  VESTING_CATEGORY_DISPUTE_REWARD = 4,
  VESTING_CATEGORY_RESEARCH_GRANT = 5,
  VESTING_CATEGORY_BOOTSTRAP = 6,
  UNRECOGNIZED = -1,
}
export function vestingCategoryFromJSON(object: any): VestingCategory {
  switch (object) {
    case 0:
    case "VESTING_CATEGORY_UNSPECIFIED":
      return VestingCategory.VESTING_CATEGORY_UNSPECIFIED;
    case 1:
    case "VESTING_CATEGORY_VERIFICATION_REWARD":
      return VestingCategory.VESTING_CATEGORY_VERIFICATION_REWARD;
    case 2:
    case "VESTING_CATEGORY_BLOCK_REWARD":
      return VestingCategory.VESTING_CATEGORY_BLOCK_REWARD;
    case 3:
    case "VESTING_CATEGORY_BOUNTY_REWARD":
      return VestingCategory.VESTING_CATEGORY_BOUNTY_REWARD;
    case 4:
    case "VESTING_CATEGORY_DISPUTE_REWARD":
      return VestingCategory.VESTING_CATEGORY_DISPUTE_REWARD;
    case 5:
    case "VESTING_CATEGORY_RESEARCH_GRANT":
      return VestingCategory.VESTING_CATEGORY_RESEARCH_GRANT;
    case 6:
    case "VESTING_CATEGORY_BOOTSTRAP":
      return VestingCategory.VESTING_CATEGORY_BOOTSTRAP;
    case -1:
    case "UNRECOGNIZED":
    default:
      return VestingCategory.UNRECOGNIZED;
  }
}
export function vestingCategoryToJSON(object: VestingCategory): string {
  switch (object) {
    case VestingCategory.VESTING_CATEGORY_UNSPECIFIED:
      return "VESTING_CATEGORY_UNSPECIFIED";
    case VestingCategory.VESTING_CATEGORY_VERIFICATION_REWARD:
      return "VESTING_CATEGORY_VERIFICATION_REWARD";
    case VestingCategory.VESTING_CATEGORY_BLOCK_REWARD:
      return "VESTING_CATEGORY_BLOCK_REWARD";
    case VestingCategory.VESTING_CATEGORY_BOUNTY_REWARD:
      return "VESTING_CATEGORY_BOUNTY_REWARD";
    case VestingCategory.VESTING_CATEGORY_DISPUTE_REWARD:
      return "VESTING_CATEGORY_DISPUTE_REWARD";
    case VestingCategory.VESTING_CATEGORY_RESEARCH_GRANT:
      return "VESTING_CATEGORY_RESEARCH_GRANT";
    case VestingCategory.VESTING_CATEGORY_BOOTSTRAP:
      return "VESTING_CATEGORY_BOOTSTRAP";
    case VestingCategory.UNRECOGNIZED:
    default:
      return "UNRECOGNIZED";
  }
}
/**
 * @name MsgCreateVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgCreateVesting
 */
export interface MsgCreateVesting {
  authority: string;
  beneficiary: string;
  amount: string;
  category: VestingCategory;
  linkedFactId: string;
  startHeight: bigint;
}
/**
 * @name MsgCreateVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgCreateVestingResponse
 */
export interface MsgCreateVestingResponse {
  vestingId: string;
}
/**
 * @name MsgClaimVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgClaimVesting
 */
export interface MsgClaimVesting {
  claimer: string;
  vestingIds: string[];
}
/**
 * @name MsgClaimVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgClaimVestingResponse
 */
export interface MsgClaimVestingResponse {
  totalClaimed: string;
  vestingsClaimed: number;
}
/**
 * @name MsgPauseVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgPauseVesting
 */
export interface MsgPauseVesting {
  authority: string;
  vestingId: string;
  reason: string;
}
/**
 * @name MsgPauseVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgPauseVestingResponse
 */
export interface MsgPauseVestingResponse {}
/**
 * @name MsgResumeVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgResumeVesting
 */
export interface MsgResumeVesting {
  authority: string;
  vestingId: string;
}
/**
 * @name MsgResumeVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgResumeVestingResponse
 */
export interface MsgResumeVestingResponse {}
/**
 * @name MsgAccelerateVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgAccelerateVesting
 */
export interface MsgAccelerateVesting {
  authority: string;
  vestingId: string;
  accelerationFactor: number;
}
/**
 * @name MsgAccelerateVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgAccelerateVestingResponse
 */
export interface MsgAccelerateVestingResponse {}
/**
 * @name MsgFalsifyVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgFalsifyVesting
 */
export interface MsgFalsifyVesting {
  challenger: string;
  vestingId: string;
  reason: string;
  counterEvidenceHash: string;
}
/**
 * @name MsgFalsifyVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgFalsifyVestingResponse
 */
export interface MsgFalsifyVestingResponse {
  vestingPaused: boolean;
}
/**
 * @name MsgCompleteVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgCompleteVesting
 */
export interface MsgCompleteVesting {
  authority: string;
  vestingId: string;
}
/**
 * @name MsgCompleteVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgCompleteVestingResponse
 */
export interface MsgCompleteVestingResponse {
  remainingAmount: string;
}
/**
 * @name MsgUpdateParams
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
  authority: string;
  params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {}
function createBaseMsgCreateVesting(): MsgCreateVesting {
  return {
    authority: "",
    beneficiary: "",
    amount: "",
    category: 0,
    linkedFactId: "",
    startHeight: BigInt(0)
  };
}
/**
 * @name MsgCreateVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgCreateVesting
 */
export const MsgCreateVesting = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgCreateVesting",
  encode(message: MsgCreateVesting, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.beneficiary !== "") {
      writer.uint32(18).string(message.beneficiary);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    if (message.category !== 0) {
      writer.uint32(32).int32(message.category);
    }
    if (message.linkedFactId !== "") {
      writer.uint32(42).string(message.linkedFactId);
    }
    if (message.startHeight !== BigInt(0)) {
      writer.uint32(48).uint64(message.startHeight);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateVesting {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateVesting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.beneficiary = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        case 4:
          message.category = reader.int32() as any;
          break;
        case 5:
          message.linkedFactId = reader.string();
          break;
        case 6:
          message.startHeight = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCreateVesting>): MsgCreateVesting {
    const message = createBaseMsgCreateVesting();
    message.authority = object.authority ?? "";
    message.beneficiary = object.beneficiary ?? "";
    message.amount = object.amount ?? "";
    message.category = object.category ?? 0;
    message.linkedFactId = object.linkedFactId ?? "";
    message.startHeight = object.startHeight !== undefined && object.startHeight !== null ? BigInt(object.startHeight.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgCreateVestingResponse(): MsgCreateVestingResponse {
  return {
    vestingId: ""
  };
}
/**
 * @name MsgCreateVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgCreateVestingResponse
 */
export const MsgCreateVestingResponse = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgCreateVestingResponse",
  encode(message: MsgCreateVestingResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.vestingId !== "") {
      writer.uint32(10).string(message.vestingId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateVestingResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateVestingResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.vestingId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCreateVestingResponse>): MsgCreateVestingResponse {
    const message = createBaseMsgCreateVestingResponse();
    message.vestingId = object.vestingId ?? "";
    return message;
  }
};
function createBaseMsgClaimVesting(): MsgClaimVesting {
  return {
    claimer: "",
    vestingIds: []
  };
}
/**
 * @name MsgClaimVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgClaimVesting
 */
export const MsgClaimVesting = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgClaimVesting",
  encode(message: MsgClaimVesting, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.claimer !== "") {
      writer.uint32(10).string(message.claimer);
    }
    for (const v of message.vestingIds) {
      writer.uint32(18).string(v!);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgClaimVesting {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgClaimVesting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.claimer = reader.string();
          break;
        case 2:
          message.vestingIds.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgClaimVesting>): MsgClaimVesting {
    const message = createBaseMsgClaimVesting();
    message.claimer = object.claimer ?? "";
    message.vestingIds = object.vestingIds?.map(e => e) || [];
    return message;
  }
};
function createBaseMsgClaimVestingResponse(): MsgClaimVestingResponse {
  return {
    totalClaimed: "",
    vestingsClaimed: 0
  };
}
/**
 * @name MsgClaimVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgClaimVestingResponse
 */
export const MsgClaimVestingResponse = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgClaimVestingResponse",
  encode(message: MsgClaimVestingResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.totalClaimed !== "") {
      writer.uint32(10).string(message.totalClaimed);
    }
    if (message.vestingsClaimed !== 0) {
      writer.uint32(16).uint32(message.vestingsClaimed);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgClaimVestingResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgClaimVestingResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.totalClaimed = reader.string();
          break;
        case 2:
          message.vestingsClaimed = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgClaimVestingResponse>): MsgClaimVestingResponse {
    const message = createBaseMsgClaimVestingResponse();
    message.totalClaimed = object.totalClaimed ?? "";
    message.vestingsClaimed = object.vestingsClaimed ?? 0;
    return message;
  }
};
function createBaseMsgPauseVesting(): MsgPauseVesting {
  return {
    authority: "",
    vestingId: "",
    reason: ""
  };
}
/**
 * @name MsgPauseVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgPauseVesting
 */
export const MsgPauseVesting = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgPauseVesting",
  encode(message: MsgPauseVesting, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.vestingId !== "") {
      writer.uint32(18).string(message.vestingId);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgPauseVesting {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgPauseVesting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.vestingId = reader.string();
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
  fromPartial(object: DeepPartial<MsgPauseVesting>): MsgPauseVesting {
    const message = createBaseMsgPauseVesting();
    message.authority = object.authority ?? "";
    message.vestingId = object.vestingId ?? "";
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgPauseVestingResponse(): MsgPauseVestingResponse {
  return {};
}
/**
 * @name MsgPauseVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgPauseVestingResponse
 */
export const MsgPauseVestingResponse = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgPauseVestingResponse",
  encode(_: MsgPauseVestingResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgPauseVestingResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgPauseVestingResponse();
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
  fromPartial(_: DeepPartial<MsgPauseVestingResponse>): MsgPauseVestingResponse {
    const message = createBaseMsgPauseVestingResponse();
    return message;
  }
};
function createBaseMsgResumeVesting(): MsgResumeVesting {
  return {
    authority: "",
    vestingId: ""
  };
}
/**
 * @name MsgResumeVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgResumeVesting
 */
export const MsgResumeVesting = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgResumeVesting",
  encode(message: MsgResumeVesting, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.vestingId !== "") {
      writer.uint32(18).string(message.vestingId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgResumeVesting {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgResumeVesting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.vestingId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgResumeVesting>): MsgResumeVesting {
    const message = createBaseMsgResumeVesting();
    message.authority = object.authority ?? "";
    message.vestingId = object.vestingId ?? "";
    return message;
  }
};
function createBaseMsgResumeVestingResponse(): MsgResumeVestingResponse {
  return {};
}
/**
 * @name MsgResumeVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgResumeVestingResponse
 */
export const MsgResumeVestingResponse = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgResumeVestingResponse",
  encode(_: MsgResumeVestingResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgResumeVestingResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgResumeVestingResponse();
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
  fromPartial(_: DeepPartial<MsgResumeVestingResponse>): MsgResumeVestingResponse {
    const message = createBaseMsgResumeVestingResponse();
    return message;
  }
};
function createBaseMsgAccelerateVesting(): MsgAccelerateVesting {
  return {
    authority: "",
    vestingId: "",
    accelerationFactor: 0
  };
}
/**
 * @name MsgAccelerateVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgAccelerateVesting
 */
export const MsgAccelerateVesting = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgAccelerateVesting",
  encode(message: MsgAccelerateVesting, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.vestingId !== "") {
      writer.uint32(18).string(message.vestingId);
    }
    if (message.accelerationFactor !== 0) {
      writer.uint32(24).uint32(message.accelerationFactor);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAccelerateVesting {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAccelerateVesting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.vestingId = reader.string();
          break;
        case 3:
          message.accelerationFactor = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAccelerateVesting>): MsgAccelerateVesting {
    const message = createBaseMsgAccelerateVesting();
    message.authority = object.authority ?? "";
    message.vestingId = object.vestingId ?? "";
    message.accelerationFactor = object.accelerationFactor ?? 0;
    return message;
  }
};
function createBaseMsgAccelerateVestingResponse(): MsgAccelerateVestingResponse {
  return {};
}
/**
 * @name MsgAccelerateVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgAccelerateVestingResponse
 */
export const MsgAccelerateVestingResponse = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgAccelerateVestingResponse",
  encode(_: MsgAccelerateVestingResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAccelerateVestingResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAccelerateVestingResponse();
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
  fromPartial(_: DeepPartial<MsgAccelerateVestingResponse>): MsgAccelerateVestingResponse {
    const message = createBaseMsgAccelerateVestingResponse();
    return message;
  }
};
function createBaseMsgFalsifyVesting(): MsgFalsifyVesting {
  return {
    challenger: "",
    vestingId: "",
    reason: "",
    counterEvidenceHash: ""
  };
}
/**
 * @name MsgFalsifyVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgFalsifyVesting
 */
export const MsgFalsifyVesting = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgFalsifyVesting",
  encode(message: MsgFalsifyVesting, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.challenger !== "") {
      writer.uint32(10).string(message.challenger);
    }
    if (message.vestingId !== "") {
      writer.uint32(18).string(message.vestingId);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    if (message.counterEvidenceHash !== "") {
      writer.uint32(34).string(message.counterEvidenceHash);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgFalsifyVesting {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgFalsifyVesting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.challenger = reader.string();
          break;
        case 2:
          message.vestingId = reader.string();
          break;
        case 3:
          message.reason = reader.string();
          break;
        case 4:
          message.counterEvidenceHash = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgFalsifyVesting>): MsgFalsifyVesting {
    const message = createBaseMsgFalsifyVesting();
    message.challenger = object.challenger ?? "";
    message.vestingId = object.vestingId ?? "";
    message.reason = object.reason ?? "";
    message.counterEvidenceHash = object.counterEvidenceHash ?? "";
    return message;
  }
};
function createBaseMsgFalsifyVestingResponse(): MsgFalsifyVestingResponse {
  return {
    vestingPaused: false
  };
}
/**
 * @name MsgFalsifyVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgFalsifyVestingResponse
 */
export const MsgFalsifyVestingResponse = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgFalsifyVestingResponse",
  encode(message: MsgFalsifyVestingResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.vestingPaused === true) {
      writer.uint32(8).bool(message.vestingPaused);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgFalsifyVestingResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgFalsifyVestingResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.vestingPaused = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgFalsifyVestingResponse>): MsgFalsifyVestingResponse {
    const message = createBaseMsgFalsifyVestingResponse();
    message.vestingPaused = object.vestingPaused ?? false;
    return message;
  }
};
function createBaseMsgCompleteVesting(): MsgCompleteVesting {
  return {
    authority: "",
    vestingId: ""
  };
}
/**
 * @name MsgCompleteVesting
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgCompleteVesting
 */
export const MsgCompleteVesting = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgCompleteVesting",
  encode(message: MsgCompleteVesting, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.vestingId !== "") {
      writer.uint32(18).string(message.vestingId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCompleteVesting {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCompleteVesting();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.vestingId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCompleteVesting>): MsgCompleteVesting {
    const message = createBaseMsgCompleteVesting();
    message.authority = object.authority ?? "";
    message.vestingId = object.vestingId ?? "";
    return message;
  }
};
function createBaseMsgCompleteVestingResponse(): MsgCompleteVestingResponse {
  return {
    remainingAmount: ""
  };
}
/**
 * @name MsgCompleteVestingResponse
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgCompleteVestingResponse
 */
export const MsgCompleteVestingResponse = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgCompleteVestingResponse",
  encode(message: MsgCompleteVestingResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.remainingAmount !== "") {
      writer.uint32(10).string(message.remainingAmount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCompleteVestingResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCompleteVestingResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.remainingAmount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCompleteVestingResponse>): MsgCompleteVestingResponse {
    const message = createBaseMsgCompleteVestingResponse();
    message.remainingAmount = object.remainingAmount ?? "";
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
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgUpdateParams
 */
export const MsgUpdateParams = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgUpdateParams",
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
 * @package zerone.vesting_rewards.v1
 * @see proto type: zerone.vesting_rewards.v1.MsgUpdateParamsResponse
 */
export const MsgUpdateParamsResponse = {
  typeUrl: "/zerone.vesting_rewards.v1.MsgUpdateParamsResponse",
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
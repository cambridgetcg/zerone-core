//@ts-nocheck
import { Params } from "./genesis";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * MsgRegisterValidator registers a new validator.
 * @name MsgRegisterValidator
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgRegisterValidator
 */
export interface MsgRegisterValidator {
  operator: string;
  consensusPubkey: string;
  did: string;
  moniker: string;
  /**
   * uzrn
   */
  selfDelegation: string;
  commissionBps: bigint;
  website: string;
  details: string;
}
/**
 * @name MsgRegisterValidatorResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgRegisterValidatorResponse
 */
export interface MsgRegisterValidatorResponse {
  initialTier: number;
}
/**
 * MsgDelegate delegates tokens to a validator.
 * @name MsgDelegate
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgDelegate
 */
export interface MsgDelegate {
  delegator: string;
  validator: string;
  /**
   * uzrn
   */
  amount: string;
}
/**
 * @name MsgDelegateResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgDelegateResponse
 */
export interface MsgDelegateResponse {
  /**
   * total delegation after operation
   */
  newDelegation: string;
}
/**
 * MsgUndelegate initiates unbonding from a validator.
 * @name MsgUndelegate
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUndelegate
 */
export interface MsgUndelegate {
  delegator: string;
  validator: string;
  /**
   * uzrn
   */
  amount: string;
}
/**
 * @name MsgUndelegateResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUndelegateResponse
 */
export interface MsgUndelegateResponse {
  unbondingId: string;
  completesAtHeight: bigint;
}
/**
 * MsgRedelegate moves delegation between validators.
 * @name MsgRedelegate
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgRedelegate
 */
export interface MsgRedelegate {
  delegator: string;
  srcValidator: string;
  dstValidator: string;
  /**
   * uzrn
   */
  amount: string;
}
/**
 * @name MsgRedelegateResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgRedelegateResponse
 */
export interface MsgRedelegateResponse {}
/**
 * MsgUpdateValidatorStake increases or decreases self-delegation.
 * @name MsgUpdateValidatorStake
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUpdateValidatorStake
 */
export interface MsgUpdateValidatorStake {
  operator: string;
  /**
   * uzrn
   */
  amount: string;
  increase: boolean;
}
/**
 * @name MsgUpdateValidatorStakeResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUpdateValidatorStakeResponse
 */
export interface MsgUpdateValidatorStakeResponse {}
/**
 * MsgUpdateParams updates module parameters (governance-gated).
 * @name MsgUpdateParams
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
  authority: string;
  params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {}
function createBaseMsgRegisterValidator(): MsgRegisterValidator {
  return {
    operator: "",
    consensusPubkey: "",
    did: "",
    moniker: "",
    selfDelegation: "",
    commissionBps: BigInt(0),
    website: "",
    details: ""
  };
}
/**
 * MsgRegisterValidator registers a new validator.
 * @name MsgRegisterValidator
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgRegisterValidator
 */
export const MsgRegisterValidator = {
  typeUrl: "/zerone.staking.v1.MsgRegisterValidator",
  encode(message: MsgRegisterValidator, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.operator !== "") {
      writer.uint32(10).string(message.operator);
    }
    if (message.consensusPubkey !== "") {
      writer.uint32(18).string(message.consensusPubkey);
    }
    if (message.did !== "") {
      writer.uint32(26).string(message.did);
    }
    if (message.moniker !== "") {
      writer.uint32(34).string(message.moniker);
    }
    if (message.selfDelegation !== "") {
      writer.uint32(42).string(message.selfDelegation);
    }
    if (message.commissionBps !== BigInt(0)) {
      writer.uint32(48).uint64(message.commissionBps);
    }
    if (message.website !== "") {
      writer.uint32(58).string(message.website);
    }
    if (message.details !== "") {
      writer.uint32(66).string(message.details);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterValidator {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterValidator();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.operator = reader.string();
          break;
        case 2:
          message.consensusPubkey = reader.string();
          break;
        case 3:
          message.did = reader.string();
          break;
        case 4:
          message.moniker = reader.string();
          break;
        case 5:
          message.selfDelegation = reader.string();
          break;
        case 6:
          message.commissionBps = reader.uint64();
          break;
        case 7:
          message.website = reader.string();
          break;
        case 8:
          message.details = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgRegisterValidator>): MsgRegisterValidator {
    const message = createBaseMsgRegisterValidator();
    message.operator = object.operator ?? "";
    message.consensusPubkey = object.consensusPubkey ?? "";
    message.did = object.did ?? "";
    message.moniker = object.moniker ?? "";
    message.selfDelegation = object.selfDelegation ?? "";
    message.commissionBps = object.commissionBps !== undefined && object.commissionBps !== null ? BigInt(object.commissionBps.toString()) : BigInt(0);
    message.website = object.website ?? "";
    message.details = object.details ?? "";
    return message;
  }
};
function createBaseMsgRegisterValidatorResponse(): MsgRegisterValidatorResponse {
  return {
    initialTier: 0
  };
}
/**
 * @name MsgRegisterValidatorResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgRegisterValidatorResponse
 */
export const MsgRegisterValidatorResponse = {
  typeUrl: "/zerone.staking.v1.MsgRegisterValidatorResponse",
  encode(message: MsgRegisterValidatorResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.initialTier !== 0) {
      writer.uint32(8).uint32(message.initialTier);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterValidatorResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterValidatorResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.initialTier = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgRegisterValidatorResponse>): MsgRegisterValidatorResponse {
    const message = createBaseMsgRegisterValidatorResponse();
    message.initialTier = object.initialTier ?? 0;
    return message;
  }
};
function createBaseMsgDelegate(): MsgDelegate {
  return {
    delegator: "",
    validator: "",
    amount: ""
  };
}
/**
 * MsgDelegate delegates tokens to a validator.
 * @name MsgDelegate
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgDelegate
 */
export const MsgDelegate = {
  typeUrl: "/zerone.staking.v1.MsgDelegate",
  encode(message: MsgDelegate, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.delegator !== "") {
      writer.uint32(10).string(message.delegator);
    }
    if (message.validator !== "") {
      writer.uint32(18).string(message.validator);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgDelegate {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgDelegate();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.delegator = reader.string();
          break;
        case 2:
          message.validator = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgDelegate>): MsgDelegate {
    const message = createBaseMsgDelegate();
    message.delegator = object.delegator ?? "";
    message.validator = object.validator ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgDelegateResponse(): MsgDelegateResponse {
  return {
    newDelegation: ""
  };
}
/**
 * @name MsgDelegateResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgDelegateResponse
 */
export const MsgDelegateResponse = {
  typeUrl: "/zerone.staking.v1.MsgDelegateResponse",
  encode(message: MsgDelegateResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.newDelegation !== "") {
      writer.uint32(10).string(message.newDelegation);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgDelegateResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgDelegateResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.newDelegation = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgDelegateResponse>): MsgDelegateResponse {
    const message = createBaseMsgDelegateResponse();
    message.newDelegation = object.newDelegation ?? "";
    return message;
  }
};
function createBaseMsgUndelegate(): MsgUndelegate {
  return {
    delegator: "",
    validator: "",
    amount: ""
  };
}
/**
 * MsgUndelegate initiates unbonding from a validator.
 * @name MsgUndelegate
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUndelegate
 */
export const MsgUndelegate = {
  typeUrl: "/zerone.staking.v1.MsgUndelegate",
  encode(message: MsgUndelegate, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.delegator !== "") {
      writer.uint32(10).string(message.delegator);
    }
    if (message.validator !== "") {
      writer.uint32(18).string(message.validator);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUndelegate {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUndelegate();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.delegator = reader.string();
          break;
        case 2:
          message.validator = reader.string();
          break;
        case 3:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgUndelegate>): MsgUndelegate {
    const message = createBaseMsgUndelegate();
    message.delegator = object.delegator ?? "";
    message.validator = object.validator ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgUndelegateResponse(): MsgUndelegateResponse {
  return {
    unbondingId: "",
    completesAtHeight: BigInt(0)
  };
}
/**
 * @name MsgUndelegateResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUndelegateResponse
 */
export const MsgUndelegateResponse = {
  typeUrl: "/zerone.staking.v1.MsgUndelegateResponse",
  encode(message: MsgUndelegateResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.unbondingId !== "") {
      writer.uint32(10).string(message.unbondingId);
    }
    if (message.completesAtHeight !== BigInt(0)) {
      writer.uint32(16).uint64(message.completesAtHeight);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUndelegateResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUndelegateResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.unbondingId = reader.string();
          break;
        case 2:
          message.completesAtHeight = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgUndelegateResponse>): MsgUndelegateResponse {
    const message = createBaseMsgUndelegateResponse();
    message.unbondingId = object.unbondingId ?? "";
    message.completesAtHeight = object.completesAtHeight !== undefined && object.completesAtHeight !== null ? BigInt(object.completesAtHeight.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgRedelegate(): MsgRedelegate {
  return {
    delegator: "",
    srcValidator: "",
    dstValidator: "",
    amount: ""
  };
}
/**
 * MsgRedelegate moves delegation between validators.
 * @name MsgRedelegate
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgRedelegate
 */
export const MsgRedelegate = {
  typeUrl: "/zerone.staking.v1.MsgRedelegate",
  encode(message: MsgRedelegate, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.delegator !== "") {
      writer.uint32(10).string(message.delegator);
    }
    if (message.srcValidator !== "") {
      writer.uint32(18).string(message.srcValidator);
    }
    if (message.dstValidator !== "") {
      writer.uint32(26).string(message.dstValidator);
    }
    if (message.amount !== "") {
      writer.uint32(34).string(message.amount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRedelegate {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRedelegate();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.delegator = reader.string();
          break;
        case 2:
          message.srcValidator = reader.string();
          break;
        case 3:
          message.dstValidator = reader.string();
          break;
        case 4:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgRedelegate>): MsgRedelegate {
    const message = createBaseMsgRedelegate();
    message.delegator = object.delegator ?? "";
    message.srcValidator = object.srcValidator ?? "";
    message.dstValidator = object.dstValidator ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgRedelegateResponse(): MsgRedelegateResponse {
  return {};
}
/**
 * @name MsgRedelegateResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgRedelegateResponse
 */
export const MsgRedelegateResponse = {
  typeUrl: "/zerone.staking.v1.MsgRedelegateResponse",
  encode(_: MsgRedelegateResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRedelegateResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRedelegateResponse();
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
  fromPartial(_: DeepPartial<MsgRedelegateResponse>): MsgRedelegateResponse {
    const message = createBaseMsgRedelegateResponse();
    return message;
  }
};
function createBaseMsgUpdateValidatorStake(): MsgUpdateValidatorStake {
  return {
    operator: "",
    amount: "",
    increase: false
  };
}
/**
 * MsgUpdateValidatorStake increases or decreases self-delegation.
 * @name MsgUpdateValidatorStake
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUpdateValidatorStake
 */
export const MsgUpdateValidatorStake = {
  typeUrl: "/zerone.staking.v1.MsgUpdateValidatorStake",
  encode(message: MsgUpdateValidatorStake, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.operator !== "") {
      writer.uint32(10).string(message.operator);
    }
    if (message.amount !== "") {
      writer.uint32(18).string(message.amount);
    }
    if (message.increase === true) {
      writer.uint32(24).bool(message.increase);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateValidatorStake {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateValidatorStake();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.operator = reader.string();
          break;
        case 2:
          message.amount = reader.string();
          break;
        case 3:
          message.increase = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgUpdateValidatorStake>): MsgUpdateValidatorStake {
    const message = createBaseMsgUpdateValidatorStake();
    message.operator = object.operator ?? "";
    message.amount = object.amount ?? "";
    message.increase = object.increase ?? false;
    return message;
  }
};
function createBaseMsgUpdateValidatorStakeResponse(): MsgUpdateValidatorStakeResponse {
  return {};
}
/**
 * @name MsgUpdateValidatorStakeResponse
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUpdateValidatorStakeResponse
 */
export const MsgUpdateValidatorStakeResponse = {
  typeUrl: "/zerone.staking.v1.MsgUpdateValidatorStakeResponse",
  encode(_: MsgUpdateValidatorStakeResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateValidatorStakeResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateValidatorStakeResponse();
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
  fromPartial(_: DeepPartial<MsgUpdateValidatorStakeResponse>): MsgUpdateValidatorStakeResponse {
    const message = createBaseMsgUpdateValidatorStakeResponse();
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
 * MsgUpdateParams updates module parameters (governance-gated).
 * @name MsgUpdateParams
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUpdateParams
 */
export const MsgUpdateParams = {
  typeUrl: "/zerone.staking.v1.MsgUpdateParams",
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
 * @package zerone.staking.v1
 * @see proto type: zerone.staking.v1.MsgUpdateParamsResponse
 */
export const MsgUpdateParamsResponse = {
  typeUrl: "/zerone.staking.v1.MsgUpdateParamsResponse",
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
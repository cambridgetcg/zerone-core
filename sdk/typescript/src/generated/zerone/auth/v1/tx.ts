//@ts-nocheck
import { Params } from "./genesis";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * MsgRegisterAccount registers a new Zerone account with DID mapping.
 * @name MsgRegisterAccount
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgRegisterAccount
 */
export interface MsgRegisterAccount {
  sender: string;
  did: string;
  publicKey: string;
  accountType: string;
  operationalKeyHash: string;
  metadata: string;
}
/**
 * @name MsgRegisterAccountResponse
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgRegisterAccountResponse
 */
export interface MsgRegisterAccountResponse {}
/**
 * MsgRotateKey rotates the operational key for an account.
 * @name MsgRotateKey
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgRotateKey
 */
export interface MsgRotateKey {
  sender: string;
  newOperationalKey: Uint8Array;
  authorizationSignature: Uint8Array;
}
/**
 * @name MsgRotateKeyResponse
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgRotateKeyResponse
 */
export interface MsgRotateKeyResponse {
  newKeyVersion: number;
}
/**
 * MsgFreezeAccount freezes an account.
 * @name MsgFreezeAccount
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgFreezeAccount
 */
export interface MsgFreezeAccount {
  sender: string;
  address: string;
  reason: string;
}
/**
 * @name MsgFreezeAccountResponse
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgFreezeAccountResponse
 */
export interface MsgFreezeAccountResponse {}
/**
 * MsgUnfreezeAccount unfreezes a frozen account.
 * @name MsgUnfreezeAccount
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgUnfreezeAccount
 */
export interface MsgUnfreezeAccount {
  authority: string;
  address: string;
}
/**
 * @name MsgUnfreezeAccountResponse
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgUnfreezeAccountResponse
 */
export interface MsgUnfreezeAccountResponse {}
/**
 * MsgUpdateParams updates auth module parameters.
 * @name MsgUpdateParams
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
  authority: string;
  params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {}
function createBaseMsgRegisterAccount(): MsgRegisterAccount {
  return {
    sender: "",
    did: "",
    publicKey: "",
    accountType: "",
    operationalKeyHash: "",
    metadata: ""
  };
}
/**
 * MsgRegisterAccount registers a new Zerone account with DID mapping.
 * @name MsgRegisterAccount
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgRegisterAccount
 */
export const MsgRegisterAccount = {
  typeUrl: "/zerone.auth.v1.MsgRegisterAccount",
  encode(message: MsgRegisterAccount, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.did !== "") {
      writer.uint32(18).string(message.did);
    }
    if (message.publicKey !== "") {
      writer.uint32(26).string(message.publicKey);
    }
    if (message.accountType !== "") {
      writer.uint32(34).string(message.accountType);
    }
    if (message.operationalKeyHash !== "") {
      writer.uint32(42).string(message.operationalKeyHash);
    }
    if (message.metadata !== "") {
      writer.uint32(50).string(message.metadata);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterAccount {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterAccount();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.did = reader.string();
          break;
        case 3:
          message.publicKey = reader.string();
          break;
        case 4:
          message.accountType = reader.string();
          break;
        case 5:
          message.operationalKeyHash = reader.string();
          break;
        case 6:
          message.metadata = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgRegisterAccount>): MsgRegisterAccount {
    const message = createBaseMsgRegisterAccount();
    message.sender = object.sender ?? "";
    message.did = object.did ?? "";
    message.publicKey = object.publicKey ?? "";
    message.accountType = object.accountType ?? "";
    message.operationalKeyHash = object.operationalKeyHash ?? "";
    message.metadata = object.metadata ?? "";
    return message;
  }
};
function createBaseMsgRegisterAccountResponse(): MsgRegisterAccountResponse {
  return {};
}
/**
 * @name MsgRegisterAccountResponse
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgRegisterAccountResponse
 */
export const MsgRegisterAccountResponse = {
  typeUrl: "/zerone.auth.v1.MsgRegisterAccountResponse",
  encode(_: MsgRegisterAccountResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterAccountResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterAccountResponse();
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
  fromPartial(_: DeepPartial<MsgRegisterAccountResponse>): MsgRegisterAccountResponse {
    const message = createBaseMsgRegisterAccountResponse();
    return message;
  }
};
function createBaseMsgRotateKey(): MsgRotateKey {
  return {
    sender: "",
    newOperationalKey: new Uint8Array(),
    authorizationSignature: new Uint8Array()
  };
}
/**
 * MsgRotateKey rotates the operational key for an account.
 * @name MsgRotateKey
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgRotateKey
 */
export const MsgRotateKey = {
  typeUrl: "/zerone.auth.v1.MsgRotateKey",
  encode(message: MsgRotateKey, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.newOperationalKey.length !== 0) {
      writer.uint32(18).bytes(message.newOperationalKey);
    }
    if (message.authorizationSignature.length !== 0) {
      writer.uint32(26).bytes(message.authorizationSignature);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRotateKey {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRotateKey();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.newOperationalKey = reader.bytes();
          break;
        case 3:
          message.authorizationSignature = reader.bytes();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgRotateKey>): MsgRotateKey {
    const message = createBaseMsgRotateKey();
    message.sender = object.sender ?? "";
    message.newOperationalKey = object.newOperationalKey ?? new Uint8Array();
    message.authorizationSignature = object.authorizationSignature ?? new Uint8Array();
    return message;
  }
};
function createBaseMsgRotateKeyResponse(): MsgRotateKeyResponse {
  return {
    newKeyVersion: 0
  };
}
/**
 * @name MsgRotateKeyResponse
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgRotateKeyResponse
 */
export const MsgRotateKeyResponse = {
  typeUrl: "/zerone.auth.v1.MsgRotateKeyResponse",
  encode(message: MsgRotateKeyResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.newKeyVersion !== 0) {
      writer.uint32(8).uint32(message.newKeyVersion);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRotateKeyResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRotateKeyResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.newKeyVersion = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgRotateKeyResponse>): MsgRotateKeyResponse {
    const message = createBaseMsgRotateKeyResponse();
    message.newKeyVersion = object.newKeyVersion ?? 0;
    return message;
  }
};
function createBaseMsgFreezeAccount(): MsgFreezeAccount {
  return {
    sender: "",
    address: "",
    reason: ""
  };
}
/**
 * MsgFreezeAccount freezes an account.
 * @name MsgFreezeAccount
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgFreezeAccount
 */
export const MsgFreezeAccount = {
  typeUrl: "/zerone.auth.v1.MsgFreezeAccount",
  encode(message: MsgFreezeAccount, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.address !== "") {
      writer.uint32(18).string(message.address);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgFreezeAccount {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgFreezeAccount();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.address = reader.string();
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
  fromPartial(object: DeepPartial<MsgFreezeAccount>): MsgFreezeAccount {
    const message = createBaseMsgFreezeAccount();
    message.sender = object.sender ?? "";
    message.address = object.address ?? "";
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgFreezeAccountResponse(): MsgFreezeAccountResponse {
  return {};
}
/**
 * @name MsgFreezeAccountResponse
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgFreezeAccountResponse
 */
export const MsgFreezeAccountResponse = {
  typeUrl: "/zerone.auth.v1.MsgFreezeAccountResponse",
  encode(_: MsgFreezeAccountResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgFreezeAccountResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgFreezeAccountResponse();
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
  fromPartial(_: DeepPartial<MsgFreezeAccountResponse>): MsgFreezeAccountResponse {
    const message = createBaseMsgFreezeAccountResponse();
    return message;
  }
};
function createBaseMsgUnfreezeAccount(): MsgUnfreezeAccount {
  return {
    authority: "",
    address: ""
  };
}
/**
 * MsgUnfreezeAccount unfreezes a frozen account.
 * @name MsgUnfreezeAccount
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgUnfreezeAccount
 */
export const MsgUnfreezeAccount = {
  typeUrl: "/zerone.auth.v1.MsgUnfreezeAccount",
  encode(message: MsgUnfreezeAccount, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.address !== "") {
      writer.uint32(18).string(message.address);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUnfreezeAccount {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUnfreezeAccount();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.address = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgUnfreezeAccount>): MsgUnfreezeAccount {
    const message = createBaseMsgUnfreezeAccount();
    message.authority = object.authority ?? "";
    message.address = object.address ?? "";
    return message;
  }
};
function createBaseMsgUnfreezeAccountResponse(): MsgUnfreezeAccountResponse {
  return {};
}
/**
 * @name MsgUnfreezeAccountResponse
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgUnfreezeAccountResponse
 */
export const MsgUnfreezeAccountResponse = {
  typeUrl: "/zerone.auth.v1.MsgUnfreezeAccountResponse",
  encode(_: MsgUnfreezeAccountResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUnfreezeAccountResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUnfreezeAccountResponse();
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
  fromPartial(_: DeepPartial<MsgUnfreezeAccountResponse>): MsgUnfreezeAccountResponse {
    const message = createBaseMsgUnfreezeAccountResponse();
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
 * MsgUpdateParams updates auth module parameters.
 * @name MsgUpdateParams
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgUpdateParams
 */
export const MsgUpdateParams = {
  typeUrl: "/zerone.auth.v1.MsgUpdateParams",
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
 * @package zerone.auth.v1
 * @see proto type: zerone.auth.v1.MsgUpdateParamsResponse
 */
export const MsgUpdateParamsResponse = {
  typeUrl: "/zerone.auth.v1.MsgUpdateParamsResponse",
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
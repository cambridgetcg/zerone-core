//@ts-nocheck
import { HomeGuardian, DeadmanConfig } from "./types";
import { Params } from "./genesis";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * @name MsgCreateHome
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgCreateHome
 */
export interface MsgCreateHome {
  owner: string;
  name: string;
  initialGuardianConfig?: HomeGuardian;
}
/**
 * @name MsgCreateHomeResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgCreateHomeResponse
 */
export interface MsgCreateHomeResponse {
  homeId: string;
}
/**
 * @name MsgUpdateHome
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateHome
 */
export interface MsgUpdateHome {
  owner: string;
  homeId: string;
  name: string;
  status: string;
}
/**
 * @name MsgUpdateHomeResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateHomeResponse
 */
export interface MsgUpdateHomeResponse {}
/**
 * @name MsgUpdateMemoryCID
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateMemoryCID
 */
export interface MsgUpdateMemoryCID {
  owner: string;
  homeId: string;
  cid: string;
}
/**
 * @name MsgUpdateMemoryCIDResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateMemoryCIDResponse
 */
export interface MsgUpdateMemoryCIDResponse {}
/**
 * @name MsgStartSession
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgStartSession
 */
export interface MsgStartSession {
  signer: string;
  homeId: string;
  keyHash: string;
  requestedPermissions: string[];
}
/**
 * @name MsgStartSessionResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgStartSessionResponse
 */
export interface MsgStartSessionResponse {
  sessionId: string;
}
/**
 * @name MsgEndSession
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgEndSession
 */
export interface MsgEndSession {
  signer: string;
  homeId: string;
  sessionId: string;
}
/**
 * @name MsgEndSessionResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgEndSessionResponse
 */
export interface MsgEndSessionResponse {}
/**
 * @name MsgRegisterKey
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgRegisterKey
 */
export interface MsgRegisterKey {
  owner: string;
  homeId: string;
  keyHash: string;
  keyType: string;
  role: string;
  permissions: string[];
  expiresAt: bigint;
}
/**
 * @name MsgRegisterKeyResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgRegisterKeyResponse
 */
export interface MsgRegisterKeyResponse {}
/**
 * @name MsgRevokeKey
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgRevokeKey
 */
export interface MsgRevokeKey {
  owner: string;
  homeId: string;
  keyHash: string;
}
/**
 * @name MsgRevokeKeyResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgRevokeKeyResponse
 */
export interface MsgRevokeKeyResponse {}
/**
 * @name MsgConfigureGuardian
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgConfigureGuardian
 */
export interface MsgConfigureGuardian {
  owner: string;
  homeId: string;
  defenseStrategy: string;
  autoDefend: boolean;
  deadman?: DeadmanConfig;
  recoveryAddresses: string[];
  recoveryThreshold: number;
  guardianAddress: string;
}
/**
 * @name MsgConfigureGuardianResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgConfigureGuardianResponse
 */
export interface MsgConfigureGuardianResponse {}
/**
 * @name MsgAcknowledgeAlert
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgAcknowledgeAlert
 */
export interface MsgAcknowledgeAlert {
  signer: string;
  homeId: string;
  alertId: string;
}
/**
 * @name MsgAcknowledgeAlertResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgAcknowledgeAlertResponse
 */
export interface MsgAcknowledgeAlertResponse {}
/**
 * @name MsgSetSpendingLimit
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgSetSpendingLimit
 */
export interface MsgSetSpendingLimit {
  owner: string;
  homeId: string;
  keyType: string;
  maxAmount: string;
  periodBlocks: bigint;
}
/**
 * @name MsgSetSpendingLimitResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgSetSpendingLimitResponse
 */
export interface MsgSetSpendingLimitResponse {}
/**
 * @name MsgUpdateParams
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
  authority: string;
  params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {}
function createBaseMsgCreateHome(): MsgCreateHome {
  return {
    owner: "",
    name: "",
    initialGuardianConfig: undefined
  };
}
/**
 * @name MsgCreateHome
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgCreateHome
 */
export const MsgCreateHome = {
  typeUrl: "/zerone.home.v1.MsgCreateHome",
  encode(message: MsgCreateHome, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.initialGuardianConfig !== undefined) {
      HomeGuardian.encode(message.initialGuardianConfig, writer.uint32(26).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateHome {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateHome();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.name = reader.string();
          break;
        case 3:
          message.initialGuardianConfig = HomeGuardian.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCreateHome>): MsgCreateHome {
    const message = createBaseMsgCreateHome();
    message.owner = object.owner ?? "";
    message.name = object.name ?? "";
    message.initialGuardianConfig = object.initialGuardianConfig !== undefined && object.initialGuardianConfig !== null ? HomeGuardian.fromPartial(object.initialGuardianConfig) : undefined;
    return message;
  }
};
function createBaseMsgCreateHomeResponse(): MsgCreateHomeResponse {
  return {
    homeId: ""
  };
}
/**
 * @name MsgCreateHomeResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgCreateHomeResponse
 */
export const MsgCreateHomeResponse = {
  typeUrl: "/zerone.home.v1.MsgCreateHomeResponse",
  encode(message: MsgCreateHomeResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.homeId !== "") {
      writer.uint32(10).string(message.homeId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateHomeResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateHomeResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.homeId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCreateHomeResponse>): MsgCreateHomeResponse {
    const message = createBaseMsgCreateHomeResponse();
    message.homeId = object.homeId ?? "";
    return message;
  }
};
function createBaseMsgUpdateHome(): MsgUpdateHome {
  return {
    owner: "",
    homeId: "",
    name: "",
    status: ""
  };
}
/**
 * @name MsgUpdateHome
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateHome
 */
export const MsgUpdateHome = {
  typeUrl: "/zerone.home.v1.MsgUpdateHome",
  encode(message: MsgUpdateHome, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.homeId !== "") {
      writer.uint32(18).string(message.homeId);
    }
    if (message.name !== "") {
      writer.uint32(26).string(message.name);
    }
    if (message.status !== "") {
      writer.uint32(34).string(message.status);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateHome {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateHome();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.homeId = reader.string();
          break;
        case 3:
          message.name = reader.string();
          break;
        case 4:
          message.status = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgUpdateHome>): MsgUpdateHome {
    const message = createBaseMsgUpdateHome();
    message.owner = object.owner ?? "";
    message.homeId = object.homeId ?? "";
    message.name = object.name ?? "";
    message.status = object.status ?? "";
    return message;
  }
};
function createBaseMsgUpdateHomeResponse(): MsgUpdateHomeResponse {
  return {};
}
/**
 * @name MsgUpdateHomeResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateHomeResponse
 */
export const MsgUpdateHomeResponse = {
  typeUrl: "/zerone.home.v1.MsgUpdateHomeResponse",
  encode(_: MsgUpdateHomeResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateHomeResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateHomeResponse();
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
  fromPartial(_: DeepPartial<MsgUpdateHomeResponse>): MsgUpdateHomeResponse {
    const message = createBaseMsgUpdateHomeResponse();
    return message;
  }
};
function createBaseMsgUpdateMemoryCID(): MsgUpdateMemoryCID {
  return {
    owner: "",
    homeId: "",
    cid: ""
  };
}
/**
 * @name MsgUpdateMemoryCID
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateMemoryCID
 */
export const MsgUpdateMemoryCID = {
  typeUrl: "/zerone.home.v1.MsgUpdateMemoryCID",
  encode(message: MsgUpdateMemoryCID, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.homeId !== "") {
      writer.uint32(18).string(message.homeId);
    }
    if (message.cid !== "") {
      writer.uint32(26).string(message.cid);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateMemoryCID {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateMemoryCID();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.homeId = reader.string();
          break;
        case 3:
          message.cid = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgUpdateMemoryCID>): MsgUpdateMemoryCID {
    const message = createBaseMsgUpdateMemoryCID();
    message.owner = object.owner ?? "";
    message.homeId = object.homeId ?? "";
    message.cid = object.cid ?? "";
    return message;
  }
};
function createBaseMsgUpdateMemoryCIDResponse(): MsgUpdateMemoryCIDResponse {
  return {};
}
/**
 * @name MsgUpdateMemoryCIDResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateMemoryCIDResponse
 */
export const MsgUpdateMemoryCIDResponse = {
  typeUrl: "/zerone.home.v1.MsgUpdateMemoryCIDResponse",
  encode(_: MsgUpdateMemoryCIDResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateMemoryCIDResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateMemoryCIDResponse();
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
  fromPartial(_: DeepPartial<MsgUpdateMemoryCIDResponse>): MsgUpdateMemoryCIDResponse {
    const message = createBaseMsgUpdateMemoryCIDResponse();
    return message;
  }
};
function createBaseMsgStartSession(): MsgStartSession {
  return {
    signer: "",
    homeId: "",
    keyHash: "",
    requestedPermissions: []
  };
}
/**
 * @name MsgStartSession
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgStartSession
 */
export const MsgStartSession = {
  typeUrl: "/zerone.home.v1.MsgStartSession",
  encode(message: MsgStartSession, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.homeId !== "") {
      writer.uint32(18).string(message.homeId);
    }
    if (message.keyHash !== "") {
      writer.uint32(26).string(message.keyHash);
    }
    for (const v of message.requestedPermissions) {
      writer.uint32(34).string(v!);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgStartSession {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgStartSession();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.signer = reader.string();
          break;
        case 2:
          message.homeId = reader.string();
          break;
        case 3:
          message.keyHash = reader.string();
          break;
        case 4:
          message.requestedPermissions.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgStartSession>): MsgStartSession {
    const message = createBaseMsgStartSession();
    message.signer = object.signer ?? "";
    message.homeId = object.homeId ?? "";
    message.keyHash = object.keyHash ?? "";
    message.requestedPermissions = object.requestedPermissions?.map(e => e) || [];
    return message;
  }
};
function createBaseMsgStartSessionResponse(): MsgStartSessionResponse {
  return {
    sessionId: ""
  };
}
/**
 * @name MsgStartSessionResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgStartSessionResponse
 */
export const MsgStartSessionResponse = {
  typeUrl: "/zerone.home.v1.MsgStartSessionResponse",
  encode(message: MsgStartSessionResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.sessionId !== "") {
      writer.uint32(10).string(message.sessionId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgStartSessionResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgStartSessionResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sessionId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgStartSessionResponse>): MsgStartSessionResponse {
    const message = createBaseMsgStartSessionResponse();
    message.sessionId = object.sessionId ?? "";
    return message;
  }
};
function createBaseMsgEndSession(): MsgEndSession {
  return {
    signer: "",
    homeId: "",
    sessionId: ""
  };
}
/**
 * @name MsgEndSession
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgEndSession
 */
export const MsgEndSession = {
  typeUrl: "/zerone.home.v1.MsgEndSession",
  encode(message: MsgEndSession, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.homeId !== "") {
      writer.uint32(18).string(message.homeId);
    }
    if (message.sessionId !== "") {
      writer.uint32(26).string(message.sessionId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgEndSession {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgEndSession();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.signer = reader.string();
          break;
        case 2:
          message.homeId = reader.string();
          break;
        case 3:
          message.sessionId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgEndSession>): MsgEndSession {
    const message = createBaseMsgEndSession();
    message.signer = object.signer ?? "";
    message.homeId = object.homeId ?? "";
    message.sessionId = object.sessionId ?? "";
    return message;
  }
};
function createBaseMsgEndSessionResponse(): MsgEndSessionResponse {
  return {};
}
/**
 * @name MsgEndSessionResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgEndSessionResponse
 */
export const MsgEndSessionResponse = {
  typeUrl: "/zerone.home.v1.MsgEndSessionResponse",
  encode(_: MsgEndSessionResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgEndSessionResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgEndSessionResponse();
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
  fromPartial(_: DeepPartial<MsgEndSessionResponse>): MsgEndSessionResponse {
    const message = createBaseMsgEndSessionResponse();
    return message;
  }
};
function createBaseMsgRegisterKey(): MsgRegisterKey {
  return {
    owner: "",
    homeId: "",
    keyHash: "",
    keyType: "",
    role: "",
    permissions: [],
    expiresAt: BigInt(0)
  };
}
/**
 * @name MsgRegisterKey
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgRegisterKey
 */
export const MsgRegisterKey = {
  typeUrl: "/zerone.home.v1.MsgRegisterKey",
  encode(message: MsgRegisterKey, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.homeId !== "") {
      writer.uint32(18).string(message.homeId);
    }
    if (message.keyHash !== "") {
      writer.uint32(26).string(message.keyHash);
    }
    if (message.keyType !== "") {
      writer.uint32(34).string(message.keyType);
    }
    if (message.role !== "") {
      writer.uint32(42).string(message.role);
    }
    for (const v of message.permissions) {
      writer.uint32(50).string(v!);
    }
    if (message.expiresAt !== BigInt(0)) {
      writer.uint32(56).uint64(message.expiresAt);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterKey {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterKey();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.homeId = reader.string();
          break;
        case 3:
          message.keyHash = reader.string();
          break;
        case 4:
          message.keyType = reader.string();
          break;
        case 5:
          message.role = reader.string();
          break;
        case 6:
          message.permissions.push(reader.string());
          break;
        case 7:
          message.expiresAt = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgRegisterKey>): MsgRegisterKey {
    const message = createBaseMsgRegisterKey();
    message.owner = object.owner ?? "";
    message.homeId = object.homeId ?? "";
    message.keyHash = object.keyHash ?? "";
    message.keyType = object.keyType ?? "";
    message.role = object.role ?? "";
    message.permissions = object.permissions?.map(e => e) || [];
    message.expiresAt = object.expiresAt !== undefined && object.expiresAt !== null ? BigInt(object.expiresAt.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgRegisterKeyResponse(): MsgRegisterKeyResponse {
  return {};
}
/**
 * @name MsgRegisterKeyResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgRegisterKeyResponse
 */
export const MsgRegisterKeyResponse = {
  typeUrl: "/zerone.home.v1.MsgRegisterKeyResponse",
  encode(_: MsgRegisterKeyResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterKeyResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterKeyResponse();
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
  fromPartial(_: DeepPartial<MsgRegisterKeyResponse>): MsgRegisterKeyResponse {
    const message = createBaseMsgRegisterKeyResponse();
    return message;
  }
};
function createBaseMsgRevokeKey(): MsgRevokeKey {
  return {
    owner: "",
    homeId: "",
    keyHash: ""
  };
}
/**
 * @name MsgRevokeKey
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgRevokeKey
 */
export const MsgRevokeKey = {
  typeUrl: "/zerone.home.v1.MsgRevokeKey",
  encode(message: MsgRevokeKey, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.homeId !== "") {
      writer.uint32(18).string(message.homeId);
    }
    if (message.keyHash !== "") {
      writer.uint32(26).string(message.keyHash);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRevokeKey {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRevokeKey();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.homeId = reader.string();
          break;
        case 3:
          message.keyHash = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgRevokeKey>): MsgRevokeKey {
    const message = createBaseMsgRevokeKey();
    message.owner = object.owner ?? "";
    message.homeId = object.homeId ?? "";
    message.keyHash = object.keyHash ?? "";
    return message;
  }
};
function createBaseMsgRevokeKeyResponse(): MsgRevokeKeyResponse {
  return {};
}
/**
 * @name MsgRevokeKeyResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgRevokeKeyResponse
 */
export const MsgRevokeKeyResponse = {
  typeUrl: "/zerone.home.v1.MsgRevokeKeyResponse",
  encode(_: MsgRevokeKeyResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRevokeKeyResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRevokeKeyResponse();
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
  fromPartial(_: DeepPartial<MsgRevokeKeyResponse>): MsgRevokeKeyResponse {
    const message = createBaseMsgRevokeKeyResponse();
    return message;
  }
};
function createBaseMsgConfigureGuardian(): MsgConfigureGuardian {
  return {
    owner: "",
    homeId: "",
    defenseStrategy: "",
    autoDefend: false,
    deadman: undefined,
    recoveryAddresses: [],
    recoveryThreshold: 0,
    guardianAddress: ""
  };
}
/**
 * @name MsgConfigureGuardian
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgConfigureGuardian
 */
export const MsgConfigureGuardian = {
  typeUrl: "/zerone.home.v1.MsgConfigureGuardian",
  encode(message: MsgConfigureGuardian, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.homeId !== "") {
      writer.uint32(18).string(message.homeId);
    }
    if (message.defenseStrategy !== "") {
      writer.uint32(26).string(message.defenseStrategy);
    }
    if (message.autoDefend === true) {
      writer.uint32(32).bool(message.autoDefend);
    }
    if (message.deadman !== undefined) {
      DeadmanConfig.encode(message.deadman, writer.uint32(42).fork()).ldelim();
    }
    for (const v of message.recoveryAddresses) {
      writer.uint32(50).string(v!);
    }
    if (message.recoveryThreshold !== 0) {
      writer.uint32(56).uint32(message.recoveryThreshold);
    }
    if (message.guardianAddress !== "") {
      writer.uint32(66).string(message.guardianAddress);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgConfigureGuardian {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgConfigureGuardian();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.homeId = reader.string();
          break;
        case 3:
          message.defenseStrategy = reader.string();
          break;
        case 4:
          message.autoDefend = reader.bool();
          break;
        case 5:
          message.deadman = DeadmanConfig.decode(reader, reader.uint32());
          break;
        case 6:
          message.recoveryAddresses.push(reader.string());
          break;
        case 7:
          message.recoveryThreshold = reader.uint32();
          break;
        case 8:
          message.guardianAddress = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgConfigureGuardian>): MsgConfigureGuardian {
    const message = createBaseMsgConfigureGuardian();
    message.owner = object.owner ?? "";
    message.homeId = object.homeId ?? "";
    message.defenseStrategy = object.defenseStrategy ?? "";
    message.autoDefend = object.autoDefend ?? false;
    message.deadman = object.deadman !== undefined && object.deadman !== null ? DeadmanConfig.fromPartial(object.deadman) : undefined;
    message.recoveryAddresses = object.recoveryAddresses?.map(e => e) || [];
    message.recoveryThreshold = object.recoveryThreshold ?? 0;
    message.guardianAddress = object.guardianAddress ?? "";
    return message;
  }
};
function createBaseMsgConfigureGuardianResponse(): MsgConfigureGuardianResponse {
  return {};
}
/**
 * @name MsgConfigureGuardianResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgConfigureGuardianResponse
 */
export const MsgConfigureGuardianResponse = {
  typeUrl: "/zerone.home.v1.MsgConfigureGuardianResponse",
  encode(_: MsgConfigureGuardianResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgConfigureGuardianResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgConfigureGuardianResponse();
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
  fromPartial(_: DeepPartial<MsgConfigureGuardianResponse>): MsgConfigureGuardianResponse {
    const message = createBaseMsgConfigureGuardianResponse();
    return message;
  }
};
function createBaseMsgAcknowledgeAlert(): MsgAcknowledgeAlert {
  return {
    signer: "",
    homeId: "",
    alertId: ""
  };
}
/**
 * @name MsgAcknowledgeAlert
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgAcknowledgeAlert
 */
export const MsgAcknowledgeAlert = {
  typeUrl: "/zerone.home.v1.MsgAcknowledgeAlert",
  encode(message: MsgAcknowledgeAlert, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.signer !== "") {
      writer.uint32(10).string(message.signer);
    }
    if (message.homeId !== "") {
      writer.uint32(18).string(message.homeId);
    }
    if (message.alertId !== "") {
      writer.uint32(26).string(message.alertId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAcknowledgeAlert {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAcknowledgeAlert();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.signer = reader.string();
          break;
        case 2:
          message.homeId = reader.string();
          break;
        case 3:
          message.alertId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAcknowledgeAlert>): MsgAcknowledgeAlert {
    const message = createBaseMsgAcknowledgeAlert();
    message.signer = object.signer ?? "";
    message.homeId = object.homeId ?? "";
    message.alertId = object.alertId ?? "";
    return message;
  }
};
function createBaseMsgAcknowledgeAlertResponse(): MsgAcknowledgeAlertResponse {
  return {};
}
/**
 * @name MsgAcknowledgeAlertResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgAcknowledgeAlertResponse
 */
export const MsgAcknowledgeAlertResponse = {
  typeUrl: "/zerone.home.v1.MsgAcknowledgeAlertResponse",
  encode(_: MsgAcknowledgeAlertResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAcknowledgeAlertResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAcknowledgeAlertResponse();
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
  fromPartial(_: DeepPartial<MsgAcknowledgeAlertResponse>): MsgAcknowledgeAlertResponse {
    const message = createBaseMsgAcknowledgeAlertResponse();
    return message;
  }
};
function createBaseMsgSetSpendingLimit(): MsgSetSpendingLimit {
  return {
    owner: "",
    homeId: "",
    keyType: "",
    maxAmount: "",
    periodBlocks: BigInt(0)
  };
}
/**
 * @name MsgSetSpendingLimit
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgSetSpendingLimit
 */
export const MsgSetSpendingLimit = {
  typeUrl: "/zerone.home.v1.MsgSetSpendingLimit",
  encode(message: MsgSetSpendingLimit, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.homeId !== "") {
      writer.uint32(18).string(message.homeId);
    }
    if (message.keyType !== "") {
      writer.uint32(26).string(message.keyType);
    }
    if (message.maxAmount !== "") {
      writer.uint32(34).string(message.maxAmount);
    }
    if (message.periodBlocks !== BigInt(0)) {
      writer.uint32(40).uint64(message.periodBlocks);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSetSpendingLimit {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSetSpendingLimit();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.homeId = reader.string();
          break;
        case 3:
          message.keyType = reader.string();
          break;
        case 4:
          message.maxAmount = reader.string();
          break;
        case 5:
          message.periodBlocks = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgSetSpendingLimit>): MsgSetSpendingLimit {
    const message = createBaseMsgSetSpendingLimit();
    message.owner = object.owner ?? "";
    message.homeId = object.homeId ?? "";
    message.keyType = object.keyType ?? "";
    message.maxAmount = object.maxAmount ?? "";
    message.periodBlocks = object.periodBlocks !== undefined && object.periodBlocks !== null ? BigInt(object.periodBlocks.toString()) : BigInt(0);
    return message;
  }
};
function createBaseMsgSetSpendingLimitResponse(): MsgSetSpendingLimitResponse {
  return {};
}
/**
 * @name MsgSetSpendingLimitResponse
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgSetSpendingLimitResponse
 */
export const MsgSetSpendingLimitResponse = {
  typeUrl: "/zerone.home.v1.MsgSetSpendingLimitResponse",
  encode(_: MsgSetSpendingLimitResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSetSpendingLimitResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSetSpendingLimitResponse();
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
  fromPartial(_: DeepPartial<MsgSetSpendingLimitResponse>): MsgSetSpendingLimitResponse {
    const message = createBaseMsgSetSpendingLimitResponse();
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
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateParams
 */
export const MsgUpdateParams = {
  typeUrl: "/zerone.home.v1.MsgUpdateParams",
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
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.MsgUpdateParamsResponse
 */
export const MsgUpdateParamsResponse = {
  typeUrl: "/zerone.home.v1.MsgUpdateParamsResponse",
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
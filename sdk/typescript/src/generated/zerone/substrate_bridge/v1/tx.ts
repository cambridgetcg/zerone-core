//@ts-nocheck
import { AdapterRegistration } from "./adapter";
import { SubstrateLink } from "./substrate_link";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * @name MsgRegisterAdapter
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgRegisterAdapter
 */
export interface MsgRegisterAdapter {
  authority: string;
  adapter?: AdapterRegistration;
}
/**
 * @name MsgRegisterAdapterResponse
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgRegisterAdapterResponse
 */
export interface MsgRegisterAdapterResponse {}
/**
 * @name MsgSuspendAdapter
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgSuspendAdapter
 */
export interface MsgSuspendAdapter {
  authority: string;
  adapterId: string;
  reason: string;
}
/**
 * @name MsgSuspendAdapterResponse
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgSuspendAdapterResponse
 */
export interface MsgSuspendAdapterResponse {}
/**
 * @name MsgTombstoneAdapter
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgTombstoneAdapter
 */
export interface MsgTombstoneAdapter {
  authority: string;
  adapterId: string;
  reason: string;
}
/**
 * @name MsgTombstoneAdapterResponse
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgTombstoneAdapterResponse
 */
export interface MsgTombstoneAdapterResponse {}
/**
 * @name MsgSubmitExternalAttestation
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgSubmitExternalAttestation
 */
export interface MsgSubmitExternalAttestation {
  submitter: string;
  adapterId: string;
  workClassId: string;
  link?: SubstrateLink;
  bondUzrn: string;
}
/**
 * @name MsgSubmitExternalAttestationResponse
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgSubmitExternalAttestationResponse
 */
export interface MsgSubmitExternalAttestationResponse {
  attestationId: string;
}
function createBaseMsgRegisterAdapter(): MsgRegisterAdapter {
  return {
    authority: "",
    adapter: undefined
  };
}
/**
 * @name MsgRegisterAdapter
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgRegisterAdapter
 */
export const MsgRegisterAdapter = {
  typeUrl: "/zerone.substrate_bridge.v1.MsgRegisterAdapter",
  encode(message: MsgRegisterAdapter, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.adapter !== undefined) {
      AdapterRegistration.encode(message.adapter, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterAdapter {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterAdapter();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.adapter = AdapterRegistration.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgRegisterAdapter>): MsgRegisterAdapter {
    const message = createBaseMsgRegisterAdapter();
    message.authority = object.authority ?? "";
    message.adapter = object.adapter !== undefined && object.adapter !== null ? AdapterRegistration.fromPartial(object.adapter) : undefined;
    return message;
  }
};
function createBaseMsgRegisterAdapterResponse(): MsgRegisterAdapterResponse {
  return {};
}
/**
 * @name MsgRegisterAdapterResponse
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgRegisterAdapterResponse
 */
export const MsgRegisterAdapterResponse = {
  typeUrl: "/zerone.substrate_bridge.v1.MsgRegisterAdapterResponse",
  encode(_: MsgRegisterAdapterResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterAdapterResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterAdapterResponse();
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
  fromPartial(_: DeepPartial<MsgRegisterAdapterResponse>): MsgRegisterAdapterResponse {
    const message = createBaseMsgRegisterAdapterResponse();
    return message;
  }
};
function createBaseMsgSuspendAdapter(): MsgSuspendAdapter {
  return {
    authority: "",
    adapterId: "",
    reason: ""
  };
}
/**
 * @name MsgSuspendAdapter
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgSuspendAdapter
 */
export const MsgSuspendAdapter = {
  typeUrl: "/zerone.substrate_bridge.v1.MsgSuspendAdapter",
  encode(message: MsgSuspendAdapter, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.adapterId !== "") {
      writer.uint32(18).string(message.adapterId);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSuspendAdapter {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSuspendAdapter();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.adapterId = reader.string();
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
  fromPartial(object: DeepPartial<MsgSuspendAdapter>): MsgSuspendAdapter {
    const message = createBaseMsgSuspendAdapter();
    message.authority = object.authority ?? "";
    message.adapterId = object.adapterId ?? "";
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgSuspendAdapterResponse(): MsgSuspendAdapterResponse {
  return {};
}
/**
 * @name MsgSuspendAdapterResponse
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgSuspendAdapterResponse
 */
export const MsgSuspendAdapterResponse = {
  typeUrl: "/zerone.substrate_bridge.v1.MsgSuspendAdapterResponse",
  encode(_: MsgSuspendAdapterResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSuspendAdapterResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSuspendAdapterResponse();
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
  fromPartial(_: DeepPartial<MsgSuspendAdapterResponse>): MsgSuspendAdapterResponse {
    const message = createBaseMsgSuspendAdapterResponse();
    return message;
  }
};
function createBaseMsgTombstoneAdapter(): MsgTombstoneAdapter {
  return {
    authority: "",
    adapterId: "",
    reason: ""
  };
}
/**
 * @name MsgTombstoneAdapter
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgTombstoneAdapter
 */
export const MsgTombstoneAdapter = {
  typeUrl: "/zerone.substrate_bridge.v1.MsgTombstoneAdapter",
  encode(message: MsgTombstoneAdapter, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.adapterId !== "") {
      writer.uint32(18).string(message.adapterId);
    }
    if (message.reason !== "") {
      writer.uint32(26).string(message.reason);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgTombstoneAdapter {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgTombstoneAdapter();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.adapterId = reader.string();
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
  fromPartial(object: DeepPartial<MsgTombstoneAdapter>): MsgTombstoneAdapter {
    const message = createBaseMsgTombstoneAdapter();
    message.authority = object.authority ?? "";
    message.adapterId = object.adapterId ?? "";
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgTombstoneAdapterResponse(): MsgTombstoneAdapterResponse {
  return {};
}
/**
 * @name MsgTombstoneAdapterResponse
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgTombstoneAdapterResponse
 */
export const MsgTombstoneAdapterResponse = {
  typeUrl: "/zerone.substrate_bridge.v1.MsgTombstoneAdapterResponse",
  encode(_: MsgTombstoneAdapterResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgTombstoneAdapterResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgTombstoneAdapterResponse();
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
  fromPartial(_: DeepPartial<MsgTombstoneAdapterResponse>): MsgTombstoneAdapterResponse {
    const message = createBaseMsgTombstoneAdapterResponse();
    return message;
  }
};
function createBaseMsgSubmitExternalAttestation(): MsgSubmitExternalAttestation {
  return {
    submitter: "",
    adapterId: "",
    workClassId: "",
    link: undefined,
    bondUzrn: ""
  };
}
/**
 * @name MsgSubmitExternalAttestation
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgSubmitExternalAttestation
 */
export const MsgSubmitExternalAttestation = {
  typeUrl: "/zerone.substrate_bridge.v1.MsgSubmitExternalAttestation",
  encode(message: MsgSubmitExternalAttestation, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.submitter !== "") {
      writer.uint32(10).string(message.submitter);
    }
    if (message.adapterId !== "") {
      writer.uint32(18).string(message.adapterId);
    }
    if (message.workClassId !== "") {
      writer.uint32(26).string(message.workClassId);
    }
    if (message.link !== undefined) {
      SubstrateLink.encode(message.link, writer.uint32(34).fork()).ldelim();
    }
    if (message.bondUzrn !== "") {
      writer.uint32(42).string(message.bondUzrn);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitExternalAttestation {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitExternalAttestation();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.submitter = reader.string();
          break;
        case 2:
          message.adapterId = reader.string();
          break;
        case 3:
          message.workClassId = reader.string();
          break;
        case 4:
          message.link = SubstrateLink.decode(reader, reader.uint32());
          break;
        case 5:
          message.bondUzrn = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgSubmitExternalAttestation>): MsgSubmitExternalAttestation {
    const message = createBaseMsgSubmitExternalAttestation();
    message.submitter = object.submitter ?? "";
    message.adapterId = object.adapterId ?? "";
    message.workClassId = object.workClassId ?? "";
    message.link = object.link !== undefined && object.link !== null ? SubstrateLink.fromPartial(object.link) : undefined;
    message.bondUzrn = object.bondUzrn ?? "";
    return message;
  }
};
function createBaseMsgSubmitExternalAttestationResponse(): MsgSubmitExternalAttestationResponse {
  return {
    attestationId: ""
  };
}
/**
 * @name MsgSubmitExternalAttestationResponse
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.MsgSubmitExternalAttestationResponse
 */
export const MsgSubmitExternalAttestationResponse = {
  typeUrl: "/zerone.substrate_bridge.v1.MsgSubmitExternalAttestationResponse",
  encode(message: MsgSubmitExternalAttestationResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.attestationId !== "") {
      writer.uint32(10).string(message.attestationId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgSubmitExternalAttestationResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgSubmitExternalAttestationResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.attestationId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgSubmitExternalAttestationResponse>): MsgSubmitExternalAttestationResponse {
    const message = createBaseMsgSubmitExternalAttestationResponse();
    message.attestationId = object.attestationId ?? "";
    return message;
  }
};
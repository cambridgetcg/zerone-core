//@ts-nocheck
import { LogicZoneProperties } from "./state";
import { Params } from "./genesis";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * @name MsgProposeDomain
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgProposeDomain
 */
export interface MsgProposeDomain {
  proposer: string;
  name: string;
  displayName: string;
  description: string;
  stratum: number;
  /**
   * bigint as string (uzrn)
   */
  stake: string;
}
/**
 * @name MsgProposeDomainResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgProposeDomainResponse
 */
export interface MsgProposeDomainResponse {
  proposalId: string;
}
/**
 * @name MsgVoteDomainProposal
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgVoteDomainProposal
 */
export interface MsgVoteDomainProposal {
  voter: string;
  proposalId: string;
  approve: boolean;
}
/**
 * @name MsgVoteDomainProposalResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgVoteDomainProposalResponse
 */
export interface MsgVoteDomainProposalResponse {}
/**
 * @name MsgUpdateDomain
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgUpdateDomain
 */
export interface MsgUpdateDomain {
  authority: string;
  domainName: string;
  displayName: string;
  description: string;
  status: string;
}
/**
 * @name MsgUpdateDomainResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgUpdateDomainResponse
 */
export interface MsgUpdateDomainResponse {}
/**
 * @name MsgRegisterLogicZone
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgRegisterLogicZone
 */
export interface MsgRegisterLogicZone {
  authority: string;
  zoneProperties?: LogicZoneProperties;
}
/**
 * @name MsgRegisterLogicZoneResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgRegisterLogicZoneResponse
 */
export interface MsgRegisterLogicZoneResponse {}
/**
 * @name MsgAcknowledgeIncompleteness
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgAcknowledgeIncompleteness
 */
export interface MsgAcknowledgeIncompleteness {
  submitter: string;
  factId: string;
  zone: string;
  reason: string;
}
/**
 * @name MsgAcknowledgeIncompletenessResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgAcknowledgeIncompletenessResponse
 */
export interface MsgAcknowledgeIncompletenessResponse {}
/**
 * @name MsgUpdateParams
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
  authority: string;
  params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {}
function createBaseMsgProposeDomain(): MsgProposeDomain {
  return {
    proposer: "",
    name: "",
    displayName: "",
    description: "",
    stratum: 0,
    stake: ""
  };
}
/**
 * @name MsgProposeDomain
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgProposeDomain
 */
export const MsgProposeDomain = {
  typeUrl: "/zerone.ontology.v1.MsgProposeDomain",
  encode(message: MsgProposeDomain, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposer !== "") {
      writer.uint32(10).string(message.proposer);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.displayName !== "") {
      writer.uint32(26).string(message.displayName);
    }
    if (message.description !== "") {
      writer.uint32(34).string(message.description);
    }
    if (message.stratum !== 0) {
      writer.uint32(40).uint32(message.stratum);
    }
    if (message.stake !== "") {
      writer.uint32(50).string(message.stake);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeDomain {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeDomain();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.proposer = reader.string();
          break;
        case 2:
          message.name = reader.string();
          break;
        case 3:
          message.displayName = reader.string();
          break;
        case 4:
          message.description = reader.string();
          break;
        case 5:
          message.stratum = reader.uint32();
          break;
        case 6:
          message.stake = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgProposeDomain>): MsgProposeDomain {
    const message = createBaseMsgProposeDomain();
    message.proposer = object.proposer ?? "";
    message.name = object.name ?? "";
    message.displayName = object.displayName ?? "";
    message.description = object.description ?? "";
    message.stratum = object.stratum ?? 0;
    message.stake = object.stake ?? "";
    return message;
  }
};
function createBaseMsgProposeDomainResponse(): MsgProposeDomainResponse {
  return {
    proposalId: ""
  };
}
/**
 * @name MsgProposeDomainResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgProposeDomainResponse
 */
export const MsgProposeDomainResponse = {
  typeUrl: "/zerone.ontology.v1.MsgProposeDomainResponse",
  encode(message: MsgProposeDomainResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.proposalId !== "") {
      writer.uint32(10).string(message.proposalId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgProposeDomainResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgProposeDomainResponse();
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
  fromPartial(object: DeepPartial<MsgProposeDomainResponse>): MsgProposeDomainResponse {
    const message = createBaseMsgProposeDomainResponse();
    message.proposalId = object.proposalId ?? "";
    return message;
  }
};
function createBaseMsgVoteDomainProposal(): MsgVoteDomainProposal {
  return {
    voter: "",
    proposalId: "",
    approve: false
  };
}
/**
 * @name MsgVoteDomainProposal
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgVoteDomainProposal
 */
export const MsgVoteDomainProposal = {
  typeUrl: "/zerone.ontology.v1.MsgVoteDomainProposal",
  encode(message: MsgVoteDomainProposal, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
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
  decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteDomainProposal {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteDomainProposal();
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
  fromPartial(object: DeepPartial<MsgVoteDomainProposal>): MsgVoteDomainProposal {
    const message = createBaseMsgVoteDomainProposal();
    message.voter = object.voter ?? "";
    message.proposalId = object.proposalId ?? "";
    message.approve = object.approve ?? false;
    return message;
  }
};
function createBaseMsgVoteDomainProposalResponse(): MsgVoteDomainProposalResponse {
  return {};
}
/**
 * @name MsgVoteDomainProposalResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgVoteDomainProposalResponse
 */
export const MsgVoteDomainProposalResponse = {
  typeUrl: "/zerone.ontology.v1.MsgVoteDomainProposalResponse",
  encode(_: MsgVoteDomainProposalResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgVoteDomainProposalResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgVoteDomainProposalResponse();
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
  fromPartial(_: DeepPartial<MsgVoteDomainProposalResponse>): MsgVoteDomainProposalResponse {
    const message = createBaseMsgVoteDomainProposalResponse();
    return message;
  }
};
function createBaseMsgUpdateDomain(): MsgUpdateDomain {
  return {
    authority: "",
    domainName: "",
    displayName: "",
    description: "",
    status: ""
  };
}
/**
 * @name MsgUpdateDomain
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgUpdateDomain
 */
export const MsgUpdateDomain = {
  typeUrl: "/zerone.ontology.v1.MsgUpdateDomain",
  encode(message: MsgUpdateDomain, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.domainName !== "") {
      writer.uint32(18).string(message.domainName);
    }
    if (message.displayName !== "") {
      writer.uint32(26).string(message.displayName);
    }
    if (message.description !== "") {
      writer.uint32(34).string(message.description);
    }
    if (message.status !== "") {
      writer.uint32(42).string(message.status);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateDomain {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateDomain();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.domainName = reader.string();
          break;
        case 3:
          message.displayName = reader.string();
          break;
        case 4:
          message.description = reader.string();
          break;
        case 5:
          message.status = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgUpdateDomain>): MsgUpdateDomain {
    const message = createBaseMsgUpdateDomain();
    message.authority = object.authority ?? "";
    message.domainName = object.domainName ?? "";
    message.displayName = object.displayName ?? "";
    message.description = object.description ?? "";
    message.status = object.status ?? "";
    return message;
  }
};
function createBaseMsgUpdateDomainResponse(): MsgUpdateDomainResponse {
  return {};
}
/**
 * @name MsgUpdateDomainResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgUpdateDomainResponse
 */
export const MsgUpdateDomainResponse = {
  typeUrl: "/zerone.ontology.v1.MsgUpdateDomainResponse",
  encode(_: MsgUpdateDomainResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdateDomainResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdateDomainResponse();
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
  fromPartial(_: DeepPartial<MsgUpdateDomainResponse>): MsgUpdateDomainResponse {
    const message = createBaseMsgUpdateDomainResponse();
    return message;
  }
};
function createBaseMsgRegisterLogicZone(): MsgRegisterLogicZone {
  return {
    authority: "",
    zoneProperties: undefined
  };
}
/**
 * @name MsgRegisterLogicZone
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgRegisterLogicZone
 */
export const MsgRegisterLogicZone = {
  typeUrl: "/zerone.ontology.v1.MsgRegisterLogicZone",
  encode(message: MsgRegisterLogicZone, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.zoneProperties !== undefined) {
      LogicZoneProperties.encode(message.zoneProperties, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterLogicZone {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterLogicZone();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.zoneProperties = LogicZoneProperties.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgRegisterLogicZone>): MsgRegisterLogicZone {
    const message = createBaseMsgRegisterLogicZone();
    message.authority = object.authority ?? "";
    message.zoneProperties = object.zoneProperties !== undefined && object.zoneProperties !== null ? LogicZoneProperties.fromPartial(object.zoneProperties) : undefined;
    return message;
  }
};
function createBaseMsgRegisterLogicZoneResponse(): MsgRegisterLogicZoneResponse {
  return {};
}
/**
 * @name MsgRegisterLogicZoneResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgRegisterLogicZoneResponse
 */
export const MsgRegisterLogicZoneResponse = {
  typeUrl: "/zerone.ontology.v1.MsgRegisterLogicZoneResponse",
  encode(_: MsgRegisterLogicZoneResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgRegisterLogicZoneResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgRegisterLogicZoneResponse();
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
  fromPartial(_: DeepPartial<MsgRegisterLogicZoneResponse>): MsgRegisterLogicZoneResponse {
    const message = createBaseMsgRegisterLogicZoneResponse();
    return message;
  }
};
function createBaseMsgAcknowledgeIncompleteness(): MsgAcknowledgeIncompleteness {
  return {
    submitter: "",
    factId: "",
    zone: "",
    reason: ""
  };
}
/**
 * @name MsgAcknowledgeIncompleteness
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgAcknowledgeIncompleteness
 */
export const MsgAcknowledgeIncompleteness = {
  typeUrl: "/zerone.ontology.v1.MsgAcknowledgeIncompleteness",
  encode(message: MsgAcknowledgeIncompleteness, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.submitter !== "") {
      writer.uint32(10).string(message.submitter);
    }
    if (message.factId !== "") {
      writer.uint32(18).string(message.factId);
    }
    if (message.zone !== "") {
      writer.uint32(26).string(message.zone);
    }
    if (message.reason !== "") {
      writer.uint32(34).string(message.reason);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAcknowledgeIncompleteness {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAcknowledgeIncompleteness();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.submitter = reader.string();
          break;
        case 2:
          message.factId = reader.string();
          break;
        case 3:
          message.zone = reader.string();
          break;
        case 4:
          message.reason = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAcknowledgeIncompleteness>): MsgAcknowledgeIncompleteness {
    const message = createBaseMsgAcknowledgeIncompleteness();
    message.submitter = object.submitter ?? "";
    message.factId = object.factId ?? "";
    message.zone = object.zone ?? "";
    message.reason = object.reason ?? "";
    return message;
  }
};
function createBaseMsgAcknowledgeIncompletenessResponse(): MsgAcknowledgeIncompletenessResponse {
  return {};
}
/**
 * @name MsgAcknowledgeIncompletenessResponse
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgAcknowledgeIncompletenessResponse
 */
export const MsgAcknowledgeIncompletenessResponse = {
  typeUrl: "/zerone.ontology.v1.MsgAcknowledgeIncompletenessResponse",
  encode(_: MsgAcknowledgeIncompletenessResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAcknowledgeIncompletenessResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAcknowledgeIncompletenessResponse();
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
  fromPartial(_: DeepPartial<MsgAcknowledgeIncompletenessResponse>): MsgAcknowledgeIncompletenessResponse {
    const message = createBaseMsgAcknowledgeIncompletenessResponse();
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
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgUpdateParams
 */
export const MsgUpdateParams = {
  typeUrl: "/zerone.ontology.v1.MsgUpdateParams",
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
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.MsgUpdateParamsResponse
 */
export const MsgUpdateParamsResponse = {
  typeUrl: "/zerone.ontology.v1.MsgUpdateParamsResponse",
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
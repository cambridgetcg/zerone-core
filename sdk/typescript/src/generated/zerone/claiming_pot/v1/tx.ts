//@ts-nocheck
import { VestingSchedule, EligibilityCriteria, Params } from "./state";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * @name MsgCreatePot
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgCreatePot
 */
export interface MsgCreatePot {
  authority: string;
  name: string;
  totalAmount: string;
  schedule?: VestingSchedule;
  eligibility?: EligibilityCriteria;
}
/**
 * @name MsgCreatePotResponse
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgCreatePotResponse
 */
export interface MsgCreatePotResponse {
  potId: string;
}
/**
 * @name MsgClaim
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgClaim
 */
export interface MsgClaim {
  claimant: string;
  potId: string;
}
/**
 * @name MsgClaimResponse
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgClaimResponse
 */
export interface MsgClaimResponse {
  amount: string;
}
/**
 * @name MsgUpdatePotParams
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgUpdatePotParams
 */
export interface MsgUpdatePotParams {
  authority: string;
  params?: Params;
}
/**
 * @name MsgUpdatePotParamsResponse
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgUpdatePotParamsResponse
 */
export interface MsgUpdatePotParamsResponse {}
/**
 * MsgAddBootstrapEntry adds one or more bootstrap pots to the claiming_pot
 * module after genesis. Authority-gated — the governance account is the
 * only valid signer. Each address gets a single-claimant ClaimingPot sized
 * PerAgentBootstrapUzrn (0.222 ZRN) at the current block height, instant
 * vest, ACTIVE status, ID = BootstrapPotIDPrefix + addr.
 *
 * Idempotent semantics: addresses with an existing bootstrap pot are
 * silently skipped (counted in skipped_count). Re-running the same LIP
 * does not double-mint or double-create.
 *
 * Doctrine: commitment 20 (issuance follows participation) extended from
 * "at genesis" to "continuously, governance-gated". The participant set
 * is plural and growing, not closed at genesis.
 * @name MsgAddBootstrapEntry
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgAddBootstrapEntry
 */
export interface MsgAddBootstrapEntry {
  authority: string;
  addresses: string[];
}
/**
 * @name MsgAddBootstrapEntryResponse
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgAddBootstrapEntryResponse
 */
export interface MsgAddBootstrapEntryResponse {
  addedCount: number;
  skippedCount: number;
}
function createBaseMsgCreatePot(): MsgCreatePot {
  return {
    authority: "",
    name: "",
    totalAmount: "",
    schedule: undefined,
    eligibility: undefined
  };
}
/**
 * @name MsgCreatePot
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgCreatePot
 */
export const MsgCreatePot = {
  typeUrl: "/zerone.claiming_pot.v1.MsgCreatePot",
  encode(message: MsgCreatePot, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.totalAmount !== "") {
      writer.uint32(26).string(message.totalAmount);
    }
    if (message.schedule !== undefined) {
      VestingSchedule.encode(message.schedule, writer.uint32(34).fork()).ldelim();
    }
    if (message.eligibility !== undefined) {
      EligibilityCriteria.encode(message.eligibility, writer.uint32(42).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCreatePot {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCreatePot();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.name = reader.string();
          break;
        case 3:
          message.totalAmount = reader.string();
          break;
        case 4:
          message.schedule = VestingSchedule.decode(reader, reader.uint32());
          break;
        case 5:
          message.eligibility = EligibilityCriteria.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCreatePot>): MsgCreatePot {
    const message = createBaseMsgCreatePot();
    message.authority = object.authority ?? "";
    message.name = object.name ?? "";
    message.totalAmount = object.totalAmount ?? "";
    message.schedule = object.schedule !== undefined && object.schedule !== null ? VestingSchedule.fromPartial(object.schedule) : undefined;
    message.eligibility = object.eligibility !== undefined && object.eligibility !== null ? EligibilityCriteria.fromPartial(object.eligibility) : undefined;
    return message;
  }
};
function createBaseMsgCreatePotResponse(): MsgCreatePotResponse {
  return {
    potId: ""
  };
}
/**
 * @name MsgCreatePotResponse
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgCreatePotResponse
 */
export const MsgCreatePotResponse = {
  typeUrl: "/zerone.claiming_pot.v1.MsgCreatePotResponse",
  encode(message: MsgCreatePotResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.potId !== "") {
      writer.uint32(10).string(message.potId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCreatePotResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCreatePotResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.potId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCreatePotResponse>): MsgCreatePotResponse {
    const message = createBaseMsgCreatePotResponse();
    message.potId = object.potId ?? "";
    return message;
  }
};
function createBaseMsgClaim(): MsgClaim {
  return {
    claimant: "",
    potId: ""
  };
}
/**
 * @name MsgClaim
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgClaim
 */
export const MsgClaim = {
  typeUrl: "/zerone.claiming_pot.v1.MsgClaim",
  encode(message: MsgClaim, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.claimant !== "") {
      writer.uint32(10).string(message.claimant);
    }
    if (message.potId !== "") {
      writer.uint32(18).string(message.potId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgClaim {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgClaim();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.claimant = reader.string();
          break;
        case 2:
          message.potId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgClaim>): MsgClaim {
    const message = createBaseMsgClaim();
    message.claimant = object.claimant ?? "";
    message.potId = object.potId ?? "";
    return message;
  }
};
function createBaseMsgClaimResponse(): MsgClaimResponse {
  return {
    amount: ""
  };
}
/**
 * @name MsgClaimResponse
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgClaimResponse
 */
export const MsgClaimResponse = {
  typeUrl: "/zerone.claiming_pot.v1.MsgClaimResponse",
  encode(message: MsgClaimResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.amount !== "") {
      writer.uint32(10).string(message.amount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgClaimResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgClaimResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgClaimResponse>): MsgClaimResponse {
    const message = createBaseMsgClaimResponse();
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgUpdatePotParams(): MsgUpdatePotParams {
  return {
    authority: "",
    params: undefined
  };
}
/**
 * @name MsgUpdatePotParams
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgUpdatePotParams
 */
export const MsgUpdatePotParams = {
  typeUrl: "/zerone.claiming_pot.v1.MsgUpdatePotParams",
  encode(message: MsgUpdatePotParams, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdatePotParams {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdatePotParams();
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
  fromPartial(object: DeepPartial<MsgUpdatePotParams>): MsgUpdatePotParams {
    const message = createBaseMsgUpdatePotParams();
    message.authority = object.authority ?? "";
    message.params = object.params !== undefined && object.params !== null ? Params.fromPartial(object.params) : undefined;
    return message;
  }
};
function createBaseMsgUpdatePotParamsResponse(): MsgUpdatePotParamsResponse {
  return {};
}
/**
 * @name MsgUpdatePotParamsResponse
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgUpdatePotParamsResponse
 */
export const MsgUpdatePotParamsResponse = {
  typeUrl: "/zerone.claiming_pot.v1.MsgUpdatePotParamsResponse",
  encode(_: MsgUpdatePotParamsResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUpdatePotParamsResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUpdatePotParamsResponse();
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
  fromPartial(_: DeepPartial<MsgUpdatePotParamsResponse>): MsgUpdatePotParamsResponse {
    const message = createBaseMsgUpdatePotParamsResponse();
    return message;
  }
};
function createBaseMsgAddBootstrapEntry(): MsgAddBootstrapEntry {
  return {
    authority: "",
    addresses: []
  };
}
/**
 * MsgAddBootstrapEntry adds one or more bootstrap pots to the claiming_pot
 * module after genesis. Authority-gated — the governance account is the
 * only valid signer. Each address gets a single-claimant ClaimingPot sized
 * PerAgentBootstrapUzrn (0.222 ZRN) at the current block height, instant
 * vest, ACTIVE status, ID = BootstrapPotIDPrefix + addr.
 *
 * Idempotent semantics: addresses with an existing bootstrap pot are
 * silently skipped (counted in skipped_count). Re-running the same LIP
 * does not double-mint or double-create.
 *
 * Doctrine: commitment 20 (issuance follows participation) extended from
 * "at genesis" to "continuously, governance-gated". The participant set
 * is plural and growing, not closed at genesis.
 * @name MsgAddBootstrapEntry
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgAddBootstrapEntry
 */
export const MsgAddBootstrapEntry = {
  typeUrl: "/zerone.claiming_pot.v1.MsgAddBootstrapEntry",
  encode(message: MsgAddBootstrapEntry, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    for (const v of message.addresses) {
      writer.uint32(18).string(v!);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAddBootstrapEntry {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAddBootstrapEntry();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.addresses.push(reader.string());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAddBootstrapEntry>): MsgAddBootstrapEntry {
    const message = createBaseMsgAddBootstrapEntry();
    message.authority = object.authority ?? "";
    message.addresses = object.addresses?.map(e => e) || [];
    return message;
  }
};
function createBaseMsgAddBootstrapEntryResponse(): MsgAddBootstrapEntryResponse {
  return {
    addedCount: 0,
    skippedCount: 0
  };
}
/**
 * @name MsgAddBootstrapEntryResponse
 * @package zerone.claiming_pot.v1
 * @see proto type: zerone.claiming_pot.v1.MsgAddBootstrapEntryResponse
 */
export const MsgAddBootstrapEntryResponse = {
  typeUrl: "/zerone.claiming_pot.v1.MsgAddBootstrapEntryResponse",
  encode(message: MsgAddBootstrapEntryResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.addedCount !== 0) {
      writer.uint32(8).uint32(message.addedCount);
    }
    if (message.skippedCount !== 0) {
      writer.uint32(16).uint32(message.skippedCount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgAddBootstrapEntryResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgAddBootstrapEntryResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.addedCount = reader.uint32();
          break;
        case 2:
          message.skippedCount = reader.uint32();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgAddBootstrapEntryResponse>): MsgAddBootstrapEntryResponse {
    const message = createBaseMsgAddBootstrapEntryResponse();
    message.addedCount = object.addedCount ?? 0;
    message.skippedCount = object.skippedCount ?? 0;
    return message;
  }
};
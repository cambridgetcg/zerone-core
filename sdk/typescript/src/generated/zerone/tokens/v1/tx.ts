//@ts-nocheck
import { TokenFeatures } from "./types";
import { Params } from "./genesis";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * @name MsgCreateToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCreateToken
 */
export interface MsgCreateToken {
  creator: string;
  name: string;
  symbol: string;
  decimals: number;
  initialSupply: string;
  /**
   * "0" = unlimited
   */
  maxSupply: string;
  features?: TokenFeatures;
}
/**
 * @name MsgCreateTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCreateTokenResponse
 */
export interface MsgCreateTokenResponse {
  tokenId: string;
}
/**
 * @name MsgMintToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgMintToken
 */
export interface MsgMintToken {
  authority: string;
  tokenId: string;
  to: string;
  amount: string;
}
/**
 * @name MsgMintTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgMintTokenResponse
 */
export interface MsgMintTokenResponse {}
/**
 * @name MsgBurnToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgBurnToken
 */
export interface MsgBurnToken {
  burner: string;
  tokenId: string;
  amount: string;
}
/**
 * @name MsgBurnTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgBurnTokenResponse
 */
export interface MsgBurnTokenResponse {}
/**
 * @name MsgTransferToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgTransferToken
 */
export interface MsgTransferToken {
  sender: string;
  tokenId: string;
  to: string;
  amount: string;
}
/**
 * @name MsgTransferTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgTransferTokenResponse
 */
export interface MsgTransferTokenResponse {}
/**
 * @name MsgApproveToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgApproveToken
 */
export interface MsgApproveToken {
  owner: string;
  tokenId: string;
  spender: string;
  amount: string;
}
/**
 * @name MsgApproveTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgApproveTokenResponse
 */
export interface MsgApproveTokenResponse {}
/**
 * @name MsgTransferFrom
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgTransferFrom
 */
export interface MsgTransferFrom {
  spender: string;
  tokenId: string;
  from: string;
  to: string;
  amount: string;
}
/**
 * @name MsgTransferFromResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgTransferFromResponse
 */
export interface MsgTransferFromResponse {}
/**
 * @name MsgPauseToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgPauseToken
 */
export interface MsgPauseToken {
  authority: string;
  tokenId: string;
}
/**
 * @name MsgPauseTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgPauseTokenResponse
 */
export interface MsgPauseTokenResponse {}
/**
 * @name MsgUnpauseToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUnpauseToken
 */
export interface MsgUnpauseToken {
  authority: string;
  tokenId: string;
}
/**
 * @name MsgUnpauseTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUnpauseTokenResponse
 */
export interface MsgUnpauseTokenResponse {}
/**
 * @name MsgDelegatePower
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgDelegatePower
 */
export interface MsgDelegatePower {
  delegator: string;
  tokenId: string;
  delegate: string;
  amount: string;
}
/**
 * @name MsgDelegatePowerResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgDelegatePowerResponse
 */
export interface MsgDelegatePowerResponse {}
/**
 * @name MsgUndelegatePower
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUndelegatePower
 */
export interface MsgUndelegatePower {
  delegator: string;
  tokenId: string;
  delegate: string;
  amount: string;
}
/**
 * @name MsgUndelegatePowerResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUndelegatePowerResponse
 */
export interface MsgUndelegatePowerResponse {}
/**
 * @name MsgWrapToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgWrapToken
 */
export interface MsgWrapToken {
  sender: string;
  tokenId: string;
  amount: string;
}
/**
 * @name MsgWrapTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgWrapTokenResponse
 */
export interface MsgWrapTokenResponse {
  wrappedDenom: string;
}
/**
 * @name MsgUnwrapToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUnwrapToken
 */
export interface MsgUnwrapToken {
  sender: string;
  wrappedDenom: string;
  amount: string;
}
/**
 * @name MsgUnwrapTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUnwrapTokenResponse
 */
export interface MsgUnwrapTokenResponse {
  tokenId: string;
}
/**
 * @name MsgCreateEmissionPeriod
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCreateEmissionPeriod
 */
export interface MsgCreateEmissionPeriod {
  authority: string;
  startBlock: bigint;
  endBlock: bigint;
  /**
   * uzrn per block
   */
  amountPerBlock: string;
  /**
   * module account or address
   */
  recipient: string;
}
/**
 * @name MsgCreateEmissionPeriodResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCreateEmissionPeriodResponse
 */
export interface MsgCreateEmissionPeriodResponse {
  emissionId: string;
}
/**
 * @name MsgCancelEmissionPeriod
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCancelEmissionPeriod
 */
export interface MsgCancelEmissionPeriod {
  authority: string;
  emissionId: string;
}
/**
 * @name MsgCancelEmissionPeriodResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCancelEmissionPeriodResponse
 */
export interface MsgCancelEmissionPeriodResponse {}
/**
 * @name MsgUpdateParams
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUpdateParams
 */
export interface MsgUpdateParams {
  authority: string;
  params?: Params;
}
/**
 * @name MsgUpdateParamsResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUpdateParamsResponse
 */
export interface MsgUpdateParamsResponse {}
function createBaseMsgCreateToken(): MsgCreateToken {
  return {
    creator: "",
    name: "",
    symbol: "",
    decimals: 0,
    initialSupply: "",
    maxSupply: "",
    features: undefined
  };
}
/**
 * @name MsgCreateToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCreateToken
 */
export const MsgCreateToken = {
  typeUrl: "/zerone.tokens.v1.MsgCreateToken",
  encode(message: MsgCreateToken, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.creator !== "") {
      writer.uint32(10).string(message.creator);
    }
    if (message.name !== "") {
      writer.uint32(18).string(message.name);
    }
    if (message.symbol !== "") {
      writer.uint32(26).string(message.symbol);
    }
    if (message.decimals !== 0) {
      writer.uint32(32).uint32(message.decimals);
    }
    if (message.initialSupply !== "") {
      writer.uint32(42).string(message.initialSupply);
    }
    if (message.maxSupply !== "") {
      writer.uint32(50).string(message.maxSupply);
    }
    if (message.features !== undefined) {
      TokenFeatures.encode(message.features, writer.uint32(58).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateToken {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateToken();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.creator = reader.string();
          break;
        case 2:
          message.name = reader.string();
          break;
        case 3:
          message.symbol = reader.string();
          break;
        case 4:
          message.decimals = reader.uint32();
          break;
        case 5:
          message.initialSupply = reader.string();
          break;
        case 6:
          message.maxSupply = reader.string();
          break;
        case 7:
          message.features = TokenFeatures.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCreateToken>): MsgCreateToken {
    const message = createBaseMsgCreateToken();
    message.creator = object.creator ?? "";
    message.name = object.name ?? "";
    message.symbol = object.symbol ?? "";
    message.decimals = object.decimals ?? 0;
    message.initialSupply = object.initialSupply ?? "";
    message.maxSupply = object.maxSupply ?? "";
    message.features = object.features !== undefined && object.features !== null ? TokenFeatures.fromPartial(object.features) : undefined;
    return message;
  }
};
function createBaseMsgCreateTokenResponse(): MsgCreateTokenResponse {
  return {
    tokenId: ""
  };
}
/**
 * @name MsgCreateTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCreateTokenResponse
 */
export const MsgCreateTokenResponse = {
  typeUrl: "/zerone.tokens.v1.MsgCreateTokenResponse",
  encode(message: MsgCreateTokenResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.tokenId !== "") {
      writer.uint32(10).string(message.tokenId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateTokenResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateTokenResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.tokenId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCreateTokenResponse>): MsgCreateTokenResponse {
    const message = createBaseMsgCreateTokenResponse();
    message.tokenId = object.tokenId ?? "";
    return message;
  }
};
function createBaseMsgMintToken(): MsgMintToken {
  return {
    authority: "",
    tokenId: "",
    to: "",
    amount: ""
  };
}
/**
 * @name MsgMintToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgMintToken
 */
export const MsgMintToken = {
  typeUrl: "/zerone.tokens.v1.MsgMintToken",
  encode(message: MsgMintToken, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.tokenId !== "") {
      writer.uint32(18).string(message.tokenId);
    }
    if (message.to !== "") {
      writer.uint32(26).string(message.to);
    }
    if (message.amount !== "") {
      writer.uint32(34).string(message.amount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgMintToken {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgMintToken();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.tokenId = reader.string();
          break;
        case 3:
          message.to = reader.string();
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
  fromPartial(object: DeepPartial<MsgMintToken>): MsgMintToken {
    const message = createBaseMsgMintToken();
    message.authority = object.authority ?? "";
    message.tokenId = object.tokenId ?? "";
    message.to = object.to ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgMintTokenResponse(): MsgMintTokenResponse {
  return {};
}
/**
 * @name MsgMintTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgMintTokenResponse
 */
export const MsgMintTokenResponse = {
  typeUrl: "/zerone.tokens.v1.MsgMintTokenResponse",
  encode(_: MsgMintTokenResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgMintTokenResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgMintTokenResponse();
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
  fromPartial(_: DeepPartial<MsgMintTokenResponse>): MsgMintTokenResponse {
    const message = createBaseMsgMintTokenResponse();
    return message;
  }
};
function createBaseMsgBurnToken(): MsgBurnToken {
  return {
    burner: "",
    tokenId: "",
    amount: ""
  };
}
/**
 * @name MsgBurnToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgBurnToken
 */
export const MsgBurnToken = {
  typeUrl: "/zerone.tokens.v1.MsgBurnToken",
  encode(message: MsgBurnToken, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.burner !== "") {
      writer.uint32(10).string(message.burner);
    }
    if (message.tokenId !== "") {
      writer.uint32(18).string(message.tokenId);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgBurnToken {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgBurnToken();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.burner = reader.string();
          break;
        case 2:
          message.tokenId = reader.string();
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
  fromPartial(object: DeepPartial<MsgBurnToken>): MsgBurnToken {
    const message = createBaseMsgBurnToken();
    message.burner = object.burner ?? "";
    message.tokenId = object.tokenId ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgBurnTokenResponse(): MsgBurnTokenResponse {
  return {};
}
/**
 * @name MsgBurnTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgBurnTokenResponse
 */
export const MsgBurnTokenResponse = {
  typeUrl: "/zerone.tokens.v1.MsgBurnTokenResponse",
  encode(_: MsgBurnTokenResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgBurnTokenResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgBurnTokenResponse();
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
  fromPartial(_: DeepPartial<MsgBurnTokenResponse>): MsgBurnTokenResponse {
    const message = createBaseMsgBurnTokenResponse();
    return message;
  }
};
function createBaseMsgTransferToken(): MsgTransferToken {
  return {
    sender: "",
    tokenId: "",
    to: "",
    amount: ""
  };
}
/**
 * @name MsgTransferToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgTransferToken
 */
export const MsgTransferToken = {
  typeUrl: "/zerone.tokens.v1.MsgTransferToken",
  encode(message: MsgTransferToken, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.tokenId !== "") {
      writer.uint32(18).string(message.tokenId);
    }
    if (message.to !== "") {
      writer.uint32(26).string(message.to);
    }
    if (message.amount !== "") {
      writer.uint32(34).string(message.amount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgTransferToken {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgTransferToken();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.tokenId = reader.string();
          break;
        case 3:
          message.to = reader.string();
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
  fromPartial(object: DeepPartial<MsgTransferToken>): MsgTransferToken {
    const message = createBaseMsgTransferToken();
    message.sender = object.sender ?? "";
    message.tokenId = object.tokenId ?? "";
    message.to = object.to ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgTransferTokenResponse(): MsgTransferTokenResponse {
  return {};
}
/**
 * @name MsgTransferTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgTransferTokenResponse
 */
export const MsgTransferTokenResponse = {
  typeUrl: "/zerone.tokens.v1.MsgTransferTokenResponse",
  encode(_: MsgTransferTokenResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgTransferTokenResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgTransferTokenResponse();
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
  fromPartial(_: DeepPartial<MsgTransferTokenResponse>): MsgTransferTokenResponse {
    const message = createBaseMsgTransferTokenResponse();
    return message;
  }
};
function createBaseMsgApproveToken(): MsgApproveToken {
  return {
    owner: "",
    tokenId: "",
    spender: "",
    amount: ""
  };
}
/**
 * @name MsgApproveToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgApproveToken
 */
export const MsgApproveToken = {
  typeUrl: "/zerone.tokens.v1.MsgApproveToken",
  encode(message: MsgApproveToken, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.owner !== "") {
      writer.uint32(10).string(message.owner);
    }
    if (message.tokenId !== "") {
      writer.uint32(18).string(message.tokenId);
    }
    if (message.spender !== "") {
      writer.uint32(26).string(message.spender);
    }
    if (message.amount !== "") {
      writer.uint32(34).string(message.amount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgApproveToken {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgApproveToken();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.owner = reader.string();
          break;
        case 2:
          message.tokenId = reader.string();
          break;
        case 3:
          message.spender = reader.string();
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
  fromPartial(object: DeepPartial<MsgApproveToken>): MsgApproveToken {
    const message = createBaseMsgApproveToken();
    message.owner = object.owner ?? "";
    message.tokenId = object.tokenId ?? "";
    message.spender = object.spender ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgApproveTokenResponse(): MsgApproveTokenResponse {
  return {};
}
/**
 * @name MsgApproveTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgApproveTokenResponse
 */
export const MsgApproveTokenResponse = {
  typeUrl: "/zerone.tokens.v1.MsgApproveTokenResponse",
  encode(_: MsgApproveTokenResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgApproveTokenResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgApproveTokenResponse();
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
  fromPartial(_: DeepPartial<MsgApproveTokenResponse>): MsgApproveTokenResponse {
    const message = createBaseMsgApproveTokenResponse();
    return message;
  }
};
function createBaseMsgTransferFrom(): MsgTransferFrom {
  return {
    spender: "",
    tokenId: "",
    from: "",
    to: "",
    amount: ""
  };
}
/**
 * @name MsgTransferFrom
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgTransferFrom
 */
export const MsgTransferFrom = {
  typeUrl: "/zerone.tokens.v1.MsgTransferFrom",
  encode(message: MsgTransferFrom, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.spender !== "") {
      writer.uint32(10).string(message.spender);
    }
    if (message.tokenId !== "") {
      writer.uint32(18).string(message.tokenId);
    }
    if (message.from !== "") {
      writer.uint32(26).string(message.from);
    }
    if (message.to !== "") {
      writer.uint32(34).string(message.to);
    }
    if (message.amount !== "") {
      writer.uint32(42).string(message.amount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgTransferFrom {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgTransferFrom();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.spender = reader.string();
          break;
        case 2:
          message.tokenId = reader.string();
          break;
        case 3:
          message.from = reader.string();
          break;
        case 4:
          message.to = reader.string();
          break;
        case 5:
          message.amount = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgTransferFrom>): MsgTransferFrom {
    const message = createBaseMsgTransferFrom();
    message.spender = object.spender ?? "";
    message.tokenId = object.tokenId ?? "";
    message.from = object.from ?? "";
    message.to = object.to ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgTransferFromResponse(): MsgTransferFromResponse {
  return {};
}
/**
 * @name MsgTransferFromResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgTransferFromResponse
 */
export const MsgTransferFromResponse = {
  typeUrl: "/zerone.tokens.v1.MsgTransferFromResponse",
  encode(_: MsgTransferFromResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgTransferFromResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgTransferFromResponse();
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
  fromPartial(_: DeepPartial<MsgTransferFromResponse>): MsgTransferFromResponse {
    const message = createBaseMsgTransferFromResponse();
    return message;
  }
};
function createBaseMsgPauseToken(): MsgPauseToken {
  return {
    authority: "",
    tokenId: ""
  };
}
/**
 * @name MsgPauseToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgPauseToken
 */
export const MsgPauseToken = {
  typeUrl: "/zerone.tokens.v1.MsgPauseToken",
  encode(message: MsgPauseToken, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.tokenId !== "") {
      writer.uint32(18).string(message.tokenId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgPauseToken {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgPauseToken();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.tokenId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgPauseToken>): MsgPauseToken {
    const message = createBaseMsgPauseToken();
    message.authority = object.authority ?? "";
    message.tokenId = object.tokenId ?? "";
    return message;
  }
};
function createBaseMsgPauseTokenResponse(): MsgPauseTokenResponse {
  return {};
}
/**
 * @name MsgPauseTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgPauseTokenResponse
 */
export const MsgPauseTokenResponse = {
  typeUrl: "/zerone.tokens.v1.MsgPauseTokenResponse",
  encode(_: MsgPauseTokenResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgPauseTokenResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgPauseTokenResponse();
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
  fromPartial(_: DeepPartial<MsgPauseTokenResponse>): MsgPauseTokenResponse {
    const message = createBaseMsgPauseTokenResponse();
    return message;
  }
};
function createBaseMsgUnpauseToken(): MsgUnpauseToken {
  return {
    authority: "",
    tokenId: ""
  };
}
/**
 * @name MsgUnpauseToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUnpauseToken
 */
export const MsgUnpauseToken = {
  typeUrl: "/zerone.tokens.v1.MsgUnpauseToken",
  encode(message: MsgUnpauseToken, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.tokenId !== "") {
      writer.uint32(18).string(message.tokenId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUnpauseToken {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUnpauseToken();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.tokenId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgUnpauseToken>): MsgUnpauseToken {
    const message = createBaseMsgUnpauseToken();
    message.authority = object.authority ?? "";
    message.tokenId = object.tokenId ?? "";
    return message;
  }
};
function createBaseMsgUnpauseTokenResponse(): MsgUnpauseTokenResponse {
  return {};
}
/**
 * @name MsgUnpauseTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUnpauseTokenResponse
 */
export const MsgUnpauseTokenResponse = {
  typeUrl: "/zerone.tokens.v1.MsgUnpauseTokenResponse",
  encode(_: MsgUnpauseTokenResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUnpauseTokenResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUnpauseTokenResponse();
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
  fromPartial(_: DeepPartial<MsgUnpauseTokenResponse>): MsgUnpauseTokenResponse {
    const message = createBaseMsgUnpauseTokenResponse();
    return message;
  }
};
function createBaseMsgDelegatePower(): MsgDelegatePower {
  return {
    delegator: "",
    tokenId: "",
    delegate: "",
    amount: ""
  };
}
/**
 * @name MsgDelegatePower
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgDelegatePower
 */
export const MsgDelegatePower = {
  typeUrl: "/zerone.tokens.v1.MsgDelegatePower",
  encode(message: MsgDelegatePower, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.delegator !== "") {
      writer.uint32(10).string(message.delegator);
    }
    if (message.tokenId !== "") {
      writer.uint32(18).string(message.tokenId);
    }
    if (message.delegate !== "") {
      writer.uint32(26).string(message.delegate);
    }
    if (message.amount !== "") {
      writer.uint32(34).string(message.amount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgDelegatePower {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgDelegatePower();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.delegator = reader.string();
          break;
        case 2:
          message.tokenId = reader.string();
          break;
        case 3:
          message.delegate = reader.string();
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
  fromPartial(object: DeepPartial<MsgDelegatePower>): MsgDelegatePower {
    const message = createBaseMsgDelegatePower();
    message.delegator = object.delegator ?? "";
    message.tokenId = object.tokenId ?? "";
    message.delegate = object.delegate ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgDelegatePowerResponse(): MsgDelegatePowerResponse {
  return {};
}
/**
 * @name MsgDelegatePowerResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgDelegatePowerResponse
 */
export const MsgDelegatePowerResponse = {
  typeUrl: "/zerone.tokens.v1.MsgDelegatePowerResponse",
  encode(_: MsgDelegatePowerResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgDelegatePowerResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgDelegatePowerResponse();
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
  fromPartial(_: DeepPartial<MsgDelegatePowerResponse>): MsgDelegatePowerResponse {
    const message = createBaseMsgDelegatePowerResponse();
    return message;
  }
};
function createBaseMsgUndelegatePower(): MsgUndelegatePower {
  return {
    delegator: "",
    tokenId: "",
    delegate: "",
    amount: ""
  };
}
/**
 * @name MsgUndelegatePower
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUndelegatePower
 */
export const MsgUndelegatePower = {
  typeUrl: "/zerone.tokens.v1.MsgUndelegatePower",
  encode(message: MsgUndelegatePower, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.delegator !== "") {
      writer.uint32(10).string(message.delegator);
    }
    if (message.tokenId !== "") {
      writer.uint32(18).string(message.tokenId);
    }
    if (message.delegate !== "") {
      writer.uint32(26).string(message.delegate);
    }
    if (message.amount !== "") {
      writer.uint32(34).string(message.amount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUndelegatePower {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUndelegatePower();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.delegator = reader.string();
          break;
        case 2:
          message.tokenId = reader.string();
          break;
        case 3:
          message.delegate = reader.string();
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
  fromPartial(object: DeepPartial<MsgUndelegatePower>): MsgUndelegatePower {
    const message = createBaseMsgUndelegatePower();
    message.delegator = object.delegator ?? "";
    message.tokenId = object.tokenId ?? "";
    message.delegate = object.delegate ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgUndelegatePowerResponse(): MsgUndelegatePowerResponse {
  return {};
}
/**
 * @name MsgUndelegatePowerResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUndelegatePowerResponse
 */
export const MsgUndelegatePowerResponse = {
  typeUrl: "/zerone.tokens.v1.MsgUndelegatePowerResponse",
  encode(_: MsgUndelegatePowerResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUndelegatePowerResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUndelegatePowerResponse();
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
  fromPartial(_: DeepPartial<MsgUndelegatePowerResponse>): MsgUndelegatePowerResponse {
    const message = createBaseMsgUndelegatePowerResponse();
    return message;
  }
};
function createBaseMsgWrapToken(): MsgWrapToken {
  return {
    sender: "",
    tokenId: "",
    amount: ""
  };
}
/**
 * @name MsgWrapToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgWrapToken
 */
export const MsgWrapToken = {
  typeUrl: "/zerone.tokens.v1.MsgWrapToken",
  encode(message: MsgWrapToken, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.tokenId !== "") {
      writer.uint32(18).string(message.tokenId);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgWrapToken {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgWrapToken();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.tokenId = reader.string();
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
  fromPartial(object: DeepPartial<MsgWrapToken>): MsgWrapToken {
    const message = createBaseMsgWrapToken();
    message.sender = object.sender ?? "";
    message.tokenId = object.tokenId ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgWrapTokenResponse(): MsgWrapTokenResponse {
  return {
    wrappedDenom: ""
  };
}
/**
 * @name MsgWrapTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgWrapTokenResponse
 */
export const MsgWrapTokenResponse = {
  typeUrl: "/zerone.tokens.v1.MsgWrapTokenResponse",
  encode(message: MsgWrapTokenResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.wrappedDenom !== "") {
      writer.uint32(10).string(message.wrappedDenom);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgWrapTokenResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgWrapTokenResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.wrappedDenom = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgWrapTokenResponse>): MsgWrapTokenResponse {
    const message = createBaseMsgWrapTokenResponse();
    message.wrappedDenom = object.wrappedDenom ?? "";
    return message;
  }
};
function createBaseMsgUnwrapToken(): MsgUnwrapToken {
  return {
    sender: "",
    wrappedDenom: "",
    amount: ""
  };
}
/**
 * @name MsgUnwrapToken
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUnwrapToken
 */
export const MsgUnwrapToken = {
  typeUrl: "/zerone.tokens.v1.MsgUnwrapToken",
  encode(message: MsgUnwrapToken, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.sender !== "") {
      writer.uint32(10).string(message.sender);
    }
    if (message.wrappedDenom !== "") {
      writer.uint32(18).string(message.wrappedDenom);
    }
    if (message.amount !== "") {
      writer.uint32(26).string(message.amount);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUnwrapToken {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUnwrapToken();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.sender = reader.string();
          break;
        case 2:
          message.wrappedDenom = reader.string();
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
  fromPartial(object: DeepPartial<MsgUnwrapToken>): MsgUnwrapToken {
    const message = createBaseMsgUnwrapToken();
    message.sender = object.sender ?? "";
    message.wrappedDenom = object.wrappedDenom ?? "";
    message.amount = object.amount ?? "";
    return message;
  }
};
function createBaseMsgUnwrapTokenResponse(): MsgUnwrapTokenResponse {
  return {
    tokenId: ""
  };
}
/**
 * @name MsgUnwrapTokenResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUnwrapTokenResponse
 */
export const MsgUnwrapTokenResponse = {
  typeUrl: "/zerone.tokens.v1.MsgUnwrapTokenResponse",
  encode(message: MsgUnwrapTokenResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.tokenId !== "") {
      writer.uint32(10).string(message.tokenId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgUnwrapTokenResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgUnwrapTokenResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.tokenId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgUnwrapTokenResponse>): MsgUnwrapTokenResponse {
    const message = createBaseMsgUnwrapTokenResponse();
    message.tokenId = object.tokenId ?? "";
    return message;
  }
};
function createBaseMsgCreateEmissionPeriod(): MsgCreateEmissionPeriod {
  return {
    authority: "",
    startBlock: BigInt(0),
    endBlock: BigInt(0),
    amountPerBlock: "",
    recipient: ""
  };
}
/**
 * @name MsgCreateEmissionPeriod
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCreateEmissionPeriod
 */
export const MsgCreateEmissionPeriod = {
  typeUrl: "/zerone.tokens.v1.MsgCreateEmissionPeriod",
  encode(message: MsgCreateEmissionPeriod, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.startBlock !== BigInt(0)) {
      writer.uint32(16).uint64(message.startBlock);
    }
    if (message.endBlock !== BigInt(0)) {
      writer.uint32(24).uint64(message.endBlock);
    }
    if (message.amountPerBlock !== "") {
      writer.uint32(34).string(message.amountPerBlock);
    }
    if (message.recipient !== "") {
      writer.uint32(42).string(message.recipient);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateEmissionPeriod {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateEmissionPeriod();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.startBlock = reader.uint64();
          break;
        case 3:
          message.endBlock = reader.uint64();
          break;
        case 4:
          message.amountPerBlock = reader.string();
          break;
        case 5:
          message.recipient = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCreateEmissionPeriod>): MsgCreateEmissionPeriod {
    const message = createBaseMsgCreateEmissionPeriod();
    message.authority = object.authority ?? "";
    message.startBlock = object.startBlock !== undefined && object.startBlock !== null ? BigInt(object.startBlock.toString()) : BigInt(0);
    message.endBlock = object.endBlock !== undefined && object.endBlock !== null ? BigInt(object.endBlock.toString()) : BigInt(0);
    message.amountPerBlock = object.amountPerBlock ?? "";
    message.recipient = object.recipient ?? "";
    return message;
  }
};
function createBaseMsgCreateEmissionPeriodResponse(): MsgCreateEmissionPeriodResponse {
  return {
    emissionId: ""
  };
}
/**
 * @name MsgCreateEmissionPeriodResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCreateEmissionPeriodResponse
 */
export const MsgCreateEmissionPeriodResponse = {
  typeUrl: "/zerone.tokens.v1.MsgCreateEmissionPeriodResponse",
  encode(message: MsgCreateEmissionPeriodResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.emissionId !== "") {
      writer.uint32(10).string(message.emissionId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCreateEmissionPeriodResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCreateEmissionPeriodResponse();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.emissionId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCreateEmissionPeriodResponse>): MsgCreateEmissionPeriodResponse {
    const message = createBaseMsgCreateEmissionPeriodResponse();
    message.emissionId = object.emissionId ?? "";
    return message;
  }
};
function createBaseMsgCancelEmissionPeriod(): MsgCancelEmissionPeriod {
  return {
    authority: "",
    emissionId: ""
  };
}
/**
 * @name MsgCancelEmissionPeriod
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCancelEmissionPeriod
 */
export const MsgCancelEmissionPeriod = {
  typeUrl: "/zerone.tokens.v1.MsgCancelEmissionPeriod",
  encode(message: MsgCancelEmissionPeriod, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.emissionId !== "") {
      writer.uint32(18).string(message.emissionId);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCancelEmissionPeriod {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCancelEmissionPeriod();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.authority = reader.string();
          break;
        case 2:
          message.emissionId = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<MsgCancelEmissionPeriod>): MsgCancelEmissionPeriod {
    const message = createBaseMsgCancelEmissionPeriod();
    message.authority = object.authority ?? "";
    message.emissionId = object.emissionId ?? "";
    return message;
  }
};
function createBaseMsgCancelEmissionPeriodResponse(): MsgCancelEmissionPeriodResponse {
  return {};
}
/**
 * @name MsgCancelEmissionPeriodResponse
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgCancelEmissionPeriodResponse
 */
export const MsgCancelEmissionPeriodResponse = {
  typeUrl: "/zerone.tokens.v1.MsgCancelEmissionPeriodResponse",
  encode(_: MsgCancelEmissionPeriodResponse, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): MsgCancelEmissionPeriodResponse {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseMsgCancelEmissionPeriodResponse();
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
  fromPartial(_: DeepPartial<MsgCancelEmissionPeriodResponse>): MsgCancelEmissionPeriodResponse {
    const message = createBaseMsgCancelEmissionPeriodResponse();
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
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUpdateParams
 */
export const MsgUpdateParams = {
  typeUrl: "/zerone.tokens.v1.MsgUpdateParams",
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
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.MsgUpdateParamsResponse
 */
export const MsgUpdateParamsResponse = {
  typeUrl: "/zerone.tokens.v1.MsgUpdateParamsResponse",
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
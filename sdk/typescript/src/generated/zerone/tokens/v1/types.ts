//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * TokenFeatures controls which operations are enabled for a token.
 * @name TokenFeatures
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.TokenFeatures
 */
export interface TokenFeatures {
  mintable: boolean;
  burnable: boolean;
  pausable: boolean;
  wrappable: boolean;
}
/**
 * TokenDefinition is a ZRN-20 secondary token's on-chain configuration.
 * @name TokenDefinition
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.TokenDefinition
 */
export interface TokenDefinition {
  id: string;
  /**
   * address that created the token (= mint authority)
   */
  creator: string;
  name: string;
  /**
   * 1-16 uppercase alphanumeric
   */
  symbol: string;
  decimals: number;
  /**
   * bigint as string
   */
  totalSupply: string;
  /**
   * "0" = unlimited
   */
  maxSupply: string;
  features?: TokenFeatures;
  paused: boolean;
  /**
   * block height
   */
  createdAt: bigint;
}
/**
 * EmissionPeriod defines a scheduled token emission (replaces LP pools).
 * @name EmissionPeriod
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.EmissionPeriod
 */
export interface EmissionPeriod {
  id: string;
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
  active: boolean;
  /**
   * running total emitted
   */
  totalEmitted: string;
  /**
   * governance authority that created it
   */
  creator: string;
}
function createBaseTokenFeatures(): TokenFeatures {
  return {
    mintable: false,
    burnable: false,
    pausable: false,
    wrappable: false
  };
}
/**
 * TokenFeatures controls which operations are enabled for a token.
 * @name TokenFeatures
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.TokenFeatures
 */
export const TokenFeatures = {
  typeUrl: "/zerone.tokens.v1.TokenFeatures",
  encode(message: TokenFeatures, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.mintable === true) {
      writer.uint32(8).bool(message.mintable);
    }
    if (message.burnable === true) {
      writer.uint32(16).bool(message.burnable);
    }
    if (message.pausable === true) {
      writer.uint32(24).bool(message.pausable);
    }
    if (message.wrappable === true) {
      writer.uint32(32).bool(message.wrappable);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): TokenFeatures {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTokenFeatures();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.mintable = reader.bool();
          break;
        case 2:
          message.burnable = reader.bool();
          break;
        case 3:
          message.pausable = reader.bool();
          break;
        case 4:
          message.wrappable = reader.bool();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<TokenFeatures>): TokenFeatures {
    const message = createBaseTokenFeatures();
    message.mintable = object.mintable ?? false;
    message.burnable = object.burnable ?? false;
    message.pausable = object.pausable ?? false;
    message.wrappable = object.wrappable ?? false;
    return message;
  }
};
function createBaseTokenDefinition(): TokenDefinition {
  return {
    id: "",
    creator: "",
    name: "",
    symbol: "",
    decimals: 0,
    totalSupply: "",
    maxSupply: "",
    features: undefined,
    paused: false,
    createdAt: BigInt(0)
  };
}
/**
 * TokenDefinition is a ZRN-20 secondary token's on-chain configuration.
 * @name TokenDefinition
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.TokenDefinition
 */
export const TokenDefinition = {
  typeUrl: "/zerone.tokens.v1.TokenDefinition",
  encode(message: TokenDefinition, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
    }
    if (message.creator !== "") {
      writer.uint32(18).string(message.creator);
    }
    if (message.name !== "") {
      writer.uint32(26).string(message.name);
    }
    if (message.symbol !== "") {
      writer.uint32(34).string(message.symbol);
    }
    if (message.decimals !== 0) {
      writer.uint32(40).uint32(message.decimals);
    }
    if (message.totalSupply !== "") {
      writer.uint32(50).string(message.totalSupply);
    }
    if (message.maxSupply !== "") {
      writer.uint32(58).string(message.maxSupply);
    }
    if (message.features !== undefined) {
      TokenFeatures.encode(message.features, writer.uint32(66).fork()).ldelim();
    }
    if (message.paused === true) {
      writer.uint32(72).bool(message.paused);
    }
    if (message.createdAt !== BigInt(0)) {
      writer.uint32(80).uint64(message.createdAt);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): TokenDefinition {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseTokenDefinition();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
          break;
        case 2:
          message.creator = reader.string();
          break;
        case 3:
          message.name = reader.string();
          break;
        case 4:
          message.symbol = reader.string();
          break;
        case 5:
          message.decimals = reader.uint32();
          break;
        case 6:
          message.totalSupply = reader.string();
          break;
        case 7:
          message.maxSupply = reader.string();
          break;
        case 8:
          message.features = TokenFeatures.decode(reader, reader.uint32());
          break;
        case 9:
          message.paused = reader.bool();
          break;
        case 10:
          message.createdAt = reader.uint64();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<TokenDefinition>): TokenDefinition {
    const message = createBaseTokenDefinition();
    message.id = object.id ?? "";
    message.creator = object.creator ?? "";
    message.name = object.name ?? "";
    message.symbol = object.symbol ?? "";
    message.decimals = object.decimals ?? 0;
    message.totalSupply = object.totalSupply ?? "";
    message.maxSupply = object.maxSupply ?? "";
    message.features = object.features !== undefined && object.features !== null ? TokenFeatures.fromPartial(object.features) : undefined;
    message.paused = object.paused ?? false;
    message.createdAt = object.createdAt !== undefined && object.createdAt !== null ? BigInt(object.createdAt.toString()) : BigInt(0);
    return message;
  }
};
function createBaseEmissionPeriod(): EmissionPeriod {
  return {
    id: "",
    startBlock: BigInt(0),
    endBlock: BigInt(0),
    amountPerBlock: "",
    recipient: "",
    active: false,
    totalEmitted: "",
    creator: ""
  };
}
/**
 * EmissionPeriod defines a scheduled token emission (replaces LP pools).
 * @name EmissionPeriod
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.EmissionPeriod
 */
export const EmissionPeriod = {
  typeUrl: "/zerone.tokens.v1.EmissionPeriod",
  encode(message: EmissionPeriod, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.id !== "") {
      writer.uint32(10).string(message.id);
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
    if (message.active === true) {
      writer.uint32(48).bool(message.active);
    }
    if (message.totalEmitted !== "") {
      writer.uint32(58).string(message.totalEmitted);
    }
    if (message.creator !== "") {
      writer.uint32(66).string(message.creator);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): EmissionPeriod {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseEmissionPeriod();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.id = reader.string();
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
        case 6:
          message.active = reader.bool();
          break;
        case 7:
          message.totalEmitted = reader.string();
          break;
        case 8:
          message.creator = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<EmissionPeriod>): EmissionPeriod {
    const message = createBaseEmissionPeriod();
    message.id = object.id ?? "";
    message.startBlock = object.startBlock !== undefined && object.startBlock !== null ? BigInt(object.startBlock.toString()) : BigInt(0);
    message.endBlock = object.endBlock !== undefined && object.endBlock !== null ? BigInt(object.endBlock.toString()) : BigInt(0);
    message.amountPerBlock = object.amountPerBlock ?? "";
    message.recipient = object.recipient ?? "";
    message.active = object.active ?? false;
    message.totalEmitted = object.totalEmitted ?? "";
    message.creator = object.creator ?? "";
    return message;
  }
};
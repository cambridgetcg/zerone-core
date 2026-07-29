//@ts-nocheck
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * GenesisState defines the tokens module's genesis state.
 * @name GenesisState
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.GenesisState
 */
export interface GenesisState {
  params?: Params;
}
/**
 * Params holds the tokens module parameters.
 * All fields default to zero (module is a stub at genesis).
 * @name Params
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.Params
 */
export interface Params {
  /**
   * Consensus activation latch for native uzrn EmissionPeriod creation and
   * processing. 0 disables both. A nonzero value enables the per-block periods;
   * its numeric epoch cadence is otherwise reserved in current execution.
   */
  emissionEpochBlocks: bigint;
  /**
   * default swap fee in BPS (unused, reserved)
   */
  defaultFeeBps: string;
}
function createBaseGenesisState(): GenesisState {
  return {
    params: undefined
  };
}
/**
 * GenesisState defines the tokens module's genesis state.
 * @name GenesisState
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.GenesisState
 */
export const GenesisState = {
  typeUrl: "/zerone.tokens.v1.GenesisState",
  encode(message: GenesisState, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): GenesisState {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseGenesisState();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.params = Params.decode(reader, reader.uint32());
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<GenesisState>): GenesisState {
    const message = createBaseGenesisState();
    message.params = object.params !== undefined && object.params !== null ? Params.fromPartial(object.params) : undefined;
    return message;
  }
};
function createBaseParams(): Params {
  return {
    emissionEpochBlocks: BigInt(0),
    defaultFeeBps: ""
  };
}
/**
 * Params holds the tokens module parameters.
 * All fields default to zero (module is a stub at genesis).
 * @name Params
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.Params
 */
export const Params = {
  typeUrl: "/zerone.tokens.v1.Params",
  encode(message: Params, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.emissionEpochBlocks !== BigInt(0)) {
      writer.uint32(8).uint64(message.emissionEpochBlocks);
    }
    if (message.defaultFeeBps !== "") {
      writer.uint32(18).string(message.defaultFeeBps);
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): Params {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseParams();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.emissionEpochBlocks = reader.uint64();
          break;
        case 2:
          message.defaultFeeBps = reader.string();
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<Params>): Params {
    const message = createBaseParams();
    message.emissionEpochBlocks = object.emissionEpochBlocks !== undefined && object.emissionEpochBlocks !== null ? BigInt(object.emissionEpochBlocks.toString()) : BigInt(0);
    message.defaultFeeBps = object.defaultFeeBps ?? "";
    return message;
  }
};
//@ts-nocheck
import { RateLimit } from "./types";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * Params defines the ibcratelimit module parameters.
 * @name Params
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.Params
 */
export interface Params {
  /**
   * global kill switch for rate limiting
   */
  enabled: boolean;
}
/**
 * GenesisState defines the ibcratelimit module genesis state.
 * @name GenesisState
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.GenesisState
 */
export interface GenesisState {
  params?: Params;
  rateLimits: RateLimit[];
  /**
   * Refund-identity marker (zerone-core #127). Native genesis sets it; a
   * file without the field imports as off. In place, only the named carrier
   * upgrade consensus-repairs-v1 sets it.
   */
  refundIdentityEnabled: boolean;
}
function createBaseParams(): Params {
  return {
    enabled: false
  };
}
/**
 * Params defines the ibcratelimit module parameters.
 * @name Params
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.Params
 */
export const Params = {
  typeUrl: "/zerone.ibcratelimit.v1.Params",
  encode(message: Params, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.enabled === true) {
      writer.uint32(8).bool(message.enabled);
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
          message.enabled = reader.bool();
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
    message.enabled = object.enabled ?? false;
    return message;
  }
};
function createBaseGenesisState(): GenesisState {
  return {
    params: undefined,
    rateLimits: [],
    refundIdentityEnabled: false
  };
}
/**
 * GenesisState defines the ibcratelimit module genesis state.
 * @name GenesisState
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.GenesisState
 */
export const GenesisState = {
  typeUrl: "/zerone.ibcratelimit.v1.GenesisState",
  encode(message: GenesisState, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.rateLimits) {
      RateLimit.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    if (message.refundIdentityEnabled === true) {
      writer.uint32(24).bool(message.refundIdentityEnabled);
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
        case 2:
          message.rateLimits.push(RateLimit.decode(reader, reader.uint32()));
          break;
        case 3:
          message.refundIdentityEnabled = reader.bool();
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
    message.rateLimits = object.rateLimits?.map(e => RateLimit.fromPartial(e)) || [];
    message.refundIdentityEnabled = object.refundIdentityEnabled ?? false;
    return message;
  }
};
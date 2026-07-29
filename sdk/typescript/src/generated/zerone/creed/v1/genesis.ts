//@ts-nocheck
import { PinnedCreed, CreedCouncilMember } from "./types";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * GenesisState can seed an optional version-1 creed pin and history.
 * Default and published zerone-1 genesis states omit the pin.
 * @name GenesisState
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.GenesisState
 */
export interface GenesisState {
  params?: Params;
  /**
   * Optional version-1 pin. If empty, InitGenesis stores no pin; it does not
   * synthesize a placeholder. Repository CI does not inspect live state.
   */
  genesisPin?: PinnedCreed;
  /**
   * Optional historical pins for chains that migrate from a
   * pre-x/creed state. Sorted by version ascending. Each must be
   * strictly older than genesis_pin.
   */
  history: PinnedCreed[];
  /**
   * Initial Creed Council registry. This is a future two-pool
   * vote-routing surface; current ordinary LIP tally does not read it.
   * voting_weight_bps should sum to ≤ 1_000_000.
   */
  councilMembers: CreedCouncilMember[];
}
/**
 * @name Params
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.Params
 */
export interface Params {
  /**
   * Compatibility-only metadata. Runtime authorization uses the keeper
   * constructor's gov-module authority and rejects changes to this field.
   */
  authority: string;
  /**
   * Whether direct authority-gated AnchorPin calls are enabled.
   * Source default and published zerone-1 genesis: true. A future
   * governance-only activation must configure the amendment category and
   * set this false through a release-bound change.
   */
  directAnchorEnabled: boolean;
}
function createBaseGenesisState(): GenesisState {
  return {
    params: undefined,
    genesisPin: undefined,
    history: [],
    councilMembers: []
  };
}
/**
 * GenesisState can seed an optional version-1 creed pin and history.
 * Default and published zerone-1 genesis states omit the pin.
 * @name GenesisState
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.GenesisState
 */
export const GenesisState = {
  typeUrl: "/zerone.creed.v1.GenesisState",
  encode(message: GenesisState, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    if (message.genesisPin !== undefined) {
      PinnedCreed.encode(message.genesisPin, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.history) {
      PinnedCreed.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    for (const v of message.councilMembers) {
      CreedCouncilMember.encode(v!, writer.uint32(34).fork()).ldelim();
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
          message.genesisPin = PinnedCreed.decode(reader, reader.uint32());
          break;
        case 3:
          message.history.push(PinnedCreed.decode(reader, reader.uint32()));
          break;
        case 4:
          message.councilMembers.push(CreedCouncilMember.decode(reader, reader.uint32()));
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
    message.genesisPin = object.genesisPin !== undefined && object.genesisPin !== null ? PinnedCreed.fromPartial(object.genesisPin) : undefined;
    message.history = object.history?.map(e => PinnedCreed.fromPartial(e)) || [];
    message.councilMembers = object.councilMembers?.map(e => CreedCouncilMember.fromPartial(e)) || [];
    return message;
  }
};
function createBaseParams(): Params {
  return {
    authority: "",
    directAnchorEnabled: false
  };
}
/**
 * @name Params
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.Params
 */
export const Params = {
  typeUrl: "/zerone.creed.v1.Params",
  encode(message: Params, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.authority !== "") {
      writer.uint32(10).string(message.authority);
    }
    if (message.directAnchorEnabled === true) {
      writer.uint32(16).bool(message.directAnchorEnabled);
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
          message.authority = reader.string();
          break;
        case 2:
          message.directAnchorEnabled = reader.bool();
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
    message.authority = object.authority ?? "";
    message.directAnchorEnabled = object.directAnchorEnabled ?? false;
    return message;
  }
};
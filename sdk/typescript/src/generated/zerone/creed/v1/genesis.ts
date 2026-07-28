//@ts-nocheck
import { PinnedCreed, CreedCouncilMember } from "./types";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * GenesisState seeds the chain's pinned creed at version 1. The
 * genesis pin establishes the baseline against which all future
 * creed amendments are measured. Any commitment in TRUTH_SEEKING.md
 * at testnet→mainnet transition becomes part of the Genesis Creed
 * and is recorded with introduced_via_lip="" (no LIP precedes
 * genesis).
 * @name GenesisState
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.GenesisState
 */
export interface GenesisState {
  params?: Params;
  /**
   * The version-1 pin. If empty at chain start, x/creed
   * InitGenesis seeds a placeholder that subsequent governance
   * must replace before any commitment-citing event passes
   * CI's hash check. In normal mainnet startup this MUST be
   * populated with the canonical Genesis Creed.
   */
  genesisPin?: PinnedCreed;
  /**
   * Optional historical pins for chains that migrate from a
   * pre-x/creed state. Sorted by version ascending. Each must be
   * strictly older than genesis_pin.
   */
  history: PinnedCreed[];
  /**
   * Initial Creed Council members. At launch this is a curated
   * set of AI-side home addresses representing diverse capability
   * profiles. Their voting_weight_bps should sum to
   * ≤ 1_000_000; future capability-gated admissions enter via
   * Creed Amendment LIPs.
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
   * Authority that may call MsgAnchorPin pre-LIP-class. Once
   * x/gov.CategoryCreedAmendment ships, this should match the gov
   * module account so only LIP-resolved amendments succeed.
   */
  authority: string;
  /**
   * Whether direct authority-gated AnchorPin calls are enabled.
   * Pre-launch: true (so genesis can pin and one-off corrections
   * are possible). Post-launch: false, with all pins flowing
   * through the LIP class.
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
 * GenesisState seeds the chain's pinned creed at version 1. The
 * genesis pin establishes the baseline against which all future
 * creed amendments are measured. Any commitment in TRUTH_SEEKING.md
 * at testnet→mainnet transition becomes part of the Genesis Creed
 * and is recorded with introduced_via_lip="" (no LIP precedes
 * genesis).
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
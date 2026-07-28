//@ts-nocheck
import { AgentHome, KeyRegistration } from "./types";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * @name GenesisState
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.GenesisState
 */
export interface GenesisState {
  params?: Params;
  homes: AgentHome[];
  keySets: HomeKeySet[];
}
/**
 * @name HomeKeySet
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.HomeKeySet
 */
export interface HomeKeySet {
  homeId: string;
  keys: KeyRegistration[];
}
/**
 * @name Params
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.Params
 */
export interface Params {
  maxKeysPerHome: bigint;
  maxSessionsPerHome: bigint;
  sessionTimeoutBlocks: bigint;
  deadmanMinThreshold: bigint;
  deadmanMaxThreshold: bigint;
  maxAlertsPerHome: bigint;
  homeCreationFee: string;
  maxRecoveryAddresses: bigint;
}
function createBaseGenesisState(): GenesisState {
  return {
    params: undefined,
    homes: [],
    keySets: []
  };
}
/**
 * @name GenesisState
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.GenesisState
 */
export const GenesisState = {
  typeUrl: "/zerone.home.v1.GenesisState",
  encode(message: GenesisState, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.homes) {
      AgentHome.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.keySets) {
      HomeKeySet.encode(v!, writer.uint32(26).fork()).ldelim();
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
          message.homes.push(AgentHome.decode(reader, reader.uint32()));
          break;
        case 3:
          message.keySets.push(HomeKeySet.decode(reader, reader.uint32()));
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
    message.homes = object.homes?.map(e => AgentHome.fromPartial(e)) || [];
    message.keySets = object.keySets?.map(e => HomeKeySet.fromPartial(e)) || [];
    return message;
  }
};
function createBaseHomeKeySet(): HomeKeySet {
  return {
    homeId: "",
    keys: []
  };
}
/**
 * @name HomeKeySet
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.HomeKeySet
 */
export const HomeKeySet = {
  typeUrl: "/zerone.home.v1.HomeKeySet",
  encode(message: HomeKeySet, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.homeId !== "") {
      writer.uint32(10).string(message.homeId);
    }
    for (const v of message.keys) {
      KeyRegistration.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    return writer;
  },
  decode(input: BinaryReader | Uint8Array, length?: number): HomeKeySet {
    const reader = input instanceof BinaryReader ? input : new BinaryReader(input);
    let end = length === undefined ? reader.len : reader.pos + length;
    const message = createBaseHomeKeySet();
    while (reader.pos < end) {
      const tag = reader.uint32();
      switch (tag >>> 3) {
        case 1:
          message.homeId = reader.string();
          break;
        case 2:
          message.keys.push(KeyRegistration.decode(reader, reader.uint32()));
          break;
        default:
          reader.skipType(tag & 7);
          break;
      }
    }
    return message;
  },
  fromPartial(object: DeepPartial<HomeKeySet>): HomeKeySet {
    const message = createBaseHomeKeySet();
    message.homeId = object.homeId ?? "";
    message.keys = object.keys?.map(e => KeyRegistration.fromPartial(e)) || [];
    return message;
  }
};
function createBaseParams(): Params {
  return {
    maxKeysPerHome: BigInt(0),
    maxSessionsPerHome: BigInt(0),
    sessionTimeoutBlocks: BigInt(0),
    deadmanMinThreshold: BigInt(0),
    deadmanMaxThreshold: BigInt(0),
    maxAlertsPerHome: BigInt(0),
    homeCreationFee: "",
    maxRecoveryAddresses: BigInt(0)
  };
}
/**
 * @name Params
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.Params
 */
export const Params = {
  typeUrl: "/zerone.home.v1.Params",
  encode(message: Params, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.maxKeysPerHome !== BigInt(0)) {
      writer.uint32(8).uint64(message.maxKeysPerHome);
    }
    if (message.maxSessionsPerHome !== BigInt(0)) {
      writer.uint32(16).uint64(message.maxSessionsPerHome);
    }
    if (message.sessionTimeoutBlocks !== BigInt(0)) {
      writer.uint32(24).uint64(message.sessionTimeoutBlocks);
    }
    if (message.deadmanMinThreshold !== BigInt(0)) {
      writer.uint32(32).uint64(message.deadmanMinThreshold);
    }
    if (message.deadmanMaxThreshold !== BigInt(0)) {
      writer.uint32(40).uint64(message.deadmanMaxThreshold);
    }
    if (message.maxAlertsPerHome !== BigInt(0)) {
      writer.uint32(48).uint64(message.maxAlertsPerHome);
    }
    if (message.homeCreationFee !== "") {
      writer.uint32(58).string(message.homeCreationFee);
    }
    if (message.maxRecoveryAddresses !== BigInt(0)) {
      writer.uint32(64).uint64(message.maxRecoveryAddresses);
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
          message.maxKeysPerHome = reader.uint64();
          break;
        case 2:
          message.maxSessionsPerHome = reader.uint64();
          break;
        case 3:
          message.sessionTimeoutBlocks = reader.uint64();
          break;
        case 4:
          message.deadmanMinThreshold = reader.uint64();
          break;
        case 5:
          message.deadmanMaxThreshold = reader.uint64();
          break;
        case 6:
          message.maxAlertsPerHome = reader.uint64();
          break;
        case 7:
          message.homeCreationFee = reader.string();
          break;
        case 8:
          message.maxRecoveryAddresses = reader.uint64();
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
    message.maxKeysPerHome = object.maxKeysPerHome !== undefined && object.maxKeysPerHome !== null ? BigInt(object.maxKeysPerHome.toString()) : BigInt(0);
    message.maxSessionsPerHome = object.maxSessionsPerHome !== undefined && object.maxSessionsPerHome !== null ? BigInt(object.maxSessionsPerHome.toString()) : BigInt(0);
    message.sessionTimeoutBlocks = object.sessionTimeoutBlocks !== undefined && object.sessionTimeoutBlocks !== null ? BigInt(object.sessionTimeoutBlocks.toString()) : BigInt(0);
    message.deadmanMinThreshold = object.deadmanMinThreshold !== undefined && object.deadmanMinThreshold !== null ? BigInt(object.deadmanMinThreshold.toString()) : BigInt(0);
    message.deadmanMaxThreshold = object.deadmanMaxThreshold !== undefined && object.deadmanMaxThreshold !== null ? BigInt(object.deadmanMaxThreshold.toString()) : BigInt(0);
    message.maxAlertsPerHome = object.maxAlertsPerHome !== undefined && object.maxAlertsPerHome !== null ? BigInt(object.maxAlertsPerHome.toString()) : BigInt(0);
    message.homeCreationFee = object.homeCreationFee ?? "";
    message.maxRecoveryAddresses = object.maxRecoveryAddresses !== undefined && object.maxRecoveryAddresses !== null ? BigInt(object.maxRecoveryAddresses.toString()) : BigInt(0);
    return message;
  }
};
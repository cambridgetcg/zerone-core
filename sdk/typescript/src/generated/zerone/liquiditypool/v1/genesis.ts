//@ts-nocheck
import { Pool, TWAPAccumulator, TWAPObservation } from "./types";
import { BinaryReader, BinaryWriter } from "../../../binary";
import { DeepPartial } from "../../../helpers";
/**
 * Params defines the liquiditypool module parameters.
 * @name Params
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.Params
 */
export interface Params {
  /**
   * default swap fee (1M bps scale)
   */
  defaultSwapFeeBps: bigint;
  /**
   * maximum number of open (non-CLOSED) pools
   */
  maxPools: bigint;
  /**
   * minimum uzrn-side liquidity in base units (bigint string)
   */
  minInitialLiquidity: string;
  /**
   * default TWAP window in blocks
   */
  twapWindowBlocks: bigint;
  /**
   * Protocol share of the floor-rounded fee, applied only to ZRN-input swaps
   * (1M bps scale).
   * Retained for wire compatibility. Consensus v5 requires zero: the
   * protocol takes no swap skim and the full fee remains in pool reserves.
   */
  protocolFeeBps: bigint;
  /**
   * minimum reserve after swap (bigint string, default "1")
   */
  minReserve: string;
  /**
   * Quote denoms the ZRN price oracle (GetZRNPrice) may price against.
   * Empty (the default) = the oracle selects NO pool — fail-closed, so
   * consumers fall back exactly as when no pool exists (e.g. billing's
   * Tier-1 manual override).
   */
  billingQuoteDenoms: string[];
  /**
   * Unconsumed one-shot counter-denom grants for pool creation. A successful
   * creation removes its denom; empty keeps native pool creation frozen.
   */
  allowedPoolDenoms: string[];
  /**
   * Accounts governance trusts to fund/create admitted pools. Empty keeps
   * native pool creation frozen.
   */
  poolCreators: string[];
}
/**
 * GenesisState defines the liquiditypool module genesis state.
 * @name GenesisState
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.GenesisState
 */
export interface GenesisState {
  params?: Params;
  pools: Pool[];
  twapAccumulators: TWAPAccumulator[];
  /**
   * The next monotonically increasing numeric pool ID. Zero is accepted only
   * as a legacy-import sentinel and is reconstructed from the maximum pool ID.
   */
  nextPoolId: bigint;
  twapObservations: TWAPObservation[];
}
function createBaseParams(): Params {
  return {
    defaultSwapFeeBps: BigInt(0),
    maxPools: BigInt(0),
    minInitialLiquidity: "",
    twapWindowBlocks: BigInt(0),
    protocolFeeBps: BigInt(0),
    minReserve: "",
    billingQuoteDenoms: [],
    allowedPoolDenoms: [],
    poolCreators: []
  };
}
/**
 * Params defines the liquiditypool module parameters.
 * @name Params
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.Params
 */
export const Params = {
  typeUrl: "/zerone.liquiditypool.v1.Params",
  encode(message: Params, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.defaultSwapFeeBps !== BigInt(0)) {
      writer.uint32(8).uint64(message.defaultSwapFeeBps);
    }
    if (message.maxPools !== BigInt(0)) {
      writer.uint32(16).uint64(message.maxPools);
    }
    if (message.minInitialLiquidity !== "") {
      writer.uint32(26).string(message.minInitialLiquidity);
    }
    if (message.twapWindowBlocks !== BigInt(0)) {
      writer.uint32(32).uint64(message.twapWindowBlocks);
    }
    if (message.protocolFeeBps !== BigInt(0)) {
      writer.uint32(40).uint64(message.protocolFeeBps);
    }
    if (message.minReserve !== "") {
      writer.uint32(50).string(message.minReserve);
    }
    for (const v of message.billingQuoteDenoms) {
      writer.uint32(58).string(v!);
    }
    for (const v of message.allowedPoolDenoms) {
      writer.uint32(66).string(v!);
    }
    for (const v of message.poolCreators) {
      writer.uint32(74).string(v!);
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
          message.defaultSwapFeeBps = reader.uint64();
          break;
        case 2:
          message.maxPools = reader.uint64();
          break;
        case 3:
          message.minInitialLiquidity = reader.string();
          break;
        case 4:
          message.twapWindowBlocks = reader.uint64();
          break;
        case 5:
          message.protocolFeeBps = reader.uint64();
          break;
        case 6:
          message.minReserve = reader.string();
          break;
        case 7:
          message.billingQuoteDenoms.push(reader.string());
          break;
        case 8:
          message.allowedPoolDenoms.push(reader.string());
          break;
        case 9:
          message.poolCreators.push(reader.string());
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
    message.defaultSwapFeeBps = object.defaultSwapFeeBps !== undefined && object.defaultSwapFeeBps !== null ? BigInt(object.defaultSwapFeeBps.toString()) : BigInt(0);
    message.maxPools = object.maxPools !== undefined && object.maxPools !== null ? BigInt(object.maxPools.toString()) : BigInt(0);
    message.minInitialLiquidity = object.minInitialLiquidity ?? "";
    message.twapWindowBlocks = object.twapWindowBlocks !== undefined && object.twapWindowBlocks !== null ? BigInt(object.twapWindowBlocks.toString()) : BigInt(0);
    message.protocolFeeBps = object.protocolFeeBps !== undefined && object.protocolFeeBps !== null ? BigInt(object.protocolFeeBps.toString()) : BigInt(0);
    message.minReserve = object.minReserve ?? "";
    message.billingQuoteDenoms = object.billingQuoteDenoms?.map(e => e) || [];
    message.allowedPoolDenoms = object.allowedPoolDenoms?.map(e => e) || [];
    message.poolCreators = object.poolCreators?.map(e => e) || [];
    return message;
  }
};
function createBaseGenesisState(): GenesisState {
  return {
    params: undefined,
    pools: [],
    twapAccumulators: [],
    nextPoolId: BigInt(0),
    twapObservations: []
  };
}
/**
 * GenesisState defines the liquiditypool module genesis state.
 * @name GenesisState
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.GenesisState
 */
export const GenesisState = {
  typeUrl: "/zerone.liquiditypool.v1.GenesisState",
  encode(message: GenesisState, writer: BinaryWriter = BinaryWriter.create()): BinaryWriter {
    if (message.params !== undefined) {
      Params.encode(message.params, writer.uint32(10).fork()).ldelim();
    }
    for (const v of message.pools) {
      Pool.encode(v!, writer.uint32(18).fork()).ldelim();
    }
    for (const v of message.twapAccumulators) {
      TWAPAccumulator.encode(v!, writer.uint32(26).fork()).ldelim();
    }
    if (message.nextPoolId !== BigInt(0)) {
      writer.uint32(32).uint64(message.nextPoolId);
    }
    for (const v of message.twapObservations) {
      TWAPObservation.encode(v!, writer.uint32(42).fork()).ldelim();
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
          message.pools.push(Pool.decode(reader, reader.uint32()));
          break;
        case 3:
          message.twapAccumulators.push(TWAPAccumulator.decode(reader, reader.uint32()));
          break;
        case 4:
          message.nextPoolId = reader.uint64();
          break;
        case 5:
          message.twapObservations.push(TWAPObservation.decode(reader, reader.uint32()));
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
    message.pools = object.pools?.map(e => Pool.fromPartial(e)) || [];
    message.twapAccumulators = object.twapAccumulators?.map(e => TWAPAccumulator.fromPartial(e)) || [];
    message.nextPoolId = object.nextPoolId !== undefined && object.nextPoolId !== null ? BigInt(object.nextPoolId.toString()) : BigInt(0);
    message.twapObservations = object.twapObservations?.map(e => TWAPObservation.fromPartial(e)) || [];
    return message;
  }
};
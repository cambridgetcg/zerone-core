import { Pool, TWAPAccumulator, TWAPObservation } from "./types.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
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
/**
 * Params defines the liquiditypool module parameters.
 * @name Params
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.Params
 */
export declare const Params: {
    typeUrl: string;
    encode(message: Params, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params;
    fromPartial(object: DeepPartial<Params>): Params;
};
/**
 * GenesisState defines the liquiditypool module genesis state.
 * @name GenesisState
 * @package zerone.liquiditypool.v1
 * @see proto type: zerone.liquiditypool.v1.GenesisState
 */
export declare const GenesisState: {
    typeUrl: string;
    encode(message: GenesisState, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GenesisState;
    fromPartial(object: DeepPartial<GenesisState>): GenesisState;
};

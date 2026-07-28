import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
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
     * blocks per emission epoch (0 = disabled)
     */
    emissionEpochBlocks: bigint;
    /**
     * default swap fee in BPS (unused, reserved)
     */
    defaultFeeBps: string;
}
/**
 * GenesisState defines the tokens module's genesis state.
 * @name GenesisState
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.GenesisState
 */
export declare const GenesisState: {
    typeUrl: string;
    encode(message: GenesisState, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GenesisState;
    fromPartial(object: DeepPartial<GenesisState>): GenesisState;
};
/**
 * Params holds the tokens module parameters.
 * All fields default to zero (module is a stub at genesis).
 * @name Params
 * @package zerone.tokens.v1
 * @see proto type: zerone.tokens.v1.Params
 */
export declare const Params: {
    typeUrl: string;
    encode(message: Params, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params;
    fromPartial(object: DeepPartial<Params>): Params;
};

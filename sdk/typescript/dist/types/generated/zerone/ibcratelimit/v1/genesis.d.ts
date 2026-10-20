import { RateLimit } from "./types.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
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
/**
 * Params defines the ibcratelimit module parameters.
 * @name Params
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.Params
 */
export declare const Params: {
    typeUrl: string;
    encode(message: Params, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params;
    fromPartial(object: DeepPartial<Params>): Params;
};
/**
 * GenesisState defines the ibcratelimit module genesis state.
 * @name GenesisState
 * @package zerone.ibcratelimit.v1
 * @see proto type: zerone.ibcratelimit.v1.GenesisState
 */
export declare const GenesisState: {
    typeUrl: string;
    encode(message: GenesisState, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GenesisState;
    fromPartial(object: DeepPartial<GenesisState>): GenesisState;
};

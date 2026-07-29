import { PinnedCreed, CreedCouncilMember } from "./types.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
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
/**
 * GenesisState can seed an optional version-1 creed pin and history.
 * Default and published zerone-1 genesis states omit the pin.
 * @name GenesisState
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.GenesisState
 */
export declare const GenesisState: {
    typeUrl: string;
    encode(message: GenesisState, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GenesisState;
    fromPartial(object: DeepPartial<GenesisState>): GenesisState;
};
/**
 * @name Params
 * @package zerone.creed.v1
 * @see proto type: zerone.creed.v1.Params
 */
export declare const Params: {
    typeUrl: string;
    encode(message: Params, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params;
    fromPartial(object: DeepPartial<Params>): Params;
};

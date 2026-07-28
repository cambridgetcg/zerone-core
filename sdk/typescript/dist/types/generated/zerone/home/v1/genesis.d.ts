import { AgentHome, KeyRegistration } from "./types.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
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
/**
 * @name GenesisState
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.GenesisState
 */
export declare const GenesisState: {
    typeUrl: string;
    encode(message: GenesisState, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GenesisState;
    fromPartial(object: DeepPartial<GenesisState>): GenesisState;
};
/**
 * @name HomeKeySet
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.HomeKeySet
 */
export declare const HomeKeySet: {
    typeUrl: string;
    encode(message: HomeKeySet, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): HomeKeySet;
    fromPartial(object: DeepPartial<HomeKeySet>): HomeKeySet;
};
/**
 * @name Params
 * @package zerone.home.v1
 * @see proto type: zerone.home.v1.Params
 */
export declare const Params: {
    typeUrl: string;
    encode(message: Params, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params;
    fromPartial(object: DeepPartial<Params>): Params;
};

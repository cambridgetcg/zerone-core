import { StratumProperties, Domain, DomainProposal, CrossStratumLink } from "./state.js";
import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * GenesisState defines the ontology module's genesis state.
 * @name GenesisState
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.GenesisState
 */
export interface GenesisState {
    params?: Params;
    strata: StratumProperties[];
    domains: Domain[];
    proposals: DomainProposal[];
    crossLinks: CrossStratumLink[];
}
/**
 * Params defines the ontology module parameters.
 * @name Params
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.Params
 */
export interface Params {
    /**
     * bigint as string (uzrn)
     */
    minProposalStake: string;
    /**
     * blocks
     */
    proposalVotingPeriod: bigint;
    minEndorsements: number;
    /**
     * basis points
     */
    crossStratumDiscount: bigint;
    maxDomainsPerStratum: number;
    allowNewStrata: boolean;
}
/**
 * GenesisState defines the ontology module's genesis state.
 * @name GenesisState
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.GenesisState
 */
export declare const GenesisState: {
    typeUrl: string;
    encode(message: GenesisState, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GenesisState;
    fromPartial(object: DeepPartial<GenesisState>): GenesisState;
};
/**
 * Params defines the ontology module parameters.
 * @name Params
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.Params
 */
export declare const Params: {
    typeUrl: string;
    encode(message: Params, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Params;
    fromPartial(object: DeepPartial<Params>): Params;
};

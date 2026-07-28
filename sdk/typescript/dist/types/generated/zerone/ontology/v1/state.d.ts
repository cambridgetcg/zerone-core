import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * StratumProperties defines the logical properties of a knowledge stratum.
 * @name StratumProperties
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.StratumProperties
 */
export interface StratumProperties {
    stratum: number;
    name: string;
    description: string;
    /**
     * Goedel: is the system complete?
     */
    complete: boolean;
    /**
     * is there an algorithm to decide all statements?
     */
    decidable: boolean;
    /**
     * does Goedel's incompleteness apply?
     */
    goedelApplies: boolean;
    /**
     * "internal", "assumed", "external"
     */
    consistencyProof: string;
    /**
     * basis points (1000000 = 1.0)
     */
    maxConfidence: bigint;
    /**
     * confidence decay rate per day (basis points)
     */
    decayRate: bigint;
}
/**
 * Domain represents a knowledge domain within a stratum.
 * @name Domain
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.Domain
 */
export interface Domain {
    name: string;
    displayName: string;
    description: string;
    stratum: number;
    /**
     * "active", "proposed", "deprecated"
     */
    status: string;
    /**
     * block height
     */
    createdAt: bigint;
    /**
     * block height
     */
    updatedAt: bigint;
    claimCount: bigint;
    factCount: bigint;
    /**
     * bech32 address
     */
    proposedBy: string;
    /**
     * empty = root domain
     */
    parentDomain: string;
    /**
     * tree depth: root=1, child=parent.depth+1, max=5
     */
    depth: number;
}
/**
 * DomainProposal represents a proposal for adding or modifying domains.
 * @name DomainProposal
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.DomainProposal
 */
export interface DomainProposal {
    id: string;
    domain?: Domain;
    /**
     * bech32 address
     */
    proposer: string;
    /**
     * "add", "deprecate", "modify"
     */
    proposalType: string;
    /**
     * bigint as string (uzrn)
     */
    stake: string;
    votesFor: number;
    votesAgainst: number;
    voters: string[];
    /**
     * "active", "passed", "rejected", "expired"
     */
    status: string;
    /**
     * block height
     */
    createdAt: bigint;
    /**
     * block height
     */
    expiresAt: bigint;
}
/**
 * LogicZoneProperties defines the properties of a Godelian logic zone.
 * @name LogicZoneProperties
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.LogicZoneProperties
 */
export interface LogicZoneProperties {
    zone: string;
    complete: boolean;
    decidable: boolean;
    goedelApplies: boolean;
    maxConfidenceBps: bigint;
    description: string;
}
/**
 * CrossStratumLink represents a relationship between domains in different strata.
 * @name CrossStratumLink
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.CrossStratumLink
 */
export interface CrossStratumLink {
    sourceDomain: string;
    targetDomain: string;
    /**
     * "depends_on", "generalizes", "restricts"
     */
    linkType: string;
    /**
     * confidence discount for cross-stratum references (basis points)
     */
    discount: bigint;
}
/**
 * StratumProperties defines the logical properties of a knowledge stratum.
 * @name StratumProperties
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.StratumProperties
 */
export declare const StratumProperties: {
    typeUrl: string;
    encode(message: StratumProperties, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): StratumProperties;
    fromPartial(object: DeepPartial<StratumProperties>): StratumProperties;
};
/**
 * Domain represents a knowledge domain within a stratum.
 * @name Domain
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.Domain
 */
export declare const Domain: {
    typeUrl: string;
    encode(message: Domain, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): Domain;
    fromPartial(object: DeepPartial<Domain>): Domain;
};
/**
 * DomainProposal represents a proposal for adding or modifying domains.
 * @name DomainProposal
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.DomainProposal
 */
export declare const DomainProposal: {
    typeUrl: string;
    encode(message: DomainProposal, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): DomainProposal;
    fromPartial(object: DeepPartial<DomainProposal>): DomainProposal;
};
/**
 * LogicZoneProperties defines the properties of a Godelian logic zone.
 * @name LogicZoneProperties
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.LogicZoneProperties
 */
export declare const LogicZoneProperties: {
    typeUrl: string;
    encode(message: LogicZoneProperties, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): LogicZoneProperties;
    fromPartial(object: DeepPartial<LogicZoneProperties>): LogicZoneProperties;
};
/**
 * CrossStratumLink represents a relationship between domains in different strata.
 * @name CrossStratumLink
 * @package zerone.ontology.v1
 * @see proto type: zerone.ontology.v1.CrossStratumLink
 */
export declare const CrossStratumLink: {
    typeUrl: string;
    encode(message: CrossStratumLink, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): CrossStratumLink;
    fromPartial(object: DeepPartial<CrossStratumLink>): CrossStratumLink;
};

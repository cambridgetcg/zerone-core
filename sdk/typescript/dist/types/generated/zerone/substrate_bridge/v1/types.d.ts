import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * CitationType distinguishes citation strengths for lineage propagation
 * (M6 generalized). Mirrors the ToK relation-type semantics applied
 * across work classes.
 */
export declare enum CitationType {
    CITATION_TYPE_UNSPECIFIED = 0,
    /** CITATION_TYPE_CITES - 1× base weight */
    CITATION_TYPE_CITES = 1,
    /** CITATION_TYPE_SUPPORTS - 2× base weight */
    CITATION_TYPE_SUPPORTS = 2,
    /** CITATION_TYPE_EXTENDS - 3× base weight */
    CITATION_TYPE_EXTENDS = 3,
    /** CITATION_TYPE_REFINES - 3× base weight */
    CITATION_TYPE_REFINES = 4,
    /** CITATION_TYPE_GENERALIZES - 4× base weight */
    CITATION_TYPE_GENERALIZES = 5,
    UNRECOGNIZED = -1
}
export declare function citationTypeFromJSON(object: any): CitationType;
export declare function citationTypeToJSON(object: CitationType): string;
/**
 * ExternalSource is caller-declared off-chain provenance. Consensus requires
 * adapter_id to match the enclosing link and content_hash to have SHA-256
 * width. It does not fetch external content, execute an adapter/compiler, or
 * verify the declared digest against external bytes.
 * @name ExternalSource
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.ExternalSource
 */
export interface ExternalSource {
    adapterId: string;
    /**
     * e.g. Wikipedia article ID
     */
    sourceId: string;
    /**
     * optional; for audit
     */
    sourceUrl: string;
    /**
     * declared SHA-256 digest; width checked only
     */
    contentHash: Uint8Array;
    fetchedAtBlock: bigint;
}
/**
 * AxisProjection is the per-axis recursion-weight contribution of an
 * external work artifact, in the order fixed by USEFUL_WORK.md
 * section "The six recursive axes". Units are uint64 weights, bounded
 * by an adapter's AxisBounds.
 * @name AxisProjection
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.AxisProjection
 */
export interface AxisProjection {
    axisSubstrate: bigint;
    axisVerification: bigint;
    axisClassification: bigint;
    axisAttribution: bigint;
    axisTooling: bigint;
    axisInterface: bigint;
}
/**
 * AxisBounds caps the per-axis projection an adapter is allowed to
 * claim. Gov-approved at adapter registration; enforced at attestation
 * submit.
 * @name AxisBounds
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.AxisBounds
 */
export interface AxisBounds {
    axisSubstrateMax: bigint;
    axisVerificationMax: bigint;
    axisClassificationMax: bigint;
    axisAttributionMax: bigint;
    axisToolingMax: bigint;
    axisInterfaceMax: bigint;
}
/**
 * ExternalSource is caller-declared off-chain provenance. Consensus requires
 * adapter_id to match the enclosing link and content_hash to have SHA-256
 * width. It does not fetch external content, execute an adapter/compiler, or
 * verify the declared digest against external bytes.
 * @name ExternalSource
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.ExternalSource
 */
export declare const ExternalSource: {
    typeUrl: string;
    encode(message: ExternalSource, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ExternalSource;
    fromPartial(object: DeepPartial<ExternalSource>): ExternalSource;
};
/**
 * AxisProjection is the per-axis recursion-weight contribution of an
 * external work artifact, in the order fixed by USEFUL_WORK.md
 * section "The six recursive axes". Units are uint64 weights, bounded
 * by an adapter's AxisBounds.
 * @name AxisProjection
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.AxisProjection
 */
export declare const AxisProjection: {
    typeUrl: string;
    encode(message: AxisProjection, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): AxisProjection;
    fromPartial(object: DeepPartial<AxisProjection>): AxisProjection;
};
/**
 * AxisBounds caps the per-axis projection an adapter is allowed to
 * claim. Gov-approved at adapter registration; enforced at attestation
 * submit.
 * @name AxisBounds
 * @package zerone.substrate_bridge.v1
 * @see proto type: zerone.substrate_bridge.v1.AxisBounds
 */
export declare const AxisBounds: {
    typeUrl: string;
    encode(message: AxisBounds, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): AxisBounds;
    fromPartial(object: DeepPartial<AxisBounds>): AxisBounds;
};

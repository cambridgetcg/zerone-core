import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * BasisPoints represents a value on a 1,000,000 scale (100% = 1,000,000).
 * ALL Zerone modules use this scale. No exceptions.
 * @name BasisPoints
 * @package zerone.common.v1
 * @see proto type: zerone.common.v1.BasisPoints
 */
export interface BasisPoints {
    value: bigint;
}
/**
 * RevenueSplit defines how protocol revenue is allocated — governance-adjustable.
 * @name RevenueSplit
 * @package zerone.common.v1
 * @see proto type: zerone.common.v1.RevenueSplit
 */
export interface RevenueSplit {
    /**
     * default 550,000 (55%)
     */
    contributorBps: bigint;
    /**
     * default 220,000 (22%)
     */
    protocolBps: bigint;
    /**
     * default  33,300 (3.33%)
     */
    researchBps: bigint;
    /**
     * default 196,700 (19.67%) — bug bounties, truth discovery, protocol development
     */
    developmentBps: bigint;
}
/**
 * ProtocolSubSplit defines how the protocol share is divided.
 * @name ProtocolSubSplit
 * @package zerone.common.v1
 * @see proto type: zerone.common.v1.ProtocolSubSplit
 */
export interface ProtocolSubSplit {
    /**
     * default 500,000 (50% of protocol)
     */
    citationBps: bigint;
    /**
     * default 300,000 (30% of protocol)
     */
    verificationBps: bigint;
    /**
     * default 200,000 (20% of protocol)
     */
    treasuryBps: bigint;
}
/**
 * BasisPoints represents a value on a 1,000,000 scale (100% = 1,000,000).
 * ALL Zerone modules use this scale. No exceptions.
 * @name BasisPoints
 * @package zerone.common.v1
 * @see proto type: zerone.common.v1.BasisPoints
 */
export declare const BasisPoints: {
    typeUrl: string;
    encode(message: BasisPoints, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): BasisPoints;
    fromPartial(object: DeepPartial<BasisPoints>): BasisPoints;
};
/**
 * RevenueSplit defines how protocol revenue is allocated — governance-adjustable.
 * @name RevenueSplit
 * @package zerone.common.v1
 * @see proto type: zerone.common.v1.RevenueSplit
 */
export declare const RevenueSplit: {
    typeUrl: string;
    encode(message: RevenueSplit, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): RevenueSplit;
    fromPartial(object: DeepPartial<RevenueSplit>): RevenueSplit;
};
/**
 * ProtocolSubSplit defines how the protocol share is divided.
 * @name ProtocolSubSplit
 * @package zerone.common.v1
 * @see proto type: zerone.common.v1.ProtocolSubSplit
 */
export declare const ProtocolSubSplit: {
    typeUrl: string;
    encode(message: ProtocolSubSplit, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): ProtocolSubSplit;
    fromPartial(object: DeepPartial<ProtocolSubSplit>): ProtocolSubSplit;
};

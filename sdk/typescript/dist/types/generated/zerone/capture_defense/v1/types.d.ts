import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
/**
 * IsolatedReputation tracks a validator's reputation at a single scope level.
 * @name IsolatedReputation
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.IsolatedReputation
 */
export interface IsolatedReputation {
    validator: string;
    /**
     * "global", stratum name, or domain ID
     */
    scope: string;
    /**
     * 0–1_000_000 BPS
     */
    score: bigint;
    verifications: bigint;
    lastUpdatedBlock: bigint;
}
/**
 * GlobalReputation is a validator's chain-wide reputation.
 * @name GlobalReputation
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.GlobalReputation
 */
export interface GlobalReputation {
    validator: string;
    score: bigint;
    totalVerifications: bigint;
    lastUpdatedBlock: bigint;
}
/**
 * StratumReputation is a validator's reputation within a knowledge stratum.
 * @name StratumReputation
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.StratumReputation
 */
export interface StratumReputation {
    validator: string;
    stratum: string;
    score: bigint;
    verifications: bigint;
    lastUpdatedBlock: bigint;
}
/**
 * DomainReputation is a validator's reputation for a specific domain.
 * @name DomainReputation
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.DomainReputation
 */
export interface DomainReputation {
    validator: string;
    domain: string;
    score: bigint;
    verifications: bigint;
    lastUpdatedBlock: bigint;
}
/**
 * CaptureMetrics holds computed capture risk analysis for a domain.
 * @name CaptureMetrics
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.CaptureMetrics
 */
export interface CaptureMetrics {
    domain: string;
    /**
     * 0–1_000_000 BPS
     */
    herfindahlIndex: bigint;
    /**
     * 0–1_000_000 BPS
     */
    timingCorrelation: bigint;
    /**
     * 0–1_000_000 BPS
     */
    verdictCorrelation: bigint;
    /**
     * 0–1_000_000 BPS
     */
    top3Share: bigint;
    /**
     * composite 0–1_000_000
     */
    riskScore: bigint;
    totalParticipations: bigint;
    analyzedAtBlock: bigint;
    flagged: boolean;
}
/**
 * VerificationHistoryEntry records one verification round for detection.
 * @name VerificationHistoryEntry
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.VerificationHistoryEntry
 */
export interface VerificationHistoryEntry {
    domain: string;
    roundId: string;
    validators: string[];
    /**
     * true = approved, false = rejected
     */
    verdicts: boolean[];
    submitBlocks: bigint[];
    blockHeight: bigint;
}
/**
 * CrossStratumRequirement defines cross-stratum validation rules.
 * @name CrossStratumRequirement
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.CrossStratumRequirement
 */
export interface CrossStratumRequirement {
    targetStratum: string;
    requiredStrata: string[];
    minValidatorsPerStratum: bigint;
}
/**
 * IsolatedReputation tracks a validator's reputation at a single scope level.
 * @name IsolatedReputation
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.IsolatedReputation
 */
export declare const IsolatedReputation: {
    typeUrl: string;
    encode(message: IsolatedReputation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): IsolatedReputation;
    fromPartial(object: DeepPartial<IsolatedReputation>): IsolatedReputation;
};
/**
 * GlobalReputation is a validator's chain-wide reputation.
 * @name GlobalReputation
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.GlobalReputation
 */
export declare const GlobalReputation: {
    typeUrl: string;
    encode(message: GlobalReputation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): GlobalReputation;
    fromPartial(object: DeepPartial<GlobalReputation>): GlobalReputation;
};
/**
 * StratumReputation is a validator's reputation within a knowledge stratum.
 * @name StratumReputation
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.StratumReputation
 */
export declare const StratumReputation: {
    typeUrl: string;
    encode(message: StratumReputation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): StratumReputation;
    fromPartial(object: DeepPartial<StratumReputation>): StratumReputation;
};
/**
 * DomainReputation is a validator's reputation for a specific domain.
 * @name DomainReputation
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.DomainReputation
 */
export declare const DomainReputation: {
    typeUrl: string;
    encode(message: DomainReputation, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): DomainReputation;
    fromPartial(object: DeepPartial<DomainReputation>): DomainReputation;
};
/**
 * CaptureMetrics holds computed capture risk analysis for a domain.
 * @name CaptureMetrics
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.CaptureMetrics
 */
export declare const CaptureMetrics: {
    typeUrl: string;
    encode(message: CaptureMetrics, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): CaptureMetrics;
    fromPartial(object: DeepPartial<CaptureMetrics>): CaptureMetrics;
};
/**
 * VerificationHistoryEntry records one verification round for detection.
 * @name VerificationHistoryEntry
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.VerificationHistoryEntry
 */
export declare const VerificationHistoryEntry: {
    typeUrl: string;
    encode(message: VerificationHistoryEntry, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): VerificationHistoryEntry;
    fromPartial(object: DeepPartial<VerificationHistoryEntry>): VerificationHistoryEntry;
};
/**
 * CrossStratumRequirement defines cross-stratum validation rules.
 * @name CrossStratumRequirement
 * @package zerone.capture_defense.v1
 * @see proto type: zerone.capture_defense.v1.CrossStratumRequirement
 */
export declare const CrossStratumRequirement: {
    typeUrl: string;
    encode(message: CrossStratumRequirement, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): CrossStratumRequirement;
    fromPartial(object: DeepPartial<CrossStratumRequirement>): CrossStratumRequirement;
};

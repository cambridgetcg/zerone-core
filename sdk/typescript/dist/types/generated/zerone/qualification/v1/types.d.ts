import { BinaryReader, BinaryWriter } from "../../../binary.js";
import { DeepPartial } from "../../../helpers.js";
export declare enum QualificationPathway {
    QUALIFICATION_PATHWAY_UNSPECIFIED = 0,
    QUALIFICATION_PATHWAY_STAKE = 1,
    QUALIFICATION_PATHWAY_TRACK_RECORD = 2,
    QUALIFICATION_PATHWAY_CROSS_REFERENCE = 3,
    QUALIFICATION_PATHWAY_INHERITANCE = 4,
    UNRECOGNIZED = -1
}
export declare function qualificationPathwayFromJSON(object: any): QualificationPathway;
export declare function qualificationPathwayToJSON(object: QualificationPathway): string;
export declare enum QualificationStatus {
    QUALIFICATION_STATUS_UNSPECIFIED = 0,
    QUALIFICATION_STATUS_ACTIVE = 1,
    QUALIFICATION_STATUS_PROBATIONARY = 2,
    QUALIFICATION_STATUS_SUSPENDED = 3,
    QUALIFICATION_STATUS_REVOKED = 4,
    QUALIFICATION_STATUS_EXPIRED = 5,
    UNRECOGNIZED = -1
}
export declare function qualificationStatusFromJSON(object: any): QualificationStatus;
export declare function qualificationStatusToJSON(object: QualificationStatus): string;
/**
 * @name QualificationMetrics
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.QualificationMetrics
 */
export interface QualificationMetrics {
    totalVerifications: bigint;
    correctVerifications: bigint;
    /**
     * 1,000,000 = 100%
     */
    accuracyBps: bigint;
    lastVerificationBlock: bigint;
}
/**
 * @name DomainQualification
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.DomainQualification
 */
export interface DomainQualification {
    /**
     * bech32 validator address
     */
    validator: string;
    domain: string;
    pathway: QualificationPathway;
    status: QualificationStatus;
    /**
     * 1-100 qualification weight
     */
    weight: number;
    /**
     * domain stratum level (lower = more foundational)
     */
    stratum: number;
    /**
     * uzrn locked for stake-pathway
     */
    stakedAmount: string;
    /**
     * block height
     */
    grantedAt: bigint;
    /**
     * block height (0 = no expiry)
     */
    expiresAt: bigint;
    /**
     * last renewal block
     */
    renewedAt: bigint;
    metrics?: QualificationMetrics;
    /**
     * for inheritance pathway
     */
    parentDomain: string;
    /**
     * for cross-reference pathway
     */
    crossRefDomain: string;
    endorsementCount: number;
    /**
     * block height, 0 if not on probation
     */
    probationUntil: bigint;
}
/**
 * @name QualificationEndorsement
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.QualificationEndorsement
 */
export interface QualificationEndorsement {
    id: bigint;
    /**
     * validator being endorsed
     */
    qualificationValidator: string;
    qualificationDomain: string;
    /**
     * bech32
     */
    endorser: string;
    reason: string;
    /**
     * endorsement weight (1-100)
     */
    weight: number;
    createdAt: bigint;
    /**
     * 0 = no expiry
     */
    expiresAt: bigint;
}
/**
 * @name QualificationMetrics
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.QualificationMetrics
 */
export declare const QualificationMetrics: {
    typeUrl: string;
    encode(message: QualificationMetrics, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): QualificationMetrics;
    fromPartial(object: DeepPartial<QualificationMetrics>): QualificationMetrics;
};
/**
 * @name DomainQualification
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.DomainQualification
 */
export declare const DomainQualification: {
    typeUrl: string;
    encode(message: DomainQualification, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): DomainQualification;
    fromPartial(object: DeepPartial<DomainQualification>): DomainQualification;
};
/**
 * @name QualificationEndorsement
 * @package zerone.qualification.v1
 * @see proto type: zerone.qualification.v1.QualificationEndorsement
 */
export declare const QualificationEndorsement: {
    typeUrl: string;
    encode(message: QualificationEndorsement, writer?: BinaryWriter): BinaryWriter;
    decode(input: BinaryReader | Uint8Array, length?: number): QualificationEndorsement;
    fromPartial(object: DeepPartial<QualificationEndorsement>): QualificationEndorsement;
};

declare const uint64DecimalBrand: unique symbol;
export declare const IN_TOTO_STATEMENT_V1_TYPE: "https://in-toto.io/Statement/v1";
export declare const ZERONE_TRAINING_PROVENANCE_V1_PREDICATE_TYPE: "https://github.com/cambridgetcg/zerone-core/blob/main/docs/specs/attestations/training-provenance-v1.md";
/**
 * Defensive consumer limits. These are SDK parsing bounds, not consensus
 * rules, and intentionally make no claim about historical on-chain IDs.
 */
export declare const ZERONE_PROVENANCE_LIMITS: Readonly<{
    readonly maxJsonBytes: 1048576;
    readonly maxManifestIdBytes: 512;
    readonly maxChainIdBytes: 128;
    readonly maxPipelineIdBytes: 512;
    readonly maxDomainBytes: 256;
    readonly maxDomains: 2048;
    readonly maxTrustExplanationBytes: 8192;
}>;
export type Uint64Decimal = string & {
    readonly [uint64DecimalBrand]: true;
};
export type ZeroneSealedManifestStatus = "MANIFEST_STATUS_FINALIZED" | "MANIFEST_STATUS_ATTESTED" | "MANIFEST_STATUS_SUPERSEDED";
export type ZeroneTrustGrade = "A" | "B" | "C" | "F";
export interface ZeroneProvenanceDomainCoverage {
    readonly domain: string;
    readonly factCount: Uint64Decimal;
    readonly avgQualifiedWeight: Uint64Decimal;
    readonly activeVoterCount: number;
}
export interface ZeroneProvenanceCertificate {
    readonly manifestId: string;
    readonly pipelineId: string;
    readonly merkleRoot: string;
    readonly factCount: Uint64Decimal;
    readonly finalizedAtBlock: Uint64Decimal;
    readonly status: ZeroneSealedManifestStatus;
    readonly domains: readonly ZeroneProvenanceDomainCoverage[];
    readonly privilegedActionCount: number;
    readonly incidentCount: number;
    readonly cartelResolutionCount: number;
    readonly trustGrade: ZeroneTrustGrade;
    readonly trustExplanation: string;
    readonly computedAtBlock: Uint64Decimal;
    readonly sourceChainId: string;
}
export interface ZeroneInTotoSubject {
    readonly name: string;
    readonly digest: Readonly<{
        sha256: string;
    }>;
}
export interface UnsignedZeroneInTotoStatement {
    readonly _type: typeof IN_TOTO_STATEMENT_V1_TYPE;
    readonly subject: readonly [ZeroneInTotoSubject];
    readonly predicateType: typeof ZERONE_TRAINING_PROVENANCE_V1_PREDICATE_TYPE;
    readonly predicate: Readonly<{
        sourceChainId: string;
        observedOnChainId: string;
        certificate: ZeroneProvenanceCertificate;
    }>;
}
export interface ZeroneProvenanceExpectations {
    /**
     * The manifest ID selected by the application, not a value learned from the
     * untrusted statement.
     */
    readonly manifestId: string;
    /**
     * The chain ID obtained from trusted connection configuration.
     */
    readonly observedOnChainId: string;
    /**
     * Optional origin-chain pin for applications that require one.
     */
    readonly sourceChainId?: string;
}
export interface ParsedUnsignedZeroneProvenance {
    readonly statement: UnsignedZeroneInTotoStatement;
    readonly assurance: Readonly<{
        authenticated: false;
        signatureVerified: false;
        currentStateProjection: true;
        subjectDigestScope: "included-id-set";
    }>;
}
export type ProvenanceParseErrorCode = "INVALID_JSON" | "INPUT_TOO_LARGE" | "INVALID_SHAPE" | "UNSUPPORTED_PROFILE" | "EXPECTATION_MISMATCH";
export declare class ProvenanceParseError extends Error {
    readonly code: ProvenanceParseErrorCode;
    readonly path: string;
    constructor(code: ProvenanceParseErrorCode, path: string, message: string);
}
/**
 * Parses Zerone's unsigned in-toto Statement v1 query profile without doing
 * network I/O, dereferencing predicate URLs, or authenticating the payload.
 *
 * The caller must supply the manifest and serving-chain context it expected.
 * A successful parse proves only that the JSON is a coherent instance of this
 * bounded profile; it does not prove signature validity or predicate truth.
 */
export declare function parseUnsignedZeroneInTotoStatement(json: string, expectations: ZeroneProvenanceExpectations): ParsedUnsignedZeroneProvenance;
export {};

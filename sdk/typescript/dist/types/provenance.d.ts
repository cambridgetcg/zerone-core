declare const uint64DecimalBrand: unique symbol;
export declare const IN_TOTO_STATEMENT_V1_TYPE: "https://in-toto.io/Statement/v1";
export declare const ZERONE_TRAINING_PROVENANCE_V1_PREDICATE_TYPE: "https://github.com/cambridgetcg/zerone-core/blob/394bbef01df1b131223b1e874d554932d8dcd87c/docs/specs/attestations/training-provenance-v1.md";
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
export declare const ZERONE_TRAINING_PROVENANCE_V2_PREDICATE_TYPE: "https://github.com/cambridgetcg/zerone-core/blob/43f3a04178acab749758e383e064154854ad8b96/docs/specs/attestations/training-provenance-v2.md";
/**
 * Says which of the three counted warning signals are present. It is not
 * evidence of review: "A" means none of the signals is present.
 */
export type ZeroneWarningGrade = "A" | "B" | "C" | "F";
/**
 * Review coverage of the directly included fact IDs only. Included trace,
 * pair, drift-augmentation and normative-commitment IDs, and a composed
 * manifest's parent, are not counted.
 */
export type ZeroneReviewState = "reviewed" | "partly_reviewed" | "unreviewed" | "empty";
export interface ZeroneProvenanceCertificateV2 {
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
    /** The v1 rubric, renamed. Must not be read as review. */
    readonly warningGrade: ZeroneWarningGrade;
    readonly warningExplanation: string;
    readonly computedAtBlock: Uint64Decimal;
    readonly sourceChainId: string;
    /** Included fact IDs with a retained round naming their claim and at least one reveal. */
    readonly reviewedFactCount: Uint64Decimal;
    /** Every other included fact ID, missing ones included. */
    readonly unreviewedFactCount: Uint64Decimal;
    /** Unreviewed facts without a claim that carry the doctrine category or method. */
    readonly doctrineFactCount: Uint64Decimal;
    /** Included fact IDs with no fact in the store, counted as unreviewed. */
    readonly missingFactCount: Uint64Decimal;
    readonly reviewState: ZeroneReviewState;
    readonly reviewExplanation: string;
}
export interface UnsignedZeroneInTotoStatementV2 {
    readonly _type: typeof IN_TOTO_STATEMENT_V1_TYPE;
    readonly subject: readonly [ZeroneInTotoSubject];
    readonly predicateType: typeof ZERONE_TRAINING_PROVENANCE_V2_PREDICATE_TYPE;
    readonly predicate: Readonly<{
        sourceChainId: string;
        observedOnChainId: string;
        certificate: ZeroneProvenanceCertificateV2;
    }>;
}
export interface ParsedUnsignedZeroneProvenanceV2 {
    readonly statement: UnsignedZeroneInTotoStatementV2;
    readonly assurance: ParsedUnsignedZeroneProvenance["assurance"];
}
/**
 * Parses Zerone's unsigned in-toto Statement v1 carrying training-provenance
 * predicate v2, without network I/O, URL dereferencing, or authentication.
 *
 * It is as strict as the v1 parser: exact keys, caller-pinned manifest and
 * serving chain, and the warning grade, review state and both explanations
 * recomputed from the counts. A v1 statement is refused here, and a v2
 * statement is refused by the v1 parser.
 */
export declare function parseUnsignedZeroneInTotoStatementV2(json: string, expectations: ZeroneProvenanceExpectations): ParsedUnsignedZeroneProvenanceV2;
export {};

declare const uint64DecimalBrand: unique symbol;

export const IN_TOTO_STATEMENT_V1_TYPE =
  "https://in-toto.io/Statement/v1" as const;
export const ZERONE_TRAINING_PROVENANCE_V1_PREDICATE_TYPE =
  "https://github.com/cambridgetcg/zerone-core/blob/394bbef01df1b131223b1e874d554932d8dcd87c/docs/specs/attestations/training-provenance-v1.md" as const;

/**
 * Defensive consumer limits. These are SDK parsing bounds, not consensus
 * rules, and intentionally make no claim about historical on-chain IDs.
 */
export const ZERONE_PROVENANCE_LIMITS = Object.freeze({
  maxJsonBytes: 1_048_576,
  maxManifestIdBytes: 512,
  maxChainIdBytes: 128,
  maxPipelineIdBytes: 512,
  maxDomainBytes: 256,
  maxDomains: 2_048,
  maxTrustExplanationBytes: 8_192,
} as const);

export type Uint64Decimal = string & {
  readonly [uint64DecimalBrand]: true;
};

export type ZeroneSealedManifestStatus =
  | "MANIFEST_STATUS_FINALIZED"
  | "MANIFEST_STATUS_ATTESTED"
  | "MANIFEST_STATUS_SUPERSEDED";

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
  readonly digest: Readonly<{ sha256: string }>;
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

export type ProvenanceParseErrorCode =
  | "INVALID_JSON"
  | "INPUT_TOO_LARGE"
  | "INVALID_SHAPE"
  | "UNSUPPORTED_PROFILE"
  | "EXPECTATION_MISMATCH";

export class ProvenanceParseError extends Error {
  readonly code: ProvenanceParseErrorCode;
  readonly path: string;

  constructor(code: ProvenanceParseErrorCode, path: string, message: string) {
    super(`${path}: ${message}`);
    this.name = "ProvenanceParseError";
    this.code = code;
    this.path = path;
  }
}

const UINT64_MAX = 18_446_744_073_709_551_615n;
const UINT32_MAX = 4_294_967_295;
const SHA256_PATTERN = /^[0-9a-f]{64}$/;
const UINT_DECIMAL_PATTERN = /^(?:0|[1-9][0-9]*)$/;
const SEALED_STATUSES = new Set<ZeroneSealedManifestStatus>([
  "MANIFEST_STATUS_FINALIZED",
  "MANIFEST_STATUS_ATTESTED",
  "MANIFEST_STATUS_SUPERSEDED",
]);
const TRUST_GRADES = new Set<ZeroneTrustGrade>(["A", "B", "C", "F"]);

const TOP_LEVEL_KEYS = ["_type", "subject", "predicateType", "predicate"] as const;
const SUBJECT_KEYS = ["name", "digest"] as const;
const DIGEST_KEYS = ["sha256"] as const;
const PREDICATE_KEYS = [
  "sourceChainId",
  "observedOnChainId",
  "certificate",
] as const;
const CERTIFICATE_KEYS = [
  "manifestId",
  "pipelineId",
  "merkleRoot",
  "factCount",
  "finalizedAtBlock",
  "status",
  "domains",
  "privilegedActionCount",
  "incidentCount",
  "cartelResolutionCount",
  "trustGrade",
  "trustExplanation",
  "computedAtBlock",
  "sourceChainId",
] as const;
const DOMAIN_KEYS = [
  "domain",
  "factCount",
  "avgQualifiedWeight",
  "activeVoterCount",
] as const;

function fail(
  code: ProvenanceParseErrorCode,
  path: string,
  message: string,
): never {
  throw new ProvenanceParseError(code, path, message);
}

function utf8Length(value: string): number {
  return new TextEncoder().encode(value).length;
}

function hasIllFormedUtf16(value: string): boolean {
  for (let index = 0; index < value.length; index += 1) {
    const codeUnit = value.charCodeAt(index);
    if (codeUnit >= 0xd800 && codeUnit <= 0xdbff) {
      const next = value.charCodeAt(index + 1);
      if (!(next >= 0xdc00 && next <= 0xdfff)) {
        return true;
      }
      index += 1;
    } else if (codeUnit >= 0xdc00 && codeUnit <= 0xdfff) {
      return true;
    }
  }
  return false;
}

function boundedString(
  value: unknown,
  path: string,
  maxBytes: number,
  allowEmpty = false,
): string {
  if (typeof value !== "string") {
    return fail("INVALID_SHAPE", path, "must be a string");
  }
  if (!allowEmpty && value.length === 0) {
    return fail("INVALID_SHAPE", path, "must not be empty");
  }
  if (hasIllFormedUtf16(value)) {
    return fail("INVALID_SHAPE", path, "must contain well-formed Unicode");
  }
  if (utf8Length(value) > maxBytes) {
    return fail("INPUT_TOO_LARGE", path, `must be at most ${maxBytes} UTF-8 bytes`);
  }
  return value;
}

function record(value: unknown, path: string): Record<string, unknown> {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    return fail("INVALID_SHAPE", path, "must be an object");
  }
  return value as Record<string, unknown>;
}

function exactKeys(
  value: Record<string, unknown>,
  allowed: readonly string[],
  path: string,
): void {
  const allowedSet = new Set(allowed);
  for (const key of Object.keys(value)) {
    if (!allowedSet.has(key)) {
      fail("INVALID_SHAPE", `${path}.${key}`, "is not part of this profile");
    }
  }
}

function canonicalUint64(
  value: unknown,
  path: string,
  defaultValue = "0",
): Uint64Decimal {
  const candidate = value === undefined ? defaultValue : value;
  if (typeof candidate !== "string" || !UINT_DECIMAL_PATTERN.test(candidate)) {
    return fail(
      "INVALID_SHAPE",
      path,
      "must be a canonical decimal uint64 string",
    );
  }
  if (candidate.length > 20 || BigInt(candidate) > UINT64_MAX) {
    return fail("INVALID_SHAPE", path, "exceeds uint64");
  }
  return candidate as Uint64Decimal;
}

function uint32(value: unknown, path: string): number {
  if (value === undefined) return 0;
  if (
    typeof value !== "number" ||
    !Number.isInteger(value) ||
    value < 0 ||
    value > UINT32_MAX
  ) {
    return fail("INVALID_SHAPE", path, "must be an integer uint32");
  }
  return value;
}

function expectedGrade(
  privilegedActionCount: number,
  incidentCount: number,
  cartelResolutionCount: number,
): ZeroneTrustGrade {
  if (cartelResolutionCount > 0) return "F";
  if (privilegedActionCount === 0 && incidentCount === 0) return "A";
  if (incidentCount === 0 && privilegedActionCount <= 2) return "B";
  return "C";
}

function expectedTrustExplanation(
  privilegedActionCount: number,
  incidentCount: number,
  cartelResolutionCount: number,
): string {
  if (cartelResolutionCount > 0) {
    return `cartel_resolutions=${cartelResolutionCount} in covered domains; this manifest's adjudication panel was demonstrably compromised`;
  }
  if (privilegedActionCount === 0 && incidentCount === 0) {
    return "no privileged actions touched the manifest's facts; no incidents touched the knowledge module; no cartel resolutions in covered domains";
  }
  if (incidentCount === 0 && privilegedActionCount <= 2) {
    return `privileged_action_count=${privilegedActionCount} affecting manifest facts; manifest is largely unintervened`;
  }
  return `privileged_action_count=${privilegedActionCount}, incident_count=${incidentCount}, cartel_resolution_count=${cartelResolutionCount} — yellow flags accumulating; downstream consumer should review the audit trail before relying on this manifest`;
}

function parseDomains(value: unknown): readonly ZeroneProvenanceDomainCoverage[] {
  if (value === undefined) return Object.freeze([]);
  if (!Array.isArray(value)) {
    return fail("INVALID_SHAPE", "$.predicate.certificate.domains", "must be an array");
  }
  if (value.length > ZERONE_PROVENANCE_LIMITS.maxDomains) {
    return fail(
      "INPUT_TOO_LARGE",
      "$.predicate.certificate.domains",
      `must contain at most ${ZERONE_PROVENANCE_LIMITS.maxDomains} entries`,
    );
  }

  const seenDomains = new Set<string>();
  const domains = value.map((entry, index) => {
    const path = `$.predicate.certificate.domains[${index}]`;
    const object = record(entry, path);
    exactKeys(object, DOMAIN_KEYS, path);
    const domain = boundedString(
      object.domain,
      `${path}.domain`,
      ZERONE_PROVENANCE_LIMITS.maxDomainBytes,
    );
    if (seenDomains.has(domain)) {
      fail("INVALID_SHAPE", `${path}.domain`, "must be unique");
    }
    seenDomains.add(domain);
    return Object.freeze({
      domain,
      factCount: canonicalUint64(object.factCount, `${path}.factCount`),
      avgQualifiedWeight: canonicalUint64(
        object.avgQualifiedWeight,
        `${path}.avgQualifiedWeight`,
      ),
      activeVoterCount: uint32(
        object.activeVoterCount,
        `${path}.activeVoterCount`,
      ),
    });
  });
  return Object.freeze(domains);
}

function pathEscape(value: string): string {
  const hex = "0123456789ABCDEF";
  let escaped = "";
  for (const byte of new TextEncoder().encode(value)) {
    const isAlphaNumeric =
      (byte >= 0x30 && byte <= 0x39) ||
      (byte >= 0x41 && byte <= 0x5a) ||
      (byte >= 0x61 && byte <= 0x7a);
    const isAllowedPunctuation =
      byte === 0x2d || // -
      byte === 0x2e || // .
      byte === 0x5f || // _
      byte === 0x7e || // ~
      byte === 0x24 || // $
      byte === 0x26 || // &
      byte === 0x2b || // +
      byte === 0x3a || // :
      byte === 0x3d || // =
      byte === 0x40; // @
    if (isAlphaNumeric || isAllowedPunctuation) {
      escaped += String.fromCharCode(byte);
    } else {
      escaped += `%${hex[(byte >> 4) & 0x0f]}${hex[byte & 0x0f]}`;
    }
  }
  return escaped;
}

function parseCertificate(
  value: unknown,
  sourceChainId: string,
  manifestId: string,
  subjectSha256: string,
): ZeroneProvenanceCertificate {
  const path = "$.predicate.certificate";
  const object = record(value, path);
  exactKeys(object, CERTIFICATE_KEYS, path);

  const certificateManifestId = boundedString(
    object.manifestId,
    `${path}.manifestId`,
    ZERONE_PROVENANCE_LIMITS.maxManifestIdBytes,
  );
  if (certificateManifestId !== manifestId) {
    fail(
      "EXPECTATION_MISMATCH",
      `${path}.manifestId`,
      "does not match the expected manifest ID",
    );
  }

  const certificateSourceChainId = boundedString(
    object.sourceChainId,
    `${path}.sourceChainId`,
    ZERONE_PROVENANCE_LIMITS.maxChainIdBytes,
  );
  if (certificateSourceChainId !== sourceChainId) {
    fail(
      "INVALID_SHAPE",
      `${path}.sourceChainId`,
      "does not match predicate.sourceChainId",
    );
  }

  const merkleRoot = boundedString(object.merkleRoot, `${path}.merkleRoot`, 64);
  if (!SHA256_PATTERN.test(merkleRoot)) {
    fail(
      "INVALID_SHAPE",
      `${path}.merkleRoot`,
      "must be a lowercase 32-byte SHA-256 hex digest",
    );
  }
  if (merkleRoot !== subjectSha256) {
    fail(
      "INVALID_SHAPE",
      `${path}.merkleRoot`,
      "does not match subject[0].digest.sha256",
    );
  }

  const status = boundedString(object.status, `${path}.status`, 64);
  if (!SEALED_STATUSES.has(status as ZeroneSealedManifestStatus)) {
    fail(
      "UNSUPPORTED_PROFILE",
      `${path}.status`,
      "must be FINALIZED, ATTESTED, or SUPERSEDED",
    );
  }

  const privilegedActionCount = uint32(
    object.privilegedActionCount,
    `${path}.privilegedActionCount`,
  );
  const incidentCount = uint32(object.incidentCount, `${path}.incidentCount`);
  const cartelResolutionCount = uint32(
    object.cartelResolutionCount,
    `${path}.cartelResolutionCount`,
  );
  const trustGrade = boundedString(object.trustGrade, `${path}.trustGrade`, 1);
  if (!TRUST_GRADES.has(trustGrade as ZeroneTrustGrade)) {
    fail("INVALID_SHAPE", `${path}.trustGrade`, "must be A, B, C, or F");
  }
  const computedGrade = expectedGrade(
    privilegedActionCount,
    incidentCount,
    cartelResolutionCount,
  );
  if (trustGrade !== computedGrade) {
    fail(
      "INVALID_SHAPE",
      `${path}.trustGrade`,
      `does not match the v1 count rubric (expected ${computedGrade})`,
    );
  }
  const trustExplanation = boundedString(
    object.trustExplanation,
    `${path}.trustExplanation`,
    ZERONE_PROVENANCE_LIMITS.maxTrustExplanationBytes,
  );
  if (
    trustExplanation !==
    expectedTrustExplanation(
      privilegedActionCount,
      incidentCount,
      cartelResolutionCount,
    )
  ) {
    fail(
      "INVALID_SHAPE",
      `${path}.trustExplanation`,
      "does not match the v1 count rubric",
    );
  }

  const domains = parseDomains(object.domains);
  const factCount = canonicalUint64(object.factCount, `${path}.factCount`);
  const coveredFactCount = domains.reduce(
    (sum, domain) => sum + BigInt(domain.factCount),
    0n,
  );
  if (coveredFactCount > BigInt(factCount)) {
    fail(
      "INVALID_SHAPE",
      `${path}.domains`,
      "covered domain fact counts exceed certificate.factCount",
    );
  }

  return Object.freeze({
    manifestId: certificateManifestId,
    pipelineId:
      object.pipelineId === undefined
        ? ""
        : boundedString(
            object.pipelineId,
            `${path}.pipelineId`,
            ZERONE_PROVENANCE_LIMITS.maxPipelineIdBytes,
            true,
          ),
    merkleRoot,
    factCount,
    finalizedAtBlock: canonicalUint64(
      object.finalizedAtBlock,
      `${path}.finalizedAtBlock`,
    ),
    status: status as ZeroneSealedManifestStatus,
    domains,
    privilegedActionCount,
    incidentCount,
    cartelResolutionCount,
    trustGrade: trustGrade as ZeroneTrustGrade,
    trustExplanation,
    computedAtBlock: canonicalUint64(
      object.computedAtBlock,
      `${path}.computedAtBlock`,
    ),
    sourceChainId: certificateSourceChainId,
  });
}

function validateExpectations(
  expectations: ZeroneProvenanceExpectations,
): Readonly<Required<Pick<ZeroneProvenanceExpectations, "manifestId" | "observedOnChainId">> &
  Pick<ZeroneProvenanceExpectations, "sourceChainId">> {
  const object = record(expectations, "$expectations");
  exactKeys(
    object,
    ["manifestId", "observedOnChainId", "sourceChainId"],
    "$expectations",
  );
  return Object.freeze({
    manifestId: boundedString(
      object.manifestId,
      "$expectations.manifestId",
      ZERONE_PROVENANCE_LIMITS.maxManifestIdBytes,
    ),
    observedOnChainId: boundedString(
      object.observedOnChainId,
      "$expectations.observedOnChainId",
      ZERONE_PROVENANCE_LIMITS.maxChainIdBytes,
    ),
    ...(object.sourceChainId === undefined
      ? {}
      : {
          sourceChainId: boundedString(
            object.sourceChainId,
            "$expectations.sourceChainId",
            ZERONE_PROVENANCE_LIMITS.maxChainIdBytes,
          ),
        }),
  });
}

/**
 * Parses Zerone's unsigned in-toto Statement v1 query profile without doing
 * network I/O, dereferencing predicate URLs, or authenticating the payload.
 *
 * The caller must supply the manifest and serving-chain context it expected.
 * A successful parse proves only that the JSON is a coherent instance of this
 * bounded profile; it does not prove signature validity or predicate truth.
 */
export function parseUnsignedZeroneInTotoStatement(
  json: string,
  expectations: ZeroneProvenanceExpectations,
): ParsedUnsignedZeroneProvenance {
  if (typeof json !== "string") {
    return fail("INVALID_JSON", "$", "input must be a JSON string");
  }
  if (utf8Length(json) > ZERONE_PROVENANCE_LIMITS.maxJsonBytes) {
    return fail(
      "INPUT_TOO_LARGE",
      "$",
      `JSON must be at most ${ZERONE_PROVENANCE_LIMITS.maxJsonBytes} UTF-8 bytes`,
    );
  }
  const expected = validateExpectations(expectations);

  let parsed: unknown;
  try {
    parsed = JSON.parse(json) as unknown;
  } catch {
    return fail("INVALID_JSON", "$", "input is not valid JSON");
  }

  const topLevel = record(parsed, "$");
  exactKeys(topLevel, TOP_LEVEL_KEYS, "$");
  if (topLevel._type !== IN_TOTO_STATEMENT_V1_TYPE) {
    return fail(
      "UNSUPPORTED_PROFILE",
      "$._type",
      `must equal ${IN_TOTO_STATEMENT_V1_TYPE}`,
    );
  }
  if (
    topLevel.predicateType !==
    ZERONE_TRAINING_PROVENANCE_V1_PREDICATE_TYPE
  ) {
    return fail(
      "UNSUPPORTED_PROFILE",
      "$.predicateType",
      `must equal ${ZERONE_TRAINING_PROVENANCE_V1_PREDICATE_TYPE}`,
    );
  }

  if (!Array.isArray(topLevel.subject) || topLevel.subject.length !== 1) {
    return fail(
      "INVALID_SHAPE",
      "$.subject",
      "must contain exactly one subject",
    );
  }
  const subjectObject = record(topLevel.subject[0], "$.subject[0]");
  exactKeys(subjectObject, SUBJECT_KEYS, "$.subject[0]");
  const subjectName = boundedString(
    subjectObject.name,
    "$.subject[0].name",
    3 *
      (ZERONE_PROVENANCE_LIMITS.maxManifestIdBytes +
        ZERONE_PROVENANCE_LIMITS.maxChainIdBytes) +
      64,
  );
  const digestObject = record(subjectObject.digest, "$.subject[0].digest");
  exactKeys(digestObject, DIGEST_KEYS, "$.subject[0].digest");
  const subjectSha256 = boundedString(
    digestObject.sha256,
    "$.subject[0].digest.sha256",
    64,
  );
  if (!SHA256_PATTERN.test(subjectSha256)) {
    return fail(
      "INVALID_SHAPE",
      "$.subject[0].digest.sha256",
      "must be a lowercase 32-byte SHA-256 hex digest",
    );
  }

  const predicateObject = record(topLevel.predicate, "$.predicate");
  exactKeys(predicateObject, PREDICATE_KEYS, "$.predicate");
  const sourceChainId = boundedString(
    predicateObject.sourceChainId,
    "$.predicate.sourceChainId",
    ZERONE_PROVENANCE_LIMITS.maxChainIdBytes,
  );
  const observedOnChainId = boundedString(
    predicateObject.observedOnChainId,
    "$.predicate.observedOnChainId",
    ZERONE_PROVENANCE_LIMITS.maxChainIdBytes,
  );
  if (observedOnChainId !== expected.observedOnChainId) {
    return fail(
      "EXPECTATION_MISMATCH",
      "$.predicate.observedOnChainId",
      "does not match the expected serving chain",
    );
  }
  if (
    expected.sourceChainId !== undefined &&
    sourceChainId !== expected.sourceChainId
  ) {
    return fail(
      "EXPECTATION_MISMATCH",
      "$.predicate.sourceChainId",
      "does not match the expected source chain",
    );
  }

  const canonicalSubjectName =
    `zerone://${pathEscape(sourceChainId)}/training-corpus/` +
    pathEscape(expected.manifestId);
  if (subjectName !== canonicalSubjectName) {
    return fail(
      "INVALID_SHAPE",
      "$.subject[0].name",
      "does not match the canonical source-chain and manifest subject URI",
    );
  }

  const certificate = parseCertificate(
    predicateObject.certificate,
    sourceChainId,
    expected.manifestId,
    subjectSha256,
  );
  const subject = Object.freeze({
    name: subjectName,
    digest: Object.freeze({ sha256: subjectSha256 }),
  });
  const statement: UnsignedZeroneInTotoStatement = Object.freeze({
    _type: IN_TOTO_STATEMENT_V1_TYPE,
    subject: Object.freeze([subject]) as readonly [ZeroneInTotoSubject],
    predicateType: ZERONE_TRAINING_PROVENANCE_V1_PREDICATE_TYPE,
    predicate: Object.freeze({
      sourceChainId,
      observedOnChainId,
      certificate,
    }),
  });

  return Object.freeze({
    statement,
    assurance: Object.freeze({
      authenticated: false,
      signatureVerified: false,
      currentStateProjection: true,
      subjectDigestScope: "included-id-set" as const,
    }),
  });
}

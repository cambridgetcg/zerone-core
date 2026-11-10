import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import {
  IN_TOTO_STATEMENT_V1_TYPE,
  ProvenanceParseError,
  ZERONE_TRAINING_PROVENANCE_V1_PREDICATE_TYPE,
  ZERONE_TRAINING_PROVENANCE_V2_PREDICATE_TYPE,
  parseUnsignedZeroneInTotoStatement,
  parseUnsignedZeroneInTotoStatementV2,
  type ProvenanceParseErrorCode,
} from "../src/provenance";

const ROOT =
  "7e4a9b03c4d6f8e1023456789abcdef07e4a9b03c4d6f8e1023456789abcdef0";
const EXPECTATIONS = Object.freeze({
  manifestId: "manifest/a",
  observedOnChainId: "zerone-observer-2",
  sourceChainId: "zerone-origin-1",
});
const WARNING_A =
  "no privileged actions touched the manifest's facts; no incidents touched the knowledge module; no cartel resolutions in covered domains";
const EMPTY_REVIEW =
  "fact_count=0: no fact IDs are included directly in this manifest, so there is no review to count";

function reviewSentence(
  reviewed: number,
  total: number,
  unreviewed: number,
  doctrine: number,
  missing: number,
): string {
  return `reviewed_fact_count=${reviewed} of fact_count=${total} have a retained verification round with at least one revealed vote; unreviewed_fact_count=${unreviewed}, including doctrine_fact_count=${doctrine} and missing_fact_count=${missing}`;
}

// A doctrine-only manifest: warning grade A, and no fact reviewed.
function fixture(): Record<string, unknown> {
  return {
    _type: IN_TOTO_STATEMENT_V1_TYPE,
    subject: [
      {
        name: "zerone://zerone-origin-1/training-corpus/manifest%2Fa",
        digest: { sha256: ROOT },
      },
    ],
    predicateType: ZERONE_TRAINING_PROVENANCE_V2_PREDICATE_TYPE,
    predicate: {
      sourceChainId: "zerone-origin-1",
      observedOnChainId: "zerone-observer-2",
      certificate: {
        manifestId: "manifest/a",
        pipelineId: "pipeline-1",
        merkleRoot: ROOT,
        factCount: "3",
        finalizedAtBlock: "42",
        status: "MANIFEST_STATUS_FINALIZED",
        domains: [
          {
            domain: "doctrine_truth_seeking",
            factCount: "3",
          },
        ],
        warningGrade: "A",
        warningExplanation: WARNING_A,
        computedAtBlock: "77",
        sourceChainId: "zerone-origin-1",
        unreviewedFactCount: "3",
        doctrineFactCount: "3",
        reviewState: "unreviewed",
        reviewExplanation: reviewSentence(0, 3, 3, 3, 0),
      },
    },
  };
}

function parse(value: unknown = fixture()) {
  return parseUnsignedZeroneInTotoStatementV2(
    JSON.stringify(value),
    EXPECTATIONS,
  );
}

function assertParseError(
  operation: () => unknown,
  code: ProvenanceParseErrorCode,
  path?: string,
): void {
  assert.throws(
    operation,
    (error: unknown) =>
      error instanceof ProvenanceParseError &&
      error.code === code &&
      (path === undefined || error.path === path),
  );
}

function certificate(value: Record<string, unknown>): Record<string, unknown> {
  return (value.predicate as Record<string, unknown>)
    .certificate as Record<string, unknown>;
}

function withReview(
  value: Record<string, unknown>,
  counts: {
    total: number;
    reviewed: number;
    unreviewed: number;
    doctrine: number;
    missing: number;
  },
  state: string,
): Record<string, unknown> {
  const cert = certificate(value);
  cert.factCount = String(counts.total);
  cert.reviewedFactCount = String(counts.reviewed);
  cert.unreviewedFactCount = String(counts.unreviewed);
  cert.doctrineFactCount = String(counts.doctrine);
  cert.missingFactCount = String(counts.missing);
  cert.domains = [];
  cert.reviewState = state;
  cert.reviewExplanation =
    counts.total === 0
      ? EMPTY_REVIEW
      : reviewSentence(
          counts.reviewed,
          counts.total,
          counts.unreviewed,
          counts.doctrine,
          counts.missing,
        );
  return value;
}

describe("unsigned Zerone in-toto provenance, predicate v2", () => {
  it("accepts the shared Go-produced review-coverage statement without authenticating it", () => {
    // Also checked against the real app query in
    // TestTrainingProvenanceV2ReviewCoverageComposition; not a live response.
    const json = readFileSync(
      new URL("./fixtures/provenance-v2-review-coverage.json", import.meta.url),
      "utf8",
    );
    const parsed = parseUnsignedZeroneInTotoStatementV2(json, {
      manifestId: "review-coverage",
      observedOnChainId: "provenance-observer",
      sourceChainId: "provenance-origin",
    });
    const cert = parsed.statement.predicate.certificate;
    assert.equal(parsed.statement.predicateType,
      ZERONE_TRAINING_PROVENANCE_V2_PREDICATE_TYPE);
    assert.equal(cert.warningGrade, "A");
    assert.equal(cert.reviewState, "partly_reviewed");
    assert.equal(cert.factCount, "6");
    assert.equal(cert.reviewedFactCount, "1");
    assert.equal(cert.unreviewedFactCount, "5");
    assert.equal(cert.doctrineFactCount, "1");
    assert.equal(cert.missingFactCount, "1");
    assert.equal(cert.reviewExplanation,
      JSON.parse(json).predicate.certificate.reviewExplanation);
    assert.deepEqual(parsed.assurance, {
      authenticated: false,
      signatureVerified: false,
      currentStateProjection: true,
      subjectDigestScope: "included-id-set",
    });
    assert.ok(Object.isFrozen(parsed));
    assert.ok(Object.isFrozen(cert));
  });

  it("reads a doctrine-only manifest as warning grade A and unreviewed", () => {
    const cert = parse().statement.predicate.certificate;
    assert.equal(cert.warningGrade, "A");
    assert.equal(cert.reviewState, "unreviewed");
    assert.equal(cert.reviewedFactCount, "0");
    assert.equal(cert.missingFactCount, "0");
    assert.equal(cert.doctrineFactCount, "3");
  });

  it("recomputes every review state and its sentence from the counts", () => {
    for (const [counts, state] of [
      [{ total: 0, reviewed: 0, unreviewed: 0, doctrine: 0, missing: 0 }, "empty"],
      [{ total: 2, reviewed: 2, unreviewed: 0, doctrine: 0, missing: 0 }, "reviewed"],
      [{ total: 2, reviewed: 0, unreviewed: 2, doctrine: 1, missing: 1 }, "unreviewed"],
      [{ total: 6, reviewed: 1, unreviewed: 5, doctrine: 1, missing: 1 }, "partly_reviewed"],
    ] as const) {
      const parsed = parse(withReview(fixture(), counts, state))
        .statement.predicate.certificate;
      assert.equal(parsed.reviewState, state);
      assert.equal(parsed.reviewedFactCount, String(counts.reviewed));
      assert.equal(parsed.unreviewedFactCount, String(counts.unreviewed));
    }
  });

  it("rejects a review state or sentence that the counts do not support", () => {
    for (const state of ["reviewed", "partly_reviewed", "empty"]) {
      const value = fixture();
      certificate(value).reviewState = state;
      assertParseError(() => parse(value), "INVALID_SHAPE",
        "$.predicate.certificate.reviewState");
    }
    for (const state of ["A", "REVIEWED", "Reviewed", "partly-reviewed", ""]) {
      const value = fixture();
      certificate(value).reviewState = state;
      assertParseError(() => parse(value), "INVALID_SHAPE",
        "$.predicate.certificate.reviewState");
    }
    const absent = fixture();
    delete certificate(absent).reviewState;
    assertParseError(() => parse(absent), "INVALID_SHAPE",
      "$.predicate.certificate.reviewState");

    for (const sentence of [
      reviewSentence(0, 3, 3, 3, 0) + " ",
      reviewSentence(0, 3, 3, 2, 0),
      EMPTY_REVIEW,
      "every included fact was reviewed",
    ]) {
      const value = fixture();
      certificate(value).reviewExplanation = sentence;
      assertParseError(() => parse(value), "INVALID_SHAPE",
        "$.predicate.certificate.reviewExplanation");
    }
  });

  it("enforces the review count invariants", () => {
    const sum = fixture();
    certificate(sum).reviewedFactCount = "1";
    assertParseError(() => parse(sum), "INVALID_SHAPE",
      "$.predicate.certificate.unreviewedFactCount");

    const subsets = fixture();
    certificate(subsets).missingFactCount = "1";
    assertParseError(() => parse(subsets), "INVALID_SHAPE",
      "$.predicate.certificate.doctrineFactCount");

    // Missing facts cannot appear in domain coverage.
    const coverage = withReview(fixture(),
      { total: 3, reviewed: 0, unreviewed: 3, doctrine: 2, missing: 1 }, "unreviewed");
    certificate(coverage).domains = [{ domain: "doctrine_tok", factCount: "3" }];
    assertParseError(() => parse(coverage), "INVALID_SHAPE",
      "$.predicate.certificate.domains");
    certificate(coverage).domains = [{ domain: "doctrine_tok", factCount: "2" }];
    assert.equal(parse(coverage).statement.predicate.certificate.domains.length, 1);
  });

  it("rejects noncanonical review counts", () => {
    for (const field of [
      "reviewedFactCount",
      "unreviewedFactCount",
      "doctrineFactCount",
      "missingFactCount",
    ]) {
      for (const invalid of [3, "-1", "03", "18446744073709551616", "1.0"]) {
        const value = fixture();
        certificate(value)[field] = invalid;
        assertParseError(() => parse(value), "INVALID_SHAPE",
          `$.predicate.certificate.${field}`);
      }
    }
  });

  it("keeps the v1 warning rubric and current wording under warningGrade", () => {
    for (const [privileged, incidents, cartel, grade, explanation] of [
      [0, 0, 0, "A", WARNING_A],
      [2, 0, 0, "B", "privileged_action_count=2 affecting manifest facts; manifest is largely unintervened"],
      [3, 0, 0, "C", "privileged_action_count=3, incident_count=0, cartel_resolution_count=0 — yellow flags accumulating; downstream consumer should review the audit trail before relying on this manifest"],
      [0, 1, 0, "C", "privileged_action_count=0, incident_count=1, cartel_resolution_count=0 — yellow flags accumulating; downstream consumer should review the audit trail before relying on this manifest"],
      [3, 4, 1, "F", "cartel_resolutions=1 in covered domains; domain-level warning only, not evidence about this manifest's specific adjudication panel"],
    ] as const) {
      const value = fixture();
      Object.assign(certificate(value), {
        privilegedActionCount: privileged,
        incidentCount: incidents,
        cartelResolutionCount: cartel,
        warningGrade: grade,
        warningExplanation: explanation,
      });
      const parsed = parse(value).statement.predicate.certificate;
      assert.equal(parsed.warningGrade, grade);
      assert.equal(parsed.warningExplanation, explanation);
      // The review reading does not move with the warning grade.
      assert.equal(parsed.reviewState, "unreviewed");
    }

    const mismatched = fixture();
    certificate(mismatched).incidentCount = 1;
    assertParseError(() => parse(mismatched), "INVALID_SHAPE",
      "$.predicate.certificate.warningGrade");

    // No v2 producer ever emitted the legacy v1 F wording.
    const legacy = fixture();
    Object.assign(certificate(legacy), {
      cartelResolutionCount: 1,
      warningGrade: "F",
      warningExplanation:
        "cartel_resolutions=1 in covered domains; this manifest's adjudication panel was demonstrably compromised",
    });
    assertParseError(() => parse(legacy), "INVALID_SHAPE",
      "$.predicate.certificate.warningExplanation");
  });

  it("keeps v1 and v2 statements apart", () => {
    const v1Field = fixture();
    certificate(v1Field).trustGrade = "A";
    assertParseError(() => parse(v1Field), "INVALID_SHAPE",
      "$.predicate.certificate.trustGrade");

    const v1Type = fixture();
    v1Type.predicateType = ZERONE_TRAINING_PROVENANCE_V1_PREDICATE_TYPE;
    assertParseError(() => parse(v1Type), "UNSUPPORTED_PROFILE",
      "$.predicateType");

    const v1Statement = readFileSync(
      new URL("./fixtures/provenance-domain-warning.json", import.meta.url),
      "utf8",
    );
    assertParseError(
      () => parseUnsignedZeroneInTotoStatementV2(v1Statement, {
        manifestId: "domain-warning",
        observedOnChainId: "provenance-observer",
      }),
      "UNSUPPORTED_PROFILE",
      "$.predicateType",
    );
    assertParseError(
      () => parseUnsignedZeroneInTotoStatement(JSON.stringify(fixture()), EXPECTATIONS),
      "UNSUPPORTED_PROFILE",
      "$.predicateType",
    );
  });

  it("binds manifest, serving chain, source chain, subject and status as v1 does", () => {
    const manifest = fixture();
    certificate(manifest).manifestId = "other";
    assertParseError(() => parse(manifest), "EXPECTATION_MISMATCH",
      "$.predicate.certificate.manifestId");

    const observed = fixture();
    (observed.predicate as Record<string, unknown>).observedOnChainId = "other-chain";
    assertParseError(() => parse(observed), "EXPECTATION_MISMATCH",
      "$.predicate.observedOnChainId");

    const subject = fixture();
    (subject.subject as Record<string, unknown>[])[0]!.name =
      "zerone://zerone-origin-1/training-corpus/other";
    assertParseError(() => parse(subject), "INVALID_SHAPE", "$.subject[0].name");

    const root = fixture();
    certificate(root).merkleRoot = "0".repeat(64);
    assertParseError(() => parse(root), "INVALID_SHAPE",
      "$.predicate.certificate.merkleRoot");

    const draft = fixture();
    certificate(draft).status = "MANIFEST_STATUS_DRAFT";
    assertParseError(() => parse(draft), "UNSUPPORTED_PROFILE",
      "$.predicate.certificate.status");

    const extra = fixture();
    extra.signature = "not part of Statement v1";
    assertParseError(() => parse(extra), "INVALID_SHAPE", "$.signature");

    assertParseError(
      () => parseUnsignedZeroneInTotoStatementV2("{", EXPECTATIONS),
      "INVALID_JSON",
      "$",
    );
  });

  it("normalizes omitted protobuf JSON zero counts", () => {
    const value = withReview(fixture(),
      { total: 0, reviewed: 0, unreviewed: 0, doctrine: 0, missing: 0 }, "empty");
    const cert = certificate(value);
    for (const field of [
      "factCount",
      "reviewedFactCount",
      "unreviewedFactCount",
      "doctrineFactCount",
      "missingFactCount",
      "domains",
    ]) {
      delete cert[field];
    }
    const parsed = parse(value).statement.predicate.certificate;
    assert.equal(parsed.factCount, "0");
    assert.equal(parsed.reviewedFactCount, "0");
    assert.equal(parsed.missingFactCount, "0");
    assert.equal(parsed.reviewState, "empty");
    assert.equal(parsed.reviewExplanation, EMPTY_REVIEW);
  });

  it("pins predicate v2 to the commit that added its specification", () => {
    assert.equal(
      ZERONE_TRAINING_PROVENANCE_V2_PREDICATE_TYPE,
      "https://github.com/cambridgetcg/zerone-core/blob/fbfd5ef046c4b0c5eb263ff9a8d14d62796c2219/docs/specs/attestations/training-provenance-v2.md",
    );
    assert.notEqual(
      ZERONE_TRAINING_PROVENANCE_V2_PREDICATE_TYPE,
      ZERONE_TRAINING_PROVENANCE_V1_PREDICATE_TYPE,
    );
  });
});

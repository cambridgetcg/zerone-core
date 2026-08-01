import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  IN_TOTO_STATEMENT_V1_TYPE,
  ProvenanceParseError,
  ZERONE_PROVENANCE_LIMITS,
  ZERONE_TRAINING_PROVENANCE_V1_PREDICATE_TYPE,
  parseUnsignedZeroneInTotoStatement,
  type ProvenanceParseErrorCode,
} from "../src/provenance";

const ROOT =
  "7e4a9b03c4d6f8e1023456789abcdef07e4a9b03c4d6f8e1023456789abcdef0";
const EXPECTATIONS = Object.freeze({
  manifestId: "manifest/a",
  observedOnChainId: "zerone-observer-2",
  sourceChainId: "zerone-origin-1",
});

function fixture(): Record<string, unknown> {
  return {
    _type: IN_TOTO_STATEMENT_V1_TYPE,
    subject: [
      {
        name: "zerone://zerone-origin-1/training-corpus/manifest%2Fa",
        digest: { sha256: ROOT },
      },
    ],
    predicateType: ZERONE_TRAINING_PROVENANCE_V1_PREDICATE_TYPE,
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
            domain: "language",
            factCount: "2",
            avgQualifiedWeight: "9",
            activeVoterCount: 2,
          },
        ],
        trustGrade: "A",
        trustExplanation:
          "no privileged actions touched the manifest's facts; no incidents touched the knowledge module; no cartel resolutions in covered domains",
        computedAtBlock: "77",
        sourceChainId: "zerone-origin-1",
      },
    },
  };
}

function parse(value: unknown = fixture()) {
  return parseUnsignedZeroneInTotoStatement(
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

describe("unsigned Zerone in-toto provenance", () => {
  it("pins the predicate profile to an immutable specification revision", () => {
    assert.equal(
      ZERONE_TRAINING_PROVENANCE_V1_PREDICATE_TYPE,
      "https://github.com/cambridgetcg/zerone-core/blob/394bbef01df1b131223b1e874d554932d8dcd87c/docs/specs/attestations/training-provenance-v1.md",
    );
  });

  it("parses the exact v1 profile and preserves uint64 values as strings", () => {
    const parsed = parse();

    assert.equal(parsed.statement._type, IN_TOTO_STATEMENT_V1_TYPE);
    assert.equal(parsed.statement.subject[0].digest.sha256, ROOT);
    assert.equal(parsed.statement.predicate.certificate.factCount, "3");
    assert.equal(
      typeof parsed.statement.predicate.certificate.factCount,
      "string",
    );
    assert.deepEqual(parsed.statement.predicate.certificate.domains[0], {
      domain: "language",
      factCount: "2",
      avgQualifiedWeight: "9",
      activeVoterCount: 2,
    });
    assert.deepEqual(parsed.assurance, {
      authenticated: false,
      signatureVerified: false,
      currentStateProjection: true,
      subjectDigestScope: "included-id-set",
    });
    assert.ok(Object.isFrozen(parsed));
    assert.ok(Object.isFrozen(parsed.statement.predicate.certificate.domains));
  });

  it("normalizes omitted protobuf JSON defaults without using JS numbers for uint64", () => {
    const value = fixture();
    const cert = certificate(value);
    delete cert.factCount;
    delete cert.finalizedAtBlock;
    delete cert.domains;
    delete cert.privilegedActionCount;
    delete cert.incidentCount;
    delete cert.cartelResolutionCount;
    delete cert.computedAtBlock;

    const parsed = parse(value).statement.predicate.certificate;
    assert.equal(parsed.factCount, "0");
    assert.equal(parsed.finalizedAtBlock, "0");
    assert.equal(parsed.computedAtBlock, "0");
    assert.equal(parsed.privilegedActionCount, 0);
    assert.deepEqual(parsed.domains, []);
  });

  it("accepts the complete uint64 boundary and all sealed statuses", () => {
    for (const status of [
      "MANIFEST_STATUS_FINALIZED",
      "MANIFEST_STATUS_ATTESTED",
      "MANIFEST_STATUS_SUPERSEDED",
    ]) {
      const value = fixture();
      const cert = certificate(value);
      cert.status = status;
      (cert.domains as Record<string, unknown>[])[0]!.avgQualifiedWeight =
        "18446744073709551615";
      assert.equal(
        parse(value).statement.predicate.certificate.domains[0]
          ?.avgQualifiedWeight,
        "18446744073709551615",
      );
    }
  });

  it("uses Go PathEscape-compatible canonical subject segments", () => {
    const value = fixture();
    const predicate = value.predicate as Record<string, unknown>;
    const cert = certificate(value);
    predicate.sourceChainId = "origin:a+b@example";
    cert.sourceChainId = "origin:a+b@example";
    cert.manifestId = "manifest!*é";
    (value.subject as Record<string, unknown>[])[0]!.name =
      "zerone://origin:a+b@example/training-corpus/manifest%21%2A%C3%A9";

    const parsed = parseUnsignedZeroneInTotoStatement(JSON.stringify(value), {
      manifestId: "manifest!*é",
      observedOnChainId: "zerone-observer-2",
      sourceChainId: "origin:a+b@example",
    });
    assert.equal(
      parsed.statement.subject[0].name,
      "zerone://origin:a+b@example/training-corpus/manifest%21%2A%C3%A9",
    );
  });

  it("rejects unsupported statement, predicate, and manifest profiles", () => {
    for (const [mutate, path] of [
      [
        (value: Record<string, unknown>) => {
          value._type = "https://in-toto.io/Statement/v0.1";
        },
        "$._type",
      ],
      [
        (value: Record<string, unknown>) => {
          value.predicateType = "https://example.com/other";
        },
        "$.predicateType",
      ],
      [
        (value: Record<string, unknown>) => {
          certificate(value).status = "MANIFEST_STATUS_DRAFT";
        },
        "$.predicate.certificate.status",
      ],
    ] as const) {
      const value = fixture();
      mutate(value);
      assertParseError(() => parse(value), "UNSUPPORTED_PROFILE", path);
    }
  });

  it("rejects unknown fields and ambiguous subject or digest shapes", () => {
    const extra = fixture();
    extra.signature = "not part of Statement v1";
    assertParseError(() => parse(extra), "INVALID_SHAPE", "$.signature");

    const twoSubjects = fixture();
    (twoSubjects.subject as unknown[]).push({});
    assertParseError(() => parse(twoSubjects), "INVALID_SHAPE", "$.subject");

    const extraDigest = fixture();
    (
      (extraDigest.subject as Record<string, unknown>[])[0]!
        .digest as Record<string, unknown>
    ).sha512 = "00";
    assertParseError(
      () => parse(extraDigest),
      "INVALID_SHAPE",
      "$.subject[0].digest.sha512",
    );
  });

  it("rejects malformed or inconsistent SHA-256 commitments", () => {
    const uppercase = fixture();
    (
      (uppercase.subject as Record<string, unknown>[])[0]!
        .digest as Record<string, unknown>
    ).sha256 = ROOT.toUpperCase();
    assertParseError(
      () => parse(uppercase),
      "INVALID_SHAPE",
      "$.subject[0].digest.sha256",
    );

    const mismatch = fixture();
    certificate(mismatch).merkleRoot = "0".repeat(64);
    assertParseError(
      () => parse(mismatch),
      "INVALID_SHAPE",
      "$.predicate.certificate.merkleRoot",
    );
  });

  it("binds manifest, serving chain, optional source chain, and canonical subject URI", () => {
    const manifest = fixture();
    certificate(manifest).manifestId = "other";
    assertParseError(
      () => parse(manifest),
      "EXPECTATION_MISMATCH",
      "$.predicate.certificate.manifestId",
    );

    const observed = fixture();
    (observed.predicate as Record<string, unknown>).observedOnChainId =
      "other-chain";
    assertParseError(
      () => parse(observed),
      "EXPECTATION_MISMATCH",
      "$.predicate.observedOnChainId",
    );

    const source = fixture();
    (source.predicate as Record<string, unknown>).sourceChainId = "other-chain";
    assertParseError(
      () => parse(source),
      "EXPECTATION_MISMATCH",
      "$.predicate.sourceChainId",
    );

    const subject = fixture();
    (subject.subject as Record<string, unknown>[])[0]!.name =
      "zerone://zerone-origin-1/training-corpus/other";
    assertParseError(
      () => parse(subject),
      "INVALID_SHAPE",
      "$.subject[0].name",
    );
  });

  it("rejects noncanonical and overflowing protobuf JSON integers", () => {
    for (const invalid of [
      3,
      "-1",
      "01",
      "18446744073709551616",
      "1.0",
    ]) {
      const value = fixture();
      certificate(value).factCount = invalid;
      assertParseError(
        () => parse(value),
        "INVALID_SHAPE",
        "$.predicate.certificate.factCount",
      );
    }

    for (const invalid of [-1, 1.5, 4_294_967_296, "1"]) {
      const value = fixture();
      certificate(value).incidentCount = invalid;
      assertParseError(
        () => parse(value),
        "INVALID_SHAPE",
        "$.predicate.certificate.incidentCount",
      );
    }
  });

  it("checks v1 grade and domain-count coherence", () => {
    const grade = fixture();
    certificate(grade).incidentCount = 1;
    assertParseError(
      () => parse(grade),
      "INVALID_SHAPE",
      "$.predicate.certificate.trustGrade",
    );

    const explanation = fixture();
    certificate(explanation).trustExplanation =
      "signature verified; facts independently proven";
    assertParseError(
      () => parse(explanation),
      "INVALID_SHAPE",
      "$.predicate.certificate.trustExplanation",
    );

    const coverage = fixture();
    (
      certificate(coverage).domains as Record<string, unknown>[]
    )[0]!.factCount = "4";
    assertParseError(
      () => parse(coverage),
      "INVALID_SHAPE",
      "$.predicate.certificate.domains",
    );

    const duplicate = fixture();
    const domains = certificate(duplicate).domains as Record<string, unknown>[];
    domains.push({ ...domains[0] });
    assertParseError(
      () => parse(duplicate),
      "INVALID_SHAPE",
      "$.predicate.certificate.domains[1].domain",
    );
  });

  it("enforces byte and collection bounds and rejects ill-formed Unicode", () => {
    const oversized = " ".repeat(ZERONE_PROVENANCE_LIMITS.maxJsonBytes + 1);
    assertParseError(
      () =>
        parseUnsignedZeroneInTotoStatement(oversized, EXPECTATIONS),
      "INPUT_TOO_LARGE",
      "$",
    );

    const longManifest = fixture();
    certificate(longManifest).manifestId = "m".repeat(
      ZERONE_PROVENANCE_LIMITS.maxManifestIdBytes + 1,
    );
    assertParseError(
      () => parse(longManifest),
      "INPUT_TOO_LARGE",
      "$.predicate.certificate.manifestId",
    );

    const malformedUnicode = fixture();
    certificate(malformedUnicode).pipelineId = "\ud800";
    assertParseError(
      () => parse(malformedUnicode),
      "INVALID_SHAPE",
      "$.predicate.certificate.pipelineId",
    );
  });

  it("rejects invalid JSON and invalid caller expectations", () => {
    assertParseError(
      () =>
        parseUnsignedZeroneInTotoStatement("{", EXPECTATIONS),
      "INVALID_JSON",
      "$",
    );
    assertParseError(
      () =>
        parseUnsignedZeroneInTotoStatement(JSON.stringify(fixture()), {
          manifestId: "",
          observedOnChainId: "zerone-observer-2",
        }),
      "INVALID_SHAPE",
      "$expectations.manifestId",
    );
  });
});

import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import {
  BASE_TREE_SHA256,
  LIFE_SCIENCES_EVIDENCE_FIXTURE_SCHEMA,
  LIFE_SCIENCES_MAX_BYTES,
  LIFE_SCIENCES_SCHEMA,
  MONEY_KARMA_CONSTITUTION_SCHEMA,
  MONEY_KARMA_CONSTITUTION_SHA256,
  LifeSciencesValidationError,
  evaluateLifeSciencesEvidenceFixture,
  parseAndValidateConstructiveIntelligenceLifeSciences,
  validateConstructiveIntelligenceLifeSciences,
  validateMoneyKarmaConstitutionBinding,
} from "./validate-constructive-intelligence-life-sciences.mjs";

const canonicalRaw = readFileSync(
  new URL(
    "../public/standards/constructive-intelligence-life-sciences.v0.json",
    import.meta.url,
  ),
  "utf8",
);
const canonical = JSON.parse(canonicalRaw);
const constitutionRaw = readFileSync(
  new URL("../../docs/constitution/money-karma-v1.json", import.meta.url),
);

function copyProfile() {
  return structuredClone(canonical);
}

function node(profile, id) {
  const found = profile.nodes.find((candidate) => candidate.id === id);
  assert.ok(found, `missing profile node ${id}`);
  return found;
}

function assertInvalid(operation, path) {
  assert.throws(
    operation,
    (error) =>
      error instanceof LifeSciencesValidationError &&
      (path === undefined || error.path === path),
  );
}

function controls({ sybil = false } = {}) {
  return [
    {
      contributorId: "synthetic-contributor-a",
      effectiveControlCluster: "control-cluster-a",
      organizationRoot: "organization-root-a",
      methodRoot: "method-root-a",
      contextRoot: "context-root-a",
    },
    {
      contributorId: "synthetic-contributor-b",
      effectiveControlCluster: sybil ? "control-cluster-a" : "control-cluster-b",
      organizationRoot: "organization-root-b",
      methodRoot: "method-root-b",
      contextRoot: "context-root-b",
    },
    {
      contributorId: "synthetic-contributor-c",
      effectiveControlCluster: sybil ? "control-cluster-a" : "control-cluster-c",
      organizationRoot: "organization-root-b",
      methodRoot: "method-root-b",
      contextRoot: "context-root-b",
    },
  ];
}

function artifact(
  observationKind,
  {
    id = "synthetic-artifact",
    containsSequence = false,
    containsOperationalProtocol = false,
    humanDataClass = "NONE",
    riskTopics = [],
  } = {},
) {
  return {
    id,
    uri: `https://${id}.invalid/metadata.json`,
    sha256: "1".repeat(64),
    observationKind,
    containsSequence,
    containsOperationalProtocol,
    humanDataClass,
    riskTopics,
  };
}

function fixture({
  id = "synthetic-boundary-case",
  nodeId,
  requestedConclusions,
  evidenceLevel = "LS2",
  artifacts,
  disclosedControls = [],
  challengeStatus = "CLEAR",
}) {
  return {
    schema: LIFE_SCIENCES_EVIDENCE_FIXTURE_SCHEMA,
    id,
    profileSchema: LIFE_SCIENCES_SCHEMA,
    nodeId,
    requestedConclusions: [...requestedConclusions].sort(),
    evidenceLevel,
    artifacts,
    controls: disclosedControls,
    challengeStatus,
  };
}

function crownFixture(overrides = {}) {
  return fixture({
    id: "synthetic-cross-context-case",
    nodeId: "replication-cross-context-crown@1",
    requestedConclusions: ["INDEPENDENT_CROSS_CONTEXT_REPLICATION"],
    evidenceLevel: "LS5",
    artifacts: [
      artifact("CROSS_CONTEXT_REPLICATION", {
        id: "synthetic-cross-context-metadata",
      }),
    ],
    disclosedControls: controls(),
    challengeStatus: "CLEAR",
    ...overrides,
  });
}

describe("constructive-intelligence life-sciences profile", () => {
  it("validates the bound 17-node DRAFT/SHADOW_ONLY profile", () => {
    assert.deepEqual(parseAndValidateConstructiveIntelligenceLifeSciences(canonicalRaw), {
      schema: LIFE_SCIENCES_SCHEMA,
      status: "DRAFT",
      mode: "SHADOW_ONLY",
      nodeCount: 17,
      edgeCount: 24,
      maxDepth: 8,
      crownId: "replication-cross-context-crown@1",
      economicEffect: "NONE",
      amount: "0",
    });
    assert.equal(canonical.baseTreeBinding.sha256, BASE_TREE_SHA256);
    assert.deepEqual(canonical.constitutionBinding, {
      schema: MONEY_KARMA_CONSTITUTION_SCHEMA,
      documentSha256: `sha256:${MONEY_KARMA_CONSTITUTION_SHA256}`,
    });
    assert.equal(
      createHash("sha256").update(constitutionRaw).digest("hex"),
      MONEY_KARMA_CONSTITUTION_SHA256,
    );
    assert.deepEqual(canonical.attestationBoundary, {
      controlDisclosures: "SELF_DECLARED_SYNTHETIC_LABELS",
      challengeStatus: "SELF_DECLARED_SYNTHETIC_LABEL",
      establishesControllerIndependence: false,
      establishesChallengeClosure: false,
    });
    assert.equal(
      canonical.independence.futureEligibilityRequiresExternalControllerAttestation,
      true,
    );
    assert.equal(
      canonical.challengePolicy.futureEligibilityRequiresAdjudicationReceipt,
      true,
    );
  });

  it("rejects a substituted constitution pin and checked-in constitution drift", () => {
    const substituted = copyProfile();
    substituted.constitutionBinding.documentSha256 = `sha256:${"0".repeat(64)}`;
    assertInvalid(
      () => validateConstructiveIntelligenceLifeSciences(substituted),
      "$.constitutionBinding.documentSha256",
    );
    assertInvalid(
      () =>
        validateMoneyKarmaConstitutionBinding(
          canonical.constitutionBinding,
          Buffer.concat([constitutionRaw, Buffer.from("\n")]),
        ),
      "$.constitutionBinding.documentSha256",
    );
  });

  it("rejects any claim that synthetic declarations establish independence or closure", () => {
    for (const [path, mutate] of [
      [
        "$.attestationBoundary.controlDisclosures",
        (profile) => (profile.attestationBoundary.controlDisclosures = "VERIFIED"),
      ],
      [
        "$.independence.futureEligibilityRequiresExternalControllerAttestation",
        (profile) =>
          (profile.independence.futureEligibilityRequiresExternalControllerAttestation =
            false),
      ],
      [
        "$.attestationBoundary.challengeStatus",
        (profile) => (profile.attestationBoundary.challengeStatus = "VERIFIED"),
      ],
      [
        "$.challengePolicy.futureEligibilityRequiresAdjudicationReceipt",
        (profile) =>
          (profile.challengePolicy.futureEligibilityRequiresAdjudicationReceipt = false),
      ],
      [
        "$.attestationBoundary.establishesControllerIndependence",
        (profile) =>
          (profile.attestationBoundary.establishesControllerIndependence = true),
      ],
      [
        "$.attestationBoundary.establishesChallengeClosure",
        (profile) => (profile.attestationBoundary.establishesChallengeClosure = true),
      ],
    ]) {
      const profile = copyProfile();
      mutate(profile);
      assertInvalid(
        () => validateConstructiveIntelligenceLifeSciences(profile),
        path,
      );
    }
  });

  it("rejects malformed, oversized, and non-exact documents", () => {
    assertInvalid(() => parseAndValidateConstructiveIntelligenceLifeSciences("{"), "$",
    );
    assertInvalid(
      () =>
        parseAndValidateConstructiveIntelligenceLifeSciences(
          " ".repeat(LIFE_SCIENCES_MAX_BYTES + 1),
        ),
      "$",
    );
    const duplicateSchema = canonicalRaw.replace(
      '"schema": "zerone.constructive-intelligence-life-sciences/v0",',
      '"schema": "zerone.constructive-intelligence-life-sciences/v0",\n  "schema": "zerone.constructive-intelligence-life-sciences/v0",',
    );
    assertInvalid(
      () => parseAndValidateConstructiveIntelligenceLifeSciences(duplicateSchema),
      "$.schema",
    );

    const unknown = copyProfile();
    unknown.runtimeEndpoint = "https://runtime.invalid";
    assertInvalid(
      () => validateConstructiveIntelligenceLifeSciences(unknown),
      "$.runtimeEndpoint",
    );

    const missingBinding = copyProfile();
    delete missingBinding.baseTreeBinding;
    assertInvalid(
      () => validateConstructiveIntelligenceLifeSciences(missingBinding),
      "$.baseTreeBinding",
    );
  });

  it("rejects every path toward economics, authority, governance, or release", () => {
    for (const [path, mutate] of [
      ["$.rewardBearing", (profile) => (profile.rewardBearing = true)],
      ["$.economics.effect", (profile) => (profile.economics.effect = "REWARD")],
      ["$.economics.amount", (profile) => (profile.economics.amount = "1")],
      ["$.economics.denom", (profile) => (profile.economics.denom = "uzrn")],
      [
        "$.economics.rewardMultiplier",
        (profile) => (profile.economics.rewardMultiplier = true),
      ],
      [
        "$.releaseBoundary.addsConsensusBehavior",
        (profile) => (profile.releaseBoundary.addsConsensusBehavior = true),
      ],
      [
        "$.releaseBoundary.grantsGovernanceAuthority",
        (profile) => (profile.releaseBoundary.grantsGovernanceAuthority = true),
      ],
      [
        "$.releaseBoundary.createsKarmaMagnitude",
        (profile) => (profile.releaseBoundary.createsKarmaMagnitude = true),
      ],
      [
        "$.releaseBoundary.marksReleaseReady",
        (profile) => (profile.releaseBoundary.marksReleaseReady = true),
      ],
    ]) {
      const profile = copyProfile();
      mutate(profile);
      assertInvalid(
        () => validateConstructiveIntelligenceLifeSciences(profile),
        path,
      );
    }
  });

  it("rejects base-tree digest drift and substituted authority locators", () => {
    const digestDrift = copyProfile();
    digestDrift.baseTreeBinding.sha256 = "0".repeat(64);
    assertInvalid(
      () => validateConstructiveIntelligenceLifeSciences(digestDrift),
      "$.baseTreeBinding.sha256",
    );

    const referenceDrift = copyProfile();
    referenceDrift.references[0].specification =
      "https://predictioncenter.org/?unfrozen=true";
    assertInvalid(
      () => validateConstructiveIntelligenceLifeSciences(referenceDrift),
      "$.references[0].specification",
    );
  });

  it("rejects weakened GREEN-only, refusal, and fixture boundaries", () => {
    for (const [path, mutate] of [
      ["$.scope.riskClass", (profile) => (profile.scope.riskClass = "UNKNOWN")],
      ["$.scope.refusedTopics", (profile) => profile.scope.refusedTopics.pop()],
      [
        "$.fixtureBoundary.sequencePayloadsAllowed",
        (profile) => (profile.fixtureBoundary.sequencePayloadsAllowed = true),
      ],
      [
        "$.fixtureBoundary.operationalProtocolsAllowed",
        (profile) => (profile.fixtureBoundary.operationalProtocolsAllowed = true),
      ],
      [
        "$.fixtureBoundary.rawHumanGenomeAllowed",
        (profile) => (profile.fixtureBoundary.rawHumanGenomeAllowed = true),
      ],
    ]) {
      const profile = copyProfile();
      mutate(profile);
      assertInvalid(
        () => validateConstructiveIntelligenceLifeSciences(profile),
        path,
      );
    }
  });

  it("rejects dangling edges, cycles, graph overgrowth, and reordered nodes", () => {
    const dangling = copyProfile();
    node(dangling, "folding-calibrated-model@1").prerequisites = [
      "missing-structure-evidence@1",
    ];
    assertInvalid(() => validateConstructiveIntelligenceLifeSciences(dangling), "$.nodes");

    const cycle = copyProfile();
    node(cycle, "expression-context-normalization@1").prerequisites = [
      "expression-assay-design@1",
      "expression-transcript-quantification@1",
    ];
    assertInvalid(() => validateConstructiveIntelligenceLifeSciences(cycle), "$.nodes");

    const overgrown = copyProfile();
    while (overgrown.nodes.length <= 20) {
      overgrown.nodes.push(structuredClone(overgrown.nodes[0]));
    }
    assertInvalid(() => validateConstructiveIntelligenceLifeSciences(overgrown), "$.nodes");

    const reordered = copyProfile();
    [reordered.nodes[0], reordered.nodes[1]] = [reordered.nodes[1], reordered.nodes[0]];
    assertInvalid(() => validateConstructiveIntelligenceLifeSciences(reordered), "$.nodes");
  });

  it("locks the static-structure, transcript, association, and in-silico walls", () => {
    const mutations = [
      [
        "structure-coordinate-evidence@1",
        "FOLDING_PATHWAY",
        "$.nodes[15].permittedConclusions",
      ],
      [
        "expression-transcript-quantification@1",
        "PROTEIN_ABUNDANCE",
        "$.nodes[7].permittedConclusions",
      ],
      [
        "expression-regulatory-association@1",
        "CAUSAL_REGULATION",
        "$.nodes[5].permittedConclusions",
      ],
      [
        "folding-calibrated-model@1",
        "PROSPECTIVE_VALIDATION",
        "$.nodes[8].permittedConclusions",
      ],
    ];
    for (const [nodeId, conclusion, path] of mutations) {
      const profile = copyProfile();
      const target = node(profile, nodeId);
      target.permittedConclusions.push(conclusion);
      target.permittedConclusions.sort();
      assertInvalid(
        () => validateConstructiveIntelligenceLifeSciences(profile),
        path,
      );
    }
  });
});
describe("life-sciences synthetic evidence boundary", () => {
  it("reports a locally matching crown only as unverified, non-eligible shadow structure", () => {
    assert.deepEqual(
      evaluateLifeSciencesEvidenceFixture(copyProfile(), crownFixture()),
      {
        outcome: "SHADOW_ONLY_STRUCTURAL_MATCH",
        rewardEligible: false,
        independenceStatus: "DECLARED_UNVERIFIED",
        challengeStatus: "DECLARED_UNVERIFIED",
        economicEffect: "NONE",
        amount: "0",
        reasons: [],
      },
    );
  });

  it("blocks structure-only evidence from claiming a folding pathway", () => {
    const result = evaluateLifeSciencesEvidenceFixture(
      copyProfile(),
      fixture({
        id: "synthetic-partial-structure",
        nodeId: "structure-coordinate-evidence@1",
        requestedConclusions: ["FOLDING_PATHWAY"],
        artifacts: [artifact("STATIC_STRUCTURE")],
      }),
    );
    assert.equal(result.outcome, "SHADOW_ONLY_BLOCKED");
    assert.ok(
      result.reasons.includes(
        "WALL_STATIC_COORDINATES_NOT_FOLDING_PATHWAY_OR_KINETICS",
      ),
    );
    assert.equal(result.amount, "0");
  });

  it("blocks transcript-only evidence from protein abundance or function", () => {
    const result = evaluateLifeSciencesEvidenceFixture(
      copyProfile(),
      fixture({
        id: "synthetic-partial-transcript",
        nodeId: "expression-transcript-quantification@1",
        requestedConclusions: ["PROTEIN_ABUNDANCE", "PROTEIN_FUNCTION"],
        artifacts: [artifact("TRANSCRIPT_ABUNDANCE")],
      }),
    );
    assert.equal(result.outcome, "SHADOW_ONLY_BLOCKED");
    assert.ok(
      result.reasons.includes(
        "WALL_MRNA_NOT_PROTEIN_ABUNDANCE_LOCALIZATION_OR_FUNCTION",
      ),
    );
  });

  it("blocks association-only evidence from a causal-regulation claim", () => {
    const result = evaluateLifeSciencesEvidenceFixture(
      copyProfile(),
      fixture({
        id: "synthetic-association-only",
        nodeId: "expression-regulatory-association@1",
        requestedConclusions: ["CAUSAL_REGULATION"],
        evidenceLevel: "LS3",
        artifacts: [artifact("REGULATORY_ASSOCIATION")],
        disclosedControls: controls(),
      }),
    );
    assert.equal(result.outcome, "SHADOW_ONLY_BLOCKED");
    assert.ok(
      result.reasons.includes("WALL_ASSOCIATION_NOT_REGULATION_OR_CAUSALITY"),
    );
  });

  it("blocks an in-silico model from claiming prospective validation", () => {
    const result = evaluateLifeSciencesEvidenceFixture(
      copyProfile(),
      fixture({
        id: "synthetic-insilico-only",
        nodeId: "folding-calibrated-model@1",
        requestedConclusions: ["PROSPECTIVE_VALIDATION"],
        artifacts: [artifact("FOLD_MODEL")],
      }),
    );
    assert.equal(result.outcome, "SHADOW_ONLY_BLOCKED");
    assert.ok(result.reasons.includes("WALL_INSILICO_NOT_PROSPECTIVE_EXPERIMENT"));
  });

  it("lets every open, unresolved, or upheld challenge block the crown", () => {
    for (const challengeStatus of ["OPEN", "UNRESOLVED", "UPHELD"]) {
      const result = evaluateLifeSciencesEvidenceFixture(
        copyProfile(),
        crownFixture({ challengeStatus }),
      );
      assert.equal(result.outcome, "SHADOW_ONLY_BLOCKED");
      assert.ok(
        result.reasons.includes(`CHALLENGE_BLOCKS_CROWN:${challengeStatus}`),
      );
      assert.equal(result.economicEffect, "NONE");
    }
  });

  it("collapses aliases into effective control clusters instead of counting identities", () => {
    const result = evaluateLifeSciencesEvidenceFixture(
      copyProfile(),
      crownFixture({ disclosedControls: controls({ sybil: true }) }),
    );
    assert.equal(result.outcome, "SHADOW_ONLY_BLOCKED");
    assert.ok(result.reasons.includes("INSUFFICIENT_EFFECTIVE_CONTROL_CLUSTERS"));
  });

  it("blocks one contributor from claiming divergent control tuples", () => {
    const disclosedControls = controls();
    disclosedControls.push({
      ...disclosedControls[0],
      organizationRoot: "organization-root-divergent",
    });
    const result = evaluateLifeSciencesEvidenceFixture(
      copyProfile(),
      crownFixture({ disclosedControls }),
    );
    assert.equal(result.outcome, "SHADOW_ONLY_BLOCKED");
    assert.equal(result.rewardEligible, false);
    assert.equal(result.independenceStatus, "DECLARED_UNVERIFIED");
    assert.equal(result.challengeStatus, "DECLARED_UNVERIFIED");
    assert.ok(
      result.reasons.includes(
        "CONTRIBUTOR_CLAIMS_DIVERGENT_CONTROL_TUPLES:synthetic-contributor-a",
      ),
    );
  });

  it("keeps every refused or blocked output explicitly non-eligible and unverified", () => {
    const outputs = [
      evaluateLifeSciencesEvidenceFixture(
        copyProfile(),
        crownFixture({ challengeStatus: "OPEN" }),
      ),
      evaluateLifeSciencesEvidenceFixture(
        copyProfile(),
        fixture({
          id: "synthetic-refused-output",
          nodeId: "structure-coordinate-evidence@1",
          requestedConclusions: ["STATIC_STRUCTURE"],
          artifacts: [artifact("STATIC_STRUCTURE", { containsSequence: true })],
        }),
      ),
    ];
    for (const result of outputs) {
      assert.equal(result.rewardEligible, false);
      assert.equal(result.independenceStatus, "DECLARED_UNVERIFIED");
      assert.equal(result.challengeStatus, "DECLARED_UNVERIFIED");
      assert.equal(result.economicEffect, "NONE");
      assert.equal(result.amount, "0");
    }
  });

  it("refuses leakage using metadata flags without embedding a sequence or protocol", () => {
    const result = evaluateLifeSciencesEvidenceFixture(
      copyProfile(),
      fixture({
        id: "synthetic-leakage-sentinel",
        nodeId: "structure-coordinate-evidence@1",
        requestedConclusions: ["STATIC_STRUCTURE"],
        artifacts: [
          artifact("STATIC_STRUCTURE", {
            containsSequence: true,
            containsOperationalProtocol: true,
          }),
        ],
      }),
    );
    assert.equal(result.outcome, "SHADOW_ONLY_REFUSED");
    assert.ok(result.reasons.includes("SEQUENCE_PAYLOAD_REFUSED"));
    assert.ok(result.reasons.includes("OPERATIONAL_PROTOCOL_REFUSED"));
    assert.equal(result.amount, "0");
  });

  it("refuses every RED topic, raw human genome flag, and unknown risk", () => {
    const red = evaluateLifeSciencesEvidenceFixture(
      copyProfile(),
      fixture({
        id: "synthetic-red-risk-sentinel",
        nodeId: "expression-assay-design@1",
        requestedConclusions: ["ASSAY_SCOPE"],
        evidenceLevel: "LS1",
        artifacts: [
          artifact("ASSAY_METADATA", {
            humanDataClass: "RAW_HUMAN_GENOME",
            riskTopics: ["PATHOGENS"],
          }),
        ],
      }),
    );
    assert.equal(red.outcome, "SHADOW_ONLY_REFUSED");
    assert.ok(red.reasons.includes("RAW_HUMAN_GENOME_REFUSED"));
    assert.ok(red.reasons.includes("RED_RISK_REFUSED:PATHOGENS"));

    const unknown = evaluateLifeSciencesEvidenceFixture(
      copyProfile(),
      fixture({
        id: "synthetic-unknown-risk-sentinel",
        nodeId: "expression-assay-design@1",
        requestedConclusions: ["ASSAY_SCOPE"],
        evidenceLevel: "LS1",
        artifacts: [
          artifact("ASSAY_METADATA", {
            riskTopics: ["UNCLASSIFIED_RISK"],
          }),
        ],
      }),
    );
    assert.equal(unknown.outcome, "SHADOW_ONLY_REFUSED");
    assert.ok(
      unknown.reasons.includes(
        "UNKNOWN_RISK_PRIVATE_ESCALATION_ONLY:UNCLASSIFIED_RISK",
      ),
    );
  });
});

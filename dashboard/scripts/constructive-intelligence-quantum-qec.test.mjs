import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";

import {
  QUANTUM_EXTENSION_MAX_BYTES,
  QuantumExtensionValidationError,
  MONEY_KARMA_CONSTITUTION_SCHEMA,
  MONEY_KARMA_CONSTITUTION_SHA256,
  REVIEWED_BASE_SHA256,
  REVIEWED_QUANTUM_NORMATIVE_SHA256,
  parseAndValidateQuantumExtension,
  validateQuantumExtension,
  validateMoneyKarmaConstitutionBinding,
} from "./validate-constructive-intelligence-quantum-qec.mjs";

const extensionRaw = readFileSync(
  new URL("../public/standards/constructive-intelligence-quantum-qec.v0.json", import.meta.url),
  "utf8",
);
const baseRaw = readFileSync(
  new URL("../public/standards/constructive-intelligence-tree.v1.json", import.meta.url),
  "utf8",
);
const constitutionRaw = readFileSync(
  new URL("../../docs/constitution/money-karma-v1.json", import.meta.url),
);
const canonical = JSON.parse(extensionRaw);

function copy() {
  return structuredClone(canonical);
}

function node(tree, id) {
  const found = tree.nodes.find((candidate) => candidate.id === id);
  assert.ok(found, `missing node ${id}`);
  return found;
}

function assertInvalid(operation, path) {
  assert.throws(
    operation,
    (error) =>
      error instanceof QuantumExtensionValidationError &&
      (path === undefined || error.path === path),
  );
}

describe("quantum QEC constructive-intelligence extension", () => {
  it("validates the reviewed 13-node extension and its exact digest", () => {
    const result = parseAndValidateQuantumExtension(extensionRaw, baseRaw);
    assert.deepEqual(result, {
      nodeCount: 13,
      edgeCount: 24,
      maxDepth: 8,
      maxFanOut: 16,
      questCount: 1,
      normativeDigest: REVIEWED_QUANTUM_NORMATIVE_SHA256,
    });
  });

  it("pins the immutable base bytes and canonical policy", () => {
    assert.equal(canonical.base.documentSha256, `sha256:${REVIEWED_BASE_SHA256}`);
    const driftedBase = baseRaw.replace('"snapshotDate": "2026-07-29"', '"snapshotDate": "2026-07-30"');
    assertInvalid(
      () => validateQuantumExtension(copy(), driftedBase, { enforceReviewedDigest: false }),
      "$.base.documentSha256",
    );
    const driftedPin = copy();
    driftedPin.base.policySha256 = `sha256:${"0".repeat(64)}`;
    assertInvalid(
      () => validateQuantumExtension(driftedPin, baseRaw, { enforceReviewedDigest: false }),
      "$.base.policySha256",
    );
  });

  it("pins the raw Money-KARMA constitution and rejects checked-in drift", () => {
    assert.deepEqual(canonical.constitutionBinding, {
      schema: MONEY_KARMA_CONSTITUTION_SCHEMA,
      documentSha256: `sha256:${MONEY_KARMA_CONSTITUTION_SHA256}`,
    });
    assert.equal(
      createHash("sha256").update(constitutionRaw).digest("hex"),
      MONEY_KARMA_CONSTITUTION_SHA256,
    );

    const substituted = copy();
    substituted.constitutionBinding.documentSha256 = `sha256:${"0".repeat(64)}`;
    assertInvalid(
      () => validateQuantumExtension(substituted, baseRaw, { enforceReviewedDigest: false }),
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

  it("keeps every authority, network, money, qualification, and safety boundary false", () => {
    for (const key of ["authoritative", "networkObserved", "rewardBearing"]) {
      const tree = copy();
      tree[key] = true;
      assertInvalid(() => validateQuantumExtension(tree, baseRaw, { enforceReviewedDigest: false }), `$.${key}`);
    }
    for (const key of Object.keys(canonical.releaseBoundary)) {
      const tree = copy();
      tree.releaseBoundary[key] = true;
      assertInvalid(
        () => validateQuantumExtension(tree, baseRaw, { enforceReviewedDigest: false }),
        `$.releaseBoundary.${key}`,
      );
    }
  });

  it("refuses any funded, claimable, founder, KARMA, or time-based authority", () => {
    const mutations = [
      ["fundedAmount", "1"],
      ["claimable", true],
      ["founderShareBps", 1],
      ["founderReservedSeats", 1],
      ["karmaWeightBps", 1],
      ["rewardDirectlyGrantsGovernanceAuthority", true],
      ["rewardDenomIsBondableUnderCurrentProtocol", false],
      ["skillUnlockCreatesReward", true],
      ["timeAloneUnlocksEvidence", true],
    ];
    for (const [key, value] of mutations) {
      const tree = copy();
      tree.rewardPolicy[key] = value;
      assertInvalid(
        () => validateQuantumExtension(tree, baseRaw, { enforceReviewedDigest: false }),
        `$.rewardPolicy.${key}`,
      );
    }
    for (const key of [
      "zeroneMintsOrCreates",
      "assignable",
      "operatorAssignable",
      "founderAssignable",
      "recordingClaimsRelationOwnership",
      "rawEventsEstablishCandidateStatus",
      "rawEventCountEstablishesCandidateStatus",
      "rawEventCountAffectsSelectionProbability",
      "transferable",
      "scalarRank",
      "truthOracle",
      "payoutWeight",
      "voteWeight",
      "founderReservedPower",
    ]) {
      const tree = copy();
      tree.karma[key] = true;
      assertInvalid(
        () => validateQuantumExtension(tree, baseRaw, { enforceReviewedDigest: false }),
        `$.karma.${key}`,
      );
    }
    assert.ok(
      canonical.karma.activationRequires.includes(
        "prospective-named-upgrade-replay-tests-and-activation-height",
      ),
    );
    const missingProspectiveGate = copy();
    missingProspectiveGate.karma.activationRequires =
      missingProspectiveGate.karma.activationRequires.filter(
        (gate) => gate !== "prospective-named-upgrade-replay-tests-and-activation-height",
      );
    assertInvalid(
      () => validateQuantumExtension(missingProspectiveGate, baseRaw, { enforceReviewedDigest: false }),
      "$.karma.activationRequires",
    );
  });

  it("never emits a performance pass without a prospective funded-case rule", () => {
    assert.deepEqual(canonical.performanceDecision, {
      mode: "MEASUREMENT_COMPLETENESS_ONLY",
      mayEmitPerformancePass: false,
      measurementPrecisionAloneEstablishesPerformance: false,
      missingProspectiveRuleDisposition: "INCONCLUSIVE_NO_PASS",
      fundedCaseRequiredBindings: [
        "baseline-comparator-digest",
        "baseline-comparator-independent-review-receipt",
        "comparison-direction",
        "confidence-bound-decision-rule",
        "effect-size-or-equivalence-margin",
        "estimand-null-direction",
        "latency-deadline-and-quantile",
        "multi-metric-tradeoff-or-pareto-rule",
        "multiple-comparison-policy",
        "negative-result-routing",
        "resource-match",
      ],
    });

    const preciseButBadDecoder = copy();
    preciseButBadDecoder.performanceDecision.measurementPrecisionAloneEstablishesPerformance = true;
    assertInvalid(
      () => validateQuantumExtension(preciseButBadDecoder, baseRaw, { enforceReviewedDigest: false }),
      "$.performanceDecision.measurementPrecisionAloneEstablishesPerformance",
    );

    const inventedPass = copy();
    inventedPass.performanceDecision.mayEmitPerformancePass = true;
    assertInvalid(
      () => validateQuantumExtension(inventedPass, baseRaw, { enforceReviewedDigest: false }),
      "$.performanceDecision.mayEmitPerformancePass",
    );

    const missingRuleBinding = copy();
    missingRuleBinding.performanceDecision.fundedCaseRequiredBindings.pop();
    assertInvalid(
      () => validateQuantumExtension(missingRuleBinding, baseRaw, { enforceReviewedDigest: false }),
      "$.performanceDecision.fundedCaseRequiredBindings",
    );
  });

  it("preserves both conserved reward axes without creating an entitlement", () => {
    const milestoneDrift = copy();
    milestoneDrift.rewardPolicy.milestones[3].rewardBps += 1;
    assertInvalid(
      () => validateQuantumExtension(milestoneDrift, baseRaw, { enforceReviewedDigest: false }),
      "$.rewardPolicy.milestones[3].rewardBps",
    );
    const attributionDrift = copy();
    attributionDrift.rewardPolicy.attributionBps.originatingArtifact += 1;
    assertInvalid(
      () => validateQuantumExtension(attributionDrift, baseRaw, { enforceReviewedDigest: false }),
      "$.rewardPolicy.attributionBps.originatingArtifact",
    );
    assert.equal(
      canonical.rewardPolicy.milestones.reduce((sum, milestone) => sum + milestone.rewardBps, 0) +
        canonical.rewardPolicy.challengeReserveBps,
      10_000,
    );
    assert.equal(
      Object.values(canonical.rewardPolicy.attributionBps).reduce((sum, value) => sum + value, 0),
      10_000,
    );
    assert.equal(
      canonical.rewardAccounting.accountingBoundary.milestoneAndAttributionAxesAreAdditive,
      false,
    );

    const accountingMutations = [
      ["milestoneAndAttributionAxesAreAdditive", true],
      ["crossAxisAllocationRule", "ASSUME_MULTIPLICATION"],
      ["roundingRule", "round-half-up"],
      ["escrowCompartmentsBound", true],
      ["singleSettlementImplemented", true],
      ["verifiedCostCapAmount", "1"],
      ["reviewerBudgetCapAmount", "1"],
      ["futureReviewerBudgetMustBeOutcomeIndependent", false],
      ["roleCollapseRule", "ALLOW_SELF_REVIEW"],
      ["deterministicRefundRule", "REFUND_ORIGINATOR"],
      ["reviewAttributionPaysAdjudicator", true],
      ["unusedChallengeReserveRoute", "founder"],
    ];
    for (const [key, value] of accountingMutations) {
      const tree = copy();
      tree.rewardAccounting.accountingBoundary[key] = value;
      assertInvalid(
        () => validateQuantumExtension(tree, baseRaw, { enforceReviewedDigest: false }),
        `$.rewardAccounting.accountingBoundary.${key}`,
      );
    }
    const eligibilityEntitlement = copy();
    eligibilityEntitlement.rewardAccounting.nodeEligibilitySemantics = "QUALIFIES";
    assertInvalid(
      () => validateQuantumExtension(eligibilityEntitlement, baseRaw, { enforceReviewedDigest: false }),
      "$.rewardAccounting.nodeEligibilitySemantics",
    );
  });

  it("keeps future KARMA filtering controller-capped, randomized, and unimplemented", () => {
    assert.equal(canonical.karma.eventType, "zerone.karma.edge");
    assert.equal(canonical.karma.eventRegister, "priced-coherence");
    assert.equal(canonical.karma.meaning, "DOMAIN_RELATIONS_NOT_HUMAN_WORTH_OR_TRUTH");
    assert.equal(
      canonical.karma.futureUse,
      "domain-scoped-controller-capped-randomized-candidate-filter-only",
    );
    assert.deepEqual(canonical.karma.futureCandidateFilterRequirements, {
      runtimeEnforced: false,
      sameControllerEdgesExcluded: true,
      selfEdgesExcluded: true,
      reciprocalEdgesExcluded: true,
      correlatedFunderEdgesExcluded: true,
      controllerMergesOnlyReduceUnits: true,
      maximumLotteryUnitsPerController: 1,
      candidateSetFrozenBeforeRandomness: true,
      unbiasedRandomnessRequired: true,
      operatorOverrideAllowed: false,
      countProportionalProbabilityAllowed: false,
    });

    const operatorOverride = copy();
    operatorOverride.karma.futureCandidateFilterRequirements.operatorOverrideAllowed = true;
    assertInvalid(
      () => validateQuantumExtension(operatorOverride, baseRaw, { enforceReviewedDigest: false }),
      "$.karma.futureCandidateFilterRequirements.operatorOverrideAllowed",
    );
    const countWeighted = copy();
    countWeighted.karma.futureCandidateFilterRequirements.countProportionalProbabilityAllowed = true;
    assertInvalid(
      () => validateQuantumExtension(countWeighted, baseRaw, { enforceReviewedDigest: false }),
      "$.karma.futureCandidateFilterRequirements.countProportionalProbabilityAllowed",
    );
    const extraControllerUnit = copy();
    extraControllerUnit.karma.futureCandidateFilterRequirements.maximumLotteryUnitsPerController = 2;
    assertInvalid(
      () => validateQuantumExtension(extraControllerUnit, baseRaw, { enforceReviewedDigest: false }),
      "$.karma.futureCandidateFilterRequirements.maximumLotteryUnitsPerController",
    );
  });

  it("keeps B0-B5 retrospective and unassignable", () => {
    const assigned = copy();
    assigned.breakthroughLens[4].assignable = true;
    assertInvalid(
      () => validateQuantumExtension(assigned, baseRaw, { enforceReviewedDigest: false }),
      "$.breakthroughLens[4].assignable",
    );
    const selfSelected = copy();
    selfSelected.breakthroughLens[4].requires = "author says breakthrough";
    assertInvalid(
      () => validateQuantumExtension(selfSelected, baseRaw, { enforceReviewedDigest: false }),
      "$.breakthroughLens[4].requires",
    );
  });

  it("rejects malformed, duplicate-key, unknown-field, and oversized documents", () => {
    assertInvalid(() => parseAndValidateQuantumExtension("{", baseRaw), "$");
    const duplicate = extensionRaw.replace(
      '"authoritative": false,',
      '"authoritative": true, "authoritative": false,',
    );
    assertInvalid(() => parseAndValidateQuantumExtension(duplicate, baseRaw), "$.authoritative");
    const unknown = copy();
    unknown.livePrize = "1000000";
    assertInvalid(() => validateQuantumExtension(unknown, baseRaw, { enforceReviewedDigest: false }), "$");
    assertInvalid(
      () => parseAndValidateQuantumExtension(" ".repeat(QUANTUM_EXTENSION_MAX_BYTES + 1), baseRaw),
      "$",
    );
  });

  it("rejects missing, cyclic, duplicate, and unsorted prerequisites", () => {
    const missing = copy();
    node(missing, "math-complex-linear-algebra@1").prerequisites = ["missing@1"];
    assertInvalid(() => validateQuantumExtension(missing, baseRaw, { enforceReviewedDigest: false }));

    const cyclic = copy();
    node(cyclic, "math-complex-linear-algebra@1").prerequisites = ["math-tensor-spectral-operators@1"];
    assertInvalid(() => validateQuantumExtension(cyclic, baseRaw, { enforceReviewedDigest: false }), "$.nodes");

    const duplicated = copy();
    const quest = node(duplicated, "quest-quantum-decoder-correlated-noise@1");
    quest.prerequisites.push(quest.prerequisites[0]);
    quest.prerequisites.sort();
    assertInvalid(() => validateQuantumExtension(duplicated, baseRaw, { enforceReviewedDigest: false }));

    const unsorted = copy();
    node(unsorted, "math-quantum-states-dynamics@1").prerequisites.reverse();
    assertInvalid(() => validateQuantumExtension(unsorted, baseRaw, { enforceReviewedDigest: false }));
  });

  it("rejects prerequisites from a later stage", () => {
    const inverted = copy();
    node(inverted, "math-complex-linear-algebra@1").prerequisites = [
      "assurance-formal-verification@1",
    ];
    assertInvalid(
      () => validateQuantumExtension(inverted, baseRaw, { enforceReviewedDigest: false }),
      "$.nodes[3].prerequisites[0]",
    );
  });

  it("requires frozen scope, held-out seeds, matched resources, uncertainty, and independent adoption", () => {
    const quest = canonical.nodes.at(-1);
    assert.match(quest.acceptance.scopeBounds.join(" "), /random-seeds=committed-before-execution/);
    assert.match(quest.artifactRequirements.join(" "), /uncertainty intervals/);
    assert.match(quest.artifactRequirements.join(" "), /energy accounting/);
    assert.deepEqual(
      quest.acceptance.fixtures.map(({ n, k, distance }) => [n, k, distance]),
      [[72, 12, 6], [90, 8, 10], [144, 12, 12]],
    );
    assert.deepEqual(
      quest.acceptance.physicalErrorGrid,
      ["0.001", "0.002", "0.003", "0.004", "0.005", "0.006"],
    );

    const scopeDrift = copy();
    node(scopeDrift, quest.id).acceptance.scopeBounds[0] = "baseline=chosen-after-results";
    assertInvalid(() => validateQuantumExtension(scopeDrift, baseRaw, { enforceReviewedDigest: false }));

    const collapsedReplication = copy();
    node(collapsedReplication, quest.id).acceptance.minimumOrganizationRoots = 1;
    assertInvalid(
      () => validateQuantumExtension(collapsedReplication, baseRaw, { enforceReviewedDigest: false }),
      "$.nodes[12].acceptance.minimumOrganizationRoots",
    );

    const noAdoption = copy();
    node(noAdoption, quest.id).acceptance.adoptionReceiptTypes = ["self-use"];
    assertInvalid(() => validateQuantumExtension(noAdoption, baseRaw, { enforceReviewedDigest: false }));
  });

  it("binds exact per-cell science, digests, precision gates, and compute-cap semantics", () => {
    const questId = "quest-quantum-decoder-correlated-noise@1";
    const acceptance = node(canonical, questId).acceptance;
    assert.equal(
      acceptance.cellDefinition,
      "fixture-x-physical-error-x-implementation-root-x-execution-environment",
    );
    assert.ok(acceptance.requiredCaseBindings.includes("circuit-bundle-sha256-per-fixture"));
    assert.equal(
      acceptance.circuitProvenance,
      "stim-circuits-from-version-of-record-references-27-and-50",
    );
    assert.equal(acceptance.distance10And12CircuitEnsembleSize, 24);
    assert.equal(acceptance.computeCapExhaustion, "inconclusive-no-pass");
    assert.equal(acceptance.rareEventAlternative.appliesTo, "logical-error-cells-only");
    assert.equal(acceptance.rareEventAlternative.unbiasedEstimatorRequired, true);
    assert.deepEqual(acceptance.rareEventAlternative.replacesDirectGates, [
      "minimum-cases-per-cell",
      "minimum-logical-failures-per-cell",
    ]);
    assert.equal(acceptance.rareEventAlternative.confidenceLevelBps, 9900);
    assert.equal(
      acceptance.rareEventAlternative.confidenceProcedure,
      "two-sided-estimator-specific-interval",
    );
    assert.equal(acceptance.rareEventAlternative.maximumRelativeHalfWidthBps, 3000);
    assert.equal(acceptance.rareEventAlternative.mayEmitPerformancePass, false);

    const wrongFixture = copy();
    node(wrongFixture, questId).acceptance.fixtures[0].n = 73;
    assertInvalid(
      () => validateQuantumExtension(wrongFixture, baseRaw, { enforceReviewedDigest: false }),
      "$.nodes[12].acceptance.fixtures[0].n",
    );

    const missingGridCell = copy();
    node(missingGridCell, questId).acceptance.physicalErrorGrid.pop();
    assertInvalid(
      () => validateQuantumExtension(missingGridCell, baseRaw, { enforceReviewedDigest: false }),
      "$.nodes[12].acceptance.physicalErrorGrid",
    );

    const missingCircuitBinding = copy();
    node(missingCircuitBinding, questId).acceptance.requiredCaseBindings =
      node(missingCircuitBinding, questId).acceptance.requiredCaseBindings.filter(
        (binding) => binding !== "circuit-bundle-sha256-per-fixture",
      );
    assertInvalid(
      () => validateQuantumExtension(missingCircuitBinding, baseRaw, { enforceReviewedDigest: false }),
      "$.nodes[12].acceptance.requiredCaseBindings",
    );

    const computePass = copy();
    node(computePass, questId).acceptance.computeCapExhaustion = "pass";
    assertInvalid(
      () => validateQuantumExtension(computePass, baseRaw, { enforceReviewedDigest: false }),
      "$.nodes[12].acceptance.computeCapExhaustion",
    );

    const smallerEnsemble = copy();
    node(smallerEnsemble, questId).acceptance.distance10And12CircuitEnsembleSize = 1;
    assertInvalid(
      () => validateQuantumExtension(smallerEnsemble, baseRaw, { enforceReviewedDigest: false }),
      "$.nodes[12].acceptance.distance10And12CircuitEnsembleSize",
    );

    const biasedAlternative = copy();
    node(biasedAlternative, questId).acceptance.rareEventAlternative.unbiasedEstimatorRequired = false;
    assertInvalid(
      () => validateQuantumExtension(biasedAlternative, baseRaw, { enforceReviewedDigest: false }),
      "$.nodes[12].acceptance.rareEventAlternative.unbiasedEstimatorRequired",
    );

    const replacedTooMuch = copy();
    node(replacedTooMuch, questId).acceptance.rareEventAlternative.replacesDirectGates.push(
      "confidence-interval",
    );
    assertInvalid(
      () => validateQuantumExtension(replacedTooMuch, baseRaw, { enforceReviewedDigest: false }),
      "$.nodes[12].acceptance.rareEventAlternative.replacesDirectGates",
    );

    const oneSided = copy();
    node(oneSided, questId).acceptance.rareEventAlternative.confidenceProcedure =
      "one-sided-estimator-interval";
    assertInvalid(
      () => validateQuantumExtension(oneSided, baseRaw, { enforceReviewedDigest: false }),
      "$.nodes[12].acceptance.rareEventAlternative.confidenceProcedure",
    );

    const inventedRarePass = copy();
    node(inventedRarePass, questId).acceptance.rareEventAlternative.mayEmitPerformancePass = true;
    assertInvalid(
      () => validateQuantumExtension(inventedRarePass, baseRaw, { enforceReviewedDigest: false }),
      "$.nodes[12].acceptance.rareEventAlternative.mayEmitPerformancePass",
    );
  });

  it("requires exact logical-error coverage and limits zero-miss latency to measurement completeness", () => {
    const questId = "quest-quantum-decoder-correlated-noise@1";
    const targets = node(canonical, questId).acceptance.coverageTargets;
    assert.deepEqual(
      targets.map(({ id, minimumCasesPerCell }) => [id, minimumCasesPerCell]),
      [
        ["baseline-regression", 100_000],
        ["correlated-noise-logical-error-rate", 1_000_000],
        ["matched-resource-latency", 100_000],
      ],
    );
    assert.equal(targets[0].minimumLogicalFailuresPerCell, 100);
    assert.equal(targets[1].confidenceLevelBps, 9900);
    assert.equal(targets[1].maximumRelativeHalfWidthBps, 3000);
    assert.equal(targets[2].zeroDeadlineMissesMaySatisfyMeasurementCompleteness, true);
    assert.equal(
      targets[2].confidenceProcedure,
      "two-sided-quantile-interval-or-one-sided-deadline-miss-upper-bound",
    );

    const arbitraryCoverage = copy();
    node(arbitraryCoverage, questId).acceptance.coverageTargets[0].id = "easy-target";
    assertInvalid(
      () => validateQuantumExtension(arbitraryCoverage, baseRaw, { enforceReviewedDigest: false }),
      "$.nodes[12].acceptance.coverageTargets[0].id",
    );

    const zeroCases = copy();
    node(zeroCases, questId).acceptance.coverageTargets[1].minimumCasesPerCell = 0;
    assertInvalid(
      () => validateQuantumExtension(zeroCases, baseRaw, { enforceReviewedDigest: false }),
      "$.nodes[12].acceptance.coverageTargets[1].minimumCasesPerCell",
    );

    const tooFewFailures = copy();
    node(tooFewFailures, questId).acceptance.coverageTargets[1].minimumLogicalFailuresPerCell = 99;
    assertInvalid(
      () => validateQuantumExtension(tooFewFailures, baseRaw, { enforceReviewedDigest: false }),
      "$.nodes[12].acceptance.coverageTargets[1].minimumLogicalFailuresPerCell",
    );

    const latencyNeedsMisses = copy();
    node(latencyNeedsMisses, questId).acceptance.coverageTargets[2].zeroDeadlineMissesMaySatisfyMeasurementCompleteness = false;
    assertInvalid(
      () => validateQuantumExtension(latencyNeedsMisses, baseRaw, { enforceReviewedDigest: false }),
      "$.nodes[12].acceptance.coverageTargets[2].zeroDeadlineMissesMaySatisfyMeasurementCompleteness",
    );
  });

  it("pins the paper as a reproduction target rather than a truth oracle", () => {
    assert.equal(canonical.standards[0].treatment, "reproduction-target-not-truth-oracle");
    assert.equal(canonical.standards[0].revision, "version of record 2026-05-01");
    assert.equal(canonical.standards[0].authorityStatus, "open-access version of record");
    const oracle = copy();
    oracle.standards[0].treatment = "truth-oracle";
    assertInvalid(
      () => validateQuantumExtension(oracle, baseRaw, { enforceReviewedDigest: false }),
      "$.standards[0].treatment",
    );
  });

  it("detects any unreviewed normative drift after structural validation", () => {
    const drift = copy();
    drift.title = "Unreviewed title";
    assertInvalid(() => validateQuantumExtension(drift, baseRaw), "$");
  });
});

import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";

import {
  MATH_FRONTIER_MAX_BYTES,
  MathFrontierValidationError,
  parseAndValidateConstructiveIntelligenceMathFrontier,
  parseAndValidateMathProblemPacket,
  validateConstructiveIntelligenceMathFrontier,
  validateMathProblemPacket,
} from "./validate-constructive-intelligence-math-frontier.mjs";

const frontierRaw = readFileSync(
  new URL(
    "../public/standards/constructive-intelligence-math-frontier.v0.json",
    import.meta.url,
  ),
  "utf8",
);
const baseTreeRaw = readFileSync(
  new URL(
    "../public/standards/constructive-intelligence-tree.v1.json",
    import.meta.url,
  ),
  "utf8",
);
const problemRaw = readFileSync(
  new URL(
    "../../docs/examples/math-frontier/formal-construction-v0.problem.json",
    import.meta.url,
  ),
  "utf8",
);
const canonical = JSON.parse(frontierRaw);
const canonicalProblem = JSON.parse(problemRaw);

function frontierCopy() {
  return structuredClone(canonical);
}

function problemCopy() {
  return structuredClone(canonicalProblem);
}

function validate(value) {
  return validateConstructiveIntelligenceMathFrontier(value, {
    baseTreeRaw,
    exactRaw: frontierRaw,
  });
}

function assertInvalid(operation, path) {
  assert.throws(
    operation,
    (error) =>
      error instanceof MathFrontierValidationError &&
      (path === undefined || error.path === path),
  );
}

describe("constructive-intelligence Math Frontier v0", () => {
  it("validates the reviewed frontier and known-answer packet", () => {
    const result = parseAndValidateConstructiveIntelligenceMathFrontier(
      frontierRaw,
      baseTreeRaw,
    );
    assert.deepEqual(
      {
        nodeCount: result.nodeCount,
        stageCount: result.stageCount,
        liveAmount: result.liveAmount,
        karmaState: result.karmaState,
        economicEffect: result.economicEffect,
        controlEffect: result.controlEffect,
        canonicalSha256: result.canonicalSha256,
        documentSha256: result.documentSha256,
      },
      {
        nodeCount: 13,
        stageCount: 4,
        liveAmount: "0",
        karmaState: "ORDINAL",
        economicEffect: "NONE",
        controlEffect: "NONE",
        canonicalSha256:
          "b6260de31969a56e601a5a81f1b4f7c1c68fcd34f9aa68b35ce2701c3f012503",
        documentSha256:
          "4fdcd54c35c69c26a28c385275688351ee2a9131702e81bacf100de8d7612456",
      },
    );
    const packet = parseAndValidateMathProblemPacket(
      problemRaw,
      canonical,
      frontierRaw,
    );
    assert.equal(
      packet.packetSha256,
      "f0961813b83cbd9f127290c19cf5bc98c07cad2dc1158bbab26243edd7af9ae3",
    );
  });

  it("rejects malformed, duplicate-key, null and oversized documents", () => {
    assertInvalid(
      () =>
        parseAndValidateConstructiveIntelligenceMathFrontier(
          "{",
          baseTreeRaw,
        ),
      "$",
    );
    const duplicate = frontierRaw.replace(
      '"authoritative": false',
      '"authoritative": false,\n  "authoritative": false',
    );
    assertInvalid(
      () =>
        parseAndValidateConstructiveIntelligenceMathFrontier(
          duplicate,
          baseTreeRaw,
        ),
      "$.authoritative",
    );
    const withNull = frontierRaw.replace(
      '"authoritative": false',
      '"authoritative": null',
    );
    assertInvalid(
      () =>
        parseAndValidateConstructiveIntelligenceMathFrontier(
          withNull,
          baseTreeRaw,
        ),
      "$.authoritative",
    );
    assertInvalid(
      () =>
        parseAndValidateConstructiveIntelligenceMathFrontier(
          " ".repeat(MATH_FRONTIER_MAX_BYTES + 1),
          baseTreeRaw,
        ),
      "$",
    );
  });

  it("rejects unknown and identity-privilege fields", () => {
    for (const [key, value] of [
      ["founder", "zrn1founder"],
      ["owner", "zrn1owner"],
      ["admin", "zrn1admin"],
      ["beneficiary", "zrn1beneficiary"],
      ["payoutAddress", "zrn1payout"],
      ["reservedShareBps", 1],
    ]) {
      const tree = frontierCopy();
      tree[key] = value;
      assertInvalid(() => validate(tree), `$.${key}`);
    }
  });

  it("keeps every consensus, value, qualification and control boundary false", () => {
    for (const key of [
      "authoritative",
      "networkObserved",
      "rewardBearing",
      "governanceBearing",
    ]) {
      const tree = frontierCopy();
      tree[key] = true;
      assertInvalid(() => validate(tree), `$.${key}`);
    }
    for (const key of Object.keys(canonical.releaseBoundary)) {
      const tree = frontierCopy();
      tree.releaseBoundary[key] = true;
      assertInvalid(() => validate(tree), `$.releaseBoundary.${key}`);
    }
  });

  it("allows no creator, operator, human, AI, stake or wealth privilege", () => {
    const mutations = {
      identityLabelInvariant: false,
      reservedSeatsAllowed: true,
      reservedSharesAllowed: true,
      creatorPrivilege: true,
      operatorPrivilege: true,
      humanAiClassMultiplier: true,
      stakeAffectsValidity: true,
      stakeAffectsVoice: true,
      wealthAffectsEligibility: true,
      wealthAffectsReward: true,
      rewardBalanceAffectsVoice: true,
      karmaMagnitudeAffectsVoice: true,
    };
    for (const [key, value] of Object.entries(mutations)) {
      const tree = frontierCopy();
      tree.constitution[key] = value;
      assertInvalid(() => validate(tree), `$.constitution.${key}`);
    }
  });

  it("makes future KARMA governance controller-equal, conflict-aware and dark", () => {
    const mutations = {
      karmaEligibilityOnly: false,
      controllerMergeCanOnlyReduceVoice: false,
      controllerConflictRecusalRequired: false,
      recusalQuorumRecomputed: false,
      selectedControllerVoice: "KARMA_WEIGHTED",
      currentActivationAuthority: "FOUNDER",
      futureGovernance: "LIVE",
      ordinaryStakeVoteCanActivate: true,
      emergencyAuthorityCanActivate: true,
      emergencyAuthorityScope: "AWARD",
    };
    for (const [key, value] of Object.entries(mutations)) {
      const tree = frontierCopy();
      tree.constitution[key] = value;
      assertInvalid(() => validate(tree), `$.constitution.${key}`);
    }
  });

  it("keeps KARMA ordinal, non-recognizing, non-transferable and weightless", () => {
    const mutations = {
      mode: "RECOGNIZED",
      state: "RECOGNIZED",
      truthOracle: true,
      onChainRecognition: true,
      magnitude: "1",
      transferable: true,
      spendable: true,
      rewardMultiplier: true,
      governanceWeight: true,
      economicEffect: "REWARD",
      controlEffect: "VOTE",
      futureRecognitionRequiresSignedReceipts: false,
      futureRecognitionRequiresControllerResolution: false,
    };
    for (const [key, value] of Object.entries(mutations)) {
      const tree = frontierCopy();
      tree.karma[key] = value;
      assertInvalid(() => validate(tree), `$.karma.${key}`);
    }
  });

  it("does not recognize self, same-controller, reciprocal or raw external edges", () => {
    const tree = frontierCopy();
    tree.karma.excludedFromRecognition = ["self"];
    assertInvalid(() => validate(tree), "$.karma.excludedFromRecognition");
  });

  it("keeps the live amount zero and issuance disabled", () => {
    const mutations = {
      economicEffect: "TRANSFER",
      liveAmount: "1",
      skillUnlockCreatesEntitlement: true,
      breakthroughCreatesEntitlement: true,
      protocolIssuanceAllowed: true,
      dedicatedEscrowRequired: false,
      policyFrozenBeforeAdmission: false,
      singleSettlementRequired: false,
      specializedResearchSpendPathAllowed: true,
      disproofDisposition: "E4_TO_CLAIMANT",
    };
    for (const [key, value] of Object.entries(mutations)) {
      const tree = frontierCopy();
      tree.rewardTemplate[key] = value;
      assertInvalid(() => validate(tree), `$.rewardTemplate.${key}`);
    }
  });

  it("pins the balanced milestone envelope and counterexample role", () => {
    const changed = frontierCopy();
    changed.rewardTemplate.milestones[0].outcomePoolBps += 1;
    assertInvalid(
      () => validate(changed),
      "$.rewardTemplate.milestones[0].outcomePoolBps",
    );
    const noFalsifier = frontierCopy();
    noFalsifier.rewardTemplate.roles =
      noFalsifier.rewardTemplate.roles.filter((role) => role !== "falsifier");
    assertInvalid(() => validate(noFalsifier), "$.rewardTemplate.roles");
  });

  it("binds exact reviewed base-tree bytes and policy", () => {
    const wrongDocument = frontierCopy();
    wrongDocument.baseTree.documentSha256 =
      "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
    assertInvalid(
      () => validate(wrongDocument),
      "$.baseTree.documentSha256",
    );
    const mutatedBase = baseTreeRaw.replace(
      '"authoritative": false',
      '"authoritative": true',
    );
    assertInvalid(
      () =>
        validateConstructiveIntelligenceMathFrontier(frontierCopy(), {
          baseTreeRaw: mutatedBase,
          exactRaw: frontierRaw,
        }),
    );
  });

  it("rejects missing, duplicate, unsorted and unknown node references", () => {
    const duplicate = frontierCopy();
    duplicate.nodes[1].id = duplicate.nodes[0].id;
    assertInvalid(() => validate(duplicate), "$.nodes");

    const unsorted = frontierCopy();
    [unsorted.nodes[0], unsorted.nodes[1]] = [
      unsorted.nodes[1],
      unsorted.nodes[0],
    ];
    assertInvalid(() => validate(unsorted), "$.nodes");

    const unknown = frontierCopy();
    unknown.nodes[0].prerequisites.push("math-missing@1");
    unknown.nodes[0].prerequisites.sort();
    assertInvalid(() => validate(unknown), "$.nodes[0].prerequisites");
  });

  it("rejects prerequisite cycles and reward-unlocking nodes", () => {
    const cycle = frontierCopy();
    const foundations = cycle.nodes.find(
      (node) => node.id === "math-foundations-logic@1",
    );
    foundations.prerequisites.push("math-prior-art-semantic-root@1");
    foundations.prerequisites.sort();
    assertInvalid(() => validate(cycle), "$.nodes");

    const reward = frontierCopy();
    reward.nodes[0].unlocksReward = true;
    assertInvalid(() => validate(reward), "$.nodes[0].unlocksReward");
  });

  it("derives breakthrough only after E5 and never from popularity", () => {
    const selfDeclared = frontierCopy();
    selfDeclared.questTemplate.selfDeclaredBreakthrough = true;
    assertInvalid(
      () => validate(selfDeclared),
      "$.questTemplate.selfDeclaredBreakthrough",
    );
    const popular = frontierCopy();
    popular.questTemplate.popularityCanOverrideValidity = true;
    assertInvalid(
      () => validate(popular),
      "$.questTemplate.popularityCanOverrideValidity",
    );
    const early = frontierCopy();
    early.questTemplate.breakthroughStatus = "DERIVED_AFTER_E2";
    assertInvalid(
      () => validate(early),
      "$.questTemplate.breakthroughStatus",
    );
    const noRelationPolicy = frontierCopy();
    noRelationPolicy.questTemplate.relationSpecificValidityRequired = false;
    assertInvalid(
      () => validate(noRelationPolicy),
      "$.questTemplate.relationSpecificValidityRequired",
    );
    const implementationClaimsTruth = frontierCopy();
    implementationClaimsTruth.questTemplate.implementsEstablishesTheoremValidity = true;
    assertInvalid(
      () => validate(implementationClaimsTruth),
      "$.questTemplate.implementsEstablishesTheoremValidity",
    );
  });

  it("requires exactly one frozen E2 domain-attainment receipt", () => {
    const mode = frontierCopy();
    mode.questTemplate.domainSelectionMode = "ANY";
    assertInvalid(
      () => validate(mode),
      "$.questTemplate.domainSelectionMode",
    );
    const evidence = frontierCopy();
    evidence.questTemplate.selectedDomainEvidenceMinimum = "E1";
    assertInvalid(
      () => validate(evidence),
      "$.questTemplate.selectedDomainEvidenceMinimum",
    );
    const noReceipt = frontierCopy();
    noReceipt.questTemplate.selectedDomainReceiptRequired = false;
    assertInvalid(
      () => validate(noReceipt),
      "$.questTemplate.selectedDomainReceiptRequired",
    );
  });

  it("pins prospective controller, organization, implementation, environment and kernel diversity floors", () => {
    for (const key of [
      "minimumEffectiveClusters",
      "minimumOrganizationRoots",
      "minimumImplementationRoots",
      "minimumExecutionEnvironments",
      "minimumKernelFamilies",
    ]) {
      const tree = frontierCopy();
      tree.questTemplate[key] = 1;
      assertInvalid(() => validate(tree), `$.questTemplate.${key}`);
    }
  });

  it("prevents semantic-root replay and preserves counterexample eligibility", () => {
    const replay = frontierCopy();
    replay.questTemplate.semanticRootReplayAllowed = true;
    assertInvalid(
      () => validate(replay),
      "$.questTemplate.semanticRootReplayAllowed",
    );
    const noCounterexample = frontierCopy();
    noCounterexample.questTemplate.counterexampleEligible = false;
    assertInvalid(
      () => validate(noCounterexample),
      "$.questTemplate.counterexampleEligible",
    );
  });
});

describe("Math Frontier problem packet", () => {
  it("rejects identity privilege, beneficiary and payout fields", () => {
    for (const key of ["founder", "owner", "admin", "beneficiary", "payoutAddress"]) {
      const packet = problemCopy();
      packet[key] = "forbidden";
      assertInvalid(
        () => validateMathProblemPacket(packet, canonical, frontierRaw),
        `$.${key}`,
      );
    }
  });

  it("rejects a different frontier, quest, domain or relation", () => {
    const frontierDigest = problemCopy();
    frontierDigest.frontierDocumentSha256 =
      "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
    assertInvalid(
      () => validateMathProblemPacket(frontierDigest, canonical, frontierRaw),
      "$.frontierDocumentSha256",
    );
    const questDigest = problemCopy();
    questDigest.questTemplateSha256 =
      "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";
    assertInvalid(
      () => validateMathProblemPacket(questDigest, canonical, frontierRaw),
      "$.questTemplateSha256",
    );
    const domain = problemCopy();
    domain.domainCapabilityId = "math-foundations-logic@1";
    assertInvalid(
      () => validateMathProblemPacket(domain, canonical, frontierRaw),
      "$.domainCapabilityId",
    );
    const relation = problemCopy();
    relation.artifactRelation = "LIKES";
    assertInvalid(
      () => validateMathProblemPacket(relation, canonical, frontierRaw),
      "$.artifactRelation",
    );
  });

  it("requires two distinct checker digests without claiming family independence", () => {
    const packet = problemCopy();
    packet.checkerKernelDigests = [packet.checkerKernelDigests[0]];
    assertInvalid(
      () => validateMathProblemPacket(packet, canonical, frontierRaw),
      "$.checkerKernelDigests",
    );
  });

  it("binds the selected domain's E2 receipt", () => {
    const weak = problemCopy();
    weak.selectedDomainEvidence = "E1";
    assertInvalid(
      () => validateMathProblemPacket(weak, canonical, frontierRaw),
      "$.selectedDomainEvidence",
    );
    const malformed = problemCopy();
    malformed.selectedDomainReceiptDigest = "not-a-digest";
    assertInvalid(
      () => validateMathProblemPacket(malformed, canonical, frontierRaw),
      "$.selectedDomainReceiptDigest",
    );
  });

  it("keeps breakthrough, KARMA, economics, control and qualification inert", () => {
    const mutations = [
      ["breakthroughStatus", "BREAKTHROUGH"],
      ["economicEffect", "REWARD"],
      ["controlEffect", "VOTE"],
      ["qualification", "GRANTED"],
      ["liveAmount", "1"],
    ];
    for (const [key, value] of mutations) {
      const packet = problemCopy();
      packet[key] = value;
      assertInvalid(
        () => validateMathProblemPacket(packet, canonical, frontierRaw),
        `$.${key}`,
      );
    }
    const recognized = problemCopy();
    recognized.karmaProjection.state = "RECOGNIZED";
    assertInvalid(
      () => validateMathProblemPacket(recognized, canonical, frontierRaw),
      "$.karmaProjection.state",
    );
  });

  it("rejects malformed, duplicate-key and future-cutoff packets", () => {
    assertInvalid(
      () => parseAndValidateMathProblemPacket("{", canonical, frontierRaw),
      "$",
    );
    const duplicate = problemRaw.replace(
      '"synthetic": true',
      '"synthetic": true,\n  "synthetic": true',
    );
    assertInvalid(
      () => parseAndValidateMathProblemPacket(duplicate, canonical, frontierRaw),
      "$.synthetic",
    );
    const future = problemCopy();
    future.priorArtCutoff = "2026-08-02";
    assertInvalid(
      () => validateMathProblemPacket(future, canonical, frontierRaw),
      "$.priorArtCutoff",
    );
  });
});

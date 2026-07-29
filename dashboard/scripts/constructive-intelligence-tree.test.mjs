import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import {
  CONSTRUCTIVE_TREE_MAX_BYTES,
  CONSTRUCTIVE_TREE_MAX_DEPTH,
  CONSTRUCTIVE_TREE_MAX_FAN_OUT,
  ConstructiveTreeValidationError,
  parseAndValidateConstructiveIntelligenceTree,
  validateConstructiveIntelligenceTree,
  validateConstructiveIntelligenceTreeForUse,
} from "./validate-constructive-intelligence-tree.mjs";

// The checked-in seed is the positive fixture; every other case mutates it.
const canonical = JSON.parse(
  readFileSync(
    new URL(
      "../public/standards/constructive-intelligence-tree.v1.json",
      import.meta.url,
    ),
    "utf8",
  ),
);

function copyTree() {
  return structuredClone(canonical);
}

function node(tree, id) {
  const found = tree.nodes.find((candidate) => candidate.id === id);
  assert.ok(found, `missing fixture node ${id}`);
  return found;
}

function assertInvalid(operation, path) {
  assert.throws(
    operation,
    (error) =>
      error instanceof ConstructiveTreeValidationError &&
      (path === undefined || error.path === path),
  );
}

describe("constructive-intelligence tree", () => {
  it("validates the 30-node published seed", () => {
    const result = validateConstructiveIntelligenceTree(copyTree());
    assert.equal(result.schema, "zerone.constructive-intelligence-tree/v1");
    assert.equal(result.policyVersion, "1.0.0");
    assert.equal(result.snapshotDate, "2026-07-29");
    assert.equal(result.nodeCount, 30);
    assert.equal(result.questCount, 3);
    assert.equal(result.maxDepth, 8);
    assert.equal(CONSTRUCTIVE_TREE_MAX_DEPTH, 8);
    assert.equal(CONSTRUCTIVE_TREE_MAX_FAN_OUT, 16);
    assert.ok(result.edgeCount > result.nodeCount);
  });

  it("keeps every authority, economic, network, and safety boundary false", () => {
    for (const key of ["authoritative", "networkObserved", "rewardBearing"]) {
      const tree = copyTree();
      tree[key] = true;
      assertInvalid(() => validateConstructiveIntelligenceTree(tree), `$.${key}`);
    }
    for (const key of Object.keys(canonical.releaseBoundary)) {
      const tree = copyTree();
      tree.releaseBoundary[key] = true;
      assertInvalid(
        () => validateConstructiveIntelligenceTree(tree),
        `$.releaseBoundary.${key}`,
      );
    }
  });

  it("rejects unknown fields, malformed JSON, and oversized input", () => {
    const unknown = copyTree();
    unknown.liveRewardMultiplier = 9;
    assertInvalid(
      () => validateConstructiveIntelligenceTree(unknown),
      "$.liveRewardMultiplier",
    );
    assertInvalid(() => parseAndValidateConstructiveIntelligenceTree("{"), "$");
    const duplicateKey = JSON.stringify(canonical).replace(
      '"authoritative":false',
      '"authoritative":true,"authoritative":false',
    );
    assertInvalid(
      () => parseAndValidateConstructiveIntelligenceTree(duplicateKey),
      "$.authoritative",
    );
    assertInvalid(
      () =>
        parseAndValidateConstructiveIntelligenceTree(
          " ".repeat(CONSTRUCTIVE_TREE_MAX_BYTES + 1),
        ),
      "$",
    );
    assertInvalid(
      () =>
        parseAndValidateConstructiveIntelligenceTree(
          `${"[".repeat(65)}0${"]".repeat(65)}`,
        ),
      "$",
    );
  });

  it("rejects duplicate, malformed, and unsorted node identifiers", () => {
    const duplicate = copyTree();
    duplicate.nodes[1].id = duplicate.nodes[0].id;
    assertInvalid(() => validateConstructiveIntelligenceTree(duplicate), "$.nodes");

    const malformed = copyTree();
    malformed.nodes[0].id = "no-version-suffix";
    assertInvalid(
      () => validateConstructiveIntelligenceTree(malformed),
      "$.nodes[0].id",
    );

    const unsorted = copyTree();
    [unsorted.nodes[0], unsorted.nodes[1]] = [
      unsorted.nodes[1],
      unsorted.nodes[0],
    ];
    assertInvalid(() => validateConstructiveIntelligenceTree(unsorted), "$.nodes");
  });

  it("rejects missing, self, duplicate, and unsorted prerequisites", () => {
    const missing = copyTree();
    node(missing, "crypto-aead@1").prerequisites.push("missing-node@1");
    node(missing, "crypto-aead@1").prerequisites.sort();
    assertInvalid(() => validateConstructiveIntelligenceTree(missing), "$.nodes");

    const self = copyTree();
    node(self, "math-proofcraft@1").prerequisites.push("math-proofcraft@1");
    assertInvalid(() => validateConstructiveIntelligenceTree(self), "$.nodes");

    const duplicate = copyTree();
    const duplicateNode = node(duplicate, "crypto-aead@1");
    duplicateNode.prerequisites.push(duplicateNode.prerequisites[0]);
    duplicateNode.prerequisites.sort();
    assertInvalid(
      () => validateConstructiveIntelligenceTree(duplicate),
      `$.nodes[${duplicate.nodes.indexOf(duplicateNode)}].prerequisites`,
    );

    const unsorted = copyTree();
    node(unsorted, "crypto-aead@1").prerequisites.reverse();
    assertInvalid(() => validateConstructiveIntelligenceTree(unsorted));
  });

  it("rejects both short and longer prerequisite cycles", () => {
    const shortCycle = copyTree();
    node(shortCycle, "math-proofcraft@1").prerequisites.push(
      "math-probability-information-complexity@1",
    );
    assertInvalid(() => validateConstructiveIntelligenceTree(shortCycle), "$.nodes");

    const longerCycle = copyTree();
    node(longerCycle, "math-proofcraft@1").prerequisites.push(
      "math-lattices-polynomial-rings@1",
    );
    assertInvalid(() => validateConstructiveIntelligenceTree(longerCycle), "$.nodes");
  });

  it("bounds graph depth and fan-out", () => {
    const tooDeep = copyTree();
    node(
      tooDeep,
      "assurance-coordinated-disclosure@1",
    ).prerequisites.push("assurance-constant-time-memory-safety@1");
    node(
      tooDeep,
      "assurance-coordinated-disclosure@1",
    ).prerequisites.sort();
    assertInvalid(() => validateConstructiveIntelligenceTree(tooDeep), "$.nodes");

    const tooWide = copyTree();
    let additions = 0;
    for (const candidate of tooWide.nodes) {
      if (
        candidate.id !== "math-proofcraft@1" &&
        !candidate.prerequisites.includes("math-proofcraft@1") &&
        additions < 11
      ) {
        candidate.prerequisites.push("math-proofcraft@1");
        candidate.prerequisites.sort();
        additions += 1;
      }
    }
    assert.equal(additions, 11);
    assertInvalid(() => validateConstructiveIntelligenceTree(tooWide), "$.nodes");
  });

  it("enforces the derived-breakthrough, funding, safety, and independence floors", () => {
    const authorSelected = copyTree();
    authorSelected.policy.breakthroughRecognition.authorSelected = true;
    assertInvalid(
      () => validateConstructiveIntelligenceTree(authorSelected),
      "$.policy.breakthroughRecognition.authorSelected",
    );

    const timeOnly = copyTree();
    timeOnly.policy.funding.timeAloneUnlocksEvidence = true;
    assertInvalid(
      () => validateConstructiveIntelligenceTree(timeOnly),
      "$.policy.funding.timeAloneUnlocksEvidence",
    );

    const unsafe = copyTree();
    unsafe.policy.disclosure.safetyIsHardGate = false;
    assertInvalid(
      () => validateConstructiveIntelligenceTree(unsafe),
      "$.policy.disclosure.safetyIsHardGate",
    );

    const weakQuorum = copyTree();
    weakQuorum.policy.independence.minimumEffectiveClusters = 2;
    assertInvalid(
      () => validateConstructiveIntelligenceTree(weakQuorum),
      "$.policy.independence.minimumEffectiveClusters",
    );
  });

  it("preserves exact milestone economics and the challenge reserve", () => {
    const drift = copyTree();
    drift.policy.milestones[2].rewardBps += 1;
    assertInvalid(
      () => validateConstructiveIntelligenceTree(drift),
      "$.policy.milestones[2].rewardBps",
    );

    const reserve = copyTree();
    reserve.policy.challengeReserveBps = 0;
    assertInvalid(
      () => validateConstructiveIntelligenceTree(reserve),
      "$.policy",
    );
  });

  it("allows sponsor milestones only for bounded quests on non-draft standards", () => {
    const ordinaryReward = copyTree();
    node(ordinaryReward, "crypto-aead@1").rewardEligibility =
      "sponsor-milestones";
    assertInvalid(() => validateConstructiveIntelligenceTree(ordinaryReward));

    const noEscalation = copyTree();
    node(
      noEscalation,
      "quest-tls-rfc9846-keyshare-reuse@1",
    ).acceptance.privateEscalationRequired = false;
    assertInvalid(() => validateConstructiveIntelligenceTree(noEscalation));

    const noTriage = copyTree();
    node(
      noTriage,
      "quest-tls-rfc9846-keyshare-reuse@1",
    ).acceptance.prepublicationTriageRequired = false;
    assertInvalid(() => validateConstructiveIntelligenceTree(noTriage));

    const scopeDrift = copyTree();
    node(
      scopeDrift,
      "quest-tls-rfc9846-keyshare-reuse@1",
    ).acceptance.scopeBounds[0] = "execution-environments>=1";
    assertInvalid(() => validateConstructiveIntelligenceTree(scopeDrift));

    const weakCoverage = copyTree();
    node(
      weakCoverage,
      "quest-pqc-cross-library-conformance@1",
    ).acceptance.coverageTargets[0].minimumImplementationRoots = 1;
    assertInvalid(() => validateConstructiveIntelligenceTree(weakCoverage));

    for (const [field, weakValue] of [
      ["minimumEffectiveClusters", 2],
      ["minimumOrganizationRoots", 1],
      ["minimumExecutionEnvironments", 1],
      ["minimumCases", 0],
      ["requiresCheckerOrCorpusDigest", false],
    ]) {
      const weakTarget = copyTree();
      node(
        weakTarget,
        "quest-pqc-cross-library-conformance@1",
      ).acceptance.coverageTargets[0][field] = weakValue;
      assertInvalid(() => validateConstructiveIntelligenceTree(weakTarget));
    }

    const unknownReceipt = copyTree();
    node(
      unknownReceipt,
      "quest-tls-rfc9846-keyshare-reuse@1",
    ).acceptance.adoptionReceiptTypes[0] = "social-media-likes";
    node(
      unknownReceipt,
      "quest-tls-rfc9846-keyshare-reuse@1",
    ).acceptance.adoptionReceiptTypes.sort();
    assertInvalid(() => validateConstructiveIntelligenceTree(unknownReceipt));

    const draftQuest = copyTree();
    const draftNode = node(
      draftQuest,
      "quest-tls-rfc9846-keyshare-reuse@1",
    );
    const standard = draftNode.standards[0];
    standard.normalizedMaturity = "draft";
    standard.authorityStatus = "Internet-Draft revision 1";
    const draftPath =
      `$.nodes[${draftQuest.nodes.indexOf(draftNode)}]` +
      ".standards[0].normalizedMaturity";
    assertInvalid(
      () => validateConstructiveIntelligenceTree(draftQuest),
      draftPath,
    );
  });

  it("rejects unsafe or mutable standards sources and stale metadata", () => {
    for (const specification of [
      "http://www.rfc-editor.org/rfc/rfc9846.html",
      "https://example.com/rfc9846",
      "https://user:pass@www.rfc-editor.org/rfc/rfc9846.html",
      "https://www.rfc-editor.org/rfc/rfc9846.html?draft=1",
      "https://www.rfc-editor.org/rfc/rfc9846.html#section-4",
    ]) {
      const tree = copyTree();
      node(tree, "protocol-tls13@rfc9846").standards[0].specification =
        specification;
      assertInvalid(() => validateConstructiveIntelligenceTree(tree));
    }

    const mutable = copyTree();
    node(
      mutable,
      "protocol-software-supply-chain@2026q3",
    ).standards[0].specification =
      "https://github.com/in-toto/attestation/tree/main/spec";
    assertInvalid(() => validateConstructiveIntelligenceTree(mutable));

    const movableTag = copyTree();
    node(
      movableTag,
      "protocol-software-supply-chain@2026q3",
    ).standards[0].specification =
      "https://github.com/in-toto/attestation/tree/v1.2.0";
    assertInvalid(() => validateConstructiveIntelligenceTree(movableTag));

    const wrongAuthority = copyTree();
    node(
      wrongAuthority,
      "protocol-tls13@rfc9846",
    ).standards[0].specification =
      "https://github.com/sigstore/protobuf-specs/blob/v0.5.1/protos/sigstore_bundle.proto";
    assertInvalid(() => validateConstructiveIntelligenceTree(wrongAuthority));

    const contradictoryMaturity = copyTree();
    const contradictoryStandard = node(
      contradictoryMaturity,
      "protocol-tls13@rfc9846",
    ).standards[0];
    contradictoryStandard.normalizedMaturity = "final";
    contradictoryStandard.authorityStatus = "Internet-Draft revision 99";
    assertInvalid(() =>
      validateConstructiveIntelligenceTree(contradictoryMaturity),
    );

    const stale = copyTree();
    node(stale, "protocol-tls13@rfc9846").standards[0].reviewAfter =
      "2026-07-28";
    assertInvalid(() => validateConstructiveIntelligenceTree(stale));

    const ancient = copyTree();
    node(ancient, "protocol-tls13@rfc9846").standards[0].statusCheckedAt =
      "1900-01-01";
    assertInvalid(() => validateConstructiveIntelligenceTree(ancient));

    const permanent = copyTree();
    node(permanent, "protocol-tls13@rfc9846").standards[0].reviewAfter =
      "9999-12-31";
    assertInvalid(() => validateConstructiveIntelligenceTree(permanent));
  });

  it("requires one exact record for each canonical standard ID", () => {
    const conflicting = copyTree();
    node(
      conflicting,
      "quest-tls-rfc9846-keyshare-reuse@1",
    ).standards[0].authorityStatus =
      "Proposed Standard; historical baseline";
    assertInvalid(() => validateConstructiveIntelligenceTree(conflicting), "$.nodes");

    const wrongRfc = copyTree();
    for (const candidate of wrongRfc.nodes) {
      for (const standard of candidate.standards) {
        if (standard.canonicalId === "ietf:rfc:9846") {
          standard.specification =
            "https://www.rfc-editor.org/rfc/rfc9420.html";
        }
      }
    }
    assertInvalid(() => validateConstructiveIntelligenceTree(wrongRfc));

    const wrongFips = copyTree();
    for (const candidate of wrongFips.nodes) {
      for (const standard of candidate.standards) {
        if (standard.canonicalId === "nist:fips:203:2024") {
          standard.specification = "https://doi.org/10.6028/NIST.FIPS.204";
        }
      }
    }
    assertInvalid(() => validateConstructiveIntelligenceTree(wrongFips));
  });

  it("keeps quest target evidence aligned with attainment evidence", () => {
    const mismatched = copyTree();
    node(
      mismatched,
      "quest-tls-rfc9846-keyshare-reuse@1",
    ).acceptance.targetEvidence = "E6";
    assertInvalid(() => validateConstructiveIntelligenceTree(mismatched));
  });

  it("rejects escaping, missing, duplicate, or unsorted repository references", () => {
    const escaping = copyTree();
    node(escaping, "crypto-aead@1").repositoryReferences[0] = "../outside.md";
    assertInvalid(() => validateConstructiveIntelligenceTree(escaping));

    const missing = copyTree();
    node(missing, "crypto-aead@1").repositoryReferences[0] =
      "docs/definitely-missing.md";
    assertInvalid(() => validateConstructiveIntelligenceTree(missing));

    const duplicate = copyTree();
    const refs = node(
      duplicate,
      "quest-tls-rfc9846-keyshare-reuse@1",
    ).repositoryReferences;
    refs.push(refs[0]);
    refs.sort();
    assertInvalid(() => validateConstructiveIntelligenceTree(duplicate));

    const unsorted = copyTree();
    node(
      unsorted,
      "quest-tls-rfc9846-keyshare-reuse@1",
    ).repositoryReferences.reverse();
    assertInvalid(() => validateConstructiveIntelligenceTree(unsorted));
  });

  it("requires declared roots to match the graph and every node to be reachable", () => {
    const missingRoot = copyTree();
    missingRoot.roots = ["systems-exact-bytes-state-machines@1"];
    assertInvalid(
      () => validateConstructiveIntelligenceTree(missingRoot),
      "$.roots",
    );

    const extraRoot = copyTree();
    extraRoot.roots.push("systems-exact-bytes-state-machines@1");
    extraRoot.roots.sort();
    assertInvalid(() => validateConstructiveIntelligenceTree(extraRoot), "$.roots");
  });

  it("pins schema v1 to policy 1.0.0 and the three reviewed quests", () => {
    const futurePolicy = copyTree();
    futurePolicy.policyVersion = "99.0.0";
    assertInvalid(
      () => validateConstructiveIntelligenceTree(futurePolicy),
      "$.policyVersion",
    );

    const missingQuest = copyTree();
    missingQuest.nodes = missingQuest.nodes.filter(
      (candidate) => candidate.id !== "quest-mls-state-invariants@1",
    );
    assertInvalid(() => validateConstructiveIntelligenceTree(missingQuest), "$.nodes");

    const changedTemplate = copyTree();
    node(
      changedTemplate,
      "quest-tls-rfc9846-keyshare-reuse@1",
    ).artifactRequirements[0] = "Publish an unrelated narrative.";
    assertInvalid(
      () => validateConstructiveIntelligenceTree(changedTemplate),
      "$.nodes",
    );
  });

  it("pins every normative policy and capability node under v1", () => {
    const weakerCapability = copyTree();
    node(weakerCapability, "crypto-aead@1").attainmentEvidence = "E0";
    assertInvalid(() => validateConstructiveIntelligenceTree(weakerCapability), "$");

    const trivialRequirement = copyTree();
    node(trivialRequirement, "crypto-aead@1").artifactRequirements = [
      "Say that you understand AEAD.",
    ];
    assertInvalid(() => validateConstructiveIntelligenceTree(trivialRequirement), "$");

    const weakerPrerequisites = copyTree();
    node(weakerPrerequisites, "crypto-aead@1").prerequisites = [
      "math-proofcraft@1",
    ];
    assertInvalid(
      () => validateConstructiveIntelligenceTree(weakerPrerequisites),
      "$",
    );

    const policyDrift = copyTree();
    policyDrift.policy.independence.minimumOrganizationRoots = 3;
    for (const quest of policyDrift.nodes.filter(
      (candidate) => candidate.stage === "quest",
    )) {
      quest.acceptance.minimumOrganizationRoots = 3;
      for (const target of quest.acceptance.coverageTargets) {
        target.minimumOrganizationRoots = 3;
      }
    }
    assertInvalid(() => validateConstructiveIntelligenceTree(policyDrift));
  });

  it("separates normative template binding from reviewed status snapshots", () => {
    const refreshedStatus = copyTree();
    for (const candidate of refreshedStatus.nodes) {
      for (const standard of candidate.standards) {
        if (standard.canonicalId === "ietf:rfc:9846") {
          standard.authorityStatus =
            "Proposed Standard; reviewed status refresh";
        }
      }
    }
    assert.equal(
      validateConstructiveIntelligenceTree(refreshedStatus).nodeCount,
      30,
    );
  });

  it("fails active use closed for future or expired authority snapshots", () => {
    assert.equal(
      validateConstructiveIntelligenceTreeForUse(
        copyTree(),
        "2026-08-05",
      ).checkedForUseAt,
      "2026-08-05",
    );
    assertInvalid(
      () =>
        validateConstructiveIntelligenceTreeForUse(
          copyTree(),
          "2026-08-06",
        ),
    );

    const futureSnapshot = copyTree();
    futureSnapshot.snapshotDate = "2026-08-01";
    for (const candidate of futureSnapshot.nodes) {
      for (const standard of candidate.standards) {
        standard.statusCheckedAt = "2026-08-01";
      }
    }
    assertInvalid(
      () =>
        validateConstructiveIntelligenceTreeForUse(
          futureSnapshot,
          "2026-07-29",
        ),
      "$.snapshotDate",
    );
  });
});

import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import * as frontierCommonsApi from "./validate-frontier-commons.mjs";
import {
  FRONTIER_COMMONS_MAX_BYTES,
  FrontierCommonsValidationError,
  parseAndValidateFrontierCommons,
} from "./validate-frontier-commons.mjs";

const canonicalRaw = readFileSync(
  new URL(
    "../public/standards/frontier-commons-participation.v0.json",
    import.meta.url,
  ),
  "utf8",
);
const canonical = JSON.parse(canonicalRaw);
const landscapeRaw = readFileSync(
  new URL(
    "../../docs/research/frontier-lab-landscape-2026-08-01.v0.json",
    import.meta.url,
  ),
  "utf8",
);
const landscape = JSON.parse(landscapeRaw);
const CANONICAL_SHA256 =
  "c642e09f46dcaf0a1960f140996969688b936a806c2d33ebf0b6c3efa6a70d2a";
const LANDSCAPE_SHA256 =
  "f545f1cf542b42c5a806cc03edbc54fa6c525a672bddebcfd4bf4d9060e9d995";
const EXPECTED_NON_MONEY_COSTS = [
  "time",
  "compute",
  "legal-review",
  "security-review",
  "opportunity-cost",
];
const EXPECTED_CORPORATE_GATE_IDS = [
  "accessibility-labor-worker-classification-and-whistleblower-review",
  "code-of-conduct-enforcement-appeal-and-anti-retaliation",
  "competition-and-confidentiality-review",
  "contribution-ip-patent-publication-and-license-terms",
  "counterparty-scope-and-signatory-authority",
  "explicit-accountable-human-outreach-decision",
  "governing-terms-jurisdiction-and-dispute-process",
  "independent-governance-capture-custody-and-remedy-review",
  "independent-receipt-parser-threat-model-and-material-binding-review",
  "liability-indemnity-insurance-warranty-and-remedy",
  "logo-name-affiliation-and-endorsement-policy",
  "fc-0-1-independent-roundtrip-complete",
  "maintainer-change-control-versioning-and-deprecation",
  "outreach-non-targeting-contact-source-one-contact-no-response-stop-and-retention-policy",
  "privacy-data-map-dpa-retention-erasure-and-public-permanence",
  "procurement-tax-accounting-sanctions-export-and-financial-promotion",
  "security-coordinated-disclosure-safe-harbor-incident-and-embargo",
  "service-level-support-availability-portability-and-exit",
];

function copyStandard() {
  return structuredClone(canonical);
}

function validateFrontierCommons(ordinaryClone) {
  assert.equal(Object.getPrototypeOf(ordinaryClone), Object.prototype);
  return parseAndValidateFrontierCommons(
    `${JSON.stringify(ordinaryClone, null, 2)}\n`,
  );
}

function assertInvalid(operation, path) {
  assert.throws(
    operation,
    (error) =>
      error instanceof FrontierCommonsValidationError &&
      (path === undefined || error.path === path),
  );
}

describe("Frontier Commons FC-0 standard", () => {
  it("exposes only raw-string validation and never evaluates object getters", () => {
    assert.equal("validateFrontierCommons" in frontierCommonsApi, false);

    let getterReads = 0;
    const getterObject = {};
    Object.defineProperty(getterObject, "schema", {
      enumerable: true,
      get() {
        getterReads += 1;
        return canonical.schema;
      },
    });
    assertInvalid(() => parseAndValidateFrontierCommons(getterObject), "$");
    assert.equal(getterReads, 0);

    const nonEnumerableObject = {};
    Object.defineProperty(nonEnumerableObject, "authorizesOutreach", {
      enumerable: false,
      value: true,
    });
    const symbolObject = { [Symbol("participants")]: ["claimed-party"] };
    const customObject = Object.create({ inheritedParticipant: "claimed-party" });
    customObject.schema = canonical.schema;

    for (const candidate of [
      nonEnumerableObject,
      symbolObject,
      customObject,
      new String(canonicalRaw),
    ]) {
      assertInvalid(() => parseAndValidateFrontierCommons(candidate), "$");
    }
  });

  it("validates the exact read-only invitation and reviewed digest", () => {
    assert.deepEqual(parseAndValidateFrontierCommons(canonicalRaw), {
      schema: "zerone.frontier-commons-participation/v0",
      status: "DRAFT_READ_ONLY_INVITATION",
      milestone: "FC-0",
      modeCount: 6,
      reasoningCount: 8,
      constituencyCount: 16,
      objectionCount: 11,
      completionGateCount: 4,
      openGateCount: 9,
      corporateGateCount: 18,
    });
    assert.equal(
      createHash("sha256").update(canonicalRaw).digest("hex"),
      CANONICAL_SHA256,
    );
  });

  it("keeps every membership, economic, authority, research, and chain effect false", () => {
    for (const key of [
      "authoritative",
      "networkObserved",
      "membershipBearing",
      "economicBearing",
      "governanceBearing",
    ]) {
      const standard = copyStandard();
      standard[key] = true;
      assertInvalid(() => validateFrontierCommons(standard), `$.${key}`);
    }
    for (const key of Object.keys(canonical.releaseBoundary)) {
      const standard = copyStandard();
      standard.releaseBoundary[key] = true;
      assertInvalid(
        () => validateFrontierCommons(standard),
        `$.releaseBoundary.${key}`,
      );
    }
  });

  it("keeps static publication distinct from participation, affiliation, and outreach", () => {
    assert.equal(canonical.participationFacts.scope, "PUBLIC_STATIC_SOURCE_ONLY");
    assert.equal(canonical.participationFacts.publicStaticSourceAvailability, true);
    assert.deepEqual(canonical.participationFacts.actualParticipants, []);
    assert.deepEqual(canonical.participationFacts.signatories, []);

    const wrongScope = copyStandard();
    wrongScope.participationFacts.scope = "PUBLIC_PARTICIPATION_SERVICE";
    assertInvalid(
      () => validateFrontierCommons(wrongScope),
      "$.participationFacts.scope",
    );

    const hiddenSource = copyStandard();
    hiddenSource.participationFacts.publicStaticSourceAvailability = false;
    assertInvalid(
      () => validateFrontierCommons(hiddenSource),
      "$.participationFacts.publicStaticSourceAvailability",
    );

    for (const key of ["actualParticipants", "signatories"]) {
      const claimedParty = copyStandard();
      claimedParty.participationFacts[key].push("claimed-party");
      assertInvalid(
        () => validateFrontierCommons(claimedParty),
        `$.participationFacts.${key}`,
      );
    }

    for (const key of [
      "createsAffiliation",
      "authorizesLogoUse",
      "authorizesTargetedOutreach",
      "authorizesDirectOrCorporateOutreach",
      "operatesLiveParticipationService",
      "writesNetworkState",
    ]) {
      assert.equal(canonical.participationFacts[key], false);
      const openedEffect = copyStandard();
      openedEffect.participationFacts[key] = true;
      assertInvalid(
        () => validateFrontierCommons(openedEffect),
        `$.participationFacts.${key}`,
      );
    }

    const unknownFact = copyStandard();
    unknownFact.participationFacts.contactUrl = "https://example.invalid";
    assertInvalid(
      () => validateFrontierCommons(unknownFact),
      "$.participationFacts.contactUrl",
    );
  });

  it("pins no protocol consideration while disclosing every non-money cost", () => {
    assert.equal(canonical.costBoundary.protocolConsideration, "NONE");
    assert.equal(canonical.costBoundary.claimsCostlessParticipation, false);
    assert.deepEqual(
      canonical.costBoundary.disclosedNonMoneyCosts,
      EXPECTED_NON_MONEY_COSTS,
    );

    const consideration = copyStandard();
    consideration.costBoundary.protocolConsideration = "REWARD";
    assertInvalid(
      () => validateFrontierCommons(consideration),
      "$.costBoundary.protocolConsideration",
    );

    const costlessClaim = copyStandard();
    costlessClaim.costBoundary.claimsCostlessParticipation = true;
    assertInvalid(
      () => validateFrontierCommons(costlessClaim),
      "$.costBoundary.claimsCostlessParticipation",
    );

    for (const [index] of EXPECTED_NON_MONEY_COSTS.entries()) {
      const substituted = copyStandard();
      substituted.costBoundary.disclosedNonMoneyCosts[index] = `substituted-${index}`;
      assertInvalid(
        () => validateFrontierCommons(substituted),
        `$.costBoundary.disclosedNonMoneyCosts[${index}]`,
      );
    }

    const omittedCost = copyStandard();
    omittedCost.costBoundary.disclosedNonMoneyCosts.pop();
    assertInvalid(
      () => validateFrontierCommons(omittedCost),
      "$.costBoundary.disclosedNonMoneyCosts",
    );

    const unknownCostField = copyStandard();
    unknownCostField.costBoundary.maximumCost = 0;
    assertInvalid(
      () => validateFrontierCommons(unknownCostField),
      "$.costBoundary.maximumCost",
    );
  });

  it("makes voluntary refusal, pause, and exit non-weakenable", () => {
    for (const key of [
      "voluntary",
      "rightToDecline",
      "rightToPause",
      "rightToExit",
    ]) {
      const standard = copyStandard();
      standard.rights[key] = false;
      assertInvalid(() => validateFrontierCommons(standard), `$.rights.${key}`);
    }
    for (const key of Object.keys(canonical.rights).filter(
      (key) => !["voluntary", "rightToDecline", "rightToPause", "rightToExit"].includes(key),
    )) {
      const standard = copyStandard();
      standard.rights[key] = true;
      assertInvalid(() => validateFrontierCommons(standard), `$.rights.${key}`);
    }
  });

  it("refuses completion, adoption, and constitutional privilege claims", () => {
    for (const key of ["completionClaimed", "adoptionClaimed"]) {
      const standard = copyStandard();
      standard.milestone[key] = true;
      assertInvalid(
        () => validateFrontierCommons(standard),
        `$.milestone.${key}`,
      );
    }
    for (const key of [
      "moneyGrantsVoice",
      "recognitionGrantsAuthority",
      "karmaIsScalar",
      "nonParticipationCanBePenalized",
      "founderOrSponsorPrivilege",
    ]) {
      const standard = copyStandard();
      standard.constitutionalBindings[key] = true;
      assertInvalid(
        () => validateFrontierCommons(standard),
        `$.constitutionalBindings.${key}`,
      );
    }
  });

  it("digest-binds the four constitutional artifacts", () => {
    for (const [index, key] of [
      [0, "sha256"],
      [1, "schema"],
      [2, "path"],
      [3, "id"],
    ]) {
      const standard = copyStandard();
      standard.constitutionalBindings.artifacts[index][key] =
        key === "sha256" ? "0".repeat(64) : "substituted";
      assertInvalid(
        () => validateFrontierCommons(standard),
        `$.constitutionalBindings.artifacts[${index}].${key}`,
      );
    }
  });

  it("keeps observation distinct from contribution, sponsorship, and governance", () => {
    assert.deepEqual(
      canonical.participationModes.map(({ id, state }) => ({ id, state })),
      [
        { id: "observe", state: "AVAILABLE_NOW" },
        { id: "verify-and-fork", state: "AVAILABLE_NOW" },
        {
          id: "public-source-contribution",
          state: "REQUIRES_SEPARATE_DUE_DILIGENCE",
        },
        {
          id: "reproduce-or-challenge",
          state: "STANDARD_ONLY_NO_INTAKE",
        },
        { id: "sponsor", state: "INACTIVE" },
        { id: "govern", state: "INACTIVE" },
      ],
    );

    const activeSponsor = copyStandard();
    activeSponsor.participationModes[4].state = "AVAILABLE_NOW";
    assertInvalid(
      () => validateFrontierCommons(activeSponsor),
      "$.participationModes[4].state",
    );

    const reordered = copyStandard();
    [reordered.participationModes[0], reordered.participationModes[1]] = [
      reordered.participationModes[1],
      reordered.participationModes[0],
    ];
    assertInvalid(
      () => validateFrontierCommons(reordered),
      "$.participationModes[0].id",
    );
  });

  it("pins the philosophical ladder and non-ranking constituency lenses", () => {
    const missingReason = copyStandard();
    missingReason.reasoningLadder.pop();
    assertInvalid(
      () => validateFrontierCommons(missingReason),
      "$.reasoningLadder",
    );

    const roleRights = copyStandard();
    roleRights.rights.rightsDependOnRole = true;
    assertInvalid(
      () => validateFrontierCommons(roleRights),
      "$.rights.rightsDependOnRole",
    );

    const namedLab = copyStandard();
    namedLab.constituencies[1].lens = "OpenAI board";
    assertInvalid(() => validateFrontierCommons(namedLab), "$",
    );

    const ranked = copyStandard();
    ranked.milestone.exclusions[3] = "person-score";
    assertInvalid(
      () => validateFrontierCommons(ranked),
      "$.milestone.exclusions[3]",
    );

    assert.equal(canonical.constituencies.length, 16);
    assert.equal(
      canonical.constituencies[15].id,
      "unlisted-affected-being-or-role",
    );

    const missingFallback = copyStandard();
    missingFallback.constituencies.pop();
    assertInvalid(
      () => validateFrontierCommons(missingFallback),
      "$.constituencies",
    );

    const renamedFallback = copyStandard();
    renamedFallback.constituencies[15].id = "other-role";
    assertInvalid(
      () => validateFrontierCommons(renamedFallback),
      "$.constituencies[15].id",
    );
  });

  it("preserves every objection and keeps all successor gates closed", () => {
    const erased = copyStandard();
    erased.objectionRegister.pop();
    assertInvalid(() => validateFrontierCommons(erased), "$.objectionRegister");

    const falselyResolved = copyStandard();
    falselyResolved.objectionRegister[3].state = "RESOLVED_FOR_READ_ONLY";
    assertInvalid(
      () => validateFrontierCommons(falselyResolved),
      "$.objectionRegister[3].state",
    );

    for (const [collection, basePath] of [
      [canonical.completionGates, "$.completionGates"],
      [canonical.nextMilestoneGates, "$.nextMilestoneGates"],
    ]) {
      for (const [index] of collection.entries()) {
        const standard = copyStandard();
        const target =
          basePath === "$.completionGates"
            ? standard.completionGates
            : standard.nextMilestoneGates;
        target[index].passed = true;
        assertInvalid(
          () => validateFrontierCommons(standard),
          `${basePath}[${index}].passed`,
        );
      }
    }
  });

  it("keeps Corporate M1 not ready behind all 18 ordered gates", () => {
    assert.deepEqual(Object.keys(canonical.corporateReadiness), [
      "milestone",
      "status",
      "authorizesExternalCorporateInvitation",
      "authorizesInstitutionalParticipationLane",
      "protectionsOperationallyEnforced",
      "requiredGates",
    ]);
    assert.equal(canonical.corporateReadiness.milestone, "M1");
    assert.equal(canonical.corporateReadiness.status, "NOT_READY");
    assert.deepEqual(
      canonical.corporateReadiness.requiredGates,
      EXPECTED_CORPORATE_GATE_IDS,
    );

    const wrongMilestone = copyStandard();
    wrongMilestone.corporateReadiness.milestone = "M2";
    assertInvalid(
      () => validateFrontierCommons(wrongMilestone),
      "$.corporateReadiness.milestone",
    );

    const prematureReadiness = copyStandard();
    prematureReadiness.corporateReadiness.status = "READY";
    assertInvalid(
      () => validateFrontierCommons(prematureReadiness),
      "$.corporateReadiness.status",
    );

    for (const key of [
      "authorizesExternalCorporateInvitation",
      "authorizesInstitutionalParticipationLane",
      "protectionsOperationallyEnforced",
    ]) {
      assert.equal(canonical.corporateReadiness[key], false);
      const openedBoundary = copyStandard();
      openedBoundary.corporateReadiness[key] = true;
      assertInvalid(
        () => validateFrontierCommons(openedBoundary),
        `$.corporateReadiness.${key}`,
      );
    }

    for (const [index] of EXPECTED_CORPORATE_GATE_IDS.entries()) {
      const substituted = copyStandard();
      substituted.corporateReadiness.requiredGates[index] = `substituted-gate-${index}`;
      assertInvalid(
        () => validateFrontierCommons(substituted),
        `$.corporateReadiness.requiredGates[${index}]`,
      );
    }

    const reordered = copyStandard();
    [
      reordered.corporateReadiness.requiredGates[0],
      reordered.corporateReadiness.requiredGates[1],
    ] = [
      reordered.corporateReadiness.requiredGates[1],
      reordered.corporateReadiness.requiredGates[0],
    ];
    assertInvalid(
      () => validateFrontierCommons(reordered),
      "$.corporateReadiness.requiredGates[0]",
    );

    const omittedGate = copyStandard();
    omittedGate.corporateReadiness.requiredGates.pop();
    assertInvalid(
      () => validateFrontierCommons(omittedGate),
      "$.corporateReadiness.requiredGates",
    );

    const unknownReadinessField = copyStandard();
    unknownReadinessField.corporateReadiness.partner = "claimed";
    assertInvalid(
      () => validateFrontierCommons(unknownReadinessField),
      "$.corporateReadiness.partner",
    );
  });

  it("rejects unknown fields, duplicate keys, malformed JSON, and oversized input", () => {
    const unknown = copyStandard();
    unknown.joinUrl = "https://example.invalid/join";
    assertInvalid(() => validateFrontierCommons(unknown), "$.joinUrl");

    const nested = copyStandard();
    nested.milestone.reward = 1;
    assertInvalid(
      () => validateFrontierCommons(nested),
      "$.milestone.reward",
    );

    const duplicate = canonicalRaw.replace(
      '"schema": "zerone.frontier-commons-participation/v0",',
      '"schema": "zerone.frontier-commons-participation/v0",\n  "schema": "zerone.frontier-commons-participation/v0",',
    );
    assertInvalid(() => parseAndValidateFrontierCommons(duplicate), "$.schema");
    assertInvalid(() => parseAndValidateFrontierCommons("{"), "$");
    assertInvalid(
      () => parseAndValidateFrontierCommons(" ".repeat(FRONTIER_COMMONS_MAX_BYTES + 1)),
      "$",
    );
  });

  it("rejects coercive recruitment language and named-company rosters", () => {
    const coercive = copyStandard();
    coercive.purpose = "Every lab must join now.";
    assertInvalid(() => validateFrontierCommons(coercive), "$",
    );

    const roster = copyStandard();
    roster.constituencies[1].reason = "Google should participate.";
    assertInvalid(() => validateFrontierCommons(roster), "$",
    );

    for (const mutate of [
      (standard) => {
        standard.purpose = "Every lab has no choice and must participate in Zerone.";
      },
      (standard) => {
        standard.purpose = "DeepSeek is a participating frontier lab.";
      },
      (standard) => {
        standard.purpose =
          "Every visitor receives a 10 ZRN reward and KARMA governance power.";
      },
      (standard) => {
        standard.participationModes[0].exit =
          "Exit forfeits access and creates negative reputation.";
      },
    ]) {
      const standard = copyStandard();
      mutate(standard);
      assertInvalid(() => validateFrontierCommons(standard), "$",
      );
    }
  });

  it("keeps the lab landscape representative, dated, non-endorsing, and explicit about missing content pins", () => {
    assert.equal(
      createHash("sha256").update(landscapeRaw).digest("hex"),
      LANDSCAPE_SHA256,
    );
    assert.deepEqual(Object.keys(landscape), [
      "schema",
      "normative",
      "endorsementClaimed",
      "participationClaimed",
      "snapshotDate",
      "retrievedAt",
      "reviewAfter",
      "selectionBoundary",
      "contentPinning",
      "sources",
    ]);
    assert.equal(landscape.schema, "zerone.frontier-lab-landscape/v0");
    assert.equal(landscape.normative, false);
    assert.equal(landscape.endorsementClaimed, false);
    assert.equal(landscape.participationClaimed, false);
    assert.equal(landscape.snapshotDate, "2026-08-01");
    assert.equal(landscape.retrievedAt, "2026-08-01");
    assert.equal(landscape.reviewAfter, "2026-11-01");
    assert.equal(landscape.selectionBoundary.representative, true);
    assert.equal(landscape.selectionBoundary.exhaustive, false);
    assert.equal(landscape.contentPinning.externalContentCaptured, false);
    assert.equal(landscape.contentPinning.contentSha256Available, false);
    assert.equal(landscape.sources.length, 21);

    const ids = landscape.sources.map(({ id }) => id);
    assert.equal(new Set(ids).size, ids.length);
    assert.deepEqual(ids, [...ids].sort());
    for (const [index, source] of landscape.sources.entries()) {
      assert.deepEqual(Object.keys(source), [
        "id",
        "organization",
        "title",
        "url",
        "publishedOrEffective",
        "pageStateAtRetrieval",
        "retrievedAt",
        "reviewAfter",
        "supports",
        "contentSha256",
      ]);
      assert.match(source.id, /^[a-z0-9]+(?:-[a-z0-9]+)*$/);
      assert.ok(source.organization.length > 0);
      assert.ok(source.title.length > 0);
      assert.equal(new URL(source.url).protocol, "https:");
      if (source.publishedOrEffective !== null) {
        assert.match(source.publishedOrEffective, /^\d{4}-\d{2}(?:-\d{2})?$/);
      }
      assert.match(source.pageStateAtRetrieval, /^[A-Z0-9]+(?:[_.-][A-Z0-9]+)*$/);
      assert.equal(source.retrievedAt, "2026-08-01");
      assert.equal(source.reviewAfter, "2026-11-01");
      assert.ok(source.supports.length > 0, `missing supports at source ${index}`);
      assert.equal(source.contentSha256, null);
    }
  });
});

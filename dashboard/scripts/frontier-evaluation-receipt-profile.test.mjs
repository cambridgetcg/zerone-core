import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import {
  EXPECTED_COMPACT_SHA256,
  EXPECTED_DOGFOOD_RECEIPT_SHA256,
  FrontierCompactValidationError,
  parseAndValidateFrontierCompact,
  parseAndValidateFrontierReceipt,
  validateCanonicalFrontierBundle,
  validateFrontierCompact,
  validateFrontierReceipt,
} from "./validate-frontier-evaluation-receipt-profile.mjs";

const compactRaw = readFileSync(
  new URL(
    "../public/standards/frontier-evaluation-receipt-profile.v0.json",
    import.meta.url,
  ),
  "utf8",
);
const receiptRaw = readFileSync(
  new URL(
    "../../docs/examples/frontier-evaluation-receipt/one-bounded-inconclusive-receipt.v0.json",
    import.meta.url,
  ),
  "utf8",
);
const canonicalCompact = JSON.parse(compactRaw);
const canonicalReceipt = JSON.parse(receiptRaw);
const evaluationDigestKeys = [
  "protocolDigest",
  "threatModelDigest",
  "fixtureDigest",
  "acceptancePolicyDigest",
  "environmentDigest",
  "evidenceDigest",
  "challengePolicyDigest",
];

function compactCopy() {
  return structuredClone(canonicalCompact);
}

function receiptCopy() {
  return structuredClone(canonicalReceipt);
}

function publicEvaluationMaterials() {
  return Object.fromEntries(
    evaluationDigestKeys.map((key) => [key, Buffer.from(`${key}: bounded public fixture v0`)]),
  );
}

function publicReceipt(materials = publicEvaluationMaterials()) {
  const receipt = receiptCopy();
  receipt.subject[0].name = "public-evaluation.example.json";
  receipt.subject[0].digest.sha256 = "a".repeat(64);
  receipt.predicate.receiptKind = "PUBLIC_EVALUATION";
  receipt.predicate.issuer.claimedId = "example-project-role";
  receipt.predicate.issuer.authorityScope = "ARTIFACT_ONLY";
  receipt.predicate.issuer.identityDisclosure = "PROJECT_ALIAS";
  receipt.predicate.issuer.claimedControlRoot = "example-control-root";
  for (const key of evaluationDigestKeys) {
    receipt.predicate.evaluation[key] = `sha256:${createHash("sha256")
      .update(materials[key])
      .digest("hex")}`;
  }
  receipt.predicate.evaluation.result = "DIVERGED";
  receipt.predicate.evaluation.reasonCodes = ["OUTSIDE_PRECOMMITTED_TOLERANCE"];
  receipt.predicate.evaluation.limitationCodes = ["PUBLIC_FIXTURE_ONLY"];
  receipt.predicate.evaluation.relations = [
    {
      type: "DIVERGES_FROM",
      receiptDigest: `sha256:${"b".repeat(64)}`,
    },
  ];
  return receipt;
}

function publicOptions(materials = publicEvaluationMaterials(), asOfOn = "2026-08-02") {
  return { publicEvaluationMaterials: materials, asOfOn };
}

function assertInvalid(operation, path) {
  assert.throws(
    operation,
    (error) =>
      error instanceof FrontierCompactValidationError &&
      (path === undefined || error.path === path),
  );
}

describe("frontier evaluation receipt shadow FL-0", () => {
  it("validates the exact compact and its deliberately inconclusive dogfood receipt", () => {
    const result = validateCanonicalFrontierBundle(compactRaw, receiptRaw);
    assert.equal(result.compact.digest, EXPECTED_COMPACT_SHA256);
    assert.equal(result.receipt.digest, EXPECTED_DOGFOOD_RECEIPT_SHA256);
    assert.equal(result.compact.rightCount, 24);
    assert.equal(result.compact.actorCount, 21);
    assert.equal(result.compact.objectionCount, 16);
    assert.equal(result.receipt.claimedResult, "INCONCLUSIVE");
  });

  it("keeps every release, outreach, endorsement, and logo effect false", () => {
    for (const key of Object.keys(canonicalCompact.releaseBoundary)) {
      const compact = compactCopy();
      compact.releaseBoundary[key] = true;
      assertInvalid(
        () => validateFrontierCompact(compact),
        `$.releaseBoundary.${key}`,
      );
    }
  });

  it("refuses fabricated participants and signatories", () => {
    const participant = compactCopy();
    participant.actualParticipants.push("example-lab");
    assertInvalid(
      () => validateFrontierCompact(participant),
      "$.actualParticipants",
    );

    const signatory = compactCopy();
    signatory.signatories.push("example-signer");
    assertInvalid(() => validateFrontierCompact(signatory), "$.signatories");
  });

  it("refuses removed, reordered, or falsely enforced rights", () => {
    const removed = compactCopy();
    removed.rights.pop();
    assertInvalid(() => validateFrontierCompact(removed), "$.rights");

    const reordered = compactCopy();
    [reordered.rights[0], reordered.rights[1]] = [
      reordered.rights[1],
      reordered.rights[0],
    ];
    assertInvalid(() => validateFrontierCompact(reordered), "$.rights[0].id");

    const enforcement = compactCopy();
    enforcement.rights[0].operationalStatus = "ENFORCED";
    assertInvalid(
      () => validateFrontierCompact(enforcement),
      "$.rights[0].operationalStatus",
    );

    const overclaim = compactCopy();
    overclaim.rights[0].sourceStatus = "VALIDATED_DRAFT";
    assertInvalid(
      () => validateFrontierCompact(overclaim),
      "$.rights[0].sourceStatus",
    );
  });

  it("pins reviewed promises, source-ID bindings, and reference-ID bindings semantically", () => {
    const coercive = compactCopy();
    coercive.rights[0].promise = "Participation may be coerced.";
    assertInvalid(() => validateFrontierCompact(coercive), "$");

    const reboundSource = compactCopy();
    reboundSource.sourceBindings[0].path = reboundSource.sourceBindings[1].path;
    reboundSource.sourceBindings[0].sha256 = reboundSource.sourceBindings[1].sha256;
    assertInvalid(() => validateFrontierCompact(reboundSource), "$");

    const reboundReference = compactCopy();
    reboundReference.externalReferences.find(({ id }) => id === "nist-ai-rmf").uri =
      "https://www.nist.gov/";
    assertInvalid(() => validateFrontierCompact(reboundReference), "$");
  });

  it("names bounded duty holders and treats actor value as untested hypotheses", () => {
    const thirdPartyPromise = compactCopy();
    thirdPartyPromise.rightsBoundary.thirdPartyConductGuaranteed = true;
    assertInvalid(
      () => validateFrontierCompact(thirdPartyPromise),
      "$.rightsBoundary.thirdPartyConductGuaranteed",
    );

    const hiddenDuty = compactCopy();
    hiddenDuty.rightsBoundary.dutyHolders.pop();
    assertInvalid(
      () => validateFrontierCompact(hiddenDuty),
      "$.rightsBoundary.dutyHolders",
    );

    const provenValue = compactCopy();
    provenValue.actorClaimBoundary.evidenceStatus = "PROVEN";
    assertInvalid(
      () => validateFrontierCompact(provenValue),
      "$.actorClaimBoundary.evidenceStatus",
    );
  });

  it("is subordinate to the exact FC-0 invitation and creates no second invitation", () => {
    const relationship = canonicalCompact.relationshipToFc0;
    assert.equal(relationship.invitationSurfaceOfRecord, "FC-0");
    assert.equal(relationship.role, "SUBORDINATE_INTERNAL_RECEIPT_SHADOW");
    assert.equal(
      canonicalCompact.sourceBindings[0].id,
      "frontier-commons-participation-v0",
    );

    for (const key of [
      "replacesOrAmendsFc0",
      "extendsInvitationBeyondFc0",
      "satisfiesFc0CompletionGates",
      "authorizesOutreach",
    ]) {
      const profile = compactCopy();
      profile.relationshipToFc0[key] = true;
      assertInvalid(
        () => validateFrontierCompact(profile),
        `$.relationshipToFc0.${key}`,
      );
    }

    const rebound = compactCopy();
    rebound.relationshipToFc0.fc0SourceBindingId = "adapter-index-v1";
    assertInvalid(
      () => validateFrontierCompact(rebound),
      "$.relationshipToFc0.fc0SourceBindingId",
    );
  });

  it("keeps every lane optional, identity-minimizing, tokenless, and correctly gated", () => {
    for (const key of [
      "requiresAccount",
      "requiresWallet",
      "requiresTokenOrStake",
      "requiresPublicIdentity",
      "requiresConfidentialDisclosure",
      "confersParticipantStatus",
    ]) {
      const compact = compactCopy();
      compact.participationLanes[0][key] = true;
      assertInvalid(
        () => validateFrontierCompact(compact),
        `$.participationLanes[0].${key}`,
      );
    }

    const status = compactCopy();
    status.participationLanes.find(({ id }) => id === "contribute").status =
      "SOURCE_AVAILABLE";
    assertInvalid(
      () => validateFrontierCompact(status),
      "$.participationLanes[4].status",
    );

    const minimum = compactCopy();
    minimum.participationLanes[2].minimumContribution = "ONE_RECEIPT";
    assertInvalid(
      () => validateFrontierCompact(minimum),
      "$.participationLanes[2].minimumContribution",
    );
  });

  it("preserves every actor scope, honest reason to decline, and minimum answer", () => {
    const removed = compactCopy();
    removed.actorScopes.pop();
    assertInvalid(() => validateFrontierCompact(removed), "$.actorScopes");

    const noDecline = compactCopy();
    noDecline.actorScopes[14].legitimateDeclineReasons = [];
    assertInvalid(
      () => validateFrontierCompact(noDecline),
      "$.actorScopes[14].legitimateDeclineReasons",
    );

    const unknownLevel = compactCopy();
    unknownLevel.actorScopes[17].level = "LEGAL_PERSON";
    assertInvalid(
      () => validateFrontierCompact(unknownLevel),
      "$.actorScopes[17].level",
    );
  });

  it("does not permit honest blockers or residual objections to disappear", () => {
    const limit = compactCopy();
    limit.honestLimits.shift();
    assertInvalid(() => validateFrontierCompact(limit), "$.honestLimits");

    const severity = compactCopy();
    severity.honestLimits[2].severity = "RESOLVED";
    assertInvalid(
      () => validateFrontierCompact(severity),
      "$.honestLimits[2].severity",
    );

    const objection = compactCopy();
    objection.objections.pop();
    assertInvalid(() => validateFrontierCompact(objection), "$.objections");

    const erasedDecline = compactCopy();
    erasedDecline.objections[0].residualReasonToDecline = "";
    assertInvalid(
      () => validateFrontierCompact(erasedDecline),
      "$.objections[0].residualReasonToDecline",
    );
  });

  it("keeps the pilot internal, zero-consideration, no-cost-claim-free, and promotion-gated", () => {
    const external = compactCopy();
    external.pilot.externalParticipationTarget = true;
    assertInvalid(
      () => validateFrontierCompact(external),
      "$.pilot.externalParticipationTarget",
    );

    for (const key of Object.keys(canonicalCompact.pilot.economics)) {
      const compact = compactCopy();
      compact.pilot.economics[key] = true;
      assertInvalid(
        () => validateFrontierCompact(compact),
        `$.pilot.economics.${key}`,
      );
    }

    const gate = compactCopy();
    gate.pilot.promotionGates.pop();
    assertInvalid(
      () => validateFrontierCompact(gate),
      "$.pilot.promotionGates",
    );
  });

  it("keeps Corporate M1 explicitly not ready and unable to authorize invitations", () => {
    const ready = compactCopy();
    ready.corporateReadiness.status = "READY";
    assertInvalid(
      () => validateFrontierCompact(ready),
      "$.corporateReadiness.status",
    );

    const invitation = compactCopy();
    invitation.corporateReadiness.authorizesExternalCorporateInvitation = true;
    assertInvalid(
      () => validateFrontierCompact(invitation),
      "$.corporateReadiness.authorizesExternalCorporateInvitation",
    );

    const missingGate = compactCopy();
    missingGate.corporateReadiness.requiredGates.pop();
    assertInvalid(
      () => validateFrontierCompact(missingGate),
      "$.corporateReadiness.requiredGates",
    );
  });

  it("binds exact existing source artifacts and refuses path traversal", () => {
    const drift = compactCopy();
    drift.sourceBindings[0].sha256 = "0".repeat(64);
    assertInvalid(
      () => validateFrontierCompact(drift),
      "$.sourceBindings[0].sha256",
    );

    const traversal = compactCopy();
    traversal.sourceBindings[0].path = "../outside.json";
    assertInvalid(
      () => validateFrontierCompact(traversal),
      "$.sourceBindings[0].path",
    );
  });

  it("rejects unknown and reordered schema fields", () => {
    const unknown = compactCopy();
    unknown.memberLabs = [];
    assertInvalid(() => validateFrontierCompact(unknown), "$");

    const reordered = Object.fromEntries(
      Object.entries(compactCopy()).reverse(),
    );
    assertInvalid(() => validateFrontierCompact(reordered), "$");
  });

  it("bounds and strictly parses raw profile JSON", () => {
    assertInvalid(() => parseAndValidateFrontierCompact("{"), "$");
    assertInvalid(
      () =>
        parseAndValidateFrontierCompact(
          compactRaw.replace(
            '"schema": "zerone.frontier-evaluation-receipt-profile/v0",',
            '"schema": "zerone.frontier-evaluation-receipt-profile/v0", "schema": "duplicate",',
          ),
        ),
      "$.schema",
    );
    assertInvalid(
      () => parseAndValidateFrontierCompact(" ".repeat(65_537)),
      "$",
    );
    assertInvalid(
      () => parseAndValidateFrontierCompact(`${"[".repeat(33)}0${"]".repeat(33)}`),
      "$",
    );
  });

  it("pins canonical compact and receipt bytes", () => {
    assertInvalid(() => parseAndValidateFrontierCompact(`${compactRaw}\n`), "$");
    assertInvalid(
      () => parseAndValidateFrontierCompact(`${compactRaw}\n`, { pinDigest: true }),
      "$",
    );
    assertInvalid(
      () =>
        parseAndValidateFrontierReceipt(
          `${receiptRaw}\n`,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          { pinDigest: true },
        ),
      "$receipt",
    );
  });
});

describe("One Bounded Inconclusive Receipt v0", () => {
  it("parses a bounded public divergence only as current, unauthenticated metadata", () => {
    const materials = publicEvaluationMaterials();
    const receipt = publicReceipt(materials);
    assert.deepEqual(
      validateFrontierReceipt(
        receipt,
        canonicalCompact,
        EXPECTED_COMPACT_SHA256,
        publicOptions(materials),
      ),
      {
        schema: "zerone.frontier-evaluation-receipt/v0",
        structuralStatus: "PARSED_UNVERIFIED",
        claimedResult: "DIVERGED",
        relationCount: 1,
        effectiveRelationCount: 0,
        relationEffect: "NONE",
        freshness: "CURRENT_UNVERIFIED",
        freshnessEvaluatedOn: "2026-08-02",
        subjectDigestVerified: false,
        evaluationMaterialDigestsVerified: true,
        materialSemanticsVerified: false,
        claimSemanticsVerified: false,
        authenticated: false,
        authorizedToRepresentOrganization: false,
        privacyClassificationVerified: false,
        eligibleForAutomaticReliance: false,
      },
    );
  });

  it("snapshots hostile accessor and Proxy inputs exactly once before validation", () => {
    const materials = publicEvaluationMaterials();
    const accessorReceipt = publicReceipt(materials);
    let resultReads = 0;
    Object.defineProperty(accessorReceipt.predicate.evaluation, "result", {
      enumerable: true,
      configurable: true,
      get() {
        resultReads += 1;
        return resultReads === 1 ? "DIVERGED" : "CERTIFIED";
      },
    });

    const accessorResult = validateFrontierReceipt(
      accessorReceipt,
      canonicalCompact,
      EXPECTED_COMPACT_SHA256,
      publicOptions(materials),
    );
    assert.equal(accessorResult.claimedResult, "DIVERGED");
    assert.equal(resultReads, 1);

    const compact = compactCopy();
    const canonicalPilot = compact.pilot;
    let pilotReads = 0;
    const proxyCompact = new Proxy(compact, {
      get(target, property, receiver) {
        if (property !== "pilot") return Reflect.get(target, property, receiver);
        pilotReads += 1;
        return pilotReads === 1
          ? canonicalPilot
          : { ...canonicalPilot, resultVocabulary: [...canonicalPilot.resultVocabulary, "CERTIFIED"] };
      },
    });

    const proxyResult = validateFrontierReceipt(
      publicReceipt(materials),
      proxyCompact,
      EXPECTED_COMPACT_SHA256,
      publicOptions(materials),
    );
    assert.equal(proxyResult.claimedResult, "DIVERGED");
    assert.equal(pilotReads, 1);
  });

  it("revalidates the canonical compact and rejects attacker-chosen vocabularies or digests", () => {
    const materials = publicEvaluationMaterials();
    const hostileCompact = compactCopy();
    hostileCompact.pilot.resultVocabulary[0] = "CERTIFIED";
    const certified = publicReceipt(materials);
    certified.predicate.evaluation.result = "CERTIFIED";
    assertInvalid(
      () =>
        validateFrontierReceipt(
          certified,
          hostileCompact,
          EXPECTED_COMPACT_SHA256,
          publicOptions(materials),
        ),
      "$.pilot.resultVocabulary",
    );

    assertInvalid(
      () =>
        validateFrontierReceipt(
          receiptCopy(),
          canonicalCompact,
          "a".repeat(64),
        ),
      "$compactDigest",
    );
  });

  it("rejects unsigned organization identity or authorization claims", () => {
    const materials = publicEvaluationMaterials();

    const scope = publicReceipt(materials);
    scope.predicate.issuer.authorityScope = "ORGANIZATION";
    assertInvalid(
      () =>
        validateFrontierReceipt(
          scope,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          publicOptions(materials),
        ),
      "$receipt.predicate.issuer.authorityScope",
    );

    const identity = publicReceipt(materials);
    identity.predicate.issuer.identityDisclosure = "ORGANIZATION";
    assertInvalid(
      () =>
        validateFrontierReceipt(
          identity,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          publicOptions(materials),
        ),
      "$receipt.predicate.issuer.identityDisclosure",
    );

    const authorization = publicReceipt(materials);
    authorization.predicate.issuer.authorizedToRepresentOrganization = true;
    assertInvalid(
      () =>
        validateFrontierReceipt(
          authorization,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          publicOptions(materials),
        ),
      "$receipt.predicate.issuer.authorizedToRepresentOrganization",
    );

    const assent = publicReceipt(materials);
    assent.predicate.issuer.systemAssentClaimed = true;
    assertInvalid(
      () =>
        validateFrontierReceipt(
          assent,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          publicOptions(materials),
        ),
      "$receipt.predicate.issuer.systemAssentClaimed",
    );
  });

  it("requires explicit currentness policy and rejects future or expired receipts", () => {
    const materials = publicEvaluationMaterials();
    const receipt = publicReceipt(materials);

    assertInvalid(
      () =>
        validateFrontierReceipt(
          receipt,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          { publicEvaluationMaterials: materials },
        ),
      "$policy.asOfOn",
    );
    assertInvalid(
      () =>
        validateFrontierReceipt(
          receipt,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          publicOptions(materials, "2026-07-31"),
        ),
      "$policy.asOfOn",
    );
    assertInvalid(
      () =>
        validateFrontierReceipt(
          receipt,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          publicOptions(materials, "2026-11-02"),
        ),
      "$policy.asOfOn",
    );

    const due = validateFrontierReceipt(
      receipt,
      canonicalCompact,
      EXPECTED_COMPACT_SHA256,
      publicOptions(materials, "2026-08-15"),
    );
    assert.equal(due.freshness, "REVIEW_DUE_UNVERIFIED");
    assert.equal(due.eligibleForAutomaticReliance, false);
  });

  it("reserves refusal records for a future authenticated profile", () => {
    const refusalKind = receiptCopy();
    refusalKind.predicate.receiptKind = "SELF_REFUSAL";
    assertInvalid(
      () =>
        validateFrontierReceipt(
          refusalKind,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
        ),
      "$receipt.predicate.receiptKind",
    );

    const refusalResult = publicReceipt();
    refusalResult.predicate.evaluation.result = "REFUSED";
    refusalResult.predicate.evaluation.relations = [];
    assertInvalid(
      () =>
        validateFrontierReceipt(
          refusalResult,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          publicOptions(),
        ),
      "$receipt.predicate.evaluation.result",
    );
  });

  it("uses code-only receipt text and rejects mutable predicate semantics", () => {
    const materials = publicEvaluationMaterials();
    const freeText = publicReceipt(materials);
    freeText.predicate.evaluation.limitationCodes = ["secret customer name: alice"];
    assertInvalid(
      () =>
        validateFrontierReceipt(
          freeText,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          publicOptions(materials),
        ),
      "$receipt.predicate.evaluation.limitationCodes[0]",
    );

    const schemaDrift = publicReceipt(materials);
    schemaDrift.predicate.schemaDigest = `sha256:${"0".repeat(64)}`;
    assertInvalid(
      () =>
        validateFrontierReceipt(
          schemaDrift,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          publicOptions(materials),
        ),
      "$receipt.predicate.schemaDigest",
    );
  });

  it("never treats code-shaped third-party allegations as semantically verified", () => {
    const materials = publicEvaluationMaterials();
    const allegation = publicReceipt(materials);
    allegation.subject[0].name = "third-party-refusal-claim.example.json";
    allegation.predicate.evaluation.reasonCodes = ["OTHER_PARTY_REFUSED"];
    allegation.predicate.evaluation.limitationCodes = ["THIRD_PARTY_SILENT"];

    const parsed = validateFrontierReceipt(
      allegation,
      canonicalCompact,
      EXPECTED_COMPACT_SHA256,
      publicOptions(materials),
    );
    assert.equal(parsed.claimSemanticsVerified, false);
    assert.equal(parsed.eligibleForAutomaticReliance, false);
  });

  it("makes all seven internal dogfood material digests explicitly absent", () => {
    for (const key of evaluationDigestKeys) {
      assert.equal(canonicalReceipt.predicate.evaluation[key], null);
      const receipt = receiptCopy();
      receipt.predicate.evaluation[key] = `sha256:${"a".repeat(64)}`;
      assertInvalid(
        () =>
          validateFrontierReceipt(
            receipt,
            canonicalCompact,
            EXPECTED_COMPACT_SHA256,
          ),
        `$receipt.predicate.evaluation.${key}`,
      );
    }
  });

  it("requires every public digest to be recomputed from supplied bounded bytes", () => {
    const materials = publicEvaluationMaterials();
    const receipt = publicReceipt(materials);

    assertInvalid(
      () =>
        validateFrontierReceipt(
          receipt,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          { asOfOn: "2026-08-02" },
        ),
      "$materials",
    );

    const altered = {
      ...materials,
      protocolDigest: Buffer.from("altered protocol bytes"),
    };
    assertInvalid(
      () =>
        validateFrontierReceipt(
          receipt,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          publicOptions(altered),
        ),
      "$receipt.predicate.evaluation.protocolDigest",
    );

    const missing = { ...materials };
    delete missing.protocolDigest;
    assertInvalid(
      () =>
        validateFrontierReceipt(
          receipt,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          publicOptions(missing),
        ),
      "$materials",
    );

    const empty = { ...materials, protocolDigest: Buffer.alloc(0) };
    assertInvalid(
      () =>
        validateFrontierReceipt(
          receipt,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          publicOptions(empty),
        ),
      "$materials.protocolDigest",
    );
  });

  it("refuses null, placeholder, and contradiction-shaped public material claims", () => {
    const materials = publicEvaluationMaterials();

    const absent = publicReceipt(materials);
    absent.predicate.evaluation.protocolDigest = null;
    assertInvalid(
      () =>
        validateFrontierReceipt(
          absent,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          publicOptions(materials),
        ),
      "$receipt.predicate.evaluation.protocolDigest",
    );

    const placeholder = publicReceipt(materials);
    placeholder.predicate.evaluation.protocolDigest = `sha256:${"0".repeat(64)}`;
    assertInvalid(
      () =>
        validateFrontierReceipt(
          placeholder,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          publicOptions(materials),
        ),
      "$receipt.predicate.evaluation.protocolDigest",
    );

    const contradiction = publicReceipt(materials);
    contradiction.predicate.evaluation.reasonCodes = ["NO_BOUND_EVALUATION_MATERIALS"];
    assertInvalid(
      () =>
        validateFrontierReceipt(
          contradiction,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          publicOptions(materials),
        ),
      "$receipt.predicate.evaluation.reasonCodes",
    );
  });

  it("refuses a dogfood subject that does not bind the exact compact", () => {
    const receipt = receiptCopy();
    receipt.subject[0].digest.sha256 = "0".repeat(64);
    assertInvalid(
      () =>
        validateFrontierReceipt(
          receipt,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
        ),
      "$receipt.subject[0].digest.sha256",
    );
  });

  it("does not allow the unreviewed dogfood result to become favourable", () => {
    const receipt = receiptCopy();
    receipt.predicate.evaluation.result = "REPRODUCED";
    assertInvalid(
      () =>
        validateFrontierReceipt(
          receipt,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
        ),
      "$receipt.predicate.evaluation.relations",
    );
  });

  it("refuses privacy, authority, and scoring effects", () => {
    const confidential = receiptCopy();
    confidential.predicate.privacy.declaresContainsConfidentialData = true;
    assertInvalid(
      () =>
        validateFrontierReceipt(
          confidential,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
        ),
      "$receipt.predicate.privacy.declaresContainsConfidentialData",
    );

    for (const key of Object.keys(canonicalReceipt.predicate.effects)) {
      const receipt = receiptCopy();
      receipt.predicate.effects[key] = true;
      assertInvalid(
        () =>
          validateFrontierReceipt(
            receipt,
            canonicalCompact,
            EXPECTED_COMPACT_SHA256,
          ),
        `$receipt.predicate.effects.${key}`,
      );
    }
  });

  it("keeps corrections assertion-only, automatic effects off, and exit unrelated", () => {
    const erasure = receiptCopy();
    erasure.predicate.correction.historicalPublicCopiesDeletable = true;
    assertInvalid(
      () =>
        validateFrontierReceipt(
          erasure,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
        ),
      "$receipt.predicate.correction.historicalPublicCopiesDeletable",
    );

    const automatic = receiptCopy();
    automatic.predicate.correction.automaticEffect = true;
    assertInvalid(
      () =>
        validateFrontierReceipt(
          automatic,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
        ),
      "$receipt.predicate.correction.automaticEffect",
    );

    const unsignedAuthority = receiptCopy();
    unsignedAuthority.predicate.correction.authenticatedAuthorizationRequiredForFutureEffect = false;
    assertInvalid(
      () =>
        validateFrontierReceipt(
          unsignedAuthority,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
        ),
      "$receipt.predicate.correction.authenticatedAuthorizationRequiredForFutureEffect",
    );

    const access = receiptCopy();
    access.predicate.correction.exitAffectsUnrelatedAccess = true;
    assertInvalid(
      () =>
        validateFrontierReceipt(
          access,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
        ),
      "$receipt.predicate.correction.exitAffectsUnrelatedAccess",
    );
  });

  it("enforces result-relation compatibility and gives relations zero correction authority", () => {
    const materials = publicEvaluationMaterials();

    const contradictory = publicReceipt(materials);
    contradictory.predicate.evaluation.relations.push({
      type: "REPLICATES",
      receiptDigest: `sha256:${"c".repeat(64)}`,
    });
    assertInvalid(
      () =>
        validateFrontierReceipt(
          contradictory,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          publicOptions(materials),
        ),
      "$receipt.predicate.evaluation.relations",
    );

    const recordedDivergence = publicReceipt(materials);
    recordedDivergence.predicate.evaluation.result = "RECORDED";
    assertInvalid(
      () =>
        validateFrontierReceipt(
          recordedDivergence,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          publicOptions(materials),
        ),
      "$receipt.predicate.evaluation.relations",
    );

    const supersession = publicReceipt(materials);
    supersession.predicate.evaluation.relations = [
      { type: "SUPERSEDES", receiptDigest: `sha256:${"d".repeat(64)}` },
    ];
    assertInvalid(
      () =>
        validateFrontierReceipt(
          supersession,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          publicOptions(materials),
        ),
      "$receipt.predicate.evaluation.relations[0].type",
    );
  });

  it("requires coherent dates, bounded reasons, and known relations", () => {
    const dates = receiptCopy();
    dates.predicate.evaluation.evidenceCutoffOn = "2026-08-02";
    assertInvalid(
      () =>
        validateFrontierReceipt(
          dates,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
        ),
      "$receipt.predicate.evaluation.evidenceCutoffOn",
    );

    const reasons = receiptCopy();
    reasons.predicate.evaluation.reasonCodes = [];
    assertInvalid(
      () =>
        validateFrontierReceipt(
          reasons,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
        ),
      "$receipt.predicate.evaluation.reasonCodes",
    );

    const materials = publicEvaluationMaterials();
    const relation = publicReceipt(materials);
    relation.predicate.evaluation.relations = [
      { type: "ENDORSES", receiptDigest: `sha256:${"a".repeat(64)}` },
    ];
    assertInvalid(
      () =>
        validateFrontierReceipt(
          relation,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
          publicOptions(materials),
        ),
      "$receipt.predicate.evaluation.relations[0].type",
    );
  });

  it("rejects receipt duplicate keys, oversized documents, and unknown fields", () => {
    assertInvalid(
      () =>
        parseAndValidateFrontierReceipt(
          receiptRaw.replace(
            '"receiptKind": "ZERONE_SELF_DOGFOOD",',
            '"receiptKind": "ZERONE_SELF_DOGFOOD", "receiptKind": "PUBLIC_EVALUATION",',
          ),
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
        ),
      "$.predicate.receiptKind",
    );
    assertInvalid(
      () =>
        parseAndValidateFrontierReceipt(
          " ".repeat(32_769),
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
        ),
      "$receipt",
    );

    const unknown = receiptCopy();
    unknown.predicate.member = true;
    assertInvalid(
      () =>
        validateFrontierReceipt(
          unknown,
          canonicalCompact,
          EXPECTED_COMPACT_SHA256,
        ),
      "$receipt.predicate",
    );
  });
});

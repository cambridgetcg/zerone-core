import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import {
  FRONTIER_COMMONS_MAX_BYTES,
  FrontierCommonsValidationError,
  parseAndValidateFrontierCommons,
  validateFrontierCommons,
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
  "322faf0eec5ab22a040fe69545b23bece990e626bca77baf96b101f8f7325862";
const LANDSCAPE_SHA256 =
  "f545f1cf542b42c5a806cc03edbc54fa6c525a672bddebcfd4bf4d9060e9d995";

function copyStandard() {
  return structuredClone(canonical);
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
  it("validates the exact read-only invitation and reviewed digest", () => {
    assert.deepEqual(parseAndValidateFrontierCommons(canonicalRaw), {
      schema: "zerone.frontier-commons-participation/v0",
      status: "DRAFT_READ_ONLY_INVITATION",
      milestone: "FC-0",
      modeCount: 6,
      reasoningCount: 8,
      constituencyCount: 15,
      objectionCount: 11,
      completionGateCount: 4,
      openGateCount: 9,
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

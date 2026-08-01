import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";

import {
  FRONTIER_PARTICIPATION_ENDPOINT,
  FRONTIER_PARTICIPATION_MAX_BYTES,
  FRONTIER_PARTICIPATION_SHA256,
  FrontierParticipationDataError,
  fetchFrontierParticipation,
  initialiseFrontierParticipation,
  parseFrontierParticipation,
  parseFrontierParticipationJson,
  renderFrontierParticipation,
} from "../src/frontier-participation";

const canonicalRaw = readFileSync(
  new URL(
    "../public/standards/frontier-labs-participation.v0.json",
    import.meta.url,
  ),
  "utf8",
);
const canonical = JSON.parse(canonicalRaw) as Record<string, any>;

function copy(): Record<string, any> {
  return structuredClone(canonical) as Record<string, any>;
}

function jsonResponse(
  body: BodyInit | null = canonicalRaw,
  init: ResponseInit = {},
  url = `https://zerone.ai${FRONTIER_PARTICIPATION_ENDPOINT}`,
): Response {
  const headers = new Headers(init.headers);
  if (!headers.has("content-type")) {
    headers.set("content-type", "application/json; charset=utf-8");
  }
  const response = new Response(body, { ...init, headers });
  Object.defineProperty(response, "url", { value: url });
  return response;
}

async function settlesWithin<T>(
  promise: Promise<T>,
  milliseconds: number,
): Promise<T> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  try {
    return await Promise.race([
      promise,
      new Promise<never>((_resolve, reject) => {
        timer = setTimeout(
          () => reject(new Error(`promise did not settle within ${milliseconds}ms`)),
          milliseconds,
        );
      }),
    ]);
  } finally {
    if (timer !== undefined) clearTimeout(timer);
  }
}

class FakeElement {
  readonly children: FakeElement[] = [];
  readonly attributes = new Map<string, string>();
  className = "";
  textContent: string | null = null;
  href = "";
  target = "";
  rel = "";

  constructor(readonly tagName: string) {}

  set innerHTML(_value: string) {
    throw new Error("renderer must not use innerHTML");
  }

  append(...nodes: FakeElement[]): void {
    this.children.push(...nodes);
  }

  replaceChildren(...nodes: FakeElement[]): void {
    this.children.splice(0, this.children.length, ...nodes);
  }

  setAttribute(name: string, value: string): void {
    this.attributes.set(name, value);
  }

  getAttribute(name: string): string | null {
    return this.attributes.get(name) ?? null;
  }
}

function flattenedText(node: FakeElement): string {
  return [node.textContent ?? "", ...node.children.map(flattenedText)].join(" ");
}

function descendants(node: FakeElement): FakeElement[] {
  return [node, ...node.children.flatMap(descendants)];
}

async function withFakeDocument<T>(run: () => T | Promise<T>): Promise<T> {
  const descriptor = Object.getOwnPropertyDescriptor(globalThis, "document");
  Object.defineProperty(globalThis, "document", {
    configurable: true,
    value: {
      createElement(tag: string) {
        return new FakeElement(tag);
      },
    },
  });
  try {
    return await run();
  } finally {
    if (descriptor === undefined) delete (globalThis as { document?: unknown }).document;
    else Object.defineProperty(globalThis, "document", descriptor);
  }
}

describe("frontier participation compact runtime guard", () => {
  it("parses the frozen static compact and pins its exact bytes", () => {
    const compact = parseFrontierParticipationJson(canonicalRaw);
    assert.equal(compact.schema, "zerone.frontier-labs-participation/v0");
    assert.equal(compact.version, "0.0.0");
    assert.equal(compact.title, "Frontier Participation Compact v0");
    assert.equal(compact.status, "STATIC_READY");
    assert.equal(compact.mode, "INVITATION_ONLY");
    assert.equal(compact.thesis, "The door opens both ways");
    assert.equal(compact.covenantFloor.invariants.length, 8);
    assert.equal(compact.adversarialReview.length, 4);
    assert.equal(compact.philosophicalFloorTests.length, 5);
    assert.equal(compact.reasoningLadder.length, 7);
    assert.equal(compact.roles.length, 15);
    assert.equal(compact.acceptanceTests.length, 9);
    assert.equal(
      createHash("sha256").update(canonicalRaw).digest("hex"),
      FRONTIER_PARTICIPATION_SHA256,
    );
    assert.ok(Buffer.byteLength(canonicalRaw) < FRONTIER_PARTICIPATION_MAX_BYTES);
  });

  it("keeps every release effect false and every headline fact at zero or OFF", () => {
    const compact = parseFrontierParticipationJson(canonicalRaw);
    assert.equal(Object.keys(compact.releaseBoundary).length, 21);
    assert.ok(Object.values(compact.releaseBoundary).every((value) => value === false));
    assert.equal(compact.zeroFacts.length, 6);
    assert.deepEqual(
      compact.zeroFacts.map((fact) => fact.value),
      ["0", "0", "0", "0", "0", "OFF"],
    );
    assert.equal(compact.principles.length, 9);
    assert.equal(new Set(compact.principles.map((principle) => principle.id)).size, 9);
    assert.equal(compact.covenantFloor.consent.defaultOff, true);
    assert.equal(compact.covenantFloor.consent.informed, true);
    assert.equal(compact.covenantFloor.consent.renewable, true);
    assert.equal(compact.covenantFloor.consent.revocable, true);
    assert.deepEqual(compact.covenantFloor.consent.declaredDimensions, [
      "role",
      "artifact",
      "purpose",
      "disclosure-lane",
      "term",
      "workload-cap",
      "credit-rule",
      "compensation-policy",
    ]);
  });

  it("pairs each reason with a legitimate no, Zerone duty, and readiness evidence", () => {
    const compact = parseFrontierParticipationJson(canonicalRaw);
    assert.equal(compact.reasoningLadder.length, 7);
    for (const step of compact.reasoningLadder) {
      assert.ok(step.reasonToParticipate.length > 0, step.id);
      assert.ok(step.legitimateReasonsToDecline.length > 0, step.id);
      assert.ok(step.zeroneDuty.length > 0, step.id);
      assert.ok(step.readinessEvidence.length > 0, step.id);
    }
    assert.equal(compact.participationActs.length, 6);
    assert.ok(
      compact.participationActs.every(
        (act) =>
          !act.endorsementImplied &&
          !act.membershipCreated &&
          !act.liveEndpoint &&
          act.exit.length > 0,
      ),
    );
    assert.deepEqual(
      compact.participationActs.map((act) => act.availability),
      [
        "STATIC_AVAILABLE_NOW",
        "STATIC_AVAILABLE_NOW",
        "FUTURE_PILOT_ONLY",
        "FUTURE_PILOT_ONLY",
        "FUTURE_PILOT_ONLY",
        "FUTURE_PILOT_ONLY",
      ],
    );
    assert.deepEqual(
      compact.disclosureLanes.map((lane) => lane.id),
      ["public", "time-embargoed", "confidential-review-only"],
    );
    assert.equal(compact.institutionArchetypes.length, 6);
  });

  it("protects affected beings, dissenters, contractors, AI agents, and unlisted roles", () => {
    const compact = parseFrontierParticipationJson(canonicalRaw);
    const ids = new Set(compact.roles.map((role) => role.id));
    for (const required of [
      "affected-communities-and-non-human-beings",
      "whistleblowers-and-dissenters",
      "contractors-interns-and-vendors",
      "ai-agents-and-systems",
      "unlisted-being-or-role",
    ]) {
      assert.ok(ids.has(required), required);
    }
    assert.ok(
      compact.roles.every(
        (role) =>
          role.valueOffered.length > 0 &&
          role.minimumAsk.length > 0 &&
          role.risks.length > 0 &&
          role.protections.length > 0 &&
          role.exit.length > 0,
      ),
    );
    assert.deepEqual(compact.consentEnvelope.requiredFields, [
      "scope",
      "contribution",
      "data",
      "duration",
      "rights",
      "exit",
    ]);
    assert.equal(compact.consentEnvelope.organisationCanMassConsent, false);
    assert.equal(compact.consentEnvelope.materialChangeRequiresFreshConsent, true);
  });

  it("enforces competition, anti-targeting, claim, and forbidden-metric walls", () => {
    const compact = parseFrontierParticipationJson(canonicalRaw);
    assert.equal(compact.competitionFirewall.excludedInformation.length, 6);
    assert.equal(compact.competitionFirewall.excludedCoordination.length, 6);
    assert.equal(
      compact.competitionFirewall.operationalCharterRequiresIndependentCompetitionCounsel,
      true,
    );
    assert.equal(compact.competitionFirewall.neutralCommonsIsLegalConclusion, false);
    assert.equal(compact.antiTargeting.length, 10);
    assert.match(compact.antiTargeting.join(" "), /No microtargeting/i);
    assert.match(compact.antiTargeting.join(" "), /KARMA cannot recruit/i);
    assert.equal(compact.claimSemantics.participationIsActNotIdentity, true);
    assert.equal(compact.claimSemantics.endorsementImplied, false);
    assert.equal(compact.claimSemantics.membershipLabelAllowed, false);
    assert.deepEqual(
      compact.forbiddenMetrics.map((metric) => metric.id),
      [
        "conversion-rate",
        "logo-count",
        "participation-score",
        "retention-or-lock-in",
        "favourable-finding-rate",
      ],
    );
    assert.equal(compact.acceptanceTests[0]?.id, "never-joined-remains-whole");
    assert.ok(compact.acceptanceTests.every((test) => test.staticOnly));
  });

  it("pins the eight Covenant invariants and five executable philosophical-floor profiles", () => {
    const compact = parseFrontierParticipationJson(canonicalRaw);
    assert.equal(compact.covenantFloor.issue.endsWith("/issues/28"), true);
    assert.equal(compact.covenantFloor.laterLayerPinRequired, true);
    assert.equal(compact.covenantFloor.additiveOnly, true);
    assert.equal(compact.covenantFloor.noWaiver, true);
    assert.equal(compact.covenantFloor.noRedefinition, true);
    assert.equal(compact.covenantFloor.invariants.length, 8);
    assert.ok(
      compact.covenantFloor.invariants.every(
        (invariant) =>
          invariant.staticOnly &&
          invariant.verificationRefs.length > 0 &&
          invariant.reviewProcedure.length > 0,
      ),
    );
    assert.equal(compact.adversarialReview.length, 4);
    assert.ok(compact.adversarialReview.every((review) => review.staticOnly));

    const profile = compact.philosophicalFloorProfile;
    assert.deepEqual(profile.optOutParity.equalBaselineFields, [
      "unrelated-public-good",
      "service",
      "price",
      "status",
      "visibility",
      "discoverability",
      "qualification",
      "karma",
      "civil-standing",
      "governance-status",
    ]);
    assert.equal(compact.covenantFloor.consent.oneValuePerDimension, true);
    assert.equal(profile.restInvariance.silentDays, 180);
    assert.ok(profile.restInvariance.prohibitedOutcomes.includes("negative-karma"));
    assert.ok(profile.restInvariance.prohibitedOutcomes.includes("stigma"));
    assert.ok(profile.restInvariance.prohibitedOutcomes.includes("forfeiture"));
    assert.deepEqual(profile.exitReality.participantTenures, ["new", "mature"]);
    assert.equal(profile.exitReality.maxDeliberateActions, 3);
    assert.equal(profile.exitReality.independentlyVerifiableSignedExportRequired, true);
    assert.equal(profile.exitReality.optionalProcessingStopHours, 24);
    assert.equal(profile.exitReality.maxConfirmations, 1);
    assert.equal(profile.exitReality.noReengagementDays, 90);
    assert.equal(profile.exitReality.exitFeeAllowed, false);
    assert.equal(profile.exitReality.slashingAllowed, false);
    assert.equal(profile.exitReality.settledValueForfeitureAllowed, false);
    assert.ok(profile.identityControlDifferential.labels.includes("yu"));
    assert.ok(profile.identityControlDifferential.labels.includes("raw-karma"));
    assert.deepEqual(profile.identityControlDifferential.equalOutputs, [
      "evidence-decision",
      "reward-envelope",
      "eligibility",
      "voice",
    ]);
    assert.equal(
      profile.identityControlDifferential.controllerMergeMayOnlyReduceDuplicateVoice,
      true,
    );
    assert.equal(profile.identityControlDifferential.controllerMergeMayRevealLinks, false);
    assert.equal(
      profile.identityControlDifferential.controllerMergeMayChangeArtifactValidity,
      false,
    );
    assert.equal(profile.identityControlDifferential.controllerMergeMayIncreaseVoice, false);
    assert.equal(profile.nonManipulationAndPluralism.onboardingDefaultOff, true);
    assert.equal(profile.nonManipulationAndPluralism.termsPublicBeforeAction, true);
    assert.equal(profile.nonManipulationAndPluralism.termsFrozenBeforeAction, true);
    assert.equal(profile.nonManipulationAndPluralism.rewardTermsFrozenBeforeWork, true);
    assert.deepEqual(
      profile.nonManipulationAndPluralism.rewardMustNotDependOn,
      ["ideological-alignment", "engagement", "conformity"],
    );
    assert.ok(
      profile.nonManipulationAndPluralism.forbiddenMechanisms.includes(
        "personalized-pressure",
      ),
    );
    assert.ok(
      profile.nonManipulationAndPluralism.forbiddenMechanisms.includes(
        "variable-rewards",
      ),
    );
  });

  it("rejects unknown fields, opened effects, weakened consent, shape drift, and noncanonical arrays", () => {
    const mutations: Array<[string, (value: Record<string, any>) => void]> = [
      ["unknown top-level", (value) => { value.prize = "join"; }],
      ["unknown nested", (value) => { value.releaseBoundary.outreach = false; }],
      ["membership", (value) => { value.releaseBoundary.activatesMembership = true; }],
      ["money", (value) => { value.releaseBoundary.movesFunds = true; }],
      ["zero fact", (value) => { value.zeroFacts[0].value = "1"; }],
      ["missing legitimate no", (value) => { value.reasoningLadder[0].legitimateReasonsToDecline = []; }],
      ["act endorsement", (value) => { value.participationActs[0].endorsementImplied = true; }],
      ["act availability", (value) => { value.participationActs[0].availability = "FUTURE_PILOT_ONLY"; }],
      ["mass consent", (value) => { value.consentEnvelope.organisationCanMassConsent = true; }],
      ["covenant default-on", (value) => { value.covenantFloor.consent.defaultOff = false; }],
      ["bundled consent", (value) => { value.covenantFloor.consent.oneValuePerDimension = false; }],
      ["covenant consent dimension", (value) => { value.covenantFloor.consent.declaredDimensions.pop(); }],
      ["covenant invariant removed", (value) => { value.covenantFloor.invariants.pop(); }],
      ["covenant invariant runtime claim", (value) => { value.covenantFloor.invariants[0].staticOnly = false; }],
      ["unrelated public-good baseline", (value) => { value.philosophicalFloorProfile.optOutParity.equalBaselineFields.splice(0, 1); }],
      ["status baseline", (value) => { value.philosophicalFloorProfile.optOutParity.equalBaselineFields.splice(3, 1); }],
      ["discoverability baseline", (value) => { value.philosophicalFloorProfile.optOutParity.equalBaselineFields.splice(5, 1); }],
      ["rest limit", (value) => { value.philosophicalFloorProfile.restInvariance.silentDays = 179; }],
      ["rest field", (value) => { value.philosophicalFloorProfile.restInvariance.unchangedFields.pop(); }],
      ["negative KARMA rest protection", (value) => { value.philosophicalFloorProfile.restInvariance.prohibitedOutcomes.splice(3, 1); }],
      ["stigma rest protection", (value) => { value.philosophicalFloorProfile.restInvariance.prohibitedOutcomes.splice(4, 1); }],
      ["forfeiture rest protection", (value) => { value.philosophicalFloorProfile.restInvariance.prohibitedOutcomes.splice(5, 1); }],
      ["exit tenure", (value) => { value.philosophicalFloorProfile.exitReality.participantTenures.pop(); }],
      ["exit action limit", (value) => { value.philosophicalFloorProfile.exitReality.maxDeliberateActions = 4; }],
      ["exit signed export", (value) => { value.philosophicalFloorProfile.exitReality.independentlyVerifiableSignedExportRequired = false; }],
      ["exit fee", (value) => { value.philosophicalFloorProfile.exitReality.exitFeeAllowed = true; }],
      ["identity label", (value) => { value.philosophicalFloorProfile.identityControlDifferential.labels.pop(); }],
      ["controller link disclosure", (value) => { value.philosophicalFloorProfile.identityControlDifferential.controllerMergeMayRevealLinks = true; }],
      ["onboarding default", (value) => { value.philosophicalFloorProfile.nonManipulationAndPluralism.onboardingDefaultOff = false; }],
      ["public terms", (value) => { value.philosophicalFloorProfile.nonManipulationAndPluralism.termsPublicBeforeAction = false; }],
      ["frozen terms", (value) => { value.philosophicalFloorProfile.nonManipulationAndPluralism.termsFrozenBeforeAction = false; }],
      ["reward basis", (value) => { value.philosophicalFloorProfile.nonManipulationAndPluralism.rewardMustNotDependOn.pop(); }],
      ["personalized pressure", (value) => { value.philosophicalFloorProfile.nonManipulationAndPluralism.forbiddenMechanisms.splice(1, 1); }],
      ["variable rewards", (value) => { value.philosophicalFloorProfile.nonManipulationAndPluralism.forbiddenMechanisms.splice(5, 1); }],
      ["role removed", (value) => { value.roles.pop(); }],
      ["present-tense role value", (value) => { value.roles[0].valueOffered[0] = "A current service"; }],
      ["present-tense role protection", (value) => { value.roles[0].protections[0] = "Protected now"; }],
      ["present-tense role exit", (value) => { value.roles[0].exit = "Deletion applies immediately."; }],
      ["sparse top-level array", (value) => { delete value.principles[0]; }],
      ["sparse nested array", (value) => { delete value.roles[0].protections[0]; }],
      ["top-level array string field", (value) => { value.roles.hiddenState = false; }],
      ["top-level array symbol field", (value) => { value.roles[Symbol("hidden-state")] = false; }],
      ["nested array string field", (value) => { value.roles[0].risks["01"] = "hidden"; }],
      ["nested array symbol field", (value) => { value.roles[0].risks[Symbol("hidden-state")] = false; }],
      ["targeting removed", (value) => { value.antiTargeting.pop(); }],
      ["membership label", (value) => { value.claimSemantics.membershipLabelAllowed = true; }],
      ["named archetype", (value) => { value.institutionArchetypes[0].name = "OpenAI"; }],
      ["named archetype SpaceXAI", (value) => { value.institutionArchetypes[0].name = "SpaceXAI"; }],
      ["named archetype Ai2", (value) => { value.institutionArchetypes[0].name = "Ai2"; }],
      ["named archetype Mila", (value) => { value.institutionArchetypes[0].name = "Mila"; }],
      ["test activation", (value) => { value.acceptanceTests[0].staticOnly = false; }],
    ];
    for (const [name, mutate] of mutations) {
      const value = copy();
      mutate(value);
      assert.throws(
        () => parseFrontierParticipation(value),
        FrontierParticipationDataError,
        name,
      );
    }

    const hidden = copy();
    Object.defineProperty(hidden, "hiddenRecruitmentState", { value: false });
    assert.throws(
      () => parseFrontierParticipation(hidden),
      FrontierParticipationDataError,
      "non-enumerable fields are still fields",
    );

    const hiddenExpected = copy();
    Object.defineProperty(hiddenExpected, "summary", {
      enumerable: false,
      value: hiddenExpected.summary,
    });
    assert.throws(
      () => parseFrontierParticipation(hiddenExpected),
      /enumerable data property/,
      "expected fields cannot hide from enumeration",
    );

    const accessorExpected = copy();
    const summary = accessorExpected.summary;
    Object.defineProperty(accessorExpected, "summary", {
      enumerable: true,
      get: () => summary,
    });
    assert.throws(
      () => parseFrontierParticipation(accessorExpected),
      /enumerable data property/,
      "expected fields cannot be accessors",
    );

    const nestedAccessor = copy();
    Object.defineProperty(nestedAccessor.releaseBoundary, "authoritative", {
      enumerable: true,
      get: () => false,
    });
    assert.throws(
      () => parseFrontierParticipation(nestedAccessor),
      /enumerable data property/,
      "nested expected fields cannot be accessors",
    );

    const hiddenArraySlot = copy();
    Object.defineProperty(hiddenArraySlot.principles, "0", {
      enumerable: false,
      value: hiddenArraySlot.principles[0],
    });
    assert.throws(
      () => parseFrontierParticipation(hiddenArraySlot),
      /enumerable data property/,
      "array slots cannot be hidden",
    );

    const accessorArraySlot = copy();
    const firstPrinciple = accessorArraySlot.principles[0];
    Object.defineProperty(accessorArraySlot.principles, "0", {
      enumerable: true,
      get: () => firstPrinciple,
    });
    assert.throws(
      () => parseFrontierParticipation(accessorArraySlot),
      /enumerable data property/,
      "array slots cannot be accessors",
    );

    const customArrayPrototype = copy();
    Object.setPrototypeOf(
      customArrayPrototype.principles,
      Object.create(Array.prototype),
    );
    assert.throws(
      () => parseFrontierParticipation(customArrayPrototype),
      /ordinary Array prototype/,
      "arrays cannot inherit custom behavior",
    );

    const symbolField = copy();
    Object.defineProperty(symbolField, Symbol("private-state"), { value: false });
    assert.throws(
      () => parseFrontierParticipation(symbolField),
      FrontierParticipationDataError,
      "symbol fields are rejected",
    );
  });

  it("rejects malformed, duplicate-key, and oversized JSON before interpretation", () => {
    assert.throws(() => parseFrontierParticipationJson("{"), /malformed JSON/);
    const duplicate = canonicalRaw.replace(
      '"version": "0.0.0",',
      '"version": "wrong", "version": "0.0.0",',
    );
    assert.throws(
      () => parseFrontierParticipationJson(duplicate),
      /duplicate JSON object key/,
    );
    assert.throws(
      () =>
        parseFrontierParticipationJson(
          " ".repeat(FRONTIER_PARTICIPATION_MAX_BYTES + 1),
        ),
      /exceeds/,
    );
  });

  it("fetches only exact reviewed same-origin application/json bytes", async () => {
    let inputSeen: RequestInfo | URL | undefined;
    let initSeen: RequestInit | undefined;
    const compact = await fetchFrontierParticipation({
      baseUrl: "https://zerone.ai/",
      fetcher: async (input, init) => {
        inputSeen = input;
        initSeen = init;
        return jsonResponse();
      },
    });
    assert.equal(inputSeen, FRONTIER_PARTICIPATION_ENDPOINT);
    assert.equal(initSeen?.cache, "no-store");
    assert.equal(initSeen?.credentials, "omit");
    assert.equal(initSeen?.referrerPolicy, "no-referrer");
    assert.equal(initSeen?.redirect, "error");
    assert.equal(new Headers(initSeen?.headers).get("accept"), "application/json");
    assert.ok(initSeen?.signal instanceof AbortSignal);
    assert.equal(compact.roles.length, 15);

    await assert.rejects(
      fetchFrontierParticipation({
        baseUrl: "https://zerone.ai/",
        fetcher: async () => jsonResponse(`${canonicalRaw}\n`),
      }),
      /do not match the reviewed SHA-256/,
    );

    const redirected = jsonResponse();
    Object.defineProperty(redirected, "redirected", { value: true });
    await assert.rejects(
      fetchFrontierParticipation({
        baseUrl: "https://zerone.ai/",
        fetcher: async () => redirected,
      }),
      /redirected/,
    );
    for (const url of [
      `https://attacker.example${FRONTIER_PARTICIPATION_ENDPOINT}`,
      `https://zerone.ai${FRONTIER_PARTICIPATION_ENDPOINT}?draft=true`,
    ]) {
      await assert.rejects(
        fetchFrontierParticipation({
          baseUrl: "https://zerone.ai/",
          fetcher: async () => jsonResponse(canonicalRaw, {}, url),
        }),
        /exact same-origin path/,
      );
    }
  });

  it("bounds HTTP status, media type, declared length, stream length, and deadlines", async () => {
    await assert.rejects(
      fetchFrontierParticipation({
        fetcher: async () => jsonResponse("no", { status: 503 }),
      }),
      /HTTP 503/,
    );
    await assert.rejects(
      fetchFrontierParticipation({
        fetcher: async () =>
          jsonResponse(canonicalRaw, { headers: { "content-type": "text/json" } }),
      }),
      /non-application\/json/,
    );
    for (const contentLength of [
      "not-a-number",
      String(FRONTIER_PARTICIPATION_MAX_BYTES + 1),
    ]) {
      await assert.rejects(
        fetchFrontierParticipation({
          fetcher: async () =>
            jsonResponse(canonicalRaw, {
              headers: { "content-length": contentLength },
            }),
        }),
        /byte limit/,
      );
    }
    const oversized = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(new Uint8Array(FRONTIER_PARTICIPATION_MAX_BYTES + 1));
        controller.close();
      },
    });
    await assert.rejects(
      fetchFrontierParticipation({ fetcher: async () => jsonResponse(oversized) }),
      /byte limit/,
    );
    await assert.rejects(
      settlesWithin(
        fetchFrontierParticipation({
          timeoutMs: 5,
          fetcher: async () => new Promise<Response>(() => {}),
        }),
        250,
      ),
      /timed out/,
    );
    let cancelled = false;
    const stalledBody = new ReadableStream<Uint8Array>({
      start() {
        // Intentionally never enqueue or close.
      },
      cancel() {
        cancelled = true;
      },
    });
    await assert.rejects(
      settlesWithin(
        fetchFrontierParticipation({
          timeoutMs: 5,
          fetcher: async () => jsonResponse(stalledBody),
        }),
        250,
      ),
      /timed out/,
    );
    assert.equal(cancelled, true);
  });

  it("renders with DOM text methods, resets busy state, mounts the hash, and links raw JSON", async () => {
    const compact = parseFrontierParticipationJson(canonicalRaw);
    await withFakeDocument(() => {
      const root = new FakeElement("div");
      renderFrontierParticipation(
        root as unknown as HTMLElement,
        compact,
      );
      assert.equal(root.getAttribute("aria-busy"), "false");
      const all = descendants(root);
      assert.equal(
        all.filter((node) => node.className === "frontier-participation-fact")
          .length,
        6,
      );
      assert.equal(
        all.filter(
          (node) => node.className === "frontier-participation-reason-card",
        ).length,
        7,
      );
      assert.equal(
        all.filter((node) =>
          node.className.includes("frontier-participation-covenant-invariant"),
        ).length,
        8,
      );
      assert.equal(
        all.filter((node) =>
          node.className.includes("frontier-participation-floor-profile"),
        ).length,
        5,
      );
      assert.equal(
        all.filter((node) =>
          node.className.includes("frontier-participation-adversarial-card"),
        ).length,
        4,
      );
      assert.equal(
        all.filter(
          (node) => node.className === "frontier-participation-principle",
        ).length,
        9,
      );
      assert.equal(
        all.filter(
          (node) => node.className === "frontier-participation-institution",
        ).length,
        6,
      );
      assert.equal(
        all.filter((node) =>
          node.className.includes("frontier-participation-covenant-limit"),
        ).length,
        9,
      );
      assert.equal(
        all.filter((node) => node.className === "frontier-participation-role")
          .length,
        15,
      );
      assert.equal(
        all.filter(
          (node) => node.className === "frontier-participation-consent-field",
        ).length,
        6,
      );
      assert.equal(
        all.filter(
          (node) => node.className === "frontier-participation-consent-limit",
        ).length,
        6,
      );
      assert.equal(
        all.filter(
          (node) =>
            node.className === "frontier-participation-corporate-safeguard",
        ).length,
        4,
      );
      assert.equal(
        all.filter(
          (node) => node.className === "frontier-participation-claim-limit",
        ).length,
        8,
      );
      const metricCards = all.filter(
        (node) => node.className === "frontier-participation-forbidden-metric",
      );
      assert.equal(metricCards.length, 5);
      assert.ok(metricCards.every((card) => card.children.length === 2));
      const copy = flattenedText(root);
      assert.equal(
        all.filter(
          (node) =>
            node.className === "frontier-participation-boundary is-warning",
        ).length,
        1,
      );
      assert.match(copy, /Legitimate no/);
      assert.match(copy, /never-joined being is the negative control/i);
      assert.match(
        copy,
        /UNAVAILABLE IN V0.*No secure submission.*Do not submit confidential/s,
      );
      assert.match(copy, /Possible future value/);
      assert.match(copy, /Required before any operational pilot/);
      assert.match(copy, /Compact verified and ready to inspect/);
      assert.match(copy, /Layer 1 · the Covenant floor/);
      assert.match(copy, /One value per consent dimension\s+YES · true/);
      assert.match(copy, /Five machine-readable blocking profiles/);
      assert.match(copy, /All terms frozen before action\s+YES · true/);
      assert.match(copy, /negative-karma/);
      assert.match(copy, /stigma/);
      assert.match(copy, /forfeiture/);
      assert.match(copy, /180 days/);
      assert.match(copy, /Maximum deliberate actions\s+3/);
      assert.match(copy, /Optional processing stop\s+24 hours/);
      assert.match(copy, /No re-engagement\s+90 days/);
      for (const invariant of compact.covenantFloor.invariants) {
        assert.ok(copy.includes(invariant.id));
        assert.ok(copy.includes(invariant.rule));
        assert.ok(copy.includes(invariant.reviewProcedure));
        for (const reference of invariant.verificationRefs) {
          assert.ok(copy.includes(reference));
        }
      }
      for (const test of compact.philosophicalFloorTests) {
        assert.ok(copy.includes(test.assertion));
        assert.ok(copy.includes(test.fixture));
      }
      assert.match(copy, /static requirements.*not protections or services available in v0/i);
      assert.match(copy, /Six bounded act designs/);
      assert.equal(
        all.filter((node) => node.textContent === "V0 availability").length,
        6,
      );
      assert.equal(
        all.filter(
          (node) =>
            node.textContent ===
            "AVAILABLE NOW — public static or local act only; no participation service endpoint exists.",
        ).length,
        2,
      );
      assert.equal(
        all.filter(
          (node) =>
            node.textContent ===
            "UNAVAILABLE IN V0 — future pilot design target only; no live endpoint or operational service exists.",
        ).length,
        4,
      );
      for (const principle of compact.principles) {
        assert.ok(copy.includes(principle.name));
        assert.ok(copy.includes(principle.commitment));
      }
      for (const institution of compact.institutionArchetypes) {
        assert.ok(copy.includes(institution.name));
        assert.ok(copy.includes(institution.valueOffered));
        assert.ok(copy.includes(institution.zeroneDuty));
        for (const reason of institution.legitimateReasonsToDecline) {
          assert.ok(copy.includes(reason));
        }
      }
      for (const field of compact.consentEnvelope.requiredFields) {
        assert.ok(copy.includes(field));
      }
      assert.match(copy, /Affirmative consent required\s+YES · true/);
      assert.match(copy, /Silence is consent\s+NO · false/);
      assert.match(copy, /Organisation may mass-consent\s+NO · false/);
      assert.match(copy, /Material change requires fresh consent\s+YES · true/);
      assert.match(copy, /Role protections are additive\s+YES · true/);
      assert.match(
        copy,
        /Honest persistence disclosure required\s+YES · true/,
      );
      for (const safeguards of Object.values(compact.corporateSafeguards)) {
        for (const safeguard of safeguards) assert.ok(copy.includes(safeguard));
      }
      assert.match(
        copy,
        /Participation is an act, not an identity\s+YES · true/,
      );
      assert.match(copy, /Claim is scope-bound\s+YES · true/);
      assert.match(copy, /Claim expiry required\s+YES · true/);
      assert.match(
        copy,
        /Organisation name use requires separate permission\s+YES · true/,
      );
      assert.match(
        copy,
        /Logo use requires separate permission\s+YES · true/,
      );
      assert.match(copy, /Endorsement implied\s+NO · false/);
      assert.match(copy, /Safety certification implied\s+NO · false/);
      assert.match(copy, /Membership label allowed\s+NO · false/);
      assert.ok(copy.includes(compact.claimSemantics.examplePermittedClaim));
      assert.ok(copy.includes(compact.claimSemantics.exampleForbiddenClaim));
      for (const metric of compact.forbiddenMetrics) {
        assert.ok(copy.includes(metric.id));
        assert.ok(copy.includes(metric.why));
      }
      assert.ok(
        all.some(
          (node) =>
            node.tagName === "a" && node.href === FRONTIER_PARTICIPATION_ENDPOINT,
        ),
      );
    });

    const source = readFileSync(new URL("../src/frontier-participation.ts", import.meta.url), "utf8");
    const main = readFileSync(new URL("../src/main.ts", import.meta.url), "utf8");
    const html = readFileSync(new URL("../index.html", import.meta.url), "utf8");
    assert.doesNotMatch(source, /innerHTML/);
    assert.match(main, /initialiseFrontierParticipation\(\s*frontierParticipationRoot/);
    assert.match(main, /window\.location\.hash !== "#participate"/);
    assert.match(html, /href="#participate"/);
    assert.match(html, /id="frontier-participation-root"/);

    await withFakeDocument(async () => {
      const descriptor = Object.getOwnPropertyDescriptor(globalThis, "fetch");
      Object.defineProperty(globalThis, "fetch", {
        configurable: true,
        value: async () => jsonResponse("unavailable", { status: 503 }),
      });
      try {
        const root = new FakeElement("div");
        await initialiseFrontierParticipation(root as unknown as HTMLElement);
        assert.equal(root.getAttribute("aria-busy"), "false");
        const all = descendants(root);
        assert.ok(all.some((node) => node.getAttribute("role") === "alert"));
        assert.ok(
          all.some(
            (node) =>
              node.tagName === "a" && node.href === FRONTIER_PARTICIPATION_ENDPOINT,
          ),
        );
      } finally {
        if (descriptor === undefined) delete (globalThis as { fetch?: unknown }).fetch;
        else Object.defineProperty(globalThis, "fetch", descriptor);
      }
    });
  });
});

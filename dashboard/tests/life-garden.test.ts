import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import {
  EPIGENETICS_GARDEN_ENDPOINT,
  EPIGENETICS_GARDEN_SHA256,
  KARMA_FOUNDATION_ENDPOINT,
  KARMA_FOUNDATION_SHA256,
  LIFE_GARDEN_STAGES,
  LifeGardenDataError,
  STATIC_STANDARD_MAX_BYTES,
  evaluateLifeGardenSourceReview,
  fetchEpigeneticsCapabilityGarden,
  fetchKarmaFoundation,
  filterLifeGardenNodes,
  formatBasisPoints,
  parseEpigeneticsCapabilityGarden,
  parseEpigeneticsCapabilityGardenJson,
  parseKarmaFoundation,
  parseKarmaFoundationJson,
  type LifeGardenFilters,
} from "../src/life-garden";
import {
  validateGardenRaw,
  validateKarmaRaw,
} from "../scripts/validate-life-garden";

type MutableDocument = Record<string, any>;

const gardenRaw = readFileSync(
  new URL(
    "../public/standards/epigenetics-capability-garden.v1.json",
    import.meta.url,
  ),
  "utf8",
);
const karmaRaw = readFileSync(
  new URL("../public/standards/karma-foundation.v1.json", import.meta.url),
  "utf8",
);
const gardenDocument = JSON.parse(gardenRaw) as MutableDocument;
const karmaDocument = JSON.parse(karmaRaw) as MutableDocument;

function copyGarden(): MutableDocument {
  return structuredClone(gardenDocument) as MutableDocument;
}

function copyKarma(): MutableDocument {
  return structuredClone(karmaDocument) as MutableDocument;
}

function node(document: MutableDocument, id: string): MutableDocument {
  const found = document.nodes.find(
    (candidate: MutableDocument) => candidate.id === id,
  );
  assert.ok(found, `missing fixture node ${id}`);
  return found as MutableDocument;
}

function filters(
  overrides: Partial<LifeGardenFilters> = {},
): LifeGardenFilters {
  return {
    query: "",
    stage: "all",
    domain: "all",
    safetyTier: "all",
    ...overrides,
  };
}

function jsonResponse(body: string, init: ResponseInit = {}): Response {
  return new Response(body, {
    ...init,
    headers: {
      "content-type": "application/json; charset=utf-8",
      ...init.headers,
    },
  });
}

describe("epigenetics capability garden", () => {
  it("parses the reviewed 25-node, 56-edge, seven-stage garden", () => {
    const garden = parseEpigeneticsCapabilityGardenJson(gardenRaw);
    assert.equal(garden.schema, "zerone.epigenetics-capability-garden/v1");
    assert.equal(garden.nodes.length, 25);
    assert.equal(
      garden.nodes.reduce(
        (total, capability) => total + capability.prerequisites.length,
        0,
      ),
      56,
    );
    assert.deepEqual(
      [...new Set(garden.nodes.map((capability) => capability.stage))].sort(
        (left, right) =>
          LIFE_GARDEN_STAGES.indexOf(left) - LIFE_GARDEN_STAGES.indexOf(right),
      ),
      [...LIFE_GARDEN_STAGES],
    );
    assert.equal(
      garden.nodes.filter((capability) => capability.kind === "quest").length,
      3,
    );
    assert.equal(garden.rewardBearing, false);
    assert.equal(garden.policy.funding.protocolIssuance, "disabled");
    assert.equal(garden.sources.length, 11);
    assert.equal(
      node(garden as unknown as MutableDocument, "analysis-causal-graphs@1")
        .evidenceContribution,
      "E4",
    );
    assert.equal(
      node(garden as unknown as MutableDocument, "validation-prior-art-delta@1")
        .evidenceContribution,
      "cross-cutting",
    );
    assert.equal(
      garden.evidenceLadder.reduce(
        (total, step) => total + step.rewardBps,
        0,
      ) + garden.policy.funding.challengeReserveBps,
      10_000,
    );
  });

  it("filters by biological concept, stage, domain, and safety lane", () => {
    const garden = parseEpigeneticsCapabilityGardenJson(gardenRaw);
    assert.ok(
      filterLifeGardenNodes(garden, filters({ query: "methylation" })).length >= 3,
    );
    assert.equal(
      filterLifeGardenNodes(garden, filters({ stage: "quest" })).length,
      3,
    );
    assert.ok(
      filterLifeGardenNodes(garden, filters({ domain: "single-cell" })).every(
        (capability) => capability.domain === "single-cell",
      ),
    );
    assert.ok(
      filterLifeGardenNodes(
        garden,
        filters({ safetyTier: "regulated-human" }),
      ).every((capability) => capability.safetyTier === "regulated-human"),
    );
  });

  it("fails closed if authority, rewards, money, qualification, or experimentation turn on", () => {
    for (const key of ["authoritative", "networkObserved", "rewardBearing"] as const) {
      const garden = copyGarden();
      garden[key] = true;
      assert.throws(
        () => parseEpigeneticsCapabilityGarden(garden),
        LifeGardenDataError,
      );
    }
    for (const key of Object.keys(gardenDocument.releaseBoundary)) {
      const garden = copyGarden();
      garden.releaseBoundary[key] = true;
      assert.throws(
        () => parseEpigeneticsCapabilityGarden(garden),
        new RegExp(`releaseBoundary\\.${key}`),
      );
    }
  });

  it("freezes breakthrough, independence, human-data, and safety gates", () => {
    const authorSelected = copyGarden();
    authorSelected.policy.breakthroughRecognition.authorSelected = true;
    assert.throws(
      () => parseEpigeneticsCapabilityGarden(authorSelected),
      /authorSelected/,
    );

    const weakIndependence = copyGarden();
    weakIndependence.policy.independence.minimumEffectiveClusters = 2;
    assert.throws(
      () => parseEpigeneticsCapabilityGarden(weakIndependence),
      /minimumEffectiveClusters/,
    );

    const rawGenomics = copyGarden();
    rawGenomics.policy.humanData.rawIdentifiableDataOnPublicLedger = true;
    assert.throws(
      () => parseEpigeneticsCapabilityGarden(rawGenomics),
      /rawIdentifiableDataOnPublicLedger/,
    );

    const humanIntervention = copyGarden();
    humanIntervention.policy.safety.unapprovedHumanInterventionEligible = true;
    assert.throws(
      () => parseEpigeneticsCapabilityGarden(humanIntervention),
      /unapprovedHumanInterventionEligible/,
    );

    for (const key of [
      "animalWelfareReviewRequiredWhereApplicable",
      "biosafetyRiskAssessmentRequired",
    ]) {
      const missingReview = copyGarden();
      missingReview.policy.safety[key] = false;
      assert.throws(
        () => parseEpigeneticsCapabilityGarden(missingReview),
        new RegExp(key),
      );
    }
  });

  it("rejects duplicate, unresolved, cyclic, backward, and mispriced graph edges", () => {
    const duplicate = copyGarden();
    duplicate.nodes[1].id = duplicate.nodes[0].id;
    assert.throws(() => parseEpigeneticsCapabilityGarden(duplicate));

    const unresolved = copyGarden();
    node(unresolved, "analysis-causal-graphs@1").prerequisites.push("missing@1");
    node(unresolved, "analysis-causal-graphs@1").prerequisites.sort();
    assert.throws(
      () => parseEpigeneticsCapabilityGarden(unresolved),
      /missing node/,
    );

    const cyclic = copyGarden();
    node(cyclic, "integrity-experimental-design@1").prerequisites = [
      "analysis-causal-graphs@1",
    ];
    assert.throws(
      () => parseEpigeneticsCapabilityGarden(cyclic),
      /points backward|cycle/,
    );

    const ordinaryTemplate = copyGarden();
    node(ordinaryTemplate, "foundation-gene-regulation@1").rewardEligibility =
      "sponsor-template-only";
    assert.throws(
      () => parseEpigeneticsCapabilityGarden(ordinaryTemplate),
      /quest-only/,
    );

    assert.deepEqual(
      node(gardenDocument, "validation-independent-replication@1").prerequisites,
      [
        "integrity-experimental-design@1",
        "integrity-provenance-metadata@1",
        "integrity-statistics-reproducibility@1",
      ],
    );
    assert.deepEqual(
      node(gardenDocument, "validation-orthogonal-assays@1").prerequisites,
      [
        "foundation-gene-regulation@1",
        "integrity-experimental-design@1",
        "integrity-provenance-metadata@1",
      ],
    );
  });

  it("pins exact milestone splits and formats basis points", () => {
    const garden = parseEpigeneticsCapabilityGardenJson(gardenRaw);
    assert.deepEqual(evaluateLifeGardenSourceReview(garden, "2026-11-01"), {
      checkedThrough: "2026-08-01",
      reviewAfter: "2026-11-01",
      due: false,
    });
    assert.equal(
      evaluateLifeGardenSourceReview(garden, "2026-11-02").due,
      true,
    );

    const drift = copyGarden();
    drift.evidenceLadder[4].rewardBps += 1;
    assert.throws(
      () => parseEpigeneticsCapabilityGarden(drift),
      /rewardBps/,
    );
    assert.equal(formatBasisPoints(0), "0%");
    assert.equal(formatBasisPoints(1500), "15%");
    assert.equal(formatBasisPoints(125), "1.25%");
  });
});

describe("KARMA zero-authority foundation", () => {
  it("preserves one event register, exact non-summable states, and eight closed gates", () => {
    const karma = parseKarmaFoundationJson(karmaRaw);
    assert.equal(karma.eventRegister, "priced-coherence");
    assert.deepEqual(
      karma.eventVocabulary.map(({ id, state }) => [id, state]),
      [
        ["cited", "RECOGNIZED"],
        ["corroborate", "RECOGNIZED"],
        ["corroborated", "RECOGNIZED"],
        ["external", "ORDINAL"],
        ["pending_open", "ORDINAL"],
        ["pending_settle", "ORDINAL"],
        ["verify", "RECOGNIZED"],
      ],
    );
    assert.ok(karma.states.every((state) => !state.summable));
    assert.ok(karma.futureGovernanceGates.every((gate) => !gate.passed));
    assert.equal(karma.futureGovernanceGates.length, 8);
    assert.match(
      karma.eventVocabulary.find((event) => event.id === "verify")?.meaning ?? "",
      /no magnitude.*amount due.*transfer success.*graded/i,
    );
    assert.match(
      karma.eventVocabulary.find((event) => event.id === "corroborate")?.meaning ?? "",
      /challenger.*fact survived.*challenge was rejected.*cooldown/i,
    );
    assert.match(
      karma.eventVocabulary.find((event) => event.id === "external")?.meaning ?? "",
      /settled or partially settled external attestation.*resolvable fact/i,
    );

    const stateDrift = copyKarma();
    stateDrift.eventVocabulary[0].state = "ORDINAL";
    assert.throws(
      () => parseKarmaFoundation(stateDrift),
      /cited\.state.*RECOGNIZED/,
    );
  });

  it("refuses economic, governance, transfer, purchase, rank, and privileged-origin claims", () => {
    for (const key of [
      "authoritative",
      "networkObserved",
      "economicBearing",
      "governanceBearing",
      "transferable",
      "purchasable",
      "delegable",
      "founderPrivilege",
      "operatorPrivilege",
      "scalarRank",
    ] as const) {
      const karma = copyKarma();
      karma[key] = true;
      assert.throws(() => parseKarmaFoundation(karma), LifeGardenDataError);
    }
  });

  it("keeps creator economics declarative and every governance gate closed", () => {
    for (const key of [
      "founderShare",
      "founderControl",
      "operatorShare",
      "operatorControl",
      "creatorRoyalty",
      "structurallyEnforced",
    ]) {
      const karma = copyKarma();
      karma.economicCovenant[key] = true;
      assert.throws(
        () => parseKarmaFoundation(karma),
        new RegExp(`economicCovenant\\.${key}`),
      );
    }
    const opened = copyKarma();
    opened.futureGovernanceGates[0].passed = true;
    assert.throws(() => parseKarmaFoundation(opened), /passed/);
  });

  it("protects non-coercion and prohibits raw-count inference", () => {
    const karma = parseKarmaFoundationJson(karmaRaw);
    assert.match(
      karma.invariants.find((entry) => entry.id === "non-coercion")?.statement ?? "",
      /Rest, silence, refusal, exit, inactivity, privacy/,
    );
    assert.ok(
      karma.prohibitedUses.some((use) => /raw event counts/.test(use)),
    );
    assert.ok(
      karma.prohibitedUses.some((use) => /social credit/.test(use)),
    );

    const missingInvariant = copyKarma();
    missingInvariant.invariants.pop();
    assert.throws(
      () => parseKarmaFoundation(missingInvariant),
      /exactly nine unique invariants/,
    );
    const missingProhibition = copyKarma();
    missingProhibition.prohibitedUses.pop();
    assert.throws(
      () => parseKarmaFoundation(missingProhibition),
      /exactly nine prohibited uses/,
    );
  });
});

describe("life-garden static-standard integrity", () => {
  it("pins both reviewed document digests", () => {
    assert.equal(
      createHash("sha256").update(gardenRaw).digest("hex"),
      EPIGENETICS_GARDEN_SHA256,
    );
    assert.equal(
      createHash("sha256").update(karmaRaw).digest("hex"),
      KARMA_FOUNDATION_SHA256,
    );
  });

  it("passes the offline graph and constitutional validators", () => {
    const garden = validateGardenRaw(gardenRaw);
    const karma = validateKarmaRaw(karmaRaw);
    assert.deepEqual(
      {
        nodes: garden.nodeCount,
        edges: garden.edgeCount,
        depth: garden.maximumDepth,
        fanOut: garden.maximumFanOut,
      },
      { nodes: 25, edges: 56, depth: 9, fanOut: 7 },
    );
    assert.equal(karma.closedGateCount, 8);
  });

  it("rejects duplicate JSON keys before digest review", () => {
    assert.throws(
      () =>
        validateGardenRaw(
          gardenRaw.replace(
            '"authoritative": false,',
            '"authoritative": false, "authoritative": false,',
          ),
        ),
      /duplicate JSON object key/,
    );
  });

  it("fetches each exact same-origin endpoint with refusal controls", async () => {
    const seen: Array<{ input: RequestInfo | URL; init?: RequestInit }> = [];
    const garden = await fetchEpigeneticsCapabilityGarden({
      fetcher: async (input, init) => {
        seen.push({ input, init });
        return jsonResponse(gardenRaw);
      },
    });
    const karma = await fetchKarmaFoundation({
      fetcher: async (input, init) => {
        seen.push({ input, init });
        return jsonResponse(karmaRaw);
      },
    });
    assert.equal(garden.nodes.length, 25);
    assert.equal(karma.invariants.length, 9);
    assert.deepEqual(
      seen.map((request) => request.input),
      [EPIGENETICS_GARDEN_ENDPOINT, KARMA_FOUNDATION_ENDPOINT],
    );
    for (const request of seen) {
      assert.equal(request.init?.cache, "no-store");
      assert.equal(request.init?.credentials, "same-origin");
      assert.equal(request.init?.redirect, "error");
      assert.equal(
        new Headers(request.init?.headers).get("accept"),
        "application/json",
      );
      assert.ok(request.init?.signal instanceof AbortSignal);
    }
  });

  it("rejects digest, final-URL, media-type, status, and size failures", async () => {
    await assert.rejects(
      fetchEpigeneticsCapabilityGarden({
        fetcher: async () => jsonResponse(gardenRaw.replace("Questions before results", "Answers before questions")),
      }),
      /reviewed canonical digest/,
    );

    const redirected = jsonResponse(karmaRaw);
    Object.defineProperty(redirected, "url", {
      value: "https://attacker.example/standards/karma-foundation.v1.json",
    });
    await assert.rejects(
      fetchKarmaFoundation({
        baseUrl: "https://zerone.ai/",
        fetcher: async () => redirected,
      }),
      /canonical same-origin path/,
    );

    await assert.rejects(
      fetchKarmaFoundation({
        fetcher: async () => new Response(karmaRaw, { headers: { "content-type": "text/html" } }),
      }),
      /non-JSON/,
    );
    await assert.rejects(
      fetchKarmaFoundation({
        fetcher: async () =>
          new Response(new TextEncoder().encode(karmaRaw)),
      }),
      /non-JSON/,
    );
    await assert.rejects(
      fetchKarmaFoundation({
        fetcher: async () => jsonResponse(karmaRaw, { status: 503 }),
      }),
      /HTTP 503/,
    );
    await assert.rejects(
      fetchKarmaFoundation({
        fetcher: async () =>
          jsonResponse(karmaRaw, {
            headers: { "content-length": `${STATIC_STANDARD_MAX_BYTES + 1}` },
          }),
      }),
      /size limit/,
    );

    const overflow = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(new Uint8Array(STATIC_STANDARD_MAX_BYTES + 1));
        controller.close();
      },
    });
    await assert.rejects(
      fetchKarmaFoundation({
        fetcher: async () =>
          new Response(overflow, {
            headers: { "content-type": "application/json" },
          }),
      }),
      /size limit/,
    );
    await assert.rejects(
      fetchKarmaFoundation({
        fetcher: async () =>
          new Response(null, {
            headers: { "content-type": "application/json" },
          }),
      }),
      /empty response body/,
    );
  });

  it("times out a response stream that never completes", async () => {
    const never = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(new TextEncoder().encode("{"));
      },
    });
    await assert.rejects(
      fetchEpigeneticsCapabilityGarden({
        timeoutMs: 10,
        fetcher: async () =>
          new Response(never, {
            headers: { "content-type": "application/json" },
          }),
      }),
      /timed out/,
    );
  });
});

import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";

import {
  QUANTUM_SEASON_ENDPOINT,
  QUANTUM_SEASON_MAX_BYTES,
  QUANTUM_SEASON_SHA256,
  QuantumSeasonDataError,
  fetchQuantumSeason,
  parseQuantumSeason,
  parseQuantumSeasonJson,
  quantumSeasonFreshness,
  setQuantumCurrentNode,
} from "../src/quantum-season";

const canonicalRaw = readFileSync(
  new URL(
    "../public/standards/constructive-intelligence-quantum-qec.v0.json",
    import.meta.url,
  ),
  "utf8",
);
const canonical = JSON.parse(canonicalRaw) as Record<string, any>;

function copy(): Record<string, any> {
  return structuredClone(canonical) as Record<string, any>;
}

function response(
  body: BodyInit | null = canonicalRaw,
  init: ResponseInit = {},
  url = `https://zerone.ai${QUANTUM_SEASON_ENDPOINT}`,
): Response {
  const result = new Response(body, {
    ...init,
    headers: {
      "content-type": "application/json; charset=utf-8",
      ...init.headers,
    },
  });
  Object.defineProperty(result, "url", { value: url });
  return result;
}

describe("quantum Season 1 runtime guard", () => {
  it("parses the reviewed 13-node QEC graph", () => {
    const season = parseQuantumSeasonJson(canonicalRaw);
    assert.equal(
      season.schema,
      "zerone.constructive-intelligence-tree-extension/v0",
    );
    assert.equal(season.nodes.length, 13);
    assert.equal(
      season.nodes.reduce(
        (count, node) => count + node.prerequisites.length,
        0,
      ),
      24,
    );
    assert.equal(
      season.nodes.filter((node) => node.stage === "quest").length,
      1,
    );
    assert.equal(season.rewardPolicy.fundedAmount, "0");
    assert.equal(season.rewardPolicy.escrowReceipt, null);
    assert.equal(season.rewardPolicy.claimable, false);
  });

  it("pins the exact static document digest", () => {
    assert.equal(
      createHash("sha256").update(canonicalRaw).digest("hex"),
      QUANTUM_SEASON_SHA256,
    );
  });

  it("refuses authority, money, founder, and KARMA activation", () => {
    const mutations: Array<[string, (season: Record<string, any>) => void]> = [
      ["authoritative", (season) => { season.authoritative = true; }],
      ["moves funds", (season) => { season.releaseBoundary.movesFunds = true; }],
      ["funded", (season) => { season.rewardPolicy.fundedAmount = "1"; }],
      ["claimable", (season) => { season.rewardPolicy.claimable = true; }],
      ["founder share", (season) => { season.rewardPolicy.founderShareBps = 1; }],
      ["founder seat", (season) => { season.rewardPolicy.founderReservedSeats = 1; }],
      ["KARMA vote", (season) => { season.karma.voteWeight = true; }],
      ["KARMA payout", (season) => { season.karma.payoutWeight = true; }],
      ["KARMA transfer", (season) => { season.karma.transferable = true; }],
    ];
    for (const [name, mutate] of mutations) {
      const season = copy();
      mutate(season);
      assert.throws(
        () => parseQuantumSeason(season),
        QuantumSeasonDataError,
        name,
      );
    }
  });

  it("requires precommitment, matched-resource artifacts, independent roots, and adoption", () => {
    const season = parseQuantumSeasonJson(canonicalRaw);
    const quest = season.nodes.find((node) => node.stage === "quest");
    assert.ok(quest?.acceptance);
    assert.equal(quest.acceptance.minimumEffectiveClusters, 3);
    assert.equal(quest.acceptance.minimumOrganizationRoots, 2);
    assert.equal(quest.acceptance.minimumImplementationRoots, 2);
    assert.equal(quest.acceptance.minimumExecutionEnvironments, 2);
    assert.match(
      quest.artifactRequirements.join(" "),
      /energy accounting/,
    );
    assert.match(
      quest.artifactRequirements.join(" "),
      /uncertainty intervals/,
    );
    assert.deepEqual(
      quest.acceptance.fixtures.map(({ n, k, distance }) => [n, k, distance]),
      [[72, 12, 6], [90, 8, 10], [144, 12, 12]],
    );
    assert.deepEqual(
      quest.acceptance.physicalErrorGrid,
      ["0.001", "0.002", "0.003", "0.004", "0.005", "0.006"],
    );

    const leaked = copy();
    leaked.nodes.at(-1).acceptance.scopeBounds =
      leaked.nodes.at(-1).acceptance.scopeBounds.filter(
        (item: string) => item !== "random-seeds=committed-before-execution",
      );
    assert.throws(() => parseQuantumSeason(leaked), /scopeBounds/);
  });

  it("keeps B0-B5 unassignable and the publication non-oracular", () => {
    const season = parseQuantumSeasonJson(canonicalRaw);
    assert.deepEqual(
      season.breakthroughLens.map((level) => level.level),
      ["B0", "B1", "B2", "B3", "B4", "B5"],
    );
    assert.ok(season.breakthroughLens.every((level) => !level.assignable));
    assert.equal(
      season.standards[0]?.treatment,
      "reproduction-target-not-truth-oracle",
    );
    assert.equal(season.standards[0]?.revision, "version of record 2026-05-01");
    assert.equal(season.standards[0]?.authorityStatus, "open-access version of record");
  });

  it("rejects malformed, duplicate-key, unknown-field, and oversized documents", () => {
    assert.throws(() => parseQuantumSeasonJson("{"), /malformed JSON/);
    const duplicate = canonicalRaw.replace(
      '"authoritative": false,',
      '"authoritative": true, "authoritative": false,',
    );
    assert.throws(() => parseQuantumSeasonJson(duplicate), /duplicate JSON object key/);
    const unknown = copy();
    unknown.prize = "1";
    assert.throws(() => parseQuantumSeason(unknown), /unknown or missing fields/);
    assert.throws(
      () => parseQuantumSeasonJson(" ".repeat(QUANTUM_SEASON_MAX_BYTES + 1)),
      /exceeds/,
    );
  });

  it("rejects cycles, stage inversions, graph-shape drift, and arbitrary coverage", () => {
    const cyclic = copy();
    cyclic.nodes.find((node: Record<string, any>) => node.id === "math-complex-linear-algebra@1").prerequisites = [
      "math-tensor-spectral-operators@1",
    ];
    assert.throws(() => parseQuantumSeason(cyclic), /cycle/);

    const inverted = copy();
    inverted.nodes.find((node: Record<string, any>) => node.id === "math-complex-linear-algebra@1").prerequisites = [
      "assurance-formal-verification@1",
    ];
    assert.throws(() => parseQuantumSeason(inverted), /cannot be a prerequisite/);

    const edgeDrift = copy();
    edgeDrift.nodes.at(-1).prerequisites.pop();
    assert.throws(() => parseQuantumSeason(edgeDrift), /edge count/);

    const arbitraryCoverage = copy();
    arbitraryCoverage.nodes.at(-1).acceptance.coverageTargets[0].id = "easy-target";
    assert.throws(() => parseQuantumSeason(arbitraryCoverage), /coverageTargets\[0\]\.id/);

    const zeroCases = copy();
    zeroCases.nodes.at(-1).acceptance.coverageTargets[1].minimumCasesPerCell = 0;
    assert.throws(() => parseQuantumSeason(zeroCases), /minimumCasesPerCell/);
  });

  it("enforces per-cell logical precision and the distinct zero-miss latency rule", () => {
    const season = parseQuantumSeasonJson(canonicalRaw);
    const acceptance = season.nodes.find((node) => node.stage === "quest")?.acceptance;
    assert.ok(acceptance);
    const logical = acceptance.coverageTargets[1];
    const latency = acceptance.coverageTargets[2];
    assert.ok(logical?.analysisMode === "bernoulli-logical-failure");
    assert.equal(logical.minimumLogicalFailuresPerCell, 100);
    assert.equal(logical.confidenceLevelBps, 9900);
    assert.equal(logical.maximumRelativeHalfWidthBps, 3000);
    assert.ok(latency?.analysisMode === "latency-quantile-and-deadline");
    assert.equal(latency.zeroDeadlineMissesMayBeConclusive, true);
    assert.equal(acceptance.computeCapExhaustion, "inconclusive-no-pass");
    assert.equal(
      acceptance.circuitProvenance,
      "stim-circuits-from-version-of-record-references-27-and-50",
    );
    assert.equal(acceptance.distance10And12CircuitEnsembleSize, 24);
    assert.equal(acceptance.rareEventAlternative.appliesTo, "logical-error-cells-only");
    assert.equal(acceptance.rareEventAlternative.unbiasedEstimatorRequired, true);

    const tooFewFailures = copy();
    tooFewFailures.nodes.at(-1).acceptance.coverageTargets[1].minimumLogicalFailuresPerCell = 99;
    assert.throws(() => parseQuantumSeason(tooFewFailures), /minimumLogicalFailuresPerCell/);

    const forcedMisses = copy();
    forcedMisses.nodes.at(-1).acceptance.coverageTargets[2].zeroDeadlineMissesMayBeConclusive = false;
    assert.throws(() => parseQuantumSeason(forcedMisses), /zeroDeadlineMissesMayBeConclusive/);

    const biased = copy();
    biased.nodes.at(-1).acceptance.rareEventAlternative.unbiasedEstimatorRequired = false;
    assert.throws(() => parseQuantumSeason(biased), /unbiasedEstimatorRequired/);

    const smallerCircuitEnsemble = copy();
    smallerCircuitEnsemble.nodes.at(-1).acceptance.distance10And12CircuitEnsembleSize = 1;
    assert.throws(() => parseQuantumSeason(smallerCircuitEnsemble), /distance10And12CircuitEnsembleSize/);
  });

  it("reports source freshness with the review date valid through its boundary", () => {
    const season = parseQuantumSeasonJson(canonicalRaw);
    const before = quantumSeasonFreshness(season, "2026-08-31");
    assert.equal(before.earliestReviewAfter, "2026-09-01");
    assert.equal(before.isExpiredForActiveUse, false);
    assert.equal(quantumSeasonFreshness(season, "2026-09-01").isExpiredForActiveUse, false);
    const after = quantumSeasonFreshness(season, "2026-09-02");
    assert.equal(after.isExpiredForActiveUse, true);
    assert.equal(after.expiredStandardCount, 1);
    assert.throws(() => quantumSeasonFreshness(season, "2026-02-30"), /real calendar date/);
  });

  it("writes literal aria-current=true and removes it from the prior selection", () => {
    const attributes = new Map<string, Map<string, string>>([
      ["first", new Map()],
      ["second", new Map()],
    ]);
    const controls = new Map(
      [...attributes.entries()].map(([id, values]) => [
        id,
        {
          setAttribute(name: string, value: string) { values.set(name, value); },
          removeAttribute(name: string) { values.delete(name); },
        },
      ]),
    );
    setQuantumCurrentNode(controls, "first");
    assert.equal(attributes.get("first")?.get("aria-current"), "true");
    assert.equal(attributes.get("second")?.has("aria-current"), false);
    setQuantumCurrentNode(controls, "second");
    assert.equal(attributes.get("first")?.has("aria-current"), false);
    assert.equal(attributes.get("second")?.get("aria-current"), "true");
  });
});

describe("quantum Season 1 bounded static fetch", () => {
  it("uses the exact same-origin path and restrictive fetch options", async () => {
    let inputSeen: RequestInfo | URL | undefined;
    let initSeen: RequestInit | undefined;
    const season = await fetchQuantumSeason({
      baseUrl: "https://zerone.ai",
      fetcher: async (input, init) => {
        inputSeen = input;
        initSeen = init;
        return response();
      },
    });
    assert.equal(new URL(String(inputSeen)).pathname, QUANTUM_SEASON_ENDPOINT);
    assert.equal(initSeen?.cache, "no-store");
    assert.equal(initSeen?.credentials, "same-origin");
    assert.equal(initSeen?.redirect, "error");
    assert.equal(season.seasonId, "quantum-qec-2026q3");
  });

  it("refuses digest drift, non-JSON, redirects, final-URL drift, errors, and declared oversize", async () => {
    await assert.rejects(
      fetchQuantumSeason({
        baseUrl: "https://zerone.ai",
        fetcher: async () => response(canonicalRaw.replace('"fundedAmount": "0"', '"fundedAmount": "1"')),
      }),
      /reviewed digest/,
    );
    await assert.rejects(
      fetchQuantumSeason({
        baseUrl: "https://zerone.ai",
        fetcher: async () => response(canonicalRaw, { headers: { "content-type": "text/plain" } }),
      }),
      /did not return JSON/,
    );
    const redirected = response();
    Object.defineProperty(redirected, "redirected", { value: true });
    await assert.rejects(
      fetchQuantumSeason({
        baseUrl: "https://zerone.ai",
        fetcher: async () => redirected,
      }),
      /unavailable/,
    );
    await assert.rejects(
      fetchQuantumSeason({
        baseUrl: "https://zerone.ai",
        fetcher: async () => response(canonicalRaw, {}, "https://example.org/quantum.json"),
      }),
      /same-origin path/,
    );
    await assert.rejects(
      fetchQuantumSeason({
        baseUrl: "https://zerone.ai",
        fetcher: async () => new Response("missing", { status: 404 }),
      }),
      /unavailable/,
    );
    await assert.rejects(
      fetchQuantumSeason({
        baseUrl: "https://zerone.ai",
        fetcher: async () => response(canonicalRaw, { headers: { "content-length": String(QUANTUM_SEASON_MAX_BYTES + 1) } }),
      }),
      /byte limit/,
    );
    const oversizedStream = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(new Uint8Array(QUANTUM_SEASON_MAX_BYTES + 1));
        controller.close();
      },
    });
    await assert.rejects(
      fetchQuantumSeason({
        baseUrl: "https://zerone.ai",
        fetcher: async () => response(oversizedStream),
      }),
      /byte limit/,
    );
  });

  it("bounds both a stalled fetch and a stalled response body", async () => {
    await assert.rejects(
      fetchQuantumSeason({
        baseUrl: "https://zerone.ai",
        timeoutMs: 5,
        fetcher: async () => new Promise<Response>(() => {}),
      }),
      /timed out/,
    );

    const stalledBody = new ReadableStream<Uint8Array>({
      start() {
        // Intentionally never enqueue or close.
      },
    });
    await assert.rejects(
      fetchQuantumSeason({
        baseUrl: "https://zerone.ai",
        timeoutMs: 5,
        fetcher: async () => response(stalledBody),
      }),
      /timed out/,
    );
  });
});

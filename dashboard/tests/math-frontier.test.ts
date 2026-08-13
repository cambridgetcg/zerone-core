import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import {
  MATH_FRONTIER_ENDPOINT,
  MATH_FRONTIER_MAX_BYTES,
  MATH_FRONTIER_SHA256,
  buildMathFrontierIndex,
  fetchMathFrontier,
  filterMathFrontierNodes,
  parseMathFrontier,
  parseMathFrontierJson,
  type MathFrontierFilters,
} from "../src/math-frontier";

type MutableFrontier = Record<string, any> & {
  baseTree: Record<string, any>;
  releaseBoundary: Record<string, boolean>;
  constitution: Record<string, any>;
  karma: Record<string, any>;
  rewardTemplate: Record<string, any> & {
    milestones: Array<Record<string, any>>;
  };
  nodes: Array<Record<string, any>>;
  questTemplate: Record<string, any>;
};

const canonicalRaw = readFileSync(
  new URL(
    "../public/standards/constructive-intelligence-math-frontier.v0.json",
    import.meta.url,
  ),
  "utf8",
);
const canonical = JSON.parse(canonicalRaw) as MutableFrontier;
const runtimeSource = readFileSync(
  new URL("../src/math-frontier.ts", import.meta.url),
  "utf8",
);
const mainSource = readFileSync(
  new URL("../src/main.ts", import.meta.url),
  "utf8",
);

function copyFrontier(): MutableFrontier {
  return structuredClone(canonical) as MutableFrontier;
}

function node(frontier: MutableFrontier, id: string): Record<string, any> {
  const found = frontier.nodes.find((candidate) => candidate.id === id);
  assert.ok(found, "missing fixture node " + id);
  return found;
}

function filters(
  overrides: Partial<MathFrontierFilters> = {},
): MathFrontierFilters {
  return {
    query: "",
    stage: "all",
    capabilityClass: "all",
    ...overrides,
  };
}

function jsonResponse(
  body = canonicalRaw,
  init: ResponseInit = {},
): Response {
  return new Response(body, {
    ...init,
    headers: {
      "content-type": "application/json; charset=utf-8",
      ...init.headers,
    },
  });
}

describe("Math Frontier runtime constitution", () => {
  it("parses the canonical four-stage mathematics graph", () => {
    const frontier = parseMathFrontierJson(canonicalRaw);
    assert.equal(
      frontier.schema,
      "zerone.constructive-intelligence-math-frontier/v0",
    );
    assert.deepEqual(frontier.stages, [
      "ground",
      "craft",
      "assurance",
      "frontier",
    ]);
    assert.equal(frontier.nodes.length, 13);
    assert.equal(
      frontier.nodes.reduce(
        (total, capability) => total + capability.prerequisites.length,
        0,
      ),
      23,
    );
    assert.ok(frontier.nodes.every((capability) => !capability.unlocksReward));
  });

  it("keeps economics, KARMA, ranking, and activation at zero", () => {
    const frontier = parseMathFrontierJson(canonicalRaw);
    assert.equal(frontier.rewardTemplate.liveAmount, "0");
    assert.equal(frontier.rewardTemplate.economicEffect, "NONE");
    assert.equal(frontier.karma.mode, "ORDINAL_SHADOW_ONLY");
    assert.equal(frontier.karma.state, "ORDINAL");
    assert.equal(frontier.karma.magnitude, "NONE");
    assert.equal(frontier.karma.controlEffect, "NONE");
    assert.equal(frontier.constitution.currentActivationAuthority, "NONE");
    assert.equal(
      frontier.constitution.futureGovernance,
      "KARMA_ELIGIBILITY_AND_SORTITION_NOT_IMPLEMENTED",
    );
    assert.equal(frontier.releaseBoundary.ranksPersons, false);
    assert.equal(frontier.constitution.wealthAffectsEligibility, false);
    assert.equal(frontier.constitution.wealthAffectsReward, false);
  });

  it("rejects authority switches and every unknown field", () => {
    for (const key of [
      "authoritative",
      "networkObserved",
      "rewardBearing",
      "governanceBearing",
    ]) {
      const frontier = copyFrontier();
      frontier[key] = true;
      assert.throws(() => parseMathFrontier(frontier), new RegExp(key));
    }
    for (const key of Object.keys(canonical.releaseBoundary)) {
      const frontier = copyFrontier();
      frontier.releaseBoundary[key] = true;
      assert.throws(
        () => parseMathFrontier(frontier),
        new RegExp("releaseBoundary\\." + key),
      );
    }

    const topLevel = copyFrontier();
    topLevel.owner = "founder";
    assert.throws(() => parseMathFrontier(topLevel), /owner: unknown field/);

    const nested = copyFrontier();
    nested.constitution.reservedFounderSeat = true;
    assert.throws(
      () => parseMathFrontier(nested),
      /reservedFounderSeat: unknown field/,
    );

    const nodeField = copyFrontier();
    node(nodeField, "math-foundations-logic@1").payoutAddress = "zrn1...";
    assert.throws(
      () => parseMathFrontier(nodeField),
      /payoutAddress: unknown field/,
    );
  });

  it("refuses KARMA recognition, wealth dominance, rewards, and governance shortcuts", () => {
    const recognized = copyFrontier();
    recognized.karma.state = "RECOGNIZED";
    assert.throws(() => parseMathFrontier(recognized), /karma\.state/);

    const truthOracle = copyFrontier();
    truthOracle.karma.truthOracle = true;
    assert.throws(() => parseMathFrontier(truthOracle), /truthOracle/);

    const magnitude = copyFrontier();
    magnitude.karma.magnitude = "1";
    assert.throws(() => parseMathFrontier(magnitude), /karma\.magnitude/);

    const reward = copyFrontier();
    reward.rewardTemplate.liveAmount = "1";
    assert.throws(() => parseMathFrontier(reward), /liveAmount/);

    const wealth = copyFrontier();
    wealth.constitution.wealthAffectsEligibility = true;
    assert.throws(
      () => parseMathFrontier(wealth),
      /wealthAffectsEligibility/,
    );

    const vote = copyFrontier();
    vote.constitution.ordinaryStakeVoteCanActivate = true;
    assert.throws(
      () => parseMathFrontier(vote),
      /ordinaryStakeVoteCanActivate/,
    );
  });

  it("validates graph identity, ordering, reachability, and zero reward unlocks", () => {
    const duplicate = copyFrontier();
    duplicate.nodes[1]!.id = duplicate.nodes[0]!.id;
    assert.throws(() => parseMathFrontier(duplicate), /sorted|duplicate node/);

    const unresolved = copyFrontier();
    node(unresolved, "math-foundations-logic@1").prerequisites = [
      "math-missing@1",
      "math-proofcraft@1",
    ];
    assert.throws(() => parseMathFrontier(unresolved), /missing capability/);

    const cyclic = copyFrontier();
    node(cyclic, "math-foundations-logic@1").prerequisites = [
      "math-prior-art-semantic-root@1",
      "math-proofcraft@1",
    ];
    assert.throws(() => parseMathFrontier(cyclic), /prerequisite cycle/);

    const unsorted = copyFrontier();
    node(unsorted, "math-foundations-logic@1").prerequisites = [
      "math-proofcraft@1",
      "math-algebra-finite-fields@1",
    ];
    assert.throws(() => parseMathFrontier(unsorted), /must be sorted/);

    const unlock = copyFrontier();
    node(unlock, "math-foundations-logic@1").unlocksReward = true;
    assert.throws(() => parseMathFrontier(unlock), /unlocksReward/);
  });

  it("conserves the hypothetical escrow percentages without creating an entitlement", () => {
    const frontier = parseMathFrontierJson(canonicalRaw);
    const outcome = frontier.rewardTemplate.milestones.reduce(
      (total, milestone) => total + milestone.outcomePoolBps,
      0,
    );
    assert.equal(outcome, 8_500);
    assert.equal(frontier.rewardTemplate.challengeReserveBps, 1_500);
    assert.equal(outcome + frontier.rewardTemplate.challengeReserveBps, 10_000);
    assert.equal(
      frontier.rewardTemplate.breakthroughCreatesEntitlement,
      false,
    );
    assert.equal(
      frontier.rewardTemplate.disproofDisposition,
      "E4_TO_COMPLIANT_FALSIFIER_CLAIMANT_UNPAID",
    );

    const drift = copyFrontier();
    drift.rewardTemplate.milestones[0]!.outcomePoolBps += 1;
    assert.throws(
      () => parseMathFrontier(drift),
      /reviewed prospective allocation/,
    );

    const redistributed = copyFrontier();
    redistributed.rewardTemplate.milestones[0]!.outcomePoolBps += 100;
    redistributed.rewardTemplate.milestones[1]!.outcomePoolBps -= 100;
    assert.throws(
      () => parseMathFrontier(redistributed),
      /reviewed prospective allocation/,
    );

    const lowIndependence = copyFrontier();
    lowIndependence.questTemplate.minimumEffectiveClusters = 1;
    assert.throws(
      () => parseMathFrontier(lowIndependence),
      /minimumEffectiveClusters/,
    );

    const wrongDisproofRecipient = copyFrontier();
    wrongDisproofRecipient.rewardTemplate.disproofDisposition =
      "E4_TO_CLAIMANT";
    assert.throws(
      () => parseMathFrontier(wrongDisproofRecipient),
      /disproofDisposition/,
    );

    const selfDeclared = copyFrontier();
    selfDeclared.questTemplate.selfDeclaredBreakthrough = true;
    assert.throws(
      () => parseMathFrontier(selfDeclared),
      /selfDeclaredBreakthrough/,
    );
  });

  it("requires exactly one frozen domain receipt and relation-specific validity", () => {
    const parsed = parseMathFrontierJson(canonicalRaw);
    assert.equal(
      parsed.questTemplate.domainSelectionMode,
      "EXACTLY_ONE_AT_PACKET_FREEZE",
    );
    assert.equal(parsed.questTemplate.selectedDomainEvidenceMinimum, "E2");
    assert.equal(parsed.questTemplate.selectedDomainReceiptRequired, true);
    assert.equal(parsed.questTemplate.relationSpecificValidityRequired, true);
    assert.equal(
      parsed.questTemplate.implementsEstablishesTheoremValidity,
      false,
    );

    const missingReceipt = copyFrontier();
    missingReceipt.questTemplate.selectedDomainReceiptRequired = false;
    assert.throws(
      () => parseMathFrontier(missingReceipt),
      /selectedDomainReceiptRequired/,
    );

    const implementationClaimsTruth = copyFrontier();
    implementationClaimsTruth.questTemplate.implementsEstablishesTheoremValidity =
      true;
    assert.throws(
      () => parseMathFrontier(implementationClaimsTruth),
      /implementsEstablishesTheoremValidity/,
    );
  });

  it("pins the reviewed base-tree identity and frontier revision", () => {
    const baseDigest = copyFrontier();
    baseDigest.baseTree.documentSha256 =
      "sha256:" + "0".repeat(64);
    assert.throws(() => parseMathFrontier(baseDigest), /documentSha256/);

    const basePolicy = copyFrontier();
    basePolicy.baseTree.policyVersion = "1.0.1";
    assert.throws(() => parseMathFrontier(basePolicy), /baseTree\.policyVersion/);

    const revision = copyFrontier();
    revision.policyVersion = "0.1.1";
    assert.throws(() => parseMathFrontier(revision), /policyVersion/);
  });
});

describe("Math Frontier bounded static fetch", () => {
  it("pins the reviewed canonical bytes", () => {
    assert.equal(
      createHash("sha256").update(canonicalRaw).digest("hex"),
      MATH_FRONTIER_SHA256,
    );
    assert.ok(
      Buffer.byteLength(canonicalRaw, "utf8") < MATH_FRONTIER_MAX_BYTES,
    );
  });

  it("uses a versioned same-origin endpoint and refuses redirects", async () => {
    let inputSeen: RequestInfo | URL | undefined;
    let initSeen: RequestInit | undefined;
    const frontier = await fetchMathFrontier({
      fetcher: async (input, init) => {
        inputSeen = input;
        initSeen = init;
        return jsonResponse();
      },
    });
    assert.equal(inputSeen, MATH_FRONTIER_ENDPOINT);
    assert.equal(initSeen?.cache, "no-store");
    assert.equal(initSeen?.credentials, "same-origin");
    assert.equal(initSeen?.redirect, "error");
    assert.equal(
      new Headers(initSeen?.headers).get("accept"),
      "application/json",
    );
    assert.ok(initSeen?.signal instanceof AbortSignal);
    assert.equal(frontier.nodes.length, 13);
  });

  it("rejects byte drift and a final URL outside the exact path", async () => {
    await assert.rejects(
      fetchMathFrontier({
        fetcher: async () =>
          jsonResponse(
            canonicalRaw.replace('"policyVersion": "0.1.0"', '"policyVersion": "0.1.1"'),
          ),
      }),
      /reviewed canonical digest/,
    );

    const foreign = jsonResponse();
    Object.defineProperty(foreign, "url", {
      value:
        "https://attacker.example/standards/constructive-intelligence-math-frontier.v0.json",
    });
    await assert.rejects(
      fetchMathFrontier({
        baseUrl: "https://zerone.ai/",
        fetcher: async () => foreign,
      }),
      /canonical same-origin path/,
    );

    const queryDrift = jsonResponse();
    Object.defineProperty(queryDrift, "url", {
      value:
        "https://zerone.ai/standards/constructive-intelligence-math-frontier.v0.json?unreviewed=1",
    });
    await assert.rejects(
      fetchMathFrontier({
        baseUrl: "https://zerone.ai/",
        fetcher: async () => queryDrift,
      }),
      /canonical same-origin path/,
    );
  });

  it("bounds response type, declared size, streamed size, and time", async () => {
    await assert.rejects(
      fetchMathFrontier({
        fetcher: async () =>
          new Response(canonicalRaw, {
            headers: { "content-type": "text/html" },
          }),
      }),
      /non-JSON/,
    );
    await assert.rejects(
      fetchMathFrontier({
        fetcher: async () =>
          jsonResponse("{}", {
            headers: {
              "content-length": String(MATH_FRONTIER_MAX_BYTES + 1),
            },
          }),
      }),
      /size limit/,
    );
    await assert.rejects(
      fetchMathFrontier({
        fetcher: async () =>
          jsonResponse(" ".repeat(MATH_FRONTIER_MAX_BYTES + 1)),
      }),
      /exceeds/,
    );
    await assert.rejects(
      fetchMathFrontier({
        timeoutMs: 5,
        fetcher: async (_input, init) =>
          await new Promise<Response>((_resolve, reject) => {
            const signal = init?.signal;
            assert.ok(signal);
            signal.addEventListener(
              "abort",
              () => reject(signal.reason),
              { once: true },
            );
          }),
      }),
      /timed out/,
    );
  });
});

describe("Math Frontier presentation helpers", () => {
  const frontier = parseMathFrontierJson(canonicalRaw);
  const index = buildMathFrontierIndex(frontier);

  it("indexes next steps without inventing imported nodes", () => {
    assert.equal(index.byId.size, 13);
    assert.ok(!index.byId.has("math-proofcraft@1"));
    assert.ok(
      index.dependentsById
        .get("math-independent-proof@1")
        ?.some(
          (capability) =>
            capability.id === "math-multi-kernel-assurance@1",
        ),
    );
  });

  it("combines text, stage, and capability-class filters", () => {
    assert.equal(
      filterMathFrontierNodes(
        frontier,
        filters({ stage: "assurance" }),
      ).length,
      2,
    );
    assert.equal(
      filterMathFrontierNodes(
        frontier,
        filters({ capabilityClass: "domain" }),
      ).length,
      5,
    );
    assert.ok(
      filterMathFrontierNodes(
        frontier,
        filters({ query: "counterexample" }),
      ).some(
        (capability) =>
          capability.id === "math-counterexample-falsification@1",
      ),
    );
    assert.deepEqual(
      filterMathFrontierNodes(
        frontier,
        filters({
          query: "downstream use",
          stage: "frontier",
          capabilityClass: "quest",
        }),
      ).map((capability) => capability.id),
      ["quest-math-formal-construction@1"],
    );
  });

  it("corrects a cold direct anchor without a competing smooth scroll", () => {
    assert.match(
      runtimeSource,
      /#math-frontier[\s\S]*scrollIntoView\(\{[\s\S]*block: "start",[\s\S]*behavior: "instant",/,
    );
    assert.match(
      mainSource,
      /Promise\.allSettled\(\[\s*constructiveTreeReady,\s*lifeSciencesTreeReady,\s*quantumSeasonReady,\s*mathFrontierReady,\s*foldToFireReady,\s*lifeGardenReady,\s*\]\)\.then\(alignInitialHash\)/,
    );
    assert.match(
      mainSource,
      /Promise\.allSettled\(\[\s*constructiveTreeReady,\s*lifeSciencesTreeReady,\s*quantumSeasonReady,\s*mathFrontierReady,\s*foldToFireReady,\s*lifeGardenReady,\s*initialNetworkReady,\s*\]\)\.then\(alignInitialHash\)/,
    );
  });
});

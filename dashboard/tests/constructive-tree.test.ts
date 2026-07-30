import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import {
  CONSTRUCTIVE_TREE_ENDPOINT,
  CONSTRUCTIVE_TREE_MAX_BYTES,
  CONSTRUCTIVE_TREE_SHA256,
  CONSTRUCTIVE_TREE_STAGES,
  ConstructiveTreeDataError,
  buildConstructiveTreeIndex,
  constructiveTreeFreshness,
  constructiveTreeRewardSummary,
  fetchConstructiveIntelligenceTree,
  filterConstructiveTreeNodes,
  formatBasisPoints,
  parseConstructiveIntelligenceTree,
  parseConstructiveIntelligenceTreeJson,
  prerequisiteClosure,
  type ConstructiveIntelligenceTree,
  type ConstructiveTreeFilters,
} from "../src/constructive-tree";

type MutableTree = Record<string, any> & {
  nodes: Array<Record<string, any>>;
  releaseBoundary: Record<string, boolean>;
  policy: Record<string, any>;
};

const canonicalRaw = readFileSync(
  new URL(
    "../public/standards/constructive-intelligence-tree.v1.json",
    import.meta.url,
  ),
  "utf8",
);
const canonical = JSON.parse(canonicalRaw) as MutableTree;

function copyTree(): MutableTree {
  return structuredClone(canonical) as MutableTree;
}

function node(tree: MutableTree, id: string): Record<string, any> {
  const found = tree.nodes.find((candidate) => candidate.id === id);
  assert.ok(found, `missing fixture node ${id}`);
  return found;
}

function allFilters(
  overrides: Partial<ConstructiveTreeFilters> = {},
): ConstructiveTreeFilters {
  return {
    query: "",
    stage: "all",
    domain: "all",
    rewardEligibility: "all",
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

describe("constructive-tree runtime presentation guard", () => {
  it("parses the canonical 30-node, 95-edge, five-stage seed", () => {
    const tree = parseConstructiveIntelligenceTreeJson(canonicalRaw);
    assert.equal(tree.schema, "zerone.constructive-intelligence-tree/v1");
    assert.equal(tree.nodes.length, 30);
    assert.equal(
      tree.nodes.reduce(
        (edgeCount, capability) =>
          edgeCount + capability.prerequisites.length,
        0,
      ),
      95,
    );
    assert.deepEqual(
      [
        ...new Set(
          tree.nodes
            .map((capability) => capability.stage)
            .sort(
              (left, right) =>
                CONSTRUCTIVE_TREE_STAGES.indexOf(left) -
                CONSTRUCTIVE_TREE_STAGES.indexOf(right),
            ),
        ),
      ],
      [...CONSTRUCTIVE_TREE_STAGES],
    );
    assert.equal(
      tree.nodes.filter(
        (capability) =>
          capability.rewardEligibility === "sponsor-milestones",
      ).length,
      3,
    );
  });

  it("fails closed if authority, reward, or release boundaries turn on", () => {
    for (const key of [
      "authoritative",
      "networkObserved",
      "rewardBearing",
    ]) {
      const tree = copyTree();
      tree[key] = true;
      assert.throws(
        () => parseConstructiveIntelligenceTree(tree),
        ConstructiveTreeDataError,
      );
    }
    for (const key of Object.keys(canonical.releaseBoundary)) {
      const tree = copyTree();
      tree.releaseBoundary[key] = true;
      assert.throws(
        () => parseConstructiveIntelligenceTree(tree),
        new RegExp(`releaseBoundary\\.${key}`),
      );
    }

    const rewardUnlock = copyTree();
    rewardUnlock.policy.funding.skillUnlockCreatesReward = true;
    assert.throws(
      () => parseConstructiveIntelligenceTree(rewardUnlock),
      /critical recognition, funding, independence, or safety gate changed/,
    );

    const selectedBreakthrough = copyTree();
    selectedBreakthrough.policy.breakthroughRecognition.authorSelected = true;
    assert.throws(
      () => parseConstructiveIntelligenceTree(selectedBreakthrough),
      /critical recognition, funding, independence, or safety gate changed/,
    );
  });

  it("rejects malformed, duplicate, unresolved, cyclic, and unsafe graph data", () => {
    assert.throws(
      () => parseConstructiveIntelligenceTreeJson("{"),
      /malformed JSON/,
    );
    assert.throws(
      () =>
        parseConstructiveIntelligenceTreeJson(
          " ".repeat(CONSTRUCTIVE_TREE_MAX_BYTES + 1),
        ),
      /exceeds/,
    );

    const duplicate = copyTree();
    duplicate.nodes[1]!.id = duplicate.nodes[0]!.id;
    assert.throws(
      () => parseConstructiveIntelligenceTree(duplicate),
      /duplicate node/,
    );

    const unresolved = copyTree();
    node(unresolved, "crypto-aead@1").prerequisites.push("missing-node@1");
    assert.throws(
      () => parseConstructiveIntelligenceTree(unresolved),
      /missing node/,
    );

    const cyclic = copyTree();
    node(cyclic, "systems-exact-bytes-state-machines@1").prerequisites.push(
      "crypto-hash-mac-kdf@1",
    );
    assert.throws(
      () => parseConstructiveIntelligenceTree(cyclic),
      /prerequisite cycle/,
    );

    const unsafeReference = copyTree();
    node(unsafeReference, "math-proofcraft@1").repositoryReferences = [
      "../private",
    ];
    assert.throws(
      () => parseConstructiveIntelligenceTree(unsafeReference),
      /safe repository-relative path/,
    );

    const unsafeUrl = copyTree();
    const protocol = node(unsafeUrl, "protocol-tls13@rfc9846");
    protocol.standards[0].specification =
      "https://user:secret@example.com/spec?token=secret";
    assert.throws(
      () => parseConstructiveIntelligenceTree(unsafeUrl),
      /credential-free HTTPS URL/,
    );
  });

  it("keeps sponsor milestones quest-only and the outcome pool conserved", () => {
    const nonQuestReward = copyTree();
    node(nonQuestReward, "math-proofcraft@1").rewardEligibility =
      "sponsor-milestones";
    assert.throws(
      () => parseConstructiveIntelligenceTree(nonQuestReward),
      /only quest nodes may use sponsor milestones/,
    );

    const unfrozenQuest = copyTree();
    node(unfrozenQuest, "quest-mls-state-invariants@1").acceptance = null;
    assert.throws(
      () => parseConstructiveIntelligenceTree(unfrozenQuest),
      /quest nodes require acceptance bounds/,
    );

    const driftedPool = copyTree();
    driftedPool.policy.milestones[2].rewardBps += 1;
    assert.throws(
      () => parseConstructiveIntelligenceTree(driftedPool),
      /must equal 10,000 bps/,
    );
  });
});

describe("constructive-tree bounded static fetch", () => {
  it("pins the reviewed canonical document digest", () => {
    assert.equal(
      createHash("sha256").update(canonicalRaw).digest("hex"),
      CONSTRUCTIVE_TREE_SHA256,
    );
  });

  it("uses the same-origin versioned endpoint with no-store, redirect refusal, and a timeout", async () => {
    let inputSeen: RequestInfo | URL | undefined;
    let initSeen: RequestInit | undefined;
    const tree = await fetchConstructiveIntelligenceTree({
      fetcher: async (input, init) => {
        inputSeen = input;
        initSeen = init;
        return jsonResponse();
      },
    });
    assert.equal(inputSeen, CONSTRUCTIVE_TREE_ENDPOINT);
    assert.equal(initSeen?.cache, "no-store");
    assert.equal(initSeen?.credentials, "same-origin");
    assert.equal(initSeen?.redirect, "error");
    assert.equal(
      new Headers(initSeen?.headers).get("accept"),
      "application/json",
    );
    assert.ok(initSeen?.signal instanceof AbortSignal);
    assert.equal(tree.nodes.length, 30);
  });

  it("rejects content drift and a final URL outside the exact same-origin path", async () => {
    await assert.rejects(
      fetchConstructiveIntelligenceTree({
        fetcher: async () =>
          jsonResponse(canonicalRaw.replace('"policyVersion": "1.0.0"', '"policyVersion": "1.0.1"')),
      }),
      /reviewed canonical digest/,
    );

    const redirected = jsonResponse();
    Object.defineProperty(redirected, "url", {
      value:
        "https://attacker.example/standards/constructive-intelligence-tree.v1.json",
    });
    await assert.rejects(
      fetchConstructiveIntelligenceTree({
        baseUrl: "https://zerone.ai/",
        fetcher: async () => redirected,
      }),
      /canonical same-origin path/,
    );

    const queryDrift = jsonResponse();
    Object.defineProperty(queryDrift, "url", {
      value:
        "https://zerone.ai/standards/constructive-intelligence-tree.v1.json?revision=unreviewed",
    });
    await assert.rejects(
      fetchConstructiveIntelligenceTree({
        baseUrl: "https://zerone.ai/",
        fetcher: async () => queryDrift,
      }),
      /canonical same-origin path/,
    );
  });

  it("rejects HTTP, media-type, declared-size, and streamed-size failures", async () => {
    await assert.rejects(
      fetchConstructiveIntelligenceTree({
        fetcher: async () => jsonResponse("{}", { status: 503 }),
      }),
      /HTTP 503/,
    );
    await assert.rejects(
      fetchConstructiveIntelligenceTree({
        fetcher: async () =>
          new Response(canonicalRaw, {
            headers: { "content-type": "text/html" },
          }),
      }),
      /non-JSON/,
    );
    await assert.rejects(
      fetchConstructiveIntelligenceTree({
        fetcher: async () =>
          jsonResponse("{}", {
            headers: {
              "content-length": `${CONSTRUCTIVE_TREE_MAX_BYTES + 1}`,
            },
          }),
      }),
      /size limit/,
    );
    await assert.rejects(
      fetchConstructiveIntelligenceTree({
        fetcher: async () =>
          jsonResponse(" ".repeat(CONSTRUCTIVE_TREE_MAX_BYTES + 1)),
      }),
      /exceeds/,
    );

    const oversizedChunk = new Uint8Array(
      Math.floor(CONSTRUCTIVE_TREE_MAX_BYTES / 2) + 1,
    ).fill(0x20);
    await assert.rejects(
      fetchConstructiveIntelligenceTree({
        fetcher: async () =>
          new Response(
            new ReadableStream<Uint8Array>({
              start(controller) {
                controller.enqueue(oversizedChunk);
                controller.enqueue(oversizedChunk);
                controller.close();
              },
            }),
            { headers: { "content-type": "application/json" } },
          ),
      }),
      /exceeds/,
    );
  });

  it("bounds a stalled request and can be called again after a failure", async () => {
    await assert.rejects(
      fetchConstructiveIntelligenceTree({
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

    let attempt = 0;
    const fetcher = async (): Promise<Response> => {
      attempt += 1;
      return attempt === 1
        ? jsonResponse("{}", { status: 502 })
        : jsonResponse();
    };
    await assert.rejects(
      fetchConstructiveIntelligenceTree({ fetcher }),
      /HTTP 502/,
    );
    const recovered = await fetchConstructiveIntelligenceTree({ fetcher });
    assert.equal(recovered.nodes.length, 30);
    assert.equal(attempt, 2);
  });

  it("applies the same deadline while consuming a stalled response body", async () => {
    let cancelled = false;
    await assert.rejects(
      fetchConstructiveIntelligenceTree({
        timeoutMs: 5,
        fetcher: async () =>
          new Response(
            new ReadableStream<Uint8Array>({
              start(controller) {
                controller.enqueue(new TextEncoder().encode('{"schema":'));
              },
              cancel() {
                cancelled = true;
              },
            }),
            { headers: { "content-type": "application/json" } },
          ),
      }),
      /timed out/,
    );
    assert.equal(cancelled, true);
  });

  it("refuses overflow and timeout even when hostile stream cancellation never settles", async () => {
    const neverCancels = (): Promise<void> => new Promise(() => {});
    const oversizedChunk = new Uint8Array(
      CONSTRUCTIVE_TREE_MAX_BYTES + 1,
    ).fill(0x20);
    await assert.rejects(
      settlesWithin(
        fetchConstructiveIntelligenceTree({
          fetcher: async () =>
            new Response(
              new ReadableStream<Uint8Array>({
                start(controller) {
                  controller.enqueue(oversizedChunk);
                },
                cancel: neverCancels,
              }),
              { headers: { "content-type": "application/json" } },
            ),
        }),
        100,
      ),
      /exceeds/,
    );

    await assert.rejects(
      settlesWithin(
        fetchConstructiveIntelligenceTree({
          timeoutMs: 5,
          fetcher: async () =>
            new Response(
              new ReadableStream<Uint8Array>({
                start(controller) {
                  controller.enqueue(new TextEncoder().encode('{"schema":'));
                },
                cancel: neverCancels,
              }),
              { headers: { "content-type": "application/json" } },
            ),
        }),
        100,
      ),
      /timed out/,
    );
  });
});

describe("constructive-tree graph and filters", () => {
  const tree: ConstructiveIntelligenceTree =
    parseConstructiveIntelligenceTreeJson(canonicalRaw);
  const index = buildConstructiveTreeIndex(tree);

  it("indexes direct dependents and deduplicates transitive prerequisites", () => {
    assert.equal(
      index.dependentsById.get("math-proofcraft@1")?.length,
      6,
    );
    const closure = prerequisiteClosure(
      index,
      "quest-tls-rfc9846-keyshare-reuse@1",
    );
    assert.equal(closure.length, 14);
    assert.equal(new Set(closure.map((capability) => capability.id)).size, 14);
    assert.ok(
      closure.some((capability) => capability.id === "math-proofcraft@1"),
    );
    assert.ok(
      closure.some(
        (capability) => capability.id === "protocol-tls13@rfc9846",
      ),
    );
    assert.throws(
      () => prerequisiteClosure(index, "missing@1"),
      /Unknown capability/,
    );
  });

  it("combines text, stage, domain, and funding filters", () => {
    const mathematics = filterConstructiveTreeNodes(
      tree,
      allFilters({ stage: "foundation", domain: "mathematics" }),
    );
    assert.equal(mathematics.length, 4);
    assert.ok(
      mathematics.every(
        (capability) =>
          capability.stage === "foundation" &&
          capability.domain === "mathematics",
      ),
    );

    const tls = filterConstructiveTreeNodes(
      tree,
      allFilters({
        query: "RFC 9846",
        stage: "protocol",
        domain: "protocols",
      }),
    );
    assert.deepEqual(
      tls.map((capability) => capability.id),
      ["protocol-tls13@rfc9846"],
    );

    const quests = filterConstructiveTreeNodes(
      tree,
      allFilters({ rewardEligibility: "sponsor-milestones" }),
    );
    assert.equal(quests.length, 3);
    assert.ok(quests.every((capability) => capability.stage === "quest"));

    assert.equal(
      filterConstructiveTreeNodes(
        tree,
        allFilters({ query: "not-a-capability-or-standard" }),
      ).length,
      0,
    );
  });

  it("matches identifiers, standard authorities, and artifact language", () => {
    assert.ok(
      filterConstructiveTreeNodes(
        tree,
        allFilters({ query: "mlswg" }),
      ).some(
        (capability) =>
          capability.id === "quest-mls-state-invariants@1",
      ),
    );
    assert.ok(
      filterConstructiveTreeNodes(
        tree,
        allFilters({ query: "NIST" }),
      ).some(
        (capability) => capability.id === "crypto-ml-kem@fips203",
      ),
    );
    assert.ok(
      filterConstructiveTreeNodes(
        tree,
        allFilters({ query: "counterexamples" }),
      ).some((capability) => capability.id === "math-proofcraft@1"),
    );
  });
});

describe("constructive-tree economics and freshness presentation", () => {
  const tree = parseConstructiveIntelligenceTreeJson(canonicalRaw);

  it("reports inactive template economics without turning E0/E1 into payouts", () => {
    const summary = constructiveTreeRewardSummary(tree);
    assert.deepEqual(summary, {
      qualificationOnlyCount: 27,
      sponsorMilestoneCount: 3,
      milestoneBps: 8_500,
      challengeReserveBps: 1_500,
      totalOutcomePoolBps: 10_000,
      rewardBearing: false,
    });
    assert.deepEqual(
      tree.policy.milestones.slice(0, 2).map((milestone) => ({
        level: milestone.level,
        bps: milestone.rewardBps,
        treatment: milestone.treatment,
      })),
      [
        { level: "E0", bps: 0, treatment: "precedence-only" },
        { level: "E1", bps: 0, treatment: "verified-cost-only" },
      ],
    );
    assert.equal(formatBasisPoints(1_500), "15%");
    assert.equal(formatBasisPoints(125), "1.25%");
    assert.throws(() => formatBasisPoints(-1), /non-negative integer/);
  });

  it("treats reviewAfter as valid through that date and stale only after it", () => {
    const before = constructiveTreeFreshness(tree, "2026-08-04");
    assert.equal(before.earliestReviewAfter, "2026-08-05");
    assert.equal(before.isExpiredForActiveUse, false);
    assert.equal(before.expiredStandardCount, 0);

    const onBoundary = constructiveTreeFreshness(tree, "2026-08-05");
    assert.equal(onBoundary.isExpiredForActiveUse, false);

    const after = constructiveTreeFreshness(tree, "2026-08-06");
    assert.equal(after.isExpiredForActiveUse, true);
    assert.ok(after.expiredStandardCount > 0);

    assert.throws(
      () => constructiveTreeFreshness(tree, "2026-02-30"),
      /real calendar date/,
    );
  });
});

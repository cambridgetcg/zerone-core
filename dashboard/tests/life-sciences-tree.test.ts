import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import {
  LIFE_SCIENCES_TREE_ENDPOINT,
  LIFE_SCIENCES_TREE_MAX_BYTES,
  LIFE_SCIENCES_TREE_SHA256,
  LifeSciencesTreeError,
  fetchLifeSciencesOverlay,
  parseLifeSciencesOverlay,
} from "../src/life-sciences-tree";

type MutableOverlay = Record<string, any> & {
  baseTreeBinding: Record<string, any>;
  releaseBoundary: Record<string, any>;
  scope: Record<string, any>;
  nodes: Array<Record<string, any>>;
};

const canonicalRaw = readFileSync(
  new URL(
    "../public/standards/constructive-intelligence-life-sciences.v0.json",
    import.meta.url,
  ),
  "utf8",
);
const canonical = JSON.parse(canonicalRaw) as MutableOverlay;

function copyOverlay(): MutableOverlay {
  return structuredClone(canonical) as MutableOverlay;
}

function jsonResponse(
  body: BodyInit | null = canonicalRaw,
  init: ResponseInit = {},
  url = `https://zerone.ai${LIFE_SCIENCES_TREE_ENDPOINT}`,
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

describe("life-sciences shadow tree runtime guard", () => {
  it("binds the reviewed 17-node profile to the immutable core tree", () => {
    const overlay = parseLifeSciencesOverlay(canonical);
    assert.equal(overlay.status, "DRAFT");
    assert.equal(overlay.mode, "SHADOW_ONLY");
    assert.equal(overlay.nodes.length, 17);
    assert.equal(
      overlay.nodes.reduce((count, node) => count + node.prerequisites.length, 0),
      24,
    );
    assert.equal(overlay.nodes.filter((node) => node.crown).length, 1);
    assert.equal(
      overlay.baseTreeSha256,
      "8070d8d1b7ea28a314f5a8550c675d7ccbe5d9b234ef02d54d4913c650c01aaf",
    );
    assert.equal(createHash("sha256").update(canonicalRaw).digest("hex"), LIFE_SCIENCES_TREE_SHA256);
  });

  it("refuses activation, authority, economics, and core-tree drift", () => {
    for (const key of ["authoritative", "networkObserved", "rewardBearing"]) {
      const overlay = copyOverlay();
      overlay[key] = true;
      assert.throws(() => parseLifeSciencesOverlay(overlay), LifeSciencesTreeError);
    }

    const reward = copyOverlay();
    reward.releaseBoundary.activatesRewards = true;
    assert.throws(() => parseLifeSciencesOverlay(reward), /activatesRewards/);

    const money = copyOverlay();
    money.economics.amount = "1";
    assert.throws(() => parseLifeSciencesOverlay(money), /NONE\/0/);

    const coreDrift = copyOverlay();
    coreDrift.baseTreeBinding.sha256 = "0".repeat(64);
    assert.throws(() => parseLifeSciencesOverlay(coreDrift), /core-tree binding drifted/);
  });

  it("requires the refusal boundary, valid prerequisites, and one crown", () => {
    const missingRefusal = copyOverlay();
    missingRefusal.scope.refusedTopics = missingRefusal.scope.refusedTopics.filter(
      (topic: string) => topic !== "PATHOGENS",
    );
    assert.throws(() => parseLifeSciencesOverlay(missingRefusal), /missing PATHOGENS/);

    const dangling = copyOverlay();
    dangling.nodes[0]!.prerequisites = ["missing@1"];
    assert.throws(() => parseLifeSciencesOverlay(dangling), /dangling prerequisite/);

    const secondCrown = copyOverlay();
    secondCrown.nodes[0]!.crown = true;
    assert.throws(() => parseLifeSciencesOverlay(secondCrown), /exactly one crown/);
  });

  it("fetches exact reviewed bytes with restrictive same-origin options", async () => {
    let inputSeen: RequestInfo | URL | undefined;
    let initSeen: RequestInit | undefined;
    const overlay = await fetchLifeSciencesOverlay({
      baseUrl: "https://zerone.ai/",
      fetcher: async (input, init) => {
        inputSeen = input;
        initSeen = init;
        return jsonResponse();
      },
    });
    assert.equal(inputSeen, LIFE_SCIENCES_TREE_ENDPOINT);
    assert.equal(initSeen?.cache, "no-store");
    assert.equal(initSeen?.credentials, "same-origin");
    assert.equal(initSeen?.redirect, "error");
    assert.equal(
      new Headers(initSeen?.headers).get("accept"),
      "application/json",
    );
    assert.ok(initSeen?.signal instanceof AbortSignal);
    assert.equal(overlay.nodes.length, 17);

    await assert.rejects(
      fetchLifeSciencesOverlay({
        fetcher: async () => jsonResponse(`${canonicalRaw}\n`),
      }),
      /do not match the reviewed SHA-256/,
    );
  });

  it("refuses redirects and final URLs outside the exact path", async () => {
    const redirected = jsonResponse();
    Object.defineProperty(redirected, "redirected", { value: true });
    await assert.rejects(
      fetchLifeSciencesOverlay({
        baseUrl: "https://zerone.ai/",
        fetcher: async () => redirected,
      }),
      /redirected/,
    );

    await assert.rejects(
      fetchLifeSciencesOverlay({
        baseUrl: "https://zerone.ai/",
        fetcher: async () =>
          jsonResponse(
            canonicalRaw,
            {},
            "https://attacker.example/constructive-intelligence-life-sciences.v0.json",
          ),
      }),
      /canonical same-origin path/,
    );

    await assert.rejects(
      fetchLifeSciencesOverlay({
        baseUrl: "https://zerone.ai/",
        fetcher: async () =>
          jsonResponse(
            canonicalRaw,
            {},
            `https://zerone.ai${LIFE_SCIENCES_TREE_ENDPOINT}?revision=unreviewed`,
          ),
      }),
      /canonical same-origin path/,
    );
  });

  it("bounds media type, declared length, and streamed length", async () => {
    await assert.rejects(
      fetchLifeSciencesOverlay({
        fetcher: async () =>
          jsonResponse(canonicalRaw, {
            headers: { "content-type": "text/html" },
          }),
      }),
      /non-JSON/,
    );

    await assert.rejects(
      fetchLifeSciencesOverlay({
        fetcher: async () =>
          jsonResponse(canonicalRaw, {
            headers: {
              "content-length": String(LIFE_SCIENCES_TREE_MAX_BYTES + 1),
            },
          }),
      }),
      /byte limit/,
    );

    await assert.rejects(
      fetchLifeSciencesOverlay({
        fetcher: async () =>
          jsonResponse(canonicalRaw, {
            headers: { "content-length": "not-a-number" },
          }),
      }),
      /byte limit/,
    );

    const oversizedStream = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(new Uint8Array(LIFE_SCIENCES_TREE_MAX_BYTES + 1));
        controller.close();
      },
    });
    await assert.rejects(
      fetchLifeSciencesOverlay({
        fetcher: async () => jsonResponse(oversizedStream),
      }),
      /byte limit/,
    );
  });

  it("applies one deadline to stalled fetches and response bodies", async () => {
    await assert.rejects(
      settlesWithin(
        fetchLifeSciencesOverlay({
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
        return new Promise<void>(() => {});
      },
    });
    await assert.rejects(
      settlesWithin(
        fetchLifeSciencesOverlay({
          timeoutMs: 5,
          fetcher: async () => jsonResponse(stalledBody),
        }),
        250,
      ),
      /timed out/,
    );
    assert.equal(cancelled, true);
  });
});

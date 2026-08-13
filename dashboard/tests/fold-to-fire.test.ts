import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";

import {
  FOLD_TO_FIRE_ENDPOINT,
  FOLD_TO_FIRE_MAX_BYTES,
  FOLD_TO_FIRE_SHA256,
  fetchFoldToFire,
  parseFoldToFire,
  parseFoldToFireJson,
} from "../src/fold-to-fire";

const canonicalRaw = readFileSync(
  new URL(
    "../public/standards/constructive-intelligence-fold-to-fire.v0.json",
    import.meta.url,
  ),
  "utf8",
);
const canonical = JSON.parse(canonicalRaw) as Record<string, any>;
const runtimeSource = readFileSync(
  new URL("../src/fold-to-fire.ts", import.meta.url),
  "utf8",
);
const mainSource = readFileSync(new URL("../src/main.ts", import.meta.url), "utf8");
const htmlSource = readFileSync(new URL("../index.html", import.meta.url), "utf8");

function copy(): Record<string, any> {
  return structuredClone(canonical) as Record<string, any>;
}

function jsonResponse(body = canonicalRaw, init: ResponseInit = {}): Response {
  return new Response(body, {
    ...init,
    headers: {
      "content-type": "application/json; charset=utf-8",
      ...init.headers,
    },
  });
}

describe("Fold-to-Fire runtime contract", () => {
  it("parses the sealed exact profile and corrected finite rows", () => {
    const profile = parseFoldToFireJson(canonicalRaw);
    assert.equal(profile.schema, "zerone.constructive-intelligence-fold-to-fire/v0");
    assert.equal(profile.enumeration.rows.length, 7);
    assert.deepEqual(
      profile.enumeration.rows.map(({ stepCount, totalWalks, activeWalks }) => [
        stepCount,
        totalWalks,
        activeWalks,
      ]),
      [
        [3, "9", "2"],
        [5, "71", "6"],
        [7, "543", "28"],
        [9, "4067", "140"],
        [11, "30073", "744"],
        [13, "220375", "4116"],
        [15, "1604149", "23504"],
      ],
    );
    assert.equal(createHash("sha256").update(canonicalRaw).digest("hex"), FOLD_TO_FIRE_SHA256);
  });

  it("keeps the open conjecture distinct from the weighted bridge", () => {
    const profile = parseFoldToFire(canonical);
    assert.equal(profile.frontierProblem.status, "ESTABLISHED_OPEN_CONJECTURE");
    assert.equal(profile.frontierProblem.exponent, "59/32");
    assert.equal(profile.frontierProblem.computationDoesNotProve, true);
    assert.equal(profile.weightedBridge.status, "BESPOKE_RESEARCH_BRIDGE");
    assert.equal(profile.weightedBridge.qNotEqualOneExponentTransfer, "OUT_OF_SCOPE_NOT_CLAIMED");
    assert.equal(profile.weightedBridge.noveltyAuditRequired, true);
  });

  it("rejects unknown fields, altered counts, source drift, and effect switches", () => {
    const unknown = copy();
    unknown.owner = "founder";
    assert.throws(() => parseFoldToFire(unknown), /missing or unknown fields/);

    const count = copy();
    count.enumeration.rows[5].activeWalks = "4212";
    assert.throws(() => parseFoldToFire(count), /reviewed v0 contract/);

    const source = copy();
    source.sources[0].url = "https://example.com/substitute";
    assert.throws(() => parseFoldToFire(source), /must equal/);

    const authority = copy();
    authority.releaseBoundary.createsKarmaEvent = true;
    assert.throws(() => parseFoldToFire(authority), /createsKarmaEvent/);

    const weighted = copy();
    weighted.weightedBridge.qNotEqualOneExponentTransfer = "CLAIMED";
    assert.throws(() => parseFoldToFire(weighted), /qNotEqualOneExponentTransfer/);
  });

  it("rejects malformed and oversized JSON before presentation", () => {
    assert.throws(() => parseFoldToFireJson("{"), /malformed JSON/);
    assert.throws(
      () => parseFoldToFireJson(" ".repeat(FOLD_TO_FIRE_MAX_BYTES + 1)),
      /document exceeds/,
    );
  });
});

describe("Fold-to-Fire bounded fetch", () => {
  it("uses one exact static request and returns only reviewed bytes", async () => {
    const calls: Array<{ input: RequestInfo | URL; init?: RequestInit }> = [];
    const profile = await fetchFoldToFire({
      fetcher: async (input, init) => {
        calls.push({ input, init });
        return jsonResponse();
      },
    });
    assert.equal(profile.enumeration.rows.length, 7);
    assert.equal(calls.length, 1);
    assert.equal(calls[0]?.input, FOLD_TO_FIRE_ENDPOINT);
    assert.deepEqual(
      {
        cache: calls[0]?.init?.cache,
        credentials: calls[0]?.init?.credentials,
        redirect: calls[0]?.init?.redirect,
      },
      { cache: "no-store", credentials: "same-origin", redirect: "error" },
    );
  });

  it("fails closed on tamper, redirects, foreign paths, and non-JSON", async () => {
    await assert.rejects(
      fetchFoldToFire({ fetcher: async () => jsonResponse(canonicalRaw + " ") }),
      /reviewed digest/,
    );

    const redirected = jsonResponse();
    Object.defineProperty(redirected, "redirected", { value: true });
    await assert.rejects(
      fetchFoldToFire({ fetcher: async () => redirected }),
      /redirected/,
    );

    const foreign = jsonResponse();
    Object.defineProperty(foreign, "url", {
      value: "https://attacker.example/standards/constructive-intelligence-fold-to-fire.v0.json",
    });
    await assert.rejects(
      fetchFoldToFire({ baseUrl: "https://zerone.ai/", fetcher: async () => foreign }),
      /canonical same-origin path/,
    );

    await assert.rejects(
      fetchFoldToFire({
        fetcher: async () =>
          new Response(canonicalRaw, { headers: { "content-type": "text/html" } }),
      }),
      /non-JSON/,
    );
    for (const contentType of [null, "text/json", "application/jsonp"]) {
      await assert.rejects(
        fetchFoldToFire({
          fetcher: async () =>
            new Response(canonicalRaw, {
              headers: contentType === null ? {} : { "content-type": contentType },
            }),
        }),
        /non-JSON/,
      );
    }
  });

  it("bounds declared size, streamed size, and request time", async () => {
    await assert.rejects(
      fetchFoldToFire({
        fetcher: async () =>
          jsonResponse("{}", {
            headers: { "content-length": String(FOLD_TO_FIRE_MAX_BYTES + 1) },
          }),
      }),
      /size limit/,
    );
    await assert.rejects(
      fetchFoldToFire({ fetcher: async () => jsonResponse(" ".repeat(FOLD_TO_FIRE_MAX_BYTES + 1)) }),
      /size limit/,
    );
    await assert.rejects(
      fetchFoldToFire({
        timeoutMs: 5,
        fetcher: async (_input, init) =>
          await new Promise<Response>((_resolve, reject) => {
            init?.signal?.addEventListener("abort", () => reject(init.signal?.reason), { once: true });
          }),
      }),
      /timed out/,
    );
    await assert.rejects(
      fetchFoldToFire({
        timeoutMs: 5,
        fetcher: async () => await new Promise<Response>(() => undefined),
      }),
      /timed out/,
    );
  });
});

describe("Fold-to-Fire public presentation", () => {
  it("uses text-node rendering, direct-anchor settling, and a complete no-JS account", () => {
    assert.doesNotMatch(runtimeSource, /innerHTML/);
    assert.match(runtimeSource, /#fold-to-fire[\s\S]*scrollIntoView/);
    assert.match(mainSource, /foldToFireReady/);
    assert.match(mainSource, /window\.location\.hash !== "#fold-to-fire"/);
    assert.match(htmlSource, /<noscript>[\s\S]*59\/32[\s\S]*23,504[\s\S]*Zero effects/);
    assert.match(htmlSource, /rapid-equilibrium, unit-occupancy toy assumption/);
  });
});

import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import {
  LIFE_SCIENCES_TREE_ENDPOINT,
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

  it("fetches only exact reviewed bytes and fails closed on tampering", async () => {
    let requested = "";
    const fetcher = (async (input: RequestInfo | URL) => {
      requested = String(input);
      return new Response(canonicalRaw, { status: 200 });
    }) as typeof fetch;
    const overlay = await fetchLifeSciencesOverlay(fetcher);
    assert.equal(requested, LIFE_SCIENCES_TREE_ENDPOINT);
    assert.equal(overlay.nodes.length, 17);

    const tamperedFetcher = (async () =>
      new Response(`${canonicalRaw}\n`, { status: 200 })) as typeof fetch;
    await assert.rejects(
      fetchLifeSciencesOverlay(tamperedFetcher),
      /do not match the reviewed SHA-256/,
    );
  });
});

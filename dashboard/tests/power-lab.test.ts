import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import {
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, it } from "node:test";
import {
  CAPACITY_SHADOW_EVENTS,
  LEDGER_LANE_IDS,
  POWER_LAB_SCHEMA,
  POWER_SURFACE_IDS,
  PowerLabValidationError,
  formatModelUnits,
  mathNodeLevels,
  parsePowerLabFixture,
  type PowerLabFixture,
} from "../src/power-lab/model";

const fixturePath = new URL(
  "../src/power-lab/shadow.generated.json",
  import.meta.url,
);
const rawFixture: unknown = JSON.parse(readFileSync(fixturePath, "utf8"));
const canonical = parsePowerLabFixture(rawFixture);

function copyFixture(): PowerLabFixture {
  return structuredClone(canonical);
}

function assertInvalid(value: unknown, path?: string): void {
  assert.throws(
    () => parsePowerLabFixture(value),
    (error) =>
      error instanceof PowerLabValidationError &&
      (path === undefined || error.path === path),
  );
}

describe("local power-lab fixture", () => {
  it("validates the deterministic zero-value projection", () => {
    assert.equal(canonical.schema, POWER_LAB_SCHEMA);
    assert.equal(canonical.authoritative, false);
    assert.equal(canonical.networkObserved, false);
    assert.equal(canonical.rewardBearing, false);
    assert.equal(canonical.transferableValue, false);
    assert.equal(canonical.chainStateRead, false);
    assert.equal(canonical.capability.nodeCount, 30);
    assert.equal(canonical.capability.questCount, 3);
    assert.equal(canonical.capability.mathNodes.length, 4);
    assert.equal(canonical.powerStress.surfaces.length, 12);
    assert.deepEqual(
      canonical.powerStress.surfaces.map((surface) => surface.id),
      POWER_SURFACE_IDS,
    );
    assert.equal(canonical.ledgerLanes.length, 6);
    assert.deepEqual(
      canonical.ledgerLanes.map((lane) => lane.id),
      LEDGER_LANE_IDS,
    );
    assert.equal(canonical.release.modelPassCount, 12);
    assert.equal(canonical.release.integrationFailCount, 19);
    assert.equal(canonical.release.integrationReady, false);
    assert.equal(canonical.release.settlementZrn, 0);
    assert.equal(canonical.capacityShadow.settlementZrn, 0);
    assert.equal(canonical.capacityShadow.integrationReady, false);
    assert.equal(canonical.capacityShadow.checks.passed, true);
    assert.deepEqual(
      canonical.capacityShadow.trace.map((step) => step.event),
      CAPACITY_SHADOW_EVENTS,
    );
    assert.equal(formatModelUnits(1000), "1,000 MU");
    assert.doesNotMatch(formatModelUnits(1000), /ZRN/);
  });

  it("fails closed if any authority, observation, value, or release boundary opens", () => {
    for (const key of [
      "authoritative",
      "networkObserved",
      "rewardBearing",
      "transferableValue",
      "chainStateRead",
    ] as const) {
      const fixture = copyFixture() as unknown as Record<string, unknown>;
      fixture[key] = true;
      assertInvalid(fixture, `$.${key}`);
    }

    for (const key of Object.keys(canonical.treeBoundary)) {
      const fixture = copyFixture() as unknown as {
        treeBoundary: Record<string, unknown>;
      };
      fixture.treeBoundary[key] = true;
      assertInvalid(fixture, `$.treeBoundary.${key}`);
    }

    const settlement = copyFixture() as unknown as {
      release: { settlementEnabled: boolean; settlementZrn: number };
    };
    settlement.release.settlementEnabled = true;
    settlement.release.settlementZrn = 1;
    assertInvalid(settlement, "$.release.settlementEnabled");

    const claim = copyFixture() as unknown as {
      release: { claimable: boolean };
    };
    claim.release.claimable = true;
    assertInvalid(claim, "$.release.claimable");

    const clusterSettlement = copyFixture() as unknown as {
      shadowEpoch: {
        clusters: Array<{ settlementZrn: number }>;
      };
    };
    const firstCluster = clusterSettlement.shadowEpoch.clusters[0];
    assert.ok(firstCluster);
    firstCluster.settlementZrn = 0.000001;
    assertInvalid(
      clusterSettlement,
      "$.shadowEpoch.clusters[0].settlementZrn",
    );
  });

  it("preserves the exact unblended surface vector and scenario separation", () => {
    const missing = copyFixture();
    missing.powerStress.surfaces.pop();
    assertInvalid(missing, "$.powerStress.surfaces");

    const extra = copyFixture();
    extra.powerStress.surfaces.push(
      structuredClone(extra.powerStress.surfaces[0]!),
    );
    assertInvalid(extra, "$.powerStress.surfaces");

    const reordered = copyFixture();
    [reordered.powerStress.surfaces[0], reordered.powerStress.surfaces[1]] = [
      reordered.powerStress.surfaces[1]!,
      reordered.powerStress.surfaces[0]!,
    ];
    assertInvalid(reordered, "$.powerStress.surfaces[0].id");

    const linked = copyFixture() as unknown as {
      powerStress: { causallyLinkedToIllustrativeEpoch: boolean };
    };
    linked.powerStress.causallyLinkedToIllustrativeEpoch = true;
    assertInvalid(
      linked,
      "$.powerStress.causallyLinkedToIllustrativeEpoch",
    );

    const falsePass = copyFixture() as unknown as {
      powerStress: {
        surfaces: Array<{ passesIllustrativeFloor: boolean }>;
      };
    };
    falsePass.powerStress.surfaces[0]!.passesIllustrativeFloor = false;
    assertInvalid(
      falsePass,
      "$.powerStress.surfaces[0].passesIllustrativeFloor",
    );

    assert.equal(canonical.powerStress.history, null);
    assert.equal(canonical.powerStress.uncertainty, null);
    assert.equal(canonical.powerStress.jointPathCut, null);
  });

  it("conserves model units while keeping backlog and missing state non-economic", () => {
    const epoch = canonical.shadowEpoch;
    assert.ok(
      Math.abs(
        epoch.direct + epoch.commons + epoch.unallocated - epoch.budget,
      ) < 1e-8,
    );
    assert.ok(
      Math.abs(
        epoch.clusters.reduce(
          (sum, cluster) => sum + cluster.unfundedDemand,
          0,
        ) - epoch.unfundedDemand,
      ) < 1e-8,
    );
    for (const cluster of epoch.clusters) {
      assert.equal(cluster.canonicalTreeReceipt, null);
      assert.equal(cluster.evidenceMilestone, null);
      assert.equal(cluster.extinguishedToDate, null);
      assert.equal(cluster.eligibilityLots, null);
      assert.equal(cluster.settlementState, "blocked");
      assert.equal(cluster.settlementZrn, 0);
    }

    const overspend = copyFixture();
    overspend.shadowEpoch.clusters[0]!.counterfactualFunded += 1;
    assertInvalid(overspend, "$.shadowEpoch.clusters[0]");

    const inventedExtinguishment = copyFixture() as unknown as {
      shadowEpoch: {
        clusters: Array<{ extinguishedToDate: number | null }>;
      };
    };
    inventedExtinguishment.shadowEpoch.clusters[0]!.extinguishedToDate = 0;
    assertInvalid(
      inventedExtinguishment,
      "$.shadowEpoch.clusters[0].extinguishedToDate",
    );
  });

  it("preserves the exact one-shot quarantine and successor vector", () => {
    assert.deepEqual(
      canonical.capacityShadow.trace.map((step) => [
        step.accrued,
        step.funded,
        step.live,
        step.quarantined,
        step.extinguished,
        step.replacementUsed,
        step.deadline,
      ]),
      [
        [100, 0, 100, 0, 0, 0, 10],
        [100, 30, 70, 0, 0, 0, 10],
        [100, 30, 0, 60, 10, 0, 10],
        [100, 30, 50, 10, 10, 50, 10],
      ],
    );
    for (const step of canonical.capacityShadow.trace) {
      assert.equal(
        step.accrued,
        step.funded + step.live + step.quarantined + step.extinguished,
      );
    }
    assert.equal(canonical.capacityShadow.authoritative, false);
    assert.equal(canonical.capacityShadow.networkObserved, false);
    assert.equal(canonical.capacityShadow.rewardBearing, false);
    assert.equal(canonical.capacityShadow.transferableValue, false);
    assert.equal(canonical.capacityShadow.movesFunds, false);

    const minted = copyFixture() as unknown as {
      capacityShadow: {
        settlementZrn: number;
      };
    };
    minted.capacityShadow.settlementZrn = 1;
    assertInvalid(minted, "$.capacityShadow.settlementZrn");

    const brokenPartition = copyFixture();
    brokenPartition.capacityShadow.trace[3]!.live += 1;
    assertInvalid(brokenPartition, "$.capacityShadow.trace[3].live");

    const falseCheck = copyFixture() as unknown as {
      capacityShadow: {
        checks: { exact_partition: boolean };
      };
    };
    falseCheck.capacityShadow.checks.exact_partition = false;
    assertInvalid(falseCheck, "$.capacityShadow.checks.exact_partition");
  });

  it("keeps mathematics qualification-only and the six ledgers separate", () => {
    assert.ok(
      canonical.capability.mathNodes.every(
        (node) => node.rewardEligibility === "qualification-only",
      ),
    );
    assert.equal(canonical.capability.skillUnlockCreatesReward, false);

    const levels = mathNodeLevels(canonical.capability.mathNodes);
    assert.deepEqual(
      levels.map((level) => level.map((node) => node.id)),
      [
        ["math-proofcraft@1"],
        [
          "math-algebra-finite-fields@1",
          "math-probability-information-complexity@1",
        ],
        ["math-lattices-polynomial-rings@1"],
      ],
    );

    const rewardedMath = copyFixture() as unknown as {
      capability: {
        mathNodes: Array<{ rewardEligibility: string }>;
      };
    };
    rewardedMath.capability.mathNodes[0]!.rewardEligibility =
      "sponsor-milestones";
    assertInvalid(
      rewardedMath,
      "$.capability.mathNodes[0].rewardEligibility",
    );

    const collapsedLedgers = copyFixture();
    collapsedLedgers.ledgerLanes.pop();
    assertInvalid(collapsedLedgers, "$.ledgerLanes");
  });
});

describe("local-only source boundary", () => {
  it("has no API, wallet, storage, or browser network call path", () => {
    const main = readFileSync(
      new URL("../src/power-lab/main.ts", import.meta.url),
      "utf8",
    );
    assert.doesNotMatch(
      main,
      /\b(?:fetch|XMLHttpRequest|WebSocket|localStorage|sessionStorage)\b/,
    );
    assert.doesNotMatch(
      main,
      /from\s+["'][^"']*(?:api|wallet|config)["']/,
    );

    const vite = readFileSync(
      new URL("../vite.power-lab.config.mjs", import.meta.url),
      "utf8",
    );
    assert.match(vite, /host:\s*"127\.0\.0\.1"/);
    assert.match(vite, /strictPort:\s*true/);
    assert.doesNotMatch(vite, /\bproxy\s*:/);
  });

  it("is no-index, unlinked, and guarded by a post-build exclusion check", () => {
    const labHtml = readFileSync(
      new URL("../power-lab.html", import.meta.url),
      "utf8",
    );
    assert.match(labHtml, /noindex, nofollow, noarchive/);
    assert.match(labHtml, /zerone\.local-power-reward-lab\/v1/);
    assert.match(labHtml, /NOT NETWORK-OBSERVED/);
    assert.match(labHtml, /0 ZRN/);
    assert.match(labHtml, /RELEASE CLOSED/);

    const productionHtml = readFileSync(
      new URL("../index.html", import.meta.url),
      "utf8",
    );
    assert.doesNotMatch(productionHtml, /power-lab/i);

    const packageJson = JSON.parse(
      readFileSync(new URL("../package.json", import.meta.url), "utf8"),
    ) as { scripts?: Record<string, string> };
    assert.match(
      packageJson.scripts?.build ?? "",
      /vite build && npm run check:lab-excluded/,
    );
    assert.match(
      packageJson.scripts?.["check:lab-excluded"] ?? "",
      /assert-power-lab-excluded\.mjs dist/,
    );
  });

  it("makes the production exclusion tripwire fail on a bundled lab marker", () => {
    const temporaryRoot = mkdtempSync(
      join(tmpdir(), "zerone-power-lab-exclusion-"),
    );
    const dist = join(temporaryRoot, "dist");
    const script = fileURLToPath(
      new URL("../scripts/assert-power-lab-excluded.mjs", import.meta.url),
    );
    mkdirSync(dist);
    try {
      writeFileSync(join(dist, "index.html"), "<p>production-only</p>");
      const clean = spawnSync(process.execPath, [script, dist], {
        encoding: "utf8",
      });
      assert.equal(clean.status, 0, clean.stderr);

      writeFileSync(
        join(dist, "index.html"),
        `<p>${POWER_LAB_SCHEMA}</p>`,
      );
      const contaminated = spawnSync(process.execPath, [script, dist], {
        encoding: "utf8",
      });
      assert.notEqual(contaminated.status, 0);
      assert.match(
        contaminated.stderr,
        /local power-lab marker was published/,
      );
    } finally {
      rmSync(temporaryRoot, { recursive: true, force: true });
    }
  });
});

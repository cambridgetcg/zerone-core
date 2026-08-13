import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import {
  mkdtempSync,
  readFileSync,
  rmSync,
  symlinkSync,
  writeFileSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { spawnSync } from "node:child_process";
import { describe, it } from "node:test";

import {
  FOLD_TO_FIRE_CANONICAL_SHA256,
  FOLD_TO_FIRE_MAX_BYTES,
  FOLD_TO_FIRE_RAW_SHA256,
  FoldToFireValidationError,
  canonicalJson,
  parseAndValidateConstructiveIntelligenceFoldToFire,
  validateConstructiveIntelligenceFoldToFire,
} from "./validate-constructive-intelligence-fold-to-fire.mjs";

const profileUrl = new URL(
  "../public/standards/constructive-intelligence-fold-to-fire.v0.json",
  import.meta.url,
);
const profileRaw = readFileSync(profileUrl, "utf8");
const canonical = JSON.parse(profileRaw);
const validatorPath = new URL(
  "./validate-constructive-intelligence-fold-to-fire.mjs",
  import.meta.url,
);

function copy() {
  return structuredClone(canonical);
}

function assertInvalid(operation, expectedPath) {
  assert.throws(
    operation,
    (error) =>
      error instanceof FoldToFireValidationError &&
      (expectedPath === undefined || error.path === expectedPath),
  );
}

describe("constructive-intelligence Fold-to-Fire v0", () => {
  it("validates the reviewed profile and both digest identities", () => {
    const result = parseAndValidateConstructiveIntelligenceFoldToFire(profileRaw);
    assert.deepEqual(result, {
      schema: "zerone.constructive-intelligence-fold-to-fire/v0",
      rowCount: 7,
      maximumExactSteps: 15,
      sourceCount: 9,
      sourceBindingCount: 3,
      canonicalSha256: FOLD_TO_FIRE_CANONICAL_SHA256,
      rawSha256: FOLD_TO_FIRE_RAW_SHA256,
    });
    assert.equal(
      createHash("sha256").update(profileRaw).digest("hex"),
      FOLD_TO_FIRE_RAW_SHA256,
    );
    assert.equal(
      createHash("sha256").update(canonicalJson(canonical)).digest("hex"),
      FOLD_TO_FIRE_CANONICAL_SHA256,
    );
  });

  it("pins exact upstream raw and canonical bytes", () => {
    const expected = [
      [
        "constructive-intelligence-math-frontier-v0",
        "dashboard/public/standards/constructive-intelligence-math-frontier.v0.json",
        "4fdcd54c35c69c26a28c385275688351ee2a9131702e81bacf100de8d7612456",
        "b6260de31969a56e601a5a81f1b4f7c1c68fcd34f9aa68b35ce2701c3f012503",
      ],
      [
        "constructive-intelligence-life-sciences-v0",
        "dashboard/public/standards/constructive-intelligence-life-sciences.v0.json",
        "64dc2c5b2e21dfc9697d173317254ce651dede8661993ece7b380b7e1421496e",
        "a208ea9e30a16ccfbb74f3f19298a5d3f93d7f87273b0b5aa10bf72e0e708822",
      ],
      [
        "money-karma-v1",
        "docs/constitution/money-karma-v1.json",
        "f22e62f0706971c569bb2156400b6dbeaf72a005d822b1e40c4e2691e7a98c24",
        "a41286c936d3ab83d1cbd782b119cf3b434518ba80859edfe76f0de184143b7b",
      ],
    ];
    for (const [index, [id, path, rawSha256, canonicalSha256]] of expected.entries()) {
      assert.deepEqual(canonical.sourceBindings[index], {
        ...canonical.sourceBindings[index],
        id,
        path,
        rawSha256,
        canonicalSha256,
      });
    }
  });

  it("pins all finite polynomials and independently recomputes counts", () => {
    assert.deepEqual(
      canonical.enumeration.rows.map(({ stepCount, totalWalks, activeWalks }) => [
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
    for (const row of canonical.enumeration.rows) {
      assert.equal(
        row.allByContacts.reduce((sum, value) => sum + BigInt(value), 0n),
        BigInt(row.totalWalks),
      );
      assert.equal(
        row.activeByContacts.reduce((sum, value) => sum + BigInt(value), 0n),
        BigInt(row.activeWalks),
      );
    }
  });

  it("separates the established q=1 conjecture from the bespoke weighted bridge", () => {
    assert.equal(canonical.frontierProblem.status, "ESTABLISHED_OPEN_CONJECTURE");
    assert.equal(canonical.frontierProblem.exponent, "59/32");
    assert.equal(canonical.frontierProblem.computationDoesNotProve, true);
    assert.equal(canonical.weightedBridge.status, "BESPOKE_RESEARCH_BRIDGE");
    assert.equal(
      canonical.weightedBridge.qNotEqualOneExponentTransfer,
      "OUT_OF_SCOPE_NOT_CLAIMED",
    );
    assert.equal(canonical.weightedBridge.noveltyAuditRequired, true);
    assert.equal(canonical.weightedBridge.notClaimedAsEstablishedOpenProblem, true);
  });

  it("keeps every effect and authority switch false", () => {
    for (const key of Object.keys(canonical.releaseBoundary)) {
      const mutation = copy();
      mutation.releaseBoundary[key] = true;
      assertInvalid(
        () => validateConstructiveIntelligenceFoldToFire(mutation),
        `$.releaseBoundary.${key}`,
      );
    }
    assert.deepEqual(canonical.economics, {
      effect: "NONE",
      amount: "0",
      denom: null,
      rewardMultiplier: false,
      escrowReference: null,
    });
  });

  it("rejects weakened science, proof, protein, and KARMA walls", () => {
    for (let index = 0; index < canonical.nonImplicationWalls.length; index += 1) {
      const mutation = copy();
      mutation.nonImplicationWalls[index].doesNotEstablish.pop();
      assertInvalid(
        () => validateConstructiveIntelligenceFoldToFire(mutation),
        "$.nonImplicationWalls",
      );
    }
  });

  it("rejects changed counts, equations, conjecture status, and source URLs", () => {
    const count = copy();
    count.enumeration.rows[5].activeWalks = "4114";
    assertInvalid(
      () => validateConstructiveIntelligenceFoldToFire(count),
      "$.enumeration.rows[5]",
    );

    const equation = copy();
    equation.weightedBridge.effectiveFlux = "J_n(q)=A_n(q)/Z_n(q)";
    assertInvalid(
      () => validateConstructiveIntelligenceFoldToFire(equation),
      "$.weightedBridge",
    );

    const solved = copy();
    solved.frontierProblem.status = "PROVED";
    assertInvalid(
      () => validateConstructiveIntelligenceFoldToFire(solved),
      "$.frontierProblem",
    );

    const source = copy();
    source.sources[0].url = "https://doi.org/10.1214/14-AOP994";
    assertInvalid(
      () => validateConstructiveIntelligenceFoldToFire(source),
      "$.sources",
    );
  });

  it("rejects missing, reordered, and unknown fields", () => {
    const missing = copy();
    delete missing.economics.effect;
    assertInvalid(() => validateConstructiveIntelligenceFoldToFire(missing), "$.economics");

    const unknown = copy();
    unknown.releaseBoundary.activatesKarma = false;
    assertInvalid(
      () => validateConstructiveIntelligenceFoldToFire(unknown),
      "$.releaseBoundary",
    );

    const reordered = copy();
    [reordered.sources[0], reordered.sources[1]] = [
      reordered.sources[1],
      reordered.sources[0],
    ];
    assertInvalid(() => validateConstructiveIntelligenceFoldToFire(reordered), "$.sources");
  });

  it("rejects duplicate keys, malformed JSON, excessive nesting, and oversized input", () => {
    const duplicate = profileRaw.replace(
      '"schema": "zerone.constructive-intelligence-fold-to-fire/v0",',
      '"schema": "zerone.constructive-intelligence-fold-to-fire/v0",\n  "schema": "zerone.constructive-intelligence-fold-to-fire/v0",',
    );
    assertInvalid(
      () =>
        parseAndValidateConstructiveIntelligenceFoldToFire(duplicate, {
          pinRawDigest: false,
        }),
      "$.schema",
    );
    assertInvalid(
      () =>
        parseAndValidateConstructiveIntelligenceFoldToFire("{", {
          pinRawDigest: false,
        }),
      "$",
    );
    assertInvalid(
      () =>
        parseAndValidateConstructiveIntelligenceFoldToFire(
          `${"[".repeat(17)}0${"]".repeat(17)}`,
          { pinRawDigest: false },
        ),
      "$",
    );
    assertInvalid(
      () =>
        parseAndValidateConstructiveIntelligenceFoldToFire(
          " ".repeat(FOLD_TO_FIRE_MAX_BYTES + 1),
          { pinRawDigest: false },
        ),
      "$",
    );
  });

  it("CLI accepts only a regular non-symlink file and is offline", () => {
    const temporary = mkdtempSync(join(tmpdir(), "fold-to-fire-validator-"));
    try {
      const link = join(temporary, "profile-link.json");
      symlinkSync(profileUrl, link);
      const linked = spawnSync(process.execPath, [validatorPath.pathname, link], {
        encoding: "utf8",
      });
      assert.equal(linked.status, 1);
      assert.match(linked.stderr, /regular non-symlink file/);

      const oversized = join(temporary, "oversized.json");
      writeFileSync(oversized, " ".repeat(FOLD_TO_FIRE_MAX_BYTES + 1));
      const tooLarge = spawnSync(process.execPath, [validatorPath.pathname, oversized], {
        encoding: "utf8",
      });
      assert.equal(tooLarge.status, 1);
      assert.match(tooLarge.stderr, /exceeds 65536 bytes/);

      const success = spawnSync(process.execPath, [validatorPath.pathname, profileUrl.pathname], {
        encoding: "utf8",
        env: {
          PATH: process.env.PATH,
          NO_PROXY: "*",
          HTTP_PROXY: "http://127.0.0.1:1",
          HTTPS_PROXY: "http://127.0.0.1:1",
        },
      });
      assert.equal(success.status, 0, success.stderr);
      assert.match(success.stdout, /PASS \(7 exact finite rows through n=15/);
    } finally {
      rmSync(temporary, { recursive: true, force: true });
    }
  });
});

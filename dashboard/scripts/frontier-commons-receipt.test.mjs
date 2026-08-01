import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";

import * as receiptApi from "./validate-frontier-commons-receipt.mjs";
import {
  EXPECTED_FRONTIER_COMMONS_RECEIPT_SHA256,
  EXPECTED_FRONTIER_COMMONS_SHA256,
  FRONTIER_COMMONS_RECEIPT_MAX_BYTES,
  FRONTIER_COMMONS_RECEIPT_SCHEMA,
  FrontierCommonsReceiptValidationError,
  parseAndValidateFrontierCommonsReceipt,
} from "./validate-frontier-commons-receipt.mjs";

const commonsRaw = readFileSync(
  new URL(
    "../public/standards/frontier-commons-participation.v0.json",
    import.meta.url,
  ),
  "utf8",
);
const receiptRaw = readFileSync(
  new URL(
    "../public/standards/frontier-commons-self-receipt.v0.json",
    import.meta.url,
  ),
  "utf8",
);
const canonicalReceipt = JSON.parse(receiptRaw);
const CANONICAL_COMMONS_SHA256 =
  "c642e09f46dcaf0a1960f140996969688b936a806c2d33ebf0b6c3efa6a70d2a";
const CANONICAL_RECEIPT_SHA256 =
  "3e36185eaba207d1defec4b488f9070683f184fffe7ee5b6f487ac1e15c95b92";
const EVALUATION_DIGEST_KEYS = [
  "protocolDigest",
  "threatModelDigest",
  "fixtureDigest",
  "acceptancePolicyDigest",
  "environmentDigest",
  "evidenceDigest",
  "challengePolicyDigest",
];

function mutatedReceipt(mutator) {
  const receipt = structuredClone(canonicalReceipt);
  mutator(receipt);
  return `${JSON.stringify(receipt, null, 2)}\n`;
}

function assertInvalid(operation, path) {
  assert.throws(
    operation,
    (error) =>
      error instanceof FrontierCommonsReceiptValidationError &&
      (path === undefined || error.path === path),
  );
}

describe("Frontier Commons FC-0.1 self-receipt", () => {
  it("validates only the exact canonical FC-0 and receipt bytes", () => {
    assert.equal(
      createHash("sha256").update(commonsRaw).digest("hex"),
      CANONICAL_COMMONS_SHA256,
    );
    assert.equal(
      createHash("sha256").update(receiptRaw).digest("hex"),
      CANONICAL_RECEIPT_SHA256,
    );
    assert.equal(EXPECTED_FRONTIER_COMMONS_SHA256, CANONICAL_COMMONS_SHA256);
    assert.equal(
      EXPECTED_FRONTIER_COMMONS_RECEIPT_SHA256,
      CANONICAL_RECEIPT_SHA256,
    );
    assert.deepEqual(
      parseAndValidateFrontierCommonsReceipt(commonsRaw, receiptRaw),
      {
        schema: FRONTIER_COMMONS_RECEIPT_SCHEMA,
        receiptKind: "ZERONE_SELF_DOGFOOD",
        subjectDigest: CANONICAL_COMMONS_SHA256,
        receiptDigest: CANONICAL_RECEIPT_SHA256,
        result: "INCONCLUSIVE",
        relationCount: 0,
        temporalStatus: "NOT_EVALUATED",
      },
    );

    assertInvalid(
      () => parseAndValidateFrontierCommonsReceipt(`${commonsRaw}\n`, receiptRaw),
      "$commons",
    );
    assertInvalid(
      () => parseAndValidateFrontierCommonsReceipt(commonsRaw, `${receiptRaw}\n`),
      "$receipt",
    );
  });

  it("makes FC-0 the sole exact subject", () => {
    assert.equal(canonicalReceipt.subject.length, 1);
    assert.equal(
      canonicalReceipt.subject[0].name,
      "frontier-commons-participation.v0.json",
    );
    assert.equal(
      canonicalReceipt.subject[0].digest.sha256,
      CANONICAL_COMMONS_SHA256,
    );

    assertInvalid(
      () =>
        parseAndValidateFrontierCommonsReceipt(
          commonsRaw,
          mutatedReceipt((receipt) => {
            receipt.subject.push(structuredClone(receipt.subject[0]));
          }),
        ),
      "$receipt.subject",
    );
    assertInvalid(
      () =>
        parseAndValidateFrontierCommonsReceipt(
          commonsRaw,
          mutatedReceipt((receipt) => {
            receipt.subject[0].name = "frontier-commons-self-receipt.v0.json";
          }),
        ),
      "$receipt.subject[0].name",
    );
    assertInvalid(
      () =>
        parseAndValidateFrontierCommonsReceipt(
          commonsRaw,
          mutatedReceipt((receipt) => {
            receipt.subject[0].digest.sha256 = "0".repeat(64);
          }),
        ),
      "$receipt.subject[0].digest.sha256",
    );
  });

  it("uses the reviewed generic receipt schema and predicate type", () => {
    assert.equal(
      canonicalReceipt.predicate.schema,
      "zerone.frontier-evaluation-receipt/v0",
    );
    assert.equal(
      canonicalReceipt.predicateType,
      "https://github.com/cambridgetcg/zerone-core/blob/main/docs/specs/frontier-commons-receipt-v0.md#fc-01-self-receipt-profile",
    );

    assertInvalid(
      () =>
        parseAndValidateFrontierCommonsReceipt(
          commonsRaw,
          mutatedReceipt((receipt) => {
            receipt.predicate.schema = "zerone.frontier-commons-self-receipt/v0";
          }),
        ),
      "$receipt.predicate.schema",
    );
    assertInvalid(
      () =>
        parseAndValidateFrontierCommonsReceipt(
          commonsRaw,
          mutatedReceipt((receipt) => {
            receipt.predicateType = "https://example.invalid/predicate";
          }),
        ),
      "$receipt.predicateType",
    );
  });

  it("rejects public, signed, or favorable receipt claims", () => {
    for (const [path, mutate] of [
      ["$receipt.predicate.receiptKind", (receipt) => {
        receipt.predicate.receiptKind = "PUBLIC_EVALUATION";
      }],
      ["$receipt.predicate.assurance", (receipt) => {
        receipt.predicate.assurance = "SIGNED_VERIFIED";
      }],
      ["$receipt.predicate.evaluation.result", (receipt) => {
        receipt.predicate.evaluation.result = "REPRODUCED";
      }],
    ]) {
      assertInvalid(
        () =>
          parseAndValidateFrontierCommonsReceipt(
            commonsRaw,
            mutatedReceipt(mutate),
          ),
        path,
      );
    }
  });

  it("requires all seven evaluation-material digests to be exactly null", () => {
    for (const key of EVALUATION_DIGEST_KEYS) {
      assert.equal(canonicalReceipt.predicate.evaluation[key], null);
      assertInvalid(
        () =>
          parseAndValidateFrontierCommonsReceipt(
            commonsRaw,
            mutatedReceipt((receipt) => {
              receipt.predicate.evaluation[key] = `sha256:${"a".repeat(64)}`;
            }),
          ),
        `$receipt.predicate.evaluation.${key}`,
      );
    }
  });

  it("keeps privacy, network, economic, governance, and authority effects false", () => {
    for (const key of Object.keys(canonicalReceipt.predicate.privacy)) {
      assertInvalid(
        () =>
          parseAndValidateFrontierCommonsReceipt(
            commonsRaw,
            mutatedReceipt((receipt) => {
              receipt.predicate.privacy[key] = true;
            }),
          ),
        `$receipt.predicate.privacy.${key}`,
      );
    }
    for (const key of Object.keys(canonicalReceipt.predicate.effects)) {
      assertInvalid(
        () =>
          parseAndValidateFrontierCommonsReceipt(
            commonsRaw,
            mutatedReceipt((receipt) => {
              receipt.predicate.effects[key] = true;
            }),
          ),
        `$receipt.predicate.effects.${key}`,
      );
    }
  });

  it("requires empty relations and exact correction semantics", () => {
    for (const digest of ["a".repeat(64), "0".repeat(64)]) {
      assertInvalid(
        () =>
          parseAndValidateFrontierCommonsReceipt(
            commonsRaw,
            mutatedReceipt((receipt) => {
              receipt.predicate.evaluation.relations = [
                {
                  type: "REPLICATES",
                  receiptDigest: `sha256:${digest}`,
                },
              ];
            }),
          ),
        "$receipt.predicate.evaluation.relations",
      );
    }

    for (const [path, mutate] of [
      ["$receipt.predicate.correction.mode", (receipt) => {
        receipt.predicate.correction.mode = "DELETE_HISTORY";
      }],
      ["$receipt.predicate.correction.futureRelianceMayBeWithdrawn", (receipt) => {
        receipt.predicate.correction.futureRelianceMayBeWithdrawn = false;
      }],
      ["$receipt.predicate.correction.historicalPublicCopiesDeletable", (receipt) => {
        receipt.predicate.correction.historicalPublicCopiesDeletable = true;
      }],
      ["$receipt.predicate.correction.exitAffectsUnrelatedAccess", (receipt) => {
        receipt.predicate.correction.exitAffectsUnrelatedAccess = true;
      }],
    ]) {
      assertInvalid(
        () =>
          parseAndValidateFrontierCommonsReceipt(
            commonsRaw,
            mutatedReceipt(mutate),
          ),
        path,
      );
    }
  });

  it("reports deterministic temporal status only for an explicit as-of date", () => {
    for (const [asOf, expected] of [
      [undefined, "NOT_EVALUATED"],
      ["2026-07-31", "NOT_YET_CREATED"],
      ["2026-08-01", "CURRENT"],
      ["2026-08-15", "REVIEW_DUE"],
      ["2026-11-01", "REVIEW_DUE"],
      ["2026-11-02", "EXPIRED"],
    ]) {
      assert.equal(
        parseAndValidateFrontierCommonsReceipt(
          commonsRaw,
          receiptRaw,
          asOf === undefined ? {} : { asOf },
        ).temporalStatus,
        expected,
      );
    }

    assertInvalid(
      () =>
        parseAndValidateFrontierCommonsReceipt(commonsRaw, receiptRaw, {
          asOf: "2026-02-30",
        }),
      "$asOf",
    );
    assertInvalid(
      () =>
        parseAndValidateFrontierCommonsReceipt(
          commonsRaw,
          mutatedReceipt((receipt) => {
            receipt.predicate.evaluation.evidenceCutoffOn = "2026-08-02";
          }),
        ),
      "$receipt.predicate.evaluation.evidenceCutoffOn",
    );
  });

  it("exposes no object-taking receipt validator or receipt getter path", () => {
    assert.equal("validateFrontierCommonsReceipt" in receiptApi, false);

    let getterReads = 0;
    const objectReceipt = {};
    Object.defineProperty(objectReceipt, "predicate", {
      enumerable: true,
      get() {
        getterReads += 1;
        return canonicalReceipt.predicate;
      },
    });
    assertInvalid(
      () => parseAndValidateFrontierCommonsReceipt(commonsRaw, objectReceipt),
      "$receipt",
    );
    assert.equal(getterReads, 0);
    assertInvalid(
      () => parseAndValidateFrontierCommonsReceipt(JSON.parse(commonsRaw), receiptRaw),
      "$commons",
    );
  });

  it("rejects malformed, duplicate-key, oversized, deep, and drifted receipt JSON", () => {
    assertInvalid(
      () => parseAndValidateFrontierCommonsReceipt(commonsRaw, "{"),
      "$receipt",
    );

    const duplicate = receiptRaw.replace(
      '"receiptKind": "ZERONE_SELF_DOGFOOD",',
      '"receiptKind": "ZERONE_SELF_DOGFOOD",\n    "receiptKind": "PUBLIC_EVALUATION",',
    );
    assertInvalid(
      () => parseAndValidateFrontierCommonsReceipt(commonsRaw, duplicate),
      "$receipt.predicate.receiptKind",
    );
    assertInvalid(
      () =>
        parseAndValidateFrontierCommonsReceipt(
          commonsRaw,
          " ".repeat(FRONTIER_COMMONS_RECEIPT_MAX_BYTES + 1),
        ),
      "$receipt",
    );

    const deep = `${"[".repeat(33)}0${"]".repeat(33)}`;
    assertInvalid(
      () => parseAndValidateFrontierCommonsReceipt(commonsRaw, deep),
      "$receipt",
    );

    assertInvalid(
      () =>
        parseAndValidateFrontierCommonsReceipt(
          commonsRaw,
          mutatedReceipt((receipt) => {
            receipt.participants = ["example-lab"];
          }),
        ),
      "$receipt",
    );
    assertInvalid(
      () =>
        parseAndValidateFrontierCommonsReceipt(
          commonsRaw,
          `${JSON.stringify(
            Object.fromEntries(Object.entries(canonicalReceipt).reverse()),
            null,
            2,
          )}\n`,
        ),
      "$receipt",
    );
  });
});

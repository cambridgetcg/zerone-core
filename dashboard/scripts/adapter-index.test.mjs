import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import {
  AdapterIndexValidationError,
  parseAndValidateAdapterIndex,
  validateAdapterIndex,
} from "./validate-adapter-index.mjs";

const canonical = JSON.parse(
  readFileSync(
    new URL("../public/standards/adapter-index.v1.json", import.meta.url),
    "utf8",
  ),
);

function copyIndex() {
  return structuredClone(canonical);
}

function entry(index, id) {
  const found = index.entries.find((candidate) => candidate.id === id);
  assert.ok(found, `missing fixture entry ${id}`);
  return found;
}

function assertInvalid(operation, path) {
  assert.throws(
    operation,
    (error) =>
      error instanceof AdapterIndexValidationError &&
      (path === undefined || error.path === path),
  );
}

describe("adapter capability index", () => {
  it("validates the published static index", () => {
    assert.deepEqual(validateAdapterIndex(copyIndex()), {
      schema: "zerone.adapter-index/v1",
      entryCount: 5,
    });
  });

  it("rejects duplicate or unsorted IDs", () => {
    const duplicate = copyIndex();
    duplicate.entries[1].id = duplicate.entries[0].id;
    assertInvalid(() => validateAdapterIndex(duplicate), "$.entries");

    const unsorted = copyIndex();
    [unsorted.entries[0], unsorted.entries[1]] = [
      unsorted.entries[1],
      unsorted.entries[0],
    ];
    assertInvalid(() => validateAdapterIndex(unsorted), "$.entries");
  });

  it("rejects an endpoint field because this is not a discovery document", () => {
    const index = copyIndex();
    entry(index, "a2a-agent-card").serviceEndpoint = "https://example.com/a2a";
    assertInvalid(
      () => validateAdapterIndex(index),
      "$.entries[0].serviceEndpoint",
    );
  });

  it("prevents A2A and x402 entries from advertising runtime support", () => {
    const a2a = copyIndex();
    entry(a2a, "a2a-agent-card").capabilities.push("a2a-jsonrpc");
    assertInvalid(
      () => validateAdapterIndex(a2a),
      "$.entries[0].capabilities",
    );

    const x402 = copyIndex();
    const payment = entry(x402, "x402-zerone");
    payment.status = "implemented-source";
    payment.availability = "external-service-required";
    payment.capabilities.push("x402-exact");
    assertInvalid(
      () => validateAdapterIndex(x402),
      "$.entries[4].status",
    );
  });

  it("keeps every release-level consensus, network, and deployment assertion false", () => {
    for (const key of [
      "addsConsensusBehavior",
      "activatesAdapters",
      "performsNetworkRequests",
      "assertsLiveDeployment",
    ]) {
      const index = copyIndex();
      index.releaseBoundary[key] = true;
      assertInvalid(
        () => validateAdapterIndex(index),
        `$.releaseBoundary.${key}`,
      );
    }
  });

  it("rejects live deployment claims and unsafe standard URLs", () => {
    const deployment = copyIndex();
    entry(
      deployment,
      "training-provenance-in-toto-v1",
    ).trust.liveDeploymentVerified = true;
    assertInvalid(
      () => validateAdapterIndex(deployment),
      "$.entries[3].trust.liveDeploymentVerified",
    );

    for (const url of [
      "http://example.com/spec",
      "https://example.com/spec",
      "https://user:pass@example.com/spec",
      "https://example.com/spec?version=1",
      "https://example.com/spec#section",
    ]) {
      const index = copyIndex();
      entry(index, "a2a-agent-card").standards[0].specification = url;
      assertInvalid(
        () => validateAdapterIndex(index),
        "$.entries[0].standards[0].specification",
      );
    }
  });

  it("rejects repository path traversal and unknown schema fields", () => {
    const pathTraversal = copyIndex();
    entry(pathTraversal, "a2a-agent-card").repositoryReferences[0] =
      "../outside.md";
    assertInvalid(
      () => validateAdapterIndex(pathTraversal),
      "$.entries[0].repositoryReferences[0]",
    );

    const missing = copyIndex();
    entry(missing, "a2a-agent-card").repositoryReferences[0] =
      "docs/definitely-missing.md";
    assertInvalid(
      () => validateAdapterIndex(missing),
      "$.entries[0].repositoryReferences[0]",
    );

    const unknown = copyIndex();
    unknown.live = true;
    assertInvalid(() => validateAdapterIndex(unknown), "$.live");
  });

  it("bounds the raw static document before JSON parsing", () => {
    assertInvalid(() => parseAndValidateAdapterIndex(" ".repeat(65_537)), "$");
    assertInvalid(() => parseAndValidateAdapterIndex("{"), "$");
  });
});

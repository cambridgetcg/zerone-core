import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";
import { queryClaimHistory, type ClaimHistoryRpc } from "../src/claim-history";
import { QueryClaimHistoryRequest, QueryClaimHistoryResponse } from "../src/generated/zerone/knowledge/v1/claim_history";

const fixture = JSON.parse(readFileSync(new URL("./fixtures/claim-history-response.json", import.meta.url), "utf8"));
const golden = Uint8Array.from(Buffer.from(fixture.protobufHex, "hex"));
const response = () => QueryClaimHistoryResponse.decode(golden);
const rpc = (bytes: Uint8Array = golden): ClaimHistoryRpc => ({ request: async () => bytes });
const encoded = (value: QueryClaimHistoryResponse) => QueryClaimHistoryResponse.encode(value).finish();

test("claim history reads the Go-produced protobuf fixture without losing retained fields", async () => {
  let requests = 0;
  const observed = await queryClaimHistory({ request: async (service, method, bytes) => {
    requests++;
    assert.equal(service, "zerone.knowledge.v1.Query");
    assert.equal(method, "ClaimHistory");
    assert.deepEqual(QueryClaimHistoryRequest.decode(bytes), { id: fixture.claimId, atBlockHeight: BigInt(fixture.blockHeight) });
    return golden;
  } }, fixture.claimId, { expectedChainId: fixture.chainId, expectedHeight: BigInt(fixture.blockHeight) });
  assert.equal(requests, 1);
  assert.deepEqual(encoded(observed), golden);
  assert.equal(observed.blockHeight, 9_007_199_254_740_993n);
  assert.equal(observed.record?.claim?.argumentText, "Reason retained verbatim.");
  assert.deepEqual(observed.record?.claim?.evidenceIds, ["test-evidence"]);
  const review = observed.record?.rounds[0]?.reveals[0];
  assert.equal(review?.confidence, 765432n);
  assert.deepEqual(review?.attestation, { methodId: "test-method", reason: "February 30 is invalid.", scope: "Calendar validity", evidenceIds: ["test-counterexample"] });
  assert.deepEqual(Array.from(review?.salt ?? []), [1, 2, 3]);
  assert.deepEqual(observed.record?.facts[0]?.outgoingRelations[0], {
    sourceFactId: "fact-root", targetFactId: "fact-neighbor", relation: 1,
    createdAtBlock: 9_007_199_254_740_992n, creator: "test-author", inference: 1,
    inferenceStrengthBps: 654321n, methodId: "test-method",
  });
  assert.deepEqual(observed.record?.missingRoundIds, ["historical-missing-round"]);
  assert.equal(observed.relatedClaims[0]?.record?.claim?.argumentText, "Contradiction reason");
});

test("opaque UTF-8 claim IDs are not trimmed and missing historical claim rows stay absent", async () => {
  const value = response();
  value.record!.claimId = " claim/雪 ";
  value.record!.claim = undefined;
  value.record!.rounds[0]!.claimId = value.record!.claimId;
  value.record!.facts[0]!.fact!.claimId = value.record!.claimId;
  const observed = await queryClaimHistory({ request: async (_service, _method, bytes) => {
    assert.deepEqual(QueryClaimHistoryRequest.decode(bytes), { id: " claim/雪 ", atBlockHeight: 0n });
    return encoded(value);
  } }, " claim/雪 ");
  assert.equal(observed.record?.claim, undefined);
  assert.deepEqual(observed.record?.rounds, value.record?.rounds);
});

test("invalid request identities and resource options fail before the transport is called", async () => {
  let requests = 0;
  const transport: ClaimHistoryRpc = { request: async () => { requests++; return golden; } };
  for (const id of ["", "x".repeat(257), "雪".repeat(86), "\ud800"]) {
    await assert.rejects(queryClaimHistory(transport, id), /claim ID/);
  }
  for (const expectedHeight of [0n, -1n, 1n << 64n]) {
    await assert.rejects(queryClaimHistory(transport, fixture.claimId, { expectedHeight }), /height/);
  }
  for (const maximumResponseBytes of [0, -1, 1.5, 8 * 1024 * 1024 + 1, NaN]) {
    await assert.rejects(queryClaimHistory(transport, fixture.claimId, { maximumResponseBytes }), /bytes/);
  }
  await assert.rejects(queryClaimHistory(transport, fixture.claimId, { expectedChainId: "" }), /chain ID/);
  assert.equal(requests, 0);
});

test("response limits and unsupported protobuf cannot silently become a partial observation", async () => {
  await assert.rejects(queryClaimHistory(rpc(), fixture.claimId, { maximumResponseBytes: golden.length - 1 }), /byte limit/);
  await assert.rejects(queryClaimHistory(rpc(new Uint8Array()), fixture.claimId), /empty/);
  await assert.rejects(queryClaimHistory(rpc(Uint8Array.from([0xff])), fixture.claimId));
  // Unknown field 100 would otherwise be dropped by the generated decoder.
  await assert.rejects(queryClaimHistory(rpc(Uint8Array.from([...golden, 0xa0, 0x06, 0x01])), fixture.claimId), /noncanonical/);
  // A duplicate chain_id would otherwise silently replace the first value.
  const chain = new TextEncoder().encode(fixture.chainId);
  await assert.rejects(queryClaimHistory(rpc(Uint8Array.from([...golden, 0x0a, chain.length, ...chain])), fixture.claimId), /noncanonical/);
  const failure = new Error("historical state unavailable");
  await assert.rejects(queryClaimHistory({ request: async () => { throw failure; } }, fixture.claimId), error => error === failure);
});

test("chain, height and retained-record identity mismatches refuse the whole response", async () => {
  await assert.rejects(queryClaimHistory(rpc(), fixture.claimId, { expectedChainId: "other-chain" }), /chain ID mismatch/);
  await assert.rejects(queryClaimHistory(rpc(), fixture.claimId, { expectedHeight: 1n }), /height mismatch/);
  const changes: ((value: QueryClaimHistoryResponse) => void)[] = [
    value => { value.record = undefined; },
    value => { value.chainId = ""; },
    value => { value.record!.claimId = "wrong-root"; value.record!.claim = undefined; value.record!.rounds = []; value.record!.facts = []; },
    value => { value.record!.claim!.id = "other-claim"; },
    value => { value.record!.rounds[0]!.claimId = "other-claim"; },
    value => { value.record!.rounds.push(value.record!.rounds[0]!); },
    value => { value.record!.missingRoundIds.push(value.record!.rounds[0]!.id); },
    value => { value.record!.facts[0]!.fact!.claimId = "other-claim"; },
    value => { value.record!.facts[0]!.outgoingRelations[0]!.sourceFactId = "other-fact"; },
    value => { value.relatedClaims.push(value.relatedClaims[0]!); },
    value => { value.relatedClaims[0]!.links[0]!.field = "invented-link"; },
  ];
  for (const change of changes) {
    const value = response(); change(value);
    await assert.rejects(queryClaimHistory(rpc(encoded(value)), fixture.claimId));
  }
});

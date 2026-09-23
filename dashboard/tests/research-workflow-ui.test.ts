import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import { buildDevelopmentProfile, DEVELOPMENT_GATEWAY, type DevelopmentPublication } from "../development-profile";
import { researchReviewPage } from "../research-review-page";
import { readClaimHistory } from "../src/development-reader";

const publication = JSON.parse(readFileSync(new URL("../development-publication.json", import.meta.url), "utf8")) as DevelopmentPublication;
const descriptor = readFileSync(new URL("../../deploy/networks/zerone-dev-1/network.json", import.meta.url));
const predecessor = createHash("sha256").update(readFileSync(new URL("../../deploy/networks/zerone-dev-1/network-knowledge-10.json", import.meta.url))).digest("hex");
const pins = { descriptor: publication.descriptor_sha256, genesis: publication.genesis_sha256, rpcGenesis: publication.rpc_genesis_sha256, source: publication.runtime_source_commit, binary: publication.binary_sha256 };
const id = "ab".repeat(16);
const history = { chain_id: "zerone-dev-1", block_height: "900001", record: { claim_id: id, claim: { id }, rounds: [], facts: [] }, related_claims: [] };
const body = `${JSON.stringify(history)}\n`;
const json = (value: string | Uint8Array) => new Response(typeof value === "string" ? value : Uint8Array.from(value).buffer, { headers: { "Content-Type": "application/json" } });

describe("shared claim observation reads", () => {
  it("verifies the descriptor before reading a claim and preserves exact history bytes", async () => {
    const calls: string[] = [];
    const result = await readClaimHistory(id, pins, new AbortController().signal, async (url, init) => {
      calls.push(String(url));
      assert.equal(init?.method, "GET"); assert.equal(init?.credentials, "omit"); assert.equal(init?.redirect, "error");
      return json(calls.length === 1 ? descriptor : body);
    });
    assert.deepEqual(calls, [`${DEVELOPMENT_GATEWAY}/network.json`, `${DEVELOPMENT_GATEWAY}/claims/${id}`]);
    assert.equal(result.historyUtf8, body); assert.deepEqual(result.history, history);
  });
  it("never asks for claim history after a descriptor mismatch or cancellation", async () => {
    let calls = 0;
    await assert.rejects(readClaimHistory(id, pins, new AbortController().signal, async () => { calls++; return json("{}"); }), /descriptor differs/u);
    assert.equal(calls, 1);
    const controller = new AbortController(); calls = 0;
    await assert.rejects(readClaimHistory(id, pins, controller.signal, async () => { calls++; controller.abort(); return json(descriptor); }), /cancelled/u);
    assert.equal(calls, 1);
  });
  it("refuses identity mismatch and non-JSON history instead of returning a partial snapshot", async () => {
    let calls = 0;
    await assert.rejects(readClaimHistory(id, pins, new AbortController().signal, async () => ++calls === 1 ? json(descriptor) : json(JSON.stringify({ ...history, chain_id: "zerone-1" }))), /wrong chain/u);
    calls = 0;
    await assert.rejects(readClaimHistory(id, pins, new AbortController().signal, async () => ++calls === 1 ? json(descriptor) : new Response("<html>unavailable</html>")), /JSON/u);
  });
});

describe("research workflow publication injection", () => {
  const source = "ab".repeat(20), page = readFileSync(new URL("../research/review/index.html", import.meta.url), "utf8");
  it("injects current and preserved predecessor pins from the published build profile", () => {
    const html = researchReviewPage(page, buildDevelopmentProfile(source, publication), predecessor);
    assert.ok(html.includes(`data-descriptor-sha256="${publication.descriptor_sha256}"`));
    assert.ok(html.includes(`data-predecessor-descriptor-sha256="${predecessor}"`));
    assert.ok(html.includes(`data-genesis-sha256="${publication.genesis_sha256}"`));
    assert.ok(html.includes(`tree/${source}/tools/research-review`));
    assert.doesNotMatch(html, /RESEARCH_DEVELOPMENT_PINS|tree\/main\/tools\/research-review/u);
  });
  it("provides no read configuration without a verified publication", () => {
    const html = researchReviewPage(page, buildDevelopmentProfile(source));
    assert.doesNotMatch(html, /id="research-development-reader"|data-descriptor-sha256/u);
    assert.match(html, /id="workflow-read"[^>]*disabled/u);
    assert.throws(() => researchReviewPage(page, buildDevelopmentProfile(source), predecessor));
    assert.throws(() => researchReviewPage(page, buildDevelopmentProfile(source, publication), "not-a-pin"));
  });
});

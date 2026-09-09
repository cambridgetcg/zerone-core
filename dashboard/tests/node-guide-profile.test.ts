import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, it } from "node:test";

import { buildNodeGuideProfile } from "../node-guide-profile";

const inputs = {
  sourceCommit: "74e2bbac317dacb1695ed8aed20830310617ea7a",
  helperSha256: "a1".repeat(32),
};
const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));

describe("node guide source and network boundaries", () => {
  it("refuses moving references and command/URL injection in build pins", () => {
    for (const sourceCommit of [
      "main", "74e2bbac", "0".repeat(40), inputs.sourceCommit.toUpperCase(),
      `${inputs.sourceCommit}\nmake install`, `../${inputs.sourceCommit}`,
    ]) {
      assert.throws(() => buildNodeGuideProfile({ ...inputs, sourceCommit }));
    }
    for (const helperSha256 of [
      "", "0".repeat(64), "g".repeat(64), "A".repeat(64),
      `${inputs.helperSha256}; curl https://invalid.test`,
    ]) {
      assert.throws(() => buildNodeGuideProfile({ ...inputs, helperSha256 }));
    }
  });

  it("keeps executable setup local and the legacy network read-only in this guide", () => {
    const guide = buildNodeGuideProfile(inputs);
    assert.notEqual(guide.local.chainId, guide.live.chainId);
    assert.match(guide.local.chainId, /^zerone-local-/u);
    assert.equal(guide.local.liveNetworkConnection, false);
    for (const endpoint of Object.values(guide.local.endpoints)) {
      if (typeof endpoint === "string") {
        assert.equal(new URL(endpoint).hostname, "127.0.0.1");
      }
    }
    const commands = guide.local.steps.map((step) => step.command).join("\n");
    assert.doesNotMatch(commands, /zerone-1\b|169\.155\.|fly\.dev|create-validator|unsafe-reset|rm -rf/u);
    assert.doesNotMatch(commands, /curl[^\n]*\|\s*(?:sh|bash)/u);
    for (const step of guide.local.steps.filter((step) => ["init", "start", "status"].includes(step.id))) {
      assert.match(step.command, /--home "\$HOME\/zerone-local"/u);
    }
    assert.equal(guide.live.role, "gateway-reader");
    assert.equal(guide.live.replicaInstallation.availability, "not-published");
    assert.equal(guide.live.validatorJoining.availability, "not-open");
    assert.equal(guide.live.newAccountAdmission.availability, "paused");
    assert.equal(guide.live.newAccountAdmission.websiteSignup, false);
    assert.ok(guide.live.reads.every((read) => read.method === "GET"));
    assert.equal(guide.boundaries.liveNetworkMutationBySetup, false);
    assert.equal(guide.boundaries.localSetupWritesStateAndCreatesTestKeys, true);
    assert.equal(guide.boundaries.readingGuideRequestsWallet, false);
  });

  it("binds helper integrity and source links to the supplied immutable checkout", () => {
    const guide = buildNodeGuideProfile(inputs);
    const helper = guide.source.localNodeHelper;
    assert.equal(helper.sha256, inputs.helperSha256);
    assert.match(helper.rawUrl, new RegExp(`/${inputs.sourceCommit}/scripts/local-node\\.py$`, "u"));
    const clone = guide.local.steps.find((step) => step.id === "source");
    const verify = guide.local.steps.find((step) => step.id === "verify-helper");
    assert.ok(clone && verify);
    assert.ok(clone.command.includes(`git checkout --detach ${inputs.sourceCommit}`));
    assert.ok(verify.command.includes(inputs.helperSha256));
    assert.ok(verify.command.includes(helper.path));

    const strings: string[] = [];
    function visit(value: unknown): void {
      if (typeof value === "string") strings.push(value);
      else if (Array.isArray(value)) value.forEach(visit);
      else if (value && typeof value === "object") Object.values(value).forEach(visit);
    }
    visit(guide);
    const sourceLinks = strings.filter((value) => value.startsWith("https://github.com/cambridgetcg/zerone-core/blob/"));
    assert.ok(sourceLinks.length >= 5);
    for (const link of sourceLinks) {
      const prefix = `https://github.com/cambridgetcg/zerone-core/blob/${inputs.sourceCommit}/`;
      assert.ok(link.startsWith(prefix));
      const relative = link.slice(prefix.length);
      assert.ok(!relative.includes(".."));
      assert.ok(existsSync(resolve(repositoryRoot, relative)), `missing source ${relative}`);
    }
  });

  it("keeps dated checkpoint evidence separate from live freshness and admission", () => {
    const guide = buildNodeGuideProfile(inputs);
    assert.equal(guide.live.checkpoint.currentFreshnessClaim, false);
    assert.equal(guide.live.checkpoint.independentTrustAnchor, false);
    assert.equal(BigInt(guide.live.checkpoint.bindingHeaderHeight), BigInt(guide.live.checkpoint.applicationHeight) + 1n);
    assert.match(guide.live.checkpoint.postCommitAppHash, /^[a-f0-9]{64}$/u);
    assert.equal(guide.source.productionBinaryProvenance, false);
    assert.equal(guide.documentation.agentProtocolEndpoint, false);
    assert.equal(guide.boundaries.activatesSuccessorNetwork, false);
    assert.equal(guide.boundaries.provesHistoricalPayments, false);
  });

  it("keeps the optional signed transfer local and derives its query hash from the real response", () => {
    const demo = buildNodeGuideProfile(inputs).local.testTransfer;
    assert.equal(demo.localOnly, true);
    assert.match(demo.sendCommand, /LOCAL_BIN="\$LOCAL_HOME\/bin\/zeroned"/u);
    assert.match(demo.sendCommand, /status --home "\$LOCAL_HOME" &&/u);
    assert.match(demo.sendCommand, /--chain-id zerone-local-1/u);
    assert.match(demo.sendCommand, /--node http:\/\/127\.0\.0\.1:47657/u);
    assert.match(demo.sendCommand, /--keyring-backend test/u);
    assert.match(demo.sendCommand, /--fees 250000uzrn --gas 250000/u);
    assert.match(demo.sendCommand, /TX_HASH=""/u);
    assert.match(demo.sendCommand, /json\.load\(sys\.stdin\)/u);
    assert.match(demo.sendCommand, /result\.get\("txhash", ""\)/u);
    assert.match(demo.queryCommand, /query tx "\$TX_HASH"/u);
    assert.doesNotMatch(demo.queryCommand, /tx bank send/u);
    assert.doesNotMatch(demo.sendCommand + demo.queryCommand, /zerone-1\b|fly\.dev|169\.155\.|PASTE|<TX|[a-fA-F0-9]{64}/u);
    assert.match(demo.completion, /height greater than zero and code 0/u);
    assert.match(demo.completion, /height 0 is only mempool acceptance/u);
    assert.match(demo.retry, /retry only the query/u);
  });

  it("serializes deterministically without changing its build inputs", () => {
    const frozen = Object.freeze({ ...inputs });
    const first = buildNodeGuideProfile(frozen);
    const second = buildNodeGuideProfile(frozen);
    assert.notEqual(first, second);
    assert.equal(JSON.stringify(first), JSON.stringify(second));
    assert.deepEqual(frozen, inputs);
  });
});

describe("agent documentation index", () => {
  it("discovers the node guide without inventing an agent endpoint or mutable recipe", () => {
    const text = readFileSync(new URL("../public/llms.txt", import.meta.url), "utf8");
    assert.match(text, /https:\/\/zerone\.ai\/nodes\//u);
    assert.match(text, /https:\/\/zerone\.ai\/nodes\/guide\.json/u);
    assert.match(text, /documentation index, not an agent protocol endpoint/u);
    assert.match(text, /A2A and x402 services are not published/u);
    assert.doesNotMatch(text, /\.well-known|seed phrase|\/blob\/main\//u);
    const links = [...text.matchAll(/\]\((https:\/\/[^)]+)\)/gu)].map((match) => match[1]!);
    for (const link of links) {
      const url = new URL(link);
      assert.equal(url.username, "");
      assert.equal(url.password, "");
      assert.ok(["zerone.ai", "github.com"].includes(url.hostname));
      if (url.pathname.includes("/blob/")) {
        assert.match(url.pathname, /\/blob\/[a-f0-9]{40}\//u);
        const relative = url.pathname.split("/").slice(5).join("/");
        assert.ok(existsSync(resolve(repositoryRoot, relative)), `missing index target ${relative}`);
      }
    }
  });
});

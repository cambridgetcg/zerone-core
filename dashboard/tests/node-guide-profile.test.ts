import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, it } from "node:test";

import { nodeGuidePage } from "../node-guide-page";
import { buildNodeGuideProfile, PRIVATE_NETWORK_TOOL_PATH } from "../node-guide-profile";

const inputs = {
  sourceCommit: "74e2bbac317dacb1695ed8aed20830310617ea7a",
  helperSha256: "a1".repeat(32),
  privateNetworkToolSha256: "c3".repeat(32),
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

describe("closed-beta private network lane", () => {
  it("pins the private-network tool only to a full hexadecimal digest, and omits it when absent", () => {
    for (const privateNetworkToolSha256 of ["", "0".repeat(64), "C3".repeat(32), "c3".repeat(31), `${"c3".repeat(32)}\n`]) {
      assert.throws(() => buildNodeGuideProfile({ ...inputs, privateNetworkToolSha256 }));
    }
    const tool = buildNodeGuideProfile(inputs).source.privateNetworkTool;
    assert.equal(tool?.sha256, inputs.privateNetworkToolSha256);
    assert.equal(tool?.url, `https://github.com/cambridgetcg/zerone-core/blob/${inputs.sourceCommit}/${PRIVATE_NETWORK_TOOL_PATH}`);
    const { privateNetworkToolSha256: _omitted, ...withoutTool } = inputs;
    const bare = buildNodeGuideProfile(withoutTool);
    assert.equal(bare.source.privateNetworkTool, null);
    assert.doesNotMatch(nodeGuidePage(bare), /Private-network tool/u);
  });

  it("stays invitation-only and publishes no way to join from this page", () => {
    const guide = buildNodeGuideProfile(inputs);
    const beta = guide.privateNetwork;
    assert.equal(beta.availability, "closed-beta");
    assert.equal(beta.admission, "invitation-only");
    assert.equal(beta.signUp, false);
    assert.equal(beta.publishesJoinCoordinates, false);
    assert.equal(beta.valueless, true);
    assert.equal(beta.bridged, false);
    assert.equal(beta.chainIdPattern, "zerone-rehearsal-lan-*");
    assert.deepEqual(beta.admissionChecks.map((check) => check.id), ["invited", "private-addresses", "seats-only", "fixed-at-genesis"]);
    assert.equal(guide.source.repositoryAccess, "restricted");
    assert.match(guide.source.accessNote, /private/u);
    const text = JSON.stringify({ ...beta, evidence: { ...beta.evidence, record: "" } });
    // Address ranges may be named; no host address, endpoint, node id, genesis hash or key may appear.
    for (const match of text.matchAll(/\b\d{1,3}(?:\.\d{1,3}){3}\b(?!\/\d)/gu)) assert.fail(`host address ${match[0]} in the beta lane`);
    assert.doesNotMatch(text, /\b[a-fA-F0-9]{40,}\b|:3665[67]\b|\.local\b|\/Users\/|priv_validator|mnemonic/u);
    assert.doesNotMatch(text, /zerone-rehearsal-lan-[0-9a-z]+-|zerone-1-observer/u);
    assert.equal(guide.boundaries.admitsValidatorsOrNewLiveAccounts, false);
    assert.equal(guide.boundaries.activatesSuccessorNetwork, false);
    assert.equal(guide.live.validatorJoining.availability, "not-open");
  });

  it("seats only our own Macs for now, and leaves other people's Macs to a later gated stage", () => {
    const guide = buildNodeGuideProfile(inputs);
    const beta = guide.privateNetwork;
    assert.equal(beta.seatOwnership, "our-own-macs-only");
    assert.match(beta.whoCanJoin, /only our own/u);
    const ids = beta.roadmap.map((stage) => stage.id);
    const now = beta.roadmap.find((stage) => stage.status === "now")!;
    assert.match(`${now.title} ${now.text}`, /our own Macs/u);
    const others = beta.roadmap.find((stage) => stage.id === "others-at-home");
    assert.equal(others?.status, "planned");
    assert.ok(ids.indexOf("others-at-home") > ids.indexOf(now.id), "other people's Macs come after the current stage");
    assert.match(nodeGuidePage(guide), /Closed beta · our own Macs/u);
  });

  it("dates its evidence and states its limits instead of claiming more", () => {
    const evidence = buildNodeGuideProfile(inputs).privateNetwork.evidence;
    assert.equal(evidence.currentFreshnessClaim, false);
    assert.equal(evidence.date, "2026-10-17");
    assert.match(evidence.record, new RegExp(`/blob/${inputs.sourceCommit}/skills/run-a-private-zerone-net/references/protocol\\.md$`, "u"));
    const limits = evidence.limits.join(" ");
    assert.match(limits, /no claim of decentralisation/u);
    assert.match(limits, /only in automated tests/u);
    assert.doesNotMatch(JSON.stringify(evidence), /decentrali[sz]ed\b|production-ready|guarantee/u);
    assert.equal(evidence.matchesPinnedTool, false);
  });

  it("cites only figures that its linked run record states", () => {
    const evidence = buildNodeGuideProfile(inputs).privateNetwork.evidence;
    const record = readFileSync(resolve(repositoryRoot, "skills/run-a-private-zerone-net/references/protocol.md"), "utf8");
    // Each phrase the page shows, beside the phrase in the run record that backs it.
    for (const [shown, recorded] of [
      [`${evidence.blocks} blocks`, `height ${evidence.blocks}`], [`${evidence.averageBlockSeconds} seconds`, `${evidence.averageBlockSeconds} s`],
      ["all 21 heights", "all 21 shared heights"], ["half of them", "223 each"], ["1,000 test ZRN", "1,000 ZRN each"],
    ] as const) {
      assert.ok(evidence.summary.includes(shown), `summary lacks ${shown}`);
      assert.ok(record.includes(recorded), `run record lacks ${recorded}`);
    }
  });

  it("orders the roadmap from what is done to what is planned, each later stage with its gates and no date", () => {
    const { roadmap, roadmapNote } = buildNodeGuideProfile(inputs).privateNetwork;
    const rank = { done: 0, now: 1, planned: 2 } as const;
    const ranks = roadmap.map((stage) => rank[stage.status]);
    assert.ok(ranks.every((value, index) => index === 0 || value >= ranks[index - 1]!), "stages out of order");
    assert.deepEqual([...new Set(ranks)], [0, 1, 2]);
    assert.equal(ranks.filter((value) => value === 1).length, 1);
    assert.equal(roadmap.at(-1)!.id, "open");
    for (const stage of roadmap.filter((item) => item.status !== "done")) {
      assert.ok(stage.gates.length > 0, `${stage.id} has no gates`);
      assert.doesNotMatch(`${stage.title} ${stage.text} ${stage.gates.join(" ")}`, /\b20\d\d-\d\d-\d\d\b|\bsoon\b|\bQ[1-4]\b/u);
    }
    assert.match(roadmapNote, /No dates/u);
  });

  it("renders the lane without JavaScript, between the local setup and shared development", () => {
    const html = nodeGuidePage(buildNodeGuideProfile(inputs));
    const order = ["id=\"local\"", "id=\"beta\"", "id=\"development\"", "id=\"live\""].map((marker) => html.indexOf(marker));
    assert.ok(order.every((index) => index > 0));
    assert.deepEqual([...order].sort((a, b) => a - b), order);
    assert.match(html, /<a href="#beta">Closed beta<\/a>/u);
    assert.equal([...html.matchAll(/class="scope-row[ "]/gu)].length, 4);
    assert.equal([...html.matchAll(/class="roadmap-stage"/gu)].length, buildNodeGuideProfile(inputs).privateNetwork.roadmap.length);
    assert.match(html, /The zerone-core repository is private/u);
    assert.doesNotMatch(html, /<form\b|\sonclick=|\sstyle=/u);
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

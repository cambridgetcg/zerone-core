import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { readFileSync } from "node:fs";
import { describe, it } from "node:test";
import { buildDevelopmentProfile, DEVELOPMENT_UPGRADE_PROPOSAL, validateDevelopmentRelease, type DevelopmentPublication, type DevelopmentUpgradeProposal } from "../development-profile";
import { developmentPage } from "../development-page";
import { buildNodeGuideProfile } from "../node-guide-profile";
import { nodeGuidePage } from "../node-guide-page";
import { enumLabel, fundingSummary, settlementSummary, money, getBytes, height, queryClaimId, validateHistory, verifyDescriptor } from "../src/development-reader";

const commit = "ab".repeat(20);
const publication: DevelopmentPublication = { schema: "zerone.development-publication/v1", chain_id: "zerone-dev-1", gateway: "https://zerone-dev-1.fly.dev", descriptor_sha256: "12".repeat(32), genesis_sha256: "23".repeat(32), rpc_genesis_sha256: "34".repeat(32), runtime_source_commit: "bc".repeat(20), binary_sha256: "45".repeat(32), verified_at: "2026-09-11T21:00:00Z" };
const id = "ab".repeat(16);
function historyFixture() { return { chain_id: "zerone-dev-1", block_height: "123", record: { claim_id: id, claim: { id, submitter: "zrn1author", fact_content: "<img src=x onerror=alert(1)>", reasoning_trace: "A bounded reason", status: 5 }, rounds: [{ id: "round-1", claim_id: id, phase: 2, reveals: [{ verifier: "zrn1reviewer", vote: "reject", attestation: { reason: "A counterexample", scope: "One input", evidence_ids: ["https://evidence.invalid/"] } }] }], facts: [{ fact: { id: "fact-1", claim_id: id, status: 5 }, outgoing_relations: [{ source_fact_id: "fact-1", target_fact_id: "fact-2", relation: 2 }] }], missing_round_ids: ["old-round"] }, related_claims: [] }; }

describe("separate development publication and read-only surface", () => {
  it("defaults to pending with no reader configuration and preserves legacy gates", () => {
    const guide = buildNodeGuideProfile({ sourceCommit: commit, helperSha256: "56".repeat(32) });
    assert.equal(guide.development.publication, null);
    assert.equal(guide.development.packages, null);
    assert.equal(guide.development.exampleClaim, null);
    assert.equal(guide.development.effects.browserReads, "none");
    assert.equal(guide.live.chainId, "zerone-1");
    assert.equal(guide.live.newAccountAdmission.availability, "paused");
    assert.equal(guide.boundaries.activatesSuccessorNetwork, false);
    const html = developmentPage(guide.development);
    assert.match(html, /Deployment verification pending/u);
    assert.doesNotMatch(html, /id="development-reader"|data-descriptor-sha256/u);
    assert.doesNotMatch(html, /id="example-claim"/u);
    assert.match(html, /id="claim-read"[^>]*disabled/u);
    assert.match(html, /<noscript>/u);
    assert.match(nodeGuidePage(guide), /id="development"/u);
    assert.match(nodeGuidePage(guide), /On zerone-1, validator joining/u);
  });
  it("derives compatible package URLs from the runtime pin and hashes from an external source record", () => {
    const profile = buildDevelopmentProfile(commit, publication);
    const packages = profile.packages!;
    const tag = `zerone-dev-1-${publication.runtime_source_commit.slice(0, 12)}`;
    assert.equal(packages.tag, tag);
    assert.equal(packages.recordRawUrl, `https://raw.githubusercontent.com/cambridgetcg/zerone-core/${commit}/deploy/networks/zerone-dev-1/release.json`);
    for (const artifact of packages.artifacts) assert.equal(artifact.url, `https://github.com/cambridgetcg/zerone-core/releases/download/${tag}/${tag}-${artifact.platform}.tar.gz`);
    const release = { schema: "zerone-development-release/v1", chain_id: "zerone-dev-1", source_commit: publication.runtime_source_commit, source_tree: "cd".repeat(20), release_tag: packages.tag, release_url: packages.releaseUrl, runtime_image: `registry.fly.io/zerone-dev-1@sha256:${"ef".repeat(32)}`, descriptor_sha256: publication.descriptor_sha256, genesis_sha256: publication.genesis_sha256, rpc_genesis_sha256: publication.rpc_genesis_sha256, runtime_binary_sha256: publication.binary_sha256, artifacts: packages.artifacts.map(({name,platform,url}) => ({name,platform,url,sha256:"67".repeat(32),bytes:1000,binary_sha256:platform === "linux-amd64" ? publication.binary_sha256 : "78".repeat(32)})), bootstrap_consensus: "single-operator", funds: "valueless-development-only", private_keys_in_packages: false };
    validateDevelopmentRelease(release, publication);
    for (const edit of [
      (value: typeof release) => { value.artifacts[0]!.url = "https://unrelated.invalid/package.tar.gz"; },
      (value: typeof release) => { value.artifacts[0]!.binary_sha256 = "89".repeat(32); },
      (value: typeof release) => { value.artifacts[0]!.bytes = 268435457; },
      (value: typeof release) => { value.artifacts[1] = value.artifacts[0]!; },
      (value: typeof release) => { value.descriptor_sha256 = "9a".repeat(32); },
      (value: typeof release) => { value.source_commit = commit; },
      (value: typeof release) => { value.private_keys_in_packages = true; },
    ]) { const value = structuredClone(release); edit(value); assert.throws(() => validateDevelopmentRelease(value, publication)); }
    const html = developmentPage(profile);
    assert.match(html, /id="packages"/u);
    assert.ok(html.includes(packages.recordUrl));
    assert.ok(packages.artifacts.every(({url}) => html.includes(url)));
    assert.doesNotMatch(developmentPage(buildDevelopmentProfile(commit)), /id="packages"/u);
  });
  it("binds dated publication and source roles without fresh-read or authority claims", () => {
    const profile = buildDevelopmentProfile(commit, publication);
    assert.equal(profile.publication?.runtime_source_commit, publication.runtime_source_commit);
    assert.match(profile.source.client, new RegExp(`/${commit}/scripts/shared-claims.py$`, "u"));
    assert.match(profile.source.guide, new RegExp(`/${commit}/docs/SHARED-DEVELOPMENT.md$`, "u"));
    assert.equal(profile.effects.transactions, false);
    assert.equal(profile.effects.browserFaucet, false);
    assert.equal(profile.effects.browserStorage, false);
    assert.match(profile.status, /Availability can change/u);
    assert.match(profile.consensus, /addresses do not prove independent/u);
    const html = developmentPage(profile);
    assert.match(html, new RegExp(`data-descriptor-sha256="${publication.descriptor_sha256}"`, "u"));
    assert.match(html, /not a pending-work feed/u);
    assert.match(html, /runtime source/u);
    assert.equal(profile.exampleClaim?.includedAtHeight, "432");
    assert.match(profile.exampleClaim!.scope, /no reviews or challenges/u);
    assert.match(profile.exampleClaim!.scope, /no claim of acceptance or independent endorsement/u);
    assert.ok(html.includes(profile.exampleClaim!.url));
  });
  it("refuses arbitrary endpoints, incomplete pins, extra fields and invalid dates", () => {
    for (const change of [{ gateway: "https://evil.invalid" }, { chain_id: "zerone-1" }, { binary_sha256: "0".repeat(64) }, { runtime_source_commit: "main" }, { descriptor_sha256: publication.descriptor_sha256.toUpperCase().replace("12", "AB") }, { verified_at: "2026-02-31T00:00:00Z" }, { extra: true }]) assert.throws(() => buildDevelopmentProfile(commit, { ...publication, ...change } as DevelopmentPublication));
    assert.throws(() => buildDevelopmentProfile("main", publication));
  });
  it("contains only fixed explicit GET reads and safe text rendering", () => {
    const source = readFileSync(new URL("../src/development-reader.ts", import.meta.url), "utf8") + readFileSync(new URL("../src/development.ts", import.meta.url), "utf8");
    assert.doesNotMatch(source, /innerHTML|insertAdjacentHTML|localStorage|sessionStorage|indexedDB|eval\(|method:\s*["']POST/u);
    assert.match(source, /textContent/u); assert.match(source, /credentials: "omit"/u); assert.match(source, /redirect: "error"/u);
    assert.match(source, /params.has\("claim"\)/u);
  });
});

describe("bounded endpoint observations", () => {
  it("accepts exact IDs and large lossless heights while refusing URL intent and lossy numbers", () => {
    assert.equal(queryClaimId(id), id); assert.equal(height("18446744073709551615"), "18446744073709551615");
    for (const value of ["", `${id} `, id.toUpperCase(), "../status", "é".repeat(32), null]) assert.throws(() => queryClaimId(value));
    for (const value of [-1, 1.5, Number.MAX_SAFE_INTEGER + 1, "01", "18446744073709551616"]) assert.throws(() => height(value));
  });
  it("keeps missing historical rows explicit and refuses cross-chain or mismatched joins", () => {
    assert.equal(validateHistory(historyFixture(), id).chain_id, "zerone-dev-1");
    const absent = historyFixture(); delete (absent.record as { claim?: unknown }).claim;
    assert.ok(validateHistory(absent, id));
    const mutations = [
      (v: ReturnType<typeof historyFixture>) => { v.chain_id = "zerone-1"; },
      (v: ReturnType<typeof historyFixture>) => { v.block_height = "0"; },
      (v: ReturnType<typeof historyFixture>) => { v.record.claim.id = "other"; },
      (v: ReturnType<typeof historyFixture>) => { v.record.rounds[0]!.claim_id = "other"; },
      (v: ReturnType<typeof historyFixture>) => { v.record.facts[0]!.fact.claim_id = "other"; },
      (v: ReturnType<typeof historyFixture>) => { v.record.facts[0]!.outgoing_relations[0]!.source_fact_id = "other"; },
      (v: ReturnType<typeof historyFixture>) => { v.record.rounds = Array(2001).fill(v.record.rounds[0]); },
    ];
    for (const change of mutations) { const value = historyFixture(); change(value); assert.throws(() => validateHistory(value, id)); }
  });
  it("renders numeric and symbolic enums with their own domain and explicit unknowns", () => {
    assert.equal(enumLabel("CLAIM_STATUS", 5), "IN VERIFICATION");
    assert.equal(enumLabel("FACT_STATUS", 5), "CONTESTED");
    assert.equal(enumLabel("VERIFICATION_PHASE", 2), "REVEAL");
    assert.equal(enumLabel("RELATION_TYPE", "RELATION_TYPE_CONTRADICTS"), "CONTRADICTS");
    assert.equal(enumLabel("INFERENCE_TYPE", "INFERENCE_TYPE_EMPIRICAL"), "EMPIRICAL");
    assert.match(enumLabel("VERDICT", 99), /Unknown VERDICT: 99/u);
  });
  it("joins descriptor bytes and runtime policy to the publication pins", async () => {
    const descriptor = { schema: "zerone-shared-development/v1", chain_id: "zerone-dev-1", rpc_url: publication.gateway, genesis_url: `${publication.gateway}/genesis.json`, faucet_url: `${publication.gateway}/faucet`, genesis_sha256: publication.genesis_sha256, rpc_genesis_sha256: publication.rpc_genesis_sha256, source_commit: publication.runtime_source_commit, runtime_binary_sha256: publication.binary_sha256, bootstrap_consensus: "single-operator", reset_policy: "new-chain-id", knowledge_version: 10, commitment_scheme: 2, review_policy_version: 1, local_test: false, denom: "uzrn", gas_limit: 2000000, tx_fee_uzrn: "2000000" };
    const bytes = new TextEncoder().encode(JSON.stringify(descriptor));
    const pins = { descriptor: createHash("sha256").update(bytes).digest("hex"), genesis: publication.genesis_sha256, rpcGenesis: publication.rpc_genesis_sha256, source: publication.runtime_source_commit, binary: publication.binary_sha256 };
    await verifyDescriptor(bytes, pins);
    await assert.rejects(verifyDescriptor(bytes, { ...pins, binary: "ff".repeat(32) }));
    await assert.rejects(verifyDescriptor(new Uint8Array([...bytes, 10]), pins));
  });
  it("bounds chunked reads before decode, refuses foreign routes, and omits credentials", async () => {
    const requests: RequestInit[] = [];
    const fetcher: typeof fetch = async (_url, init) => { requests.push(init!); return new Response(new Uint8Array([1, 2, 3, 4]), { headers: { "Content-Type": "application/json" } }); };
    const signal = new AbortController().signal;
    assert.deepEqual(await getBytes(`${publication.gateway}/network.json`, 4, signal, fetcher), new Uint8Array([1, 2, 3, 4]));
    assert.equal(requests[0]?.method, "GET"); assert.equal(requests[0]?.credentials, "omit");
    await assert.rejects(getBytes(`${publication.gateway}/claims/${id}`, 3, signal, fetcher));
    for (const url of ["https://evil.invalid/network.json", `${publication.gateway}/faucet`, `${publication.gateway}/network.json?proxy=foo`, `${publication.gateway}/claims/../status`]) await assert.rejects(getBytes(url, 10, signal, fetcher));
    await assert.rejects(getBytes(`${publication.gateway}/network.json`, 10, signal, async () => new Response("oops", { headers: { "Content-Type": "text/html" } })));
  });
});

describe("recorded funding and transfers", () => {
  const terms = { policy_version: 1, kind: 1, paid_amount: "200001", review_budget: "110000", refundable_amount: "0", retained_fee: "90001" };
  it("preserves integer amounts and historical absence", () => {
    assert.equal(money("18446744073709551615"), "18446744073709.551615 development ZRN (18446744073709551615 uzrn)");
    assert.match(fundingSummary({})[0]![1], /Historical message-specific rules/u);
    assert.ok(fundingSummary({ funding_terms: terms }).some(([, text]) => text.includes("200001 uzrn")));
    for (const value of ["01", "-1", "1.0", 2, "18446744073709551616"]) assert.throws(() => money(value));
    assert.throws(() => fundingSummary({ funding_terms: { ...terms, paid_amount: "200000" } }));
    assert.throws(() => fundingSummary({ funding_terms: { ...terms, policy_version: 2 } }));
  });
  it("separates pending reviewer and refund obligations from transferred records", () => {
    const refund = { recipient: "zrn1author", amount: "90001", created_at_block: "42" };
    assert.match(settlementSummary(refund, true)[0]![1], /awaiting transfer/u);
    assert.match(settlementSummary({ ...refund, paid_at_block: "44" }, true)[0]![1], /transferred at block 44/u);
    const review = { created_at_block: "42", paid_at_block: "43", payments: [{ verifier: "zrn1reviewer", amount: "110000", withheld: "0" }], withheld_total: "0" };
    assert.ok(settlementSummary(review, false).some(([label,text]) => label.includes("zrn1reviewer") && text.includes("110000 uzrn")));
    assert.match(settlementSummary(null, true)[0]![1], /does not establish/u);
    assert.throws(() => settlementSummary({ ...refund, paid_at_block: "41" }, true));
  });
});

describe("proposed upgrade notice remains separate from live participation", () => {
  // Final release bytes, with a synthetic proposal ID/time/observation height for rendering only.
  // This fixture is separate from the actual dated governance observation below.
  const proposal: DevelopmentUpgradeProposal = { status: "proposed", target_source_commit: "5542b221864ca0080e143fa82abbef4cd2064f71", height: "300000", proposal_id: "7", recorded_at: "2026-09-13T12:00:00Z", observed_height: "15000", packet_url: "https://github.com/cambridgetcg/zerone-core/releases/download/zerone-dev-1-5542b221864c/UPGRADE.json", packet_sha256: "04f0415a361396e28397155ecb3c0b13d578475ebf1b86c3e2e2b5d533006f75", release_record_url: "https://github.com/cambridgetcg/zerone-core/releases/download/zerone-dev-1-5542b221864c/UPGRADE-RELEASE.json", release_record_sha256: "be85503652e9c467e074970c7169c80d5864a2a34fc620db7ca83d0b435d1d2b" };
  it("publishes the actual dated proposal without replacing the original participant release", () => {
    assert.equal(DEVELOPMENT_UPGRADE_PROPOSAL?.proposal_id, "1");
    assert.equal(DEVELOPMENT_UPGRADE_PROPOSAL?.recorded_at, "2026-09-13T01:12:41Z");
    assert.equal(DEVELOPMENT_UPGRADE_PROPOSAL?.observed_height, "44135");
    const live = buildDevelopmentProfile(commit, publication);
    assert.equal(live.upgradeProposal?.height, "300000");
    assert.equal(live.upgradeProposal?.target_source_commit, proposal.target_source_commit);
    assert.equal(live.upgradeProposal?.packet_sha256, proposal.packet_sha256);
    assert.equal(live.upgradeProposal?.release_record_sha256, proposal.release_record_sha256);
    assert.deepEqual(live.packages, buildDevelopmentProfile(commit, publication, null).packages);
    assert.equal(buildDevelopmentProfile(commit).upgradeProposal, null);
    assert.match(developmentPage(live), /Status observed 2026-09-13T01:12:41Z at block 44135/u);
  });
  it("renders exact proposal metadata with no replacement of live pins or packages", () => {
    const original = buildDevelopmentProfile(commit, publication, null);
    const proposed = buildDevelopmentProfile(commit, publication, proposal);
    assert.deepEqual(proposed.publication, original.publication);
    assert.deepEqual(proposed.packages, original.packages);
    assert.deepEqual(proposed.effects, original.effects);
    assert.equal(proposed.upgradeProposal?.height, "300000");
    const html = developmentPage(proposed);
    assert.match(html, /Proposed · activation not verified here/u);
    assert.match(html, /Proposal 7 · proposed activation block 300000/u);
    assert.match(html, /Status observed 2026-09-13T12:00:00Z at block 15000/u);
    assert.doesNotMatch(html, /not activated|until then/iu);
    assert.ok(html.includes(proposal.packet_url) && html.includes(proposal.packet_sha256));
    assert.ok(html.includes(proposal.release_record_url) && html.includes(proposal.release_record_sha256));
    assert.match(html, /Fresh full nodes need both verified packages/u);
    assert.match(html, /Ordinary claims and conjectures allocate 55%/u);
    assert.match(html, /A refundable amount is not evidence of payment/u);
    assert.match(html, /original compatible participant packages are retained below/u);
    assert.match(html, /<details><summary>Keep participating/u);
    assert.doesNotMatch(developmentPage(original), /id="fund-upgrade"/u);
  });
  it("keeps an unknown proposal ID unpublished", () => {
    assert.equal(buildDevelopmentProfile(commit, publication, null).upgradeProposal, null);
    assert.doesNotMatch(developmentPage(buildDevelopmentProfile(commit, publication, null)), /id="fund-upgrade"/u);
    for (const pending of [undefined, null, "", "pending"]) {
      assert.throws(() => buildDevelopmentProfile(commit, publication, { ...proposal, proposal_id: pending } as unknown as DevelopmentUpgradeProposal));
    }
  });
  it("refuses activation claims, unpinned inputs, cross-release links and ambiguous numbers", () => {
    for (const change of [
      (value: DevelopmentUpgradeProposal) => { (value as { status: string }).status = "activated"; },
      (value: DevelopmentUpgradeProposal) => { value.target_source_commit = publication.runtime_source_commit; },
      (value: DevelopmentUpgradeProposal) => { value.packet_sha256 = ""; },
      (value: DevelopmentUpgradeProposal) => { value.height = "0200000"; },
      (value: DevelopmentUpgradeProposal) => { value.proposal_id = "0"; },
      (value: DevelopmentUpgradeProposal) => { value.observed_height = ""; },
      (value: DevelopmentUpgradeProposal) => { value.observed_height = "015000"; },
      (value: DevelopmentUpgradeProposal) => { value.recorded_at = "2026-99-99T00:00:00Z"; },
      (value: DevelopmentUpgradeProposal) => { value.packet_url = value.packet_url.replace("github.com", "untrusted.invalid"); },
      (value: DevelopmentUpgradeProposal) => { value.release_record_url = value.release_record_url.replace("zerone-dev-1-5542b221864c", "different-release"); },
    ]) { const value = structuredClone(proposal); change(value); assert.throws(() => buildDevelopmentProfile(commit, publication, value)); }
    const { observed_height: _missing, ...missingObservation } = proposal;
    assert.throws(() => buildDevelopmentProfile(commit, publication, missingObservation as DevelopmentUpgradeProposal));
    assert.throws(() => buildDevelopmentProfile(commit, undefined, proposal));
  });
});

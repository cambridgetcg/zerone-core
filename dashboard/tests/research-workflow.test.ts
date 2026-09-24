import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { describe, it } from "node:test";
import { canonicalJSON, EXPORT_LIMIT, sha256 } from "../src/research-review-journal";
import { appendWorkflowObservation, boundReasoning, prepareWorkflow, validateBoundHistory, validateResearchFile, validateWorkflow, workflowClaimId, WORKFLOW_LIMIT, type ResearchWorkflow, type WorkflowIntent } from "../src/research-workflow";

const source = readFileSync(new URL("../public/research/review/fictional-journal.v1.json", import.meta.url), "utf8");
const pythonRoot = fileURLToPath(new URL("../../tools/research-review/", import.meta.url));
const intent: WorkflowIntent = { content: "Fictional sorting experiment preserves equal-element order.", reasoning: "A synthetic interoperability check; no real scientific result or transaction.", domain: "mathematics", category: "computational", method: "M-COMPUTATIONAL" };
const address = `zrn1${"q".repeat(38)}`;
const claimId = "a".repeat(32), roundId = "b".repeat(32);
const stringify = (value: unknown): string => JSON.stringify(value);
const utf8 = new TextEncoder();
const python = (code: string, value: unknown): unknown => JSON.parse(execFileSync("python3", ["-B", "-c", "import json,sys,workflow\n" + code], { cwd: pythonRoot, input: stringify(value), encoding: "utf8", maxBuffer: 32 * 1024 * 1024 }));
async function draft(): Promise<ResearchWorkflow> { return (await prepareWorkflow(source, "finding-v1", intent)).packet; }
async function submitted(): Promise<ResearchWorkflow> {
  const packet = await draft();
  packet.participant = { chain_id: "zerone-dev-1", address, descriptor_sha256: "1".repeat(64), genesis_sha256: "2".repeat(64), binary_sha256: "3".repeat(64), local_test: true, client_files: { "shared-claims.py": "4".repeat(64), "claim-workflow.py": "5".repeat(64), "local-node.py": "6".repeat(64) } };
  packet.submission = { state: "observed", receipt: { id: "e".repeat(32), action: "submit", actor: "participant", status: "committed", txhash: "A".repeat(64), height: "100", code: 0, claim_id: claimId, round_id: roundId, review_fee_uzrn: "1000000" } };
  return packet;
}
function history(packet: ResearchWorkflow, blockHeight = "120"): Record<string, unknown> {
  return { chain_id: "zerone-dev-1", block_height: blockHeight, record: {
    claim_id: claimId,
    claim: { id: claimId, fact_content: packet.intent.content, reasoning_trace: boundReasoning(packet), domain: packet.intent.domain, category: packet.intent.category, method_id: packet.intent.method, submitter: address, submitted_at_block: "100", verification_round_id: roundId, status: "CLAIM_STATUS_INSUFFICIENT", funding_terms: { policy_version: 1, kind: "CLAIM_FUNDING_KIND_REVIEW_FEE", paid_amount: "1000000", review_budget: "550000", refundable_amount: "0", retained_fee: "450000" } },
    rounds: [{ id: roundId, claim_id: claimId, phase: "VERIFICATION_PHASE_COMPLETE", verdict: "VERDICT_INCONCLUSIVE", commits: [], reveals: [], verifier_reward_settlement: null, claim_refund_settlement: { created_at_block: "115", paid_at_block: "116", recipient: address, amount: "550000" } }],
    facts: [], missing_round_ids: [],
  }, related_claims: [] };
}
async function rehashIntent(packet: ResearchWorkflow): Promise<void> {
  const { export_sha256, record_id, entry_sha256 } = packet.journal;
  packet.intent_sha256 = await sha256(canonicalJSON({ export_sha256, record_id, entry_sha256, intent: packet.intent }));
}

describe("research/native workflow binding", () => {
  it("prepares byte-identical Python/browser drafts without promoting the journal to a native claim", async () => {
    const checked = await prepareWorkflow(source, "finding-v1", intent);
    const packet = checked.packet;
    const other = python("v=json.load(sys.stdin)\nprint(json.dumps(workflow.prepare(v['source'].encode(),v['record'],v['intent'])))", { source, record: "finding-v1", intent });
    assert.equal(canonicalJSON(packet), canonicalJSON(other));
    assert.equal(checked.journal.entries.length, 27);
    assert.equal(packet.journal.export_utf8, source);
    assert.equal(checked.entry.record.id, "finding-v1");
    assert.equal(packet.participant, null);
    assert.equal(packet.submission, null);
    assert.equal(workflowClaimId(packet), null);
    assert.equal(checked.history, undefined);
    assert.deepEqual(packet.observations, []);
  });
  it("binds exact native assertion, author, admission height and journal digest; Python agrees", async () => {
    const packet = await submitted();
    const checked = await appendWorkflowObservation(packet, stringify(history(packet)), "2026-09-23T12:00:00.000Z");
    const complete = checked.packet;
    assert.equal(workflowClaimId(complete), claimId);
    assert.equal(checked.history?.block_height, "120");
    assert.equal(checked.entry.record.id, packet.journal.record_id);
    assert.equal(checked.journal.entries.length, 27);
    assert.equal(packet.observations.length, 0, "append returns a new record rather than mutating a retained input");
    assert.equal(python("v=json.load(sys.stdin)\nworkflow.validate_packet(v)\nprint(json.dumps(True))", complete), true);
    assert.ok(boundReasoning(packet).includes(packet.intent_sha256));
    assert.ok(!boundReasoning(packet).includes("Fictional researcher A"), "the complete journal is not a native reasoning payload");
  });
  it("retains corrections and earlier assessments while a new correction is a separate admission", async () => {
    const original = await draft();
    const checked = await prepareWorkflow(source, "finding-v2", { ...intent, content: "Fictional revised sorting result narrows its tested input assumptions." });
    const correction = checked.packet;
    assert.equal(checked.entry.record.supersedes, "finding-v1");
    assert.equal(checked.journal.entries.length, 27);
    assert.notEqual(correction.intent_sha256, original.intent_sha256);
    assert.equal(correction.submission, null);
    assert.deepEqual(correction.observations, []);
  });
  it("refuses hidden source edits, record substitution and claim-field changes", async () => {
    const modifications: ((p: ResearchWorkflow) => void)[] = [
      p => { p.journal.export_utf8 += " "; },
      p => { p.journal.record_id = "finding-v2"; },
      p => { p.journal.head_sha256 = "f".repeat(64); },
      p => { p.intent.content += " changed"; },
    ];
    for (const change of modifications) { const packet = await draft(); change(packet); await assert.rejects(validateWorkflow(stringify(packet))); }
    const packet = await draft(); packet.journal.export_utf8 = source.replace("Fictional researcher A", "An asserted different author"); packet.journal.export_sha256 = await sha256(packet.journal.export_utf8); await rehashIntent(packet);
    await assert.rejects(validateWorkflow(stringify(packet)), /SHA256 mismatch/u);
  });
  it("rejects unrelated, stale, differently authored or unbound native observations", async () => {
    const packet = await submitted();
    const mutate: ((h: any) => void)[] = [
      h => { h.chain_id = "another-chain"; },
      h => { h.record.claim_id = "c".repeat(32); },
      h => { h.record.claim.submitter = `zrn1${"p".repeat(38)}`; },
      h => { h.record.claim.reasoning_trace = packet.intent.reasoning; },
      h => { h.record.claim.fact_content += " different assertion"; },
      h => { h.record.claim.submitted_at_block = "99"; },
      h => { h.record.claim.verification_round_id = "d".repeat(32); },
      h => { h.block_height = "99"; },
    ];
    for (const change of mutate) { const response = history(packet); change(response); assert.throws(() => validateBoundHistory(packet, response)); }
    const retained = await appendWorkflowObservation(packet, stringify(history(packet)), "2026-09-23T12:00:00.000Z");
    await assert.rejects(appendWorkflowObservation(retained.packet, stringify(history(packet, "119"))), /regressed/u);
  });
  it("keeps unresolved and failed submissions distinct from a successful receipt", async () => {
    for (const status of ["pending", "failed", "committed-unresolved"]) {
      const packet = await submitted(); packet.submission!.receipt!.status = status;
      assert.equal(workflowClaimId(packet), null);
      await assert.rejects(appendWorkflowObservation(packet, stringify(history(packet))), /successful bound submission/u);
    }
    for (const change of [(p: ResearchWorkflow) => { p.submission!.receipt!.height = "0"; }, (p: ResearchWorkflow) => { p.submission!.receipt!.code = 1; }, (p: ResearchWorkflow) => { p.submission!.state = "started"; }]) {
      const packet = await submitted(); change(packet); await assert.rejects(validateWorkflow(stringify(packet)));
    }
  });
  it("refuses impossible funding allocations and settlement chronology", async () => {
    const packet = await submitted();
    const response = history(packet) as any; response.record.claim.funding_terms.retained_fee = "1";
    assert.throws(() => validateBoundHistory(packet, response), /allocations/u);
    const payment = history(packet) as any; payment.record.rounds[0].claim_refund_settlement.paid_at_block = "114";
    assert.throws(() => validateBoundHistory(packet, payment), /settlement heights/u);
    payment.record.rounds[0].claim_refund_settlement.paid_at_block = "121";
    assert.throws(() => validateBoundHistory(packet, payment), /future height/u);
  });
  it("rejects unknown/secret receipt fields, duplicate keys and corrupt retained history", async () => {
    const packet = await submitted(); packet.submission!.receipt!.salt = "never-export-private-review-material";
    await assert.rejects(validateWorkflow(stringify(packet)), /unsupported workflow fields/u);
    const draftPacket = await draft();
    await assert.rejects(validateWorkflow(stringify(draftPacket).replace('"observations":[]', '"observations":[],"observations":[]')), /Duplicate/u);
    const bound = await submitted(); const complete = await appendWorkflowObservation(bound, stringify(history(bound)));
    const duplicate = await appendWorkflowObservation(complete.packet, stringify(history(bound)));
    assert.equal(duplicate.packet.observations.length, 1);
    assert.equal(canonicalJSON(duplicate.packet), canonicalJSON(complete.packet));
    assert.deepEqual(duplicate.history, complete.history);
    assert.notStrictEqual(duplicate.packet, complete.packet, "duplicate observation returns a separately validated packet");
    duplicate.packet.observations[0]!.history_utf8 += " ";
    assert.equal(complete.packet.observations[0]!.history_utf8, stringify(history(bound)), "changing a returned packet cannot change its input");
    await assert.rejects(validateWorkflow(stringify(duplicate.packet)), /history SHA/u);
  });
  it("enforces native UTF-8 limits including binding overhead, calendar dates and retained-observation caps", async () => {
    for (const change of [{ content: "too short" }, { content: "Ω".repeat(501) }, { reasoning: "x".repeat(8192) }, { method: "a".repeat(129) }, { reasoning: "NUL\0value" }]) await assert.rejects(prepareWorkflow(source, "finding-v1", { ...intent, ...change }));
    const packet = await submitted();
    for (const date of ["2026-02-30T12:00:00.000Z", "0000-01-01T00:00:00.000Z", "2026-09-23T12:00:00Z"]) await assert.rejects(appendWorkflowObservation(packet, stringify(history(packet)), date));
    const full = structuredClone(packet);
    for (let i = 0; i < 32; i++) {
      const historyUtf8 = stringify(history(packet, String(120 + i)));
      full.observations.push({ observed_at: "2026-09-23T12:00:00.000Z", history_utf8: historyUtf8, history_sha256: await sha256(historyUtf8) });
    }
    await validateWorkflow(stringify(full));
    await assert.rejects(appendWorkflowObservation(full, stringify(history(packet, "152"))), /32 observations/u);
  });
});

describe("complete research file detection", () => {
  it("opens a journal with its original bytes and a workflow with its exact embedded journal", async () => {
    const journalBytes = utf8.encode(source);
    const standalone = await validateResearchFile(journalBytes);
    assert.deepEqual(standalone.bytes, journalBytes);
    assert.equal(standalone.hash, await sha256(source));
    assert.equal(standalone.journal.entries.length, 27);
    assert.equal(standalone.workflow, undefined);

    const packet = await submitted();
    const complete = await appendWorkflowObservation(packet, stringify(history(packet)));
    const wrapper = JSON.stringify(complete.packet, null, 2) + "\n";
    const opened = await validateResearchFile(utf8.encode(wrapper));
    assert.deepEqual(opened.bytes, journalBytes, "returns embedded bytes rather than a reconstructed journal or wrapper");
    assert.equal(opened.hash, standalone.hash);
    assert.notEqual(opened.hash, await sha256(wrapper));
    assert.strictEqual(opened.journal, opened.workflow!.journal, "reuses the complete validated journal");
    assert.equal(opened.workflow!.entry.record.id, "finding-v1");
    assert.equal(opened.workflow!.history!.block_height, "120");
  });

  it("keeps the journal's 8 MiB bound inside the outer 16 MiB file bound", async () => {
    function padded(raw: string, length: number): Uint8Array {
      const bytes = new Uint8Array(length).fill(32);
      bytes.set(utf8.encode(raw));
      return bytes;
    }
    await validateResearchFile(padded(source, EXPORT_LIMIT));
    await assert.rejects(validateResearchFile(padded(source, EXPORT_LIMIT + 1)), /8 MiB/u);
    const packet = await draft();
    await validateResearchFile(padded(stringify(packet), WORKFLOW_LIMIT));
    await assert.rejects(validateResearchFile(padded(stringify(packet), WORKFLOW_LIMIT + 1)), /16 MiB/u);
    packet.journal.export_utf8 = new TextDecoder().decode(padded(source, EXPORT_LIMIT + 1));
    await assert.rejects(validateResearchFile(utf8.encode(stringify(packet))), /journal export/u);
    await assert.rejects(validateResearchFile(new Uint8Array()), /nonempty/u);
  });

  it("refuses unknown schemas, malformed UTF-8, duplicate keys and source tampering before opening", async () => {
    for (const value of [{}, { schema: "zerone-research-export/v2" }, { schema: "__proto__" }]) {
      await assert.rejects(validateResearchFile(utf8.encode(stringify(value))), /Unsupported research file schema/u);
    }
    await assert.rejects(validateResearchFile(new Uint8Array([0xff, 0xfe])));
    await assert.rejects(validateResearchFile(utf8.encode('{"schema":"zerone-research-export/v1","schema":"zerone-research-workflow/v1"}')), /Duplicate/u);
    await assert.rejects(validateResearchFile(utf8.encode("\uFEFF" + source)), /Invalid JSON/u);
    const packet = await draft();
    packet.journal.export_utf8 += " ";
    await assert.rejects(validateResearchFile(utf8.encode(stringify(packet))), /journal export SHA/u);
    const altered = JSON.parse(source);
    altered.entries[0].record.summary += " unauthenticated change";
    await assert.rejects(validateResearchFile(utf8.encode(stringify(altered))), /SHA256 mismatch/u);
  });
});

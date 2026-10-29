// The run loop for one request: verify -> membership -> read the chain ->
// gates -> build -> check -> simulate -> (live only) sign -> check -> record
// -> broadcast with a long inclusion wait and a result.code check ->
// postcheck -> reply. Dry run is the default and stops after simulate.
import { existsSync, mkdirSync, renameSync, rmSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { addressPayloadHex } from "./bech32.ts";
import type { ChainReader, Grant } from "./chain.ts";
import { loadAllowlist, type DoorConfig, type DoorPaths } from "./config.ts";
import {
  channelKeyOf,
  checkMembership,
  currentWindow,
  evaluateGates,
  gatherGlobalFacts,
  gatherSubjectFacts,
  haltFilesPresent,
  type GateReason,
  type GlobalFacts,
} from "./gates.ts";
import type { IntakeItem } from "./intake.ts";
import { loadLedger, saveLedger, utcDay, type Ledger, type LedgerEntry } from "./ledger.ts";
import type { PassphraseSource } from "./passphrase.ts";
import {
  ADMIT_GAS,
  ALLOWANCE_PERIOD_LIMIT_UZRN,
  ALLOWANCE_PERIOD_SECONDS,
  ALLOWANCE_SPEND_LIMIT_UZRN,
  ALLOWED_MESSAGES,
  CHAIN_STALE_SECONDS,
  INCLUSION_TIMEOUT_MS,
  REGRANT_GAS,
  SIMULATE_MARGIN_GAS,
} from "./policy.ts";
import { parseDoorRequest, sha256Hex, verifyDoorRequest } from "./request.ts";
import {
  broadcastTx,
  buildAdmissionTx,
  buildRegrantTx,
  checkBuiltTx,
  checkSignedTx,
  encodeTx,
  grantExpiration,
  signTx,
  tcpNode,
  txHashOf,
  withPublicSigner,
  type UnsignedTx,
  type ZeronedRunner,
} from "./tx.ts";

export type Decision = "admitted" | "regranted" | "refused" | "dry_run_ok" | "pending" | "failed";

export type DoorReason =
  | GateReason
  | "allowlist_unreadable"
  | "build_failed"
  | "simulate_failed"
  | "gas_over_limit"
  | "passphrase_unavailable"
  | "sign_failed"
  | "signed_tx_mismatch"
  | "broadcast_rejected"
  | "broadcast_failed"
  | "inclusion_timeout"
  | "delivery_failed"
  | "postcheck_failed"
  | "dropped";

export interface Reply {
  schema: "zerone-door-reply/v1";
  request_id: string;
  decision: Decision;
  mode: "dry-run" | "live";
  reasons: DoorReason[];
  address?: string;
  did?: string;
  kind?: "admit" | "regrant";
  registrar?: string;
  pot_id?: string;
  allowance?: {
    allowed_messages: readonly string[];
    spend_limit_uzrn: number;
    period_seconds: number;
    period_limit_uzrn: number;
    expiration: string;
  };
  tx_hash?: string;
  height?: number;
  simulated_gas?: number;
  gas_limit?: number;
  detail?: string;
  at: string;
}

export interface DoorContext {
  config: DoorConfig;
  paths: DoorPaths;
  chain: ChainReader;
  zeroned: ZeronedRunner;
  passphrase: PassphraseSource;
  live: boolean;
  now: () => Date;
  log: (event: Record<string, unknown>) => void;
  sleep?: (ms: number) => Promise<void>;
  inclusionTimeoutMs?: number;
}

function ledgerPath(ctx: DoorContext): string {
  return join(ctx.paths.state, ctx.live ? "ledger.json" : "dry-run-ledger.json");
}

export function readLedger(ctx: DoorContext): Ledger {
  return loadLedger(ledgerPath(ctx));
}

export function writeLedger(ctx: DoorContext, ledger: Ledger): void {
  saveLedger(ledgerPath(ctx), ledger);
}

/** Replies are written once per request id and never overwritten, except
 * that a pending reply is replaced by its final outcome. */
export function writeReply(ctx: DoorContext, reply: Reply): void {
  const dir = ctx.live ? ctx.paths.replies : join(ctx.paths.replies, "dry-run");
  mkdirSync(dir, { recursive: true, mode: 0o700 });
  const path = join(dir, `${reply.request_id}.json`);
  const tmp = `${path}.tmp-${process.pid}`;
  writeFileSync(tmp, `${JSON.stringify(reply, null, 2)}\n`, { mode: 0o600 });
  renameSync(tmp, path);
}

function allowanceSummary(expiration: string): NonNullable<Reply["allowance"]> {
  return {
    allowed_messages: ALLOWED_MESSAGES,
    spend_limit_uzrn: ALLOWANCE_SPEND_LIMIT_UZRN,
    period_seconds: ALLOWANCE_PERIOD_SECONDS,
    period_limit_uzrn: ALLOWANCE_PERIOD_LIMIT_UZRN,
    expiration,
  };
}

/** The allowance on chain must be the one the door built (spend may already
 * have begun, so the remaining limit may be lower but never higher). */
export function allowanceMatches(grant: Grant, expiration: string): boolean {
  const a = grant.allowance;
  return JSON.stringify(a.types) === JSON.stringify([
    "/cosmos.feegrant.v1beta1.AllowedMsgAllowance",
    "/cosmos.feegrant.v1beta1.PeriodicAllowance",
    "/cosmos.feegrant.v1beta1.BasicAllowance",
  ])
    && JSON.stringify(a.allowedMessages) === JSON.stringify(ALLOWED_MESSAGES)
    && a.spendLimitUzrn !== null && a.spendLimitUzrn > 0n && a.spendLimitUzrn <= BigInt(ALLOWANCE_SPEND_LIMIT_UZRN)
    && a.periodSeconds === ALLOWANCE_PERIOD_SECONDS
    && a.periodLimitUzrn === BigInt(ALLOWANCE_PERIOD_LIMIT_UZRN)
    && a.expiration !== null && Date.parse(a.expiration) === Date.parse(expiration);
}

async function postcheck(ctx: DoorContext, entry: LedgerEntry): Promise<string[]> {
  const problems: string[] = [];
  const pot = await ctx.chain.pot(`bootstrap-${entry.address}`);
  if (!pot.ok) problems.push(`pot read: ${pot.error}`);
  else if (pot.value === null) problems.push("no seed pot");
  else if (JSON.stringify(pot.value.whitelist) !== JSON.stringify([entry.address])) problems.push("pot whitelist");
  // The agent may already have claimed its seed in the next block.
  else if (!["POT_STATUS_ACTIVE", "POT_STATUS_DEPLETED"].includes(pot.value.status)) problems.push(`pot status ${pot.value.status}`);
  const grant = await ctx.chain.grant(ctx.config.registrar.address, entry.address);
  if (!grant.ok) problems.push(`grant read: ${grant.error}`);
  else if (grant.value === null) problems.push("no allowance from R");
  else if (!allowanceMatches(grant.value, entry.grant_expiration)) problems.push("allowance differs from the one built");
  return problems;
}

/** Looks up a broadcast transaction until it is included or the wait ends. */
async function awaitInclusion(ctx: DoorContext, txHash: string): Promise<{ height: number; code: number; log: string } | null | "error"> {
  const sleep = ctx.sleep ?? ((ms: number) => Bun.sleep(ms));
  const deadline = Date.now() + (ctx.inclusionTimeoutMs ?? INCLUSION_TIMEOUT_MS);
  for (;;) {
    const r = await ctx.chain.txByHash(txHash);
    if (r.ok && r.value !== null) return r.value;
    if (Date.now() >= deadline) return r.ok ? null : "error";
    await sleep(5_000);
  }
}

function refused(req: { requestId: string; address?: string; did?: string }, reasons: DoorReason[], ctx: DoorContext, detail?: string): Reply {
  return {
    schema: "zerone-door-reply/v1",
    request_id: req.requestId,
    decision: "refused",
    mode: ctx.live ? "live" : "dry-run",
    reasons,
    ...(req.address ? { address: req.address } : {}),
    ...(req.did ? { did: req.did } : {}),
    ...(detail ? { detail } : {}),
    at: ctx.now().toISOString(),
  };
}

export interface Outcome {
  reply: Reply | null;
  /** False for replays: nothing is written or answered for them. */
  answer: boolean;
}

export async function processItem(ctx: DoorContext, item: IntakeItem): Promise<Outcome> {
  const now = ctx.now();
  const parsed = parseDoorRequest(item.text);
  if (!parsed.ok) {
    const reply = refused({ requestId: sha256Hex(item.text) }, ["request_malformed"], ctx, parsed.detail);
    ctx.log({ event: "refused", request_id: reply.request_id, reasons: reply.reasons, channel: channelKeyOf(item.channel) });
    writeReply(ctx, reply);
    return { reply, answer: true };
  }
  const req = parsed.value;
  const ledger = readLedger(ctx);
  if (ledger.seen[req.requestId] !== undefined) {
    ctx.log({ event: "replay_ignored", request_id: req.requestId });
    return { reply: null, answer: false };
  }
  const markSeen = (l: Ledger) => {
    l.seen[req.requestId] = now.toISOString();
  };
  const finish = (reply: Reply): Outcome => {
    const l = readLedger(ctx);
    markSeen(l);
    writeLedger(ctx, l);
    writeReply(ctx, reply);
    ctx.log({ event: reply.decision, request_id: req.requestId, address: req.address, reasons: reply.reasons, tx_hash: reply.tx_hash });
    return { reply, answer: true };
  };

  let allowlist;
  try {
    allowlist = loadAllowlist(ctx.paths.allowlist);
  } catch (error) {
    return finish(refused(req, ["allowlist_unreadable"], ctx, error instanceof Error ? error.message : String(error)));
  }
  const membership = checkMembership(item.channel, allowlist);
  const requestReasons = verifyDoorRequest(req, now, ctx.config.chain_id);
  if (requestReasons.length > 0 || membership.reasons.length > 0) {
    return finish(refused(req, [...requestReasons, ...membership.reasons], ctx));
  }

  const registrar = ctx.config.registrar.address;
  const g = await gatherGlobalFacts(ctx.chain, registrar);
  const s = await gatherSubjectFacts(ctx.chain, req);
  if (!g.facts || !s.facts) {
    return finish(refused(req, ["chain_read_failed"], ctx, [...g.errors, ...s.errors].join("; ").slice(0, 400)));
  }
  const verdict = evaluateGates({
    request: req,
    requestReasons,
    membership,
    registrar,
    chainVersionPin: ctx.config.chain_version,
    knownKeys: ctx.config.known_keys.map((k) => k.address),
    ledger,
    global: g.facts,
    subject: s.facts,
    haltFiles: haltFilesPresent(ctx.paths.haltFiles),
    now,
    chainId: ctx.config.chain_id,
  });
  if (verdict.kind === "refuse") return finish(refused(req, verdict.reasons, ctx));
  const kind = verdict.kind;

  // ── build exactly one transaction and check it before anything else ──
  const workHome = join(ctx.paths.work, req.requestId.slice(0, 16));
  rmSync(workHome, { recursive: true, force: true });
  mkdirSync(workHome, { recursive: true, mode: 0o700 });
  const expiration = grantExpiration(now);
  let unsigned: UnsignedTx;
  try {
    unsigned = kind === "admit"
      ? buildAdmissionTx(ctx.zeroned, workHome, registrar, req.address, expiration)
      : buildRegrantTx(ctx.zeroned, workHome, registrar, req.address, expiration, verdict.revokeFirst);
  } catch (error) {
    return finish(refused(req, ["build_failed"], ctx, error instanceof Error ? error.message : String(error)));
  }
  const problems = checkBuiltTx(unsigned, { kind, registrar, address: req.address, expiration, revokeFirst: verdict.revokeFirst });
  if (problems.length > 0) return finish(refused(req, ["build_failed"], ctx, problems.join(", ")));

  const auth = g.facts.registrarAuth!;
  const signer = { pubKeyB64: auth.pubKeyB64!, sequence: auth.sequence };
  const simTx = withPublicSigner(unsigned, signer);
  const gasLimit = kind === "admit" ? ADMIT_GAS : REGRANT_GAS;
  let simulatedGas: number;
  try {
    const sim = await ctx.chain.simulate(encodeTx(ctx.zeroned, workHome, simTx, "simulate"));
    if (!sim.ok) return finish(refused(req, ["simulate_failed"], ctx, sim.error));
    simulatedGas = sim.value.gasUsed;
  } catch (error) {
    return finish(refused(req, ["simulate_failed"], ctx, error instanceof Error ? error.message : String(error)));
  }
  if (simulatedGas + SIMULATE_MARGIN_GAS > gasLimit) {
    return finish(refused(req, ["gas_over_limit"], ctx, `simulated ${simulatedGas} + ${SIMULATE_MARGIN_GAS} > ${gasLimit}`));
  }

  const base = {
    schema: "zerone-door-reply/v1" as const,
    request_id: req.requestId,
    address: req.address,
    did: req.did,
    kind,
    registrar,
    pot_id: `bootstrap-${req.address}`,
    allowance: allowanceSummary(expiration),
    simulated_gas: simulatedGas,
    gas_limit: gasLimit,
  };

  if (!ctx.live) {
    rmSync(workHome, { recursive: true, force: true });
    return finish({ ...base, decision: "dry_run_ok", mode: "dry-run", reasons: [], at: ctx.now().toISOString() });
  }

  // ── live: sign with R, check what was signed, record, broadcast ──
  // Sign the transaction without the simulation's placeholder signer: `tx
  // sign` appends to existing signatures rather than replacing them.
  const pass = ctx.passphrase(ctx.config.registrar.keychain_service);
  if (pass === null) return finish(refused(req, ["passphrase_unavailable"], ctx));
  let signed: UnsignedTx;
  try {
    signed = signTx(ctx.zeroned, workHome, unsigned, {
      keyName: ctx.config.registrar.key_name,
      keyringDir: ctx.config.registrar.keyring_dir,
      accountNumber: auth.accountNumber,
      sequence: auth.sequence,
      passphrase: pass,
    });
  } catch (error) {
    return finish(refused(req, ["sign_failed"], ctx, error instanceof Error ? error.message : String(error)));
  } finally {
    pass.fill(0);
  }
  const signedProblems = checkSignedTx(signed, simTx, signer);
  if (signedProblems.length > 0) return finish(refused(req, ["signed_tx_mismatch"], ctx, signedProblems.join(", ")));
  const txHash = txHashOf(encodeTx(ctx.zeroned, workHome, signed, "signed-encode"));

  const entry: LedgerEntry = {
    request_id: req.requestId,
    kind,
    address: req.address,
    address_payload: addressPayloadHex(req.address)!,
    identity_public_key: req.identityPublicKeyHex,
    did: req.did,
    channel_key: channelKeyOf(item.channel),
    tx_hash: txHash,
    status: "pending_inclusion",
    sequence: auth.sequence,
    // Until the transaction is found: then the window of its inclusion height.
    chain_window: currentWindow(g.facts.status.height),
    utc_day: utcDay(now),
    at: now.toISOString(),
    grant_expiration: expiration,
  };
  {
    const l = readLedger(ctx);
    markSeen(l);
    l.entries.push(entry);
    writeLedger(ctx, l);
  }
  const live = { ...base, mode: "live" as const, tx_hash: txHash };

  let broadcast;
  try {
    broadcast = broadcastTx(ctx.zeroned, workHome, signed, tcpNode(ctx.config.rpc));
  } catch (error) {
    // The node may or may not have it: stay pending and reconcile later.
    const reply: Reply = { ...live, decision: "pending", reasons: ["broadcast_failed"], detail: error instanceof Error ? error.message : String(error), at: ctx.now().toISOString() };
    writeReply(ctx, reply);
    ctx.log({ event: "pending", request_id: req.requestId, tx_hash: txHash, reasons: reply.reasons });
    return { reply, answer: true };
  }
  if (broadcast.code === 0 && broadcast.txHash !== txHash) {
    // Accepted under a hash the door did not compute: follow the node's hash.
    ctx.log({ event: "hash_differs", request_id: req.requestId, computed: txHash, node: broadcast.txHash });
    entry.tx_hash = broadcast.txHash;
    updateEntry(ctx, req.requestId, { tx_hash: broadcast.txHash });
  }
  if (broadcast.code !== 0) {
    updateEntry(ctx, req.requestId, { status: "rejected", detail: `code ${broadcast.code}: ${broadcast.log.slice(0, 200)}` });
    const reply: Reply = { ...live, decision: "failed", reasons: ["broadcast_rejected"], detail: `code ${broadcast.code}: ${broadcast.log.slice(0, 300)}`, at: ctx.now().toISOString() };
    writeReply(ctx, reply);
    ctx.log({ event: "failed", request_id: req.requestId, tx_hash: txHash, reasons: reply.reasons });
    return { reply, answer: true };
  }
  const reply = await settle(ctx, entry, { ...live, tx_hash: entry.tx_hash });
  rmSync(workHome, { recursive: true, force: true });
  return { reply, answer: true };
}

function updateEntry(ctx: DoorContext, requestId: string, patch: Partial<LedgerEntry>): LedgerEntry | null {
  const l = readLedger(ctx);
  const e = l.entries.find((x) => x.request_id === requestId);
  if (!e) return null;
  Object.assign(e, patch);
  writeLedger(ctx, l);
  return e;
}

/** Waits for inclusion, checks result.code, postchecks, writes the receipt.
 * A transaction found in a block moves to the chain window of its inclusion
 * height, the window the chain itself counts the admission in, whichever run
 * finds it. */
async function settle(ctx: DoorContext, entry: LedgerEntry, base: Omit<Reply, "decision" | "reasons" | "at">): Promise<Reply> {
  const included = await awaitInclusion(ctx, entry.tx_hash);
  if (included === null || included === "error") {
    const reply: Reply = { ...base, decision: "pending", reasons: ["inclusion_timeout"], at: ctx.now().toISOString() } as Reply;
    writeReply(ctx, reply);
    ctx.log({ event: "pending", request_id: entry.request_id, tx_hash: entry.tx_hash, reasons: reply.reasons });
    return reply;
  }
  const placed = { height: included.height, chain_window: currentWindow(included.height) };
  if (included.code !== 0) {
    updateEntry(ctx, entry.request_id, { status: "delivery_failed", ...placed, detail: included.log.slice(0, 200) });
    const reply: Reply = { ...base, decision: "failed", reasons: ["delivery_failed"], height: included.height, detail: included.log.slice(0, 300), at: ctx.now().toISOString() } as Reply;
    writeReply(ctx, reply);
    ctx.log({ event: "failed", request_id: entry.request_id, tx_hash: entry.tx_hash, reasons: reply.reasons });
    return reply;
  }
  const problems = await postcheck(ctx, entry);
  if (problems.length > 0) {
    updateEntry(ctx, entry.request_id, { status: "postcheck_failed", ...placed, detail: problems.join(", ") });
    const reply: Reply = { ...base, decision: "failed", reasons: ["postcheck_failed"], height: included.height, detail: problems.join(", "), at: ctx.now().toISOString() } as Reply;
    writeReply(ctx, reply);
    ctx.log({ event: "postcheck_failed", request_id: entry.request_id, tx_hash: entry.tx_hash, detail: problems });
    return reply;
  }
  updateEntry(ctx, entry.request_id, { status: "included", ...placed });
  const reply: Reply = {
    ...base,
    decision: entry.kind === "admit" ? "admitted" : "regranted",
    reasons: [],
    height: included.height,
    at: ctx.now().toISOString(),
  } as Reply;
  writeReply(ctx, reply);
  ctx.log({ event: reply.decision, request_id: entry.request_id, address: entry.address, tx_hash: entry.tx_hash, height: included.height });
  return reply;
}

/** Live runs first settle anything a previous run left pending. Nothing is
 * ever re-broadcast: a pending entry is only looked up. */
export async function reconcilePending(ctx: DoorContext): Promise<void> {
  if (!ctx.live) return;
  const pending = readLedger(ctx).entries.filter((e) => e.status === "pending_inclusion");
  for (const entry of pending) {
    const base = {
      schema: "zerone-door-reply/v1" as const,
      request_id: entry.request_id,
      mode: "live" as const,
      address: entry.address,
      did: entry.did,
      kind: entry.kind,
      registrar: ctx.config.registrar.address,
      pot_id: `bootstrap-${entry.address}`,
      allowance: allowanceSummary(entry.grant_expiration),
      tx_hash: entry.tx_hash,
    };
    const reply = await settle({ ...ctx, inclusionTimeoutMs: 0 }, entry, base);
    if (reply.decision !== "pending" || entry.sequence === undefined) continue;
    // Still unseen. Once R's sequence on chain is still the one this tx was
    // signed with, well after broadcast, the tx was dropped and can never land
    // (any later R transaction would take that sequence first).
    const ageMs = ctx.now().getTime() - Date.parse(entry.at);
    if (ageMs < DROP_AFTER_MS) continue;
    const status = await ctx.chain.status();
    const chainMoving = status.ok && (ctx.now().getTime() - Date.parse(status.value.time)) / 1000 <= CHAIN_STALE_SECONDS;
    const r = await ctx.chain.authAccount(ctx.config.registrar.address);
    if (!chainMoving || !r.ok || r.value === null || r.value.sequence !== entry.sequence) {
      ctx.log({ event: "pending_needs_attention", request_id: entry.request_id, tx_hash: entry.tx_hash });
      continue;
    }
    updateEntry(ctx, entry.request_id, { status: "dropped", detail: "never included; R's sequence did not move" });
    writeReply(ctx, { ...base, decision: "failed", reasons: ["dropped"], at: ctx.now().toISOString() } as Reply);
    ctx.log({ event: "dropped", request_id: entry.request_id, tx_hash: entry.tx_hash });
  }
}

/** How long an unseen broadcast stays pending before the door checks whether
 * it was dropped (sixty 30-second blocks). */
export const DROP_AFTER_MS = 30 * 60 * 1000;

export function replyText(reply: Reply): string {
  if (reply.decision === "admitted" || reply.decision === "regranted") {
    return `zerone-door: ${reply.decision} ${reply.address} in tx ${reply.tx_hash} at height ${reply.height}. ` +
      `Seed pot ${reply.pot_id}; allowance from ${reply.registrar} until ${reply.allowance?.expiration}.`;
  }
  if (reply.decision === "pending") return `zerone-door: ${reply.address} pending in tx ${reply.tx_hash}; check the chain before asking again.`;
  return `zerone-door: ${reply.decision} request ${reply.request_id.slice(0, 16)}: ${reply.reasons.join(", ") || "no reason given"}.`;
}

export function globalSummary(facts: GlobalFacts): Record<string, unknown> {
  return {
    height: facts.status.height,
    block_time: facts.status.time,
    registrar_param: facts.params.registrar,
    window_cap: facts.params.windowCap,
    upgrade_pending: facts.upgradePending,
    chain_version: {
      last_upgrade: facts.chainVersion.lastUpgrade?.name ?? null,
      last_upgrade_height: facts.chainVersion.lastUpgrade?.height ?? null,
      module_versions_sha256: facts.chainVersion.moduleVersionsSha256,
    },
    registrar_spendable_uzrn: facts.registrarSpendable.toString(),
    registrar_registered: facts.registrarZerone !== null,
    registrar_frozen: facts.registrarZerone?.frozen ?? null,
    registrar_grants: facts.registrarIssued.length,
  };
}

export function hasReply(ctx: DoorContext, requestId: string): boolean {
  const dir = ctx.live ? ctx.paths.replies : join(ctx.paths.replies, "dry-run");
  return existsSync(join(dir, `${requestId}.json`));
}

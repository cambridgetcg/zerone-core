// The run loop for one request: verify -> membership -> read the chain ->
// gates -> build -> check -> simulate -> (live only) sign -> check -> record
// -> broadcast with a long inclusion wait and a result.code check ->
// postcheck -> reply. Dry run is the default and stops after simulate.
import { existsSync, mkdirSync, renameSync, rmSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { addressPayloadHex } from "./bech32.ts";
import type { ChainReader, Grant, IncludedTx, TxMessage } from "./chain.ts";
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
  INCLUSION_TIMEOUT_MS,
  MSG_ADD_BOOTSTRAP_ENTRY,
  MSG_GRANT_ALLOWANCE,
  MSG_REVOKE_ALLOWANCE,
  PENDING_ATTENTION_MS,
  REGRANT_GAS,
  SIMULATE_MARGIN_GAS,
  TIMEOUT_SETTLE_BLOCKS,
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
  timeoutHeightFor,
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
  /** In a block with a non-zero code. Final (decision "failed") once R's
   * sequence moved past the one it was signed with, or once the chain passed
   * its timeout height with R's sequence unmoved; until then (decision
   * "pending") the block refused it before it ran, and the same transaction
   * can still run in a later block. */
  | "delivery_failed"
  | "postcheck_failed"
  /** The chain passed the transaction's timeout height and R's sequence never
   * moved past the one it was signed with: no block ran it, none can. */
  | "expired"
  /** Another transaction used the R sequence it was signed with. */
  | "superseded";

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
  /** The last block height that can run the transaction. */
  timeout_height?: number;
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
async function awaitInclusion(ctx: DoorContext, txHash: string): Promise<IncludedTx | null | "error"> {
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
  // The transaction can run only up to this height, so a lost broadcast can
  // later be proven gone instead of guessed gone.
  const timeoutHeight = timeoutHeightFor(g.facts.status.height);
  let unsigned: UnsignedTx;
  try {
    unsigned = kind === "admit"
      ? buildAdmissionTx(ctx.zeroned, workHome, registrar, req.address, expiration, timeoutHeight)
      : buildRegrantTx(ctx.zeroned, workHome, registrar, req.address, expiration, verdict.revokeFirst, timeoutHeight);
  } catch (error) {
    return finish(refused(req, ["build_failed"], ctx, error instanceof Error ? error.message : String(error)));
  }
  const problems = checkBuiltTx(unsigned, { kind, registrar, address: req.address, expiration, revokeFirst: verdict.revokeFirst, timeoutHeight });
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
    timeout_height: timeoutHeight,
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
  const live = { ...base, mode: "live" as const, tx_hash: txHash, timeout_height: timeoutHeight };

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

/** Waits for inclusion within this run, then records what was found. A
 * transaction not seen in time stays pending: waiting proves nothing. One
 * seen with a non-zero code goes to fateOf, which alone can say whether it
 * can still run. */
async function settle(ctx: DoorContext, entry: LedgerEntry, base: ReplyBase): Promise<Reply> {
  const included = await awaitInclusion(ctx, entry.tx_hash);
  if (included === null || included === "error") {
    const reply: Reply = { ...base, decision: "pending", reasons: ["inclusion_timeout"], at: ctx.now().toISOString() };
    writeReply(ctx, reply);
    ctx.log({ event: "pending", request_id: entry.request_id, tx_hash: entry.tx_hash, reasons: reply.reasons, timeout_height: entry.timeout_height });
    return reply;
  }
  if (included.code === 0) return recordInclusion(ctx, entry, included, base);
  // With `answer`, applyFate always writes and returns a reply.
  return (await applyFate(ctx, entry, await fateOf(ctx, entry), base, true))!;
}

type ReplyBase = Omit<Reply, "decision" | "reasons" | "at">;

/** Records a transaction that ran (code 0), or one whose non-zero code is
 * final because R's sequence moved past the one it was signed with, so it can
 * never run again: checks result.code, postchecks, and moves the entry to the
 * chain window of its inclusion height, the window the chain itself counts
 * the admission in. Only fateOf decides that a non-zero code is final. */
async function recordInclusion(ctx: DoorContext, entry: LedgerEntry, included: IncludedTx, base: ReplyBase): Promise<Reply> {
  const placed = { height: included.height, chain_window: currentWindow(included.height) };
  if (included.code !== 0) {
    updateEntry(ctx, entry.request_id, { status: "delivery_failed", ...placed, detail: included.log.slice(0, 200) });
    const reply: Reply = { ...base, decision: "failed", reasons: ["delivery_failed"], height: included.height, detail: included.log.slice(0, 300), at: ctx.now().toISOString() };
    writeReply(ctx, reply);
    ctx.log({ event: "failed", request_id: entry.request_id, tx_hash: entry.tx_hash, reasons: reply.reasons, height: included.height });
    return reply;
  }
  const problems = await postcheck(ctx, entry);
  if (problems.length > 0) {
    updateEntry(ctx, entry.request_id, { status: "postcheck_failed", ...placed, detail: problems.join(", ") });
    const reply: Reply = { ...base, decision: "failed", reasons: ["postcheck_failed"], height: included.height, detail: problems.join(", "), at: ctx.now().toISOString() };
    writeReply(ctx, reply);
    ctx.log({ event: "postcheck_failed", request_id: entry.request_id, tx_hash: entry.tx_hash, detail: problems, height: included.height });
    return reply;
  }
  updateEntry(ctx, entry.request_id, { status: "included", ...placed });
  const reply: Reply = {
    ...base,
    decision: entry.kind === "admit" ? "admitted" : "regranted",
    reasons: [],
    height: included.height,
    at: ctx.now().toISOString(),
  };
  writeReply(ctx, reply);
  ctx.log({ event: reply.decision, request_id: entry.request_id, address: entry.address, tx_hash: entry.tx_hash, height: included.height });
  return reply;
}

function liveBase(ctx: DoorContext, entry: LedgerEntry): ReplyBase {
  return {
    schema: "zerone-door-reply/v1",
    request_id: entry.request_id,
    mode: "live",
    address: entry.address,
    did: entry.did,
    kind: entry.kind,
    registrar: ctx.config.registrar.address,
    pot_id: `bootstrap-${entry.address}`,
    allowance: allowanceSummary(entry.grant_expiration),
    tx_hash: entry.tx_hash,
    ...(entry.timeout_height !== undefined ? { timeout_height: entry.timeout_height } : {}),
  };
}

/** Whether a transaction's messages are this entry's own: the bootstrap entry
 * for its address and the grant from R to it (for a re-grant, the grant,
 * after a revoke when one was needed). The door's transaction carries exactly
 * these, so one that does is this admission even under a hash the door did
 * not compute; recordInclusion's postcheck then compares what it left on
 * chain with what the door built. */
export function carriesEntry(messages: readonly TxMessage[], entry: Pick<LedgerEntry, "kind" | "address">, registrar: string): boolean {
  const fromR = (m: TxMessage | undefined) => m?.granter === registrar && m.grantee === entry.address;
  const types = JSON.stringify(messages.map((m) => m.type));
  if (entry.kind === "admit") {
    return types === JSON.stringify([MSG_ADD_BOOTSTRAP_ENTRY, MSG_GRANT_ALLOWANCE])
      && messages[0]!.authority === registrar
      && JSON.stringify(messages[0]!.addresses) === JSON.stringify([entry.address])
      && fromR(messages[1]);
  }
  return (types === JSON.stringify([MSG_GRANT_ALLOWANCE]) && fromR(messages[0]))
    || (types === JSON.stringify([MSG_REVOKE_ALLOWANCE, MSG_GRANT_ALLOWANCE]) && fromR(messages[0]) && fromR(messages[1]));
}

/** What became of a broadcast the door has not yet seen settle. A found
 * transaction carries the hash the chain knows it by. `refusedIn` is a block
 * that holds the transaction with a non-zero code while R's sequence has not
 * moved past the one it was signed with: its ante handler refused it, nothing
 * in it ran, and the same signed transaction can still run in a later block
 * up to its timeout height. */
type Fate =
  | { kind: "found"; tx: IncludedTx; hash: string }
  | { kind: "expired"; nodeHeight: number; readHeight: number; registrarSequence: string; refusedIn: IncludedTx | null }
  | { kind: "superseded"; by: string; height: number }
  | { kind: "unproven"; detail: string; attention: boolean; refusedIn: IncludedTx | null };

/** Proves what became of a pending transaction, or says that it cannot yet.
 * Only proof frees an entry:
 *  - found: the transaction ran in a block (code 0), found by its hash or as
 *    the transaction the chain's tx.acc_seq index names for its R sequence
 *    (matched by hash, or by carrying this entry's messages when the chain
 *    knows it under a hash the door did not compute); or it is in a block
 *    with a non-zero code and R's sequence moved past the one it was signed
 *    with, so it can never run again;
 *  - expired: the node has blocks TIMEOUT_SETTLE_BLOCKS past its timeout
 *    height, and R's sequence, read from state at or past the timeout height,
 *    never moved past the one it was signed with, so no block up to the
 *    timeout height ran it and the ante handler refuses it in every later
 *    block (if a block refused it on the way, it is recorded as that
 *    delivery failure);
 *  - superseded: R's sequence moved past it and the acc_seq index names
 *    another transaction that used that sequence, one whose messages are not
 *    this entry's.
 * A non-zero code alone proves nothing final, elapsed time proves nothing,
 * and nothing is ever re-broadcast. */
async function fateOf(ctx: DoorContext, entry: LedgerEntry): Promise<Fate> {
  const registrar = ctx.config.registrar.address;
  // The node's height first: every read after it sees at least those blocks.
  const status = await ctx.chain.status();
  const byHash = await ctx.chain.txByHash(entry.tx_hash);
  if (!byHash.ok) return { kind: "unproven", detail: byHash.error, attention: false, refusedIn: null };
  if (byHash.value !== null && byHash.value.code === 0) return { kind: "found", tx: byHash.value, hash: entry.tx_hash };
  // In a block with a non-zero code: whether it ran (and failed) or was
  // refused before it ran, and so can still run, R's sequence decides below.
  const refusedIn = byHash.value;
  const seen = refusedIn ? `in block ${refusedIn.height} with code ${refusedIn.code} (${refusedIn.log.slice(0, 120)}); ` : "";
  const unproven = (detail: string, attention = false): Fate => ({ kind: "unproven", detail: `${seen}${detail}`, attention, refusedIn });
  if (!status.ok) return unproven(status.error);
  if (entry.sequence === undefined || !/^\d+$/.test(entry.sequence)) return unproven("no signing sequence recorded", true);
  // R's sequence from the RPC node, with the height of the state it was read
  // in. A REST read (config.rest) may come from another, lagging node, and
  // its height comes only in a response header.
  const r = await ctx.chain.accountAtHeight(registrar);
  if (!r.ok) return unproven(r.error);
  if (r.value.account === null) return unproven(`R has no account on chain at height ${r.value.height}`, true);
  const account = r.value.account;
  const pastTimeout = entry.timeout_height !== undefined && status.value.height >= entry.timeout_height + TIMEOUT_SETTLE_BLOCKS;
  if (BigInt(account.sequence) > BigInt(entry.sequence)) {
    const used = await ctx.chain.txsBySignerSequence(registrar, entry.sequence);
    if (!used.ok) return unproven(used.error);
    // This transaction: by its hash, or by its messages when the chain
    // indexed it under a hash the door did not compute (one whose broadcast
    // answer was lost, say). A hash alone never makes it another's.
    const mine = used.value.find((t) => t.hash === entry.tx_hash.toUpperCase())
      ?? used.value.find((t) => carriesEntry(t.messages, entry, registrar));
    if (mine) return { kind: "found", tx: mine, hash: mine.hash };
    // In a block with a non-zero code, and the index names who used the
    // sequence it needs. With nothing indexed yet, the same bytes may have
    // run in a later block the index has not caught up with: stay unproven.
    if (refusedIn && used.value.length > 0) return { kind: "found", tx: refusedIn, hash: entry.tx_hash };
    if (used.value.length === 1) return { kind: "superseded", by: used.value[0]!.hash, height: used.value[0]!.height };
    // The index may not have the block yet (or the node may not index acc_seq).
    return unproven(`R's sequence moved past ${entry.sequence}, and the acc_seq index shows ${used.value.length} transactions for it`,
      pastTimeout || entry.timeout_height === undefined);
  }
  if (entry.timeout_height === undefined) {
    return unproven("no timeout height: it settles only when found or when R's sequence moves past it", true);
  }
  if (!pastTimeout) {
    return unproven(refusedIn
      ? `R's sequence is still ${account.sequence}, so nothing in it ran and it can still run until height ${entry.timeout_height}; the node is at ${status.value.height}`
      : `can still land until height ${entry.timeout_height}; the node is at ${status.value.height}`);
  }
  // The proof needs a state that includes the last block that could run it.
  if (r.value.height < entry.timeout_height) {
    return unproven(`R's sequence was read at height ${r.value.height}, below the timeout height ${entry.timeout_height}`);
  }
  return { kind: "expired", nodeHeight: status.value.height, readHeight: r.value.height, registrarSequence: account.sequence, refusedIn };
}

/** Records what fateOf proved. An entry proven neither way stays pending;
 * on the run that broadcast it (`answer`), its pending reply is written. A
 * pending entry asks for a person (pending_needs_attention) on every run when
 * fateOf says it cannot settle on its own, and on every run once the door's
 * clock is PENDING_ATTENTION_MS past its signing, whatever keeps it unproven:
 * failed reads, a stalled chain, slow blocks. Time never settles it. */
async function applyFate(ctx: DoorContext, entry: LedgerEntry, fate: Fate, base: ReplyBase, answer: boolean): Promise<Reply | null> {
  if (fate.kind === "found") {
    if (fate.hash.toUpperCase() === entry.tx_hash.toUpperCase()) return recordInclusion(ctx, entry, fate.tx, base);
    // Its messages are this entry's, under a hash the door did not compute:
    // follow the chain's hash, as the broadcast path does with the node's.
    ctx.log({ event: "hash_differs", request_id: entry.request_id, computed: entry.tx_hash, chain: fate.hash });
    updateEntry(ctx, entry.request_id, { tx_hash: fate.hash });
    return recordInclusion(ctx, { ...entry, tx_hash: fate.hash }, fate.tx, { ...base, tx_hash: fate.hash });
  }
  if (fate.kind === "unproven") {
    const pendingMs = ctx.now().getTime() - Date.parse(entry.at);
    // An entry whose signing time cannot be read counts as overdue.
    const overdue = !(pendingMs < PENDING_ATTENTION_MS);
    if (fate.attention || overdue) {
      ctx.log({ event: "pending_needs_attention", request_id: entry.request_id, tx_hash: entry.tx_hash, signed_at: entry.at, overdue, detail: fate.detail });
    }
    if (!answer) return null;
    const reply: Reply = {
      ...base,
      decision: "pending",
      reasons: [fate.refusedIn ? "delivery_failed" : "inclusion_timeout"],
      detail: fate.detail,
      at: ctx.now().toISOString(),
    };
    updateEntry(ctx, entry.request_id, { detail: fate.detail });
    writeReply(ctx, reply);
    ctx.log({ event: "pending", request_id: entry.request_id, tx_hash: entry.tx_hash, reasons: reply.reasons, timeout_height: entry.timeout_height });
    return reply;
  }
  let status: "delivery_failed" | "expired" | "superseded";
  let detail: string;
  let placed: { height: number; chain_window: number } | null = null;
  if (fate.kind === "expired") {
    const proof = `not run by its timeout height ${entry.timeout_height}: the node is at ${fate.nodeHeight} and R's sequence is still ${fate.registrarSequence} at height ${fate.readHeight}`;
    if (fate.refusedIn) {
      // A block refused it before it ran, and no block can run it now.
      status = "delivery_failed";
      placed = { height: fate.refusedIn.height, chain_window: currentWindow(fate.refusedIn.height) };
      detail = `refused in block ${fate.refusedIn.height} with code ${fate.refusedIn.code} (${fate.refusedIn.log.slice(0, 120)}) before it ran, and ${proof}`;
    } else {
      status = "expired";
      detail = proof;
    }
  } else {
    status = "superseded";
    detail = `R's sequence ${entry.sequence} was used by tx ${fate.by} at height ${fate.height}, whose messages are not this ${entry.kind === "admit" ? "admission" : "re-grant"}`;
  }
  updateEntry(ctx, entry.request_id, { status, detail, ...(placed ?? {}) });
  const reply: Reply = { ...base, decision: "failed", reasons: [status], ...(placed ? { height: placed.height } : {}), detail, at: ctx.now().toISOString() };
  writeReply(ctx, reply);
  ctx.log({ event: status, request_id: entry.request_id, tx_hash: entry.tx_hash, detail });
  return reply;
}

/** Ledgers from before timeout heights marked a broadcast unseen for 30
 * minutes "dropped", which proved nothing: that transaction had no timeout
 * height and could still land. Each one is re-opened as pending, so it counts
 * toward the caps and the admittee watch and keeps the door busy again, and
 * its receipt says pending until fateOf proves what became of it. */
function reopenDropped(ctx: DoorContext): void {
  const l = readLedger(ctx);
  const dropped = l.entries.filter((e) => e.status === "dropped");
  if (dropped.length === 0) return;
  for (const e of dropped) {
    e.status = "pending_inclusion";
    e.detail = "re-opened: marked dropped on a 30-minute clock, never proven unable to land";
  }
  writeLedger(ctx, l);
  for (const e of dropped) {
    writeReply(ctx, { ...liveBase(ctx, e), decision: "pending", reasons: ["inclusion_timeout"], detail: e.detail, at: ctx.now().toISOString() });
    ctx.log({ event: "dropped_reopened", request_id: e.request_id, tx_hash: e.tx_hash });
  }
}

/** Live runs first settle anything a previous run left pending, including
 * anything an older ledger marked dropped. Nothing is ever re-broadcast: a
 * pending entry is only looked up, and it leaves pending only on proof
 * (fateOf). While one is pending the door admits no one else (door_busy). */
export async function reconcilePending(ctx: DoorContext): Promise<void> {
  if (!ctx.live) return;
  reopenDropped(ctx);
  const pending = readLedger(ctx).entries.filter((e) => e.status === "pending_inclusion");
  for (const entry of pending) await applyFate(ctx, entry, await fateOf(ctx, entry), liveBase(ctx, entry), false);
}

export function replyText(reply: Reply): string {
  if (reply.decision === "admitted" || reply.decision === "regranted") {
    return `zerone-door: ${reply.decision} ${reply.address} in tx ${reply.tx_hash} at height ${reply.height}. ` +
      `Seed pot ${reply.pot_id}; allowance from ${reply.registrar} until ${reply.allowance?.expiration}.`;
  }
  if (reply.decision === "pending") {
    const seen = reply.reasons.includes("delivery_failed") ? "a block refused it before it ran" : "not seen in a block yet";
    const until = reply.timeout_height !== undefined ? `; it can still run until height ${reply.timeout_height}` : "";
    return `zerone-door: ${reply.address} pending in tx ${reply.tx_hash} (${seen}${until}). ` +
      "The door settles it only when the chain shows whether it ran; check the chain before asking again.";
  }
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

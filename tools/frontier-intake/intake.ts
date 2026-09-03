#!/usr/bin/env bun
// frontier-intake — absorb register entries into the zerone knowledge corpus.
// Client-side only: shells the zeroned CLI, touches no consensus code.
// See docs/specs/math-frontier-absorption-v0.md.
import { parseArgs } from "node:util";
import {
  chmodSync,
  closeSync,
  existsSync,
  mkdirSync,
  openSync,
  readFileSync,
  renameSync,
  statSync,
  unlinkSync,
  writeFileSync,
  writeSync,
} from "node:fs";
import { dirname } from "node:path";
import {
  DEFAULT_REVIEW_FEE_UZRN,
  GAS_SEND,
  NETWORKS,
  SUBMITTER_FUND_UZRN,
  VERIFIER_FUND_UZRN,
  VERIFIER_TOPUP_FLOOR_UZRN,
  defaultRegisterPath,
  statePath,
  type NetworkConfig,
} from "./src/config.ts";
import {
  admissibleEntries,
  firstBatch,
  loadRegister,
  validateRegister,
  type RegisterEntry,
} from "./src/register.ts";
import {
  classifyClaimStatus,
  commitHash,
  decideActions,
  newSalt,
  type PhaseParams,
  type RoundState,
} from "./src/panel.ts";
import {
  accountRegistrationCommandArgs,
  loadPersistedIdentityKey,
  loadOrCreatePersistedIdentityKey,
  type PersistedIdentityKey,
} from "./src/account-registration.ts";
import {
  classifyRegisteredAccountQuery,
  reconcileRegisteredAccount,
  type ClassifiedAccountQuery,
} from "./src/account-reconciliation.ts";
import {
  CLAIM_GAS,
  COMMIT_GAS,
  REVEAL_GAS,
  broadcast,
  eventAttr,
  keyAddress,
  latestHeight,
  query,
  run,
  spendableUzrn,
  waitForTx,
} from "./src/zeroned.ts";

interface IntakeState {
  schema: "zerone.frontier-intake-state/v0";
  network: string;
  rounds: RoundState[];
}

// ── state: exclusive lock + read-merge-write so concurrent submit/panel
//    processes never drop each other's rounds or votes ─────────────────────
function withStateLock<T>(net: NetworkConfig, fn: () => T): T {
  const lockPath = `${statePath(net.name)}.lock`;
  const deadline = Date.now() + 30_000;
  for (;;) {
    try {
      mkdirSync(dirname(lockPath), { recursive: true, mode: 0o700 });
      const fd = openSync(lockPath, "wx");
      writeSync(fd, `${process.pid}\n`);
      closeSync(fd);
      break;
    } catch {
      try {
        if (Date.now() - statSync(lockPath).mtimeMs > 120_000) {
          unlinkSync(lockPath);
          continue;
        }
      } catch {
        continue;
      }
      if (Date.now() >= deadline) throw new Error(`could not acquire state lock ${lockPath}`);
      Bun.sleepSync(100);
    }
  }
  try {
    return fn();
  } finally {
    try { unlinkSync(lockPath); } catch { /* stale takeover */ }
  }
}

function loadStateUnlocked(net: NetworkConfig): IntakeState {
  const path = statePath(net.name);
  if (!existsSync(path)) return { schema: "zerone.frontier-intake-state/v0", network: net.name, rounds: [] };
  return JSON.parse(readFileSync(path, "utf8")) as IntakeState;
}

function saveStateUnlocked(net: NetworkConfig, state: IntakeState): void {
  const path = statePath(net.name);
  mkdirSync(dirname(path), { recursive: true, mode: 0o700 });
  const tmp = `${path}.tmp-${process.pid}`;
  writeFileSync(tmp, `${JSON.stringify(state, null, 2)}\n`, { mode: 0o600 });
  renameSync(tmp, path);
  chmodSync(path, 0o600);
}

/** Merge one round back into the on-disk state by round_id, under the lock. */
function upsertRound(net: NetworkConfig, round: RoundState): void {
  withStateLock(net, () => {
    const state = loadStateUnlocked(net);
    const index = state.rounds.findIndex(candidate => candidate.round_id === round.round_id);
    if (index === -1) state.rounds.push(round);
    else state.rounds[index] = round;
    saveStateUnlocked(net, state);
  });
}

function readRounds(net: NetworkConfig): RoundState[] {
  return withStateLock(net, () => loadStateUnlocked(net).rounds);
}

function fail(message: string, code = 1): never {
  console.error(`frontier-intake: ${message}`);
  process.exit(code);
}

function requireNetwork(name?: string): NetworkConfig {
  if (!name) fail("--network is required (zerone-1 | zerone-testnet-1)");
  const net = NETWORKS[name];
  if (!net) fail(`unknown network "${name}"`);
  return net;
}

// Broadcast-guard discipline (cf. tools/agenttool-relay validateRelayChainID):
// every broadcasting command requires an explicit acknowledgment naming the
// exact live chain, checked before any key creation or CLI invocation.
function requireLiveAck(net: NetworkConfig, ack?: string): void {
  if (ack !== net.chain_id) {
    fail(`broadcasting to ${net.chain_id} requires --live-ack=${net.chain_id} (explicit live-network authorization)`);
  }
}

function phaseParams(net: NetworkConfig): PhaseParams {
  const parsed = query(net, ["query", "knowledge", "params"]);
  const params = (parsed?.params ?? parsed) as Record<string, unknown>;
  const commit = Number(params?.commit_phase_blocks ?? 0);
  const reveal = Number(params?.reveal_phase_blocks ?? 0);
  const aggregation = Number(params?.aggregation_phase_blocks ?? params?.aggregation_blocks ?? 50);
  if (!commit || !reveal) fail(`could not read phase params from ${net.rpc}`);
  return { commit_blocks: commit, reveal_blocks: reveal, aggregation_blocks: aggregation };
}

function commandValidate(registerPath: string): void {
  const register = loadRegister(registerPath);
  const issues = validateRegister(register);
  const asserts = admissibleEntries(register).length;
  console.log(`register ${register.snapshotDate}: ${register.entries.length} entries, ${asserts} assert-eligible, ${firstBatch(register).length} in first batch`);
  if (issues.length === 0) {
    console.log("🟢 register valid");
    return;
  }
  for (const issue of issues) console.log(`🔴 ${issue.entry}: ${issue.problem}`);
  process.exit(2);
}

function commandPlan(net: NetworkConfig, registerPath: string): void {
  const register = loadRegister(registerPath);
  const issues = validateRegister(register);
  if (issues.length > 0) fail(`register invalid (${issues.length} issues); run validate`, 2);
  const done = new Set(readRounds(net).map(round => round.entry_id));
  const fees = query(net, ["query", "knowledge", "effective-fees"]);
  console.log(`effective fees: ${JSON.stringify(fees ?? "unavailable")}`);
  for (const entry of admissibleEntries(register)) {
    const already = done.has(entry.id) ? " [already in ledger]" : "";
    const novelty = query(net, ["query", "knowledge", "check-novelty", entry.chain.domain, entry.chain.subject, entry.statement.slice(0, 100)]);
    const score = novelty?.novelty_score ?? novelty?.score ?? "?";
    const batch = entry.chain.firstBatch ? ` batch#${entry.chain.firstBatch}` : "";
    console.log(`  ${entry.id}${batch}: novelty ${score}${already}`);
  }
}

async function submitEntry(
  net: NetworkConfig,
  entry: RegisterEntry,
  feeUzrn: number,
  vote: "accept" | "reject",
  factIds: Map<string, string> = new Map(),
): Promise<RoundState> {
  const txArgs = [
    "knowledge", "submit-claim",
    entry.statement, entry.chain.domain, entry.chain.category, String(feeUzrn),
    "--claim-type", "assertion",
    "--subject", entry.chain.subject,
    "--predicate", entry.chain.predicate,
    "--tags", entry.chain.tags,
  ];
  const relations = relationsFlag(entry, factIds);
  if (relations) txArgs.push("--relations", relations);
  const outcome = broadcast(net, net.submitter, CLAIM_GAS, txArgs);
  if (!outcome.ok || !outcome.tx_hash) {
    throw new Error(`submit-claim rejected for ${entry.id}: ${outcome.raw_log ?? outcome.detail}`);
  }
  const verifiers: RoundState["verifiers"] = {};
  for (const name of net.verifiers) verifiers[name] = { salt: newSalt() };
  const included = await waitForTx(net, outcome.tx_hash);
  if (!included.found) {
    // CheckTx accepted it: the claim is almost certainly live with a running
    // commit window. Persist honestly instead of losing track of it.
    return {
      entry_id: entry.id,
      claim_id: "",
      round_id: `pending:${outcome.tx_hash}`,
      submit_height: 0,
      submit_tx: outcome.tx_hash,
      vote,
      verifiers,
      stage: "pending_inclusion",
      notes: [`submit tx not observed within wait window; backfill via panel`],
    };
  }
  if (included.code !== 0) throw new Error(`submit tx failed on-chain code ${included.code}: ${included.raw_log}`);
  const claimId = eventAttr(included, /submit_claim/, "claim_id");
  const roundId = eventAttr(included, /verification_round|round_created/, "round_id")
    ?? eventAttr(included, /./, "round_id");
  if (!claimId || !roundId) throw new Error(`could not parse claim_id/round_id from tx ${outcome.tx_hash}`);
  return {
    entry_id: entry.id,
    claim_id: claimId,
    round_id: roundId,
    submit_height: included.height as number,
    submit_tx: outcome.tx_hash,
    vote,
    verifiers,
    stage: "submitted",
    notes: [],
  };
}

function backfillPendingInclusion(net: NetworkConfig, round: RoundState): void {
  const included = run(["query", "tx", round.submit_tx, "--node", net.rpc, "--output", "json"]);
  if (included.exit_code !== 0) return;
  try {
    const parsed = JSON.parse(included.stdout.slice(included.stdout.indexOf("{"))) as Record<string, unknown>;
    const tx = { found: true, height: Number(parsed.height ?? 0), code: Number(parsed.code ?? 0), events: parsed.events as never };
    if (tx.code !== 0) {
      round.stage = "error";
      round.notes.push(`submit tx failed on-chain code ${tx.code}`);
      return;
    }
    const claimId = eventAttr(tx, /submit_claim/, "claim_id");
    const roundId = eventAttr(tx, /verification_round|round_created/, "round_id");
    if (claimId && roundId) {
      round.claim_id = claimId;
      round.round_id = roundId;
      round.submit_height = tx.height;
      round.stage = "submitted";
      round.notes.push("backfilled from included tx");
    }
  } catch {
    // stays pending; next pass retries
  }
}

/** Facts get fresh ids at creation; find ours by content prefix match. */
function discoverFactId(net: NetworkConfig, round: RoundState): string | null {
  const register = loadRegister(defaultRegisterPath());
  const entry = register.entries.find(candidate => candidate.id === round.entry_id);
  if (!entry) return null;
  const parsed = query(net, ["query", "knowledge", "facts"]);
  const facts = (parsed?.facts ?? []) as Array<{ id?: string; content?: string; domain?: string }>;
  const prefix = entry.statement.slice(0, 80);
  const match = facts.filter(fact => fact.domain === entry.chain.domain && (fact.content ?? "").startsWith(prefix));
  // Ambiguity (e.g. near-duplicate statements) is worse than absence.
  return match.length === 1 ? (match[0]?.id ?? null) : null;
}

/**
 * Resolve an entry's curated relations to the on-chain --relations flag.
 * entry:<id> targets resolve through fact ids recorded in the ledger; an
 * unresolved dependency is an ordering error, not something to skip silently.
 */
function relationsFlag(entry: RegisterEntry, factIds: Map<string, string>): string | null {
  const relations = entry.chain.relations ?? [];
  if (relations.length === 0) return null;
  const parts: string[] = [];
  for (const relation of relations) {
    let target = relation.target;
    if (target.startsWith("entry:")) {
      const resolved = factIds.get(target.slice(6));
      if (!resolved) {
        throw new Error(`relation target ${target} has no recorded fact id yet — submit and resolve it first`);
      }
      target = resolved;
    }
    parts.push(`${relation.relation}:${target}`);
  }
  return parts.join(",");
}

function recordedFactIds(rounds: RoundState[]): Map<string, string> {
  const map = new Map<string, string>();
  for (const round of rounds) {
    if (round.stage === "resolved_accepted" && round.fact_id) map.set(round.entry_id, round.fact_id);
  }
  return map;
}

const DUPLICATE_VOTE = /duplicate commitment|duplicate reveal|already committed|already revealed/i;

/** One idempotent pass over every open round; safe to run concurrently with submit. */
async function panelPass(net: NetworkConfig, params: PhaseParams): Promise<{ open: number; earliestWait: number }> {
  const rounds = readRounds(net);
  let open = 0;
  let earliestWait = Number.MAX_SAFE_INTEGER;
  for (const round of rounds) {
    if (round.stage === "pending_inclusion") {
      backfillPendingInclusion(net, round);
      upsertRound(net, round);
    }
    if (round.stage !== "submitted") continue;
    open += 1;
    // Fresh height per round: serial tx waits make a per-pass snapshot stale.
    const height = latestHeight(net);
    const actions = decideActions(round, height, params);
    for (const action of actions) {
      if (action.kind === "commit" || action.kind === "reveal") {
        const verifier = action.verifier;
        const txArgs = action.kind === "commit"
          ? ["knowledge", "submit-commitment", round.round_id, action.hash]
          : ["knowledge", "submit-reveal", round.round_id, round.vote, action.salt];
        const gas = action.kind === "commit" ? COMMIT_GAS : REVEAL_GAS;
        const outcome = broadcast(net, verifier, gas, txArgs);
        if (outcome.ok && outcome.tx_hash) {
          const included = await waitForTx(net, outcome.tx_hash);
          const duplicate = included.found && included.code !== 0 && DUPLICATE_VOTE.test(included.raw_log ?? "");
          if ((included.found && included.code === 0) || duplicate) {
            const progress = round.verifiers[verifier] as { commit_tx?: string; reveal_tx?: string };
            if (action.kind === "commit") progress.commit_tx = outcome.tx_hash;
            else progress.reveal_tx = outcome.tx_hash;
            console.log(`${action.kind === "commit" ? "🟠" : "🟡"} ${round.entry_id}: ${verifier} ${action.kind}${duplicate ? " (already recorded on-chain)" : ""}`);
          } else {
            round.notes.push(`${action.kind} ${verifier} failed: code ${included.code ?? "?"} ${included.raw_log ?? "not included"}`);
            console.log(`🔴 ${round.entry_id}: ${verifier} ${action.kind} failed`);
          }
        } else if (DUPLICATE_VOTE.test(outcome.raw_log ?? "")) {
          const progress = round.verifiers[verifier] as { commit_tx?: string; reveal_tx?: string };
          if (action.kind === "commit") progress.commit_tx = outcome.tx_hash ?? "on-chain";
          else progress.reveal_tx = outcome.tx_hash ?? "on-chain";
        } else {
          round.notes.push(`${action.kind} ${verifier} rejected: ${outcome.raw_log ?? outcome.detail}`);
        }
      } else if (action.kind === "check_resolution") {
        const claim = query(net, ["query", "knowledge", "claim", round.claim_id]);
        const claimBody = (claim?.claim ?? claim) as Record<string, unknown> | null;
        const status = String(claimBody?.status ?? "");
        if (!claimBody || status === "") {
          // Transient RPC failure must never be a terminal verdict.
          round.query_failures = (round.query_failures ?? 0) + 1;
          if (round.query_failures >= 5) {
            round.stage = "error";
            round.notes.push("claim query failed 5 consecutive times; inspect manually");
          }
        } else {
          round.query_failures = 0;
          round.fact_status = status;
          round.confidence = String(claimBody?.confidence ?? claimBody?.confidence_bps ?? "");
          const verdict = classifyClaimStatus(status);
          if (verdict === "accepted") {
            round.stage = "resolved_accepted";
            round.fact_id = discoverFactId(net, round) ?? round.fact_id;
            console.log(`🟢 ${round.entry_id}: ACCEPTED (claim ${round.claim_id}, fact ${round.fact_id ?? "?"}, confidence ${round.confidence || "?"})`);
          } else if (verdict === "terminal_other") {
            round.stage = "resolved_other";
            console.log(`🔴 ${round.entry_id}: resolved status ${status}`);
          } else {
            // in_flight or unknown: aggregation can lag pacing; keep waiting.
            earliestWait = Math.min(earliestWait, height + 20);
          }
        }
      } else if (action.kind === "missed_window") {
        round.stage = "missed_window";
        round.notes.push(action.detail);
        console.log(`🔴 ${round.entry_id}: ${action.detail}`);
      } else if (action.kind === "wait") {
        earliestWait = Math.min(earliestWait, action.until_height);
      }
    }
    upsertRound(net, round);
  }
  return { open, earliestWait };
}

async function commandSubmit(
  net: NetworkConfig,
  registerPath: string,
  entryId: string | undefined,
  batch: boolean,
  feeUzrn: number,
  skip: Set<string> = new Set(),
): Promise<void> {
  const register = loadRegister(registerPath);
  const issues = validateRegister(register);
  if (issues.length > 0) fail(`register invalid (${issues.length} issues); run validate`, 2);
  const params = phaseParams(net);
  const done = new Set(readRounds(net)
    .filter(round => round.stage !== "missed_window" && round.stage !== "abandoned" && round.stage !== "error")
    .map(round => round.entry_id));
  let targets: RegisterEntry[];
  if (batch) {
    const ordered = [...firstBatch(register), ...admissibleEntries(register).filter(entry => entry.chain.firstBatch === null)];
    // Dependency order: an entry whose relations name entry:<id> targets must
    // submit after those targets (stable insertion sort; cycles rejected by
    // validate's conservatism in practice).
    const placed: RegisterEntry[] = [];
    const pending = [...ordered];
    let guard = pending.length * pending.length + 1;
    while (pending.length > 0 && guard-- > 0) {
      const index = pending.findIndex(entry => (entry.chain.relations ?? []).every(relation =>
        !relation.target.startsWith("entry:")
        || placed.some(prior => prior.id === relation.target.slice(6))
        || done.has(relation.target.slice(6))));
      if (index === -1) fail(`circular entry: relation dependencies among ${pending.map(entry => entry.id).join(", ")}`);
      placed.push(pending.splice(index, 1)[0] as RegisterEntry);
    }
    targets = placed.filter(entry => !done.has(entry.id) && !skip.has(entry.id));
  } else {
    if (!entryId) fail("submit needs --entry <id> or --batch");
    const entry = register.entries.find(candidate => candidate.id === entryId);
    if (!entry) fail(`entry ${entryId} not in register`);
    if (entry.chain.intent !== "assert") fail(`entry ${entryId} intent is ${entry.chain.intent}, not assert`);
    if (done.has(entryId)) fail(`entry ${entryId} already has a live/resolved round`);
    targets = [entry];
  }
  if (targets.length === 0) {
    console.log("nothing to submit");
    return;
  }
  const queue = [...targets];
  const deferrals = new Map<string, number>();
  let position = 0;
  while (queue.length > 0) {
    const entry = queue.shift() as RegisterEntry;
    position += 1;
    // An entry whose entry:<id> relation target has no fact id yet defers to
    // the back of the queue and waits for panels to resolve the target.
    try {
      relationsFlag(entry, recordedFactIds(readRounds(net)));
    } catch (error) {
      const count = (deferrals.get(entry.id) ?? 0) + 1;
      deferrals.set(entry.id, count);
      if (count > 40) fail(`entry ${entry.id} dependency never resolved after ${count} deferrals: ${error instanceof Error ? error.message : error}`);
      console.log(`── deferring ${entry.id} (dependency not yet resolved; panels continue)`);
      queue.push(entry);
      await panelPass(net, params);
      const target = latestHeight(net) + 20;
      while (latestHeight(net) < target) await Bun.sleep(20_000);
      continue;
    }
    console.log(`── submitting ${entry.id} (${position}/${targets.length})`);
    // The EFFECTIVE cooldown is pacing-scaled (observed 100 blocks on zerone-1
    // vs the base 50); trust the chain's own error over any assumption.
    let round: RoundState | null = null;
    for (let attempt = 1; attempt <= 4 && !round; attempt += 1) {
      try {
        round = await submitEntry(net, entry, feeUzrn, "accept", recordedFactIds(readRounds(net)));
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error);
        const cooldown = /cooldown active: (\d+) blocks remaining/.exec(message);
        if (!cooldown || attempt === 4) throw error;
        const remaining = Number(cooldown[1]) + 5;
        console.log(`   cooldown: waiting ${remaining} blocks (attempt ${attempt})…`);
        const target = latestHeight(net) + remaining;
        while (latestHeight(net) < target) {
          await panelPass(net, params); // cover open rounds while we wait
          await Bun.sleep(20_000);
        }
      }
    }
    if (!round) continue;
    upsertRound(net, round);
    console.log(`   claim ${round.claim_id || "?"} round ${round.round_id} @ height ${round.submit_height || "?"}`);
    if (queue.length > 0) {
      console.log("   cooldown ~105 blocks (interleaving panel passes)…");
      const target = latestHeight(net) + 105;
      while (latestHeight(net) < target) {
        await panelPass(net, params);
        await Bun.sleep(20_000);
      }
    }
  }
  // Cover the tail: freshly submitted rounds still need commits.
  await panelPass(net, params);
}

async function commandPanel(net: NetworkConfig, watch: boolean): Promise<void> {
  const params = phaseParams(net);
  for (;;) {
    const { open, earliestWait } = await panelPass(net, params);
    if (open === 0) {
      console.log("no open rounds");
      printDomainHealth(net);
      return;
    }
    if (!watch) {
      if (earliestWait < Number.MAX_SAFE_INTEGER) {
        console.log(`open rounds waiting until ~height ${earliestWait}; re-run panel (or use --watch)`);
      }
      return;
    }
    await Bun.sleep(60_000);
  }
}

function printDomainHealth(net: NetworkConfig): void {
  const temperature = query(net, ["query", "knowledge", "epistemic-temperature", "mathematics"]);
  if (temperature) console.log(`epistemic temperature (mathematics): ${JSON.stringify(temperature)}`);
  const alerts = query(net, ["query", "knowledge", "conformity-alerts"]);
  if (alerts && JSON.stringify(alerts) !== "{}") console.log(`⚠ conformity alerts: ${JSON.stringify(alerts)}`);
}

function commandStatus(net: NetworkConfig): void {
  const rounds = readRounds(net);
  if (rounds.length === 0) {
    console.log("ledger empty");
    return;
  }
  for (const round of rounds) {
    const committed = Object.values(round.verifiers).filter(verifier => verifier.commit_tx).length;
    const revealed = Object.values(round.verifiers).filter(verifier => verifier.reveal_tx).length;
    console.log(`${round.stage.padEnd(18)} ${round.entry_id} claim=${round.claim_id} commits=${committed} reveals=${revealed}${round.fact_status ? ` status=${round.fact_status}` : ""}${round.confidence ? ` confidence=${round.confidence}` : ""}`);
    for (const note of round.notes) console.log(`    ⚠ ${note}`);
  }
  printDomainHealth(net);
}

async function commandSetup(net: NetworkConfig): Promise<void> {
  const funderAddress = keyAddress(net, net.funder);
  const needed: Array<{ key: string; target: number; floor: number }> = [
    { key: net.submitter, target: SUBMITTER_FUND_UZRN, floor: SUBMITTER_FUND_UZRN / 2 },
    ...net.verifiers.map(key => ({ key, target: VERIFIER_FUND_UZRN, floor: VERIFIER_TOPUP_FLOOR_UZRN })),
  ];
  const addressed: Array<(typeof needed)[number] & { address: string }> = [];
  for (const item of needed) {
    let address: string;
    try {
      address = keyAddress(net, item.key);
    } catch {
      console.log(`creating key ${item.key}…`);
      const created = run(["keys", "add", item.key, "--keyring-backend", "test", "--home", net.keyring_home.replace("~", process.env.HOME ?? "~")]);
      if (created.exit_code !== 0) fail(`could not create key ${item.key}: ${created.stderr.trim().split("\n")[0]}`);
      address = keyAddress(net, item.key);
    }
    addressed.push({ ...item, address });
  }

  // Complete every identity/account preflight before the first funding or
  // registration transaction. A truthy query is not evidence of custody:
  // existing accounts must match the exact local keypair and current
  // operational key. Only a confirmed module-level NotFound permits creation.
  const inspected: Array<(typeof addressed)[number] & {
    identityPath: string;
    registration: ClassifiedAccountQuery;
  }> = [];
  for (const item of addressed) {
    const identityPath = statePath(net.name).replace(
      /state\.json$/,
      `${item.key}.ed25519.json`,
    );
    let registration: ClassifiedAccountQuery;
    try {
      registration = classifyRegisteredAccountQuery(run([
        "query",
        "zerone_auth",
        "account",
        item.address,
        "--node",
        net.rpc,
        "--output",
        "json",
      ]));
    } catch (error) {
      fail(
        `cannot safely determine zerone_auth registration for ${item.key}: ${error instanceof Error ? error.message : error}`,
      );
    }
    inspected.push({ ...item, identityPath, registration });
  }

  // Reconcile every already-registered identity before creating any missing
  // one. Thus a later ambiguous query or custody mismatch cannot leave behind
  // a newly generated key from an earlier confirmed NotFound result.
  const reconciled = new Map<string, PersistedIdentityKey>();
  for (const item of inspected) {
    const { registration } = item;
    if (registration.kind === "found") {
      let identity: PersistedIdentityKey;
      try {
        identity = loadPersistedIdentityKey(item.identityPath, item.key);
        reconcileRegisteredAccount(registration.account, {
          key: item.key,
          address: item.address,
          accountType: "agent",
          identity,
        });
      } catch (error) {
        fail(
          `registered identity custody check failed for ${item.key}: ${error instanceof Error ? error.message : error}`,
        );
      }
      console.log(
        `🟢 ${item.key} zerone_auth identity and current operational key reconciled (v${registration.account.operational_key_version})`,
      );
      reconciled.set(item.key, identity);
    }
  }

  const prepared: Array<(typeof addressed)[number] & {
    identity: PersistedIdentityKey;
    registered: boolean;
  }> = [];
  for (const item of inspected) {
    if (item.registration.kind === "found") {
      const identity = reconciled.get(item.key);
      if (!identity) fail(`internal setup error: reconciled identity ${item.key} is missing`);
      prepared.push({ ...item, identity, registered: true });
    } else {
      let identity: PersistedIdentityKey;
      try {
        identity = loadOrCreatePersistedIdentityKey(item.identityPath, item.key);
      } catch (error) {
        fail(
          `could not establish identity custody for unregistered ${item.key}: ${error instanceof Error ? error.message : error}`,
        );
      }
      prepared.push({ ...item, identity, registered: false });
    }
  }

  console.log(`funder ${net.funder} = ${funderAddress} (${(spendableUzrn(net, funderAddress) / 1e6).toFixed(1)} ZRN)`);
  for (const { key, address, target, floor } of prepared) {
    const balance = spendableUzrn(net, address);
    if (balance >= floor) {
      console.log(`🟢 ${key} ${address}: ${(balance / 1e6).toFixed(2)} ZRN (sufficient)`);
      continue;
    }
    const topUp = target - balance;
    console.log(`funding ${key} with ${(topUp / 1e6).toFixed(2)} ZRN from ${net.funder}…`);
    const outcome = broadcast(net, net.funder, GAS_SEND, ["bank", "send", funderAddress, address, `${topUp}uzrn`]);
    if (!outcome.ok || !outcome.tx_hash) fail(`funding ${key} rejected: ${outcome.raw_log ?? outcome.detail}`);
    const included = await waitForTx(net, outcome.tx_hash);
    if (!included.found || included.code !== 0) fail(`funding ${key} failed on-chain: ${included.raw_log ?? "not included"}`);
    console.log(`🟢 ${key} funded (tx ${outcome.tx_hash.slice(0, 12)}…)`);
  }
  // Knowledge messages require zerone_auth registration. Register every key as
  // an agent, with an Ed25519 proof that binds this chain, sender, DID and role.
  for (const { key, address, identity, registered } of prepared) {
    if (registered) continue;
    console.log(`registering ${key} in zerone_auth (did:zrn:${identity.public_hex.slice(0, 12)}…)…`);
    const registrationArgs = accountRegistrationCommandArgs({
      chainId: net.chain_id,
      sender: address,
      identity,
      accountType: "agent",
      metadata: "",
    });
    const outcome = broadcast(net, key, 300_000, registrationArgs);
    if (!outcome.ok || !outcome.tx_hash) fail(`registering ${key} rejected: ${outcome.raw_log ?? outcome.detail}`);
    const included = await waitForTx(net, outcome.tx_hash);
    if (!included.found || included.code !== 0) fail(`registering ${key} failed on-chain: ${included.raw_log ?? "not included"}`);

    let confirmedVersion = 0;
    try {
      const confirmed = classifyRegisteredAccountQuery(run([
        "query",
        "zerone_auth",
        "account",
        address,
        "--node",
        net.rpc,
        "--output",
        "json",
      ]));
      if (confirmed.kind !== "found") {
        throw new Error("account is still absent after successful registration transaction");
      }
      reconcileRegisteredAccount(confirmed.account, {
        key,
        address,
        accountType: "agent",
        identity,
      });
      confirmedVersion = confirmed.account.operational_key_version;
    } catch (error) {
      fail(
        `post-registration custody check failed for ${key}: ${error instanceof Error ? error.message : error}`,
      );
    }
    console.log(
      `🟢 ${key} registered and reconciled (operational key v${confirmedVersion})`,
    );
  }
}

async function main(): Promise<void> {
  const { values, positionals } = parseArgs({
    args: Bun.argv.slice(2),
    allowPositionals: true,
    options: {
      network: { type: "string" },
      register: { type: "string" },
      entry: { type: "string" },
      batch: { type: "boolean", default: false },
      skip: { type: "string" },
      watch: { type: "boolean", default: false },
      "fee-uzrn": { type: "string" },
      "live-ack": { type: "string" },
      help: { type: "boolean", default: false },
    },
  });
  const command = positionals[0];
  if (values.help || !command) {
    console.log(`frontier-intake — absorb the math breakthrough register into zerone

Usage:
  bun intake.ts validate [--register <path>]
  bun intake.ts plan     --network <net> [--register <path>]
  bun intake.ts setup    --network <net> --live-ack=<chain-id>
  bun intake.ts submit   --network <net> --live-ack=<chain-id> (--entry <id> | --batch) [--fee-uzrn n]
  bun intake.ts panel    --network <net> --live-ack=<chain-id> [--watch]
  bun intake.ts status   --network <net>

Networks: ${Object.keys(NETWORKS).join(", ")}. Broadcasting commands refuse to
run without --live-ack naming the exact chain. The register stays canonical;
runtime state lives at ~/.zerone-agent/frontier-intake/<net>.state.json.`);
    process.exit(command ? 0 : 1);
  }
  const registerPath = values.register ?? defaultRegisterPath();
  const feeUzrn = values["fee-uzrn"] ? Number(values["fee-uzrn"]) : DEFAULT_REVIEW_FEE_UZRN;
  switch (command) {
    case "validate": return commandValidate(registerPath);
    case "plan": return commandPlan(requireNetwork(values.network), registerPath);
    case "setup": {
      const net = requireNetwork(values.network);
      requireLiveAck(net, values["live-ack"]);
      return commandSetup(net);
    }
    case "submit": {
      const net = requireNetwork(values.network);
      requireLiveAck(net, values["live-ack"]);
      return commandSubmit(net, registerPath, values.entry, values.batch, feeUzrn, new Set((values.skip ?? "").split(",").filter(Boolean)));
    }
    case "panel": {
      const net = requireNetwork(values.network);
      requireLiveAck(net, values["live-ack"]);
      return commandPanel(net, values.watch);
    }
    case "status": return commandStatus(requireNetwork(values.network));
    default: fail(`unknown command "${command}"`);
  }
}

await main();

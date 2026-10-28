#!/usr/bin/env bun
// zerone-door: the zerone-1 admission door for household agents.
// Dry run is the default. A live run needs --live-ack=<chain id>, and the id
// must match both the config and the node.
//
//   bun door.ts run    [--config PATH] [--once] [--interval SECONDS] [--live-ack=zerone-1]
//   bun door.ts status [--config PATH]   # read-only; exits 4 when the admittee watch flags anyone, 5 when a chain read failed
//   bun door.ts verify FILE          # request format, signatures and proof of work; no chain
import { readFileSync } from "node:fs";
import { parseArgs } from "node:util";
import { ChainReader } from "./src/chain.ts";
import { defaultConfigPath, doorPaths, loadConfig, type DoorConfig, type DoorPaths } from "./src/config.ts";
import { globalSummary, readLedger, type DoorContext } from "./src/door.ts";
import { chainVersionMatches, currentWindow, gatherGlobalFacts, haltFilesPresent, registrarLiability, seedCounts } from "./src/gates.ts";
import { ensurePrivateDir, IntakeError, makeLobbyRunner } from "./src/intake.ts";
import { counted, utcDay } from "./src/ledger.ts";
import { keychainPassphrase } from "./src/passphrase.ts";
import { parseDoorRequest, verifyDoorRequest } from "./src/request.ts";
import { runOnce } from "./src/run.ts";
import { requireZeroneDoorBunVersion } from "./src/runtime.ts";
import { makeZeronedRunner, sha256File } from "./src/tx.ts";
import { watchAdmittees } from "./src/watch.ts";

function log(event: Record<string, unknown>): void {
  process.stdout.write(`${JSON.stringify({ at: new Date().toISOString(), ...event })}\n`);
}

function fail(message: string, code = 2): never {
  process.stderr.write(`zerone-door: ${message}\n`);
  process.exit(code);
}

function context(config: DoorConfig, paths: DoorPaths, live: boolean): DoorContext {
  return {
    config,
    paths,
    chain: new ChainReader({ rpc: config.rpc, rest: config.rest }),
    zeroned: makeZeronedRunner(config.zeroned.path),
    passphrase: keychainPassphrase,
    live,
    now: () => new Date(),
    log,
  };
}

/** Read-only. Reports the live ledger (and how many dry-run requests were
 * seen), the chain facts every gate reads, the chain version against its pin,
 * lapsed door seeds, and the admittee watch: each admittee's zerone_auth
 * record against its request. Exits 4 when the watch flags an admittee, and 5
 * when nothing is flagged but a chain read failed (an unknown is not a clean result). */
async function status(ctx: DoorContext): Promise<void> {
  const out: Record<string, unknown> = { halt_files_present: haltFilesPresent(ctx.paths.haltFiles) };
  const g = await gatherGlobalFacts(ctx.chain, ctx.config.registrar.address);
  const ledger = readLedger({ ...ctx, live: true });
  out.dry_run_requests_seen = Object.keys(readLedger({ ...ctx, live: false }).seen).length;
  const entries = counted(ledger);
  const today = utcDay(new Date());
  out.door = {
    admitted_total: entries.filter((e) => e.kind === "admit").length,
    regranted_total: entries.filter((e) => e.kind === "regrant").length,
    today: entries.filter((e) => e.utc_day === today).length,
    pending: ledger.entries.filter((e) => e.status === "pending_inclusion").map((e) => e.tx_hash),
    failed: ledger.entries.filter((e) => e.status !== "included" && e.status !== "pending_inclusion").length,
  };
  if (!g.facts) {
    out.chain = { unknown: true, errors: g.errors };
  } else {
    const seeds = seedCounts(g.facts, ctx.config.registrar.address);
    const window = currentWindow(g.facts.status.height);
    const admittees = new Set(ledger.entries.map((e) => e.address));
    out.chain = {
      ...globalSummary(g.facts),
      chain_version_pinned: ctx.config.chain_version,
      chain_version_matches_pin: chainVersionMatches(g.facts.chainVersion, ctx.config.chain_version),
      registrar_is_r: g.facts.params.registrar === ctx.config.registrar.address,
      chain_window: window,
      chain_window_used: g.facts.window.storedWindow === window ? g.facts.window.storedCount : 0,
      door_window_used: entries.filter((e) => e.kind === "admit" && e.chain_window === window).length,
      registrar_liability_uzrn: registrarLiability(g.facts)?.toString() ?? "unbounded",
      unclaimed_live_seeds: seeds.unclaimedLive,
      // An alert, not a stop: door admittees whose seed sits unclaimed with no live allowance.
      unclaimed_lapsed_door_seeds: seeds.unclaimedLapsed.filter((a) => admittees.has(a)),
      unclaimed_lapsed_other_seeds: seeds.unclaimedLapsed.filter((a) => !admittees.has(a)).length,
    };
  }
  const watch = await watchAdmittees(ctx.chain, ledger);
  out.admittee_watch = watch;
  process.stdout.write(`${JSON.stringify(out, null, 2)}\n`);
  if (watch.flagged.length > 0) process.exitCode = 4;
  else if (!g.facts || watch.unknown.length > 0) process.exitCode = 5;
}

async function main(): Promise<void> {
  requireZeroneDoorBunVersion();
  const [command, ...rest] = process.argv.slice(2);
  if (command === "verify") {
    const file = rest[0];
    if (!file) fail("usage: door.ts verify FILE");
    const parsed = parseDoorRequest(readFileSync(file, "utf8"));
    if (!parsed.ok) {
      process.stdout.write(`${JSON.stringify({ ok: false, reasons: ["request_malformed"], detail: parsed.detail })}\n`);
      process.exit(1);
    }
    const reasons = verifyDoorRequest(parsed.value, new Date());
    process.stdout.write(`${JSON.stringify({ ok: reasons.length === 0, request_id: parsed.value.requestId, reasons })}\n`);
    process.exit(reasons.length === 0 ? 0 : 1);
  }
  if (command !== "run" && command !== "status") fail("usage: door.ts run|status|verify (see the header of door.ts)");
  const { values } = parseArgs({
    args: rest,
    options: {
      config: { type: "string" },
      once: { type: "boolean" },
      interval: { type: "string" },
      "live-ack": { type: "string" },
    },
    strict: true,
  });
  const configPath = values.config ?? defaultConfigPath();
  let config: DoorConfig;
  try {
    config = loadConfig(configPath);
  } catch (error) {
    fail(error instanceof Error ? error.message : String(error));
  }
  const paths = doorPaths(configPath);
  const live = values["live-ack"] !== undefined;
  if (live && values["live-ack"] !== config.chain_id) fail(`--live-ack must name ${config.chain_id}`);
  let actual: string;
  try {
    actual = sha256File(config.zeroned.path);
  } catch {
    fail(`cannot read the pinned zeroned at ${config.zeroned.path}`);
  }
  if (actual !== config.zeroned.sha256) fail("the zeroned binary does not match its pinned sha256");
  const ctx = context(config, paths, live);
  if (command === "status") return status(ctx);

  for (const dir of [paths.inbox, paths.processed, paths.replies, paths.state, paths.work]) ensurePrivateDir(dir);
  if (live) {
    const s = await ctx.chain.status();
    if (!s.ok) fail(`cannot read the node: ${s.error}`, 3);
    if (s.value.network !== config.chain_id) fail(`the node serves ${s.value.network}, not ${config.chain_id}`);
  }
  log({ event: "start", mode: live ? "live" : "dry-run", registrar: config.registrar.address });
  const intervalSeconds = Number(values.interval ?? "60");
  if (!Number.isInteger(intervalSeconds) || intervalSeconds < 10 || intervalSeconds > 3600) fail("--interval is 10-3600 seconds");
  const deps = { lobbyRun: config.lobby ? makeLobbyRunner(config.lobby.cli) : null };
  for (;;) {
    try {
      await runOnce(ctx, deps);
    } catch (error) {
      if (error instanceof IntakeError) log({ event: "intake_error", detail: error.message });
      else log({ event: "run_error", detail: error instanceof Error ? error.message : String(error) });
      if (values.once) process.exit(1);
    }
    if (values.once) return;
    await Bun.sleep(intervalSeconds * 1000);
  }
}

await main();

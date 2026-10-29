import { describe, expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { existsSync, readdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { addressPayloadHex } from "../src/bech32.ts";
import { ChainReader, txMessagesOf, type FetchLike } from "../src/chain.ts";
import { doorPaths } from "../src/config.ts";
import { allowanceMatches, carriesEntry, processItem, readLedger, reconcilePending, replyText, type DoorContext, type Reply } from "../src/door.ts";
import type { LobbyRunner } from "../src/intake.ts";
import { loadLedger, type LedgerEntry } from "../src/ledger.ts";
import { runOnce } from "../src/run.ts";
import type { ZeronedRunner } from "../src/tx.ts";
import { checkBuiltTx } from "../src/tx.ts";
import { BOOTSTRAP_WINDOW_BLOCKS, DOOR_WINDOW_CAP, PENDING_ATTENTION_MS, TIMEOUT_SETTLE_BLOCKS, TX_TIMEOUT_BLOCKS } from "../src/policy.ts";
import { admitteesOf, judgeAdmittee, watchAdmittees } from "../src/watch.ts";
import {
  addressFor,
  fakeChainState,
  fakeFetch,
  grantFromR,
  HOUSEHOLD,
  makeDoorHome,
  makeRequest,
  NOW,
  R,
  R_PUBKEY_B64,
  restDoorGrant,
  restPot,
  txRawOf,
  type FakeChainState,
} from "./helpers.ts";

const PASSPHRASE = "correct horse battery staple";

interface Call {
  args: string[];
  stdin: string | null;
  /** The tx file the call read, captured before the door cleans up. */
  file: string | null;
}

function flag(args: string[], name: string): string | undefined {
  const i = args.indexOf(name);
  return i === -1 ? undefined : args[i + 1];
}

function genTx(messages: unknown[], timeoutHeight = "0") {
  return {
    body: { messages, memo: "", timeout_height: timeoutHeight, extension_options: [], non_critical_extension_options: [] },
    auth_info: { signer_infos: [], fee: { amount: [{ denom: "uzrn", amount: "200000" }], gas_limit: "200000", payer: "", granter: "" }, tip: null },
    signatures: [],
  };
}

/** A stand-in for the pinned CLI that builds the same JSON shapes zeroned
 * 2e37c4c prints, and applies a broadcast admission to the fake chain. */
function fakeZeroned(state: FakeChainState, calls: Call[], opts: { broadcastCode?: number; include?: boolean; refuseInBlock?: number; signWith?: string } = {}): ZeronedRunner {
  return (args, stdin) => {
    const reads = args[0] === "tx" && ["encode", "sign", "broadcast"].includes(args[1]!) ? readFileSync(args[2]!, "utf8") : null;
    calls.push({ args, stdin: stdin ? Buffer.from(stdin).toString("utf8") : null, file: reads });
    const ok = (stdout: string) => ({ code: 0, stdout, stderr: "" });
    if (args[0] === "tx" && args[1] === "claiming_pot" && args[2] === "add-bootstrap-entry") {
      return ok(JSON.stringify(genTx([{ "@type": "/zerone.claiming_pot.v1.MsgAddBootstrapEntry", authority: flag(args, "--from"), addresses: [args[3]] }], flag(args, "--timeout-height"))));
    }
    if (args[0] === "tx" && args[1] === "feegrant" && args[2] === "grant") {
      const spend = flag(args, "--spend-limit")!.replace("uzrn", "");
      const period = flag(args, "--period-limit")!.replace("uzrn", "");
      return ok(JSON.stringify(genTx([{
        "@type": "/cosmos.feegrant.v1beta1.MsgGrantAllowance",
        granter: args[3],
        grantee: args[4],
        allowance: {
          "@type": "/cosmos.feegrant.v1beta1.AllowedMsgAllowance",
          allowance: {
            "@type": "/cosmos.feegrant.v1beta1.PeriodicAllowance",
            basic: { spend_limit: [{ denom: "uzrn", amount: spend }], expiration: flag(args, "--expiration") },
            period: `${flag(args, "--period")}s`,
            period_spend_limit: [{ denom: "uzrn", amount: period }],
            period_can_spend: [{ denom: "uzrn", amount: period }],
            period_reset: "2026-10-28T17:46:00Z",
          },
          allowed_messages: flag(args, "--allowed-messages")!.split(","),
        },
      }], flag(args, "--timeout-height"))));
    }
    if (args[0] === "tx" && args[1] === "feegrant" && args[2] === "revoke") {
      return ok(JSON.stringify(genTx([{ "@type": "/cosmos.feegrant.v1beta1.MsgRevokeAllowance", granter: args[3], grantee: args[4] }], flag(args, "--timeout-height"))));
    }
    if (args[0] === "tx" && args[1] === "encode") {
      return ok(`${createHash("sha256").update(readFileSync(args[2]!)).digest("base64")}\n`);
    }
    if (args[0] === "tx" && args[1] === "sign") {
      if (stdin === undefined || Buffer.from(stdin).toString("utf8") !== `${PASSPHRASE}\n`) return { code: 1, stdout: "", stderr: "Error: invalid passphrase" };
      const tx = JSON.parse(readFileSync(args[2]!, "utf8"));
      // zeroned appends to existing signatures, so the door must hand it none.
      if (tx.auth_info.signer_infos.length !== 0 || tx.signatures.length !== 0) return { code: 1, stdout: "", stderr: "already signed" };
      tx.auth_info.signer_infos = [{
        public_key: { "@type": "/cosmos.crypto.secp256k1.PubKey", key: opts.signWith ?? R_PUBKEY_B64 },
        mode_info: { single: { mode: "SIGN_MODE_DIRECT" } },
        sequence: flag(args, "--sequence"),
      }];
      tx.signatures = ["c2lnbmF0dXJl"];
      writeFileSync(flag(args, "--output-document")!, JSON.stringify(tx));
      return ok("");
    }
    if (args[0] === "tx" && args[1] === "broadcast") {
      const raw = readFileSync(args[2]!);
      const hash = createHash("sha256").update(Buffer.from(createHash("sha256").update(raw).digest("base64"), "base64")).digest("hex").toUpperCase();
      const code = opts.broadcastCode ?? 0;
      if (code === 0 && opts.refuseInBlock !== undefined) refuse(state, hash, state.height + 1, opts.refuseInBlock);
      else if (code === 0 && opts.include !== false) land(state, JSON.parse(raw.toString("utf8")), hash, state.height + 1);
      return ok(JSON.stringify({ height: "0", txhash: hash, code, raw_log: code ? "insufficient fees" : "" }));
    }
    return { code: 1, stdout: "", stderr: `unexpected zeroned call ${args.slice(0, 3).join(" ")}` };
  };
}

/** Runs a signed door transaction on the fake chain at `height`: its
 * messages take effect, R's sequence moves on, and the tx and acc_seq
 * indexes name it. */
function land(state: FakeChainState, tx: any, hash: string, height: number, code = 0): void {
  if (code === 0) {
    for (const m of tx.body.messages) {
      if (m["@type"].endsWith("MsgAddBootstrapEntry")) state.pots[`bootstrap-${m.addresses[0]}`] = restPot(m.addresses[0]);
      if (m["@type"].endsWith("MsgRevokeAllowance")) state.grants = state.grants.filter((g) => !(g.granter === m.granter && g.grantee === m.grantee));
      if (m["@type"].endsWith("MsgGrantAllowance")) state.grants.push({ granter: m.granter, grantee: m.grantee, allowance: m.allowance });
    }
  }
  const sequence = String(tx.auth_info.signer_infos[0].sequence);
  state.auth[R]!.sequence = String(Number(sequence) + 1);
  state.accSeq[`${R}/${sequence}`] = [hash];
  state.txs[hash] = {
    height: String(height),
    tx_result: { code, log: code ? "failed in block" : "", gas_used: "121000", gas_wanted: "150000" },
    tx: txRawOf(tx.body.messages, Number(tx.body.timeout_height)),
  };
}

/** A transaction R signed that is no door admission (a transfer), as the indexes hold it. */
function otherTx(height: number): FakeChainState["txs"][string] {
  return {
    height: String(height),
    tx_result: { code: 0, log: "", gas_used: "90000", gas_wanted: "100000" },
    tx: txRawOf([{ "@type": "/cosmos.bank.v1beta1.MsgSend", from_address: R, to_address: HOUSEHOLD }]),
  };
}

/** A block holds the transaction but its ante handler refuses it (a fee R
 * cannot pay at that moment, say): the tx index records the code, nothing in
 * it runs, R's sequence stays, and no acc_seq event is kept. */
function refuse(state: FakeChainState, hash: string, height: number, code: number): void {
  state.txs[hash] = { height: String(height), tx_result: { code, log: "insufficient funds", gas_used: "40000", gas_wanted: "150000" } };
}

function setup(opts: { live?: boolean; state?: Partial<FakeChainState>; zeroned?: { broadcastCode?: number; include?: boolean; refuseInBlock?: number; signWith?: string }; passphrase?: string | null; home?: Parameters<typeof makeDoorHome>[0] } = {}) {
  const { configPath, config } = makeDoorHome(opts.home);
  const state = fakeChainState(opts.state);
  const calls: Call[] = [];
  const logs: Array<Record<string, unknown>> = [];
  let passphraseReads = 0;
  const ctx: DoorContext = {
    config,
    paths: { ...doorPaths(configPath), haltFiles: [join(config.home, "HALT")] },
    chain: new ChainReader({ rpc: config.rpc, rest: config.rest, fetch: fakeFetch(state), retries: 1 }),
    zeroned: fakeZeroned(state, calls, opts.zeroned),
    passphrase: (service) => {
      passphraseReads += 1;
      expect(service).toBe("door-test-service");
      const p = opts.passphrase === undefined ? PASSPHRASE : opts.passphrase;
      return p === null ? null : new TextEncoder().encode(p);
    },
    live: opts.live ?? false,
    now: () => NOW,
    log: (e) => logs.push(e),
    sleep: async () => {},
    inclusionTimeoutMs: 0,
  };
  return { ctx, state, calls, logs, passphraseReads: () => passphraseReads };
}

const studio = { kind: "studio_local" as const, file: "agent.request" };

describe("dry run (the default)", () => {
  test("checks, builds and simulates the real admission, signs nothing", async () => {
    const { ctx, calls, passphraseReads } = setup();
    const req = makeRequest();
    const { reply } = await processItem(ctx, { text: req.text, channel: studio });
    expect(reply?.decision).toBe("dry_run_ok");
    expect(reply?.reasons).toEqual([]);
    expect(reply?.simulated_gas).toBe(109_000);
    expect(reply?.gas_limit).toBe(150_000);
    expect(reply?.allowance).toEqual({
      allowed_messages: ["/zerone.claiming_pot.v1.MsgClaim", "/zerone.auth.v1.MsgRegisterAccount", "/zerone.knowledge.v1.MsgSubmitClaim"],
      spend_limit_uzrn: 1_100_000,
      period_seconds: 900,
      period_limit_uzrn: 400_000,
      expiration: "2026-11-11T17:31:00Z",
    });
    expect(calls.some((c) => c.args[1] === "sign" || c.args[1] === "broadcast")).toBe(false);
    expect(passphraseReads()).toBe(0);
    expect(existsSync(join(ctx.paths.replies, "dry-run", `${reply!.request_id}.json`))).toBe(true);
    expect(readLedger({ ...ctx, live: true }).entries).toEqual([]);
  });

  test("a replayed request is ignored and its first reply is kept", async () => {
    const { ctx } = setup();
    const req = makeRequest();
    const first = await processItem(ctx, { text: req.text, channel: studio });
    const second = await processItem(ctx, { text: req.text, channel: studio });
    expect(first.answer).toBe(true);
    expect(second).toEqual({ reply: null, answer: false });
  });

  test("a failed simulation refuses", async () => {
    const { ctx } = setup({ state: { simulateCode: 5 } });
    const { reply } = await processItem(ctx, { text: makeRequest().text, channel: studio });
    expect(reply?.reasons).toEqual(["simulate_failed"]);
  });

  test("simulated gas too close to the limit refuses", async () => {
    const { ctx } = setup({ state: { simulateGas: 140_000 } });
    const { reply } = await processItem(ctx, { text: makeRequest().text, channel: studio });
    expect(reply?.reasons).toEqual(["gas_over_limit"]);
  });

  test("an unreadable chain refuses rather than guessing", async () => {
    const { ctx } = setup({ state: { fail: new Set(["/cosmos/feegrant/v1beta1/allowances/"]) } });
    const { reply } = await processItem(ctx, { text: makeRequest().text, channel: studio });
    expect(reply?.reasons).toEqual(["chain_read_failed"]);
  });

  test("a malformed request is refused under the hash of its text", async () => {
    const { ctx } = setup();
    const { reply } = await processItem(ctx, { text: "hello", channel: studio });
    expect(reply?.reasons).toEqual(["request_malformed"]);
    expect(reply?.request_id).toBe(createHash("sha256").update("hello").digest("hex"));
  });

  test("a closed channel is refused before any chain read", async () => {
    const { ctx, state } = setup();
    writeFileSync(ctx.paths.allowlist, JSON.stringify({ schema: "zerone-door-allowlist/v1", studio_local: { enabled: false }, lobby_desks: [] }), { mode: 0o600 });
    const { reply } = await processItem(ctx, { text: makeRequest().text, channel: studio });
    expect(reply?.reasons).toEqual(["channel_not_allowed"]);
    expect(state.requests).toEqual([]);
  });

  test("the HALT file refuses", async () => {
    const { ctx } = setup();
    writeFileSync(join(ctx.config.home, "HALT"), "");
    const { reply } = await processItem(ctx, { text: makeRequest().text, channel: studio });
    expect(reply?.reasons).toEqual(["halt_file"]);
  });

  test("a chain that took an upgrade since the pin refuses, nothing built", async () => {
    const { ctx, state, calls } = setup();
    state.upgrades.push({ name: "h1-v1", height: 1_500_000 });
    const { reply } = await processItem(ctx, { text: makeRequest().text, channel: studio });
    expect(reply?.reasons).toEqual(["chain_version_changed"]);
    expect(calls).toEqual([]);
  });
});

describe("one pass with lobby intake", () => {
  const DESK = "3f2b1c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d";
  const SEASON = "11111111-1111-4111-8111-111111111111";
  const NEXT_SEASON = "22222222-2222-4222-8222-222222222222";

  /** A stand-in for agenttool-lobby: one request in the door conversation of
   * the room's current season, and a record of every call. */
  function fakeLobby(text: string, season = SEASON) {
    const calls: string[][] = [];
    const run: LobbyRunner = (args) => {
      calls.push(args);
      if (args[0] === "room" && args[1] === "read") {
        const after = args[args.indexOf("--after") + 1]!;
        const messages = after === "0" ? [{
          event_id: "evt_req", cursor: "1", conversation: "zerone-door", kind: "message", text,
          from: { identity_id: DESK, fingerprint: "ab12", current_member: true, label: "verified", basis: "door_roster_pin" }, from_me: false,
        }] : [];
        return { code: 0, stdout: JSON.stringify({ room_id: `room_${"a1".repeat(16)}`, mission_id: season, messages, unverified: [], next_cursor: after === "0" ? "1" : after, has_more: false }), stderr: "" };
      }
      if (args[0] === "room" && args[1] === "say") return { code: 0, stdout: "{}", stderr: "" };
      return { code: 1, stdout: "", stderr: "unexpected lobby call" };
    };
    return { run, calls, says: () => calls.filter((c) => c[1] === "say") };
  }

  test("a dry run never posts in the room, even with a reply lease", async () => {
    const { ctx } = setup({ home: { lobby: true, desks: [DESK] } });
    expect(ctx.config.lobby?.reply_lease).toBe("lease_test");
    const lobby = fakeLobby(makeRequest().text);
    await runOnce(ctx, { lobbyRun: lobby.run });
    const dir = join(ctx.paths.replies, "dry-run");
    const files = readdirSync(dir);
    expect(files).toHaveLength(1);
    expect(JSON.parse(readFileSync(join(dir, files[0]!), "utf8")).decision).toBe("dry_run_ok");
    expect(lobby.says()).toEqual([]);
  });

  test("a live run answers in the room under the lease, in reply to the request", async () => {
    const { ctx } = setup({ live: true, home: { lobby: true, desks: [DESK] } });
    const lobby = fakeLobby(makeRequest().text);
    await runOnce(ctx, { lobbyRun: lobby.run });
    expect(lobby.says()).toHaveLength(1);
    const say = lobby.says()[0]!;
    expect(say[say.indexOf("--lease") + 1]).toBe("lease_test");
    expect(say[say.indexOf("--in-reply-to") + 1]).toBe("evt_req");
    expect(say[say.indexOf("--text") + 1]).toContain("admitted");
  });

  test("the cursor is stored with its season; a new season resets it and is logged", async () => {
    const { ctx, logs } = setup({ home: { lobby: true, desks: [DESK] } });
    await runOnce(ctx, { lobbyRun: fakeLobby(makeRequest().text).run });
    expect(readLedger(ctx).lobby_cursor).toEqual({ mission_id: SEASON, cursor: "1" });

    // The room rolls to a new season whose receipts start again at 1.
    const next = fakeLobby(makeRequest({ nonceHex: "c".repeat(32) }).text, NEXT_SEASON);
    await runOnce(ctx, { lobbyRun: next.run });
    const reads = next.calls.filter((c) => c[1] === "read").map((c) => c[c.indexOf("--after") + 1]);
    expect(reads).toEqual(["1", "0"]);
    expect(logs).toContainEqual({ event: "lobby_season_changed", from: SEASON, to: NEXT_SEASON, stale_cursor: "1" });
    expect(readLedger(ctx).lobby_cursor).toEqual({ mission_id: NEXT_SEASON, cursor: "1" });
    expect(Object.keys(readLedger(ctx).seen)).toHaveLength(2);
  });

  test("a ledger with a bare cursor from before seasons loads with no season", () => {
    const { ctx } = setup();
    const path = join(ctx.paths.state, "dry-run-ledger.json");
    writeFileSync(path, JSON.stringify({ schema: "zerone-door-ledger/v1", entries: [], seen: {}, lobby_cursor: "12" }), { mode: 0o600 });
    expect(loadLedger(path).lobby_cursor).toEqual({ mission_id: "", cursor: "12" });
  });
});

describe("the admittee watch (door.ts status)", () => {
  function admittee(address: string, identityHex: string, status: LedgerEntry["status"] = "included"): LedgerEntry {
    return {
      request_id: createHash("sha256").update(address).digest("hex"), kind: "admit", address, address_payload: addressPayloadHex(address)!,
      identity_public_key: identityHex, did: `did:zrn:${identityHex}`, channel_key: "studio_local", tx_hash: "C".repeat(64),
      status, chain_window: 40, utc_day: "2026-10-27", at: "2026-10-27T10:00:00.000Z", grant_expiration: "2026-11-10T10:00:00Z",
    };
  }

  test("flags a type other than agent and a DID other than the request's; unregistered is not a flag", async () => {
    const good = makeRequest({ label: "watch-good" });
    const human = makeRequest({ label: "watch-human" });
    const squat = makeRequest({ label: "watch-squat" });
    const later = makeRequest({ label: "watch-later" });
    const { ctx, state } = setup({ live: true });
    const ledger = readLedger(ctx);
    ledger.entries.push(admittee(good.address, good.identityHex), admittee(human.address, human.identityHex),
      admittee(squat.address, squat.identityHex), admittee(later.address, later.identityHex), admittee(later.address, later.identityHex, "rejected"));
    state.zerone[good.address] = { address: good.address, did: `did:zrn:${good.identityHex}`, accountType: "agent", flags: { frozen: false } };
    state.zerone[human.address] = { address: human.address, did: `did:zrn:${human.identityHex}`, accountType: "human", flags: { frozen: false } };
    state.zerone[squat.address] = { address: squat.address, did: `did:zrn:${"9a".repeat(32)}`, accountType: "agent", flags: { frozen: false } };
    expect(admitteesOf(ledger).map((a) => a.address)).toEqual([good.address, human.address, squat.address, later.address]);

    const report = await watchAdmittees(ctx.chain, ledger);
    expect(report.admittees).toBe(4);
    expect(report.registered).toBe(3);
    expect(report.unregistered).toEqual([later.address]);
    expect(report.flagged).toEqual([
      { address: human.address, request_did: `did:zrn:${human.identityHex}`, registered_did: `did:zrn:${human.identityHex}`, account_type: "human", flags: ["type_not_agent"] },
      { address: squat.address, request_did: `did:zrn:${squat.identityHex}`, registered_did: `did:zrn:${"9a".repeat(32)}`, account_type: "agent", flags: ["did_differs"] },
    ]);
    expect(report.unknown).toEqual([]);
  }, 30_000); // four requests, each with its proof of work: slower than bun's 5 s default on CI runners

  test("an upper-case spelling of the agent's own key still differs from its request", () => {
    const hex = "ab".repeat(32);
    expect(judgeAdmittee({ address: "zrn1x", did: `did:zrn:${hex}` }, { address: "zrn1x", did: `did:zrn:${hex.toUpperCase()}`, accountType: "agent", frozen: false }))
      .toEqual(["did_differs"]);
  });

  test("a failed read leaves the admittee unknown, never clean", async () => {
    const a = makeRequest({ label: "watch-unknown" });
    const { ctx } = setup({ live: true, state: { fail: new Set(["/zerone/auth/v1/account/"]) } });
    const ledger = readLedger(ctx);
    ledger.entries.push(admittee(a.address, a.identityHex));
    const report = await watchAdmittees(ctx.chain, ledger);
    expect(report.unknown).toHaveLength(1);
    expect(report.flagged).toEqual([]);
    expect(report.registered).toBe(0);
  });
});

describe("live", () => {
  test("signs with the passphrase on stdin only, broadcasts, waits, postchecks and records the receipt", async () => {
    const { ctx, calls, logs, state } = setup({ live: true });
    const req = makeRequest();
    const { reply } = await processItem(ctx, { text: req.text, channel: studio });
    expect(reply?.decision).toBe("admitted");
    expect(reply?.tx_hash).toMatch(/^[0-9A-F]{64}$/);
    expect(reply?.height).toBe(state.height + 1);

    const sign = calls.find((c) => c.args[1] === "sign")!;
    expect(sign.stdin).toBe(`${PASSPHRASE}\n`);
    expect(flag(sign.args, "--keyring-backend")).toBe("file");
    expect(flag(sign.args, "--keyring-dir")).toBe("/opt/test/kr");
    expect(flag(sign.args, "--account-number")).toBe("77");
    expect(flag(sign.args, "--sequence")).toBe("4");
    for (const c of calls) expect(c.args.join(" ")).not.toContain(PASSPHRASE);
    expect(JSON.stringify(logs)).not.toContain(PASSPHRASE);
    expect(readFileSync(join(ctx.paths.replies, `${reply!.request_id}.json`), "utf8")).not.toContain(PASSPHRASE);

    const ledger = readLedger(ctx);
    expect(ledger.entries).toHaveLength(1);
    expect(ledger.entries[0]).toMatchObject({ kind: "admit", status: "included", address: req.address, tx_hash: reply!.tx_hash });
    expect(ledger.seen[reply!.request_id]).toBeDefined();
    expect(replyText(reply as Reply)).toContain("admitted");
  });

  test("the same agent cannot be admitted twice", async () => {
    const { ctx } = setup({ live: true });
    await processItem(ctx, { text: makeRequest().text, channel: studio });
    const again = makeRequest({ nonceHex: "a".repeat(32) });
    const { reply } = await processItem(ctx, { text: again.text, channel: studio });
    expect(reply?.decision).toBe("refused");
    expect(reply?.reasons).toContain("already_admitted");
  });

  test("no passphrase in the Keychain: refused, nothing signed", async () => {
    const { ctx, calls } = setup({ live: true, passphrase: null });
    const { reply } = await processItem(ctx, { text: makeRequest().text, channel: studio });
    expect(reply?.reasons).toEqual(["passphrase_unavailable"]);
    expect(calls.some((c) => c.args[1] === "sign")).toBe(false);
  });

  test("a signature by a key other than R's on-chain key is never broadcast", async () => {
    const { ctx, calls } = setup({ live: true, zeroned: { signWith: Buffer.alloc(33, 2).toString("base64") } });
    const { reply } = await processItem(ctx, { text: makeRequest().text, channel: studio });
    expect(reply?.reasons).toEqual(["signed_tx_mismatch"]);
    expect(calls.some((c) => c.args[1] === "broadcast")).toBe(false);
  });

  test("a CheckTx rejection is recorded as rejected and does not count as admitted", async () => {
    const { ctx } = setup({ live: true, zeroned: { broadcastCode: 13 } });
    const { reply } = await processItem(ctx, { text: makeRequest().text, channel: studio });
    expect(reply?.decision).toBe("failed");
    expect(reply?.reasons).toEqual(["broadcast_rejected"]);
    expect(readLedger(ctx).entries[0]?.status).toBe("rejected");
  });

  test("an inclusion timeout stays pending, is never re-broadcast, and a later run settles it", async () => {
    const { ctx, state, calls } = setup({ live: true, zeroned: { include: false } });
    const req = makeRequest();
    const { reply } = await processItem(ctx, { text: req.text, channel: studio });
    expect(reply?.decision).toBe("pending");
    expect(reply?.reasons).toEqual(["inclusion_timeout"]);
    const entry = readLedger(ctx).entries[0]!;
    expect(entry.status).toBe("pending_inclusion");

    // The transaction lands later; the next run only looks it up.
    state.pots[`bootstrap-${req.address}`] = restPot(req.address);
    state.grants.push(restDoorGrant(R, req.address, entry.grant_expiration));
    state.txs[entry.tx_hash] = { height: "1402802", tx_result: { code: 0, log: "", gas_used: "120000", gas_wanted: "150000" } };
    const broadcasts = calls.filter((c) => c.args[1] === "broadcast").length;
    await reconcilePending(ctx);
    expect(calls.filter((c) => c.args[1] === "broadcast").length).toBe(broadcasts);
    expect(readLedger(ctx).entries[0]?.status).toBe("included");
    const settled = JSON.parse(readFileSync(join(ctx.paths.replies, `${entry.request_id}.json`), "utf8"));
    expect(settled.decision).toBe("admitted");
  });

  test("a transaction that ran and failed in its messages (R's sequence used) is a final failure", async () => {
    const { ctx, state, calls } = setup({ live: true, zeroned: { include: false } });
    const req = makeRequest();
    await processItem(ctx, { text: req.text, channel: studio });
    const entry = readLedger(ctx).entries[0]!;
    // Out of gas: the ante handler passed (R's sequence moved, acc_seq names it), the messages failed.
    land(state, JSON.parse(calls.find((c) => c.args[1] === "broadcast")!.file!), entry.tx_hash, 1_402_802, 11);
    await reconcilePending(ctx);
    expect(readLedger(ctx).entries[0]).toMatchObject({ status: "delivery_failed", height: 1_402_802, chain_window: Math.floor(1_402_802 / 34_272) });
  });

  test("a re-grant revokes an expired stored grant and grants afresh, once", async () => {
    const req = makeRequest();
    const { ctx, state, calls } = setup({ live: true });
    const ledger = readLedger(ctx);
    ledger.entries.push({
      request_id: "0".repeat(64),
      kind: "admit",
      address: req.address,
      address_payload: addressPayloadHex(req.address)!,
      identity_public_key: req.identityHex,
      did: `did:zrn:${req.identityHex}`,
      channel_key: "studio_local",
      tx_hash: "B".repeat(64),
      status: "included",
      chain_window: 1,
      utc_day: "2026-10-01",
      at: "2026-10-01T00:00:00.000Z",
      grant_expiration: "2026-10-15T00:00:00Z",
    });
    writeFileSync(join(ctx.paths.state, "ledger.json"), JSON.stringify(ledger), { mode: 0o600 });
    state.pots[`bootstrap-${req.address}`] = restPot(req.address);
    state.grants.push(restDoorGrant(R, req.address, "2026-10-15T00:00:00Z"));
    state.auth[req.address] = { account_number: "99", sequence: "0", pub_key: null };
    const { reply } = await processItem(ctx, { text: req.text, channel: studio });
    expect(reply?.decision).toBe("regranted");
    const unsigned = JSON.parse(calls.find((c) => c.args[1] === "sign")!.file!);
    expect(unsigned.body.messages.map((m: any) => m["@type"])).toEqual([
      "/cosmos.feegrant.v1beta1.MsgRevokeAllowance",
      "/cosmos.feegrant.v1beta1.MsgGrantAllowance",
    ]);
    expect(unsigned.auth_info.fee.gas_limit).toBe("140000");

    state.grants = [];
    const again = await processItem(ctx, { text: makeRequest({ nonceHex: "b".repeat(32) }).text, channel: studio });
    expect(again.reply?.reasons).toContain("already_admitted");
  });
});

describe("a broadcast the door has not seen land", () => {
  const READ_HEIGHT = 1_402_800;
  const T = READ_HEIGHT + TX_TIMEOUT_BLOCKS;
  const at = (minutes: number) => new Date(NOW.getTime() + minutes * 60_000);
  const later = (ctx: DoorContext, minutes: number): DoorContext => ({ ...ctx, now: () => at(minutes) });
  /** The chain moves on: a fresh newest block at `height`. */
  function advance(state: FakeChainState, height: number, minutes: number): void {
    state.height = height;
    state.time = new Date(at(minutes).getTime() - 10_000).toISOString();
  }
  function replyOf(ctx: DoorContext, requestId: string): Reply {
    return JSON.parse(readFileSync(join(ctx.paths.replies, `${requestId}.json`), "utf8"));
  }
  /** A live admission whose broadcast was accepted but not seen in a block. */
  async function lost(state: Partial<FakeChainState> = {}) {
    const s = setup({ live: true, zeroned: { include: false }, state });
    const req = makeRequest();
    const { reply } = await processItem(s.ctx, { text: req.text, channel: studio });
    expect(reply?.decision).toBe("pending");
    const entry = readLedger(s.ctx).entries[0]!;
    const signed = JSON.parse(s.calls.find((c) => c.args[1] === "broadcast")!.file!);
    return { ...s, req, entry, signed, reply: reply! };
  }

  test("every transaction R signs carries a timeout height, kept in the ledger and the reply", async () => {
    const { entry, signed, reply, calls } = await lost();
    expect(signed.body.timeout_height).toBe(String(T));
    expect(entry.timeout_height).toBe(T);
    expect(entry.sequence).toBe("4");
    expect(reply.timeout_height).toBe(T);
    expect(replyText(reply)).toContain(`(not seen in a block yet; it can still run until height ${T})`);
    expect(replyText(reply)).toContain("settles it only when the chain shows whether it ran");
    const built = calls.filter((c) => c.args.includes("--generate-only"));
    expect(built.length).toBe(2);
    for (const c of built) expect(flag(c.args, "--timeout-height")).toBe(String(T));
  });

  test("delayed past 30 minutes it stays pending, keeps the door busy and asks for attention; when it lands it is settled and counted", async () => {
    const { ctx, state, req, entry, signed, logs } = await lost();
    // 31 minutes on, blocks still coming, R's sequence unmoved, the timeout
    // not reached: nothing is proven, so nothing is freed. Blocks are slower
    // than 30 s, so the door is past the time it should have settled in.
    advance(state, T - 1, 31);
    await reconcilePending(later(ctx, 31));
    expect(readLedger(ctx).entries[0]?.status).toBe("pending_inclusion");
    expect(logs.filter((l) => l.event === "pending_needs_attention")).toEqual([
      { event: "pending_needs_attention", request_id: entry.request_id, tx_hash: entry.tx_hash, signed_at: entry.at, overdue: true, detail: `can still land until height ${T}; the node is at ${T - 1}` },
    ]);
    const other = makeRequest({ label: "agent-2", issuedAt: at(31).toISOString() });
    const busy = await processItem(later(ctx, 31), { text: other.text, channel: studio });
    expect(busy.reply?.reasons).toEqual(["door_busy"]);

    // It lands in the last block that may run it.
    land(state, signed, entry.tx_hash, T);
    advance(state, T + 1, 32);
    await reconcilePending(later(ctx, 32));
    expect(readLedger(ctx).entries[0]).toMatchObject({ status: "included", height: T, chain_window: Math.floor(T / BOOTSTRAP_WINDOW_BLOCKS) });
    expect(replyOf(ctx, entry.request_id)).toMatchObject({ decision: "admitted", height: T });
    // Counted: in the caps, in the admittee watch, and against a second admission.
    expect(admitteesOf(readLedger(ctx)).map((a) => a.address)).toEqual([req.address]);
    const again = makeRequest({ nonceHex: "e".repeat(32), issuedAt: at(32).toISOString() });
    const second = await processItem(later(ctx, 32), { text: again.text, channel: studio });
    expect(second.reply?.reasons).toContain("already_admitted");
  }, 30_000);

  test("past its timeout height, never seen and R's sequence unmoved: expired, and the door is free", async () => {
    const { ctx, state, entry, calls } = await lost();
    // The node's index gets TIMEOUT_SETTLE_BLOCKS blocks to show it.
    advance(state, T + TIMEOUT_SETTLE_BLOCKS - 1, 21);
    await reconcilePending(later(ctx, 21));
    expect(readLedger(ctx).entries[0]?.status).toBe("pending_inclusion");
    advance(state, T + TIMEOUT_SETTLE_BLOCKS, 22);
    await reconcilePending(later(ctx, 22));
    expect(readLedger(ctx).entries[0]?.status).toBe("expired");
    const failed = replyOf(ctx, entry.request_id);
    expect(failed).toMatchObject({ decision: "failed", reasons: ["expired"] });
    expect(failed.detail).toContain(`R's sequence is still 4 at height ${T + TIMEOUT_SETTLE_BLOCKS}`);

    // The agent may ask again, and R signs with the sequence the expired one never used.
    const again = makeRequest({ nonceHex: "d".repeat(32), issuedAt: at(22).toISOString() });
    const { reply } = await processItem(later(ctx, 22), { text: again.text, channel: studio });
    expect(reply?.decision).toBe("pending");
    expect(reply?.timeout_height).toBe(T + TIMEOUT_SETTLE_BLOCKS + TX_TIMEOUT_BLOCKS);
    expect(flag(calls.filter((c) => c.args[1] === "sign").at(-1)!.args, "--sequence")).toBe("4");
  }, 30_000);

  test("refused in its block before it ran (R's sequence unmoved): pending until its timeout height passes, then failed and not counted", async () => {
    const { ctx, state, calls } = setup({ live: true, zeroned: { refuseInBlock: 5 } });
    const { reply } = await processItem(ctx, { text: makeRequest().text, channel: studio });
    // Not final: the same signed transaction can still run in a later block.
    expect(reply).toMatchObject({ decision: "pending", reasons: ["delivery_failed"], timeout_height: T });
    expect(reply?.detail).toContain(`in block ${READ_HEIGHT + 1} with code 5`);
    expect(reply?.detail).toContain(`can still run until height ${T}`);
    expect(replyText(reply!)).toContain(`(a block refused it before it ran; it can still run until height ${T})`);
    const entry = readLedger(ctx).entries[0]!;
    expect(entry.status).toBe("pending_inclusion");

    advance(state, T + TIMEOUT_SETTLE_BLOCKS - 1, 21);
    await reconcilePending(later(ctx, 21));
    expect(readLedger(ctx).entries[0]?.status).toBe("pending_inclusion");
    const other = makeRequest({ label: "agent-2", issuedAt: at(21).toISOString() });
    expect((await processItem(later(ctx, 21), { text: other.text, channel: studio })).reply?.reasons).toEqual(["door_busy"]);

    // Past its timeout height, R's sequence still 4: it never ran and never can.
    advance(state, T + TIMEOUT_SETTLE_BLOCKS, 22);
    await reconcilePending(later(ctx, 22));
    expect(readLedger(ctx).entries[0]).toMatchObject({ status: "delivery_failed", height: READ_HEIGHT + 1 });
    const failed = replyOf(ctx, entry.request_id);
    expect(failed).toMatchObject({ decision: "failed", reasons: ["delivery_failed"], height: READ_HEIGHT + 1 });
    expect(failed.detail).toContain(`R's sequence is still 4 at height ${T + TIMEOUT_SETTLE_BLOCKS}`);

    // Not counted: the agent may ask again, and R signs with the sequence it never used.
    const again = makeRequest({ nonceHex: "d".repeat(32), issuedAt: at(22).toISOString() });
    const next = await processItem(later(ctx, 22), { text: again.text, channel: studio });
    expect(next.reply?.decision).toBe("pending");
    expect(flag(calls.filter((c) => c.args[1] === "sign").at(-1)!.args, "--sequence")).toBe("4");
  }, 30_000);

  test("refused in its block, then run by a later block before its timeout height: admitted and counted", async () => {
    const { ctx, state, calls } = setup({ live: true, zeroned: { refuseInBlock: 5 } });
    const req = makeRequest();
    await processItem(ctx, { text: req.text, channel: studio });
    const entry = readLedger(ctx).entries[0]!;
    // The same signed bytes run once R can pay; the index keeps the success.
    land(state, JSON.parse(calls.find((c) => c.args[1] === "broadcast")!.file!), entry.tx_hash, READ_HEIGHT + 9);
    advance(state, READ_HEIGHT + 10, 6);
    await reconcilePending(later(ctx, 6));
    expect(readLedger(ctx).entries[0]).toMatchObject({ status: "included", height: READ_HEIGHT + 9 });
    expect(replyOf(ctx, entry.request_id)).toMatchObject({ decision: "admitted", height: READ_HEIGHT + 9 });
    expect(admitteesOf(readLedger(ctx)).map((a) => a.address)).toEqual([req.address]);
  });

  test("refused in its block, and another transaction then used its sequence: failed at once", async () => {
    const { ctx, state } = setup({ live: true, zeroned: { refuseInBlock: 5 } });
    await processItem(ctx, { text: makeRequest().text, channel: studio });
    const entry = readLedger(ctx).entries[0]!;
    const other = "D".repeat(64);
    state.auth[R]!.sequence = "5";
    state.txs[other] = otherTx(READ_HEIGHT + 3);
    state.accSeq[`${R}/4`] = [other];
    advance(state, READ_HEIGHT + 5, 3);
    await reconcilePending(later(ctx, 3));
    expect(readLedger(ctx).entries[0]).toMatchObject({ status: "delivery_failed", height: READ_HEIGHT + 1 });
  });

  test("expired needs R's sequence read on the RPC node at or past the timeout height, never over REST", async () => {
    const { ctx, state, entry, signed } = await lost();
    // It ran in block T. The node's index has not caught up, the REST node
    // lags where R's sequence is still 4, and so does the backend answering
    // the RPC account query (still at T - 1).
    land(state, signed, entry.tx_hash, T);
    const indexed = state.txs[entry.tx_hash]!;
    delete state.txs[entry.tx_hash];
    delete state.accSeq[`${R}/4`];
    state.restSequence[R] = "4";
    state.laggingAccountRead = { height: T - 1, registrarSequence: "4" };
    advance(state, T + TIMEOUT_SETTLE_BLOCKS, 22);
    const from = state.requests.length;
    await reconcilePending(later(ctx, 22));
    expect(readLedger(ctx).entries[0]?.status).toBe("pending_inclusion");
    expect(replyOf(ctx, entry.request_id).decision).toBe("pending");

    // The RPC backend catches up: R's sequence moved, so it did not expire;
    // the index still names nothing, so it stays pending.
    state.laggingAccountRead = null;
    await reconcilePending(later(ctx, 22));
    expect(readLedger(ctx).entries[0]?.status).toBe("pending_inclusion");
    expect(state.requests.slice(from).some((u) => u.includes("/cosmos/auth/v1beta1/accounts/"))).toBe(false);

    // The index catches up: it ran in block T.
    state.txs[entry.tx_hash] = indexed;
    state.accSeq[`${R}/4`] = [entry.tx_hash];
    await reconcilePending(later(ctx, 23));
    expect(readLedger(ctx).entries[0]).toMatchObject({ status: "included", height: T });
    expect(replyOf(ctx, entry.request_id)).toMatchObject({ decision: "admitted", height: T });
  });

  test("R's sequence used by another transaction: superseded, naming that transaction", async () => {
    const { ctx, state, entry } = await lost();
    const other = "D".repeat(64);
    state.auth[R]!.sequence = "5";
    state.txs[other] = otherTx(1_402_803);
    state.accSeq[`${R}/4`] = [other];
    // Before the timeout height: the used sequence alone proves it.
    advance(state, READ_HEIGHT + 5, 3);
    await reconcilePending(later(ctx, 3));
    expect(readLedger(ctx).entries[0]?.status).toBe("superseded");
    const failed = replyOf(ctx, entry.request_id);
    expect(failed.reasons).toEqual(["superseded"]);
    expect(failed.detail).toContain(other);
  });

  test("its own transaction indexed under another hash: found by its messages, not superseded", async () => {
    const { ctx, state, entry, signed, logs } = await lost();
    // The chain holds the signed admission under a hash the door did not compute.
    const chainHash = "AB".repeat(32);
    land(state, signed, chainHash, READ_HEIGHT + 2);
    advance(state, READ_HEIGHT + 4, 2);
    await reconcilePending(later(ctx, 2));
    expect(readLedger(ctx).entries[0]).toMatchObject({ status: "included", tx_hash: chainHash, height: READ_HEIGHT + 2 });
    expect(replyOf(ctx, entry.request_id)).toMatchObject({ decision: "admitted", tx_hash: chainHash, height: READ_HEIGHT + 2 });
    expect(logs).toContainEqual({ event: "hash_differs", request_id: entry.request_id, computed: entry.tx_hash, chain: chainHash });
  });

  test("an admission of another address under R's sequence is another transaction: superseded", async () => {
    const { ctx, state, entry, signed } = await lost();
    const elsewhere = addressFor("someone-else");
    const theirs = structuredClone(signed);
    theirs.body.messages[0].addresses = [elsewhere];
    theirs.body.messages[1].grantee = elsewhere;
    const theirHash = "CD".repeat(32);
    land(state, theirs, theirHash, READ_HEIGHT + 2);
    advance(state, READ_HEIGHT + 4, 2);
    await reconcilePending(later(ctx, 2));
    expect(readLedger(ctx).entries[0]?.status).toBe("superseded");
    expect(replyOf(ctx, entry.request_id).detail).toContain(`used by tx ${theirHash}`);
  });

  test("reads that keep failing: still pending and busy, and past the time it should have settled in, every run asks for attention", async () => {
    const { ctx, state, entry, logs } = await lost();
    const asked = () => logs.filter((l) => l.event === "pending_needs_attention");
    // The node answers its status but every transaction lookup fails.
    state.fail.add("/tx");
    advance(state, READ_HEIGHT + 20, 10);
    await reconcilePending(later(ctx, 10));
    expect(asked()).toEqual([]);
    const settleBy = PENDING_ATTENTION_MS / 60_000;
    expect(settleBy).toBe(21.5);
    advance(state, T + 10, settleBy);
    await reconcilePending(later(ctx, settleBy));
    expect(asked()).toHaveLength(1);
    expect(asked()[0]).toMatchObject({ request_id: entry.request_id, signed_at: entry.at, overdue: true });
    expect(String(asked()[0]!.detail)).toContain("rpc.test:26657/tx: HTTP 503");
    await reconcilePending(later(ctx, settleBy + 2));
    expect(asked()).toHaveLength(2);
    // Time never frees it: still pending, still counted, still busy.
    expect(readLedger(ctx).entries[0]?.status).toBe("pending_inclusion");
    const other = makeRequest({ label: "agent-2", issuedAt: at(settleBy + 2).toISOString() });
    expect((await processItem(later(ctx, settleBy + 2), { text: other.text, channel: studio })).reply?.reasons).toEqual(["door_busy"]);
  });

  test("R's sequence moved but the index names no transaction for it yet: still pending, even past the timeout", async () => {
    const { ctx, state, logs } = await lost();
    state.auth[R]!.sequence = "5";
    advance(state, T + 10, 25);
    await reconcilePending(later(ctx, 25));
    expect(readLedger(ctx).entries[0]?.status).toBe("pending_inclusion");
    expect(logs.some((l) => l.event === "pending_needs_attention")).toBe(true);
  });

  test("the acc_seq index naming this very transaction settles it, though the hash lookup missed it", async () => {
    const { ctx, state, entry, signed } = await lost();
    land(state, signed, entry.tx_hash, READ_HEIGHT + 2);
    // The by-hash lookup misses it; the acc_seq search answers.
    const chainFetch = fakeFetch(state);
    const missesByHash: FetchLike = async (url, init) => new URL(url).pathname === "/tx"
      ? { status: 200, text: async () => JSON.stringify({ error: { code: -32603, message: "Internal error", data: `tx (${entry.tx_hash}) not found` } }) }
      : chainFetch(url, init);
    const chain = new ChainReader({ rpc: ctx.config.rpc, rest: ctx.config.rest, retries: 1, fetch: missesByHash });
    advance(state, READ_HEIGHT + 4, 2);
    await reconcilePending({ ...later(ctx, 2), chain });
    expect(readLedger(ctx).entries[0]).toMatchObject({ status: "included", height: READ_HEIGHT + 2 });
  });
});

describe("the chain window an admission counts in", () => {
  const W = 41;
  const BOUNDARY = W * BOOTSTRAP_WINDOW_BLOCKS;

  function windowFull(ctx: DoorContext): void {
    // 19 earlier door admissions already ran in window W.
    const ledger = readLedger(ctx);
    for (let i = 0; i < DOOR_WINDOW_CAP - 1; i += 1) {
      const a = addressFor(`window-${i}`);
      ledger.entries.push({
        request_id: String(i).padStart(64, "f"), kind: "admit", address: a, address_payload: addressPayloadHex(a)!,
        identity_public_key: String(i).padStart(64, "1"), did: `did:zrn:${String(i).padStart(64, "1")}`, channel_key: "studio_local",
        tx_hash: String(i).padStart(64, "A"), status: "included", chain_window: W, utc_day: "2026-10-20",
        at: "2026-10-20T00:00:00.000Z", grant_expiration: "2026-11-03T00:00:00Z", height: BOUNDARY + 1,
      });
    }
    writeFileSync(join(ctx.paths.state, "ledger.json"), JSON.stringify(ledger), { mode: 0o600 });
  }

  test("built in the last block of a window and included in the next, in this run: it counts in the next", async () => {
    const { ctx, state } = setup({ live: true, state: { height: BOUNDARY - 1 } });
    const { reply } = await processItem(ctx, { text: makeRequest().text, channel: studio });
    expect(reply).toMatchObject({ decision: "admitted", height: BOUNDARY });
    expect(readLedger(ctx).entries[0]).toMatchObject({ status: "included", height: BOUNDARY, chain_window: W });
    // The door's window cap for W now counts it.
    windowFull(ctx);
    state.height = BOUNDARY + 5;
    const next = await processItem(ctx, { text: makeRequest({ label: "agent-3" }).text, channel: studio });
    expect(next.reply?.reasons).toEqual(["door_window_cap"]);
  }, 30_000);

  test("built before a window boundary, settled by a later run after it: it counts in the window it ran in", async () => {
    const { ctx, state, calls } = setup({ live: true, zeroned: { include: false }, state: { height: BOUNDARY - 2 } });
    await processItem(ctx, { text: makeRequest().text, channel: studio });
    const pending = readLedger(ctx).entries[0]!;
    expect(pending).toMatchObject({ status: "pending_inclusion", chain_window: W - 1, timeout_height: BOUNDARY - 2 + TX_TIMEOUT_BLOCKS });
    land(state, JSON.parse(calls.find((c) => c.args[1] === "broadcast")!.file!), pending.tx_hash, BOUNDARY + 1);
    state.height = BOUNDARY + 2;
    await reconcilePending(ctx);
    expect(readLedger(ctx).entries[0]).toMatchObject({ status: "included", height: BOUNDARY + 1, chain_window: W });
  });
});

describe("a ledger from before timeout heights", () => {
  function droppedEntry(address: string, identityHex: string): LedgerEntry {
    return {
      request_id: "9".repeat(64), kind: "admit", address, address_payload: addressPayloadHex(address)!, identity_public_key: identityHex,
      did: `did:zrn:${identityHex}`, channel_key: "studio_local", tx_hash: "E".repeat(64), status: "dropped", sequence: "4",
      chain_window: 40, utc_day: "2026-10-28", at: "2026-10-28T16:00:00.000Z", grant_expiration: "2026-11-11T16:00:00Z",
      detail: "never included; R's sequence did not move",
    };
  }
  function withLedger(ctx: DoorContext, entries: LedgerEntry[]): void {
    const ledger = readLedger(ctx);
    ledger.entries.push(...entries);
    ledger.seen["9".repeat(64)] = "2026-10-28T16:00:00.000Z";
    writeFileSync(join(ctx.paths.state, "ledger.json"), JSON.stringify(ledger), { mode: 0o600 });
  }

  test("a 'dropped' transaction that landed after all is settled as admitted and counted", async () => {
    const a = makeRequest({ label: "legacy-landed" });
    const { ctx, state } = setup({ live: true });
    const old = droppedEntry(a.address, a.identityHex);
    withLedger(ctx, [old]);
    state.pots[`bootstrap-${a.address}`] = restPot(a.address);
    state.grants.push(restDoorGrant(R, a.address, old.grant_expiration));
    state.txs[old.tx_hash] = { height: "1402790", tx_result: { code: 0, log: "", gas_used: "120000", gas_wanted: "150000" } };
    state.auth[R]!.sequence = "5";
    await reconcilePending(ctx);
    expect(readLedger(ctx).entries[0]).toMatchObject({ status: "included", height: 1_402_790, chain_window: Math.floor(1_402_790 / BOOTSTRAP_WINDOW_BLOCKS) });
    expect(JSON.parse(readFileSync(join(ctx.paths.replies, `${old.request_id}.json`), "utf8")).decision).toBe("admitted");
  });

  test("a 'dropped' transaction never seen is re-opened as pending and keeps the door busy until R's sequence moves", async () => {
    const a = makeRequest({ label: "legacy-unseen" });
    const { ctx, state, logs } = setup({ live: true });
    const old = droppedEntry(a.address, a.identityHex);
    withLedger(ctx, [old]);
    await reconcilePending(ctx);
    expect(readLedger(ctx).entries[0]?.status).toBe("pending_inclusion");
    expect(JSON.parse(readFileSync(join(ctx.paths.replies, `${old.request_id}.json`), "utf8")).decision).toBe("pending");
    expect(logs.map((l) => l.event)).toEqual(["dropped_reopened", "pending_needs_attention"]);
    const busy = await processItem(ctx, { text: makeRequest({ label: "legacy-next" }).text, channel: studio });
    expect(busy.reply?.reasons).toEqual(["door_busy"]);

    // Any R transaction that uses sequence 4 settles it.
    const other = "F".repeat(64);
    state.auth[R]!.sequence = "5";
    state.txs[other] = otherTx(1_402_805);
    state.accSeq[`${R}/4`] = [other];
    await reconcilePending(ctx);
    expect(readLedger(ctx).entries[0]?.status).toBe("superseded");
  }, 30_000);
});

describe("what is signed", () => {
  test("checkBuiltTx refuses anything but the exact admission", () => {
    const addr = makeRequest().address;
    const good = {
      body: {
        messages: [
          { "@type": "/zerone.claiming_pot.v1.MsgAddBootstrapEntry", authority: R, addresses: [addr] },
          { "@type": "/cosmos.feegrant.v1beta1.MsgGrantAllowance", granter: R, grantee: addr, allowance: restDoorGrant(R, addr, "2026-11-11T17:31:00Z").allowance },
        ],
        memo: "zerone-door/v1", timeout_height: "1402840", extension_options: [], non_critical_extension_options: [],
      },
      auth_info: { signer_infos: [], fee: { amount: [{ denom: "uzrn", amount: "150000" }], gas_limit: "150000", payer: "", granter: "" } },
      signatures: [],
    };
    const want = { kind: "admit" as const, registrar: R, address: addr, expiration: "2026-11-11T17:31:00Z", revokeFirst: false, timeoutHeight: 1_402_840 };
    expect(checkBuiltTx(good, want)).toEqual([]);
    const noTimeout = structuredClone(good);
    noTimeout.body.timeout_height = "0";
    expect(checkBuiltTx(noTimeout, want)).toEqual(["timeout height"]);
    expect(checkBuiltTx(good, { ...want, timeoutHeight: 1_402_841 })).toEqual(["timeout height"]);
    const twoAddresses = structuredClone(good);
    twoAddresses.body.messages[0]!.addresses = [addr, R];
    expect(checkBuiltTx(twoAddresses, want)).toContain("bootstrap addresses");
    const wider = structuredClone(good);
    (wider.body.messages[1] as any).allowance.allowed_messages.push("/cosmos.bank.v1beta1.MsgSend");
    expect(checkBuiltTx(wider, want)).toContain("allowed messages");
    const unbounded = structuredClone(good);
    (unbounded.body.messages[1] as any).allowance.allowance.basic.spend_limit = [];
    expect(checkBuiltTx(unbounded, want)).toContain("spend limit");
    const feeGranted = structuredClone(good);
    feeGranted.auth_info.fee.granter = addr;
    expect(checkBuiltTx(feeGranted, want)).toContain("fee");
  });

  test("a transaction in a block is the door's only when its messages are: the bootstrap entry for the address and the grant from R", () => {
    // The genesis registrar's July admission of one agent, as two transactions
    // (h103,864 and h103,865), as zerone-1's tx_search served them on 2026-10-29.
    const add = Buffer.from(
      "CosBCogBCiwvemVyb25lLmNsYWltaW5nX3BvdC52MS5Nc2dBZGRCb290c3RyYXBFbnRyeRJYCip6cm4xN2g1c2N2M3p1N3hhOGVwOWthcXk0N2FlMDhoOXg2YzVmYW53a2gSKnpybjFtdGE4dWd5MndudDRoenA0aGR2bXdoODcyNTZuejZ6aDJjMGV4eRJoClAKRgofL2Nvc21vcy5jcnlwdG8uc2VjcDI1NmsxLlB1YktleRIjCiEDxGTJBXCDNZlNLqxaYD2A3XU5t2mC5rENxz4I5RebH1sSBAoCCAEYDRIUCg4KBHV6cm4SBjIwMDAwMBDAmgwaQB95FyBZr4xUbqeu57+2/ywzTRaR3rzo30+3ArYJyrsIYBWciV8asEpJyJtWYzq2g7nA0p1paNdACrHybHjYRK8=",
      "base64");
    const grant = Buffer.from(
      "CqQCCqECCiovY29zbW9zLmZlZWdyYW50LnYxYmV0YTEuTXNnR3JhbnRBbGxvd2FuY2US8gEKKnpybjE3aDVzY3YzenU3eGE4ZXA5a2FxeTQ3YWUwOGg5eDZjNWZhbndraBIqenJuMW10YTh1Z3kyd250NGh6cDRoZHZtd2g4NzI1Nm56NnpoMmMwZXh5GpcBCiwvY29zbW9zLmZlZWdyYW50LnYxYmV0YTEuQWxsb3dlZE1zZ0FsbG93YW5jZRJnCkMKJy9jb3Ntb3MuZmVlZ3JhbnQudjFiZXRhMS5CYXNpY0FsbG93YW5jZRIYCg4KBHV6cm4SBjUwMDAwMBIGCP3l59MGEiAvemVyb25lLmNsYWltaW5nX3BvdC52MS5Nc2dDbGFpbRJoClAKRgofL2Nvc21vcy5jcnlwdG8uc2VjcDI1NmsxLlB1YktleRIjCiEDxGTJBXCDNZlNLqxaYD2A3XU5t2mC5rENxz4I5RebH1sSBAoCCAEYDhIUCg4KBHV6cm4SBjIwMDAwMBDAmgwaQKJQfLf1wWUDj1HauvrPiQOXqB2tGbi96JSwXfYAbn3mILq5RCwKR9GarD5q+Ym+FJprnqILuky7rbEyKRgffEY=",
      "base64");
    expect(createHash("sha256").update(add).digest("hex").toUpperCase()).toBe("B60F67E9CDBFF1C58BBE9F9A2EC33E3E48CBF865AACB882EC816A378D36E8C16");
    expect(createHash("sha256").update(grant).digest("hex").toUpperCase()).toBe("AC4E51A6DCFF4DCE45B543C255CC9490C8612716FBC330E8B0BA2F329BBCD907");
    const registrar = "zrn17h5scv3zu7xa8ep9kaqy47ae08h9x6c5fanwkh";
    const agent = "zrn1mta8ugy2wnt4hzp4hdvmwh87256nz6zh2c0exy";
    expect(txMessagesOf(add)).toEqual([{ type: "/zerone.claiming_pot.v1.MsgAddBootstrapEntry", authority: registrar, addresses: [agent] }]);
    expect(txMessagesOf(grant)).toEqual([{ type: "/cosmos.feegrant.v1beta1.MsgGrantAllowance", granter: registrar, grantee: agent }]);
    expect(() => txMessagesOf(new Uint8Array())).toThrow("no body");
    expect(() => txMessagesOf(add.subarray(0, add.length - 10))).toThrow("truncated");

    const [addMsg] = txMessagesOf(add);
    const [grantMsg] = txMessagesOf(grant);
    const admit = { kind: "admit" as const, address: agent };
    expect(carriesEntry([addMsg!, grantMsg!], admit, registrar)).toBe(true);
    // The door signs both in one transaction; each alone, or reordered, is not it.
    expect(carriesEntry([addMsg!], admit, registrar)).toBe(false);
    expect(carriesEntry([grantMsg!, addMsg!], admit, registrar)).toBe(false);
    expect(carriesEntry([{ ...addMsg!, addresses: [agent, registrar] }, grantMsg!], admit, registrar)).toBe(false);
    expect(carriesEntry([{ ...addMsg!, authority: agent }, grantMsg!], admit, registrar)).toBe(false);
    expect(carriesEntry([addMsg!, { ...grantMsg!, grantee: registrar }], admit, registrar)).toBe(false);
    expect(carriesEntry([addMsg!, grantMsg!], admit, R)).toBe(false);
    const regrant = { kind: "regrant" as const, address: agent };
    const revoke = { type: "/cosmos.feegrant.v1beta1.MsgRevokeAllowance", granter: registrar, grantee: agent };
    expect(carriesEntry([grantMsg!], regrant, registrar)).toBe(true);
    expect(carriesEntry([revoke, grantMsg!], regrant, registrar)).toBe(true);
    expect(carriesEntry([addMsg!, grantMsg!], regrant, registrar)).toBe(false);
  });

  test("the postcheck accepts only the allowance the door built", () => {
    const g = grantFromR("zrn1x", "2026-11-11T17:31:00Z", 900_000n);
    expect(allowanceMatches(g, "2026-11-11T17:31:00Z")).toBe(true);
    expect(allowanceMatches(g, "2026-11-12T17:31:00Z")).toBe(false);
    expect(allowanceMatches({ ...g, allowance: { ...g.allowance, spendLimitUzrn: 5_000_000n } }, "2026-11-11T17:31:00Z")).toBe(false);
    expect(allowanceMatches({ ...g, allowance: { ...g.allowance, periodLimitUzrn: 900_000n } }, "2026-11-11T17:31:00Z")).toBe(false);
  });
});

describe("the public stand-in for R", () => {
  test("simulation carries R's public key and live sequence and no signature", async () => {
    const { ctx, calls } = setup();
    await processItem(ctx, { text: makeRequest().text, channel: studio });
    const encode = calls.find((c) => c.args[1] === "encode")!;
    const tx = JSON.parse(encode.file!);
    expect(tx.auth_info.signer_infos).toEqual([{
      public_key: { "@type": "/cosmos.crypto.secp256k1.PubKey", key: R_PUBKEY_B64 },
      mode_info: { single: { mode: "SIGN_MODE_DIRECT" } },
      sequence: "4",
    }]);
    expect(tx.signatures).toEqual([""]);
  });
});

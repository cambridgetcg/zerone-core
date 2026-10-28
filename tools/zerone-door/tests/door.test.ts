import { describe, expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { existsSync, readdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { addressPayloadHex } from "../src/bech32.ts";
import { ChainReader } from "../src/chain.ts";
import { doorPaths } from "../src/config.ts";
import { allowanceMatches, processItem, readLedger, reconcilePending, replyText, type DoorContext, type Reply } from "../src/door.ts";
import type { LobbyRunner } from "../src/intake.ts";
import { loadLedger, type LedgerEntry } from "../src/ledger.ts";
import { runOnce } from "../src/run.ts";
import type { ZeronedRunner } from "../src/tx.ts";
import { checkBuiltTx } from "../src/tx.ts";
import { admitteesOf, judgeAdmittee, watchAdmittees } from "../src/watch.ts";
import {
  fakeChainState,
  fakeFetch,
  grantFromR,
  makeDoorHome,
  makeRequest,
  NOW,
  R,
  R_PUBKEY_B64,
  restDoorGrant,
  restPot,
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

function genTx(messages: unknown[]) {
  return {
    body: { messages, memo: "", timeout_height: "0", extension_options: [], non_critical_extension_options: [] },
    auth_info: { signer_infos: [], fee: { amount: [{ denom: "uzrn", amount: "200000" }], gas_limit: "200000", payer: "", granter: "" }, tip: null },
    signatures: [],
  };
}

/** A stand-in for the pinned CLI that builds the same JSON shapes zeroned
 * 2e37c4c prints, and applies a broadcast admission to the fake chain. */
function fakeZeroned(state: FakeChainState, calls: Call[], opts: { broadcastCode?: number; include?: boolean; signWith?: string } = {}): ZeronedRunner {
  return (args, stdin) => {
    const reads = args[0] === "tx" && ["encode", "sign", "broadcast"].includes(args[1]!) ? readFileSync(args[2]!, "utf8") : null;
    calls.push({ args, stdin: stdin ? Buffer.from(stdin).toString("utf8") : null, file: reads });
    const ok = (stdout: string) => ({ code: 0, stdout, stderr: "" });
    if (args[0] === "tx" && args[1] === "claiming_pot" && args[2] === "add-bootstrap-entry") {
      return ok(JSON.stringify(genTx([{ "@type": "/zerone.claiming_pot.v1.MsgAddBootstrapEntry", authority: flag(args, "--from"), addresses: [args[3]] }])));
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
      }])));
    }
    if (args[0] === "tx" && args[1] === "feegrant" && args[2] === "revoke") {
      return ok(JSON.stringify(genTx([{ "@type": "/cosmos.feegrant.v1beta1.MsgRevokeAllowance", granter: args[3], grantee: args[4] }])));
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
      if (code === 0 && opts.include !== false) {
        const tx = JSON.parse(raw.toString("utf8"));
        for (const m of tx.body.messages) {
          if (m["@type"].endsWith("MsgAddBootstrapEntry")) state.pots[`bootstrap-${m.addresses[0]}`] = restPot(m.addresses[0]);
          if (m["@type"].endsWith("MsgRevokeAllowance")) state.grants = state.grants.filter((g) => !(g.granter === m.granter && g.grantee === m.grantee));
          if (m["@type"].endsWith("MsgGrantAllowance")) state.grants.push({ granter: m.granter, grantee: m.grantee, allowance: m.allowance });
        }
        state.txs[hash] = { height: String(state.height + 1), tx_result: { code: 0, log: "", gas_used: "121000", gas_wanted: "150000" } };
      }
      return ok(JSON.stringify({ height: "0", txhash: hash, code, raw_log: code ? "insufficient fees" : "" }));
    }
    return { code: 1, stdout: "", stderr: `unexpected zeroned call ${args.slice(0, 3).join(" ")}` };
  };
}

function setup(opts: { live?: boolean; state?: Partial<FakeChainState>; zeroned?: { broadcastCode?: number; include?: boolean; signWith?: string }; passphrase?: string | null; home?: Parameters<typeof makeDoorHome>[0] } = {}) {
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

  test("a broadcast never seen while R's sequence stays put is dropped after 30 minutes, freeing the agent", async () => {
    const { ctx, state } = setup({ live: true, zeroned: { include: false } });
    const req = makeRequest();
    await processItem(ctx, { text: req.text, channel: studio });
    expect(readLedger(ctx).entries[0]?.sequence).toBe("4");
    await reconcilePending({ ...ctx, now: () => new Date(NOW.getTime() + 10 * 60_000) });
    expect(readLedger(ctx).entries[0]?.status).toBe("pending_inclusion");
    // A halted chain proves nothing: stay pending.
    await reconcilePending({ ...ctx, now: () => new Date(NOW.getTime() + 31 * 60_000) });
    expect(readLedger(ctx).entries[0]?.status).toBe("pending_inclusion");
    state.time = new Date(NOW.getTime() + 30 * 60_000).toISOString();
    await reconcilePending({ ...ctx, now: () => new Date(NOW.getTime() + 31 * 60_000) });
    expect(readLedger(ctx).entries[0]?.status).toBe("dropped");
    const reply = JSON.parse(readFileSync(join(ctx.paths.replies, `${readLedger(ctx).entries[0]!.request_id}.json`), "utf8"));
    expect(reply.reasons).toEqual(["dropped"]);
  });

  test("an unseen broadcast is kept pending when R's sequence moved on", async () => {
    const { ctx, state } = setup({ live: true, zeroned: { include: false } });
    await processItem(ctx, { text: makeRequest().text, channel: studio });
    state.auth[R]!.sequence = "5";
    await reconcilePending({ ...ctx, now: () => new Date(NOW.getTime() + 31 * 60_000) });
    expect(readLedger(ctx).entries[0]?.status).toBe("pending_inclusion");
  });

  test("a delivered transaction with a non-zero code is a failure", async () => {
    const { ctx, state } = setup({ live: true, zeroned: { include: false } });
    const req = makeRequest();
    await processItem(ctx, { text: req.text, channel: studio });
    const entry = readLedger(ctx).entries[0]!;
    state.txs[entry.tx_hash] = { height: "1402802", tx_result: { code: 11, log: "out of gas", gas_used: "150001", gas_wanted: "150000" } };
    await reconcilePending(ctx);
    expect(readLedger(ctx).entries[0]?.status).toBe("delivery_failed");
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

describe("what is signed", () => {
  test("checkBuiltTx refuses anything but the exact admission", () => {
    const addr = makeRequest().address;
    const good = {
      body: {
        messages: [
          { "@type": "/zerone.claiming_pot.v1.MsgAddBootstrapEntry", authority: R, addresses: [addr] },
          { "@type": "/cosmos.feegrant.v1beta1.MsgGrantAllowance", granter: R, grantee: addr, allowance: restDoorGrant(R, addr, "2026-11-11T17:31:00Z").allowance },
        ],
        memo: "zerone-door/v1", timeout_height: "0", extension_options: [], non_critical_extension_options: [],
      },
      auth_info: { signer_infos: [], fee: { amount: [{ denom: "uzrn", amount: "150000" }], gas_limit: "150000", payer: "", granter: "" } },
      signatures: [],
    };
    const want = { kind: "admit" as const, registrar: R, address: addr, expiration: "2026-11-11T17:31:00Z", revokeFirst: false };
    expect(checkBuiltTx(good, want)).toEqual([]);
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

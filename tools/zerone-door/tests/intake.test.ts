import { describe, expect, test } from "bun:test";
import { chmodSync, existsSync, mkdirSync, readdirSync, symlinkSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import type { LobbyConfig } from "../src/config.ts";
import { IntakeError, readLobby, replyInLobby, takeSpool } from "../src/intake.ts";
import { makeRequest, NOW, tempDir } from "./helpers.ts";

const lobby: LobbyConfig = { cli: "/opt/test/lobby", room_id: `room_${"a1".repeat(16)}`, conversation: "zerone-door", reply_lease: "lease_1" };
const desk = "3f2b1c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d";

function message(over: Record<string, unknown> = {}) {
  return {
    event_id: "evt_1",
    cursor: "7",
    conversation: "zerone-door",
    kind: "message",
    text: makeRequest().text,
    to_identity: "11111111-1111-4111-8111-111111111111",
    to_me: true,
    in_reply_to: null,
    from: { identity_id: desk, signing_key_id: "k1", fingerprint: "ab12", role: "player", tier: "player", current_member: true, label: "verified", basis: "door_roster_pin" },
    from_me: false,
    received_at: { value: "2026-10-28T17:30:30Z", label: "self-declared", basis: "service_metadata" },
    ...over,
  };
}

const SEASON = "11111111-1111-4111-8111-111111111111";
const NEXT_SEASON = "22222222-2222-4222-8222-222222222222";

function readAnswer(messages: unknown[], next = "7", hasMore = false, mission: string | null = SEASON) {
  return JSON.stringify({
    untrusted_data: true, room_id: lobby.room_id, ...(mission === null ? {} : { mission_id: mission }), conversation: "zerone-door",
    messages, unverified: [{ event_id: "x", cursor: "6", reason: "not_on_roster", authority: "none" }], next_cursor: next, has_more: hasMore,
  });
}

describe("Studio-local spool", () => {
  function inbox() {
    const home = tempDir("spool");
    const inboxDir = join(home, "inbox");
    mkdirSync(inboxDir, { mode: 0o700 });
    return { inboxDir, processed: join(home, "processed") };
  }

  test("takes *.request files once, moving them aside before reading", () => {
    const { inboxDir, processed } = inbox();
    writeFileSync(join(inboxDir, "a.request"), "zerone-door-request/v1\n");
    writeFileSync(join(inboxDir, "notes.txt"), "ignored");
    const items = takeSpool(inboxDir, processed, NOW);
    expect(items).toHaveLength(1);
    expect(items[0]!.channel).toEqual({ kind: "studio_local", file: "a.request" });
    expect(existsSync(join(inboxDir, "a.request"))).toBe(false);
    expect(readdirSync(processed)).toHaveLength(1);
    expect(takeSpool(inboxDir, processed, NOW)).toEqual([]);
  });

  test("an inbox others can write is not a membership proof: refused", () => {
    const { inboxDir, processed } = inbox();
    chmodSync(inboxDir, 0o733);
    expect(() => takeSpool(inboxDir, processed, NOW)).toThrow(IntakeError);
  });

  test("links and oversize files are moved aside unread", () => {
    const { inboxDir, processed } = inbox();
    const outside = join(tempDir("outside"), "secret.txt");
    writeFileSync(outside, "do not read");
    symlinkSync(outside, join(inboxDir, "link.request"));
    writeFileSync(join(inboxDir, "big.request"), "x".repeat(5000));
    expect(takeSpool(inboxDir, processed, NOW)).toEqual([]);
    expect(readdirSync(processed)).toHaveLength(2);
  });
});

describe("lobby table talk", () => {
  test("reads the door conversation after the cursor and keeps the roster verdict", () => {
    const argv: string[][] = [];
    const read = readLobby((args) => {
      argv.push(args);
      return { code: 0, stdout: readAnswer([message(), message({ event_id: "evt_2", from_me: true }), message({ event_id: "evt_3", kind: "read_ack" })]), stderr: "" };
    }, lobby, { mission_id: SEASON, cursor: "5" });
    expect(argv[0]).toEqual(["room", "read", lobby.room_id, "--conversation", "zerone-door", "--after", "5", "--limit", "50", "--json"]);
    expect(read.cursor).toEqual({ mission_id: SEASON, cursor: "7" });
    expect(read.seasonChanged).toBeNull();
    expect(read.unverified).toBe(1);
    expect(read.items).toHaveLength(1);
    expect(read.items[0]!.channel).toMatchObject({ kind: "lobby", identity_id: desk, event_id: "evt_1", verified: true, currentMember: true });
  });

  test("a message without the door-roster label is carried as unverified, never trusted", () => {
    const read = readLobby(() => ({ code: 0, stdout: readAnswer([message({ from: { identity_id: desk, current_member: true, label: "self-declared", basis: "service_metadata" } })]), stderr: "" }), lobby, null);
    expect(read.items[0]!.channel).toMatchObject({ verified: false });
  });

  test("pages until has_more is false", () => {
    let calls = 0;
    const read = readLobby(() => {
      calls += 1;
      return { code: 0, stdout: readAnswer([message({ event_id: `evt_${calls}` })], String(10 + calls), calls < 3), stderr: "" };
    }, lobby, null);
    expect(calls).toBe(3);
    expect(read.cursor).toEqual({ mission_id: SEASON, cursor: "13" });
  });

  test("a new season starts the read over from 0, because receipts restart at 1 in each season", () => {
    const afters: string[] = [];
    const read = readLobby((args) => {
      const after = args[args.indexOf("--after") + 1]!;
      afters.push(after);
      // The new season has two receipts; read after the old cursor 40 it shows nothing.
      if (after === "40") return { code: 0, stdout: readAnswer([], "40", false, NEXT_SEASON), stderr: "" };
      return { code: 0, stdout: readAnswer([message({ event_id: "evt_new", cursor: "2" })], "2", false, NEXT_SEASON), stderr: "" };
    }, lobby, { mission_id: SEASON, cursor: "40" });
    expect(afters).toEqual(["40", "0"]);
    expect(read.items.map((i) => (i.channel as any).event_id)).toEqual(["evt_new"]);
    expect(read.cursor).toEqual({ mission_id: NEXT_SEASON, cursor: "2" });
    expect(read.seasonChanged).toEqual({ from: SEASON, to: NEXT_SEASON, stale_cursor: "40" });
  });

  test("the first read records its season; a season that changes mid-read is an intake error", () => {
    const first = readLobby(() => ({ code: 0, stdout: readAnswer([], "0"), stderr: "" }), lobby, null);
    expect(first.cursor).toEqual({ mission_id: SEASON, cursor: "0" });
    expect(first.seasonChanged).toBeNull();
    let page = 0;
    expect(() => readLobby(() => {
      page += 1;
      return { code: 0, stdout: readAnswer([], String(page), page === 1, page === 1 ? SEASON : NEXT_SEASON), stderr: "" };
    }, lobby, { mission_id: SEASON, cursor: "0" })).toThrow("season changed during the read");
  });

  test("a bare cursor from an older ledger is treated as stale and the read starts over", () => {
    const afters: string[] = [];
    const read = readLobby((args) => {
      afters.push(args[args.indexOf("--after") + 1]!);
      return { code: 0, stdout: readAnswer([], "3"), stderr: "" };
    }, lobby, { mission_id: "", cursor: "9" });
    expect(afters).toEqual(["9", "0"]);
    expect(read.seasonChanged).toEqual({ from: "", to: SEASON, stale_cursor: "9" });
  });

  test("a failed or odd lobby answer is an intake error", () => {
    expect(() => readLobby(() => ({ code: 1, stdout: "", stderr: "grant expired" }), lobby, null)).toThrow(IntakeError);
    expect(() => readLobby(() => ({ code: 0, stdout: "{}", stderr: "" }), lobby, null)).toThrow(IntakeError);
    expect(() => readLobby(() => ({ code: 0, stdout: readAnswer([], "x"), stderr: "" }), lobby, null)).toThrow(IntakeError);
    expect(() => readLobby(() => ({ code: 0, stdout: readAnswer([], "1", false, null), stderr: "" }), lobby, null)).toThrow("mission_id");
  });

  test("replies go under the lease, once per request, in reply to the request event", () => {
    const argv: string[][] = [];
    const r = replyInLobby((args) => { argv.push(args); return { code: 0, stdout: "{}", stderr: "" }; }, lobby, "evt_1", "c".repeat(64), "zerone-door: refused request cccc: pow_insufficient.");
    expect(r.ok).toBe(true);
    expect(argv[0]).toEqual([
      "room", "say", lobby.room_id, "--lease", "lease_1", "--idempotency-key", `zerone-door-${"c".repeat(32)}`,
      "--in-reply-to", "evt_1", "--conversation", "zerone-door", "--text", "zerone-door: refused request cccc: pow_insufficient.", "--json",
    ]);
    expect(replyInLobby(() => ({ code: 0, stdout: "", stderr: "" }), { ...lobby, reply_lease: null }, "e", "c".repeat(64), "t").ok).toBe(false);
  });
});

describe("the lobby's egress gate", () => {
  // agenttool packages/lobby/src/egress.ts (68044b45, installed on the hub
  // 2026-10-28) refuses a run of 40+ base64 characters that mixes upper case,
  // lower case and digits, and lets hex runs through so digests stay citable.
  // A door request carries only public values, all as lowercase hex, so it
  // passes. This restates that one rule; the full gate was checked by hand.
  function keyLikeBase64(text: string): boolean {
    for (const m of text.matchAll(/[A-Za-z0-9+/_-]{40,}={0,2}/g)) {
      const run = m[0].replace(/=+$/, "");
      if (/^[0-9a-f]+$/i.test(run)) continue;
      if (/[A-Z]/.test(run) && /[a-z]/.test(run) && /[0-9]/.test(run)) return true;
    }
    return false;
  }

  test("a door request has no key-like base64 run and fits one table message", () => {
    const text = makeRequest().text;
    expect(keyLikeBase64(text)).toBe(false);
    expect(keyLikeBase64(Buffer.from(text).toString("base64"))).toBe(true);
    expect(text.length).toBeLessThan(900);
  });
});

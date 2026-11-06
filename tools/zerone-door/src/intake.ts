// Where requests come from. Two channels, both inside the household:
//
//  hub_local     A being on the household hub writes a .request file into the
//                door's inbox. The inbox is a 0700 directory owned by the hub
//                user that runs the door, so only processes running as that
//                user can write it.
//  lobby         A desk on another of Yu's Macs posts the request as table talk
//                in the door's fleet room. The hub's own desk reads it with
//                `agenttool-lobby room read --json`, which verifies the sender
//                against the room's door-signed roster. Posting needs no ZRN,
//                and the lobby client has no payment step; whether the
//                agenttool service meters mission messages was not checked.
import { lstatSync, mkdirSync, readdirSync, readFileSync, renameSync } from "node:fs";
import { join } from "node:path";
import type { LobbyConfig } from "./config.ts";
import type { Channel } from "./gates.ts";
import type { LobbyCursor } from "./ledger.ts";
import { REQUEST_MAX_BYTES } from "./policy.ts";
import { sha256Hex } from "./request.ts";

export interface IntakeItem {
  text: string;
  channel: Channel;
}

export class IntakeError extends Error {}

/** The inbox must be private to the door's user on the hub: that is its membership proof. */
export function assertPrivateDir(path: string): void {
  const st = lstatSync(path);
  if (!st.isDirectory()) throw new IntakeError(`${path} must be a directory, not a link`);
  if (typeof process.getuid === "function" && st.uid !== process.getuid()) throw new IntakeError(`${path} must be owned by the door's user`);
  if (st.mode & 0o077) throw new IntakeError(`${path} must be mode 0700`);
}

export function ensurePrivateDir(path: string): void {
  mkdirSync(path, { recursive: true, mode: 0o700 });
  assertPrivateDir(path);
}

/** Takes every *.request file from the inbox, moving each to processed/ first
 * so a crash never processes one twice. Links, other owners and oversize
 * files are moved aside unread. */
export function takeSpool(inbox: string, processed: string, now: Date): IntakeItem[] {
  assertPrivateDir(inbox);
  ensurePrivateDir(processed);
  const items: IntakeItem[] = [];
  const names = readdirSync(inbox).filter((n) => n.endsWith(".request")).sort();
  for (const name of names) {
    const from = join(inbox, name);
    const stamp = now.toISOString().replace(/[:.]/g, "");
    const to = join(processed, `${stamp}-${name}`);
    let st;
    try {
      st = lstatSync(from);
    } catch {
      continue;
    }
    renameSync(from, to);
    const ownerOk = typeof process.getuid !== "function" || st.uid === process.getuid();
    if (!st.isFile() || !ownerOk || st.size > REQUEST_MAX_BYTES) continue;
    items.push({ text: readFileSync(to, "utf8"), channel: { kind: "hub_local", file: name } });
  }
  return items;
}

export type LobbyRunner = (args: string[]) => { code: number; stdout: string; stderr: string };

export function makeLobbyRunner(cli: string): LobbyRunner {
  return (args) => {
    const proc = Bun.spawnSync({
      cmd: [cli, ...args],
      stdin: "ignore",
      stdout: "pipe",
      stderr: "pipe",
      env: { HOME: process.env.HOME ?? "/nonexistent", PATH: "/usr/bin:/bin:/usr/sbin:/sbin" },
    });
    return { code: proc.exitCode ?? -1, stdout: proc.stdout.toString(), stderr: proc.stderr.toString() };
  };
}

export interface LobbyRead {
  items: IntakeItem[];
  /** Where the next read starts: the season read and the cursor within it. */
  cursor: LobbyCursor;
  unverified: number;
  /** Set when the room had rolled to a new season since the stored cursor:
   * the read started over at the new season's beginning. */
  seasonChanged: { from: string; to: string; stale_cursor: string } | null;
}

const CURSOR = /^(0|[1-9][0-9]{0,18})$/;
const MISSION_ID = /^[A-Za-z0-9][A-Za-z0-9_-]{0,79}$/;
const MAX_PAGES = 10;

/** Reads the door's conversation after the cursor, at most ten pages. Only
 * messages the lobby itself labels verified against the door-signed roster
 * (basis door_roster_pin) become requests; the allowlist decides later.
 *
 * `room read` reads the room's current season, and agenttool numbers receipts
 * per season from 1, so a cursor from an earlier season would skip the new
 * season's first messages without a word. Every page names its season; when
 * it is not the stored cursor's season, the read starts over from 0. */
export function readLobby(run: LobbyRunner, lobby: LobbyConfig, stored: LobbyCursor | null): LobbyRead {
  const items: IntakeItem[] = [];
  let unverified = 0;
  let mission: string | null = stored?.mission_id ?? null;
  let after = stored?.cursor ?? "0";
  let seasonChanged: LobbyRead["seasonChanged"] = null;
  for (let page = 0; page < MAX_PAGES; page += 1) {
    const r = run(["room", "read", lobby.room_id, "--conversation", lobby.conversation, "--after", after, "--limit", "50", "--json"]);
    if (r.code !== 0) throw new IntakeError(`lobby read failed: ${r.stderr.trim().slice(0, 200)}`);
    let body: any;
    try {
      body = JSON.parse(r.stdout);
    } catch {
      throw new IntakeError("lobby read did not answer JSON");
    }
    if (body?.room_id !== lobby.room_id || !Array.isArray(body?.messages) || typeof body?.next_cursor !== "string" || !CURSOR.test(body.next_cursor)) {
      throw new IntakeError("lobby read answered an unexpected shape");
    }
    if (typeof body.mission_id !== "string" || !MISSION_ID.test(body.mission_id)) {
      throw new IntakeError("lobby read did not name its season (mission_id)");
    }
    if (body.mission_id !== mission) {
      if (page > 0 || seasonChanged !== null) throw new IntakeError("the lobby season changed during the read; the next run starts over");
      if (stored !== null) {
        // A new season: this page was read with a cursor from the old one.
        // Drop it and read the new season from its beginning.
        seasonChanged = { from: stored.mission_id, to: body.mission_id, stale_cursor: after };
        mission = body.mission_id;
        after = "0";
        page = -1;
        continue;
      }
      mission = body.mission_id;
    }
    unverified += Array.isArray(body.unverified) ? body.unverified.length : 0;
    for (const m of body.messages) {
      if (m?.kind !== "message" || m?.from_me === true || m?.conversation !== lobby.conversation) continue;
      if (typeof m.text !== "string" || typeof m.event_id !== "string") continue;
      const from = m.from ?? {};
      items.push({
        text: m.text,
        channel: {
          kind: "lobby",
          room_id: lobby.room_id,
          identity_id: String(from.identity_id ?? ""),
          fingerprint: String(from.fingerprint ?? ""),
          event_id: m.event_id,
          verified: from.label === "verified" && from.basis === "door_roster_pin",
          currentMember: from.current_member === true,
        },
      });
    }
    after = body.next_cursor;
    if (body.has_more !== true) break;
  }
  return { items, cursor: { mission_id: mission!, cursor: after }, unverified, seasonChanged };
}

/** Answers one lobby request under the reply lease. Short text, no paths, no
 * base64: it passes the lobby's egress gate. Never retried. */
export function replyInLobby(run: LobbyRunner, lobby: LobbyConfig, eventId: string, requestId: string, text: string): { ok: boolean; detail: string } {
  if (lobby.reply_lease === null) return { ok: false, detail: "no reply lease" };
  const r = run([
    "room", "say", lobby.room_id,
    "--lease", lobby.reply_lease,
    "--idempotency-key", `zerone-door-${requestId.slice(0, 32)}`,
    "--in-reply-to", eventId,
    "--conversation", lobby.conversation,
    "--text", text,
    "--json",
  ]);
  return { ok: r.code === 0, detail: r.code === 0 ? "sent" : r.stderr.trim().slice(0, 200) };
}

export function itemKey(item: IntakeItem): string {
  return sha256Hex(item.text);
}

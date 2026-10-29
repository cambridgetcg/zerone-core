// The door's own record: every request it has seen and every transaction it
// signed. It lives beside config.json (state/ledger.json, 0600), is written
// read-merge-write under an exclusive lock, and is the basis of the
// one-per-address, one-per-identity and daily/window caps. Dry runs keep a
// separate file so they never consume a live cap.
import {
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
import { dirname, join } from "node:path";

export type ChannelRef =
  | { kind: "studio_local"; file: string }
  | { kind: "lobby"; room_id: string; identity_id: string; fingerprint: string; event_id: string };

export type EntryStatus =
  | "pending_inclusion"
  | "included"
  | "rejected"
  | "delivery_failed"
  | "postcheck_failed"
  /** Never seen in a block, and R's sequence never moved past it: it can no
   * longer land. Not counted, so the agent may ask again. */
  | "dropped";

export interface LedgerEntry {
  request_id: string;
  kind: "admit" | "regrant";
  address: string;
  address_payload: string;
  identity_public_key: string;
  did: string;
  channel_key: string;
  tx_hash: string;
  status: EntryStatus;
  /** The chain admission window the admission counts in: the window of the
   * door's read height while the transaction is pending, then the window of
   * the block that included it, because the chain counts an admission in the
   * window where it runs. The door's window cap counts by this field. */
  chain_window: number;
  /** The UTC day, on the door's clock, the door signed it. Unlike the window
   * this stays at signing time: the daily caps limit how many transactions the
   * door signs per day of the same clock the gates read "today" from, and the
   * chain has no days. With one transaction in flight at a time, at most one
   * signed just before midnight runs on the next day. */
  utc_day: string;
  at: string;
  grant_expiration: string;
  /** R's account sequence the transaction was signed with. */
  sequence?: string;
  /** The inclusion height, once the transaction is found in a block. */
  height?: number;
  detail?: string;
}

/** Where the door stopped reading the lobby room. agenttool numbers receipts
 * per season (mission), starting again at 1, so a cursor means something only
 * together with the season it was read in. */
export interface LobbyCursor {
  mission_id: string;
  cursor: string;
}

export interface Ledger {
  schema: "zerone-door-ledger/v1";
  entries: LedgerEntry[];
  /** request_id -> first time seen. A request is processed once, ever. */
  seen: Record<string, string>;
  lobby_cursor: LobbyCursor | null;
}

/** Entries that changed (or may have changed) chain state. */
export const COUNTED: ReadonlySet<EntryStatus> = new Set(["pending_inclusion", "included", "postcheck_failed"]);

export function emptyLedger(): Ledger {
  return { schema: "zerone-door-ledger/v1", entries: [], seen: {}, lobby_cursor: null };
}

export function ledgerPathFor(stateDir: string, live: boolean): string {
  return join(stateDir, live ? "ledger.json" : "dry-run-ledger.json");
}

export function loadLedger(path: string): Ledger {
  if (!existsSync(path)) return emptyLedger();
  const parsed = JSON.parse(readFileSync(path, "utf8")) as Ledger;
  if (parsed.schema !== "zerone-door-ledger/v1" || !Array.isArray(parsed.entries) || typeof parsed.seen !== "object") {
    throw new Error(`${path} is not a zerone-door ledger`);
  }
  // A bare cursor from before it carried its season: keep it, with no season,
  // so the next lobby read treats it as stale, starts over and logs that.
  const cursor = parsed.lobby_cursor as unknown;
  if (typeof cursor === "string") parsed.lobby_cursor = { mission_id: "", cursor };
  else if (cursor === undefined) parsed.lobby_cursor = null;
  return parsed;
}

export function saveLedger(path: string, ledger: Ledger): void {
  mkdirSync(dirname(path), { recursive: true, mode: 0o700 });
  const tmp = `${path}.tmp-${process.pid}`;
  writeFileSync(tmp, `${JSON.stringify(ledger, null, 2)}\n`, { mode: 0o600 });
  renameSync(tmp, path);
}

/** One door process at a time. A lock older than an hour is taken over. */
export function withLock<T>(lockPath: string, fn: () => Promise<T>): Promise<T> {
  mkdirSync(dirname(lockPath), { recursive: true, mode: 0o700 });
  let fd: number;
  try {
    fd = openSync(lockPath, "wx", 0o600);
  } catch {
    let stale = false;
    try {
      stale = Date.now() - statSync(lockPath).mtimeMs > 60 * 60 * 1000;
    } catch {
      stale = true;
    }
    if (!stale) return Promise.reject(new Error(`another zerone-door run holds ${lockPath}`));
    try { unlinkSync(lockPath); } catch { /* raced */ }
    fd = openSync(lockPath, "wx", 0o600);
  }
  writeSync(fd, `${process.pid}\n`);
  closeSync(fd);
  return fn().finally(() => {
    try { unlinkSync(lockPath); } catch { /* taken over */ }
  });
}

export function utcDay(date: Date): string {
  return date.toISOString().slice(0, 10);
}

export function counted(ledger: Ledger): LedgerEntry[] {
  return ledger.entries.filter((e) => COUNTED.has(e.status));
}

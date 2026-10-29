// Local configuration. Two files live in the door's home directory and are
// never committed: config.json (where R's file keyring and its Keychain item
// are, the pinned CLI, the endpoints, household keys) and allowlist.json (who
// may ask). The repository carries only examples with placeholders, because a
// file that maps where keys live must not be published.
import { lstatSync, readFileSync } from "node:fs";
import { dirname, isAbsolute, join } from "node:path";
import { parseZeroneAddress } from "./bech32.ts";
import { CHAIN_ID } from "./policy.ts";

export interface RegistrarConfig {
  /** R's zrn1 address: the on-chain bootstrap_registrar and the granter. */
  address: string;
  /** Key name inside R's file keyring. */
  key_name: string;
  /** --keyring-dir of R's file keyring (absolute). */
  keyring_dir: string;
  /** Login-Keychain generic-password service holding the keyring passphrase. */
  keychain_service: string;
}

export interface LobbyConfig {
  /** Absolute path of the agenttool-lobby CLI on the hub. */
  cli: string;
  /** The door's fleet room. */
  room_id: string;
  /** Conversation slug inside the room; requests outside it are not read. */
  conversation: string;
  /** Optional room seat lease a person made at the hub terminal. When it is
   * set and the door runs live, the door answers each well-formed request in
   * the room under it. A dry run never posts in the room. */
  reply_lease: string | null;
}

/** The chain version the door's gas figures, REST shapes and pinned CLI were
 * measured on. Any difference refuses every request (chain_version_changed)
 * until a person re-measures and re-pins; `door.ts status` prints both. */
export interface ChainVersionPin {
  /** Name of the last plan x/upgrade applied on the chain. */
  last_upgrade: string;
  /** sha256 of the chain's module_versions (chain.ts moduleVersionsDigest). */
  module_versions_sha256: string;
}

export interface KnownKey {
  address: string;
  label: string;
}

export interface DoorConfig {
  schema: "zerone-door-config/v1";
  chain_id: string;
  rpc: string;
  rest: string;
  zeroned: { path: string; sha256: string };
  chain_version: ChainVersionPin;
  registrar: RegistrarConfig;
  known_keys: KnownKey[];
  lobby: LobbyConfig | null;
  /** Directory holding config.json; every other door path derives from it. */
  home: string;
}

export interface AllowlistDesk {
  identity_id: string;
  daily_cap?: number;
  note?: string;
}

export interface Allowlist {
  schema: "zerone-door-allowlist/v1";
  studio_local: { enabled: boolean; daily_cap?: number };
  lobby_desks: AllowlistDesk[];
}

export class ConfigError extends Error {}

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;
const HEX64 = /^[0-9a-f]{64}$/;
const KEY_NAME = /^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$/;
const ROOM_ID = /^room_[0-9a-f]{32}$/;
const CONVERSATION = /^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$/;
const LEASE_ID = /^[A-Za-z0-9_-]{1,80}$/;
const UPGRADE_NAME = /^[\x21-\x7e]{1,128}$/;

export function expandHome(path: string, env: Record<string, string | undefined> = process.env): string {
  const home = env.HOME;
  if (!home) return path;
  if (path === "~") return home;
  if (path.startsWith("~/")) return `${home}${path.slice(1)}`;
  return path;
}

export function defaultConfigPath(env: Record<string, string | undefined> = process.env): string {
  return expandHome("~/.zerone-agent/door/config.json", env);
}

export interface DoorPaths {
  home: string;
  config: string;
  allowlist: string;
  inbox: string;
  processed: string;
  replies: string;
  state: string;
  ledger: string;
  lock: string;
  work: string;
  /** Any of these existing (any type) stops every admission. */
  haltFiles: string[];
}

export function doorPaths(configPath: string, env: Record<string, string | undefined> = process.env): DoorPaths {
  const home = dirname(configPath);
  return {
    home,
    config: configPath,
    allowlist: join(home, "allowlist.json"),
    inbox: join(home, "inbox"),
    processed: join(home, "processed"),
    replies: join(home, "replies"),
    state: join(home, "state"),
    ledger: join(home, "state", "ledger.json"),
    lock: join(home, "state", "door.lock"),
    work: join(home, "state", "work"),
    haltFiles: [join(home, "HALT"), expandHome("~/.local/state/zerone/rung-window.freeze", env)],
  };
}

/** A local file the door trusts must be a regular file owned by this user and
 * not writable by group or others. config.json must also be unreadable by them. */
export function assertPrivateFile(path: string, opts: { privateRead: boolean }): void {
  let st;
  try {
    st = lstatSync(path);
  } catch {
    throw new ConfigError(`${path} is missing`);
  }
  if (!st.isFile()) throw new ConfigError(`${path} must be a regular file, not a link or directory`);
  if (typeof process.getuid === "function" && st.uid !== process.getuid()) {
    throw new ConfigError(`${path} must be owned by the user running the door`);
  }
  const mode = st.mode & 0o777;
  if (mode & 0o022) throw new ConfigError(`${path} must not be writable by group or others`);
  if (opts.privateRead && mode & 0o044) throw new ConfigError(`${path} must be mode 0600`);
}

function str(value: unknown, field: string): string {
  if (typeof value !== "string" || value.length === 0) throw new ConfigError(`${field} must be a non-empty string`);
  return value;
}

function httpUrl(value: unknown, field: string): string {
  const s = str(value, field);
  if (!/^https?:\/\/[^\s/]+(?::\d+)?$/.test(s)) throw new ConfigError(`${field} must be an http(s) origin without a path`);
  return s;
}

function absolute(value: unknown, field: string): string {
  const s = str(value, field);
  if (!isAbsolute(s)) throw new ConfigError(`${field} must be an absolute path`);
  return s;
}

function capOf(value: unknown, field: string): number | undefined {
  if (value === undefined) return undefined;
  if (!Number.isInteger(value) || (value as number) < 0 || (value as number) > 100) {
    throw new ConfigError(`${field} must be an integer 0-100`);
  }
  return value as number;
}

export function parseConfig(raw: unknown, configPath: string): DoorConfig {
  if (raw === null || typeof raw !== "object" || Array.isArray(raw)) throw new ConfigError("config must be an object");
  const c = raw as Record<string, unknown>;
  if (c.schema !== "zerone-door-config/v1") throw new ConfigError("config schema must be zerone-door-config/v1");
  if (c.chain_id !== CHAIN_ID) throw new ConfigError(`chain_id must be ${CHAIN_ID}`);
  const z = (c.zeroned ?? {}) as Record<string, unknown>;
  const zeroned = { path: absolute(z.path, "zeroned.path"), sha256: str(z.sha256, "zeroned.sha256") };
  if (!HEX64.test(zeroned.sha256)) throw new ConfigError("zeroned.sha256 must be 64 lowercase hex");
  const v = (c.chain_version ?? {}) as Record<string, unknown>;
  const chain_version: ChainVersionPin = {
    last_upgrade: str(v.last_upgrade, "chain_version.last_upgrade"),
    module_versions_sha256: str(v.module_versions_sha256, "chain_version.module_versions_sha256"),
  };
  if (!UPGRADE_NAME.test(chain_version.last_upgrade)) throw new ConfigError("chain_version.last_upgrade has the wrong shape");
  if (!HEX64.test(chain_version.module_versions_sha256)) throw new ConfigError("chain_version.module_versions_sha256 must be 64 lowercase hex");
  const r = (c.registrar ?? {}) as Record<string, unknown>;
  const registrar: RegistrarConfig = {
    address: str(r.address, "registrar.address"),
    key_name: str(r.key_name, "registrar.key_name"),
    keyring_dir: absolute(r.keyring_dir, "registrar.keyring_dir"),
    keychain_service: str(r.keychain_service, "registrar.keychain_service"),
  };
  if (!parseZeroneAddress(registrar.address).ok) throw new ConfigError("registrar.address must be a canonical zrn1 address");
  if (!KEY_NAME.test(registrar.key_name)) throw new ConfigError("registrar.key_name has the wrong shape");
  if (!KEY_NAME.test(registrar.keychain_service)) throw new ConfigError("registrar.keychain_service has the wrong shape");
  if (!Array.isArray(c.known_keys)) throw new ConfigError("known_keys must be an array");
  const known_keys = c.known_keys.map((k, i) => {
    const entry = (k ?? {}) as Record<string, unknown>;
    const address = str(entry.address, `known_keys[${i}].address`);
    if (!parseZeroneAddress(address).ok) throw new ConfigError(`known_keys[${i}].address must be a canonical zrn1 address`);
    return { address, label: str(entry.label, `known_keys[${i}].label`) };
  });
  let lobby: LobbyConfig | null = null;
  if (c.lobby !== null && c.lobby !== undefined) {
    const l = c.lobby as Record<string, unknown>;
    lobby = {
      cli: absolute(l.cli, "lobby.cli"),
      room_id: str(l.room_id, "lobby.room_id"),
      conversation: str(l.conversation, "lobby.conversation"),
      reply_lease: l.reply_lease === null || l.reply_lease === undefined ? null : str(l.reply_lease, "lobby.reply_lease"),
    };
    if (!ROOM_ID.test(lobby.room_id)) throw new ConfigError("lobby.room_id must be room_ + 32 hex");
    if (!CONVERSATION.test(lobby.conversation)) throw new ConfigError("lobby.conversation has the wrong shape");
    if (lobby.reply_lease !== null && !LEASE_ID.test(lobby.reply_lease)) throw new ConfigError("lobby.reply_lease has the wrong shape");
  }
  return {
    schema: "zerone-door-config/v1",
    chain_id: CHAIN_ID,
    rpc: httpUrl(c.rpc, "rpc"),
    rest: httpUrl(c.rest, "rest"),
    zeroned,
    chain_version,
    registrar,
    known_keys,
    lobby,
    home: dirname(configPath),
  };
}

export function parseAllowlist(raw: unknown): Allowlist {
  if (raw === null || typeof raw !== "object" || Array.isArray(raw)) throw new ConfigError("allowlist must be an object");
  const a = raw as Record<string, unknown>;
  if (a.schema !== "zerone-door-allowlist/v1") throw new ConfigError("allowlist schema must be zerone-door-allowlist/v1");
  const s = (a.studio_local ?? {}) as Record<string, unknown>;
  if (typeof s.enabled !== "boolean") throw new ConfigError("studio_local.enabled must be true or false");
  const studioCap = capOf(s.daily_cap, "studio_local.daily_cap");
  if (!Array.isArray(a.lobby_desks)) throw new ConfigError("lobby_desks must be an array");
  const seen = new Set<string>();
  const lobby_desks = a.lobby_desks.map((d, i) => {
    const desk = (d ?? {}) as Record<string, unknown>;
    const identity_id = str(desk.identity_id, `lobby_desks[${i}].identity_id`);
    if (!UUID.test(identity_id)) throw new ConfigError(`lobby_desks[${i}].identity_id must be a lowercase UUID`);
    if (seen.has(identity_id)) throw new ConfigError(`lobby_desks[${i}] repeats an identity`);
    seen.add(identity_id);
    const cap = capOf(desk.daily_cap, `lobby_desks[${i}].daily_cap`);
    return {
      identity_id,
      ...(cap === undefined ? {} : { daily_cap: cap }),
      ...(typeof desk.note === "string" ? { note: desk.note } : {}),
    };
  });
  return {
    schema: "zerone-door-allowlist/v1",
    studio_local: { enabled: s.enabled, ...(studioCap === undefined ? {} : { daily_cap: studioCap }) },
    lobby_desks,
  };
}

export function loadConfig(configPath: string): DoorConfig {
  assertPrivateFile(configPath, { privateRead: true });
  return parseConfig(JSON.parse(readFileSync(configPath, "utf8")), configPath);
}

/** Read fresh for every request, so a household edit takes effect at once. */
export function loadAllowlist(path: string): Allowlist {
  assertPrivateFile(path, { privateRead: false });
  return parseAllowlist(JSON.parse(readFileSync(path, "utf8")));
}

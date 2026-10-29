// Read-only chain access over the public REST and CometBFT RPC endpoints,
// through an injected fetch. Only an exact, known not-found answer counts as
// "absent"; every other failure is an error and the door refuses (fails closed).
import { createHash } from "node:crypto";
import { MSG_ADD_BOOTSTRAP_ENTRY, MSG_GRANT_ALLOWANCE, MSG_REVOKE_ALLOWANCE, READ_TIMEOUT_MS } from "./policy.ts";

export type Read<T> = { ok: true; value: T } | { ok: false; error: string };
export type FetchLike = (url: string, init?: { signal?: AbortSignal }) => Promise<{ status: number; text(): Promise<string> }>;

export interface Coin {
  denom: string;
  amount: string;
}

export interface NodeStatus {
  network: string;
  height: number;
  time: string;
}

export interface PotParams {
  registrar: string;
  windowCap: number;
}

export interface WindowState {
  height: number;
  storedWindow: number;
  storedCount: number;
}

export interface AuthAccount {
  accountNumber: string;
  sequence: string;
  pubKeyType: string | null;
  pubKeyB64: string | null;
}

/** An account as one node's state answers it, with the height of that state. */
export interface AccountAtHeight {
  height: number;
  /** Null when the account does not exist at that height. */
  account: { accountNumber: string; sequence: string } | null;
}

export interface ZeroneAccount {
  address: string;
  did: string;
  accountType: string;
  frozen: boolean;
}

export interface Pot {
  id: string;
  status: string;
  total: bigint;
  claimed: bigint;
  whitelist: string[];
}

export interface AllowanceSummary {
  /** The @type chain from the outside in. */
  types: string[];
  allowedMessages: string[] | null;
  /** Remaining lifetime spend in uzrn; null means unbounded. */
  spendLimitUzrn: bigint | null;
  expiration: string | null;
  periodSeconds: number | null;
  periodLimitUzrn: bigint | null;
}

export interface Grant {
  granter: string;
  grantee: string;
  allowance: AllowanceSummary;
}

export interface IncludedTx {
  height: number;
  code: number;
  log: string;
  gasUsed: number;
  gasWanted: number;
}

/** One message of a transaction in a block: its type and, for the messages
 * the door signs, the parties that make it this admission or another. */
export interface TxMessage {
  type: string;
  /** MsgAddBootstrapEntry */
  authority?: string;
  addresses?: string[];
  /** MsgGrantAllowance and MsgRevokeAllowance */
  granter?: string;
  grantee?: string;
}

/** Which chain version the node runs, as the door pins it. */
export interface ChainVersion {
  /** The last plan x/upgrade applied (highest height), or null if none ever was. */
  lastUpgrade: { name: string; height: number } | null;
  /** sha256 hex of the module_versions list (see moduleVersionsDigest). */
  moduleVersionsSha256: string;
}

const MAX_PAGES = 50;

/** sha256 hex over "name=version\n" lines sorted by name: one value that
 * changes whenever any module's consensus version changes. */
export function moduleVersionsDigest(modules: ReadonlyArray<{ name: string; version: string }>): string {
  const lines = [...modules].sort((a, b) => (a.name < b.name ? -1 : a.name > b.name ? 1 : 0))
    .map((m) => `${m.name}=${m.version}\n`).join("");
  return createHash("sha256").update(lines).digest("hex");
}

/** Decodes x/upgrade's done markers, as an ABCI subspace query of prefix 0x01
 * returns them: Pairs { repeated Pair pairs = 1 }, Pair { key = 1; value = 2 },
 * each key 0x01 || height (u64 big-endian) || plan name. Returns the plan
 * applied at the greatest height, or null when there is none. */
export function lastUpgradeFromDoneMarkers(bytes: Uint8Array): { name: string; height: number } | null {
  let last: { name: string; height: number } | null = null;
  for (const [field, pair] of protoFields(bytes)) {
    if (field !== 1 || !(pair instanceof Uint8Array)) throw new Error("upgrade markers: unexpected field");
    let key: Uint8Array | null = null;
    for (const [f, v] of protoFields(pair)) if (f === 1 && v instanceof Uint8Array) key = v;
    if (key === null || key.length < 10 || key[0] !== 0x01) throw new Error("upgrade markers: unexpected key");
    const height = Number(Buffer.from(key.subarray(1, 9)).readBigUInt64BE(0));
    const name = new TextDecoder("utf-8", { fatal: true }).decode(key.subarray(9));
    if (!Number.isSafeInteger(height) || !/^[\x21-\x7e]{1,128}$/.test(name)) throw new Error("upgrade markers: unexpected entry");
    if (last === null || height > last.height) last = { name, height };
  }
  return last;
}

function uzrnOf(coins: unknown): bigint | null {
  if (!Array.isArray(coins) || coins.length === 0) return null;
  let total = 0n;
  for (const coin of coins as Coin[]) {
    if (coin?.denom === "uzrn" && /^\d+$/.test(String(coin.amount))) total += BigInt(coin.amount);
  }
  return total;
}

function seconds(duration: unknown): number | null {
  if (typeof duration !== "string") return null;
  const m = /^(\d+)(?:\.\d+)?s$/.exec(duration);
  return m ? Number(m[1]) : null;
}

/** Normalizes Basic, Periodic and AllowedMsg allowances (any nesting). */
export function summarizeAllowance(raw: unknown): AllowanceSummary {
  const types: string[] = [];
  let allowedMessages: string[] | null = null;
  let node = raw as Record<string, unknown> | undefined;
  let spendLimitUzrn: bigint | null = null;
  let expiration: string | null = null;
  let periodSeconds: number | null = null;
  let periodLimitUzrn: bigint | null = null;
  for (let depth = 0; node && depth < 4; depth += 1) {
    const type = String(node["@type"] ?? "");
    types.push(type);
    if (type.endsWith(".AllowedMsgAllowance")) {
      allowedMessages = Array.isArray(node.allowed_messages) ? (node.allowed_messages as unknown[]).map(String) : [];
      node = node.allowance as Record<string, unknown> | undefined;
      continue;
    }
    if (type.endsWith(".PeriodicAllowance")) {
      periodSeconds = seconds(node.period);
      periodLimitUzrn = uzrnOf(node.period_spend_limit);
      node = { "@type": "/cosmos.feegrant.v1beta1.BasicAllowance", ...(node.basic as Record<string, unknown>) };
      continue;
    }
    if (type.endsWith(".BasicAllowance")) {
      spendLimitUzrn = uzrnOf(node.spend_limit);
      expiration = typeof node.expiration === "string" ? node.expiration : null;
    }
    break;
  }
  return { types, allowedMessages, spendLimitUzrn, expiration, periodSeconds, periodLimitUzrn };
}

function decodeVarint(bytes: Uint8Array, start: number): [number, number] {
  let value = 0;
  let shift = 0;
  let i = start;
  for (;;) {
    if (i >= bytes.length) throw new Error("truncated varint");
    const b = bytes[i]!;
    i += 1;
    value += (b & 0x7f) * 2 ** shift;
    if (b < 0x80) return [value, i];
    shift += 7;
    if (shift > 63) throw new Error("varint too long");
  }
}

/** Top-level protobuf fields: [field, wire value]. Varints and bytes only. */
export function protoFields(bytes: Uint8Array): Array<[number, number | Uint8Array]> {
  const out: Array<[number, number | Uint8Array]> = [];
  let i = 0;
  while (i < bytes.length) {
    const [key, next] = decodeVarint(bytes, i);
    i = next;
    const field = Math.floor(key / 8);
    const wire = key % 8;
    if (wire === 0) {
      const [v, n] = decodeVarint(bytes, i);
      out.push([field, v]);
      i = n;
    } else if (wire === 2) {
      const [len, n] = decodeVarint(bytes, i);
      if (n + len > bytes.length) throw new Error("truncated field");
      out.push([field, bytes.subarray(n, n + len)]);
      i = n + len;
    } else {
      throw new Error(`unsupported wire type ${wire}`);
    }
  }
  return out;
}

function encodeVarint(n: number): number[] {
  const out: number[] = [];
  let v = n;
  while (v >= 0x80) {
    out.push((v & 0x7f) | 0x80);
    v = Math.floor(v / 128);
  }
  out.push(v);
  return out;
}

const utf8 = new TextDecoder("utf-8", { fatal: true });

/** google.protobuf.Any { type_url = 1; value = 2 }. */
function anyOf(bytes: Uint8Array): { typeUrl: string; value: Uint8Array } {
  let typeUrl = "";
  let value: Uint8Array = new Uint8Array();
  for (const [f, v] of protoFields(bytes)) {
    if (f === 1 && v instanceof Uint8Array) typeUrl = utf8.decode(v);
    if (f === 2 && v instanceof Uint8Array) value = v;
  }
  return { typeUrl, value };
}

/** The messages of a transaction as a block holds it: TxRaw { body_bytes = 1 }
 * -> TxBody { repeated Any messages = 1 }. The messages the door signs are
 * read down to their parties (MsgAddBootstrapEntry { authority = 1; repeated
 * addresses = 2 }; MsgGrantAllowance and MsgRevokeAllowance { granter = 1;
 * grantee = 2 }); any other message is read as its type only. */
export function txMessagesOf(raw: Uint8Array): TxMessage[] {
  let body: Uint8Array | null = null;
  for (const [f, v] of protoFields(raw)) if (f === 1 && v instanceof Uint8Array) body = v;
  if (body === null) throw new Error("no body");
  const messages: TxMessage[] = [];
  for (const [f, v] of protoFields(body)) {
    if (f !== 1) continue;
    if (!(v instanceof Uint8Array)) throw new Error("a message that is not an Any");
    const { typeUrl, value } = anyOf(v);
    const m: TxMessage = { type: typeUrl };
    if (typeUrl === MSG_ADD_BOOTSTRAP_ENTRY) {
      const addresses: string[] = [];
      for (const [mf, mv] of protoFields(value)) {
        if (mf === 1 && mv instanceof Uint8Array) m.authority = utf8.decode(mv);
        if (mf === 2 && mv instanceof Uint8Array) addresses.push(utf8.decode(mv));
      }
      m.addresses = addresses;
    } else if (typeUrl === MSG_GRANT_ALLOWANCE || typeUrl === MSG_REVOKE_ALLOWANCE) {
      for (const [mf, mv] of protoFields(value)) {
        if (mf === 1 && mv instanceof Uint8Array) m.granter = utf8.decode(mv);
        if (mf === 2 && mv instanceof Uint8Array) m.grantee = utf8.decode(mv);
      }
    }
    messages.push(m);
  }
  return messages;
}

/** Decodes the value of the auth module's gRPC Account query:
 * QueryAccountResponse { Any account = 1 } holding a BaseAccount { address = 1;
 * pub_key = 2; account_number = 3; sequence = 4 }. proto3 leaves zero values
 * out, so an absent number is 0. Only a BaseAccount for `address` is accepted. */
export function baseAccountOf(bytes: Uint8Array, address: string): { accountNumber: string; sequence: string } {
  let any: Uint8Array | null = null;
  for (const [f, v] of protoFields(bytes)) if (f === 1 && v instanceof Uint8Array) any = v;
  if (any === null) throw new Error("no account in the answer");
  const { typeUrl, value } = anyOf(any);
  if (typeUrl !== "/cosmos.auth.v1beta1.BaseAccount") throw new Error(`account type ${typeUrl.slice(0, 80) || "missing"}`);
  let owner = "";
  let accountNumber = 0;
  let sequence = 0;
  for (const [f, v] of protoFields(value)) {
    if (f === 1 && v instanceof Uint8Array) owner = utf8.decode(v);
    if (f === 3 && typeof v === "number") accountNumber = v;
    if (f === 4 && typeof v === "number") sequence = v;
  }
  if (owner !== address) throw new Error("the answer is for another account");
  if (!Number.isSafeInteger(accountNumber) || !Number.isSafeInteger(sequence)) throw new Error("account number or sequence out of range");
  return { accountNumber: String(accountNumber), sequence: String(sequence) };
}

export class ChainReader {
  readonly rpc: string;
  readonly rest: string;
  private readonly fetchImpl: FetchLike;
  private readonly retries: number;

  constructor(opts: { rpc: string; rest: string; fetch?: FetchLike; retries?: number }) {
    this.rpc = opts.rpc.replace(/\/$/, "");
    this.rest = opts.rest.replace(/\/$/, "");
    this.fetchImpl = opts.fetch ?? ((url, init) => fetch(url, init));
    this.retries = opts.retries ?? 3;
  }

  private async getJson(url: string): Promise<Read<{ status: number; body: any }>> {
    let lastError = "unreachable";
    for (let attempt = 0; attempt < this.retries; attempt += 1) {
      try {
        const res = await this.fetchImpl(url, { signal: AbortSignal.timeout(READ_TIMEOUT_MS) });
        const text = await res.text();
        if (res.status === 502 || res.status === 503 || res.status === 504) {
          lastError = `HTTP ${res.status}`;
        } else {
          try {
            return { ok: true, value: { status: res.status, body: JSON.parse(text) } };
          } catch {
            lastError = `HTTP ${res.status} with a body that is not JSON`;
          }
        }
      } catch (error) {
        lastError = error instanceof Error ? error.message : String(error);
      }
      if (attempt + 1 < this.retries) await Bun.sleep(1_000 * (attempt + 1));
    }
    return { ok: false, error: `${url.split("?")[0]}: ${lastError}` };
  }

  private async restOk(path: string): Promise<Read<any>> {
    const r = await this.getJson(`${this.rest}${path}`);
    if (!r.ok) return r;
    if (r.value.status !== 200) {
      return { ok: false, error: `${path.split("?")[0]}: HTTP ${r.value.status} ${String(r.value.body?.message ?? "").slice(0, 160)}` };
    }
    return { ok: true, value: r.value.body };
  }

  /** A REST read whose only accepted failure is the exact not-found answer. */
  private async restMaybe(path: string, notFound: (status: number, body: any) => boolean): Promise<Read<any | null>> {
    const r = await this.getJson(`${this.rest}${path}`);
    if (!r.ok) return r;
    if (r.value.status === 200) return { ok: true, value: r.value.body };
    if (notFound(r.value.status, r.value.body)) return { ok: true, value: null };
    return { ok: false, error: `${path.split("?")[0]}: HTTP ${r.value.status} ${String(r.value.body?.message ?? "").slice(0, 160)}` };
  }

  private async restPaged(path: string, key: string): Promise<Read<any[]>> {
    const items: any[] = [];
    let next: string | null = null;
    for (let page = 0; page < MAX_PAGES; page += 1) {
      const sep = path.includes("?") ? "&" : "?";
      const q = `${sep}pagination.limit=100${next ? `&pagination.key=${encodeURIComponent(next)}` : ""}`;
      const r = await this.restOk(`${path}${q}`);
      if (!r.ok) return r;
      const list = r.value?.[key];
      if (!Array.isArray(list)) return { ok: false, error: `${path}: no ${key} array` };
      items.push(...list);
      const pg = r.value?.pagination ?? {};
      next = pg.next_key ?? pg.nextKey ?? null;
      if (!next) return { ok: true, value: items };
    }
    return { ok: false, error: `${path}: more than ${MAX_PAGES} pages` };
  }

  async status(): Promise<Read<NodeStatus>> {
    const r = await this.getJson(`${this.rpc}/status`);
    if (!r.ok) return r;
    const res = r.value.body?.result;
    const height = Number(res?.sync_info?.latest_block_height);
    const network = res?.node_info?.network;
    const time = res?.sync_info?.latest_block_time;
    if (!Number.isSafeInteger(height) || typeof network !== "string" || typeof time !== "string") {
      return { ok: false, error: "status: unexpected shape" };
    }
    return { ok: true, value: { network, height, time } };
  }

  async potParams(): Promise<Read<PotParams>> {
    const r = await this.restOk("/zerone/claiming_pot/v1/params");
    if (!r.ok) return r;
    const p = r.value?.params ?? {};
    const registrar = p.bootstrapRegistrar ?? p.bootstrap_registrar;
    const cap = Number(p.bootstrapDailyAdmissionCap ?? p.bootstrap_daily_admission_cap);
    if (typeof registrar !== "string" || !Number.isSafeInteger(cap)) return { ok: false, error: "claiming_pot params: unexpected shape" };
    return { ok: true, value: { registrar, windowCap: cap } };
  }

  /** claiming_pot store key 0x07: window index (u64 BE) || count (u64 BE). */
  async windowState(): Promise<Read<WindowState>> {
    const r = await this.getJson(`${this.rpc}/abci_query?path=%22/store/claiming_pot/key%22&data=0x07`);
    if (!r.ok) return r;
    const resp = r.value.body?.result?.response;
    const height = Number(resp?.height);
    if (!resp || resp.code !== 0 || !Number.isSafeInteger(height)) return { ok: false, error: "window state: unexpected answer" };
    const value = Buffer.from(String(resp.value ?? ""), "base64");
    if (value.length === 0) return { ok: true, value: { height, storedWindow: -1, storedCount: 0 } };
    if (value.length !== 16) return { ok: false, error: `window state: ${value.length} bytes` };
    return {
      ok: true,
      value: { height, storedWindow: Number(value.readBigUInt64BE(0)), storedCount: Number(value.readBigUInt64BE(8)) },
    };
  }

  async upgradePlanPending(): Promise<Read<boolean>> {
    const r = await this.restOk("/cosmos/upgrade/v1beta1/current_plan");
    if (!r.ok) return r;
    if (!("plan" in (r.value ?? {}))) return { ok: false, error: "current_plan: unexpected shape" };
    return { ok: true, value: r.value.plan !== null };
  }

  /** The version the node runs: the last applied upgrade (from x/upgrade's
   * done markers in its store) and a digest of every module's version. */
  async chainVersion(): Promise<Read<ChainVersion>> {
    const mv = await this.restOk("/cosmos/upgrade/v1beta1/module_versions");
    if (!mv.ok) return mv;
    const list = mv.value?.module_versions;
    if (!Array.isArray(list) || list.length === 0
      || !list.every((m: any) => typeof m?.name === "string" && /^\d+$/.test(String(m?.version)))) {
      return { ok: false, error: "module_versions: unexpected shape" };
    }
    const done = await this.getJson(`${this.rpc}/abci_query?path=%22/store/upgrade/subspace%22&data=0x01`);
    if (!done.ok) return done;
    const resp = done.value.body?.result?.response;
    if (!resp || resp.code !== 0) return { ok: false, error: "upgrade markers: unexpected answer" };
    let lastUpgrade: ChainVersion["lastUpgrade"];
    try {
      lastUpgrade = lastUpgradeFromDoneMarkers(Uint8Array.from(Buffer.from(String(resp.value ?? ""), "base64")));
    } catch (error) {
      return { ok: false, error: error instanceof Error ? error.message : String(error) };
    }
    return {
      ok: true,
      value: {
        lastUpgrade,
        moduleVersionsSha256: moduleVersionsDigest(list.map((m: any) => ({ name: m.name, version: String(m.version) }))),
      },
    };
  }

  async balances(address: string): Promise<Read<Coin[]>> {
    return this.restPaged(`/cosmos/bank/v1beta1/balances/${address}`, "balances");
  }

  async spendableUzrn(address: string): Promise<Read<bigint>> {
    const r = await this.restPaged(`/cosmos/bank/v1beta1/spendable_balances/${address}`, "balances");
    if (!r.ok) return r;
    return { ok: true, value: uzrnOf(r.value) ?? 0n };
  }

  async authAccount(address: string): Promise<Read<AuthAccount | null>> {
    const r = await this.restMaybe(`/cosmos/auth/v1beta1/accounts/${address}`,
      (status, body) => status === 404 && body?.code === 5 && body?.message === `account ${address} not found`);
    if (!r.ok || r.value === null) return r;
    const acct = r.value.account ?? {};
    const base = acct.base_account ?? acct;
    if (typeof base.account_number !== "string" || typeof base.sequence !== "string") {
      return { ok: false, error: "auth account: unexpected shape" };
    }
    return {
      ok: true,
      value: {
        accountNumber: base.account_number,
        sequence: base.sequence,
        pubKeyType: base.pub_key?.["@type"] ?? null,
        pubKeyB64: typeof base.pub_key?.key === "string" ? base.pub_key.key : null,
      },
    };
  }

  /** An account as the RPC node's own state answers it, with the height of
   * that state: abci_query of the auth module's gRPC Account query. A REST
   * read may come from another node (the public endpoints sit behind a proxy)
   * and its height travels only in a gRPC-gateway header; this answer carries
   * the height in its body, so a proof can require one at or past a given
   * block, whichever node answered. */
  async accountAtHeight(address: string): Promise<Read<AccountAtHeight>> {
    if (!/^[a-z0-9]{1,83}$/.test(address)) return { ok: false, error: "account query: bad address" };
    const addr = Buffer.from(address, "utf8");
    const request = Buffer.from([0x0a, ...encodeVarint(addr.length), ...addr]);
    const r = await this.getJson(`${this.rpc}/abci_query?path=%22/cosmos.auth.v1beta1.Query/Account%22&data=0x${request.toString("hex")}`);
    if (!r.ok) return r;
    const resp = r.value.body?.result?.response;
    const height = Number(resp?.height);
    if (!resp || !Number.isSafeInteger(height) || height <= 0) return { ok: false, error: "account query: unexpected answer" };
    if (resp.code === 22 && resp.codespace === "sdk"
      && resp.log === `rpc error: code = NotFound desc = account ${address} not found: key not found`) {
      return { ok: true, value: { height, account: null } };
    }
    if (resp.code) return { ok: false, error: `account query refused (code ${resp.code}): ${String(resp.log ?? "").slice(0, 160)}` };
    try {
      return { ok: true, value: { height, account: baseAccountOf(Uint8Array.from(Buffer.from(String(resp.value ?? ""), "base64")), address) } };
    } catch (error) {
      return { ok: false, error: `account query: ${error instanceof Error ? error.message : String(error)}` };
    }
  }

  private static zeroneAuthNotFound(status: number, body: any): boolean {
    return status === 500 && body?.code === 2 && body?.message === "codespace zerone_auth code 2: account not found";
  }

  private static zeroneAccountOf(body: any): Read<ZeroneAccount> {
    const a = body?.account;
    if (!a || typeof a.address !== "string" || typeof a.did !== "string") return { ok: false, error: "zerone_auth: unexpected shape" };
    return {
      ok: true,
      value: { address: a.address, did: a.did, accountType: String(a.accountType ?? a.account_type ?? ""), frozen: a.flags?.frozen === true },
    };
  }

  async zeroneAccount(address: string): Promise<Read<ZeroneAccount | null>> {
    const r = await this.restMaybe(`/zerone/auth/v1/account/${address}`, ChainReader.zeroneAuthNotFound);
    if (!r.ok || r.value === null) return r;
    return ChainReader.zeroneAccountOf(r.value);
  }

  async accountByDid(did: string): Promise<Read<ZeroneAccount | null>> {
    const r = await this.restMaybe(`/zerone/auth/v1/account_by_did/${encodeURIComponent(did)}`, ChainReader.zeroneAuthNotFound);
    if (!r.ok || r.value === null) return r;
    return ChainReader.zeroneAccountOf(r.value);
  }

  private static potOf(p: any): Pot | null {
    if (!p || typeof p.id !== "string") return null;
    const total = String(p.totalAmount ?? p.total_amount ?? "");
    const claimed = String(p.claimedAmount ?? p.claimed_amount ?? "");
    if (!/^\d+$/.test(total) || !/^\d+$/.test(claimed)) return null;
    const status = typeof p.status === "string" ? p.status : String(p.status);
    const whitelist = Array.isArray(p.eligibility?.whitelist) ? p.eligibility.whitelist.map(String) : [];
    return { id: p.id, status, total: BigInt(total), claimed: BigInt(claimed), whitelist };
  }

  async pot(id: string): Promise<Read<Pot | null>> {
    const r = await this.restMaybe(`/zerone/claiming_pot/v1/pot/${encodeURIComponent(id)}`,
      (status, body) => status === 500 && body?.code === 2 && body?.message === `pot not found: ${id}`);
    if (!r.ok || r.value === null) return r;
    const pot = ChainReader.potOf(r.value.pot);
    return pot ? { ok: true, value: pot } : { ok: false, error: "pot: unexpected shape" };
  }

  async allPots(): Promise<Read<Pot[]>> {
    const r = await this.restPaged("/zerone/claiming_pot/v1/pots", "pots");
    if (!r.ok) return r;
    const pots: Pot[] = [];
    for (const raw of r.value) {
      const pot = ChainReader.potOf(raw);
      if (!pot) return { ok: false, error: "pots: unexpected shape" };
      pots.push(pot);
    }
    return { ok: true, value: pots };
  }

  private static grantsOf(list: any[]): Read<Grant[]> {
    const grants: Grant[] = [];
    for (const g of list) {
      if (typeof g?.granter !== "string" || typeof g?.grantee !== "string") return { ok: false, error: "grants: unexpected shape" };
      grants.push({ granter: g.granter, grantee: g.grantee, allowance: summarizeAllowance(g.allowance) });
    }
    return { ok: true, value: grants };
  }

  async grantsByGrantee(address: string): Promise<Read<Grant[]>> {
    const r = await this.restPaged(`/cosmos/feegrant/v1beta1/allowances/${address}`, "allowances");
    return r.ok ? ChainReader.grantsOf(r.value) : r;
  }

  async grantsByGranter(address: string): Promise<Read<Grant[]>> {
    const r = await this.restPaged(`/cosmos/feegrant/v1beta1/issued/${address}`, "allowances");
    return r.ok ? ChainReader.grantsOf(r.value) : r;
  }

  async grant(granter: string, grantee: string): Promise<Read<Grant | null>> {
    const r = await this.restMaybe(`/cosmos/feegrant/v1beta1/allowance/${granter}/${grantee}`,
      (status, body) => status === 500 && body?.code === 13 && body?.message === "fee-grant not found: not found");
    if (!r.ok || r.value === null) return r;
    const list = ChainReader.grantsOf([r.value.allowance]);
    return list.ok ? { ok: true, value: list.value[0]! } : list;
  }

  async txByHash(hashHex: string): Promise<Read<IncludedTx | null>> {
    const hash = hashHex.toUpperCase();
    if (!/^[0-9A-F]{64}$/.test(hash)) return { ok: false, error: "tx hash has the wrong shape" };
    const r = await this.getJson(`${this.rpc}/tx?hash=0x${hash}`);
    if (!r.ok) return r;
    const body = r.value.body;
    if (body?.error) {
      if (body.error.data === `tx (${hash}) not found`) return { ok: true, value: null };
      return { ok: false, error: `tx lookup: ${String(body.error.data ?? body.error.message).slice(0, 160)}` };
    }
    const res = body?.result;
    const height = Number(res?.height);
    if (!res?.tx_result || !Number.isSafeInteger(height)) return { ok: false, error: "tx lookup: unexpected shape" };
    return {
      ok: true,
      value: {
        height,
        code: Number(res.tx_result.code ?? 0),
        log: String(res.tx_result.log ?? ""),
        gasUsed: Number(res.tx_result.gas_used ?? 0),
        gasWanted: Number(res.tx_result.gas_wanted ?? 0),
      },
    };
  }

  /** The transactions in blocks that used `address`'s account sequence
   * `sequence`, found by the tx.acc_seq event ("<address>/<sequence>") the
   * SDK's SetPubKeyDecorator emits for each signer. Only a transaction that
   * passed the whole ante handler keeps its events, and the same handler
   * increments the sequence, so a correct chain shows at most one. An empty
   * list proves nothing: the index may lag, or the node may not index it.
   * Each comes with its messages, read from the transaction bytes the search
   * returns; a transaction whose bytes cannot be read fails the whole read. */
  async txsBySignerSequence(address: string, sequence: string): Promise<Read<Array<IncludedTx & { hash: string; messages: TxMessage[] }>>> {
    if (!/^[a-z0-9]{1,83}$/.test(address) || !/^\d{1,20}$/.test(sequence)) return { ok: false, error: "acc_seq search: bad address or sequence" };
    const query = encodeURIComponent(`"tx.acc_seq='${address}/${sequence}'"`);
    const r = await this.getJson(`${this.rpc}/tx_search?query=${query}&page=1&per_page=10`);
    if (!r.ok) return r;
    const body = r.value.body;
    if (body?.error) return { ok: false, error: `acc_seq search: ${String(body.error.data ?? body.error.message).slice(0, 160)}` };
    const list = body?.result?.txs;
    if (!Array.isArray(list)) return { ok: false, error: "acc_seq search: unexpected shape" };
    const out: Array<IncludedTx & { hash: string; messages: TxMessage[] }> = [];
    for (const t of list) {
      const height = Number(t?.height);
      if (typeof t?.hash !== "string" || !/^[0-9A-Fa-f]{64}$/.test(t.hash) || !t?.tx_result || !Number.isSafeInteger(height)) {
        return { ok: false, error: "acc_seq search: unexpected shape" };
      }
      let messages: TxMessage[];
      try {
        messages = txMessagesOf(Uint8Array.from(Buffer.from(String(t.tx ?? ""), "base64")));
      } catch (error) {
        return { ok: false, error: `acc_seq search: tx ${t.hash.slice(0, 16)} unreadable: ${error instanceof Error ? error.message : String(error)}` };
      }
      out.push({
        messages,
        hash: t.hash.toUpperCase(),
        height,
        code: Number(t.tx_result.code ?? 0),
        log: String(t.tx_result.log ?? ""),
        gasUsed: Number(t.tx_result.gas_used ?? 0),
        gasWanted: Number(t.tx_result.gas_wanted ?? 0),
      });
    }
    return { ok: true, value: out };
  }

  /** Read-only ABCI Simulate of encoded tx bytes. Simulate skips signature
   * checks and some decorators, so it measures gas and execution only. */
  async simulate(txBytes: Uint8Array): Promise<Read<{ gasUsed: number; height: number }>> {
    const request = Uint8Array.from([0x12, ...encodeVarint(txBytes.length), ...txBytes]);
    const url = `${this.rpc}/abci_query?path=%22/cosmos.tx.v1beta1.Service/Simulate%22&data=0x${Buffer.from(request).toString("hex")}`;
    const r = await this.getJson(url);
    if (!r.ok) return r;
    const resp = r.value.body?.result?.response;
    if (!resp) return { ok: false, error: "simulate: unexpected answer" };
    if (resp.code) return { ok: false, error: `simulate refused (code ${resp.code}): ${String(resp.log ?? "").slice(0, 400)}` };
    try {
      for (const [field, value] of protoFields(Uint8Array.from(Buffer.from(String(resp.value ?? ""), "base64")))) {
        if (field === 1 && value instanceof Uint8Array) {
          for (const [f, v] of protoFields(value)) {
            if (f === 2 && typeof v === "number") return { ok: true, value: { gasUsed: v, height: Number(resp.height) } };
          }
        }
      }
    } catch (error) {
      return { ok: false, error: `simulate: ${error instanceof Error ? error.message : String(error)}` };
    }
    return { ok: false, error: "simulate: no gas_used in the answer" };
  }
}

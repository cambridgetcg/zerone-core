// Test support. Every key here is a public test vector (SHA-256 of a fixed
// label), never funded and never used outside tests.
import { createHash, createPrivateKey, createPublicKey, sign as edSign } from "node:crypto";
import { chmodSync, mkdirSync, mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { encodeBech32 } from "../src/bech32.ts";
import { moduleVersionsDigest, protoFields, type AllowanceSummary, type FetchLike, type Grant, type Pot } from "../src/chain.ts";
import type { ChainVersionPin, DoorConfig } from "../src/config.ts";
import type { GlobalFacts, SubjectFacts } from "../src/gates.ts";
import { emptyLedger, type Ledger } from "../src/ledger.ts";
import {
  adr036SignBytes,
  envelopeText,
  registrationProofBytes,
  requestText,
  solvePow,
} from "../src/request.ts";
import { accountAddressBytes, publicKeyFromPrivate, signSecp256k1ForTest } from "../src/secp256k1.ts";

export const NOW = new Date("2026-10-28T17:31:00.000Z");
export const ISSUED = "2026-10-28T17:30:00.000Z";
export const BLOCK_TIME = "2026-10-28T17:30:50.000Z";

/** A small stand-in for the chain's module_versions and its upgrade history. */
export const TEST_MODULES = [{ name: "claiming_pot", version: "1" }, { name: "auth", version: "5" }, { name: "feegrant", version: "2" }];
export const TEST_UPGRADES = [{ name: "liquiditypool-hardening-v1", height: 44_636 }, { name: "agenttool-seam-v1", height: 527_603 }];
export const TEST_PIN: ChainVersionPin = { last_upgrade: "agenttool-seam-v1", module_versions_sha256: moduleVersionsDigest(TEST_MODULES) };

/** x/upgrade done markers as an ABCI subspace query returns them (Pairs). */
export function doneMarkers(upgrades: ReadonlyArray<{ name: string; height: number }>): Buffer {
  const out: number[] = [];
  for (const u of upgrades) {
    const key = Buffer.alloc(9 + u.name.length);
    key[0] = 0x01;
    key.writeBigUInt64BE(BigInt(u.height), 1);
    key.write(u.name, 9, "ascii");
    const pair = [0x0a, ...encodeVarint(key.length), ...key, 0x12, 0x01, 0x01];
    out.push(0x0a, ...encodeVarint(pair.length), ...pair);
  }
  return Buffer.from(out);
}

export function seed(label: string): Uint8Array {
  return Uint8Array.from(createHash("sha256").update(`zerone-door test vector: ${label}`).digest());
}

export function addressFor(label: string): string {
  return encodeBech32("zrn", accountAddressBytes(publicKeyFromPrivate(seed(`${label}/account`))));
}

/** A stand-in for R: its address and public key only. */
export const R = addressFor("registrar");
export const R_PUBKEY_B64 = Buffer.from(publicKeyFromPrivate(seed("registrar/account"))).toString("base64");
export const HOUSEHOLD = addressFor("household");

const PKCS8_ED25519 = Buffer.from("302e020100300506032b657004220420", "hex");

export function identityKeys(label: string): { publicKey: Uint8Array; sign: (msg: Uint8Array) => Uint8Array } {
  const priv = createPrivateKey({ key: Buffer.concat([PKCS8_ED25519, Buffer.from(seed(`${label}/identity`))]), format: "der", type: "pkcs8" });
  const spki = createPublicKey(priv).export({ format: "der", type: "spki" });
  return {
    publicKey: Uint8Array.from(spki.subarray(spki.length - 32)),
    sign: (msg) => Uint8Array.from(edSign(null, msg, priv)),
  };
}

export interface MadeRequest {
  text: string;
  address: string;
  identityHex: string;
}

export interface MakeOptions {
  label?: string;
  chainId?: string;
  issuedAt?: string;
  nonceHex?: string;
  /** Mutations applied after signing, to produce specific defects. */
  tamper?: (fields: Record<string, string>) => void;
  addressOverride?: string;
  identityLabel?: string;
}

const cache = new Map<string, MadeRequest>();

export function makeRequest(opts: MakeOptions = {}): MadeRequest {
  const key = JSON.stringify({ ...opts, tamper: opts.tamper ? String(opts.tamper) : null });
  const hit = cache.get(key);
  if (hit) return hit;
  const label = opts.label ?? "agent-1";
  const priv = seed(`${label}/account`);
  const pub = publicKeyFromPrivate(priv);
  const address = opts.addressOverride ?? encodeBech32("zrn", accountAddressBytes(pub));
  const id = identityKeys(opts.identityLabel ?? label);
  const identityHex = Buffer.from(id.publicKey).toString("hex");
  const chainId = opts.chainId ?? "zerone-1";
  const request = requestText({
    chainId,
    address,
    identityPublicKeyHex: identityHex,
    nonceHex: opts.nonceHex ?? createHash("sha256").update(`nonce/${label}`).digest("hex").slice(0, 32),
    issuedAt: opts.issuedAt ?? ISSUED,
  });
  const digest = Uint8Array.from(createHash("sha256").update(adr036SignBytes(address, new TextEncoder().encode(request))).digest());
  const k = BigInt(`0x${createHash("sha256").update(`k/${request}`).digest("hex")}`);
  const accountSig = signSecp256k1ForTest(digest, priv, k);
  let identitySig = new Uint8Array(64);
  try {
    identitySig = id.sign(registrationProofBytes(chainId, address, id.publicKey));
  } catch { /* a non-canonical address cannot be proven: leave zeros */ }
  const fields: Record<string, string> = {
    request,
    accountPublicKeyHex: Buffer.from(pub).toString("hex"),
    accountSignatureHex: Buffer.from(accountSig).toString("hex"),
    identitySignatureHex: Buffer.from(identitySig).toString("hex"),
    powNonce: solvePow(request),
  };
  opts.tamper?.(fields);
  const made = {
    text: envelopeText(fields.request!, {
      accountPublicKeyHex: fields.accountPublicKeyHex!,
      accountSignatureHex: fields.accountSignatureHex!,
      identitySignatureHex: fields.identitySignatureHex!,
      powNonce: fields.powNonce!,
    }),
    address,
    identityHex,
  };
  cache.set(key, made);
  return made;
}

export function doorAllowance(expiration: string, spend = 1_100_000n): AllowanceSummary {
  return {
    types: [
      "/cosmos.feegrant.v1beta1.AllowedMsgAllowance",
      "/cosmos.feegrant.v1beta1.PeriodicAllowance",
      "/cosmos.feegrant.v1beta1.BasicAllowance",
    ],
    allowedMessages: [
      "/zerone.claiming_pot.v1.MsgClaim",
      "/zerone.auth.v1.MsgRegisterAccount",
      "/zerone.knowledge.v1.MsgSubmitClaim",
    ],
    spendLimitUzrn: spend,
    expiration,
    periodSeconds: 900,
    periodLimitUzrn: 400_000n,
  };
}

export function grantFromR(grantee: string, expiration = "2026-11-11T17:00:00Z", spend = 1_100_000n): Grant {
  return { granter: R, grantee, allowance: doorAllowance(expiration, spend) };
}

export function unclaimedPot(address: string): Pot {
  return { id: `bootstrap-${address}`, status: "POT_STATUS_ACTIVE", total: 222_000n, claimed: 0n, whitelist: [address] };
}

export function baseGlobal(over: Partial<GlobalFacts> = {}): GlobalFacts {
  return {
    status: { network: "zerone-1", height: 1_402_800, time: BLOCK_TIME },
    params: { registrar: R, windowCap: 25 },
    window: { height: 1_402_800, storedWindow: 3, storedCount: 1 },
    upgradePending: false,
    chainVersion: { lastUpgrade: { name: "agenttool-seam-v1", height: 527_603 }, moduleVersionsSha256: TEST_PIN.module_versions_sha256 },
    registrarSpendable: 30_000_000n,
    registrarAuth: { accountNumber: "77", sequence: "4", pubKeyType: "/cosmos.crypto.secp256k1.PubKey", pubKeyB64: R_PUBKEY_B64 },
    registrarZerone: { address: R, did: `did:zrn:${"ab".repeat(32)}`, accountType: "agent", frozen: false },
    registrarIssued: [],
    pots: [],
    ...over,
  };
}

export function baseSubject(over: Partial<SubjectFacts> = {}): SubjectFacts {
  return {
    balances: [],
    auth: null,
    zerone: null,
    potLower: null,
    potUpper: null,
    grantsToAddress: [],
    didRegistered: [],
    ...over,
  };
}

export function ledgerWith(over: Partial<Ledger> = {}): Ledger {
  return { ...emptyLedger(), ...over };
}

export function tempDir(prefix: string): string {
  return mkdtempSync(join(tmpdir(), `zerone-door-${prefix}-`));
}

export const ZERONED_SHA = "00".repeat(32);

/** A door home with config.json (0600) and allowlist.json in a temp dir. */
export function makeDoorHome(opts: { hubLocal?: boolean; desks?: string[]; lobby?: boolean } = {}): { home: string; configPath: string; config: DoorConfig } {
  const home = tempDir("home");
  chmodSync(home, 0o700);
  const configRaw = {
    schema: "zerone-door-config/v1",
    chain_id: "zerone-1",
    rpc: "http://rpc.test:26657",
    rest: "http://rest.test:1317",
    zeroned: { path: "/opt/test/zeroned", sha256: ZERONED_SHA },
    chain_version: TEST_PIN,
    registrar: { address: R, key_name: "door-key", keyring_dir: "/opt/test/kr", keychain_service: "door-test-service" },
    known_keys: [{ address: HOUSEHOLD, label: "household test key" }],
    lobby: opts.lobby ? { cli: "/opt/test/lobby", room_id: `room_${"a1".repeat(16)}`, conversation: "zerone-door", reply_lease: "lease_test" } : null,
  };
  const configPath = join(home, "config.json");
  writeFileSync(configPath, JSON.stringify(configRaw), { mode: 0o600 });
  writeFileSync(join(home, "allowlist.json"), JSON.stringify({
    schema: "zerone-door-allowlist/v1",
    hub_local: { enabled: opts.hubLocal ?? true },
    lobby_desks: (opts.desks ?? []).map((identity_id) => ({ identity_id })),
  }), { mode: 0o600 });
  for (const d of ["inbox", "processed", "replies", "state"]) mkdirSync(join(home, d), { recursive: true, mode: 0o700 });
  return {
    home,
    configPath,
    config: { ...(configRaw as any), home },
  };
}

/** A fake zerone-1 behind the REST and RPC URLs the door reads. */
export interface FakeChainState {
  height: number;
  time: string;
  network: string;
  registrar: string;
  windowCap: number;
  windowValue: Buffer;
  plan: unknown;
  modules: Array<{ name: string; version: string }>;
  upgrades: Array<{ name: string; height: number }>;
  spendable: Record<string, string>;
  balances: Record<string, Array<{ denom: string; amount: string }>>;
  auth: Record<string, { account_number: string; sequence: string; pub_key: { "@type": string; key: string } | null }>;
  zerone: Record<string, { address: string; did: string; accountType: string; flags: { frozen: boolean } }>;
  dids: Record<string, string>;
  pots: Record<string, any>;
  grants: Array<{ granter: string; grantee: string; allowance: any }>;
  /** The tx index by hash; `tx` is the transaction's bytes (TxRaw, base64), as tx_search returns them. */
  txs: Record<string, { height: string; tx_result: { code: number; log: string; gas_used: string; gas_wanted: string }; tx?: string }>;
  /** The tx.acc_seq index: "<address>/<sequence>" -> hashes of the
   * transactions in blocks that used that sequence (each also in txs). */
  accSeq: Record<string, string[]>;
  /** A REST node that lags: its auth account answers show these sequences
   * (address -> sequence) instead of the current ones. */
  restSequence: Record<string, string>;
  /** A backend behind the RPC URL that lags: its gRPC Account answers come
   * from the state at `height`, where R's sequence is `registrarSequence`.
   * Null: the current state, at `height` above. */
  laggingAccountRead: { height: number; registrarSequence: string } | null;
  simulateGas: number;
  simulateCode: number;
  fail: Set<string>;
  requests: string[];
}

export function fakeChainState(over: Partial<FakeChainState> = {}): FakeChainState {
  const windowValue = Buffer.alloc(16);
  windowValue.writeBigUInt64BE(3n, 0);
  windowValue.writeBigUInt64BE(1n, 8);
  return {
    height: 1_402_800,
    time: BLOCK_TIME,
    network: "zerone-1",
    registrar: R,
    windowCap: 25,
    windowValue,
    plan: null,
    modules: TEST_MODULES.map((m) => ({ ...m })),
    upgrades: TEST_UPGRADES.map((u) => ({ ...u })),
    spendable: { [R]: "30000000" },
    balances: {},
    auth: { [R]: { account_number: "77", sequence: "4", pub_key: { "@type": "/cosmos.crypto.secp256k1.PubKey", key: R_PUBKEY_B64 } } },
    zerone: { [R]: { address: R, did: `did:zrn:${"ab".repeat(32)}`, accountType: "agent", flags: { frozen: false } } },
    dids: {},
    pots: {},
    grants: [],
    txs: {},
    accSeq: {},
    restSequence: {},
    laggingAccountRead: null,
    simulateGas: 109_000,
    simulateCode: 0,
    fail: new Set(),
    requests: [],
    ...over,
  };
}

function json(status: number, body: unknown) {
  return { status, text: async () => JSON.stringify(body) };
}

export function fakeFetch(state: FakeChainState): FetchLike {
  return async (url: string) => {
    state.requests.push(url);
    const u = new URL(url);
    const path = decodeURIComponent(u.pathname);
    // A failure names any part of the request: a REST path, or an RPC route and its query.
    for (const f of state.fail) if (decodeURIComponent(url).includes(f)) return json(503, { error: "down" });
    if (u.host === "rpc.test:26657") {
      if (path === "/status") {
        return json(200, { result: { node_info: { network: state.network }, sync_info: { latest_block_height: String(state.height), latest_block_time: state.time } } });
      }
      if (path === "/abci_query") {
        const q = u.searchParams.get("path");
        if (q === '"/store/claiming_pot/key"') {
          return json(200, { result: { response: { code: 0, value: state.windowValue.toString("base64"), height: String(state.height) } } });
        }
        if (q === '"/cosmos.auth.v1beta1.Query/Account"') {
          const request = protoFields(Buffer.from((u.searchParams.get("data") ?? "").replace(/^0x/, ""), "hex"));
          const address = Buffer.from(request.find(([f]) => f === 1)![1] as Uint8Array).toString("utf8");
          const lag = state.laggingAccountRead;
          const height = String(lag?.height ?? state.height);
          const a = state.auth[address];
          if (!a) {
            return json(200, { result: { response: { code: 22, codespace: "sdk", log: `rpc error: code = NotFound desc = account ${address} not found: key not found`, value: null, height } } });
          }
          const sequence = lag && address === R ? lag.registrarSequence : a.sequence;
          const value = Buffer.from(queryAccountAnswer(address, Number(a.account_number), Number(sequence))).toString("base64");
          return json(200, { result: { response: { code: 0, codespace: "", log: "", value, height } } });
        }
        if (q === '"/store/upgrade/subspace"' && u.searchParams.get("data") === "0x01") {
          return json(200, { result: { response: { code: 0, value: doneMarkers(state.upgrades).toString("base64"), height: String(state.height) } } });
        }
        if (q === '"/cosmos.tx.v1beta1.Service/Simulate"') {
          if (state.simulateCode) return json(200, { result: { response: { code: state.simulateCode, log: "simulated failure", height: String(state.height) } } });
          // SimulateResponse { gas_info (1) { gas_wanted (1), gas_used (2) } }
          const gasUsed = encodeVarint(state.simulateGas);
          const gasInfo = Buffer.from([0x10, ...gasUsed]);
          const value = Buffer.from([0x0a, gasInfo.length, ...gasInfo]);
          return json(200, { result: { response: { code: 0, value: value.toString("base64"), height: String(state.height) } } });
        }
      }
      if (path === "/tx_search") {
        const m = /^"tx\.acc_seq='([^']+)'"$/.exec(u.searchParams.get("query") ?? "");
        if (!m) return json(200, { error: { code: -32603, message: "Internal error", data: "unsupported query" } });
        const txs = (state.accSeq[m[1]!] ?? []).map((hash) => ({ hash, height: state.txs[hash]!.height, index: 0, tx_result: state.txs[hash]!.tx_result, tx: state.txs[hash]!.tx ?? "" }));
        return json(200, { result: { txs, total_count: String(txs.length) } });
      }
      if (path === "/tx") {
        const hash = (u.searchParams.get("hash") ?? "").replace(/^0x/, "");
        const tx = state.txs[hash];
        if (!tx) return json(200, { error: { code: -32603, message: "Internal error", data: `tx (${hash}) not found` } });
        return json(200, { result: { hash, ...tx } });
      }
      return json(404, {});
    }
    const page = (items: unknown[], key: string) => json(200, { [key]: items, pagination: { next_key: null, total: String(items.length) } });
    let m: RegExpMatchArray | null;
    if (path === "/zerone/claiming_pot/v1/params") {
      return json(200, { params: { bootstrapRegistrar: state.registrar, bootstrapDailyAdmissionCap: String(state.windowCap) } });
    }
    if (path === "/cosmos/upgrade/v1beta1/current_plan") return json(200, { plan: state.plan });
    if (path === "/cosmos/upgrade/v1beta1/module_versions") return json(200, { module_versions: state.modules });
    if ((m = path.match(/^\/cosmos\/bank\/v1beta1\/spendable_balances\/(.+)$/))) {
      const amt = state.spendable[m[1]!];
      return page(amt ? [{ denom: "uzrn", amount: amt }] : [], "balances");
    }
    if ((m = path.match(/^\/cosmos\/bank\/v1beta1\/balances\/(.+)$/))) return page(state.balances[m[1]!] ?? [], "balances");
    if ((m = path.match(/^\/cosmos\/auth\/v1beta1\/accounts\/(.+)$/))) {
      const a = state.auth[m[1]!];
      if (!a) return json(404, { code: 5, message: `account ${m[1]} not found` });
      const sequence = state.restSequence[m[1]!] ?? a.sequence;
      return json(200, { account: { "@type": "/cosmos.auth.v1beta1.BaseAccount", address: m[1], ...a, sequence } });
    }
    if ((m = path.match(/^\/zerone\/auth\/v1\/account\/(.+)$/))) {
      const z = state.zerone[m[1]!];
      if (!z) return json(500, { code: 2, message: "codespace zerone_auth code 2: account not found" });
      return json(200, { account: z });
    }
    if ((m = path.match(/^\/zerone\/auth\/v1\/account_by_did\/(.+)$/))) {
      const owner = state.dids[m[1]!];
      if (!owner) return json(500, { code: 2, message: "codespace zerone_auth code 2: account not found" });
      return json(200, { account: { address: owner, did: m[1], accountType: "agent", flags: { frozen: false } } });
    }
    if ((m = path.match(/^\/zerone\/claiming_pot\/v1\/pot\/(.+)$/))) {
      const p = state.pots[m[1]!];
      if (!p) return json(500, { code: 2, message: `pot not found: ${m[1]}` });
      return json(200, { pot: p });
    }
    if (path === "/zerone/claiming_pot/v1/pots") return page(Object.values(state.pots), "pots");
    if ((m = path.match(/^\/cosmos\/feegrant\/v1beta1\/allowances\/(.+)$/))) return page(state.grants.filter((g) => g.grantee === m![1]), "allowances");
    if ((m = path.match(/^\/cosmos\/feegrant\/v1beta1\/issued\/(.+)$/))) return page(state.grants.filter((g) => g.granter === m![1]), "allowances");
    if ((m = path.match(/^\/cosmos\/feegrant\/v1beta1\/allowance\/([^/]+)\/([^/]+)$/))) {
      const g = state.grants.find((x) => x.granter === m![1] && x.grantee === m![2]);
      if (!g) return json(500, { code: 13, message: "fee-grant not found: not found" });
      return json(200, { allowance: g });
    }
    return json(404, { code: 5, message: "Not Found" });
  };
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

/** A length-delimited protobuf field (strings as UTF-8). */
export function pbBytes(field: number, value: Uint8Array | number[] | string): number[] {
  const bytes = typeof value === "string" ? [...Buffer.from(value, "utf8")] : [...value];
  return [...encodeVarint(field * 8 + 2), ...encodeVarint(bytes.length), ...bytes];
}

/** A varint protobuf field; proto3 leaves a zero out. */
export function pbVarint(field: number, value: number): number[] {
  return value === 0 ? [] : [...encodeVarint(field * 8), ...encodeVarint(value)];
}

/** A transaction as a block holds it (TxRaw, base64) for the fake chain's
 * indexes: its messages from their JSON form, with as much of each as the door
 * reads (type and parties), and its timeout height. */
export function txRawOf(messages: Array<Record<string, any>>, timeoutHeight = 0): string {
  const anys = messages.map((m) => {
    const type = String(m["@type"]);
    let value: number[] = [];
    if (type.endsWith(".MsgAddBootstrapEntry")) value = [...pbBytes(1, m.authority), ...m.addresses.flatMap((a: string) => pbBytes(2, a))];
    else if (type.endsWith(".MsgGrantAllowance")) value = [...pbBytes(1, m.granter), ...pbBytes(2, m.grantee), ...pbBytes(3, pbBytes(1, String(m.allowance?.["@type"] ?? "")))];
    else if (type.endsWith(".MsgRevokeAllowance")) value = [...pbBytes(1, m.granter), ...pbBytes(2, m.grantee)];
    else if (type.endsWith(".MsgSend")) value = [...pbBytes(1, m.from_address), ...pbBytes(2, m.to_address)];
    return pbBytes(1, [...pbBytes(1, type), ...pbBytes(2, value)]);
  });
  const body = [...anys.flat(), ...pbBytes(2, "zerone-door/v1"), ...pbVarint(3, timeoutHeight)];
  return Buffer.from([...pbBytes(1, body), ...pbBytes(2, []), ...pbBytes(3, [1, 2, 3])]).toString("base64");
}

/** QueryAccountResponse { Any(BaseAccount { address, account_number, sequence }) }. */
export function queryAccountAnswer(address: string, accountNumber: number, sequence: number): Uint8Array {
  const base = [...pbBytes(1, address), ...pbVarint(3, accountNumber), ...pbVarint(4, sequence)];
  return Uint8Array.from(pbBytes(1, [...pbBytes(1, "/cosmos.auth.v1beta1.BaseAccount"), ...pbBytes(2, base)]));
}

export function restPot(address: string, status = "POT_STATUS_ACTIVE", claimed = "0") {
  return {
    id: `bootstrap-${address}`,
    name: "Bootstrap seed (commitment 20)",
    totalAmount: "222000",
    claimedAmount: claimed,
    eligibility: { whitelist: [address] },
    status,
  };
}

export function restDoorGrant(granter: string, grantee: string, expiration: string, spend = "1100000") {
  return {
    granter,
    grantee,
    allowance: {
      "@type": "/cosmos.feegrant.v1beta1.AllowedMsgAllowance",
      allowance: {
        "@type": "/cosmos.feegrant.v1beta1.PeriodicAllowance",
        basic: { spend_limit: [{ denom: "uzrn", amount: spend }], expiration },
        period: "900s",
        period_spend_limit: [{ denom: "uzrn", amount: "400000" }],
        period_can_spend: [{ denom: "uzrn", amount: "400000" }],
        period_reset: expiration,
      },
      allowed_messages: [
        "/zerone.claiming_pot.v1.MsgClaim",
        "/zerone.auth.v1.MsgRegisterAccount",
        "/zerone.knowledge.v1.MsgSubmitClaim",
      ],
    },
  };
}

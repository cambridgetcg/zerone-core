import type { NetworkProfile, ObserverProfile } from "../network-profile";
import type { NetworkReadiness } from "./onboarding";

// Match the active query gateway, independently of legacy's 75-second window.
export const OBSERVER_FRESHNESS_WINDOW_MS = 30_000;
export const OBSERVER_MAX_FUTURE_SKEW_MS = 10_000;
export function observerReadinessAtAge(state: NetworkReadiness | "archive", blockAgeMs: number): NetworkReadiness | "archive" {
  if (state === "archive") return state;
  if (!Number.isFinite(blockAgeMs) || blockAgeMs < -OBSERVER_MAX_FUTURE_SKEW_MS || blockAgeMs > OBSERVER_FRESHNESS_WINDOW_MS) return "stale";
  return state; // Aging must never promote a retained regression/syncing state.
}

export const OBSERVER_INTERVAL_MS = 550; // All requests share one <=2 r/s lane.
export const OBSERVER_BLOCK_CAP = 4;
export interface ObserverBlock {
  height: number; time: string; hash: string; appHash: string; transactionCount: number;
}
export interface ObserverSnapshot {
  block: ObserverBlock;
  readiness: NetworkReadiness | "archive";
  supplyUzrn: string | null;
  validators: number | null;
  issues: string[];
}
interface Runtime {
  fetch: typeof fetch;
  now(): number;
  sleep(ms: number): Promise<void>;
}
function object(value: unknown): Record<string, unknown> {
  if (typeof value !== "object" || value === null || Array.isArray(value)) throw new Error("Incomplete observer response");
  return value as Record<string, unknown>;
}
function height(value: unknown): number {
  if (typeof value !== "string" || !/^[1-9]\d{0,9}$/.test(value) || Number(value) > 1_000_000_000) throw new Error("Invalid block height");
  return Number(value);
}
function hash(value: unknown): string {
  if (typeof value !== "string" || !/^[a-f0-9]{64}$/i.test(value)) throw new Error("Incomplete block hash");
  return value.toLowerCase();
}
function amount(value: unknown): string {
  if (typeof value !== "string" || !/^(0|[1-9]\d{0,77})$/.test(value)) throw new Error("Incomplete native amount");
  return value;
}
function nativeBytes(value: unknown): string {
  if (typeof value !== "string" || !value.length) throw new Error("Incomplete native bytes");
  const bytes = atob(value);
  if (!bytes.length || btoa(bytes) !== value) throw new Error("Noncanonical native base64");
  return bytes;
}
function validateValidator(value: unknown): void {
  const validator = object(value);
  const key = object(validator.pub_key);
  const bytes = nativeBytes(key.value);
  const validKey = key.type === "tendermint/PubKeyEd25519" ? bytes.length === 32
    : key.type === "tendermint/PubKeySecp256k1" && bytes.length === 33 && [2, 3].includes(bytes.charCodeAt(0));
  if (typeof validator.address !== "string" || !/^[a-f0-9]{40}$/i.test(validator.address) || !validKey ||
      typeof validator.voting_power !== "string" || !/^[1-9]\d{0,18}$/.test(validator.voting_power) || BigInt(validator.voting_power) > 9_223_372_036_854_775_807n) throw new Error("Incomplete consensus validator");
  // Wire shape only: no key/address derivation, signature or consensus proof.
}
async function boundedJson(response: Response): Promise<unknown> {
  const reader = response.body?.getReader();
  if (!reader) throw new Error("Empty observer response");
  let total = 0;
  const chunks: Uint8Array[] = [];
  try {
    for (;;) {
      const { done, value } = await reader.read();
      if (done) break;
      total += value.byteLength;
      if (total > 2_097_152) throw new Error("Observer response exceeds size limit");
      chunks.push(value);
    }
  } catch (error) { await reader.cancel().catch(() => {}); throw error; }
  const bytes = new Uint8Array(total);
  let offset = 0;
  for (const chunk of chunks) { bytes.set(chunk, offset); offset += chunk.length; }
  return JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(bytes)) as unknown;
}

export function createObserverClient(profile: NetworkProfile, origin: string, overrides: Partial<Runtime> = {}) {
  const runtime: Runtime = { fetch: (...args) => globalThis.fetch(...args), now: Date.now, sleep: (ms) => new Promise((resolve) => setTimeout(resolve, ms)), ...overrides };
  let queue: Promise<unknown> = Promise.resolve();
  let nextStart = 0;
  let pending = 0;
  let acceptedHeight: number | null = null;
  let previousHeight: number | null = null;
  const blocks = new Map<number, ObserverBlock>();
  const inFlight = new Map<number, Promise<ObserverBlock>>();
  function configured(): ObserverProfile {
    if (profile.mode === "legacy" || profile.mode === "unconfigured") throw new Error("Observer unconfigured; no legacy fallback");
    return profile;
  }
  function requireIdentity(): void {
    configured();
    if (acceptedHeight === null) throw new Error("Check network identity before lookup");
  }
  async function read(path: string): Promise<Record<string, unknown>> {
    configured();
    if (pending >= 16) throw new Error("Observer request queue is full");
    pending += 1;
    const work = queue.then(async () => {
      await runtime.sleep(Math.max(0, nextStart - runtime.now()));
      nextStart = runtime.now() + OBSERVER_INTERVAL_MS;
      const response = await runtime.fetch(new URL(`/api/${path}`, origin), {
        method: "GET", redirect: "error", headers: { Accept: "application/json" }, signal: AbortSignal.timeout(10_000),
      });
      if (!response.ok) throw new Error(response.status === 404 ? "Record not found or query unavailable (HTTP 404)" : `Observer unavailable (HTTP ${response.status})`);
      const body = object(await boundedJson(response));
      if (body.error) throw new Error("Query unavailable; record may be absent, unindexed or pruned");
      return body;
    });
    queue = work.catch(() => {}).finally(() => { pending -= 1; });
    return work;
  }
  async function pointBlock(wanted: number): Promise<ObserverBlock> {
    height(String(wanted));
    const cached = blocks.get(wanted);
    if (cached) return cached;
    const existing = inFlight.get(wanted);
    if (existing) return existing;
    const work = (async () => {
      const result = object((await read(`rpc/block?height=${wanted}`)).result);
      const block = object(result.block);
      const header = object(block.header);
      if (header.chain_id !== configured().chainId || height(header.height) !== wanted) throw new Error("Block identity mismatch");
      if (typeof header.time !== "string" || !Number.isFinite(Date.parse(header.time))) throw new Error("Invalid block time");
      const txs = object(block.data).txs;
      if (txs !== null) {
        if (!Array.isArray(txs)) throw new Error("Incomplete block transactions");
        for (const tx of txs) nativeBytes(tx);
      }
      const normalized: ObserverBlock = { height: wanted, time: header.time, hash: hash(object(result.block_id).hash), appHash: hash(header.app_hash), transactionCount: txs === null ? 0 : txs.length };
      blocks.set(wanted, normalized);
      if (blocks.size > 16) blocks.delete(blocks.keys().next().value!);
      return normalized;
    })();
    inFlight.set(wanted, work);
    try { return await work; } finally { inFlight.delete(wanted); }
  }
  return {
    async snapshot(): Promise<ObserverSnapshot> {
      acceptedHeight = null;
      try {
        const selected = configured();
        const result = object((await read("rpc/status")).result);
        const info = object(result.node_info);
        const sync = object(result.sync_info);
        if (info.network !== selected.chainId) throw new Error("Wrong chain identity; no fallback");
        const latest = height(sync.latest_block_height);
        if (typeof sync.catching_up !== "boolean" || typeof sync.latest_block_time !== "string" || !Number.isFinite(Date.parse(sync.latest_block_time))) throw new Error("Incomplete network status");
        const latestHash = hash(sync.latest_block_hash);
        const appHash = hash(sync.latest_app_hash);
        const checkpoint = selected.checkpoint;
        if (checkpoint && (latest !== checkpoint.height || latestHash !== checkpoint.blockHash || appHash !== checkpoint.appHash || !sync.catching_up)) throw new Error("Archive checkpoint mismatch");
        const block = await pointBlock(latest);
        if (block.hash !== latestHash || block.appHash !== appHash || Date.parse(block.time) !== Date.parse(sync.latest_block_time)) throw new Error("Status / block identity mismatch");
        const state = checkpoint ? "archive" : previousHeight !== null && latest < previousHeight ? "stale" : sync.catching_up ? "syncing" : "ready";
        const readiness = observerReadinessAtAge(state, runtime.now() - Date.parse(block.time));
        previousHeight = Math.max(previousHeight ?? 0, latest);
        const issues: string[] = [];
        let supplyUzrn: string | null = null;
        let validators: number | null = null;
        try {
          const supply = object((await read("rest/cosmos/bank/v1beta1/supply/by_denom?denom=uzrn")).amount);
          if (supply.denom !== "uzrn") throw new Error("Wrong denomination");
          supplyUzrn = amount(supply.amount);
        } catch { issues.push("Supply unknown: native supply query failed or was incomplete."); }
        try {
          const set = object((await read("rpc/validators?page=1&per_page=100")).result);
          if (typeof set.total !== "string" || !/^[1-9]\d{0,5}$/.test(set.total) || !Array.isArray(set.validators) || set.validators.length > 100 || set.validators.length !== Math.min(Number(set.total), 100)) throw new Error("Incomplete consensus set");
          const addresses = new Set<string>();
          for (const validator of set.validators) {
            validateValidator(validator);
            const address = String(object(validator).address).toLowerCase();
            if (addresses.has(address)) throw new Error("Duplicate consensus validator");
            addresses.add(address);
          }
          validators = Number(set.total);
        } catch { issues.push("Validator count unknown: consensus query failed or was incomplete."); }
        acceptedHeight = latest;
        return { block, readiness, supplyUzrn, validators, issues };
      } catch (error) {
        blocks.clear();
        throw error;
      }
    },
    async recentBlocks(count = OBSERVER_BLOCK_CAP): Promise<ObserverBlock[]> {
      requireIdentity();
      const bounded = Math.min(OBSERVER_BLOCK_CAP, Math.max(1, Math.floor(count)));
      if (!Number.isFinite(bounded)) throw new Error("Invalid block count");
      const result: ObserverBlock[] = [];
      const latest = acceptedHeight!;
      for (let value = latest; value > 0 && result.length < bounded; value -= 1) result.push(await pointBlock(value));
      return result;
    },
    async block(wanted: number): Promise<ObserverBlock> {
      requireIdentity();
      if (wanted > acceptedHeight!) throw new Error("Height exceeds the observed record");
      return pointBlock(wanted);
    },
    async balance(address: string): Promise<string> {
      requireIdentity();
      if (!/^zrn1[023456789acdefghjklmnpqrstuvwxyz]{38}$/.test(address)) throw new Error("Enter a native zrn1 address");
      const balance = object((await read(`rest/cosmos/bank/v1beta1/balances/${address}/by_denom?denom=uzrn`)).balance);
      if (balance.denom !== "uzrn") throw new Error("Native balance unknown");
      return amount(balance.amount);
    },
    async transaction(wanted: string): Promise<{ hash: string; height: number; code: number }> {
      requireIdentity();
      const normalized = hash(wanted);
      const tx = object((await read(`rpc/tx?hash=0x${normalized}&prove=false`)).result);
      const code = object(tx.tx_result).code;
      if (hash(tx.hash) !== normalized || height(tx.height) > acceptedHeight! || !Number.isSafeInteger(code) || Number(code) < 0) throw new Error("Transaction result unknown or mismatched");
      return { hash: normalized, height: height(tx.height), code: Number(code) };
    },
  };
}

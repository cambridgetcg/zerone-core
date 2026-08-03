import { existsSync } from "node:fs";
import { GAS_CLAIM, GAS_COMMIT, GAS_REVEAL, expandHome, type NetworkConfig } from "./config.ts";

export function zeronedBin(env: Record<string, string | undefined> = process.env): string {
  if (env.ZERONED_BIN) return env.ZERONED_BIN;
  const agentCopy = expandHome("~/.zerone-agent/bin/zeroned", env);
  if (existsSync(agentCopy)) return agentCopy;
  return "zeroned";
}

export interface RunResult {
  exit_code: number;
  stdout: string;
  stderr: string;
}

export function run(args: string[]): RunResult {
  const proc = Bun.spawnSync({ cmd: [zeronedBin(), ...args], stdout: "pipe", stderr: "pipe" });
  return {
    exit_code: proc.exitCode ?? -1,
    stdout: proc.stdout.toString(),
    stderr: proc.stderr.toString(),
  };
}

function extractJson(output: string): Record<string, unknown> | null {
  const start = output.indexOf("{");
  if (start === -1) return null;
  try {
    return JSON.parse(output.slice(start)) as Record<string, unknown>;
  } catch {
    return null;
  }
}

export function query(net: NetworkConfig, args: string[]): Record<string, unknown> | null {
  const result = run([...args, "--node", net.rpc, "--output", "json"]);
  if (result.exit_code !== 0) return null;
  return extractJson(result.stdout);
}

export function latestHeight(net: NetworkConfig): number {
  const result = run(["status", "--node", net.rpc]);
  const parsed = extractJson(result.stdout) ?? extractJson(result.stderr);
  const info = parsed?.sync_info as { latest_block_height?: string } | undefined;
  const height = Number(info?.latest_block_height ?? 0);
  if (!height) throw new Error(`could not read height from ${net.rpc}`);
  return height;
}

export interface TxOutcome {
  ok: boolean;
  tx_hash: string | null;
  code?: number;
  raw_log?: string;
  detail?: string;
}

export function broadcast(net: NetworkConfig, key: string, gas: number, txArgs: string[]): TxOutcome {
  const args = [
    "tx", ...txArgs,
    "--from", key,
    "--chain-id", net.chain_id,
    "--node", net.rpc,
    "--home", expandHome(net.keyring_home),
    "--keyring-backend", "test",
    "--gas", String(gas),
    "--fees", `${gas}uzrn`,
    "--broadcast-mode", "sync",
    "--yes",
    "--output", "json",
  ];
  const result = run(args);
  const parsed = extractJson(result.stdout);
  if (parsed && typeof parsed.txhash === "string") {
    const code = Number(parsed.code ?? 0);
    if (code === 0) return { ok: true, tx_hash: parsed.txhash };
    return { ok: false, tx_hash: parsed.txhash, code, raw_log: String(parsed.raw_log ?? "") };
  }
  return { ok: false, tx_hash: null, detail: result.stderr.trim().split("\n")[0] || result.stdout.trim().slice(0, 200) };
}

export interface IncludedTx {
  found: boolean;
  height?: number;
  code?: number;
  raw_log?: string;
  events?: Array<{ type: string; attributes: Array<{ key: string; value: string }> }>;
}

export function lookupTx(net: NetworkConfig, txHash: string): IncludedTx {
  const result = run(["query", "tx", txHash, "--node", net.rpc, "--output", "json"]);
  if (result.exit_code !== 0) return { found: false };
  const parsed = extractJson(result.stdout);
  if (!parsed) return { found: false };
  return {
    found: true,
    height: Number(parsed.height ?? 0),
    code: Number(parsed.code ?? 0),
    raw_log: typeof parsed.raw_log === "string" ? parsed.raw_log : undefined,
    events: parsed.events as IncludedTx["events"],
  };
}

export async function waitForTx(net: NetworkConfig, txHash: string, timeoutMs = 90_000): Promise<IncludedTx> {
  const deadline = Date.now() + timeoutMs;
  for (;;) {
    const lookup = lookupTx(net, txHash);
    if (lookup.found) return lookup;
    if (Date.now() >= deadline) return lookup;
    await Bun.sleep(4_000);
  }
}

export function eventAttr(tx: IncludedTx, typePattern: RegExp, key: string): string | null {
  for (const event of tx.events ?? []) {
    if (!typePattern.test(event.type)) continue;
    for (const attr of event.attributes) {
      if (attr.key === key || attr.key.endsWith(key)) return attr.value.replaceAll('"', "");
    }
  }
  return null;
}

export function keyAddress(net: NetworkConfig, key: string): string {
  const result = run(["keys", "show", key, "-a", "--keyring-backend", "test", "--home", expandHome(net.keyring_home)]);
  const address = result.stdout.trim();
  if (result.exit_code !== 0 || !/^zrn1[0-9a-z]{20,90}$/.test(address)) {
    throw new Error(`cannot resolve key "${key}": ${result.stderr.trim() || address}`);
  }
  return address;
}

export function spendableUzrn(net: NetworkConfig, address: string): number {
  const parsed = query(net, ["query", "bank", "balances", address]);
  const balances = (parsed?.balances ?? []) as Array<{ denom: string; amount: string }>;
  const entry = balances.find(b => b.denom === "uzrn");
  return entry ? Number(entry.amount) : 0;
}

export const CLAIM_GAS = GAS_CLAIM;
export const COMMIT_GAS = GAS_COMMIT;
export const REVEAL_GAS = GAS_REVEAL;

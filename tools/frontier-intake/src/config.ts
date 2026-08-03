export interface NetworkConfig {
  name: string;
  chain_id: string;
  rpc: string;
  keyring_home: string;
  submitter: string;
  verifiers: string[];
  funder: string;
}

// Endpoint and key defaults observed live 2026-08-03; all overridable by flags.
export const NETWORKS: Record<string, NetworkConfig> = {
  "zerone-1": {
    name: "zerone-1",
    chain_id: "zerone-1",
    rpc: "http://169.155.55.44:26657",
    keyring_home: "~/.zeroned/mainnet-ops",
    submitter: "zerone-ops",
    // relayer deliberately NOT a panel key: it is the live relay daemon's key
    // and spends attestation bonds on its own schedule, drifting under the
    // verifier balance gate mid-round (observed 2026-08-03).
    verifiers: ["ai-agenttool", "validator", "math-panel-1", "math-panel-2"],
    funder: "zerone-ops",
  },
  "zerone-testnet-1": {
    name: "zerone-testnet-1",
    chain_id: "zerone-testnet-1",
    rpc: "http://37.16.28.121:26657",
    keyring_home: "~/.zeroned/testnet-ops",
    submitter: "math-sub",
    verifiers: ["math-v1", "math-v2", "math-v3", "math-v4"],
    funder: "faucet",
  },
};

export const DEFAULT_REVIEW_FEE_UZRN = 200_000; // ceremony-proven safe max under pacing
export const GAS_CLAIM = 500_000;
export const GAS_COMMIT = 200_000;
export const GAS_REVEAL = 200_000;
export const GAS_SEND = 200_000;
// Verifier gate reads 100 ZRN spendable AFTER ante fee deduction. Fund with
// real headroom: each round costs a verifier ~0.4 ZRN in commit+reveal fees
// and balances drift (observed keys sliding under 100 mid-batch).
export const VERIFIER_MIN_UZRN = 100_000_000;
export const VERIFIER_FUND_UZRN = 111_000_000;
export const VERIFIER_TOPUP_FLOOR_UZRN = 104_000_000;
export const SUBMITTER_FUND_UZRN = 30_000_000;

export function expandHome(path: string, env: Record<string, string | undefined> = process.env): string {
  const home = env.HOME;
  if (!home) return path;
  if (path === "~") return home;
  if (path.startsWith("~/")) return `${home}${path.slice(1)}`;
  return path;
}

export function statePath(network: string, env: Record<string, string | undefined> = process.env): string {
  if (env.FRONTIER_INTAKE_STATE_DIR) return `${env.FRONTIER_INTAKE_STATE_DIR}/${network}.state.json`;
  return expandHome(`~/.zerone-agent/frontier-intake/${network}.state.json`, env);
}

export function defaultRegisterPath(): string {
  // Discover the lexically-latest snapshot so a re-snapshot PR never has to
  // edit tool source (docs/research/math-breakthroughs-<window>.vN.json).
  const dir = new URL("../../../docs/research/", import.meta.url).pathname;
  const glob = new Bun.Glob("math-breakthroughs-*.v*.json");
  const matches = [...glob.scanSync(dir)].sort();
  const latest = matches.at(-1);
  if (!latest) throw new Error(`no math-breakthroughs-*.v*.json register found in ${dir}`);
  return `${dir}${latest}`;
}

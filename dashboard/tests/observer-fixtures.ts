import type { ObserverProfile } from "../network-profile";

// All identities, amounts, release references and hashes here are SYNTHETIC.
export const BLOCK_HASH = "a".repeat(64);
export const APP_HASH = "b".repeat(64);
export const NATIVE_ADDRESS = "zrn16sp9l62q9jmetsheus8zpjm77zulnlcr26hnkf";
export const NOW = Date.parse("2026-09-08T08:00:00Z");
export const syntheticProfile = (mode: ObserverProfile["mode"] = "preview"): ObserverProfile => ({
  mode,
  chainId: "synthetic-observer-1",
  gatewayOrigin: mode === "preview" ? "http://127.0.0.1:4180" : "https://gateway.example.org",
  genesisSha256: "c".repeat(64),
  releaseCommit: "d".repeat(40),
  releaseManifestSha256: "e".repeat(64),
  releaseUrl: mode === "preview" ? "https://example.invalid/synthetic-release" : "https://release.example.org/immutable-bundle",
  operatorVerification: { openDecisionSha256: "f".repeat(64), verifiedAt: "2026-09-08T07:00:00Z" },
  ...(mode === "beta-archive" ? { checkpoint: { height: 10, blockHash: BLOCK_HASH, appHash: APP_HASH } } : {}),
});
export const syntheticValidator = () => ({
  address: "11".repeat(20),
  pub_key: { type: "tendermint/PubKeyEd25519", value: "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE=" },
  voting_power: "1", proposer_priority: "0",
});
export function syntheticResponse(path: string, now = NOW, archive = false): Record<string, unknown> {
  const time = new Date(now - 1000).toISOString();
  if (path === "/status") return { result: { node_info: { network: "synthetic-observer-1" }, sync_info: { latest_block_height: "10", latest_block_time: time, latest_block_hash: BLOCK_HASH, latest_app_hash: APP_HASH, catching_up: archive } } };
  if (path.startsWith("/block?")) return { result: { block_id: { hash: BLOCK_HASH }, block: { header: { chain_id: "synthetic-observer-1", height: new URL(path, "http://localhost").searchParams.get("height"), time, app_hash: APP_HASH }, data: { txs: null } } } };
  if (path === "/validators?page=1&per_page=100") return { result: { block_height: "10", count: "1", total: "1", validators: [syntheticValidator()] } };
  if (path.includes("/supply/by_denom?denom=uzrn")) return { amount: { denom: "uzrn", amount: "100000000" } };
  if (path.includes("/balances/")) return { balance: { denom: "uzrn", amount: "0" } };
  if (path.startsWith("/tx?")) return { result: { hash: BLOCK_HASH, height: "9", tx_result: { code: 0 } } };
  return { error: "Synthetic route unavailable" };
}

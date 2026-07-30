#!/usr/bin/env node

import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { fileURLToPath } from "node:url";
import path from "node:path";

const scriptDirectory = path.dirname(fileURLToPath(import.meta.url));
const repositoryRoot = path.resolve(scriptDirectory, "..");
const readJson = async (relativePath) =>
  JSON.parse(await readFile(path.join(repositoryRoot, relativePath), "utf8"));

const chain = await readJson("integrations/chain-registry/zerone/chain.json");
const assetList = await readJson("integrations/chain-registry/zerone/assetlist.json");
const genesis = await readJson("deploy/mainnet/artifacts/genesis.json");
const dashboardConfig = await readFile(
  path.join(repositoryRoot, "dashboard/src/config.ts"),
  "utf8",
);
const appTemplate = await readFile(
  path.join(repositoryRoot, "config/app.toml.template"),
  "utf8",
);
const mainnetEntrypoint = await readFile(
  path.join(repositoryRoot, "deploy/mainnet/entrypoint.sh"),
  "utf8",
);
const sharedValidatorEntrypoint = await readFile(
  path.join(repositoryRoot, "deploy/fly-validator-entrypoint-common.sh"),
  "utf8",
);

assert.equal(chain.chain_name, "zerone");
assert.equal(assetList.chain_name, chain.chain_name);
assert.equal(chain.chain_id, genesis.chain_id);
assert.equal(chain.chain_id, "zerone-1");
assert.equal(chain.chain_type, "cosmos");
assert.equal(chain.network_type, "mainnet");
assert.equal(chain.bech32_prefix, "zrn");
assert.equal(chain.slip44, 118);
assert.match(dashboardConfig, /CHAIN_ID = "zerone-1"/);
assert.match(dashboardConfig, /DENOM = "uzrn"/);
assert.match(dashboardConfig, /coinType: 118/);

const nativeAsset = assetList.assets.find((asset) => asset.base === "uzrn");
assert.ok(nativeAsset, "assetlist must include the native uzrn asset");
assert.equal(nativeAsset.display, "zrn");
assert.equal(nativeAsset.symbol, "ZRN");
assert.deepEqual(
  nativeAsset.denom_units.map(({ denom, exponent }) => [denom, exponent]),
  [
    ["uzrn", 0],
    ["mzrn", 3],
    ["zrn", 6],
  ],
);

const feeToken = chain.fees.fee_tokens[0];
assert.equal(feeToken.denom, nativeAsset.base);
assert.equal(feeToken.fixed_min_gas_price, 0.025);
assert.match(appTemplate, /minimum-gas-prices = "0\.025uzrn"/);
assert.match(
  mainnetEntrypoint,
  /zerone-fly-validator-entrypoint/,
  "mainnet wrapper must delegate to the reviewed shared validator entrypoint",
);
assert.match(
  sharedValidatorEntrypoint,
  /minimum-gas-prices 0\.025uzrn/,
  "shared validator entrypoint must enforce the published fixed gas price",
);
assert.equal(chain.staking.staking_tokens[0].denom, nativeAsset.base);
assert.equal(
  chain.staking.lock_duration.time,
  genesis.app_state.staking.params.unbonding_time,
);
assert.equal(genesis.app_state.staking.params.bond_denom, nativeAsset.base);

assert.equal(
  chain.codebase.genesis.genesis_url,
  "https://raw.githubusercontent.com/cambridgetcg/zerone-core/main/deploy/mainnet/artifacts/genesis.json",
);
assert.equal(chain.apis.rpc[0].address, "http://169.155.55.44:26657");
assert.equal(chain.apis.rest, undefined);
assert.equal(chain.codebase.recommended_version, undefined);
assert.equal(chain.codebase.binaries, undefined);
assert.equal(chain.codebase.ibc, undefined);
assert.equal(chain.explorers[0].url, "https://zerone.ai/");
assert.equal(chain.snapshots, undefined);

console.log("Zerone Chain Registry facts are internally consistent.");

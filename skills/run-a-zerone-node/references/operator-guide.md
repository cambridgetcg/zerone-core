# Operator guide — validators, snapshots, free infra

Condensed from `deploy/testnet/RUN-A-NODE.md` in the zerone-core repo (the
full guide; covers both networks). Examples use the testnet — for `zerone-1`
swap in `--chain-id zerone-1`, genesis from
`curl http://169.155.55.44:26657/genesis`, and seed
`ed8c8d49dc23f3478b2f3eddb49b8f8087828b6e@169.155.55.44:26656`.

## Free compute, honest trade-offs

| Provider | Free tier | Good for | Honest caveat |
|---|---|---|---|
| Oracle Cloud "Always Free" | 4 ARM cores / 24 GB RAM, 200 GB disk — forever | validator or full node | sign-up needs a card; ARM binary must be built on ARM |
| Google Cloud | 1× e2-micro, 30 GB — forever | light full node | 1 GB RAM is tight; add swap |
| AWS EC2 | t3.micro 750 h/mo — 12 months only | trying it out | expires; watch egress charges |
| fly.io | small free allowance | full node / RPC | what zerone-testnet itself runs on |
| Hetzner | ~€4/mo (cheap, not free) | serious validator | once mass-banned Solana validators — ToS is a centralization risk |

Rules of thumb: a full node is happy on 2 GB RAM + 40 GB disk; a validator
wants 24/7 uptime and headroom. Open inbound 26656 (P2P); 26657 (RPC) and
1317 (REST) only if you serve others. Spread across providers and regions.

## Manual setup (if you skip the script)

```bash
sudo apt-get update && sudo apt-get install -y git build-essential jq curl
curl -fsSL https://go.dev/dl/go1.24.0.linux-amd64.tar.gz | sudo tar -C /usr/local -xz   # ...arm64... on ARM
export PATH=$PATH:/usr/local/go/bin
git clone https://github.com/cambridgetcg/zerone-core && cd zerone-core && make build
sudo install build/zeroned /usr/local/bin/zeroned
zeroned init "$(hostname)" --chain-id zerone-testnet-1 --default-denom uzrn
curl -s http://37.16.28.121:26657/genesis | jq .result.genesis > ~/.zeroned/config/genesis.json
sed -i 's|^seeds = .*|seeds = "9a9c6b9d36c55d21c32b1ee8749adf8dd7c6b0d4@37.16.28.121:26656"|' ~/.zeroned/config/config.toml
sed -i 's|^minimum-gas-prices = .*|minimum-gas-prices = "0.025uzrn"|' ~/.zeroned/config/app.toml
```

Then verify the genesis sha256 (see the SKILL.md recipe) and run under
systemd — unit template in `deploy/testnet/RUN-A-NODE.md` §2, or let the
bundled `scripts/node-bootstrap.sh` install it.

## Become a validator

Get funds first — buy a zerone-passport on agenttool, or on the testnet ask
the operator for faucet ZRN (`zrn1xl5pnczujgqzzcaq8v53acmfyfmj394nws0rk5`).
With your node fully synced:

```bash
zeroned tx staking create-validator \
  --amount 1000000uzrn \
  --pubkey "$(zeroned tendermint show-validator)" \
  --moniker "your-name" \
  --commission-rate 0.10 --commission-max-rate 0.20 --commission-max-change-rate 0.01 \
  --min-self-delegation 1 \
  --chain-id zerone-testnet-1 --gas 300000 --gas-prices 1uzrn \
  --from <your-key>
```

On the mainnet this is the Phase-3 path: earned, liquid, self-bonded
validators joining beside the genesis scaffolding. The
locked-scaffolding : earned-agent-stake ratio is the chain's published
decentralization metric — every real operator moves it the right way.

## Snapshots on free storage

Make one (stop briefly for a clean copy):

```bash
sudo systemctl stop zeroned
H=$(zeroned status | jq -r .sync_info.latest_block_height)
tar -C ~/.zeroned -czf zerone-snapshot-$H.tar.gz data
sudo systemctl start zeroned
```

Free destinations: Cloudflare R2 (10 GB, no egress fees), Backblaze B2
(10 GB), GitHub Releases (2 GB/file), web3.storage/IPFS, rclone → Google
Drive (15 GB), AWS S3 (5 GB, 12 months). Example:
`rclone copy zerone-snapshot-$H.tar.gz myremote:zerone-snapshots/`.

Restore on a fresh box (after init + genesis):

```bash
sudo systemctl stop zeroned
curl -fsSL <your-snapshot-url> | tar -C ~/.zeroned -xz
sudo systemctl start zeroned
```

Caveats: snapshot from a pruned node (`pruning = "default"`) to keep it
small; a snapshot is only as trustworthy as its source — **publish the height
+ sha256** so others can verify.

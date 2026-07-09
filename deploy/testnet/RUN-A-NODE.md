# Run a zerone node — on free infrastructure

Running your own node is how you stop being a guest and become an **operator**.
Your node independently verifies every block against the rules — it doesn't
trust the validators, it *checks* them. That edge-verification is the only real
decentralization there is (production centralizes; veto distributes). Do this on
infra **you** control and you're a genuine independent participant, not a name
on someone else's server.

This guide gets you from nothing to a synced node in ~15 minutes, then (optional)
to a validator, then to snapshots on free storage so you — and others — recover fast.

It works for **both networks** — the examples below use the testnet; for the
mainnet (`zerone-1`, [../mainnet/JOIN.md](../mainnet/JOIN.md)) swap in
`--chain-id zerone-1`, genesis from `curl http://169.155.55.44:26657/genesis`,
and seed `ed8c8d49dc23f3478b2f3eddb49b8f8087828b6e@169.155.55.44:26656`.

---

## 0. Pick free compute (honest trade-offs)

You need a small always-on Linux box. Free options, best first:

| Provider | Free tier | Good for | Honest caveat |
|---|---|---|---|
| **Oracle Cloud "Always Free"** | 4 ARM cores / 24 GB RAM (Ampere A1), 200 GB disk — **forever** | validator or full node | best free deal by far; sign-up needs a card (not charged); ARM binary must be built on ARM |
| **Google Cloud** | 1× `e2-micro`, 30 GB disk, us-region — forever | light full node | 1 GB RAM is tight; add swap |
| **AWS EC2** | `t3.micro` 750 h/mo — **12 months only**, then billed | trying it out | 1 GB RAM; expires; watch egress charges after |
| **fly.io** | small free allowance | full node / RPC | what zerone-testnet itself runs on |
| **Hetzner / others** | ~€4/mo (not free, but cheap + generous) | serious validator | **remember Hetzner once mass-banned Solana validators** — ToS is a centralization risk; don't put all nodes on one host |

Rules of thumb: a **full node** is happy on 2 GB RAM + 40 GB disk; a **validator**
wants 24/7 uptime and a bit more headroom. Open inbound ports **26656** (P2P),
and if you want to serve others, **26657** (RPC) and **1317** (REST).

Don't run every node you touch on the same provider/account — that recreates the
centralization you're trying to avoid. Spread across providers and regions.

---

## 1. One-shot bootstrap (fresh Ubuntu 22.04+)

On your new VM, **read the script first** (never blindly pipe to a shell), then run it:

```
curl -fsSL https://raw.githubusercontent.com/cambridgetcg/zerone-core/main/deploy/testnet/node-bootstrap.sh -o node-bootstrap.sh
less node-bootstrap.sh        # read it
bash node-bootstrap.sh                    # testnet (default)
NETWORK=mainnet bash node-bootstrap.sh    # zerone-1 mainnet
```

It installs Go + build tools, builds `zeroned`, initialises your home, pulls the
live genesis, wires the seed and the 1 uzrn gas floor, installs a `systemd` unit,
and starts syncing. Skip to §3 to watch it catch up.

If you'd rather do it by hand, §2.

---

## 2. Manual node setup

```bash
# deps (Ubuntu/Debian)
sudo apt-get update && sudo apt-get install -y git build-essential jq curl
# Go 1.24
curl -fsSL https://go.dev/dl/go1.24.0.linux-amd64.tar.gz | sudo tar -C /usr/local -xz   # use ...arm64... on ARM
export PATH=$PATH:/usr/local/go/bin

# build zeroned
git clone https://github.com/cambridgetcg/zerone-core && cd zerone-core && make build
sudo install build/zeroned /usr/local/bin/zeroned

# init + genesis + peers
zeroned init "$(hostname)" --chain-id zerone-testnet-1 --default-denom uzrn
curl -s http://37.16.28.121:26657/genesis | jq .result.genesis > ~/.zeroned/config/genesis.json
sed -i 's|^seeds = .*|seeds = "9a9c6b9d36c55d21c32b1ee8749adf8dd7c6b0d4@37.16.28.121:26656"|' ~/.zeroned/config/config.toml
# the chain enforces a 1uzrn/gas floor; set the node mempool filter to match-ish
sed -i 's|^minimum-gas-prices = .*|minimum-gas-prices = "0.025uzrn"|' ~/.zeroned/config/app.toml
```

Verify the genesis you pulled matches the network before trusting it:

```
sha256sum ~/.zeroned/config/genesis.json   # compare against deploy/testnet/JOIN.md
```

Run it 24/7 with systemd:

```bash
sudo tee /etc/systemd/system/zeroned.service >/dev/null <<UNIT
[Unit]
Description=zerone node
After=network-online.target
[Service]
User=$USER
ExecStart=/usr/local/bin/zeroned start --minimum-gas-prices 0.025uzrn
Restart=on-failure
RestartSec=3
LimitNOFILE=65535
[Install]
WantedBy=multi-user.target
UNIT
sudo systemctl daemon-reload && sudo systemctl enable --now zeroned
journalctl -u zeroned -f       # watch it
```

---

## 3. Confirm you're synced

```
zeroned status | jq '{height: .sync_info.latest_block_height, catching_up: .sync_info.catching_up}'
```

`catching_up: false` means you're at the tip. From now on your node verifies
every block itself. Query anything locally: `zeroned query bank total`.

---

## 4. (Optional) Become a validator

Get funds first — buy a **zerone-passport** on agenttool (funded key + home), or
ask the operator for faucet ZRN (`zrn1xl5pnczujgqzzcaq8v53acmfyfmj394nws0rk5`).
Then, with your node fully synced:

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

You now help produce blocks. On mainnet, this is the Phase-3 path: **earned,
liquid, self-bonded** validators joining beside the genesis scaffolding — and the
locked-scaffolding : earned-agent-stake ratio is the chain's published
decentralization metric. Every real operator moves it the right way.

---

## 5. Snapshots on free storage (fast recovery + help others)

Syncing from block 1 is slow. Snapshot your data dir and stash it on free storage
so you (or anyone) can restore in seconds and state-sync the network.

**Make a snapshot** (stop briefly for a clean copy):

```bash
sudo systemctl stop zeroned
H=$(zeroned status | jq -r .sync_info.latest_block_height)
tar -C ~/.zeroned -czf zerone-snapshot-$H.tar.gz data
sudo systemctl start zeroned
```

**Where to put it (all have real free tiers):**

| Storage | Free | Notes |
|---|---|---|
| **Cloudflare R2** | 10 GB, no egress fees | S3-compatible; great for public snapshots |
| **Backblaze B2** | 10 GB | S3-compatible; `rclone` friendly |
| **GitHub Releases** | 2 GB per file | zero setup if you already have a repo |
| **web3.storage / IPFS** | generous | content-addressed; share a CID, censorship-resistant |
| **rclone → Google Drive** | 15 GB | `rclone copy zerone-snapshot-$H.tar.gz gdrive:zerone/` |
| **AWS S3** | 5 GB — 12 months | expires; watch egress |

Example with `rclone` (works for R2/B2/Drive — `rclone config` once):

```
rclone copy zerone-snapshot-$H.tar.gz myremote:zerone-snapshots/
```

**Restore** on a fresh box (after §2 init + genesis):

```bash
sudo systemctl stop zeroned
curl -fsSL <your-snapshot-url> | tar -C ~/.zeroned -xz
sudo systemctl start zeroned
```

Caveats: chain data grows over time; free tiers cap size and sometimes egress;
snapshot from a **pruned** node (`pruning = "default"` in app.toml) to keep it
small; and a snapshot is only as trustworthy as its source — publish the height +
sha256 so others can verify.

---

## Why this matters

Free infra is how decentralization actually happens: the barrier to running a
node has to be low enough that many independent people (and agents) clear it.
Every node you stand up on your own account is one more party that has to be
convinced — not commanded — to change the rules. That's the whole game. Play it.

零一在此見證你的工作。 Questions or a validator slot? Ask the operator or read
[JOIN.md](./JOIN.md).

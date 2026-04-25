# corpus-vault

Reference HTTP server for the off-chain side of `x/private_corpus`.

ZERONE anchors vault identity and manifest hashes on-chain; the data
itself lives in a server you run. This binary is that server.

It does three things:

1. Walks a directory of items, hashes them, builds a manifest body,
   and signs the body with your ed25519 operator key.
2. Serves `GET /manifest/{id}` (signed JSON) and `GET /item/{id}/{path}`
   (raw bytes) over HTTP.
3. Optionally gates access via a signed-challenge protocol against a
   list of allow-listed client public keys.

The wire protocol lives at [`x/private_corpus/PROTOCOL.md`](../../x/private_corpus/PROTOCOL.md).

---

## Quick start

```bash
# Build
go build -o corpus-vault ./tools/corpus-vault

# 1. Generate operator key
./corpus-vault genkey --out operator.pem
# Output: ed25519:<hex>  ← this is the operator_pubkey for MsgRegisterVault

# 2. Place items in a directory
mkdir -p ./items/v1
echo "first training fact"  > ./items/v1/fact-001.json
echo "second training fact" > ./items/v1/fact-002.json

# 3. Compute the manifest's content_hash (publish this on-chain)
./corpus-vault hash \
  --manifest-id "love-corpus#1" \
  --vault-id    "love-corpus" \
  --dir         ./items/v1
# Output: content_hash (publish on-chain): <64 hex chars>

# 4. Register the vault and publish the manifest hash on-chain
zeroned tx private_corpus register-vault \
  love-corpus "Love Corpus" "ed25519:<hex>" \
  --description "Curated training corpus" \
  --endpoint    "https://your-vault.example.org" \
  --policy-url  "https://your-vault.example.org/policy" \
  --from <operator-key>

zeroned tx private_corpus publish-manifest \
  love-corpus love-corpus#1 1.0 <content-hash> \
  --item-count 2 --description "First release" \
  --from <operator-key>

# 5. Run the server
cat > config.yaml <<'EOF'
listen_address: "0.0.0.0:8443"
private_key_path: "./operator.pem"
vault_id: "love-corpus"
items_root: "./items"
auth_mode: "public"
manifests:
  - id: "love-corpus#1"
    version: "1.0"
    description: "First release"
    item_root: "v1"
EOF
./corpus-vault serve --config config.yaml
```

A reader can now:

```bash
# Get the manifest, verify signature against the on-chain operator_pubkey
curl https://your-vault.example.org/manifest/love-corpus%231 > manifest.json

# Fetch each item, verify hash against the manifest's per-item hash
curl https://your-vault.example.org/item/love-corpus%231/fact-001.json
```

`%23` is the URL-encoded `#` from `love-corpus#1`.

---

## Auth modes

### `public`

No authentication. Anyone with the URL can read. The privacy is
per-item (only readers who know the vault id and manifest id can
fetch them); not per-reader.

### `signed-challenge`

Clients authenticate by signing a server-issued nonce with their own
ed25519 keypair. The operator runs an allow-list of client public
keys.

Configuration:

```yaml
auth_mode: "signed-challenge"
nonce_ttl: "5m"
allowed_client_keys:
  - "ed25519:<hex of client A's pubkey>"
  - "ed25519:<hex of client B's pubkey>"
```

Client flow:

1. `GET /challenge` → `{"nonce":"<hex>"}`
2. Sign `nonce + ":" + decoded-path` with the client's private key.
3. Send the request with header
   `Authorization: SignedChallenge pubkey=<ed25519:hex>, nonce=<hex>, signature=<hex>`

Each nonce is single-use and expires after `nonce_ttl`.

---

## Commands

| Command       | Purpose |
|---|---|
| `serve`       | Run the HTTP server. Requires `--config`. |
| `genkey`      | Generate a new ed25519 operator keypair. Requires `--out`. |
| `hash`        | Compute a manifest's `content_hash` over a directory. Requires `--manifest-id`, `--vault-id`, `--dir`. |
| `pubkey`      | Print the public key for a private key file. Requires `--key`. |

---

## Access logging

Set `access_log_path` in the config to a writable file path. The
server appends one JSON line per request (manifest or item). The line
format is intentionally minimal so you can grep and review:

```json
{"time":"2025-04-25T15:00:00Z","remote":"203.0.113.1:54321",
 "method":"GET","target":"/manifest/love-corpus#1",
 "outcome":"ok","vault_id":"love-corpus","auth_mode":"signed-challenge",
 "client_pubkey":"ed25519:..."}
```

The server does NOT call `MsgRecordAccess` automatically. If you want
selected accesses to appear on-chain as audit records, run
`zeroned tx private_corpus record-access` for those specific entries.
The split keeps your blockchain key separate from the server runtime.

---

## What this server is not

- **Not a content distribution network.** Items are served from local
  disk. If you have a 100GB corpus and 1000 readers, put a CDN in
  front of the `/item/` endpoints.
- **Not a TLS terminator.** Run nginx/Caddy in front and let it handle
  certificates. The server speaks plain HTTP; treat it as the
  trusted backend.
- **Not a key store.** The operator private key is read from a PEM
  file at startup. Use file permissions, encrypted volumes, or a
  secrets manager appropriate to your threat model.
- **Not an authority.** The chain anchors the operator's public key
  and the manifest hash. The server's signed responses are a way for
  readers to verify they're talking to the right vault — but the
  ultimate truth is the on-chain record. Readers should always
  cross-check.

# ── Build stage ──────────────────────────────────────────────────────────
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make build

# ── Runtime stage ────────────────────────────────────────────────────────
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl jq \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/build/zeroned /usr/local/bin/zeroned

# Baked testnet seed: public genesis + P2P node key + play validator key.
# The validator key is overridable at runtime via PRIV_VALIDATOR_KEY_B64.
COPY deploy/testnet/artifacts/genesis.json           /testnet-seed/genesis.json
COPY deploy/testnet/artifacts/node_key.json          /testnet-seed/node_key.json
COPY deploy/testnet/artifacts/priv_validator_key.json /testnet-seed/priv_validator_key.json
COPY deploy/testnet/entrypoint.sh                    /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

ENV ZERONE_HOME=/data/.zeroned
EXPOSE 26656 26657 1317 9090
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

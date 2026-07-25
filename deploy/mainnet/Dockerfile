# Build stage
FROM golang:1.24-bookworm AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make build

# Runtime stage
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl jq \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/build/zeroned /usr/local/bin/zeroned
# Baked zerone-1 seed: honest genesis + P2P node key + operator validator key
# (custodial launch — the validator key is play-experimental, overridable via secret).
COPY deploy/mainnet/artifacts/genesis.json            /mainnet-seed/genesis.json
COPY deploy/mainnet/artifacts/node_key.json           /mainnet-seed/node_key.json
COPY deploy/mainnet/artifacts/priv_validator_key.json /mainnet-seed/priv_validator_key.json
COPY deploy/mainnet/entrypoint.sh                     /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh
ENV ZERONE_HOME=/data/.zeroned
EXPOSE 26656 26657 1317 9090
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

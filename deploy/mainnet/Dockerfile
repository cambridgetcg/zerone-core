# syntax=docker/dockerfile:1.7

# This Dockerfile is only for the custody-free context assembled by
# deploy/mainnet/build-image.sh. Never point a Docker client at the repository
# root: even files which are not copied can be uploaded to a remote builder.
ARG GO_IMAGE=golang:1.24-bookworm
ARG RUNTIME_IMAGE=debian:bookworm-slim
FROM ${GO_IMAGE} AS builder
WORKDIR /src
COPY .sanitized-mainnet-context-v1 /src/.sanitized-mainnet-context-v1
COPY go.mod go.sum ./
RUN go mod download
COPY app ./app
COPY cmd ./cmd
COPY x ./x
COPY docs/swagger-ui ./docs/swagger-ui

ARG VERSION
ARG COMMIT
ARG SOURCE_DATE_EPOCH
ARG GENESIS_SHA256
ARG BUILD_PROFILE
ARG RELEASE_TAG
ARG RELEASE_SIGNER_FINGERPRINT
ARG TARGETOS
ARG TARGETARCH
RUN test "${TARGETOS}/${TARGETARCH}" = "linux/amd64" && \
    test -n "${VERSION}" && test -n "${COMMIT}" && \
    test -n "${SOURCE_DATE_EPOCH}" && \
    grep -Fqx "profile=${BUILD_PROFILE}" /src/.sanitized-mainnet-context-v1 && \
    grep -Fqx "commit=${COMMIT}" /src/.sanitized-mainnet-context-v1 && \
    grep -Fqx "genesis_sha256=${GENESIS_SHA256}" /src/.sanitized-mainnet-context-v1 && \
    grep -Fqx "release_tag=${RELEASE_TAG}" /src/.sanitized-mainnet-context-v1 && \
    grep -Fqx "release_signer_fingerprint=${RELEASE_SIGNER_FINGERPRINT}" /src/.sanitized-mainnet-context-v1 && \
    grep -Fqx "platform=${TARGETOS}/${TARGETARCH}" /src/.sanitized-mainnet-context-v1 && \
    CGO_ENABLED=0 GOOS="${TARGETOS}" GOARCH="${TARGETARCH}" \
      go build -trimpath -buildvcs=false \
      -ldflags "-s -w \
        -X github.com/cosmos/cosmos-sdk/version.Name=zerone \
        -X github.com/cosmos/cosmos-sdk/version.AppName=zeroned \
        -X github.com/cosmos/cosmos-sdk/version.Version=${VERSION} \
        -X github.com/cosmos/cosmos-sdk/version.Commit=${COMMIT}" \
      -o /out/zeroned ./cmd/zeroned

# Runtime stage
FROM ${RUNTIME_IMAGE}
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl jq util-linux \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /out/zeroned /usr/local/bin/zeroned
# The public genesis is safe to bake into the image. Private P2P and validator
# keys are deliberately excluded from the build context and must arrive as
# first-boot secrets; image layers and remote builder caches are not key vaults.
COPY public/genesis.json /mainnet-seed/genesis.json
COPY runtime/entrypoint.sh /usr/local/bin/entrypoint.sh
ARG GENESIS_SHA256
RUN echo "${GENESIS_SHA256}" | grep -Eq '^[0-9a-f]{64}$' && \
    test "$(sha256sum /mainnet-seed/genesis.json | awk '{print $1}')" = "${GENESIS_SHA256}" && \
    chmod 0555 /usr/local/bin/entrypoint.sh /usr/local/bin/zeroned && \
    chmod 0444 /mainnet-seed/genesis.json
ENV ZERONE_HOME=/data/.zeroned
EXPOSE 26656 26657 1317 9090
STOPSIGNAL SIGTERM
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

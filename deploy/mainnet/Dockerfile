# Build stage
FROM golang:1.25.12-bookworm@sha256:ea341baa9bd5ba6784f6d7161ace70544349a6242d54d34a0fbfd2c4d51c9d58 AS builder

ARG VERSION
ARG COMMIT
ARG SOURCE_DATE_EPOCH

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN printf '%s' "${VERSION}" | grep -Eq '^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$' \
    && printf '%s' "${COMMIT}" | grep -Eq '^[0-9a-f]{40}$' \
    && printf '%s' "${SOURCE_DATE_EPOCH}" | grep -Eq '^[1-9][0-9]*$' \
    && SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH}" make build VERSION="${VERSION}" COMMIT="${COMMIT}" \
    && test "$(/app/build/zeroned version)" = "${VERSION}" \
    && /app/build/zeroned version --long | grep -F "commit: ${COMMIT}" >/dev/null

# Runtime stage
FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818

ARG VERSION
ARG COMMIT
ARG SOURCE_DATE_EPOCH

LABEL org.opencontainers.image.source="https://github.com/zerone-chain/zerone" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.version="${VERSION}" \
      io.zerone.source-date-epoch="${SOURCE_DATE_EPOCH}"

RUN sed -i \
      -e 's|http://deb.debian.org/debian-security|http://snapshot.debian.org/archive/debian-security/20260713T000000Z|g' \
      -e 's|http://deb.debian.org/debian|http://snapshot.debian.org/archive/debian/20260713T000000Z|g' \
      /etc/apt/sources.list.d/debian.sources \
    && printf 'Acquire::Check-Valid-Until "false";\n' >/etc/apt/apt.conf.d/99snapshot \
    && apt-get update && apt-get install -y --no-install-recommends ca-certificates curl jq openssl \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /app/build/zeroned /usr/local/bin/zeroned
# Only public consensus material belongs in an image. Node identity and
# validator signing keys are supplied at runtime from a mounted file or secret.
COPY deploy/mainnet/artifacts/genesis.json /mainnet-seed/genesis.json
COPY deploy/mainnet/entrypoint.sh          /usr/local/bin/entrypoint.sh
COPY deploy/fly-validator-entrypoint-common.sh /usr/local/libexec/zerone-fly-validator-entrypoint
RUN chmod +x /usr/local/bin/entrypoint.sh /usr/local/libexec/zerone-fly-validator-entrypoint
ENV ZERONE_HOME=/data/.zeroned
EXPOSE 26656 26657 1317 9090
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

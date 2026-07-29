# syntax=docker/dockerfile:1.7

# This Dockerfile accepts only the custody-free context assembled by
# build-image.sh. The release binary is already audited and is carried through
# byte-for-byte; there is intentionally no source builder stage.
ARG RUNTIME_IMAGE=debian:bookworm-slim
FROM ${RUNTIME_IMAGE} AS runtime

RUN apt-get update && apt-get install -y --no-install-recommends \
      bash ca-certificates coreutils curl jq util-linux \
    && rm -rf /var/lib/apt/lists/*

COPY .sanitized-zerone-2-context-v2 /network/build-context.env
COPY release/zeroned /usr/local/bin/zeroned
COPY runtime/entrypoint.sh /usr/local/bin/zerone-2-entrypoint
COPY public/genesis.json /network/genesis.json
COPY public/network-manifest.json /network/network-manifest.json

ARG SOURCE_COMMIT
ARG RELEASE_TAG
ARG TAG_SIGNER_FINGERPRINT
ARG GENESIS_SHA256
ARG NETWORK_MANIFEST_SHA256
ARG BINARY_SHA256
ARG BINARY_VERSION
ARG BINARY_GOOS
ARG BINARY_GOARCH
ARG VALIDATOR_NODE_ID
ARG TARGETOS
ARG TARGETARCH
RUN echo "${SOURCE_COMMIT}" | grep -Eq '^[0-9a-f]{40}$' && \
    echo "${TAG_SIGNER_FINGERPRINT}" | grep -Eq '^([0-9a-f]{40}|[0-9a-f]{64})$' && \
    echo "${GENESIS_SHA256}" | grep -Eq '^[0-9a-f]{64}$' && \
    echo "${NETWORK_MANIFEST_SHA256}" | grep -Eq '^[0-9a-f]{64}$' && \
    echo "${BINARY_SHA256}" | grep -Eq '^[0-9a-f]{64}$' && \
    test -n "${BINARY_VERSION}" && \
    test "${BINARY_GOOS}" = linux && \
    echo "${BINARY_GOARCH}" | grep -Eq '^(amd64|arm64)$' && \
    test "${TARGETOS}" = "${BINARY_GOOS}" && \
    test "${TARGETARCH}" = "${BINARY_GOARCH}" && \
    echo "${VALIDATOR_NODE_ID}" | grep -Eq '^[0-9a-f]{40}$' && \
    grep -qx 'schema=zerone-2-sanitized-image-context-v2' /network/build-context.env && \
    test "$(sha256sum /usr/local/bin/zeroned | awk '{print $1}')" = "${BINARY_SHA256}" && \
    test "$(sha256sum /network/genesis.json | awk '{print $1}')" = "${GENESIS_SHA256}" && \
    test "$(sha256sum /network/network-manifest.json | awk '{print $1}')" = "${NETWORK_MANIFEST_SHA256}" && \
    jq -e \
      --arg commit "${SOURCE_COMMIT}" \
      --arg tag "${RELEASE_TAG}" \
      --arg signer "${TAG_SIGNER_FINGERPRINT}" \
      --arg genesis "${GENESIS_SHA256}" \
      --arg binary "${BINARY_SHA256}" \
      --arg version "${BINARY_VERSION}" \
      --arg goos "${BINARY_GOOS}" \
      --arg goarch "${BINARY_GOARCH}" \
      --arg node "${VALIDATOR_NODE_ID}" \
      '.schema == "zerone-2-network-manifest-v2" and
       .mode == "real" and
       .chain_id == "zerone-2" and
       .genesis_sha256 == $genesis and
       .release.source_commit == $commit and
       .release.tag == $tag and
       .release.tag_signer_fingerprint == $signer and
       .release.binary_sha256 == $binary and
       .release.binary_version == $version and
       .release.binary_goos == $goos and
       .release.binary_goarch == $goarch and
       .validator.node_id == $node' \
      /network/network-manifest.json >/dev/null && \
    BINARY_VERSION_OUTPUT=$(/usr/local/bin/zeroned version 2>&1) && \
    ACTUAL_BINARY_VERSION=$(printf '%s\n' "${BINARY_VERSION_OUTPUT}" | sed -n '1p') && \
    test "${ACTUAL_BINARY_VERSION}" = "${BINARY_VERSION}" && \
    printf 'schema=zerone-2-runtime-provenance-v2\nmode=real\nchain_id=zerone-2\nsource_commit=%s\nrelease_tag=%s\ntag_signer_fingerprint=%s\ngenesis_sha256=%s\nnetwork_manifest_sha256=%s\nbinary_sha256=%s\nbinary_version=%s\nbinary_goos=%s\nbinary_goarch=%s\nvalidator_node_id=%s\n' \
      "${SOURCE_COMMIT}" "${RELEASE_TAG}" "${TAG_SIGNER_FINGERPRINT}" \
      "${GENESIS_SHA256}" "${NETWORK_MANIFEST_SHA256}" "${BINARY_SHA256}" \
      "${BINARY_VERSION}" "${BINARY_GOOS}" "${BINARY_GOARCH}" "${VALIDATOR_NODE_ID}" \
      > /network/network.env && \
    chmod 0555 /usr/local/bin/zerone-2-entrypoint /usr/local/bin/zeroned && \
    chmod 0444 /network/genesis.json /network/network-manifest.json \
      /network/network.env /network/build-context.env

ENV ZERONE_HOME=/data/.zeroned
VOLUME ["/data"]
EXPOSE 26656 26657 1317 9090

HEALTHCHECK --interval=15s --timeout=3s --start-period=30s --retries=4 \
  CMD curl -fsS http://127.0.0.1:26657/status | \
      jq -e '.result.node_info.network == "zerone-2"' >/dev/null || exit 1

ENTRYPOINT ["/usr/local/bin/zerone-2-entrypoint"]

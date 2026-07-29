# Fail-closed tombstone for the legacy zerone-testnet-1 image path.
#
# This deliberately contains no validator binary, genesis, node identity, or
# signing key. The running network predates current main and may only be
# replaced through a signed, governance-scheduled release packet.
FROM debian:bookworm-slim

COPY deploy/testnet/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

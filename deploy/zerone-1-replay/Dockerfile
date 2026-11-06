# zerone-1 non-voting observer image. COPY-only on a digest-pinned base: no package installs, no
# network during the build, no keys. bin/ holds one zeroned per ladder rung, each named by its own
# sha256 (tools/extract-rung-binary.sh for the historical rungs, the lane build for the last one).
# Build for linux/amd64 with tools/build-image.sh, which checks every input before docker sees it.
FROM docker.io/library/debian@sha256:5ae3c39ebd15e229dcedd5cee596b2497182493d41ff162e824ba13fc1b2b867

# COPY keeps the build context's modes (build-image.sh sets 0444 data, 0555 programs); setting
# them in a RUN would duplicate every binary into another layer.
COPY ladder.tsv genesis.json observer-start.sh /opt/observer/
COPY bin/ /opt/observer/bin/

# Fail the build unless: the genesis is zerone-1's; every binary hashes to its file name; every
# ladder row has its binary and nothing else is in bin/; and no key-shaped file exists anywhere.
RUN set -eu; cd /opt/observer; \
    for f in observer-start.sh bin/*; do [ -x "$f" ] || { echo "$f not executable in the build context"; exit 1; }; done; \
    echo "c30a523b9764fb76c84a53d99fcdabb966d16e7a4d3f15426ab7af5e8576170e  genesis.json" | sha256sum -c --quiet -; \
    for f in bin/*; do echo "${f#bin/}  $f" | sha256sum -c --quiet -; done; \
    want="$(awk -F'\t' '!/^#/ && NF >= 5 { print $5 }' ladder.tsv | sort -u)"; \
    have="$(ls bin | sort -u)"; \
    [ "$want" = "$have" ] || { echo "bin/ does not match ladder.tsv"; exit 1; }; \
    if find / -xdev \( -name 'priv_validator_key.json' -o -name 'node_key.json' -o -name '*.mnemonic' \
         -o -name 'keyring-*' \) -print | grep -q .; then \
      echo "key-shaped file in image"; exit 1; fi

ENV HOME=/tmp
STOPSIGNAL SIGTERM
ENTRYPOINT ["/opt/observer/observer-start.sh"]

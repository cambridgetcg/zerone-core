#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
POLICY="${ROOT}/deploy/validate-fly-phase-config.py"
TMP=$(mktemp -d "${TMPDIR:-/tmp}/zerone-fly-policy-test.XXXXXX")
trap 'rm -rf "${TMP}"' EXIT HUP INT TERM

run_policy() {
  local config=$1 schema=$2 key=$3 upstream=- f=- a=- h=- e=- b=-
  case "${key}" in
    zerone_2_gateway_private|zerone_2_gateway_public)
      upstream=replace-zerone-2-edge-app.internal
      ;;
    zerone_1_archive_gateway)
      upstream=replace-zerone-1-archive-app.internal
      a=100
      e=$(printf 'a%.0s' {1..64})
      b=$(printf 'b%.0s' {1..64})
      ;;
    zerone_1_halt_signer|zerone_1_observer)
      f=REPLACE_WITH_F
      a=REPLACE_WITH_A
      h=REPLACE_WITH_H
      ;;
  esac
  if [ "$#" -ge 7 ]; then
    upstream=$4
    f=$5
    a=$6
    h=$7
  fi
  if [ "$#" -eq 9 ]; then
    e=$8
    b=$9
  fi
  python3 "${POLICY}" "${config}" "${schema}" "${key}" \
    "${upstream}" "${f}" "${a}" "${h}" "${e}" "${b}"
}

run_policy \
  "${ROOT}/deploy/networks/zerone-2/runtime/fly.edge.example.toml" \
  zerone-2-dark-start-decision-v1 zerone_2_edge_private
SIGNER_CONFIG="${TMP}/halt-signer.toml"
OBSERVER_CONFIG="${TMP}/observer.toml"
for tuple in \
  "${ROOT}/deploy/mainnet/fly.halt-signer.example.toml|${SIGNER_CONFIG}" \
  "${ROOT}/deploy/mainnet/fly.observer.example.toml|${OBSERVER_CONFIG}"; do
  source=${tuple%%|*}
  destination=${tuple##*|}
  sed -e 's/REPLACE_WITH_F/100/g' -e 's/REPLACE_WITH_A/101/g' \
    -e 's/REPLACE_WITH_H/102/g' "${source}" > "${destination}"
done
run_policy "${SIGNER_CONFIG}" zerone-2-cutover-decision-v1 \
  zerone_1_halt_signer - 100 101 102
run_policy "${OBSERVER_CONFIG}" zerone-2-cutover-decision-v1 \
  zerone_1_observer - 100 101 102
run_policy \
  "${ROOT}/deploy/networks/zerone-2/runtime/fly.edge.public.example.toml" \
  zerone-2-open-beta-decision-v1 zerone_2_edge_public
run_policy \
  "${ROOT}/deploy/query-gateway/fly.zerone-2.public.example.toml" \
  zerone-2-open-beta-decision-v1 zerone_2_gateway_public
ARCHIVE_GATEWAY_CONFIG="${TMP}/archive-gateway.toml"
sed \
  -e 's/REPLACE_WITH_A/100/g' \
  -e "s/REPLACE_WITH_LOWERCASE_POST_A_APP_HASH/$(printf 'a%.0s' {1..64})/g" \
  -e "s/REPLACE_WITH_LOWERCASE_FINAL_APPLICATION_BLOCK_ID_HASH/$(printf 'b%.0s' {1..64})/g" \
  "${ROOT}/deploy/query-gateway/fly.zerone-1-archive.public.example.toml" \
  > "${ARCHIVE_GATEWAY_CONFIG}"
run_policy \
  "${ARCHIVE_GATEWAY_CONFIG}" \
  zerone-2-open-beta-decision-v1 zerone_1_archive_gateway

BAD_PRIVATE="${TMP}/bad-private.toml"
cp "${ROOT}/deploy/networks/zerone-2/runtime/fly.edge.example.toml" \
  "${BAD_PRIVATE}"
cat >> "${BAD_PRIVATE}" <<'EOF'

  [[services]] # intentionally indented/commented bypass form
    internal_port = 26_657
    protocol = 'tcp'
EOF
if run_policy "${BAD_PRIVATE}" \
  zerone-2-dark-start-decision-v1 zerone_2_edge_private >/dev/null 2>&1; then
  printf 'Fly phase policy test: private service bypass was accepted\n' >&2
  exit 1
fi

for override in files processes experimental; do
  BAD_OVERRIDE="${TMP}/bad-${override}.toml"
  cp "${ROOT}/deploy/networks/zerone-2/runtime/fly.edge.example.toml" \
    "${BAD_OVERRIDE}"
  case "${override}" in
    files)
      cat >> "${BAD_OVERRIDE}" <<'EOF'

[[files]]
  guest_path = "/usr/local/bin/zeroned"
  raw_value = "override"
EOF
      ;;
    processes)
      cat >> "${BAD_OVERRIDE}" <<'EOF'

[processes]
  app = "/bin/sh"
EOF
      ;;
    experimental)
      cat >> "${BAD_OVERRIDE}" <<'EOF'

[experimental]
  cmd = ["/bin/sh"]
EOF
      ;;
  esac
  if run_policy "${BAD_OVERRIDE}" \
    zerone-2-dark-start-decision-v1 zerone_2_edge_private \
    >/dev/null 2>&1; then
    printf 'Fly phase policy test: %s execution override was accepted\n' \
      "${override}" >&2
    exit 1
  fi
done

BAD_RELEASE_COMMAND="${TMP}/bad-release-command.toml"
awk '
  { print }
  /strategy = "immediate"/ { print "  release_command = \"/bin/sh\"" }
' "${ROOT}/deploy/networks/zerone-2/runtime/fly.edge.example.toml" \
  > "${BAD_RELEASE_COMMAND}"
if run_policy "${BAD_RELEASE_COMMAND}" \
  zerone-2-dark-start-decision-v1 zerone_2_edge_private \
  >/dev/null 2>&1; then
  printf 'Fly phase policy test: deploy release command was accepted\n' >&2
  exit 1
fi

BAD_ENV="${TMP}/bad-env.toml"
awk '
  { print }
  /NODE_ROLE = "edge"/ { print "  LD_PRELOAD = \"/data/override.so\"" }
' "${ROOT}/deploy/networks/zerone-2/runtime/fly.edge.example.toml" \
  > "${BAD_ENV}"
if run_policy "${BAD_ENV}" \
  zerone-2-dark-start-decision-v1 zerone_2_edge_private >/dev/null 2>&1; then
  printf 'Fly phase policy test: loader environment override was accepted\n' >&2
  exit 1
fi

BAD_GATEWAY_MOUNT="${TMP}/bad-gateway-mount.toml"
cp "${ROOT}/deploy/query-gateway/fly.zerone-2.public.example.toml" \
  "${BAD_GATEWAY_MOUNT}"
cat >> "${BAD_GATEWAY_MOUNT}" <<'EOF'

[mounts]
  source = "gateway_override"
  destination = "/etc/nginx"
EOF
if run_policy "${BAD_GATEWAY_MOUNT}" \
  zerone-2-open-beta-decision-v1 zerone_2_gateway_public \
  >/dev/null 2>&1; then
  printf 'Fly phase policy test: stateless gateway mount was accepted\n' >&2
  exit 1
fi

BAD_PUBLIC="${TMP}/bad-public.toml"
cp "${ROOT}/deploy/networks/zerone-2/runtime/fly.edge.public.example.toml" \
  "${BAD_PUBLIC}"
cat >> "${BAD_PUBLIC}" <<'EOF'

  [[services]] # second direct RPC origin
    internal_port = 26_657
    protocol = 'tcp'
    auto_stop_machines = false
    auto_start_machines = true
    min_machines_running = 1
EOF
if run_policy "${BAD_PUBLIC}" \
  zerone-2-open-beta-decision-v1 zerone_2_edge_public >/dev/null 2>&1; then
  printf 'Fly phase policy test: extra public RPC service was accepted\n' >&2
  exit 1
fi

BAD_GATEWAY="${TMP}/bad-gateway.toml"
sed 's/internal_port = 8080/internal_port = 26657/' \
  "${ROOT}/deploy/query-gateway/fly.zerone-2.public.example.toml" \
  > "${BAD_GATEWAY}"
if run_policy "${BAD_GATEWAY}" \
  zerone-2-open-beta-decision-v1 zerone_2_gateway_public >/dev/null 2>&1; then
  printf 'Fly phase policy test: direct gateway origin was accepted\n' >&2
  exit 1
fi

BAD_Z2_UPSTREAM="${TMP}/bad-z2-upstream.toml"
sed 's/replace-zerone-2-edge-app.internal/replace-validator-app.internal/' \
  "${ROOT}/deploy/query-gateway/fly.zerone-2.public.example.toml" \
  > "${BAD_Z2_UPSTREAM}"
if run_policy "${BAD_Z2_UPSTREAM}" zerone-2-open-beta-decision-v1 \
  zerone_2_gateway_public >/dev/null 2>&1; then
  printf 'Fly phase policy test: z2 gateway targeted the validator app\n' >&2
  exit 1
fi

BAD_ARCHIVE_UPSTREAM="${TMP}/bad-archive-upstream.toml"
sed 's/replace-zerone-1-archive-app.internal/replace-edge-app.internal/' \
  "${ARCHIVE_GATEWAY_CONFIG}" \
  > "${BAD_ARCHIVE_UPSTREAM}"
if run_policy "${BAD_ARCHIVE_UPSTREAM}" zerone-2-open-beta-decision-v1 \
  zerone_1_archive_gateway >/dev/null 2>&1; then
  printf 'Fly phase policy test: archive gateway targeted the z2 edge app\n' >&2
  exit 1
fi

ARCHIVE_E=$(printf 'a%.0s' {1..64})
ARCHIVE_B=$(printf 'b%.0s' {1..64})
for mutation in height app_hash block_hash; do
  bad_archive="${TMP}/bad-archive-${mutation}.toml"
  case "${mutation}" in
    height) sed 's/EXPECTED_ARCHIVE_HEIGHT = "100"/EXPECTED_ARCHIVE_HEIGHT = "101"/' \
      "${ARCHIVE_GATEWAY_CONFIG}" > "${bad_archive}" ;;
    app_hash) sed "s/${ARCHIVE_E}/$(printf 'c%.0s' {1..64})/" \
      "${ARCHIVE_GATEWAY_CONFIG}" > "${bad_archive}" ;;
    block_hash) sed "s/${ARCHIVE_B}/$(printf 'd%.0s' {1..64})/" \
      "${ARCHIVE_GATEWAY_CONFIG}" > "${bad_archive}" ;;
  esac
  if run_policy "${bad_archive}" zerone-2-open-beta-decision-v1 \
    zerone_1_archive_gateway >/dev/null 2>&1; then
    printf 'Fly phase policy test: archive gateway accepted mismatched %s\n' \
      "${mutation}" >&2
    exit 1
  fi
done

BAD_ARCHIVE_CASE="${TMP}/bad-archive-hash-case.toml"
sed "s/${ARCHIVE_E}/$(printf 'A%.0s' {1..64})/" \
  "${ARCHIVE_GATEWAY_CONFIG}" > "${BAD_ARCHIVE_CASE}"
if run_policy "${BAD_ARCHIVE_CASE}" zerone-2-open-beta-decision-v1 \
  zerone_1_archive_gateway - - 100 - "${ARCHIVE_E}" "${ARCHIVE_B}" \
  >/dev/null 2>&1; then
  printf 'Fly phase policy test: archive gateway accepted uppercase E\n' >&2
  exit 1
fi

BAD_ARCHIVE_B_CASE="${TMP}/bad-archive-block-hash-case.toml"
sed "s/${ARCHIVE_B}/$(printf 'B%.0s' {1..64})/" \
  "${ARCHIVE_GATEWAY_CONFIG}" > "${BAD_ARCHIVE_B_CASE}"
if run_policy "${BAD_ARCHIVE_B_CASE}" zerone-2-open-beta-decision-v1 \
  zerone_1_archive_gateway - - 100 - "${ARCHIVE_E}" "${ARCHIVE_B}" \
  >/dev/null 2>&1; then
  printf 'Fly phase policy test: archive gateway accepted uppercase B\n' >&2
  exit 1
fi

for bad_a in 0 01 1000000000000000000; do
  if run_policy "${ARCHIVE_GATEWAY_CONFIG}" \
    zerone-2-open-beta-decision-v1 zerone_1_archive_gateway \
    replace-zerone-1-archive-app.internal - "${bad_a}" - \
    "${ARCHIVE_E}" "${ARCHIVE_B}" >/dev/null 2>&1; then
    printf 'Fly phase policy test: archive gateway accepted malformed A %s\n' \
      "${bad_a}" >&2
    exit 1
  fi
done

BAD_ARCHIVE_ENV="${TMP}/bad-archive-missing-pin.toml"
sed '/EXPECTED_ARCHIVE_BLOCK_HASH/d' "${ARCHIVE_GATEWAY_CONFIG}" \
  > "${BAD_ARCHIVE_ENV}"
if run_policy "${BAD_ARCHIVE_ENV}" zerone-2-open-beta-decision-v1 \
  zerone_1_archive_gateway >/dev/null 2>&1; then
  printf 'Fly phase policy test: archive gateway accepted a missing B pin\n' >&2
  exit 1
fi

if run_policy \
  "${ROOT}/deploy/query-gateway/fly.zerone-2.public.example.toml" \
  zerone-2-open-beta-decision-v1 zerone_2_gateway_public \
  replace-zerone-2-edge-app.internal - - - "${ARCHIVE_E}" "${ARCHIVE_B}" \
  >/dev/null 2>&1; then
  printf 'Fly phase policy test: zerone-2 gateway accepted archive hash context\n' >&2
  exit 1
fi

BAD_STRATEGY="${TMP}/bad-strategy.toml"
sed 's/strategy = "immediate"/strategy = "rolling"/' \
  "${SIGNER_CONFIG}" > "${BAD_STRATEGY}"
if run_policy "${BAD_STRATEGY}" zerone-2-cutover-decision-v1 \
  zerone_1_halt_signer - 100 101 102 >/dev/null 2>&1; then
  printf 'Fly phase policy test: rolling signer transition was accepted\n' >&2
  exit 1
fi

BAD_MOUNT="${TMP}/bad-mount.toml"
sed 's/source = "zerone_observer_data"/source = "zerone_data"/' \
  "${OBSERVER_CONFIG}" > "${BAD_MOUNT}"
if run_policy "${BAD_MOUNT}" zerone-2-cutover-decision-v1 \
  zerone_1_observer - 100 101 102 >/dev/null 2>&1; then
  printf 'Fly phase policy test: observer mounted signer volume\n' >&2
  exit 1
fi

for key_config in \
  "zerone_1_halt_signer|${SIGNER_CONFIG}" \
  "zerone_1_observer|${OBSERVER_CONFIG}"; do
  key=${key_config%%|*}
  config=${key_config##*|}
  for mutation in f a h; do
    bad_height="${TMP}/bad-${key}-${mutation}.toml"
    case "${mutation}" in
      f) sed 's/ZERONE_CHECKPOINT_STATE_HEIGHT = "100"/ZERONE_CHECKPOINT_STATE_HEIGHT = "99"/' \
        "${config}" > "${bad_height}" ;;
      a) sed 's/ZERONE_FINAL_COMMITTED_HEIGHT = "101"/ZERONE_FINAL_COMMITTED_HEIGHT = "99"/' \
        "${config}" > "${bad_height}" ;;
      h) sed 's/ZERONE_HALT_TRIGGER_HEIGHT = "102"/ZERONE_HALT_TRIGGER_HEIGHT = "99"/' \
        "${config}" > "${bad_height}" ;;
    esac
    if run_policy "${bad_height}" zerone-2-cutover-decision-v1 \
      "${key}" - 100 101 102 >/dev/null 2>&1; then
      printf 'Fly phase policy test: %s accepted mismatched %s height\n' \
        "${key}" "${mutation}" >&2
      exit 1
    fi
  done
done

printf 'Fly structural phase config policy tests: PASS\n'

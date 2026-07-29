#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
GATE="${ROOT}/deploy/fly-deploy-pinned.sh"
TMP_ROOT=$(mktemp -d "${TMPDIR:-/tmp}/zerone-fly-deploy-test.XXXXXX")
trap 'rm -rf "${TMP_ROOT}"' EXIT HUP INT TERM

DIGEST=$(printf 'a%.0s' {1..64})
IMAGE="registry.fly.io/zerone-2-edge@sha256:${DIGEST}"
VALID_CONFIG="${TMP_ROOT}/valid.toml"
FLY_LOG="${TMP_ROOT}/fly.log"
FAKE_BIN="${TMP_ROOT}/bin"
mkdir -p "${FAKE_BIN}"

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

cat > "${VALID_CONFIG}" <<EOF
app = "test-only"
[build]
  image = "${IMAGE}"
[env]
  NODE_ROLE = "edge"
EOF

cat > "${FAKE_BIN}/flyctl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
: "${FAKE_FLY_LOG:?}"
printf '%s\n' "$@" > "${FAKE_FLY_LOG}"
printf '%s|%s|%s\n' "${FLY_APP-unset}" "${FLY_APP_NAME-unset}" \
  "${FLY_CONFIG-unset}" > "${FAKE_FLY_LOG}.env"
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--config" ]; then
    shift
    cp "$1" "${FAKE_FLY_LOG}.config"
    break
  fi
  shift
done
EOF
chmod +x "${FAKE_BIN}/flyctl"

VALID_CONFIG_SHA=$(sha256_file "${VALID_CONFIG}")
[ "$("${GATE}" --check "${VALID_CONFIG}" test-only "${IMAGE}" edge "${VALID_CONFIG_SHA}")" = "${IMAGE}" ]
[ ! -e "${FLY_LOG}" ]

PATH="${FAKE_BIN}:${PATH}" FAKE_FLY_LOG="${FLY_LOG}" \
  FLY_APP=hostile-ambient-app FLY_CONFIG=/tmp/hostile.toml \
  "${GATE}" "${VALID_CONFIG}" test-only "${IMAGE}" edge "${VALID_CONFIG_SHA}"
EXPECTED_ARGS=$(cat <<EOF
deploy
--app
test-only
--config
CONFIG_SNAPSHOT
--image
${IMAGE}
EOF
)
ACTUAL_ARGS=$(awk '
  NR == 5 { print "CONFIG_SNAPSHOT"; next }
  { print }
' "${FLY_LOG}")
[ "${ACTUAL_ARGS}" = "${EXPECTED_ARGS}" ]
cmp -s "${VALID_CONFIG}" "${FLY_LOG}.config"
[ "$(cat "${FLY_LOG}.env")" = 'unset|unset|unset' ]

OTHER_DIGEST=$(printf 'b%.0s' {1..64})
if "${GATE}" --check "${VALID_CONFIG}" \
  test-only "registry.fly.io/zerone-2-edge@sha256:${OTHER_DIGEST}" edge \
  "${VALID_CONFIG_SHA}" \
  >/dev/null 2>&1; then
  printf 'expected signed image-reference mismatch rejection\n' >&2
  exit 1
fi
if "${GATE}" --check "${VALID_CONFIG}" test-only "${IMAGE}" validator "${VALID_CONFIG_SHA}" \
  >/dev/null 2>&1; then
  printf 'expected signed role mismatch rejection\n' >&2
  exit 1
fi
if "${GATE}" --check "${VALID_CONFIG}" test-only "${IMAGE}" edge \
  "${OTHER_DIGEST}" >/dev/null 2>&1; then
  printf 'expected signed config-hash mismatch rejection\n' >&2
  exit 1
fi

assert_rejected() {
  local name=$1
  local image=$2
  local config="${TMP_ROOT}/${name}.toml"
  printf 'app = "test-only"\n[build]\n  image = "%s"\n[env]\n  NODE_ROLE = "edge"\n' \
    "${image}" > "${config}"
  local config_sha
  config_sha=$(sha256_file "${config}")
  rm -f "${FLY_LOG}"
  if PATH="${FAKE_BIN}:${PATH}" FAKE_FLY_LOG="${FLY_LOG}" \
    "${GATE}" "${config}" test-only "${image}" edge "${config_sha}" >/dev/null 2>&1; then
    printf 'expected rejection: %s\n' "${name}" >&2
    exit 1
  fi
  [ ! -e "${FLY_LOG}" ] || {
    printf 'flyctl ran for rejected config: %s\n' "${name}" >&2
    exit 1
  }
}

assert_rejected placeholder 'replace-with-pinned-image-digest'
assert_rejected tag 'registry.fly.io/zerone-2-edge:latest'
assert_rejected tagged_digest "registry.fly.io/zerone-2-edge:v2@sha256:${DIGEST}"
assert_rejected short_digest 'registry.fly.io/zerone-2-edge@sha256:abc123'
assert_rejected uppercase_digest "registry.fly.io/zerone-2-edge@sha256:$(printf 'A%.0s' {1..64})"
assert_rejected uppercase_repository "Registry.fly.io/zerone-2-edge@sha256:${DIGEST}"

DUPLICATE_CONFIG="${TMP_ROOT}/duplicate.toml"
cat > "${DUPLICATE_CONFIG}" <<EOF
app = "test-only"
[build]
  image = "${IMAGE}"
  image = "${IMAGE}"
[env]
  NODE_ROLE = "edge"
EOF
if "${GATE}" --check "${DUPLICATE_CONFIG}" test-only "${IMAGE}" edge \
  "$(sha256_file "${DUPLICATE_CONFIG}")" >/dev/null 2>&1; then
  printf 'expected duplicate [build].image rejection\n' >&2
  exit 1
fi

OUTSIDE_CONFIG="${TMP_ROOT}/outside.toml"
cat > "${OUTSIDE_CONFIG}" <<EOF
app = "test-only"
[build]
  image = "${IMAGE}"
[env]
  NODE_ROLE = "edge"
  image = "${IMAGE}"
EOF
if "${GATE}" --check "${OUTSIDE_CONFIG}" test-only "${IMAGE}" edge \
  "$(sha256_file "${OUTSIDE_CONFIG}")" >/dev/null 2>&1; then
  printf 'expected image assignment outside [build] rejection\n' >&2
  exit 1
fi

SYMLINK_CONFIG="${TMP_ROOT}/symlink.toml"
ln -s "${VALID_CONFIG}" "${SYMLINK_CONFIG}"
if "${GATE}" --check "${SYMLINK_CONFIG}" test-only "${IMAGE}" edge \
  "${VALID_CONFIG_SHA}" >/dev/null 2>&1; then
  printf 'expected config symlink rejection\n' >&2
  exit 1
fi

# Every checked-in v1/v2 Fly profile must remain inert as committed, but a
# reviewed copy with its image replaced by a digest must pass the same gate.
PROFILES=(
  'deploy/mainnet/fly.toml|signer'
  'deploy/mainnet/fly.halt-signer.example.toml|signer'
  'deploy/mainnet/fly.observer.example.toml|observer'
  'deploy/mainnet/fly.archive-candidate.example.toml|archive-candidate'
  'deploy/mainnet/fly.archive.example.toml|archive'
  'deploy/networks/zerone-2/runtime/fly.validator.example.toml|validator'
  'deploy/networks/zerone-2/runtime/fly.edge.example.toml|edge'
  'deploy/networks/zerone-2/runtime/fly.edge.query-soak.example.toml|edge'
  'deploy/networks/zerone-2/runtime/fly.edge.public.example.toml|edge'
  'deploy/query-gateway/fly.zerone-2.private.example.toml|zerone-2-query'
  'deploy/query-gateway/fly.zerone-2.public.example.toml|zerone-2-query'
  'deploy/query-gateway/fly.zerone-1-archive.public.example.toml|zerone-1-archive-query'
)
for profile in "${PROFILES[@]}"; do
  relative="${profile%%|*}"
  role="${profile##*|}"
  source_config="${ROOT}/${relative}"
  app=$(awk -F'"' '/^app = "[a-z0-9-]+"$/ {print $2; exit}' "${source_config}")
  [ -n "${app}" ]
  source_sha=$(sha256_file "${source_config}")
  if "${GATE}" --check "${source_config}" "${app}" "${IMAGE}" "${role}" "${source_sha}" >/dev/null 2>&1; then
    printf 'checked-in placeholder unexpectedly deployable: %s\n' "${relative}" >&2
    exit 1
  fi
  reviewed_config="${TMP_ROOT}/$(basename "${relative}").reviewed.toml"
  sed -E "s|^([[:space:]]*image[[:space:]]*=[[:space:]]*)\"[^\"]+\"|\\1\"${IMAGE}\"|" \
    "${source_config}" > "${reviewed_config}"
  reviewed_sha=$(sha256_file "${reviewed_config}")
  [ "$("${GATE}" --check "${reviewed_config}" "${app}" "${IMAGE}" "${role}" "${reviewed_sha}")" = "${IMAGE}" ]
  case "${role}" in
    signer|observer|archive-candidate|archive|validator|edge)
      awk '
        /^\[/ { in_deploy = ($0 == "[deploy]") }
        in_deploy && /^[[:space:]]*strategy[[:space:]]*=[[:space:]]*"immediate"[[:space:]]*$/ { found = 1 }
        END { exit(found ? 0 : 1) }
      ' "${source_config}" || {
        printf 'stateful profile lacks immediate deployment strategy: %s\n' \
          "${relative}" >&2
        exit 1
      }
      ;;
  esac
done

# Public z2 edge exposes direct P2P only. Private edge and validator expose no
# Fly services at all.
PUBLIC_PROFILE="${ROOT}/deploy/networks/zerone-2/runtime/fly.edge.public.example.toml"
[ "$(grep -c '^\[\[services\]\]$' "${PUBLIC_PROFILE}")" -eq 1 ]
grep -q '^  internal_port = 26656$' "${PUBLIC_PROFILE}"
if grep -Eq '^  internal_port = (26657|1317|9090)$' "${PUBLIC_PROFILE}"; then
  printf 'public edge directly exposes a query service\n' >&2
  exit 1
fi
if grep -q '^\[http_service\]$' "${PUBLIC_PROFILE}"; then
  printf 'public edge has an unexpected http_service\n' >&2
  exit 1
fi
for private_profile in \
  "${ROOT}/deploy/networks/zerone-2/runtime/fly.edge.example.toml" \
  "${ROOT}/deploy/networks/zerone-2/runtime/fly.edge.query-soak.example.toml" \
  "${ROOT}/deploy/networks/zerone-2/runtime/fly.validator.example.toml" \
  "${ROOT}/deploy/query-gateway/fly.zerone-2.private.example.toml"; do
  if grep -Eq '^\[\[services\]\]$|^\[http_service\]$' "${private_profile}"; then
    printf 'private profile unexpectedly exposes a Fly service: %s\n' \
      "${private_profile}" >&2
    exit 1
  fi
done

printf 'fly pinned-image deploy gate tests: PASS\n'

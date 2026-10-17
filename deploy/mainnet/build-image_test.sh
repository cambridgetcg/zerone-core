#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
TMP=$(mktemp -d "${TMPDIR:-/tmp}/zerone-1-image-test.XXXXXX")
trap 'rm -rf "${TMP}"' EXIT
mkdir -p "${TMP}/bin"

fail() { printf 'mainnet build-image test: FAIL: %s\n' "$*" >&2; exit 1; }

cat > "${TMP}/bin/docker" <<'DOCKER'
#!/usr/bin/env bash
set -euo pipefail
context="${!#}"
[ -f "${context}/.sanitized-mainnet-context-v1" ]
find "${context}" -type f -print | LC_ALL=C sort > "${DOCKER_CONTEXT_FILES:?}"
cp "${context}/.sanitized-mainnet-context-v1" "${DOCKER_CONTEXT_MARKER:?}"
printf '%s\n' "$*" > "${DOCKER_ARGS:?}"
(cd "${context}" && GOPROXY=off CGO_ENABLED=0 \
  go build -trimpath -buildvcs=false -o "${DOCKER_COMPILE_OUT:?}" ./cmd/zeroned)
DOCKER
chmod +x "${TMP}/bin/docker"

DOCKER_CONTEXT_FILES="${TMP}/context-files" \
DOCKER_CONTEXT_MARKER="${TMP}/context-marker" \
DOCKER_ARGS="${TMP}/docker-args" \
DOCKER_COMPILE_OUT="${TMP}/zeroned-from-sanitized-context" \
PATH="${TMP}/bin:${PATH}" \
MAINNET_BUILD_PROFILE=development \
  "${ROOT}/deploy/mainnet/build-image.sh" zerone-1-context-test:local \
  > "${TMP}/build.log"
[ -x "${TMP}/zeroned-from-sanitized-context" ] || \
  fail "sanitized source context did not produce a zeroned binary"

for required in \
  '/.sanitized-mainnet-context-v1' '/Dockerfile' '/go.mod' '/go.sum' \
  '/internal/accountingmigration/context.go' \
  '/internal/survivalmigration/context.go' \
  '/internal/recordmigration/context.go' \
  '/internal/reviewmigration/context.go' \
  '/internal/claimrecordmigration/context.go' \
  '/internal/fundsettlementmigration/context.go' \
  '/internal/claimintegritymigration/context.go' \
  '/runtime/entrypoint.sh' '/public/genesis.json'; do
  grep -q "${required}$" "${TMP}/context-files" || fail "context omitted ${required}"
done
if grep -Eq '/(deploy|[.]git|build|scripts|prompts|cosmovisor|keyring|artifacts)/' \
  "${TMP}/context-files"; then
  sed -n '1,80p' "${TMP}/context-files" >&2
  fail "repository or custody directory entered the Docker context"
fi
if grep -Eq '([.]mnemonic|node_key[.]json|priv_validator_(key|state)[.]json)$' \
  "${TMP}/context-files"; then
  fail "custody file entered the Docker context"
fi
grep -q -- '--label money.zerone.build-profile=development' "${TMP}/docker-args" || \
  fail "development build was not visibly labelled"
grep -q -- '--platform linux/amd64' "${TMP}/docker-args" || \
  fail "mainnet image platform was not fixed to linux/amd64"
grep -q -- '--build-arg BUILD_PROFILE=development' "${TMP}/docker-args" || \
  fail "context profile was not bound into the Docker build"
grep -q -- '--label money.zerone.platform=linux/amd64' "${TMP}/docker-args" || \
  fail "image platform label is missing"
grep -q '^platform=linux/amd64$' "${TMP}/context-marker" || \
  fail "sanitized context marker omitted the fixed target platform"
grep -q '^release_tag=development$' "${TMP}/context-marker" || \
  fail "development context marker omitted its release status"
if grep -Eq '^COPY[[:space:]]+\.[[:space:]]+\.' "${ROOT}/deploy/mainnet/Dockerfile"; then
  fail "mainnet Dockerfile still uses broad COPY . ."
fi
grep -q '^COPY internal/accountingmigration ./internal/accountingmigration$' \
  "${ROOT}/deploy/mainnet/Dockerfile" || \
  fail "Dockerfile omits the accounting migration dependency from its builder"
grep -q '^COPY internal/reviewmigration ./internal/reviewmigration$' \
  "${ROOT}/deploy/mainnet/Dockerfile" || \
  fail "Dockerfile omits the review neutrality migration dependency from its builder"
grep -q '^COPY internal/claimrecordmigration ./internal/claimrecordmigration$' \
  "${ROOT}/deploy/mainnet/Dockerfile" || \
  fail "Dockerfile omits the claim records migration dependency from its builder"
grep -q '^COPY internal/fundsettlementmigration ./internal/fundsettlementmigration$' \
  "${ROOT}/deploy/mainnet/Dockerfile" || \
  fail "Dockerfile omits the fund settlement migration dependency from its builder"
grep -q '^COPY internal/claimintegritymigration ./internal/claimintegritymigration$' \
  "${ROOT}/deploy/mainnet/Dockerfile" || \
  fail "Dockerfile omits the claim integrity migration dependency from its builder"
grep -q '^COPY internal/recordmigration ./internal/recordmigration$' \
  "${ROOT}/deploy/mainnet/Dockerfile" || \
  fail "Dockerfile omits the record integrity migration dependency from its builder"
grep -q '^COPY internal/survivalmigration ./internal/survivalmigration$' \
  "${ROOT}/deploy/mainnet/Dockerfile" || \
  fail "Dockerfile omits the survival migration dependency from its builder"

# The sanitized context above compiled zeroned, so its internal packages are a
# sufficient set. Every other hand-written enumeration must name exactly that
# set: the Dockerfile builder, the mainnet release allowlist (which this test
# cannot compile without a signed tag), and the zerone-2 artifact-auditor
# allowlists, which only run in release mode. A new internal migration package
# then fails here instead of at a release ceremony.
internal_packages() {
  tr ' ' '\n' | { grep -E '^internal/[A-Za-z0-9_]+$' || true; } | LC_ALL=C sort -u
}
context_root=$(sed -n 's|/[.]sanitized-mainnet-context-v1$||p' "${TMP}/context-files")
[ -n "${context_root}" ] || fail "could not locate the sanitized context root"
compiled_internal=$(awk -v root="${context_root}/" '
  index($0, root) == 1 {
    rel = substr($0, length(root) + 1)
    if (split(rel, part, "/") > 2 && part[1] == "internal") print "internal/" part[2]
  }' "${TMP}/context-files" | LC_ALL=C sort -u)
[ -n "${compiled_internal}" ] || fail "compiled context contained no internal packages"
dockerfile_internal=$(sed -n 's|^COPY \(internal/[A-Za-z0-9_]*\) [.]/\1$|\1|p' \
  "${ROOT}/deploy/mainnet/Dockerfile" | LC_ALL=C sort -u)
if [ "${dockerfile_internal}" != "${compiled_internal}" ]; then
  printf 'compiled:\n%s\nDockerfile:\n%s\n' "${compiled_internal}" "${dockerfile_internal}" >&2
  fail "Dockerfile internal COPY set differs from the compiled context"
fi
for allowlist_script in \
  deploy/mainnet/build-image.sh \
  deploy/networks/zerone-2/runtime/build-image.sh \
  scripts/zerone-2-ceremony.sh; do
  allowlist_lines=0
  while IFS= read -r allowlist_line; do
    allowlist_lines=$((allowlist_lines + 1))
    listed_internal=$(printf '%s\n' "${allowlist_line}" | internal_packages)
    if [ "${listed_internal}" != "${compiled_internal}" ]; then
      printf 'compiled:\n%s\n%s:\n%s\n' "${compiled_internal}" \
        "${allowlist_script}" "${listed_internal}" >&2
      fail "${allowlist_script} source allowlist differs from the compiled context"
    fi
  done < <(grep -F ' docs/swagger-ui' "${ROOT}/${allowlist_script}")
  [ "${allowlist_lines}" -gt 0 ] || fail "${allowlist_script} has no source allowlist"
done
# shellcheck disable=SC2016
grep -q 'test "${TARGETOS}/${TARGETARCH}" = "linux/amd64"' \
  "${ROOT}/deploy/mainnet/Dockerfile" || fail "Dockerfile does not enforce linux/amd64"
# shellcheck disable=SC2016
grep -q 'GOOS="${TARGETOS}" GOARCH="${TARGETARCH}"' \
  "${ROOT}/deploy/mainnet/Dockerfile" || fail "Go target is not controlled by BuildKit"
# shellcheck disable=SC2016
grep -q 'grep -Fqx "commit=${COMMIT}"' "${ROOT}/deploy/mainnet/Dockerfile" || \
  fail "Dockerfile does not bind the source commit to its context marker"
# shellcheck disable=SC2016
grep -q 'grep -Fqx "release_signer_fingerprint=${RELEASE_SIGNER_FINGERPRINT}"' \
  "${ROOT}/deploy/mainnet/Dockerfile" || \
  fail "Dockerfile does not bind the release signer to its context marker"
grep -q 'verify-tag --raw' "${ROOT}/deploy/mainnet/build-image.sh" || \
  fail "release build does not verify an annotated tag signature"
grep -q 'ZERONE_AUTHORIZED_RELEASE_SIGNER_FINGERPRINT' \
  "${ROOT}/deploy/mainnet/build-image.sh" || \
  fail "release build does not require an authorized signer fingerprint"
# shellcheck disable=SC2016
grep -q 'materialize_git_file "${HEAD_COMMIT}" deploy/mainnet/entrypoint.sh' \
  "${ROOT}/deploy/mainnet/build-image.sh" || \
  fail "release entrypoint is not materialized from the signed Git object"
grep -q 'verify_clean_release_tree' "${ROOT}/deploy/mainnet/build-image.sh" || \
  fail "release worktree is not rechecked before Docker"
for provenance_script in \
  deploy/mainnet/build-image.sh \
  deploy/networks/zerone-2/runtime/build-image.sh \
  scripts/zerone-2-ceremony.sh; do
  script="${ROOT}/${provenance_script}"
  grep -q '^export GIT_NO_REPLACE_OBJECTS=1$' "${script}" || \
    fail "${provenance_script} does not disable Git replacement objects"
  grep -q 'refs/replace/' "${script}" || \
    fail "${provenance_script} does not reject replacement refs"
  grep -q 'info/grafts' "${script}" || \
    fail "${provenance_script} does not reject legacy grafts"
  grep -q -- '-c gpg.format=openpgp -c gpg.program=gpg -c gpg.openpgp.program=gpg' \
    "${script}" || \
    fail "${provenance_script} does not pin OpenPGP verification config"
done
grep -q '^  image = "REPLACE_WITH_PINNED_ZERONE_1_HALT_IMAGE_DIGEST"$' \
  "${ROOT}/deploy/mainnet/fly.toml" || fail "Fly config can still trigger a repository-root build"

printf 'mainnet build-image test: PASS (fake Docker; allowlisted context compiled)\n'

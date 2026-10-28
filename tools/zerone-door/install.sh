#!/usr/bin/env bash
# install.sh: copy a pinned build of zerone-door out of this checkout into the
# door's home (default ~/.zerone-agent/door), where launchd can run it. launchd
# jobs cannot read ~/Desktop (macOS TCC), so nothing the job runs or reads may
# live there.
#
# It builds from committed source only, runs the tests first, and records the
# commit and every sha256 in the release's MANIFEST.json. It never reads R's
# keyring or the Keychain, never overwrites an existing config.json or
# allowlist.json, and never copies the plist into ~/Library/LaunchAgents or
# loads it: enabling the job is a separate act by the household.
#
#   tools/zerone-door/install.sh --bun /abs/bun-1.3.5 \
#       --zeroned /abs/zeroned --zeroned-sha256 <64 hex> [--door-home DIR]
set -euo pipefail
umask 077

die() { printf 'install.sh: %s\n' "$1" >&2; exit 1; }

bun="" zeroned="" zeroned_sha="" door_home="${HOME}/.zerone-agent/door"
while [ $# -gt 0 ]; do
  case "$1" in
    --bun) bun="${2:-}"; shift 2 ;;
    --zeroned) zeroned="${2:-}"; shift 2 ;;
    --zeroned-sha256) zeroned_sha="${2:-}"; shift 2 ;;
    --door-home) door_home="${2:-}"; shift 2 ;;
    *) die "unknown argument: $1" ;;
  esac
done
[ -n "$bun" ] && [ -n "$zeroned" ] && [ -n "$zeroned_sha" ] || die "--bun, --zeroned and --zeroned-sha256 are required"
case "$bun$zeroned$door_home" in *[[:space:]]*) die "paths with spaces are not supported" ;; esac
for p in "$bun" "$zeroned" "$door_home"; do
  case "$p" in /*) ;; *) die "paths must be absolute: $p" ;; esac
done
case "$door_home/" in "$HOME/Desktop/"*) die "the door home must be outside ~/Desktop: launchd cannot read it there" ;; esac

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo="$(git -C "$script_dir" rev-parse --show-toplevel)"
commit="$(git -C "$repo" rev-parse HEAD)"
[ -z "$(git -C "$repo" status --porcelain -- tools/zerone-door sdk/typescript/src sdk/typescript/package-lock.json)" ] \
  || die "tools/zerone-door or the SDK source has uncommitted changes: install only builds committed source"

[ "$("$bun" --version)" = "1.3.5" ] || die "--bun must be Bun 1.3.5 exactly"
actual_sha="$(/usr/bin/shasum -a 256 "$zeroned" | awk '{print $1}')"
[ "$actual_sha" = "$zeroned_sha" ] || die "zeroned sha256 is $actual_sha, not the pinned $zeroned_sha"
[ -d "$repo/sdk/typescript/node_modules/@cosmjs/encoding" ] \
  || die "the SDK's dependencies are missing: run (cd sdk/typescript && npm ci --ignore-scripts) first"

printf '==> tests\n'
(cd "$repo/tools/zerone-door" && "$bun" test ./tests)

short="${commit:0:12}"
release="$door_home/releases/$short"
stage="$(mktemp -d "${TMPDIR:-/tmp}/zerone-door-build.XXXXXX")"
trap 'rm -rf "$stage"' EXIT
printf '==> build %s\n' "$short"
"$bun" build "$repo/tools/zerone-door/door.ts" --target=bun --outfile "$stage/zerone-door.js" >/dev/null
smoke="$("$bun" --no-env-file --config=/dev/null "$stage/zerone-door.js" verify \
  "$repo/tools/zerone-door/tests/fixtures/door-request.stale.txt" || true)"
case "$smoke" in *'"reasons":["request_stale"]'*) ;; *) die "the bundle failed its smoke check: $smoke" ;; esac
bundle_sha="$(/usr/bin/shasum -a 256 "$stage/zerone-door.js" | awk '{print $1}')"
bun_sha="$(/usr/bin/shasum -a 256 "$bun" | awk '{print $1}')"

mkdir -p "$door_home"
chmod 700 "$door_home"
for d in bin releases inbox processed replies state logs launchd; do
  mkdir -p "$door_home/$d"
  chmod 700 "$door_home/$d"
done

if [ -e "$release" ]; then
  [ "$(/usr/bin/shasum -a 256 "$release/zerone-door.js" | awk '{print $1}')" = "$bundle_sha" ] \
    || die "$release exists with a different bundle; move it aside first"
else
  mkdir "$release"
  cp "$stage/zerone-door.js" "$release/zerone-door.js"
  cat > "$release/MANIFEST.json" <<EOF
{
  "schema": "zerone-door-release/v1",
  "source_commit": "$commit",
  "bun_version": "1.3.5",
  "bun_sha256": "$bun_sha",
  "bundle_sha256": "$bundle_sha",
  "zeroned_sha256": "$zeroned_sha",
  "built_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
  chmod 444 "$release/zerone-door.js" "$release/MANIFEST.json"
  chmod 555 "$release"
fi

install_bin() {
  local src="$1" dest="$2" sha="$3"
  if [ -e "$dest" ]; then
    [ "$(/usr/bin/shasum -a 256 "$dest" | awk '{print $1}')" = "$sha" ] || die "$dest exists with different contents"
  else
    cp "$src" "$dest.tmp.$$"
    chmod 555 "$dest.tmp.$$"
    mv "$dest.tmp.$$" "$dest"
  fi
}
install_bin "$bun" "$door_home/bin/bun-1.3.5" "$bun_sha"
zeroned_dest="$door_home/bin/zeroned-${zeroned_sha:0:16}"
install_bin "$zeroned" "$zeroned_dest" "$zeroned_sha"

ln -sfhn "releases/$short" "$door_home/current"

if [ ! -e "$door_home/config.json" ]; then
  ZD_EXAMPLE="$repo/tools/zerone-door/config.example.json" ZD_OUT="$door_home/config.json" \
  ZD_ZERONED="$zeroned_dest" ZD_SHA="$zeroned_sha" "$bun" --no-env-file -e '
    const fs = require("node:fs");
    const c = JSON.parse(fs.readFileSync(process.env.ZD_EXAMPLE, "utf8"));
    c.zeroned = { path: process.env.ZD_ZERONED, sha256: process.env.ZD_SHA };
    fs.writeFileSync(process.env.ZD_OUT, JSON.stringify(c, null, 2) + "\n", { mode: 0o600 });
  '
  printf '==> wrote %s from the template: fill every <...> before the first run\n' "$door_home/config.json"
else
  printf '==> kept the existing %s (check its zeroned path and sha256 match %s)\n' "$door_home/config.json" "$zeroned_dest"
fi
if [ ! -e "$door_home/allowlist.json" ]; then
  cp "$repo/tools/zerone-door/allowlist.example.json" "$door_home/allowlist.json"
  chmod 600 "$door_home/allowlist.json"
fi

plist="$door_home/launchd/com.zerone.door.plist"
sed -e "s#@DOOR_HOME@#$door_home#g" -e "s#@HOME@#$HOME#g" \
  "$repo/tools/zerone-door/launchd/com.zerone.door.plist.template" > "$plist"
/usr/bin/plutil -lint "$plist" >/dev/null || die "the rendered plist does not lint"

cat <<EOF
==> installed zerone-door $short in $door_home (dry run; nothing is loaded)
Next, by the household's hand:
  1. Fill $door_home/config.json and $door_home/allowlist.json.
  2. Put R's keyring passphrase in the login Keychain under the service named in
     config.json (security add-generic-password ... -w with no value prompts for it).
  3. One dry pass by hand:
       $door_home/bin/bun-1.3.5 --no-env-file --config=/dev/null $door_home/current/zerone-door.js run --once --config $door_home/config.json
  4. Only then, to enable the job:
       cp $plist ~/Library/LaunchAgents/com.zerone.door.plist
       launchctl bootstrap gui/\$(id -u) ~/Library/LaunchAgents/com.zerone.door.plist
EOF

#!/bin/sh
set -eu

: "${DAEMON_HOME:?DAEMON_HOME must be set}"
: "${DAEMON_NAME:?DAEMON_NAME must be set}"
: "${DAEMON_BINARY_SHA256:?DAEMON_BINARY_SHA256 must be set from the authenticated release manifest}"
: "${DAEMON_GENESIS_BINARY_SHA256:?DAEMON_GENESIS_BINARY_SHA256 must bind the original genesis binary}"
: "${DAEMON_CURRENT_BINARY_SHA256:?DAEMON_CURRENT_BINARY_SHA256 must bind the authorized predecessor binary}"

if [ "${DAEMON_NAME}" != "zeroned" ]; then
  echo "validator entrypoint: DAEMON_NAME is image-frozen to zeroned" >&2
  exit 1
fi
case "${DAEMON_HOME}" in
  /*) ;;
  *)
    echo "validator entrypoint: DAEMON_HOME must be an absolute path" >&2
    exit 1
    ;;
esac
case "/${DAEMON_HOME#/}/" in
  *"//"* | *"/./"* | *"/../"*)
    echo "validator entrypoint: DAEMON_HOME must be a canonical, non-root path" >&2
    exit 1
    ;;
esac
if [ -L "${DAEMON_HOME}" ] || { [ -e "${DAEMON_HOME}" ] && [ ! -d "${DAEMON_HOME}" ]; }; then
  echo "validator entrypoint: DAEMON_HOME must be a regular non-symlink directory" >&2
  exit 1
fi

require_policy_value() {
  variable_name="$1"
  expected_value="$2"
  actual_value="$3"
  if [ "${actual_value}" != "${expected_value}" ]; then
    echo "validator entrypoint: ${variable_name} must be image-frozen to ${expected_value}" >&2
    exit 1
  fi
}
require_policy_value DAEMON_ALLOW_DOWNLOAD_BINARIES false "${DAEMON_ALLOW_DOWNLOAD_BINARIES:-}"
require_policy_value DAEMON_DOWNLOAD_MUST_HAVE_CHECKSUM true "${DAEMON_DOWNLOAD_MUST_HAVE_CHECKSUM:-}"
require_policy_value DAEMON_RESTART_AFTER_UPGRADE true "${DAEMON_RESTART_AFTER_UPGRADE:-}"
require_policy_value UNSAFE_SKIP_BACKUP false "${UNSAFE_SKIP_BACKUP:-}"
if [ "$#" -ne 2 ] || [ "$1" != "run" ] || [ "$2" != "start" ]; then
  echo "validator entrypoint: runtime command is image-frozen to 'cosmovisor run start'" >&2
  exit 1
fi

require_sha256() {
  variable_name="$1"
  digest="$2"
  if [ "${#digest}" -ne 64 ]; then
    echo "validator entrypoint: ${variable_name} must be exactly 64 lowercase hexadecimal characters" >&2
    exit 1
  fi
  case "${digest}" in
    *[!0-9a-f]*)
      echo "validator entrypoint: ${variable_name} must be exactly 64 lowercase hexadecimal characters" >&2
      exit 1
      ;;
  esac
}
require_sha256 DAEMON_BINARY_SHA256 "${DAEMON_BINARY_SHA256}"
require_sha256 DAEMON_GENESIS_BINARY_SHA256 "${DAEMON_GENESIS_BINARY_SHA256}"
require_sha256 DAEMON_CURRENT_BINARY_SHA256 "${DAEMON_CURRENT_BINARY_SHA256}"

require_upgrade_name() {
  variable_name="$1"
  name="$2"
  [ -z "${name}" ] && return
  case "${name}" in
    *[!a-z0-9._-]* | "." | "..")
      echo "validator entrypoint: ${variable_name} must be canonical lowercase ASCII and must not be a dot segment" >&2
      exit 1
      ;;
  esac
  case "${name}" in
    [a-z0-9] | [a-z0-9]*[a-z0-9]) ;;
    *)
      echo "validator entrypoint: ${variable_name} must start and end with a lowercase letter or digit" >&2
      exit 1
      ;;
  esac
}

source_binary="/usr/local/bin/${DAEMON_NAME}"
supervisor_binary="/usr/local/bin/cosmovisor"
genesis_binary="${DAEMON_HOME}/cosmovisor/genesis/bin/${DAEMON_NAME}"
upgrade_name="${DAEMON_UPGRADE_NAME:-}"
current_upgrade_name="${DAEMON_CURRENT_UPGRADE_NAME:-}"
current_link="${DAEMON_HOME}/cosmovisor/current"
upgrades_dir="${DAEMON_HOME}/cosmovisor/upgrades"

require_upgrade_name DAEMON_UPGRADE_NAME "${upgrade_name}"
require_upgrade_name DAEMON_CURRENT_UPGRADE_NAME "${current_upgrade_name}"
if [ -n "${upgrade_name}" ] && [ "${upgrade_name}" = "${current_upgrade_name}" ]; then
  echo "validator entrypoint: pending and current upgrade names must differ" >&2
  exit 1
fi

for managed_directory in \
  "${DAEMON_HOME}/cosmovisor" \
  "${DAEMON_HOME}/cosmovisor/genesis" \
  "${DAEMON_HOME}/cosmovisor/genesis/bin" \
  "${DAEMON_HOME}/cosmovisor/upgrades"; do
  if [ -L "${managed_directory}" ]; then
    echo "validator entrypoint: refusing mutable Cosmovisor directory symlink: ${managed_directory}" >&2
    exit 1
  fi
done
for persisted_directory in "${DAEMON_HOME}/config" "${DAEMON_HOME}/data"; do
  if [ -L "${persisted_directory}" ]; then
    echo "validator entrypoint: refusing persisted validator-directory symlink: ${persisted_directory}" >&2
    exit 1
  fi
done
if [ -L "${genesis_binary}" ]; then
  echo "validator entrypoint: refusing mutable genesis binary symlink: ${genesis_binary}" >&2
  exit 1
fi
if [ -e "${genesis_binary}" ] &&
  { [ ! -f "${genesis_binary}" ] || [ ! -x "${genesis_binary}" ]; }; then
  echo "validator entrypoint: genesis binary must be an executable regular file: ${genesis_binary}" >&2
  exit 1
fi

if [ ! -f "${source_binary}" ] || [ ! -x "${source_binary}" ] || [ -L "${source_binary}" ]; then
  echo "validator entrypoint: image binary must be an executable regular non-symlink file: ${source_binary}" >&2
  exit 1
fi
if [ ! -f "${supervisor_binary}" ] || [ ! -x "${supervisor_binary}" ] || [ -L "${supervisor_binary}" ]; then
  echo "validator entrypoint: Cosmovisor must be an executable regular non-symlink image binary" >&2
  exit 1
fi

source_digest="$(sha256sum "${source_binary}" | awk '{print $1}')"
if [ "${source_digest}" != "${DAEMON_BINARY_SHA256}" ]; then
  echo "validator entrypoint: image binary SHA-256 mismatch: expected ${DAEMON_BINARY_SHA256}, got ${source_digest}" >&2
  exit 1
fi

verify_binary_digest() {
  binary_path="$1"
  expected_digest="$2"
  role="$3"

  if [ -L "${binary_path}" ] ||
    [ ! -f "${binary_path}" ] ||
    [ ! -x "${binary_path}" ]; then
    echo "validator entrypoint: ${role} binary must be an executable regular non-symlink file: ${binary_path}" >&2
    exit 1
  fi
  actual_digest="$(sha256sum "${binary_path}" | awk '{print $1}')"
  if [ "${actual_digest}" != "${expected_digest}" ]; then
    echo "validator entrypoint: ${role} binary SHA-256 mismatch: expected ${expected_digest}, got ${actual_digest}" >&2
    exit 1
  fi
}

install_verified_binary() {
  destination="$1"
  expected_digest="$2"
  role="$3"
  destination_dir="$(dirname "${destination}")"

  if [ "${source_digest}" != "${expected_digest}" ]; then
    echo "validator entrypoint: image payload does not match the authorized ${role} digest" >&2
    exit 1
  fi
  if [ -L "${destination}" ]; then
    echo "validator entrypoint: refusing mutable ${role} binary symlink: ${destination}" >&2
    exit 1
  fi
  if [ -e "${destination}" ]; then
    if [ ! -f "${destination}" ] || [ ! -x "${destination}" ]; then
      echo "validator entrypoint: ${role} binary must be an executable regular file: ${destination}" >&2
      exit 1
    fi
    installed_digest="$(sha256sum "${destination}" | awk '{print $1}')"
    if [ "${installed_digest}" = "${expected_digest}" ]; then
      return
    fi
    echo "validator entrypoint: persisted ${role} binary digest ${installed_digest} differs from the authorized manifest digest; refusing artifact replacement" >&2
    exit 1
  fi

  if [ -L "${destination_dir}" ]; then
    echo "validator entrypoint: refusing mutable ${role} binary directory symlink: ${destination_dir}" >&2
    exit 1
  fi
  mkdir -p "${destination_dir}"
  temporary="$(mktemp "${destination_dir}/.${DAEMON_NAME}.tmp.XXXXXX")"
  if ! install -m 0755 "${source_binary}" "${temporary}"; then
    rm -f "${temporary}"
    echo "validator entrypoint: failed to stage ${role} binary" >&2
    exit 1
  fi
  staged_digest="$(sha256sum "${temporary}" | awk '{print $1}')"
  if [ "${staged_digest}" != "${expected_digest}" ]; then
    rm -f "${temporary}"
    echo "validator entrypoint: staged ${role} binary SHA-256 mismatch" >&2
    exit 1
  fi
  if ! mv -f "${temporary}" "${destination}"; then
    rm -f "${temporary}"
    echo "validator entrypoint: atomic ${role} binary installation failed" >&2
    exit 1
  fi
  chmod 0755 "${destination}"
  installed_digest="$(sha256sum "${destination}" | awk '{print $1}')"
  if [ "${installed_digest}" != "${expected_digest}" ]; then
    echo "validator entrypoint: installed ${role} binary failed post-install digest verification" >&2
    exit 1
  fi
}

if [ -e "${genesis_binary}" ] || [ -L "${genesis_binary}" ]; then
  # The genesis artifact is the immutable root of the whole local lineage. It
  # is checked before any staging mutation on every boot, including later
  # upgrades whose currently selected binary is no longer genesis.
  verify_binary_digest \
    "${genesis_binary}" \
    "${DAEMON_GENESIS_BINARY_SHA256}" \
    "genesis"
  genesis_present=true
else
  genesis_present=false
fi

selected_binary="${genesis_binary}"
selected_digest="${DAEMON_CURRENT_BINARY_SHA256}"
selected_role="current genesis"
current_target="genesis"

if [ "${genesis_present}" = false ]; then
  if [ -n "${upgrade_name}" ] ||
    [ -n "${current_upgrade_name}" ] ||
    [ "${DAEMON_BINARY_SHA256}" != "${DAEMON_GENESIS_BINARY_SHA256}" ] ||
    [ "${DAEMON_BINARY_SHA256}" != "${DAEMON_CURRENT_BINARY_SHA256}" ]; then
    echo "validator entrypoint: an empty Cosmovisor home requires identical image, genesis, and current digests with no upgrade names" >&2
    exit 1
  fi
  if [ -e "${current_link}" ] || [ -L "${current_link}" ]; then
    echo "validator entrypoint: an empty Cosmovisor home must not contain a current selector" >&2
    exit 1
  fi
  if [ -d "${DAEMON_HOME}/data" ] &&
    [ -n "$(find "${DAEMON_HOME}/data" -mindepth 1 -maxdepth 1 -print -quit)" ]; then
    echo "validator entrypoint: populated application data has no pinned Cosmovisor genesis binary; refusing implicit early activation" >&2
    exit 1
  fi
  install_verified_binary \
    "${genesis_binary}" \
    "${DAEMON_GENESIS_BINARY_SHA256}" \
    "genesis"
else
  if [ -L "${current_link}" ]; then
    current_target="$(readlink "${current_link}")"
    case "${current_target}" in
      genesis)
        if [ -n "${current_upgrade_name}" ]; then
          echo "validator entrypoint: current selector names genesis but DAEMON_CURRENT_UPGRADE_NAME is set" >&2
          exit 1
        fi
        selected_binary="${genesis_binary}"
        selected_digest="${DAEMON_CURRENT_BINARY_SHA256}"
        selected_role="current genesis"
        ;;
      "upgrades/${current_upgrade_name}")
        if [ -z "${current_upgrade_name}" ]; then
          echo "validator entrypoint: current selector has no authorized predecessor upgrade name" >&2
          exit 1
        fi
        current_upgrade_dir="${upgrades_dir}/${current_upgrade_name}"
        if [ -L "${upgrades_dir}" ] ||
          [ -L "${current_upgrade_dir}" ] ||
          [ -L "${current_upgrade_dir}/bin" ]; then
          echo "validator entrypoint: refusing current Cosmovisor upgrade-directory symlink" >&2
          exit 1
        fi
        selected_binary="${current_upgrade_dir}/bin/${DAEMON_NAME}"
        selected_digest="${DAEMON_CURRENT_BINARY_SHA256}"
        selected_role="current predecessor"
        ;;
      "upgrades/${upgrade_name}")
        if [ -z "${upgrade_name}" ]; then
          echo "validator entrypoint: current selector has no authorized pending upgrade name" >&2
          exit 1
        fi
        selected_binary="${upgrades_dir}/${upgrade_name}/bin/${DAEMON_NAME}"
        selected_digest="${DAEMON_BINARY_SHA256}"
        selected_role="activated pending upgrade"
        ;;
      *)
        echo "validator entrypoint: current selector must be exactly genesis, upgrades/DAEMON_CURRENT_UPGRADE_NAME, or upgrades/DAEMON_UPGRADE_NAME" >&2
        exit 1
        ;;
    esac
  elif [ -e "${current_link}" ]; then
    echo "validator entrypoint: Cosmovisor current selector must be a relative symlink" >&2
    exit 1
  elif [ -n "${current_upgrade_name}" ]; then
    echo "validator entrypoint: current selector may be absent only while genesis is the authorized predecessor" >&2
    exit 1
  fi

  if [ "${current_target}" != "upgrades/${upgrade_name}" ]; then
    verify_binary_digest "${selected_binary}" "${selected_digest}" "${selected_role}"
  fi
fi

if [ -z "${upgrade_name}" ]; then
  if [ "${DAEMON_BINARY_SHA256}" != "${DAEMON_CURRENT_BINARY_SHA256}" ]; then
    echo "validator entrypoint: without a pending upgrade, the image payload must match the authorized current binary" >&2
    exit 1
  fi
else
  upgrade_dir="${upgrades_dir}/${upgrade_name}"
  if [ -L "${upgrades_dir}" ] ||
    [ -L "${upgrade_dir}" ] ||
    [ -L "${upgrade_dir}/bin" ]; then
    echo "validator entrypoint: refusing mutable Cosmovisor upgrade-directory symlink" >&2
    exit 1
  fi
  upgrade_binary="${upgrade_dir}/bin/${DAEMON_NAME}"
  install_verified_binary \
    "${upgrade_binary}" \
    "${DAEMON_BINARY_SHA256}" \
    "upgrade"
fi

# Re-read the selector after staging and bind the exact selected artifact
# before handing control to Cosmovisor. No absolute, traversing, or
# operator-chosen selector is accepted.
if [ -L "${current_link}" ]; then
  final_target="$(readlink "${current_link}")"
else
  final_target="genesis"
fi
case "${final_target}" in
  genesis)
    final_binary="${genesis_binary}"
    final_digest="${DAEMON_CURRENT_BINARY_SHA256}"
    final_role="selected genesis"
    ;;
  "upgrades/${current_upgrade_name}")
    if [ -z "${current_upgrade_name}" ]; then
      echo "validator entrypoint: final selector does not name an authorized predecessor" >&2
      exit 1
    fi
    if [ -L "${upgrades_dir}" ] ||
      [ -L "${upgrades_dir}/${current_upgrade_name}" ] ||
      [ -L "${upgrades_dir}/${current_upgrade_name}/bin" ]; then
      echo "validator entrypoint: final predecessor path contains a directory symlink" >&2
      exit 1
    fi
    final_binary="${upgrades_dir}/${current_upgrade_name}/bin/${DAEMON_NAME}"
    final_digest="${DAEMON_CURRENT_BINARY_SHA256}"
    final_role="selected predecessor"
    ;;
  "upgrades/${upgrade_name}")
    if [ -z "${upgrade_name}" ]; then
      echo "validator entrypoint: final selector does not name an authorized pending upgrade" >&2
      exit 1
    fi
    if [ -L "${upgrades_dir}" ] ||
      [ -L "${upgrades_dir}/${upgrade_name}" ] ||
      [ -L "${upgrades_dir}/${upgrade_name}/bin" ]; then
      echo "validator entrypoint: final pending path contains a directory symlink" >&2
      exit 1
    fi
    final_binary="${upgrades_dir}/${upgrade_name}/bin/${DAEMON_NAME}"
    final_digest="${DAEMON_BINARY_SHA256}"
    final_role="selected activated upgrade"
    ;;
  *)
    echo "validator entrypoint: final current selector left the authenticated binary lineage" >&2
    exit 1
    ;;
esac
verify_binary_digest "${final_binary}" "${final_digest}" "${final_role}"

exec "${supervisor_binary}" "$@"

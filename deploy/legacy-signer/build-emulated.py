#!/usr/bin/env python3
"""Reproduce the legacy signer binary (2e37c4c + CometBFT 0.38.25) on a non-native Docker host.

Written by an agent (Claude Code, delegate of session 8d4c201e) on 2026-11-06.

This is a thin wrapper around the repository's own
deploy/legacy-observer/observer-build.py. It reuses that recipe's source
authentication, patch application, builder image, environment, build script and
receipt schema unchanged. It changes only these things so the recipe can run on
a Darwin/arm64 Studio:

1. The Docker socket is an explicit literal unix:// path (Colima), not
   /var/run/docker.sock.
2. The native-Linux-amd64 platform check is bypassed. The container still runs
   --platform linux/amd64 (Colima vz + Rosetta), and the receipt records the
   emulation instead of the recipe's hard-coded "linux/amd64" native claim.
3. With --tmpfs-tmp, the container's /tmp is a tmpfs instead of the recipe's
   bind mount. Go's $WORK paths never reach the executable under -trimpath.

Because emulation is not the recorded native host, a build here only counts if it
is byte-identical to the signed 20260909.2 observer executable (pass
--expected-sha256). It starts no node, pushes nothing and uses no registry
credential (run it with DOCKER_CONFIG pointing at an empty config).
"""
from __future__ import annotations

import argparse
import importlib.util
import json
import os
from pathlib import Path
import re
import sys
import types


def load_recipe(repository: Path):
    path = repository / "deploy/legacy-observer/observer-build.py"
    spec = importlib.util.spec_from_file_location("zerone_observer_build", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module, path


def main() -> int:
    os.umask(0o077)
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--repository", required=True, type=Path, help="zerone-core checkout holding the recipe")
    parser.add_argument("--source-archive", required=True, type=Path)
    parser.add_argument("--dependency-patch", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--docker-host", required=True, help="literal unix:///absolute/socket")
    parser.add_argument("--expected-sha256", required=True,
                        help="the signed release executable digest; an emulated build counts only if it matches")
    parser.add_argument("--tmpfs-tmp", action="store_true",
                        help="use a container tmpfs for /tmp instead of the recipe's bind mount")
    args = parser.parse_args()

    if not re.fullmatch(r"unix:///[A-Za-z0-9._/-]+", args.docker_host) or "/../" in args.docker_host:
        print(json.dumps({"result": "REFUSED", "reason": "docker host must be a literal unix:/// path"}), file=sys.stderr)
        return 1

    recipe, recipe_path = load_recipe(args.repository.resolve())
    original_argv = recipe.build_argv

    def build_argv(output):
        argv = original_argv(output)
        index = argv.index("unix:///var/run/docker.sock")
        argv[index] = args.docker_host
        if args.tmpfs_tmp:
            # Colima's virtiofs bind mount intermittently refused reads of the Go
            # $WORK importcfg files ("permission denied") on 2026-11-06.
            # $WORK paths never reach the executable under -trimpath, so a
            # tmpfs /tmp does not change the output bytes.
            mount = f"type=bind,src={output}/tmp,dst=/tmp"
            index = argv.index(mount)
            argv[index - 1:index + 1] = ["--tmpfs", "/tmp:rw,nosuid,nodev,exec,size=4g"]
        return argv

    recipe.build_argv = build_argv
    # The recipe refuses anything but native Linux amd64. Emulation is declared
    # in the receipt below instead.
    recipe.platform = types.SimpleNamespace(system=lambda: "Linux", machine=lambda: "x86_64")

    try:
        report = recipe.build(args.source_archive, args.dependency_patch, args.output, args.expected_sha256)
    except (OSError, ValueError) as error:
        print(json.dumps({"result": "REFUSED", "reason": str(error)}), file=sys.stderr)
        return 1

    report["native_platform"] = None
    report["execution_platform"] = {
        "container_platform": "linux/amd64",
        "host": "darwin/arm64 Mac Studio",
        "virtualisation": "Colima (Lima, macOS Virtualization.framework) with Rosetta amd64 translation",
        "docker_host": args.docker_host,
        "note": "Not the recipe's native Linux amd64 host. Counts only through byte identity with the expected executable.",
    }
    report["recipe_file"] = str(recipe_path.relative_to(args.repository.resolve()))
    report["recipe_file_sha256"] = recipe.base.sha256(recipe_path)
    report["wrapper"] = "deploy/legacy-signer/build-emulated.py (socket, platform-check and optional tmpfs override only)"
    report["tmp_mount"] = "tmpfs" if args.tmpfs_tmp else "bind (recipe default)"
    report["docker_argv"] = build_argv(args.output.expanduser().absolute())
    output = args.output.expanduser().absolute()
    (output / "provenance.json").write_text(json.dumps(report, indent=2, sort_keys=True) + "\n")
    print(json.dumps({k: report[k] for k in ("passed", "source_status", "executable_sha256", "executable_bytes",
                                             "matches_expected_executable")}, sort_keys=True))
    return 0


if __name__ == "__main__":
    sys.exit(main())

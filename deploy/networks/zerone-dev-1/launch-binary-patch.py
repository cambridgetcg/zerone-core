#!/usr/bin/env python3
"""Boot handoff to an explicitly pinned binary patch of an applied upgrade; never resets a home.

Both pins come from the caller (the machine's launch arguments), not from the volume.
The first boot stages the patch once; every boot then hands the node to run-upgrade
with both pins. The launcher never removes, repairs or retries a staged directory.
"""
import argparse
import os
from pathlib import Path
import re
import subprocess
import sys

PATCH_PREFIX = "binary-patch-"  # runtime.py's staged directory, named by the external patch pin.


def absolute_path(value):
    path = Path(value)
    if not path.is_absolute() or ".." in path.parts:
        raise argparse.ArgumentTypeError("An absolute path without parent traversal is required.")
    return path


def sha256_pin(value):
    if not re.fullmatch(r"[0-9a-f]{64}", value) or value == "0" * 64:
        raise argparse.ArgumentTypeError("An exact external nonzero lowercase SHA256 is required.")
    return value


def main():
    parser = argparse.ArgumentParser(description=__doc__, allow_abbrev=False)
    parser.add_argument("--home", required=True, type=absolute_path)
    parser.add_argument("--runtime", required=True, type=absolute_path)
    parser.add_argument("--upgrade-packet-sha256", required=True, type=sha256_pin)
    parser.add_argument("--patch-packet", required=True, type=absolute_path)
    parser.add_argument("--patch-packet-sha256", required=True, type=sha256_pin)
    parser.add_argument("--patch-binary", required=True, type=absolute_path)
    args = parser.parse_args()
    if args.patch_packet_sha256 == args.upgrade_packet_sha256:
        parser.error("The patch and upgrade pins must name different packets.")
    interpreter = sys.executable
    runtime = [interpreter, "-I", "-B", str(args.runtime)]
    # lexists includes a partial directory, ordinary file and broken symlink.
    # Every existing form goes to the runtime's strict load_patch path.
    if not os.path.lexists(args.home / (PATCH_PREFIX + args.patch_packet_sha256)):
        staged = subprocess.run(runtime + [
            "stage-patch", "--home", str(args.home),
            "--upgrade-packet-sha256", args.upgrade_packet_sha256,
            "--patch-packet", str(args.patch_packet),
            "--patch-packet-sha256", args.patch_packet_sha256,
            "--binary", str(args.patch_binary),
        ], check=False)
        if staged.returncode != 0:
            # A signalled staging child also cannot progress to node launch.
            return staged.returncode if staged.returncode > 0 else 128 - staged.returncode
    # The runtime takes its own home lock and rechecks both external pins, the
    # staged upgrade and patch, the first applied observation and the controls.
    # exec leaves signal handling and the node lifecycle with that supervisor.
    os.execv(interpreter, runtime + [
        "run-upgrade", "--home", str(args.home),
        "--upgrade-packet-sha256", args.upgrade_packet_sha256,
        "--patch-packet-sha256", args.patch_packet_sha256,
    ])


if __name__ == "__main__":
    raise SystemExit(main())

#!/usr/bin/env python3
"""Check that a swap image differs from its base only in /usr/local/bin/zeroned.

Written by an agent (Claude Code, delegate of session 8d4c201e) on 2026-11-06.

Both images must already be in the local Docker store. Nothing is pulled,
pushed or published, and no node is started: zeroned runs only as
`version --long` in a throwaway home with --network none. File contents under
the seed directory (which holds key material in the real signer images) are
never printed: the script prints names, sizes, modes and equal/differs only.

Exit status 0 means every check passed; 1 means at least one failed.
"""
from __future__ import annotations

import argparse
import hashlib
import io
import json
import re
import subprocess
import sys
import tarfile
import tempfile
from pathlib import Path

CONFIG_FIELDS = ("Entrypoint", "Cmd", "Env", "WorkingDir", "ExposedPorts", "User", "Volumes",
                 "Labels", "StopSignal", "Shell", "OnBuild", "Healthcheck")
ALLOWED_LAYER_DIRS = {"usr", "usr/local", "usr/local/bin"}
TARGET = "usr/local/bin/zeroned"
FS_DIGEST = r"""set -eu
find / -xdev \( -path /proc -o -path /sys -o -path /dev -o -path /etc/hostname -o -path /etc/hosts \
  -o -path /etc/resolv.conf -o -path /usr/local/bin/zeroned \) -prune -o \( -type f -o -type l \) -print0 \
  | sort -z | xargs -0 -r stat -c '%n %s %a %u %g' | sha256sum | cut -d' ' -f1
find / -xdev \( -path /proc -o -path /sys -o -path /dev -o -path /etc/hostname -o -path /etc/hosts \
  -o -path /etc/resolv.conf -o -path /usr/local/bin/zeroned \) -prune -o -type f -print0 \
  | sort -z | xargs -0 -r sha256sum | sha256sum | cut -d' ' -f1
"""


class Checks:
    def __init__(self):
        self.results = []

    def add(self, name, ok, detail=""):
        self.results.append({"check": name, "ok": bool(ok), "detail": detail})
        print(("PASS " if ok else "FAIL ") + name + (f": {detail}" if detail else ""))

    @property
    def passed(self):
        return all(r["ok"] for r in self.results)


def docker(host, *args, capture=True, input_bytes=None):
    result = subprocess.run(["docker", "--host", host, *args], input=input_bytes,
                            stdout=subprocess.PIPE if capture else None, stderr=subprocess.PIPE, check=False)
    if result.returncode != 0:
        raise RuntimeError(f"docker {' '.join(args[:3])} failed: {result.stderr.decode(errors='replace')[-400:]}")
    return result.stdout


def inspect(host, ref):
    return json.loads(docker(host, "image", "inspect", ref))[0]


def run(host, image, entrypoint, *argv):
    return docker(host, "run", "--rm", "--network", "none", "--platform", "linux/amd64",
                  "--entrypoint", entrypoint, image, *argv).decode()


def top_layer_members(host, ref, diff_id):
    """Return (members, zeroned bytes) of the layer whose uncompressed digest is diff_id."""
    with tempfile.TemporaryDirectory(prefix="swap-verify-") as tmp:
        archive = Path(tmp) / "image.tar"
        with archive.open("wb") as out:
            subprocess.run(["docker", "--host", host, "save", ref], stdout=out, check=True)
        with tarfile.open(archive) as outer:
            want = diff_id.split(":", 1)[1]
            # docker save writes manifest.json (docker-archive) beside an OCI
            # layout; its last Layers entry is the top layer, possibly gzipped.
            manifest = json.load(outer.extractfile("manifest.json"))
            require_one = [m for m in manifest if ref in (m.get("RepoTags") or [])] or manifest
            last = require_one[0]["Layers"][-1]
            raw = outer.extractfile(last).read()
            if raw[:2] == b"\x1f\x8b":
                import gzip
                raw = gzip.decompress(raw)
            if hashlib.sha256(raw).hexdigest() == want:
                body = io.BytesIO(raw)
                with tarfile.open(fileobj=body) as layer:
                    members = []
                    data = None
                    for item in layer.getmembers():
                        members.append({"name": item.name.rstrip("/"), "type": item.type.decode() if isinstance(item.type, bytes) else str(item.type),
                                        "mode": oct(item.mode), "uid": item.uid, "gid": item.gid, "size": item.size})
                        if item.name == TARGET and item.isfile():
                            data = layer.extractfile(item).read()
                    return members, data
    raise RuntimeError("top layer with diff_id %s not found in docker save output" % diff_id)


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--docker-host", required=True, help="literal unix:///absolute/socket")
    parser.add_argument("--base", required=True, help="base image reference, pinned by digest")
    parser.add_argument("--new", required=True, help="swap image reference")
    parser.add_argument("--expected-binary-sha256", required=True)
    parser.add_argument("--seed-dir", required=True, choices=("/mainnet-seed", "/testnet-seed"))
    parser.add_argument("--expected-base-binary-sha256",
                        default="94d76a0a2a8dc6667e1c6ae504d37e0f5874e64b9026aabff35bb22978667ea8")
    parser.add_argument("--receipt", type=Path, help="write the check results as JSON here (new file)")
    args = parser.parse_args()
    if not re.fullmatch(r"unix:///[A-Za-z0-9._/-]+", args.docker_host):
        sys.exit("docker host must be a literal unix:/// path")

    c = Checks()
    base, new = inspect(args.docker_host, args.base), inspect(args.docker_host, args.new)
    c.add("platform linux/amd64", (base["Os"], base["Architecture"], new["Os"], new["Architecture"]) ==
          ("linux", "amd64", "linux", "amd64"))
    for field in CONFIG_FIELDS:
        c.add(f"config.{field} unchanged", base["Config"].get(field) == new["Config"].get(field),
              json.dumps(new["Config"].get(field))[:160])
    base_layers, new_layers = base["RootFS"]["Layers"], new["RootFS"]["Layers"]
    c.add("base layers kept in order, one layer added",
          new_layers[:-1] == base_layers and len(new_layers) == len(base_layers) + 1,
          f"base {len(base_layers)} layers, new {len(new_layers)}")

    members, data = top_layer_members(args.docker_host, args.new, new_layers[-1])
    names = {m["name"] for m in members}
    c.add("added layer holds only /usr/local/bin/zeroned (and its parent dirs)",
          TARGET in names and names <= ALLOWED_LAYER_DIRS | {TARGET} and not any("/.wh." in n or n.startswith(".wh.") for n in names),
          ", ".join(sorted(names)))
    actual = hashlib.sha256(data).hexdigest() if data is not None else None
    c.add("added zeroned sha256", actual == args.expected_binary_sha256, str(actual))
    target = next((m for m in members if m["name"] == TARGET), {})
    c.add("added zeroned is root:root 0755", (target.get("uid"), target.get("gid"), target.get("mode")) == (0, 0, "0o755"),
          json.dumps(target))

    base_bin = run(args.docker_host, args.base, "/bin/sh", "-c",
                   "sha256sum /usr/local/bin/zeroned | cut -d' ' -f1; stat -c '%a %u %g' /usr/local/bin/zeroned").split()
    c.add("base zeroned is the expected live binary", base_bin[0] == args.expected_base_binary_sha256, base_bin[0])
    c.add("zeroned owner and mode match base", base_bin[1:] == ["755", "0", "0"], " ".join(base_bin[1:]))

    for label, ref in (("base", args.base), ("new", args.new)):
        which = run(args.docker_host, ref, "/bin/sh", "-c", "which -a zeroned; command -v entrypoint.sh || true").split()
        c.add(f"{label}: PATH resolves zeroned only to /usr/local/bin/zeroned", which[:1] == ["/usr/local/bin/zeroned"]
              and which.count("/usr/local/bin/zeroned") == 1 and sum(1 for w in which if w.endswith("/zeroned")) == 1,
              " ".join(which))

    digests = {}
    for label, ref in (("base", args.base), ("new", args.new)):
        digests[label] = run(args.docker_host, ref, "/bin/sh", "-c", FS_DIGEST).split()
    c.add("every other file (names, sizes, modes, owners, contents) identical",
          digests["base"] == digests["new"] and len(digests["base"]) == 2, "aggregate digests compared, not printed")

    seed = {}
    for label, ref in (("base", args.base), ("new", args.new)):
        seed[label] = run(args.docker_host, ref, "/bin/sh", "-c",
                          f"cd {args.seed_dir} && for f in $(ls -A | sort); do printf '%s %s %s ' \"$f\" $(stat -c '%s %a' \"$f\"); "
                          "sha256sum \"$f\" | cut -d' ' -f1; done").splitlines()
    shown = [" ".join(line.split()[:3]) for line in seed["new"]]
    c.add(f"{args.seed_dir} present and identical", seed["base"] == seed["new"] and len(seed["new"]) > 0,
          "; ".join(shown))
    entry = {label: run(args.docker_host, ref, "/bin/sh", "-c", "sha256sum /usr/local/bin/entrypoint.sh | cut -d' ' -f1").strip()
             for label, ref in (("base", args.base), ("new", args.new))}
    c.add("entrypoint.sh identical", entry["base"] == entry["new"], entry["new"])

    version = run(args.docker_host, args.new, "/usr/local/bin/zeroned", "version", "--long", "--output", "json",
                  "--home", "/tmp/swap-verify-home")
    info = json.loads(version)
    deps = {d.split("@")[0]: d.split("@")[1] for d in info.get("build_deps", []) if "@" in d}
    c.add("new zeroned reports go1.25.14 and cometbft v0.38.25",
          "go1.25.14" in info.get("go", "") and deps.get("github.com/cometbft/cometbft") == "v0.38.25",
          f"go={info.get('go')} cometbft={deps.get('github.com/cometbft/cometbft')} sdk={info.get('cosmos_sdk_version')} "
          f"version={info.get('version')} commit={info.get('commit')}")
    c.add("new zeroned keeps Cosmos SDK v0.50.15 and ibc-go v8.8.0",
          info.get("cosmos_sdk_version") == "v0.50.15" and deps.get("github.com/cosmos/ibc-go/v8") == "v8.8.0",
          f"ibc-go={deps.get('github.com/cosmos/ibc-go/v8')}")

    receipt = {"schema": "zerone-signer-swap-image-check/v1", "base": args.base, "new": args.new,
               "base_image_id": base["Id"], "new_image_id": new["Id"], "new_layers": new_layers,
               "added_layer_members": members, "expected_binary_sha256": args.expected_binary_sha256,
               "seed_dir": args.seed_dir, "checks": c.results, "passed": c.passed,
               "written_by": "verify-swap-image.py (agent-written, delegate of session 8d4c201e)"}
    if args.receipt:
        with args.receipt.open("x") as out:
            json.dump(receipt, out, indent=2, sort_keys=True)
            out.write("\n")
    print("RESULT", "PASS" if c.passed else "FAIL")
    return 0 if c.passed else 1


if __name__ == "__main__":
    sys.exit(main())

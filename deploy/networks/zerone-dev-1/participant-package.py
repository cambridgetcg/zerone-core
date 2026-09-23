#!/usr/bin/env python3
"""Compose pinned knowledge11 participant packages; no build, network or activation."""
from __future__ import annotations

import argparse
import gzip
import hashlib
import io
import json
import os
from pathlib import Path
import stat
import subprocess
import tarfile

SOURCE = "5542b221864ca0080e143fa82abbef4cd2064f71"
TREE = "bb41fa89bbf5db36da09f9c1c70a5e626b1649bc"
PREDECESSOR = "ac5684e33b6946a31f6ea260ebf280eace55b033"
UPSTREAM_TAG = "zerone-dev-1-5542b221864c"
TAG = UPSTREAM_TAG + "-participants-v1"
BASE = "https://github.com/cambridgetcg/zerone-core/releases"
IMAGE = "registry.fly.io/zerone-dev-1@sha256:a07481a5d9f1f8a434ac23f204a28e2fe022f3ed8a77bd7e53889000dcc569be"
DESCRIPTOR_SHA = "768d7dd7b389d844670fa7f4ac7b34b41c7b9ac4994f88526369f7726dd993e1"
ORIGINAL_DESCRIPTOR_SHA = "e01f9e472630d98faef57341cb431acb6a0e733873b79ca95103ab9cba043e2b"
GENESIS_SHA = "f8b4570b37fd41d5b63c02fbae1f89225c197c75315e8629650716f6e5f79256"
UPGRADE_SHA = "04f0415a361396e28397155ecb3c0b13d578475ebf1b86c3e2e2b5d533006f75"
UPSTREAM_RELEASE_SHA = "be85503652e9c467e074970c7169c80d5864a2a34fc620db7ca83d0b435d1d2b"
HELPER_SHA = "df92be14543c6fab2a3d6dad846d74becded381628746382af09302271ae1a50"
BINARY_HASHES = {
    "linux-amd64": "6cbafddf0d1c8dda3e6ab9f8a5d87d49992f7b699f746ee2d7632e494115a2c6",
    "darwin-arm64": "25afbe28a488cf03021fdf6c59836bad1f4040076acea961770c5d7e4486e83c",
}
ARCHIVES = {
    "linux-amd64": (36057667, "ddd41bea901a7f114979d787629874a71a2ea0d806c01c4be53afbf3cf2a43c5"),
    "darwin-arm64": (33840002, "5bb6cad6d981a3601aa503329f04c36d3d7a49a6b1ed8b3aee3e0098341a6b8b"),
}
SOURCE_FILES = {
    "scripts/shared-claims.py": "scripts/shared-claims.py",
    "scripts/claim-workflow.py": "scripts/claim-workflow.py",
    "scripts/local-node.py": "scripts/local-node.py",
    "runtime/runtime.py": "deploy/networks/zerone-dev-1/runtime.py",
}
COMMON_MEMBERS = {
    "BINARY-BUILDINFO.txt", "LINUX-BUILD.json", "README.md", "RUNTIME-IDENTITY.json",
    "SHA256SUMS", "SOURCE.json", "UPGRADE.json", "bin/zeroned", "genesis.json", "network.json",
    *SOURCE_FILES,
}
MAX_ARCHIVE = 64 << 20
MAX_MEMBER = 160 << 20
MAX_TOTAL = 200 << 20


def require(condition, message):
    if not condition:
        raise ValueError(message)


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


def encoded(value):
    return (json.dumps(value, indent=2, sort_keys=True) + "\n").encode()


def parse(raw):
    def unique(items):
        value = {}
        for key, item in items:
            require(key not in value, "Duplicate JSON field")
            value[key] = item
        return value
    return json.loads(raw, object_pairs_hook=unique)


def read(path, maximum):
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, "rb") as stream:
        info = os.fstat(stream.fileno())
        require(stat.S_ISREG(info.st_mode) and info.st_nlink == 1 and info.st_size <= maximum,
                "Expected a bounded, single-link regular input file")
        raw = stream.read(maximum + 1)
        require(len(raw) == info.st_size, "Input changed or exceeded its size bound")
    return raw


def checked(raw, expected, label):
    require(sha(raw) == expected, label + " SHA-256 mismatch")
    return raw


def unpack(raw, expected_sha, platform):
    checked(raw, expected_sha, "Upstream archive")
    allowed = COMMON_MEMBERS | ({"bin/darwin-acl-check"} if platform == "darwin-arm64" else set())
    files = {}
    with tarfile.open(fileobj=io.BytesIO(raw), mode="r:gz") as archive:
        total = 0
        for member in archive:
            require(member.name in allowed and member.name not in files and member.isfile(),
                    "Unexpected, duplicate or non-regular archive member")
            total += member.size
            require(0 <= member.size <= MAX_MEMBER and total <= MAX_TOTAL, "Archive size bound exceeded")
            stream = archive.extractfile(member)
            require(stream is not None, "Missing archive member content")
            value = stream.read(MAX_MEMBER + 1)
            require(len(value) == member.size, "Archive member size mismatch")
            files[member.name] = value
    require(set(files) == allowed, "Archive member set mismatch")
    checksums = {}
    for line in files["SHA256SUMS"].decode().splitlines():
        digest, name = line.split("  ", 1)
        require(name not in checksums, "Duplicate inner checksum")
        checksums[name] = digest
    require(set(checksums) == set(files) - {"SHA256SUMS"}, "Incomplete inner checksum coverage")
    for name, digest in checksums.items():
        checked(files[name], digest, name)
    return files


def check_descriptor(raw, original_raw):
    checked(raw, DESCRIPTOR_SHA, "Knowledge11 descriptor")
    checked(original_raw, ORIGINAL_DESCRIPTOR_SHA, "Original descriptor")
    original = parse(original_raw)
    require(original.get("source_commit") == PREDECESSOR and original.get("knowledge_version") == 10,
            "Original descriptor is not the predecessor")
    expected = {**original, "knowledge_version": 11, "source_commit": SOURCE,
                "runtime_binary_sha256": BINARY_HASHES["linux-amd64"]}
    require(parse(raw) == expected, "Descriptor changes fields outside the applied source/version/binary")


def check_source_receipt(files, platform):
    source = parse(files["SOURCE.json"])
    require(source.get("schema") == "zerone-development-upgrade-package/v1" and
            source.get("chain_id") == "zerone-dev-1" and source.get("platform") == platform and
            source.get("source_commit") == SOURCE and source.get("source_tree") == TREE and
            source.get("runtime_image") == IMAGE and source.get("private_keys_in_package") is False,
            "Upstream source receipt identity mismatch")
    require(source.get("target_binary_sha256") == BINARY_HASHES[platform], "Source receipt binary mismatch")
    require(source.get("original_descriptor_sha256") == ORIGINAL_DESCRIPTOR_SHA and
            source.get("original_genesis_sha256") == GENESIS_SHA and
            source.get("upgrade_packet_sha256") == UPGRADE_SHA, "Source receipt public pins mismatch")
    require(set(source.get("source_files", {})) == set(SOURCE_FILES.values()), "Source receipt file set mismatch")
    for name, path in SOURCE_FILES.items():
        checked(files[name], source["source_files"][path], name)
    checked(files["bin/zeroned"], BINARY_HASHES[platform], "Target binary")
    checked(files["genesis.json"], GENESIS_SHA, "Original genesis")
    checked(files["UPGRADE.json"], UPGRADE_SHA, "Original upgrade packet")
    if platform == "darwin-arm64":
        checked(files["bin/darwin-acl-check"], HELPER_SHA, "Darwin ACL helper")
    identity = parse(files["RUNTIME-IDENTITY.json"])
    require(identity.get("source_commit") == SOURCE and identity.get("binary_sha256") == BINARY_HASHES[platform]
            and identity.get("cosmos_sdk_version") == "v0.53.8", "Runtime identity receipt mismatch")
    return source


def git(repo, *args):
    environment = {key: value for key, value in os.environ.items() if not key.startswith("GIT_")}
    environment.update(GIT_NO_REPLACE_OBJECTS="1", GIT_CONFIG_NOSYSTEM="1", GIT_CONFIG_GLOBAL=os.devnull)
    return subprocess.check_output(["git", "-C", str(repo), *args], env=environment)


def compose(files, platform, descriptor, recipe_sha):
    source = check_source_receipt(files, platform)
    check_descriptor(descriptor, files["network.json"])
    result = {name: raw for name, raw in files.items() if name not in {"SOURCE.json", "README.md", "SHA256SUMS", "network.json"}}
    result.update({"network.json": descriptor, "UPSTREAM-SOURCE.json": files["SOURCE.json"],
                   "UPSTREAM-README.md": files["README.md"], "predecessor-network.json": files["network.json"]})
    archive_name = f"{UPSTREAM_TAG}-{platform}.tar.gz"
    receipt = {
        "schema": "zerone-development-participant-package/v1", "chain_id": "zerone-dev-1", "platform": platform,
        "source_commit": SOURCE, "source_tree": TREE, "source_files": source["source_files"],
        "source_scope": "Unchanged executable, client scripts and runtime helper from the exact upstream release; newly composed package metadata is identified separately by recipe hash.",
        "binary_sha256": BINARY_HASHES[platform], "descriptor_sha256": DESCRIPTOR_SHA,
        "genesis_sha256": GENESIS_SHA, "knowledge_version": 11, "release_tag": TAG,
        "packaging_recipe": {"path": "deploy/networks/zerone-dev-1/participant-package.py", "sha256": recipe_sha},
        "upstream_archive": {"name": archive_name, "bytes": ARCHIVES[platform][0], "sha256": ARCHIVES[platform][1],
                             "url": f"{BASE}/download/{UPSTREAM_TAG}/{archive_name}"},
        "upstream_release_sha256": UPSTREAM_RELEASE_SHA, "upstream_source_receipt_sha256": sha(files["SOURCE.json"]),
        "upgrade_packet_sha256": UPGRADE_SHA, "predecessor_descriptor_sha256": ORIGINAL_DESCRIPTOR_SHA,
        "build_image": IMAGE, "build_image_basis": "Inherited upstream build provenance; not a new observation of the hosted machine.",
        "publication_basis": "Composition of verified public inputs; this packager does not query activation, build or execute a binary, or perform a transaction.",
        "private_keys_in_package": False,
    }
    result["SOURCE.json"] = encoded(receipt)
    result["README.md"] = f"""# Zerone knowledge11 participant package ({platform})

This additive package initializes a new participant on zerone-dev-1. It reuses
the verified executable and client source at {SOURCE}; it does not rebuild them.
The separately published, dated development notice supplies the applied-upgrade
observation. A package or descriptor alone is not an independent activation proof.
Development funds have no promised monetary value; consensus is single-operator.

Authenticate this archive against the externally published release.json hash
before extracting into a NEW directory. Inner checksums do not authenticate an
untrusted archive. Python 3.11+ is required. Keep all files together and the macOS
ACL helper beside bin/zeroned. No participant, faucet or validator keys are included.

For a NEW private participant home:

```sh
python3 -I -B scripts/shared-claims.py init \\
  --home "$HOME/zerone-dev-participant" \\
  --descriptor "$PWD/network.json" --descriptor-sha256 {DESCRIPTOR_SHA} \\
  --binary "$PWD/bin/zeroned" --binary-sha256 {BINARY_HASHES[platform]}
```

Initialization creates an owner-only, unencrypted development test keyring. It
does not request funds, register, sign a transaction, or create consensus keys.
Keep the complete home private. Existing participant homes keep their original
verified binary, descriptor and all three client scripts; do not edit their pins.
Continue onboarding and review instructions at https://zerone.ai/development/.

The original version10 genesis is unchanged. This is NOT a fresh follower join
package: the successor initializer must not start that genesis at block1.
Followers need the verified predecessor package's join command, followed by the
original staged upgrade instructions with both exact binaries:
{BASE}/tag/{UPSTREAM_TAG}
UPGRADE.json is retained verbatim (SHA-256 {UPGRADE_SHA}).
predecessor-network.json, UPSTREAM-SOURCE.json and UPSTREAM-README.md are historical
inputs, not current participant instructions. Original release archives remain
unchanged. SOURCE.json identifies this new composition and its recipe hash.
The inherited build/identity receipts do not claim a new build, independent
reproduction, security audit, or current hosted-machine inspection.
""".encode()
    result["SHA256SUMS"] = "".join(sha(raw) + "  " + name + "\n" for name, raw in sorted(result.items())).encode()
    return result


def write_archive(path, files, epoch):
    with path.open("xb") as output:
        with gzip.GzipFile(filename="", fileobj=output, mode="wb", mtime=epoch) as zipped:
            with tarfile.open(fileobj=zipped, mode="w", format=tarfile.PAX_FORMAT) as archive:
                for name, raw in sorted(files.items()):
                    entry = tarfile.TarInfo(name)
                    entry.size, entry.mtime = len(raw), epoch
                    entry.mode = 0o555 if name.startswith("bin/") else 0o444
                    entry.uid = entry.gid = 0
                    archive.addfile(entry, io.BytesIO(raw))


def build(repo, upstream, descriptor_path, output):
    repo, upstream = Path(repo).resolve(), Path(upstream).resolve()
    output = Path(output).absolute()
    require(not os.path.lexists(output) and output.parent.is_dir(), "Output must be a fresh directory")
    output = output.parent.resolve() / output.name
    require(repo != output and repo not in output.parents, "Output must be outside the source repository")
    recipe_sha = sha(read(Path(__file__), 1 << 20))
    descriptor = read(descriptor_path, 1 << 20)
    release_raw = checked(read(upstream / "UPGRADE-RELEASE.json", 1 << 20), UPSTREAM_RELEASE_SHA, "Upstream release")
    release = parse(release_raw)
    require(release.get("source_commit") == SOURCE and release.get("source_tree") == TREE and
            release.get("runtime_image") == IMAGE and release.get("release_tag") == UPSTREAM_TAG,
            "Upstream release identity mismatch")
    checked(read(upstream / "UPGRADE.json", 1 << 20), UPGRADE_SHA, "Upgrade packet")
    require(git(repo, "rev-parse", SOURCE + "^{tree}").decode().strip() == TREE, "Git source tree mismatch")
    epoch = int(git(repo, "show", "-s", "--format=%ct", SOURCE))
    prepared = {}
    for platform, (size, digest) in ARCHIVES.items():
        name = f"{UPSTREAM_TAG}-{platform}.tar.gz"
        rows = [row for row in release["artifacts"] if row.get("platform") == platform]
        require(len(rows) == 1 and rows[0].get("name") == name and rows[0].get("sha256") == digest and
                rows[0].get("bytes") == size and rows[0].get("binary_sha256") == BINARY_HASHES[platform],
                "Upstream artifact identity mismatch")
        raw = read(upstream / name, MAX_ARCHIVE)
        require(len(raw) == size, "Upstream archive byte count mismatch")
        files = unpack(raw, digest, platform)
        for name, path in SOURCE_FILES.items():
            require(files[name] == git(repo, "show", SOURCE + ":" + path), "Packaged source differs from exact Git blob")
        prepared[platform] = compose(files, platform, descriptor, recipe_sha)
    output.mkdir(mode=0o700)
    artifacts = []
    for platform, files in prepared.items():
        name = f"{TAG}-{platform}.tar.gz"
        path = output / name
        write_archive(path, files, epoch)
        raw = read(path, MAX_ARCHIVE)
        artifacts.append({"name": name, "platform": platform, "bytes": len(raw), "sha256": sha(raw),
                          "binary_sha256": BINARY_HASHES[platform], "url": f"{BASE}/download/{TAG}/{name}"})
    candidate = {
        "schema": "zerone-development-participant-release/v1", "chain_id": "zerone-dev-1", "knowledge_version": 11,
        "source_commit": SOURCE, "source_tree": TREE, "release_tag": TAG, "release_url": f"{BASE}/tag/{TAG}",
        "descriptor_sha256": DESCRIPTOR_SHA, "genesis_sha256": GENESIS_SHA,
        "rpc_genesis_sha256": parse(descriptor)["rpc_genesis_sha256"], "runtime_binary_sha256": BINARY_HASHES["linux-amd64"],
        "runtime_image": IMAGE, "runtime_image_basis": "Upstream build-image provenance; no new hosted-machine observation.",
        "packaging_recipe_sha256": recipe_sha, "upstream_release_sha256": UPSTREAM_RELEASE_SHA,
        "upgrade_packet_sha256": UPGRADE_SHA, "artifacts": artifacts, "private_keys_in_packages": False,
        "bootstrap_consensus": "single-operator", "funds": "valueless-development-only",
        "publication_status": "candidate; separate dated applied-upgrade and publication verification required",
    }
    release_bytes = encoded(candidate)
    (output / "release.json").write_bytes(release_bytes)
    (output / "network.json").write_bytes(descriptor)
    (output / "SHA256SUMS").write_text("".join(row["sha256"] + "  " + row["name"] + "\n" for row in artifacts) +
                                      sha(release_bytes) + "  release.json\n" + DESCRIPTOR_SHA + "  network.json\n")
    return candidate


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    for name in ("repo", "upstream", "descriptor", "output"):
        parser.add_argument("--" + name, type=Path, required=True)
    args = parser.parse_args()
    os.umask(0o077)
    print(json.dumps(build(args.repo, args.upstream, args.descriptor, args.output), indent=2, sort_keys=True))


if __name__ == "__main__":
    main()

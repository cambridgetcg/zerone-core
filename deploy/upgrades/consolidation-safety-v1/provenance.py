#!/usr/bin/env python3
"""Local H1 testnet source export, offline reproducible build and OCI packaging.

Adapted from deploy/legacy-observer/provenance.py at
46fa54369353b85438d649b0985299159601cd4b (SHA256
2480cb874eb7510e9712ec2d42652b13f7ef21b3c65d5e12298fd8a46df7677e).
No signing, registry access, pulling, node startup or publication. Python 3.11+.

The frozen H1 application source is unchanged. Its go.mod and go.sum are
built through one pinned, reviewed dependency patch (h1-dependencies.patch,
byte-identical to trunk's deploy/legacy-observer/observer-dependencies.patch):
CometBFT v0.38.20 -> v0.38.25, plus four minimum-version-selection bumps. The
strict applier is adapted from deploy/legacy-observer/observer-build.py.
"""
from __future__ import annotations

import argparse
import hashlib
import io
import json
import os
from pathlib import Path
import platform
import re
import subprocess
import sys
import tarfile

SOURCE_COMMIT = "65c19cd8b00bdfff9b80705b776fd0d49719398a"
SOURCE_TREE = "8e6a2df14088522e7103da7f76dd8b6f3b5cda29"
SOURCE_ARCHIVE_SHA256 = "2ab7a6b96ad6f2294ad7bbb7ddf5140c47763b8fca3ad9ae864e23927e49e722"
SOURCE_FILE_COUNT = 674
OLD_COMMIT = "2e37c4c86c31e67515d6aa07b6fd406083e012ea"
BUILDER_IMAGE = "registry-1.docker.io/library/golang@sha256:c268a04d59aea0b180ed9946a658cfab9e7b3391dc90eed6e4969ccff98c851f"
HERE = Path(__file__).resolve().parent
# Consensus review 2026-11-05, finding cometbft-3: the frozen H1 closure pins
# CometBFT v0.38.20, whose P2P layer can be crashed by one oversized blocksync
# message (cometbft-1) and which carries GHSA-c32p-wcqj-j677 (cometbft-2).
# The dependency-only patch below is the reviewed observer recipe; application
# source, Cosmos SDK v0.50.15 and IBC v8.8.0 are unchanged.
DEPENDENCY_PATCH = HERE / "h1-dependencies.patch"
DEPENDENCY_PATCH_SHA256 = "3eb79d1fa44f76fe8a5c8edbcf90445b5cd78b7b2d08d51a296143135fdd14ce"
DEPENDENCY_PATCH_MAX_BYTES = 64 * 1024
# Frozen H1 go.mod/go.sum (byte-identical to the legacy 2e37c4c files) and the patched result.
DEPENDENCY_FROZEN_SHA256 = {
    "go.mod": "4ff80378f0462ecca11ce9000a7960f9bcbb84a33c665a5c108183c932969ea9",
    "go.sum": "1b9e15452454f8768f0cfdb553e23371f49194cab84ff04ca1829bac3aae42bf",
}
DEPENDENCY_TARGET_SHA256 = {
    "go.mod": "0a0e55141285237cc5439c182b67cf0fe3560436d3a2a49a9d4e9a3681371420",
    "go.sum": "748a52dc39ab9329f9f0cadf42d807de541ca07f8ff97f84d871b063d148d30a",
}
COMETBFT_VERSION = "v0.38.25"
# Normalized tar of the compile closure actually built (frozen files plus the two patched files).
BUILD_SOURCE_SHA256 = "7fee65036f2dddb6697ba6c4bece2c277825d8e15cead37778a5608b67730e7f"
# Authorized emulation must reproduce the pinned patched cross-build, not merely itself.
CROSS_BUILD_SHA256 = "df70456f9296b3bd6593652466a405e1ee34a93f2136b5d668888764aaf8c592"
# The unpatched (CometBFT v0.38.20) cross-build pinned before 2026-11-06. Never accepted now.
RETIRED_CROSS_BUILD_SHA256 = "f35c98b59aba9d9b77062dede87bbe18a688ee26adad5c0b961f156598633495"
RESOURCES = {"go.mod", "go.sum", "Makefile", "LICENSE", "docs/swagger-ui/embed.go",
             "docs/swagger-ui/index.html", "docs/swagger-ui/swagger.json"}
VERSION = "h1-testnet-65c19cd8-cometbft-v0.38.25"
DOCKER_HOST = "unix:///var/run/docker.sock"
CA_PATH = "/etc/ssl/certs/ca-certificates.crt"
MIN_GAS_PRICES = "0.025uzrn"
BUILD_ENV = {"GOTOOLCHAIN": "local", "GOWORK": "off", "GOMAXPROCS": "4",
             "GOCACHE": "/cache", "GOMODCACHE": "/modules", "GOPATH": "/gopath",
             "CGO_ENABLED": "0", "GOOS": "linux", "GOARCH": "amd64", "GOAMD64": "v1",
             "GOFLAGS": "-mod=readonly", "GOPROXY": "off", "GOSUMDB": "off", "HOME": "/tmp",
             "GOENV": "off", "GOPRIVATE": "", "GONOPROXY": "none", "GONOSUMDB": "none", "GOVCS": "*:off"}
LDFLAGS = ("-s -w -X github.com/cosmos/cosmos-sdk/version.Name=zerone "
           "-X github.com/cosmos/cosmos-sdk/version.AppName=zeroned "
           f"-X github.com/cosmos/cosmos-sdk/version.Version={VERSION} "
           f"-X github.com/cosmos/cosmos-sdk/version.Commit={SOURCE_COMMIT}")


def require(condition, message):
    if not condition:
        raise ValueError(message)


def sha256(path):
    digest = hashlib.sha256()
    with Path(path).open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def write_json(path, value):
    Path(path).write_text(json.dumps(value, sort_keys=True, indent=2) + "\n")


def new_output(path):
    path = Path(path).absolute()
    require(not path.exists() and not path.is_symlink(), "output must be a NEW directory")
    require(path.parent.is_dir(), "output parent must exist")
    path.mkdir(mode=0o700)
    return path.resolve()


def selected_source(name):
    parts = name.split("/")
    if any(p in ("", ".", "..", ".git", "deploy", "cosmovisor", "testdata", "keyring-test") for p in parts):
        return False
    return name in RESOURCES or (parts[0] in ("app", "cmd", "x") and name.endswith(".go")
                                 and not name.endswith("_test.go"))


def tar_bytes(entries):
    output = io.BytesIO()
    with tarfile.open(fileobj=output, mode="w", format=tarfile.PAX_FORMAT) as archive:
        for name, mode, body in sorted(entries):
            item = tarfile.TarInfo(name)
            item.mode, item.size = mode, len(body)
            item.uid = item.gid = item.mtime = 0
            archive.addfile(item, io.BytesIO(body))
    return output.getvalue()


def read_dependency_patch(path=None):
    path = DEPENDENCY_PATCH if path is None else Path(path)
    require(path.is_file() and not path.is_symlink(), "dependency patch must be a regular file")
    require(path.stat().st_size <= DEPENDENCY_PATCH_MAX_BYTES, "dependency patch exceeds its bound")
    patch = path.read_bytes()
    require(hashlib.sha256(patch).hexdigest() == DEPENDENCY_PATCH_SHA256, "dependency patch hash differs from pin")
    return patch


def apply_dependency_patch(originals, patch):
    """Strict unified-diff applier: exact context, exact counts, pinned inputs and outputs.

    Adapted from deploy/legacy-observer/observer-build.py apply_patch. Only the
    two declared dependency files may change; no fuzz, offsets or new files.
    """
    require(hashlib.sha256(patch).hexdigest() == DEPENDENCY_PATCH_SHA256, "dependency patch hash differs from pin")
    for name, digest in DEPENDENCY_FROZEN_SHA256.items():
        require(name in originals and hashlib.sha256(originals[name]).hexdigest() == digest,
                f"frozen {name} differs from the dependency patch's pinned input")
    lines = patch.decode("utf-8").splitlines(keepends=True)
    replacements = {}
    i = 0
    while i < len(lines):
        require(lines[i].startswith("--- a/"), "expected exact patch file header")
        name = lines[i][6:].rstrip("\n")
        require(name in DEPENDENCY_TARGET_SHA256 and name not in replacements, "undeclared or duplicate patch target")
        i += 1
        require(i < len(lines) and lines[i] == "+++ b/" + name + "\n", "patch target differs")
        i += 1
        source = originals[name].decode("utf-8").splitlines(keepends=True)
        result, cursor = [], 0
        while i < len(lines) and not lines[i].startswith("--- a/"):
            match = re.fullmatch(r"@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@\n", lines[i])
            require(match is not None, "malformed patch hunk")
            old_start, old_count = int(match[1]), int(match[2] or 1)
            new_start, new_count = int(match[3]), int(match[4] or 1)
            require(cursor <= old_start - 1 <= len(source), "patch hunk offset differs")
            result.extend(source[cursor:old_start - 1])
            cursor = old_start - 1
            require(len(result) == new_start - 1, "patch output offset differs")
            i += 1
            old_used = new_used = 0
            while i < len(lines) and not lines[i].startswith(("@@ ", "--- a/")):
                line = lines[i]
                require(line[:1] in (" ", "+", "-"), "malformed patch body")
                if line[0] in " -":
                    require(cursor < len(source) and source[cursor] == line[1:], "patch source context differs")
                    cursor += 1
                    old_used += 1
                if line[0] in " +":
                    result.append(line[1:])
                    new_used += 1
                i += 1
            require((old_used, new_used) == (old_count, new_count), "patch hunk counts differ")
        result.extend(source[cursor:])
        body = "".join(result).encode("utf-8")
        require(hashlib.sha256(body).hexdigest() == DEPENDENCY_TARGET_SHA256[name], f"patched {name} hash differs from pin")
        replacements[name] = body
    require(set(replacements) == set(DEPENDENCY_TARGET_SHA256), "dependency patch is incomplete")
    return replacements


def export_source(repository, output, commit=SOURCE_COMMIT):
    """Export the frozen closure; for H1, write the dependency-patched build closure to src/.

    source.tar is always the FROZEN closure (its pin is unchanged). For H1,
    build-source.tar and src/ carry the two patched dependency files.
    """
    require(commit in (SOURCE_COMMIT, OLD_COMMIT), "only the two frozen source commits are allowed")
    env = dict(os.environ, GIT_NO_REPLACE_OBJECTS="1")
    def git(*args):
        return subprocess.check_output(["git", "-C", str(repository), *args], env=env)
    tree = git("rev-parse", commit + "^{tree}").decode().strip()
    if commit == SOURCE_COMMIT:
        require(tree == SOURCE_TREE, "H1 tree differs from pin")
    patch = read_dependency_patch() if commit == SOURCE_COMMIT else None
    files, entries = [], []
    for raw in git("ls-tree", "-r", "-z", commit).split(b"\0"):
        if not raw:
            continue
        meta, name = raw.decode().split("\t", 1)
        if not selected_source(name):
            continue
        mode, kind, oid = meta.split()
        require(kind == "blob" and mode in ("100644", "100755"), "non-regular source object")
        body = git("cat-file", "blob", oid)
        entries.append((name, int(mode, 8) & 0o777, body))
        files.append({"path": name, "git_blob": oid, "mode": mode, "bytes": len(body),
                      "sha256": hashlib.sha256(body).hexdigest()})
    require(len(files) > 2, "empty compile closure")
    archive = output / "source.tar"
    archive.write_bytes(tar_bytes(entries))
    if commit == SOURCE_COMMIT:
        require(len(files) == SOURCE_FILE_COUNT and sha256(archive) == SOURCE_ARCHIVE_SHA256,
                "H1 normalized compile closure differs from pin")
    replacements = {} if patch is None else apply_dependency_patch({n: b for n, _, b in entries}, patch)
    build_entries = [(name, mode, replacements.get(name, body)) for name, mode, body in entries]
    source = output / "src"
    source.mkdir()
    build_files = []
    for name, mode, body in build_entries:
        destination = source / name
        destination.parent.mkdir(parents=True, exist_ok=True)
        destination.write_bytes(body)
        destination.chmod(mode)
        build_files.append({"path": name, "sha256": hashlib.sha256(body).hexdigest(),
                            "patched": name in replacements})
    build_archive = output / "build-source.tar"
    build_archive.write_bytes(tar_bytes(build_entries))
    if commit == SOURCE_COMMIT:
        require(sha256(build_archive) == BUILD_SOURCE_SHA256, "H1 patched build closure differs from pin")
    receipt = {"schema": "zerone-h1-source/v2", "source_commit": commit, "source_tree": tree,
               "source_sha256": sha256(archive), "file_count": len(files), "files": files,
               "dependency_patch_sha256": None if patch is None else DEPENDENCY_PATCH_SHA256,
               "patched_files": {name: DEPENDENCY_TARGET_SHA256[name] for name in sorted(replacements)},
               "build_source_sha256": sha256(build_archive), "build_files": build_files}
    write_json(output / "source-closure.json", receipt)
    return receipt


def linked_module_version(metadata, module):
    """Version of a dependency recorded by `go version -m` (a `dep` line), or None."""
    for line in metadata.splitlines():
        fields = line.split("\t")
        if len(fields) >= 4 and fields[1] == "dep" and fields[2] == module:
            return fields[3]
    return None


def validate_docker_host(value):
    # No contexts, TCP, SSH, URL authorities, escapes or relative socket names.
    require(isinstance(value, str) and value.startswith("unix:///"),
            "--docker-host must be unix:///<absolute socket path>")
    path = value[len("unix://"):]
    require(all(part not in ("", ".", "..") for part in path.split("/")[1:]) and
            not any(c in value for c in "?#%\\") and not any(ord(c) < 32 or ord(c) == 127 for c in value),
            "--docker-host must be a literal absolute Unix socket path, without URL components")
    return value


def require_native_host():
    require(platform.system() == "Linux" and platform.machine() in ("x86_64", "amd64"),
            "build requires a native Linux/amd64 host; Darwin/ARM and emulation do not qualify")


def build_argv(output, modules, run, verify=False, docker_host=DOCKER_HOST, metadata=False, ca=False):
    # An explicit local socket, no pull and no network. A missing input is fatal.
    suffix = "-verify" if verify else "-metadata" if metadata else "-ca" if ca else ""
    argv = ["docker", "--host", validate_docker_host(docker_host), "run", "--rm", "--pull=never",
            "--cidfile", str(output / f"container-{run}{suffix}.cid"),
            "--platform", "linux/amd64", "--network=none", "--read-only", "--cap-drop=ALL",
            "--security-opt=no-new-privileges:true", "--user", f"{os.getuid()}:{os.getgid()}",
            "--cpus=4", "--memory=6g", "--pids-limit=1024", "--tmpfs", "/tmp:rw,nosuid,nodev",
            "--mount", f"type=bind,src={output}/src,dst=/app,readonly",
            "--mount", f"type=bind,src={modules},dst=/modules,readonly",
            "--mount", f"type=bind,src={output}/build-{run},dst=/out",
            "--mount", f"type=bind,src={output}/cache-{run},dst=/cache", "--workdir=/app"]
    for name, value in sorted(BUILD_ENV.items()):
        argv += ["--env", name + "=" + value]
    if verify:
        return argv + [BUILDER_IMAGE, "go", "mod", "verify"]
    if metadata:
        return argv + [BUILDER_IMAGE, "go", "version", "-m", "/out/zeroned"]
    if ca:
        return argv + [BUILDER_IMAGE, "cat", CA_PATH]
    return argv + [BUILDER_IMAGE, "go", "build", "-p", "4", "-buildvcs=false", "-trimpath",
                   "-ldflags", LDFLAGS, "-o", "/out/zeroned", "./cmd/zeroned"]


def package_oci(binary, output, source_sha, ca_bundle, build_source_sha, dependency_patch_sha):
    """Deterministic scratch OCI image; manifest digest is NOT a Docker config ID."""
    require(binary.is_file() and not binary.is_symlink(), "binary must be regular")
    executable = binary.read_bytes()
    require(executable[:4] == b"\x7fELF", "runtime must be a Linux ELF")
    require(ca_bundle.is_file() and not ca_bundle.is_symlink(), "public CA bundle must be regular")
    certificates = ca_bundle.read_bytes()
    require(b"-----BEGIN CERTIFICATE-----" in certificates and b"PRIVATE KEY" not in certificates,
            "builder did not supply a public PEM CA bundle")
    layer = tar_bytes([("usr/local/bin/zeroned", 0o755, executable),
                       (CA_PATH.lstrip("/"), 0o644, certificates)])
    def encoded(value):
        return json.dumps(value, sort_keys=True, separators=(",", ":")).encode()
    blobs = {}
    def descriptor(body, media):
        digest = hashlib.sha256(body).hexdigest()
        blobs["blobs/sha256/" + digest] = body
        return {"mediaType": media, "digest": "sha256:" + digest, "size": len(body)}
    layer_desc = descriptor(layer, "application/vnd.oci.image.layer.v1.tar")
    config = {"architecture": "amd64", "os": "linux", "created": "1970-01-01T00:00:00Z",
              "config": {"Entrypoint": ["/usr/local/bin/zeroned"],
                         "Cmd": ["start", "--home", "/data/.zeroned", "--minimum-gas-prices", MIN_GAS_PRICES],
                         "Env": ["SSL_CERT_FILE=" + CA_PATH], "WorkingDir": "/",
                         "Labels": {"org.opencontainers.image.revision": SOURCE_COMMIT,
                                    "money.zerone.profile": "h1-testnet-unsigned",
                                    "money.zerone.source-sha256": source_sha,
                                    "money.zerone.build-source-sha256": build_source_sha,
                                    "money.zerone.dependency-patch-sha256": dependency_patch_sha,
                                    "money.zerone.cometbft": COMETBFT_VERSION,
                                    "money.zerone.binary-sha256": sha256(binary)}},
              "rootfs": {"type": "layers", "diff_ids": [layer_desc["digest"]]}}
    config_desc = descriptor(encoded(config), "application/vnd.oci.image.config.v1+json")
    manifest = {"schemaVersion": 2, "mediaType": "application/vnd.oci.image.manifest.v1+json",
                "config": config_desc, "layers": [layer_desc]}
    manifest_desc = descriptor(encoded(manifest), manifest["mediaType"])
    index = {"schemaVersion": 2, "manifests": [manifest_desc]}
    blobs["index.json"] = encoded(index)
    blobs["oci-layout"] = encoded({"imageLayoutVersion": "1.0.0"})
    archive = output / "image.oci.tar"
    archive.write_bytes(tar_bytes([(name, 0o644, body) for name, body in blobs.items()]))
    return {"image_digest": manifest_desc["digest"], "oci_archive_sha256": sha256(archive),
            "binary_sha256": sha256(binary), "ca_bundle_sha256": sha256(ca_bundle)}


def build(repository, destination, modules, dry_run, docker_host=DOCKER_HOST, *, allow_emulated=False):
    validate_docker_host(docker_host)
    if not dry_run and not allow_emulated:
        require_native_host()
    output = new_output(destination)
    receipt = export_source(repository, output)
    for run in (1, 2):
        for name in (f"build-{run}", f"cache-{run}"):
            (output / name).mkdir()
    modules = modules.absolute()
    require("," not in str(modules) and "," not in str(output), "Docker mount paths cannot contain commas")
    commands = [build_argv(output, modules, run, docker_host=docker_host, **mode)
                for run in (1, 2) for mode in ({"verify": True}, {}, {"metadata": True}, {"ca": True})]
    report = {"schema": "zerone-h1-testnet-image/v1", "source_commit": SOURCE_COMMIT,
              "source_sha256": receipt["source_sha256"],
              "dependency_patch_sha256": receipt["dependency_patch_sha256"],
              "build_source_sha256": receipt["build_source_sha256"], "cometbft_version": COMETBFT_VERSION,
              "builder_image": BUILDER_IMAGE,
              "build_environment": BUILD_ENV, "commands": commands, "docker_host": docker_host,
              "native_platform": None if dry_run else "emulated" if allow_emulated else "linux/amd64",
              "allow_emulated": allow_emulated,
              "expected_binary_sha256": CROSS_BUILD_SHA256 if allow_emulated else None,
              "binary_go_metadata": [],
              "status": "dry-run", "reproduced": False, "binary_sha256": None,
              "image_digest": None, "published": False, "signed": False}
    write_json(output / "provenance.json", report)
    if dry_run:
        return report
    require(modules.is_dir() and not modules.is_symlink(), "provide a local public Go module cache")
    report["status"] = "building"
    ppath = output / "provenance.json"
    write_json(ppath, report)
    try:
        for step, argv in enumerate(commands, 1):
            try:
                # Metadata and CA bytes are stdout artifacts, never Docker diagnostics.
                with (output / f"build-step-{step}.log").open("wb") as log, \
                     (output / f"build-step-{step}.stderr.log").open("wb") as errors:
                    subprocess.run(argv, check=True, stdout=log, stderr=errors, timeout=1800)
            except (KeyboardInterrupt, subprocess.TimeoutExpired):
                cid = Path(argv[argv.index("--cidfile") + 1])
                if cid.is_file() and re.fullmatch(r"[0-9a-f]{64}", cid.read_text().strip()):
                    subprocess.run(["docker", "--host", docker_host, "stop", "--time", "10",
                                    cid.read_text().strip()], check=False, timeout=30, stdout=subprocess.DEVNULL)
                raise
        first, second = output / "build-1/zeroned", output / "build-2/zeroned"
        report["binary_sha256"] = sha256(first)
        require(report["binary_sha256"] == sha256(second), "fresh-cache binary builds differ")
        if allow_emulated:
            require(report["binary_sha256"] == CROSS_BUILD_SHA256,
                    "emulated binary SHA-256 differs from the pinned cross-build")
        for item in receipt["build_files"]:
            require(sha256(output / "src" / item["path"]) == item["sha256"], "source changed during build")
        require((output / "build-step-4.log").read_bytes() == (output / "build-step-8.log").read_bytes(),
                "pinned-builder CA bundles differ")
        report["binary_go_metadata"] = [(output / f"build-step-{step}.log").read_text() for step in (3, 7)]
        require(all("GOOS=linux" in text and "GOARCH=amd64" in text and "CGO_ENABLED=0" in text
                    for text in report["binary_go_metadata"]), "binary Go metadata differs from build profile")
        require(all(linked_module_version(text, "github.com/cometbft/cometbft") == COMETBFT_VERSION and
                    linked_module_version(text, "github.com/cosmos/cosmos-sdk") == "v0.50.15"
                    for text in report["binary_go_metadata"]), "binary does not link the patched CometBFT closure")
        report.update(package_oci(first, output, receipt["source_sha256"], output / "build-step-4.log",
                                  receipt["build_source_sha256"], receipt["dependency_patch_sha256"]))
    except (OSError, ValueError, subprocess.SubprocessError, KeyboardInterrupt):
        report["status"] = "build-failed"
        write_json(ppath, report)
        raise
    report.update(status="built-local", reproduced=True)
    write_json(output / "provenance.json", report)
    return report


def main():
    os.umask(0o077)
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repository", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--module-cache", type=Path, required=True,
                        help="explicit prepopulated public module cache; mounted read-only")
    parser.add_argument("--docker-host", default=DOCKER_HOST, help="literal local unix:///<absolute socket path>")
    parser.add_argument("--allow-emulated", action="store_true",
                        help="allow emulation only if both binaries match the pinned H1 cross-build SHA-256")
    parser.add_argument("--dry-run", action="store_true", help="export real source; never invoke Docker")
    args = parser.parse_args()
    try:
        print(json.dumps(build(args.repository, args.output, args.module_cache, args.dry_run, args.docker_host,
                               allow_emulated=args.allow_emulated), sort_keys=True))
    except (OSError, ValueError, subprocess.SubprocessError) as error:
        print(f"REFUSED: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())

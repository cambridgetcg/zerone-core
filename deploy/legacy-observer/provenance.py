#!/usr/bin/env python3
"""Export and reproduce the exact source closure for the legacy observer ELF.

Export reads only the pinned Git objects, never a working tree or deployment
seeds. Build uses a fresh, isolated native Linux amd64 Docker workspace.
Neither command signs, publishes, starts a node, or connects to Zerone.
"""
from __future__ import annotations

import argparse
import gzip
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

SOURCE_COMMIT = "2e37c4c86c31e67515d6aa07b6fd406083e012ea"
SOURCE_TREE = "5c85dd9f4f3c62a2494142509bc013e4678b0936"
SOURCE_ARCHIVE_SHA256 = "62054eed39981c367c9c6d5bdf7bb19e54766afe73d336c2ea7f4a45cbfb400f"
SOURCE_ARCHIVE_SIZE = 8140800
SOURCE_FILE_COUNT = 657
EXECUTABLE_SHA256 = "94d76a0a2a8dc6667e1c6ae504d37e0f5874e64b9026aabff35bb22978667ea8"
EXECUTABLE_SIZE = 99292776
BUILDER_IMAGE = "registry-1.docker.io/library/golang@sha256:98d673f18a1aac43da744209873cb79323e11706f909251bcfb131828b95559d"
RESOURCES = {"go.mod", "go.sum", "Makefile", "LICENSE", "docs/swagger-ui/embed.go",
             "docs/swagger-ui/index.html", "docs/swagger-ui/swagger.json"}
BUILD_ENV = {"GOTOOLCHAIN": "local", "GOMAXPROCS": "3", "GOCACHE": "/cache/build",
             "GOMODCACHE": "/cache/mod", "GOPATH": "/cache/gopath", "CGO_ENABLED": "1",
             "GOOS": "linux", "GOARCH": "amd64", "GOAMD64": "v1"}
BUILD_SCRIPT = """set -eu
go version > /app/build/go-version.txt
gcc --version > /app/build/gcc-version.txt
go env -json GOVERSION GOOS GOARCH GOAMD64 CGO_ENABLED GOROOT GOTOOLCHAIN CC CXX GOMOD GOMODCACHE GOCACHE > /app/build/build-environment.json
go mod download
make build VERSION=dev COMMIT=unknown
go version -m /app/build/zeroned > /app/build/binary-go-metadata.txt
/app/build/zeroned version --long --home /tmp/metadata-home > /app/build/binary-cli-version.txt
"""


def require(condition, message):
    if not condition:
        raise ValueError(message)


def sha256(path):
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def selected_source(name):
    parts = name.split("/")
    if any(part in ("", ".", "..", ".git", "deploy", "cosmovisor", "testdata", "keyring-test") for part in parts):
        return False
    return name in RESOURCES or (parts[0] in ("app", "cmd", "x") and name.endswith(".go")
                                 and not name.endswith("_test.go"))


def new_output(path):
    path = path.expanduser().absolute()
    require(not path.exists() and not path.is_symlink(), "Output must be a new directory")
    require(path.parent.is_dir(), "Output parent must already exist")
    path.mkdir(mode=0o700)
    return path.resolve()


def export_source(repository, destination):
    def git(*args):
        return subprocess.check_output(["git", "-C", str(repository), *args])
    require(git("rev-parse", SOURCE_COMMIT + "^{tree}").decode().strip() == SOURCE_TREE,
            "Pinned source tree differs")
    entries = []
    for raw in git("ls-tree", "-r", "-z", SOURCE_COMMIT).split(b"\0"):
        if not raw:
            continue
        meta, name = raw.decode().split("\t", 1)
        if not selected_source(name):
            continue
        mode, kind, oid = meta.split()
        require(kind == "blob" and mode in ("100644", "100755"), "Source must be a tracked regular file")
        entries.append((name, mode, oid))
    require(len(entries) == SOURCE_FILE_COUNT, "Pinned compile closure file count differs")
    output = new_output(destination)
    archive = output / "legacy-source.tar"
    files = []
    with tarfile.open(archive, "w", format=tarfile.PAX_FORMAT) as stream:
        for name, mode, oid in entries:
            body = git("cat-file", "blob", oid)
            item = tarfile.TarInfo(name)
            item.mode, item.size = int(mode, 8) & 0o777, len(body)
            item.uid = item.gid = item.mtime = 0
            stream.addfile(item, io.BytesIO(body))
            files.append({"path": name, "git_blob": oid, "mode": mode, "bytes": len(body),
                          "sha256": hashlib.sha256(body).hexdigest()})
    require(archive.stat().st_size == SOURCE_ARCHIVE_SIZE and sha256(archive) == SOURCE_ARCHIVE_SHA256,
            "Export differs from the reproduced compile closure")
    with archive.open("rb") as source, (output / "legacy-source.tar.gz").open("xb") as raw:
        with gzip.GzipFile(filename="", mode="wb", fileobj=raw, mtime=0) as compressed:
            while chunk := source.read(1024 * 1024):
                compressed.write(chunk)
    manifest = {"schema": "zerone-legacy-observer-source-closure/v1", "source_commit": SOURCE_COMMIT,
                "source_tree": SOURCE_TREE, "file_count": len(files), "files": files,
                "uncompressed_archive_sha256": sha256(archive),
                "compressed_archive_sha256": sha256(output / "legacy-source.tar.gz"),
                "scope": "Production Go compile closure and declared Swagger embed resources; excludes deployment seeds, node homes, keys, tests, working-tree changes and caches."}
    (output / "source-closure.json").write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n")
    return manifest


def read_archive(path):
    require(path.is_file() and not path.is_symlink(), "Source archive must be a regular file")
    require(path.stat().st_size <= 4 * 1024 * 1024, "Compressed source archive exceeds its bound")
    with gzip.open(path, "rb") as stream:
        data = stream.read(SOURCE_ARCHIVE_SIZE + 1)
    require(len(data) == SOURCE_ARCHIVE_SIZE and hashlib.sha256(data).hexdigest() == SOURCE_ARCHIVE_SHA256,
            "Source archive differs from the exact reproduced closure")
    files = []
    with tarfile.open(fileobj=io.BytesIO(data), mode="r:") as stream:
        members = stream.getmembers()
        require(len(members) == SOURCE_FILE_COUNT, "Source file count differs")
        names = set()
        for item in members:
            require(item.isfile() and selected_source(item.name) and item.name not in names,
                    "Source contains an undeclared, linked or duplicate entry")
            require(item.mode in (0o644, 0o755), "Source mode differs")
            names.add(item.name)
            files.append((item.name, item.mode, stream.extractfile(item).read()))
    return files


def build_argv(output):
    # An explicit local socket prevents an inherited remote Docker context from
    # turning local bind paths into paths on an unrelated daemon.
    argv = ["docker", "--host", "unix:///var/run/docker.sock", "run", "--rm",
            "--cidfile", str(output / "container.cid"), "--platform", "linux/amd64",
            "--user", f"{os.getuid()}:{os.getgid()}", "--read-only", "--cap-drop", "ALL",
            "--security-opt", "no-new-privileges:true", "--network", "bridge",
            "--cpus", "3", "--memory", "6g", "--memory-swap", "6g", "--pids-limit", "1024",
            "--mount", f"type=bind,src={output}/src,dst=/app,readonly",
            "--mount", f"type=bind,src={output}/out,dst=/app/build",
            "--mount", f"type=bind,src={output}/cache,dst=/cache",
            "--mount", f"type=bind,src={output}/tmp,dst=/tmp", "--workdir", "/app"]
    for name, value in BUILD_ENV.items():
        argv += ["--env", name + "=" + value]
    return argv + [BUILDER_IMAGE, "sh", "-c", BUILD_SCRIPT]


def reproduce(archive, destination):
    require(platform.system() == "Linux" and platform.machine() in ("x86_64", "amd64"),
            "Reproduction requires native Linux amd64 with Docker on /var/run/docker.sock")
    files = read_archive(archive)
    output = new_output(destination)
    for name in ("src", "out", "cache", "tmp"):
        (output / name).mkdir(mode=0o700)
    (output / "src").chmod(0o755)
    for name, mode, body in files:
        path = output / "src" / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(body)
        path.chmod(mode)
    for path in (output / "src").rglob("*"):
        if path.is_dir():
            path.chmod(0o755)
    (output / "src/build").mkdir(mode=0o755)
    argv = build_argv(output)
    try:
        with (output / "build.log").open("wb") as log:
            result = subprocess.run(argv, stdout=log, stderr=subprocess.STDOUT, timeout=1800)
    except (KeyboardInterrupt, subprocess.TimeoutExpired):
        cid = output / "container.cid"
        if cid.is_file() and re.fullmatch(r"[0-9a-f]{64}", cid.read_text().strip()):
            subprocess.run(["docker", "--host", "unix:///var/run/docker.sock", "stop", "--time", "10",
                            cid.read_text().strip()], check=False, timeout=30, stdout=subprocess.DEVNULL)
        raise
    binary = output / "out/zeroned"
    unchanged = all((output / "src" / name).read_bytes() == body for name, _mode, body in files)
    match = binary.is_file() and binary.stat().st_size == EXECUTABLE_SIZE and sha256(binary) == EXECUTABLE_SHA256
    passed = result.returncode == 0 and unchanged and match
    report = {"schema": "zerone-legacy-observer-reproduction/v1", "passed": passed,
              "source_status": "reproduced-exact" if passed else "reproduction-failed",
              "source_commit": SOURCE_COMMIT, "source_tree": SOURCE_TREE,
              "original_build_vcs_revision": None, "executable_sha256": sha256(binary) if binary.is_file() else None,
              "expected_executable_sha256": EXECUTABLE_SHA256, "byte_identical": match,
              "source_unchanged": unchanged, "build_exit_code": result.returncode,
              "source_archive_sha256": SOURCE_ARCHIVE_SHA256, "builder_image": BUILDER_IMAGE,
              "build_environment": BUILD_ENV, "build_script": BUILD_SCRIPT,
              "fresh_caches": True, "native_platform": "linux/amd64",
              "scope": "Byte identity and source reproduction only; not a security audit, vulnerability clearance, signer authorization or independent network trust anchor."}
    (output / "provenance.json").write_text(json.dumps(report, indent=2, sort_keys=True) + "\n")
    require(passed, "Reproduction failed; inspect the retained build.log and provenance.json")
    return report


def main():
    os.umask(0o077)
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    export = commands.add_parser("export", help="Export only the pinned source objects from a Git repository")
    export.add_argument("--repository", required=True, type=Path)
    export.add_argument("--output", required=True, type=Path)
    build = commands.add_parser("build", help="Reproduce the exact executable in an isolated Linux amd64 builder")
    build.add_argument("--source-archive", required=True, type=Path)
    build.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    try:
        report = export_source(args.repository, args.output) if args.command == "export" else reproduce(args.source_archive, args.output)
        print(json.dumps({key: value for key, value in report.items() if key != "files"}, sort_keys=True))
    except (OSError, ValueError, subprocess.SubprocessError, tarfile.TarError, EOFError) as error:
        print(json.dumps({"result": "REFUSED", "reason": str(error)}), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())

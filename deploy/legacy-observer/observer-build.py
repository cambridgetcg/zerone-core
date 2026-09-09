#!/usr/bin/env python3
"""Build the bounded legacy observer dependency patch in a fresh Linux workspace.

This does not start a node or authorize a release. The original 657-file source
archive and the two-file dependency patch must match their reviewed hashes.
"""
from __future__ import annotations

import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import platform
import re
import subprocess
import sys

_base_spec = importlib.util.spec_from_file_location("zerone_provenance", Path(__file__).resolve().with_name("provenance.py"))
base = importlib.util.module_from_spec(_base_spec)
_base_spec.loader.exec_module(base)

BUILDER_IMAGE = "registry-1.docker.io/library/golang@sha256:c268a04d59aea0b180ed9946a658cfab9e7b3391dc90eed6e4969ccff98c851f"
PATCH_SHA256 = "3eb79d1fa44f76fe8a5c8edbcf90445b5cd78b7b2d08d51a296143135fdd14ce"
TARGET_SHA256 = {
    "go.mod": "0a0e55141285237cc5439c182b67cf0fe3560436d3a2a49a9d4e9a3681371420",
    "go.sum": "748a52dc39ab9329f9f0cadf42d807de541ca07f8ff97f84d871b063d148d30a",
}
BUILD_ENV = dict(base.BUILD_ENV, GOWORK="off", GOFLAGS="-mod=readonly")
BUILD_SCRIPT = base.BUILD_SCRIPT.replace("go mod download\n", "go mod download\ngo mod verify\n")


def apply_patch(files, patch_path):
    base.require(patch_path.is_file() and not patch_path.is_symlink(), "Patch must be a regular file")
    base.require(patch_path.stat().st_size <= 64 * 1024, "Dependency patch exceeds its bound")
    patch = patch_path.read_bytes()
    base.require(hashlib.sha256(patch).hexdigest() == PATCH_SHA256, "Dependency patch hash differs")
    lines = patch.decode("utf-8").splitlines(keepends=True)
    original = {name: body for name, _mode, body in files}
    replacements = {}
    i = 0
    while i < len(lines):
        base.require(lines[i].startswith("--- a/"), "Expected exact patch file header")
        name = lines[i][6:].rstrip("\n")
        base.require(name in TARGET_SHA256 and name not in replacements, "Undeclared or duplicate patch target")
        i += 1
        base.require(i < len(lines) and lines[i] == "+++ b/" + name + "\n", "Patch target differs")
        i += 1
        source = original[name].decode("utf-8").splitlines(keepends=True)
        result, cursor = [], 0
        while i < len(lines) and not lines[i].startswith("--- a/"):
            match = re.fullmatch(r"@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@\n", lines[i])
            base.require(match is not None, "Malformed patch hunk")
            old_start, old_count, new_start, new_count = (int(match[1]), int(match[2] or 1),
                                                         int(match[3]), int(match[4] or 1))
            base.require(cursor <= old_start - 1 <= len(source), "Patch hunk offset differs")
            result.extend(source[cursor:old_start - 1])
            cursor = old_start - 1
            base.require(len(result) == new_start - 1, "Patch output offset differs")
            i += 1
            old_used = new_used = 0
            while i < len(lines) and not lines[i].startswith(("@@ ", "--- a/")):
                line = lines[i]
                base.require(line[:1] in (" ", "+", "-"), "Malformed patch body")
                if line[0] in " -":
                    base.require(cursor < len(source) and source[cursor] == line[1:], "Patch source context differs")
                    cursor += 1
                    old_used += 1
                if line[0] in " +":
                    result.append(line[1:])
                    new_used += 1
                i += 1
            base.require((old_used, new_used) == (old_count, new_count), "Patch hunk counts differ")
        result.extend(source[cursor:])
        body = "".join(result).encode("utf-8")
        base.require(hashlib.sha256(body).hexdigest() == TARGET_SHA256[name], "Patched dependency file hash differs")
        replacements[name] = body
    base.require(set(replacements) == set(TARGET_SHA256), "Dependency patch is incomplete")
    return [(name, mode, replacements.get(name, body)) for name, mode, body in files]


def build_argv(output):
    argv = base.build_argv(output)
    index = argv.index(base.BUILDER_IMAGE)
    argv = argv[:index]
    for name in ("GOWORK", "GOFLAGS"):
        argv += ["--env", name + "=" + BUILD_ENV[name]]
    return argv + [BUILDER_IMAGE, "sh", "-c", BUILD_SCRIPT]


def build(archive, patch, destination, expected_sha256=None):
    if expected_sha256 is not None:
        base.require(re.fullmatch(r"[0-9a-f]{64}", expected_sha256) is not None, "Expected executable hash must be lowercase SHA-256")
    base.require(platform.system() == "Linux" and platform.machine() in ("x86_64", "amd64"),
                 "Build requires native Linux amd64 with Docker on /var/run/docker.sock")
    files = apply_patch(base.read_archive(archive), patch)
    output = base.new_output(destination)
    for name in ("src", "out", "cache", "tmp"):
        (output / name).mkdir(mode=0o700)
    for name, mode, body in files:
        path = output / "src" / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(body)
        path.chmod(mode)
    for path in [output / "src", *(output / "src").rglob("*")]:
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
    actual = base.sha256(binary) if binary.is_file() else None
    matched = expected_sha256 is not None and actual == expected_sha256
    passed = result.returncode == 0 and unchanged and actual is not None and (expected_sha256 is None or matched)
    report = {
        "schema": "zerone-legacy-observer-patch-build/v1", "passed": passed,
        "source_status": "reproduced-observer-patch" if passed and matched else "observer-candidate-built" if passed else "build-failed",
        "base_commit": base.SOURCE_COMMIT, "base_tree": base.SOURCE_TREE,
        "original_build_vcs_revision": None, "original_executable_sha256": base.EXECUTABLE_SHA256,
        "dependency_patch_sha256": PATCH_SHA256, "patched_dependency_files": TARGET_SHA256,
        "unchanged_original_files": base.SOURCE_FILE_COUNT - len(TARGET_SHA256),
        "source_unchanged_during_build": unchanged, "executable_sha256": actual,
        "executable_bytes": binary.stat().st_size if binary.is_file() else None,
        "expected_executable_sha256": expected_sha256, "matches_expected_executable": matched,
        "build_exit_code": result.returncode, "source_archive_sha256": base.SOURCE_ARCHIVE_SHA256,
        "builder_image": BUILDER_IMAGE, "build_environment": BUILD_ENV, "build_script": BUILD_SCRIPT,
        "fresh_caches": True, "native_platform": "linux/amd64",
        "scope": "Dependency-only observer build from the exact historical compile closure. A matching build is not an application upgrade, signer authorization, security clearance, independent checkpoint or complete compatibility test."
    }
    (output / "provenance.json").write_text(json.dumps(report, indent=2, sort_keys=True) + "\n")
    base.require(passed, "Observer build failed; inspect retained build.log and provenance.json")
    return report


def main():
    os.umask(0o077)
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source-archive", required=True, type=Path)
    parser.add_argument("--dependency-patch", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--expected-sha256", help="Require equality to the independently supplied executable digest")
    args = parser.parse_args()
    try:
        print(json.dumps(build(args.source_archive, args.dependency_patch, args.output, args.expected_sha256), sort_keys=True))
    except (OSError, ValueError, subprocess.SubprocessError, base.tarfile.TarError, EOFError) as error:
        print(json.dumps({"result": "REFUSED", "reason": str(error)}), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())

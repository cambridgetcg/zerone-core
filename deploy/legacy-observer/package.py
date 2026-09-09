#!/usr/bin/env python3
"""Assemble or archive a flat, exact observer release. Never signs or uploads.

Assembly uses explicit public inputs and refuses an existing output directory.
Archive creation first verifies the signature and the entire flat file set.
"""
from __future__ import annotations

import argparse
import datetime as dt
import gzip
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import shutil
import stat
import subprocess
import sys
import tarfile

ROOT = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("release", ROOT / "verify-release.py")
release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(release)


def require(condition, message):
    if not condition:
        raise ValueError(message)


def pin(path):
    checksum = hashlib.sha256()
    with path.open("rb") as stream:
        while chunk := stream.read(1024 * 1024):
            checksum.update(chunk)
    return {"sha256": checksum.hexdigest(), "size": path.stat().st_size}


def copy(source, target, executable=False):
    require(source.is_file() and not source.is_symlink(), "Explicit public input must be a regular file")
    require(not target.exists(), "Refusing artifact overwrite")
    shutil.copyfile(source, target)
    target.chmod(0o755 if executable else 0o644)


def git(*args):
    return subprocess.check_output(["git", "-C", str(ROOT), *args])


def assemble(args):
    output = args.output.absolute()
    require(not output.exists() and not output.is_symlink(), "Output directory already exists")
    require(output.parent.is_dir(), "Output parent must already exist")
    commit = git("rev-parse", "HEAD").decode().strip()
    require(release.COMMIT.fullmatch(commit), "Invalid tooling commit")
    public_sources = {
        "observer.py": ROOT / "observer.py",
        "verify-release.py": ROOT / "verify-release.py",
        "README.md": ROOT / "README.md",
        "package.py": ROOT / "package.py",
        "provenance.py": ROOT / "provenance.py",
        "observer-build.py": ROOT / "observer-build.py",
        "verify-rehearsal.py": ROOT / "verify-rehearsal.py",
        "README-build.md": ROOT / "README-build.md",
    }
    for name, source in public_sources.items():
        tracked = git("show", commit + ":deploy/legacy-observer/" + name)
        require(source.read_bytes() == tracked, "Tooling source must match its committed version: " + name)
    inputs = {
        **public_sources, "zeroned": args.executable, "verify-checkpoint": args.checkpoint_verifier,
        "checkpoint.json": args.checkpoint, "genesis.json": args.genesis,
        "main-public.gpg": args.public_key, "provenance.json": args.provenance,
        "VULNERABILITY-REVIEW.json": args.vulnerability_report,
        "legacy-source.tar.gz": args.source_archive,
        "checkpoint-verifier-source.tar.gz": args.verifier_source_archive,
    }
    provenance = json.loads(args.provenance.read_bytes())
    require(provenance.get("source_status") in ("unattributed", "reproduced-exact", "reproduced-observer-patch"),
            "Explicit source status required")
    source_commit = provenance.get("source_commit")
    if provenance["source_status"] == "reproduced-observer-patch":
        require(args.dependency_patch is not None, "Patched observer requires its exact dependency patch")
        inputs["observer-dependencies.patch"] = args.dependency_patch
        source_declaration = {"status": "reproduced-observer-patch", "base_commit": provenance.get("base_commit"),
                              "patch_file": "observer-dependencies.patch", "evidence_file": "provenance.json"}
        require(provenance.get("dependency_patch_sha256") == pin(args.dependency_patch)["sha256"],
                "Provenance dependency patch binding differs")
    else:
        require(args.dependency_patch is None, "Unpatched source cannot include an undeclared dependency patch")
        source_declaration = {"status": provenance["source_status"], "commit": source_commit,
                              "evidence_file": "provenance.json"}
    require(provenance.get("executable_sha256") == pin(args.executable)["sha256"], "Provenance executable binding differs")
    checkpoint = json.loads(args.checkpoint.read_bytes())
    header_time = dt.datetime.fromisoformat(checkpoint["header_time"].replace("Z", "+00:00"))
    created = dt.datetime.now(dt.timezone.utc).replace(microsecond=0)
    expires = min(created + dt.timedelta(days=7), header_time + dt.timedelta(days=7, minutes=-5)).replace(microsecond=0)
    require(expires > created + dt.timedelta(hours=1), "Checkpoint needs at least one hour of remaining bootstrap time")
    output.mkdir(mode=0o755)
    for name, source in inputs.items():
        copy(source, output / name, name in ("zeroned", "verify-checkpoint"))
    files = [{"name": name, **pin(output / name)} for name in sorted(inputs)]
    executable = pin(output / "zeroned")
    manifest = {
        "schema": "zerone-1-legacy-observer-release/v1", "release_id": args.release_id,
        "created_at": created.strftime("%Y-%m-%dT%H:%M:%SZ"),
        "expires_at": expires.strftime("%Y-%m-%dT%H:%M:%SZ"), "expiry_scope": "bootstrap-only",
        "chain_id": "zerone-1", "role": "observer",
        "signature_authority": {"algorithm": "openpgp", "fingerprint": release.MAIN_FINGERPRINT,
                                "signature_file": release.SIGNATURE, "public_key_file": release.PUBLIC_KEY},
        "capabilities": release.CAPABILITIES,
        "executable": {"file": "zeroned", **executable, "os": "linux", "arch": "amd64",
                       "source": source_declaration},
        "tooling": {"commit": commit}, "runtime": release.RUNTIME, "network": release.NETWORK,
        "checkpoint": {"file": "checkpoint.json", "sha256": pin(output / "checkpoint.json")["sha256"],
                       "trust_period_seconds": release.TRUST_PERIOD}, "files": files}
    release.validate_manifest(manifest, release.MAIN_FINGERPRINT, created, "bootstrap")
    (output / release.MANIFEST).write_bytes(release.canonical(manifest))
    (output / release.MANIFEST).chmod(0o644)
    print(json.dumps({"result": "ASSEMBLED_UNSIGNED", "directory": str(output),
                      "manifest": pin(output / release.MANIFEST), "expires_at": manifest["expires_at"],
                      "effects": "local files only; no signature or publication"}, sort_keys=True))


def archive(args):
    result = release.verify(args.bundle, release.MAIN_FINGERPRINT, args.gpgv)
    require(not args.output.exists(), "Archive output already exists")
    require(not args.output.is_symlink(), "Archive output is a symlink")
    # Exactly the verified flat set; no recursive directory walks, links,
    # private home files, owner names, or host timestamps enter the archive.
    frozen = release.Bundle(args.bundle)
    try:
        with args.output.open("xb") as raw:
            with gzip.GzipFile(filename="", mode="wb", fileobj=raw, mtime=0) as compressed:
                with tarfile.open(fileobj=compressed, mode="w", format=tarfile.USTAR_FORMAT) as archive_file:
                    for name, expected in sorted(result["input_sha256_and_size"].items()):
                        content = frozen.read(name, expected["size"])
                        require(frozen.pins[name] == expected, "Artifact changed before archive creation")
                        info = tarfile.TarInfo(name)
                        info.size = len(content)
                        info.mode = 0o755 if name in ("zeroned", "verify-checkpoint") else 0o644
                        info.uid = info.gid = info.mtime = 0
                        info.uname = info.gname = ""
                        archive_file.addfile(info, io.BytesIO(content))
        frozen.unchanged(set(result["input_sha256_and_size"]))
    finally:
        frozen.close()
    print(json.dumps({"result": "ARCHIVED", "archive": pin(args.output),
                      "manifest_sha256": result["manifest_sha256"]}, sort_keys=True))


def unpack(args):
    require(args.archive.is_file() and not args.archive.is_symlink(), "Archive must be a regular file")
    require(0 < args.archive.stat().st_size <= release.MAX_TOTAL, "Archive compressed size bound")
    require(not args.output.exists() and not args.output.is_symlink(), "Extraction output must be a fresh directory")
    require(args.output.parent.is_dir(), "Extraction parent must already exist")
    args.output.mkdir(mode=0o700)
    names, total = set(), 0
    def exact_read(stream, length):
        data = stream.read(length)
        require(len(data) == length, "Truncated archive")
        return data
    descriptor = os.open(args.archive, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(descriptor, "rb") as raw:
        before = os.fstat(raw.fileno())
        require(stat.S_ISREG(before.st_mode) and 0 < before.st_size <= release.MAX_TOTAL,
                "Archive must be a bounded regular file")
        with gzip.GzipFile(fileobj=raw, mode="rb") as stream:
            while True:
                header = exact_read(stream, 512)
                if header == bytes(512):
                    require(exact_read(stream, 512) == bytes(512), "Invalid archive end marker")
                    trailing = stream.read(10240 + 1)
                    require(len(trailing) <= 10240 and not any(trailing), "Excessive or nonzero trailing archive content")
                    break
                # Parse just this fixed header. TarFile would process GNU/PAX
                # extension bodies before yielding a member, outside our bounds.
                require(header[257:265] == b"ustar\x0000", "Only USTAR headers are accepted")
                member = tarfile.TarInfo.frombuf(header, "utf-8", "strict")
                release.filename(member.name)
                require(member.type in (tarfile.REGTYPE, tarfile.AREGTYPE),
                        "Only flat regular files are allowed in this archive")
                require(member.name not in names and len(names) < release.MAX_FILES + 2, "Duplicate or excessive archive entries")
                require(0 < member.size <= release.MAX_ARTIFACT, "Archive member size bound")
                names.add(member.name)
                total += member.size
                require(total <= release.MAX_TOTAL, "Archive total size bound")
                destination = args.output / member.name
                with destination.open("xb") as target:
                    destination.chmod(0o600)
                    remaining = member.size
                    while remaining:
                        chunk = exact_read(stream, min(1024 * 1024, remaining))
                        target.write(chunk)
                        remaining -= len(chunk)
                require(not any(exact_read(stream, (-member.size) % 512)), "Nonzero archive member padding")
        require(release.stable(before) == release.stable(os.fstat(raw.fileno())), "Archive changed during extraction")
    result = release.verify(args.output, release.MAIN_FINGERPRINT, args.gpgv, args.purpose)
    for name in names:
        (args.output / name).chmod(0o755 if name in ("zeroned", "verify-checkpoint") else 0o644)
    args.output.chmod(0o755)
    print(json.dumps(result, sort_keys=True))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    make = commands.add_parser("assemble")
    make.add_argument("--release-id", required=True)
    for field in ("output", "executable", "checkpoint-verifier", "checkpoint", "genesis",
                  "public-key", "provenance", "source-archive", "verifier-source-archive", "vulnerability-report"):
        make.add_argument("--" + field, required=True, type=Path)
    make.add_argument("--dependency-patch", type=Path)
    pack = commands.add_parser("archive")
    pack.add_argument("--bundle", required=True, type=Path)
    pack.add_argument("--output", required=True, type=Path)
    pack.add_argument("--gpgv", required=True)
    extract = commands.add_parser("unpack")
    extract.add_argument("--archive", required=True, type=Path)
    extract.add_argument("--output", required=True, type=Path)
    extract.add_argument("--gpgv", required=True)
    extract.add_argument("--purpose", choices=("bootstrap", "resume"), default="bootstrap")
    args = parser.parse_args()
    try:
        {"assemble": assemble, "archive": archive, "unpack": unpack}[args.command](args)
    except (OSError, ValueError, KeyError, EOFError, tarfile.TarError, subprocess.SubprocessError) as error:
        print(json.dumps({"result": "REFUSED", "reason": str(error)}), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())

#!/usr/bin/env python3
"""Check a flat packet's exact file set and hashes; no authenticity claim."""
import argparse
import hashlib
import json
from pathlib import Path
import sys


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--manifest-sha256", help="optional separately retained manifest digest")
    args = parser.parse_args()
    root = Path(__file__).resolve().parent
    manifest = root / "MANIFEST.json"
    if manifest.is_symlink():
        raise ValueError("Manifest must be a regular file")
    raw = manifest.read_bytes()
    digest = hashlib.sha256(raw).hexdigest()
    if args.manifest_sha256 is not None and args.manifest_sha256 != digest:
        raise ValueError("Manifest differs from the supplied digest")
    record = json.loads(raw)
    if record.get("schema") != "research-review-packet/v1":
        raise ValueError("Unexpected manifest schema")
    files = record["files"]
    if not isinstance(files, list) or not files:
        raise ValueError("Empty file inventory")
    names = set()
    for item in files:
        name = item["name"]
        if (not isinstance(name, str) or not name or name in (".", "..", "MANIFEST.json")
                or "/" in name or "\\" in name or name in names):
            raise ValueError("Unsafe or duplicate packet filename")
        names.add(name)
        path = root / name
        if path.is_symlink() or not path.is_file():
            raise ValueError("Missing or non-regular packet file: " + name)
        content = path.read_bytes()
        if len(content) != item["bytes"] or hashlib.sha256(content).hexdigest() != item["sha256"]:
            raise ValueError("File differs from manifest: " + name)
    actual = {p.name for p in root.iterdir()}
    if actual != names | {"MANIFEST.json"}:
        raise ValueError("Unexpected files in packet directory")
    print(json.dumps({"pass": True, "files_verified": len(names), "manifest_sha256": digest,
                      "scope": "byte integrity only; no identity, timestamp or scientific authentication"}))


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, KeyError, TypeError) as error:
        print("Packet verification failed: " + str(error), file=sys.stderr)
        sys.exit(1)

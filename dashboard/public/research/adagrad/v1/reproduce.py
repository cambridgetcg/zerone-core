#!/usr/bin/env python3
"""Offline reproduction of the pinned AdaGrad v1 evidence package. Stdlib only."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import platform
import re
import sys
import types


FILES = {
    "README.md", "LICENSE", "PROVENANCE.json", "claim.json", "protocol.json",
    "dataset.json", "experiment.py", "reproduce.py", "results.json", "report.md",
    "loss-scaling.png", "loss-scaling.svg",
}
MAX_FILE_BYTES = 8 * 1024 * 1024


def sha(data):
    return hashlib.sha256(data).hexdigest()


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), allow_nan=False).encode()


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise ValueError("Duplicate JSON object key")
        result[key] = value
    return result


def json_value(data):
    return json.loads(data, object_pairs_hook=unique_object,
                      parse_constant=lambda _: (_ for _ in ()).throw(ValueError("Nonfinite JSON number")))


def read_regular(path, cap):
    # No traversal or special-file access through package entries.
    import stat
    flags = os.O_RDONLY | getattr(os, "O_NOFOLLOW", 0) | getattr(os, "O_NONBLOCK", 0)
    if path.is_symlink():
        raise ValueError("Package symlink refused")
    fd = os.open(path, flags)
    with os.fdopen(fd, "rb") as stream:
        metadata = os.fstat(stream.fileno())
        if not stat.S_ISREG(metadata.st_mode) or metadata.st_size > cap:
            raise ValueError("Package entry is not a bounded regular file")
        data = stream.read(cap + 1)
    if len(data) > cap:
        raise ValueError("Package file exceeds limit")
    return data


def verified_payload(expected_manifest_sha256):
    if re.fullmatch(r"[0-9a-f]{64}", expected_manifest_sha256) is None:
        raise ValueError("Expected manifest hash must be 64 lowercase hexadecimal characters")
    root = Path(__file__).resolve().parent
    raw = read_regular(root / "MANIFEST.json", 64 * 1024)
    if sha(raw) != expected_manifest_sha256:
        raise ValueError("Manifest SHA256 mismatch")
    manifest = json_value(raw)
    if set(manifest) != {"schema", "files"} or manifest["schema"] != "zerone-adagrad-evidence/v1":
        raise ValueError("Unsupported manifest")
    entries = manifest["files"]
    if not isinstance(entries, list) or len(entries) != len(FILES):
        raise ValueError("Unexpected manifest file count")
    payload = {}
    for entry in entries:
        if not isinstance(entry, dict) or set(entry) != {"name", "bytes", "sha256"}:
            raise ValueError("Malformed manifest entry")
        name = entry["name"]
        if not isinstance(name, str) or name not in FILES or name in payload:
            raise ValueError("Unexpected or duplicate package entry")
        if type(entry["bytes"]) is not int or not 0 <= entry["bytes"] <= MAX_FILE_BYTES:
            raise ValueError("Invalid package size")
        if not isinstance(entry["sha256"], str) or re.fullmatch(r"[0-9a-f]{64}", entry["sha256"]) is None:
            raise ValueError("Invalid package hash")
        data = read_regular(root / name, MAX_FILE_BYTES)
        if len(data) != entry["bytes"] or sha(data) != entry["sha256"]:
            raise ValueError("Package file mismatch: " + name)
        payload[name] = data
    if set(payload) != FILES:
        raise ValueError("Missing package entry")
    expected = json_value(payload["results.json"])
    if set(expected) != {"pass", "runs", "trajectory_comparisons"} or canonical(expected) != payload["results.json"]:
        raise ValueError("Expected results are not the canonical computational object")
    return payload, expected


def main():
    if not __debug__:
        raise RuntimeError("Run without -O; assertions are experiment checks")
    os.umask(0o077)
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--manifest-sha256", required=True,
                        help="Expected MANIFEST.json SHA256 from the evidence page or claim")
    parser.add_argument("--output", type=Path, help="New output JSON file; existing files are never overwritten")
    parser.add_argument("--verify-only", action="store_true", help="Verify files without running the optimizer")
    args = parser.parse_args()
    if not args.verify_only and args.output is None:
        parser.error("--output is required unless --verify-only is selected")
    try:
        payload, expected = verified_payload(args.manifest_sha256)
    except (OSError, ValueError, TypeError) as error:
        print(json.dumps({"pass": False, "stage": "package_verification", "error": str(error)}))
        return 2
    if args.verify_only:
        print(json.dumps({"pass": True, "stage": "package_verification",
                          "manifest_sha256": args.manifest_sha256, "files": len(payload),
                          "scope": "File integrity only; no optimizer execution or scientific review."}))
        return 0
    if args.output.exists() or args.output.is_symlink():
        print(json.dumps({"pass": False, "stage": "output", "error": "Output already exists"}))
        return 2
    # Execute the exact already-verified bytes, not a second filesystem read.
    module = types.ModuleType("frozen_adagrad_experiment")
    module.__file__ = "experiment.py"
    exec(compile(payload["experiment.py"], "experiment.py", "exec"), module.__dict__)
    content = module.execute(json_value(payload["protocol.json"]), json_value(payload["dataset.json"]))
    actual_digest = sha(canonical(content))
    expected_digest = sha(payload["results.json"])
    identical = actual_digest == expected_digest and content == expected
    result = {
        "schema": "zerone-adagrad-reproduction/v1", "manifest_sha256": args.manifest_sha256,
        "expected_computational_sha256": expected_digest, "actual_computational_sha256": actual_digest,
        "exact_result_match": identical, "prespecified_checks_pass": content["pass"],
        "environment": {"python": sys.version, "implementation": platform.python_implementation(),
                        "platform": platform.platform(), "optimizer_dependencies": "Python standard library only"},
        "scope": "This is computational reproduction, not independent scientific judgment or endorsement. Exact float disagreement may reflect platform rounding; retain this output and inspect all checks and values.",
        "computation": content,
    }
    with args.output.open("x", encoding="utf-8") as stream:
        json.dump(result, stream, sort_keys=True, separators=(",", ":"), allow_nan=False)
        stream.write("\n")
    passed = identical and content["pass"]
    print(json.dumps({"pass": passed, "exact_result_match": identical,
                      "prespecified_checks_pass": content["pass"],
                      "expected_computational_sha256": expected_digest,
                      "actual_computational_sha256": actual_digest,
                      "runs": len(content["runs"])}))
    return 0 if passed else 1


if __name__ == "__main__":
    sys.exit(main())

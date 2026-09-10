#!/usr/bin/env python3
"""Skip unrelated CI jobs only for regular, known website files in a PR diff."""

import argparse
from pathlib import PurePosixPath
import re
import subprocess
import sys


WEBSITE_FILES = {
    "dashboard/" + name for name in (
        "README.md", "index.html", "nodes/index.html", "understand/index.html",
        "research/index.html", "pi/callback/index.html", "package.json",
        "package-lock.json", "tsconfig.json", "tsconfig.functions.json",
        "vite.config.ts", "network-profile.ts", "node-guide-build.ts",
        "node-guide-page.ts", "node-guide-profile.ts", "observer-page.ts",
        "observer-release-profile.ts", "understand-guide.ts",
        "public/_headers", "public/_redirects", "public/llms.txt",
        "public/favicon.svg",
    )
}


def website_path(path):
    # Git names are untrusted input. Do not normalize a different path into an
    # allowed one; all shared standards, Functions and unknown paths stay full.
    if not path or any(ord(c) < 32 or ord(c) == 127 for c in path):
        return False
    parts = path.split("/")
    if any(part in ("", ".", "..") for part in parts) or "\\" in path:
        return False
    if path in WEBSITE_FILES:
        return True
    suffix = PurePosixPath(path).suffix
    return (
        path.startswith("dashboard/src/") and suffix in (".ts", ".css")
        or path.startswith("dashboard/tests/") and path.endswith(".test.ts")
        or path.startswith("dashboard/scripts/") and suffix in (".ts", ".mjs")
    )


def website_diff(raw):
    """Parse --raw -z --no-renames; a move must pass for BOTH old and new paths."""
    try:
        records = raw.decode("utf-8").split("\0")
    except UnicodeError:
        return False
    if len(records) < 3 or records[-1] != "" or len(records) % 2 != 1:
        return False
    for index in range(0, len(records) - 1, 2):
        fields = records[index].split()
        if len(fields) != 5 or not fields[0].startswith(":"):
            return False
        before, after = fields[0][1:], fields[1]
        expected_modes = {"A": ("000000", "100644"), "D": ("100644", "000000"), "M": ("100644", "100644")}
        if (before, after) != expected_modes.get(fields[4]):
            return False
        for mode, object_id in ((before, fields[2]), (after, fields[3])):
            if not re.fullmatch(r"[a-f0-9]{40}", object_id) or (mode == "000000") != (object_id == "0" * 40):
                return False
        if not website_path(records[index + 1]):
            return False
    return True


def full_checks(event, base, head):
    if event != "pull_request" or not all(re.fullmatch(r"[a-f0-9]{40}", ref) for ref in (base, head)):
        return True
    try:
        diff = subprocess.run(
            ["git", "diff", "--raw", "--no-abbrev", "-z", "--no-renames", f"{base}...{head}", "--"],
            check=True, capture_output=True, timeout=30,
        )
        return not website_diff(diff.stdout)
    except (OSError, subprocess.SubprocessError, UnicodeError):
        print("Unable to establish a website-only diff; keeping full checks.", file=sys.stderr)
        return True


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--event", required=True)
    parser.add_argument("--base", default="")
    parser.add_argument("--head", default="")
    args = parser.parse_args()
    print(f"full_checks={str(full_checks(args.event, args.base, args.head)).lower()}")

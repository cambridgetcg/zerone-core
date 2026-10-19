#!/usr/bin/env python3
"""Select CI jobs for a pull request from its exact diff; anything uncertain runs.

Each changed path gets one known kind or none. A job is skipped only when
every changed path has a kind that the job is known not to read. Unknown
paths, malformed or unavailable diffs, symlinks, executable bits outside the
private-network skill, secret-shaped names, files under a Go package that
uses go:embed, and every event other than pull_request keep every job.
"""

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

# The private multi-Mac network skill. Its Python tests run in Build,
# Consensus rehearsal and the macOS job, and its script reads its own
# Markdown, so the whole directory is one kind.
PRIVATE_NET_SKILL = "skills/run-a-private-zerone-net/"

# Names the repository treats as validator secrets (check-production-key-
# boundary.sh and .dockerignore). They never narrow CI, so the key boundary
# and the image key checks always see them.
SECRET_NAME = re.compile(
    r"(priv_validator_key\.json|priv_validator_state\.json|node_key\.json)(\..*)?"
    r"|.*\.(mnemonic|seed)|\.env(\..*)?|.*(secret|credentials).*\.json",
    re.IGNORECASE,
)

REGULAR_MODES = {"A": ("000000", "100644"), "D": ("100644", "000000"), "M": ("100644", "100644")}
SKILL_FILE_MODES = ("100644", "100755")

# Kinds each scoped job is known not to read, with the directories whose
# *_test.go files it does compile. Every other kind, and every unknown path,
# runs the job. full_checks keeps its original website-only meaning for the
# remaining jobs: Lint hashes pinned documents and Go tests read documents,
# so Markdown still runs Build, Lint and Test.
JOBS = {
    "full_checks": ({"website"}, ()),
    # go test ./x/auth/client/cli, the Darwin ACL helper, the private-net
    # skill on system Python, and the Frontier Bun tests and package.
    "frontier_macos": ({"website", "markdown", "go-test"}, ("x/auth/client/cli/",)),
    # docker build of make build plus explicitly copied, non-test files.
    "deployment_images": ({"website", "markdown", "skill", "go-test"}, ()),
    # Local four-validator network, go test ./tests/multivalidator, and the
    # node, claim, shared-network and private-net skill suites.
    "consensus_rehearsal": ({"website", "markdown", "go-test"}, ("tests/multivalidator/",)),
    # go build of candidate and frozen binaries plus the rehearsal scripts.
    "accounting_rehearsal": ({"website", "markdown", "skill", "go-test"}, ()),
    # Its own module, which builds against the root module's non-test source.
    "sigstore_compiler": ({"website", "markdown", "skill", "go-test"}, ("tools/sigstore-substrate-compiler/",)),
    # Key boundary, authority graph and go test/vet of ./deploy and ./tools/...
    # The authority graph pins docs/AUTHORITATIVE-STATE.md, so Markdown runs it.
    "operations_safety": ({"website", "skill", "go-test"}, ("deploy/", "tools/")),
}


def safe_path(path):
    # Git names are untrusted input. Do not normalize a different path into an
    # allowed one.
    if not path or any(ord(c) < 32 or ord(c) == 127 for c in path):
        return False
    parts = path.split("/")
    return not any(part in ("", ".", "..") for part in parts) and "\\" not in path


def secret_shaped(path):
    parts = path.split("/")
    return any(part.lower().startswith("keyring") for part in parts) or bool(SECRET_NAME.fullmatch(parts[-1]))


def website_path(path):
    # All shared standards, Functions and unknown paths stay full.
    if not safe_path(path):
        return False
    if path in WEBSITE_FILES:
        return True
    suffix = PurePosixPath(path).suffix
    return (
        path.startswith("dashboard/src/") and suffix in (".ts", ".css")
        or path.startswith("dashboard/tests/") and path.endswith(".test.ts")
        or path.startswith("dashboard/scripts/") and suffix in (".ts", ".mjs")
    )


def diff_records(raw):
    """Parse --raw -z --no-renames into (before mode, after mode, path); None if malformed."""
    try:
        records = raw.decode("utf-8").split("\0")
    except UnicodeError:
        return None
    if len(records) < 3 or records[-1] != "" or len(records) % 2 != 1:
        return None
    parsed = []
    for index in range(0, len(records) - 1, 2):
        fields = records[index].split()
        if len(fields) != 5 or not fields[0].startswith(":") or fields[4] not in REGULAR_MODES:
            return None
        before, after, status = fields[0][1:], fields[1], fields[4]
        if (before == "000000") != (status == "A") or (after == "000000") != (status == "D"):
            return None
        for mode, object_id in ((before, fields[2]), (after, fields[3])):
            if not re.fullmatch(r"[a-f0-9]{40}", object_id) or (mode == "000000") != (object_id == "0" * 40):
                return None
        parsed.append((before, after, records[index + 1]))
    return parsed


def path_kind(before, after, path, embed_dirs=()):
    """Return the known kind of one changed path, or None when it must run everything."""
    if not safe_path(path) or secret_shaped(path):
        return None
    # go:embed reaches only files below its package directory.
    if any(path.startswith(directory) for directory in embed_dirs):
        return None
    regular = (before, after) in REGULAR_MODES.values()
    if regular and website_path(path):
        return "website"
    if path.startswith(PRIVATE_NET_SKILL):
        modes_ok = all(mode in ("000000",) + SKILL_FILE_MODES for mode in (before, after))
        return "skill" if modes_ok else None
    if regular and path.endswith("_test.go"):
        return "go-test"
    if regular and path.endswith(".md"):
        return "markdown"
    return None


def website_diff(raw):
    """True only when every record is a regular known website file; a move must pass for BOTH paths."""
    records = diff_records(raw)
    return bool(records) and all(path_kind(*record) == "website" for record in records)


def select(records, embed_dirs):
    """Map each output to whether its job must run for these diff records."""
    kinds = [(path_kind(before, after, path, embed_dirs), path) for before, after, path in records]
    selection = {}
    for job, (independent, compiled_tests) in JOBS.items():
        selection[job] = not records or any(
            kind not in independent or kind == "go-test" and path.startswith(compiled_tests)
            for kind, path in kinds
        )
    return selection


def embed_directories(revision):
    """Directories (with trailing slash) of Go files that mention go:embed at a commit."""
    found = subprocess.run(
        ["git", "grep", "-l", "-z", "-F", "go:embed", revision, "--", "*.go"],
        capture_output=True, timeout=60,
    )
    if found.returncode not in (0, 1):
        raise subprocess.SubprocessError("git grep failed")
    directories = set()
    for name in found.stdout.decode("utf-8").split("\0"):
        if name:
            path = name[len(revision) + 1:] if name.startswith(revision + ":") else None
            if path is None or not safe_path(path):
                raise subprocess.SubprocessError("unexpected git grep output")
            parent = str(PurePosixPath(path).parent)
            directories.add("" if parent == "." else parent + "/")
    return directories


def checks(event, base, head):
    everything = {job: True for job in JOBS}
    if event != "pull_request" or not all(re.fullmatch(r"[a-f0-9]{40}", ref) for ref in (base, head)):
        return everything
    try:
        diff = subprocess.run(
            ["git", "diff", "--raw", "--no-abbrev", "-z", "--no-renames", f"{base}...{head}", "--"],
            check=True, capture_output=True, timeout=30,
        )
        records = diff_records(diff.stdout)
        if records is None:
            raise ValueError("malformed diff")
        embed_dirs = embed_directories(base) | embed_directories(head)
    except (OSError, subprocess.SubprocessError, UnicodeError, ValueError):
        print("Unable to classify the pull request diff; keeping every check.", file=sys.stderr)
        return everything
    return select(records, embed_dirs)


def full_checks(event, base, head):
    return checks(event, base, head)["full_checks"]


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--event", required=True)
    parser.add_argument("--base", default="")
    parser.add_argument("--head", default="")
    args = parser.parse_args()
    for job, run in checks(args.event, args.base, args.head).items():
        print(f"{job}={str(run).lower()}")

#!/usr/bin/env python3
"""Export only committed application/build inputs into a NEW Docker context."""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path, PurePosixPath
import re
import subprocess

NETWORK = "deploy/networks/zerone-dev-1/"
FIXED = {"go.mod": "go.mod", "go.sum": "go.sum",
         "docs/swagger-ui/embed.go": "docs/swagger-ui/embed.go",
         "docs/swagger-ui/index.html": "docs/swagger-ui/index.html",
         "docs/swagger-ui/swagger.json": "docs/swagger-ui/swagger.json",
         NETWORK + "runtime.py": "runtime/runtime.py", NETWORK + "gateway.py": "runtime/gateway.py",
         NETWORK + "launch-binary-patch.py": "runtime/launch-binary-patch.py",
         NETWORK + "Dockerfile": "Dockerfile"}
MAX_BYTES = 64 * 1024 * 1024
# #169's release gate, read from this checkout. Its structural check runs here.
GATE = Path(__file__).resolve().parents[3] / "scripts" / "proposal-coherence-gate.py"


def git(repo, *args):
    env = {key: value for key, value in os.environ.items() if not key.startswith("GIT_")}
    env.update(GIT_NO_REPLACE_OBJECTS="1", GIT_CONFIG_NOSYSTEM="1", GIT_CONFIG_GLOBAL=os.devnull)
    return subprocess.run(["git", "-C", str(repo), *args], check=True, capture_output=True, env=env).stdout


def proposal_coherence(repo, commit):
    """Refuse what scripts/proposal-coherence-gate.py refuses without running tests.

    Since b8cff9c1 ProcessProposal replays the ante handler; a source in which it
    does, but PrepareProposal does not replay its candidates with #167's coherence
    tests in place, lets one public transaction stall the chain. The original
    consensus-repairs carrier heads (a72ca85e, 7253100b) are such sources. A
    source that passes this check still needs the full gate's COHERENT report
    (docs/UPGRADES.md); the gate's ancestry checks need a full clone and are not
    repeated here.
    """
    spec = importlib.util.spec_from_file_location("proposal_coherence_gate", GATE)
    if spec is None or spec.loader is None:
        raise ValueError("The proposal-coherence gate is missing from this checkout")
    gate = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(gate)
    try:
        replay = any(gate.files_naming(repo, commit, marker) for marker in gate.REPLAY_MARKERS)
        if not replay:
            return "not-affected"
        prepare = bool(gate.files_naming(repo, commit, gate.PREPARE_MARKER))
        test_file = git(repo, "ls-tree", "--name-only", commit, "--", gate.COHERENCE_TEST_FILE).strip() != b""
        missing = [name for directory, names in gate.REQUIRED_TESTS.items()
                   for name in sorted(set(names) - gate.test_functions(repo, commit, directory))]
    except gate.GateError as error:
        raise ValueError(f"Proposal-coherence check could not read the source: {error}") from error
    if not prepare or not test_file or missing:
        raise ValueError("ProcessProposal replays the ante handler but PrepareProposal does not replay its "
                         "candidates with the coherence tests in place; include cd6cd145 (#167) and run "
                         "scripts/proposal-coherence-gate.py")
    return "replays-with-coherence-tests"


def destination(name):
    if name in FIXED:
        return FIXED[name]
    path = PurePosixPath(name)
    if path.parts and path.parts[0] in ("app", "cmd", "x", "internal") and path.suffix == ".go" and not name.endswith("_test.go"):
        return name
    return None


def export(repo, commit, output):
    repo = Path(repo).resolve()
    output = Path(output).absolute()
    if not re.fullmatch(r"[0-9a-f]{40}", commit):
        raise ValueError("An exact 40-hex source commit is required")
    if git(repo, "rev-parse", "HEAD").decode().strip() != commit:
        raise ValueError("Requested source must equal checked-out HEAD")
    if git(repo, "status", "--porcelain=v1", "--untracked-files=all"):
        raise ValueError("Commit the intended source first; dirty or untracked inputs are refused")
    if output.exists() or output.is_symlink() or not output.parent.is_dir():
        raise ValueError("Output must be a new path with an existing parent")
    if output == repo or repo in output.parents:
        raise ValueError("Keep the build context outside the repository")
    proposal_coherence(repo, commit)
    entries, total = [], 0
    for row in git(repo, "ls-tree", "-rz", "--full-tree", commit).split(b"\0"):
        if not row:
            continue
        meta, raw_name = row.split(b"\t", 1)
        mode, kind, blob = meta.decode().split()
        name = raw_name.decode("utf-8")
        target = destination(name)
        if target is None:
            continue
        if mode not in ("100644", "100755") or kind != "blob" or any(part in ("", ".", "..") for part in PurePosixPath(name).parts):
            raise ValueError("Selected source must be an ordinary tracked file")
        raw = git(repo, "cat-file", "blob", blob)
        total += len(raw)
        if total > MAX_BYTES or len(entries) >= 10000:
            raise ValueError("Source context exceeds its bounded production closure")
        entries.append((name, target, mode, blob, raw))
    included = {row[0] for row in entries}
    if not set(FIXED).issubset(included) or not any(row[0].startswith("cmd/zeroned/") for row in entries):
        raise ValueError("Missing required application or runtime source")
    output.mkdir(mode=0o700)
    rows = []
    for name, target, mode, blob, raw in sorted(entries):
        path = output / target
        path.parent.mkdir(parents=True, exist_ok=True, mode=0o700)
        with path.open("xb") as stream:
            stream.write(raw)
        path.chmod(0o755 if mode == "100755" else 0o644)
        rows.append({"source_path": name, "context_path": target, "git_blob": blob, "mode": mode,
                     "bytes": len(raw), "sha256": hashlib.sha256(raw).hexdigest()})
    manifest = {"schema": "zerone-dev-source-context/v1", "source_commit": commit,
                "source_tree": git(repo, "rev-parse", commit + "^{tree}").decode().strip(),
                "version": "dev-1-" + commit[:12], "file_count": len(rows), "bytes": total, "files": rows}
    (output / "source-context.json").write_text(json.dumps(manifest, sort_keys=True, indent=2) + "\n")
    return manifest


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", required=True, type=Path)
    parser.add_argument("--commit", required=True)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    try:
        result = export(args.repo, args.commit, args.output)
    except (ValueError, OSError, subprocess.CalledProcessError) as error:
        parser.exit(1, f"Build-context export refused: {error}\n")
    print(json.dumps({key: result[key] for key in ("schema", "source_commit", "source_tree", "version", "file_count", "bytes")}))


if __name__ == "__main__":
    main()

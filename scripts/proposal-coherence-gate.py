#!/usr/bin/env python3
"""Refuse a release source whose PrepareProposal is not coherent with ProcessProposal.

Run from a full (not shallow) clone of main, with Python 3.12 or later, the
isolated interpreter and the standard library only, against the exact commit a
release binary is built from:

    python3 -I scripts/proposal-coherence-gate.py --commit <SOURCE> [--out REPORT.json]

CometBFT requires that a block a correct proposer prepares is accepted by
ProcessProposal. Since b8cff9c1 (2026-09-03) Zerone's ProcessProposal replays
every regular transaction through the ante handler with
BaseApp.ProcessProposalVerifyTx; until cd6cd145 (#167, 2026-11-04)
PrepareProposal did not replay its candidates, so a proposer could offer a
block its own ProcessProposal refuses. The need for the fix follows the
mechanism, not the branch, so the commit is read from Git objects:

- NOT_AFFECTED: no non-test Go file names ProcessProposalVerifyTx or the SDK's
  NewDefaultProposalHandler, and the commit descends from neither b8cff9c1 nor
  cd6cd145. The sources of zerone-1, H1 and H2 (SDK 0.50) and the 2026-08-01
  H3 candidate 335bb94f (SDK 0.53), all older than b8cff9c1, are of this kind.
- COHERENT: ProcessProposal replays the ante, PrepareProposal replays with
  BaseApp.PrepareProposalVerifyTx, the coherence test file and every test named
  below exist at the commit, and each of those tests, with all of its subtests,
  passes when run in a `git archive` of the commit.
- REFUSED: anything else that the gate can establish, including a descendant of
  b8cff9c1 in which no replay is found (this gate no longer recognises the
  tree), a descendant of cd6cd145 that has lost the fix or its tests, and a
  required test that fails, is skipped, stops partway (os.Exit, a fatal error,
  go test's own alarm) or never starts.

The gate exits 2 without a verdict when it cannot decide: a shallow clone or a
missing anchor commit (ancestry would be unknown), an older Python, a Go
toolchain that cannot build or start the tests, or a package whose go test does
not finish within --timeout (its whole process group is stopped). Go's output
is printed in each case.

The report goes to stdout as indented JSON with sorted keys and no timestamps,
so two full clones with the same Go toolchain produce the same bytes; `--out`
also writes it to a new file, whose SHA-256 is what release evidence binds. The
repository and its worktrees are only read. Exit status: 0 for COHERENT or
NOT_AFFECTED, 1 for REFUSED, 2 when the gate could not decide.
"""
from __future__ import annotations

import argparse
import json
import os
from pathlib import Path
import re
import shutil
import signal
import subprocess
import sys
import tarfile
import tempfile

SCHEMA = "zerone-proposal-coherence-gate-v1"
ANCHORS = {
    # The signature-hardening merge: ProcessProposal replays the ante from here.
    "ante_replay": "b8cff9c1ca2ee20fa795f5968c4434be010ac42e",
    # #167: PrepareProposal replays its candidates from here.
    "fix": "cd6cd14509472ccf711128677314e6ff2b815c99",
}
REPLAY_MARKERS = ("ProcessProposalVerifyTx", "NewDefaultProposalHandler")
PREPARE_MARKER = "PrepareProposalVerifyTx"
COHERENCE_TEST_FILE = "tests/cross_stack/prepare_proposal_coherence_test.go"
REQUIRED_TESTS = {
    "tests/cross_stack": (
        "TestPrepareProposalCoherence_KeepsValidCandidatesUnchanged",
        "TestPrepareProposalCoherence_HeightBoundedCandidate",
        "TestPrepareProposalCoherence_TimeBoundedCandidate",
        "TestPrepareProposalCoherence_DependentCandidates",
        "TestPrepareProposalCoherence_GasLimitedSelectionReplaysOnlyTakenCandidates",
        "TestPrepareProposalCoherence_ByteLimitedSelectionReplaysOnlyTakenCandidates",
        "TestPrepareProposalCoherence_EmptySelectionIsAccepted",
        "TestPrepareProposalCoherence_ProposesReplayedBytes",
    ),
    "app": (
        "TestPrepareProposalLeavesOutCandidatesWhoseReplayFailsOrPanics",
        "TestPrepareProposalAssumesNoPostHandler",
        "TestProposalTxFitsMirrorsDefaultSelector",
    ),
}
TEST_FUNC = re.compile(r"^func (Test[A-Za-z0-9_]+)\(")
OUTPUT_TAIL = 40


class GateError(Exception):
    """The gate could not establish what it needs to decide."""


def git(repo, *args, check=True, binary=False):
    result = subprocess.run(["git", "-C", str(repo), *args], capture_output=True)
    result.stderr = result.stderr.decode(errors="replace")
    if check and result.returncode != 0:
        raise GateError(f"git {args[0]} failed: {result.stderr.strip()}")
    if not binary:
        result.stdout = result.stdout.decode(errors="replace")
    return result


def repository_root(repo):
    """The work tree's top level (or the bare repository), so paths are whole-tree."""
    top = git(repo, "rev-parse", "--show-toplevel", check=False)
    if top.returncode == 0 and top.stdout.strip():
        return Path(top.stdout.strip())
    if git(repo, "rev-parse", "--is-bare-repository", check=False).stdout.strip() == "true":
        return Path(repo)
    raise GateError(f"{repo} is not a Git repository")


def require_full_history(repo, anchors):
    if git(repo, "rev-parse", "--is-shallow-repository").stdout.strip() != "false":
        raise GateError("this is a shallow clone, so ancestry is unknown; run from a full clone")
    for name, anchor in sorted(anchors.items()):
        if git(repo, "cat-file", "-e", f"{anchor}^{{commit}}", check=False).returncode != 0:
            raise GateError(f"anchor {name} ({anchor}) is not in this clone; fetch main")


def resolve(repo, revision):
    if not revision or revision.startswith("-"):
        raise GateError("--commit must name a commit")
    result = git(repo, "rev-parse", "--verify", "--quiet", "--end-of-options", f"{revision}^{{commit}}", check=False)
    if result.returncode != 0:
        raise GateError(f"{revision} is not a commit in {repo}")
    return result.stdout.strip()


def files_naming(repo, commit, marker):
    """Non-test Go files at commit that contain marker as a word."""
    result = git(repo, "grep", "--no-color", "-l", "-z", "-w", "-F", "-e", marker, commit, "--",
                 ":(top)*.go", ":(top,exclude)*_test.go", check=False)
    if result.returncode not in (0, 1):
        raise GateError(f"git grep failed: {result.stderr.strip()}")
    prefix = commit + ":"
    return sorted(entry[len(prefix):] if entry.startswith(prefix) else entry
                  for entry in result.stdout.split("\0") if entry)


def test_functions(repo, commit, directory):
    """Top-level Test functions declared in directory's _test.go files at commit."""
    listing = git(repo, "ls-tree", "-z", "--name-only", f"{commit}:{directory}", check=False)
    if listing.returncode != 0:
        return set()
    names = set()
    for name in listing.stdout.split("\0"):
        if not name.endswith("_test.go"):
            continue
        blob = git(repo, "cat-file", "blob", f"{commit}:{directory}/{name}", binary=True).stdout
        for line in blob.decode(errors="replace").splitlines():
            match = TEST_FUNC.match(line)
            if match:
                names.add(match.group(1))
    return names


def contains(repo, anchor, commit):
    result = git(repo, "merge-base", "--is-ancestor", anchor, commit, check=False)
    if result.returncode in (0, 1):
        return result.returncode == 0
    raise GateError(f"git merge-base failed: {result.stderr.strip()}")


def inspect(repo, commit, anchors=ANCHORS):
    replay = [{"marker": marker, "path": path}
              for marker in REPLAY_MARKERS for path in files_naming(repo, commit, marker)]
    prepare = [{"marker": PREPARE_MARKER, "path": path}
               for path in files_naming(repo, commit, PREPARE_MARKER)]
    test_file = git(repo, "cat-file", "-e", f"{commit}:{COHERENCE_TEST_FILE}", check=False).returncode == 0
    missing = {directory: sorted(set(names) - test_functions(repo, commit, directory))
               for directory, names in REQUIRED_TESTS.items()}
    return {
        "mechanism": {
            "process_proposal_replays_ante": bool(replay),
            "replay_markers": replay,
            "prepare_proposal_replays": bool(prepare),
            "prepare_markers": prepare,
            "coherence_test_file": COHERENCE_TEST_FILE,
            "coherence_test_file_present": test_file,
            "missing_tests": {directory: names for directory, names in missing.items() if names},
        },
        "ancestry": {name: {"commit": anchor, "contained": contains(repo, anchor, commit)}
                     for name, anchor in sorted(anchors.items())},
    }


def extract(repo, commit, destination):
    archive = destination / "source.tar"
    with archive.open("wb") as stream:
        result = subprocess.run(["git", "-C", str(repo), "archive", "--format=tar", commit],
                                stdout=stream, stderr=subprocess.PIPE)
    if result.returncode != 0:
        raise GateError(f"git archive failed: {result.stderr.decode(errors='replace').strip()}")
    source = destination / "source"
    source.mkdir()
    with tarfile.open(archive) as tar:
        tar.extractall(source, filter="data")
    archive.unlink()
    return source


def tail(text):
    return "\n".join(text.splitlines()[-OUTPUT_TAIL:])


def run_bounded(argv, cwd, env, timeout):
    """Run argv in its own process group; on timeout stop the whole group, test binaries included."""
    try:
        process = subprocess.Popen(argv, cwd=cwd, env=env, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                                   text=True, start_new_session=True)
    except OSError as error:
        raise GateError(f"could not run {argv[0]}: {error}") from error
    try:
        stdout, stderr = process.communicate(timeout=timeout)
    except subprocess.TimeoutExpired:
        os.killpg(process.pid, signal.SIGKILL)
        stdout, stderr = process.communicate()
        raise GateError(f"{' '.join(argv[:2])} {argv[-1]} did not finish within {timeout} s and was stopped:\n"
                        + tail(stdout + "\n" + stderr)) from None
    return subprocess.CompletedProcess(argv, process.returncode, stdout, stderr)


def go_test(go, source, env, directory, names, timeout, log):
    """Run the named tests in one package and report each one's outcome."""
    package = f"./{directory}"
    pattern = "^(" + "|".join(names) + ")$"
    result = run_bounded([go, "test", "-json", "-count=1", "-run", pattern, package], source, env, timeout)
    actions, started, unsound, output = {}, set(), set(), []
    for line in result.stdout.splitlines():
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            output.append(line)
            continue
        if not isinstance(event, dict):
            continue
        if isinstance(event.get("Output"), str):
            output.append(event["Output"].rstrip("\n"))
        test, action = event.get("Test"), event.get("Action")
        if not isinstance(test, str) or action not in ("run", "pass", "fail", "skip"):
            continue
        parent, _, subtest = test.partition("/")
        if parent not in names:
            continue
        if subtest:
            if action in ("fail", "skip"):
                unsound.add(parent)
        elif action == "run":
            started.add(test)
        else:
            actions[test] = action
    if result.returncode != 0 and not started:
        raise GateError(f"go test {package} exited {result.returncode} before any required test started:\n"
                        + tail("\n".join(output) + "\n" + result.stderr))
    record = {
        "package": package,
        "exit_code": result.returncode,
        "passed": sorted(name for name in names if actions.get(name) == "pass" and name not in unsound),
        "failed": sorted(name for name in names if actions.get(name) == "fail"),
        "skipped": sorted(name for name in names if actions.get(name) == "skip"),
        # Started, then the test binary stopped (os.Exit, a fatal error, go test's own alarm).
        "interrupted": sorted(name for name in names if name in started and name not in actions),
        "missing": sorted(name for name in names if name not in started and name not in actions),
        "subtests_not_passed": sorted(unsound),
    }
    if result.returncode != 0 or len(record["passed"]) != len(names):
        log(f"go test {package} output (last {OUTPUT_TAIL} lines):\n" + tail("\n".join(output) + "\n" + result.stderr))
    return record


def go_version(go, source, env):
    result = run_bounded([go, "env", "GOVERSION"], source, env, 300)
    if result.returncode != 0 or not result.stdout.strip():
        raise GateError(f"{go} env GOVERSION failed in the source archive: {tail(result.stderr)}")
    return result.stdout.strip()


def run_tests(repo, commit, go, work_dir, timeout, log):
    env = {**os.environ, "GOWORK": "off", "GOFLAGS": "-mod=readonly"}
    directory = Path(tempfile.mkdtemp(prefix="zerone-proposal-coherence-", dir=work_dir))
    try:
        directory.chmod(0o700)
        source = extract(repo, commit, directory)
        version = go_version(go, source, env)
        packages = [go_test(go, source, env, package, names, timeout, log)
                    for package, names in REQUIRED_TESTS.items()]
    finally:
        shutil.rmtree(directory, ignore_errors=True)
    return {"go_version": version, "packages": packages}


def tests_passed(tests):
    return all(package["exit_code"] == 0 and len(package["passed"]) == len(REQUIRED_TESTS[package["package"][2:]])
               for package in tests["packages"])


def decide(found, tests=None):
    mechanism, ancestry = found["mechanism"], found["ancestry"]
    fixed = (mechanism["prepare_proposal_replays"] and mechanism["coherence_test_file_present"]
             and not mechanism["missing_tests"])
    if not mechanism["process_proposal_replays_ante"]:
        if ancestry["ante_replay"]["contained"] or ancestry["fix"]["contained"]:
            return "REFUSED", ("descends from the commit that made ProcessProposal replay the ante, "
                               "but no replay marker was found; this gate no longer recognises the tree")
        return "NOT_AFFECTED", "ProcessProposal does not replay the ante handler"
    if not fixed:
        return "REFUSED", ("ProcessProposal replays the ante handler but PrepareProposal does not replay "
                           "its candidates with the coherence tests in place; include cd6cd145 (#167)")
    if tests is None:
        return None, "the coherence tests must run"
    if not tests_passed(tests):
        return "REFUSED", "a required coherence test did not pass at this commit"
    return "COHERENT", "PrepareProposal replays its candidates and every required coherence test passed"


def gate(repo, revision, go="go", work_dir=None, timeout=1800, anchors=ANCHORS, log=None):
    log = log or (lambda message: print(message, file=sys.stderr))
    if not hasattr(tarfile, "data_filter"):
        raise GateError("this gate needs Python 3.12 or later (tarfile extraction filters)")
    repo = repository_root(repo)
    require_full_history(repo, anchors)
    commit = resolve(repo, revision)
    found = inspect(repo, commit, anchors)
    verdict, reason = decide(found)
    tests = None
    if verdict is None:
        tests = run_tests(repo, commit, go, work_dir, timeout, log)
        verdict, reason = decide(found, tests)
    report = {
        "schema": SCHEMA,
        "commit": commit,
        "tree": git(repo, "rev-parse", f"{commit}^{{tree}}").stdout.strip(),
        **found,
        "tests": tests,
        "verdict": verdict,
        "reason": reason,
    }
    return report, 0 if verdict in ("COHERENT", "NOT_AFFECTED") else 1


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    parser.add_argument("--commit", required=True, help="the release source commit")
    parser.add_argument("--repo", type=Path, default=Path(__file__).resolve().parent.parent,
                        help="repository holding the commit (default: this checkout)")
    parser.add_argument("--go", default="go", help="Go toolchain to test with (default: go)")
    parser.add_argument("--work-dir", type=Path, help="parent for the temporary source archive")
    parser.add_argument("--timeout", type=int, default=1800,
                        help="seconds before one package's go test (build and run) is stopped")
    parser.add_argument("--out", type=Path, help="also write the report to this new file")
    args = parser.parse_args(argv)
    try:
        report, status = gate(args.repo, args.commit, go=args.go, work_dir=args.work_dir,
                              timeout=args.timeout)
    except (GateError, OSError, tarfile.TarError) as error:
        print(f"proposal-coherence-gate: could not decide: {error}", file=sys.stderr)
        return 2
    except Exception as error:  # exit 1 means REFUSED, so a crash must not end with it
        print(f"proposal-coherence-gate: internal error: {error!r}", file=sys.stderr)
        return 2
    text = json.dumps(report, indent=2, sort_keys=True) + "\n"
    if args.out:
        try:
            with args.out.open("x") as stream:
                stream.write(text)
        except OSError as error:
            print(f"proposal-coherence-gate: {error}", file=sys.stderr)
            return 2
    sys.stdout.write(text)
    print(f"proposal-coherence-gate: {report['verdict']}: {report['reason']}", file=sys.stderr)
    return status


if __name__ == "__main__":
    raise SystemExit(main())

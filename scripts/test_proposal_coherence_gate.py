#!/usr/bin/env python3
"""Synthetic-history tests for proposal-coherence-gate.py (stdlib and git only).

Run: python3 -I -B -m unittest discover -s scripts -p 'test_proposal_coherence_gate.py' -v
"""

import contextlib
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

SPEC = importlib.util.spec_from_file_location("proposal_coherence_gate", Path(__file__).with_name("proposal-coherence-gate.py"))
gate = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(gate)

ROOT = Path(__file__).resolve().parent.parent
UNAFFECTED = "package app\n\nfunc (app *ZeroneApp) processProposal() {}\n"
REPLAYING = "package app\n\nfunc (app *ZeroneApp) processProposal() { app.BaseApp.ProcessProposalVerifyTx(nil) }\n"
FIXED = REPLAYING + "\nfunc (app *ZeroneApp) prepare() { app.BaseApp.PrepareProposalVerifyTx(nil) }\n"
# A stand-in go: records each call, reports the version file of its working
# directory, and prints the test2json events its plan names for a package.
FAKE_GO = """#!{python}
import json, os, sys, time
plan = json.load(open(os.environ["FAKE_GO_PLAN"]))
def read(name):
    return open(name).read() if os.path.exists(name) else None
with open(plan["record"], "a") as log:
    log.write(json.dumps({{"argv": sys.argv[1:], "cwd": os.getcwd(), "GOWORK": os.environ.get("GOWORK"),
                          "GOFLAGS": os.environ.get("GOFLAGS"), "source": read("marker.txt")}}) + "\\n")
if sys.argv[1:] == ["env", "GOVERSION"]:
    print(read("go-version.txt") or "go1.0-fake")
    sys.exit(0)
result = plan["results"].get(sys.argv[-1], {{}})
if result.get("spawn"):
    import subprocess
    child = subprocess.Popen([sys.executable, "-c", "import time; time.sleep(60)"],
                             stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    open(result["spawn"], "w").write(str(child.pid))
for name, action in result.get("events", []):
    print(json.dumps({{"Action": "run", "Package": "zerone", "Test": name}}))
    print(json.dumps({{"Action": "output", "Package": "zerone", "Test": name, "Output": name + " says hello\\n"}}))
    if action != "run":
        print(json.dumps({{"Action": action, "Package": "zerone", "Test": name}}))
print("not json: build noise", flush=True)
time.sleep(result.get("sleep", 0))
sys.stderr.write(result.get("stderr", ""))
sys.exit(result.get("exit", 0))
"""
PACKAGES = tuple(gate.REQUIRED_TESTS)


def test_file(names):
    return "package x_test\n\n" + "".join(f"func {name}(t *testing.T) {{}}\n" for name in names)


def required(package):
    return gate.REQUIRED_TESTS[package]


def passing(package):
    return {"exit": 0, "events": [[name, "pass"] for name in required(package)]}


def fixed_tree(source="fixed"):
    return {
        "go.mod": "module zerone\n",
        "marker.txt": source,
        "app/abci.go": FIXED,
        "app/abci_test.go": test_file(required("app")),
        gate.COHERENCE_TEST_FILE: test_file(required("tests/cross_stack")),
    }


class GitCase(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="zerone-coherence-gate-test-")
        self.addCleanup(self.tmp.cleanup)
        self.base = Path(self.tmp.name)
        self.repo = self.base / "repo"
        self.repo.mkdir()
        self.git("init", "-q", "-b", "main")
        # Anchors outside the candidates' history, so a candidate descends from neither.
        empty = self.git("hash-object", "-t", "tree", "--stdin", input="")
        ante = self.git("commit-tree", empty, "-m", "ante replay anchor")
        fix = self.git("commit-tree", empty, "-p", ante, "-m", "fix anchor")
        self.git("update-ref", "refs/anchors/fix", fix)
        self.anchors = {"ante_replay": ante, "fix": fix}
        self.go = self.base / "go"
        self.go.write_text(FAKE_GO.format(python=sys.executable))
        self.go.chmod(0o755)
        self.record = self.base / "go-calls.jsonl"
        self.work = self.base / "work"
        self.work.mkdir()
        self.plan_path = self.base / "plan.json"
        self.plan({package: passing(package) for package in PACKAGES})
        self.logs = []
        patcher = mock.patch.dict(os.environ, {"FAKE_GO_PLAN": str(self.plan_path)})
        patcher.start()
        self.addCleanup(patcher.stop)

    def git(self, *args, input=None, cwd=None):
        return subprocess.run(["git", "-C", str(cwd or self.repo), "-c", "user.name=t",
                               "-c", "user.email=t@example.invalid", "-c", "commit.gpgsign=false",
                               "-c", "tag.gpgsign=false", *args],
                              check=True, capture_output=True, text=True, input=input).stdout.strip()

    def commit(self, files, message="c"):
        for path, text in files.items():
            target = self.repo / path
            if text is None:
                target.unlink(missing_ok=True)
                continue
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_text(text)
        self.git("add", "-A")
        self.git("commit", "-q", "--allow-empty", "-m", message)
        return self.git("rev-parse", "HEAD")

    def candidate(self, files, later):
        """Commit files, then a later commit that would flip the verdict if the gate read HEAD."""
        commit = self.commit(files, "candidate")
        self.commit(later, "later")
        return commit

    def plan(self, results):
        results = {(package if package.startswith("./") else f"./{package}"): result
                   for package, result in results.items()}
        self.plan_path.write_text(json.dumps({"record": str(self.record), "results": results}))

    def calls(self):
        if not self.record.exists():
            return []
        return [json.loads(line) for line in self.record.read_text().splitlines()]

    def go_test_calls(self):
        return [call for call in self.calls() if call["argv"][0] == "test"]

    def run_gate(self, revision, anchors=None, repo=None, **options):
        return gate.gate(repo or self.repo, revision, go=str(self.go), work_dir=self.work,
                         anchors=anchors or self.anchors, log=self.logs.append, **options)

    def main(self, *argv):
        stdout, stderr = io.StringIO(), io.StringIO()
        with mock.patch.dict(gate.ANCHORS, self.anchors), contextlib.redirect_stdout(stdout), \
                contextlib.redirect_stderr(stderr):
            status = gate.main(["--repo", str(self.repo), "--go", str(self.go), "--work-dir", str(self.work), *argv])
        return status, stdout.getvalue(), stderr.getvalue()


class VerdictTests(GitCase):
    def test_tree_without_replay_is_not_affected_and_runs_no_tests(self):
        commit = self.candidate({"go.mod": "module zerone\n", "app/abci.go": UNAFFECTED,
                                 "app/abci_test.go": "package app\n// ProcessProposalVerifyTx in a test only\n"},
                                {"app/abci.go": REPLAYING})
        report, status = self.run_gate(commit)
        self.assertEqual((report["verdict"], status), ("NOT_AFFECTED", 0))
        self.assertIsNone(report["tests"])
        self.assertEqual(self.calls(), [])
        self.assertEqual(report["commit"], commit)
        self.assertEqual(report["tree"], self.git("rev-parse", f"{commit}^{{tree}}"))
        self.assertEqual(report["ancestry"]["ante_replay"]["contained"], False)

    def test_sdk_default_proposal_handler_counts_as_a_replay(self):
        commit = self.candidate({"go.mod": "module zerone\n",
                                 "app/app.go": "package app\nvar _ = baseapp.NewDefaultProposalHandler\n"},
                                {"app/app.go": UNAFFECTED})
        report, status = self.run_gate(commit)
        self.assertEqual((report["verdict"], status), ("REFUSED", 1))
        self.assertTrue(report["mechanism"]["process_proposal_replays_ante"])

    def test_each_missing_part_of_the_fix_is_refused_without_running_tests(self):
        cases = {
            "no PrepareProposal replay": ({"app/abci.go": REPLAYING}, "prepare_proposal_replays", False),
            "no coherence test file": ({gate.COHERENCE_TEST_FILE: None,
                                        "tests/cross_stack/other_test.go": test_file(required("tests/cross_stack"))},
                                       "coherence_test_file_present", False),
            "an app test missing": ({"app/abci_test.go": test_file(required("app")[1:])},
                                    "missing_tests", {"app": [required("app")[0]]}),
            "a cross-stack test missing": ({gate.COHERENCE_TEST_FILE: test_file(required("tests/cross_stack")[:-1])},
                                           "missing_tests", {"tests/cross_stack": [required("tests/cross_stack")[-1]]}),
        }
        for label, (change, field, value) in cases.items():
            with self.subTest(label):
                self.commit({**fixed_tree(), "tests/cross_stack/other_test.go": None})
                commit = self.candidate(change, {**fixed_tree("later"), "tests/cross_stack/other_test.go": None})
                report, status = self.run_gate(commit)
                self.assertEqual((report["verdict"], status), ("REFUSED", 1))
                self.assertEqual(report["mechanism"][field], value)
                self.assertEqual(self.calls(), [])

    def test_fixed_tree_is_coherent_after_its_tests_run_in_an_archive_of_that_commit(self):
        commit = self.candidate({**fixed_tree("at-commit"), "go-version.txt": "go1.0-candidate"},
                                {"marker.txt": "later-head", "go-version.txt": "go1.0-later"})
        (self.repo / "marker.txt").write_text("worktree edit")
        report, status = self.run_gate(commit)
        self.assertEqual((report["verdict"], status), ("COHERENT", 0), report["reason"])
        self.assertEqual(report["tests"]["go_version"], "go1.0-candidate")
        calls = self.go_test_calls()
        self.assertEqual(sorted(call["argv"][-1] for call in calls), ["./app", "./tests/cross_stack"])
        for call in self.calls():
            self.assertEqual(call["source"], "at-commit")
            self.assertEqual((call["GOWORK"], call["GOFLAGS"]), ("off", "-mod=readonly"))
            self.assertTrue(Path(call["cwd"]).resolve().is_relative_to(self.work.resolve()))
        for call in calls:
            names = required(call["argv"][-1][2:])
            self.assertEqual(call["argv"][:5], ["test", "-json", "-count=1", "-run", "^(" + "|".join(names) + ")$"])
        self.assertEqual(list(self.work.iterdir()), [], "the temporary archive must be removed")
        self.assertEqual((self.repo / "marker.txt").read_text(), "worktree edit")

    def test_any_failed_skipped_missing_or_unsound_test_is_refused_in_either_package(self):
        for package in PACKAGES:
            names = list(required(package))
            modes = {
                "failed": (1, [[names[0], "fail"]] + [[name, "pass"] for name in names[1:]]),
                "skipped": (0, [[names[0], "skip"]] + [[name, "pass"] for name in names[1:]]),
                "missing": (0, [[name, "pass"] for name in names[1:]]),
                "nonzero exit": (1, [[name, "pass"] for name in names]),
                "skipped subtest": (0, [[name, "pass"] for name in names] + [[names[-1] + "/case", "skip"]]),
                "failed subtest": (1, [[names[0] + "/case", "fail"]] + [[name, "pass"] for name in names]),
                "stopped in the first test": (2, [[names[0], "run"]]),
                "stopped in the last test": (2, [[name, "pass"] for name in names[:-1]] + [[names[-1], "run"]]),
            }
            for mode, (exit_code, events) in modes.items():
                with self.subTest(package=package, mode=mode):
                    self.logs.clear()
                    self.commit(fixed_tree())
                    self.plan({**{other: passing(other) for other in PACKAGES},
                               package: {"exit": exit_code, "events": events}})
                    report, status = self.run_gate(self.git("rev-parse", "HEAD"))
                    self.assertEqual((report["verdict"], status), ("REFUSED", 1))
                    self.assertTrue(self.logs, "a failing package's output must be shown")
                    if mode.startswith("stopped"):
                        record = next(p for p in report["tests"]["packages"] if p["package"] == f"./{package}")
                        self.assertEqual(record["interrupted"], [events[-1][0]])

    def test_ancestry_contradicting_the_scan_is_refused(self):
        hardening = self.commit({"go.mod": "module zerone\n", "app/abci.go": REPLAYING}, "hardening")
        fix = self.commit(fixed_tree(), "fix")
        hidden = self.commit({"app/abci.go": UNAFFECTED + "// replay moved where this gate cannot see\n"})
        for label, anchors in (("ante replay anchor", {"ante_replay": hardening, "fix": self.anchors["fix"]}),
                               ("fix anchor alone", {"ante_replay": self.anchors["ante_replay"], "fix": fix})):
            with self.subTest(label):
                report, status = self.run_gate(hidden, anchors)
                self.assertEqual((report["verdict"], status), ("REFUSED", 1))

    def test_reverting_the_fix_after_it_landed_is_refused(self):
        hardening = self.commit({"go.mod": "module zerone\n", "app/abci.go": REPLAYING}, "hardening")
        fix = self.commit(fixed_tree(), "fix")
        revert = self.commit({"app/abci.go": REPLAYING}, "revert")
        report, status = self.run_gate(revert, {"ante_replay": hardening, "fix": fix})
        self.assertEqual((report["verdict"], status), ("REFUSED", 1))
        self.assertTrue(report["ancestry"]["fix"]["contained"])
        self.assertEqual(self.calls(), [])

    def test_whole_tree_is_scanned_from_a_subdirectory_argument(self):
        commit = self.commit({"go.mod": "module zerone\n", "app/abci.go": UNAFFECTED,
                              "x/proposal/handler.go": "package proposal\nvar _ = ProcessProposalVerifyTx\n"})
        report, status = self.run_gate(commit, repo=self.repo / "app")
        self.assertEqual((report["verdict"], status), ("REFUSED", 1))
        self.assertEqual(report["mechanism"]["replay_markers"],
                         [{"marker": "ProcessProposalVerifyTx", "path": "x/proposal/handler.go"}])

    def test_tags_and_abbreviated_commits_resolve_to_the_peeled_commit(self):
        commit = self.candidate(fixed_tree(), {"marker.txt": "later"})
        self.git("tag", "-a", "-m", "release", "release-1", commit)
        for revision in ("release-1", commit[:12]):
            with self.subTest(revision):
                report, _ = self.run_gate(revision)
                self.assertEqual(report["commit"], commit)
                self.assertEqual(report["tree"], self.git("rev-parse", f"{commit}^{{tree}}"))


class UndecidedTests(GitCase):
    def test_shallow_clone_and_missing_anchor_cannot_decide(self):
        self.commit(fixed_tree())
        self.commit({"marker.txt": "second"})
        shallow = self.base / "shallow"
        subprocess.run(["git", "clone", "-q", "--depth", "1", f"file://{self.repo}", str(shallow)],
                       check=True, capture_output=True)
        with self.assertRaisesRegex(gate.GateError, "shallow"):
            self.run_gate("HEAD", repo=shallow)
        with self.assertRaisesRegex(gate.GateError, "anchor"):
            self.run_gate("HEAD", anchors={"ante_replay": "1" * 40, "fix": self.anchors["fix"]})

    def test_go_that_cannot_run_or_finish_the_tests_exits_two(self):
        commit = self.commit(fixed_tree())
        spawned = self.base / "grandchild.pid"
        for label, plan, shown in (
            ("no events", {"exit": 1, "stderr": "go: go.mod requires go >= 9.9\n"}, "go.mod requires go >= 9.9"),
            ("timeout", {"sleep": 30, "spawn": str(spawned), "events": [[required("tests/cross_stack")[0], "run"]]},
             "says hello"),
        ):
            with self.subTest(label):
                self.plan({"./tests/cross_stack": plan, "./app": passing("app")})
                status, stdout, stderr = self.main("--commit", commit, "--timeout", "3")
                self.assertEqual((status, stdout), (2, ""))
                self.assertIn("could not decide", stderr)
                self.assertIn(shown, stderr)
        pid = int(spawned.read_text())
        for _ in range(50):
            try:
                os.kill(pid, 0)
            except ProcessLookupError:
                break
            subprocess.run(["sleep", "0.1"])
        else:
            os.kill(pid, 9)
            self.fail("a test binary outlived the gate's timeout")
        status, stdout, _ = self.main("--commit", commit, "--go", str(self.base / "absent-go"))
        self.assertEqual((status, stdout), (2, ""))

    def test_bad_revisions_exit_two_never_one(self):
        self.commit({"go.mod": "module zerone\n", "app/abci.go": UNAFFECTED})
        for revision in ("no-such-commit", "--help-me"):
            with self.subTest(revision):
                status, stdout, _ = self.main(f"--commit={revision}")
                self.assertEqual((status, stdout), (2, ""))


class ReportTests(GitCase):
    def test_cli_report_is_deterministic_and_out_never_overwrites(self):
        commit = self.commit(fixed_tree())
        out = self.base / "report.json"
        first = self.main("--commit", commit, "--out", str(out))
        second = self.main("--commit", commit)
        self.assertEqual((first[0], second[0]), (0, 0))
        self.assertEqual(first[1], second[1])
        self.assertEqual(out.read_text(), first[1])
        self.assertEqual(json.loads(first[1])["schema"], gate.SCHEMA)
        self.assertEqual(self.main("--commit", commit, "--out", str(out))[0], 2)
        self.assertEqual(out.read_text(), first[1])

    def test_gate_leaves_the_repository_unchanged(self):
        commit = self.commit({**fixed_tree(), "notes.txt": "committed"})
        (self.repo / "marker.txt").write_text("unstaged tracked edit")
        (self.repo / "notes.txt").write_text("staged edit")
        self.git("add", "notes.txt")
        (self.repo / "untracked.txt").write_text("keep me")

        def state():
            return [self.git(*args) for args in (("status", "--porcelain", "--untracked-files=all"), ("diff",),
                                                 ("diff", "--cached"), ("rev-parse", "HEAD"), ("worktree", "list"),
                                                 ("for-each-ref",), ("stash", "list"))]

        before = state()
        report, _ = self.run_gate(commit)
        self.assertEqual(report["verdict"], "COHERENT")
        self.assertEqual(state(), before)


class RepositoryTests(unittest.TestCase):
    """The gate's markers, anchors and required tests must still describe this checkout."""

    @classmethod
    def setUpClass(cls):
        shallow = gate.git(ROOT, "rev-parse", "--is-shallow-repository").stdout.strip()
        cls.full = shallow == "false"

    def test_checkout_still_carries_every_marker_and_required_test(self):
        found = gate.inspect(ROOT, "HEAD", {})["mechanism"]
        self.assertTrue(found["process_proposal_replays_ante"])
        self.assertTrue(found["prepare_proposal_replays"])
        self.assertTrue(found["coherence_test_file_present"])
        self.assertEqual(found["missing_tests"], {})

    def test_release_checklists_require_the_gate(self):
        for path in ("docs/UPGRADES.md", "deploy/networks/zerone-2/GO-NO-GO.md"):
            with self.subTest(path):
                text = (ROOT / path).read_text()
                for needed in ("scripts/proposal-coherence-gate.py", "`COHERENT`", "`NOT_AFFECTED`", "full clone"):
                    self.assertIn(needed, text)

    def test_anchors_are_the_commits_that_introduced_their_markers(self):
        if not self.full:
            self.skipTest("a shallow clone cannot show the anchors")
        for name, marker in (("ante_replay", "ProcessProposalVerifyTx"), ("fix", gate.PREPARE_MARKER)):
            with self.subTest(name):
                anchor = gate.ANCHORS[name]
                self.assertTrue(gate.contains(ROOT, anchor, "HEAD"), f"{name} must be in this checkout's history")
                self.assertTrue(gate.files_naming(ROOT, anchor, marker))
                self.assertEqual(gate.files_naming(ROOT, gate.resolve(ROOT, anchor + "^1"), marker), [])

    def test_known_sources_classify_as_recorded(self):
        if not self.full:
            self.skipTest("a shallow clone cannot show ancestry")
        known = {
            "2e37c4c": "NOT_AFFECTED",   # zerone-1's running source
            "65c19cd8": "NOT_AFFECTED",  # H1 source pin
            "36728afb": "NOT_AFFECTED",  # H2 source pin
            "335bb94f": "NOT_AFFECTED",  # H3 candidate, 2026-08-01
            "b8cff9c1": "REFUSED",       # the signature-hardening merge
            "5542b221": "REFUSED",       # zerone-dev-1 base before its binary patch
            "cd6cd145": None,            # #167: decided only by running its tests
        }
        for revision, verdict in known.items():
            with self.subTest(revision):
                if gate.git(ROOT, "cat-file", "-e", f"{revision}^{{commit}}", check=False).returncode != 0:
                    self.skipTest(f"{revision} is not in this clone")
                found = gate.inspect(ROOT, gate.resolve(ROOT, revision), gate.ANCHORS)
                self.assertEqual(gate.decide(found)[0], verdict)


if __name__ == "__main__":
    unittest.main()

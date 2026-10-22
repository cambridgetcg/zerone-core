#!/usr/bin/env python3
"""Offline guard for the split rehearsal's coverage and binary ownership.

Actionlint validates YAML/Actions syntax separately. These checks deliberately
pin the scheduling and suite contracts, not the unrelated jobs in this file.
"""
from pathlib import Path
import re
import unittest


WORKFLOW = Path(__file__).resolve().parents[1] / ".github/workflows/ci.yml"
PIN = "a4272bfa9953e979463d3a310985bbacd5838a8e"
CONDITION = "${{ always() && !cancelled() && !github.event.pull_request.draft && (github.event_name != 'pull_request' || needs.scope.result != 'success' || needs.scope.outputs.consensus_rehearsal != 'false') }}"
SUITES = {
    "consensus-rehearsal": (
        ("scripts", "test_local_node.py"),
        ("scripts", "test_claim_workflow*.py"),
        ("skills/run-a-private-zerone-net/scripts", "test_*.py"),
    ),
    "consensus-shared-workflows": (
        ("deploy/networks/zerone-dev-1", "test_*.py"),
        ("scripts", "test_shared_claims*.py"),
        ("tools/research-review", "test_workflow_native.py"),
    ),
}


class RehearsalCIContractTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        text = WORKFLOW.read_text()
        cls.jobs = dict(re.findall(r"^  ([a-z][a-z0-9-]*):\n(.*?)(?=^  [a-z][a-z0-9-]*:\n|\Z)", text, re.M | re.S))

    def test_parallel_jobs_keep_original_scope_draft_guard_and_cap(self):
        for name in SUITES:
            with self.subTest(job=name):
                job = self.jobs[name]
                self.assertEqual(re.findall(r"^    needs: (.+)$", job, re.M), ["scope"])
                self.assertEqual(re.findall(r"^    if: (.+)$", job, re.M), [CONDITION])
                self.assertEqual(re.findall(r"^    timeout-minutes: (.+)$", job, re.M), ["30"])
                self.assertIn("    runs-on: ubuntu-24.04\n", job)

    def test_every_original_discovery_runs_once_with_its_binary_owner(self):
        for name, expected in SUITES.items():
            with self.subTest(job=name):
                actual = re.findall(r"python3 -B -m unittest discover -s (\S+) -p '([^']+)' -v", self.jobs[name])
                self.assertCountEqual(actual, expected)
        candidate = self.jobs["consensus-rehearsal"]
        shared = self.jobs["consensus-shared-workflows"]
        self.assertEqual(candidate.count("bash scripts/local-consensus-rehearsal.sh --keep"), 1)
        self.assertNotIn("local-consensus-rehearsal.sh", shared)
        for variable in ("ZERONE_LOCAL_NODE_TEST_BINARY", "ZERONE_PRIVATE_NET_TEST_BINARY"):
            self.assertIn(f'echo "{variable}=${{binaries[0]}}" >> "$GITHUB_ENV"', candidate)
            self.assertNotIn(variable, shared)
        for variable in ("ZERONE_SHARED_TEST_BINARY", "ZERONE_RESEARCH_TEST_BINARY"):
            self.assertIn(f'echo "{variable}=$RUNNER_TEMP/zerone-knowledge-11/zeroned"', shared)
            self.assertNotIn(variable, candidate)
        self.assertIn('          } >> "$GITHUB_ENV"', shared)

    def test_frozen_build_keeps_exact_pin_and_is_not_duplicated(self):
        candidate = self.jobs["consensus-rehearsal"]
        shared = self.jobs["consensus-shared-workflows"]
        self.assertNotIn(PIN, candidate)
        self.assertEqual(shared.count(PIN), 3)  # checkout, exact HEAD check, version metadata
        self.assertIn(f"          ref: {PIN}\n", shared)
        self.assertIn(f'test "$(git -C knowledge-11-source rev-parse HEAD)" = {PIN}', shared)
        self.assertIn(f"version.Commit={PIN}", shared)
        self.assertEqual(shared.count("go build -mod=readonly -trimpath -buildvcs=true"), 1)
        self.assertIn('git diff --exit-code HEAD', shared)
        self.assertIn('test -z "$(git status --porcelain)"', shared)

    def test_signing_still_waits_for_both_rehearsal_halves(self):
        signing = self.jobs["sign-release-components"]
        dependencies = re.search(r"^    needs:\n((?:      - [a-z0-9-]+\n)+)", signing, re.M)
        self.assertIsNotNone(dependencies)
        assert dependencies is not None
        for name in SUITES:
            self.assertIn(f"      - {name}\n", dependencies[1])

    def test_artifacts_remain_separate_always_uploaded_public_evidence(self):
        names = []
        for name in SUITES:
            job = self.jobs[name]
            self.assertEqual(job.count("        if: always()"), 1)
            self.assertEqual(job.count("          retention-days: 30"), 1)
            self.assertEqual(job.count("          if-no-files-found: error"), 1)
            names.extend(re.findall(r"^          name: (.+)$", job, re.M))
        self.assertEqual(len(set(names)), 2)
        self.assertIn("/zerone-consensus-rehearsal.*/reports/**", self.jobs["consensus-rehearsal"])
        self.assertIn("/zerone-consensus-rehearsal.*/logs/**", self.jobs["consensus-rehearsal"])
        self.assertIn("/shared-participant-reports/**", self.jobs["consensus-rehearsal"])
        self.assertIn("path: ${{ runner.temp }}/zerone-shared-workflows-evidence/**", self.jobs["consensus-shared-workflows"])


if __name__ == "__main__":
    unittest.main()

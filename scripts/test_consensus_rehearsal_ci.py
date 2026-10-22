#!/usr/bin/env python3
"""Offline guard for the split rehearsal's coverage and binary ownership.

Actionlint validates YAML/Actions syntax separately. These checks deliberately
pin the scheduling and suite contracts, not the unrelated jobs in this file.
"""
import io
from pathlib import Path
import re
import unittest
from unittest import mock


WORKFLOW = Path(__file__).resolve().parents[1] / ".github/workflows/ci.yml"
PIN = "a4272bfa9953e979463d3a310985bbacd5838a8e"
CONDITION = "${{ always() && !cancelled() && !github.event.pull_request.draft && (github.event_name != 'pull_request' || needs.scope.result != 'success' || needs.scope.outputs.consensus_rehearsal != 'false') }}"
SUITES = {
    "consensus-rehearsal": (
        ("ZERONE_LOCAL_NODE_TEST_BINARY", "scripts", "test_local_node.py"),
        ("ZERONE_LOCAL_NODE_TEST_BINARY", "scripts", "test_claim_workflow*.py"),
        ("ZERONE_PRIVATE_NET_TEST_BINARY", "skills/run-a-private-zerone-net/scripts", "test_*.py"),
    ),
    "consensus-shared-workflows": (
        ("ZERONE_SHARED_TEST_BINARY", "deploy/networks/zerone-dev-1", "test_*.py"),
        ("ZERONE_SHARED_TEST_BINARY", "scripts", "test_shared_claims*.py"),
        ("ZERONE_RESEARCH_TEST_BINARY", "tools/research-review", "test_workflow_native.py"),
    ),
}
SELECTIONS = {
    "consensus-rehearsal": "Select the rehearsed candidate binary",
    "consensus-shared-workflows": "Select frozen binary and public evidence directory",
}
# Match executable run lines, not comments or an echo of the unittest command.
# The && also keeps a missing/non-executable binary from becoming skipUnless green.
SUITE_RUN = re.compile(
    r'^        run: test -x "\$(\w+)" && '
    r'(?:mkdir -p "\$ZERONE_PRIVATE_NET_EVIDENCE_PARENT" && )?'
    r"python3 -B -m unittest discover -s (\S+) -p '([^']+)' -v$", re.M)


def jobs(text):
    return dict(re.findall(r"^  ([a-z][a-z0-9-]*):\n(.*?)(?=^  [a-z][a-z0-9-]*:\n|\Z)", text, re.M | re.S))


def steps(job):
    return re.findall(r"^      - .*?(?=^      - |\Z)", job, re.M | re.S)


class RehearsalCIContractTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        text = WORKFLOW.read_text()
        cls.jobs = jobs(text)

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
                actual = SUITE_RUN.findall(self.jobs[name])
                self.assertCountEqual(actual, expected)
                self.assertEqual(self.jobs[name].count("python3 -B -m unittest discover"), len(expected))
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

    def test_binary_selection_precedes_every_unconditional_suite(self):
        for name, expected in SUITES.items():
            job_steps = steps(self.jobs[name])
            selection = [index for index, step in enumerate(job_steps)
                         if step.startswith(f"      - name: {SELECTIONS[name]}\n")]
            self.assertEqual(len(selection), 1, name)
            for binary, directory, pattern in expected:
                with self.subTest(job=name, suite=pattern):
                    matches = [(index, step) for index, step in enumerate(job_steps)
                               if (binary, directory, pattern) in SUITE_RUN.findall(step)]
                    self.assertEqual(len(matches), 1)
                    index, step = matches[0]
                    self.assertLess(selection[0], index)
                    self.assertNotRegex(step, r"(?m)^(?:      - |        )(?:continue-on-error|if):")
                    # Permit only the existing evidence env, not inline mappings,
                    # expressions, quoted binary keys or any binary override.
                    environment = re.findall(r"^(?:      - |        )env:([^\n]*(?:\n          [^\n]*)*)", step, re.M)
                    allowed = []
                    if binary == "ZERONE_PRIVATE_NET_TEST_BINARY":
                        allowed = ["\n          ZERONE_PRIVATE_NET_EVIDENCE_PARENT: ${{ runner.temp }}/zerone-consensus-rehearsal-evidence/shared-participant-reports"]
                    self.assertEqual(environment, allowed)

    def test_guide_ports_are_reserved_before_the_rehearsal_opens_any_socket(self):
        helper = (Path(__file__).resolve().parent / "local-node.py").read_text()
        rpc = re.search(r'"--rpc-port", type=port, default=(\d+)\)', helper)
        p2p = re.search(r'"--p2p-port", type=port, default=(\d+)\)', helper)
        self.assertIsNotNone(rpc)
        self.assertIsNotNone(p2p)
        assert rpc is not None and p2p is not None
        low, high = sorted((int(rpc[1]), int(p2p[1])))
        self.assertEqual(high - low, 1, "reserve one contiguous range for both guide ports")
        job_steps = steps(self.jobs["consensus-rehearsal"])
        first = job_steps[0]
        self.assertTrue(first.startswith("      - name: Reserve the local node guide's loopback ports\n"))
        self.assertNotRegex(first, r"(?m)^        (?:continue-on-error|if):")
        self.assertIn(f"sudo sysctl -w net.ipv4.ip_local_reserved_ports={low}-{high}\n", first)
        self.assertIn(f'test "$(cat /proc/sys/net/ipv4/ip_local_reserved_ports)" = {low}-{high}\n', first)
        rehearsal = [index for index, step in enumerate(job_steps) if "local-consensus-rehearsal.sh" in step]
        self.assertEqual(len(rehearsal), 1)
        self.assertLess(0, rehearsal[0])

    def test_frozen_evidence_context_is_written_before_any_checkout_or_build(self):
        first = steps(self.jobs["consensus-shared-workflows"])[0]
        self.assertTrue(first.startswith("      - name: Initialize shared workflow evidence\n"))
        self.assertNotRegex(first, r"(?m)^        (?:continue-on-error|if):")
        self.assertIn('          mkdir -p "$RUNNER_TEMP/zerone-shared-workflows-evidence"\n', first)
        self.assertIn('> "$RUNNER_TEMP/zerone-shared-workflows-evidence/job-context.txt"\n', first)
        self.assertIn("status=started; suite success is not asserted", first)

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


class RehearsalCIMutationTests(unittest.TestCase):
    def setUp(self):
        self.text = WORKFLOW.read_text()

    def assert_rejected(self, mutated):
        self.assertNotEqual(mutated, self.text, "mutation must change the fixture")
        with mock.patch(__name__ + ".WORKFLOW", mock.Mock(read_text=lambda: mutated)):
            result = unittest.TextTestRunner(stream=io.StringIO()).run(
                unittest.defaultTestLoader.loadTestsFromTestCase(RehearsalCIContractTests))
        self.assertEqual(result.errors, [], "a mutation must fail an assertion, not crash the guard")
        self.assertTrue(result.failures, "mutation escaped the CI contract")

    def test_original_mutation_controls(self):
        mutations = {
            "missing-suite": self.text.replace("-p 'test_shared_claims*.py'", "-p 'test_shared_claims.py'"),
            "wrong-frozen-binary": self.text.replace("ZERONE_RESEARCH_TEST_BINARY=$RUNNER_TEMP/zerone-knowledge-11/zeroned", "ZERONE_RESEARCH_TEST_BINARY=$RUNNER_TEMP/candidate/zeroned"),
            "pin-drift": self.text.replace(PIN, "0" * 40),
            "raised-cap": self.text.replace('    name: "Consensus shared workflows"\n    runs-on: ubuntu-24.04\n    timeout-minutes: 30', '    name: "Consensus shared workflows"\n    runs-on: ubuntu-24.04\n    timeout-minutes: 45'),
            "serialized-jobs": self.text.replace("  consensus-shared-workflows:\n    needs: scope", "  consensus-shared-workflows:\n    needs: [scope, consensus-rehearsal]"),
            "dropped-draft-guard": self.text.replace("!github.event.pull_request.draft && ", ""),
            "colliding-artifact": self.text.replace("name: consensus-shared-workflows-${{ github.run_id }}-${{ github.run_attempt }}", "name: consensus-rehearsal-${{ github.run_id }}-${{ github.run_attempt }}"),
            "signing-bypasses-frozen-suites": self.text.replace("      - consensus-shared-workflows\n", ""),
        }
        for name, mutated in mutations.items():
            with self.subTest(mutation=name):
                self.assert_rejected(mutated)

    def test_disabled_or_unguarded_suite_mutations_in_both_jobs(self):
        for name in SUITES:
            for step in steps(jobs(self.text)[name]):
                matched = SUITE_RUN.search(step)
                if matched is None:
                    continue
                binary, directory, pattern = matched.groups()
                first, rest = step.split("\n", 1)
                override = f'          {binary}: ""\n'
                if "        env:\n" in rest:
                    overridden = step.replace("        env:\n", "        env:\n" + override)
                else:
                    overridden = first + "\n        env:\n" + override + rest
                mutations = {
                    "commented-out-command": step.replace("        run: ", "        run: echo disabled # "),
                    "continue-on-error": first + "\n        continue-on-error: true\n" + rest,
                    "conditional-suite": first + "\n        if: false\n" + rest,
                    "empty-step-binary": overridden,
                    "missing-executable-check": step.replace(f'test -x "${binary}" && ', ""),
                }
                for mutation, changed in mutations.items():
                    with self.subTest(job=name, suite=pattern, mutation=mutation):
                        self.assert_rejected(self.text.replace(step, changed, 1))

    def test_selection_moved_after_suites_in_both_jobs(self):
        for name in SUITES:
            job = jobs(self.text)[name]
            selection, = [step for step in steps(job)
                          if step.startswith(f"      - name: {SELECTIONS[name]}\n")]
            with self.subTest(job=name, mutation="selection-after-suites"):
                self.assert_rejected(self.text.replace(job, job.replace(selection, "") + selection, 1))


if __name__ == "__main__":
    unittest.main()

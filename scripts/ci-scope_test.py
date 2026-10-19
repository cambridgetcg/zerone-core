"""Exercise scope selection against actual Git changes, including unsafe moves."""

import importlib.util
from pathlib import Path
import re
import subprocess
import sys
import tempfile
import unittest

SCRIPT = Path(__file__).with_name("ci-scope.py")
WORKFLOW = Path(__file__).resolve().parents[1] / ".github/workflows/ci.yml"
spec = importlib.util.spec_from_file_location("ci_scope", SCRIPT)
scope = importlib.util.module_from_spec(spec)
spec.loader.exec_module(scope)

EVERYTHING = {job: "true" for job in scope.JOBS}
SKILL = "skills/run-a-private-zerone-net/"


def only(*jobs):
    """Expected outputs where exactly these jobs run."""
    return {job: "true" if job in jobs else "false" for job in scope.JOBS}


class ScopeTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="zerone-ci-scope-")
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.git("init", "--quiet")
        self.git("config", "user.name", "CI Scope Test")
        self.git("config", "user.email", "ci-scope@example.invalid")
        self.git("config", "commit.gpgsign", "false")
        self.write("dashboard/index.html", "original page")
        self.write("x/knowledge/rule.go", "preserved runtime")
        self.write(SKILL + "scripts/private-net.py", "original skill")
        self.write("tests/cross_stack/harness_test.go", "package cross_stack")
        self.base = self.commit()

    def git(self, *args):
        return subprocess.check_output(["git", *args], cwd=self.root, stderr=subprocess.DEVNULL).decode().strip()

    def write(self, path, value):
        target = self.root / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(value)

    def commit(self):
        self.git("add", "-A")
        self.git("commit", "--quiet", "-m", "test change")
        return self.git("rev-parse", "HEAD")

    def classified(self, head, event="pull_request", base=None):
        output = subprocess.check_output(
            [sys.executable, str(SCRIPT), "--event", event, "--base", self.base if base is None else base, "--head", head],
            cwd=self.root, stderr=subprocess.DEVNULL, text=True,
        )
        lines = output.splitlines()
        pairs = [line.split("=", 1) for line in lines]
        self.assertTrue(all(len(pair) == 2 and pair[1] in ("true", "false") for pair in pairs), output)
        result = dict(pairs)
        self.assertEqual(len(result), len(lines), output)
        return result

    def test_presentation_change_and_same_diff_on_main_or_manual(self):
        self.write("dashboard/index.html", "new page")
        self.write("dashboard/src/research.ts", "export {};")
        head = self.commit()
        self.assertEqual(self.classified(head), only())
        for event in ("push", "workflow_dispatch", "release", "unknown"):
            with self.subTest(event=event):
                self.assertEqual(self.classified(head, event), EVERYTHING)

    def test_shared_authority_functions_runtime_and_unknown_paths_stay_full(self):
        for path in (
            "dashboard/public/standards/authority-geometry.v1.json",
            "dashboard/functions/api/_knowledge.ts", "dashboard/observer-release.json",
            "dashboard/new-build-system/config.ts", "docs/AUTHORITATIVE-STATE.md",
            "README.md", "x/knowledge/rule.go", "go.mod", "sdk/typescript/src/index.ts",
            ".github/workflows/ci.yml", "scripts/ci-scope.py",
        ):
            with self.subTest(path=path):
                self.git("reset", "--hard", self.base)
                self.write("dashboard/index.html", "website change in a mixed diff")
                self.write(path, "changed shared input")
                self.assertEqual(self.classified(self.commit())["full_checks"], "true")

    def test_rename_from_runtime_to_website_stays_full(self):
        (self.root / "dashboard/src").mkdir()
        self.git("mv", "x/knowledge/rule.go", "dashboard/src/rule.ts")
        self.assertEqual(self.classified(self.commit()), EVERYTHING)

    def test_rename_from_website_to_unknown_stays_full(self):
        self.git("mv", "dashboard/index.html", "new-entry.html")
        self.assertEqual(self.classified(self.commit()), EVERYTHING)

    def test_delete_regular_website_file_is_website_only(self):
        (self.root / "dashboard/index.html").unlink()
        self.assertEqual(self.classified(self.commit()), only())

    def test_symlink_and_executable_mode_stay_full(self):
        target = self.root / "dashboard/index.html"
        target.unlink()
        target.symlink_to("../x/knowledge/rule.go")
        self.assertEqual(self.classified(self.commit()), EVERYTHING)
        self.git("reset", "--hard", self.base)
        target.chmod(0o755)
        self.assertEqual(self.classified(self.commit()), EVERYTHING)

    def test_empty_or_unavailable_comparison_stays_full(self):
        for base in (self.base, "0" * 40, "", "main; echo untrusted"):
            with self.subTest(base=base):
                self.assertEqual(self.classified(self.base, base=base), EVERYTHING)

    def test_ambiguous_paths_and_malformed_records_are_not_website(self):
        for path in ("/dashboard/index.html", "dashboard/src/../rule.ts", "dashboard//src/a.ts", "dashboard/src/a\nb.ts", "dashboard/src/a\\b.ts"):
            self.assertFalse(scope.website_path(path))
        for raw in (
            b"", b"a\0", b":100644 100644 a b M\0dashboard/index.html",
            b":100644 100644 a b R100\0dashboard/index.html\0",
            b":100644 100644 a b M\0dashboard/index.html\0",
            b":100644 100644 a b M\0dashboard/\xff\0",
            f":000000 000000 {'0' * 40} {'0' * 40} M\0dashboard/index.html\0".encode(),
            f":100644 160000 {'a' * 40} {'b' * 40} M\0dashboard/index.html\0".encode(),
        ):
            self.assertFalse(scope.website_diff(raw))

    def test_private_network_skill_runs_only_the_suites_that_execute_it(self):
        self.write(SKILL + "SKILL.md", "protocol")
        self.write(SKILL + "scripts/private-net.py", "changed skill")
        (self.root / SKILL / "scripts/private-net.py").chmod(0o755)
        self.write(SKILL + "references/protocol.md", "reference")
        self.assertEqual(self.classified(self.commit()), only("full_checks", "frontier_macos", "consensus_rehearsal"))

    def test_go_test_files_run_only_jobs_that_compile_them(self):
        cases = {
            "tests/cross_stack/harness_test.go": only("full_checks"),
            "x/auth/client/cli/tx_test.go": only("full_checks", "frontier_macos"),
            "tests/multivalidator/multivalidator_test.go": only("full_checks", "consensus_rehearsal"),
            "tools/sigstore-substrate-compiler/compiler_test.go": only("full_checks", "sigstore_compiler", "operations_safety"),
            "deploy/fly_test.go": only("full_checks", "operations_safety"),
            "tools/fork-genesis/main_test.go": only("full_checks", "operations_safety"),
        }
        for path, expected in cases.items():
            with self.subTest(path=path):
                self.git("reset", "--hard", self.base)
                self.write(path, "package changed")
                self.assertEqual(self.classified(self.commit()), expected)

    def test_known_kinds_mixed_with_runtime_or_ci_inputs_run_everything(self):
        for path in ("x/knowledge/rule.go", "go.sum", "Makefile", ".github/workflows/ci.yml",
                     "scripts/ci-scope.py", "tests/cross_stack/helpers.go", "skills/README.md"):
            with self.subTest(path=path):
                self.git("reset", "--hard", self.base)
                self.write(SKILL + "scripts/private-net.py", "skill change")
                self.write("tests/cross_stack/harness_test.go", "test change")
                self.write(path, "shared input")
                self.assertEqual(self.classified(self.commit()), EVERYTHING)

    def test_secret_shaped_names_never_narrow_checks(self):
        for path in (SKILL + "node_key.json", SKILL + "priv_validator_key.json.bak", SKILL + "wallet.mnemonic",
                     SKILL + "genesis.seed", SKILL + ".env", SKILL + ".env.local", SKILL + "fly-credentials.json",
                     SKILL + "keyring-test/key.py", "x/keyring/store_test.go", "dashboard/src/keyring-view.ts"):
            with self.subTest(path=path):
                self.git("reset", "--hard", self.base)
                self.write(path, "secret-shaped")
                self.assertEqual(self.classified(self.commit()), EVERYTHING)

    def test_files_below_a_go_embed_package_never_narrow_checks(self):
        self.write("x/assets/embed.go", "package assets\n\n//go:embed fixtures\nvar fixtures embed.FS\n")
        self.base = self.commit()
        self.write("x/assets/fixtures/case_test.go", "embedded bytes")
        self.assertEqual(self.classified(self.commit()), EVERYTHING)
        self.git("reset", "--hard", self.base)
        self.write("tests/cross_stack/harness_test.go", "outside the embedding package")
        self.assertEqual(self.classified(self.commit()), only("full_checks"))
        self.git("reset", "--hard", self.base)
        self.write("embed.go", "package root\n\n//go:embed tests\nvar all embed.FS\n")
        self.base = self.commit()
        self.write("tests/cross_stack/harness_test.go", "a root-level embed reaches every path")
        self.assertEqual(self.classified(self.commit()), EVERYTHING)

    def test_skill_symlink_and_moves_out_of_the_skill_run_everything(self):
        link = self.root / SKILL / "scripts/link.py"
        link.symlink_to("../../../x/knowledge/rule.go")
        self.assertEqual(self.classified(self.commit()), EVERYTHING)
        self.git("reset", "--hard", self.base)
        (self.root / "scripts").mkdir()
        self.git("mv", SKILL + "scripts/private-net.py", "scripts/private-net.py")
        self.assertEqual(self.classified(self.commit()), EVERYTHING)
        self.git("reset", "--hard", self.base)
        target = self.root / "tests/cross_stack/harness_test.go"
        target.chmod(0o755)
        self.assertEqual(self.classified(self.commit()), EVERYTHING)

    def test_workflow_reads_exactly_the_published_selection(self):
        text = WORKFLOW.read_text()
        published = set(re.findall(r"^      ([a-z_]+): \$\{\{ steps\.classify\.outputs\.\1 \}\}$", text, re.M))
        used = set(re.findall(r"needs\.scope\.outputs\.([a-z_]+)", text))
        self.assertEqual(published, set(scope.JOBS))
        self.assertEqual(used, set(scope.JOBS))


if __name__ == "__main__":
    unittest.main()

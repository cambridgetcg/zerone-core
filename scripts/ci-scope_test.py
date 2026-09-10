"""Exercise scope selection against actual Git changes, including unsafe moves."""

import importlib.util
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

SCRIPT = Path(__file__).with_name("ci-scope.py")
spec = importlib.util.spec_from_file_location("ci_scope", SCRIPT)
scope = importlib.util.module_from_spec(spec)
spec.loader.exec_module(scope)


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
        return subprocess.check_output(
            [sys.executable, str(SCRIPT), "--event", event, "--base", self.base if base is None else base, "--head", head],
            cwd=self.root, stderr=subprocess.DEVNULL, text=True,
        ).strip()

    def test_presentation_change_and_same_diff_on_main_or_manual(self):
        self.write("dashboard/index.html", "new page")
        self.write("dashboard/src/research.ts", "export {};")
        head = self.commit()
        self.assertEqual(self.classified(head), "full_checks=false")
        for event in ("push", "workflow_dispatch", "release", "unknown"):
            with self.subTest(event=event):
                self.assertEqual(self.classified(head, event), "full_checks=true")

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
                self.assertEqual(self.classified(self.commit()), "full_checks=true")

    def test_rename_from_runtime_to_website_stays_full(self):
        (self.root / "dashboard/src").mkdir()
        self.git("mv", "x/knowledge/rule.go", "dashboard/src/rule.ts")
        self.assertEqual(self.classified(self.commit()), "full_checks=true")

    def test_rename_from_website_to_unknown_stays_full(self):
        self.git("mv", "dashboard/index.html", "new-entry.html")
        self.assertEqual(self.classified(self.commit()), "full_checks=true")

    def test_delete_regular_website_file_is_website_only(self):
        (self.root / "dashboard/index.html").unlink()
        self.assertEqual(self.classified(self.commit()), "full_checks=false")

    def test_symlink_and_executable_mode_stay_full(self):
        target = self.root / "dashboard/index.html"
        target.unlink()
        target.symlink_to("../x/knowledge/rule.go")
        self.assertEqual(self.classified(self.commit()), "full_checks=true")
        self.git("reset", "--hard", self.base)
        target.chmod(0o755)
        self.assertEqual(self.classified(self.commit()), "full_checks=true")

    def test_empty_or_unavailable_comparison_stays_full(self):
        for base in (self.base, "0" * 40, "", "main; echo untrusted"):
            with self.subTest(base=base):
                self.assertEqual(self.classified(self.base, base=base), "full_checks=true")

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


if __name__ == "__main__":
    unittest.main()

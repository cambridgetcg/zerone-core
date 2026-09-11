import importlib.util
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("dev_context", Path(__file__).with_name("build-context.py"))
context = importlib.util.module_from_spec(spec)
spec.loader.exec_module(context)


class BuildContextTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        self.repo = self.root / "repo"
        self.repo.mkdir()
        context.git(self.repo, "init", "-q")
        for name in (*context.FIXED, "cmd/zeroned/main.go", "internal/claimrecordmigration/migration.go", "app/app_test.go", "app/fixture.key", "deploy/private.env"):
            path = self.repo / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text("public synthetic fixture\n")
        self.commit()

    def commit(self):
        context.git(self.repo, "add", ".")
        context.git(self.repo, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "-qm", "fixture")
        self.head = context.git(self.repo, "rev-parse", "HEAD").decode().strip()

    def tearDown(self):
        self.temp.cleanup()

    def test_exports_exact_committed_regular_closure(self):
        output = self.root / "context"
        result = context.export(self.repo, self.head, output)
        self.assertEqual(result["source_commit"], self.head)
        self.assertTrue((output / "internal/claimrecordmigration/migration.go").is_file())
        self.assertTrue((output / "runtime/runtime.py").is_file())
        for name in ("app/app_test.go", "app/fixture.key", "deploy/private.env", ".git"):
            self.assertFalse((output / name).exists())
        for row in result["files"]:
            self.assertEqual((output / row["context_path"]).read_bytes(), (self.repo / row["source_path"]).read_bytes())

    def test_dirty_wrong_commit_existing_output_refused(self):
        with self.assertRaises(ValueError):
            context.export(self.repo, "0" * 40, self.root / "wrong")
        (self.repo / "dirty.txt").write_text("fixture")
        with self.assertRaises(ValueError):
            context.export(self.repo, self.head, self.root / "dirty")
        (self.repo / "dirty.txt").unlink()
        output = self.root / "existing"
        output.mkdir()
        with self.assertRaises(ValueError):
            context.export(self.repo, self.head, output)
        self.assertEqual(list(output.iterdir()), [])

    def test_selected_symlink_refused_without_output(self):
        (self.repo / "app/link.go").symlink_to("fixture.key")
        self.commit()
        with self.assertRaises(ValueError):
            context.export(self.repo, self.head, self.root / "context")
        self.assertFalse((self.root / "context").exists())


if __name__ == "__main__":
    unittest.main()

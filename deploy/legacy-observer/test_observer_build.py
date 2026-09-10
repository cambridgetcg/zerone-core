import hashlib
import importlib.util
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("observer_builder", HERE / "observer-build.py")
builder = importlib.util.module_from_spec(spec)
spec.loader.exec_module(builder)


class ObserverBuildTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.temp = tempfile.TemporaryDirectory()
        cls.root = Path(cls.temp.name)
        cls.archive = cls.root / "export/legacy-source.tar.gz"
        try:
            builder.base.export_source(HERE.parents[1], cls.root / "export")
        except subprocess.CalledProcessError:
            cls.temp.cleanup()
            raise unittest.SkipTest("Pinned historical Git objects are unavailable in this clone")
        cls.files = builder.base.read_archive(cls.archive)

    @classmethod
    def tearDownClass(cls):
        cls.temp.cleanup()

    def test_exact_patch_changes_only_two_dependency_files(self):
        changed = builder.apply_patch(self.files, HERE / "observer-dependencies.patch")
        before = {name: body for name, _, body in self.files}
        self.assertEqual(len(changed), 657)
        self.assertEqual({name for name, _, body in changed if body != before[name]}, {"go.mod", "go.sum"})
        for name, _, body in changed:
            if name in builder.TARGET_SHA256:
                self.assertEqual(hashlib.sha256(body).hexdigest(), builder.TARGET_SHA256[name])
        gomod = dict((name, body) for name, _, body in changed)["go.mod"]
        self.assertIn(b"github.com/cosmos/cosmos-sdk v0.50.15\n", gomod)
        self.assertIn(b"github.com/cosmos/ibc-go/v8 v8.8.0\n", gomod)
        self.assertIn(b"github.com/cometbft/cometbft v0.38.25\n", gomod)

    def test_tampered_patch_refused_before_application(self):
        bad = self.root / "tampered.patch"
        bad.write_bytes((HERE / "observer-dependencies.patch").read_bytes().replace(b"v0.38.25", b"v0.38.99", 1))
        with self.assertRaisesRegex(ValueError, "hash differs"):
            builder.apply_patch(self.files, bad)

    def test_symlink_patch_refused(self):
        link = self.root / "linked.patch"
        link.symlink_to(HERE / "observer-dependencies.patch")
        with self.assertRaisesRegex(ValueError, "regular file"):
            builder.apply_patch(self.files, link)

    def test_wrong_base_context_refused(self):
        files = [(name, mode, body.replace(b"v0.38.20", b"v0.38.19", 1) if name == "go.mod" else body)
                 for name, mode, body in self.files]
        with self.assertRaisesRegex(ValueError, "context differs"):
            builder.apply_patch(files, HERE / "observer-dependencies.patch")

    def test_container_uses_exact_builder_and_readonly_modules(self):
        args = builder.build_argv(Path("/tmp/reproduction"))
        self.assertEqual(args[:3], ["docker", "--host", "unix:///var/run/docker.sock"])
        self.assertIn(builder.BUILDER_IMAGE, args)
        self.assertNotIn(builder.base.BUILDER_IMAGE, args)
        self.assertIn("GOFLAGS=-mod=readonly", args)
        self.assertIn("GOWORK=off", args)
        self.assertIn("go mod verify\n", args[-1])
        self.assertIn("type=bind,src=/tmp/reproduction/src,dst=/app,readonly", args)

    def test_isolated_python_ignores_ambient_provenance(self):
        evil = self.root / "ambient"
        evil.mkdir()
        (evil / "provenance.py").write_text("raise RuntimeError('ambient import executed')\n")
        import os
        env = dict(os.environ, PYTHONPATH=str(evil))
        result = subprocess.run([sys.executable, "-I", "-B", str(HERE / "observer-build.py"), "--help"],
                                cwd=evil, env=env, capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_failed_build_cannot_claim_reproduction(self):
        destination = self.root / "failed-build"
        with patch.object(builder.platform, "system", return_value="Linux"), \
             patch.object(builder.platform, "machine", return_value="x86_64"), \
             patch.object(builder.subprocess, "run", return_value=subprocess.CompletedProcess([], 7)):
            with self.assertRaisesRegex(ValueError, "build failed"):
                builder.build(self.archive, HERE / "observer-dependencies.patch", destination, "a" * 64)
        import json
        result = json.loads((destination / "provenance.json").read_text())
        self.assertFalse(result["passed"])
        self.assertEqual(result["source_status"], "build-failed")
        self.assertFalse(result["matches_expected_executable"])

    def test_expected_hash_must_be_exact(self):
        with self.assertRaisesRegex(ValueError, "lowercase SHA-256"):
            builder.build(self.archive, HERE / "observer-dependencies.patch", self.root / "invalid-hash", "../../anything")
        self.assertFalse((self.root / "invalid-hash").exists())


if __name__ == "__main__":
    unittest.main()

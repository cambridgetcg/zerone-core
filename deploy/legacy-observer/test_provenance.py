import gzip
import importlib.util
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

ROOT = Path(__file__).resolve().parent
SPEC = importlib.util.spec_from_file_location("provenance", ROOT / "provenance.py")
provenance = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(provenance)


class ProvenanceTests(unittest.TestCase):
    def test_export_allowlist_excludes_private_state_and_tests(self):
        for name in ("deploy/mainnet/artifacts/priv_validator_key.json", "deploy/mainnet/artifacts/node_key.json",
                     "x/staking/keeper/accounting_test.go", "x/keeper/testdata/key.go", "x/../deploy/key.go",
                     "/app/main.go", "app//main.go", "app/.git/config.go", "cosmovisor/genesis/bin/zeroned",
                     "config/app.toml", ".env", "keyring-test/key", "docs/report.md"):
            self.assertFalse(provenance.selected_source(name), name)
        for name in ("app/app.go", "cmd/zeroned/main.go", "x/staking/keeper/keeper.go", "go.mod",
                     "docs/swagger-ui/embed.go", "docs/swagger-ui/index.html", "docs/swagger-ui/swagger.json"):
            self.assertTrue(provenance.selected_source(name), name)

    def test_existing_or_symlink_output_never_reused(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            populated = root / "populated"
            populated.mkdir()
            (populated / "keep").write_bytes(b"retained")
            link = root / "link"
            link.symlink_to(populated, target_is_directory=True)
            for path in (populated, link):
                with self.assertRaises(ValueError):
                    provenance.new_output(path)
            self.assertEqual((populated / "keep").read_bytes(), b"retained")

    def test_wrong_or_expanding_source_archive_refused_before_extract(self):
        with tempfile.TemporaryDirectory() as temporary:
            archive = Path(temporary) / "wrong.tar.gz"
            for raw in (b"not the source", b"x" * (provenance.SOURCE_ARCHIVE_SIZE + 2)):
                archive.write_bytes(gzip.compress(raw))
                with self.assertRaisesRegex(ValueError, "exact reproduced closure"):
                    provenance.read_archive(archive)
            self.assertEqual([p.name for p in Path(temporary).iterdir()], ["wrong.tar.gz"])

    def test_build_is_pinned_local_and_source_read_only(self):
        argv = provenance.build_argv(Path("/tmp/owned-output"))
        self.assertEqual(argv[:3], ["docker", "--host", "unix:///var/run/docker.sock"])
        self.assertIn(provenance.BUILDER_IMAGE, argv)
        self.assertIn("type=bind,src=/tmp/owned-output/src,dst=/app,readonly", argv)
        self.assertIn("--read-only", argv)
        self.assertIn("no-new-privileges:true", argv)
        self.assertEqual(argv[argv.index("--network") + 1], "bridge")
        self.assertIn("--home /tmp/metadata-home", argv[-1])
        self.assertNotIn(" start ", argv[-1])
        self.assertNotIn("HOME=", "\n".join(argv))

    def test_failed_build_cannot_claim_reproduced_exact(self):
        with tempfile.TemporaryDirectory() as temporary:
            output = Path(temporary) / "failed"
            with mock.patch.object(provenance.platform, "system", return_value="Linux"), \
                 mock.patch.object(provenance.platform, "machine", return_value="x86_64"), \
                 mock.patch.object(provenance, "read_archive", return_value=[("go.mod", 0o644, b"module fixture\n")]), \
                 mock.patch.object(provenance.subprocess, "run", return_value=subprocess.CompletedProcess([], 1)):
                with self.assertRaisesRegex(ValueError, "Reproduction failed"):
                    provenance.reproduce(Path("unused"), output)
            report = json.loads((output / "provenance.json").read_text())
            self.assertFalse(report["passed"])
            self.assertEqual(report["source_status"], "reproduction-failed")
            self.assertIsNone(report["executable_sha256"])

    def test_pinned_git_export_reproduces_the_exact_archive(self):
        repository = ROOT.parent.parent
        available = subprocess.run(["git", "-C", str(repository), "cat-file", "-e", provenance.SOURCE_COMMIT],
                                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        if available.returncode:
            self.skipTest("Historical source objects need a full clone or explicit fetch of the pinned commit")
        with tempfile.TemporaryDirectory() as temporary:
            output = Path(temporary) / "export"
            manifest = provenance.export_source(repository, output)
            self.assertEqual(manifest["file_count"], 657)
            self.assertEqual(provenance.sha256(output / "legacy-source.tar"), provenance.SOURCE_ARCHIVE_SHA256)
            self.assertEqual(len(provenance.read_archive(output / "legacy-source.tar.gz")), 657)
            self.assertNotIn(str(repository), json.dumps(manifest))


if __name__ == "__main__":
    unittest.main()

#!/usr/bin/env python3
"""H1Kc regressions. Docker and positive cross-build hashes are UNIT MOCKS."""
import contextlib
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tarfile
import tempfile
import unittest
from unittest import mock

import provenance as p
from test_kit import archive_bytes, drill


class H1KcTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="h1kc-unit-")
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)
        self.host = "unix:///unit-test/docker.sock"
        self.binary = b"\x7fELFunit mock only, not a cross-build"
        self.ca = b"-----BEGIN CERTIFICATE-----\nunit mock only\n-----END CERTIFICATE-----\n"
        self.metadata = b"/out/zeroned: go1.25.14\n build GOOS=linux\n build GOARCH=amd64\n build CGO_ENABLED=0\n"

    def inventories(self):
        source = self.root / "copy"
        (source / "application.db").mkdir(parents=True)
        (source / "copy.json").write_text("{}")
        home = self.root / "home"
        (home / "data/application.db").mkdir(parents=True)
        (home / ".h1-disposable-copy").write_text("local-only; never a validator home\n")
        for root, relative, reader in ((source, "application.db", drill.inventory),
                                      (home, "data/application.db", drill.working_inventory)):
            database = root / relative
            (database / "CURRENT").write_text("MANIFEST-000001\n")
            (database / "MANIFEST-000001").write_text("unit inventory fixture, not LevelDB")
            yield root, relative, reader

    def test_current_backup_is_hashed_in_both_inventories(self):
        for root, relative, reader in self.inventories():
            backup = root / relative / "CURRENT.bak"
            backup.write_text("MANIFEST-000000\n")
            self.assertEqual(reader(root)[relative + "/CURRENT.bak"], p.sha256(backup))
            (root / relative / "CURRENT").unlink()
            with self.assertRaisesRegex(ValueError, "CURRENT"):
                reader(root)

    def test_backup_lookalikes_are_rejected_before_reading(self):
        for root, relative, reader in self.inventories():
            for name in ("CURRENT.bak.old", "CURRENTxbak", "CURRENT.bak.key", "CURRENT.BAK"):
                path = root / relative / name
                path.write_text("must not be read")
                with self.subTest(reader=reader.__name__, name=name), \
                     mock.patch.object(p, "sha256", side_effect=AssertionError("opened before allowlist")), \
                     self.assertRaisesRegex(ValueError, "filename"):
                    reader(root)
                path.unlink()

    def test_backup_links_and_directories_are_rejected_before_reading(self):
        for root, relative, reader in self.inventories():
            path = root / relative / "CURRENT.bak"
            for kind in ("symlink", "hardlink", "directory"):
                if kind == "symlink":
                    path.symlink_to(self.root / "never-open")
                elif kind == "hardlink":
                    donor = self.root / (root.name + "-donor")
                    donor.write_text("unit backup donor outside the inventory")
                    os.link(donor, path)
                else:
                    path.mkdir()
                with self.subTest(reader=reader.__name__, kind=kind), \
                     mock.patch.object(p, "sha256", side_effect=AssertionError("opened linked backup")), \
                     self.assertRaises(ValueError):
                    reader(root)
                path.rmdir() if kind == "directory" else path.unlink()

    def fake_export(self, repository, output):
        (output / "src").mkdir()
        (output / "src/go.mod").write_text("unit fixture only\n")
        return {"source_sha256": p.SOURCE_ARCHIVE_SHA256,
                "files": [{"path": "go.mod", "sha256": p.sha256(output / "src/go.mod")}]}

    def docker(self, argv, **kwargs):
        self.assertEqual(argv[:3], ["docker", "--host", self.host])
        self.assertIn(p.BUILDER_IMAGE, argv)
        self.assertIn("--pull=never", argv)
        self.assertIn("--network=none", argv)
        # Real adapters must not contaminate artifact bytes with Docker warnings.
        self.assertIsNot(kwargs["stderr"], subprocess.STDOUT)
        self.assertIsNot(kwargs["stderr"], kwargs["stdout"])
        kwargs["stderr"].write(b"UNIT MOCK Docker warning, not artifact data\n")
        if "build" in argv:
            mount = next(value for value in argv if value.endswith(",dst=/out"))
            directory = Path(mount.split("src=", 1)[1].split(",dst=")[0])
            (directory / "zeroned").write_bytes(self.binary)
        elif "-m" in argv:
            kwargs["stdout"].write(self.metadata)
        elif "cat" in argv:
            kwargs["stdout"].write(self.ca)
        return subprocess.CompletedProcess(argv, 0)

    @contextlib.contextmanager
    def mocked_build(self, *, matching_pin=False):
        real_sha256 = p.sha256
        def digest(path):
            # Positive branch only: no real binary is claimed to reproduce the pin.
            if matching_pin and Path(path).name == "zeroned":
                return p.CROSS_BUILD_SHA256
            return real_sha256(path)
        with mock.patch.object(p, "export_source", side_effect=self.fake_export), \
             mock.patch.object(p.platform, "system", return_value="Darwin"), \
             mock.patch.object(p.platform, "machine", return_value="arm64"), \
             mock.patch.object(p, "sha256", side_effect=digest), \
             mock.patch.object(subprocess, "run", side_effect=self.docker):
            yield

    def test_emulated_build_refuses_reproducible_but_unpinned_binary(self):
        output = self.root / "mismatch"
        with self.mocked_build(), self.assertRaisesRegex(ValueError, "pinned cross-build"):
            p.build(self.root, output, self.root, False, self.host, allow_emulated=True)
        report = json.loads((output / "provenance.json").read_text())
        self.assertEqual(report["status"], "build-failed")
        self.assertEqual(report["native_platform"], "emulated")
        self.assertFalse(report["reproduced"])
        self.assertEqual(report["binary_sha256"], p.sha256(output / "build-1/zeroned"))
        self.assertIsNone(report["image_digest"])
        self.assertFalse((output / "image.oci.tar").exists())

    def test_matching_emulated_build_records_pin_and_stdout_only_artifacts(self):
        output = self.root / "matched"
        with self.mocked_build(matching_pin=True):
            report = p.build(self.root, output, self.root, False, self.host, allow_emulated=True)
        self.assertEqual(report["native_platform"], "emulated")
        self.assertEqual(report["binary_sha256"], "f35c98b59aba9d9b77062dede87bbe18a688ee26adad5c0b961f156598633495")
        self.assertEqual(report["expected_binary_sha256"], report["binary_sha256"])
        self.assertTrue(report["reproduced"])
        self.assertEqual(report["status"], "built-local")
        self.assertEqual(report["binary_go_metadata"], [self.metadata.decode()] * 2)
        for step in (3, 7):
            self.assertEqual((output / f"build-step-{step}.log").read_bytes(), self.metadata)
        for step in (4, 8):
            self.assertEqual((output / f"build-step-{step}.log").read_bytes(), self.ca)
        for step in range(1, 9):
            self.assertIn(b"Docker warning", (output / f"build-step-{step}.stderr.log").read_bytes())
        with tarfile.open(output / "image.oci.tar") as archive:
            index = json.loads(archive_bytes(archive, "index.json"))
            manifest = json.loads(archive_bytes(archive, "blobs/sha256/" + index["manifests"][0]["digest"].split(":")[1]))
            layer = archive_bytes(archive, "blobs/sha256/" + manifest["layers"][0]["digest"].split(":")[1])
            with tarfile.open(fileobj=io.BytesIO(layer)) as contents:
                self.assertEqual(archive_bytes(contents, p.CA_PATH.lstrip("/")), self.ca)

    def test_opt_in_also_enforces_pin_on_native_hosts(self):
        with self.mocked_build(), mock.patch.object(p.platform, "system", return_value="Linux"), \
             mock.patch.object(p.platform, "machine", return_value="x86_64"), \
             self.assertRaisesRegex(ValueError, "pinned cross-build"):
            p.build(self.root, self.root / "native-opt-in", self.root, False, self.host, allow_emulated=True)

    def test_emulated_build_still_requires_two_identical_binaries(self):
        docker = self.docker
        def differing(argv, **kwargs):
            result = docker(argv, **kwargs)
            if "build" in argv:
                self.binary += b"different next build"
            return result
        with self.mocked_build(), mock.patch.object(subprocess, "run", side_effect=differing), \
             self.assertRaisesRegex(ValueError, "fresh-cache binary builds differ"):
            p.build(self.root, self.root / "different", self.root, False, self.host, allow_emulated=True)
        self.assertFalse((self.root / "different/image.oci.tar").exists())

    def test_emulated_dry_run_does_not_claim_execution_or_call_docker(self):
        with mock.patch.object(p, "export_source", side_effect=self.fake_export), \
             mock.patch.object(p, "require_native_host", side_effect=AssertionError("native guard in dry run")), \
             mock.patch.object(subprocess, "run", side_effect=AssertionError("Docker in dry run")):
            report = p.build(self.root, self.root / "dry", self.root / "missing", True, self.host, allow_emulated=True)
        self.assertTrue(report["allow_emulated"])
        self.assertIsNone(report["native_platform"])
        self.assertEqual(report["expected_binary_sha256"], p.CROSS_BUILD_SHA256)
        self.assertIsNone(report["binary_sha256"])
        self.assertIsNone(report["image_digest"])
        self.assertFalse(report["reproduced"])

    def test_cli_passes_explicit_opt_in_and_defaults_to_native(self):
        for opt_in in ([], ["--allow-emulated"]):
            argv = ["provenance.py", "--repository", str(self.root), "--output", str(self.root / "new"),
                    "--module-cache", str(self.root), *opt_in]
            with mock.patch.object(sys, "argv", argv), mock.patch.object(p, "build", return_value={}) as build, \
                 contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(p.main(), 0)
            self.assertEqual(build.call_args.kwargs["allow_emulated"], bool(opt_in))


if __name__ == "__main__":
    unittest.main(verbosity=2)

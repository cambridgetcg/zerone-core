#!/usr/bin/env python3
"""cometbft-3 regressions: H1 is built from its frozen source plus ONE pinned dependency patch.

The export tests read real Git objects from this repository (no network). Docker,
chain and Fly responses are UNIT MOCKS.
"""
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest import mock

import provenance as p
import runtime_smoke as runtime
from test_kit import IMAGE, drill, load, ship

HERE = Path(__file__).resolve().parent
REPOSITORY = HERE.parents[2]


def has_commit(commit):
    return subprocess.run(["git", "-C", str(REPOSITORY), "cat-file", "-e", commit + "^{commit}"],
                          capture_output=True).returncode == 0


class CometClosureTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory(prefix="h1-comet-unit-")
        self.addCleanup(self.tmp.cleanup)
        self.root = Path(self.tmp.name)

    def test_patch_file_is_the_reviewed_observer_patch(self):
        patch = p.read_dependency_patch().decode()
        self.assertIn("+\tgithub.com/cometbft/cometbft v0.38.25\n", patch)
        self.assertIn("-\tgithub.com/cometbft/cometbft v0.38.20\n", patch)
        changed = {line[1:].split()[0] for line in patch.splitlines()
                   if line[:1] in "+-" and not line.startswith(("+++", "---"))}
        self.assertEqual(changed, {"github.com/cometbft/cometbft", "github.com/lib/pq", "github.com/minio/highwayhash",
                                   "github.com/petermattis/goid", "github.com/sasha-s/go-deadlock"})
        self.assertFalse(any(line.startswith("-") and not line.startswith("---") for line in
                             patch.split("--- a/go.sum", 1)[1].splitlines()), "go.sum lines may only be added")

    @unittest.skipUnless(has_commit(p.SOURCE_COMMIT), "frozen H1 commit not in this clone")
    def test_real_export_keeps_frozen_pin_and_builds_patched_closure(self):
        output = self.root / "export"
        output.mkdir()
        receipt = p.export_source(REPOSITORY, output)
        self.assertEqual(receipt["source_sha256"], p.SOURCE_ARCHIVE_SHA256)
        self.assertEqual(p.sha256(output / "source.tar"), p.SOURCE_ARCHIVE_SHA256)
        self.assertEqual(receipt["build_source_sha256"], p.BUILD_SOURCE_SHA256)
        self.assertEqual(receipt["dependency_patch_sha256"], p.DEPENDENCY_PATCH_SHA256)
        self.assertEqual(receipt["patched_files"], p.DEPENDENCY_TARGET_SHA256)
        self.assertEqual([f["path"] for f in receipt["build_files"] if f["patched"]], ["go.mod", "go.sum"])
        self.assertEqual(len(receipt["build_files"]), p.SOURCE_FILE_COUNT)
        self.assertIn("github.com/cometbft/cometbft v0.38.25\n", (output / "src/go.mod").read_text())
        for item in receipt["build_files"]:
            self.assertEqual(p.sha256(output / "src" / item["path"]), item["sha256"])
        frozen = {f["path"]: f["sha256"] for f in receipt["files"]}
        self.assertEqual({k: frozen[k] for k in ("go.mod", "go.sum")}, p.DEPENDENCY_FROZEN_SHA256)

    @unittest.skipUnless(has_commit(p.OLD_COMMIT), "legacy commit not in this clone")
    def test_legacy_helper_export_is_unpatched(self):
        output = self.root / "old"
        output.mkdir()
        receipt = p.export_source(REPOSITORY, output, p.OLD_COMMIT)
        self.assertIsNone(receipt["dependency_patch_sha256"])
        self.assertEqual(receipt["build_source_sha256"], receipt["source_sha256"])
        self.assertIn("github.com/cometbft/cometbft v0.38.20\n", (output / "src/go.mod").read_text())

    def test_applier_refuses_any_drift(self):
        patch = p.read_dependency_patch()
        original = {"go.mod": b"not the frozen go.mod\n", "go.sum": b""}
        with self.assertRaisesRegex(ValueError, "pinned input"):
            p.apply_dependency_patch(original, patch)
        with self.assertRaisesRegex(ValueError, "hash differs"):
            p.apply_dependency_patch(original, patch + b" ")
        link = self.root / "patch"
        link.symlink_to(p.DEPENDENCY_PATCH)
        with self.assertRaises(ValueError):
            p.read_dependency_patch(link)
        large = self.root / "large"
        large.write_bytes(b"x" * (p.DEPENDENCY_PATCH_MAX_BYTES + 1))
        with self.assertRaisesRegex(ValueError, "bound"):
            p.read_dependency_patch(large)

    def test_linked_module_version_reads_go_version_m(self):
        text = ("/out/zeroned: go1.25.14\n\tpath\tgithub.com/zerone-chain/zerone/cmd/zeroned\n"
                "\tdep\tgithub.com/cometbft/cometbft\tv0.38.25\th1:x=\n\tdep\tgithub.com/cosmos/cosmos-sdk\tv0.50.15\th1:y=\n")
        self.assertEqual(p.linked_module_version(text, "github.com/cometbft/cometbft"), "v0.38.25")
        self.assertEqual(p.linked_module_version(text, "github.com/cosmos/cosmos-sdk"), "v0.50.15")
        self.assertIsNone(p.linked_module_version(text, "github.com/cometbft/cometbft-db"))

    def test_build_refuses_a_binary_still_linking_0_38_20(self):
        files = [{"path": "go.mod", "sha256": None}]
        def export(repository, output):
            (output / "src").mkdir()
            (output / "src/go.mod").write_text("unit\n")
            files[0]["sha256"] = p.sha256(output / "src/go.mod")
            return {"source_sha256": p.SOURCE_ARCHIVE_SHA256, "files": files, "build_files": files,
                    "dependency_patch_sha256": p.DEPENDENCY_PATCH_SHA256, "build_source_sha256": p.BUILD_SOURCE_SHA256}
        old = b"/out/zeroned: go1.25.14\n\tdep\tgithub.com/cometbft/cometbft\tv0.38.20\th1:x\n" \
              b"\tdep\tgithub.com/cosmos/cosmos-sdk\tv0.50.15\th1:y\n build GOOS=linux\n build GOARCH=amd64\n build CGO_ENABLED=0\n"
        def run(argv, **kwargs):
            if "build" in argv:
                out = next(v for v in argv if v.endswith(",dst=/out")).split("src=", 1)[1].split(",dst=")[0]
                Path(out, "zeroned").write_bytes(b"\x7fELF unit mock")
            elif "-m" in argv:
                kwargs["stdout"].write(old)
            elif "cat" in argv:
                kwargs["stdout"].write(b"-----BEGIN CERTIFICATE-----\nunit\n-----END CERTIFICATE-----\n")
            return subprocess.CompletedProcess(argv, 0)
        with mock.patch.object(p, "export_source", side_effect=export), mock.patch.object(p, "require_native_host"), \
             mock.patch.object(subprocess, "run", side_effect=run), self.assertRaisesRegex(ValueError, "patched CometBFT"):
            p.build(self.root, self.root / "build", self.root, False, "unix:///unit/docker.sock")
        self.assertFalse((self.root / "build/image.oci.tar").exists())

    def test_receipts_from_the_unpatched_build_are_refused(self):
        # The 2026-10-28 image and drill receipts carry neither field.
        for receipt in ({}, {"dependency_patch_sha256": p.DEPENDENCY_PATCH_SHA256},
                        {"dependency_patch_sha256": p.DEPENDENCY_PATCH_SHA256, "build_source_sha256": p.SOURCE_ARCHIVE_SHA256}):
            with self.assertRaisesRegex(ValueError, "cometbft-3"):
                runtime.check_build_closure(receipt, "receipt")
        runtime.check_build_closure({"dependency_patch_sha256": p.DEPENDENCY_PATCH_SHA256,
                                     "build_source_sha256": p.BUILD_SOURCE_SHA256}, "receipt")

    def test_cli_version_must_link_the_patched_engine(self):
        version = {"name": "zerone", "server_name": "zeroned", "commit": p.SOURCE_COMMIT, "version": p.VERSION,
                   "build_deps": ["github.com/cometbft/cometbft@v0.38.25"]}
        runtime.check_version(version)
        for deps in (None, [], ["github.com/cometbft/cometbft@v0.38.20"], "github.com/cometbft/cometbft@v0.38.25"):
            with self.subTest(deps=deps), self.assertRaises(ValueError):
                runtime.check_version(dict(version, build_deps=deps))
        with self.assertRaises(ValueError):
            runtime.check_version(dict(version, version="h1-testnet-65c19cd8"))

    def test_machine_update_swap_and_drill_freshness(self):
        command = ship.deployment_command(None, IMAGE, "d8d3736f455508")
        self.assertEqual(command, ["fly", "machine", "update", "d8d3736f455508", "--app", ship.CHAIN, "--image", IMAGE])
        for machine in ("d8d3736f45550", "D8D3736F455508", "d8d3736f455508 --yes", "../x"):
            with self.assertRaises(ValueError):
                ship.deployment_command(None, IMAGE, machine)
        receipt = self.root / "drill.json"
        target = 3_900_000
        lag = ship.MAX_DRILL_LAG_BLOCKS
        for copied, ok in ((target - 1, True), (target - lag, True), (target - lag - 1, False),
                           (3_585_179, False), (target, False), (0, False), (str(target - 1), False)):
            p.write_json(receipt, {"copy_metadata": {"height": copied}})
            with self.subTest(copied=copied):
                if ok:
                    ship.check_drill_fresh(receipt, target)
                else:
                    with self.assertRaises(ValueError):
                        ship.check_drill_fresh(receipt, target)

    def test_drill_refuses_helpers_built_before_the_patch(self):
        helpers = self.root / "helpers"
        helpers.mkdir()
        p.write_json(helpers / "helpers.json", {"schema": "zerone-h1-drill-helpers/v1",
                                                "helper_source_sha256": p.sha256(HERE / "rehearsal.go"),
                                                "h1": {"source_commit": p.SOURCE_COMMIT, "source_sha256": p.SOURCE_ARCHIVE_SHA256}})
        copy = self.root / "copy"
        (copy / "application.db").mkdir(parents=True)
        (copy / "application.db/CURRENT").write_text("MANIFEST-000001\n")
        p.write_json(copy / "copy.json", {"schema": "zerone-h1-scrubbed-copy/v1", "chain_id": ship.CHAIN, "height": 5,
                                          "app_hash": "0" * 64, "backend": "goleveldb", "key_scrubbed": True,
                                          "stopped_consistent_copy": True, "block_time": "2026-11-06T00:00:00Z",
                                          "evidence_class": "provided-state-copy", "provenance": "unit mock"})
        with self.assertRaisesRegex(ValueError, "rebuild helpers"):
            drill.drill(copy, helpers, self.root / "out")
        self.assertFalse((self.root / "out").exists())

    def test_image_receipt_records_the_closure(self):
        binary = self.root / "zeroned"
        binary.write_bytes(b"\x7fELF unit")
        ca = self.root / "ca"
        ca.write_bytes(b"-----BEGIN CERTIFICATE-----\nunit\n-----END CERTIFICATE-----\n")
        first = p.package_oci(binary, self.root, p.SOURCE_ARCHIVE_SHA256, ca, p.BUILD_SOURCE_SHA256, p.DEPENDENCY_PATCH_SHA256)
        self.assertRegex(first["image_digest"], r"sha256:[0-9a-f]{64}")
        self.assertNotEqual(p.CROSS_BUILD_SHA256, p.RETIRED_CROSS_BUILD_SHA256)
        self.assertEqual(json.loads(json.dumps(p.DEPENDENCY_TARGET_SHA256)), p.DEPENDENCY_TARGET_SHA256)


if __name__ == "__main__":
    unittest.main(verbosity=2)

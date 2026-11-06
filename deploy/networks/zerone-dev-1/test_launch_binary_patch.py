#!/usr/bin/env python3
"""Run the binary-patch launcher against a fake runtime only; no node, key or network."""
import hashlib
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

LAUNCHER = Path(__file__).with_name("launch-binary-patch.py")
UPGRADE = "a" * 64
# The fake runtime keeps only the launcher-facing contract: stage-patch checks the
# packet against its pin and creates the pin-named directory; run-upgrade needs it.
FAKE_RUNTIME = '''import argparse, hashlib, json, os, signal
from pathlib import Path
p = argparse.ArgumentParser()
p.add_argument("command")
for flag in ("home", "upgrade-packet-sha256", "patch-packet", "patch-packet-sha256", "binary"):
    p.add_argument("--" + flag)
a = p.parse_args()
base = Path(__file__).parent
with (base / "calls.jsonl").open("a") as out:
    out.write(json.dumps(vars(a), sort_keys=True) + "\\n")
stage = Path(a.home) / ("binary-patch-" + a.patch_packet_sha256)
if a.command == "stage-patch":
    if hashlib.sha256(Path(a.patch_packet).read_bytes()).hexdigest() != a.patch_packet_sha256:
        raise SystemExit(41)
    stage.mkdir()
    if (base / "signal-stage").exists():
        os.kill(os.getpid(), signal.SIGTERM)
    if (base / "fail-stage").exists():
        raise SystemExit(23)
    if (base / "success-incomplete").exists():
        raise SystemExit(0)
    (stage / "validated.json").write_text(json.dumps({"upgrade": a.upgrade_packet_sha256, "patch": a.patch_packet_sha256}))
    raise SystemExit(0)
if a.command == "run-upgrade":
    if stage.is_symlink() or not stage.is_dir() or not (stage / "validated.json").is_file():
        raise SystemExit(31)
    if json.loads((stage / "validated.json").read_text()) != {"upgrade": a.upgrade_packet_sha256, "patch": a.patch_packet_sha256}:
        raise SystemExit(32)
    print(json.dumps({"fake_runtime_only": True, "status": "run-upgrade-selected"}))
    raise SystemExit(0)
raise SystemExit(33)
'''


class LaunchBinaryPatchTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="binary patch launch ")
        self.root = Path(self.temp.name)
        self.home = self.root / "owned home"
        self.home.mkdir(mode=0o700)
        self.runtime = self.root / "fake runtime.py"
        self.runtime.write_text(FAKE_RUNTIME)
        self.packet, self.binary = self.root / "patch [literal].json", self.root / "patched binary"
        self.packet.write_text('{"synthetic":"patch packet, never executed"}')
        self.binary.write_text("synthetic input, never executed")
        self.pin = hashlib.sha256(self.packet.read_bytes()).hexdigest()
        self.stage = self.home / ("binary-patch-" + self.pin)
        self.preserved = {p: hashlib.sha256(p.read_bytes()).hexdigest() for p in (self.packet, self.binary, self.runtime)}

    def tearDown(self):
        self.assertEqual(self.preserved, {p: hashlib.sha256(p.read_bytes()).hexdigest() for p in self.preserved})
        self.temp.cleanup()

    def invoke(self, pin=None, upgrade=UPGRADE):
        return subprocess.run([sys.executable, "-I", "-B", str(LAUNCHER),
            "--home", str(self.home), "--runtime", str(self.runtime),
            "--upgrade-packet-sha256", upgrade, "--patch-packet", str(self.packet),
            "--patch-packet-sha256", pin or self.pin, "--patch-binary", str(self.binary)],
            capture_output=True, text=True, timeout=10)

    def calls(self):
        path = self.root / "calls.jsonl"
        return [json.loads(line) for line in path.read_text().splitlines()] if path.exists() else []

    def commands(self):
        return [call["command"] for call in self.calls()]

    def test_first_start_stages_once_then_executes_exact_argv(self):
        result = self.invoke()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(json.loads(result.stdout)["status"], "run-upgrade-selected")
        stage, run = self.calls()
        self.assertEqual(stage, {"command": "stage-patch", "home": str(self.home), "upgrade_packet_sha256": UPGRADE,
            "patch_packet": str(self.packet), "patch_packet_sha256": self.pin, "binary": str(self.binary)})
        self.assertEqual(run, {"command": "run-upgrade", "home": str(self.home), "upgrade_packet_sha256": UPGRADE,
            "patch_packet": None, "patch_packet_sha256": self.pin, "binary": None})

    def test_restart_does_not_restage(self):
        self.assertEqual(self.invoke().returncode, 0)
        self.assertEqual(self.invoke().returncode, 0)
        self.assertEqual(self.commands(), ["stage-patch", "run-upgrade", "run-upgrade"])

    def test_failed_stage_preserved_and_never_runs_or_retries(self):
        (self.root / "fail-stage").touch()
        self.assertEqual(self.invoke().returncode, 23)
        self.assertTrue(self.stage.is_dir())
        self.assertEqual(self.commands(), ["stage-patch"])
        self.assertEqual(self.invoke().returncode, 31)
        self.assertEqual(self.commands(), ["stage-patch", "run-upgrade"])

    def test_signalled_stage_cannot_start_runtime(self):
        (self.root / "signal-stage").touch()
        self.assertEqual(self.invoke().returncode, 143)
        self.assertEqual(self.commands(), ["stage-patch"])

    def test_stage_exit_zero_does_not_bypass_runtime_validation(self):
        (self.root / "success-incomplete").touch()
        self.assertEqual(self.invoke().returncode, 31)
        self.assertEqual(self.commands(), ["stage-patch", "run-upgrade"])
        self.assertTrue(self.stage.is_dir())

    def test_partial_file_and_broken_symlink_go_to_strict_loader(self):
        self.stage.write_text("partial staging")
        self.assertEqual(self.invoke().returncode, 31)
        self.assertEqual(self.stage.read_text(), "partial staging")
        self.assertEqual(self.commands(), ["run-upgrade"])
        # Only the test's own fixture changes shape; the launcher has no unlink,
        # rmdir, symlink creation, cleanup or repair operation.
        self.stage.unlink()
        self.stage.symlink_to(self.root / "missing")
        self.assertEqual(self.invoke().returncode, 31)
        self.assertTrue(self.stage.is_symlink())
        self.assertEqual(self.commands(), ["run-upgrade", "run-upgrade"])

    def test_changed_external_pins_are_not_rebound(self):
        self.assertEqual(self.invoke().returncode, 0)
        # Another patch pin names another directory, so it must stage its own packet,
        # which differs from that pin; nothing runs.
        self.assertEqual(self.invoke(pin="b" * 64).returncode, 41)
        self.assertFalse((self.home / ("binary-patch-" + "b" * 64)).exists())
        self.assertEqual(self.invoke(upgrade="c" * 64).returncode, 32)
        self.assertEqual(self.commands(), ["stage-patch", "run-upgrade", "stage-patch", "run-upgrade"])

    def test_invalid_or_equal_pins_refused_before_fake_commands(self):
        for pin, upgrade in (("0" * 64, UPGRADE), ("A" * 64, UPGRADE), ("bad", UPGRADE), (None, "0" * 64), (None, "short"), (None, None)):
            result = self.invoke(pin, upgrade or self.pin)
            self.assertEqual(result.returncode, 2, (pin, upgrade))
        relative = subprocess.run([sys.executable, "-I", "-B", str(LAUNCHER), "--home", "relative/home",
            "--runtime", str(self.runtime), "--upgrade-packet-sha256", UPGRADE, "--patch-packet", str(self.packet),
            "--patch-packet-sha256", self.pin, "--patch-binary", str(self.binary)], capture_output=True, text=True, timeout=10)
        self.assertEqual(relative.returncode, 2)
        self.assertEqual(self.calls(), [])


if __name__ == "__main__":
    unittest.main()

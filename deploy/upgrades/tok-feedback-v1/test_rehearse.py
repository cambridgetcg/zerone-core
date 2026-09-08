import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock

spec = importlib.util.spec_from_file_location("driver", Path(__file__).with_name("rehearse.py"))
driver = importlib.util.module_from_spec(spec)
spec.loader.exec_module(driver)

class ValidationTests(unittest.TestCase):
    def test_duplicate_json_rejected(self):
        with tempfile.TemporaryDirectory() as d:
            p = Path(d) / "input.json"
            p.write_text('{"a":1,"a":2}')
            with self.assertRaises(ValueError):
                driver.load(p)

    def test_outside_and_symlink_refused(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d).resolve()
            p = root / "inside"
            p.write_bytes(b"data")
            link = root / "link"
            link.symlink_to(p)
            with self.assertRaises(ValueError):
                driver.confined(link, root)
            with self.assertRaises(ValueError):
                driver.confined(root.parent / "elsewhere", root)

    def test_dirty_stop_never_force_kills(self):
        import subprocess
        proc = mock.Mock()
        proc.poll.return_value = None
        proc.wait.side_effect = subprocess.TimeoutExpired("fixture", 30)
        with self.assertRaises(RuntimeError):
            driver.stop(proc)
        proc.kill.assert_not_called()

    def test_validation_cannot_launch(self):
        with tempfile.TemporaryDirectory() as d:
            root = Path(d).resolve()
            home = root / "fixture"
            (home / "config").mkdir(parents=True)
            (home / "config/genesis.json").write_text('{"chain_id":"tok-feedback-rehearsal-unit"}')
            (home / "config/config.toml").write_text('[p2p]\npersistent_peers=""\nseeds=""\npex=false\n')
            old, new = root / "old", root / "new"
            # Intentionally non-executable validation fixtures, never binaries.
            old.write_bytes(b"old-input"); new.write_bytes(b"new-input")
            m = {"schema":"zerone.tok-feedback-v1/local-rehearsal/v1",
                 "synthetic_fixture_only":True,"candidate_consensus_delta_reviewed":True,
                 "h3_source":driver.H3,"h3_tree":driver.TREE,
                 "old_binary":str(old),"new_binary":str(new),
                 "old_binary_sha256":driver.digest(old),"new_binary_sha256":driver.digest(new),
                 "stopped_fixture_home":str(home),"stopped_fixture_files_sha256":driver.tree_manifest(home),
                 "chain_id":"tok-feedback-rehearsal-unit","upgrade_height":10,
                 "ports":[20111,20112,20113,20114,20115,20116]}
            manifest = root / "manifest.json"
            manifest.write_text(json.dumps(m))
            with mock.patch.object(driver,"SANDBOX",root), mock.patch("sys.argv",["driver",str(manifest)]), mock.patch.object(driver.subprocess,"Popen") as launch, mock.patch("builtins.print") as output:
                driver.main()
                launch.assert_not_called()
                result = json.loads(output.call_args.args[0])
                self.assertEqual(result["status"],"NO_GO")
                self.assertFalse(result["executed"])
                self.assertFalse(result["custody_verified"])

if __name__ == "__main__":
    unittest.main()

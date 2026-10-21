"""Offline version-contract tests. No node or key is opened."""
import copy
import importlib.util
from pathlib import Path
import unittest
from unittest import mock

spec = importlib.util.spec_from_file_location("carrier_runtime", Path(__file__).with_name("runtime.py"))
assert spec is not None and spec.loader is not None
r = importlib.util.module_from_spec(spec)
spec.loader.exec_module(r)


def packet(name="consensus-repairs-v1"):
    source, target = r.UPGRADE_VERSIONS[name]
    return {"schema": "zerone-development-upgrade/v1", "chain_id": r.CHAIN,
            "genesis_sha256": "a" * 64, "predecessor_descriptor_sha256": "b" * 64,
            "plan": {"name": name, "height": "42", "info": ""},
            "predecessor": {"knowledge_version": source, "source_commit": "a" * 40,
                            "binaries": {p: "c" * 64 for p in ("linux-amd64", "darwin-arm64")}},
            "target": {"knowledge_version": target, "source_commit": "b" * 40,
                       "binaries": {p: "d" * 64 for p in ("linux-amd64", "darwin-arm64")}}}


class ConsensusRepairsVersions(unittest.TestCase):
    def test_successors_are_exact_not_jumps(self):
        for name in r.UPGRADE_VERSIONS:
            p = packet(name)
            self.assertEqual(r.validate_upgrade_packet(p), p)
            r.matching_upgrade_plan(p["plan"], p)
            wrong = copy.deepcopy(p)
            wrong["predecessor"]["knowledge_version"] -= 1
            with self.assertRaises(r.RuntimeError):
                r.validate_upgrade_packet(wrong)
            with self.assertRaises(r.RuntimeError):
                r.matching_upgrade_plan({"name": "wrong", "height": "42"}, p)

    def test_applied_carrier_requires_all_versions(self):
        for name, expected in (("consensus-repairs-v1", r.CONSENSUS_REPAIRS_VERSIONS),
                               ("consensus-repairs-v2", r.CONSENSUS_REPAIRS_V2_VERSIONS)):
            p = packet(name)
            for missing in (None, *expected):
                vm = dict(expected)
                if missing:
                    del vm[missing]
                results = [{"module_versions": [{"name": n, "version": str(v)} for n, v in vm.items()]}, {"height": "42"}]
                with mock.patch.object(r, "upgrade_query", side_effect=results) as query:
                    if missing:
                        with self.assertRaises(r.RuntimeError):
                            r.verify_applied_upgrade(None, None, None, p)
                    else:
                        result = r.verify_applied_upgrade(None, None, None, p)
                        self.assertEqual(result["module_versions"], vm)
                        self.assertEqual(query.call_args.args[-1], name)


if __name__ == "__main__":
    unittest.main()

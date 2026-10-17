#!/usr/bin/env python3
"""Synthetic-export tests for audit-augmentation-edge-overwrites.py (stdlib only).

Run: python3 -I -m unittest discover -s scripts -p 'test_audit_augmentation_edge_overwrites.py' -v
"""

import copy
import hashlib
import importlib.util
import io
import json
from pathlib import Path
import tempfile
import unittest

SPEC = importlib.util.spec_from_file_location("audit_augmentation_edge_overwrites", Path(__file__).with_name("audit-augmentation-edge-overwrites.py"))
audit = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(audit)

SUBMITTER = "zrn1synthetic-submitter"


def fact_id(claim_id, height):
    return hashlib.sha256(f"ZRN.fact.id.v1:{claim_id}:{height}".encode()).hexdigest()[:32]


def aug_edge(aug_id, target, block):
    """The edge a passing verdict wrote, as an export lists it."""
    return {"source_fact_id": aug_id, "target_fact_id": target, "relation": "RELATION_TYPE_REFORMULATES",
            "created_at_block": str(block), "creator": "", "inference": "INFERENCE_TYPE_UNSPECIFIED",
            "inference_strength_bps": "0", "method_id": ""}


class Export:
    """Builds a synthetic knowledge genesis in the exported JSON shape."""

    def __init__(self):
        self.state = {"facts": [], "pending_claims": [], "active_rounds": [], "completed_rounds": [],
                      "augmentations": [], "fact_relation_state": {"relations": []}}
        self.target("synthetic-d")

    def target(self, fid):
        self.state["facts"].append({"id": fid, "claim_id": "", "verified_at_block": "1"})
        return fid

    def claim_born(self, tag, relations, created=10, scheme=2):
        claim_id = f"synthetic-claim-{tag}"
        fid = fact_id(claim_id, created)
        round_id = f"synthetic-round-{tag}"
        self.state["pending_claims"].append({"id": claim_id, "submitter": SUBMITTER, "verification_round_id": round_id, "relations": relations})
        self.state["completed_rounds"].append({"id": round_id, "claim_id": claim_id, "commitment_scheme": scheme})
        self.state["facts"].append({"id": fid, "claim_id": claim_id, "verified_at_block": str(created)})
        return claim_id, fid

    def replaced(self, aug_id, target="synthetic-d", block=12, verdict="AUGMENTATION_VERDICT_EQUIVALENT"):
        self.state["augmentations"].append({"id": aug_id, "original_fact_id": target, "verdict": verdict, "verdict_block": str(block)})
        self.state["fact_relation_state"]["relations"].append(aug_edge(aug_id, target, block))


def requires(target="synthetic-d", **extra):
    relation = {"target_fact_id": target, "relation": "RELATION_TYPE_REQUIRES", "inference": "INFERENCE_TYPE_UNSPECIFIED", "inference_strength_bps": "0", "method_id": ""}
    relation.update(extra)
    return relation


class AuditTest(unittest.TestCase):
    def test_restorable_pair_lists_the_edge_the_chain_writes(self):
        export = Export()
        _, fid = export.claim_born("a", [
            requires(relation="RELATION_TYPE_SUPPORTS"),
            requires(inference="INFERENCE_TYPE_INDUCTIVE", inference_strength_bps="700000", method_id="M-HISTORICAL"),
        ])
        export.replaced(fid)
        report = audit.audit({"app_state": {"knowledge": export.state}})
        self.assertEqual({"restorable": 1, "unrestorable": 0}, report["counts"])
        self.assertEqual([{
            "augmentation_id": fid, "fact_id": fid, "target_id": "synthetic-d", "verdict_block": 12, "status": "restorable",
            "restored_edge": {"source_fact_id": fid, "target_fact_id": "synthetic-d", "relation": "RELATION_TYPE_REQUIRES",
                              "created_at_block": 10, "creator": SUBMITTER, "inference": "INFERENCE_TYPE_INDUCTIVE",
                              "inference_strength_bps": 700000, "method_id": "M-HISTORICAL"},
        }], report["pairs"], "the claim's last relation naming the target is the stored one")

    def test_same_block_verdict_is_listed(self):
        export = Export()
        _, fid = export.claim_born("b", [requires()], created=12)
        export.replaced(fid, block=12)
        self.assertEqual(1, audit.audit(export.state)["counts"]["restorable"])

    def test_unrestorable_reasons_match_the_chain(self):
        for reason in (
            "fact has no claim",
            "claim synthetic-claim-m of the fact is missing",
            "fact id does not derive from its claim and verification height",
            "relation method_id is set but the claim's round does not prove record integrity at the fact's creation",
        ):
            with self.subTest(reason=reason):
                export = Export()
                if reason == "fact has no claim":
                    fid = export.target("synthetic-genesis-f")
                elif reason.startswith("claim "):
                    claim_id, fid = export.claim_born("m", [requires()])
                    export.state["pending_claims"] = [c for c in export.state["pending_claims"] if c["id"] != claim_id]
                elif reason.startswith("fact id"):
                    _, fid = export.claim_born("x", [requires()])
                    next(f for f in export.state["facts"] if f["id"] == fid)["verified_at_block"] = "9"
                else:
                    _, fid = export.claim_born("l", [requires(method_id="M-HISTORICAL")], scheme=0)
                export.replaced(fid)
                report = audit.audit(export.state)
                self.assertEqual([{"augmentation_id": fid, "fact_id": fid, "target_id": "synthetic-d", "verdict_block": 12,
                                   "status": "unrestorable", "reason": reason}], report["pairs"])
                self.assertEqual({"restorable": 0, "unrestorable": 1}, report["counts"])

    def test_pairs_that_replaced_nothing_are_not_listed(self):
        export = Export()
        _, named = export.claim_born("n", [requires()])
        _, unnamed = export.claim_born("u", [requires(target="synthetic-other")])
        export.target("synthetic-other")
        _, late = export.claim_born("t", [requires()], created=20)
        _, drift = export.claim_born("r", [requires()])
        export.replaced("synthetic-ordinary-variant")  # not a fact
        export.replaced(unnamed)  # a stray edge: F's claim does not name D
        export.replaced(late, block=12)  # F created after the verdict
        export.replaced(drift, verdict="AUGMENTATION_VERDICT_DRIFT")  # no edge written
        export.state["augmentations"].append({"id": named, "original_fact_id": "synthetic-d", "verdict": "AUGMENTATION_VERDICT_SUPERIOR", "verdict_block": "12"})
        own = requires()
        export.state["fact_relation_state"]["relations"].append({"source_fact_id": named, "target_fact_id": "synthetic-d", "relation": own["relation"],
                                                                 "created_at_block": "10", "creator": SUBMITTER, "inference": own["inference"],
                                                                 "inference_strength_bps": "0", "method_id": ""})  # F's own edge was written last
        report = audit.audit(export.state)
        self.assertEqual([], report["pairs"])
        self.assertEqual(5, report["augmentations"])

    def test_numeric_enums_and_ordering_are_deterministic(self):
        export = Export()
        ids = []
        for tag in ("z", "y", "x"):
            _, fid = export.claim_born(tag, [requires()])
            export.replaced(fid)
            ids.append(fid)
        numeric = copy.deepcopy(export.state)
        for relation in numeric["fact_relation_state"]["relations"]:
            relation["relation"], relation["inference"] = 8, 0
        for aug in numeric["augmentations"]:
            aug["verdict"] = 1
        for claim in numeric["pending_claims"]:
            claim["relations"][0]["relation"] = 3
        named, numbered = audit.audit(export.state), audit.audit(numeric)
        self.assertEqual(named, numbered)
        self.assertEqual(sorted(ids), [p["augmentation_id"] for p in named["pairs"]], "augmentation key order")

    def test_absent_relation_state_is_reported(self):
        export = Export()
        _, fid = export.claim_born("s", [requires()])
        export.replaced(fid)
        export.state["fact_relation_state"] = None
        report = audit.audit(export.state)
        self.assertFalse(report["fact_relation_state_present"])
        self.assertEqual([], report["pairs"])

    def test_malformed_input_is_refused(self):
        for bad in ([], {"unrelated": 1}, {"app_state": {}}, {"facts": [{"id": "a"}, {"id": "a"}]},
                    {"facts": [], "augmentations": [{"id": "a", "verdict": "NOT_A_VERDICT"}]},
                    {"facts": [], "augmentations": [{"id": "a", "verdict": 1, "verdict_block": "-3"}]}):
            with self.subTest(bad=bad):
                with self.assertRaises(audit.AuditError):
                    audit.audit(bad)

    def test_cli_exit_status(self):
        export = Export()
        _, fid = export.claim_born("c", [requires()])
        export.replaced(fid)
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "export.json"
            path.write_text(json.dumps({"app_state": {"knowledge": export.state}}))
            out, err = io.StringIO(), io.StringIO()
            self.assertEqual(0, audit.main([str(path)], out, err))
            self.assertEqual(1, json.loads(out.getvalue())["counts"]["restorable"])
            self.assertEqual(1, audit.main([str(path), "--fail-on-any"], io.StringIO(), err))
            clean = Path(tmp) / "clean.json"
            clean.write_text(json.dumps(Export().state))
            self.assertEqual(0, audit.main([str(clean), "--fail-on-any"], io.StringIO(), err))
            broken = Path(tmp) / "broken.json"
            broken.write_text("{")
            err = io.StringIO()
            self.assertEqual(2, audit.main([str(broken)], io.StringIO(), err))
            self.assertIn("audit-augmentation-edge-overwrites:", err.getvalue())
            self.assertEqual(2, audit.main([str(Path(tmp) / "missing.json")], io.StringIO(), io.StringIO()))


if __name__ == "__main__":
    unittest.main()

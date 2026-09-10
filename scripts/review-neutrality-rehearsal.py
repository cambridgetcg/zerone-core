#!/usr/bin/env python3
"""Signed loopback-only record-integrity predecessor -> review-neutrality boundary.

Reuses the authenticated record rehearsal; adds preserved policy-0 in-flight
reviews and equal payment for a policy-1 panel containing scientific dissent.
Synthetic local participants are not independent scientific validators.
"""
import importlib.util
from pathlib import Path

spec = importlib.util.spec_from_file_location("record_rehearsal", Path(__file__).with_name("record-integrity-rehearsal.py"))
record = importlib.util.module_from_spec(spec)
spec.loader.exec_module(record)
record.PLAN = record.base.PLAN = "knowledge-review-neutrality-v1"

class Rehearsal(record.Rehearsal):
    source_knowledge_version = 8
    target_knowledge_version = 9
    predecessor_v2 = True
    neutral_reviews = True

    def __init__(self, before, after, directory):
        super().__init__(before, after, directory)
        self.evidence["schema"] = "zerone.review-neutrality/local-rehearsal-v1"
        self.evidence["script_sha256"] = record.base.sha256(Path(__file__))
        self.evidence["record_harness_sha256"] = record.base.sha256(Path(record.__file__))

if __name__ == "__main__":
    record.Rehearsal = Rehearsal
    record.main()

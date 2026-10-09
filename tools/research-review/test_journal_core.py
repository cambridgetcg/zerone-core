"""Standalone model and adapter composition contracts; synthetic, offline data."""

import hashlib
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

import journal as j
from test_journal import COLLECTION, STAMP, contribution, fixture, relation


CORE = Path(__file__).with_name("journal_core.py")

# The consumer has only the copied core, never the repository or journal store.
CONSUMER = r'''
import datetime as dt
import hashlib
import importlib.abc
import importlib.util
import ipaddress
import json
import os
import re
import socket
import sys
import uuid
from unittest.mock import patch
from urllib.parse import urlsplit

payload = json.loads(sys.stdin.buffer.read())
source = os.path.abspath(sys.argv[1])

class NoAdapters(importlib.abc.MetaPathFinder):
    def find_spec(self, fullname, path=None, target=None):
        if fullname.split(".")[0] in {"fcntl", "journal", "workflow", "scan", "checkpoint"}:
            raise AssertionError("standalone core imported an adapter: " + fullname)

sys.meta_path.insert(0, NoAdapters())

def no_effects(event, args):
    if event == "open":
        name, mode, flags = args
        # Importlib probes its own bytecode path even with Python -B; neither
        # this absent cache nor the source is application data, and writes stay denied.
        if name not in (source, importlib.util.cache_from_source(source)) or flags & (
                os.O_WRONLY | os.O_RDWR | os.O_CREAT | os.O_TRUNC):
            raise AssertionError("core attempted data-file I/O: " + repr(args))
    if event.startswith(("socket.", "subprocess.", "os.spawn", "os.exec")) or event in {
            "os.system", "os.mkdir", "os.remove", "os.rename", "os.rmdir"}:
        raise AssertionError("core attempted an external effect: " + event)

sys.addaudithook(no_effects)
spec = importlib.util.spec_from_file_location("detached_journal_core", source)
core = importlib.util.module_from_spec(spec)
spec.loader.exec_module(core)

class NoClock(dt.datetime):
    @classmethod
    def now(cls, *args, **kwargs):
        raise AssertionError("core read the clock")

with patch.object(dt, "datetime", NoClock), \
        patch.object(uuid, "uuid4", side_effect=AssertionError("core generated identity")), \
        patch.object(os, "urandom", side_effect=AssertionError("core requested randomness")):
    export = core.make_export(payload["records"], collection_id=payload["collection_id"],
                              created_at=payload["created_at"], recorded_at=payload["recorded_at"])
    assert core.validate_export(core.parse_json(core.canonical(export))) == export
    result = {"export": export, "prefix": core.select_snapshot(export, 1),
              "comparison": core.compare_exports(core.select_snapshot(export, 1), export),
              "impact": core.impact(export, "a")}
    try:
        core.make_export(payload["records"])
    except TypeError:
        pass
    else:
        raise AssertionError("core must require caller-owned identity and time")
    refused = []
    for effect in (lambda: open("must-not-exist", "w"), lambda: socket.socket()):
        try:
            effect()
        except AssertionError:
            refused.append(True)
    assert refused == [True, True], "I/O guards need positive controls"
    sys.stdout.buffer.write(core.canonical(result))
'''


class StandaloneCoreTests(unittest.TestCase):
    def test_v1_constructor_refuses_invalid_records_before_clock_or_identity(self):
        from unittest.mock import patch

        for name, records in (("none", None), ("tuple", ()), ("mapping", {}),
                              ("string", "[]"), ("overcapacity", [None] * (j.MAX_ENTRIES + 1))):
            with self.subTest(container=name), \
                    patch.object(j, "_now", side_effect=AssertionError("clock called")) as clock, \
                    patch.object(j.uuid, "uuid4", side_effect=AssertionError("UUID called")) as identity:
                with self.assertRaises(j.JournalError) as refusal:
                    j.make_export(records)
                clock.assert_not_called()
                identity.assert_not_called()
                self.assertEqual(str(refusal.exception),
                                 "records must be a list of at most 1000 entries")

    def test_v1_constructor_retains_frozen_bytes_and_adapter_defaults(self):
        from unittest.mock import patch

        # Captured from journal.py at df848ce79c0b9ff02279a3ca2b3e46b216ae4574.
        records = [{"id": "a", "kind": "contribution", "title": "Synthetic é😀",
                    "summary": "Fixture contribution; no scientific assertion.",
                    "attributed_to": "Fictional reviewer", "occurred_on": None,
                    "evidence": [], "contribution_type": "question", "doi": None}]
        stamp = "2026-09-15T12:00:00.000000Z"
        explicit = j.core.make_export(records, collection_id=COLLECTION, created_at=stamp)
        with patch.object(j, "_now", return_value=stamp) as clock, \
                patch.object(j.uuid, "uuid4", return_value=j.uuid.UUID(COLLECTION)) as identity:
            legacy = j.make_export(records)
        clock.assert_called_once_with()
        identity.assert_called_once_with()
        self.assertEqual(legacy, explicit)
        self.assertEqual(hashlib.sha256(j.canonical(explicit)).hexdigest(),
                         "30b41353bf6e5c1649a66f02a0bb4132131125e4578aca9286a9be69421ddb1a")

    def test_file_adapter_uses_the_same_record_contract_as_the_standalone_core(self):
        from unittest.mock import patch

        self.assertTrue(hasattr(j, "core"), "journal storage must compose the shared core, not copy its rules")
        core = j.core
        self.assertIs(j.JournalError, core.JournalError)
        self.assertIs(j.validate_export, core.validate_export)
        self.assertIs(j.compare_exports, core.compare_exports)
        self.assertIs(j.select_snapshot, core.select_snapshot)
        self.assertIs(j.impact, core.impact)
        with tempfile.TemporaryDirectory() as directory:
            store = Path(directory).resolve() / "journal"
            j.create_store(store)
            with patch.object(core, "validate_record", wraps=core.validate_record) as validate:
                result = j.append_records(store, [contribution("persisted")])
            self.assertGreater(validate.call_count, 0)
            self.assertEqual(core.validate_export(j.export_store(store)), result)
            self.assertEqual(result["entries"][0]["record"]["id"], "persisted")

    def test_core_runs_alone_without_storage_clock_network_or_adapter_imports(self):
        self.assertTrue(CORE.is_file(), "research graph needs its standalone storage-free core")
        records = [contribution("a", title="Synthetic é😀"), contribution("b"),
                   relation("a-b", "a", "b")]
        export = fixture(records)
        expected = {"export": export, "prefix": j.select_snapshot(export, 1),
                    "comparison": j.compare_exports(j.select_snapshot(export, 1), export),
                    "impact": j.impact(export, "a")}
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
            copied = root / CORE.name
            shutil.copyfile(CORE, copied)
            consumer = root / "consumer.py"
            consumer.write_text(CONSUMER, encoding="utf-8")
            before = {p.name: p.read_bytes() for p in root.iterdir()}
            payload = {"records": records, "collection_id": COLLECTION, "created_at": STAMP,
                       "recorded_at": [row["recorded_at"] for row in export["entries"]]}
            result = subprocess.run([sys.executable, "-I", "-B", str(consumer), str(copied)],
                                    cwd=root, input=j.canonical(payload), stdout=subprocess.PIPE,
                                    stderr=subprocess.PIPE, timeout=15)
            self.assertEqual(result.returncode, 0, result.stderr.decode())
            self.assertEqual(result.stderr, b"")
            self.assertEqual(result.stdout, j.canonical(expected))
            self.assertEqual({p.name: p.read_bytes() for p in root.iterdir()}, before)


if __name__ == "__main__":
    unittest.main()

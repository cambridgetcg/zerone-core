"""Meaningful contract tests; no network or external identities."""

import copy
import hashlib
import json
import os
from pathlib import Path
import stat
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch

import journal as j


STAMP = "2026-01-01T00:00:00.000000Z"
COLLECTION = "12345678-1234-4234-8234-123456789abc"


def contribution(rid, **changes):
    record = {"id": rid, "kind": "contribution", "title": "Fictional " + rid,
              "summary": "A scoped fictional record.", "attributed_to": "Fixture author",
              "occurred_on": None, "evidence": [], "contribution_type": "claim", "doi": None}
    record.update(changes)
    return record


def relation(rid, source, target, kind="supports", **changes):
    record = contribution(rid)
    del record["contribution_type"], record["doi"]
    record.update(kind="relation", from_id=source, to_id=target, relation_type=kind)
    record.update(changes)
    return record


def concern(rid, target):
    record = contribution(rid)
    del record["contribution_type"], record["doi"]
    record.update(kind="concern", target_id=target, category="reported-concern",
                  notice_type=None, provider="Fictional provider")
    return record


def assessment(rid, target, disposition="needs-review"):
    record = contribution(rid)
    del record["contribution_type"], record["doi"]
    record.update(kind="assessment", target_id=target, disposition=disposition)
    return record


def fixture(records):
    times = [f"2026-01-01T00:00:{n:02d}.000000Z" for n in range(len(records))]
    return j.make_export(records, collection_id=COLLECTION, created_at=STAMP, recorded_at=times)


class JournalTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        # macOS /var is a symlink; use the actual parent explicitly.
        self.parent = Path(self.temporary.name).resolve()
        self.store = self.parent / "collection"
        j.create_store(self.store)

    def tearDown(self):
        self.temporary.cleanup()

    def test_canonical_unicode_hash_vector_and_tamper(self):
        raw = j.canonical({"z": "é😀", "a": {"y": None, "b": 1}})
        self.assertEqual(raw, '{"a":{"b":1,"y":null},"z":"é😀"}'.encode())
        export = fixture([contribution("a", title="é😀")])
        self.assertEqual(export["entries"][0]["previous_sha256"],
                         hashlib.sha256(j.canonical(export["header"])).hexdigest())
        tampered = copy.deepcopy(export)
        tampered["entries"][0]["record"]["summary"] = "Changed after hashing"
        with self.assertRaisesRegex(j.JournalError, "hash mismatch"):
            j.validate_export(tampered)
        with self.assertRaises(j.JournalError):
            j.parse_json(b'{"a":1,"a":2}')
        for raw in [b'{"x":NaN}', b'{"x":Infinity}', b'"\xff"']:
            with self.subTest(raw=raw), self.assertRaises(j.JournalError):
                j.parse_json(raw)

    def test_locked_batch_preserves_previous_bytes_and_roundtrips(self):
        initial = (self.store / "journal.jsonl").read_bytes()
        first = j.append_records(self.store, [contribution("a")])
        before = (self.store / "journal.jsonl").read_bytes()
        second = j.append_records(self.store, [contribution("b"), relation("ab", "a", "b")])
        after = (self.store / "journal.jsonl").read_bytes()
        self.assertTrue(before.startswith(initial))
        self.assertTrue(after.startswith(before))
        self.assertEqual(first["entries"], second["entries"][:1])
        self.assertEqual([e["sequence"] for e in second["entries"]], [1, 2, 3])
        destination = self.parent / "export.json"
        j._write_output(destination, second)
        self.assertEqual(j.load_export(destination), j.load_export(self.store))
        self.assertEqual(os.stat(self.store).st_mode & 0o777, 0o700)
        self.assertEqual(os.stat(self.store / "journal.jsonl").st_mode & 0o777, 0o600)

    def test_entire_invalid_batch_refused_without_partial_record(self):
        before = (self.store / "journal.jsonl").read_bytes()
        for records in [[contribution("a"), relation("x", "a", "missing")],
                        [contribution("a"), contribution("a")]]:
            with self.assertRaises(j.JournalError):
                j.append_records(self.store, records)
            self.assertEqual((self.store / "journal.jsonl").read_bytes(), before)
        j.append_records(self.store, [contribution("a")])
        before = (self.store / "journal.jsonl").read_bytes()
        with self.assertRaises(j.JournalError):
            j.append_records(self.store, [contribution("a", summary="Replacement")])
        self.assertEqual((self.store / "journal.jsonl").read_bytes(), before)

    def test_schema_reference_and_text_refusals(self):
        earlier = {"a": contribution("a"), "b": contribution("b"), "notice": concern("notice", "a")}
        invalid = [contribution("Bad ID"), contribution("a"), contribution("new", extra="hidden"),
                   contribution("new", summary="😀" * 2049), contribution("new", title="x" * 301),
                   contribution("new", title="\ud800"), contribution("new", occurred_on="2025-02-29"),
                   contribution("new", occurred_on="0000-01-01"), contribution("new", doi="10.1234/ABC"),
                   contribution("new", doi="https://doi.org/10.1234/abc"),
                   contribution("new", supersedes=None), contribution("new", supersedes="notice"),
                   relation("r", "a", "a"), relation("r", "a", "notice"),
                   relation("r", "a", "b", "deductive-proof"), assessment("r", "a")]
        for record in invalid:
            with self.subTest(record=record), self.assertRaises(j.JournalError):
                j.validate_record(record, earlier)
        j.validate_record(contribution("c", supersedes="a", occurred_on="0001-01-01", doi="10.1234/a:b"), earlier)
        j.validate_record(assessment("review", "notice"), earlier)

    def test_evidence_urls_digests_and_record_byte_ceiling(self):
        for url in ["http://example.org", "https://user:password@example.org", "https://example.org/white space",
                    "https://example.org:bad", "https://example.org:", "https://example.org:65536",
                    "https://example.org\\@elsewhere.org", "https://", "https://[broken",
                    "https://[::1]ignored", "https://[::1]:", "https://[v1.name]", "https://[fe80::1%25zone]/"]:
            record = contribution("a", evidence=[{"label": "source", "url": url, "sha256": None}])
            with self.subTest(url=url), self.assertRaises(j.JournalError):
                j.validate_record(record, {})
        for evidence in [[{"label": "x", "url": None, "sha256": None}],
                         [{"label": "x", "url": None, "sha256": "A" * 64}],
                         [{"label": "x", "url": "https://example.org", "sha256": None}] * 17]:
            with self.assertRaises(j.JournalError):
                j.validate_record(contribution("a", evidence=evidence), {})
        good = {"label": "Exact artifact", "url": "https://example.org:443/a#b", "sha256": "a" * 64}
        j.validate_record(contribution("a", evidence=[good]), {})
        for url in ["https://[::1]/a", "https://[2001:DB8::1]:443/a", "https://[::ffff:192.0.2.1]/"]:
            j.validate_record(contribution("a", evidence=[dict(good, url=url)]), {})
        empty_label = concern("notice", "a")
        empty_label["notice_type"] = ""
        with self.assertRaises(j.JournalError):
            j.validate_record(empty_label, {"a": contribution("a")})
        huge = {"label": "x", "url": "https://example.org/" + "😀" * 2000, "sha256": None}
        with self.assertRaisesRegex(j.JournalError, "64 KiB"):
            j.validate_record(contribution("a", evidence=[huge] * 16), {})

    def test_cutoff_excludes_late_origins_reviews_dates_and_assessments(self):
        records = [contribution("claim", occurred_on="2024-06-01"),
                   contribution("late-source", occurred_on="1990-01-01"),
                   relation("source-support", "late-source", "claim"),
                   concern("notice", "claim"), assessment("response", "notice", "disputed"),
                   contribution("claim-v2", supersedes="claim", occurred_on="1989-01-01"),
                   assessment("other-response", "notice", "addressed")]
        export = fixture(records)
        early = j.select_snapshot(export, 1)
        self.assertEqual([row["record"]["id"] for row in early["entries"]], ["claim"])
        self.assertEqual(early["entries"][0]["record"]["occurred_on"], "2024-06-01")
        self.assertEqual(len(j.select_snapshot(export, 0)["entries"]), 0)
        self.assertEqual(len(j.select_snapshot(export, 4)["entries"]), 4)
        self.assertEqual([r["record"]["disposition"] for r in export["entries"] if r["record"]["kind"] == "assessment"],
                         ["disputed", "addressed"])
        early["entries"][0]["record"]["title"] = "Detached copy"
        self.assertNotEqual(export["entries"][0]["record"]["title"], "Detached copy")
        tampered_future = copy.deepcopy(export)
        tampered_future["entries"][-1]["sha256"] = "0" * 64
        with self.assertRaises(j.JournalError):
            j.select_snapshot(tampered_future, 1)
        for cutoff in [-1, 8, True, 1.0, "1"]:
            with self.subTest(cutoff=cutoff), self.assertRaises(j.JournalError):
                j.select_snapshot(export, cutoff)

    def test_impact_direction_attribution_cycles_and_controls(self):
        records = [contribution(rid) for rid in ["a", "b", "c", "d", "cite", "inspiration", "refinement"]]
        records += [relation("ab", "a", "b"), relation("cb", "c", "b", "requires", attributed_to="Reviewer B"),
                    relation("dc", "d", "c", "uses-input"), relation("cycle", "d", "a"),
                    relation("citation", "cite", "a", "cites"),
                    relation("inspired", "inspiration", "a", "inspired-by"),
                    relation("refinement-link", "refinement", "a", "refines")]
        export = fixture(records)
        result = j.impact(export, "a")
        self.assertEqual([p["record_ids"] for p in result["paths"]], [["a", "b"], ["a", "b", "c"], ["a", "b", "c", "d"]])
        self.assertEqual(result["paths"][-1]["propagation"], ["support", "dependency", "input"])
        self.assertEqual(result["paths"][-1]["attributed_to"][1], "Reviewer B")
        self.assertEqual(j.impact(export, "a", through=7)["paths"], [])
        with self.assertRaises(j.JournalError):
            j.impact(export, "ab")

    def test_competing_relation_paths_are_not_collapsed_to_one_writer(self):
        export = fixture([contribution("a"), contribution("b"), relation("ab", "a", "b"),
                          relation("ab-two", "a", "b", attributed_to="Another declarant", supersedes="ab"),
                          assessment("question", "ab", "disputed")])
        self.assertEqual([p["relation_ids"] for p in j.impact(export, "a")["paths"]], [["ab"], ["ab-two"]])
        self.assertEqual([p["relation_ids"] for p in j.impact(export, "a", 3)["paths"]], [["ab"]])

    def test_corrupt_partial_or_noncanonical_store_never_appends(self):
        valid = (self.store / "journal.jsonl").read_bytes()
        for raw in [valid[:-1], valid + b"\n", valid + b'{"incomplete":', b"{}\n", b" " + valid]:
            (self.store / "journal.jsonl").write_bytes(raw)
            with self.subTest(raw=raw), self.assertRaises(j.JournalError):
                j.export_store(self.store)
            with self.assertRaises(j.JournalError):
                j.append_records(self.store, [contribution("a")])
            self.assertEqual((self.store / "journal.jsonl").read_bytes(), raw)

    def test_time_sequence_and_export_schema_refusals(self):
        export = fixture([contribution("a")])
        for field, value in [("sequence", True), ("sequence", 2), ("recorded_at", "2025-12-31T00:00:00.000000Z"),
                             ("recorded_at", "2026-01-01T00:00:00Z"), ("extra", "unrecognized")]:
            bad = copy.deepcopy(export)
            bad["entries"][0][field] = value
            bad["entries"][0]["sha256"] = j._hash({k: v for k, v in bad["entries"][0].items() if k != "sha256"})
            with self.subTest(field=field), self.assertRaises(j.JournalError):
                j.validate_export(bad)
        for collection_id in [COLLECTION.upper(), "not-a-uuid", None, 1]:
            bad = copy.deepcopy(export)
            bad["header"]["collection_id"] = collection_id
            with self.assertRaises(j.JournalError):
                j.validate_export(bad)
        before = (self.store / "journal.jsonl").read_bytes()
        with patch.object(j, "_now", return_value=STAMP), self.assertRaises(j.JournalError):
            j.append_records(self.store, [contribution("a")])
        self.assertEqual((self.store / "journal.jsonl").read_bytes(), before)

    def test_size_entry_and_impact_bounds_refuse_whole_result(self):
        with self.assertRaises(j.JournalError):
            j.make_export([contribution("a" + str(n)) for n in range(1001)])
        with self.assertRaisesRegex(j.JournalError, "8 MiB"):
            j.make_export([contribution("a" + str(n), summary="x" * 8192) for n in range(1000)])
        export = fixture([contribution("a"), contribution("b"), contribution("c"),
                          relation("ab", "a", "b"), relation("bc", "b", "c")])
        with patch.object(j, "MAX_IMPACT_PATHS", 1), self.assertRaises(j.JournalError):
            j.impact(export, "a")
        with patch.object(j, "MAX_IMPACT_DEPTH", 1), self.assertRaises(j.JournalError):
            j.impact(export, "a")

    def test_concurrent_process_appends_have_no_lost_rows(self):
        processes = []
        for n in range(10):
            source = self.parent / ("record-" + str(n) + ".json")
            source.write_bytes(j.canonical(contribution("parallel-" + str(n))))
            processes.append(subprocess.Popen([sys.executable, str(Path(j.__file__).resolve()), "append", str(self.store), str(source)],
                                              stdout=subprocess.PIPE, stderr=subprocess.PIPE))
        for process in processes:
            stdout, stderr = process.communicate(timeout=20)
            self.assertEqual(process.returncode, 0, stderr.decode())
            self.assertTrue(stdout)
        result = j.export_store(self.store)
        self.assertEqual(len(result["entries"]), 10)
        self.assertEqual({row["record"]["id"] for row in result["entries"]}, {"parallel-" + str(n) for n in range(10)})

    def test_write_and_replace_failure_preserve_old_bytes(self):
        before = (self.store / "journal.jsonl").read_bytes()

        def fail_write(fd, raw):
            os.write(fd, raw[:10])
            raise OSError("injected write failure")

        for target, replacement in [("_write_all", fail_write), ("os.replace", None)]:
            context = patch.object(j, target, replacement) if replacement else patch.object(j.os, "replace", side_effect=OSError("injected rename failure"))
            with context, self.assertRaises(j.JournalError):
                j.append_records(self.store, [contribution("a")])
            self.assertEqual((self.store / "journal.jsonl").read_bytes(), before)
            self.assertEqual(set(p.name for p in self.store.iterdir()), {".lock", "journal.jsonl"})

    def test_symlinks_hardlinks_fifos_and_existing_outputs_refused(self):
        link = self.parent / "linked-store"
        link.symlink_to(self.store, target_is_directory=True)
        with self.assertRaises(j.JournalError):
            j.export_store(link)
        original = self.store / "journal.jsonl"
        relocated = self.parent / "saved.jsonl"
        original.rename(relocated)
        original.symlink_to(relocated)
        with self.assertRaises(j.JournalError):
            j.export_store(self.store)
        original.unlink()
        os.link(relocated, original)
        with self.assertRaises(j.JournalError):
            j.export_store(self.store)
        original.unlink()
        relocated.rename(original)
        fifo = self.parent / "fifo"
        os.mkfifo(fifo)
        check = subprocess.run([sys.executable, str(Path(j.__file__).resolve()), "append", str(self.store), str(fifo)],
                               capture_output=True, timeout=5)
        self.assertEqual(check.returncode, 1)
        output = self.parent / "existing.json"
        output.write_bytes(b"must survive")
        with self.assertRaises(j.JournalError):
            j._write_output(output, j.export_store(self.store))
        self.assertEqual(output.read_bytes(), b"must survive")
        with self.assertRaises(j.JournalError):
            j.create_store(self.store)

    def test_missing_or_symlinked_lock_is_not_recreated(self):
        lock = self.store / ".lock"
        lock.unlink()
        with self.assertRaises(j.JournalError):
            j.append_records(self.store, [contribution("a")])
        self.assertFalse(lock.exists())
        elsewhere = self.parent / "lock-target"
        elsewhere.write_bytes(b"")
        lock.symlink_to(elsewhere)
        with self.assertRaises(j.JournalError):
            j.export_store(self.store)

    def test_exclusive_temporary_collision_preserves_existing_file(self):
        collision = self.store / ".journal-fixed.tmp"
        collision.write_bytes(b"existing unrelated bytes")
        before = (self.store / "journal.jsonl").read_bytes()
        with patch.object(j.uuid, "uuid4") as unique:
            unique.return_value.hex = "fixed"
            with self.assertRaises(j.JournalError):
                j.append_records(self.store, [contribution("a")])
        self.assertEqual(collision.read_bytes(), b"existing unrelated bytes")
        self.assertEqual((self.store / "journal.jsonl").read_bytes(), before)

    def test_after_replace_sync_failure_is_observable_not_a_silent_retry(self):
        before = (self.store / "journal.jsonl").read_bytes()
        real_fsync = os.fsync

        def fail_directory_sync(fd):
            if stat.S_ISDIR(os.fstat(fd).st_mode):
                raise OSError("injected directory sync failure after rename")
            return real_fsync(fd)

        with patch.object(j.os, "fsync", side_effect=fail_directory_sync), self.assertRaises(j.JournalError):
            j.append_records(self.store, [contribution("a")])
        # The caller must inspect: rename is atomic, but persistence was not acknowledged.
        result = j.export_store(self.store)
        self.assertEqual([row["record"]["id"] for row in result["entries"]], ["a"])
        self.assertTrue((self.store / "journal.jsonl").read_bytes().startswith(before))
        with self.assertRaises(j.JournalError):
            j.append_records(self.store, [contribution("a")])

    def test_cli_end_to_end_and_no_overwrite(self):
        script = str(Path(j.__file__).resolve())
        record_path = self.parent / "records.json"
        record_path.write_bytes(j.canonical([contribution("a"), contribution("b"), relation("ab", "a", "b")]))
        for args in [["append", str(self.store), str(record_path)], ["inspect", str(self.store), "--through", "1"],
                     ["impact", str(self.store), "a", "--through", "3"]]:
            result = subprocess.run([sys.executable, script, *args], capture_output=True, timeout=10)
            self.assertEqual(result.returncode, 0, result.stderr.decode())
            json.loads(result.stdout)
        output = self.parent / "out.json"
        args = [sys.executable, script, "export", str(self.store), "--output", str(output)]
        self.assertEqual(subprocess.run(args, capture_output=True, timeout=10).returncode, 0)
        before = output.read_bytes()
        self.assertEqual(subprocess.run(args, capture_output=True, timeout=10).returncode, 1)
        self.assertEqual(output.read_bytes(), before)

    def test_published_fictional_fixture_python_browser_contract(self):
        path = Path(j.__file__).resolve().parents[2] / "dashboard/public/research/review/fictional-journal.v1.json"
        export = j.load_export(path)
        self.assertEqual(len(export["entries"]), 27)
        self.assertEqual(export["entries"][-1]["sha256"], "ba12850f11038e4d017aab44a08698384237a66b0ec621110625f63411fe31d6")
        early = j.select_snapshot(export, 16)
        self.assertEqual(len(early["entries"]), 16)
        late_id = export["entries"][16]["record"]["id"]
        self.assertNotIn(late_id, [entry["record"]["id"] for entry in early["entries"]])


if __name__ == "__main__":
    unittest.main()

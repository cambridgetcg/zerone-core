"""Local fork/branch tests. No signing, network or independent-review claims."""

import contextlib
import copy
import hashlib
import io
import json
import os
from pathlib import Path
import stat
import tempfile
import unittest
from unittest.mock import patch

import journal as j


STAMP = "2026-01-01T00:00:00.000000Z"
LATER = "2026-01-02T00:00:00.000000Z"
COLLECTION = "12345678-1234-4234-8234-123456789abc"


def contribution(rid, **updates):
    result = {"id": rid, "kind": "contribution", "title": "Fictional " + rid,
              "summary": "Scoped local fixture, not independent review.",
              "attributed_to": "Internal fixture", "occurred_on": None,
              "evidence": [], "contribution_type": "claim", "doi": None}
    result.update(updates)
    return result


def fixture(records, **kwargs):
    return j.make_export(records, collection_id=COLLECTION, created_at=STAMP, **kwargs)


def sha(raw):
    return hashlib.sha256(raw).hexdigest()


class BranchTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.parent = Path(self.temporary.name).resolve()
        self.source = self.parent / "source.json"
        self.store = self.parent / "fork"

    def tearDown(self):
        self.temporary.cleanup()

    def write(self, export, pretty=False):
        raw = ((json.dumps(export, ensure_ascii=False, indent=2) + "\n").encode()
               if pretty else j.canonical(export) + b"\n")
        self.source.write_bytes(raw)
        return raw

    def test_fork_preserves_raw_source_history_and_append_prefix(self):
        source = fixture([contribution("a", title="Unicode Ω"), contribution("b")])
        raw = self.write(source, pretty=True)
        with patch.object(j, "_now", side_effect=AssertionError("fork must not retimestamp")), \
                patch.object(j, "_read_fd", wraps=j._read_fd) as read:
            receipt = j.fork_store(self.source, self.store, sha(raw))
            self.assertEqual(read.call_count, 1)
        self.assertEqual(receipt, {"schema": "zerone-research-fork/v1", "collection_id": COLLECTION,
                                  "entry_count": 2, "head_sha256": source["entries"][-1]["sha256"],
                                  "source_sha256": sha(raw),
                                  "scope": "Local history copy; no authentication or branch authority."})
        self.assertEqual(self.source.read_bytes(), raw)
        self.assertEqual((self.store / "fork-source.json").read_bytes(), raw)
        self.assertEqual(j.export_store(self.store), source)
        before = (self.store / "journal.jsonl").read_bytes()
        self.assertEqual(before, j._journal_bytes(source))
        self.assertEqual(set(p.name for p in self.store.iterdir()), {".lock", "fork-source.json", "journal.jsonl"})
        self.assertEqual(stat.S_IMODE(self.store.stat().st_mode), 0o700)
        for p in self.store.iterdir():
            self.assertEqual(stat.S_IMODE(p.stat().st_mode), 0o600)
        review = contribution("review", contribution_type="review", evidence=[
            {"label": "Exact reviewed row", "url": None, "sha256": source["entries"][0]["sha256"]}])
        with patch.object(j, "_now", return_value=LATER):
            after = j.append_records(self.store, [review])
        self.assertTrue((self.store / "journal.jsonl").read_bytes().startswith(before))
        self.assertEqual(after["entries"][:2], source["entries"])
        self.assertEqual(after["entries"][2]["recorded_at"], LATER)
        self.assertEqual((self.store / "fork-source.json").read_bytes(), raw)
        self.assertEqual(j.compare_exports(source, after)["relationship"], "left-prefix")

    def test_hash_and_full_validation_happen_before_any_output(self):
        good = fixture([contribution("a")])
        raw = self.write(good)
        for expected in [None, "", "A" * 64, sha(raw) + "\n", "f" * 64]:
            with self.subTest(expected=expected), self.assertRaises(j.JournalError):
                j.fork_store(self.source, self.store, expected)
            self.assertFalse(self.store.exists())
        bad = copy.deepcopy(good)
        bad["entries"].append({"unvalidated": "malformed suffix"})
        raw = self.write(bad)
        with self.assertRaises(j.JournalError):
            j.fork_store(self.source, self.store, sha(raw))
        self.assertFalse(self.store.exists())
        for raw in [b'{"schema":1,"schema":2}', b'{} {}', b'\xff', b"x" * (j.MAX_BYTES + 1)]:
            self.source.write_bytes(raw)
            with self.subTest(size=len(raw)), self.assertRaises(j.JournalError):
                j.fork_store(self.source, self.store, sha(raw))
            self.assertFalse(self.store.exists())

    def test_existing_destination_is_never_modified(self):
        raw = self.write(fixture([contribution("a")]))
        j.create_store(self.store)
        before = {p.name: p.read_bytes() for p in self.store.iterdir()}
        with self.assertRaises(j.JournalError):
            j.fork_store(self.source, self.store, sha(raw))
        self.assertEqual({p.name: p.read_bytes() for p in self.store.iterdir()}, before)
        other = self.parent / "existing-file"
        other.write_bytes(b"keep")
        with self.assertRaises(j.JournalError):
            j.fork_store(self.source, other, sha(raw))
        self.assertEqual(other.read_bytes(), b"keep")

    def test_symlink_hardlink_and_special_source_refused_without_hang(self):
        raw = self.write(fixture([]))
        link = self.parent / "link.json"
        link.symlink_to(self.source)
        with self.assertRaises(j.JournalError):
            j.fork_store(link, self.store, sha(raw))
        self.assertFalse(self.store.exists())
        parent_link = self.parent / "linked-parent"
        parent_link.symlink_to(self.parent, target_is_directory=True)
        with self.assertRaises(j.JournalError):
            j.fork_store(parent_link / self.source.name, self.store, sha(raw))
        hard = self.parent / "hard.json"
        os.link(self.source, hard)
        for p in [self.source, hard]:
            with self.assertRaises(j.JournalError):
                j.fork_store(p, self.store, sha(raw))
        fifo = self.parent / "fifo"
        os.mkfifo(fifo)
        for p in [fifo, self.parent, self.parent / "missing"]:
            with self.assertRaises(j.JournalError):
                j.fork_store(p, self.store, sha(raw))
        self.assertFalse(self.store.exists())

    def test_destination_symlink_and_linked_parent_refused(self):
        raw = self.write(fixture([]))
        target = self.parent / "target"
        target.mkdir()
        self.store.symlink_to(target, target_is_directory=True)
        with self.assertRaises(j.JournalError):
            j.fork_store(self.source, self.store, sha(raw))
        with self.assertRaises(j.JournalError):
            j.fork_store(self.source, self.store / "child", sha(raw))
        self.assertEqual(list(target.iterdir()), [])

    def test_partial_fresh_store_retained_on_io_failure(self):
        raw = self.write(fixture([]))
        original = j._new_file
        def fail_journal(directory, name, data):
            if name == "journal.jsonl":
                raise OSError("injected journal write failure")
            original(directory, name, data)
        with patch.object(j, "_new_file", side_effect=fail_journal), self.assertRaises(j.JournalError):
            j.fork_store(self.source, self.store, sha(raw))
        self.assertEqual((self.store / "fork-source.json").read_bytes(), raw)
        with self.assertRaises(j.JournalError):
            j.fork_store(self.source, self.store, sha(raw))

    def test_comparison_exact_five_relationships_and_empty_heads(self):
        empty = fixture([])
        one = fixture([contribution("a")])
        two = fixture([contribution("a"), contribution("b")])
        other = fixture([contribution("a"), contribution("c")])
        root = j.make_export([], collection_id=COLLECTION, created_at=LATER)
        header_hash = sha(j.canonical(empty["header"]))
        cases = [(empty, empty, "same-history", 0, header_hash),
                 (one, copy.deepcopy(one), "same-history", 1, one["entries"][0]["sha256"]),
                 (empty, one, "left-prefix", 0, header_hash),
                 (one, empty, "right-prefix", 0, header_hash),
                 (one, two, "left-prefix", 1, one["entries"][0]["sha256"]),
                 (two, one, "right-prefix", 1, one["entries"][0]["sha256"]),
                 (two, other, "diverged", 1, one["entries"][0]["sha256"]),
                 (one, fixture([contribution("different")]), "diverged", 0, header_hash),
                 (one, root, "different-root", None, None)]
        for left, right, relationship, count, head in cases:
            with self.subTest(relationship=relationship, count=count):
                actual = j.compare_exports(left, right)
                self.assertEqual(set(actual), {"schema", "relationship", "left", "right", "common_prefix"})
                self.assertEqual(actual["schema"], "zerone-research-comparison/v1")
                self.assertEqual(actual["relationship"], relationship)
                self.assertEqual(actual["common_prefix"], None if count is None else {"entry_count": count, "head_sha256": head})
                for name, export in [("left", left), ("right", right)]:
                    self.assertEqual(actual[name], {"collection_id": COLLECTION, "entry_count": len(export["entries"]),
                                                   "head_sha256": export["entries"][-1]["sha256"] if export["entries"] else sha(j.canonical(export["header"]))})

    def test_same_uuid_is_not_root_authority_and_full_rows_include_time(self):
        a = fixture([contribution("a")])
        different_time = fixture([contribution("a")], recorded_at=[LATER])
        actual = j.compare_exports(a, different_time)
        self.assertEqual(actual["relationship"], "diverged")
        self.assertEqual(actual["common_prefix"]["entry_count"], 0)
        other_root = j.make_export([contribution("a")], collection_id=COLLECTION, created_at=LATER)
        self.assertEqual(j.compare_exports(a, other_root)["relationship"], "different-root")
        other_uuid = j.make_export([], collection_id="00000000-0000-0000-0000-000000000000", created_at=STAMP)
        self.assertEqual(j.compare_exports(fixture([]), other_uuid)["relationship"], "different-root")

    def test_malformed_full_suffix_refused_before_prefix_or_root_result(self):
        short = fixture([contribution("a")])
        bad = fixture([contribution("a"), contribution("b")])
        bad["entries"][-1]["record"]["summary"] = "Tampered later row"
        for left, right in [(short, bad), (bad, short), (fixture([]), bad)]:
            with self.assertRaises(j.JournalError):
                j.compare_exports(left, right)
        bad["header"]["created_at"] = LATER
        with self.assertRaises(j.JournalError):
            j.compare_exports(short, bad)

    def test_whitespace_copy_and_cli_store_export_comparison(self):
        export = fixture([contribution("a")])
        raw = self.write(export, pretty=True)
        out, err = io.StringIO(), io.StringIO()
        with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
            self.assertEqual(j.main(["fork", str(self.source), str(self.store), "--expected-sha256", sha(raw)]), 0)
        self.assertEqual(json.loads(out.getvalue())["source_sha256"], sha(raw))
        canonical_path = self.parent / "canonical.json"
        j._write_output(canonical_path, j.export_store(self.store))
        self.assertNotEqual(sha(raw), sha(canonical_path.read_bytes()))
        for left, right in [(self.source, canonical_path), (self.source, self.store), (self.store, self.source)]:
            out = io.StringIO()
            with contextlib.redirect_stdout(out):
                self.assertEqual(j.main(["compare", str(left), str(right)]), 0)
            self.assertEqual(json.loads(out.getvalue())["relationship"], "same-history")
        with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(err):
            self.assertEqual(j.main(["fork", str(self.source), str(self.store), "--expected-sha256", "f" * 64]), 1)
        self.assertNotIn("Traceback", err.getvalue())

    def test_actual_published_81_entry_source(self):
        source = Path(__file__).resolve().parents[2] / "dashboard/public/research/checkpoints/2026-09-14-demimetric-v1/journal.json"
        raw = source.read_bytes()
        self.assertEqual(sha(raw), "86c8163599c0098aa43b4f3599a81b02a03557553fc595b67b026fdfe108704e")
        receipt = j.fork_store(source, self.store, sha(raw))
        self.assertEqual(receipt["entry_count"], 81)
        self.assertEqual(receipt["head_sha256"], "23baf5b0a2e7f4a48fe2b86775ae2ef92e607f2d793270ff10219e61d2563847")
        copied = j.export_store(self.store)
        source_export = j.parse_json(raw)
        self.assertEqual(j.compare_exports(source_export, copied)["relationship"], "same-history")
        self.assertEqual((self.store / "fork-source.json").read_bytes(), raw)
        before = (self.store / "journal.jsonl").read_bytes()
        with patch.object(j, "_now", return_value="2026-09-15T00:00:00.000000Z"):
            extended = j.append_records(self.store, [contribution("internal-branch-copy-check", contribution_type="review")])
        self.assertEqual(j.compare_exports(source_export, extended)["common_prefix"],
                         {"entry_count": 81, "head_sha256": receipt["head_sha256"]})
        self.assertTrue((self.store / "journal.jsonl").read_bytes().startswith(before))


if __name__ == "__main__":
    unittest.main()

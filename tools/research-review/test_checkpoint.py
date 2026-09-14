"""A saved external commitment detects valid local rewrites and omissions."""

import copy
import json
from pathlib import Path
import tempfile
import unittest

import checkpoint as c
import journal as j
from test_journal import contribution, fixture


class CheckpointTests(unittest.TestCase):
    def setUp(self):
        self.export = fixture([contribution("first"), contribution("second")])
        self.raw = j.canonical(self.export) + b"\n"
        self.checkpoint = c.prepare(self.raw, "zerone-dev-1")

    def check(self, raw=None, checkpoint=None, memo=None, chain="zerone-dev-1"):
        return c.verify(self.raw if raw is None else raw,
                        self.checkpoint if checkpoint is None else checkpoint,
                        self.checkpoint["memo"] if memo is None else memo, chain)

    def test_exact_roundtrip_and_live_memo_bound(self):
        self.assertTrue(self.check()["export_matches_memo"])
        self.assertLessEqual(len(self.checkpoint["memo"].encode()), 256)
        self.assertEqual(self.checkpoint["entry_count"], 2)

    def test_valid_rewrite_and_valid_short_prefix_are_refused(self):
        rewritten = fixture([contribution("first", title="Replacement"), contribution("second")])
        shorter = copy.deepcopy(self.export)
        shorter["entries"] = shorter["entries"][:1]
        for export in (rewritten, shorter):
            j.validate_export(export)
            with self.subTest(export=export), self.assertRaises(j.JournalError):
                self.check(raw=j.canonical(export) + b"\n")

    def test_even_equivalent_json_formatting_changes_exact_file_commitment(self):
        changed = json.dumps(self.export, indent=2).encode()
        self.assertEqual(j.parse_json(changed), j.parse_json(self.raw))
        with self.assertRaises(j.JournalError):
            self.check(raw=changed)

    def test_wrong_context_and_inconsistent_metadata_are_refused(self):
        with self.assertRaises(j.JournalError):
            self.check(chain="zerone-1")
        for key, value in (("schema", "future"), ("collection_id", "another"),
                           ("entry_count", True), ("entry_count", 2.0),
                           ("head_sha256", "0" * 64), ("export_sha256", "0" * 64),
                           ("extra", "unrecognized")):
            checkpoint = dict(self.checkpoint, **{key: value})
            with self.subTest(key=key, value=value), self.assertRaises(j.JournalError):
                self.check(checkpoint=checkpoint)
        for memo in ("", self.checkpoint["memo"] + " ", self.checkpoint["memo"].upper()):
            with self.subTest(memo=memo), self.assertRaises(j.JournalError):
                self.check(memo=memo)

    def test_invalid_journal_and_duplicate_json_refused_before_commitment(self):
        changed = copy.deepcopy(self.export)
        changed["entries"][0]["record"]["title"] = "unhashed edit"
        for raw in (j.canonical(changed), b'{"schema":1,"schema":2}',
                    j.canonical(fixture([])), b" " * (j.MAX_BYTES + 1)):
            with self.subTest(length=len(raw)), self.assertRaises(j.JournalError):
                c.prepare(raw, "zerone-dev-1")

    def test_cli_fresh_output_and_no_symlink_export(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary).resolve()
            source, output = root / "export.json", root / "checkpoint.json"
            source.write_bytes(self.raw)
            self.assertEqual(c.main(["prepare", str(source), "--chain-id", "zerone-dev-1",
                                     "--output", str(output)]), 0)
            before = output.read_bytes()
            self.assertEqual(c.main(["prepare", str(source), "--chain-id", "zerone-dev-1",
                                     "--output", str(output)]), 1)
            self.assertEqual(before, output.read_bytes())
            link = root / "link.json"
            link.symlink_to(source)
            with self.assertRaises(j.JournalError):
                c.read_export_bytes(link)


if __name__ == "__main__":
    unittest.main()

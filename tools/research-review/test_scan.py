"""Offline regressions for scientific notice attribution and lookup boundaries."""
import json
from pathlib import Path
import tempfile
import unittest
from unittest import mock

import journal
import scan

ORIGINAL = "10.1234/example"
NOTICE = "10.1234/notice"


def metadata(doi=ORIGINAL, **fields):
    return scan.canonical({"status": "ok", "message-type": "work", "message": {
        "DOI": doi, "title": ["Fictional mathematical example"], **fields}})


def update(doi=NOTICE, type_="retraction", source="publisher", date=None):
    return {"DOI": doi, "type": type_, "source": source,
            "updated": {"date-parts": [date or [2025, 1, 2]]}}


class ScanTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.store = Path(self.temp.name).resolve() / "journal"
        journal.create_store(self.store)

    def run_scan(self, raw, dois=None):
        return scan.scan(self.store, dois or [ORIGINAL], fetch=lambda _: raw, pause=lambda _: None)

    def records(self, kind=None):
        records = [row["record"] for row in journal.export_store(self.store)["entries"]]
        return records if kind is None else [record for record in records if record["kind"] == kind]

    def test_direction_from_original_and_notice_deduplicates(self):
        self.run_scan(metadata(**{"updated-by": [update()]}))
        self.run_scan(metadata(NOTICE, **{"update-to": [update(ORIGINAL)]}), [NOTICE])
        concerns = self.records("concern")
        self.assertEqual(len(concerns), 1)
        self.assertEqual(concerns[0]["target_id"], scan.identifier("crossref-source", ORIGINAL))
        self.assertNotEqual(concerns[0]["target_id"], scan.identifier("crossref-source", NOTICE))
        self.assertEqual(len(self.records("contribution")), 4)  # two sources, two observations

    def test_two_providers_are_one_notice_not_independent_votes(self):
        result = self.run_scan(metadata(**{"updated-by": [update(), update(source="retraction-watch")]}))
        self.assertEqual(result["lookups"][0]["distinct_notice_leads"], 1)
        self.assertEqual(len(self.records("concern")), 1)
        concern = self.records("concern")[0]
        self.assertIn("publisher, retraction-watch", concern["provider"])
        self.assertEqual(concern["category"], "reported-concern")

    def test_refresh_preserves_concerns_and_each_lookup(self):
        raw = metadata(**{"updated-by": [update()]})
        self.run_scan(raw)
        old = journal.export_store(self.store)
        self.run_scan(raw)
        self.run_scan(metadata())
        new = journal.export_store(self.store)
        self.assertEqual(new["entries"][:len(old["entries"])], old["entries"])
        self.assertEqual(len(self.records("concern")), 1)
        self.assertEqual(len([r for r in self.records() if r.get("contribution_type") == "review"]), 3)
        self.assertIn("not a clean bill of health", self.records()[-1]["summary"])

    def test_reinstatement_and_conflicting_types_do_not_clear_history(self):
        raw = metadata(**{"updated-by": [update(), update(type_="reinstatement") ]})
        self.run_scan(raw)
        concerns = self.records("concern")
        self.assertEqual({r["notice_type"] for r in concerns}, {"retraction", "reinstatement", None})
        self.assertEqual(len(self.records("assessment")), 0)
        self.assertIn("metadata-conflict", {r["category"] for r in concerns})

    def test_missing_dates_and_future_labels_remain_unknown(self):
        raw = metadata(published={"date-parts": [[2020]]}, **{"updated-by": [
            update(type_="future_editorial_event", date=[2025]) ]})
        self.run_scan(raw)
        self.assertIsNone(self.records("concern")[0]["occurred_on"])
        self.assertIsNone(self.records("contribution")[0]["occurred_on"])
        self.assertEqual(self.records("concern")[0]["notice_type"], "future_editorial_event")

    def test_huge_provider_date_is_unknown(self):
        self.run_scan(metadata(**{"updated-by": [update(date=[10**40, 1, 1])]}))
        self.assertIsNone(self.records("concern")[0]["occurred_on"])

    def test_later_date_or_attribution_change_is_visible_without_rewriting(self):
        self.run_scan(metadata(**{"updated-by": [update()]}))
        first = self.records("concern")[0]
        changed = metadata(**{"updated-by": [update(date=[2025, 2, 3], source="retraction-watch")]})
        self.run_scan(changed)
        self.run_scan(changed)
        self.assertEqual(self.records("concern")[0], first)
        self.assertEqual(len(self.records("concern")), 2)
        self.assertEqual(self.records("concern")[1]["category"], "metadata-conflict")

    def test_notice_id_occupied_by_unrelated_record_is_refused(self):
        forged = scan._record(scan.identifier("crossref-notice", ORIGINAL, NOTICE, "retraction"),
            "contribution", "Other record", "This is not the provider notice.", None, [],
            contribution_type="question", doi=None)
        journal.append_records(self.store, [forged])
        with self.assertRaises(scan.ScanError):
            self.run_scan(metadata(**{"updated-by": [update()]}))
        self.assertEqual(len(self.records()), 1)

    def test_title_or_references_are_not_editorial_updates(self):
        raw = metadata(title=["A retraction in topology"], reference=[{"DOI": NOTICE}])
        result = self.run_scan(raw)
        self.assertEqual(result["lookups"][0]["distinct_notice_leads"], 0)
        self.assertEqual(self.records("concern"), [])

    def test_in_situ_same_doi_update_is_a_notice(self):
        self.run_scan(metadata(**{"updated-by": [update(ORIGINAL, type_="correction")]}))
        self.assertEqual(len(self.records("concern")), 1)
        self.assertEqual(self.records("concern")[0]["target_id"], scan.identifier("crossref-source", ORIGINAL))

    def test_malformed_label_is_retained_as_coverage_warning(self):
        result = self.run_scan(metadata(**{"updated-by": [update(type_="x" * 101)]}))
        self.assertEqual(len(result["lookups"][0]["warnings"]), 1)
        self.assertEqual(self.records("concern"), [])

    def test_failure_is_recorded_as_unknown_and_stops_on_rate_limit(self):
        def fail(_):
            raise scan.ScanError("Provider HTTP 429")
        result = scan.scan(self.store, [ORIGINAL, NOTICE], fetch=fail, pause=lambda _: None)
        self.assertEqual(result["requested"], 2)
        self.assertEqual(result["completed"], 1)
        self.assertEqual(result["lookups"][0]["status"], "lookup-failed")
        self.assertNotIn("distinct_notice_leads", result["lookups"][0])
        self.assertIn("coverage is unknown", self.records()[0]["summary"])

    def test_malformed_json_doi_mismatch_and_bad_list_preserve_failure(self):
        for raw in (b'{"bad":', metadata(NOTICE), metadata(**{"updated-by": "bad"}),
                    b'{"status":"ok","status":"no"}'):
            with self.subTest(raw=raw):
                result = self.run_scan(raw)
                self.assertEqual(result["lookups"][0]["status"], "lookup-failed")
                self.assertTrue((self.store / "artifacts" / (scan.digest(raw) + ".json")).exists())
        self.assertEqual(self.records("concern"), [])

    def test_malformed_links_are_observations_not_silent_empty(self):
        result = self.run_scan(metadata(**{"updated-by": [None, {"DOI": "bad"}]}))
        self.assertEqual(len(result["lookups"][0]["warnings"]), 2)
        self.assertIn("2 malformed-link", self.records()[-1]["summary"])

    def test_raw_artifact_and_receipt_hashes_bind_exact_bytes(self):
        raw = metadata(**{"updated-by": [update()]})
        result = self.run_scan(raw)["lookups"][0]
        artifact = self.store / "artifacts" / (result["metadata_sha256"] + ".json")
        self.assertEqual(artifact.read_bytes(), raw)
        receipt = (self.store / "artifacts" / (result["receipt_sha256"] + ".json")).read_bytes()
        self.assertEqual(scan.digest(receipt), result["receipt_sha256"])
        self.assertEqual(json.loads(receipt)["metadata_sha256"], scan.digest(raw))
        export_bytes = scan.canonical(journal.export_store(self.store))
        self.assertNotIn(str(self.store).encode(), export_bytes)
        self.assertNotIn(b'"abstract"', export_bytes)

    def test_content_addressed_file_and_directory_symlinks_refused(self):
        raw = metadata()
        outside = Path(self.temp.name) / "outside"
        outside.mkdir()
        (self.store / "artifacts").symlink_to(outside, target_is_directory=True)
        with self.assertRaises(scan.ScanError):
            self.run_scan(raw)
        self.assertEqual(list(outside.iterdir()), [])
        (self.store / "artifacts").unlink()
        (self.store / "artifacts").mkdir()
        victim = outside / "victim"
        victim.write_bytes(raw)
        (self.store / "artifacts" / (scan.digest(raw) + ".json")).symlink_to(victim)
        with self.assertRaises(OSError):
            self.run_scan(raw)
        self.assertEqual(victim.read_bytes(), raw)

    def test_doi_bounds_dedup_and_fixed_request_origin(self):
        self.assertEqual(scan.normalize_doi("https://doi.org/10.1234/EXAMPLE"), ORIGINAL)
        self.assertTrue(scan.api_url("10.1234/a?x=#secret").endswith("10.1234%2Fa%3Fx%3D%23secret"))
        for invalid in ("https://other.invalid/thing", "10.1/x", "10.1234/a\nb", "10.1234/" + "x" * 512):
            with self.assertRaises(scan.ScanError):
                self.run_scan(metadata(), [invalid])
        with self.assertRaises(scan.ScanError):
            self.run_scan(metadata(), ["10.1234/" + str(i) for i in range(26)])
        fetch = mock.Mock(return_value=metadata())
        scan.scan(self.store, [ORIGINAL, ORIGINAL.upper()], fetch=fetch, pause=lambda _: None)
        fetch.assert_called_once_with(ORIGINAL)

    def test_response_and_update_bounds(self):
        for raw in (b" " * (scan.MAX_RESPONSE + 1), metadata(**{"updated-by": [update()] * 101})):
            result = self.run_scan(raw)
            self.assertEqual(result["lookups"][0]["status"], "lookup-failed")
        self.assertEqual(self.records("concern"), [])

    def test_unrepresentable_provider_strings_and_numbers_fail_with_receipt(self):
        for extra in (b'"title":["\\ud800"]', b'"score":1e999', b'"score":' + b'9' * 5000):
            raw = b'{"status":"ok","message-type":"work","message":{"DOI":"10.1234/example",' + extra + b'}}'
            result = self.run_scan(raw)
            self.assertEqual(result["lookups"][0]["status"], "lookup-failed")
            self.assertTrue((self.store / "artifacts" / (scan.digest(raw) + ".json")).exists())

    def test_default_fetch_retains_malformed_http_response_for_receipt(self):
        response = mock.MagicMock()
        response.__enter__.return_value = response
        response.status = 200
        response.read.return_value = b"malformed JSON"
        opener = mock.Mock()
        opener.open.return_value = response
        with mock.patch.object(scan.urllib.request, "build_opener", return_value=opener):
            result = scan.scan(self.store, [ORIGINAL])
        self.assertEqual(result["lookups"][0]["status"], "lookup-failed")
        self.assertEqual((self.store / "artifacts" / (scan.digest(b"malformed JSON") + ".json")).read_bytes(),
                         b"malformed JSON")

    def test_redirects_refused(self):
        with self.assertRaises(scan.ScanError):
            scan.NoRedirect().redirect_request(None, None, 302, "Moved", {}, "http://127.0.0.1/")

    def test_store_is_checked_before_network(self):
        fetch = mock.Mock(return_value=metadata())
        (self.store / "journal.jsonl").write_text("corrupt\n")
        with self.assertRaises(journal.JournalError):
            scan.scan(self.store, [ORIGINAL], fetch=fetch)
        fetch.assert_not_called()


if __name__ == "__main__":
    unittest.main()

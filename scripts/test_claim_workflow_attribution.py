#!/usr/bin/env python3
"""Attribution encoder and quote-guard tests (plan §4.2, §4.5, §4.7); no node or network.

Every record, hash and excerpt below is synthetic test material, not a real
paper, person or research conclusion.
"""
import argparse
import copy
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

SPEC = importlib.util.spec_from_file_location("attribution_fixture", Path(__file__).with_name("test_claim_workflow.py"))
assert SPEC is not None and SPEC.loader is not None
fixture = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(fixture)
workflow = fixture.workflow
SCRIPT = Path(__file__).with_name("claim-workflow.py")

# Proposition P1 of plan §4.3 and its on-chain content hash. The Go vectors were
# produced by keeper.ComputeClaimContentHash (x/knowledge/keeper/state.go) with
# `go run` on this branch (base a4272bfa); the first equals the plan's value.
PROPOSITION = "Compound Z boils at 78.4 °C at 1 atm for a pure sample."
PROPOSITION_HASH = "f789edd890d61613208c09ebc74248b09c2f170e40c07390c74c74a7db301026"
SECOND_TEXT = "Synthetic: Jones 2021 reports compound Z boils at 81.0 °C (1 atm)."
SECOND_HASH = "7dbcc5a5e90aa14f8b23bf1d70b52cb1a08eb92714b330cc636eb0be27c8573a"
SOURCE_DIGEST = "0580a051899e6a386f8213633badf896d07f21a370a8ba915f2bf93ba054bce3"
# Synthetic Hermes record hashes. The occurrence hash keeps the plan example's
# visible prefix (the `stmt:` tag) and suffix; its middle is filler.
OCCURRENCE = "e25fa6b012331b0f8681c04b88d4ee7eacaa089d" + "0123456789abcdef0123" + "a685"
SOURCE_RECORD = "79e8088b79de7f36ae290bda69a3579109a97bb859d989a16e1c964f2212821f"
PROPOSITION_RECORD = "97b69086769686269a936b683e9b5e012adc1b743c782807aad5d540f786e24a"
EXCERPT = ("Synthetic fixture excerpt: the purified compound Z was observed to boil at 78.4 "
           "degrees Celsius under one standard atmosphere of pressure.")
EXCERPT_SHA = "0c151d503b819bc4f2a4a9802dc73153c10652e7448c9cae7619afa4a4f41f02"
EXTRACTOR = "did:zrn:synthetic-extractor-0001"

RECORD = {
    "schema": "zerone.attribution-intake/v1",
    "source_version": {"record": "sha256:" + SOURCE_RECORD, "work": "doi:10.1000/p6-smith-2019",
                       "digest": "sha256:" + SOURCE_DIGEST, "published": "2019-03-01"},
    "statement_occurrence": {"record": "sha256:" + OCCURRENCE, "locator": "p4.para2", "act": "states",
                             "voice": "author", "excerpt_sha256": "sha256:" + EXCERPT_SHA},
    "proposition": {"record": "sha256:" + PROPOSITION_RECORD, "domain": "chemistry", "text": PROPOSITION},
    "extractor": EXTRACTOR,
}

FACT_CONTENT = ("Attribution zerone.attribution/v1: doi:10.1000/p6-smith-2019 states:author "
                "zrn-content:f789edd890d61613208c09ebc74248b09c2f170e40c07390c74c74a7db301026 at "
                "ver=sha256:0580a051899e6a386f8213633badf896d07f21a370a8ba915f2bf93ba054bce3;loc=p4.para2")
TRACE = ('{"excerpt_sha256":"sha256:0c151d503b819bc4f2a4a9802dc73153c10652e7448c9cae7619afa4a4f41f02",'
         '"extractor":"did:zrn:synthetic-extractor-0001",'
         '"proposition_record":"sha256:97b69086769686269a936b683e9b5e012adc1b743c782807aad5d540f786e24a",'
         '"schema":"zerone.attribution/v1",'
         '"source_version_record":"sha256:79e8088b79de7f36ae290bda69a3579109a97bb859d989a16e1c964f2212821f",'
         '"statement_occurrence":"sha256:e25fa6b012331b0f8681c04b88d4ee7eacaa089d0123456789abcdef0123a685"}')
STRUCTURE = {
    "subject": "doi:10.1000/p6-smith-2019",
    "predicate": "states:author",
    "object": "zrn-content:" + PROPOSITION_HASH,
    "scope": "ver=sha256:" + SOURCE_DIGEST + ";loc=p4.para2",
    "temporal_scope": "2019-03-01",
    "tags": ["attr-v1", "stmt:e25fa6b012331b0f8681c04b88d4ee7eacaa089d", "zc:f789edd890d61613208c09ebc74248b09c2f170e"],
}
EXPECTED = {
    "claim_type": "CLAIM_TYPE_RELATION",
    "domain": "chemistry",
    "category": "empirical",
    "method_id": "M-HISTORICAL",
    "structure": STRUCTURE,
    "fact_content": FACT_CONTENT,
    "reasoning_trace": TRACE,
    "references": [],
    "relations": [],
}
# Eight consecutive excerpt tokens, re-cased and re-punctuated.
EIGHT = "Note: THE purified compound Z, was observed to BOIL in that study."
SEVEN = "Note: the purified compound Z was observed to melt in that study."


def record(**changes):
    value = copy.deepcopy(RECORD)
    for dotted, item in changes.items():
        target = value
        *parents, leaf = dotted.split("__")
        for parent in parents:
            target = target[parent]
        if item is KeyError:
            del target[leaf]
        else:
            target[leaf] = item
    return value


class AttributionEncoderTests(unittest.TestCase):
    def test_hermes_fixture_produces_exact_attribution_fields(self):
        self.assertEqual(workflow.attribution_claim(RECORD), EXPECTED)

    def test_fact_content_template_is_byte_exact(self):
        claim = workflow.attribution_claim(RECORD)
        self.assertEqual(claim["fact_content"].encode("utf-8"), FACT_CONTENT.encode("utf-8"))
        self.assertEqual(len(claim["fact_content"].encode("utf-8")), 243)  # plan §4.3

    def test_reasoning_trace_is_jcs_with_hashes_only(self):
        trace = workflow.attribution_claim(RECORD)["reasoning_trace"]
        self.assertEqual(trace, TRACE)
        self.assertEqual(workflow.jcs(json.loads(trace)), trace)
        self.assertEqual(sorted(json.loads(trace)), ["excerpt_sha256", "extractor", "proposition_record",
                                                     "schema", "source_version_record", "statement_occurrence"])
        self.assertNotIn(PROPOSITION, trace)

    def test_jcs_matches_rfc8785_ordering_and_string_rules(self):
        # RFC 8785 §3.2.3: properties sort by UTF-16 code units, not code points.
        value = {"€": "Euro Sign", "\r": "Carriage Return", "דּ": "Hebrew Letter Dalet With Dagesh",
                 "1": "One", "\U0001f600": "Emoji: Grinning Face", "\u0080": "Control",
                 "ö": "Latin Small Letter O With Diaeresis"}
        self.assertEqual(workflow.jcs(value),
                         '{"\\r":"Carriage Return","1":"One","\u0080":"Control","ö":"Latin Small Letter O With Diaeresis",'
                         '"€":"Euro Sign","\U0001f600":"Emoji: Grinning Face","דּ":"Hebrew Letter Dalet With Dagesh"}')
        self.assertEqual(workflow.jcs({"b": [True, None, 7], "a": "q\"\\\b\f\n\r\t\x01\x7f/"}),
                         '{"a":"q\\"\\\\\\b\\f\\n\\r\\t\\u0001\x7f/","b":[true,null,7]}')
        for bad in (1.5, {"x": float("nan")}, "\ud800", {"\udfff": "x"}, 2 ** 53, object()):
            with self.subTest(bad=repr(bad)), self.assertRaises(workflow.WorkflowError):
                workflow.jcs(bad)

    def test_content_hash_matches_go_vectors(self):
        self.assertEqual(workflow.claim_content_hash(PROPOSITION, "chemistry"), PROPOSITION_HASH)
        self.assertEqual(workflow.claim_content_hash(SECOND_TEXT, "chemistry"), SECOND_HASH)
        changed = record(proposition__text=SECOND_TEXT)
        claim = workflow.attribution_claim(changed)
        self.assertEqual(claim["structure"]["object"], "zrn-content:" + SECOND_HASH)
        self.assertEqual(claim["structure"]["tags"][2], "zc:" + SECOND_HASH[:40])

    def test_quoted_voices_and_other_work_identifiers(self):
        cases = {
            ("voice", "quoted:orcid:0000-0002-1825-0097"): ("predicate", "states:quoted:orcid:0000-0002-1825-0097"),
            ("voice", "quoted:wikidata:q42"): ("predicate", "states:quoted:wikidata:q42"),
            ("voice", "quoted:hermes:" + "ab" * 32): ("predicate", "states:quoted:hermes:" + "ab" * 32),
            ("act", "hedges"): ("predicate", "hedges:author"),
            ("work", "arxiv:2101.00001v2"): ("subject", "arxiv:2101.00001v2"),
            ("work", "isbn:9780306406157"): ("subject", "isbn:9780306406157"),
            ("work", "url-sha256:" + "cd" * 32): ("subject", "url-sha256:" + "cd" * 32),
            ("locator", "t=00:12:31"): ("scope", "ver=sha256:" + SOURCE_DIGEST + ";loc=t=00:12:31"),
            ("published", None): ("temporal_scope", ""),
        }
        for (field, value), (key, expected) in cases.items():
            parent = "source_version" if field in ("work", "published") else "statement_occurrence"
            with self.subTest(field=field, value=value):
                claim = workflow.attribution_claim(record(**{parent + "__" + field: value}))
                self.assertEqual(claim["structure"][key], expected)

    def test_uppercase_or_free_text_in_structure_fields_is_refused(self):
        bad = {
            "source_version__work": ["DOI:10.1000/p6-smith-2019", "doi:10.1000/P6-Smith-2019", "doi:10.1000/smith 2019",
                                     "https://example.org/paper", "Smith 2019", "isbn:978030640615", "arxiv:2101.00001"],
            "statement_occurrence__locator": ["p4 para2", "P4.para2", "p4;loc=p5", "page four, second paragraph", ""],
            "statement_occurrence__act": ["says", "States", "states:author"],
            "statement_occurrence__voice": ["quoted:John Smith", "quoted:hermes:" + "AB" * 32, "Author", "quoted:", "quoted:orcid:smith"],
            "source_version__digest": ["sha256:" + SOURCE_DIGEST.upper(), SOURCE_DIGEST, "sha256:abc"],
            "source_version__published": ["March 2019", "2019-02-30", "2019-3-1"],
            "statement_occurrence__record": ["sha256:" + OCCURRENCE.upper()],
            "statement_occurrence__excerpt_sha256": ["sha256:" + EXCERPT_SHA[:-1]],
            "proposition__domain": ["Chemistry", "chemistry studies", ""],
            "proposition__text": ["", 7],
            "extractor": ["Smith (extractor)", "did:zrn:", "did:ZRN:x"],
        }
        for field, values in bad.items():
            for value in values:
                with self.subTest(field=field, value=value), self.assertRaises(workflow.WorkflowError):
                    workflow.attribution_claim(record(**{field: value}))

    def test_unknown_missing_or_misversioned_intake_fields_are_refused(self):
        for changed in (record(schema="zerone.attribution-intake/v2"), record(note="free text"),
                        record(statement_occurrence__excerpt="the text itself"), record(extractor=KeyError),
                        record(source_version__digest=KeyError), record(proposition__record=KeyError)):
            with self.subTest(changed=sorted(changed)), self.assertRaises(workflow.WorkflowError):
                workflow.attribution_claim(changed)

    def test_method_and_category_defaults_and_bounded_overrides(self):
        claim = workflow.attribution_claim(RECORD, method="M-ATTRIBUTION", category="empirical")
        self.assertEqual(claim["method_id"], "M-ATTRIBUTION")
        for kwargs in ({"category": "historical"}, {"method": ""}, {"method": "m historical"}):
            with self.subTest(kwargs=kwargs), self.assertRaises(workflow.WorkflowError):
                workflow.attribution_claim(RECORD, **kwargs)


class QuoteGuardTests(unittest.TestCase):
    def test_refuses_eight_token_overlap_after_normalisation(self):
        with self.assertRaisesRegex(workflow.WorkflowError, "8 consecutive excerpt tokens"):
            workflow.quote_guard(EXCERPT, [("fact_content", EIGHT)])
        # NFKC folds the fullwidth letter and the ligature; casefold folds case.
        folded = "THE PURIFIED COMPOUND \uff3a \u2014 WAS OBSERVED, TO BOIL!"
        with self.assertRaisesRegex(workflow.WorkflowError, "8 consecutive excerpt tokens"):
            workflow.quote_guard(EXCERPT, [("review reason", folded)])
        with self.assertRaisesRegex(workflow.WorkflowError, "8 consecutive"):
            workflow.quote_guard("the \ufb01ne compound was observed to boil at last", [("x", "THE FINE COMPOUND WAS OBSERVED TO BOIL AT")])

    def test_accepts_seven_tokens_and_a_paraphrase(self):
        receipt = workflow.quote_guard(EXCERPT, [("fact_content", SEVEN), ("proposition", PROPOSITION)])
        self.assertEqual(receipt, {"version": workflow.QUOTE_GUARD_VERSION, "excerpt_sha256": "sha256:" + EXCERPT_SHA})
        self.assertEqual(workflow.QUOTE_GUARD_VERSION, "zerone.quote-guard/v1")

    def test_refuses_a_whole_short_excerpt_only(self):
        short = "Z boils near 78.4 \u2103."  # six tokens: z boils near 78 4 c
        self.assertEqual(workflow.guard_tokens(short), ["z", "boils", "near", "78", "4", "c"])
        with self.assertRaisesRegex(workflow.WorkflowError, "whole excerpt"):
            workflow.quote_guard(short, [("challenge reason", "As reported, z BOILS near 78.4°C in air.")])
        workflow.quote_guard(short, [("challenge reason", "As reported, Z boils near 78 degrees.")])

    def test_excerpt_digest_is_nfc_and_excerpt_must_have_tokens(self):
        decomposed = "cafe\u0301 excerpt"
        self.assertEqual(workflow.excerpt_digest(decomposed),
                         "sha256:" + hashlib.sha256("caf\xe9 excerpt".encode()).hexdigest())
        for empty in ("", " \n\t", "— ... !"):
            with self.subTest(empty=empty), self.assertRaises(workflow.WorkflowError):
                workflow.quote_guard(empty, [("x", "anything")])

    def test_refusal_message_never_repeats_excerpt_text(self):
        with self.assertRaises(workflow.WorkflowError) as error:
            workflow.quote_guard(EXCERPT, [("fact_content", EIGHT)])
        message = str(error.exception)
        for word in ("purified", "observed", "boil", "Synthetic fixture"):
            self.assertNotIn(word, message.lower() if word.islower() else message)
        self.assertIn("fact_content", message)


class AttributionWorkflowTests(unittest.TestCase):
    def setUp(self):
        case = fixture.WorkflowTests()
        case.setUp()
        self.addCleanup(case.doCleanups)
        self.case = case
        self.w, self.calls, self.observation, self.home = case.w, case.calls, case.observation, case.home
        self.w.check_node.return_value = 99
        values = {("knowledge", "params"): {"params": {"min_review_fee": "100000"}},
                  ("alignment", "global-pacing"): {"creation_multiplier_bps": "1000000"}}
        self.w.query = mock.Mock(side_effect=lambda *args, **kwargs: values[args])
        self.excerpt_path = Path(self.home) / "excerpt.txt"
        self.excerpt_path.write_text(EXCERPT, encoding="utf-8")

    def guarded_claim(self):
        claim = workflow.attribution_claim(RECORD)
        guard = workflow.attribution_guard(claim, RECORD, workflow.read_excerpt(self.excerpt_path))
        return claim, guard

    def generated(self):
        rows = [call for call in self.calls if "--generate-only" in call]
        self.assertEqual(len(rows), 1)
        return list(rows[0])

    def assert_excerpt_absent(self, *outputs):
        texts = [json.dumps([str(item) for item in call], ensure_ascii=False) for call in self.calls]
        texts += [json.dumps(value, ensure_ascii=False) if not isinstance(value, str) else value for value in outputs]
        for path in Path(self.home).rglob("*"):
            if path.is_file() and path != self.excerpt_path:
                texts.append(path.read_text(encoding="utf-8", errors="replace"))
        for text in texts:
            self.assertNotIn(EXCERPT, text)
            self.assertNotIn("purified compound", text.lower())

    def test_broadcast_uses_submit_path_with_structure_flags_and_never_canonical(self):
        claim, guard = self.guarded_claim()
        receipt = self.w.attribute(claim, guard)
        self.assertEqual(receipt["status"], "committed")
        self.assertEqual(receipt["claim_id"], "claim-1")
        self.assertEqual(receipt["quote_guard"], {"version": "zerone.quote-guard/v1", "excerpt_sha256": "sha256:" + EXCERPT_SHA})
        arguments = self.generated()
        self.assertEqual(arguments[:arguments.index("--from")], [
            "tx", "knowledge", "submit-claim", FACT_CONTENT, "chemistry", "empirical", "100000",
            "--method-id", "M-HISTORICAL", "--reasoning-trace", TRACE, "--claim-type", "relation",
            "--subject", STRUCTURE["subject"], "--predicate", STRUCTURE["predicate"], "--object", STRUCTURE["object"],
            "--scope", STRUCTURE["scope"], "--temporal-scope", "2019-03-01", "--tags", ",".join(STRUCTURE["tags"])])
        self.assertFalse(any("--canonical" in call for call in self.calls))
        attempt = self.w.records("attempts")[0][1]
        self.assertEqual(attempt["action"], "submit")
        self.assertEqual(attempt["context"]["structure"], STRUCTURE)
        self.assertEqual(attempt["context"]["claim_type"], "relation")
        self.assertEqual(attempt["review_fee_quote"]["offered_fee_uzrn"], "100000")
        self.assert_excerpt_absent(receipt)

    def test_signing_bridge_maps_relation_and_still_refuses_other_types(self):
        value = {"body": {"messages": [{"@type": "/zerone.knowledge.v1.MsgSubmitClaim", "claim_type": "CLAIM_TYPE_RELATION",
                                        "fact_content": FACT_CONTENT}]}}
        self.assertEqual(workflow.signing_json(value)["body"]["messages"][0]["claim_type"], 2)
        value["body"]["messages"][0]["claim_type"] = "CLAIM_TYPE_DEFINITION"
        with self.assertRaisesRegex(workflow.WorkflowError, "Unexpected ordinary"):
            workflow.signing_json(value)

    def test_submit_arguments_cannot_drift_from_retained_structure(self):
        claim, guard = self.guarded_claim()
        context = workflow.attribution_context(claim, guard)
        context["review_fee_uzrn"] = "100000"
        arguments = workflow.submit_arguments(context)
        arguments[arguments.index("--object") + 1] = "zrn-content:" + SECOND_HASH
        quote = {"height": "99", "base_fee_uzrn": "100000", "creation_multiplier_bps": "1000000",
                 "effective_multiplier_bps": "1000000", "offered_fee_uzrn": "100000"}
        with self.assertRaisesRegex(workflow.WorkflowError, "retained semantic intent"):
            self.w.send("submit", "user", arguments, context, fee_quote=quote)
        self.assertEqual(self.calls, [])

    def test_guarded_submit_challenge_and_commit_record_only_version_and_digest(self):
        args = argparse.Namespace(content=PROPOSITION, domain="chemistry", category="empirical", method="",
                                  reasoning="Paraphrase of a synthetic fixture source.", excerpt_file=str(self.excerpt_path))
        receipt = self.w.submit(args)
        self.assertEqual(receipt["quote_guard"]["excerpt_sha256"], "sha256:" + EXCERPT_SHA)
        self.assertNotIn("quote_guard", workflow.contribution_intent(self.w.records("attempts")[0][1]["context"]))

        self.observation["events"] = [{"type": "zerone.knowledge.submit_contradiction",
                                       "attributes": [{"key": "counter_claim_id", "value": "counter-1"}]}]
        challenge = argparse.Namespace(fact="fact-1", content=SECOND_TEXT, reason="Synthetic disagreement with a paraphrased source.",
                                       evidence=["hermes:ea:sha256:" + "ef" * 32], stake="11000000", excerpt_file=str(self.excerpt_path))
        result = self.w.challenge(challenge)
        self.assertEqual(result["quote_guard"]["version"], "zerone.quote-guard/v1")

        self.w.round_for_claim = mock.Mock(return_value=self.case.row())
        review = self.case.review_args(excerpt_file=str(self.excerpt_path), reason="Checked the locator and version digest.")
        result = self.w.review(review)
        self.assertEqual(result["quote_guard"]["excerpt_sha256"], "sha256:" + EXCERPT_SHA)
        self.assert_excerpt_absent(receipt, result)

    def test_guarded_attempt_keeps_snapshot_view_compatible(self):
        claim, guard = self.guarded_claim()
        self.w.attribute(claim, guard)
        self.w.history = mock.Mock(side_effect=lambda claim_id, height: {
            "chain_id": self.w.chain, "block_height": str(height), "record": {"claim_id": claim_id}})
        snapshot = self.w.snapshot()
        self.assertNotIn("quote_guard", snapshot["transactions"][0])
        view = workflow.sibling("claim-workflow-view")
        self.assertTrue(view._snapshot_bytes(snapshot))
        self.assert_excerpt_absent(snapshot)

    def test_guard_refuses_before_any_cli_call_or_saved_record(self):
        refusals = {
            "submit": lambda: self.w.submit(argparse.Namespace(content=EIGHT, domain="chemistry", category="empirical",
                                                                method="", reasoning="ok", excerpt_file=str(self.excerpt_path))),
            "submit reasoning": lambda: self.w.submit(argparse.Namespace(content=PROPOSITION, domain="chemistry", category="empirical",
                                                                          method="", reasoning=EIGHT, excerpt_file=str(self.excerpt_path))),
            "challenge": lambda: self.w.challenge(argparse.Namespace(fact="fact-1", content=SECOND_TEXT, reason="ok", evidence=[EIGHT],
                                                                     stake="11000000", excerpt_file=str(self.excerpt_path))),
            "commit": lambda: self.w.review(self.case.review_args(scope=EIGHT, excerpt_file=str(self.excerpt_path))),
        }
        self.w.round_for_claim = mock.Mock(return_value=self.case.row())
        for name, action in refusals.items():
            with self.subTest(name=name):
                self.w.query.reset_mock()
                self.w.actor.reset_mock()
                with self.assertRaisesRegex(workflow.WorkflowError, "Quote guard"):
                    action()
                self.assertEqual(self.calls, [])
                self.w.query.assert_not_called()
                self.w.actor.assert_not_called()
                self.assertEqual(self.w.records("attempts"), [])
                self.assertEqual(self.w.records("reviews"), [])

    def test_parsers_offer_excerpt_file_on_submit_challenge_and_commit(self):
        parser = workflow.parser()
        rows = {
            "submit": ["--content", "x", "--reasoning", "y"],
            "challenge": ["--fact", "f", "--content", "x", "--reason", "y"],
            "commit": ["--claim", "c", "--actor", "reviewer1", "--vote", "accept", "--reason", "r", "--scope", "s"],
        }
        for name, rest in rows.items():
            with self.subTest(name=name):
                args = parser.parse_args([name, "--home", str(self.home), *rest, "--excerpt-file", "e.txt"])
                self.assertEqual(args.excerpt_file, "e.txt")
        args = parser.parse_args(["attribute", "--record", "r.json"])
        self.assertEqual((args.broadcast, args.method, args.category, args.home), (False, "M-HISTORICAL", "empirical", None))


class AttributionCommandTests(unittest.TestCase):
    """Run the real script offline: argv carries paths only, never the excerpt."""

    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="zerone-attribution-test-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name).resolve()
        self.record_path = self.root / "record.json"
        self.record_path.write_text(json.dumps(RECORD, ensure_ascii=False), encoding="utf-8")
        self.excerpt_path = self.root / "excerpt.txt"
        self.excerpt_path.write_text(EXCERPT, encoding="utf-8")

    def run_script(self, *args):
        env = {"PATH": os.environ.get("PATH", "/usr/bin:/bin"), "HOME": str(self.root / "fake-home")}
        result = subprocess.run([sys.executable, "-I", "-B", str(SCRIPT), "attribute", *map(str, args)],
                                capture_output=True, text=True, timeout=60, env=env, cwd=self.root)
        for stream in (result.stdout, result.stderr):
            self.assertNotIn(EXCERPT, stream)
            self.assertNotIn("purified compound", stream.lower())
        self.assertFalse((self.root / "fake-home").exists())
        return result

    def test_dry_run_prints_the_attribution_fields_offline(self):
        result = self.run_script("--record", self.record_path, "--excerpt-file", self.excerpt_path)
        self.assertEqual(result.returncode, 0, result.stderr)
        value = json.loads(result.stdout)
        self.assertEqual(value["claim"], EXPECTED)
        self.assertIs(value["dry_run"], True)
        self.assertEqual(value["quote_guard"], {"version": "zerone.quote-guard/v1", "excerpt_sha256": "sha256:" + EXCERPT_SHA})
        unguarded = self.run_script("--record", self.record_path)
        self.assertEqual(unguarded.returncode, 0, unguarded.stderr)
        self.assertIsNone(json.loads(unguarded.stdout)["quote_guard"])

    def test_eight_excerpt_words_in_a_field_are_refused_before_signing(self):
        smuggled = record(source_version__work="doi:10.1000/the-purified-compound-z-was-observed-to-boil")
        self.record_path.write_text(json.dumps(smuggled), encoding="utf-8")
        missing_home = self.root / "no-such-home"
        result = self.run_script("--record", self.record_path, "--excerpt-file", self.excerpt_path,
                                 "--broadcast", "--home", missing_home)
        self.assertEqual(result.returncode, 1)
        self.assertIn("Quote guard", result.stderr)
        self.assertFalse(missing_home.exists())

    def test_broadcast_requires_the_excerpt_and_a_matching_digest(self):
        result = self.run_script("--record", self.record_path, "--broadcast", "--home", self.root / "no-such-home")
        self.assertEqual(result.returncode, 1)
        self.assertIn("--excerpt-file", result.stderr)
        self.excerpt_path.write_text(EXCERPT + "\n", encoding="utf-8")
        result = self.run_script("--record", self.record_path, "--excerpt-file", self.excerpt_path)
        self.assertEqual(result.returncode, 1)
        self.assertIn("excerpt_sha256", result.stderr)
        result = self.run_script("--record", self.record_path, "--resubmit-after", "A" * 64)
        self.assertEqual(result.returncode, 1)
        self.assertIn("--broadcast", result.stderr)


if __name__ == "__main__":
    unittest.main()

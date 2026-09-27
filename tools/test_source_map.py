"""Disposable, synthetic Git fixtures for the read-only source navigation CLI."""
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch as mock_patch


SCRIPT = Path(__file__).with_name("source-map.py")
CATALOG = "dashboard/module-guide.json"


class SourceMapTests(unittest.TestCase):
    def setUp(self):
        # Never use the platform's default temp directory or real user fixtures.
        self.temp = tempfile.TemporaryDirectory(prefix="source-map-test-", dir=os.environ["TMPDIR"])
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name) / "repo"
        self.root.mkdir()
        self.git("init", "-q")
        self.put(CATALOG, json.dumps({"groups": [], "modules": [], "sharedUtilities": [], "workflow": []}))
        self.git("add", "--", CATALOG)

    def git(self, *args, input=None):
        return subprocess.run(["git", "-C", str(self.root), *args], input=input,
                              capture_output=True, check=True).stdout

    def put(self, path, content="SYNTHETIC CONTENT MUST NOT BE READ\n", tracked=False):
        target = self.root / path
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(content, encoding="utf-8")
        if tracked:
            self.git("add", "--", path)
        return target

    def cli(self, *args):
        self.assertTrue(SCRIPT.is_file(), "source-map CLI has not been implemented")
        return subprocess.run([sys.executable, "-B", str(SCRIPT), "--root", str(self.root), *args],
                              capture_output=True, text=True)

    def result(self, *args):
        process = self.cli(*args)
        self.assertEqual(process.returncode, 0, process.stderr)
        self.assertEqual(process.stderr, "")
        return json.loads(process.stdout)

    def test_export_covers_only_indexed_paths_with_exact_identifiers(self):
        self.put("docs/a space and ü\tline\nbreak.md", tracked=True)
        self.put(".gitignore", "ignored/\n", tracked=True)
        self.put("untracked.go")
        self.put("ignored/local.json")
        result = self.result("export")
        self.assertEqual([r["path"] for r in result["files"]],
                         [".gitignore", CATALOG, "docs/a space and ü\tline\nbreak.md"])
        self.assertEqual(result["total"], 3)
        for record in result["files"]:
            for field in ("purpose", "purpose_basis", "kind", "area", "inspection", "tags"):
                self.assertTrue(record[field], (record["path"], field))
        self.assertNotIn("SYNTHETIC CONTENT", json.dumps(result))

    def test_unmerged_index_is_refused(self):
        oid = self.git("hash-object", "-w", "--stdin", input=b"synthetic conflict\n").strip()
        self.git("update-index", "--index-info", input=b"100644 " + oid + b" 1\tconflict.go\n")
        process = self.cli("export")
        self.assertEqual(process.returncode, 1)
        self.assertIn("unresolved index stage", process.stderr)
        self.assertEqual(process.stdout, "")

    def test_missing_tracked_input_is_refused(self):
        target = self.put("missing.go", tracked=True)
        target.unlink()
        process = self.cli("export")
        self.assertEqual(process.returncode, 1)
        self.assertIn("missing tracked input", process.stderr)
        self.assertIn("missing.go", process.stderr)
        self.assertEqual(process.stdout, "")

    def test_pointers_are_listed_without_following_targets(self):
        outside = Path(self.temp.name) / "outside.go"
        outside.write_text("SYNTHETIC OUTSIDE SENTINEL", encoding="utf-8")
        (self.root / "pointer.go").symlink_to(outside)
        (self.root / "dangling").symlink_to(Path(self.temp.name) / "absent")
        self.git("add", "--", "pointer.go", "dangling")
        # A gitlink need not have an initialized checkout or even a local directory.
        self.git("update-index", "--add", "--cacheinfo",
                 "160000,0123456789012345678901234567890123456789,dependency")
        result = self.result("export")
        records = {r["path"]: r for r in result["files"]}
        for name in ("pointer.go", "dangling", "dependency"):
            self.assertEqual(records[name]["inspection"], "pointer-not-followed")
        self.assertEqual(records["pointer.go"]["kind"], "symlink")
        self.assertEqual(records["dependency"]["kind"], "gitlink")
        self.assertNotIn("SYNTHETIC OUTSIDE", json.dumps(result))

    def test_symlink_ancestor_is_refused(self):
        target = self.put("redirect/item.go", tracked=True)
        target.unlink()
        target.parent.rmdir()
        outside = Path(self.temp.name) / "outside-dir"
        outside.mkdir()
        (outside / "item.go").write_text("SYNTHETIC OUTSIDE", encoding="utf-8")
        target.parent.symlink_to(outside, target_is_directory=True)
        process = self.cli("export")
        self.assertEqual(process.returncode, 1)
        self.assertIn("unsafe ancestor", process.stderr)
        self.assertEqual(process.stdout, "")

    def test_export_reuses_working_tree_module_catalog(self):
        guide = self.install_guide()
        self.put(CATALOG, json.dumps(guide))  # unstaged reviewed-label edits are visible
        guide["modules"][0]["purpose"] = "Unstaged reviewed purpose."
        self.put(CATALOG, json.dumps(guide))
        result = self.result("export")
        self.assertIn("modules", result)
        modules = {m["id"]: m for m in result["modules"]}
        self.assertEqual(modules["zerone_staking"]["purpose"], "Unstaged reviewed purpose.")
        self.assertEqual(modules["zerone_staking"]["purpose_basis"], "reviewed-catalog")
        self.assertEqual(modules["zerone_staking"]["purpose_source"],
                         {"path": CATALOG, "module_id": "zerone_staking"})
        self.assertEqual(modules["zerone_staking"]["connections"], guide["modules"][0]["connections"])
        self.assertEqual(result["sharedUtilities"], guide["sharedUtilities"])
        self.assertEqual(result["catalog"], CATALOG)

    def install_guide(self):
        guide = {
            "groups": [{"id": "consensus", "title": "Consensus"}],
            "modules": [
                {"id": "zerone_staking", "label": "Custom staking", "family": "custom",
                 "group": "consensus", "purpose": "Synthetic custom ledger purpose.",
                 "owns": "Custom ledger.", "boundary": "Not consensus staking.",
                 "sourcePath": "x/staking/keeper/keeper.go",
                 "connections": [{"module": "staking", "why": "Synthetic connection."}]},
                {"id": "staking", "label": "SDK staking", "family": "sdk-ibc",
                 "group": "consensus", "purpose": "Synthetic consensus purpose.",
                 "owns": "Consensus.", "boundary": "Not custom staking.",
                 "sourcePath": "app/app.go", "connections": []}],
            "sharedUtilities": [{"label": "x/common", "text": "Synthetic shared helpers.",
                                 "sourceKind": "tree", "sourcePath": "x/common"}],
            "workflow": [{"id": "stake", "moduleIds": ["staking"],
                          "link": {"path": "docs/staking.md#overview"}}],
        }
        for path in ("x/staking/keeper/keeper.go", "app/app.go", "x/common/helper.go", "docs/staking.md"):
            self.put(path, tracked=True)
        self.put(CATALOG, json.dumps(guide), tracked=True)
        return guide

    def test_catalog_symlinks_are_refused_before_read(self):
        outside = Path(self.temp.name) / "outside-catalog.json"
        outside.write_text("INVALID SENTINEL MUST NOT BE PARSED", encoding="utf-8")
        (self.root / CATALOG).unlink()
        (self.root / CATALOG).symlink_to(outside)
        process = self.cli("export")
        self.assertEqual(process.returncode, 1)
        self.assertIn("catalog must be an indexed regular file", process.stderr)
        self.assertNotIn("Traceback", process.stderr)
        self.assertEqual(process.stdout, "")

    def test_module_command_accepts_only_exact_ids_or_enumerated_path_aliases(self):
        guide = self.install_guide()
        custom = self.result("module", "zerone_staking")
        self.assertEqual(custom["id"], "zerone_staking")
        self.assertEqual(custom["aliases"], ["x/staking"])
        self.assertEqual(custom, self.result("module", "x/staking"))
        self.assertEqual(custom["connections"], guide["modules"][0]["connections"])
        sdk = self.result("module", "staking")
        self.assertEqual(sdk["family"], "sdk-ibc")
        self.assertEqual(sdk["aliases"], [])
        for selector in ("Staking", "stake", "staking ", "./x/staking", "x/staking/", "x/unknown", "SDK staking"):
            with self.subTest(selector=selector):
                process = self.cli("module", selector)
                self.assertEqual(process.returncode, 1)
                self.assertIn("unknown exact module ID or alias", process.stderr)
                self.assertEqual(process.stdout, "")

    def test_show_returns_exact_path_with_separate_module_context(self):
        self.install_guide()
        self.put("x/staking/keeper/a space.go", tracked=True)
        record = self.result("show", "x/staking/keeper/a space.go")
        self.assertEqual(record["path"], "x/staking/keeper/a space.go")
        self.assertEqual(record["module_ids"], ["zerone_staking"])
        self.assertEqual(record["area"], "module:zerone_staking")
        self.assertEqual(record["purpose_basis"], "path-inference")
        self.assertNotIn("Synthetic custom ledger", record["purpose"])
        self.assertEqual([m["id"] for m in record["modules"]], ["zerone_staking"])
        for selector in ("a space.go", "./x/staking/keeper/a space.go", "x/staking/keeper", "../outside"):
            with self.subTest(selector=selector):
                process = self.cli("show", selector)
                self.assertEqual(process.returncode, 1)
                self.assertIn("unknown exact indexed path", process.stderr)
                self.assertEqual(process.stdout, "")
        self.put("x/stakingish/main.go", tracked=True)
        other = self.result("show", "x/stakingish/main.go")
        self.assertEqual(other["module_ids"], [])

    def test_file_roles_are_honest_path_inferences(self):
        cases = {
            "app/ante.go": ("source", "application"),
            "tools/test_source_map.py": ("test", "tooling"),
            "app/ante_test.go": ("test", "application"),
            "dashboard/a.test.ts": ("test", "dashboard"),
            "proto/zerone/staking/v1/tx.proto": ("schema", "schema"),
            "tools/contracts/record.schema.json": ("schema", "tooling"),
            "x/unknown/types/query.pb.go": ("generated", "review-needed"),
            "sdk/client/query_pb.ts": ("source", "sdk"),
            "docs/specs/settlement.md": ("spec", "documentation"),
            "docs/plans/settlement.md": ("plan", "documentation"),
            "docs/README.md": ("doc", "documentation"),
            "docs/evidence/snapshot.json": ("evidence", "documentation"),
            "dashboard/public/research/checkpoints/run/README.md": ("evidence", "dashboard"),
            "dashboard/public/favicon.svg": ("asset", "dashboard"),
            "config/app.toml.template": ("config", "operations"),
            "prompts/private.md": ("prompt", "prompts"),
            "private/keys.json": ("private", "review-needed"),
            "data/events.jsonl": ("data", "review-needed"),
            "sdk/package.json": ("config", "sdk"),
            "tests/conftest.py": ("test-support", "tests"),
            "tests/__init__.py": ("package", "tests"),
            "unfamiliar.blob": ("review-needed", "review-needed"),
        }
        for path in cases:
            self.put(path, tracked=True)
        records = {r["path"]: r for r in self.result("export")["files"]}
        for path, (kind, area) in cases.items():
            with self.subTest(path=path):
                record = records[path]
                self.assertEqual((record["kind"], record["area"]), (kind, area))
                self.assertEqual(record["inspection"], "metadata-only")
                self.assertEqual(record["purpose_basis"], "path-inference")
                self.assertIn(kind, record["tags"])
                self.assertIn(path.rsplit("/", 1)[-1], record["purpose"])
        self.assertIn("Review-needed", records["unfamiliar.blob"]["purpose"])

    def test_find_is_bounded_with_truthful_counts(self):
        self.install_guide()
        self.put("x/staking/keeper/another.go", tracked=True)
        result = self.result("find", "staking", "--limit", "2")
        self.assertEqual(result["total"], 4)
        self.assertEqual(result["returned"], 2)
        self.assertEqual(len(result["files"]), 2)
        self.assertEqual(result["limit"], 2)
        self.assertEqual(result["files"], sorted(result["files"], key=lambda record: record["path"]))
        self.assertEqual(self.result("find", "STAKING", "--limit", "2")["files"], result["files"])
        by_purpose = self.result("find", "Synthetic custom ledger")
        self.assertEqual(by_purpose["total"], 2)
        self.assertEqual(by_purpose["returned"], 2)
        empty = self.result("find", "no-such-metadata")
        self.assertEqual((empty["total"], empty["returned"]), (0, 0))

    def test_find_rejects_empty_queries_or_out_of_bounds_limits(self):
        for limit in ("0", "-1", "201", "100000000000000", "abc"):
            with self.subTest(limit=limit):
                process = self.cli("find", "file", "--limit", limit)
                self.assertEqual(process.returncode, 2)
                self.assertEqual(process.stdout, "")
        for query in ("", "   ", "\t\n"):
            with self.subTest(query=query):
                process = self.cli("find", query)
                self.assertEqual(process.returncode, 2)
                self.assertIn("nonempty", process.stderr)
        self.assertEqual(self.result("find", "file", "--limit", "200")["limit"], 200)

    def test_summary_reports_counts_scope_and_review_queue(self):
        self.install_guide()
        self.put("unfamiliar.blob", tracked=True)
        result = self.result("summary")
        self.assertEqual(result["total"], 6)
        self.assertEqual(result["module_total"], 2)
        self.assertEqual(sum(result["kinds"].values()), 6)
        self.assertEqual(sum(result["areas"].values()), 6)
        self.assertEqual(result["review_needed"], 1)
        self.assertIn("Git-indexed working-tree paths", result["scope"])
        self.assertEqual(result["content_reads"], [CATALOG])
        self.assertNotIn("files", result)

    def test_check_verifies_coverage_without_claiming_runtime_registration(self):
        self.install_guide()
        result = self.result("check")
        self.assertEqual(result["status"], "ok")
        self.assertEqual(result["total"], 5)
        self.assertEqual(result["module_total"], 2)
        self.assertEqual(result["runtime_authority"],
                         "go test ./app -run '^TestModuleGuideCoversRegisteredApplication$' -count=1")
        self.assertFalse(result["runtime_registration_checked"])
        self.assertFalse(result["document_anchors_checked"])

    def test_invalid_catalog_ids_and_connections_are_rejected(self):
        original = self.install_guide()
        variants = [
            (lambda g: g["modules"].append(g["modules"][0].copy()), "duplicate module ID"),
            (lambda g: g["modules"][0].update(id="Bad ID"), "invalid module ID"),
            (lambda g: g["modules"][0].update(purpose="  "), "nonempty purpose"),
            (lambda g: g["modules"][0].update(family="imaginary"), "unknown family"),
            (lambda g: g["modules"][0].update(group="missing"), "unknown group"),
            (lambda g: g["modules"][0]["connections"][0].update(module="missing"), "unknown connection target"),
            (lambda g: g["modules"][0]["connections"][0].update(why=""), "nonempty why"),
            (lambda g: g["groups"].append(g["groups"][0].copy()), "duplicate group ID"),
        ]
        for change, expected in variants:
            with self.subTest(expected=expected):
                guide = json.loads(json.dumps(original))
                change(guide)
                self.put(CATALOG, json.dumps(guide))
                process = self.cli("check")
                self.assertEqual(process.returncode, 1)
                self.assertIn(expected, process.stderr)
                self.assertEqual(process.stdout, "")

    def test_catalog_references_must_be_safe_indexed_paths(self):
        original = self.install_guide()
        self.put("untracked.go")
        (self.root / "linked.go").symlink_to(Path(self.temp.name) / "absent")
        self.git("add", "--", "linked.go")
        for path in ("absent.go", "untracked.go", "linked.go", "../outside.go", "/absolute", "app/../app/app.go", "./app/app.go", "app//app.go"):
            with self.subTest(path=path):
                guide = json.loads(json.dumps(original))
                guide["modules"][0]["sourcePath"] = path
                self.put(CATALOG, json.dumps(guide))
                process = self.cli("check")
                self.assertEqual(process.returncode, 1)
                self.assertIn("catalog path", process.stderr)
                self.assertEqual(process.stdout, "")
        self.put(CATALOG, json.dumps(original))
        (self.root / "docs/staking.md").unlink()
        (self.root / "docs/staking.md").symlink_to(Path(self.temp.name) / "absent-doc")
        process = self.cli("check")
        self.assertEqual(process.returncode, 1)
        self.assertIn("catalog path", process.stderr)

    def test_shared_and_workflow_references_are_validated(self):
        original = self.install_guide()
        variants = [
            (lambda g: g["sharedUtilities"][0].update(sourcePath="x/absent"), "catalog tree"),
            (lambda g: g["sharedUtilities"][0].update(sourcePath="../outside"), "catalog path"),
            (lambda g: g["sharedUtilities"][0].update(sourceKind="blob"), "catalog path"),
            (lambda g: g["sharedUtilities"][0].update(sourceKind="invalid"), "sourceKind"),
            (lambda g: g["workflow"][0].update(moduleIds=["absent"]), "workflow module"),
            (lambda g: g["workflow"][0]["link"].update(path="docs/absent.md#section"), "catalog path"),
            (lambda g: g["modules"][1].update(family="custom", sourcePath="x/staking/keeper/keeper.go"), "ambiguous module alias"),
        ]
        for change, expected in variants:
            with self.subTest(expected=expected):
                guide = json.loads(json.dumps(original))
                change(guide)
                self.put(CATALOG, json.dumps(guide))
                process = self.cli("check")
                self.assertEqual(process.returncode, 1)
                self.assertIn(expected, process.stderr)
                self.assertEqual(process.stdout, "")

    def test_check_detects_new_custom_module_directory_without_miscounting_utilities(self):
        self.install_guide()
        self.put("x/newmodule/module.go", tracked=True)
        process = self.cli("check")
        self.assertEqual(process.returncode, 1)
        self.assertIn("uncatalogued x/ area", process.stderr)
        self.assertIn("x/newmodule", process.stderr)
        self.assertNotIn("x/common", process.stderr)
        # Navigation remains available to review the drift; check is the failing gate.
        self.assertEqual(self.result("show", "x/newmodule/module.go")["module_ids"], [])

    def test_malformed_catalog_shapes_fail_cleanly(self):
        original = self.install_guide()
        documents = ["not JSON", "[]", "{}", "null",
                     json.dumps({**original, "modules": {}}),
                     json.dumps({**original, "modules": [None]}),
                     '{"groups": [], "modules": [], "modules": [], "sharedUtilities": []}',
                     json.dumps({**original, "workflow": [{"moduleIds": "staking"}]}),
                     json.dumps({**original, "sharedUtilities": [None]})]
        altered = json.loads(json.dumps(original))
        altered["modules"][0]["connections"] = None
        documents.append(json.dumps(altered))
        for document in documents:
            with self.subTest(document=document[:80]):
                self.put(CATALOG, document)
                process = self.cli("check")
                self.assertEqual(process.returncode, 1)
                self.assertIn("catalog", process.stderr)
                if document.startswith('{"groups": []'):
                    self.assertIn("duplicate catalog key", process.stderr)
                self.assertNotIn("Traceback", process.stderr)
                self.assertEqual(process.stdout, "")

    def test_wrong_repository_root_fails_cleanly(self):
        nested = self.root / "nested"
        nested.mkdir()
        process = self.cli("--root", str(nested), "export")
        self.assertEqual(process.returncode, 1)
        self.assertIn("repository root", process.stderr)
        self.assertNotIn("Traceback", process.stderr)

    def test_nonregular_inputs_fail_cleanly(self):
        target = self.put("not-a-file.go", tracked=True)
        target.unlink()
        target.mkdir()
        process = self.cli("export")
        self.assertEqual(process.returncode, 1)
        self.assertIn("not a regular file or symlink", process.stderr)
        self.assertEqual(process.stdout, "")

    def load_cli_module(self):
        spec = importlib.util.spec_from_file_location("zerone_source_map", SCRIPT)
        assert spec is not None and spec.loader is not None
        module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(module)
        return module

    def test_only_reviewed_catalog_contents_are_opened(self):
        self.install_guide()
        for path in ("source.py", "prompts/agent.md", "private/key.json", "data/chat.jsonl", "config/local.json"):
            self.put(path, "raise RuntimeError('SYNTHETIC NEVER READ OR RUN')", tracked=True)
        module = self.load_cli_module()
        real_open = os.open
        real_io_open = io.open
        reads = []

        def guarded_open(path, flags, *args, **kwargs):
            if not flags & os.O_DIRECTORY:
                self.assertEqual(os.fspath(path), "module-guide.json", "unexpected content read")
                self.assertTrue(flags & os.O_NOFOLLOW)
                self.assertFalse(flags & (os.O_WRONLY | os.O_RDWR | os.O_CREAT | os.O_TRUNC))
                reads.append(path)
            return real_open(path, flags, *args, **kwargs)

        def fd_only_open(path, *args, **kwargs):
            self.assertIsInstance(path, int, "unexpected path content read")
            return real_io_open(path, *args, **kwargs)

        # File descriptors wrap already-guarded catalog reads and subprocess pipes.
        # No other open API may read source/prompt/config paths.
        with mock_patch("builtins.open", side_effect=AssertionError("unexpected open")), \
             mock_patch("io.open", side_effect=fd_only_open), \
             mock_patch.object(module.os, "open", side_effect=guarded_open):
            census = module.export(self.root)
        self.assertEqual(reads, ["module-guide.json"])
        self.assertNotIn("SYNTHETIC NEVER", json.dumps(census))
        record = next(r for r in census["files"] if r["path"] == CATALOG)
        self.assertEqual(record["inspection"], "reviewed-catalog")
        self.assertEqual(record["kind"], "catalog")
        self.assertEqual(record["purpose_basis"], "explicit-path-rule")

    def test_export_is_deterministic_read_only_and_self_describing(self):
        self.install_guide()
        def snapshot():
            return {str(path.relative_to(self.root)): (path.lstat().st_mtime_ns, path.lstat().st_size)
                    for path in self.root.rglob("*")}
        before = snapshot()
        first = self.cli("export")
        second = self.cli("export")
        self.assertEqual(first.returncode, 0, first.stderr)
        self.assertEqual(first.stdout, second.stdout)
        self.assertEqual(before, snapshot())
        result = json.loads(first.stdout)
        self.assertIn("scope", result)
        self.assertIn("Git-indexed working-tree paths", result["scope"])
        self.assertEqual(result["schema_version"], 1)
        self.assertEqual(result["content_reads"], [CATALOG])
        self.assertNotIn(str(self.root), first.stdout)

    def test_catalog_context_labels_proto_and_shared_utility_paths(self):
        self.install_guide()
        self.put("proto/zerone/staking/v1/tx.proto", tracked=True)
        self.put("sdk/staking.ts", tracked=True)
        schema = self.result("show", "proto/zerone/staking/v1/tx.proto")
        self.assertEqual(schema["module_ids"], ["zerone_staking"])
        self.assertEqual(schema["area"], "module:zerone_staking")
        self.assertTrue({"schema", "module:zerone_staking", "family:custom", "group:consensus"}.issubset(schema["tags"]))
        self.assertNotIn("review-needed", schema["tags"])
        utility = self.result("show", "x/common/helper.go")
        self.assertEqual(utility["area"], "shared-utilities")
        self.assertEqual(utility["module_ids"], [])
        self.assertEqual(utility["shared_utility_paths"], ["x/common"])
        sdk = self.result("show", "sdk/staking.ts")
        self.assertEqual(sdk["module_ids"], [])  # SDK client filename is not SDK staking registration.
        app = self.result("show", "app/app.go")
        self.assertEqual(app["area"], "application")

    def test_all_indexed_paths_are_validated_before_labels(self):
        module = self.load_cli_module()
        self.assertTrue(hasattr(module, "parse_index"), "index parser must be independently testable")
        oid = b"1" * 40
        for path in (b"../outside", b"/absolute", b"a//b", b"a/./b", b"a/../b", b".git/config"):
            with self.subTest(path=path):
                with self.assertRaisesRegex(module.MapError, "unsafe indexed path"):
                    module.parse_index(b"100644 " + oid + b" 0\t" + path + b"\0")
        with self.assertRaisesRegex(module.MapError, "duplicate indexed path"):
            module.parse_index((b"100644 " + oid + b" 0\ta\0") * 2)
        for data in (b"broken", b"100644 " + oid + b" 0\ta", b"000000 " + oid + b" 0\ta\0"):
            with self.subTest(data=data):
                with self.assertRaises(module.MapError):
                    module.parse_index(data)
        data = b"100644 " + oid + b" 0\tname\twith\nspaces \0"
        self.assertEqual(module.parse_index(data)[0]["path"], "name\twith\nspaces ")

    def test_coverage_validation_rejects_missing_or_empty_labels(self):
        module = self.load_cli_module()
        self.assertTrue(hasattr(module, "validate_coverage"), "coverage must have an independent set check")
        entries = module.indexed_paths(self.root)
        files = module.export(self.root)["files"]
        module.validate_coverage(entries, files)
        for damaged in ([], files + files, [{**files[0], "path": "untracked"}]):
            with self.subTest(damaged=damaged):
                with self.assertRaisesRegex(module.MapError, "coverage"):
                    module.validate_coverage(entries, damaged)
        for field in ("purpose", "purpose_basis", "kind", "area", "inspection", "tags"):
            with self.subTest(field=field):
                with self.assertRaisesRegex(module.MapError, "label"):
                    module.validate_coverage(entries, [{**files[0], field: ""}])

    def test_io_failure_reports_cleanly_without_partial_output(self):
        link = Path(self.temp.name) / "linked-repo"
        link.symlink_to(self.root, target_is_directory=True)
        process = self.cli("--root", str(link), "export")
        self.assertEqual(process.returncode, 1)
        self.assertNotIn("Traceback", process.stderr)
        self.assertIn("source-map:", process.stderr)
        self.assertEqual(process.stdout, "")

    def test_repository_path_rules_distinguish_operations_artifacts_and_generated_code(self):
        cases = {
            "cmd/zeroned/main.go": ("source", "commands"),
            "internal/helpers.go": ("source", "internal"),
            "integrations/client.go": ("source", "integrations"),
            "scripts/check.sh": ("source", "automation"),
            "deploy/entrypoint_test.sh": ("test", "operations"),
            "deploy/fly.toml": ("config", "operations"),
            "networks/local.json": ("data", "operations"),
            "cosmovisor/README.md": ("doc", "operations"),
            "canon/README.md": ("doc", "canon"),
            "skills/helper.md": ("doc", "agent-tooling"),
            ".claude/settings.json": ("config", "agent-tooling"),
            ".github/workflows/ci.yml": ("config", "automation"),
            "go.mod": ("config", "repository"),
            "Dockerfile": ("config", "repository"),
            "Makefile": ("config", "repository"),
            "README.md": ("doc", "repository"),
            "dashboard/.dev.vars.example": ("config", "dashboard"),
            "sdk/typescript/src/generated/tx.ts": ("generated", "sdk"),
            "sdk/typescript/dist/index.js": ("generated", "sdk"),
            "sdk/typescript/tsconfig.build.json": ("config", "sdk"),
            "sdk/typescript/package-lock.json": ("config", "sdk"),
            "testdata/vector.json": ("data", "tests"),
            "docs/reports/report.md": ("evidence", "documentation"),
            "docs/plot.pdf": ("asset", "documentation"),
        }
        for path in cases:
            self.put(path, tracked=True)
        records = {r["path"]: r for r in self.result("export")["files"]}
        for path, expected in cases.items():
            with self.subTest(path=path):
                self.assertEqual((records[path]["kind"], records[path]["area"]), expected)
                self.assertEqual(records[path]["purpose_basis"], "path-inference")

    def test_metadata_roles_cover_integrity_licenses_migrations_and_opaque_assets(self):
        cases = {
            ".readme-hash": "integrity-pin", ".specs-and-plans-hashes": "integrity-pin",
            "LICENSE": "license", "sdk/typescript/LICENSE": "license",
            "docs/examples/NOTICE": "license", "docs/.gitkeep": "placeholder",
            "dashboard/migrations/0001_pi_identity.sql": "migration",
            "dashboard/public/_headers": "config", "dashboard/public/llms.txt": "doc",
            "dashboard/public/evidence.zip": "archive",
            "deploy/query-gateway/default.conf.template": "config",
            "deploy/legacy-observer/dependencies.patch": "patch",
            "proto/buf.lock": "config", "deploy/testnet/node-id.txt": "data",
            "sdk/typescript/tests/consumer-nodenext.mts": "test", "seeds.txt": "data",
        }
        for path in cases:
            self.put(path, tracked=True)
        records = {r["path"]: r for r in self.result("export")["files"]}
        for path, kind in cases.items():
            with self.subTest(path=path):
                self.assertEqual(records[path]["kind"], kind)
                self.assertEqual(records[path]["inspection"], "metadata-only")
                self.assertEqual(records[path]["purpose_basis"], "path-inference")
        self.assertEqual(self.result("summary")["review_needed"], 0)

    def test_filename_purposes_remain_inferences_not_module_descriptions(self):
        self.install_guide()
        cases = {"module.go": "module integration", "keeper.go": "keeper definition",
                 "grpc_query.go": "query handlers", "msg_server.go": "transaction message handlers",
                 "expected_keepers.go": "keeper dependency interfaces", "genesis.go": "genesis state",
                 "codec.go": "encoding registration", "main.go": "entry point"}
        for name in cases:
            self.put("x/staking/" + name, tracked=True)
        records = {r["path"]: r for r in self.result("export")["files"]}
        for name, expected in cases.items():
            with self.subTest(name=name):
                record = records["x/staking/" + name]
                self.assertIn(expected, record["purpose"])
                self.assertIn("filename suggests", record["purpose"])
                self.assertEqual(record["purpose_basis"], "path-inference")
                self.assertNotIn("Synthetic custom ledger", record["purpose"])


if __name__ == "__main__":
    unittest.main()

#!/usr/bin/env python3
"""Read-only navigation of Git-indexed Zerone working-tree paths (Python 3.9+, POSIX).

Only dashboard/module-guide.json is content-read, from the working tree after
no-symlink checks. Git supplies path membership, including staged additions.
All other files are labeled from paths and filesystem metadata, never parsed or
imported. Module purposes/connections retain the reviewed catalog's provenance;
they are not an inferred runtime call graph or a claim about a deployed network.

All commands emit JSON; export writes only to stdout. Exit 0: success, 1: invalid
or incomplete census/unknown exact selector, 2: usage error. --root precedes the
command and defaults to this script's repository, independent of the current cwd.
Run tests with: python3 -B tools/test_source_map.py (set TMPDIR to fixture scratch).
"""
import argparse
from collections import Counter
from contextlib import contextmanager
import json
import os
import re
from pathlib import Path
import stat
import subprocess
import sys


CATALOG = "dashboard/module-guide.json"
SCOPE = "Git-indexed working-tree paths; untracked/ignored files excluded; gitlinks not traversed."
ROOT_AREAS = {
    "app": "application", "tools": "tooling", "dashboard": "dashboard", "proto": "schema",
    "sdk": "sdk", "docs": "documentation", "config": "operations", "prompts": "prompts",
    "tests": "tests", "testdata": "tests", "cmd": "commands", "internal": "internal",
    "integrations": "integrations", "scripts": "automation", ".github": "automation",
    "deploy": "operations", "networks": "operations", "cosmovisor": "operations",
    "canon": "canon", "skills": "agent-tooling", ".claude": "agent-tooling",
}


FILENAME_HINTS = {
    "module.go": "module integration", "keeper.go": "keeper definition",
    "grpc_query.go": "query handlers", "msg_server.go": "transaction message handlers",
    "expected_keepers.go": "keeper dependency interfaces", "genesis.go": "genesis state",
    "codec.go": "encoding registration", "main.go": "entry point",
}


class MapError(ValueError):
    """An invalid or incomplete census; no partial successful output."""


@contextmanager
def parent_fd(root, path):
    """Traverse directories without following symlinks, including during races."""
    fd = os.open(root, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
    try:
        for part in path.split("/")[:-1]:
            try:
                child = os.open(part, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=fd)
            except FileNotFoundError:
                raise MapError(f"missing tracked input: {path!r}") from None
            except OSError:
                raise MapError(f"unsafe ancestor for {path!r}") from None
            os.close(fd)
            fd = child
        yield fd, path.split("/")[-1]
    finally:
        os.close(fd)


def git_output(root, *args):
    process = subprocess.run(["git", "--no-optional-locks", "-c", "core.fsmonitor=false",
                              "-C", str(root), *args], capture_output=True)
    if process.returncode:
        raise MapError("cannot read Git index; --root must name a Git working-tree repository root")
    return process.stdout


def parse_index(data):
    """Parse the NUL-delimited stage census without normalizing any identifier."""
    if data and not data.endswith(b"\0"):
        raise MapError("incomplete Git index census")
    entries = {}
    for row in data.split(b"\0")[:-1]:
        try:
            meta, raw_path = row.split(b"\t", 1)
            mode, oid, stage = meta.split()
        except ValueError:
            raise MapError("malformed Git index census") from None
        path = os.fsdecode(raw_path)
        if stage != b"0":
            raise MapError(f"unresolved index stage for {path!r}")
        if not safe_path(path):
            raise MapError(f"unsafe indexed path: {path!r}")
        if mode not in {b"100644", b"100755", b"120000", b"160000"}:
            raise MapError(f"unsupported index mode for {path!r}")
        if path in entries:
            raise MapError(f"duplicate indexed path: {path!r}")
        entries[path] = {"path": path, "git_mode": mode.decode("ascii")}
    return [entries[path] for path in sorted(entries)]


def indexed_paths(root):
    top = os.fsdecode(git_output(root, "rev-parse", "--show-toplevel").removesuffix(b"\n"))
    if not os.path.samefile(root, top):
        raise MapError("--root must name the repository root, not a subdirectory")
    entries = parse_index(git_output(root, "ls-files", "--stage", "-z"))
    for entry in entries:
        path = entry["path"]
        if entry["git_mode"] == "160000":
            node_kind = "gitlink"
        else:
            try:
                with parent_fd(root, path) as (fd, name):
                    info = os.stat(name, dir_fd=fd, follow_symlinks=False)
            except FileNotFoundError:
                raise MapError(f"missing tracked input: {path!r}") from None
            if not (stat.S_ISLNK(info.st_mode) or stat.S_ISREG(info.st_mode)):
                raise MapError(f"tracked input is not a regular file or symlink: {path!r}")
            node_kind = "symlink" if stat.S_ISLNK(info.st_mode) else "file"
        entry["node_kind"] = node_kind
    return entries


def infer_kind(path):
    """Conservative path conventions, not a claim about inspected file contents."""
    parts = path.lower().split("/")
    name = parts[-1]
    suffix = Path(name).suffix
    if "private" in parts or "secrets" in parts:
        return "private"
    if "prompts" in parts:
        return "prompt"
    if "data" in parts or suffix == ".jsonl":
        return "data"
    if "evidence" in parts or "checkpoints" in parts or "reports" in parts:
        return "evidence"
    if name.endswith(("-hash", "-hashes")) and name.startswith("."):
        return "integrity-pin"
    if name in {"license", "notice"}:
        return "license"
    if name == ".gitkeep":
        return "placeholder"
    if suffix == ".sql" and "migrations" in parts:
        return "migration"
    if name == "llms.txt":
        return "doc"
    if suffix in {".zip", ".gz", ".tar"}:
        return "archive"
    if suffix == ".patch":
        return "patch"
    if suffix == ".lock" or name.endswith(".conf.template") or name in {"_headers", "_redirects"}:
        return "config"
    if suffix == ".mts" and "tests" in parts:
        return "test"
    if suffix == ".txt":
        return "data"
    if name == "__init__.py":
        return "package"
    if name == "conftest.py":
        return "test-support"
    if (name.startswith("test_") or name.endswith(("_test.go", "_test.py", "_test.sh")) or
            ".test." in name or ".spec." in name):
        return "test"
    if suffix == ".proto" or name.endswith(".schema.json"):
        return "schema"
    if name.endswith((".pb.go", ".pb.gw.go")) or "generated" in parts or path.startswith("sdk/typescript/dist/"):
        return "generated"
    if "specs" in parts:
        return "spec"
    if "plans" in parts:
        return "plan"
    if suffix in {".md", ".rst"}:
        return "doc"
    if suffix in {".svg", ".png", ".jpg", ".gif", ".ico", ".woff", ".woff2", ".pdf"}:
        return "asset"
    if ("config" in parts or suffix in {".toml", ".yaml", ".yml", ".ini", ".conf"} or
            name in {"package.json", "package-lock.json", "go.mod", "go.sum", "makefile",
                     "settings.json", ".gitignore", ".gitattributes", ".dockerignore"} or
            name.startswith(("dockerfile", "tsconfig.", ".env", ".dev.vars"))):
        return "config"
    if suffix in {".json", ".csv", ".jsonl"} or "testdata" in parts:
        return "data"
    if suffix in {".go", ".py", ".ts", ".tsx", ".js", ".mjs", ".sh", ".c", ".h", ".html", ".css"}:
        return "source"
    return "review-needed"


def label(entry):
    if entry["path"] == CATALOG:
        return {**entry, "purpose": "Reviewed module purposes, boundaries and documented connections.",
                "purpose_basis": "explicit-path-rule", "kind": "catalog", "area": "dashboard",
                "inspection": "reviewed-catalog", "tags": ["catalog", "dashboard"]}
    if entry["node_kind"] in {"symlink", "gitlink"}:
        kind = entry["node_kind"]
        return {**entry, "purpose": f"{kind.capitalize()} pointer; target not inspected.",
                "purpose_basis": "git/filesystem-metadata", "kind": kind, "area": "review-needed",
                "inspection": "pointer-not-followed", "tags": [kind]}
    path = entry["path"]
    kind = infer_kind(path)
    fallback = "repository" if "/" not in path and kind != "review-needed" else "review-needed"
    area = ROOT_AREAS.get(path.split("/")[0], fallback)
    name = path.rsplit("/", 1)[-1]
    hint = FILENAME_HINTS.get(name) if kind == "source" else None
    inference = f"filename suggests {hint}" if hint else "inferred from path"
    purpose = f"{kind.capitalize()} file {name}; {inference}, contents uninspected."
    return {**entry, "purpose": purpose, "purpose_basis": "path-inference", "kind": kind,
            "area": area, "inspection": "metadata-only", "tags": sorted({kind, area})}


def unique_object(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise MapError(f"duplicate catalog key: {key!r}")
        result[key] = value
    return result


def read_catalog(root, entries):
    entry = next((entry for entry in entries if entry["path"] == CATALOG), None)
    if not entry or entry["node_kind"] != "file" or entry["git_mode"] not in {"100644", "100755"}:
        raise MapError("catalog must be an indexed regular file")
    with parent_fd(root, CATALOG) as (parent, name):
        fd = os.open(name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=parent)
        with os.fdopen(fd, "r", encoding="utf-8") as stream:
            if not stat.S_ISREG(os.fstat(stream.fileno()).st_mode):
                raise MapError("catalog must be an indexed regular file")
            try:
                return json.load(stream, object_pairs_hook=unique_object)
            except (json.JSONDecodeError, UnicodeError):
                raise MapError("catalog is not valid UTF-8 JSON") from None


def nonempty(item, key):
    value = item.get(key) if isinstance(item, dict) else None
    if not isinstance(value, str) or not value.strip():
        raise MapError(f"catalog requires nonempty {key}")
    return value


def list_field(item, key, element_type=dict):
    value = item.get(key) if isinstance(item, dict) else None
    if not isinstance(value, list) or any(not isinstance(member, element_type) for member in value):
        raise MapError(f"catalog requires a list of {element_type.__name__} for {key}")
    return value


def catalog_ids(items, kind):
    ids = set()
    for item in items:
        value = nonempty(item, "id")
        if not re.fullmatch(r"[a-z0-9][a-z0-9_-]*", value):
            raise MapError(f"invalid {kind} ID: {value!r}")
        if value in ids:
            raise MapError(f"duplicate {kind} ID: {value!r}")
        ids.add(value)
    return ids


def safe_path(path):
    return (isinstance(path, str) and bool(path) and "\0" not in path and
            all(part not in {"", ".", "..", ".git"} for part in path.split("/")))


def catalog_path(path, entries, kind="blob"):
    if not safe_path(path):
        raise MapError(f"unsafe catalog path: {path!r}")
    if kind == "tree":
        if not any(p.startswith(path + "/") for p in entries):
            raise MapError(f"catalog tree has no indexed members: {path!r}")
        return
    if kind != "blob":
        raise MapError(f"unknown sourceKind: {kind!r}")
    entry = entries.get(path)
    if not entry or entry["node_kind"] != "file" or entry["git_mode"] not in {"100644", "100755"}:
        raise MapError(f"catalog path is not an indexed regular file: {path!r}")


def validate_catalog(guide, entries):
    by_path = {entry["path"]: entry for entry in entries}
    groups = catalog_ids(list_field(guide, "groups"), "group")
    ids = catalog_ids(list_field(guide, "modules"), "module")
    aliases = set()
    for module in guide["modules"]:
        for key in ("label", "purpose", "owns", "boundary", "sourcePath", "family", "group"):
            nonempty(module, key)
        if module["family"] not in {"custom", "sdk-ibc"}:
            raise MapError(f"unknown family for {module['id']!r}")
        if module["group"] not in groups:
            raise MapError(f"unknown group for {module['id']!r}")
        catalog_path(module["sourcePath"], by_path)
        for alias in module_aliases(module):
            if alias in aliases or alias in ids:
                raise MapError(f"ambiguous module alias: {alias!r}")
            aliases.add(alias)
        for connection in list_field(module, "connections"):
            if nonempty(connection, "module") not in ids:
                raise MapError(f"unknown connection target for {module['id']!r}")
            nonempty(connection, "why")
    for utility in list_field(guide, "sharedUtilities"):
        nonempty(utility, "label")
        nonempty(utility, "text")
        catalog_path(nonempty(utility, "sourcePath"), by_path, nonempty(utility, "sourceKind"))
    for step in list_field(guide, "workflow") if "workflow" in guide else []:
        if any(module_id not in ids for module_id in list_field(step, "moduleIds", str)):
            raise MapError("unknown workflow module reference")
        link = step.get("link", {})
        if not isinstance(link, dict):
            raise MapError("catalog workflow link must be an object")
        if "path" in link:
            catalog_path(nonempty(link, "path").split("#", 1)[0], by_path)


def module_aliases(module):
    # Catalog-derived, explicitly enumerated in export/module output; never fuzzy.
    parts = module["sourcePath"].split("/")
    return ["/".join(parts[:2])] if module["family"] == "custom" and parts[0] == "x" else []


def select_module(census, selector):
    for module in census["modules"]:
        if selector == module["id"] or selector in module["aliases"]:
            return module
    raise MapError(f"unknown exact module ID or alias: {selector!r}")


def select_path(census, selector):
    for record in census["files"]:
        if record["path"] == selector:
            return {**record, "modules": [module for module in census["modules"]
                                          if module["id"] in record["module_ids"]]}
    raise MapError(f"unknown exact indexed path: {selector!r}")


def associate(record, modules, utilities):
    path = record["path"]
    related = [module for module in modules if path == module["sourcePath"] or
               any(path.startswith(prefix + "/") for alias in module["aliases"]
                   for prefix in (alias, "proto/zerone/" + alias.split("/")[1]))]
    shared = [utility["sourcePath"] for utility in utilities
              if path == utility["sourcePath"] or
              (utility["sourceKind"] == "tree" and path.startswith(utility["sourcePath"] + "/"))]
    area = record["area"]
    if len(related) == 1 and related[0]["family"] == "custom":
        area = "module:" + related[0]["id"]
    elif shared:
        area = "shared-utilities"
    tags = {record["kind"], area}
    for module in related:
        tags.update(("module:" + module["id"], "family:" + module["family"], "group:" + module["group"]))
    return {**record, "area": area, "tags": sorted(tags),
            "module_ids": [module["id"] for module in related], "shared_utility_paths": sorted(shared)}


def find(census, query, limit):
    needle = query.casefold()
    modules = {module["id"]: module for module in census["modules"]}
    matches = []
    for record in census["files"]:
        metadata = [record["path"], record["purpose"], record["area"], record["kind"], *record["tags"]]
        for module_id in record["module_ids"]:
            module = modules[module_id]
            metadata.extend(module[key] for key in ("id", "label", "purpose", "family", "group"))
        if needle in "\n".join(metadata).casefold():
            matches.append(record)
    returned = matches[:limit]
    return {"query": query, "limit": limit, "total": len(matches), "returned": len(returned), "files": returned}


def validate_coverage(entries, files):
    expected = {entry["path"] for entry in entries}
    actual = [record["path"] for record in files]
    if len(actual) != len(set(actual)) or set(actual) != expected:
        raise MapError("file coverage differs from the Git index census")
    for record in files:
        for field in ("purpose", "purpose_basis", "kind", "area", "inspection", "tags"):
            if not record.get(field):
                raise MapError(f"missing {field} label for {record['path']!r}")


def export(root):
    entries = indexed_paths(root)
    guide = read_catalog(root, entries)
    validate_catalog(guide, entries)
    modules = [{**module, "aliases": module_aliases(module), "purpose_basis": "reviewed-catalog",
                "tags": ["registered-source", module["family"], module["group"]],
                "purpose_source": {"path": CATALOG, "module_id": module["id"]}}
               for module in sorted(guide["modules"], key=lambda module: module["id"])]
    files = [associate(label(entry), modules, guide["sharedUtilities"]) for entry in entries]
    validate_coverage(entries, files)
    return {"schema_version": 1, "scope": SCOPE, "content_reads": [CATALOG],
            "total": len(files), "files": files, "catalog": CATALOG,
            "modules": modules, "sharedUtilities": guide["sharedUtilities"]}


def summary(census):
    return {"total": census["total"], "module_total": len(census["modules"]),
            "scope": SCOPE,
            "content_reads": [CATALOG],
            "review_needed": sum(record["kind"] == "review-needed" for record in census["files"]),
            "kinds": dict(Counter(record["kind"] for record in census["files"])),
            "areas": dict(Counter(record["area"] for record in census["files"]))}


def check(census):
    known = {alias for module in census["modules"] for alias in module["aliases"]}
    known.update("/".join(utility["sourcePath"].split("/")[:2])
                 for utility in census["sharedUtilities"] if utility["sourcePath"].startswith("x/"))
    observed = {"/".join(record["path"].split("/")[:2]) for record in census["files"]
                if record["path"].startswith("x/") and len(record["path"].split("/")) > 2}
    if observed - known:
        raise MapError("uncatalogued x/ area (review needed): " + ", ".join(sorted(observed - known)))
    return {**summary(census), "status": "ok", "runtime_registration_checked": False,
            "document_anchors_checked": False,
            "runtime_authority": "go test ./app -run '^TestModuleGuideCoversRegisteredApplication$' -count=1"}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).absolute().parent.parent)
    commands = parser.add_subparsers(dest="command", required=True)
    commands.add_parser("export", help="full deterministic JSON census on stdout; no writes")
    commands.add_parser("summary", help="counts, scope and review-needed total")
    commands.add_parser("check", help="validate metadata coverage and catalog references; no writes")
    commands.add_parser("module", help="exact catalog ID or enumerated path alias").add_argument("selector")
    commands.add_parser("show", help="exact indexed repository-relative path").add_argument("path")
    search = commands.add_parser("find", help="case-insensitive metadata substring search")
    search.add_argument("query")
    search.add_argument("--limit", type=int, default=20)
    args = parser.parse_args(argv)
    if args.command == "find":
        if not args.query.strip():
            parser.error("find requires a nonempty query")
        if not 1 <= args.limit <= 200:
            parser.error("--limit must be between 1 and 200")
    try:
        result = export(args.root)
        if args.command == "module":
            result = select_module(result, args.selector)
        elif args.command == "show":
            result = select_path(result, args.path)
        elif args.command == "find":
            result = find(result, args.query, args.limit)
        elif args.command == "summary":
            result = summary(result)
        elif args.command == "check":
            result = check(result)
    except (MapError, OSError) as error:
        print(f"source-map: {error}", file=sys.stderr)
        return 1
    print(json.dumps(result, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

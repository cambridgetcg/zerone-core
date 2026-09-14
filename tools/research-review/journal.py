#!/usr/bin/env python3
"""Bounded, local research journal. Hashes attest byte consistency, not truth.

Store layout: journal.jsonl and .lock in an explicitly created directory.
load_export accepts either a store directory or a standalone exported JSON file.
All views retain competing assertions; supersedes never transfers an assessment.
Only append_records assigns acquisition timestamps to persistent records.
"""

import argparse
import contextlib
import datetime as dt
import fcntl
import hashlib
import ipaddress
import json
import os
from pathlib import Path
import re
import stat
import sys
import uuid
from urllib.parse import urlsplit


JOURNAL_SCHEMA = "zerone-research-journal/v1"
EXPORT_SCHEMA = "zerone-research-export/v1"
MAX_BYTES = 8 * 1024 * 1024
MAX_ENTRIES = 1000
MAX_RECORD_BYTES = 64 * 1024
MAX_IMPACT_PATHS = 4096
MAX_IMPACT_DEPTH = 64
MAX_IMPACT_WORK = 16384
ID_RE = re.compile(r"[a-z0-9][a-z0-9:._-]{0,127}\Z", re.ASCII)
HASH_RE = re.compile(r"[0-9a-f]{64}\Z", re.ASCII)
DATE_RE = re.compile(r"[0-9]{4}-[0-9]{2}-[0-9]{2}\Z", re.ASCII)
TIME_RE = re.compile(r"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{6}Z\Z", re.ASCII)
DOI_RE = re.compile(r"10\.[0-9]{4,9}/[^\s]+\Z")
COMMON = {"id", "kind", "title", "summary", "attributed_to", "occurred_on", "evidence"}
FIELDS = {
    "contribution": {"contribution_type", "doi"},
    "relation": {"from_id", "to_id", "relation_type"},
    "concern": {"target_id", "category", "notice_type", "provider"},
    "assessment": {"target_id", "disposition"},
}
CONTRIBUTIONS = {"source", "question", "claim", "method", "experiment", "finding", "review"}
RELATIONS = {"supports", "requires", "inspired-by", "uses-input", "refines", "cites"}
CONCERNS = {"publisher-notice", "reported-concern", "metadata-conflict", "artifact-discrepancy"}
DISPOSITIONS = {"needs-review", "notice-checked", "disputed", "addressed", "inconclusive"}


class JournalError(ValueError):
    """Refused malformed, unavailable or excessive journal work."""


def canonical(value):
    """The shared Python/browser hash encoding; no Unicode normalization."""
    try:
        return json.dumps(value, ensure_ascii=False, sort_keys=True,
                          separators=(",", ":"), allow_nan=False).encode("utf-8")
    except (ValueError, TypeError, UnicodeError, RecursionError) as exc:
        raise JournalError("value cannot be encoded as canonical JSON") from exc


def _hash(value):
    return hashlib.sha256(canonical(value)).hexdigest()


def _pairs(pairs):
    result = {}
    for key, value in pairs:
        if key in result:
            raise JournalError("duplicate JSON key")
        result[key] = value
    return result


def _bad_constant(_value):
    raise JournalError("nonfinite JSON number")


def parse_json(raw, limit=MAX_BYTES):
    if not isinstance(raw, bytes) or len(raw) > limit:
        raise JournalError("JSON input exceeds byte limit")
    try:
        return json.loads(raw.decode("utf-8"), object_pairs_hook=_pairs,
                          parse_constant=_bad_constant)
    except (UnicodeError, ValueError, RecursionError) as exc:
        raise JournalError("invalid JSON: " + str(exc)) from exc


def _keys(value, required, optional=frozenset()):
    if not isinstance(value, dict) or set(value) - required - optional or required - set(value):
        raise JournalError("missing or unsupported fields")


def _text(value, label, minimum=0, maximum=None, byte_limit=None):
    if not isinstance(value, str):
        raise JournalError(label + " must be text")
    try:
        raw = value.encode("utf-8")
    except UnicodeError as exc:
        raise JournalError(label + " must be valid Unicode") from exc
    if len(value) < minimum or (maximum is not None and len(value) > maximum):
        raise JournalError(label + " character bound exceeded")
    if byte_limit is not None and len(raw) > byte_limit:
        raise JournalError(label + " byte bound exceeded")
    return value


def _one_of(value, allowed, label):
    if not isinstance(value, str) or value not in allowed:
        raise JournalError("unsupported " + label)


def _identifier(value):
    if not isinstance(value, str) or not ID_RE.fullmatch(value):
        raise JournalError("invalid record ID")
    return value


def _digest(value):
    if not isinstance(value, str) or not HASH_RE.fullmatch(value):
        raise JournalError("invalid SHA256")


def _date(value):
    if value is None:
        return
    if not isinstance(value, str) or not DATE_RE.fullmatch(value):
        raise JournalError("date must be null or YYYY-MM-DD")
    try:
        dt.date.fromisoformat(value)
    except ValueError as exc:
        raise JournalError("invalid Gregorian date") from exc


def _timestamp(value):
    if not isinstance(value, str) or not TIME_RE.fullmatch(value):
        raise JournalError("timestamp must use UTC with six fractional digits")
    try:
        dt.datetime.fromisoformat(value[:-1] + "+00:00")
    except ValueError as exc:
        raise JournalError("invalid UTC timestamp") from exc
    return value


def _now():
    return dt.datetime.now(dt.timezone.utc).isoformat(timespec="microseconds").replace("+00:00", "Z")


def _url(value):
    _text(value, "evidence URL", minimum=1, maximum=2048)
    if any(c.isspace() or ord(c) < 32 or ord(c) == 127 or c == "\\" for c in value):
        raise JournalError("invalid evidence URL characters")
    try:
        parsed = urlsplit(value)
        port = parsed.port
        if (parsed.scheme != "https" or not parsed.hostname or "@" in parsed.netloc
                or parsed.netloc.endswith(":") or (port is not None and not 0 <= port <= 65535)):
            raise ValueError("HTTPS without credentials required")
        if parsed.netloc.startswith("["):
            if not re.fullmatch(r"\[[0-9A-Fa-f:.]+\](?::[0-9]+)?", parsed.netloc):
                raise ValueError("invalid bracketed IPv6 authority")
            ipaddress.IPv6Address(parsed.hostname)
    except ValueError as exc:
        raise JournalError("invalid HTTPS evidence URL") from exc


def _reference(value, earlier, kinds):
    _identifier(value)
    if value not in earlier or earlier[value]["kind"] not in kinds:
        raise JournalError("reference must name an earlier record of the required kind")


def validate_record(record, earlier):
    """Validate a complete record against the already acquired records by ID."""
    if not isinstance(record, dict):
        raise JournalError("record must be an object")
    kind = record.get("kind")
    _one_of(kind, FIELDS, "record kind")
    _keys(record, COMMON | FIELDS[kind], {"supersedes"})
    rid = _identifier(record["id"])
    if rid in earlier:
        raise JournalError("record IDs are immutable and unique")
    _text(record["title"], "title", 1, 300)
    _text(record["summary"], "summary", 1, byte_limit=8192)
    _text(record["attributed_to"], "attributed_to", 1, 200)
    _date(record["occurred_on"])
    evidence = record["evidence"]
    if not isinstance(evidence, list) or len(evidence) > 16:
        raise JournalError("evidence must contain at most 16 entries")
    for item in evidence:
        _keys(item, {"label", "url", "sha256"})
        _text(item["label"], "evidence label", 1, 200)
        if item["url"] is None and item["sha256"] is None:
            raise JournalError("evidence needs a URL or SHA256")
        if item["url"] is not None:
            _url(item["url"])
        if item["sha256"] is not None:
            _digest(item["sha256"])
    if "supersedes" in record:
        _reference(record["supersedes"], earlier, {kind})
    if kind == "contribution":
        _one_of(record["contribution_type"], CONTRIBUTIONS, "contribution type")
        doi = record["doi"]
        if doi is not None:
            _text(doi, "DOI", 1)
            if (doi != doi.lower() or not DOI_RE.fullmatch(doi)
                    or any(ord(c) < 32 or ord(c) == 127 for c in doi)):
                raise JournalError("DOI must be a normalized lowercase identifier")
    elif kind == "relation":
        _one_of(record["relation_type"], RELATIONS, "relation type")
        _reference(record["from_id"], earlier, {"contribution"})
        _reference(record["to_id"], earlier, {"contribution"})
        if record["from_id"] == record["to_id"]:
            raise JournalError("self relations are not allowed")
    elif kind == "concern":
        _reference(record["target_id"], earlier, {"contribution"})
        _one_of(record["category"], CONCERNS, "concern category")
        _text(record["provider"], "provider", 1, 200)
        if record["notice_type"] is not None:
            _text(record["notice_type"], "notice_type", minimum=1, maximum=100)
    else:
        _reference(record["target_id"], earlier, {"concern", "relation"})
        _one_of(record["disposition"], DISPOSITIONS, "assessment disposition")
    if len(canonical(record)) > MAX_RECORD_BYTES:
        raise JournalError("single record exceeds 64 KiB")


def _validate_header(header):
    _keys(header, {"schema", "collection_id", "created_at"})
    if header["schema"] != JOURNAL_SCHEMA:
        raise JournalError("unsupported journal schema")
    try:
        if str(uuid.UUID(header["collection_id"])) != header["collection_id"]:
            raise ValueError("noncanonical UUID")
    except (ValueError, TypeError, AttributeError) as exc:
        raise JournalError("collection_id must be a canonical lowercase UUID") from exc
    _timestamp(header["created_at"])


def _journal_bytes(export):
    return b"\n".join(canonical(row) for row in [export["header"], *export["entries"]]) + b"\n"


def validate_export(export):
    """Validate the complete object before selecting any historical subset."""
    _keys(export, {"schema", "header", "entries"})
    if export["schema"] != EXPORT_SCHEMA:
        raise JournalError("unsupported export schema")
    _validate_header(export["header"])
    entries = export["entries"]
    if not isinstance(entries, list) or len(entries) > MAX_ENTRIES:
        raise JournalError("journal must contain at most 1000 entries")
    previous = _hash(export["header"])
    timestamp = export["header"]["created_at"]
    earlier = {}
    for sequence, row in enumerate(entries, 1):
        _keys(row, {"sequence", "recorded_at", "previous_sha256", "record", "sha256"})
        if type(row["sequence"]) is not int or row["sequence"] != sequence:
            raise JournalError("entry sequence must be contiguous integers from 1")
        if _timestamp(row["recorded_at"]) < timestamp:
            raise JournalError("recorded timestamps must be nondecreasing")
        timestamp = row["recorded_at"]
        _digest(row["previous_sha256"])
        _digest(row["sha256"])
        if row["previous_sha256"] != previous:
            raise JournalError("broken previous-hash linkage")
        if _hash({k: v for k, v in row.items() if k != "sha256"}) != row["sha256"]:
            raise JournalError("record hash mismatch")
        validate_record(row["record"], earlier)
        earlier[row["record"]["id"]] = row["record"]
        previous = row["sha256"]
    if len(canonical(export)) > MAX_BYTES or len(_journal_bytes(export)) > MAX_BYTES:
        raise JournalError("journal/export exceeds 8 MiB")
    return export


def make_export(records, *, collection_id=None, created_at=None, recorded_at=None):
    """Pure constructor for deterministic fixtures; never writes a store.

    Predetermined timestamps describe fixture data, not acquired live evidence.
    Persistent callers must use create_store/append_records instead.
    """
    if not isinstance(records, list) or len(records) > MAX_ENTRIES:
        raise JournalError("records must be a list of at most 1000 entries")
    created_at = _now() if created_at is None else created_at
    header = {"schema": JOURNAL_SCHEMA, "collection_id": str(uuid.uuid4()) if collection_id is None else collection_id,
              "created_at": created_at}
    _validate_header(header)
    if recorded_at is None:
        recorded_at = [created_at] * len(records)
    if not isinstance(recorded_at, list) or len(recorded_at) != len(records):
        raise JournalError("recorded_at must contain one timestamp per fixture record")
    export = {"schema": EXPORT_SCHEMA, "header": header, "entries": []}
    previous = _hash(header)
    for sequence, (record, timestamp) in enumerate(zip(records, recorded_at), 1):
        row = {"sequence": sequence, "recorded_at": timestamp,
               "previous_sha256": previous, "record": record}
        row["sha256"] = _hash(row)
        export["entries"].append(row)
        previous = row["sha256"]
    validate_export(export)
    return parse_json(canonical(export))


def _path_parts(path):
    value = Path(path)
    if ".." in value.parts:
        raise JournalError("parent traversal is not allowed")
    return Path(os.path.abspath(value)).parts


def _open_dir(path):
    """Walk using directory descriptors, never following a symlink component."""
    parts = _path_parts(path)
    fd = os.open(parts[0], os.O_RDONLY | os.O_DIRECTORY | os.O_CLOEXEC)
    try:
        for part in parts[1:]:
            child = os.open(part, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW | os.O_CLOEXEC, dir_fd=fd)
            os.close(fd)
            fd = child
        return fd
    except BaseException:
        os.close(fd)
        raise


def _regular(fd):
    info = os.fstat(fd)
    if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1:
        raise JournalError("file must be a regular file with one link")
    return info


def _read_fd(fd, limit=MAX_BYTES):
    info = _regular(fd)
    if info.st_size > limit:
        raise JournalError("file exceeds byte bound")
    chunks, total = [], 0
    while True:
        chunk = os.read(fd, min(65536, limit + 1 - total))
        if not chunk:
            break
        chunks.append(chunk)
        total += len(chunk)
        if total > limit:
            raise JournalError("file exceeds byte bound")
    return b"".join(chunks)


def _open_file(name, directory, flags=os.O_RDONLY):
    fd = os.open(name, flags | os.O_NOFOLLOW | os.O_NONBLOCK | os.O_CLOEXEC, dir_fd=directory)
    try:
        _regular(fd)
        return fd
    except BaseException:
        os.close(fd)
        raise


def read_json_file(path, limit=MAX_BYTES):
    """Bounded no-symlink JSON input, including scanner/CLI input files."""
    value = Path(path)
    try:
        directory = _open_dir(value.parent)
        try:
            fd = _open_file(value.name, directory)
            try:
                return parse_json(_read_fd(fd, limit), limit)
            finally:
                os.close(fd)
        finally:
            os.close(directory)
    except OSError as exc:
        raise JournalError("cannot read regular JSON input: " + str(exc)) from exc


@contextlib.contextmanager
def _locked(store_path, exclusive=False):
    directory = lock = None
    try:
        directory = _open_dir(store_path)
        lock = _open_file(".lock", directory, os.O_RDWR)
        fcntl.flock(lock, fcntl.LOCK_EX if exclusive else fcntl.LOCK_SH)
        yield directory
    except OSError as exc:
        raise JournalError("journal filesystem operation failed: " + str(exc)) from exc
    finally:
        if lock is not None:
            os.close(lock)
        if directory is not None:
            os.close(directory)


def _load_locked(directory):
    fd = _open_file("journal.jsonl", directory)
    try:
        info = os.fstat(fd)
        raw = _read_fd(fd)
    finally:
        os.close(fd)
    if not raw.endswith(b"\n"):
        raise JournalError("partial journal: final newline is missing")
    lines = raw[:-1].split(b"\n")
    if len(lines) > MAX_ENTRIES + 1:
        raise JournalError("too many journal rows")
    values = [parse_json(line) for line in lines]
    if any(canonical(value) != line for value, line in zip(values, lines)):
        raise JournalError("journal rows must use canonical JSON bytes")
    export = {"schema": EXPORT_SCHEMA, "header": values[0], "entries": values[1:]}
    validate_export(export)
    return export, raw, info


def _write_all(fd, raw):
    offset = 0
    while offset < len(raw):
        wrote = os.write(fd, raw[offset:])
        if wrote <= 0:
            raise JournalError("short journal write")
        offset += wrote
    os.fsync(fd)


def _new_file(directory, name, raw):
    fd = os.open(name, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW | os.O_CLOEXEC,
                 0o600, dir_fd=directory)
    try:
        _write_all(fd, raw)
    finally:
        os.close(fd)


def create_store(path):
    """Create one new private directory; existing directories are never reused."""
    path = Path(path)
    directory = parent = None
    try:
        parent = _open_dir(path.parent)
        os.mkdir(path.name, 0o700, dir_fd=parent)
        directory = os.open(path.name, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW | os.O_CLOEXEC, dir_fd=parent)
        export = make_export([])
        _new_file(directory, ".lock", b"")
        _new_file(directory, "journal.jsonl", _journal_bytes(export))
        os.fsync(directory)
        os.fsync(parent)
        return export
    except OSError as exc:
        # A partial new directory is retained for inspection, never silently reset.
        raise JournalError("cannot create a fresh journal store: " + str(exc)) from exc
    finally:
        if directory is not None:
            os.close(directory)
        if parent is not None:
            os.close(parent)


def export_store(store_path):
    with _locked(store_path) as directory:
        return _load_locked(directory)[0]


def load_export(path):
    try:
        info = os.lstat(path)
    except OSError as exc:
        raise JournalError("cannot inspect journal/export input: " + str(exc)) from exc
    if stat.S_ISDIR(info.st_mode):
        return export_store(path)
    if not stat.S_ISREG(info.st_mode):
        raise JournalError("input must be a regular export file or journal directory")
    return validate_export(read_json_file(path))


def append_records(store_path, records):
    """Validate the whole batch, then replace JSONL atomically under flock.

    Prior row bytes are retained verbatim. No caller-supplied acquisition times.
    All participating writers/readers must use this API and preserve the lock.
    If directory fsync fails after replacement, the complete batch may already
    be visible; inspect the journal before retrying its immutable IDs.
    """
    if not isinstance(records, list) or not records or len(records) > MAX_ENTRIES:
        raise JournalError("append requires a nonempty list of at most 1000 records")
    # Freeze the caller's objects before locking, including encoding/size checks.
    records = parse_json(canonical(records))
    with _locked(store_path, exclusive=True) as directory:
        export, old_raw, old_info = _load_locked(directory)
        if len(export["entries"]) + len(records) > MAX_ENTRIES:
            raise JournalError("journal would exceed 1000 entries")
        now = _now()
        prior_time = export["entries"][-1]["recorded_at"] if export["entries"] else export["header"]["created_at"]
        if now < prior_time:
            raise JournalError("local clock precedes the last recorded timestamp")
        previous = export["entries"][-1]["sha256"] if export["entries"] else _hash(export["header"])
        new_rows = []
        for record in records:
            row = {"sequence": len(export["entries"]) + 1, "recorded_at": now,
                   "previous_sha256": previous, "record": record}
            row["sha256"] = _hash(row)
            export["entries"].append(row)
            new_rows.append(row)
            previous = row["sha256"]
        validate_export(export)
        raw = old_raw + b"".join(canonical(row) + b"\n" for row in new_rows)
        temporary = ".journal-" + uuid.uuid4().hex + ".tmp"
        owned_temporary = False
        try:
            fd = os.open(temporary, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW | os.O_CLOEXEC,
                         0o600, dir_fd=directory)
            owned_temporary = True
            try:
                _write_all(fd, raw)
            finally:
                os.close(fd)
            current = os.stat("journal.jsonl", dir_fd=directory, follow_symlinks=False)
            if (current.st_dev, current.st_ino, current.st_size, current.st_mtime_ns) != (
                    old_info.st_dev, old_info.st_ino, old_info.st_size, old_info.st_mtime_ns):
                raise JournalError("journal changed outside its lock")
            os.replace(temporary, "journal.jsonl", src_dir_fd=directory, dst_dir_fd=directory)
            os.fsync(directory)
        finally:
            if owned_temporary:
                try:
                    os.unlink(temporary, dir_fd=directory)
                except FileNotFoundError:
                    pass
        return export


def select_snapshot(export_dict, through=None):
    validate_export(export_dict)
    count = len(export_dict["entries"])
    if through is None:
        through = count
    if type(through) is not int or not 0 <= through <= count:
        raise JournalError("cutoff must be an integer between 0 and the last sequence")
    result = {"schema": EXPORT_SCHEMA, "header": export_dict["header"],
              "entries": export_dict["entries"][:through]}
    return parse_json(canonical(result))


def impact(export_dict, target_id, through=None):
    selected = select_snapshot(export_dict, through)
    records = {row["record"]["id"]: row["record"] for row in selected["entries"]}
    _reference(target_id, records, {"contribution"})
    adjacency = {}
    for record in records.values():
        if record["kind"] != "relation":
            continue
        kind = record["relation_type"]
        if kind == "supports":
            source, target, propagation = record["from_id"], record["to_id"], "support"
        elif kind in ("requires", "uses-input"):
            source, target = record["to_id"], record["from_id"]
            propagation = "dependency" if kind == "requires" else "input"
        else:
            continue
        adjacency.setdefault(source, []).append((target, record, propagation))
    result = {"schema": "zerone-research-impact/v1", "collection_id": selected["header"]["collection_id"],
              "through": len(selected["entries"]), "target_id": target_id,
              "interpretation": "possible-reassessment-not-falsity", "paths": [], "truncated": False}
    queue = [([target_id], [], [], [])]
    cursor = work = size = 0
    while cursor < len(queue):
        nodes, relations, propagation, actors = queue[cursor]
        cursor += 1
        for target, relation, role in adjacency.get(nodes[-1], []):
            work += 1
            if work > MAX_IMPACT_WORK:
                raise JournalError("impact traversal work bound exceeded")
            if target in nodes:
                continue  # All simple paths; a cycle is not evidence of falsity.
            if len(relations) >= MAX_IMPACT_DEPTH or len(result["paths"]) >= MAX_IMPACT_PATHS:
                raise JournalError("impact path bound exceeded")
            next_path = (nodes + [target], relations + [relation["id"]], propagation + [role], actors + [relation["attributed_to"]])
            path = dict(zip(("record_ids", "relation_ids", "propagation", "attributed_to"), next_path))
            size += len(canonical(path))
            if size > MAX_BYTES:
                raise JournalError("impact output byte bound exceeded")
            result["paths"].append(path)
            queue.append(next_path)
    if len(canonical(result)) > MAX_BYTES:
        raise JournalError("impact output byte bound exceeded")
    return result


def _write_output(path, value):
    path = Path(path)
    parent = None
    try:
        raw = canonical(value) + b"\n"
        if len(raw) > MAX_BYTES:
            raise JournalError("export output exceeds 8 MiB")
        parent = _open_dir(path.parent)
        _new_file(parent, path.name, raw)
        os.fsync(parent)
    except OSError as exc:
        raise JournalError("output must be a new regular file: " + str(exc)) from exc
    finally:
        if parent is not None:
            os.close(parent)


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    commands.add_parser("init").add_argument("store")
    append = commands.add_parser("append")
    append.add_argument("store")
    append.add_argument("record_json")
    export = commands.add_parser("export")
    export.add_argument("store")
    export.add_argument("--output", required=True)
    for name in ("inspect", "impact"):
        command = commands.add_parser(name)
        command.add_argument("store")
        if name == "impact":
            command.add_argument("target")
        command.add_argument("--through", type=int)
    args = parser.parse_args(argv)
    try:
        if args.command == "init":
            result = create_store(args.store)
        elif args.command == "append":
            records = read_json_file(args.record_json)
            result = append_records(args.store, records if isinstance(records, list) else [records])
        elif args.command == "export":
            result = export_store(args.store)
            _write_output(args.output, result)
            result = {"output": args.output, "entries": len(result["entries"])}
        elif args.command == "inspect":
            result = select_snapshot(load_export(args.store), args.through)
        else:
            result = impact(load_export(args.store), args.target, args.through)
        print(canonical(result).decode("utf-8"))
        return 0
    except JournalError as exc:
        print("research journal refused: " + str(exc), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())

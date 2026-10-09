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
import importlib.util
import os
from pathlib import Path
import stat
import sys
import uuid


# Load the exact sibling even when imported by file path from another working
# directory (external-worker does this). Never mutate sys.path or search for a
# same-named third-party module. All model rules live in this one dependency.
_core_spec = importlib.util.spec_from_file_location(
    "zerone_research_journal_core", Path(__file__).resolve().with_name("journal_core.py"))
if _core_spec is None or _core_spec.loader is None:
    raise ImportError("cannot load the research journal core")
core = importlib.util.module_from_spec(_core_spec)
_core_spec.loader.exec_module(core)

# Preserve the established journal import surface for existing callers.
JOURNAL_SCHEMA = core.JOURNAL_SCHEMA
EXPORT_SCHEMA = core.EXPORT_SCHEMA
MAX_BYTES = core.MAX_BYTES
MAX_ENTRIES = core.MAX_ENTRIES
MAX_RECORD_BYTES = core.MAX_RECORD_BYTES
MAX_IMPACT_PATHS = core.MAX_IMPACT_PATHS
MAX_IMPACT_DEPTH = core.MAX_IMPACT_DEPTH
MAX_IMPACT_WORK = core.MAX_IMPACT_WORK
ID_RE = core.ID_RE
HASH_RE = core.HASH_RE
DATE_RE = core.DATE_RE
TIME_RE = core.TIME_RE
DOI_RE = core.DOI_RE
COMMON = core.COMMON
FIELDS = core.FIELDS
CONTRIBUTIONS = core.CONTRIBUTIONS
RELATIONS = core.RELATIONS
CONCERNS = core.CONCERNS
DISPOSITIONS = core.DISPOSITIONS
JournalError = core.JournalError
canonical = core.canonical
_hash = core._hash
_pairs = core._pairs
_bad_constant = core._bad_constant
parse_json = core.parse_json
_keys = core._keys
_text = core._text
_one_of = core._one_of
_identifier = core._identifier
_digest = core._digest
_date = core._date
_timestamp = core._timestamp
_url = core._url
_reference = core._reference
validate_record = core.validate_record
_validate_header = core._validate_header
_journal_bytes = core._journal_bytes
validate_export = core.validate_export
_history_summary = core._history_summary
compare_exports = core.compare_exports
select_snapshot = core.select_snapshot
impact = core.impact


def _now():
    return dt.datetime.now(dt.timezone.utc).isoformat(timespec="microseconds").replace("+00:00", "Z")


def make_export(records, *, collection_id=None, created_at=None, recorded_at=None):
    """Compatibility constructor; this adapter owns default clock and identity.

    The standalone core requires both values explicitly. Supplied timestamps
    describe fixtures, not acquired live evidence. This helper writes no store.
    """
    if not isinstance(records, list) or len(records) > MAX_ENTRIES:
        raise JournalError("records must be a list of at most 1000 entries")
    created_at = _now() if created_at is None else created_at
    collection_id = str(uuid.uuid4()) if collection_id is None else collection_id
    return core.make_export(records, collection_id=collection_id,
                            created_at=created_at, recorded_at=recorded_at)


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


def fork_store(source_path, store_path, expected_sha256):
    """Copy an exact validated export into a fresh store without rebuilding rows.

    The source is read once and checked before creating any output. Its original
    bytes are retained separately from canonical JSONL; export whitespace may
    differ even when the complete history is equal. A fork confers no authority
    over its lineage and authenticates no attributed author or operator.
    """
    _digest(expected_sha256)
    source = Path(source_path)
    directory = fd = None
    try:
        directory = _open_dir(source.parent)
        fd = _open_file(source.name, directory)
        raw = _read_fd(fd)
    except OSError as exc:
        raise JournalError("cannot read regular fork source: " + str(exc)) from exc
    finally:
        if fd is not None:
            os.close(fd)
        if directory is not None:
            os.close(directory)
    if hashlib.sha256(raw).hexdigest() != expected_sha256:
        raise JournalError("fork source does not match expected SHA256")
    export = validate_export(parse_json(raw))
    journal_raw = _journal_bytes(export)
    destination = Path(store_path)
    directory = parent = None
    try:
        parent = _open_dir(destination.parent)
        os.mkdir(destination.name, 0o700, dir_fd=parent)
        directory = os.open(destination.name, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW | os.O_CLOEXEC,
                            dir_fd=parent)
        _new_file(directory, ".lock", b"")
        _new_file(directory, "fork-source.json", raw)
        _new_file(directory, "journal.jsonl", journal_raw)
        os.fsync(directory)
        os.fsync(parent)
    except OSError as exc:
        # As for init, failed fresh stores remain inspectable; never reset/reuse.
        raise JournalError("cannot create a fresh fork store: " + str(exc)) from exc
    finally:
        if directory is not None:
            os.close(directory)
        if parent is not None:
            os.close(parent)
    return {"schema": "zerone-research-fork/v1", **_history_summary(export),
            "source_sha256": expected_sha256,
            "scope": "Local history copy; no authentication or branch authority."}


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
    fork = commands.add_parser("fork")
    fork.add_argument("source")
    fork.add_argument("store")
    fork.add_argument("--expected-sha256", required=True)
    compare = commands.add_parser("compare")
    compare.add_argument("left")
    compare.add_argument("right")
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
        elif args.command == "fork":
            result = fork_store(args.source, args.store, args.expected_sha256)
        elif args.command == "compare":
            result = compare_exports(load_export(args.left), load_export(args.right))
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

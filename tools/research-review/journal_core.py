"""Storage-free research journal model, usable as a single stdlib-only module.

Owns the existing bounded record/hash/history/impact contracts. Every operation
uses supplied data; no storage, locks, clock, randomness, network or adapters.
Hashes describe supplied bytes, not identity, truth or historical completeness.
Caller-owned identity/timestamps in make_export are not acquisition evidence.
The POSIX journal.py adapter remains responsible for persistent acquisition.
"""

import datetime as dt
import hashlib
import ipaddress
import json
import re
import uuid
from urllib.parse import urlsplit

__all__ = (
    "JournalError", "JOURNAL_SCHEMA", "EXPORT_SCHEMA", "MAX_BYTES", "MAX_ENTRIES",
    "MAX_RECORD_BYTES", "MAX_IMPACT_PATHS", "MAX_IMPACT_DEPTH", "MAX_IMPACT_WORK",
    "canonical", "parse_json", "validate_record", "validate_export", "make_export",
    "compare_exports", "select_snapshot", "impact",
)


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


def _history_summary(export):
    entries = export["entries"]
    return {"collection_id": export["header"]["collection_id"],
            "entry_count": len(entries),
            "head_sha256": entries[-1]["sha256"] if entries else _hash(export["header"])}


def compare_exports(left, right):
    """Compare fully validated histories; no merge, authority or signature claim."""
    validate_export(left)
    validate_export(right)
    result = {"schema": "zerone-research-comparison/v1", "relationship": "different-root",
              "left": _history_summary(left), "right": _history_summary(right),
              "common_prefix": None}
    if canonical(left["header"]) != canonical(right["header"]):
        return result
    count = 0
    for left_row, right_row in zip(left["entries"], right["entries"]):
        if canonical(left_row) != canonical(right_row):
            break
        count += 1
    result["common_prefix"] = {
        "entry_count": count,
        "head_sha256": left["entries"][count - 1]["sha256"] if count else _hash(left["header"])}
    left_count, right_count = len(left["entries"]), len(right["entries"])
    if count == left_count == right_count:
        result["relationship"] = "same-history"
    elif count == left_count:
        result["relationship"] = "left-prefix"
    elif count == right_count:
        result["relationship"] = "right-prefix"
    else:
        result["relationship"] = "diverged"
    return result


def make_export(records, *, collection_id, created_at, recorded_at=None):
    """Construct a detached export from caller-owned identity and timestamps.

    No clock, randomness, file storage or acquisition authority is used.
    Predetermined timestamps describe fixture data, not acquired live evidence.
    Persistent callers must use create_store/append_records instead.
    """
    if not isinstance(records, list) or len(records) > MAX_ENTRIES:
        raise JournalError("records must be a list of at most 1000 entries")
    header = {"schema": JOURNAL_SCHEMA, "collection_id": collection_id,
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

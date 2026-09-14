#!/usr/bin/env python3
"""Bounded, explicit Crossref notice lookup into a local research journal.

Stdlib only. Fetches metadata, never paper content or executable artifacts.
Returned notice assertions are leads, not determinations of scientific validity.
"""
from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import math
import os
from pathlib import Path
import re
import stat
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid

import journal

MAX_DOIS = 25
MAX_RESPONSE = 2 * 1024 * 1024
MAX_UPDATES = 100
API = "https://api.crossref.org/works/"
AGENT = "Zerone-Research-Review/1 (https://zerone.ai/research/review/)"


class ScanError(ValueError):
    pass


def canonical(value):
    return json.dumps(value, ensure_ascii=False, sort_keys=True,
                      separators=(",", ":"), allow_nan=False).encode("utf-8")


def digest(data):
    return hashlib.sha256(data).hexdigest()


def identifier(prefix, *parts):
    return prefix + ":" + digest(canonical(parts))


def normalize_doi(value):
    if not isinstance(value, str):
        raise ScanError("DOI must be a string")
    doi = value.strip().lower()
    for prefix in ("https://doi.org/", "http://doi.org/", "doi:"):
        if doi.startswith(prefix):
            doi = doi[len(prefix):]
            break
    if (len(doi) > 512 or not re.fullmatch(r"10\.\d{4,9}/[^\s]+", doi)
            or any(ord(c) < 33 or ord(c) == 127 for c in doi)):
        raise ScanError("Invalid DOI (expected 10.<registrant>/<suffix>)")
    return doi


def api_url(doi):
    return API + urllib.parse.quote(normalize_doi(doi), safe="")


def doi_url(doi):
    return "https://doi.org/" + urllib.parse.quote(doi, safe="/")


def _unique_pairs(pairs):
    value = {}
    for key, item in pairs:
        if key in value:
            raise ScanError("Provider JSON contains duplicate keys")
        value[key] = item
    return value


def _integer(value):
    if len(value) > 128:
        raise ScanError("Provider integer exceeds 128 digits")
    return int(value)


def decode_metadata(raw, doi):
    if not isinstance(raw, bytes) or len(raw) > MAX_RESPONSE:
        raise ScanError("Provider response exceeds 2 MiB")
    try:
        obj = json.loads(raw.decode("utf-8"), object_pairs_hook=_unique_pairs, parse_int=_integer,
                         parse_constant=lambda _: (_ for _ in ()).throw(ScanError("Non-finite JSON")))
        if (not isinstance(obj, dict) or obj.get("status") != "ok"
                or obj.get("message-type") != "work" or not isinstance(obj.get("message"), dict)):
            raise ScanError("Provider response is not a Crossref work")
        message = obj["message"]
        if normalize_doi(message.get("DOI")) != doi:
            raise ScanError("Provider returned a different DOI")
        pending = [obj]
        visited = 0
        while pending:
            value = pending.pop()
            visited += 1
            if visited > 100000:
                raise ScanError("Provider JSON has too many values")
            if isinstance(value, str):
                value.encode("utf-8")
            elif isinstance(value, dict):
                pending.extend(value.keys())
                pending.extend(value.values())
            elif isinstance(value, list):
                pending.extend(value)
            elif isinstance(value, float) and not math.isfinite(value):
                raise ScanError("Non-finite provider number")
        return message
    except ScanError:
        raise
    except (UnicodeError, ValueError, RecursionError) as exc:
        raise ScanError("Provider response is not bounded valid UTF-8 JSON") from exc


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise ScanError("Provider redirect refused")


def fetch_metadata(doi):
    request = urllib.request.Request(api_url(doi), headers={
        "Accept": "application/json", "User-Agent": AGENT,
        "Accept-Encoding": "identity",
    })
    try:
        with urllib.request.build_opener(NoRedirect()).open(request, timeout=20) as response:
            if response.status != 200:
                raise ScanError("Provider returned non-200 status")
            raw = response.read(MAX_RESPONSE + 1)
    except urllib.error.HTTPError as exc:
        raise ScanError("Provider HTTP " + str(exc.code)) from exc
    except (urllib.error.URLError, TimeoutError, OSError) as exc:
        # Do not put proxy credentials, private paths or network diagnostics in exports.
        raise ScanError("Provider network request failed") from exc
    return raw


def _short(value, fallback, limit):
    if not isinstance(value, str) or not value.strip():
        return fallback
    return value.strip()[:limit]


def _date(value):
    """Keep incomplete or invalid bibliographic dates unknown, never invent Jan 1."""
    try:
        parts = value["date-parts"][0]
        if len(parts) != 3 or any(type(n) is not int for n in parts):
            return None
        return dt.date(*parts).isoformat()
    except (KeyError, TypeError, IndexError, ValueError, OverflowError):
        return None


def notices(message, doi):
    """Normalize direction, preserving provider assertions and unknown type labels.

    Original.updated-by -> notice DOI. Notice.update-to -> original DOI.
    Multiple providers of an identical original/notice/type are one notice lead.
    """
    grouped = {}
    warnings = []
    for field in ("updated-by", "update-to"):
        values = message.get(field, [])
        if not isinstance(values, list) or len(values) > MAX_UPDATES:
            raise ScanError("Provider update list is malformed or exceeds 100 entries")
        for index, value in enumerate(values):
            if not isinstance(value, dict):
                warnings.append(f"{field}[{index}] is not an object; inspect raw metadata")
                continue
            try:
                other = normalize_doi(value.get("DOI"))
            except ScanError:
                warnings.append(f"{field}[{index}] has no valid linked DOI; inspect raw metadata")
                continue
            if (not isinstance(value.get("type", "unspecified"), str)
                    or not 1 <= len(value.get("type", "unspecified")) <= 100
                    or not isinstance(value.get("source", "unspecified-source"), str)
                    or not 1 <= len(value.get("source", "unspecified-source")) <= 100):
                warnings.append(f"{field}[{index}] has malformed type/source labels; inspect raw metadata")
                continue
            original, notice = (doi, other) if field == "updated-by" else (other, doi)
            label = _short(value.get("type"), "unspecified", 100)
            provider = _short(value.get("source"), "unspecified-source", 100)
            key = (original, notice, label)
            item = grouped.setdefault(key, {"original_doi": original, "notice_doi": notice,
                "type": label, "sources": [], "dates": [], "directions": []})
            if provider not in item["sources"]:
                item["sources"].append(provider)
            date = _date(value.get("updated"))
            if date not in item["dates"]:
                item["dates"].append(date)
            if field not in item["directions"]:
                item["directions"].append(field)
    for item in grouped.values():
        item["sources"].sort()
        item["dates"].sort(key=lambda n: n or "")
        item["directions"].sort()
    return [grouped[key] for key in sorted(grouped)], warnings


def save_artifact(store, raw):
    """Create immutable content-addressed evidence; refuse links and mismatched bytes."""
    folder = Path(store) / "artifacts"
    try:
        folder.mkdir(mode=0o700)
    except FileExistsError:
        if folder.is_symlink() or not folder.is_dir():
            raise ScanError("Artifacts path must be a regular directory")
    sha = digest(raw)
    name = sha + ".json"
    directory = os.open(folder, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
    scratch = ".pending-" + uuid.uuid4().hex
    created = False
    try:
        fd = os.open(scratch, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW,
                     0o600, dir_fd=directory)
        created = True
        with os.fdopen(fd, "wb") as stream:
            stream.write(raw)
            stream.flush()
            os.fsync(stream.fileno())
        try:
            os.link(scratch, name, src_dir_fd=directory, dst_dir_fd=directory, follow_symlinks=False)
        except FileExistsError:
            fd = os.open(name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=directory)
            with os.fdopen(fd, "rb") as stream:
                if not stat.S_ISREG(os.fstat(stream.fileno()).st_mode):
                    raise ScanError("Artifact is not a regular file")
                if stream.read(MAX_RESPONSE + 1) != raw:
                    raise ScanError("Existing content-addressed artifact differs")
        os.fsync(directory)
        return sha
    finally:
        try:
            if created:
                os.unlink(scratch, dir_fd=directory)
        except FileNotFoundError:
            pass
        os.close(directory)


def _record(id_, kind, title, summary, occurred_on, evidence, **extra):
    return {"id": id_, "kind": kind, "title": title, "summary": summary,
            "attributed_to": "Zerone Crossref importer (local, unsigned)",
            "occurred_on": occurred_on, "evidence": evidence, **extra}


def plan_records(export, doi, raw, observed_at, report_sha):
    """Pure conversion; fetched authors are never treated as authenticated signers."""
    prior = {entry["record"]["id"]: entry["record"] for entry in export["entries"]}
    planned = []
    evidence = [{"label": "Crossref metadata captured at " + observed_at,
                 "url": api_url(doi), "sha256": digest(raw)}]
    message = decode_metadata(raw, doi)
    updates, warnings = notices(message, doi)

    def add(record):
        existing = prior.get(record["id"])
        if existing is not None:
            identity = {"contribution": ("contribution_type", "doi"),
                        "concern": ("target_id", "notice_type", "category")}[record["kind"]]
            # Source aggregation may change category as another provider starts
            # relaying the same notice, but cannot change the notice identity.
            if record["id"].startswith("crossref-notice:"):
                identity = ("target_id", "notice_type")
            if existing["kind"] != record["kind"] or any(
                    existing.get(field) != record.get(field) for field in identity):
                raise ScanError("Importer ID is occupied by a different record")
            return
        planned.append(record)
        prior[record["id"]] = record

    def source(source_doi):
        id_ = identifier("crossref-source", source_doi)
        if id_ in prior:
            existing = prior[id_]
            if (existing["kind"] != "contribution" or existing["contribution_type"] != "source"
                    or existing["doi"] != source_doi):
                raise ScanError("Importer source ID is occupied by a different record")
            return id_
        titles = message.get("title")
        title = _short(titles[0], source_doi, 300) if source_doi == doi and isinstance(titles, list) and titles else source_doi
        summary = ("Bibliographic source imported from Crossref; no scientific claim or author signature inferred. "
                   "Full metadata and later changes remain in captured lookup artifacts.")
        if source_doi != doi:
            summary += " This DOI was referenced by a notice; its own work metadata was not fetched by this lookup."
        date = _date(message.get("published")) if source_doi == doi else None
        add(_record(id_, "contribution", title, summary, date, evidence,
                    contribution_type="source", doi=source_doi))
        return id_

    source(doi)
    by_pair = {}
    for update in updates:
        target = source(update["original_doi"])
        key = (update["original_doi"], update["notice_doi"])
        by_pair.setdefault(key, []).append(update)
        id_ = identifier("crossref-notice", *key, update["type"])
        notice_evidence = evidence + [{"label": "Linked notice DOI (content not fetched)",
                                      "url": doi_url(update["notice_doi"]), "sha256": None}]
        date = update["dates"][0] if len(update["dates"]) == 1 else None
        providers = ", ".join(update["sources"])[:1200]
        # Publisher and Retraction Watch can relay the same notice. Neither is a vote.
        summary = (f"Crossref metadata reports update type '{update['type']}' for {key[0]}, "
                   f"linked notice {key[1]}. Sources: {providers}. "
                   "This is a metadata lead. Read the notice and assess its exact scope; it does not establish fraud, "
                   "which claim is affected, or whether a scientific result is false. "
                   "Later absence from a lookup does not withdraw this recorded observation.")
        provider_label = ("Crossref: " + providers)[:200]
        previous_notice = prior.get(id_)
        add(_record(id_, "concern", ("Notice metadata: " + update["type"])[:300], summary,
                    date, notice_evidence, target_id=target,
                    category="publisher-notice" if update["sources"] == ["publisher"] else "reported-concern",
                    notice_type=update["type"], provider=provider_label))
        if previous_notice is not None and (previous_notice["occurred_on"] != date
                                             or previous_notice["provider"] != provider_label):
            add(_record(identifier("crossref-change", id_, update["dates"], update["sources"]),
                "concern", "Notice date or source attribution changed in a later lookup",
                "A later response reports a different date or source attribution for a previously recorded notice. "
                "This may reflect an additional metadata provider or a correction; it is not evidence of scientific misconduct. "
                "The first notice observation remains unchanged. Inspect both captured responses; no automatic precedence applies.",
                None, evidence + previous_notice["evidence"][:2], target_id=target,
                category="metadata-conflict", notice_type=None, provider="Crossref"))
    for (original, notice), versions in by_pair.items():
        labels = sorted({item["type"] for item in versions})
        dates = {date for item in versions for date in item["dates"] if date is not None}
        if len(labels) > 1 or len(dates) > 1:
            add(_record(identifier("crossref-conflict", original, notice, labels, sorted(dates)),
                "concern", "Notice metadata has differing labels or dates",
                "The same original/notice pair has multiple type labels or dates in this response. "
                "This may reflect an update, deposit inconsistency or stale metadata; check the publisher. "
                "No automatic status precedence or resolution is applied. Labels: " + ", ".join(labels),
                None, evidence, target_id=source(original), category="metadata-conflict",
                notice_type=None, provider="Crossref"))
    coverage = (f"Explicit DOI lookup at {observed_at}: {len(updates)} distinct original/notice/type leads returned; "
                f"{len(warnings)} malformed-link observations. "
                "This checks this work response only: no citation crawl, reverse search, paper text or publisher notice fetch. "
                "Zero returned leads means none returned by this lookup, not a clean bill of health. "
                "Previously recorded concerns remain; authorship is bibliographic metadata, not authenticated attribution.")
    add(_record(identifier("crossref-lookup", doi, observed_at, report_sha), "contribution",
        "Crossref lookup: " + doi[:270], coverage, observed_at[:10],
        evidence + [{"label": "Lookup receipt", "url": None, "sha256": report_sha}],
        contribution_type="review", doi=None))
    return planned


def scan(store, dois, fetch=fetch_metadata, pause=time.sleep):
    """Persist raw responses/receipts, then append one atomic batch per lookup.

    Errors produce an attributed coverage record, never a 'no concerns' result.
    A failed append leaves unreferenced artifacts, not a partially appended batch.
    """
    normalized = list(dict.fromkeys(normalize_doi(doi) for doi in dois))
    if not 1 <= len(normalized) <= MAX_DOIS:
        raise ScanError("Choose between 1 and 25 distinct explicit DOIs")
    journal.export_store(store)  # Validate existing journal before network or file effects.
    results = []
    for index, doi in enumerate(normalized):
        if index:
            pause(1.0)
        observed_at = dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H:%M:%S.%fZ")
        receipt = {"schema": "zerone-crossref-lookup/v1", "doi": doi, "url": api_url(doi),
                   "observed_at": observed_at, "scope": "single-work-response", "status": "ok"}
        raw = None
        try:
            raw = fetch(doi)
            # Preserve received bytes even when JSON or provider update structure is invalid.
            if not isinstance(raw, bytes) or len(raw) > MAX_RESPONSE:
                raise ScanError("Provider response exceeds 2 MiB or is not bytes")
            receipt["metadata_sha256"] = save_artifact(store, raw)
            message = decode_metadata(raw, doi)
            updates, warnings = notices(message, doi)
            receipt.update({"distinct_notice_leads": len(updates), "warnings": warnings,
                            "updates": updates})
        except ScanError as exc:
            receipt.update({"status": "lookup-failed", "error": str(exc)})
        receipt_sha = save_artifact(store, canonical(receipt))
        existing = journal.export_store(store)
        if receipt["status"] == "ok":
            records = plan_records(existing, doi, raw, observed_at, receipt_sha)
        else:
            records = [_record(identifier("crossref-failed", doi, observed_at, receipt_sha),
                "contribution", "Crossref lookup failed: " + doi[:250],
                "Lookup failed; coverage is unknown. " + receipt["error"] + ". No prior concern is cleared.",
                observed_at[:10], [{"label": "Failed lookup receipt", "url": api_url(doi), "sha256": receipt_sha}],
                contribution_type="review", doi=None)]
        journal.append_records(store, records)
        results.append({**receipt, "receipt_sha256": receipt_sha, "appended_records": len(records)})
        if receipt.get("error") == "Provider HTTP 429":
            # Respect rate limiting; caller may resume explicitly later. No hidden retry loop.
            break
    return {"schema": "zerone-crossref-scan/v1", "requested": len(normalized),
            "completed": len(results), "lookups": results,
            "interpretation": "sourced-leads-not-fraud-findings"}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("store", type=Path, help="Existing local journal (create with journal.py init)")
    parser.add_argument("dois", nargs="+", help="1–25 explicit DOIs; sends each to Crossref")
    args = parser.parse_args(argv)
    try:
        result = scan(args.store, args.dois)
        print(json.dumps(result, ensure_ascii=False, indent=2))
        return 0 if result["completed"] == result["requested"] and all(
            item["status"] == "ok" for item in result["lookups"]) else 2
    except (ScanError, journal.JournalError, OSError, ValueError) as exc:
        print("Research lookup refused: " + str(exc), file=sys.stderr)
        return 2


if __name__ == "__main__":
    sys.exit(main())

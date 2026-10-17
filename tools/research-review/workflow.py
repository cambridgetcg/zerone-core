#!/usr/bin/env python3
"""Bind a local research record to explicit development participation.

Preparation is offline. Submit invokes the participant's existing, pinned client;
this adapter never signs or broadcasts itself. Refresh only asks that client for
observations. A sidecar is a local association, not a chain inclusion proof, an
independent review, or a statement that a scientific claim is true.
"""

import argparse
import contextlib
import copy
import datetime as dt
import fcntl
import hashlib
import os
from pathlib import Path
import re
import secrets
import subprocess
import sys
import tempfile

import journal as j


SCHEMA = "zerone-research-workflow/v1"
CHAIN = "zerone-dev-1"
MAX_OBSERVATIONS = 32
MAX_PACKET_BYTES = 16 * 1024 * 1024
MAX_ROWS = 2000
CLIENT_NAMES = {"shared-claims.py", "claim-workflow.py", "local-node.py"}
ADDRESS = re.compile(r"zrn1[023456789acdefghjklmnpqrstuvwxyz]{38}\Z")
TXHASH = re.compile(r"[0-9A-F]{64}\Z")
RECEIPT_FIELDS = {"id", "action", "actor", "status", "txhash", "height", "code",
                  "gas_wanted", "gas_used", "claim_id", "round_id", "tx_bytes",
                  "review_fee_uzrn", "collateral_uzrn"}
STATUSES = {"prepared", "broadcasting", "pending", "rejected", "committed", "failed", "committed-unresolved"}
Error = j.JournalError


def digest(raw):
    return hashlib.sha256(raw).hexdigest()


def object_value(value, name):
    if not isinstance(value, dict):
        raise Error(name + " must be an object")
    return value


def text_value(value, name, limit, required=True):
    j._text(value, name, byte_limit=limit)
    if "\0" in value or (required and not value.strip()):
        raise Error(name + " must contain nonempty text without NUL")
    return value


def number(value, name, minimum=0):
    if type(value) is int:
        result = value
    elif isinstance(value, str) and re.fullmatch(r"0|[1-9][0-9]{0,19}", value):
        result = int(value)
    else:
        raise Error("invalid " + name)
    if not minimum <= result <= 18446744073709551615:
        raise Error("invalid " + name)
    return result


def amount(value):
    if not isinstance(value, str):
        raise Error("amount must be an exact decimal string")
    return number(value, "uzrn amount")


def read_bytes(path, limit=j.MAX_BYTES, private=False):
    path = Path(path)
    directory = fd = None
    try:
        directory = j._open_dir(path.parent)
        fd = j._open_file(path.name, directory)
        before = os.fstat(fd)
        if private and (before.st_uid != os.getuid() or before.st_mode & 0o077):
            raise Error("participant control file must be private and owned")
        raw = j._read_fd(fd, limit)
        after = os.fstat(fd)
        current = os.stat(path.name, dir_fd=directory, follow_symlinks=False)
        identity = lambda item: (item.st_dev, item.st_ino, item.st_size, item.st_mtime_ns, item.st_ctime_ns)
        if identity(before) != identity(after) or identity(after) != identity(current):
            raise Error("input changed while reading")
        return raw
    finally:
        if fd is not None:
            os.close(fd)
        if directory is not None:
            os.close(directory)


def intent_hash(binding, intent):
    return digest(j.canonical({"export_sha256": binding["export_sha256"],
                               "record_id": binding["record_id"],
                               "entry_sha256": binding["entry_sha256"], "intent": intent}))


def validate_intent(value):
    j._keys(value, {"content", "reasoning", "domain", "category", "method"})
    for name, limit, required in (("content", 1000, True), ("reasoning", 8192, True),
                                  ("domain", 128, True), ("category", 128, True),
                                  ("method", 128, False)):
        text_value(value[name], name, limit, required)
    if len(value["content"].encode("utf-8")) < 20:
        raise Error("claim content must contain at least 20 UTF-8 bytes")
    return value


def bound_reasoning(packet):
    binding = {"schema": "zerone-research-binding/v1", "intent_sha256": packet["intent_sha256"],
               **{key: packet["journal"][key] for key in ("export_sha256", "record_id", "entry_sha256")}}
    result = packet["intent"]["reasoning"] + "\n\nZerone research binding: " + j.canonical(binding).decode("utf-8")
    return text_value(result, "bound reasoning", 8192)


def prepare(raw, record_id, intent):
    export = j.validate_export(j.parse_json(raw))
    selected = [entry for entry in export["entries"] if entry["record"]["id"] == record_id]
    if len(selected) != 1 or selected[0]["record"]["kind"] != "contribution":
        raise Error("select one existing journal contribution")
    binding = {"export_utf8": raw.decode("utf-8"), "export_sha256": digest(raw),
               "collection_id": export["header"]["collection_id"],
               "head_sha256": export["entries"][-1]["sha256"], "record_id": record_id,
               "entry_sha256": selected[0]["sha256"]}
    packet = {"schema": SCHEMA, "journal": binding, "intent": validate_intent(copy.deepcopy(intent)),
              "intent_sha256": intent_hash(binding, intent), "participant": None,
              "submission": None, "observations": []}
    return validate_packet(packet)


def validate_participant(value):
    j._keys(value, {"chain_id", "address", "descriptor_sha256", "genesis_sha256", "binary_sha256", "client_files", "local_test"})
    if value["chain_id"] != CHAIN or not isinstance(value["address"], str) or not ADDRESS.fullmatch(value["address"]):
        raise Error("invalid participant chain/address")
    if type(value["local_test"]) is not bool:
        raise Error("participant local_test must be boolean")
    for field in ("descriptor_sha256", "genesis_sha256", "binary_sha256"):
        j._digest(value[field])
    j._keys(value["client_files"], CLIENT_NAMES)
    for value_hash in value["client_files"].values():
        j._digest(value_hash)
    return value


def validate_receipt(value):
    j._keys(value, {"id", "action", "actor", "status", "txhash"}, RECEIPT_FIELDS)
    if (not isinstance(value["id"], str) or not re.fullmatch(r"[0-9a-f]{32}", value["id"])
            or value["action"] != "submit" or value["actor"] != "participant"
            or value["status"] not in STATUSES or not isinstance(value["txhash"], str)
            or not TXHASH.fullmatch(value["txhash"])):
        raise Error("invalid public submit receipt")
    for field in ("height", "code", "gas_wanted", "gas_used", "tx_bytes"):
        if field in value:
            number(value[field], "receipt " + field)
    for field in ("claim_id", "round_id"):
        if field in value:
            text_value(value[field], field, 256)
    for field in ("claim_id", "round_id"):
        if field in value and not re.fullmatch(r"[0-9a-f]{32}", value[field]):
            raise Error("submit receipt needs generated development claim/round IDs")
    for field in ("review_fee_uzrn", "collateral_uzrn"):
        if field in value:
            amount(value[field])
    if value["status"] == "committed":
        number(value.get("height"), "committed height", 1)
        if number(value.get("code"), "committed code") != 0 or not value.get("claim_id"):
            raise Error("committed receipt needs a successful included transaction and claim ID")
    return value


def validate_history(value, packet, receipt):
    """Match actual fields, bound the graph, and retain unknown historical funding."""
    object_value(value, "claim history")
    if value.get("chain_id") != CHAIN:
        raise Error("history chain mismatch")
    at_height = number(value.get("block_height"), "history height", 1)
    if at_height < number(receipt.get("height"), "receipt height", 1):
        raise Error("history predates the observed transaction")
    claim_id = receipt["claim_id"]
    root = object_value(value.get("record"), "root claim record")
    claim = object_value(root.get("claim"), "root claim")
    if root.get("claim_id") != claim_id or claim.get("id") != claim_id:
        raise Error("history claim identity mismatch")
    expected = {"fact_content": packet["intent"]["content"], "domain": packet["intent"]["domain"],
                "category": packet["intent"]["category"], "method_id": packet["intent"]["method"],
                "reasoning_trace": bound_reasoning(packet),
                "submitter": packet["participant"]["address"]}
    if any(claim.get(field, "") != wanted for field, wanted in expected.items()):
        raise Error("observed claim differs from the prepared assertion or participant")
    if number(claim.get("submitted_at_block"), "claim admission height", 1) != number(receipt["height"], "receipt height", 1):
        raise Error("claim admission and transaction heights disagree")
    if "round_id" in receipt and claim.get("verification_round_id") != receipt["round_id"]:
        raise Error("submission round differs from the native claim")
    count = 0

    def rows(items):
        nonlocal count
        if items is None:
            return []
        if not isinstance(items, list):
            raise Error("malformed history list")
        count += len(items)
        if count > MAX_ROWS:
            raise Error("history exceeds the readable record budget")
        return items

    def settlement(plan, refund):
        if plan is None:
            return
        object_value(plan, "settlement")
        created = number(plan.get("created_at_block"), "settlement creation", 1)
        paid = number(plan.get("paid_at_block", "0"), "settlement payment")
        if created > at_height or paid > at_height or (paid and paid < created):
            raise Error("settlement height mismatch")
        if refund:
            text_value(plan.get("recipient"), "refund recipient", 256)
            amount(plan.get("amount"))
        else:
            amount(plan.get("withheld_total", "0"))
            for payment in rows(plan.get("payments")):
                object_value(payment, "reviewer payment")
                text_value(payment.get("verifier"), "reviewer", 256)
                amount(payment.get("amount"))
                amount(payment.get("withheld", "0"))

    def check_record(record):
        object_value(record, "claim record")
        identifier = text_value(record.get("claim_id"), "claim ID", 256)
        stored = record.get("claim")
        if stored is not None:
            object_value(stored, "claim")
            if stored.get("id") != identifier:
                raise Error("stored claim identity mismatch")
            terms = stored.get("funding_terms")
            if terms is not None:
                object_value(terms, "funding terms")
                if (type(terms.get("policy_version")) is not int or terms["policy_version"] != 1
                        or terms.get("kind") not in (1, 2, "CLAIM_FUNDING_KIND_REVIEW_FEE", "CLAIM_FUNDING_KIND_CHALLENGE_DEPOSIT")
                        or type(terms.get("kind")) is bool):
                    raise Error("unsupported funding policy/kind")
                paid, review, refundable, retained = [amount(terms.get(key)) for key in (
                    "paid_amount", "review_budget", "refundable_amount", "retained_fee")]
                if paid != review + refundable + retained:
                    raise Error("funding allocations do not sum to admission amount")
        for round_value in rows(record.get("rounds")):
            object_value(round_value, "round")
            if round_value.get("claim_id") != identifier:
                raise Error("round claim identity mismatch")
            text_value(round_value.get("id"), "round ID", 256)
            settlement(round_value.get("verifier_reward_settlement"), False)
            settlement(round_value.get("claim_refund_settlement"), True)
            for commit in rows(round_value.get("commits")):
                object_value(commit, "commit")
            for reveal in rows(round_value.get("reveals")):
                object_value(reveal, "reveal")
                if reveal.get("attestation") is not None:
                    object_value(reveal["attestation"], "attestation")
        for item in rows(record.get("facts")):
            entry = object_value(item, "derived fact")
            fact = object_value(entry.get("fact"), "fact")
            if fact.get("claim_id") != identifier:
                raise Error("fact claim identity mismatch")
            fact_id = text_value(fact.get("id"), "fact ID", 256)
            for direction, key in (("outgoing_relations", "source_fact_id"), ("incoming_relations", "target_fact_id")):
                for edge in rows(entry.get(direction)):
                    if object_value(edge, "fact relation").get(key) != fact_id:
                        raise Error("fact relation identity mismatch")
            for transition in rows(entry.get("status_transitions")):
                if object_value(transition, "fact transition").get("fact_id") != fact_id:
                    raise Error("fact transition identity mismatch")
        for missing in rows(record.get("missing_round_ids")):
            text_value(missing, "missing round ID", 256)
        return identifier

    seen = {check_record(root)}
    for related in rows(value.get("related_claims")):
        object_value(related, "related claim")
        identifier = check_record(related.get("record"))
        if identifier in seen:
            raise Error("duplicate related claim")
        seen.add(identifier)
        for edge in rows(related.get("links")):
            object_value(edge, "claim relation")
            # knowledge v12 adds content_hash, canonical_hash and structure.object (target: the root claim).
            if edge.get("field") not in ("provisional_fact_id", "challenged_claim_id", "relations.contradicts",
                                         "content_hash", "canonical_hash", "structure.object"):
                raise Error("unsupported claim relationship")
            text_value(edge.get("target_id"), "relationship target", 256)
    return value


def validate_packet(packet):
    j._keys(packet, {"schema", "journal", "intent", "intent_sha256", "participant", "submission", "observations"})
    if packet["schema"] != SCHEMA:
        raise Error("unsupported research workflow schema")
    binding = packet["journal"]
    j._keys(binding, {"export_utf8", "export_sha256", "collection_id", "head_sha256", "record_id", "entry_sha256"})
    raw = text_value(binding["export_utf8"], "journal export", j.MAX_BYTES).encode("utf-8")
    export = j.validate_export(j.parse_json(raw))
    selected = [entry for entry in export["entries"] if entry["record"]["id"] == binding["record_id"]]
    if (len(selected) != 1 or selected[0]["record"]["kind"] != "contribution"
            or selected[0]["sha256"] != binding["entry_sha256"]
            or export["header"]["collection_id"] != binding["collection_id"]
            or export["entries"][-1]["sha256"] != binding["head_sha256"]
            or digest(raw) != binding["export_sha256"]):
        raise Error("journal bytes or selected record binding mismatch")
    validate_intent(packet["intent"])
    if packet["intent_sha256"] != intent_hash(binding, packet["intent"]):
        raise Error("prepared intent hash mismatch")
    bound_reasoning(packet)
    participant, submission = packet["participant"], packet["submission"]
    if participant is not None:
        validate_participant(participant)
    receipt = None
    if submission is not None:
        if participant is None:
            raise Error("submission lacks participant binding")
        j._keys(submission, {"state", "receipt"})
        if submission["state"] not in ("started", "observed"):
            raise Error("unsupported submission state")
        receipt = submission["receipt"]
        if (submission["state"] == "started") != (receipt is None):
            raise Error("submission state/receipt mismatch")
        if receipt is not None:
            validate_receipt(receipt)
    elif participant is not None:
        raise Error("participant binding lacks a submission")
    observations = packet["observations"]
    if not isinstance(observations, list) or len(observations) > MAX_OBSERVATIONS:
        raise Error("observation inventory exceeds its bound")
    prior_time, prior_height = "", 0
    seen = set()
    for observation in observations:
        if receipt is None or receipt["status"] != "committed":
            raise Error("history requires a committed submit receipt")
        j._keys(observation, {"observed_at", "history_utf8", "history_sha256"})
        now = observation["observed_at"]
        if not isinstance(now, str) or not re.fullmatch(r"[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3}Z", now):
            raise Error("observation timestamp must use UTC milliseconds")
        try:
            dt.datetime.fromisoformat(now[:-1] + "+00:00")
        except ValueError as exc:
            raise Error("invalid observation timestamp") from exc
        history_raw = text_value(observation["history_utf8"], "history", j.MAX_BYTES).encode("utf-8")
        if digest(history_raw) != observation["history_sha256"] or observation["history_sha256"] in seen:
            raise Error("history digest mismatch or duplicate observation")
        history = validate_history(j.parse_json(history_raw), packet, receipt)
        at_height = number(history["block_height"], "history height", 1)
        if now < prior_time or at_height < prior_height:
            raise Error("observation time/height moved backwards")
        prior_time, prior_height = now, at_height
        seen.add(observation["history_sha256"])
    if len(j.canonical(packet)) + 1 > MAX_PACKET_BYTES:
        raise Error("workflow sidecar exceeds 16 MiB; nothing was truncated")
    return packet


def participant_binding(home, client):
    """Read public control pins only. Keys and private review tuples stay untouched."""
    home, client = Path(home), Path(client)
    directory = j._open_dir(home)
    try:
        info = os.fstat(directory)
        if info.st_uid != os.getuid() or info.st_mode & 0o077:
            raise Error("participant home must be private and owned")
    finally:
        os.close(directory)
    marker = object_value(j.parse_json(read_bytes(home / ".zerone-shared-participant.json", private=True)), "participant marker")
    if (marker.get("schema") != "zerone-shared-participant/v1" or marker.get("owner_uid") != os.getuid()
            or type(marker.get("allow_loopback_test")) is not bool):
        raise Error("invalid participant home marker")
    binding = {"chain_id": marker.get("chain_id"), "address": marker.get("address"),
               "descriptor_sha256": marker.get("descriptor_sha256"), "binary_sha256": marker.get("binary_sha256"),
               "genesis_sha256": marker.get("genesis_sha256"), "local_test": marker["allow_loopback_test"],
               "client_files": marker.get("client_files")}
    validate_participant(binding)
    if client.name != "shared-claims.py":
        raise Error("select the initialized package's shared-claims.py")
    for name, expected in binding["client_files"].items():
        if digest(read_bytes(client.with_name(name), 1024 * 1024)) != expected:
            raise Error("client source differs from the participant home's pin")
    for name, expected, bound in (("network.json", binding["descriptor_sha256"], 1024 * 1024),
                                   ("bin/zeroned", binding["binary_sha256"], 200 * 1024 * 1024),
                                   ("pinned-genesis.json", marker.get("genesis_sha256"), j.MAX_BYTES)):
        j._digest(expected)
        if digest(read_bytes(home / name, bound, private=True)) != expected:
            raise Error("participant control bytes differ from their pins")
    return binding


def run_client(home, client, command, arguments=()):
    """Only the existing participant client owns network, signatures and retries."""
    argv = [sys.executable, "-I", str(Path(client).absolute()), command, "--home", str(Path(home).absolute()), "--wait", "30", *arguments]
    # No shell, no automatic retry, no public echo of pre-reveal text or stderr.
    with tempfile.TemporaryFile() as output, tempfile.TemporaryFile() as errors:
        try:
            result = subprocess.run(argv, stdin=subprocess.DEVNULL, stdout=output, stderr=errors,
                                    shell=False, timeout=180, check=False)
        except subprocess.TimeoutExpired as exc:
            raise Error("participant client timed out; refresh observations before any further action") from exc
        if result.returncode:
            raise Error("participant client refused or did not complete; inspect it locally, then refresh; no automatic retry")
        output.seek(0)
        return j.parse_json(output.read(j.MAX_BYTES + 1))


@contextlib.contextmanager
def packet_lock(path):
    path = Path(path)
    parent = j._open_dir(path.parent)
    lock = None
    try:
        lock_name = "." + path.name + ".lock"
        lock = os.open(lock_name, os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW | os.O_CLOEXEC, 0o600, dir_fd=parent)
        info = j._regular(lock)
        if info.st_uid != os.getuid() or info.st_mode & 0o077:
            raise Error("workflow lock must be private and owned")
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError as exc:
            raise Error("another research workflow command holds this packet") from exc
        yield
    finally:
        if lock is not None:
            os.close(lock)
        os.close(parent)


def save_packet(path, packet):
    validate_packet(packet)
    path = Path(path)
    parent = j._open_dir(path.parent)
    temporary = ".workflow-" + secrets.token_hex(16)
    try:
        fd = j._open_file(path.name, parent)
        os.close(fd)
        j._new_file(parent, temporary, j.canonical(packet) + b"\n")
        os.replace(temporary, path.name, src_dir_fd=parent, dst_dir_fd=parent)
        os.fsync(parent)
    finally:
        os.close(parent)


def bind_participant(packet, home, client):
    actual = participant_binding(home, client)
    if packet["participant"] is not None and packet["participant"] != actual:
        raise Error("selected home/client differs from the packet participant")
    return actual


def submit(path, home, client, runner=run_client, *, _resume=False):
    with packet_lock(path):
        packet = validate_packet(j.parse_json(read_bytes(path, MAX_PACKET_BYTES), MAX_PACKET_BYTES))
        binding = bind_participant(packet, home, client)
        if _resume and packet["submission"] != {"state": "started", "receipt": None}:
            raise Error("resume requires a started packet without a receipt; use submit for a draft or refresh for an observed receipt")
        if not _resume and packet["submission"] is not None:
            raise Error("submission already started; use refresh, never an automatic second submit")
        packet["participant"] = binding
        packet["submission"] = {"state": "started", "receipt": None}
        # A downloaded/copied draft may already have been admitted from this
        # home. Recover its exact on-chain binding before invoking any signer.
        match = matching_submission(packet, runner(home, client, "snapshot"))
        if match is not None:
            apply_observation(packet, match)
            save_packet(path, packet)
            return packet
        save_packet(path, packet)  # Crash boundary precedes any signing/network work.
        arguments = ["--" + key + "=" + (bound_reasoning(packet) if key == "reasoning" else packet["intent"][key])
                     for key in ("content", "reasoning", "domain", "category", "method")]
        receipt = validate_receipt(runner(home, client, "submit", arguments))
        packet["submission"] = {"state": "observed", "receipt": receipt}
        save_packet(path, packet)
        return packet


def resume(path, home, client, runner=run_client):
    """Explicitly resume an unresolved start through the existing client's guards.

    Unlike refresh, the user deliberately authorizes a client submission here.
    A matching committed history is recovered first. Otherwise the same exact
    bound intent reaches the existing client's pending and duplicate checks;
    this adapter does not retry a signed transaction or create a new signer.
    """
    return submit(path, home, client, runner, _resume=True)


def refresh(path, home, client, runner=run_client):
    with packet_lock(path):
        packet = validate_packet(j.parse_json(read_bytes(path, MAX_PACKET_BYTES), MAX_PACKET_BYTES))
        bind_participant(packet, home, client)
        if packet["submission"] is None:
            raise Error("prepare does not create a claim; explicitly submit before refreshing")
        match = matching_submission(packet, runner(home, client, "snapshot"))
        if match is None:
            return packet  # Unknown outcome stays started; no replay or fabricated admission.
        apply_observation(packet, match)
        save_packet(path, packet)
        return packet


def matching_submission(packet, snapshot):
    """Find an unambiguous public receipt and exactly bound native assertion."""
    object_value(snapshot, "participant snapshot")
    if (snapshot.get("schema") != "zerone-shared-participant/v1" or snapshot.get("chain_id") != CHAIN
            or snapshot.get("participant_address") != packet["participant"]["address"]
            or snapshot.get("descriptor_sha256") != packet["participant"]["descriptor_sha256"]):
        raise Error("participant snapshot identity mismatch")
    if snapshot.get("loopback_test") is not packet["participant"]["local_test"]:
        raise Error("participant snapshot test mode mismatch")
    observed_height = number(snapshot.get("height"), "snapshot height", 1)
    transactions, claims = snapshot.get("transactions"), snapshot.get("claims")
    if not isinstance(transactions, list) or not isinstance(claims, list) or len(transactions) + len(claims) > MAX_ROWS:
        raise Error("invalid or excessive snapshot inventories")
    previous = packet["submission"]["receipt"]
    histories = {}
    for row in claims:
        object_value(row, "snapshot claim")
        claim_id = text_value(row.get("claim_id"), "snapshot claim ID", 256)
        if claim_id in histories:
            raise Error("duplicate snapshot claim")
        histories[claim_id] = row.get("history")
    matches = []
    for row in transactions:
        object_value(row, "snapshot transaction")
        if row.get("action") != "submit" or (previous is not None and row.get("txhash") != previous["txhash"]):
            continue
        receipt = validate_receipt(row)
        if receipt["status"] != "committed":
            if previous is not None:
                matches.append((receipt, None))
            continue
        history = histories.get(receipt["claim_id"])
        if history is None:
            if previous is not None:
                raise Error("committed submission has no observed history")
            continue
        try:
            validate_history(history, packet, receipt)
        except Error:
            if previous is not None:
                raise
            continue
        if number(history["block_height"], "history height") != observed_height:
            raise Error("snapshot histories were not read at one actual height")
        matches.append((receipt, history))
    if len(matches) > 1:
        raise Error("ambiguous matching submission; no receipt was rebound")
    if not matches:
        if previous is not None:
            raise Error("saved submission is absent from the participant snapshot")
        return None
    receipt, _ = matches[0]
    if previous is not None and (receipt["id"] != previous["id"] or
                                (previous.get("claim_id") is not None and receipt.get("claim_id") != previous["claim_id"])):
        raise Error("saved submission identity changed")
    return matches[0]


def apply_observation(packet, match):
    receipt, history = match
    packet["submission"] = {"state": "observed", "receipt": receipt}
    if history is not None:
        raw = j.canonical(history)
        # This is the exact retained client-decoded observation, not endpoint wire bytes.
        hash_value = digest(raw)
        if not any(row["history_sha256"] == hash_value for row in packet["observations"]):
            now = dt.datetime.now(dt.timezone.utc).isoformat(timespec="milliseconds").replace("+00:00", "Z")
            packet["observations"].append({"observed_at": now, "history_utf8": raw.decode("utf-8"),
                                           "history_sha256": hash_value})


def _admission_preview(packet):
    """Project an already validated packet; inspect owns validation and file I/O."""
    export = j.parse_json(packet["journal"]["export_utf8"].encode("utf-8"))
    records = [entry["record"] for entry in export["entries"]]
    local = {}
    for kind, fields in (("relation", ("id", "from_id", "to_id", "relation_type")),
                         ("concern", ("id", "target_id", "category", "attributed_to")),
                         ("assessment", ("id", "target_id", "disposition", "attributed_to"))):
        local[kind + "s"] = [{key: record[key] for key in fields}
                             for record in records if record["kind"] == kind]
    local["supersessions"] = [{key: record[key] for key in ("id", "kind", "supersedes")}
                             for record in records if "supersedes" in record]
    local["evidence_references"] = [{"record_id": record["id"], "references": record["evidence"]}
                                    for record in records if record["evidence"]]
    return {"scope": "complete_embedded_journal", "journal_entry_count": len(records),
            "network_checked": False, "evidence_availability": "not_checked", "local_only": local,
            "limitations": [
                "Only prepared_payload is handed to the participant client; the embedded journal is not uploaded.",
                "Local graph records are not translated into native relations, corrections or verdicts.",
                "Evidence references are not added to the payload automatically; no source is fetched, uploaded or availability-checked.",
                "This offline inspection does not check network admission, fees or participant readiness, and does not refresh saved observations.",
                "Sharing this output reveals local journal metadata; sharing the packet reveals its complete embedded journal.",
            ]}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    draft = commands.add_parser("prepare", help="Offline exact journal binding; does not submit")
    for name in ("journal", "record", "content", "reasoning", "domain", "category", "output"):
        draft.add_argument("--" + name, required=True)
    draft.add_argument("--method", default="")
    for name in ("submit", "resume", "refresh"):
        command = commands.add_parser(name)
        for flag in ("packet", "home", "client"):
            command.add_argument("--" + flag, required=True)
    inspect = commands.add_parser("inspect", help="Validate an offline sidecar; no network")
    inspect.add_argument("--packet", required=True)
    args = parser.parse_args(argv)
    try:
        if args.command == "prepare":
            result = prepare(read_bytes(args.journal), args.record,
                             {key: getattr(args, key) for key in ("content", "reasoning", "domain", "category", "method")})
            output = Path(args.output)
            directory = j._open_dir(output.parent)
            try:
                j._new_file(directory, output.name, j.canonical(result) + b"\n")
                os.fsync(directory)
            finally:
                os.close(directory)
        elif args.command == "inspect":
            result = validate_packet(j.parse_json(read_bytes(args.packet, MAX_PACKET_BYTES), MAX_PACKET_BYTES))
        else:
            result = {"submit": submit, "resume": resume, "refresh": refresh}[args.command](args.packet, args.home, args.client)
        summary = {"schema": SCHEMA, "record_id": result["journal"]["record_id"],
                   "intent_sha256": result["intent_sha256"], "submission": result["submission"],
                   "observation_count": len(result["observations"]),
                   "scope": "Local association and participant-client observations; no inclusion or scientific truth proof."}
        if args.command == "inspect":
            summary["prepared_payload"] = {**result["intent"], "reasoning": bound_reasoning(result)}
            summary["admission_preview"] = _admission_preview(result)
        print(j.canonical(summary).decode())
        return 0
    except (Error, OSError, ValueError, KeyError, TypeError) as exc:
        print("research workflow refused: " + str(exc), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())

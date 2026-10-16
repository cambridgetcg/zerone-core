#!/usr/bin/env python3
"""Signed claim exercises for the persistent claims-v1 LOCAL sandbox (Python 3.11+).

All accounts are disposable test identities under one controller. This helper
never starts a node, accepts remote endpoints, or retries a broadcast silently.
`attribute` encodes a zerone.attribution/v1 claim (a dry run unless --broadcast);
--excerpt-file runs the local quote guard before anything is signed.
"""
from __future__ import annotations

import argparse
import base64
from contextlib import contextmanager
import datetime
import fcntl
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import re
import secrets
import stat
import subprocess
import sys
import time
import unicodedata


def sibling(name):
    spec = importlib.util.spec_from_file_location(name.replace("-", "_"), Path(__file__).with_name(name + ".py"))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


local = sibling("local-node")
SCHEMA = "zerone-claim-workflow-v1"
NOTICE = ("Local test identities and valueless funds, controlled by one operator. "
          "Panel agreement is not independent scientific verification. Retained history is a node observation, not an inclusion proof.")
ACTORS = ("user", "reviewer1", "reviewer2", "reviewer3", "challenger")
REVIEWERS = ACTORS[1:4]
MAX_RECORDS = 100
MAX_JSON = 8 * 1024 * 1024
PENDING = ("prepared", "broadcasting", "pending")


class WorkflowError(Exception):
    pass


def integer(value, name, minimum=0):
    if isinstance(value, bool) or not re.fullmatch(r"0|[1-9][0-9]*", str(value)):
        raise WorkflowError(f"Invalid {name}.")
    result = int(value)
    if result < minimum:
        raise WorkflowError(f"Invalid {name}.")
    return result


def parse_json(raw):
    if len(raw.encode()) > MAX_JSON:
        raise WorkflowError("Local JSON exceeds the workflow bound.")
    def pairs(rows):
        result = {}
        for key, value in rows:
            if key in result:
                raise WorkflowError("Duplicate JSON field.")
            result[key] = value
        return result
    return json.loads(raw, object_pairs_hook=pairs)


def private_directory(path):
    try:
        path.mkdir(mode=0o700)
    except FileExistsError:
        pass
    info = path.lstat()
    if not stat.S_ISDIR(info.st_mode) or info.st_uid != os.getuid() or info.st_mode & 0o077:
        raise WorkflowError("Workflow directories must be private, owned, and not symlinks.")


def read_json(path):
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    with os.fdopen(fd, "rb") as stream:
        info = os.fstat(stream.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1 or info.st_uid != os.getuid() or info.st_mode & 0o077:
            raise WorkflowError("Workflow files must be private owned regular files with one link.")
        raw = stream.read(MAX_JSON + 1)
    if len(raw) > MAX_JSON:
        raise WorkflowError("Workflow record exceeds the read bound.")
    return parse_json(raw.decode())


def save_json(path, value, fresh=False):
    raw = (json.dumps(value, sort_keys=True, indent=2, ensure_ascii=False) + "\n").encode()
    if len(raw) > MAX_JSON:
        raise WorkflowError("Workflow record exceeds the write bound.")
    if fresh:
        fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
        with os.fdopen(fd, "wb") as stream:
            stream.write(raw)
            stream.flush()
            os.fsync(stream.fileno())
    else:
        read_json(path)  # Refuse links or replaced control files before update.
        temporary = path.with_name(".write-" + secrets.token_hex(12))
        save_json(temporary, value, fresh=True)
        os.replace(temporary, path)
    directory = os.open(path.parent, os.O_RDONLY)
    try:
        os.fsync(directory)
    finally:
        os.close(directory)


def public_receipt(attempt):
    # Never return command arguments, raw signed transactions or error output:
    # a prepared reveal contains the still-unpublished vote and attestation.
    fields = ("id", "action", "actor", "status", "txhash", "height", "code", "gas_wanted", "gas_used", "claim_id", "round_id")
    result = {key: attempt[key] for key in fields if key in attempt}
    if "tx_bytes_base64" in attempt:
        result["tx_bytes"] = len(base64.b64decode(attempt["tx_bytes_base64"], validate=True))
    for key in ("review_fee_uzrn", "collateral_uzrn"):
        if key in attempt.get("context", {}):
            result[key] = attempt["context"][key]
    return result


def command_receipt(attempt):
    # A command's own output also shows its quote-guard receipt (version and
    # excerpt sha256 only). Snapshots keep public_receipt's scalar metadata.
    result = public_receipt(attempt)
    if "quote_guard" in attempt.get("context", {}):
        result["quote_guard"] = attempt["context"]["quote_guard"]
    return result


def contribution_intent(value):
    # The fee is re-quoted and the quote-guard receipt is a local pre-signing
    # check; neither changes what is being contributed.
    return {key: item for key, item in value.items() if key not in ("review_fee_uzrn", "quote_guard")}


# The two claim types this workflow generates: ordinary assertions and
# zerone.attribution/v1 relations (x/knowledge types.proto ClaimType).
SIGNING_CLAIM_TYPES = {"CLAIM_TYPE_ASSERTION": 1, 1: 1, "1": 1, "CLAIM_TYPE_RELATION": 2, 2: 2, "2": 2}


def signing_json(value):
    # The current SDK signing JSON codec lacks this generated enum's symbolic
    # registration, although generate-only emits its name. Both are protobuf
    # JSON representations of the same value. Keep this bridge specific to the
    # claim types generated by this workflow; never rewrite text fields.
    for message in value.get("body", {}).get("messages", []):
        if message.get("@type") == "/zerone.knowledge.v1.MsgSubmitClaim":
            claim_type = message.get("claim_type")
            if isinstance(claim_type, bool) or not isinstance(claim_type, (str, int)) or claim_type not in SIGNING_CLAIM_TYPES:
                raise WorkflowError("Unexpected ordinary claim type in signing JSON.")
            message["claim_type"] = SIGNING_CLAIM_TYPES[claim_type]
    return value


def submit_arguments(context):
    # One source for both building and re-checking MsgSubmitClaim arguments.
    # --canonical is never sent: the chain derives it from the structure.
    result = ["knowledge", "submit-claim", *(context.get(key) for key in ("content", "domain", "category", "review_fee_uzrn")),
              "--method-id", context.get("method"), "--reasoning-trace", context.get("reasoning")]
    structure = context.get("structure")
    if structure is not None:
        keys = ("subject", "predicate", "object", "scope", "temporal_scope")
        if (context.get("claim_type") != "relation" or not isinstance(structure, dict)
                or set(structure) != {*keys, "tags"} or not all(isinstance(structure[key], str) for key in keys)
                or not isinstance(structure["tags"], list) or not all(isinstance(tag, str) for tag in structure["tags"])):
            raise WorkflowError("Unexpected structured claim intent; nothing was signed.")
        result += ["--claim-type", "relation"]
        for key in keys:
            result += ["--" + key.replace("_", "-"), structure[key]]
        result += ["--tags", ",".join(structure["tags"])]
    return result


# ─── Quote guard (plan §4.7) ────────────────────────────────────────────────
# Runs locally before signing. The excerpt is read from a local file and never
# leaves it: receipts keep only the guard version and the excerpt's sha256.
QUOTE_GUARD_VERSION = "zerone.quote-guard/v1"
QUOTE_WINDOW = 8
MAX_INPUT_FILE = 64 * 1024


def read_input(path, name):
    # Non-blocking open: a FIFO is refused instead of waiting for a writer.
    fd = os.open(path, os.O_RDONLY | os.O_NONBLOCK)
    with os.fdopen(fd, "rb") as stream:
        if not stat.S_ISREG(os.fstat(stream.fileno()).st_mode):
            raise WorkflowError(f"The {name} must be a regular file.")
        raw = stream.read(MAX_INPUT_FILE + 1)
    if len(raw) > MAX_INPUT_FILE:
        raise WorkflowError(f"The {name} exceeds {MAX_INPUT_FILE} bytes.")
    try:
        return raw.decode("utf-8")
    except UnicodeDecodeError:
        raise WorkflowError(f"The {name} must be UTF-8.") from None


def read_excerpt(path):
    return read_input(path, "excerpt file")


def excerpt_digest(excerpt):
    return "sha256:" + hashlib.sha256(unicodedata.normalize("NFC", excerpt).encode("utf-8")).hexdigest()


# Unicode Default_Ignorable_Code_Point (DerivedCoreProperties.txt); its reserved
# ranges are fixed by Unicode's stability policy. Python's unicodedata lacks it.
DEFAULT_IGNORABLE = frozenset(code for low, high in (
    (0x00AD, 0x00AD), (0x034F, 0x034F), (0x061C, 0x061C), (0x115F, 0x1160), (0x17B4, 0x17B5),
    (0x180B, 0x180F), (0x200B, 0x200F), (0x202A, 0x202E), (0x2060, 0x206F), (0x3164, 0x3164),
    (0xFE00, 0xFE0F), (0xFEFF, 0xFEFF), (0xFFA0, 0xFFA0), (0xFFF0, 0xFFF8), (0x1BCA0, 0x1BCA3),
    (0x1D173, 0x1D17A), (0xE0000, 0xE0FFF)) for code in range(low, high + 1))


def guard_tokens(text):
    # NFKD, casefold, NFKD. Then delete every combining mark (M*), format
    # character (Cf) and default-ignorable code point, so none can split a word.
    # Every other character outside letters (L*) and decimal digits (Nd) becomes
    # a space; split on whitespace. NFKD never recomposes, so a deletion cannot
    # create a new composition, and excerpt and field fold to the same base letters.
    folded = unicodedata.normalize("NFKD", unicodedata.normalize("NFKD", text).casefold())
    kept = []
    for ch in folded:
        category = unicodedata.category(ch)
        if category[0] == "M" or category == "Cf" or ord(ch) in DEFAULT_IGNORABLE:
            continue
        kept.append(ch if category[0] == "L" or category == "Nd" else " ")
    return "".join(kept).split()


def quote_guard(excerpt, fields):
    """Refuse any field holding 8 consecutive excerpt tokens, or the whole excerpt
    when it has fewer than 8. Messages name the field, never the text."""
    tokens = guard_tokens(excerpt)
    if not tokens:
        raise WorkflowError("The excerpt file has no letters or digits, so the quote guard cannot run; nothing was signed.")
    width = min(QUOTE_WINDOW, len(tokens))
    windows = {tuple(tokens[index:index + width]) for index in range(len(tokens) - width + 1)}
    rule = f"{QUOTE_WINDOW} consecutive excerpt tokens" if width == QUOTE_WINDOW else "the whole excerpt"
    for name, value in fields:
        if not isinstance(value, str):
            raise WorkflowError(f"Quote guard {QUOTE_GUARD_VERSION} cannot read {name}; nothing was signed.")
        candidate = guard_tokens(value)
        if any(tuple(candidate[index:index + width]) in windows for index in range(len(candidate) - width + 1)):
            raise WorkflowError(f"Quote guard {QUOTE_GUARD_VERSION} refused {name}: it contains {rule}. "
                                "Paraphrase it; nothing was signed or saved.")
    return {"version": QUOTE_GUARD_VERSION, "excerpt_sha256": excerpt_digest(excerpt)}


def guard_context(context, excerpt_file, fields):
    if excerpt_file:
        context["quote_guard"] = quote_guard(read_excerpt(excerpt_file), fields)
    return context


# ─── Attribution encoder (plan §4.2, §4.5; Yu's D5/D6/D7 defaults) ──────────
ATTRIBUTION_SCHEMA = "zerone.attribution/v1"
ATTRIBUTION_INTAKE = "zerone.attribution-intake/v1"
ATTRIBUTION_ACTS = ("states", "reports", "hedges", "denies")
# types.ValidAxiomCategories; the default genesis MaxClaimTextLength.
CLAIM_CATEGORIES = ("analytic", "formal", "empirical", "protocol", "computational")
MAX_FACT_CONTENT = 1000
STRUCTURE_FIELD = re.compile(r"[a-z0-9][a-z0-9:._/;=+-]*")
SHA256_REF = re.compile(r"sha256:[0-9a-f]{64}")
WORK_ID = re.compile(r"doi:10\.[0-9]{4,9}/[a-z0-9:._/;=+-]+"
                     r"|arxiv:(?:[0-9]{4}\.[0-9]{4,5}|[a-z-]+(?:\.[a-z]{2})?/[0-9]{7})v[1-9][0-9]*"
                     r"|isbn:97[89][0-9]{10}|url-sha256:[0-9a-f]{64}")
# Private persons appear only as opaque hermes: references (D6).
VOICE = re.compile(r"author|quoted:(?:orcid:[0-9]{4}-[0-9]{4}-[0-9]{4}-[0-9]{3}[0-9x]|wikidata:q[1-9][0-9]*|hermes:[0-9a-f]{64})")
LOCATOR = re.compile(r"[a-z0-9][a-z0-9:._=+-]{0,127}")
DATE = re.compile(r"[0-9]{4}-[0-9]{2}-[0-9]{2}")
DOMAIN = re.compile(r"[a-z0-9][a-z0-9_.-]{0,127}")
METHOD_ID = re.compile(r"[A-Za-z0-9][A-Za-z0-9._-]{0,127}")
DID = re.compile(r"did:[a-z0-9]+:(?:[A-Za-z0-9._-]|%[0-9A-Fa-f]{2})+(?::(?:[A-Za-z0-9._-]|%[0-9A-Fa-f]{2})+)*")


def jcs(value):
    """RFC 8785 JSON Canonicalization for strings, safe integers, booleans,
    null, arrays and objects (no floating-point numbers are needed here)."""
    def string(text):
        if not isinstance(text, str):
            raise WorkflowError("JCS object keys must be strings.")
        try:
            text.encode("utf-8")
        except UnicodeEncodeError:
            raise WorkflowError("JCS refuses unpaired surrogates.") from None
        escapes = {'"': '\\"', "\\": "\\\\", "\b": "\\b", "\f": "\\f", "\n": "\\n", "\r": "\\r", "\t": "\\t"}
        return '"' + "".join(escapes.get(ch) or (f"\\u{ord(ch):04x}" if ord(ch) < 0x20 else ch) for ch in text) + '"'
    if value is None:
        return "null"
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, int):
        if abs(value) > 2 ** 53 - 1:
            raise WorkflowError("JCS integers must be within the IEEE-754 safe range.")
        return str(value)
    if isinstance(value, str):
        return string(value)
    if isinstance(value, list):
        return "[" + ",".join(jcs(item) for item in value) + "]"
    if isinstance(value, dict):
        for key in value:
            string(key)
        # Sort by UTF-16 code units: big-endian UTF-16 bytes compare the same way.
        keys = sorted(value, key=lambda key: key.encode("utf-16-be"))
        return "{" + ",".join(string(key) + ":" + jcs(value[key]) for key in keys) + "}"
    raise WorkflowError("JCS here supports only strings, safe integers, booleans, null, arrays and objects.")


def claim_content_hash(text, domain):
    # Mirrors keeper.ComputeClaimContentHash (x/knowledge/keeper/state.go).
    try:
        return hashlib.sha256(("ZRN.claim.v1:" + domain + ":" + text).encode("utf-8")).hexdigest()
    except UnicodeEncodeError:
        raise WorkflowError("Claim text must be valid Unicode.") from None


def load_attribution_record(path):
    return parse_json(read_input(path, "attribution record"))


def attribution_claim(record, *, method="M-HISTORICAL", category="empirical"):
    """Encode a Hermes-style intake record as the plan §4.2 MsgSubmitClaim fields.
    Errors name fields, never their values."""
    def fields(value, name, keys):
        if not isinstance(value, dict) or set(value) != set(keys):
            raise WorkflowError(f"Attribution {name} must have exactly these fields: {', '.join(sorted(keys))}.")
        return value

    def grammar(pattern, value, name):
        if not isinstance(value, str) or not pattern.fullmatch(value):
            raise WorkflowError(f"Attribution {name} is outside the {ATTRIBUTION_SCHEMA} grammar (lowercase identifiers, no free text).")
        return value

    fields(record, "record", ("schema", "source_version", "statement_occurrence", "proposition", "extractor"))
    if record["schema"] != ATTRIBUTION_INTAKE:
        raise WorkflowError(f"Attribution record schema must be {ATTRIBUTION_INTAKE}.")
    source = fields(record["source_version"], "source_version", ("record", "work", "digest", "published"))
    occurrence = fields(record["statement_occurrence"], "statement_occurrence",
                        ("record", "locator", "act", "voice", "excerpt_sha256"))
    proposition = fields(record["proposition"], "proposition", ("record", "domain", "text"))
    for name, value in (("source_version.record", source["record"]), ("source_version.digest", source["digest"]),
                        ("statement_occurrence.record", occurrence["record"]),
                        ("statement_occurrence.excerpt_sha256", occurrence["excerpt_sha256"]),
                        ("proposition.record", proposition["record"])):
        grammar(SHA256_REF, value, name)
    work = grammar(WORK_ID, source["work"], "source_version.work")
    published = source["published"]
    if published is None:  # An explicit unknown stays unknown: no date is invented.
        published = ""
    else:
        grammar(DATE, published, "source_version.published")
        try:
            datetime.date.fromisoformat(published)
        except ValueError:
            raise WorkflowError("Attribution source_version.published must be a real YYYY-MM-DD date or null.") from None
    locator = grammar(LOCATOR, occurrence["locator"], "statement_occurrence.locator")
    if not isinstance(occurrence["act"], str) or occurrence["act"] not in ATTRIBUTION_ACTS:
        raise WorkflowError(f"Attribution statement_occurrence.act must be one of: {', '.join(ATTRIBUTION_ACTS)}.")
    voice = grammar(VOICE, occurrence["voice"], "statement_occurrence.voice")
    domain = grammar(DOMAIN, proposition["domain"], "proposition.domain")
    text = proposition["text"]
    if not isinstance(text, str) or not text.strip():
        raise WorkflowError("Attribution proposition.text must be the proposition claim's exact nonempty text.")
    extractor = grammar(DID, record["extractor"], "extractor")
    if len(extractor) > 256:
        raise WorkflowError("Attribution extractor DID exceeds 256 bytes.")
    grammar(METHOD_ID, method, "method_id")
    if category not in CLAIM_CATEGORIES:
        raise WorkflowError(f"Attribution category must be one of: {', '.join(CLAIM_CATEGORIES)}.")

    content = claim_content_hash(text, domain)
    structure = {
        "subject": work,
        "predicate": occurrence["act"] + ":" + voice,
        "object": "zrn-content:" + content,
        "scope": "ver=" + source["digest"] + ";loc=" + locator,
        "temporal_scope": published,
        "tags": ["attr-v1", "stmt:" + occurrence["record"].removeprefix("sha256:")[:40], "zc:" + content[:40]],
    }
    for key in ("subject", "predicate", "object", "scope", "temporal_scope"):
        if structure[key] or key != "temporal_scope":
            grammar(STRUCTURE_FIELD, structure[key], "structure." + key)
    for tag in structure["tags"]:
        if len(tag) > 50:
            raise WorkflowError("Attribution tag exceeds 50 characters.")
        grammar(STRUCTURE_FIELD, tag, "structure.tags")
    fact_content = (f"Attribution {ATTRIBUTION_SCHEMA}: {structure['subject']} {structure['predicate']} "
                    f"{structure['object']} at {structure['scope']}")
    if len(fact_content.encode("utf-8")) > MAX_FACT_CONTENT:
        raise WorkflowError(f"Attribution fact_content exceeds {MAX_FACT_CONTENT} bytes; shorten the work identifier or locator.")
    trace = jcs({"schema": ATTRIBUTION_SCHEMA, "source_version_record": source["record"],
                 "statement_occurrence": occurrence["record"], "excerpt_sha256": occurrence["excerpt_sha256"],
                 "proposition_record": proposition["record"], "extractor": extractor})
    return {"claim_type": "CLAIM_TYPE_RELATION", "domain": domain, "category": category, "method_id": method,
            "structure": structure, "fact_content": fact_content, "reasoning_trace": trace,
            "references": [], "relations": []}


def attribution_guard(claim, record, excerpt):
    if excerpt_digest(excerpt) != record["statement_occurrence"]["excerpt_sha256"]:
        raise WorkflowError("The excerpt file does not match statement_occurrence.excerpt_sha256 (sha256 of the "
                            "NFC-normalised exact file text; a trailing newline counts). Nothing was signed.")
    structure = claim["structure"]
    return quote_guard(excerpt, [("fact_content", claim["fact_content"]), ("reasoning_trace", claim["reasoning_trace"]),
                                 *(("structure." + key, structure[key])
                                   for key in ("subject", "predicate", "object", "scope", "temporal_scope")),
                                 *(("structure.tags", tag) for tag in structure["tags"])])


def attribution_context(claim, guard):
    if not isinstance(guard, dict) or guard.get("version") != QUOTE_GUARD_VERSION:
        raise WorkflowError("Broadcasting an attribution requires a passed quote guard; nothing was signed.")
    structure = claim["structure"]
    return {"content": claim["fact_content"], "domain": claim["domain"], "category": claim["category"],
            "method": claim["method_id"], "reasoning": claim["reasoning_trace"], "claim_type": "relation",
            "structure": {**{key: structure[key] for key in ("subject", "predicate", "object", "scope", "temporal_scope")},
                          "tags": list(structure["tags"])},
            "quote_guard": guard}


class Workflow:
    def __init__(self, home, wait=30):
        self.home = local.home_path(str(home))
        self.manifest = local.load_manifest(self.home)
        if self.manifest.get("knowledge_profile") != "claims-v1":
            raise WorkflowError("Use a fresh local-node init --knowledge-profile claims-v1 home.")
        self.accounts = self.manifest.get("accounts", {})
        if not isinstance(self.accounts, dict) or any(not re.fullmatch(r"zrn1[023456789acdefghjklmnpqrstuvwxyz]{38}", self.accounts.get(actor, "")) for actor in ACTORS):
            raise WorkflowError("The claims profile lacks its funded test accounts.")
        if len(set(self.accounts.values())) != len(self.accounts):
            raise WorkflowError("The claims profile aliases test accounts.")
        self.chain = self.manifest["chain_id"]
        self.rpc = f"http://127.0.0.1:{self.manifest['rpc_port']}"
        self.wait = wait
        self.directory = self.home / "claim-workflow"
        private_directory(self.directory)
        for name in ("attempts", "reviews"):
            private_directory(self.directory / name)
        self.check_node()

    def check_node(self):
        current = local.load_manifest(self.home)
        if current != self.manifest:
            raise WorkflowError("Local manifest changed during the workflow.")
        value = local.rpc_status(self.manifest)
        if value["sync_info"]["catching_up"] is not False:
            raise WorkflowError("Wait for the local node to finish catching up.")
        return integer(value["sync_info"]["latest_block_height"], "local block height", 1)

    @contextmanager
    def locked(self):
        fd = os.open(self.directory / ".lock", os.O_RDWR | os.O_CREAT | os.O_NOFOLLOW, 0o600)
        with os.fdopen(fd, "r+") as lock:
            info = os.fstat(lock.fileno())
            if not stat.S_ISREG(info.st_mode) or info.st_nlink != 1 or info.st_mode & 0o077:
                raise WorkflowError("Unsafe workflow lock.")
            try:
                fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
            except BlockingIOError as error:
                raise WorkflowError("Another workflow command is active; no transaction was sent.") from error
            yield

    def cli(self, *args, missing=False):
        result = subprocess.run([str(self.home / "bin/zeroned"), *map(str, args), "--home", str(self.home)],
                                capture_output=True, text=True, timeout=60, env=local.node_environment())
        if result.returncode:
            if missing and re.search(r"(?i)(tx .*not found|transaction .*not found)", result.stderr):
                return None
            # Computed commitment guidance contains the secret salt. Its full
            # diagnostic stays private; the reader receives neither it nor args.
            diagnostic = self.directory / "last-cli-error.json"
            save_json(diagnostic, {"command_family": list(map(str, args[:2])), "exit_code": result.returncode,
                                   "stderr": result.stderr[-65536:]}, fresh=not diagnostic.exists())
            raise WorkflowError(f"zeroned {' '.join(map(str, args[:2]))} failed (exit {result.returncode}); private diagnostic: claim-workflow/last-cli-error.json. No success is asserted.")
        return result.stdout.strip()

    def query(self, *args, height=None):
        self.check_node()
        flags = ["--node", self.rpc, "--output", "json"]
        if height is not None:
            flags += ["--height", str(height)]
        return parse_json(self.cli("query", *args, *flags))

    def records(self, name):
        paths = sorted((self.directory / name).iterdir())
        if len(paths) > MAX_RECORDS:
            raise WorkflowError("Local workflow inventory limit reached; no result was truncated.")
        result = []
        for path in paths:
            if not re.fullmatch(r"[0-9a-f]{32,64}\.json", path.name):
                raise WorkflowError("Unexpected workflow inventory entry.")
            value = read_json(path)
            self.validate_record(value)
            result.append((path, value))
        return result

    def record_metadata(self):
        return {"schema": SCHEMA, "chain_id": self.chain}

    def validate_record(self, value):
        if any(value.get(key) != expected for key, expected in self.record_metadata().items()):
            raise WorkflowError("Workflow record belongs to a different schema or chain.")

    def actor(self, actor):
        if actor not in ACTORS:
            raise WorkflowError("Choose a named claims-profile test actor.")
        self.check_node()
        address = self.cli("keys", "show", actor, "-a", "--keyring-backend", "test")
        if address != self.accounts[actor]:
            raise WorkflowError("The test signing account differs from the initialized profile.")
        return address

    def tx_observation(self, digest):
        self.check_node()
        raw = self.cli("query", "tx", digest, "--node", self.rpc, "--output", "json", missing=True)
        if raw is None:
            return None
        value = parse_json(raw)
        if value.get("txhash", "").upper() != digest:
            raise WorkflowError("Transaction query returned a different hash.")
        integer(value.get("height"), "transaction height", 1)
        integer(value.get("code", 0), "transaction code")
        for field in ("gas_used", "gas_wanted"):
            integer(value.get(field), field)
        return value

    def finish(self, path, attempt, observation):
        for field in ("height", "code", "gas_wanted", "gas_used"):
            attempt[field] = observation.get(field, 0)
        attempt["status"] = "committed" if int(attempt["code"]) == 0 else "failed"
        attempt["receipt"] = observation
        for field in ("claim_id", "round_id"):
            if field in attempt.get("context", {}):
                attempt[field] = attempt["context"][field]
        if attempt["action"] in ("submit", "challenge") and attempt["status"] == "committed":
            id_key = "counter_claim_id" if attempt["action"] == "challenge" else "claim_id"
            event_type = "zerone.knowledge.submit_contradiction" if attempt["action"] == "challenge" else "zerone.knowledge.submit_claim"
            ids = {attribute["value"] for event in observation.get("events", [])
                   if event.get("type") == event_type
                   for attribute in event.get("attributes", []) if attribute.get("key") == id_key}
            if len(ids) != 1:
                # Receipt remains durable even if a source/event format changes.
                attempt["status"] = "committed-unresolved"
            else:
                attempt["claim_id"] = ids.pop()
        save_json(path, attempt)
        return attempt

    @staticmethod
    def _terminal_submit_bytes(parent, address):
        receipt = parent.get("receipt" if parent["status"] == "failed" else "check_tx")
        if (parent.get("address") != address or not isinstance(receipt, dict)
                or not isinstance(receipt.get("txhash"), str) or receipt["txhash"].upper() != parent["txhash"]
                or integer(receipt.get("code"), "terminal receipt code", 1) != integer(parent.get("code"), "terminal code", 1)):
            raise WorkflowError("The retained terminal receipt or signing identity conflicts with this approval.")
        if parent["status"] == "failed" and integer(receipt.get("height"), "terminal receipt height", 1) != integer(parent.get("height"), "terminal height", 1):
            raise WorkflowError("The retained terminal receipt height conflicts with this approval.")
        raw_parent = base64.b64decode(parent.get("tx_bytes_base64", ""), validate=True)
        if not raw_parent or hashlib.sha256(raw_parent).hexdigest().upper() != parent["txhash"]:
            raise WorkflowError("The retained terminal signed bytes conflict with this approval.")
        return raw_parent

    def reconcile(self):
        pending = []
        records = self.records("attempts")
        for path, attempt in records:
            if "replacement" in attempt:
                reservation = attempt["replacement"]
                child_id = reservation.get("attempt_id") if isinstance(reservation, dict) else None
                if not isinstance(child_id, str) or not re.fullmatch(r"[0-9a-f]{32}", child_id):
                    raise WorkflowError("Invalid reserved replacement; operator resolution is required before new signing.")
                if "outcome" in reservation:
                    # A completed no-new-transaction result is not a child attempt.
                    # Never let it hide a conflicting child, duplicate hash or orphan.
                    if (set(reservation) != {"attempt_id", "max_review_fee_uzrn", "outcome"}
                            or reservation["outcome"] != {"status": "identical-transaction", "txhash": attempt.get("txhash")}
                            or attempt.get("action") != "submit" or child_id == attempt.get("id")
                            or not isinstance(attempt.get("txhash"), str)
                            or not re.fullmatch(r"[0-9A-F]{64}", attempt["txhash"])
                            or not isinstance(attempt.get("context"), dict)
                            or not isinstance(attempt.get("actor"), str) or attempt["actor"] not in self.accounts
                            or attempt.get("address") != self.accounts[attempt["actor"]]):
                        raise WorkflowError("Invalid identical-transaction replacement outcome; operator resolution is required.")
                    integer(reservation.get("max_review_fee_uzrn"), "reserved maximum review fee", 1)
                    if attempt.get("status") in ("failed", "rejected"):
                        self._terminal_submit_bytes(attempt, self.accounts[attempt["actor"]])
                    raw = base64.b64decode(attempt.get("tx_bytes_base64", ""), validate=True)
                    if (not raw or hashlib.sha256(raw).hexdigest().upper() != attempt["txhash"]
                            or any(child_path.name == child_id + ".json" or row.get("id") == child_id
                                   or row.get("resubmission_of") == attempt["txhash"] or row.get("txhash") == attempt["txhash"]
                                   for child_path, row in records if child_path != path)):
                        raise WorkflowError("Identical-transaction outcome conflicts with retained attempts; operator resolution is required.")
                else:
                    children = [row for child_path, row in records if child_path.name == child_id + ".json" and row.get("id") == child_id]
                    if len(children) != 1 or children[0].get("resubmission_of") != attempt.get("txhash"):
                        raise WorkflowError("A reserved replacement has no matching durable attempt; operator resolution is required before new signing.")
                    child = children[0]
                    if (attempt.get("action") != "submit" or child.get("action") != "submit"
                            or any(child.get(field) != attempt.get(field) for field in ("actor", "address"))
                            or not isinstance(attempt.get("context"), dict) or not isinstance(child.get("context"), dict)
                            or contribution_intent(child["context"]) != contribution_intent(attempt["context"])):
                        raise WorkflowError("A reserved replacement conflicts with its retained signer or intent; operator resolution is required.")
            if attempt.get("status") in PENDING:
                observed = self.tx_observation(attempt["txhash"])
                if observed is not None:
                    self.finish(path, attempt, observed)
                else:
                    pending.append(attempt["txhash"])
            if attempt.get("status") == "committed-unresolved":
                pending.append(attempt["txhash"])
        return pending

    def send(self, action, actor, arguments, context=None, *, fee_quote=None, resubmit_after=None, max_review_fee_uzrn=None):
        if self.reconcile():
            raise WorkflowError("An earlier transaction remains unresolved. Use snapshot/history to query it; no automatic retry or additional broadcast is allowed.")
        approval = resubmit_after is not None or max_review_fee_uzrn is not None
        cap = None
        if approval:
            if action != "submit" or resubmit_after is None or max_review_fee_uzrn is None:
                raise WorkflowError("Only submit accepts paired --resubmit-after and --max-review-fee-uzrn approval.")
            if not isinstance(resubmit_after, str) or not re.fullmatch(r"[0-9A-F]{64}", resubmit_after):
                raise WorkflowError("--resubmit-after must exactly match a retained uppercase transaction hash.")
            cap = integer(max_review_fee_uzrn, "maximum review fee", 1)
        if fee_quote is not None and action != "submit":
            raise WorkflowError("Review fee quotes apply only to submit.")
        address = self.actor(actor)
        parent = None
        parent_path = None
        raw_parent = None
        if action in ("submit", "challenge"):
            # A repeated documented command may be recovery from lost stdout.
            # Return its observed committed receipt instead of silently creating
            # the same contribution again at another height. The queried fee is
            # excluded because it can change between the two invocations.
            for _, prior in self.records("attempts"):
                if prior.get("action") == action and prior.get("actor") == actor and prior.get("status") == "committed" and contribution_intent(prior.get("context", {})) == contribution_intent(context or {}):
                    return prior
            if fee_quote is not None:
                if (action != "submit" or not isinstance(fee_quote, dict)
                        or len(arguments) < 6 or arguments[:2] != ["knowledge", "submit-claim"]
                        or fee_quote.get("offered_fee_uzrn") != (context or {}).get("review_fee_uzrn")
                        or fee_quote.get("offered_fee_uzrn") != arguments[5]):
                    raise WorkflowError("The review fee quote, context and transaction arguments disagree; nothing was signed.")
                if arguments != submit_arguments(context or {}):
                    raise WorkflowError("The submit arguments differ from the retained semantic intent; nothing was signed.")
                integer(fee_quote.get("height"), "quote height", 1)
                integer(fee_quote.get("offered_fee_uzrn"), "quoted review fee", 1)
            terminal = [(path, prior) for path, prior in self.records("attempts")
                        if prior.get("action") == action and prior.get("actor") == actor
                        and prior.get("status") in ("failed", "rejected")
                        and contribution_intent(prior.get("context", {})) == contribution_intent(context or {})]
            if action == "submit" and (terminal or approval):
                if not approval or cap is None:
                    raise WorkflowError("A terminal submission requires explicit --resubmit-after and --max-review-fee-uzrn approval.")
                matching = [(path, prior) for path, prior in terminal if prior.get("txhash") == resubmit_after]
                if len(matching) != 1:
                    raise WorkflowError("No unique terminal submission matches this approval and unchanged intent.")
                parent_path, parent = matching[0]
                raw_parent = self._terminal_submit_bytes(parent, address)
                if "replacement" in parent:
                    raise WorkflowError("This submission already reserved its one replacement; resolve that attempt instead.")
                if fee_quote is None or integer(fee_quote.get("offered_fee_uzrn"), "quoted review fee", 1) > cap:
                    raise WorkflowError("The fresh review fee exceeds the approved maximum or its quote is unavailable; nothing was signed.")
        if len(self.records("attempts")) >= MAX_RECORDS:
            raise WorkflowError("Local transaction inventory is full.")
        identifier = secrets.token_hex(16)
        path = self.directory / "attempts" / (identifier + ".json")
        if parent is not None and parent_path is not None:
            # Consume approval durably before generation/signing, under the caller's workflow lock.
            parent["replacement"] = {"attempt_id": identifier, "max_review_fee_uzrn": str(cap)}
            save_json(parent_path, parent)
        unsigned = signing_json(parse_json(self.cli("tx", *arguments, "--from", actor, "--chain-id", self.chain,
            "--node", self.rpc, "--keyring-backend", "test", "--gas", "2000000", "--fees", "2000000uzrn",
            "--generate-only", "--output", "json")))
        # Preparation files live separately from the bounded attempt inventory.
        unsigned_path = self.directory / (identifier + "-unsigned.json")
        signed_path = self.directory / (identifier + "-signed.json")
        save_json(unsigned_path, unsigned, fresh=True)
        signed = signing_json(parse_json(self.cli("tx", "sign", unsigned_path, "--from", actor, "--chain-id", self.chain,
            "--node", self.rpc, "--keyring-backend", "test", "--output", "json")))
        save_json(signed_path, signed, fresh=True)
        encoded = self.cli("tx", "encode", signed_path)
        raw = base64.b64decode(encoded, validate=True)
        if not raw or len(raw) > MAX_JSON:
            raise WorkflowError("Invalid encoded signed transaction.")
        digest = hashlib.sha256(raw).hexdigest().upper()
        if parent is not None:
            matches = [row for _, row in self.records("attempts") if row.get("txhash") == digest]
            if digest == parent["txhash"] and raw == raw_parent and len(matches) == 1:
                # Deterministic signing can reproduce a rejected transaction. Consume
                # the reservation without adding a second hash-addressed attempt.
                parent["replacement"]["outcome"] = {"status": "identical-transaction", "txhash": digest}
                save_json(parent_path, parent)
                raise WorkflowError(
                    "The approved replacement is identical to the retained transaction; no new transaction was saved or broadcast. "
                    f"Approval is consumed; use claim-workflow retry --txhash {digest} (shared-claims retry --tx {digest}) for the existing transaction.")
            if matches:
                raise WorkflowError("The encoded replacement conflicts with a retained transaction hash; operator resolution is required.")
        attempt = {**self.record_metadata(), "id": identifier, "action": action, "actor": actor,
            "address": address, "status": "prepared", "txhash": digest, "tx_bytes_base64": encoded,
            "context": context or {}, "created_unix": int(time.time())}
        if fee_quote is not None:
            attempt["review_fee_quote"] = fee_quote
        if parent is not None:
            attempt["resubmission_of"] = parent["txhash"]
        save_json(path, attempt, fresh=True)
        return self.broadcast(path, attempt, signed_path)

    def broadcast(self, path, attempt, signed_path):
        digest = attempt["txhash"]
        raw = base64.b64decode(attempt["tx_bytes_base64"], validate=True)
        if hashlib.sha256(raw).hexdigest().upper() != digest:
            raise WorkflowError("Saved transaction bytes do not match their hash; nothing was broadcast.")
        read_json(signed_path)
        if self.cli("tx", "encode", signed_path) != attempt["tx_bytes_base64"]:
            raise WorkflowError("Saved signed transaction changed; nothing was broadcast.")
        self.check_node()
        prior_delivery = attempt["status"] != "prepared" or attempt.get("broadcast_count", 0) > 0
        attempt["status"] = "broadcasting"
        attempt["broadcast_count"] = attempt.get("broadcast_count", 0) + 1
        save_json(path, attempt)
        try:
            response = parse_json(self.cli("tx", "broadcast", signed_path, "--node", self.rpc,
                                          "--broadcast-mode", "sync", "--output", "json"))
            if response.get("txhash", "").upper() != digest:
                raise WorkflowError("Broadcast response returned a different hash.")
            code = integer(response.get("code", 0), "CheckTx code")
            if code and not prior_delivery:
                attempt.update(status="rejected", code=code, check_tx=response)
                for field in ("gas_wanted", "gas_used"):
                    if field in response:
                        try:
                            integer(response[field], field)
                        except WorkflowError:
                            continue
                        attempt[field] = response[field]
                save_json(path, attempt)
                raise WorkflowError(f"CheckTx rejected transaction {digest} with code {code}; receipt retained.")
            if code:
                # A cache/sequence refusal on an explicit retry cannot reject
                # the original delivery. Keep blocking new signing while its
                # exact hash remains unresolved; only execution settles it.
                attempt["check_tx"] = response
            attempt["status"] = "pending"
            save_json(path, attempt)
            deadline = time.monotonic() + self.wait
            while time.monotonic() < deadline:
                observed = self.tx_observation(digest)
                if observed is not None:
                    self.finish(path, attempt, observed)
                    if attempt["status"] != "committed":
                        raise WorkflowError(f"Transaction {digest} ended {attempt['status']}; receipt retained.")
                    return attempt
                time.sleep(0.3)
        except (OSError, subprocess.TimeoutExpired, ValueError) as error:
            raise WorkflowError(f"Transaction {digest} has an unresolved outcome; retained for query, never automatically rebroadcast.") from error
        raise WorkflowError(f"Transaction {digest} is pending after {self.wait}s; use snapshot to reconcile, never silently retry.")

    def retry(self, digest):
        digest = digest.upper()
        if not re.fullmatch(r"[0-9A-F]{64}", digest):
            raise WorkflowError("--txhash must be64 hexadecimal characters.")
        matches = [(path, row) for path, row in self.records("attempts") if row.get("txhash") == digest]
        if len(matches) != 1:
            raise WorkflowError("No unique saved transaction matches this hash.")
        path, attempt = matches[0]
        self.actor(attempt["actor"])
        observed = self.tx_observation(digest)
        if observed is not None:
            self.finish(path, attempt, observed)
            if attempt["status"] != "committed":
                raise WorkflowError("The transaction already committed unsuccessfully; it was not rebroadcast.")
            return public_receipt(attempt)
        if attempt["status"] not in (*PENDING, "rejected"):
            raise WorkflowError("This saved transaction has a terminal receipt; it was not rebroadcast.")
        if any(row.get("txhash") != digest and row.get("status") in PENDING for _, row in self.records("attempts")):
            raise WorkflowError("Another transaction is unresolved; retry that exact transaction first.")
        signed_path = self.directory / (attempt["id"] + "-signed.json")
        return public_receipt(self.broadcast(path, attempt, signed_path))

    def onboard(self):
        if self.reconcile():
            raise WorkflowError("Resolve the pending transaction before onboarding another account.")
        result = []
        for actor in ACTORS:
            prior = [value for _, value in self.records("attempts") if value["action"] == "onboard" and value["actor"] == actor and value["status"] == "committed"]
            if prior:
                result.append(public_receipt(prior[-1]))
                continue
            result.append(public_receipt(self.send("onboard", actor, ["zerone_auth", "onboard", "agent"])))
        return {"chain_id": self.chain, "local_only": True, "transactions": result}

    def round_for_claim(self, claim_id):
        response = self.query("knowledge", "claim", claim_id)
        claim = response.get("claim", {})
        if claim.get("id") != claim_id or not claim.get("verification_round_id"):
            raise WorkflowError("Claim query has no matching round.")
        row = self.query("knowledge", "verification-round", claim["verification_round_id"]).get("round", {})
        if row.get("id") != claim["verification_round_id"] or row.get("claim_id") != claim_id:
            raise WorkflowError("Round and claim identities disagree.")
        if integer(row.get("commitment_scheme"), "commitment scheme") != 2 or row.get("commitment_chain_id") != self.chain or integer(row.get("review_policy_version"), "review policy") != 1:
            raise WorkflowError("This workflow requires a current bound, neutral review round.")
        return row

    def submit(self, args):
        context = guard_context({"content": args.content, "domain": args.domain, "category": args.category,
                                 "method": args.method, "reasoning": args.reasoning},
                                getattr(args, "excerpt_file", None),
                                (("fact_content", args.content), ("reasoning_trace", args.reasoning)))
        return self.submit_context(context, resubmit_after=getattr(args, "resubmit_after", None),
                                   max_review_fee_uzrn=getattr(args, "max_review_fee_uzrn", None))

    def attribute(self, claim, guard, *, resubmit_after=None, max_review_fee_uzrn=None):
        # The existing submit path, plus the attribution structure flags.
        return self.submit_context(attribution_context(claim, guard), resubmit_after=resubmit_after,
                                   max_review_fee_uzrn=max_review_fee_uzrn)

    def submit_context(self, context, *, resubmit_after=None, max_review_fee_uzrn=None):
        # Query both inputs to the existing effective-fee formula at one height.
        height = self.check_node()
        params = self.query("knowledge", "params", height=height)["params"]
        pacing = self.query("alignment", "global-pacing", height=height)
        multiplier = integer(pacing.get("creation_multiplier_bps", "0"), "creation pacing") or 1000000
        fee = integer(params["min_review_fee"], "minimum review fee", 1)
        if multiplier < 1000000:
            fee = fee * 1000000 // multiplier
        # Private provenance, not an inclusion proof or a promise of execution pricing.
        quote = {"height": str(height), "base_fee_uzrn": params["min_review_fee"],
                 "creation_multiplier_bps": pacing.get("creation_multiplier_bps"),
                 "effective_multiplier_bps": str(multiplier), "offered_fee_uzrn": str(fee)}
        context = {**context, "review_fee_uzrn": str(fee)}
        return command_receipt(self.send("submit", "user", submit_arguments(context), context, fee_quote=quote,
                                        resubmit_after=resubmit_after, max_review_fee_uzrn=max_review_fee_uzrn))

    def review_path(self, claim_id, actor):
        digest = hashlib.sha256((self.chain + "\0" + claim_id + "\0" + actor).encode()).hexdigest()
        return self.directory / "reviews" / (digest + ".json")

    @staticmethod
    def review_flags(review):
        result = ["--commitment-scheme", "2", "--commitment-chain-id", review["chain_id"],
                  "--confidence", str(review["confidence"]), "--review-reason", review["reason"], "--review-scope", review["scope"],
                  "--review-method", review["method"]]
        for item in review["evidence"]:
            result += ["--review-evidence", item]
        return result

    def review(self, args, reveal=False):
        # A commit fixes the on-chain review text; reveal only reuses the saved tuple.
        guard = None
        if not reveal and getattr(args, "excerpt_file", None):
            guard = quote_guard(read_excerpt(args.excerpt_file), [("review reason", args.reason), ("review scope", args.scope),
                                                                  *(("review evidence", item) for item in args.evidence)])
        self.actor(args.actor)
        row = self.round_for_claim(args.claim)
        path = self.review_path(args.claim, args.actor)
        existing = path.exists()
        if existing:
            review = read_json(path)
            if not reveal and any(review[field] != getattr(args, field) for field in ("vote", "confidence", "reason", "scope", "method", "evidence")):
                raise WorkflowError("A saved review tuple cannot be replaced. Repeat its exact inputs, or reveal it.")
        else:
            if reveal:
                raise WorkflowError("No saved review tuple; reveal cannot invent replacement values.")
            if len(self.records("reviews")) >= MAX_RECORDS:
                raise WorkflowError("Local review inventory is full.")
            review = {**self.record_metadata(), "claim_id": args.claim, "round_id": row["id"],
                "actor": args.actor, "address": self.accounts[args.actor], "vote": args.vote,
                "confidence": args.confidence, "reason": args.reason, "scope": args.scope, "method": args.method, "evidence": args.evidence,
                "salt": secrets.token_hex(32), "commit_deadline": row["commit_deadline"], "reveal_deadline": row["reveal_deadline"]}
        if review.get("chain_id") != self.chain or review.get("round_id") != row["id"] or review.get("actor") != args.actor or review.get("claim_id") != args.claim or review.get("address") != self.accounts[args.actor]:
            raise WorkflowError("Saved review tuple identity mismatch.")
        self.validate_review(review)
        action = "reveal" if reveal else "commit"
        self.reconcile()
        prior = [value for _, value in self.records("attempts") if value["action"] == action and value["actor"] == args.actor and value.get("context", {}).get("round_id") == row["id"]]
        for attempt in prior:
            if attempt["status"] == "committed":
                return command_receipt(attempt)
            if attempt["status"] in PENDING:
                raise WorkflowError("This review action has an unresolved attempt; no duplicate broadcast was sent.")
        phase = row.get("phase")
        wanted = ("VERIFICATION_PHASE_REVEAL", 2, "2") if reveal else ("VERIFICATION_PHASE_COMMIT", 1, "1")
        if phase not in wanted:
            raise WorkflowError(f"Round is in {phase}; {action} was not sent. Saved tuple is retained.")
        if not existing:
            save_json(path, review, fresh=True)
        arguments = ["knowledge", "submit-reveal", row["id"], review["vote"], review["salt"]] if reveal else [
            "knowledge", "submit-commitment", row["id"], "--vote", review["vote"], "--salt", review["salt"]]
        context = {"claim_id": args.claim, "round_id": row["id"]}
        if guard is not None:
            context["quote_guard"] = guard
        return command_receipt(self.send(action, args.actor, arguments + self.review_flags(review), context))

    @staticmethod
    def validate_review(review):
        if review.get("vote") not in ("accept", "reject", "malformed") or not 0 <= integer(review.get("confidence"), "confidence") <= 1000000:
            raise WorkflowError("Invalid saved review vote or confidence.")
        for field, bound in (("reason", 4096), ("scope", 1024), ("method", 128)):
            if not isinstance(review.get(field), str) or len(review[field].encode()) > bound:
                raise WorkflowError("Review text exceeds the retained-record bounds.")
        if not review["reason"].strip():
            raise WorkflowError("A review must state what was checked.")
        evidence = review.get("evidence")
        if not isinstance(evidence, list) or len(evidence) > 16 or any(not isinstance(item, str) or not item.strip() or len(item.encode()) > 256 for item in evidence) or len(set(evidence)) != len(evidence):
            raise WorkflowError("Review evidence must contain at most16 distinct, nonempty references of256 bytes each.")
        if not re.fullmatch(r"[0-9a-f]{64}", review.get("salt", "")):
            raise WorkflowError("Invalid retained review salt.")

    def challenge(self, args):
        context = guard_context({"fact_id": args.fact, "content": args.content, "reason": args.reason,
                                 "evidence": args.evidence, "collateral_uzrn": args.stake},
                                getattr(args, "excerpt_file", None),
                                (("contradiction text", args.content), ("challenge reason", args.reason),
                                 *(("challenge evidence", item) for item in args.evidence)))
        if any("," in item for item in args.evidence):
            raise WorkflowError("Contradiction evidence references cannot contain commas in the current CLI.")
        return command_receipt(self.send("challenge", "challenger", ["knowledge", "submit-contradiction", args.fact,
            args.content, args.stake, args.reason, "--evidence-ids", ",".join(args.evidence)], context))

    def history(self, claim_id, height=None):
        height = self.check_node() if height is None else height
        value = self.query("knowledge", "claim-history", claim_id, height=height)
        if value.get("chain_id") != self.chain or integer(value.get("block_height"), "history height") != height or value.get("record", {}).get("claim_id") != claim_id:
            raise WorkflowError("Claim history returned a different identity or context.")
        return value

    def snapshot(self):
        with self.locked():
            self.reconcile()
            height = self.check_node()
            attempts = [value for _, value in self.records("attempts")]
            reviews = [value for _, value in self.records("reviews")]
            ids = sorted({value["claim_id"] for value in attempts if value.get("claim_id")} |
                         {value["claim_id"] for value in reviews})
            if len(ids) > MAX_RECORDS:
                raise WorkflowError("Claim inventory exceeds the local bound.")
            pending = []
            for review in reviews:
                matching = [value for value in attempts if value.get("context", {}).get("round_id") == review["round_id"] and value["actor"] == review["actor"]]
                revealed = any(value["action"] == "reveal" and value["status"] == "committed" for value in matching)
                committed = any(value["action"] == "commit" and value["status"] == "committed" for value in matching)
                status = "revealed" if revealed else "awaiting-reveal" if committed else "commit-unconfirmed"
                if not revealed and height >= integer(review["reveal_deadline"], "reveal deadline"):
                    status = "reveal-missed"
                pending.append({**{key: review[key] for key in ("claim_id", "round_id", "actor", "commit_deadline", "reveal_deadline")}, "status": status})
            return {"schema": SCHEMA, "chain_id": self.chain, "height": str(height), "local_only": True, "notice": NOTICE,
                    "claims": [{"claim_id": claim_id, "history": self.history(claim_id, height)} for claim_id in ids],
                    "transactions": [public_receipt(value) for value in attempts], "pending_reviews": pending}


def parser():
    result = argparse.ArgumentParser(description=__doc__)
    sub = result.add_subparsers(dest="command", required=True)
    for name in ("onboard", "submit", "commit", "reveal", "challenge", "history", "snapshot", "retry", "serve"):
        command = sub.add_parser(name)
        command.add_argument("--home", required=True)
        command.add_argument("--wait", type=int, default=30, help="Seconds to observe a single broadcast (1–120).")
        if name == "submit":
            command.add_argument("--content", required=True)
            command.add_argument("--method", default="")
            command.add_argument("--reasoning", required=True)
            command.add_argument("--domain", default="physics")
            command.add_argument("--category", default="empirical")
            command.add_argument("--resubmit-after", help="Explicitly replace this retained terminal submit hash once; not an exact-byte retry.")
            command.add_argument("--max-review-fee-uzrn", help="Positive review-fee ceiling for that replacement; the transaction fee is additional.")
        if name in ("commit", "reveal", "history"):
            command.add_argument("--claim", required=True)
        if name in ("commit", "reveal"):
            command.add_argument("--actor", choices=REVIEWERS, required=True)
        if name == "reveal":
            command.add_argument("--wait-for-phase", action="store_true", help="Wait up to 900 seconds for this round's reveal phase; never broadcast outside it.")
        if name == "commit":
            command.add_argument("--vote", choices=("accept", "reject", "malformed"), required=True)
            command.add_argument("--reason", required=True)
            command.add_argument("--scope", required=True)
            command.add_argument("--method", default="")
            command.add_argument("--confidence", type=int, default=800000)
        if name == "challenge":
            command.add_argument("--fact", required=True)
            command.add_argument("--content", required=True)
            command.add_argument("--reason", required=True)
            command.add_argument("--stake", default="11000000", help="Explicit local contradiction collateral in uzrn (default11 ZRN).")
        if name in ("commit", "challenge"):
            command.add_argument("--evidence", action="append", default=[])
        if name in ("submit", "commit", "challenge"):
            command.add_argument("--excerpt-file", help=EXCERPT_HELP)
        if name == "serve":
            command.add_argument("--port", type=int, default=47658)
        if name == "retry":
            command.add_argument("--txhash", required=True, help="Query first, then explicitly rebroadcast only the identical retained signed bytes if unresolved.")
    command = sub.add_parser("attribute", help="Encode a zerone.attribution/v1 claim; a dry run unless --broadcast.")
    command.add_argument("--record", required=True, help=f"{ATTRIBUTION_INTAKE} JSON naming Hermes records by sha256 only.")
    command.add_argument("--excerpt-file", help=EXCERPT_HELP + " Required with --broadcast; must match excerpt_sha256.")
    command.add_argument("--method", default="M-HISTORICAL", help="Registered methodology (default M-HISTORICAL).")
    command.add_argument("--category", default="empirical", choices=CLAIM_CATEGORIES)
    mode = command.add_mutually_exclusive_group()
    mode.add_argument("--dry-run", action="store_true", help="Explicit form of the default: sign, save and broadcast nothing.")
    mode.add_argument("--broadcast", action="store_true", help="Sign and broadcast through the submit path (needs --home).")
    command.add_argument("--home", help="claims-v1 local home; used only with --broadcast.")
    command.add_argument("--wait", type=int, default=30, help="Seconds to observe a single broadcast (1–120).")
    command.add_argument("--resubmit-after", help="With --broadcast: replace this retained terminal submit hash once.")
    command.add_argument("--max-review-fee-uzrn", help="With --broadcast: review-fee ceiling for that replacement.")
    return result


EXCERPT_HELP = ("Local excerpt file for the quote guard: refuse on-chain text repeating 8 consecutive excerpt tokens "
                "(or a whole shorter excerpt). The excerpt is never sent, saved or printed.")


def attribute_command(args):
    if not args.broadcast and (args.resubmit_after is not None or args.max_review_fee_uzrn is not None):
        raise WorkflowError("--resubmit-after and --max-review-fee-uzrn apply only with --broadcast; a dry run signs nothing.")
    if args.broadcast and (not args.excerpt_file or not args.home):
        raise WorkflowError("attribute --broadcast requires --home and --excerpt-file, so the quote guard runs before signing.")
    record = load_attribution_record(args.record)
    claim = attribution_claim(record, method=args.method, category=args.category)
    guard = attribution_guard(claim, record, read_excerpt(args.excerpt_file)) if args.excerpt_file else None
    if not args.broadcast:
        return {"schema": ATTRIBUTION_SCHEMA, "dry_run": True, "local_only": True, "claim": claim, "quote_guard": guard,
                "notice": "Dry run: nothing was signed, saved or broadcast. The quote guard ran only if quote_guard is set."}
    workflow = Workflow(args.home, args.wait)
    with workflow.locked():
        return workflow.attribute(claim, guard, resubmit_after=args.resubmit_after,
                                  max_review_fee_uzrn=args.max_review_fee_uzrn)


def main():
    os.umask(0o077)
    args = parser().parse_args()
    try:
        if not 1 <= args.wait <= 120:
            raise WorkflowError("--wait must be1–120 seconds.")
        if args.command == "commit" and not 0 <= args.confidence <= 1000000:
            raise WorkflowError("--confidence must be0–1000000.")
        if args.command == "attribute":
            print(json.dumps(attribute_command(args), sort_keys=True, ensure_ascii=False))
            return 0
        workflow = Workflow(args.home, args.wait)
        if args.command == "serve":
            sibling("claim-workflow-view").serve(workflow.snapshot, args.port)
            return 0
        if args.command == "snapshot":
            output = workflow.snapshot()
        else:
            if args.command == "reveal" and args.wait_for_phase:
                # Waiting is read-only and holds no workflow lock, so the
                # reader can keep displaying phase progress. review() checks
                # the actual phase again under the write lock before signing.
                deadline = time.monotonic() + 900
                while workflow.round_for_claim(args.claim).get("phase") in ("VERIFICATION_PHASE_COMMIT", 1, "1") and time.monotonic() < deadline:
                    time.sleep(0.5)
            with workflow.locked():
                if args.command == "onboard":
                    output = workflow.onboard()
                elif args.command == "submit":
                    output = workflow.submit(args)
                elif args.command in ("commit", "reveal"):
                    output = workflow.review(args, args.command == "reveal")
                elif args.command == "challenge":
                    output = workflow.challenge(args)
                elif args.command == "retry":
                    output = workflow.retry(args.txhash)
                else:
                    workflow.reconcile()
                    output = workflow.history(args.claim)
        print(json.dumps(output, sort_keys=True, ensure_ascii=False))
        return 0
    except (WorkflowError, local.LocalNodeError, OSError, ValueError, KeyError, TypeError, subprocess.TimeoutExpired) as error:
        print(f"claim-workflow: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())

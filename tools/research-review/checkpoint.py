#!/usr/bin/env python3
"""Prepare or compare an exact journal-file commitment. No network or signing.

Chain inclusion and signatures require the separate research-checkpoint Go tool.
Neither tool authenticates declared historical dates or scientific conclusions.
"""

import argparse
import hashlib
import os
from pathlib import Path
import re
import sys

import journal as j


SCHEMA = "zerone-research-checkpoint/v1"
CHAIN_RE = re.compile(r"[A-Za-z0-9][A-Za-z0-9._-]{0,49}\Z", re.ASCII)


def read_export_bytes(path):
    """Read once: the file hash and journal validation concern identical bytes."""
    value = Path(path)
    directory = fd = None
    try:
        directory = j._open_dir(value.parent)
        fd = j._open_file(value.name, directory)
        return j._read_fd(fd)
    except OSError as exc:
        raise j.JournalError("cannot read regular export file: " + str(exc)) from exc
    finally:
        if fd is not None:
            os.close(fd)
        if directory is not None:
            os.close(directory)


def prepare(raw, chain_id):
    if not isinstance(chain_id, str) or not CHAIN_RE.fullmatch(chain_id):
        raise j.JournalError("invalid chain ID")
    export = j.validate_export(j.parse_json(raw))
    if not export["entries"]:
        raise j.JournalError("checkpoint requires at least one journal entry")
    collection = export["header"]["collection_id"]
    count = len(export["entries"])
    digest = hashlib.sha256(raw).hexdigest()
    head = export["entries"][-1]["sha256"]
    memo = f"zerone:research:v1:{collection}:{count}:{digest}:{head}"
    if len(memo.encode("utf-8")) > 256:
        raise j.JournalError("checkpoint exceeds the development-chain memo limit")
    return {"schema": SCHEMA, "chain_id": chain_id, "collection_id": collection,
            "entry_count": count, "export_sha256": digest, "head_sha256": head,
            "memo": memo}


def verify(raw, checkpoint, memo, chain_id):
    expected = prepare(raw, chain_id)
    # Exact types matter: JSON true must not compare equal to an entry count of 1.
    if (not isinstance(checkpoint, dict) or checkpoint.keys() != expected.keys()
            or any(type(checkpoint[k]) is not type(v) or checkpoint[k] != v
                   for k, v in expected.items())):
        raise j.JournalError("checkpoint does not match the exact export and chain ID")
    if not isinstance(memo, str) or memo != expected["memo"]:
        raise j.JournalError("transaction memo does not match the checkpoint")
    return {"export_matches_memo": True, "chain_id": chain_id,
            "entry_count": expected["entry_count"],
            "export_sha256": expected["export_sha256"],
            "head_sha256": expected["head_sha256"],
            "scope": "File and memo comparison only; verify transaction inclusion and signatures separately."}


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    for name in ("prepare", "verify"):
        command = commands.add_parser(name)
        command.add_argument("export")
        command.add_argument("--chain-id", required=True)
        if name == "prepare":
            command.add_argument("--output", required=True)
        else:
            command.add_argument("checkpoint")
            command.add_argument("--memo", required=True,
                                 help="Exact memo also checked by the chain proof verifier")
    args = parser.parse_args(argv)
    try:
        raw = read_export_bytes(args.export)
        if args.command == "prepare":
            result = prepare(raw, args.chain_id)
            j._write_output(args.output, result)
        else:
            result = verify(raw, j.read_json_file(args.checkpoint, limit=4096),
                            args.memo, args.chain_id)
        print(j.canonical(result).decode("utf-8"))
        return 0
    except j.JournalError as exc:
        print("research checkpoint refused: " + str(exc), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())

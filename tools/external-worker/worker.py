#!/usr/bin/env python3
"""Bounded, offline reproduction of Bitcoin Core v27.0's Merkle property.

Source: https://github.com/bitcoin/bitcoin/blob/v27.0/src/consensus/merkle.cpp
All leaf and root hex denotes raw digest bytes, NOT reversed display txids.
Leaf i is SHA256(seed bytes || i encoded as four big-endian bytes), starting at
zero. Internal nodes use double SHA256. Mutation checks precede odd padding at
EVERY level. This reproduces a documented property, not a new vulnerability.

The six public protocol functions are canonical, digest, make_task,
validate_task, compute and verify. Invalid protocol values raise ValueError;
inputs are not repaired or normalized. A task ID hashes the canonical task
without task_id; receipt hashes cover the entire canonical task/result, without
file-format trailing newlines. Worker labels are self-declared, not identities.

CLI (POSIX): make-task --seed-hex HEX --leaf-count N --reward PATH --output PATH;
run --task PATH --output PATH --worker-id LABEL; verify --task PATH --result PATH.
Inputs are UTF-8 JSON regular files of at most 32768 bytes. No path component may
be a symlink. Outputs are exclusively created, mode 0600, never overwritten;
make-task/run are silent and verify prints canonical JSON plus a newline.
Malformed input or I/O errors exit 2 with stderr only. No network, signing,
funding check, payment, code execution from inputs, or import-time work occurs.
"""

from contextlib import contextmanager
import hashlib
import json
import os
import re
import stat
import sys

TASK_SCHEMA = "zerone.external-work-task/v1"
RESULT_SCHEMA = "zerone.external-work-result/v1"
KIND = "bitcoin-merkle-duplicate-v1"
MAX_INPUT_BYTES = 32768
TASK_KEYS = {"schema", "task_id", "kind", "seed_hex", "leaf_count", "reward"}
REWARD_KEYS = {"chain_id", "genesis_sha256", "recipient", "amount_uzrn", "funding_txhash"}
RESULT_KEYS = {"schema", "task_id", "worker_id", "result"}
DETAIL_KEYS = {"root_hex", "extended_root_hex", "original_mutated", "extended_mutated",
               "original_count", "extended_count"}


def _json_value(value) -> None:
    if type(value) is dict:
        for key, item in value.items():
            if type(key) is not str:
                raise ValueError("JSON object keys must be strings")
            _json_value(item)
    elif type(value) is list:
        for item in value:
            _json_value(item)
    elif type(value) not in (str, int, float, bool, type(None)):
        raise ValueError("not a JSON value")


def canonical(value) -> bytes:
    """Sorted, compact, UTF-8 JSON (unescaped Unicode, no non-finite numbers).

    Only exact JSON builtin types are accepted; invalid input raises ValueError.
    """
    try:
        _json_value(value)
        return json.dumps(
            value, sort_keys=True, separators=(",", ":"), ensure_ascii=False, allow_nan=False
        ).encode("utf-8")
    except (TypeError, ValueError, RecursionError) as exc:
        raise ValueError("invalid JSON value") from exc


def digest(data: bytes) -> str:
    """SHA-256 of bytes, returned as lowercase hexadecimal."""
    if type(data) is not bytes:
        raise ValueError("digest input must be bytes")
    return hashlib.sha256(data).hexdigest()


def _keys(value, expected: set, label: str) -> None:
    if type(value) is not dict or set(value) != expected:
        raise ValueError(f"{label} must be an object with exactly the specified keys")


def _matches(value, pattern: str, label: str) -> None:
    if type(value) is not str or re.fullmatch(pattern, value) is None:
        raise ValueError(f"invalid {label}")


def _validate_recipient(address: str) -> None:
    # BIP173 Bech32, not Bech32m, no witness-version prefix. Exactly 32 data
    # symbols encode exactly 20 bytes, with no residual/padding bits.
    alphabet = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
    _matches(address, rf"zrn1[{alphabet}]{{38}}", "recipient")
    values = [alphabet.index(char) for char in address[4:]]
    hrp = "zrn"
    expanded = [ord(c) >> 5 for c in hrp] + [0] + [ord(c) & 31 for c in hrp]
    generators = (0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3)
    checksum = 1
    for value in expanded + values:
        top = checksum >> 25
        checksum = ((checksum & 0x1ffffff) << 5) ^ value
        for bit, generator in enumerate(generators):
            if (top >> bit) & 1:
                checksum ^= generator
    if checksum != 1:
        raise ValueError("invalid recipient Bech32 checksum")


def _validate_reward(reward: dict) -> None:
    _keys(reward, REWARD_KEYS, "reward")
    if type(reward["chain_id"]) is not str or reward["chain_id"] != "zerone-dev-1":
        raise ValueError("reward chain_id must be zerone-dev-1")
    _matches(reward["genesis_sha256"], r"[0-9a-f]{64}", "genesis_sha256")
    _matches(reward["funding_txhash"], r"[0-9A-F]{64}", "funding_txhash")
    amount = reward["amount_uzrn"]
    _matches(amount, r"[1-9][0-9]{0,8}", "amount_uzrn")
    if int(amount) > 100000000:
        raise ValueError("amount_uzrn exceeds 100000000")
    _validate_recipient(reward["recipient"])


def validate_task(task: dict) -> None:
    """Reject malformed tasks or IDs not bound to the entire unsigned task."""
    _keys(task, TASK_KEYS, "task")
    if type(task["schema"]) is not str or task["schema"] != TASK_SCHEMA:
        raise ValueError("unsupported task schema")
    if type(task["kind"]) is not str or task["kind"] != KIND:
        raise ValueError("unsupported task kind")
    _matches(task["seed_hex"], r"[0-9a-f]{64}", "seed_hex")
    count = task["leaf_count"]
    if type(count) is not int or not 3 <= count <= 4095 or count % 2 != 1:
        raise ValueError("leaf_count must be an odd integer in 3..4095")
    _matches(task["task_id"], r"[0-9a-f]{64}", "task_id")
    _validate_reward(task["reward"])
    unsigned = {key: value for key, value in task.items() if key != "task_id"}
    if task["task_id"] != digest(canonical(unsigned)):
        raise ValueError("task_id does not match the canonical unsigned task")


def make_task(seed_hex: str, leaf_count: int, reward_dict: dict) -> dict:
    """Create an unsigned task whose ID binds all its computation/reward fields."""
    _validate_reward(reward_dict)
    task = {"schema": TASK_SCHEMA, "kind": KIND,
            "seed_hex": seed_hex, "leaf_count": leaf_count, "reward": dict(reward_dict)}
    task["task_id"] = digest(canonical(task))
    validate_task(task)
    return task


def _worker_tree(leaves: list[bytes]) -> tuple[bytes, bool]:
    """Level-wise Bitcoin Core reduction; compare pairs BEFORE odd padding."""
    level = list(leaves)
    mutated = False
    while len(level) > 1:
        for index in range(0, len(level) - 1, 2):
            if level[index] == level[index + 1]:
                mutated = True
        if len(level) % 2:
            level.append(level[-1])
        level = [hashlib.sha256(hashlib.sha256(level[i] + level[i + 1]).digest()).digest()
                 for i in range(0, len(level), 2)]
    return level[0], mutated


def _validate_worker_id(worker_id: str) -> None:
    if (type(worker_id) is not str or not 1 <= len(worker_id) <= 160
            or not worker_id.strip() or not worker_id.isprintable()):
        raise ValueError("worker_id must be 1..160 printable characters, not all whitespace")


def compute(task: dict, worker_id: str) -> dict:
    """Compute the exact task; worker_id is self-declared, not identity evidence."""
    validate_task(task)
    _validate_worker_id(worker_id)
    seed = bytes.fromhex(task["seed_hex"])
    count = task["leaf_count"]
    leaves = [hashlib.sha256(seed + i.to_bytes(4, "big")).digest() for i in range(count)]
    root, mutated = _worker_tree(leaves)
    extended_root, extended_mutated = _worker_tree(leaves + [leaves[-1]])
    return {"schema": RESULT_SCHEMA, "task_id": task["task_id"],
            "worker_id": worker_id, "result": {
                "root_hex": root.hex(), "extended_root_hex": extended_root.hex(),
                "original_mutated": mutated, "extended_mutated": extended_mutated,
                "original_count": count, "extended_count": count + 1,
            }}


def _verification_tree(leaves: list[bytes]) -> tuple[bytes, bool]:
    """Independent depth-first reduction over implicit tree positions.

    No level array, padding, or calls to the worker reducer. Missing right
    children reuse the left digest, but only real siblings affect mutation.
    """
    def visit(height: int, position: int) -> tuple[bytes, bool]:
        if height == 0:
            return leaves[position], False
        child_span = 1 << (height - 1)
        child_width = (len(leaves) + child_span - 1) // child_span
        left, left_mutated = visit(height - 1, position * 2)
        has_right = position * 2 + 1 < child_width
        if has_right:
            right, right_mutated = visit(height - 1, position * 2 + 1)
        else:
            right, right_mutated = left, False
        first_hash = hashlib.sha256()
        first_hash.update(left)
        first_hash.update(right)
        root = hashlib.sha256(first_hash.digest()).digest()
        return root, left_mutated or right_mutated or (has_right and left == right)

    return visit((len(leaves) - 1).bit_length(), 0)


def _validate_result(result: dict) -> None:
    _keys(result, RESULT_KEYS, "result envelope")
    if type(result["schema"]) is not str or result["schema"] != RESULT_SCHEMA:
        raise ValueError("unsupported result schema")
    _matches(result["task_id"], r"[0-9a-f]{64}", "result task_id")
    _validate_worker_id(result["worker_id"])
    detail = result["result"]
    _keys(detail, DETAIL_KEYS, "result details")
    for field in ("root_hex", "extended_root_hex"):
        _matches(detail[field], r"[0-9a-f]{64}", field)
    for field in ("original_mutated", "extended_mutated"):
        if type(detail[field]) is not bool:
            raise ValueError(f"{field} must be a boolean")
    for field in ("original_count", "extended_count"):
        if type(detail[field]) is not int:
            raise ValueError(f"{field} must be an integer")


def verify(task: dict, result: dict) -> dict:
    """Independently recompute and return a hash-bound computation-only receipt.

    This verifies neither worker identity/independence, new knowledge, actual
    task funding, nor on-chain payment. Reward metadata describes an operator's
    prefunded test sponsor, not protocol escrow. Hashes are not authentication.
    The caller must retain and compare its expected task ID/reward identity.
    """
    validate_task(task)
    _validate_result(result)
    if result["task_id"] != task["task_id"]:
        raise ValueError("result task_id does not match this task/reward identity")
    seed = bytes.fromhex(task["seed_hex"])
    leaves = []
    for index in range(task["leaf_count"]):
        leaf_hash = hashlib.sha256(seed)
        leaf_hash.update(index.to_bytes(4, byteorder="big"))
        leaves.append(leaf_hash.digest())
    root, mutated = _verification_tree(leaves)
    extended_root, extended_mutated = _verification_tree([*leaves, leaves[-1]])
    expected = {"root_hex": root.hex(), "extended_root_hex": extended_root.hex(),
                "original_mutated": mutated, "extended_mutated": extended_mutated,
                "original_count": len(leaves), "extended_count": len(leaves) + 1}
    if result["result"] != expected:
        raise ValueError("result does not match independent recomputation")
    return {"schema": "zerone.external-work-verification/v1",
            "assurance": "specified-computation-only", "task_id": task["task_id"],
            "task_sha256": digest(canonical(task)), "result_sha256": digest(canonical(result)),
            "result": expected}


def _unique_object(pairs):
    value = {}
    for key, item in pairs:
        if key in value:
            raise ValueError("duplicate JSON object key")
        value[key] = item
    return value


def _reject_constant(value):
    raise ValueError("non-finite JSON constant")


@contextmanager
def _parent_directory(path):
    """Open each directory component without following symlinks (POSIX).

    Descriptor-relative traversal avoids a check-then-open symlink race. Do not
    resolve/collapse '..' before opening the preceding components.
    """
    raw = os.fspath(path)
    if type(raw) is not str or not raw or "\x00" in raw:
        raise ValueError("invalid file path")
    parts = raw.split("/")
    if parts[-1] in ("", ".", ".."):
        raise ValueError("path must name a file")
    flags = os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW
    directory = os.open("/" if raw.startswith("/") else ".", flags)
    try:
        for component in parts[:-1]:
            if component in ("", "."):
                continue
            next_directory = os.open(component, flags, dir_fd=directory)
            os.close(directory)
            directory = next_directory
        yield directory, parts[-1]
    finally:
        os.close(directory)


def _read_json(path):
    with _parent_directory(path) as (directory, name):
        fd = os.open(name, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=directory)
    with os.fdopen(fd, "rb") as stream:
        info = os.fstat(stream.fileno())
        if not stat.S_ISREG(info.st_mode):
            raise ValueError("JSON input must be a regular file, not a symlink or special file")
        if info.st_size > MAX_INPUT_BYTES:
            raise ValueError("JSON input exceeds 32768 bytes")
        data = stream.read(MAX_INPUT_BYTES + 1)
    if len(data) > MAX_INPUT_BYTES:
        raise ValueError("JSON input exceeds 32768 bytes")
    text = data.decode("utf-8")
    value = json.loads(text, object_pairs_hook=_unique_object, parse_constant=_reject_constant)
    canonical(value)  # Also reject exponent overflow and lone Unicode surrogates.
    return value


def _write_json(path, value) -> None:
    data = canonical(value) + b"\n"
    with _parent_directory(path) as (directory, name):
        fd = os.open(name, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW,
                     0o600, dir_fd=directory)
    with os.fdopen(fd, "wb") as stream:
        os.fchmod(stream.fileno(), 0o600)
        stream.write(data)


def main(argv=None) -> int:
    """Local file CLI; importing this module performs no work or I/O."""
    import argparse

    parser = argparse.ArgumentParser(description=__doc__, allow_abbrev=False)
    commands = parser.add_subparsers(dest="command", required=True)
    make = commands.add_parser("make-task", allow_abbrev=False)
    make.add_argument("--seed-hex", required=True)
    make.add_argument("--leaf-count", required=True, type=int)
    make.add_argument("--reward", required=True)
    make.add_argument("--output", required=True)
    run = commands.add_parser("run", allow_abbrev=False)
    run.add_argument("--task", required=True)
    run.add_argument("--output", required=True)
    run.add_argument("--worker-id", required=True)
    check = commands.add_parser("verify", allow_abbrev=False)
    check.add_argument("--task", required=True)
    check.add_argument("--result", required=True)
    args = parser.parse_args(argv)
    try:
        if args.command == "make-task":
            task = make_task(args.seed_hex, args.leaf_count, _read_json(args.reward))
            _write_json(args.output, task)
        elif args.command == "run":
            result = compute(_read_json(args.task), args.worker_id)
            _write_json(args.output, result)
        else:
            receipt = verify(_read_json(args.task), _read_json(args.result))
            sys.stdout.buffer.write(canonical(receipt) + b"\n")
    except (OSError, ValueError, RecursionError) as exc:
        print(f"worker: error: {exc}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

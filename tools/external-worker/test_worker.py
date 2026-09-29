#!/usr/bin/env python3
"""Synthetic-only tests. No chain, funding, private corpus or network is used.

Run with TMPDIR set explicitly to approved scratch and python3 -B test_worker.py.
"""

import ast
from contextlib import ExitStack
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import stat
import subprocess
import sys
import tempfile
import unittest
from unittest import mock

WORKER_PATH = Path(__file__).with_name("worker.py")


def synthetic_address(payload=bytes(20), hrp="zrn", constant=1):
    """Synthetic BIP173 encoder; never generates or accesses a signing key."""
    alphabet = "qpzry9x8gf2tvdw0s3jn54khce6mua7l"
    bits = "".join(f"{byte:08b}" for byte in payload)
    bits += "0" * (-len(bits) % 5)
    data = [int(bits[i:i + 5], 2) for i in range(0, len(bits), 5)]
    expanded = [ord(c) >> 5 for c in hrp] + [0] + [ord(c) & 31 for c in hrp]
    check = 1
    for value in expanded + data + [0] * 6:
        high = check >> 25
        check = ((check & 0x1ffffff) << 5) ^ value
        for bit, generator in enumerate((0x3b6a57b2, 0x26508e6d, 0x1ea119fa,
                                          0x3d4233dd, 0x2a1462b3)):
            if high & (1 << bit):
                check ^= generator
    check ^= constant
    return hrp + "1" + "".join(alphabet[x] for x in data) + "".join(
        alphabet[(check >> shift) & 31] for shift in range(25, -1, -5)
    )


def synthetic_reward():
    return {"chain_id": "zerone-dev-1", "genesis_sha256": "ab" * 32,
            "recipient": synthetic_address(), "amount_uzrn": "1000",
            "funding_txhash": "CD" * 32}


def task_hash(unsigned):
    return hashlib.sha256(json.dumps(unsigned, sort_keys=True, separators=(",", ":"),
                                     ensure_ascii=False, allow_nan=False).encode()).hexdigest()


class SyntheticWorkerTests(unittest.TestCase):
    def setUp(self):
        self.assertTrue(WORKER_PATH.is_file(), "worker module is not implemented")
        spec = importlib.util.spec_from_file_location("external_worker_under_test", WORKER_PATH)
        assert spec is not None and spec.loader is not None
        self.worker = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.worker)

    def test_canonical_is_sorted_compact_utf8_json(self):
        self.assertEqual(
            self.worker.canonical({"z": [True, None, 4], "a": "宇"}),
            '{"a":"宇","z":[true,null,4]}'.encode("utf-8"),
        )
        self.assertEqual(self.worker.canonical({"b": 1, "a": 2}), b'{"a":2,"b":1}')

    def test_canonical_rejects_non_json_values(self):
        cyclic = []
        cyclic.append(cyclic)
        for value in [float("nan"), float("inf"), -float("inf"), {1: "x"},
                      (1, 2), {1, 2}, b"bytes", object(), "\ud800", cyclic]:
            with self.subTest(value=type(value).__name__):
                with self.assertRaises(ValueError):
                    self.worker.canonical(value)

    def test_digest_hashes_only_bytes_to_lowercase_sha256(self):
        self.assertTrue(callable(getattr(self.worker, "digest", None)), "digest API missing")
        self.assertEqual(self.worker.digest(b"abc"),
                         "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad")
        for bad in ["abc", bytearray(b"abc"), memoryview(b"abc"), None, 1]:
            with self.subTest(bad=repr(bad)), self.assertRaises(ValueError):
                self.worker.digest(bad)

    def test_make_task_binds_exact_unsigned_canonical_task(self):
        self.assertTrue(callable(getattr(self.worker, "make_task", None)), "make_task API missing")
        reward = synthetic_reward()
        unsigned = {"schema": "zerone.external-work-task/v1",
                    "kind": "bitcoin-merkle-duplicate-v1", "seed_hex": "00" * 32,
                    "leaf_count": 3, "reward": dict(reward)}
        expected = {**unsigned, "task_id": task_hash(unsigned)}
        task = self.worker.make_task("00" * 32, 3, reward)
        self.assertEqual(task, expected)
        reward["amount_uzrn"] = "2000"
        self.assertEqual(task, expected, "caller reward must not alias the task")

    def test_task_validation_enforces_exact_schema_and_work_bounds(self):
        self.assertTrue(callable(getattr(self.worker, "validate_task", None)),
                        "validate_task API missing")
        good = self.worker.make_task("12" * 32, 3, synthetic_reward())
        self.assertIsNone(self.worker.validate_task(good))
        high = self.worker.make_task("ff" * 32, 4095, synthetic_reward())
        self.assertIsNone(self.worker.validate_task(high))
        invalid = {
            "schema": [None, [], 1, "zerone.external-work-task/v2"],
            "kind": [None, False, "bitcoin-merkle-duplicate-v2"],
            "seed_hex": [None, 0, [], "AB" * 32, "0" * 63, "0" * 65, "g" * 64,
                         "00" * 32 + "\n", " " + "00" * 32],
            "leaf_count": [True, False, None, "3", 3.0, 2, 4, -3, 4097, 10**100],
            "task_id": [None, False, "AA" * 32, "0" * 63, "0" * 64],
        }
        for field, values in invalid.items():
            for value in values:
                bad = {**good, field: value}
                if field != "task_id":
                    bad["task_id"] = task_hash({k: v for k, v in bad.items() if k != "task_id"})
                with self.subTest(field=field, value=value):
                    with self.assertRaises(ValueError):
                        self.worker.validate_task(bad)
                    if field in ("seed_hex", "leaf_count"):
                        with self.assertRaises(ValueError):
                            self.worker.make_task(bad["seed_hex"], bad["leaf_count"], synthetic_reward())
        for value in [None, [], "task", 1, True, {**good, "unknown": 1}]:
            with self.subTest(value=value), self.assertRaises(ValueError):
                self.worker.validate_task(value)
        for key in good:
            with self.subTest(missing=key), self.assertRaises(ValueError):
                self.worker.validate_task({k: v for k, v in good.items() if k != key})

    def test_reward_schema_is_exact_and_amount_is_bounded_canonical_decimal(self):
        good = synthetic_reward()
        bad_rewards = [None, [], list(good.items()), True, "reward", {**good, "escrow": True}]
        bad_rewards.extend({k: v for k, v in good.items() if k != field} for field in good)
        changes = {
            "chain_id": [None, [], False, "zerone-1", "zerone-dev-1 "],
            "genesis_sha256": [None, 1, "AB" * 32, "ab" * 31, "gg" * 32, "ab" * 32 + "\n"],
            "funding_txhash": [None, 1, "cd" * 32, "CD" * 31, "GG" * 32, "CD" * 32 + "\n"],
            "recipient": [None, 1, [], False],
            "amount_uzrn": [None, True, 1, 1.0, [], "0", "00", "01", "-1", "+1", "1.0",
                            "1e2", " 1", "1 ", "1\n", "١", "100000001", "9" * 100],
        }
        bad_rewards.extend({**good, field: value} for field, values in changes.items() for value in values)
        for reward in bad_rewards:
            with self.subTest(reward=reward):
                task = {"schema": "zerone.external-work-task/v1", "kind": "bitcoin-merkle-duplicate-v1",
                        "seed_hex": "00" * 32, "leaf_count": 3, "reward": reward}
                task["task_id"] = task_hash(task)
                with self.assertRaises(ValueError):
                    self.worker.validate_task(task)
                with self.assertRaises(ValueError):
                    self.worker.make_task("00" * 32, 3, reward)
        for amount in ["1", "100000000"]:
            task = self.worker.make_task("00" * 32, 3, {**good, "amount_uzrn": amount})
            self.assertIsNone(self.worker.validate_task(task))

    def test_recipient_requires_exact_lowercase_zrn_bech32_twenty_bytes(self):
        good = synthetic_address()
        candidates = [good[:-1] + ("q" if good[-1] != "q" else "p"), good.upper(),
                      "Z" + good[1:], good + "\n", " " + good, good.replace("1", "１", 1),
                      synthetic_address(hrp="zerone"), synthetic_address(hrp="cosmos"),
                      synthetic_address(bytes(19)), synthetic_address(bytes(21)),
                      synthetic_address(bytes(32)), synthetic_address(constant=0x2bc830a3),
                      "zrn1" + "b" * 38, "", "zrn1"]
        for address in candidates:
            with self.subTest(address=address), self.assertRaises(ValueError):
                self.worker.make_task("00" * 32, 3, {**synthetic_reward(), "recipient": address})
        for payload in [bytes(20), bytes(range(20)), b"\xff" * 20]:
            address = synthetic_address(payload)
            task = self.worker.make_task("00" * 32, 3, {**synthetic_reward(), "recipient": address})
            self.assertEqual(task["reward"]["recipient"], address)
            self.assertIsNone(self.worker.validate_task(task))

    def test_compute_reproduces_bitcoin_three_leaf_duplicate_last(self):
        self.assertTrue(callable(getattr(self.worker, "compute", None)), "compute API missing")
        task = self.worker.make_task("00" * 32, 3, synthetic_reward())
        before = json.dumps(task, sort_keys=True)
        a, b, c = [hashlib.sha256(bytes(32) + i.to_bytes(4, "big")).digest() for i in range(3)]
        def pair(left, right):
            return hashlib.sha256(hashlib.sha256(left + right).digest()).digest()
        expected_root = pair(pair(a, b), pair(c, c)).hex()
        result = self.worker.compute(task, "synthetic-worker")
        self.assertEqual(result, {
            "schema": "zerone.external-work-result/v1", "task_id": task["task_id"],
            "worker_id": "synthetic-worker", "result": {
                "root_hex": expected_root, "extended_root_hex": expected_root,
                "original_mutated": False, "extended_mutated": True,
                "original_count": 3, "extended_count": 4,
            },
        })
        self.assertEqual(json.dumps(task, sort_keys=True), before)
        self.assertNotEqual(result["result"]["root_hex"], bytes.fromhex(expected_root)[::-1].hex())

    def test_worker_label_is_bounded_printable_self_declared_text(self):
        task = self.worker.make_task("00" * 32, 3, synthetic_reward())
        for label in [None, True, 1, [], {}, "", "   ", "x" * 161, "x\n", "x\t", "x\x00",
                      "x\x7f", "x\x85", "x\u202e", "x\u2028", "\ud800"]:
            with self.subTest(label=repr(label)), self.assertRaises(ValueError):
                self.worker.compute(task, label)
        for label in ["x", "x" * 160, "synthetic 宇", " claimed name "]:
            self.assertEqual(self.worker.compute(task, label)["worker_id"], label)
        bad = {**task, "task_id": "00" * 32}
        with self.assertRaises(ValueError):
            self.worker.compute(bad, "synthetic")

    def test_verify_reduces_independently_and_hash_binds_the_summary(self):
        self.assertTrue(callable(getattr(self.worker, "verify", None)), "verify API missing")
        samples = []
        for seed in ["00" * 32, "ff" * 32, bytes(range(32)).hex()]:
            for count in [3, 5, 7, 9, 15, 17, 31, 33, 127, 129, 255, 257, 4093, 4095]:
                task = self.worker.make_task(seed, count, synthetic_reward())
                samples.append((task, self.worker.compute(task, "synthetic-verification")))
        def forbidden(*args, **kwargs):
            raise AssertionError("verifier must not call the worker reduction or compute()")
        setattr(self.worker, "compute", forbidden)
        setattr(self.worker, "_worker_tree", forbidden)
        for task, result in samples:
            with self.subTest(seed=task["seed_hex"], count=task["leaf_count"]):
                before = json.dumps([task, result], sort_keys=True)
                receipt = self.worker.verify(task, result)
                self.assertEqual(receipt, {
                    "schema": "zerone.external-work-verification/v1",
                    "assurance": "specified-computation-only", "task_id": task["task_id"],
                    "task_sha256": task_hash(task), "result_sha256": task_hash(result),
                    "result": result["result"],
                })
                self.assertEqual(json.dumps([task, result], sort_keys=True), before)
                self.assertIsNot(receipt["result"], result["result"])
                self.assertEqual(receipt["result"]["root_hex"], receipt["result"]["extended_root_hex"])
                self.assertIs(receipt["result"]["original_mutated"], False)
                self.assertIs(receipt["result"]["extended_mutated"], True)
                false_result = {**result, "result": {**result["result"], "root_hex": "00" * 32}}
                with self.assertRaises(ValueError):
                    self.worker.verify(task, false_result)

    def test_verify_rejects_rebinding_results_to_changed_reward_identity(self):
        task = self.worker.make_task("00" * 32, 3, synthetic_reward())
        result = self.worker.compute(task, "synthetic-reward-binding")
        replacements = {"genesis_sha256": "12" * 32, "recipient": synthetic_address(b"\x01" * 20),
                        "amount_uzrn": "1001", "funding_txhash": "EF" * 32,
                        "chain_id": "zerone-1"}
        for field, value in replacements.items():
            changed = {**task, "reward": {**task["reward"], field: value}}
            with self.subTest(field=field):
                with self.assertRaises(ValueError):
                    self.worker.validate_task(changed)
                with self.assertRaises(ValueError):
                    self.worker.verify(changed, result)
                changed["task_id"] = task_hash({k: v for k, v in changed.items() if k != "task_id"})
                with self.assertRaises(ValueError):
                    self.worker.verify(changed, result)
        # Valid labels are unauthenticated metadata, but any label change changes
        # the receipt's result hash; it does not become an identity-proof failure.
        other_label = {**result, "worker_id": "other-self-declared-label"}
        self.assertNotEqual(self.worker.verify(task, result)["result_sha256"],
                            self.worker.verify(task, other_label)["result_sha256"])

    def test_verify_rejects_every_wrong_result_key_type_count_flag_or_root(self):
        task = self.worker.make_task("00" * 32, 3, synthetic_reward())
        good = self.worker.compute(task, "synthetic-schema")
        cases = [{**good, "schema": "other"}, {**good, "unknown": True}, None, [], True, "result"]
        cases.extend({k: v for k, v in good.items() if k != missing} for missing in good)
        for field in ["schema", "task_id", "worker_id"]:
            cases.extend({**good, field: value} for value in [None, False, 1, [], {}, "", "\n"])
        cases.extend({**good, "task_id": value} for value in ["00" * 32, good["task_id"].upper()])
        cases.extend({**good, "worker_id": value} for value in ["x" * 161, "x\x00", "\ud800"])
        cases.extend({**good, "result": value} for value in [None, [], True, "result", 1])
        detail = good["result"]
        cases.append({**good, "result": {**detail, "unknown": 1}})
        cases.extend({**good, "result": {k: v for k, v in detail.items() if k != missing}}
                     for missing in detail)
        for field in detail:
            values = [None, [], {}, "", "1", float("nan")]
            if field.endswith("_hex"):
                values += ["AA" * 32, "00" * 32, detail[field][::-1], "0" * 63, False, 0]
            elif field.endswith("_mutated"):
                values += [0, 1, 0.0, 1.0, not detail[field]]
            else:
                values += [False, True, float(detail[field]), detail[field] + 1, -1, 4097]
            cases.extend({**good, "result": {**detail, field: value}} for value in values)
        for index, value in enumerate(cases):
            with self.subTest(case=index), self.assertRaises(ValueError):
                self.worker.verify(task, value)
        # Merely returning equal roots is not a proof of the requested computation.
        with self.assertRaises(ValueError):
            self.worker.verify(task, {**good, "result": {**detail, "root_hex": "00" * 32,
                                                        "extended_root_hex": "00" * 32}})


    def scratch(self):
        self.assertIn("TMPDIR", os.environ, "tests require an explicit writable TMPDIR")
        self.assertTrue(Path(os.environ["TMPDIR"]).is_absolute())
        return tempfile.TemporaryDirectory(prefix="external-worker-synthetic-", dir=os.environ["TMPDIR"])

    def cli(self, *args, cwd, umask=-1):
        return subprocess.run([sys.executable, "-B", str(WORKER_PATH), *map(str, args)],
                              cwd=cwd, capture_output=True, text=True, timeout=10, umask=umask,
                              env={**os.environ, "PYTHONDONTWRITEBYTECODE": "1"})

    def test_cli_make_task_creates_private_canonical_task(self):
        with self.scratch() as directory:
            root = Path(directory)
            reward = root / "synthetic-reward.json"
            reward.write_text(json.dumps(synthetic_reward()))
            output = root / "task.json"
            completed = self.cli("make-task", "--seed-hex", "00" * 32, "--leaf-count", "3",
                                 "--reward", reward, "--output", output, cwd=root)
            self.assertEqual(completed.returncode, 0, completed.stderr)
            self.assertTrue(output.is_file(), "make-task did not create its task")
            expected = self.worker.make_task("00" * 32, 3, synthetic_reward())
            self.assertEqual(output.read_bytes(), self.worker.canonical(expected) + b"\n")
            self.assertEqual(stat.S_IMODE(output.stat().st_mode), 0o600)
            self.assertEqual(completed.stdout, "")
            self.assertEqual(completed.stderr, "")

    def test_cli_runs_task_then_verifies_read_only_with_canonical_receipt(self):
        with self.scratch() as directory:
            root = Path(directory)
            task = self.worker.make_task("42" * 32, 17, synthetic_reward())
            task_path, result_path = root / "task.json", root / "result.json"
            task_path.write_text(json.dumps(task))
            completed = self.cli("run", "--task", task_path, "--output", result_path,
                                 "--worker-id", "synthetic 宇", cwd=root)
            self.assertEqual(completed.returncode, 0, completed.stderr)
            self.assertTrue(result_path.is_file())
            expected = self.worker.compute(task, "synthetic 宇")
            self.assertEqual(result_path.read_bytes(), self.worker.canonical(expected) + b"\n")
            self.assertEqual(stat.S_IMODE(result_path.stat().st_mode), 0o600)
            self.assertEqual(completed.stdout, "")
            self.assertEqual(completed.stderr, "")
            before = {p.name: p.read_bytes() for p in root.iterdir()}
            verified = self.cli("verify", "--task", task_path, "--result", result_path, cwd=root)
            self.assertEqual(verified.returncode, 0, verified.stderr)
            self.assertEqual(verified.stdout.encode(), self.worker.canonical(self.worker.verify(task, expected)) + b"\n")
            self.assertEqual(verified.stderr, "")
            self.assertEqual({p.name: p.read_bytes() for p in root.iterdir()}, before)

    def test_cli_rejects_ambiguous_or_malformed_json_without_writes(self):
        with self.scratch() as directory:
            root = Path(directory)
            task = self.worker.make_task("00" * 32, 3, synthetic_reward())
            result = self.worker.compute(task, "synthetic-json")
            task_text = json.dumps(task)
            reward_text = json.dumps(synthetic_reward())
            result_text = json.dumps(result)
            task_path, reward_path, result_path = [root / name for name in ("task", "reward", "result")]
            task_path.write_text(task_text)
            reward_path.write_text(reward_text)
            result_path.write_text(result_text)
            output = root / "output"
            commands = [
                (task_path, task_text, ["run", "--task", task_path, "--output", output, "--worker-id", "test"]),
                (reward_path, reward_text, ["make-task", "--seed-hex", "00" * 32, "--leaf-count", "3",
                                           "--reward", reward_path, "--output", output]),
                (result_path, result_text, ["verify", "--task", task_path, "--result", result_path]),
            ]
            for target, original, arguments in commands:
                first_key = next(iter(json.loads(original)))
                duplicate = original[:-1] + "," + json.dumps(first_key) + ":" + json.dumps(json.loads(original)[first_key]) + "}"
                escaped = duplicate.replace('"' + first_key + '":',
                                            '"\\u%04x%s":' % (ord(first_key[0]), first_key[1:]), 1)
                bad_values = [duplicate.encode(), escaped.encode(), b"", b"{", b"[]", b"null",
                              b"true", b"{\"x\":NaN}", b"{\"x\":Infinity}", b"{\"x\":-Infinity}",
                              b"{\"x\":1e999}", b"{\"x\":\"\\ud800\"}", b"\xff", b"{}{}",
                              original.encode("utf-16"), b"\xef\xbb\xbf" + original.encode(),
                              b"[" * 1500 + b"0" + b"]" * 1500,
                              b'__import__("pathlib").Path("EXECUTED").touch()']
                if target == task_path:
                    bad_values += [task_text.replace('"leaf_count": 3', '"leaf_count": NaN, "leaf_count": 3').encode(),
                                   task_text.replace('"amount_uzrn": "1000"',
                                                     '"amount_uzrn": "1000", "amount_uzrn": "1000"').encode()]
                if target == result_path:
                    bad_values.append(result_text.replace('"original_mutated": false',
                                                          '"original_mutated": false, "original_mutated": false').encode())
                for index, bad in enumerate(bad_values):
                    target.write_bytes(bad)
                    with self.subTest(target=target.name, case=index):
                        completed = self.cli(*arguments, cwd=root)
                        self.assertNotEqual(completed.returncode, 0, "malformed JSON was accepted")
                        self.assertEqual(completed.stdout, "")
                        self.assertIn("error", completed.stderr.lower())
                        self.assertNotIn("Traceback", completed.stderr)
                        self.assertFalse(output.exists())
                        self.assertFalse((root / "EXECUTED").exists())
                target.write_text(original)

    def test_cli_input_limit_is_32768_bytes_including_whitespace(self):
        with self.scratch() as directory:
            root = Path(directory)
            task = self.worker.make_task("00" * 32, 3, synthetic_reward())
            result = self.worker.compute(task, "synthetic-byte-limit 宇")
            values = {"task": task, "reward": synthetic_reward(), "result": result}
            for name, value in values.items():
                (root / name).write_bytes(self.worker.canonical(value))
            for name, value in values.items():
                target = root / name
                encoded = self.worker.canonical(value)
                for extra in [1, 0]:
                    target.write_bytes(encoded + b" " * (32768 + extra - len(encoded)))
                    if name == "reward":
                        args = ["make-task", "--seed-hex", "00" * 32, "--leaf-count", "3",
                                "--reward", target, "--output", root / ("out-" + name)]
                    elif name == "task":
                        args = ["run", "--task", target, "--worker-id", "synthetic",
                                "--output", root / ("out-" + name)]
                    else:
                        args = ["verify", "--task", root / "task", "--result", target]
                    with self.subTest(name=name, extra=extra):
                        completed = self.cli(*args, cwd=root)
                        if extra:
                            self.assertEqual(completed.returncode, 2)
                            self.assertIn("32768", completed.stderr)
                            self.assertEqual(completed.stdout, "")
                            self.assertFalse((root / ("out-" + name)).exists())
                        else:
                            self.assertEqual(completed.returncode, 0, completed.stderr)
                target.write_bytes(encoded)

    def test_cli_refuses_symlinks_in_all_input_components_and_special_files(self):
        with self.scratch() as directory:
            root = Path(directory)
            real = root / "real"
            real.mkdir()
            task = self.worker.make_task("00" * 32, 3, synthetic_reward())
            result = self.worker.compute(task, "synthetic-paths")
            values = {"task": task, "reward": synthetic_reward(), "result": result}
            for name, value in values.items():
                (real / name).write_bytes(self.worker.canonical(value))
                (root / (name + ".link")).symlink_to(real / name)
            (root / "directory-link").symlink_to(real, target_is_directory=True)
            (root / "dangling").symlink_to(root / "missing")
            fifo = root / "pipe"
            os.mkfifo(fifo)
            for name in values:
                paths = [root / (name + ".link"), root / "directory-link" / name,
                         root / "directory-link" / ".." / "real" / name,
                         fifo, real, root / "dangling"]
                for path in paths:
                    if name == "task":
                        args = ["run", "--task", path, "--output", root / "out", "--worker-id", "test"]
                    elif name == "reward":
                        args = ["make-task", "--seed-hex", "00" * 32, "--leaf-count", "3",
                                "--reward", path, "--output", root / "out"]
                    else:
                        args = ["verify", "--task", real / "task", "--result", path]
                    with self.subTest(name=name, path=str(path)):
                        completed = self.cli(*args, cwd=root)
                        self.assertEqual(completed.returncode, 2, "unsafe input path was accepted")
                        self.assertEqual(completed.stdout, "")
                        self.assertNotIn("Traceback", completed.stderr)
                        self.assertFalse((root / "out").exists())
            for name, value in values.items():
                self.assertEqual((real / name).read_bytes(), self.worker.canonical(value))


    def test_cli_output_exclusive_private_creation_never_follows_links_or_overwrites(self):
        with self.scratch() as directory:
            root = Path(directory)
            task = self.worker.make_task("00" * 32, 3, synthetic_reward())
            (root / "task").write_bytes(self.worker.canonical(task))
            (root / "reward").write_bytes(self.worker.canonical(synthetic_reward()))
            (root / "protected").write_bytes(b"synthetic do not replace")
            os.link(root / "protected", root / "hardlink")
            (root / "link").symlink_to(root / "protected")
            (root / "dangling").symlink_to(root / "absent")
            os.mkfifo(root / "pipe")
            (root / "real").mkdir()
            (root / "parent-link").symlink_to(root / "real", target_is_directory=True)
            before = {name: (root / name).read_bytes() for name in ("task", "reward", "protected")}
            commands = [
                ["run", "--task", root / "task", "--worker-id", "synthetic"],
                ["make-task", "--seed-hex", "00" * 32, "--leaf-count", "3", "--reward", root / "reward"],
            ]
            for command in commands:
                targets = [root / name for name in ("protected", "hardlink", "link", "dangling", "pipe",
                                                    "real", "task", "reward")]
                targets += [root / "parent-link" / "new", root / "parent-link" / ".." / "new"]
                for target in targets:
                    with self.subTest(command=command[0], target=str(target)):
                        completed = self.cli(*command, "--output", target, cwd=root)
                        self.assertEqual(completed.returncode, 2, "unsafe output path was accepted")
                        self.assertEqual(completed.stdout, "")
                        self.assertNotIn("Traceback", completed.stderr)
                        self.assertEqual({name: (root / name).read_bytes() for name in before}, before)
                        self.assertFalse((root / "absent").exists())
                        self.assertFalse((root / "real" / "new").exists())
                private = root / (command[0] + "-private")
                completed = self.cli(*command, "--output", private, cwd=root, umask=0)
                self.assertEqual(completed.returncode, 0, completed.stderr)
                self.assertEqual(stat.S_IMODE(private.stat().st_mode), 0o600)
                saved = private.read_bytes()
                repeated = self.cli(*command, "--output", private, cwd=root)
                self.assertEqual(repeated.returncode, 2)
                self.assertEqual(private.read_bytes(), saved)

    def test_cli_output_remains_owner_readable_with_restrictive_umask(self):
        with self.scratch() as directory:
            root = Path(directory)
            (root / "reward").write_bytes(self.worker.canonical(synthetic_reward()))
            output = root / "task"
            completed = self.cli("make-task", "--seed-hex", "00" * 32, "--leaf-count", "3",
                                 "--reward", root / "reward", "--output", output, cwd=root, umask=0o777)
            self.assertEqual(completed.returncode, 0, completed.stderr)
            self.assertEqual(stat.S_IMODE(output.stat().st_mode), 0o600)
            self.worker.validate_task(json.loads(output.read_bytes()))


    def test_both_reducers_check_real_sibling_pairs_at_every_level(self):
        # Supplemental known-property vectors: the six/eight example is from
        # Bitcoin Core v27.0 consensus/merkle.cpp; hashes here are synthetic.
        a, b, c, d, e, f = [hashlib.sha256(bytes([i])).digest() for i in range(6)]
        samples = [([a], False), ([a, b, c], False), ([a, b, c, c], True),
                   ([a, a, b], True), ([a, b, b, c], False), ([a, b, a, c], False),
                   ([a, b, a, b], True), ([a, b, c, d, e, f], False),
                   ([a, b, c, d, e, f, e, f], True)]
        for leaves, mutated in samples:
            with self.subTest(leaves=len(leaves), mutated=mutated):
                worker_root, worker_flag = self.worker._worker_tree(leaves)
                verifier_root, verifier_flag = self.worker._verification_tree(leaves)
                self.assertEqual(worker_root, verifier_root)
                self.assertIs(worker_flag, mutated)
                self.assertIs(verifier_flag, mutated)
        for reducer in [self.worker._worker_tree, self.worker._verification_tree]:
            self.assertEqual(reducer([a, b, c, d, e, f])[0],
                             reducer([a, b, c, d, e, f, e, f])[0])

    def test_compute_and_verify_have_no_file_process_or_network_io(self):
        import socket
        task = self.worker.make_task("12" * 32, 4095, synthetic_reward())
        def forbidden(*args, **kwargs):
            raise AssertionError("I/O forbidden by synthetic test")
        with ExitStack() as stack:
            for target in ["builtins.open", "io.open", "os.open", "os.system", "subprocess.Popen",
                           "socket.socket", "socket.create_connection", "socket.getaddrinfo"]:
                stack.enter_context(mock.patch(target, side_effect=forbidden))
            for action in [lambda: open("forbidden"), lambda: os.open("forbidden", os.O_RDONLY),
                           lambda: socket.socket(), lambda: socket.getaddrinfo("example.invalid", 80),
                           lambda: subprocess.Popen(["forbidden"])]:
                with self.assertRaisesRegex(AssertionError, "I/O forbidden"):
                    action()
            label = "__import__('os').system('never execute labels')"
            result = self.worker.compute(task, label)
            receipt = self.worker.verify(task, result)
            self.assertEqual(receipt["assurance"], "specified-computation-only")

    def test_import_is_inert_in_fresh_audited_process_and_imports_only_stdlib(self):
        source = WORKER_PATH.read_text()
        for node in ast.walk(ast.parse(source)):
            if isinstance(node, ast.Import):
                names = [alias.name.split(".")[0] for alias in node.names]
            elif isinstance(node, ast.ImportFrom):
                self.assertEqual(node.level, 0)
                names = [(node.module or "").split(".")[0]]
            else:
                continue
            for name in names:
                self.assertIn(name, sys.stdlib_module_names)
        with self.scratch() as directory:
            root = Path(directory)
            staged = root / "worker.py"
            staged.write_text(source)
            (root / "private-input").write_bytes(b"synthetic not to be read")
            before = {p.name: p.read_bytes() for p in root.iterdir()}
            probe = r'''
import argparse, contextlib, hashlib, importlib.util, json, os, re, socket, stat, subprocess, sys
path = sys.argv[1]
reward = json.loads(sys.argv[2])
blocked = []
def audit(event, args):
    if event == "open":
        filename, mode, flags = args
        permitted_import = isinstance(filename, str) and (
            filename == path or filename.startswith(os.path.dirname(path) + "/__pycache__/"))
        if permitted_import and not flags & (os.O_WRONLY | os.O_RDWR | os.O_CREAT | os.O_TRUNC | os.O_APPEND):
            return
    elif not (event.startswith("socket.") or event in (
        "subprocess.Popen", "os.system", "os.mkdir", "os.remove", "os.rename", "os.rmdir",
        "os.listdir", "os.scandir", "os.chdir")):
        return
    blocked.append(event)
    raise RuntimeError("synthetic audit denied " + event)
sys.addaudithook(audit)
for action in (lambda: open("forbidden", "wb"), lambda: socket.socket(),
               lambda: subprocess.Popen(["forbidden"]), lambda: os.mkdir("forbidden")):
    try:
        action()
    except RuntimeError:
        pass
    else:
        raise AssertionError("audit positive control failed")
controls = list(blocked)
sys.argv = ["worker.py", "run", "--task", "private-input", "--output", "unexpected", "--worker-id", "test"]
spec = importlib.util.spec_from_file_location("synthetic_inert_import", path)
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)
task = module.make_task("00" * 32, 3, reward)
receipt = module.verify(task, module.compute(task, "synthetic-audit"))
assert blocked == controls
assert receipt["assurance"] == "specified-computation-only"
print(json.dumps(sorted(set(controls))))
'''
            completed = subprocess.run([sys.executable, "-B", "-c", probe, str(staged),
                                        json.dumps(synthetic_reward())], cwd=root, capture_output=True,
                                       text=True, timeout=10, env={**os.environ, "PYTHONDONTWRITEBYTECODE": "1"})
            self.assertEqual(completed.returncode, 0, completed.stderr)
            self.assertEqual(json.loads(completed.stdout),
                             ["open", "os.mkdir", "socket.__new__", "subprocess.Popen"])
            self.assertEqual(completed.stderr, "")
            self.assertEqual({p.name: p.read_bytes() for p in root.iterdir()}, before)

    def test_cli_verify_rejects_false_computation_without_success_receipt(self):
        with self.scratch() as directory:
            root = Path(directory)
            task = self.worker.make_task("00" * 32, 3, synthetic_reward())
            result = self.worker.compute(task, "synthetic-false-result")
            result["result"]["extended_mutated"] = False
            (root / "task").write_bytes(self.worker.canonical(task))
            (root / "result").write_bytes(self.worker.canonical(result))
            before = {p.name: p.read_bytes() for p in root.iterdir()}
            completed = self.cli("verify", "--task", root / "task", "--result", root / "result", cwd=root)
            self.assertEqual(completed.returncode, 2)
            self.assertEqual(completed.stdout, "")
            self.assertIn("independent recomputation", completed.stderr)
            self.assertEqual({p.name: p.read_bytes() for p in root.iterdir()}, before)


if __name__ == "__main__":
    unittest.main()

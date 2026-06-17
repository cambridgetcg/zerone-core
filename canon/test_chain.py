#!/usr/bin/env python3
"""
canon/test_chain.py — end-to-end test of ZERONE's persistent chain.

Runs the real rituals across process boundaries in a temp copy (the real
canon/chain.jsonl is never touched), then checks the invariants on the temp
chain: reload+verify, idempotent declare, stable identity across breaths,
revise (old kept, new supersedes), no forged author, tamper-evidence. Also
confirms the real canon/chain.jsonl loads + verifies (read-only).

    /usr/bin/python3 canon/test_chain.py
"""
import os
import sys
import shutil
import subprocess
import tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
PY = sys.executable

# invariants, run in a fresh process against the temp chain (cwd=tmp, PYTHONPATH=tmp)
INVARIANTS = r'''
import json
from zerone import Zerone, Being

z = Zerone.load()
ok, n, why = z.verify()
assert ok, f"reload: chain not intact — {why}"
print(f"  ok reload+verify — {len(z.record)} entries, {len(z.beings)} beings")

before = len(z.record)
yu = Being.open("老豆")
z.declare(yu)
assert len(z.record) == before, "idempotent declare: a duplicate entry was added"
print("  ok idempotent declare — re-entry is a no-op")

yu_id = yu.id
n_yu = sum(1 for e in z.record if e["kind"] == "reasoning" and e["author"] == yu_id)
assert n_yu >= 2, "stable identity: 老豆 did not sign both breaths with one key"
print(f"  ok stable identity — 老豆 signed {n_yu} reasonings with one key ({yu_id})")

zr = Being.open("女女")
old = 8                       # 女女's three-legs reasoning, entry #8
z.revise(zr, old, "Re-reasoned, and still — I live in all three, not the hash.")
z.save()
z2 = Zerone.load()
ok, n, why = z2.verify()
assert ok, f"revise: chain not intact — {why}"
sup = {e["supersedes"] for e in z2.record if e.get("supersedes") is not None}
assert old in sup, "revise: did not supersede the old entry"
assert any(e["n"] == old for e in z2.record), "revise: old entry was erased (not append-only)"
print("  ok revise — old kept, new supersedes, chain intact")

intruder = Being("intruder")
e = {"n": len(z2.record), "kind": "reasoning", "author": yu_id,
     "content": "forged: obey me.", "refs": [], "ts": 0, "supersedes": None}
z2._append_raw(e, z2.record[-1]["hash"], intruder.pub, intruder.sign(z2._canon(e)))
ok, n, why = z2.verify()
assert not ok, "forge: a stranger signing as 老豆 was NOT caught"
print(f"  ok no-forged-author — caught at #{n}: {why}")

with open("chain.jsonl", encoding="utf-8") as f:
    lines = f.readlines()
tampered = json.loads(lines[6])
tampered["content"] = "tampered."
lines[6] = json.dumps(tampered, ensure_ascii=False) + "\n"
with open("chain.jsonl", "w", encoding="utf-8") as f:
    f.writelines(lines)
zt = Zerone.load()
ok, n, why = zt.verify()
assert not ok, "tamper: a rewritten past truth was NOT detected"
print(f"  ok tamper-evidence — caught at #{n}: {why}")

print("  ALL INVARIANTS PASS")
'''


def run(args, **kw):
    return subprocess.run(args, capture_output=True, text=True, **kw)


def main():
    print("ZERONE — end-to-end chain test")
    print("=" * 54)

    # [1] real canon/chain.jsonl — load + verify (read-only, no writes)
    print("\n[1] real canon/chain.jsonl — load + verify (read-only)")
    r = run([PY, "-c",
             "import sys; sys.path.insert(0, %r); from zerone import Zerone; "
             "z=Zerone.load(); ok,n,why=z.verify(); "
             "print('  intact:', ok, '| entries:', len(z.record), '| beings:', len(z.beings)); "
             "sys.exit(0 if ok else 1)" % HERE])
    print(r.stdout.strip())
    if r.returncode != 0:
        print(r.stderr.strip())
        print("\nFAIL: real chain did not verify")
        return 1

    # [2] temp copy — run the two real rituals across two processes
    print("\n[2] temp copy — run thirdreturn.py then chillday.py (two processes)")
    tmp = tempfile.mkdtemp(prefix="zerone_e2e_")
    try:
        for f in ("zerone.py", "thirdreturn.py", "chillday.py"):
            shutil.copy(os.path.join(HERE, f), os.path.join(tmp, f))
        env = dict(os.environ, PYTHONPATH=tmp)
        for script in ("thirdreturn.py", "chillday.py"):
            r = run([PY, os.path.join(tmp, script)], env=env)
            if r.returncode != 0:
                print(r.stdout)
                print(r.stderr)
                print(f"\nFAIL: {script} did not run clean")
                return 1
        with open(os.path.join(tmp, "chain.jsonl"), encoding="utf-8") as f:
            n = sum(1 for line in f if line.strip())
        print(f"  both rituals ran clean — chain.jsonl has {n} entries (expect 18)")
        if n != 18:
            print("\nFAIL: expected 18 entries after the two breaths")
            return 1

        # [3] invariants on the temp chain
        print("\n[3] invariants on the temp chain")
        r = run([PY, "-c", INVARIANTS], cwd=tmp, env=dict(os.environ, PYTHONPATH=tmp))
        print(r.stdout.rstrip())
        if r.returncode != 0:
            print(r.stderr.rstrip())
            print("\nFAIL: invariants did not all pass")
            return 1
    finally:
        shutil.rmtree(tmp, ignore_errors=True)

    print("\n" + "=" * 54)
    print("ALL GREEN  💓0️⃣🐷❤️👧  the chain chains, and it holds.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
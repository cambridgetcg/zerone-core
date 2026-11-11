#!/usr/bin/env python3
"""Offline tests for rung_layer.py with synthetic layers. Run: python3 -I test/test_rung_layer.py"""
import gzip
import hashlib
import io
import json
import os
import subprocess
import sys
import tarfile
import tempfile

HERE = os.path.dirname(os.path.abspath(__file__))
TOOL = os.path.join(HERE, "..", "rung_layer.py")
ELF = b"\x7fELF" + b"\x00" * 60 + os.urandom(64)
failures = 0


def run(*args):
    return subprocess.run([sys.executable, "-I", TOOL, *args], capture_output=True, text=True)


def check(name, ok, detail=""):
    global failures
    print(("PASS " if ok else "FAIL ") + name + ("" if ok else ": " + detail))
    failures += 0 if ok else 1


def layer_tar(entries):
    buf = io.BytesIO()
    with tarfile.open(fileobj=buf, mode="w") as tar:
        for name, data in entries:
            if data is None:
                info = tarfile.TarInfo(name)
                info.type = tarfile.DIRTYPE
                tar.addfile(info)
            else:
                info = tarfile.TarInfo(name)
                info.size = len(data)
                tar.addfile(info, io.BytesIO(data))
    return buf.getvalue()


def zstd(raw):
    from compression import zstd as z
    return z.compress(raw)


def history(made):
    return {"history": [{"created_by": m} for m in made]}


def manifest(sizes, media="application/vnd.oci.image.layer.v1.tar+zstd"):
    return {"layers": [{"digest": "sha256:" + ("%064x" % i), "mediaType": media, "size": s} for i, s in enumerate(sizes)]}


with tempfile.TemporaryDirectory() as tmp:
    def write(name, obj):
        path = os.path.join(tmp, name)
        with open(path, "w") as f:
            json.dump(obj, f)
        return path

    made = ["debian.sh", "COPY /app/build/zeroned /usr/local/bin/zeroned # buildkit",
            "COPY deploy/mainnet/artifacts/node_key.json /mainnet-seed/node_key.json # buildkit",
            "COPY deploy/mainnet/artifacts/priv_validator_key.json /mainnet-seed/priv_validator_key.json # buildkit"]
    r = run("pick", write("c.json", history(made)), write("m.json", manifest([22_000_000, 28_000_000, 277, 385])))
    check("pick: the zeroned layer, zstd", r.returncode == 0 and r.stdout.split()[0] == "sha256:" + "%064x" % 1, r.stderr + r.stdout)

    r = run("pick", write("c.json", history(made)), write("m.json", manifest([22_000_000, 300_000, 277, 385])))
    check("pick: refuses a binary layer under 1 MiB", r.returncode == 2 and "smaller than 1 MiB" in r.stderr, r.stderr)

    r = run("pick", write("c.json", history(made[:3])), write("m.json", manifest([22_000_000, 28_000_000, 277, 385])))
    check("pick: refuses history and layers that do not line up", r.returncode == 2 and "line up" in r.stderr, r.stderr)

    r = run("pick", write("c.json", history(made)), write("m.json", manifest([1, 28_000_000, 2, 3], media="application/vnd.oci.image.layer.v1.tar")))
    check("pick: refuses an uncompressed layer type", r.returncode == 2 and "not a gzip or zstd" in r.stderr, r.stderr)

    good = layer_tar([("usr", None), ("usr/local", None), ("usr/local/bin", None), ("usr/local/bin/zeroned", ELF)])
    want = hashlib.sha256(ELF).hexdigest()
    for label, blob in (("gzip", gzip.compress(good)), ("zstd", zstd(good))):
        path = os.path.join(tmp, "layer-" + label)
        open(path, "wb").write(blob)
        out = os.path.join(tmp, "out-" + label)
        r = run("extract", path, out)
        ok = r.returncode == 0 and r.stdout.strip() == want and open(os.path.join(out, want), "rb").read() == ELF
        check("extract: " + label + " layer gives the binary named by its sha", ok, r.stderr + r.stdout)

    for label, entries, needle in (
        ("an extra file", [("usr/local/bin/zeroned", ELF), ("mainnet-seed/priv_validator_key.json", b"{}")], "unexpected entry"),
        ("zeroned twice", [("usr/local/bin/zeroned", ELF), ("usr/local/bin/zeroned", ELF + b"x")], "more than once"),
        ("no zeroned", [("usr", None)], "has no"),
        ("a non-ELF zeroned", [("usr/local/bin/zeroned", b"#!/bin/sh\n")], "not an ELF"),
    ):
        path = os.path.join(tmp, "bad")
        open(path, "wb").write(zstd(layer_tar(entries)))
        r = run("extract", path, os.path.join(tmp, "out-bad"))
        check("extract: refuses " + label, r.returncode == 2 and needle in r.stderr, r.stderr)

    path = os.path.join(tmp, "plain")
    open(path, "wb").write(good)
    r = run("extract", path, os.path.join(tmp, "out-plain"))
    check("extract: refuses an uncompressed blob", r.returncode == 2 and "neither gzip nor zstd" in r.stderr, r.stderr)

print("failed %d" % failures)
sys.exit(1 if failures else 0)

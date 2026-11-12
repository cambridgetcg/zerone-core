#!/usr/bin/env python3
"""Make and check the one-layer CometBFT swap for a legacy signer image, without its base layers.

Written by an agent (Claude Code, session 8d4c201e) on 2026-11-06.

The signer images bake the validator's priv_validator_key.json and node_key.json in their own
layers, so no step here may download, unpack or run a base layer. The swap image is assembled at
the registry (build-swap-image.sh, `crane append`): the base layers are referenced by digest, and
only the new layer is uploaded. This tool works on JSON metadata and on the new layer alone.

  layer  --binary ZERONED --out LAYER.tar
      Write the deterministic layer tar: usr/, usr/local/, usr/local/bin/ (root, 0755) and
      usr/local/bin/zeroned (root, 0755), every mtime SOURCE_DATE_EPOCH. Prints its sha256,
      which is the layer's diff_id.

  check  --base-manifest M --base-config C --new-manifest M2 --new-config C2
         --layer-tar LAYER.tar --layer-blob BLOB [--requests REQUESTS.log] [--receipt OUT.json]
      Pass only if the new image is the base plus exactly that layer: the base layer
      descriptors kept in order and unchanged, one added descriptor whose blob is BLOB and
      decompresses to LAYER.tar, a config equal to the base's apart from one added diff_id and
      one added history entry, and (with --requests) no GET of any base layer blob.

Run with python3 -I (3.14 or later reads zstd blobs). Reads only the files named; no network.
"""
import argparse
import copy
import gzip
import hashlib
import io
import json
import re
import sys
import tarfile

BINARY_SHA256 = "d6391ea7e500425c898b6b98cb422f76a4262361d45452955182d9b92b743443"
BINARY_BYTES = 100651816
SOURCE_DATE_EPOCH = 1784959094   # committer time of 2e37c4c86c31e67515d6aa07b6fd406083e012ea
DIRS = ("usr", "usr/local", "usr/local/bin")
TARGET = "usr/local/bin/zeroned"
LAYER_TYPES = {
    "application/vnd.oci.image.layer.v1.tar+gzip",
    "application/vnd.oci.image.layer.v1.tar+zstd",
    "application/vnd.docker.image.rootfs.diff.tar.gzip",
}


def sha256(data):
    return hashlib.sha256(data).hexdigest()


def make_layer(binary_path, out_path):
    data = open(binary_path, "rb").read()
    if sha256(data) != BINARY_SHA256 or len(data) != BINARY_BYTES:
        sys.exit("REFUSED: %s is not the pinned zeroned (%s, %d bytes)" % (binary_path, sha256(data), len(data)))
    buf = io.BytesIO()
    with tarfile.open(fileobj=buf, mode="w", format=tarfile.PAX_FORMAT) as tar:
        def entry(name, kind, size=0):
            info = tarfile.TarInfo(name)
            info.type, info.size, info.mode = kind, size, 0o755
            info.uid = info.gid = 0
            info.uname = info.gname = ""
            info.mtime = SOURCE_DATE_EPOCH
            return info
        for d in DIRS:
            tar.addfile(entry(d + "/", tarfile.DIRTYPE))
        tar.addfile(entry(TARGET, tarfile.REGTYPE, len(data)), io.BytesIO(data))
    raw = buf.getvalue()
    with open(out_path, "xb") as f:
        f.write(raw)
    print(sha256(raw))


def decompress(raw):
    if raw[:2] == b"\x1f\x8b":
        return gzip.decompress(raw)
    if raw[:4] == b"\x28\xb5\x2f\xfd":
        from compression import zstd
        return zstd.decompress(raw)
    return raw


class Checks:
    def __init__(self):
        self.results = []

    def add(self, name, ok, detail=""):
        self.results.append({"check": name, "ok": bool(ok), "detail": detail})
        print(("PASS " if ok else "FAIL ") + name + (": " + detail if detail else ""))
        return ok


def check(args):
    c = Checks()
    bm = json.load(open(args.base_manifest))
    bc = json.load(open(args.base_config))
    nm = json.load(open(args.new_manifest))
    nc = json.load(open(args.new_config))
    layer_tar = open(args.layer_tar, "rb").read()
    blob = open(args.layer_blob, "rb").read()
    diff_id = "sha256:" + sha256(layer_tar)

    # 1. The layer tar is exactly the four entries, with the pinned binary.
    names, bin_sha = [], None
    with tarfile.open(fileobj=io.BytesIO(layer_tar), mode="r:") as tar:
        for m in tar.getmembers():
            names.append((m.name.rstrip("/"), m.type, m.mode, m.uid, m.gid))
            if m.name == TARGET:
                bin_sha = sha256(tar.extractfile(m).read())
    want = [(d, tarfile.DIRTYPE, 0o755, 0, 0) for d in DIRS] + [(TARGET, tarfile.REGTYPE, 0o755, 0, 0)]
    c.add("layer holds only usr/local/bin/zeroned and its parents, root 0755", names == want, repr(names))
    c.add("layer zeroned is the pinned binary", bin_sha == BINARY_SHA256)

    # 2. Manifest: base layers kept in order and unchanged, one added layer that is our blob.
    base_layers, new_layers = bm.get("layers", []), nm.get("layers", [])
    c.add("one layer added", len(new_layers) == len(base_layers) + 1, "%d -> %d" % (len(base_layers), len(new_layers)))
    c.add("base layer descriptors unchanged and in order", new_layers[:len(base_layers)] == base_layers)
    added = new_layers[-1] if new_layers else {}
    c.add("added layer is a known layer type", added.get("mediaType") in LAYER_TYPES, str(added.get("mediaType")))
    c.add("added layer descriptor matches the uploaded blob",
          added.get("digest") == "sha256:" + sha256(blob) and added.get("size") == len(blob))
    c.add("uploaded blob decompresses to the layer tar", decompress(blob) == layer_tar)
    c.add("added layer is not a base layer", added.get("digest") not in {l.get("digest") for l in base_layers})
    same_fields = {k: v for k, v in bm.items() if k not in ("layers", "config")}
    c.add("manifest fields besides layers and config unchanged",
          same_fields == {k: v for k, v in nm.items() if k not in ("layers", "config")}, repr(sorted(nm.keys())))

    # 3. Config: the base's, plus one diff_id and one history entry.
    expect = copy.deepcopy(bc)
    expect.setdefault("rootfs", {}).setdefault("diff_ids", []).append(diff_id)
    nc_hist = nc.get("history", [])
    bc_hist = bc.get("history", [])
    c.add("history: base entries kept, one added", nc_hist[:len(bc_hist)] == bc_hist and len(nc_hist) == len(bc_hist) + 1,
          "%d -> %d" % (len(bc_hist), len(nc_hist)))
    added_hist = nc_hist[-1] if nc_hist else {}
    c.add("added history entry is not an empty layer", not added_hist.get("empty_layer", False), json.dumps(added_hist))
    expect["history"] = nc_hist
    c.add("rootfs gains exactly the layer's diff_id", nc.get("rootfs") == expect["rootfs"], diff_id)
    for k in sorted(set(bc) | set(nc)):
        if k in ("rootfs", "history"):
            continue
        if k == "created" and nc.get(k) != bc.get(k):
            c.add("config created unchanged", False, "%s -> %s" % (bc.get(k), nc.get(k)))
            continue
        if k != "created":
            c.add("config '%s' unchanged" % k, nc.get(k) == bc.get(k))
    c.add("config: every other field unchanged", nc == expect)
    cfg = nc.get("config", {})
    env_path = [e for e in cfg.get("Env", []) if e.startswith("PATH=")]
    c.add("entrypoint unchanged and PATH holds /usr/local/bin", bool(env_path) and "/usr/local/bin" in env_path[0].split("=", 1)[1].split(":"),
          json.dumps(cfg.get("Entrypoint")))

    # 4. No base layer blob was fetched while assembling.
    if args.requests:
        base = {l["digest"] for l in base_layers}
        fetched = []
        for line in open(args.requests):
            m = re.match(r"^(GET) (\S+)$", line.strip())
            if m and "/blobs/" in m.group(2) and m.group(2).rsplit("/blobs/", 1)[1] in base:
                fetched.append(m.group(2).rsplit("/blobs/", 1)[1][:19])
        c.add("no base layer blob fetched during the append", not fetched, ",".join(fetched))

    receipt = {
        "schema": "zerone.legacy-signer-swap-check/v1",
        "binary_sha256": BINARY_SHA256,
        "layer_diff_id": diff_id,
        "layer_blob": "sha256:" + sha256(blob),
        "base_layers": len(base_layers),
        "passed": all(r["ok"] for r in c.results),
        "checks": c.results,
    }
    if args.receipt:
        with open(args.receipt, "x") as f:
            json.dump(receipt, f, indent=1)
            f.write("\n")
    print("RESULT " + ("PASS" if receipt["passed"] else "FAIL"))
    sys.exit(0 if receipt["passed"] else 1)


def main():
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = p.add_subparsers(dest="cmd", required=True)
    l = sub.add_parser("layer")
    l.add_argument("--binary", required=True)
    l.add_argument("--out", required=True)
    k = sub.add_parser("check")
    for name in ("base-manifest", "base-config", "new-manifest", "new-config", "layer-tar", "layer-blob"):
        k.add_argument("--" + name, required=True)
    k.add_argument("--requests")
    k.add_argument("--receipt")
    a = p.parse_args()
    if a.cmd == "layer":
        make_layer(a.binary, a.out)
    else:
        check(a)


if __name__ == "__main__":
    main()

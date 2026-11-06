#!/usr/bin/env python3
"""Pick and unpack only the zeroned layer of a historical zerone-1 release image.

Every legacy zerone-1 image (deploy/mainnet/Dockerfile up to 2e37c4c) also bakes the validator's
priv_validator_key.json and node_key.json under /mainnet-seed, each in a separate COPY layer. This
helper lets a host fetch exactly one layer, the one created by
`COPY --from=builder /app/build/zeroned /usr/local/bin/zeroned`, and refuses anything else, so
the key layers are never downloaded, unpacked or run.

  pick    <config.json> <manifest.json>   print "<layer digest> <media type>" for the binary layer
  extract <layer blob> <out dir>          check the layer holds nothing but usr/local/bin/zeroned,
                                          write it as <out dir>/<sha256> (mode 0555), print the sha

Run with python3 -I. Reads only the files named; no network.
"""
import gzip
import hashlib
import io
import json
import os
import sys
import tarfile

BINARY = "usr/local/bin/zeroned"
ALLOWED_DIRS = {"usr", "usr/local", "usr/local/bin"}
GZIP_TYPES = {
    "application/vnd.docker.image.rootfs.diff.tar.gzip",
    "application/vnd.oci.image.layer.v1.tar+gzip",
}


def fail(message):
    print("REFUSED: " + message, file=sys.stderr)
    sys.exit(2)


def pick(config_path, manifest_path):
    config = json.load(open(config_path))
    manifest = json.load(open(manifest_path))
    if "layers" not in manifest:
        fail("manifest has no layers (an index? pass the linux/amd64 image manifest)")
    history = [h for h in config.get("history", []) if not h.get("empty_layer")]
    layers = manifest["layers"]
    if len(history) != len(layers):
        fail("history (%d non-empty) and layers (%d) do not line up" % (len(history), len(layers)))
    hits = []
    for entry, layer in zip(history, layers):
        made = entry.get("created_by", "")
        if "/usr/local/bin/zeroned" in made and "COPY" in made.upper() and "mainnet-seed" not in made \
                and "priv_validator" not in made and "node_key" not in made:
            hits.append((layer, made))
    if len(hits) != 1:
        fail("expected exactly one zeroned COPY layer, found %d" % len(hits))
    layer, made = hits[0]
    if layer.get("mediaType") not in GZIP_TYPES:
        fail("binary layer is not a gzip tar: %s" % layer.get("mediaType"))
    if layer.get("size", 0) > 400 * 1024 * 1024:
        fail("binary layer is larger than 400 MiB")
    print(layer["digest"], layer["mediaType"])
    print("created_by: " + made[:200], file=sys.stderr)


def extract(blob_path, out_dir):
    raw = open(blob_path, "rb").read()
    with tarfile.open(fileobj=io.BytesIO(gzip.decompress(raw)), mode="r:") as tar:
        members = tar.getmembers()
        names = set()
        binary = None
        for m in members:
            name = m.name.lstrip("./").rstrip("/")
            if m.isdir() and name in ALLOWED_DIRS:
                continue
            if m.isreg() and name == BINARY:
                binary = m
                names.add(name)
                continue
            fail("layer holds an unexpected entry: %r (type %r)" % (m.name, m.type))
        if binary is None:
            fail("layer has no " + BINARY)
        data = tar.extractfile(binary).read()
    if not data.startswith(b"\x7fELF"):
        fail("zeroned in the layer is not an ELF file")
    sha = hashlib.sha256(data).hexdigest()
    os.makedirs(out_dir, exist_ok=True)
    target = os.path.join(out_dir, sha)
    if os.path.exists(target):
        print(sha)
        return
    tmp = target + ".tmp"
    with open(tmp, "xb") as stream:
        stream.write(data)
    os.chmod(tmp, 0o555)
    os.replace(tmp, target)
    print(sha)


def main():
    if len(sys.argv) == 4 and sys.argv[1] == "pick":
        pick(sys.argv[2], sys.argv[3])
    elif len(sys.argv) == 4 and sys.argv[1] == "extract":
        extract(sys.argv[2], sys.argv[3])
    else:
        print(__doc__, file=sys.stderr)
        sys.exit(64)


if __name__ == "__main__":
    main()

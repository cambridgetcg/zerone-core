#!/usr/bin/env python3
"""Offline tamper and reproducibility checks; no binary execution or keys."""
import copy
import gzip
import importlib.util
import io
import json
from pathlib import Path
import tarfile
import tempfile
import unittest
from unittest import mock

SPEC = importlib.util.spec_from_file_location("participant_package", Path(__file__).with_name("participant-package.py"))
package = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(package)


def archive_bytes(files, extra=()):
    """Permit adversarial members in an otherwise checksum-complete fixture."""
    files = dict(files)
    files["SHA256SUMS"] = "".join(package.sha(raw) + "  " + name + "\n" for name, raw in sorted(files.items())
                                  if name != "SHA256SUMS").encode()
    buffer = io.BytesIO()
    with tarfile.open(fileobj=buffer, mode="w:gz") as archive:
        for name, raw in files.items():
            entry = tarfile.TarInfo(name)
            entry.size = len(raw)
            archive.addfile(entry, io.BytesIO(raw))
        for entry, raw in extra:
            archive.addfile(entry, io.BytesIO(raw) if raw is not None else None)
    return buffer.getvalue()


class ParticipantPackageTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="zerone-participant-package-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.genesis = b'{"chain_id":"zerone-dev-1","app_state":{}}\n'
        self.packet = b'{"schema":"synthetic-upgrade-fixture"}\n'
        self.binaries = {platform: (platform + " synthetic executable; never run").encode() for platform in package.ARCHIVES}
        self.original = package.encoded({"chain_id": "zerone-dev-1", "knowledge_version": 10,
            "source_commit": package.PREDECESSOR, "runtime_binary_sha256": "old-runtime",
            "rpc_genesis_sha256": "a" * 64, "genesis_sha256": package.sha(self.genesis)})
        descriptor = {**json.loads(self.original), "knowledge_version": 11, "source_commit": package.SOURCE,
                      "runtime_binary_sha256": package.sha(self.binaries["linux-amd64"])}
        self.descriptor = package.encoded(descriptor)
        patcher = mock.patch.multiple(package, BINARY_HASHES={p: package.sha(b) for p, b in self.binaries.items()},
            ORIGINAL_DESCRIPTOR_SHA=package.sha(self.original), DESCRIPTOR_SHA=package.sha(self.descriptor),
            GENESIS_SHA=package.sha(self.genesis), UPGRADE_SHA=package.sha(self.packet), HELPER_SHA=package.sha(b"helper"))
        patcher.start()
        self.addCleanup(patcher.stop)
        self.files = {}
        for platform in package.ARCHIVES:
            files = {name: ("synthetic " + name).encode() for name in package.COMMON_MEMBERS}
            files.update({"bin/zeroned": self.binaries[platform], "genesis.json": self.genesis,
                          "network.json": self.original, "UPGRADE.json": self.packet})
            if platform == "darwin-arm64":
                files["bin/darwin-acl-check"] = b"helper"
            source = {"schema": "zerone-development-upgrade-package/v1", "chain_id": "zerone-dev-1",
                      "platform": platform, "source_commit": package.SOURCE, "source_tree": package.TREE,
                      "runtime_image": package.IMAGE, "private_keys_in_package": False,
                      "target_binary_sha256": package.BINARY_HASHES[platform],
                      "original_descriptor_sha256": package.ORIGINAL_DESCRIPTOR_SHA,
                      "original_genesis_sha256": package.GENESIS_SHA, "upgrade_packet_sha256": package.UPGRADE_SHA,
                      "source_files": {path: package.sha(files[name]) for name, path in package.SOURCE_FILES.items()}}
            files["SOURCE.json"] = package.encoded(source)
            files["RUNTIME-IDENTITY.json"] = package.encoded({"source_commit": package.SOURCE,
                "binary_sha256": package.BINARY_HASHES[platform], "cosmos_sdk_version": "v0.53.8"})
            self.files[platform] = files

    def compose(self, files=None, descriptor=None):
        return package.compose(files or self.files["linux-amd64"], "linux-amd64",
                               descriptor or self.descriptor, "b" * 64)

    def test_composition_preserves_payloads_and_labels_new_provenance(self):
        for platform, old in self.files.items():
            new = package.compose(old, platform, self.descriptor, "b" * 64)
            for name in set(old) - {"SOURCE.json", "README.md", "SHA256SUMS", "network.json"}:
                self.assertEqual(new[name], old[name], name)
            self.assertEqual(new["UPSTREAM-SOURCE.json"], old["SOURCE.json"])
            self.assertEqual(new["UPSTREAM-README.md"], old["README.md"])
            self.assertEqual(new["predecessor-network.json"], self.original)
            receipt = json.loads(new["SOURCE.json"])
            self.assertEqual(receipt["packaging_recipe"]["sha256"], "b" * 64)
            self.assertEqual(receipt["source_commit"], package.SOURCE)
            self.assertEqual(receipt["descriptor_sha256"], package.DESCRIPTOR_SHA)
            self.assertNotIn("publication_status", receipt)
            sums = {line.split("  ")[1]: line.split("  ")[0] for line in new["SHA256SUMS"].decode().splitlines()}
            self.assertEqual(set(sums), set(new) - {"SHA256SUMS"})
            self.assertTrue(all(package.sha(new[name]) == digest for name, digest in sums.items()))

    def test_descriptor_pin_source_binary_and_unchanged_fields_are_required(self):
        with self.assertRaisesRegex(ValueError, "descriptor SHA-256"):
            self.compose(descriptor=self.descriptor + b" ")
        for field, bad in (("source_commit", package.PREDECESSOR), ("runtime_binary_sha256", "0" * 64),
                           ("knowledge_version", 10), ("genesis_sha256", "c" * 64), ("peer", "unapproved")):
            value = json.loads(self.descriptor)
            value[field] = bad
            raw = package.encoded(value)
            # Even a separately repinned descriptor cannot silently change these bindings.
            with mock.patch.object(package, "DESCRIPTOR_SHA", package.sha(raw)):
                with self.assertRaisesRegex(ValueError, "Descriptor changes"):
                    self.compose(descriptor=raw)

    def test_wrong_binary_source_script_and_helper_are_rejected(self):
        for name in ("bin/zeroned", "scripts/shared-claims.py", "genesis.json", "UPGRADE.json"):
            files = copy.deepcopy(self.files["linux-amd64"])
            files[name] += b"tamper"
            with self.subTest(name=name), self.assertRaisesRegex(ValueError, "SHA-256 mismatch"):
                self.compose(files=files)
        for field, bad in (("source_commit", package.PREDECESSOR), ("source_tree", "0" * 40),
                           ("private_keys_in_package", True)):
            files = copy.deepcopy(self.files["linux-amd64"])
            source = json.loads(files["SOURCE.json"])
            source[field] = bad
            files["SOURCE.json"] = package.encoded(source)
            with self.subTest(field=field), self.assertRaisesRegex(ValueError, "receipt identity"):
                self.compose(files=files)
        files = dict(self.files["darwin-arm64"])
        files["bin/darwin-acl-check"] = b"different helper"
        with self.assertRaisesRegex(ValueError, "ACL helper"):
            package.compose(files, "darwin-arm64", self.descriptor, "b" * 64)

    def test_archive_pin_and_inner_checksums(self):
        raw = archive_bytes(self.files["linux-amd64"])
        unpacked = package.unpack(raw, package.sha(raw), "linux-amd64")
        self.assertEqual(unpacked["bin/zeroned"], self.binaries["linux-amd64"])
        with self.assertRaisesRegex(ValueError, "Upstream archive SHA-256"):
            package.unpack(raw, "0" * 64, "linux-amd64")
        with tarfile.open(fileobj=io.BytesIO(raw), mode="r:gz") as archive:
            files = {m.name: archive.extractfile(m).read() for m in archive}
        files["bin/zeroned"] += b"tampered after checksums"
        buffer = io.BytesIO()
        with tarfile.open(fileobj=buffer, mode="w:gz") as archive:
            for name, payload in files.items():
                entry = tarfile.TarInfo(name)
                entry.size = len(payload)
                archive.addfile(entry, io.BytesIO(payload))
        bad = buffer.getvalue()
        with self.assertRaisesRegex(ValueError, "bin/zeroned SHA-256"):
            package.unpack(bad, package.sha(bad), "linux-amd64")

    def test_archive_paths_private_keys_symlinks_and_duplicates_are_refused(self):
        for name in ("../escape", "/absolute", "config/priv_validator_key.json", "keyring-test/key.info"):
            files = {**self.files["linux-amd64"], name: b"must not be copied"}
            raw = archive_bytes(files)
            with self.subTest(name=name), self.assertRaisesRegex(ValueError, "archive member"):
                package.unpack(raw, package.sha(raw), "linux-amd64")
        for kind in (tarfile.SYMTYPE, tarfile.LNKTYPE, tarfile.REGTYPE):
            entry = tarfile.TarInfo("bin/zeroned")
            entry.type, entry.linkname = kind, "../../outside"
            raw = archive_bytes(self.files["linux-amd64"], [(entry, b"" if kind == tarfile.REGTYPE else None)])
            with self.subTest(kind=kind), self.assertRaisesRegex(ValueError, "archive member"):
                package.unpack(raw, package.sha(raw), "linux-amd64")

    def test_deterministic_archive_bytes_and_read_only_member_modes(self):
        files = self.compose()
        paths = [self.root / "first.tar.gz", self.root / "second.tar.gz"]
        for path in paths:
            package.write_archive(path, files, 1789259082)
        self.assertEqual(paths[0].read_bytes(), paths[1].read_bytes())
        with tarfile.open(paths[0]) as archive:
            self.assertEqual(archive.getnames(), sorted(files))
            for member in archive:
                self.assertTrue(member.isfile())
                self.assertEqual(member.mode, 0o555 if member.name.startswith("bin/") else 0o444)
                self.assertEqual(archive.extractfile(member).read(), files[member.name])
        with self.assertRaises(FileExistsError):
            package.write_archive(paths[0], files, 1789259082)

    def test_read_refuses_symlink_and_output_refuses_existing_directory(self):
        original = self.root / "original"
        original.write_bytes(b"public input")
        link = self.root / "link"
        link.symlink_to(original)
        with self.assertRaises(OSError):
            package.read(link, 1024)
        with self.assertRaisesRegex(ValueError, "fresh directory"):
            package.build(self.root, self.root, original, self.root)
        self.assertEqual(original.read_bytes(), b"public input")


if __name__ == "__main__":
    unittest.main()

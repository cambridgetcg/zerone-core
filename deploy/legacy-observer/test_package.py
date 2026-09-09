"""Archive traversal/type/overwrite tests and a real signed round trip."""
import contextlib
import gzip
import importlib.util
import io
from pathlib import Path
import tarfile
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch

import test_verify_release as fixtures

HERE = Path(__file__).resolve().parent
SPEC = importlib.util.spec_from_file_location("observer_package", HERE / "package.py")
p = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(p)


class ArchiveBoundaryTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)

    def archive(self, entries):
        path = self.root / "input.tar.gz"
        with tarfile.open(path, "w:gz") as archive:
            for name, kind, data in entries:
                member = tarfile.TarInfo(name)
                member.type = kind
                member.size = len(data)
                member.linkname = "outside"
                archive.addfile(member, io.BytesIO(data))
        return path

    def refuses(self, entries, pattern):
        archive = self.archive(entries)
        with patch.object(p.release, "verify") as verifier:
            with self.assertRaisesRegex((ValueError, OSError), pattern):
                p.unpack(SimpleNamespace(archive=archive, output=self.root / "output", gpgv="/usr/bin/gpgv", purpose="bootstrap"))
            verifier.assert_not_called()

    def test_traversal(self):
        self.refuses([("../outside", tarfile.REGTYPE, b"x")], "filename")
        self.assertFalse((self.root / "outside").exists())

    def test_absolute(self):
        self.refuses([("/outside", tarfile.REGTYPE, b"x")], "filename")

    def test_symlink(self):
        self.refuses([("linked", tarfile.SYMTYPE, b"")], "regular")

    def test_hardlink(self):
        self.refuses([("linked", tarfile.LNKTYPE, b"")], "regular")

    def test_duplicate(self):
        self.refuses([("same", tarfile.REGTYPE, b"x"), ("same", tarfile.REGTYPE, b"y")], "Duplicate")

    def test_directory(self):
        self.refuses([("dir", tarfile.DIRTYPE, b"")], "regular")

    def test_existing_output_preserved(self):
        archive = self.archive([("safe", tarfile.REGTYPE, b"x")])
        output = self.root / "output"
        output.mkdir()
        (output / "sentinel").write_bytes(b"retain")
        with self.assertRaisesRegex(ValueError, "fresh"):
            p.unpack(SimpleNamespace(archive=archive, output=output, gpgv="/usr/bin/gpgv", purpose="bootstrap"))
        self.assertEqual((output / "sentinel").read_bytes(), b"retain")

    def test_extension_rejected_before_oversized_body_read(self):
        for kind in (tarfile.XHDTYPE, tarfile.XGLTYPE, tarfile.GNUTYPE_LONGNAME):
            with self.subTest(kind=kind):
                archive = self.root / (kind.decode() + ".tar.gz")
                header = tarfile.TarInfo("extension")
                header.type = kind
                header.size = p.release.MAX_TOTAL * 4
                archive.write_bytes(gzip.compress(header.tobuf(format=tarfile.USTAR_FORMAT)))
                with patch.object(p.release, "verify") as verifier:
                    with self.assertRaisesRegex(ValueError, "regular"):
                        p.unpack(SimpleNamespace(archive=archive, output=self.root / kind.decode(), gpgv="/usr/bin/gpgv", purpose="bootstrap"))
                    verifier.assert_not_called()

    def test_unverified_outputs_stay_private_and_nonexecutable(self):
        archive = self.archive([("zeroned", tarfile.REGTYPE, b"untrusted executable")])
        output = self.root / "output"
        with patch.object(p.release, "verify", side_effect=ValueError("bad signature")):
            with self.assertRaisesRegex(ValueError, "bad signature"):
                p.unpack(SimpleNamespace(archive=archive, output=output, gpgv="/usr/bin/gpgv", purpose="bootstrap"))
        self.assertEqual(output.stat().st_mode & 0o777, 0o700)
        self.assertEqual((output / "zeroned").stat().st_mode & 0o777, 0o600)

    def test_trailing_content_refused(self):
        archive = self.archive([("safe", tarfile.REGTYPE, b"x")])
        archive.write_bytes(gzip.compress(gzip.decompress(archive.read_bytes()) + b"not tar"))
        with self.assertRaisesRegex(ValueError, "trailing"):
            p.unpack(SimpleNamespace(archive=archive, output=self.root / "output", gpgv="/usr/bin/gpgv", purpose="bootstrap"))


class SignedArchiveRoundTrip(fixtures.ReleaseVerificationTests):
    def test_signed_roundtrip_and_determinism(self):
        original = {path.name: path.read_bytes() for path in self.bundle.iterdir()}
        with patch.object(p.release, "MAIN_FINGERPRINT", self.key.fingerprint), contextlib.redirect_stdout(io.StringIO()):
            first = self.root / "first.tar.gz"
            second = self.root / "second.tar.gz"
            for path in (first, second):
                p.archive(SimpleNamespace(bundle=self.bundle, output=path, gpgv=self.gpgv))
            self.assertEqual(first.read_bytes(), second.read_bytes())
            extracted = self.root / "extracted"
            p.unpack(SimpleNamespace(archive=first, output=extracted, gpgv=self.gpgv, purpose="bootstrap"))
        self.assertEqual(original, {path.name: path.read_bytes() for path in extracted.iterdir()})
        self.assertEqual((extracted / "zeroned").stat().st_mode & 0o777, 0o755)

    def test_changed_after_verification_refused(self):
        real_verify = p.release.verify
        def changed(*args, **kwargs):
            receipt = real_verify(*args, **kwargs)
            (self.bundle / "observer.py").write_bytes(b"malicious replacement")
            return receipt
        with patch.object(p.release, "MAIN_FINGERPRINT", self.key.fingerprint), patch.object(p.release, "verify", side_effect=changed):
            with self.assertRaises((p.release.Refusal, ValueError)):
                p.archive(SimpleNamespace(bundle=self.bundle, output=self.root / "bad.tar.gz", gpgv=self.gpgv))


if __name__ == "__main__":
    unittest.main()

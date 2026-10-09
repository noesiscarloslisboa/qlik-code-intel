"""Release archive and checksum regressions; no build tools needed."""

import hashlib
import importlib.util
from pathlib import Path
import tarfile
import tempfile
import unittest
import zipfile


spec = importlib.util.spec_from_file_location("release", Path(__file__).with_name("release.py"))
release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(release)


class ReleaseTests(unittest.TestCase):
    def test_invalid_version_and_target(self):
        for version, target in [("../main", "darwin_arm64"), ("v1.2.3\n", "windows_amd64"),
                                ("v1.2.3", "linux_amd64")]:
            with self.subTest(version=version, target=target), self.assertRaises(ValueError):
                release.filename(version, target)

    def test_archive_allowlist_and_executable_mode(self):
        with tempfile.TemporaryDirectory() as directory:
            for target in release.TARGETS:
                archive = Path(directory) / release.filename("v1.2.3", target)
                files = {"qlik-repomap": (b"binary", 0o755), "licenses/example": (b"license", 0o644)}
                release.write_archive(archive, files)
                first = archive.read_bytes()
                release.write_archive(archive, dict(reversed(list(files.items()))))
                self.assertEqual(archive.read_bytes(), first)
                if archive.suffix == ".zip":
                    with zipfile.ZipFile(archive) as handle:
                        self.assertEqual(handle.namelist(), sorted(files))
                        self.assertEqual(handle.read("qlik-repomap"), b"binary")
                        self.assertEqual(handle.getinfo("qlik-repomap").external_attr >> 16 & 0o777, 0o755)
                else:
                    with tarfile.open(archive) as handle:
                        self.assertEqual(handle.getnames(), sorted(files))
                        self.assertEqual(handle.extractfile("qlik-repomap").read(), b"binary")
                        self.assertEqual(handle.getmember("qlik-repomap").mode, 0o755)

    def test_checksums_reject_tampering_missing_and_extra_files(self):
        for problem in ["tampered", "missing", "extra", None]:
            with self.subTest(problem=problem), tempfile.TemporaryDirectory() as directory:
                output = Path(directory)
                expected = []
                for target in release.TARGETS:
                    name = release.filename("v1.2.3", target)
                    (output / name).write_bytes(target.encode())
                    digest = hashlib.sha256(target.encode()).hexdigest()
                    line = f"{digest}  {name}\n"
                    (output / f"{name}.sha256").write_text(line)
                    expected.append(line)
                if problem == "tampered":
                    (output / name).write_bytes(b"different")
                elif problem == "missing":
                    (output / name).unlink()
                elif problem == "extra":
                    (output / "private.qvw").write_bytes(b"must not publish")
                if problem:
                    with self.assertRaises(ValueError):
                        release.checksums(output, "v1.2.3")
                    self.assertFalse((output / "SHA256SUMS").exists())
                else:
                    release.checksums(output, "v1.2.3")
                    self.assertEqual((output / "SHA256SUMS").read_text(), "".join(sorted(expected, key=lambda s: s[66:])))
                    self.assertEqual(len(list(output.iterdir())), 4)


if __name__ == "__main__":
    unittest.main()

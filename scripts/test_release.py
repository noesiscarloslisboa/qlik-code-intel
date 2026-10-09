"""Release archive and checksum regressions; no build tools needed."""

import hashlib
import importlib.util
import json
from pathlib import Path
import tarfile
import tempfile
from types import SimpleNamespace
import unittest
from unittest.mock import patch
import zipfile


spec = importlib.util.spec_from_file_location("release", Path(__file__).with_name("release.py"))
release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(release)


class ReleaseTests(unittest.TestCase):
    def test_windows_smoke_removes_compiler_paths(self):
        original = {"SYSTEMROOT": "C:/Windows", "PATH": "C:/compiler/bin", "OTHER": "keep"}
        with patch.object(release, "os", SimpleNamespace(name="nt", environ=original)):
            env = release.smoke_environment()
        self.assertEqual(env["PATH"], str(Path("C:/Windows") / "System32"))
        self.assertEqual(env["OTHER"], "keep")
        self.assertEqual(original["PATH"], "C:/compiler/bin")

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
        for with_skill, problem in [(skill, problem) for skill in [False, True]
                                    for problem in ["tampered", "missing", "extra", None]]:
            with self.subTest(with_skill=with_skill, problem=problem), tempfile.TemporaryDirectory() as directory:
                output = Path(directory)
                expected = []
                names = [release.filename("v1.2.3", target) for target in release.TARGETS]
                if with_skill:
                    names.append(release.skill_filename("v1.2.3"))
                for name in names:
                    (output / name).write_bytes(name.encode())
                    digest = hashlib.sha256(name.encode()).hexdigest()
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
                        release.checksums(output, "v1.2.3", with_skill=with_skill)
                    self.assertFalse((output / "SHA256SUMS").exists())
                else:
                    release.checksums(output, "v1.2.3", with_skill=with_skill)
                    self.assertEqual((output / "SHA256SUMS").read_text(), "".join(sorted(expected, key=lambda s: s[66:])))
                    self.assertEqual(len(list(output.iterdir())), len(names) + 1)

    def test_skill_archive_has_installable_folder_license_and_source_metadata(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            skill = root / "skills/qlik-code-intelligence"
            (skill / "agents").mkdir(parents=True)
            (skill / "SKILL.md").write_text("original instructions")
            (skill / "agents/openai.yaml").write_text("interface: {}")
            (root / "LICENSE").write_text("project license")
            (skill / "private.qvw").write_text("must not ship")
            output = root / "dist"
            with patch.object(release, "run", return_value="a" * 40):
                self.assertTrue(release.package_skill(root, output, "v1.2.3"))
            path = output / release.skill_filename("v1.2.3")
            with zipfile.ZipFile(path) as archive:
                prefix = "qlik-code-intelligence/"
                self.assertEqual(set(archive.namelist()), {prefix + name for name in
                                 ["SKILL.md", "agents/openai.yaml", "LICENSE", "build-info.json"]})
                self.assertEqual(archive.read(prefix + "SKILL.md"), b"original instructions")
                self.assertEqual(archive.read(prefix + "LICENSE"), b"project license")
                metadata = json.loads(archive.read(prefix + "build-info.json"))
                self.assertEqual(metadata["version"], "v1.2.3")
                self.assertEqual(metadata["commit"], "a" * 40)
                self.assertEqual(metadata["cli_min_version"], "v0.1.0")
            self.assertEqual((output / (path.name + ".sha256")).read_text(),
                             hashlib.sha256(path.read_bytes()).hexdigest() + "  " + path.name + "\n")

    def test_old_source_without_skill_is_skipped(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            self.assertFalse(release.package_skill(root, root / "dist", "v0.1.0"))
            self.assertFalse((root / "dist").exists())

    def test_incomplete_skill_fails_before_writing_archive(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "skills/qlik-code-intelligence").mkdir(parents=True)
            with self.assertRaises(ValueError):
                release.package_skill(root, root / "dist", "v1.2.3")
            self.assertFalse((root / "dist").exists())


if __name__ == "__main__":
    unittest.main()

#!/usr/bin/env python3
"""Build, package, and check native releases using only the Python stdlib."""

import argparse
import gzip
import hashlib
import io
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tarfile
import tempfile
import zipfile


TARGETS = {"darwin_arm64": "tar.gz", "darwin_amd64": "tar.gz", "windows_amd64": "zip"}
VERSION = re.compile(r"v[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?")
MODULE = "github.com/noesiscarloslisboa/qlik-code-intel"


def run(args, **kwargs):
    return subprocess.check_output([str(arg) for arg in args], text=True, encoding="utf-8", **kwargs)


def require(condition, message):
    if not condition:
        raise ValueError(message)


def filename(version, target):
    require(VERSION.fullmatch(version), "expected a version such as v0.1.0")
    require(target in TARGETS, "unsupported release target")
    return f"qlik-repomap_{version}_{target}.{TARGETS[target]}"


def write_archive(path, files):
    """Archive an explicit allowlist; never walk the repository or module cache."""
    if path.suffix == ".zip":
        with zipfile.ZipFile(path, "w", compression=zipfile.ZIP_DEFLATED) as archive:
            for name, (data, mode) in sorted(files.items()):
                info = zipfile.ZipInfo(name, date_time=(2020, 1, 1, 0, 0, 0))
                info.create_system = 3
                info.external_attr = (0o100000 | mode) << 16
                info.compress_type = zipfile.ZIP_DEFLATED
                archive.writestr(info, data)
    else:
        with path.open("wb") as raw, gzip.GzipFile(filename="", fileobj=raw, mode="wb", mtime=0) as compressed:
            with tarfile.open(fileobj=compressed, mode="w") as archive:
                for name, (data, mode) in sorted(files.items()):
                    info = tarfile.TarInfo(name)
                    info.size, info.mode = len(data), mode
                    archive.addfile(info, io.BytesIO(data))


def smoke(binary, root, version):
    env = os.environ.copy()
    if os.name == "nt":
        # A downloaded binary must work without the build toolchain on PATH.
        env["PATH"] = str(Path(env["SystemRoot"]) / "System32")
    require(run([binary, "version"], env=env).strip() == version, "wrong embedded version")
    samples = root / "testdata/repository"
    index = json.loads(run([binary, "scan", "--root", samples, "--strict", "--json"], env=env))
    files = index["files"]
    symbols = [symbol for file in files for symbol in file["symbols"]]
    require(len(files) == 5, "baseline file count changed")
    require(sum(symbol["role"] == "definition" for symbol in symbols) == 21, "baseline definitions changed")
    require(sum(symbol["role"] == "reference" for symbol in symbols) == 26, "baseline references changed")
    require(sum(len(file["edges"] or []) for file in files) == 9, "baseline dependencies changed")
    require(not any(file["diagnostics"] for file in files), "baseline diagnostics")
    for command, expected in [
        (["map", "--tokens", "512"], "table Daily Sales"),
        (["find", "Revenue", "--kind", "field"], "20_summary.qvs:4"),
        (["deps", "Daily Sales", "--direction", "upstream"], "table:Orders"),
        (["context", "Revenue", "--tokens", "512"], "4 |     Sum(GrossAmount) AS Revenue"),
    ]:
        output = run([binary, *command, "--root", samples], env=env)
        require(expected in output, f"packaged {command[0]} failed")
        if command[0] in {"map", "context"}:
            require(len(output.encode("utf-8")) <= 512, "packaged output exceeds budget")
    return binary


def package(root, output, version):
    go = json.loads(run(["go", "env", "-json", "GOOS", "GOARCH", "GOROOT", "CGO_ENABLED"], cwd=root))
    target = f"{go['GOOS']}_{go['GOARCH']}"
    name = filename(version, target)
    require(go["CGO_ENABLED"] == "1", "Tree-sitter requires CGO_ENABLED=1")
    output.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="qlik-release-") as directory:
        binary = Path(directory) / ("qlik-repomap.exe" if go["GOOS"] == "windows" else "qlik-repomap")
        run(["go", "build", "-trimpath", "-buildvcs=false", "-ldflags",
             f"-X {MODULE}/internal/cli.Version={version}", "-o", binary, "./cmd/qlik-repomap"], cwd=root)
        if go["GOOS"] == "windows":
            imports = re.findall(r"DLL Name: (\S+)", run(["objdump", "-p", binary]), re.IGNORECASE)
            require(imports, "could not inspect Windows DLL imports")
            system = Path(os.environ["SystemRoot"]) / "System32"
            require(all((system / dll).is_file() or dll.lower().startswith("api-ms-win-")
                        for dll in imports), f"non-system DLL dependency: {imports}")
        files = {binary.name: (binary.read_bytes(), 0o755)}
        if go["GOOS"] == "windows":
            prefix = Path(shutil.which(os.environ["CC"])).parent.parent
            for package_name, names in {
                "gcc": ["COPYING.RUNTIME", "COPYING3"],
                "crt": ["COPYING", "COPYING.MinGW-w64-runtime.txt", "COPYING.MinGW-w64.txt"],
                "winpthreads": ["COPYING"],
            }.items():
                for notice in names:
                    path = prefix / "share/licenses" / package_name / notice
                    files[f"licenses/{package_name}/{notice}"] = (path.read_bytes(), 0o644)
        for notice in ["LICENSE", "THIRD_PARTY_NOTICES.md"]:
            files[notice] = ((root / notice).read_bytes(), 0o644)
        go_license = Path(go["GOROOT"]) / "LICENSE"
        if not go_license.is_file():
            # Homebrew puts the original license beside its libexec directory.
            go_license = Path(go["GOROOT"]).parent / "LICENSE"
        files["licenses/Go-LICENSE"] = (go_license.read_bytes(), 0o644)
        for module, label in [("github.com/tree-sitter/go-tree-sitter", "go-tree-sitter"),
                              ("github.com/mattn/go-pointer", "go-pointer")]:
            directory = Path(run(["go", "list", "-m", "-f", "{{.Dir}}", module], cwd=root).strip())
            files[f"licenses/{label}-LICENSE"] = ((directory / "LICENSE").read_bytes(), 0o644)
            if label == "go-tree-sitter":
                files["licenses/Unicode-LICENSE"] = ((directory / "src/unicode/LICENSE").read_bytes(), 0o644)
        metadata = {"version": version, "target": target,
                    "commit": run(["git", "rev-parse", "HEAD"], cwd=root).strip(),
                    "go": run(["go", "version"], cwd=root).strip(),
                    "c_compiler": run([os.environ.get("CC", "cc"), "--version"]).splitlines()[0]}
        files["build-info.json"] = ((json.dumps(metadata, indent=2) + "\n").encode(), 0o644)
        archive = output / name
        write_archive(archive, files)
    with tempfile.TemporaryDirectory(prefix="qlik-unpacked-") as directory:
        unpacked = Path(directory)
        if archive.suffix == ".zip":
            with zipfile.ZipFile(archive) as handle:
                handle.extractall(unpacked)
        else:
            with tarfile.open(archive) as handle:
                handle.extractall(unpacked, filter="data")
        executable = smoke(unpacked / binary.name, root, version)
        benchmark = Path(__file__).with_name("benchmark-retrieval.py")
        result = json.loads(run([os.sys.executable, benchmark, "--samples-dir", root / "testdata/retrieval",
                                 "--manifest", root / "testdata/retrieval/questions.json", "--binary", executable]))
        print(json.dumps({"retrieval": result["summary"]}))
    digest = hashlib.sha256(archive.read_bytes()).hexdigest()
    (output / f"{name}.sha256").write_text(f"{digest}  {name}\n", encoding="utf-8", newline="\n")
    print(f"Verified {name}: {digest}")


def checksums(output, version):
    expected = sorted(filename(version, target) for target in TARGETS)
    require(sorted(path.name for path in output.iterdir()) ==
            sorted(expected + [name + ".sha256" for name in expected]), "unexpected or missing release files")
    lines = []
    for name in expected:
        digest = hashlib.sha256((output / name).read_bytes()).hexdigest()
        line = f"{digest}  {name}\n"
        require((output / f"{name}.sha256").read_text() == line, f"checksum mismatch: {name}")
        lines.append(line)
    (output / "SHA256SUMS").write_text("".join(lines), encoding="utf-8", newline="\n")
    for name in expected:
        (output / f"{name}.sha256").unlink()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=["package", "checksums"])
    parser.add_argument("--root", type=Path, default=Path.cwd())
    parser.add_argument("--output", type=Path, default=Path("dist"))
    parser.add_argument("--version", required=True)
    args = parser.parse_args()
    require(VERSION.fullmatch(args.version), "invalid release version")
    if args.command == "package":
        package(args.root.resolve(), args.output.resolve(), args.version)
    else:
        checksums(args.output, args.version)


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""Audit a separately obtained, pinned reference corpus without executing Qlik."""

import argparse
from collections import Counter
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys


PROJECT = Path(__file__).resolve().parent.parent


def require(condition, message):
    if not condition:
        raise RuntimeError(message)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--samples-dir", required=True, type=Path,
                        help="external checkout root containing validation/*.qvs")
    parser.add_argument("--binary", type=Path, default=PROJECT / "bin/qlik-repomap")
    parser.add_argument("--output", type=Path, help="aggregate JSON report; excludes source text")
    args = parser.parse_args()
    manifest = json.loads((PROJECT / "docs/validation/reference-samples.json").read_text())
    root, binary = args.samples_dir.resolve(), args.binary.resolve()
    sources = {}
    for entry in manifest["files"]:
        path = root / entry["path"]
        require(path.is_file() and not path.is_symlink(), f"missing regular sample: {path}")
        raw = path.read_bytes()
        digest = hashlib.sha1(b"blob " + str(len(raw)).encode() + b"\0" + raw).hexdigest()
        require(digest == entry["blob_sha"], f"sample differs from pinned revision: {path}")
        require(len(raw) == entry["size_bytes"], f"sample size differs: {path}")
        raw.decode("utf-8")
        sources[entry["path"]] = raw

    def run(*arguments):
        result = subprocess.run([str(binary), *arguments, "--root", str(root)],
                                capture_output=True, timeout=30)
        require(result.returncode == 0,
                f"command failed: {arguments}: {result.stderr.decode('utf-8', errors='replace')}")
        result.stdout.decode("utf-8")
        return result.stdout

    index = json.loads(run("scan", "--json"))
    require([f["path"] for f in index["files"]] == sorted(sources),
            "scan did not preserve the expected ten relative paths")
    counts = []
    for file in index["files"]:
        raw = sources[file["path"]]
        lines = raw.split(b"\n")
        for item in file["symbols"] + file["edges"] + file["loads"] + file["diagnostics"]:
            require(item["path"] == file["path"], "occurrence path differs from file")
            require(1 <= item["line"] <= item["end_line"] <= len(lines),
                    f"invalid source lines: {file['path']}: {item}")
            require(1 <= item["column"] <= len(lines[item["line"] - 1]) + 1,
                    f"invalid source byte column: {file['path']}: {item}")
        for item in file["symbols"] + file["loads"]:
            require(0 <= item["statement_start_byte"] < item["statement_end_byte"] <= len(raw),
                    f"invalid statement byte range: {file['path']}: {item}")
        roles = Counter(s["role"] for s in file["symbols"])
        counts.append({"path": file["path"], "loads": len(file["loads"]),
                       "definitions": roles["definition"], "references": roles["reference"],
                       "dependencies": len(file["edges"]),
                       "diagnostics": len(file["diagnostics"]),
                       "diagnostic_codes": dict(sorted(Counter(d["code"] for d in file["diagnostics"]).items()))})

    # Small source-location oracles, rather than assuming every editor example
    # is a valid Qlik application or every recovered fact is semantically complete.
    matches = json.loads(run("find", "Airports", "--kind", "table", "--role", "definition", "--json", "--limit", "0"))
    locations = [m["line"] for m in matches if m["path"] == "validation/linter_as_withtab.qvs" and m["name"] == "Airports"]
    require(locations == [1, 11, 21], f"repeated table definitions/lines lost: {locations}")
    field_matches = json.loads(run("find", "column1_test", "--kind", "field", "--role", "definition", "--json", "--limit", "0"))
    field_lines = [m["line"] for m in field_matches if m["path"] == "validation/linter_as_withtab.qvs" and m["name"] == "column1_test"]
    require(field_lines == [8, 18, 28], f"tabbed aliases/lines lost: {field_lines}")
    dependencies = json.loads(run("deps", "MyExcelData", "--kind", "table", "--direction", "upstream", "--json"))
    require(any(e["kind"] == "preceding" and e["to"]["name"] == "@validation/keywords.qvs:16:1" for e in dependencies),
            "preceding LOAD input stage missing")
    recovered = next(f for f in index["files"] if f["path"] == "validation/test.qvs")
    require(any(s["kind"] == "include" and s["line"] == 18 for s in recovered["symbols"]),
            "include after balanced literal fragments was lost")
    require(any(s["kind"] == "table" and s["role"] == "definition" and s["line"] == 35 for s in recovered["symbols"]),
            "supported LOAD after balanced fragments was lost")
    require(any(s["kind"] == "table" and s["role"] == "definition" and s["line"] == 24 for s in recovered["symbols"]),
            "labeled LOAD after unfinished unsupported statement was lost")
    require(any(load["line"] == 24 for load in recovered["loads"]),
            "recovered LOAD range missing or misplaced")

    require(not any(s["name"].lower() == "not" for s in recovered["symbols"]),
            "unary NOT indexed as a symbol")
    require(not any(d["line"] == 62 for d in recovered["diagnostics"]),
            "documented NOT IsNull clause still produces a diagnostic")

    checks = []
    for command, query in (("map", ""), ("map", "Floor_test"), ("context", "Floor_test")):
        for budget in (256, 512, 1024, 2048):
            arguments = [command, "--tokens", str(budget)]
            if query:
                arguments += ["--query", query] if command == "map" else [query]
            output = run(*arguments)
            require(output == run(*arguments), f"nondeterministic output: {arguments}")
            require(len(output) <= budget, f"budget exceeded: {arguments}")
            require(not output or output.endswith(b"\n"), "truncated record")
            text = output.decode("utf-8")
            if query and budget == 2048:
                require(query in text, f"matching field missing: {arguments}")
            if command == "context":
                current_path = None
                for line in text.splitlines():
                    header = re.fullmatch(r"(.+\.qvs):([0-9]+)", line)
                    if header:
                        current_path = header[1]
                        require(current_path in sources, "context path not indexed")
                        continue
                    numbered = re.fullmatch(r"([0-9]+) \| (.*)", line)
                    require(numbered and current_path, "context line lacks a source location")
                    line_number = int(numbered[1])
                    source_lines = sources[current_path].decode("utf-8").split("\n")
                    require(1 <= line_number <= len(source_lines), "context line out of bounds")
                    require(numbered[2] == source_lines[line_number - 1].removesuffix("\r"),
                            "context changed original source")
            checks.append({"command": command, "query": query, "budget": budget,
                           "output_bytes": len(output),
                           "sha256": hashlib.sha256(output).hexdigest()})

    report = {"repository": manifest["repository"], "revision": manifest["revision"],
              "verified_sample_blobs": len(sources), "files": counts,
              "totals": {key: sum(f[key] for f in counts) for key in
                         ("loads", "definitions", "references", "dependencies", "diagnostics")},
              "retrieval_checks": checks,
              "note": "Diagnostics are retained; these editor samples include invalid/unsupported syntax. Counts do not imply complete semantic accuracy."}
    encoded = json.dumps(report, indent=2) + "\n"
    if args.output:
        args.output.write_text(encoded)
    else:
        sys.stdout.write(encoded)


if __name__ == "__main__":
    try:
        main()
    except (RuntimeError, OSError, ValueError, subprocess.TimeoutExpired) as error:
        sys.exit(f"reference validation failed: {error}")

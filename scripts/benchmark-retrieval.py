#!/usr/bin/env python3
"""Check source-reviewed retrieval answers against the built CLI; never execute Qlik."""

import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess
import sys


PROJECT = Path(__file__).resolve().parent.parent
BUDGETS = (256, 512, 1024)


def require(condition, message):
    if not condition:
        raise ValueError(message)


def includes(actual, expected):
    """Match a manually specified subset, including typed dependency endpoints."""
    return all(key in actual and (includes(actual[key], value) if isinstance(value, dict)
               else actual[key] == value) for key, value in expected.items())


def context_blocks(text, sources):
    blocks, seen = [], set()
    for line in text.removesuffix("\n").split("\n") if text else []:
        header = re.fullmatch(r"(.+\.qvs):([0-9]+)", line, re.IGNORECASE)
        if header:
            require(header[1] in sources, "unknown-context-path")
            blocks.append({"path": header[1], "start": int(header[2]), "lines": []})
            continue
        numbered = re.fullmatch(r"([0-9]+) \| (.*)", line)
        require(numbered and blocks, "invalid-context-format")
        block, number = blocks[-1], int(numbered[1])
        require(number == block["start"] + len(block["lines"]), "noncontiguous-context")
        original = sources[block["path"]]
        require(1 <= number <= len(original), "context-line-out-of-bounds")
        require(numbered[2] == original[number - 1].removesuffix("\r"), "changed-source")
        key = (block["path"], number)
        require(key not in seen, "duplicate-context-line")
        seen.add(key)
        block["lines"].append(number)
    require(all(block["lines"] for block in blocks), "empty-context-block")
    return blocks


def answer_matches(case, text, sources):
    expected, command = case["expected"], case["command"]
    if command == "find":
        rows = json.loads(text)
        return bool(rows) and any(includes(rows[0], item) for item in expected)
    if command == "deps":
        rows = json.loads(text)
        return all(any(includes(row, item) for row in rows) for item in expected) and not any(
            includes(row, item) for row in rows for item in case.get("forbidden", []))
    if command == "context":
        blocks = context_blocks(text, sources)
        return bool(blocks) and any(item["path"] == blocks[0]["path"] and
                                   item["line"] in blocks[0]["lines"] for item in expected)
    records = []
    for line in text.removesuffix("\n").split("\n") if text else []:
        record = re.fullmatch(r"(.+\.qvs):([0-9]+) (.+)", line, re.IGNORECASE)
        require(record and record[1] in sources, "invalid-map-record")
        require(1 <= int(record[2]) <= len(sources[record[1]]), "map-line-out-of-bounds")
        records.append(record)
    return bool(records) and any(records[0][1] == item["path"] and
                                int(records[0][2]) == item["line"] and
                                all(part in records[0][3] for part in item.get("contains", []))
                                for item in expected)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--samples-dir", required=True, type=Path)
    parser.add_argument("--manifest", required=True, type=Path)
    parser.add_argument("--binary", type=Path, default=PROJECT / "bin/qlik-repomap")
    parser.add_argument("--output", type=Path, help="aggregate JSON; no query or source text")
    args = parser.parse_args()
    require(not args.samples_dir.is_symlink(), "sample root must not be a symlink")
    root, binary = args.samples_dir.resolve(), args.binary.resolve()
    manifest = json.loads(args.manifest.read_text())
    require(manifest["files"] and manifest["cases"], "empty-benchmark")
    sources = {}
    for entry in manifest["files"]:
        require(entry["path"] not in sources, "duplicate-sample-path")
        relative = Path(entry["path"])
        require(not relative.is_absolute() and ".." not in relative.parts, "invalid-sample-path")
        path = root / relative
        require(path.is_file() and not path.is_symlink() and path.resolve().is_relative_to(root),
                "missing regular sample inside root")
        raw = path.read_bytes()
        require(hashlib.sha256(raw).hexdigest() == entry["sha256"], "sample-hash-mismatch")
        sources[entry["path"]] = raw.decode("utf-8").split("\n")
    ids = set()
    for case in manifest["cases"]:
        require(re.fullmatch(r"[A-Za-z0-9_-]+", case["id"]) and case["id"] not in ids, "invalid-case-id")
        ids.add(case["id"])
        require(case["command"] in ("find", "map", "context", "deps") and case["expected"], "invalid-case")
        for item in case["expected"]:
            require(item["path"] in sources and 1 <= item["line"] <= len(sources[item["path"]]),
                    "expected-source-location-out-of-bounds")
            if case["command"] == "context":
                minimal = f"{item['path']}:{item['line']}\n{item['line']} | {sources[item['path']][item['line'] - 1].removesuffix(chr(13))}\n"
                require(len(minimal.encode()) <= min(BUDGETS), "expected-source-line-cannot-fit-smallest-budget")

    checks = []
    for case in manifest["cases"]:
        command = case["command"]
        for budget in BUDGETS if command in ("map", "context") else (None,):
            arguments = [command]
            if command == "map":
                arguments += ["--query", case["query"]]
            else:
                arguments.append(case["query"])
            if budget is not None:
                arguments += ["--tokens", str(budget)]
            if command in ("find", "deps"):
                arguments.append("--json")
                for flag in ("kind", "role", "direction"):
                    if flag in case:
                        arguments += ["--" + flag, case[flag]]
            if command == "find":
                arguments += ["--limit", "0"]
            arguments += ["--root", str(root)]
            runs = [subprocess.run([str(binary), *arguments], capture_output=True, timeout=30) for _ in range(2)]
            output, failures = runs[0].stdout, []
            if any(run.returncode != 0 for run in runs):
                failures.append("command-failed")
            if output != runs[1].stdout or runs[0].stderr != runs[1].stderr:
                failures.append("nondeterministic-output")
            if budget is not None and len(output) > budget:
                failures.append("budget-exceeded")
            if output and not output.endswith(b"\n"):
                failures.append("incomplete-record")
            correct = False
            try:
                correct = answer_matches(case, output.decode("utf-8"), sources)
            except (ValueError, KeyError, TypeError, IndexError) as error:
                failures.append(str(error) if isinstance(error, ValueError) and not isinstance(error, json.JSONDecodeError)
                                else "invalid-output")
            if not correct:
                failures.append("answer-not-first" if command != "deps" else "dependency-answer-mismatch")
            checks.append({"id": case["id"], "command": command, "budget": budget,
                           "output_bytes": len(output), "answer_passed": correct,
                           "invariants_passed": not any(f != "answer-not-first" and f != "dependency-answer-mismatch" for f in failures),
                           "passed": not failures, "failures": failures})
    summary = {}
    for command in ("find", "map", "context", "deps"):
        rows = [row for row in checks if row["command"] == command]
        summary[command] = {"checks": len(rows), "passed": sum(row["passed"] for row in rows)}
    report = {"questions": len(manifest["cases"]), "verified_source_files": len(sources),
              "budget_units": "UTF-8 bytes", "budgets": list(BUDGETS),
              "summary": summary, "checks": checks,
              "note": "Selected source-reviewed answers, not exhaustive precision/recall or runtime validation. No query/source text is included."}
    encoded = json.dumps(report, indent=2) + "\n"
    if args.output:
        args.output.write_text(encoded)
    else:
        sys.stdout.write(encoded)
    return 0 if all(row["passed"] for row in checks) else 1


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (ValueError, OSError, KeyError, TypeError, subprocess.TimeoutExpired) as error:
        sys.exit(f"retrieval benchmark failed: {error}")

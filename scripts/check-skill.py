#!/usr/bin/env python3
"""Execute the skill's documented CLI examples against original QVS fixtures."""

import argparse
import importlib.util
import json
from pathlib import Path
import re
import shlex
import shutil
import subprocess
import tempfile


spec = importlib.util.spec_from_file_location("benchmark", Path(__file__).with_name("benchmark-retrieval.py"))
benchmark = importlib.util.module_from_spec(spec)
spec.loader.exec_module(benchmark)
require = benchmark.require


def check(binary, project, skill):
    text = (skill / "SKILL.md").read_text(encoding="utf-8")
    blocks = re.findall(r"^```sh\n(.*?)^```", text, re.MULTILINE | re.DOTALL)
    commands = [shlex.split(line) for block in blocks for line in block.splitlines() if line.strip()]
    require(commands and all(args[0] == "qlik-repomap" for args in commands), "unexpected documented command")
    seen = set()
    with tempfile.TemporaryDirectory(prefix="qlik skill examples ") as directory:
        workspace = Path(directory)
        root = workspace / "scripts"
        shutil.copytree(project / "testdata/repository", root)
        sources = {path.relative_to(root).as_posix(): path.read_text(encoding="utf-8").split("\n")
                   for path in root.rglob("*.qvs")}
        for args in commands:
            command = args[1]
            seen.add(command)
            result = subprocess.run([str(binary), *args[1:]], cwd=workspace, capture_output=True, timeout=30)
            require(result.returncode == 0 and not result.stderr, f"documented {command} failed")
            output = result.stdout.decode("utf-8")
            if "--tokens" in args:
                require(len(result.stdout) <= int(args[args.index("--tokens") + 1]), "documented budget exceeded")
            if command == "version":
                require(output.strip(), "empty CLI version")
            elif command == "map":
                first = output.splitlines()[0] if output else ""
                require(first.startswith("20_summary.qvs:1 table Daily Sales") and "Revenue" in first,
                        "focused map missed the Revenue statement")
            elif command == "find":
                rows = json.loads(output)
                require(rows and benchmark.includes(rows[0], {"name": "Revenue", "kind": "field",
                        "role": "definition", "path": "20_summary.qvs", "line": 4}), "wrong field definition")
            elif command == "deps":
                query = args[2]
                expectations = {
                    "Daily Sales": [{"direction": "upstream", "kind": "resident",
                                     "from": {"kind": "table", "name": "Daily Sales"},
                                     "to": {"kind": "table", "name": "Orders"}}],
                    "daily_sales.qvd": [{"direction": "upstream", "kind": "store",
                                         "from": {"kind": "source", "name": "lib://Exports/daily_sales.qvd"},
                                         "to": {"kind": "table", "name": "Daily Sales"}}],
                    "orders.qvd": [{"direction": "downstream", "kind": "from",
                                    "from": {"kind": "table", "name": "Orders"},
                                    "to": {"kind": "source", "name": "$(vData)/orders.qvd"}}],
                }
                rows = json.loads(output)
                require(query in expectations and all(any(benchmark.includes(row, expected) for row in rows)
                        for expected in expectations[query]), "wrong dependency kinds/direction or evaluated path")
            elif command == "context":
                blocks = benchmark.context_blocks(output, sources)
                require(blocks and blocks[0]["path"] == "20_summary.qvs" and 4 in blocks[0]["lines"],
                        "context lost the original Revenue definition")
            elif command == "scan":
                files = json.loads(output)["files"]
                require(len(files) == 5 and not any(file["diagnostics"] for file in files), "wrong baseline coverage")
                symbols = [symbol for file in files for symbol in file["symbols"]]
                for name, role in [("Amount", "reference"), ("GrossAmount", "definition")]:
                    require(any(benchmark.includes(symbol, {"name": name, "role": role, "kind": "field",
                                "owner": "Orders", "path": "10_orders.qvs", "line": 5}) for symbol in symbols),
                            "input field/output alias roles changed")
        require(seen == {"version", "map", "find", "deps", "context", "scan"}, "missing documented command coverage")
        (root / "99_unsupported.qvs").write_text(
            "SQL SELECT ID FROM remote;\nSupported: LOAD ID FROM [local.qvd] (qvd);\n", encoding="utf-8")
        result = subprocess.run([str(binary), "scan", "--root", str(root), "--strict", "--json"],
                                capture_output=True, timeout=30)
        require(result.returncode == 1 and b"unsupported" in result.stderr, "strict scan hid diagnostics")
        file = next(file for file in json.loads(result.stdout)["files"] if file["path"] == "99_unsupported.qvs")
        require(any(symbol["name"] == "Supported" and symbol["role"] == "definition"
                    for symbol in file["symbols"]), "supported source lost after diagnostic")
    print(json.dumps({"documented_commands": len(commands), "strict_partial_index": "passed",
                      "source_lines_roles_directions_budgets": "passed"}))


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", type=Path, required=True)
    parser.add_argument("--project", type=Path, default=Path(__file__).resolve().parent.parent)
    parser.add_argument("--skill", type=Path)
    args = parser.parse_args()
    project = args.project.resolve()
    check(args.binary.resolve(), project, args.skill or project / "skills/qlik-code-intelligence")

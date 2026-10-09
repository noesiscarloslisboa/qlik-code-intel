"""Checks that the optional benchmark rejects wrong first answers and changed source."""

import importlib.util
from pathlib import Path
import unittest


spec = importlib.util.spec_from_file_location("benchmark", Path(__file__).with_name("benchmark-retrieval.py"))
benchmark = importlib.util.module_from_spec(spec)
spec.loader.exec_module(benchmark)


class RetrievalOracleTests(unittest.TestCase):
    def test_find_requires_first_answer(self):
        case = {"command": "find", "expected": [{"path": "a.qvs", "line": 2, "name": "Wanted"}]}
        wrong_first = '[{"path":"a.qvs","line":1,"name":"Other"},{"path":"a.qvs","line":2,"name":"Wanted"}]'
        self.assertFalse(benchmark.answer_matches(case, wrong_first, {}))

    def test_map_requires_first_record_and_answer_text(self):
        case = {"command": "map", "expected": [{"path": "a.qvs", "line": 2, "contains": ["Wanted"]}]}
        sources = {"a.qvs": ["Other: LOAD ID AUTOGENERATE 1;", "Wanted: LOAD ID AUTOGENERATE 1;"]}
        self.assertFalse(benchmark.answer_matches(case, "a.qvs:1 table Other\na.qvs:2 table Wanted\n", sources))
        self.assertFalse(benchmark.answer_matches(case, "a.qvs:2 load\n", sources))
        self.assertTrue(benchmark.answer_matches(case, "a.qvs:2 table Wanted\n", sources))

    def test_context_requires_first_block_and_original_source(self):
        case = {"command": "context", "expected": [{"path": "a.qvs", "line": 2}]}
        sources = {"a.qvs": ["Other;\r", "Wanted;\r"]}
        self.assertFalse(benchmark.answer_matches(case, "a.qvs:1\n1 | Other;\na.qvs:2\n2 | Wanted;\n", sources))
        self.assertTrue(benchmark.answer_matches(case, "a.qvs:1\n1 | Other;\n2 | Wanted;\n", sources))
        with self.assertRaisesRegex(ValueError, "changed-source"):
            benchmark.answer_matches(case, "a.qvs:2\n2 | Changed;\n", sources)
        with self.assertRaisesRegex(ValueError, "duplicate-context-line"):
            benchmark.context_blocks("a.qvs:2\n2 | Wanted;\na.qvs:2\n2 | Wanted;\n", sources)

    def test_unicode_separators_remain_inside_source_lines(self):
        sources = {"a.qvs": ["LET vText = 'a\u2028b';"]}
        blocks = benchmark.context_blocks("a.qvs:1\n1 | LET vText = 'a\u2028b';\n", sources)
        self.assertEqual(blocks[0]["lines"], [1])

    def test_dependency_checks_types_and_forbidden_edges(self):
        expected = {"path": "a.qvs", "line": 2, "kind": "store", "from": {"kind": "source", "name": "out.qvd"}}
        case = {"command": "deps", "expected": [expected], "forbidden": [{"line": 3}]}
        self.assertFalse(benchmark.answer_matches(case, '[{"path":"a.qvs","line":2,"kind":"store","from":{"kind":"table","name":"out.qvd"}}]', {}))
        self.assertFalse(benchmark.answer_matches(case, '[{"path":"a.qvs","line":2,"kind":"store","from":{"kind":"source","name":"out.qvd"}},{"line":3}]', {}))
        self.assertTrue(benchmark.answer_matches(case, '[{"path":"a.qvs","line":2,"kind":"store","from":{"kind":"source","name":"out.qvd"}}]', {}))


if __name__ == "__main__":
    unittest.main()

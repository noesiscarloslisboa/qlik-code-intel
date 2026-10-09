package repomap

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/noesiscarloslisboa/qlik-code-intel/internal/qlik"
)

func fixtureIndex(t *testing.T) *qlik.Index {
	t.Helper()
	idx, err := qlik.Scan(context.Background(), "../../testdata/repository")
	if err != nil {
		t.Fatal(err)
	}
	return idx
}

func TestMapUsefulAndDeterministic(t *testing.T) {
	t.Parallel()
	idx := fixtureIndex(t)
	got := Map(idx, "", 4096)
	for _, want := range []string{
		"10_orders.qvs:1 table Orders", "fields: OrderID, CustomerID, GrossAmount",
		"<- from $(vData)/orders.qvd", "uses: $(vCutoff), $(vData)",
		"20_summary.qvs:1 table Daily Sales", "<- resident Orders",
		"store Daily Sales -> lib://Exports/daily_sales.qvd", "SET vData = lib://Warehouse",
		"include includes/common.qvs", "extend Orders",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("map missing %q:\n%s", want, got)
		}
	}
	for range 20 {
		if next := Map(idx, "", 4096); next != got {
			t.Fatalf("nondeterministic:\n%s\n%s", got, next)
		}
	}
	focused := Map(idx, "Revenue", 4096)
	if !strings.HasPrefix(focused, "20_summary.qvs:1") {
		t.Errorf("query did not rank matching definition first:\n%s", focused)
	}
}

func TestFindRankingAndFilters(t *testing.T) {
	t.Parallel()
	idx := fixtureIndex(t)
	for _, tt := range []struct {
		name     string
		opts     Search
		wantName string
		wantKind qlik.Kind
	}{
		{"table", Search{Query: "Orders", Kind: "table"}, "Orders", qlik.Table},
		{"variable", Search{Query: "vData", Kind: "variable"}, "vData", qlik.Variable},
		{"field alias", Search{Query: "Revenue", Kind: "field"}, "Revenue", qlik.Field},
		{"qvd basename", Search{Query: "daily_sales.qvd", Kind: "qvd"}, "lib://Exports/daily_sales.qvd", qlik.Source},
		{"case insensitive", Search{Query: "REVENUE"}, "Revenue", qlik.Field},
	} {
		t.Run(tt.name, func(t *testing.T) {
			matches := Find(idx, tt.opts)
			if len(matches) == 0 || matches[0].Name != tt.wantName || matches[0].Kind != tt.wantKind {
				t.Fatalf("matches: %+v", matches)
			}
			if matches[0].Role != "definition" && tt.wantKind != qlik.Source {
				t.Errorf("definition not ranked first: %+v", matches[0])
			}
		})
	}
	if got := Find(idx, Search{Query: "does-not-exist"}); len(got) != 0 {
		t.Errorf("unexpected results %+v", got)
	}
	if got := Find(idx, Search{Query: "Orders", Limit: 1}); len(got) != 1 {
		t.Errorf("limit failed: %+v", got)
	}
	for _, m := range Find(idx, Search{Kind: "qvd"}) {
		if m.Kind != qlik.Source || !strings.HasSuffix(m.Name, ".qvd") {
			t.Errorf("incorrect qvd filter: %+v", m)
		}
	}
}

func TestDirectDependencies(t *testing.T) {
	t.Parallel()
	idx := fixtureIndex(t)
	up := Dependencies(idx, "Daily Sales", "table", "upstream")
	if len(up) != 1 || up[0].To.Name != "Orders" || up[0].Kind != "resident" {
		t.Fatalf("upstream: %+v", up)
	}
	down := Dependencies(idx, "Orders", "table", "downstream")
	if len(down) != 1 || down[0].From.Name != "Daily Sales" {
		t.Fatalf("downstream: %+v", down)
	}
	output := Dependencies(idx, "daily_sales.qvd", "qvd", "upstream")
	if len(output) != 1 || output[0].To.Name != "Daily Sales" || output[0].Kind != "store" {
		t.Fatalf("store: %+v", output)
	}
	variable := Dependencies(idx, "vData", "variable", "downstream")
	if len(variable) != 1 || variable[0].From.Name != "Orders" {
		t.Fatalf("variable: %+v", variable)
	}
	if got := Dependencies(idx, "missing", "", "both"); len(got) != 0 {
		t.Fatalf("unexpected deps %+v", got)
	}
}

func TestBudgetsNeverOverflowOrSplitUTF8(t *testing.T) {
	t.Parallel()
	idx := fixtureIndex(t)
	unicodeFile, err := qlik.Parse(context.Background(), "vendas.qvs", []byte("[Vendas Diárias]: LOAD [Preço] AS Valor AUTOGENERATE 1;"))
	if err != nil {
		t.Fatal(err)
	}
	idx.Files = append(idx.Files, unicodeFile)
	for _, tokens := range []int{-1, 0, 1, 10, 40, 80, 120, 256, 512, 1024, 4096} {
		t.Run(fmt.Sprint(tokens), func(t *testing.T) {
			for name, text := range map[string]string{"map": Map(idx, "", tokens), "context": Context(idx, "Revenue", tokens)} {
				if tokens >= 0 && EstimateTokens(text) > tokens {
					t.Errorf("%s exceeded budget: %d > %d", name, EstimateTokens(text), tokens)
				}
				if !utf8.ValidString(text) {
					t.Errorf("%s split UTF-8", name)
				}
				if text != "" && !strings.HasSuffix(text, "\n") {
					t.Errorf("%s truncated a record", name)
				}
			}
		})
	}
}

func TestContextContainsOriginalNumberedSourceAndNoDuplicates(t *testing.T) {
	t.Parallel()
	idx := fixtureIndex(t)
	got := Context(idx, "Revenue", 2048)
	for _, want := range []string{"20_summary.qvs:1", "4 |     Sum(GrossAmount) AS Revenue", "5 | RESIDENT Orders", "10_orders.qvs:1"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Count(got, "4 |     Sum(GrossAmount) AS Revenue") != 1 {
		t.Errorf("duplicated source:\n%s", got)
	}
	if Context(idx, "absent", 2048) != "" {
		t.Error("no-match context should be empty")
	}
	if next := Context(idx, "Revenue", 2048); next != got {
		t.Error("context nondeterministic")
	}
}

func TestContextLargeStatementPrioritizesExactMatchingLine(t *testing.T) {
	t.Parallel()
	var src strings.Builder
	src.WriteString("Big:\nLOAD\n")
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&src, "    Field%d,\n", i)
	}
	src.WriteString("    CriticalField\nFROM [huge.qvd] (qvd);\n")
	f, err := qlik.Parse(context.Background(), "big.qvs", []byte(src.String()))
	if err != nil {
		t.Fatal(err)
	}
	idx := &qlik.Index{Files: []qlik.File{f}}
	got := Context(idx, "CriticalField", 150)
	if !strings.Contains(got, "CriticalField") {
		t.Fatalf("matching line lost:\n%s", got)
	}
	if !strings.Contains(Map(idx, "", 150), "table Big") {
		t.Error("large field list hid table")
	}
}

func TestDuplicateDefinitionsRemainDistinct(t *testing.T) {
	t.Parallel()
	var files []qlik.File
	for _, name := range []string{"a.qvs", "b.qvs"} {
		f, err := qlik.Parse(context.Background(), name, []byte("T: LOAD ID AUTOGENERATE 1;"))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	idx := &qlik.Index{Files: files}
	got := Find(idx, Search{Query: "T", Kind: "table", Role: "definition"})
	paths := []string{}
	for _, m := range got {
		paths = append(paths, m.Path)
	}
	if !reflect.DeepEqual(paths, []string{"a.qvs", "b.qvs"}) {
		t.Fatalf("duplicates lost: %v", paths)
	}
}

func TestSameLineStatementsRemainSeparate(t *testing.T) {
	t.Parallel()
	f, err := qlik.Parse(context.Background(), "same.qvs", []byte("SET vA = 'a  b'; SET vB = 2; T: LOAD ID FROM [first.qvd] (qvd); U: LOAD ID FROM [second.qvd] (qvd);"))
	if err != nil {
		t.Fatal(err)
	}
	idx := &qlik.Index{Files: []qlik.File{f}}
	got := Map(idx, "", 2048)
	for _, want := range []string{"SET vA = 'a  b'", "SET vB = 2", "table T | fields: ID | <- from first.qvd", "table U | fields: ID | <- from second.qvd"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "first.qvd, from second.qvd") {
		t.Errorf("sources attached to wrong statements:\n%s", got)
	}
}

func TestContextLiteralIncludeAndNestedPathStaysLiteral(t *testing.T) {
	t.Parallel()
	idx := fixtureIndex(t)
	got := Context(idx, "includes/common.qvs", 2048)
	if !strings.Contains(got, "SET vCurrency = 'EUR'") {
		t.Fatalf("included file context missing:\n%s", got)
	}
	f, err := qlik.Parse(context.Background(), "sub/loader.qvs", []byte("$(Include=common.qvs);"))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Edges) != 1 || f.Edges[0].To.Name != "common.qvs" {
		t.Fatalf("invented include working directory: %+v", f.Edges)
	}
}

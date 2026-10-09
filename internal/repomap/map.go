package repomap

import (
	"fmt"
	"sort"
	"strings"

	"github.com/noesiscarloslisboa/qlik-code-intel/internal/qlik"
)

type statementKey struct {
	path        string
	start, stop int
	offset      uint
}
type candidate struct {
	key        statementKey
	load       *qlik.Load
	kind       string
	symbols    []qlik.Symbol
	edges      []qlik.Edge
	score      int
	queryScore int
	focus      *qlik.Symbol
	text       string
}

// Map generates deterministic statement summaries. Whole records are selected;
// each candidate has a compact fallback so one large LOAD cannot hide a table.
func Map(idx *qlik.Index, query string, tokens int) string {
	if tokens <= 0 {
		return ""
	}
	groups := candidates(idx)
	query = strings.TrimSpace(query)
	if query != "" {
		for _, m := range Find(idx, Search{Query: query}) {
			key := statementKey{m.Path, m.StartLine, m.StopLine, m.StartByte}
			if g := groups[key]; g != nil {
				g.queryScore = max(g.queryScore, m.Score)
				// Find orders the strongest direct occurrence first. Owner/path
				// matches rank the statement but must not select an unrelated field.
				if g.focus == nil && entityNameScore(m.Kind, m.Name, strings.ToLower(query)) > 0 {
					s := m.Symbol
					g.focus = &s
				}
			}
		}
	}
	incoming := map[qlik.Entity]int{}
	for _, e := range idx.Edges() {
		incoming[e.To]++
	}
	ordered := make([]*candidate, 0, len(groups))
	for _, g := range groups {
		for _, s := range g.symbols {
			if s.Role == "definition" && s.Kind == qlik.Table {
				g.score += 100 + 5*incoming[qlik.Entity{Kind: s.Kind, Name: s.Name}]
			}
		}
		ordered = append(ordered, g)
	}
	sort.Slice(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if a.queryScore != b.queryScore {
			return a.queryScore > b.queryScore
		}
		if a.score != b.score {
			return a.score > b.score
		}
		if a.key.path != b.key.path {
			return a.key.path < b.key.path
		}
		return a.key.offset < b.key.offset
	})
	w := budgetWriter{limit: tokens}
	for _, g := range ordered {
		if !w.add(renderCandidate(g, false, g.focus)) && !w.add(renderCandidate(g, true, g.focus)) {
			w.add(renderCandidate(g, true, nil))
		}
	}
	return w.String()
}

func candidates(idx *qlik.Index) map[statementKey]*candidate {
	groups := map[statementKey]*candidate{}
	for _, f := range idx.Files {
		byOffset := map[uint]*candidate{}
		group := func(key statementKey, stopByte uint) *candidate {
			g := groups[key]
			if g == nil {
				g = &candidate{key: key}
				if key.offset <= stopByte && stopByte <= uint(len(f.Source)) {
					g.text = strings.TrimSpace(string(f.Source[key.offset:stopByte]))
				}
				groups[key] = g
			}
			byOffset[key.offset] = g
			return g
		}
		for _, stmt := range f.Statements {
			key := statementKey{stmt.Path, stmt.Line, stmt.EndLine, stmt.StartByte}
			group(key, stmt.StopByte).kind = stmt.Kind
		}
		for _, load := range f.Loads {
			key := statementKey{load.Path, load.Line, load.EndLine, load.StartByte}
			group(key, load.StopByte).load = &load
		}
		for _, s := range f.Symbols {
			key := statementKey{s.Path, s.StartLine, s.StopLine, s.StartByte}
			g := group(key, s.StopByte)
			g.symbols = append(g.symbols, s)
		}
		for _, e := range f.Edges {
			if g := byOffset[e.StartByte]; g != nil {
				g.edges = append(g.edges, e)
				g.score += 5
			}
		}
	}
	return groups
}

func renderCandidate(g *candidate, compact bool, focus *qlik.Symbol) string {
	if text, ok := renderOperation(g, compact, focus); ok {
		return text
	}
	label := ""
	fields, uses := []string{}, []string{}
	for _, s := range g.symbols {
		switch {
		case s.Kind == qlik.Table && s.Role == "definition":
			label = "table " + s.Name
		case s.Kind == qlik.Variable && s.Role == "definition":
			label = "variable " + s.Name
			if !compact && len(g.text) <= 160 && !strings.ContainsAny(g.text, "\r\n") {
				label = strings.TrimSuffix(g.text, ";")
			}
		case s.Kind == qlik.Source && s.Role == "definition":
			label = "store " + s.Owner + " -> " + s.Name
		case s.Kind == qlik.Include:
			mode := "include"
			for _, e := range g.edges {
				if e.Kind == "must_include" {
					mode = "must_include"
				}
			}
			label = mode + " " + s.Name
		case s.Kind == qlik.Field && s.Role == "definition":
			fields = appendUnique(fields, s.Name)
		case s.Kind == qlik.Variable && s.Role == "reference":
			uses = appendUnique(uses, "$("+s.Name+")")
		}
	}
	if label == "" {
		label = "load"
		if g.load != nil && !g.load.Anonymous {
			label = "extend " + g.load.Owner
		}
	}
	parts := []string{fmt.Sprintf("%s:%d %s", g.key.path, g.key.start, label)}
	fieldCount := len(fields)
	if compact {
		fields, uses = nil, nil
		if s := focus; s != nil {
			switch {
			case s.Kind == qlik.Field && s.Role == "definition":
				fields = []string{s.Name}
			case s.Kind == qlik.Variable && s.Role == "reference":
				uses = []string{"$(" + s.Name + ")"}
			}
		}
	}
	if len(fields) > 0 {
		summary := strings.Join(fields, ", ")
		if compact && fieldCount > len(fields) {
			summary += ", …"
		}
		parts = append(parts, "fields: "+summary)
	}
	if s := focus; s != nil && s.Kind == qlik.Field && s.Role == "reference" {
		parts = append(parts, "refs: "+s.Name)
	}
	inputs := []string{}
	for _, e := range g.edges {
		if e.Kind == "from" || e.Kind == "resident" || e.Kind == "preceding" {
			inputs = appendUnique(inputs, e.Kind+" "+e.To.Name)
		}
	}
	if len(inputs) > 0 {
		parts = append(parts, "<- "+strings.Join(inputs, ", "))
	}
	if len(uses) > 0 {
		parts = append(parts, "uses: "+strings.Join(uses, ", "))
	}
	return strings.Join(parts, " | ") + "\n"
}

func appendUnique(values []string, value string) []string {
	for _, v := range values {
		if v == value {
			return values
		}
	}
	return append(values, value)
}

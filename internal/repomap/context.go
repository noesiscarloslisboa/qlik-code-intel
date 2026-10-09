package repomap

import (
	"fmt"
	"strings"

	"github.com/noesiscarloslisboa/qlik-code-intel/internal/qlik"
)

// Context selects relevant original source lines, preferring exact definitions
// and their direct dependency endpoints. Headers and line numbers count in budget.
func Context(idx *qlik.Index, query string, tokens int) string {
	if tokens <= 0 {
		return ""
	}
	query = strings.TrimSpace(query)
	lowerQuery := strings.ToLower(query)
	matches := Find(idx, Search{Query: query})
	files := map[string]qlik.File{}
	linesByPath := map[string][]string{}
	for _, f := range idx.Files {
		files[f.Path] = f
		linesByPath[f.Path] = strings.Split(string(f.Source), "\n")
	}
	definitions := map[qlik.Entity][]qlik.Symbol{}
	for _, s := range idx.Symbols() {
		if s.Role == "definition" {
			endpoint := qlik.Entity{Kind: s.Kind, Name: s.Name}
			definitions[endpoint] = append(definitions[endpoint], s)
		}
	}
	// Anonymous LOAD stages are dependency endpoints with source ranges, not
	// table definitions. Include them only in direct-neighbor source retrieval.
	for _, f := range idx.Files {
		for _, load := range f.Loads {
			if !load.Anonymous {
				continue
			}
			endpoint := qlik.Entity{Kind: qlik.Table, Name: load.Owner}
			definitions[endpoint] = append(definitions[endpoint], qlik.Symbol{
				Location: load.Location, Kind: qlik.Table, Name: load.Owner,
				StartLine: load.Line, StopLine: load.EndLine,
				StartByte: load.StartByte, StopByte: load.StopByte,
			})
		}
	}
	// Append definitions of direct neighbors after primary matches, preserving
	// stable edge order and avoiding any recursive lineage inference.
	seeds := map[qlik.Entity]bool{}
	best := 0
	for _, m := range matches {
		if score := entityNameScore(m.Kind, m.Name, lowerQuery); score > best {
			best = score
		}
	}
	if best > 0 {
		// A real symbol match should not pull in every other statement whose
		// owner or filename happens to contain the same query text.
		primary := matches[:0]
		for _, m := range matches {
			if entityNameScore(m.Kind, m.Name, lowerQuery) > 0 {
				primary = append(primary, m)
			}
		}
		matches = primary
	}
	for _, m := range matches {
		if entityNameScore(m.Kind, m.Name, lowerQuery) < best {
			continue
		}
		if m.Kind == qlik.Field && m.Owner != "" {
			seeds[qlik.Entity{Kind: qlik.Table, Name: m.Owner}] = true
		} else {
			seeds[qlik.Entity{Kind: m.Kind, Name: m.Name}] = true
		}
		// File edges (such as includes) are relevant to explicit path queries,
		// not every symbol that happens to live in the same file.
		if best == 0 && nameScore(m.Path, lowerQuery) > 0 {
			seeds[qlik.Entity{Kind: qlik.FileKind, Name: m.Path}] = true
		}
	}
	for _, d := range idx.Edges() {
		if !seeds[d.From] && !seeds[d.To] {
			continue
		}
		for _, endpoint := range []qlik.Entity{d.From, d.To} {
			if endpoint.Kind == qlik.FileKind || endpoint.Kind == qlik.Include {
				if f, ok := files[endpoint.Name]; ok && len(f.Source) > 0 {
					matches = append(matches, Match{Symbol: qlik.Symbol{Location: qlik.Location{Path: f.Path, Line: 1}, StartLine: 1, StopLine: strings.Count(string(f.Source), "\n") + 1}})
				}
			}
			for _, s := range definitions[endpoint] {
				matches = append(matches, Match{Symbol: s})
			}
		}
	}
	seen := map[string]map[int]bool{}
	w := budgetWriter{limit: tokens}
	for _, m := range matches {
		lines, ok := linesByPath[m.Path]
		if !ok {
			continue
		}
		if seen[m.Path] == nil {
			seen[m.Path] = map[int]bool{}
		}
		start, stop := m.StartLine, m.StopLine
		if start < 1 || stop < start || start > len(lines) {
			continue
		}
		if stop > len(lines) {
			stop = len(lines)
		}
		// Attempt complete statements. If too large, favor the exact occurrence
		// line and its adjacent lines so a late matching field remains visible.
		if addSourceRange(&w, m.Path, lines, start, stop, seen[m.Path]) {
			continue
		}
		for _, line := range []int{m.Line, m.Line - 1, m.Line + 1, start, stop} {
			if line >= start && line <= stop {
				addSourceRange(&w, m.Path, lines, line, line, seen[m.Path])
			}
		}
	}
	return w.String()
}

func addSourceRange(w *budgetWriter, filePath string, lines []string, start, stop int, seen map[int]bool) bool {
	var b strings.Builder
	// A header is emitted for every distinct contiguous run; this makes omitted
	// lines explicit and prevents falsely implying disjoint ranges are adjacent.
	previous := 0
	selected := []int{}
	for line := start; line <= stop; line++ {
		if seen[line] {
			continue
		}
		if previous == 0 || previous+1 != line {
			fmt.Fprintf(&b, "%s:%d\n", filePath, line)
		}
		fmt.Fprintf(&b, "%d | %s\n", line, strings.TrimSuffix(lines[line-1], "\r"))
		previous = line
		selected = append(selected, line)
	}
	if len(selected) == 0 {
		return true
	}
	if !w.add(b.String()) {
		return false
	}
	for _, line := range selected {
		seen[line] = true
	}
	return true
}

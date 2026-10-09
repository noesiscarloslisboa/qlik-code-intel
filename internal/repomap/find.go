package repomap

import (
	"path"
	"sort"
	"strings"

	"github.com/noesiscarloslisboa/qlik-code-intel/internal/qlik"
)

// Search controls ranked occurrence retrieval. Kind "qvd" filters data sources.
type Search struct {
	Query string
	Kind  string
	Role  string
	Limit int // Zero means all results.
}

// Match retains the occurrence, location, and ranking score.
type Match struct {
	qlik.Symbol
	Score int `json:"score"`
}

// Find ranks exact names before partial names, owners, and paths. Matching is
// case-insensitive for discovery; dependency endpoint identities retain case.
func Find(idx *qlik.Index, opts Search) []Match {
	query := strings.ToLower(strings.TrimSpace(opts.Query))
	matches := []Match{}
	for _, symbol := range idx.Symbols() {
		if !kindMatches(symbol.Kind, symbol.Name, opts.Kind) || (opts.Role != "" && opts.Role != symbol.Role) {
			continue
		}
		score := entityNameScore(symbol.Kind, symbol.Name, query)
		if score == 0 && query != "" {
			if strings.Contains(strings.ToLower(symbol.Owner), query) {
				score = 250
			}
			if strings.Contains(strings.ToLower(symbol.Path), query) && score < 100 {
				score = 100
			}
		}
		if score == 0 {
			continue
		}
		if symbol.Role == "definition" {
			score += 10
		}
		matches = append(matches, Match{Symbol: symbol, Score: score})
	}
	sort.SliceStable(matches, func(i, j int) bool {
		a, b := matches[i], matches[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Column != b.Column {
			return a.Column < b.Column
		}
		return a.Kind < b.Kind
	})
	if opts.Limit > 0 && len(matches) > opts.Limit {
		matches = matches[:opts.Limit]
	}
	return matches
}

// A source's literal filename after leading expansions is a discovery match,
// not an evaluated path or an assertion that two endpoints identify one file.
func entityNameScore(kind qlik.Kind, name, query string) int {
	score := nameScore(name, query)
	if score >= 900 || (kind != qlik.Source && kind != qlik.Include) {
		return score
	}
	base := path.Base(strings.ReplaceAll(name, `\`, "/"))
	if !strings.HasPrefix(base, "$(") {
		return score
	}
	for strings.HasPrefix(base, "$(") {
		end := expansionEnd(base)
		if end == 0 {
			return score
		}
		base = base[end:]
	}
	if base != "" && strings.EqualFold(base, query) {
		return 900
	}
	return score
}

// Find the end of a leading balanced expansion in an indexed name. Quotes
// protect parentheses; nesting is iterative and no expansion is evaluated.
func expansionEnd(name string) int {
	depth, quote := 0, byte(0)
	for i := 1; i < len(name); i++ {
		c := name[i]
		if quote != 0 {
			if c == quote {
				if quote != '`' && i+1 < len(name) && name[i+1] == quote {
					i++
				} else {
					quote = 0
				}
			}
			continue
		}
		switch c {
		case '(', ')':
			if c == '(' {
				depth++
			} else {
				depth--
				if depth == 0 {
					return i + 1
				}
			}
		case '\'', '"', '`':
			quote = c
		case '[':
			quote = ']'
		}
	}
	return 0
}

func nameScore(name, query string) int {
	name = strings.ToLower(name)
	switch {
	case query == "":
		return 1
	case name == query:
		return 1000
	case path.Base(strings.ReplaceAll(name, `\`, "/")) == query:
		return 900
	case strings.HasPrefix(name, query):
		return 800
	case strings.Contains(name, query):
		return 700
	default:
		return 0
	}
}

func kindMatches(kind qlik.Kind, name, filter string) bool {
	if filter == "" {
		return true
	}
	if filter == "qvd" {
		return kind == qlik.Source && strings.EqualFold(path.Ext(name), ".qvd")
	}
	return string(kind) == filter
}

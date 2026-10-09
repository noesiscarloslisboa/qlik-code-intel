package repomap

import (
	"sort"
	"strings"

	"github.com/noesiscarloslisboa/qlik-code-intel/internal/qlik"
)

// Dependency is a direct edge incident to a matching endpoint.
type Dependency struct {
	qlik.Edge
	Direction string `json:"direction"`
}

// Dependencies retrieves direct upstream/downstream edges, not runtime lineage.
// Exact endpoint names (or source basenames) take priority over partial matches.
func Dependencies(idx *qlik.Index, query, kind, direction string) []Dependency {
	q := strings.ToLower(strings.TrimSpace(query))
	edges := idx.Edges()
	bestUp, bestDown := 0, 0
	endpoints := map[qlik.Entity]int{}
	for _, e := range edges {
		for i, endpoint := range []qlik.Entity{e.From, e.To} {
			if !kindMatches(endpoint.Kind, endpoint.Name, kind) {
				continue
			}
			score := entityNameScore(endpoint.Kind, endpoint.Name, q)
			endpoints[endpoint] = score
			if i == 0 {
				bestUp = max(bestUp, score)
			} else {
				bestDown = max(bestDown, score)
			}
		}
	}
	out := []Dependency{}
	seen := map[qlik.Edge]bool{}
	for _, e := range edges {
		if bestUp > 0 && endpoints[e.From] == bestUp && (direction == "upstream" || direction == "both") {
			out = append(out, Dependency{Edge: e, Direction: "upstream"})
			seen[e] = true
		}
		if bestDown > 0 && endpoints[e.To] == bestDown && (direction == "downstream" || direction == "both") && !seen[e] {
			out = append(out, Dependency{Edge: e, Direction: "downstream"})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Column != b.Column {
			return a.Column < b.Column
		}
		return a.Direction < b.Direction
	})
	return out
}

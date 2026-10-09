package repomap

import (
	"fmt"
	"strings"

	"github.com/noesiscarloslisboa/qlik-code-intel/internal/qlik"
)

func renderOperation(g *candidate, compact bool, focus *qlik.Symbol) (string, bool) {
	label := ""
	parts := []string{}
	switch g.kind {
	case "rename_field", "rename_table":
		kind := qlik.Field
		if g.kind == "rename_table" {
			kind = qlik.Table
		}
		oldNames := operationNames(g, kind, "reference")
		newNames := operationNames(g, kind, "definition")
		if len(newNames) == 0 {
			maps := operationNames(g, qlik.Table, "reference")
			label = "rename " + string(kind) + "s using " + strings.Join(maps, ", ")
			break
		}
		pairs := []string{}
		for i, name := range newNames {
			if i >= len(oldNames) {
				break
			}
			pairs = append(pairs, oldNames[i]+" -> "+name)
		}
		if compact && len(pairs) > 1 {
			selected := 0
			if focus != nil && focus.Kind == kind {
				for i := range pairs {
					if oldNames[i] == focus.Name || newNames[i] == focus.Name {
						selected = i
						break
					}
				}
			}
			pairs = []string{pairs[selected], "…"}
		}
		label = "rename " + string(kind) + " " + strings.Join(pairs, ", ")
	case "drop_field":
		fields := operationNames(g, qlik.Field, "reference")
		tables := operationNames(g, qlik.Table, "reference")
		label = "drop fields " + operationList(fields, qlik.Field, compact, focus)
		if len(tables) > 0 {
			parts = append(parts, "from: "+operationList(tables, qlik.Table, compact, focus))
		}
	case "drop_table", "drop_mapping_table":
		label = "drop tables "
		if g.kind == "drop_mapping_table" {
			label = "drop mapping tables "
		}
		label += operationList(operationNames(g, qlik.Table, "reference"), qlik.Table, compact, focus)
	case "trace":
		label = "trace"
	default:
		return "", false
	}
	uses := operationNames(g, qlik.Variable, "reference")
	if compact && len(uses) > 0 {
		uses = []string{}
		if focus != nil && focus.Kind == qlik.Variable && focus.Role == "reference" {
			uses = []string{focus.Name}
		}
	}
	if len(uses) > 0 {
		quoted := []string{}
		for _, name := range uses {
			quoted = appendUnique(quoted, "$("+name+")")
		}
		parts = append(parts, "uses: "+strings.Join(quoted, ", "))
	}
	parts = append([]string{fmt.Sprintf("%s:%d %s", g.key.path, g.key.start, label)}, parts...)
	return strings.Join(parts, " | ") + "\n", true
}

func operationNames(g *candidate, kind qlik.Kind, role string) []string {
	names := []string{}
	for _, s := range g.symbols {
		if s.Kind == kind && s.Role == role {
			names = append(names, s.Name)
		}
	}
	return names
}

func operationList(names []string, kind qlik.Kind, compact bool, focus *qlik.Symbol) string {
	if !compact || len(names) <= 1 {
		return strings.Join(names, ", ")
	}
	name := names[0]
	if focus != nil && focus.Kind == kind {
		for _, candidate := range names {
			if candidate == focus.Name {
				name = candidate
				break
			}
		}
	}
	return name + ", …"
}

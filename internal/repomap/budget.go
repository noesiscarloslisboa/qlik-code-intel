// Package repomap retrieves compact, source-backed views of a Qlik index.
package repomap

import "strings"

// EstimateTokens deliberately counts one token per UTF-8 byte. This conservative,
// model-independent bound needs no tokenizer and never splits a source line.
// It can substantially underfill a model's context; see README for the tradeoff.
func EstimateTokens(text string) int { return len(text) }

type budgetWriter struct {
	strings.Builder
	limit int
}

func (w *budgetWriter) add(text string) bool {
	if EstimateTokens(text) > w.limit-w.Len() {
		return false
	}
	w.WriteString(text)
	return true
}

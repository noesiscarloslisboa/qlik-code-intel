// Package qlik builds a source-backed index of Qlik load scripts.
package qlik

// Kind identifies a Qlik symbol or dependency endpoint.
type Kind string

const (
	Variable Kind = "variable"
	Table    Kind = "table"
	Field    Kind = "field"
	Source   Kind = "source"
	Include  Kind = "include"
	FileKind Kind = "file"
)

// Location uses repository-relative slash paths and one-based lines and byte columns.
type Location struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	EndLine int    `json:"end_line"`
}

// Entity is a typed, case-sensitive dependency endpoint. Names remain unevaluated.
type Entity struct {
	Kind Kind   `json:"kind"`
	Name string `json:"name"`
}

// Symbol is an occurrence, not a globally deduplicated name.
type Symbol struct {
	Location
	Kind      Kind   `json:"kind"`
	Role      string `json:"role"`
	Name      string `json:"name"`
	Owner     string `json:"owner,omitempty"`
	StartLine int    `json:"statement_start"`
	StopLine  int    `json:"statement_end"`
	StartByte uint   `json:"statement_start_byte"`
	StopByte  uint   `json:"statement_end_byte"`
}

// Edge points from a consumer to an upstream input. STORE points from output to table.
type Edge struct {
	Location
	From      Entity `json:"from"`
	To        Entity `json:"to"`
	Kind      string `json:"kind"`
	StartByte uint   `json:"statement_start_byte"`
}

// Diagnostic describes unsupported syntax or a recoverable parse error.
type Diagnostic struct {
	Location
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Load preserves a statement's syntactic owner and source range, even when its
// field list is only a wildcard. Anonymous owners are source locations, not
// inferred Qlik runtime table names. Intermediate stages keep separate owners.
type Load struct {
	Location
	Owner     string `json:"owner"`
	Anonymous bool   `json:"anonymous"`
	StartByte uint   `json:"statement_start_byte"`
	StopByte  uint   `json:"statement_end_byte"`
}

// File owns the exact source used to build the index. Source is omitted from JSON.
type File struct {
	Path        string       `json:"path"`
	Symbols     []Symbol     `json:"symbols"`
	Edges       []Edge       `json:"edges"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Loads       []Load       `json:"loads"`
	Source      []byte       `json:"-"`
}

// Index is a fresh, deterministic snapshot. It has no persistent cache.
type Index struct {
	Root  string `json:"root"`
	Files []File `json:"files"`
}

// Symbols returns all occurrences in scan order.
func (idx *Index) Symbols() []Symbol {
	var symbols []Symbol
	for _, file := range idx.Files {
		symbols = append(symbols, file.Symbols...)
	}
	return symbols
}

// Edges returns direct syntactic relationships in scan order.
func (idx *Index) Edges() []Edge {
	var edges []Edge
	for _, file := range idx.Files {
		edges = append(edges, file.Edges...)
	}
	return edges
}

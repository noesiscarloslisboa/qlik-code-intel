// Package cli exposes a testable command-line interface with injectable I/O.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/noesiscarloslisboa/qlik-code-intel/internal/qlik"
	"github.com/noesiscarloslisboa/qlik-code-intel/internal/repomap"
)

// Version can be set with -ldflags '-X github.com/noesiscarloslisboa/qlik-code-intel/internal/cli.Version=...'.
var Version = "dev"

const help = `qlik-repomap — source-backed repository maps for Qlik load scripts

Usage: qlik-repomap <command> [flags] [query]

Commands:
  scan       Index .qvs files; use --json to export symbols and direct edges
  map        Generate a compact repository map within --tokens (default 4096)
  find       Locate table, variable, field, source, QVD, and include occurrences
  deps       Show direct upstream/downstream dependencies of a name
  context    Retrieve relevant original source within --tokens (default 4096)
  version    Print version

All commands accept --root DIR (default .). Flags can precede or follow queries.
Use qlik-repomap <command> --help for flags. No script execution or persisted index.
`

type options struct {
	root, query, kind, role, direction string
	tokens, limit                      int
	json, strict                       bool
}

// Run returns 0 on success, 2 for usage errors, and 1 for runtime/strict errors.
// Structured output goes to stdout; recoverable diagnostics go to stderr.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		if _, err := io.WriteString(stdout, help); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	command := args[0]
	if command == "version" || command == "--version" {
		if len(args) != 1 {
			fmt.Fprintln(stderr, "version takes no arguments")
			return 2
		}
		if _, err := fmt.Fprintln(stdout, Version); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	if command != "scan" && command != "map" && command != "find" && command != "deps" && command != "context" {
		fmt.Fprintf(stderr, "unknown command %q; use --help\n", command)
		return 2
	}
	o := options{}
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.root, "root", ".", "Repository directory to scan")
	if command == "scan" || command == "find" || command == "deps" {
		fs.BoolVar(&o.json, "json", false, "Write machine-readable JSON")
	}
	if command == "scan" {
		fs.BoolVar(&o.strict, "strict", false, "Return failure if unsupported syntax or parse diagnostics occur")
	}
	if command == "map" || command == "context" {
		fs.IntVar(&o.tokens, "tokens", 4096, "Conservative token budget (one UTF-8 byte per estimated token)")
	}
	if command == "map" {
		fs.StringVar(&o.query, "query", "", "Prioritize matching symbols or paths")
	}
	if command == "find" || command == "deps" {
		fs.StringVar(&o.kind, "kind", "", "Filter: table, variable, field, source, qvd, include (deps also file)")
	}
	if command == "find" {
		fs.StringVar(&o.role, "role", "", "Filter: definition or reference")
		fs.IntVar(&o.limit, "limit", 50, "Maximum occurrences (0 means all)")
	}
	if command == "deps" {
		fs.StringVar(&o.direction, "direction", "both", "upstream, downstream, or both")
	}
	showHelp := func() error {
		var text strings.Builder
		fmt.Fprintf(&text, "Usage: qlik-repomap %s [flags]", command)
		if command == "find" || command == "deps" || command == "context" {
			fmt.Fprint(&text, " <query>")
		}
		fmt.Fprintln(&text)
		previous := fs.Output()
		fs.SetOutput(&text)
		fs.PrintDefaults()
		fs.SetOutput(previous)
		_, err := io.WriteString(stdout, text.String())
		return err
	}
	fs.Usage = func() {}
	if err := fs.Parse(flagsFirst(fs, args[1:])); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			if err := showHelp(); err != nil {
				fmt.Fprintln(stderr, err)
				return 1
			}
			return 0
		}
		return 2
	}
	if err := validate(command, fs.Args(), &o); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	idx, err := qlik.Scan(ctx, o.root)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	diagnostics := 0
	for _, f := range idx.Files {
		for _, d := range f.Diagnostics {
			diagnostics++
			fmt.Fprintf(stderr, "%s:%d:%d [%s] %s\n", d.Path, d.Line, d.Column, d.Code, d.Message)
		}
	}
	if err := output(command, o, idx, stdout); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if o.strict && diagnostics > 0 {
		return 1
	}
	return 0
}

func validate(command string, args []string, o *options) error {
	needsQuery := command == "find" || command == "deps" || command == "context"
	if needsQuery {
		if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
			return fmt.Errorf("%s requires one nonempty query (quote names containing spaces)", command)
		}
		o.query = args[0]
	} else if len(args) != 0 {
		return fmt.Errorf("%s takes no positional arguments; use --root DIR", command)
	}
	if (command == "map" || command == "context") && o.tokens <= 0 {
		return fmt.Errorf("--tokens must be positive")
	}
	if o.limit < 0 {
		return fmt.Errorf("--limit must be nonnegative")
	}
	if o.kind != "" && o.kind != "table" && o.kind != "variable" && o.kind != "field" && o.kind != "source" && o.kind != "qvd" && o.kind != "include" && !(command == "deps" && o.kind == "file") {
		return fmt.Errorf("unknown --kind %q", o.kind)
	}
	if o.role != "" && o.role != "definition" && o.role != "reference" {
		return fmt.Errorf("unknown --role %q", o.role)
	}
	if command == "deps" && o.direction != "both" && o.direction != "upstream" && o.direction != "downstream" {
		return fmt.Errorf("unknown --direction %q", o.direction)
	}
	return nil
}

// Standard flag stops at the first positional argument. Reorder recognized flag
// values with their flags while retaining an explicit -- as an escape hatch.
func flagsFirst(fs *flag.FlagSet, args []string) []string {
	flags, positionals := []string{}, []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positionals = append(positionals, arg)
			continue
		}
		flags = append(flags, arg)
		name, _, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		f := fs.Lookup(name)
		if f == nil || hasValue {
			continue
		}
		if b, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && b.IsBoolFlag() {
			continue
		}
		if i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	if len(positionals) > 0 {
		flags = append(flags, "--")
		flags = append(flags, positionals...)
	}
	return flags
}

func output(command string, o options, idx *qlik.Index, out io.Writer) error {
	var data any
	var text strings.Builder
	switch command {
	case "scan":
		data = idx
		definitions, references, edges := 0, 0, 0
		for _, f := range idx.Files {
			defs, refs := 0, 0
			for _, s := range f.Symbols {
				if s.Role == "definition" {
					defs++
				} else {
					refs++
				}
			}
			definitions += defs
			references += refs
			edges += len(f.Edges)
			fmt.Fprintf(&text, "%s: %d definitions, %d references, %d dependencies\n", f.Path, defs, refs, len(f.Edges))
		}
		fmt.Fprintf(&text, "Scanned %d QVS files: %d definitions, %d references, %d direct dependencies\n", len(idx.Files), definitions, references, edges)
	case "map":
		text.WriteString(repomap.Map(idx, o.query, o.tokens))
	case "find":
		matches := repomap.Find(idx, repomap.Search{Query: o.query, Kind: o.kind, Role: o.role, Limit: o.limit})
		data = matches
		for _, m := range matches {
			fmt.Fprintf(&text, "%s:%d:%d %s %s %s", m.Path, m.Line, m.Column, m.Role, m.Kind, m.Name)
			if m.Owner != "" {
				fmt.Fprintf(&text, " (in %s)", m.Owner)
			}
			text.WriteByte('\n')
		}
	case "deps":
		deps := repomap.Dependencies(idx, o.query, o.kind, o.direction)
		data = deps
		for _, d := range deps {
			fmt.Fprintf(&text, "%s:%d %s: %s:%s -> %s:%s [%s]\n", d.Path, d.Line, d.Direction, d.From.Kind, d.From.Name, d.To.Kind, d.To.Name, d.Kind)
		}
	case "context":
		text.WriteString(repomap.Context(idx, o.query, o.tokens))
	}
	if o.json {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(data)
	}
	_, err := io.WriteString(out, text.String())
	return err
}

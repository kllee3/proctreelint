// Command proctreelint checks process-tree snapshots for structural
// problems: processes that reference a parent pid never seen in the
// input, duplicate pids, processes that are their own parent, and
// zombies that still have children attached.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	code, err := run(os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "proctreelint:", err)
		if code == 0 {
			code = 2
		}
	}
	os.Exit(code)
}

// jsonFinding is the wire shape for --format json: the same fields as
// Finding plus the source name, since a Finding on its own doesn't know
// which file (or stdin) it came from.
type jsonFinding struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

func run(args []string, out io.Writer) (int, error) {
	fs := flag.NewFlagSet("proctreelint", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return 2, err
	}
	if *format != "text" && *format != "json" {
		return 2, fmt.Errorf("unknown --format %q (want text or json)", *format)
	}

	rest := fs.Args()
	var r io.Reader = os.Stdin
	name := "<stdin>"

	if len(rest) > 0 && rest[0] != "-" {
		f, err := os.Open(rest[0])
		if err != nil {
			return 2, err
		}
		defer f.Close()
		r = f
		name = rest[0]
	}

	findings, err := Lint(r)
	if err != nil {
		return 2, fmt.Errorf("%s: %w", name, err)
	}

	switch *format {
	case "json":
		jf := make([]jsonFinding, len(findings))
		for i, f := range findings {
			jf[i] = jsonFinding{File: name, Line: f.Line, Rule: f.Rule, Message: f.Message}
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(jf); err != nil {
			return 2, err
		}
	default:
		for _, f := range findings {
			fmt.Fprintf(out, "%s:%s\n", name, f)
		}
	}

	if len(findings) > 0 {
		return 1, nil
	}
	return 0, nil
}

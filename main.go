// Command proctreelint checks process-tree snapshots for structural
// problems: processes that reference a parent pid never seen in the
// input, duplicate pids, processes that are their own parent, and
// zombies that still have children attached.
package main

import (
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

func run(args []string, out io.Writer) (int, error) {
	var r io.Reader = os.Stdin
	name := "<stdin>"

	if len(args) > 0 && args[0] != "-" {
		f, err := os.Open(args[0])
		if err != nil {
			return 2, err
		}
		defer f.Close()
		r = f
		name = args[0]
	}

	findings, err := Lint(r)
	if err != nil {
		return 2, fmt.Errorf("%s: %w", name, err)
	}

	for _, f := range findings {
		fmt.Fprintf(out, "%s:%s\n", name, f)
	}

	if len(findings) > 0 {
		return 1, nil
	}
	return 0, nil
}

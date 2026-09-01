package main

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// Finding is a single problem found at a specific line of the input.
type Finding struct {
	Line    int
	Rule    string
	Message string
}

func (f Finding) String() string {
	return fmt.Sprintf("%d: %s: %s", f.Line, f.Rule, f.Message)
}

// process is the small amount of state kept per process. Everything else
// about the line (its original text, surrounding whitespace, comments) is
// discarded once parsed.
type process struct {
	pid, ppid int
	line      int
	state     string
	comm      string
}

// Lint reads a process-tree snapshot from r and returns every finding, in
// the order that its checks run. The expected format is one process per
// line:
//
//	pid ppid state comm
//
// fields separated by whitespace, comm taking the rest of the line. Blank
// lines and lines starting with '#' are ignored. A ppid of 0 marks a root
// and is never treated as missing.
//
// Lint is written for streaming input: it reads line by line with a
// bounded scan buffer and never holds the raw input in memory. The only
// state that survives past a given line is one small process record per
// pid, which is what makes cross-line checks (duplicate pids, parents
// that are never defined) possible without buffering the whole file.
func Lint(r io.Reader) ([]Finding, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var findings []Finding
	byPID := make(map[int]process)

	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			findings = append(findings, Finding{lineNo, "parse-error",
				"expected at least 3 fields: pid ppid state [comm]"})
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			findings = append(findings, Finding{lineNo, "parse-error",
				fmt.Sprintf("invalid pid %q", fields[0])})
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			findings = append(findings, Finding{lineNo, "parse-error",
				fmt.Sprintf("invalid ppid %q", fields[1])})
			continue
		}

		comm := ""
		if len(fields) > 3 {
			comm = strings.Join(fields[3:], " ")
		}
		p := process{pid: pid, ppid: ppid, line: lineNo, state: fields[2], comm: comm}

		if pid == ppid {
			findings = append(findings, Finding{lineNo, "self-parent",
				fmt.Sprintf("pid %d lists itself as its own parent", pid)})
		}

		if prev, ok := byPID[pid]; ok {
			findings = append(findings, Finding{lineNo, "duplicate-pid",
				fmt.Sprintf("pid %d already defined on line %d", pid, prev.line)})
		}

		byPID[pid] = p
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Checks below need the full set of pids, so they run after the scan
	// loop. They still only touch the parsed records, not the raw text.
	hasChildren := make(map[int]bool)
	for _, p := range byPID {
		if p.ppid != p.pid {
			hasChildren[p.ppid] = true
		}
	}

	for _, p := range byPID {
		if p.ppid == 0 {
			continue
		}
		if _, ok := byPID[p.ppid]; !ok {
			findings = append(findings, Finding{p.line, "unknown-parent",
				fmt.Sprintf("pid %d has ppid %d, which never appears in the input", p.pid, p.ppid)})
		}
		if p.state == "Z" && hasChildren[p.pid] {
			findings = append(findings, Finding{p.line, "zombie-with-children",
				fmt.Sprintf("pid %d is a zombie (state Z) but still has children reparented to it", p.pid)})
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].Rule < findings[j].Rule
	})

	return findings, nil
}

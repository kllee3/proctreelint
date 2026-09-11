package main

import (
	"strings"
	"testing"
)

// rules returns the set of rule names present in findings.
func rules(findings []Finding) map[string]bool {
	set := make(map[string]bool)
	for _, f := range findings {
		set[f.Rule] = true
	}
	return set
}

func mustLint(t *testing.T, input string) []Finding {
	t.Helper()
	findings, err := Lint(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}
	return findings
}

func TestLintCleanInputHasNoFindings(t *testing.T) {
	input := `1 0 S init
100 1 S sshd
205 100 S bash
`
	findings := mustLint(t, input)
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %v", findings)
	}
}

func TestLintIgnoresBlankLinesAndComments(t *testing.T) {
	input := `# header comment

1 0 S init

# trailing comment
`
	findings := mustLint(t, input)
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %v", findings)
	}
}

func TestParseErrorTooFewFields(t *testing.T) {
	findings := mustLint(t, "1 0\n")
	if !rules(findings)["parse-error"] {
		t.Fatalf("expected parse-error, got %v", findings)
	}
	if findings[0].Line != 1 {
		t.Fatalf("expected line 1, got %d", findings[0].Line)
	}
}

func TestParseErrorInvalidPid(t *testing.T) {
	findings := mustLint(t, "abc 0 S init\n")
	if !rules(findings)["parse-error"] {
		t.Fatalf("expected parse-error, got %v", findings)
	}
}

func TestParseErrorInvalidPpid(t *testing.T) {
	findings := mustLint(t, "1 xyz S init\n")
	if !rules(findings)["parse-error"] {
		t.Fatalf("expected parse-error, got %v", findings)
	}
}

func TestParseErrorDoesNotAbortLaterLines(t *testing.T) {
	input := `bad line
2 0 S init
`
	findings := mustLint(t, input)
	got := rules(findings)
	if !got["parse-error"] {
		t.Fatalf("expected parse-error, got %v", findings)
	}
	for _, f := range findings {
		if f.Line == 2 {
			t.Fatalf("line 2 should be clean, got finding %v", f)
		}
	}
}

func TestSelfParent(t *testing.T) {
	findings := mustLint(t, "5 5 S loopy\n")
	if !rules(findings)["self-parent"] {
		t.Fatalf("expected self-parent, got %v", findings)
	}
}

func TestDuplicatePid(t *testing.T) {
	input := `1 0 S init
205 1 S bash
205 1 S bash
`
	findings := mustLint(t, input)
	if !rules(findings)["duplicate-pid"] {
		t.Fatalf("expected duplicate-pid, got %v", findings)
	}
	var dup Finding
	found := false
	for _, f := range findings {
		if f.Rule == "duplicate-pid" {
			dup = f
			found = true
		}
	}
	if !found {
		t.Fatalf("duplicate-pid finding missing")
	}
	if dup.Line != 3 {
		t.Fatalf("expected duplicate reported on line 3, got %d", dup.Line)
	}
	if !strings.Contains(dup.Message, "line 2") {
		t.Fatalf("expected message to reference original line 2, got %q", dup.Message)
	}
}

func TestUnknownParent(t *testing.T) {
	findings := mustLint(t, "7 999 S orphan\n")
	if !rules(findings)["unknown-parent"] {
		t.Fatalf("expected unknown-parent, got %v", findings)
	}
}

func TestPpidZeroIsNeverUnknownParent(t *testing.T) {
	findings := mustLint(t, "1 0 S init\n")
	if rules(findings)["unknown-parent"] {
		t.Fatalf("ppid 0 should never be flagged as unknown-parent, got %v", findings)
	}
}

func TestZombieWithChildren(t *testing.T) {
	input := `1 0 S init
2 1 Z dead
3 2 S child-of-zombie
`
	findings := mustLint(t, input)
	if !rules(findings)["zombie-with-children"] {
		t.Fatalf("expected zombie-with-children, got %v", findings)
	}
}

func TestZombieWithoutChildrenIsClean(t *testing.T) {
	input := `1 0 S init
2 1 Z dead
`
	findings := mustLint(t, input)
	if rules(findings)["zombie-with-children"] {
		t.Fatalf("expected no zombie-with-children, got %v", findings)
	}
}

func TestParentCycle(t *testing.T) {
	input := `1 0 S init
10 20 S a
20 10 S b
`
	findings := mustLint(t, input)
	if !rules(findings)["parent-cycle"] {
		t.Fatalf("expected parent-cycle, got %v", findings)
	}
	count := 0
	for _, f := range findings {
		if f.Rule == "parent-cycle" {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("expected both cycle members flagged, got %d parent-cycle findings", count)
	}
}

func TestParentCycleDoesNotDuplicateSelfParent(t *testing.T) {
	findings := mustLint(t, "5 5 S loopy\n")
	if rules(findings)["parent-cycle"] {
		t.Fatalf("a one-node self-parent cycle should be reported as self-parent only, got %v", findings)
	}
}

func TestFindingsAreSortedByLine(t *testing.T) {
	input := `9012 9012 S loopy
1 0 S init
100 999 S orphan
`
	findings := mustLint(t, input)
	for i := 1; i < len(findings); i++ {
		if findings[i].Line < findings[i-1].Line {
			t.Fatalf("findings not sorted by line: %v", findings)
		}
	}
}

func TestCommWithSpacesIsPreserved(t *testing.T) {
	input := "1 0 S kworker/0:1-events\n"
	findings := mustLint(t, input)
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %v", findings)
	}
}

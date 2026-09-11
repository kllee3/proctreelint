package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func writeSample(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sample.pt")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func TestRunExitsZeroOnCleanInput(t *testing.T) {
	path := writeSample(t, "1 0 S init\n100 1 S sshd\n")
	var out bytes.Buffer
	code, err := run([]string{path}, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if out.Len() != 0 {
		t.Fatalf("expected no output, got %q", out.String())
	}
}

func TestRunExitsOneWithFindings(t *testing.T) {
	path := writeSample(t, "5 5 S loopy\n")
	var out bytes.Buffer
	code, err := run([]string{path}, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if out.Len() == 0 {
		t.Fatalf("expected output describing the finding")
	}
}

func TestRunExitsTwoOnMissingFile(t *testing.T) {
	var out bytes.Buffer
	code, err := run([]string{filepath.Join(t.TempDir(), "does-not-exist.pt")}, &out)
	if err == nil {
		t.Fatalf("expected an error for a missing file")
	}
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}

func TestRunOutputIncludesFileNameAndFinding(t *testing.T) {
	path := writeSample(t, "9012 9012 S loopy\n")
	var out bytes.Buffer
	if _, err := run([]string{path}, &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := path + ":1: self-parent: pid 9012 lists itself as its own parent\n"
	if out.String() != want {
		t.Fatalf("output mismatch:\n got  %q\n want %q", out.String(), want)
	}
}

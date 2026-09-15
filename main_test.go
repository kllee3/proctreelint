package main

import (
	"bytes"
	"encoding/json"
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

func TestRunFormatJSONEmitsFindingsAsArray(t *testing.T) {
	path := writeSample(t, "9012 9012 S loopy\n")
	var out bytes.Buffer
	code, err := run([]string{"--format", "json", path}, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}

	var got []jsonFinding
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out.String())
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 finding, got %d: %v", len(got), got)
	}
	if got[0].File != path || got[0].Line != 1 || got[0].Rule != "self-parent" {
		t.Fatalf("unexpected finding: %+v", got[0])
	}
}

func TestRunFormatJSONEmitsEmptyArrayOnCleanInput(t *testing.T) {
	path := writeSample(t, "1 0 S init\n")
	var out bytes.Buffer
	code, err := run([]string{"--format", "json", path}, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	var got []jsonFinding
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out.String())
	}
	if len(got) != 0 {
		t.Fatalf("expected no findings, got %v", got)
	}
}

func TestRunUnknownFormatIsAnError(t *testing.T) {
	path := writeSample(t, "1 0 S init\n")
	var out bytes.Buffer
	code, err := run([]string{"--format", "xml", path}, &out)
	if err == nil {
		t.Fatalf("expected an error for an unknown format")
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

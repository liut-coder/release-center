package appreleases

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTailLogLinesReturnsTailAndTruncation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "buildctl.log")
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	lines, truncated, err := tailLogLines(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !truncated {
		t.Fatal("expected log tail to be marked truncated")
	}
	if len(lines) != 2 || lines[0] != "two" || lines[1] != "three" {
		t.Fatalf("unexpected lines: %#v", lines)
	}
}

func TestTailLogLinesHandlesEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "buildctl.log")
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	lines, truncated, err := tailLogLines(path, 200)
	if err != nil {
		t.Fatal(err)
	}
	if truncated || len(lines) != 0 {
		t.Fatalf("expected empty non-truncated lines, got truncated=%v lines=%#v", truncated, lines)
	}
}

package processor_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const binaryPath = "../bin/merge_clean_nt"

func TestMergeAndClean_MatchesExpected(t *testing.T) {
	cmd := exec.Command(binaryPath, "test_cases/input*.nt")
	cmd.Dir = "."
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("binary exited with error: %v", err)
	}

	expected, err := os.ReadFile("test_cases/expected.nt")
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}

	if string(output) != string(expected) {
		t.Errorf("output mismatch:\ngot:\n%s\nwant:\n%s", output, expected)
	}
}

func TestMergeAndClean_LineCount(t *testing.T) {
	cmd := exec.Command(binaryPath, "test_cases/input*.nt")
	cmd.Dir = "."
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("binary exited with error: %v", err)
	}

	if !strings.Contains(stderr.String(), "Done. Lines: 6") {
		t.Errorf("expected stderr 'Done. Lines: 6', got: %q", stderr.String())
	}
}

func TestMergeAndClean_OutputFlag(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.nt")
	cmd := exec.Command(binaryPath, "-o", outFile, "test_cases/input*.nt")
	cmd.Dir = "."
	if err := cmd.Run(); err != nil {
		t.Fatalf("binary exited with error: %v", err)
	}

	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}

	expected, err := os.ReadFile("test_cases/expected.nt")
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}

	if string(got) != string(expected) {
		t.Errorf("output mismatch with -o flag:\ngot:\n%s\nwant:\n%s", got, expected)
	}
}

func TestMergeAndClean_NoMatchGlob(t *testing.T) {
	cmd := exec.Command(binaryPath, "test_cases/nonexistent*.nt")
	cmd.Dir = "."
	if err := cmd.Run(); err == nil {
		t.Error("expected non-zero exit for unmatched glob, got nil")
	}
}

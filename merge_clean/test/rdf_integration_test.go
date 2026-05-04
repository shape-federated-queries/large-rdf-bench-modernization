package processor_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const rdfBinaryPath = "../bin/merge_clean_rdf"

func TestRDF_SingleFile_MatchesExpected(t *testing.T) {
	cmd := exec.Command(rdfBinaryPath, "test_cases/rdf_single_input.rdf")
	cmd.Dir = "."
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("binary exited with error: %v", err)
	}

	expected, err := os.ReadFile("test_cases/rdf_single_expected.rdf")
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}

	if string(output) != string(expected) {
		t.Errorf("single-file output mismatch:\ngot:\n%s\nwant:\n%s", output, expected)
	}
}

func TestRDF_SingleFile_NoSingleQuotes(t *testing.T) {
	cmd := exec.Command(rdfBinaryPath, "test_cases/rdf_single_input.rdf")
	cmd.Dir = "."
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("binary exited with error: %v", err)
	}
	got := string(output)

	// Single quotes are still allowed in text content and comments; what must not
	// appear is a single-quote attribute delimiter, i.e. ='...
	if strings.Contains(got, "='") {
		t.Errorf("output contains single-quoted attribute delimiter =':\n%s", got)
	}
}

func TestRDF_SingleFile_IRIEncoding(t *testing.T) {
	cmd := exec.Command(rdfBinaryPath, "test_cases/rdf_single_input.rdf")
	cmd.Dir = "."
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("binary exited with error: %v", err)
	}
	got := string(output)

	if !strings.Contains(got, "foo%20bar") {
		t.Errorf("space in IRI not encoded to %%20:\n%s", got)
	}
	if !strings.Contains(got, "%5E") {
		t.Errorf("^ in IRI not encoded to %%5E:\n%s", got)
	}
	if !strings.Contains(got, "%7B") {
		t.Errorf("{ in IRI not encoded to %%7B:\n%s", got)
	}
}

func TestRDF_SingleFile_OutputFlag(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "out.rdf")
	cmd := exec.Command(rdfBinaryPath, "-o", outFile, "test_cases/rdf_single_input.rdf")
	cmd.Dir = "."
	if err := cmd.Run(); err != nil {
		t.Fatalf("binary exited with error: %v", err)
	}

	got, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	expected, err := os.ReadFile("test_cases/rdf_single_expected.rdf")
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}

	if string(got) != string(expected) {
		t.Errorf("-o flag output mismatch:\ngot:\n%s\nwant:\n%s", got, expected)
	}
}

func TestRDF_Merge_MatchesExpected(t *testing.T) {
	cmd := exec.Command(rdfBinaryPath,
		"test_cases/rdf_merge_input1.rdf",
		"test_cases/rdf_merge_input2.rdf",
	)
	cmd.Dir = "."
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("binary exited with error: %v", err)
	}

	expected, err := os.ReadFile("test_cases/rdf_merge_expected.rdf")
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}

	if string(output) != string(expected) {
		t.Errorf("merge output mismatch:\ngot:\n%s\nwant:\n%s", output, expected)
	}
}

func TestRDF_Merge_SingleRootElement(t *testing.T) {
	cmd := exec.Command(rdfBinaryPath,
		"test_cases/rdf_merge_input1.rdf",
		"test_cases/rdf_merge_input2.rdf",
	)
	cmd.Dir = "."
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("binary exited with error: %v", err)
	}
	got := string(output)

	if c := strings.Count(got, "<rdf:RDF"); c != 1 {
		t.Errorf("expected 1 opening <rdf:RDF, got %d", c)
	}
	if c := strings.Count(got, "</rdf:RDF>"); c != 1 {
		t.Errorf("expected 1 closing </rdf:RDF>, got %d", c)
	}
}

func TestRDF_Merge_GlobPattern(t *testing.T) {
	cmd := exec.Command(rdfBinaryPath, "test_cases/rdf_merge_input*.rdf")
	cmd.Dir = "."
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("binary exited with error: %v", err)
	}

	expected, err := os.ReadFile("test_cases/rdf_merge_expected.rdf")
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}

	if string(output) != string(expected) {
		t.Errorf("glob merge output mismatch:\ngot:\n%s\nwant:\n%s", output, expected)
	}
}

func TestRDF_Merge_ByteCount(t *testing.T) {
	var stderr strings.Builder
	cmd := exec.Command(rdfBinaryPath,
		"test_cases/rdf_merge_input1.rdf",
		"test_cases/rdf_merge_input2.rdf",
	)
	cmd.Dir = "."
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("binary exited with error: %v", err)
	}

	if !strings.Contains(stderr.String(), "Done. Bytes:") {
		t.Errorf("expected stderr 'Done. Bytes: ...', got: %q", stderr.String())
	}
}

func TestRDF_NoMatchGlob(t *testing.T) {
	cmd := exec.Command(rdfBinaryPath, "test_cases/nonexistent*.rdf")
	cmd.Dir = "."
	if err := cmd.Run(); err == nil {
		t.Error("expected non-zero exit for unmatched glob, got nil")
	}
}

package testrunner

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
)

func TestRun_Success(t *testing.T) {
	tmpDir := t.TempDir()
	resultsFile := filepath.Join(tmpDir, "results.xml")

	// Execute a shell script or echo command that emits Rust test output
	err := Run("echo_pkg", "sh", []string{"-c", "echo 'test echo_test ... ok'"}, resultsFile)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if _, err := os.Stat(resultsFile); err != nil {
		t.Fatalf("results file was not created: %v", err)
	}
}

func TestRun_Failure(t *testing.T) {
	tmpDir := t.TempDir()
	resultsFile := filepath.Join(tmpDir, "fail_results.xml")

	err := Run("fail_pkg", "sh", []string{"-c", "echo 'test fail_test ... FAILED'; exit 1"}, resultsFile)
	if err == nil {
		t.Fatal("expected error from failing command, got nil")
	}

	data, err := os.ReadFile(resultsFile)
	if err != nil {
		t.Fatalf("reading failure results file: %v", err)
	}

	var suites JUnitTestSuites
	if err := xml.Unmarshal(data, &suites); err != nil {
		t.Fatalf("parsing failure xml: %v", err)
	}
	if suites.TestSuite[0].Failures != 1 {
		t.Errorf("expected 1 failure in xml, got %d", suites.TestSuite[0].Failures)
	}
}

func TestRunWithOptions_CoverageActive(t *testing.T) {
	tmpDir := t.TempDir()
	resultsFile := filepath.Join(tmpDir, "results.xml")
	coverageFile := filepath.Join(tmpDir, "cov.out")

	// Create a dummy profraw file in tmpDir
	dummyProfraw := filepath.Join(tmpDir, "test.profraw")
	_ = os.WriteFile(dummyProfraw, []byte("dummy"), 0644)

	opts := RunOptions{
		PkgName:        "cov_pkg",
		TestBinary:     "true",
		ResultsFile:    resultsFile,
		CoverageActive: true,
		CoverageFile:   coverageFile,
	}

	t.Setenv("TMP_DIR", tmpDir)
	_ = RunWithOptions(opts)

	if _, err := os.Stat(resultsFile); err != nil {
		t.Errorf("expected results.xml to exist, got %v", err)
	}
}

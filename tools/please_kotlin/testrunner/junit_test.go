package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseTestOutputSuccess(t *testing.T) {
	output := "Running MyTest... ok\ntest AnotherTest: PASS\n"
	suite := ParseTestOutput(output, "MySuite", "pkg.MySuite", 100*time.Millisecond, true)

	if suite.Name != "MySuite" {
		t.Errorf("suite.Name = %q, want MySuite", suite.Name)
	}
	if suite.Tests != 2 {
		t.Errorf("suite.Tests = %d, want 2", suite.Tests)
	}
	if suite.Failures != 0 {
		t.Errorf("suite.Failures = %d, want 0", suite.Failures)
	}
}

func TestParseTestOutputFailure(t *testing.T) {
	output := "Running FailingTest... FAILED\n"
	suite := ParseTestOutput(output, "MySuite", "pkg.MySuite", 100*time.Millisecond, false)

	if suite.Tests != 1 {
		t.Errorf("suite.Tests = %d, want 1", suite.Tests)
	}
	if suite.Failures != 1 {
		t.Errorf("suite.Failures = %d, want 1", suite.Failures)
	}
}

func TestWriteJUnitResults(t *testing.T) {
	tmpDir := t.TempDir()
	outXml := filepath.Join(tmpDir, "test.results")

	suite := ParseTestOutput("Running SingleTest... ok\n", "Suite1", "pkg.Suite1", 50*time.Millisecond, true)
	if err := WriteJUnitResults(outXml, suite); err != nil {
		t.Fatalf("WriteJUnitResults failed: %v", err)
	}

	data, err := os.ReadFile(outXml)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if !strings.Contains(string(data), "<testsuite") {
		t.Errorf("missing <testsuite in xml output: %s", string(data))
	}
}

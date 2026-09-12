package testrunner

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseTestOutput(t *testing.T) {
	output := `
running 3 tests
test tests::test_add ... ok
test tests::test_sub ... FAILED
test tests::test_ignored ... ignored

failures:

---- tests::test_sub stdout ----
thread 'tests::test_sub' panicked at 'assertion failed: (left == right)'

test result: FAILED. 1 passed; 1 failed; 1 ignored; 0 measured; 0 filtered out; finished in 0.00s
`
	suites := ParseTestOutput("my_test_pkg", output, 50*time.Millisecond)
	if suites == nil || len(suites.TestSuite) == 0 {
		t.Fatalf("expected test suites to be parsed")
	}

	suite := suites.TestSuite[0]
	if suite.Name != "my_test_pkg" {
		t.Errorf("expected suite name 'my_test_pkg', got '%s'", suite.Name)
	}
	if suite.Tests != 3 {
		t.Errorf("expected 3 tests, got %d", suite.Tests)
	}
	if suite.Failures != 1 {
		t.Errorf("expected 1 failure, got %d", suite.Failures)
	}

	data, err := xml.Marshal(suites)
	if err != nil {
		t.Fatalf("failed to marshal xml: %v", err)
	}
	if len(data) == 0 {
		t.Errorf("expected non-empty XML data")
	}
}

func TestParseTestOutput_NoTests(t *testing.T) {
	// When no tests match testLineRegex, a single synthetic testcase is produced
	suites := ParseTestOutput("empty_pkg", "no test output here", 10*time.Millisecond)
	if len(suites.TestSuite) != 1 {
		t.Fatalf("expected 1 suite, got %d", len(suites.TestSuite))
	}
	suite := suites.TestSuite[0]
	if suite.Tests != 1 {
		t.Errorf("expected 1 synthetic test, got %d", suite.Tests)
	}
	if len(suite.TestCases) != 1 || suite.TestCases[0].Name != "execution" {
		t.Errorf("expected synthetic execution testcase, got %v", suite.TestCases)
	}
}

func TestConvertOutputToJUnit_Success(t *testing.T) {
	tmpDir := t.TempDir()
	rawFile := filepath.Join(tmpDir, "raw_output.txt")
	testOutput := "test test_one ... ok\ntest test_two ... ok\n"
	if err := os.WriteFile(rawFile, []byte(testOutput), 0644); err != nil {
		t.Fatalf("writing raw output: %v", err)
	}

	resultsFile := filepath.Join(tmpDir, "custom_dir", "results.xml")
	if err := ConvertOutputToJUnit("my_pkg", rawFile, resultsFile); err != nil {
		t.Fatalf("ConvertOutputToJUnit failed: %v", err)
	}

	data, err := os.ReadFile(resultsFile)
	if err != nil {
		t.Fatalf("reading generated xml: %v", err)
	}

	var suites JUnitTestSuites
	if err := xml.Unmarshal(data, &suites); err != nil {
		t.Fatalf("unmarshalling generated xml: %v", err)
	}

	if len(suites.TestSuite) != 1 || suites.TestSuite[0].Tests != 2 {
		t.Errorf("expected 1 suite with 2 tests, got %+v", suites)
	}
}

func TestConvertOutputToJUnit_MissingFile(t *testing.T) {
	err := ConvertOutputToJUnit("pkg", "/nonexistent/output.txt", "results.xml")
	if err == nil {
		t.Fatal("expected error for non-existent input file, got nil")
	}
}

func TestEnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	nestedFile := filepath.Join(tmpDir, "a", "b", "c", "out.xml")
	if err := ensureDir(nestedFile); err != nil {
		t.Fatalf("ensureDir failed: %v", err)
	}
	if fi, err := os.Stat(filepath.Dir(nestedFile)); err != nil || !fi.IsDir() {
		t.Fatalf("expected directory to be created: %v", err)
	}
	// Calling ensureDir on a file in current dir or empty dir should succeed
	if err := ensureDir("out.xml"); err != nil {
		t.Fatalf("ensureDir on relative file failed: %v", err)
	}
}

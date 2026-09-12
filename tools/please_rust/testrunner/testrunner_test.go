package testrunner

import (
	"encoding/xml"
	"strings"
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

func TestParseLcov(t *testing.T) {
	rawLcov := `TN:
SF:/path/to/repo/src/lib.rs
DA:1,2
DA:2,0
DA:5,1
end_of_record
SF:/rustc/12345/library/core/src/lib.rs
DA:1,1
end_of_record
`
	files, err := ParseLcov([]byte(rawLcov))
	if err != nil {
		t.Fatalf("ParseLcov failed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0].Path != "/path/to/repo/src/lib.rs" {
		t.Errorf("unexpected path: %s", files[0].Path)
	}
	if len(files[0].LineHits) != 3 {
		t.Errorf("expected 3 line hits, got %d", len(files[0].LineHits))
	}
	if files[0].LineHits[1] != 2 || files[0].LineHits[2] != 0 || files[0].LineHits[5] != 1 {
		t.Errorf("unexpected line hits: %+v", files[0].LineHits)
	}

	norm := NormalizeLcovPaths(files, "/path/to/repo")
	if len(norm) != 1 {
		t.Fatalf("expected 1 file after normalization (rustc filtered), got %d", len(norm))
	}
	if norm[0].Path != "src/lib.rs" {
		t.Errorf("expected normalized path 'src/lib.rs', got '%s'", norm[0].Path)
	}

	gcov := FormatGcov(norm, "")
	gcovStr := string(gcov)
	if !strings.Contains(gcovStr, "Source:src/lib.rs") {
		t.Errorf("expected Source:src/lib.rs in gcov output, got:\n%s", gcovStr)
	}
	if !strings.Contains(gcovStr, "2:    1:code") && !strings.Contains(gcovStr, "2:  1:code") {
		t.Errorf("expected line 1 covered with 2 hits, got:\n%s", gcovStr)
	}
	if !strings.Contains(gcovStr, "#####:    2:code") && !strings.Contains(gcovStr, "#####:  2:code") {
		t.Errorf("expected line 2 uncovered (#####), got:\n%s", gcovStr)
	}
	if !strings.Contains(gcovStr, "-:    3:code") && !strings.Contains(gcovStr, "-:  3:code") {
		t.Errorf("expected line 3 not executable (-), got:\n%s", gcovStr)
	}
}

func TestProcessCoverage(t *testing.T) {
	rawLcov := `SF:/repo/root/test.rs
DA:1,1
end_of_record
`
	normLcov, gcov, err := ProcessCoverage([]byte(rawLcov), "/repo/root")
	if err != nil {
		t.Fatalf("ProcessCoverage failed: %v", err)
	}
	if !strings.Contains(string(normLcov), "SF:test.rs\nDA:1,1\nend_of_record") {
		t.Errorf("unexpected normLcov: %s", string(normLcov))
	}
	if !strings.HasPrefix(string(gcov), "        -:    0:Source:test.rs\n") {
		t.Errorf("unexpected gcov output: %s", string(gcov))
	}
}


package testrunner

import (
	"encoding/xml"
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

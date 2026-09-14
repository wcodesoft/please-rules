package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTestOutputSwiftTesting(t *testing.T) {
	output := `
◇ Test run started.
↳ Testing Library Version: 6.3.3
↳ Target Platform: x86_64-unknown-linux-gnu
◇ Test testAdd() started.
✔ Test testAdd() passed after 0.001 seconds.
◇ Test testFail() started.
✘ Test testFail() recorded an issue at MathTests.swift:15:5: Expectation failed: 1 == 2
✘ Test testFail() failed after 0.002 seconds with 1 issue.
✔ Test run with 1 test in 0 suites passed after 0.003 seconds.
`
	cases := parseTestOutput(output, "swift-testing")
	if len(cases) != 2 {
		t.Fatalf("expected 2 test cases, got %d", len(cases))
	}

	if cases[0].Name != "testAdd" || !cases[0].Passed {
		t.Errorf("expected testAdd passed, got %+v", cases[0])
	}
	if cases[1].Name != "testFail" || cases[1].Passed {
		t.Errorf("expected testFail failed, got %+v", cases[1])
	}
	if !strings.Contains(cases[1].Failure, "MathTests.swift:15:5") {
		t.Errorf("expected failure message to contain issue details, got %q", cases[1].Failure)
	}
}

func TestParseTestOutputXCTest(t *testing.T) {
	output := `
Test Suite 'All tests' started at 2026-09-14 20:00:00.000
Test Suite 'MathTests.xctest' started at 2026-09-14 20:00:00.001
Test Case '-[MathTests testAddition]' started.
Test Case '-[MathTests testAddition]' passed (0.005 seconds).
Test Case '-[MathTests testFailure]' started.
Test Case '-[MathTests testFailure]' failed (0.010 seconds).
`
	cases := parseTestOutput(output, "xctest")
	if len(cases) != 2 {
		t.Fatalf("expected 2 test cases, got %d", len(cases))
	}

	if cases[0].Name != "testAddition" || !cases[0].Passed {
		t.Errorf("expected testAddition passed, got %+v", cases[0])
	}
	if cases[1].Name != "testFailure" || cases[1].Passed {
		t.Errorf("expected testFailure failed, got %+v", cases[1])
	}
}

func TestWriteJUnitResults(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "junit-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	resultsPath := filepath.Join(tmpDir, "test.results")
	cases := []ParsedTestCase{
		{
			Suite:    "SwiftTesting",
			Name:     "testOne",
			Duration: 0.05,
			Passed:   true,
		},
		{
			Suite:    "SwiftTesting",
			Name:     "testTwo",
			Duration: 0.10,
			Passed:   false,
			Failure:  "Assertion failed",
		},
	}

	err = writeJUnitResults(resultsPath, cases, 0.15)
	if err != nil {
		t.Fatalf("failed to write junit results: %v", err)
	}

	data, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatalf("failed to read written junit results: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, `<testsuite name="SwiftTesting" tests="2" failures="1"`) {
		t.Errorf("unexpected xml testsuite header: %s", content)
	}
	if !strings.Contains(content, `<testcase name="testOne"`) {
		t.Errorf("missing testOne testcase: %s", content)
	}
	if !strings.Contains(content, `<failure message="Assertion failed"`) {
		t.Errorf("missing failure element: %s", content)
	}
}

func TestLcovToCoberturaXML(t *testing.T) {
	lcov := `
SF:/path/to/repo/test/swift/lib/Math.swift
DA:1,1
DA:2,1
DA:3,0
end_of_record
SF:/path/to/repo/__runner_main.swift
DA:1,1
end_of_record
`
	xmlBytes := LcovToCoberturaXML([]byte(lcov), "/path/to/repo")
	xmlStr := string(xmlBytes)

	if strings.Contains(xmlStr, "__runner_main.swift") {
		t.Errorf("expected __runner_main.swift to be ignored, got:\n%s", xmlStr)
	}
	if !strings.Contains(xmlStr, `filename="test/swift/lib/Math.swift"`) {
		t.Errorf("expected normalized path test/swift/lib/Math.swift, got:\n%s", xmlStr)
	}
	if !strings.Contains(xmlStr, `<line number="1" hits="1"/>`) {
		t.Errorf("expected hit on line 1, got:\n%s", xmlStr)
	}
	if !strings.Contains(xmlStr, `<line number="3" hits="0"/>`) {
		t.Errorf("expected 0 hits on line 3, got:\n%s", xmlStr)
	}
}

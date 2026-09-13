package testrunner

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// JUnitTestSuites matches standard JUnit XML testsuites container.
type JUnitTestSuites struct {
	XMLName   xml.Name         `xml:"testsuites"`
	TestSuite []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite matches standard JUnit XML testsuite element.
type JUnitTestSuite struct {
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Time      string          `xml:"time,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase matches standard JUnit XML testcase element.
type JUnitTestCase struct {
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
}

// JUnitFailure represents a test case failure.
type JUnitFailure struct {
	Message  string `xml:"message,attr"`
	Type     string `xml:"type,attr"`
	Contents string `xml:",chardata"`
}

var testLineRegex = regexp.MustCompile(`(?i)(?:test|running)\s+([^\s:]+)(?::\s*|\s*\.\.\.\s*)(ok|pass|passed|fail|failed|failure)`)

// ParseTestOutput extracts testcases from runner stdout/stderr.
func ParseTestOutput(output string, fallbackName, classname string, duration time.Duration, success bool) JUnitTestSuite {
	var testcases []JUnitTestCase
	durStr := fmt.Sprintf("%.3f", duration.Seconds())

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		matches := testLineRegex.FindStringSubmatch(trimmed)
		if len(matches) >= 3 {
			tName := matches[1]
			status := strings.ToLower(matches[2])
			tc := JUnitTestCase{
				Name:      tName,
				Classname: classname,
				Time:      durStr,
			}
			if status == "fail" || status == "failed" || status == "failure" {
				tc.Failure = &JUnitFailure{
					Message:  "Test failed",
					Type:     "AssertionError",
					Contents: trimmed,
				}
			}
			testcases = append(testcases, tc)
		}
	}

	if len(testcases) == 0 {
		tc := JUnitTestCase{
			Name:      fallbackName,
			Classname: classname,
			Time:      durStr,
		}
		if !success {
			tc.Failure = &JUnitFailure{
				Message:  "Test process exited with non-zero status",
				Type:     "ProcessFailure",
				Contents: output,
			}
		}
		testcases = append(testcases, tc)
	}

	failures := 0
	for _, tc := range testcases {
		if tc.Failure != nil {
			failures++
		}
	}

	return JUnitTestSuite{
		Name:      fallbackName,
		Tests:     len(testcases),
		Failures:  failures,
		Errors:    0,
		Time:      durStr,
		TestCases: testcases,
	}
}

// WriteJUnitResults writes a JUnitTestSuite into the designated XML file.
func WriteJUnitResults(path string, suite JUnitTestSuite) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory for results file: %w", err)
	}

	suites := JUnitTestSuites{
		TestSuite: []JUnitTestSuite{suite},
	}

	data, err := xml.MarshalIndent(suites, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JUnit XML: %w", err)
	}

	content := append([]byte(xml.Header), data...)
	content = append(content, '\n')
	return os.WriteFile(path, content, 0644)
}

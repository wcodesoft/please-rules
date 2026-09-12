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

// JUnitTestSuites matches standard JUnit XML output format.
type JUnitTestSuites struct {
	XMLName   xml.Name         `xml:"testsuites"`
	TestSuite []JUnitTestSuite `xml:"testsuite"`
}

type JUnitTestSuite struct {
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Time      string          `xml:"time,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

type JUnitTestCase struct {
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
}

type JUnitFailure struct {
	Message  string `xml:"message,attr"`
	Type     string `xml:"type,attr"`
	Contents string `xml:",chardata"`
}

var (
	testLineRegex = regexp.MustCompile(`test\s+([^\s]+)\s+\.\.\.\s+(ok|FAILED|ignored)`)
	summaryRegex  = regexp.MustCompile(`test result:\s+(ok|FAILED)\.\s+(\d+)\s+passed;\s+(\d+)\s+failed;\s+(\d+)\s+ignored;`)
)

// DefaultResultsFile is the default JUnit XML output filename.
const DefaultResultsFile = "test.results"

// ParseTestOutput parses Rust's default test runner output into JUnit format.
func ParseTestOutput(pkgName string, output string, duration time.Duration) *JUnitTestSuites {
	suite := JUnitTestSuite{
		Name: pkgName,
		Time: fmt.Sprintf("%.3f", duration.Seconds()),
	}

	for _, line := range strings.Split(output, "\n") {
		if tc, ok := parseTestCaseLine(pkgName, line); ok {
			if tc.Failure != nil {
				suite.Failures++
			}
			suite.TestCases = append(suite.TestCases, tc)
			suite.Tests++
		}
	}

	if suite.Tests == 0 {
		suite.Tests = 1
		suite.TestCases = append(suite.TestCases, JUnitTestCase{
			Name:      "execution",
			Classname: pkgName,
			Time:      fmt.Sprintf("%.3f", duration.Seconds()),
		})
	}

	return &JUnitTestSuites{
		TestSuite: []JUnitTestSuite{suite},
	}
}

func parseTestCaseLine(pkgName, line string) (JUnitTestCase, bool) {
	line = strings.TrimSpace(line)
	m := testLineRegex.FindStringSubmatch(line)
	if len(m) < 3 {
		return JUnitTestCase{}, false
	}
	tName, status := m[1], m[2]
	tc := JUnitTestCase{
		Name:      tName,
		Classname: pkgName,
		Time:      "0.000",
	}
	if status == "FAILED" {
		tc.Failure = &JUnitFailure{
			Message: "Test failed",
			Type:    "Failure",
		}
	}
	return tc, true
}

// ensureDir creates the parent directory of filePath if it does not exist.
func ensureDir(filePath string) error {
	if outDir := filepath.Dir(filePath); outDir != "" && outDir != "." {
		return os.MkdirAll(outDir, 0755)
	}
	return nil
}

// writeJUnitResults formats and writes the test suites to resultsFile.
func writeJUnitResults(suites *JUnitTestSuites, resultsFile string) error {
	if resultsFile == "" {
		resultsFile = DefaultResultsFile
	}
	xmlData, err := xml.MarshalIndent(suites, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling junit xml: %w", err)
	}

	xmlWithHeader := append([]byte(xml.Header), xmlData...)
	if err := ensureDir(resultsFile); err != nil {
		return fmt.Errorf("creating results dir: %w", err)
	}

	if err := os.WriteFile(resultsFile, xmlWithHeader, 0644); err != nil {
		return fmt.Errorf("writing junit xml %s: %w", resultsFile, err)
	}
	return nil
}

// ConvertOutputToJUnit reads raw test output from a file/string and writes test.results JUnit XML.
func ConvertOutputToJUnit(pkgName string, rawOutputFile string, resultsFile string) error {
	data, err := os.ReadFile(rawOutputFile)
	if err != nil {
		return fmt.Errorf("failed to read raw test output: %w", err)
	}

	suites := ParseTestOutput(pkgName, string(data), 0)
	return writeJUnitResults(suites, resultsFile)
}

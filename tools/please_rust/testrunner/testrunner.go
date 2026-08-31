package testrunner

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
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

// ParseTestOutput parses Rust's default test runner output into JUnit format.
func ParseTestOutput(pkgName string, output string, duration time.Duration) *JUnitTestSuites {
	suite := JUnitTestSuite{
		Name: pkgName,
		Time: fmt.Sprintf("%.3f", duration.Seconds()),
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if m := testLineRegex.FindStringSubmatch(line); len(m) >= 3 {
			tName := m[1]
			status := m[2]
			tc := JUnitTestCase{
				Name:      tName,
				Classname: pkgName,
				Time:      "0.000",
			}
			if status == "FAILED" {
				suite.Failures++
				tc.Failure = &JUnitFailure{
					Message: "Test failed",
					Type:    "Failure",
				}
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

// ConvertOutputToJUnit reads raw test output from a file/string and writes test.results JUnit XML.
func ConvertOutputToJUnit(pkgName string, rawOutputFile string, resultsFile string) error {
	if resultsFile == "" {
		resultsFile = "test.results"
	}

	data, err := os.ReadFile(rawOutputFile)
	if err != nil {
		return fmt.Errorf("failed to read raw test output: %w", err)
	}

	suites := ParseTestOutput(pkgName, string(data), 0)
	xmlData, err := xml.MarshalIndent(suites, "", "  ")
	if err != nil {
		return err
	}

	xmlWithHeader := append([]byte(xml.Header), xmlData...)
	if outDir := filepath.Dir(resultsFile); outDir != "" && outDir != "." {
		os.MkdirAll(outDir, 0755)
	}

	return os.WriteFile(resultsFile, xmlWithHeader, 0644)
}

// Run executes a Rust test binary, streaming stdout/stderr while capturing output for JUnit results.
func Run(pkgName string, testBinary string, extraArgs []string, resultsFile string) error {
	if resultsFile == "" {
		resultsFile = "test.results"
	}

	cmd := exec.Command(testBinary, extraArgs...)

	var buf bytes.Buffer
	multiOut := io.MultiWriter(os.Stdout, &buf)
	multiErr := io.MultiWriter(os.Stderr, &buf)

	cmd.Stdout = multiOut
	cmd.Stderr = multiErr
	cmd.Env = os.Environ()

	startTime := time.Now()
	runErr := cmd.Run()
	duration := time.Since(startTime)

	testSuites := ParseTestOutput(pkgName, buf.String(), duration)
	xmlData, err := xml.MarshalIndent(testSuites, "", "  ")
	if err == nil {
		xmlWithHeader := append([]byte(xml.Header), xmlData...)
		if outDir := filepath.Dir(resultsFile); outDir != "" && outDir != "." {
			os.MkdirAll(outDir, 0755)
		}
		_ = os.WriteFile(resultsFile, xmlWithHeader, 0644)
	}

	return runErr
}

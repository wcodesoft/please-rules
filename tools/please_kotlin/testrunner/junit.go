package testrunner

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	Skipped   int             `xml:"skipped,attr"`
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
	Error     *JUnitFailure `xml:"error,omitempty"`
	Skipped   *JUnitSkipped `xml:"skipped,omitempty"`
}

// JUnitFailure represents a failed assertion (failure) or an unexpected exception (error).
type JUnitFailure struct {
	Message  string `xml:"message,attr"`
	Type     string `xml:"type,attr"`
	Contents string `xml:",chardata"`
}

// JUnitSkipped marks a test case that did not run.
type JUnitSkipped struct {
	Message string `xml:"message,attr,omitempty"`
}

// reportPattern matches the report files the JUnit Platform console launcher writes with
// --reports-dir: one per test engine (TEST-junit-jupiter.xml, TEST-junit-vintage.xml, ...).
const reportPattern = "TEST-*.xml"

// ReadJUnitReports reads every report the launcher wrote into dir and returns the suites
// that ran tests. The launcher writes one file per engine, so all of them are read, in
// name order; engines that ran nothing are dropped. The counts of each suite are
// recomputed from its test cases. A directory without reports gives no suites and no error.
func ReadJUnitReports(dir string) ([]JUnitTestSuite, error) {
	files, err := filepath.Glob(filepath.Join(dir, reportPattern))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)

	var suites []JUnitTestSuite
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		parsed, err := parseJUnitReport(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filepath.Base(file), err)
		}
		for _, suite := range parsed {
			if len(suite.TestCases) > 0 {
				suites = append(suites, recount(suite))
			}
		}
	}
	return suites, nil
}

// parseJUnitReport reads a report whose root is a <testsuite> (what the launcher writes)
// or a <testsuites>.
func parseJUnitReport(data []byte) ([]JUnitTestSuite, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("no XML element found: %w", err)
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "testsuite":
			var suite JUnitTestSuite
			if err := decoder.DecodeElement(&suite, &start); err != nil {
				return nil, err
			}
			return []JUnitTestSuite{suite}, nil
		case "testsuites":
			var suites JUnitTestSuites
			if err := decoder.DecodeElement(&suites, &start); err != nil {
				return nil, err
			}
			return suites.TestSuite, nil
		default:
			return nil, fmt.Errorf("unexpected root element <%s>", start.Name.Local)
		}
	}
}

// recount sets the counters of a suite from its test cases, which are what the results
// are made of, instead of trusting the attributes.
func recount(suite JUnitTestSuite) JUnitTestSuite {
	suite.Tests, suite.Skipped, suite.Failures, suite.Errors = len(suite.TestCases), 0, 0, 0
	for _, tc := range suite.TestCases {
		switch {
		case tc.Error != nil:
			suite.Errors++
		case tc.Failure != nil:
			suite.Failures++
		case tc.Skipped != nil:
			suite.Skipped++
		}
	}
	return suite
}

// Failed reports whether any test case of the suites failed or errored.
func Failed(suites []JUnitTestSuite) bool {
	for _, s := range suites {
		if s.Failures > 0 || s.Errors > 0 {
			return true
		}
	}
	return false
}

// ProcessSuite describes a test process for which there is no report (the JVM never got to
// write one, or no test ran): a single test case that passes if the process succeeded and
// fails, with the process output, if it did not.
func ProcessSuite(name, output string, duration time.Duration, success bool) JUnitTestSuite {
	tc := JUnitTestCase{Name: name, Classname: name, Time: fmt.Sprintf("%.3f", duration.Seconds())}
	if !success {
		tc.Error = &JUnitFailure{
			Message:  "Test process exited with non-zero status",
			Type:     "ProcessFailure",
			Contents: output,
		}
	}
	return recount(JUnitTestSuite{Name: name, Time: tc.Time, TestCases: []JUnitTestCase{tc}})
}

// WriteJUnitResults writes the suites into the designated XML file.
func WriteJUnitResults(path string, suites ...JUnitTestSuite) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory for results file: %w", err)
	}

	data, err := xml.MarshalIndent(JUnitTestSuites{TestSuite: suites}, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JUnit XML: %w", err)
	}

	content := append([]byte(xml.Header), data...)
	content = append(content, '\n')
	return os.WriteFile(path, content, 0644)
}

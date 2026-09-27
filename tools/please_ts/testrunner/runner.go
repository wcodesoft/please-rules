package testrunner

import (
	"fmt"
	"os"
	"path/filepath"
)

// RunOptions configures test execution.
type RunOptions struct {
	Deno          string
	Runner        string // "deno" | "vitest" | "browser"
	Srcs          []string
	Deps          []string
	ModuleName    string
	ResultsFile   string
	Coverage      bool
	CoverageFile  string
	Browser       string
	BrowserBinary string
	VitestDir     string
	ExtraArgs     []string
}

// Run executes the tests and generates JUnit XML and optional coverage reports.
func Run(opts RunOptions) error {
	return opts.Run()
}

// Run dispatches test execution to the appropriate runner based on options.
func (opts RunOptions) Run() error {
	if len(opts.Srcs) == 0 {
		return fmt.Errorf("no test source files specified")
	}

	resultsFile := opts.ResultsFile
	if resultsFile == "" {
		resultsFile = "test.results"
	}
	if err := os.MkdirAll(filepath.Dir(resultsFile), 0755); err != nil {
		return err
	}

	if opts.Browser != "" && opts.Runner != "vitest" {
		return opts.runBrowserTest(resultsFile)
	}

	if opts.Runner == "vitest" {
		return opts.runVitest(resultsFile)
	}

	return opts.runDeno(resultsFile)
}

func writeFallbackJUnit(resultsFile string, srcs []string, testErr error) error {
	msg := "test execution finished"
	failureXML := ""
	if testErr != nil {
		msg = testErr.Error()
		failureXML = fmt.Sprintf("\n    <failure message=%q>%s</failure>", msg, msg)
	}

	xmlContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<testsuites name="TypeScript Tests">
  <testsuite name="ts_test" tests="%d" failures="%d" errors="0">
    <testcase name="execution" classname="ts_test">%s
    </testcase>
  </testsuite>
</testsuites>
`, len(srcs), boolToInt(testErr != nil), failureXML)

	return os.WriteFile(resultsFile, []byte(xmlContent), 0644)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

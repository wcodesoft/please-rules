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
	LcovFile      string // raw lcov export (functions and branches), written empty without coverage
	Browser       string
	BrowserBinary string
	VitestDir     string
	EsbuildBinary string   // pinned esbuild, to bundle npm packages for browser tests
	Viewport      string   // browser tests: "WIDTHxHEIGHT" emulated viewport, empty for the browser default
	Snapshots     []string // browser tests: golden JSON files for expectComputedStyles
	SnapshotsOut  string   // browser tests: directory receiving every recorded snapshot and screenshot
	Screenshots   []string // browser tests: golden PNG files for expectScreenshot
	ScreenshotsOn bool     // browser tests: enable expectScreenshot and the deterministic rendering flags
	PixelChannel  int      // screenshots: per-channel difference (0-255) tolerated for a pixel to match
	MaxDiffRatio  float64  // screenshots: share (0-1) of pixels allowed to differ
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

	// The raw lcov export is a declared test output: create it up front so it exists
	// (empty) when coverage is off or the tool produces nothing.
	if err := opts.prepareLcovFile(); err != nil {
		return err
	}

	// The snapshot recordings are a declared test output: create the directory up front so it
	// exists (empty) when no test records anything.
	if opts.SnapshotsOut != "" {
		if err := os.MkdirAll(opts.SnapshotsOut, 0755); err != nil {
			return err
		}
	}

	if opts.Browser != "" && opts.Runner != "vitest" {
		return opts.runBrowserTest(resultsFile)
	}

	if opts.Runner == "vitest" {
		return opts.runVitest(resultsFile)
	}

	return opts.runDeno(resultsFile)
}

// prepareLcovFile resets the raw lcov export to an empty file.
func (opts RunOptions) prepareLcovFile() error {
	if opts.LcovFile == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(opts.LcovFile), 0755); err != nil {
		return err
	}
	return os.WriteFile(opts.LcovFile, nil, 0644)
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

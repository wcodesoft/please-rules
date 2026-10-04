// Command lcovcheck asserts facts about coverage reports, for CI smoke tests.
//
// Raw lcov report (see docs/coverage-lcov.md), parsed with the shared lcov package:
//
//	lcovcheck -lcov plz-out/bin/pkg/x_test.lcov -path pkg/lib.go \
//	    -line 4=partial -line 5=uncovered -function unused=missed -function classify=2
//
// A line is asserted to be covered, uncovered, partial or not-executable. A function is
// matched by a substring of its name (names are mangled differently by each tool) that
// must identify exactly one function, and asserted to be hit, missed, or to have been
// called exactly N times.
//
// Please's merged report:
//
//	lcovcheck -coverage-json plz-out/log/coverage.json -covered test/ts/lib/calculator.ts
//
// asserts that each file is in the report with a coverage above zero.
//
// Every failed assertion is listed; the exit status is 1 if there was any.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"tools/common/lcov"
)

type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "lcovcheck:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("lcovcheck", flag.ContinueOnError)
	lcovFile := fs.String("lcov", "", "lcov report to check")
	path := fs.String("path", "", "source file in the lcov report that the assertions are about")
	var lines, funcs, covered multiFlag
	fs.Var(&lines, "line", "LINE=covered|uncovered|partial|not-executable (repeatable)")
	fs.Var(&funcs, "function", "NAMESUBSTRING=hit|missed|N (repeatable)")
	coverageJSON := fs.String("coverage-json", "", "Please's coverage.json to check")
	fs.Var(&covered, "covered", "a file that must have coverage above zero in -coverage-json (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	var failures []string
	switch {
	case *lcovFile != "" && *coverageJSON == "":
		if *path == "" {
			return fmt.Errorf("-path is required with -lcov")
		}
		if len(lines)+len(funcs) == 0 {
			return fmt.Errorf("give at least one -line or -function assertion")
		}
		f, err := os.Open(*lcovFile)
		if err != nil {
			return err
		}
		defer f.Close()
		report, err := lcov.Parse(f)
		if err != nil {
			return fmt.Errorf("%s: %w", *lcovFile, err)
		}
		failures = checkLcov(report, *path, lines, funcs)
	case *coverageJSON != "" && *lcovFile == "":
		if len(covered) == 0 {
			return fmt.Errorf("give at least one -covered file")
		}
		data, err := os.ReadFile(*coverageJSON)
		if err != nil {
			return err
		}
		failures, err = checkCoverageJSON(data, covered)
		if err != nil {
			return fmt.Errorf("%s: %w", *coverageJSON, err)
		}
	default:
		return fmt.Errorf("give either -lcov (with -path) or -coverage-json")
	}

	if len(failures) > 0 {
		return fmt.Errorf("%d assertion(s) failed:\n  %s", len(failures), strings.Join(failures, "\n  "))
	}
	fmt.Fprintln(stdout, "all assertions hold")
	return nil
}

func checkLcov(report *lcov.Report, path string, lines, funcs []string) []string {
	file := report.File(path)
	if file == nil {
		var have []string
		for _, f := range report.Files {
			have = append(have, f.Path)
		}
		return []string{fmt.Sprintf("%s is not in the report (it has: %s)", path, strings.Join(have, ", "))}
	}

	var failures []string
	for _, spec := range lines {
		num, want, ok := strings.Cut(spec, "=")
		line, err := strconv.Atoi(num)
		if !ok || err != nil || !validStatus(want) {
			failures = append(failures, fmt.Sprintf("bad -line %q: want LINE=covered|uncovered|partial|not-executable", spec))
			continue
		}
		if got := file.LineStatus(line).String(); got != want {
			failures = append(failures, fmt.Sprintf("%s line %d is %s, want %s", path, line, got, want))
		}
	}
	for _, spec := range funcs {
		substr, want, ok := strings.Cut(spec, "=")
		if !ok || substr == "" {
			failures = append(failures, fmt.Sprintf("bad -function %q: want NAMESUBSTRING=hit|missed|N", spec))
			continue
		}
		var matches []lcov.Function
		for _, fn := range file.Functions {
			if strings.Contains(fn.Name, substr) {
				matches = append(matches, fn)
			}
		}
		if len(matches) != 1 {
			failures = append(failures, fmt.Sprintf("%s: %q matches %d functions, want exactly 1", path, substr, len(matches)))
			continue
		}
		if msg := checkHits(matches[0], want); msg != "" {
			failures = append(failures, fmt.Sprintf("%s function %s: %s", path, matches[0].Name, msg))
		}
	}
	return failures
}

func validStatus(s string) bool {
	switch s {
	case "covered", "uncovered", "partial", "not-executable":
		return true
	}
	return false
}

func checkHits(fn lcov.Function, want string) string {
	switch want {
	case "hit":
		if fn.Hits == 0 {
			return "was never called, want it called"
		}
	case "missed":
		if fn.Hits != 0 {
			return fmt.Sprintf("was called %d times, want it never called", fn.Hits)
		}
	default:
		n, err := strconv.Atoi(want)
		if err != nil || n < 0 {
			return fmt.Sprintf("bad expectation %q: want hit, missed or a count", want)
		}
		if fn.Hits != n {
			return fmt.Sprintf("was called %d times, want %d", fn.Hits, n)
		}
	}
	return ""
}

func checkCoverageJSON(data []byte, files []string) ([]string, error) {
	var report struct {
		Stats struct {
			CoverageByFile map[string]float64 `json:"coverage_by_file"`
		} `json:"stats"`
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return nil, err
	}
	var failures []string
	for _, file := range files {
		pct, ok := report.Stats.CoverageByFile[file]
		switch {
		case !ok:
			failures = append(failures, fmt.Sprintf("%s is not in the coverage report", file))
		case pct <= 0:
			failures = append(failures, fmt.Sprintf("%s has no coverage (%.1f%%)", file, pct))
		}
	}
	return failures, nil
}

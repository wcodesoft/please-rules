// Command mutationcheck asserts facts about .mutation.json reports, for CI smoke tests.
//
// Example usage:
//
//	mutationcheck -report plz-out/bin/pkg/x_mutate_test.mutation.json \
//	    -lang go -min-score 70 -path pkg/lib.go \
//	    -line 12=killed -line 15=survived -function Clamp=survived
//
// Every failed assertion is listed; the exit status is 1 if there was any.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"tools/common/mutation"
)

type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

type checkFlags struct {
	reportFile string
	path       string
	lang       string
	minScore   float64
	minTotal   int
	minKilled  int
	lines      multiFlag
	funcs      multiFlag
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "mutationcheck:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	cfg, err := parseFlags(args)
	if err != nil {
		return err
	}

	rep, err := loadReport(cfg.reportFile)
	if err != nil {
		return err
	}

	failures := checkSummaryAssertions(rep, cfg)
	if len(cfg.lines)+len(cfg.funcs) > 0 {
		failures = append(failures, checkFileMutants(rep, cfg.path, cfg.lines, cfg.funcs)...)
	}

	if len(failures) > 0 {
		return fmt.Errorf("%d assertion(s) failed:\n  %s", len(failures), strings.Join(failures, "\n  "))
	}
	fmt.Fprintln(stdout, "all assertions hold")
	return nil
}

func parseFlags(args []string) (checkFlags, error) {
	var cfg checkFlags
	fs := flag.NewFlagSet("mutationcheck", flag.ContinueOnError)
	fs.StringVar(&cfg.reportFile, "report", "", "path to .mutation.json report to check")
	fs.StringVar(&cfg.path, "path", "", "source file in the mutation report that -line and -function assertions are about")
	fs.StringVar(&cfg.lang, "lang", "", "expected language in the report")
	fs.Float64Var(&cfg.minScore, "min-score", -1, "assert score >= min-score")
	fs.IntVar(&cfg.minTotal, "min-total", -1, "assert total evaluated mutants >= min-total")
	fs.IntVar(&cfg.minKilled, "min-killed", -1, "assert killed mutants >= min-killed")
	fs.Var(&cfg.lines, "line", "LINE=killed|survived|timeout|unviable|skipped (repeatable, requires -path)")
	fs.Var(&cfg.funcs, "function", "NAMESUBSTRING=killed|survived (repeatable, requires -path)")

	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if cfg.reportFile == "" {
		return cfg, fmt.Errorf("-report is required")
	}
	if len(cfg.lines)+len(cfg.funcs) > 0 && cfg.path == "" {
		return cfg, fmt.Errorf("-path is required with -line or -function")
	}
	return cfg, nil
}

func loadReport(reportFile string) (*mutation.Report, error) {
	f, err := os.Open(reportFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rep, err := mutation.ParseJSON(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", reportFile, err)
	}
	return rep, nil
}

func checkSummaryAssertions(rep *mutation.Report, cfg checkFlags) []string {
	var failures []string
	if cfg.lang != "" && rep.Language != cfg.lang {
		failures = append(failures, fmt.Sprintf("language is %q, want %q", rep.Language, cfg.lang))
	}
	if cfg.minScore >= 0 && rep.Score+1e-9 < cfg.minScore {
		failures = append(failures, fmt.Sprintf("score is %.2f%%, want >= %.2f%%", rep.Score, cfg.minScore))
	}
	if cfg.minTotal >= 0 && rep.Total < cfg.minTotal {
		failures = append(failures, fmt.Sprintf("total is %d, want >= %d", rep.Total, cfg.minTotal))
	}
	if cfg.minKilled >= 0 && rep.Killed < cfg.minKilled {
		failures = append(failures, fmt.Sprintf("killed is %d, want >= %d", rep.Killed, cfg.minKilled))
	}
	return failures
}

func checkFileMutants(rep *mutation.Report, path string, lines, funcs []string) []string {
	mutants := rep.FileMutants(path)
	if len(mutants) == 0 {
		return []string{fmt.Sprintf("%s has no mutants in the report (files present: %s)",
			path, strings.Join(reportFiles(rep), ", "))}
	}

	var failures []string
	for _, spec := range lines {
		if msg := checkLineSpec(mutants, path, spec); msg != "" {
			failures = append(failures, msg)
		}
	}
	for _, spec := range funcs {
		if msg := checkFuncSpec(mutants, path, spec); msg != "" {
			failures = append(failures, msg)
		}
	}
	return failures
}

func reportFiles(rep *mutation.Report) []string {
	seen := make(map[string]bool)
	var have []string
	for _, m := range rep.Mutants {
		if !seen[m.File] {
			seen[m.File] = true
			have = append(have, m.File)
		}
	}
	return have
}

func checkLineSpec(mutants []mutation.Mutant, path, spec string) string {
	num, wantStr, ok := strings.Cut(spec, "=")
	line, err := strconv.Atoi(num)
	want := mutation.Status(wantStr)
	if !ok || err != nil || !mutation.ValidStatus(want) {
		return fmt.Sprintf("bad -line %q: want LINE=killed|survived|timeout|unviable|skipped", spec)
	}
	onLine, matched := filterMutants(mutants, want, func(m mutation.Mutant) bool {
		return m.Line == line
	})
	if len(onLine) == 0 {
		return fmt.Sprintf("%s line %d has no mutants", path, line)
	}
	if !matched {
		return fmt.Sprintf("%s line %d has mutants [%s], none with status %s",
			path, line, formatMutantStatuses(onLine), want)
	}
	return ""
}

func checkFuncSpec(mutants []mutation.Mutant, path, spec string) string {
	substr, wantStr, ok := strings.Cut(spec, "=")
	want := mutation.Status(wantStr)
	if !ok || substr == "" || !mutation.ValidStatus(want) {
		return fmt.Sprintf("bad -function %q: want NAMESUBSTRING=killed|survived|timeout|unviable|skipped", spec)
	}
	inFunc, matched := filterMutants(mutants, want, func(m mutation.Mutant) bool {
		return strings.Contains(m.Function, substr)
	})
	if len(inFunc) == 0 {
		return fmt.Sprintf("%s has no mutants in function matching %q", path, substr)
	}
	if !matched {
		return fmt.Sprintf("%s function %q has no mutant with status %s", path, substr, want)
	}
	return ""
}

func filterMutants(mutants []mutation.Mutant, want mutation.Status, pred func(mutation.Mutant) bool) ([]mutation.Mutant, bool) {
	var matched []mutation.Mutant
	hasStatus := false
	for _, m := range mutants {
		if pred(m) {
			matched = append(matched, m)
			if m.Status == want {
				hasStatus = true
			}
		}
	}
	return matched, hasStatus
}

func formatMutantStatuses(mutants []mutation.Mutant) string {
	statuses := make([]string, 0, len(mutants))
	for _, m := range mutants {
		statuses = append(statuses, fmt.Sprintf("%s(%s->%s)", m.Status, m.Original, m.Replacement))
	}
	return strings.Join(statuses, ", ")
}

// Command mutate discovers syntax-directed source mutations using pure-Go
// Tree-sitter grammars (Go, Rust, TypeScript, Kotlin, Swift, Python), executes
// a hermetic test command against each mutant, writes the shared
// <target>.mutation.json and JUnit XML test.results reports, and enforces a
// minimum mutation score threshold.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

type multiFlag []string

func (m *multiFlag) String() string     { return strings.Join(*m, ",") }
func (m *multiFlag) Set(v string) error { *m = append(*m, v); return nil }

func main() {
	if err := runCLI(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "mutate:", err)
		os.Exit(1)
	}
}

func runCLI(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("mutate", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var srcs, mutators multiFlag
	fs.Var(&srcs, "srcs", "source files or directories to mutate (comma-separated or repeatable)")
	fs.Var(&mutators, "mutators", "mutator categories to enable (comma-separated or repeatable; default all)")
	lang := fs.String("lang", "", "explicit language override (go|rust|ts|tsx|js|kotlin|swift|python)")
	target := fs.String("target", "", "Please target label recorded in the report")
	testCmd := fs.String("test-cmd", "", "shell command to execute the test suite against each mutant")
	workDir := fs.String("work-dir", "", "working directory for source paths and test execution (default cwd)")
	stripPrefix := fs.String("strip-prefix", "", "optional path prefix to strip from reported file paths")
	outReport := fs.String("out", "", "output path for the .mutation.json report")
	resultsFile := fs.String("results-file", "", "output path for the JUnit XML test.results file")
	minScore := fs.Float64("min-score", 0, "minimum required mutation score percentage (0-100)")
	timeoutMult := fs.Float64("timeout-multiplier", 5.0, "multiplier applied to baseline test duration for per-mutant timeout")
	mutantTimeout := fs.Duration("mutant-timeout", 0, "explicit per-mutant timeout (overrides -timeout-multiplier when > 0)")
	listOnly := fs.Bool("list", false, "list discovered mutants without running tests")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(srcs) == 0 && fs.NArg() > 0 {
		srcs = append(srcs, fs.Args()...)
	}

	opts := Options{
		Srcs:              srcs,
		Lang:              *lang,
		Target:            *target,
		TestCmd:           *testCmd,
		WorkDir:           *workDir,
		StripPrefix:       *stripPrefix,
		OutReport:         *outReport,
		ResultsFile:       *resultsFile,
		MinScore:          *minScore,
		TimeoutMultiplier: *timeoutMult,
		MutantTimeout:     *mutantTimeout,
		Mutators:          mutators,
		ListOnly:          *listOnly,
		Stdout:            stdout,
		Stderr:            stderr,
	}

	_, err := Run(context.Background(), opts)
	return err
}

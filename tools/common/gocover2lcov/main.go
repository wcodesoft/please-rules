// Command gocover2lcov converts a Go cover profile to lcov.
//
// Usage:
//
//	gocover2lcov -profile <file> [-src-root <dir>] [-strip-prefix <prefix>] [-out <file>]
//
// Under Please, each go_test keeps its profile at
// plz-out/bin/<package>/.test_coverage_<name> after `plz cover`. Source files are
// read from -src-root (to find functions) and the profile's file names lose
// -strip-prefix, typically the module path when the import path is not repository
// relative.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tools/common/lcov"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "gocover2lcov:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("gocover2lcov", flag.ContinueOnError)
	profile := fs.String("profile", "", "Go cover profile to convert (required)")
	srcRoot := fs.String("src-root", ".", "directory the profile's file names are relative to, for reading sources")
	prefix := fs.String("strip-prefix", "", "prefix to remove from the profile's file names, e.g. the module path")
	out := fs.String("out", "", "output file (default: stdout)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *profile == "" {
		return fmt.Errorf("-profile is required")
	}

	in, err := os.Open(*profile)
	if err != nil {
		return err
	}
	defer in.Close()

	mapPath := func(p string) string { return strings.TrimPrefix(strings.TrimPrefix(p, *prefix), "/") }
	report, err := lcov.FromGoProfile(in, func(p string) ([]byte, error) {
		return os.ReadFile(filepath.Join(*srcRoot, mapPath(p)))
	}, mapPath)
	if err != nil {
		return err
	}

	w := stdout
	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}
	return report.Write(w)
}

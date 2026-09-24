package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"tools/please_ts/binary"
	"tools/please_ts/bundle"
	"tools/please_ts/compile"
	"tools/please_ts/testrunner"
	"tools/please_ts/unpack"
)

func handleCompile(args []string) error {
	cmd := flag.NewFlagSet("compile", flag.ContinueOnError)
	cmd.Usage = func() {
		fmt.Fprintf(cmd.Output(), "Usage:\n  please_ts compile [options] [-- sources...]\n\nOptions:\n")
		cmd.PrintDefaults()
	}
	out := cmd.String("out", "", "Output path/directory for the compiled library")
	srcsFlag := cmd.String("srcs", "", "Comma-separated sources")
	depsFlag := cmd.String("deps", "", "Comma-separated dependency paths")
	deno := cmd.String("deno", "deno", "Path to deno binary")
	moduleName := cmd.String("module-name", "", "Module name or package specifier alias")
	flags := cmd.String("flags", "", "Additional deno check flags")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	srcs := compile.ExpandCommaSeparated([]string{*srcsFlag})
	srcs = append(srcs, cmd.Args()...)

	deps := compile.ExpandCommaSeparated([]string{*depsFlag})

	var extraFlags []string
	if *flags != "" {
		extraFlags = strings.Fields(*flags)
	}

	opts := compile.Options{
		Deno:       *deno,
		Out:        *out,
		Srcs:       srcs,
		Deps:       deps,
		ModuleName: *moduleName,
		Flags:      extraFlags,
	}
	return compile.Run(opts)
}

func handleBundle(args []string) error {
	cmd := flag.NewFlagSet("bundle", flag.ContinueOnError)
	cmd.Usage = func() {
		fmt.Fprintf(cmd.Output(), "Usage:\n  please_ts bundle [options] [-- sources...]\n\nOptions:\n")
		cmd.PrintDefaults()
	}
	out := cmd.String("out", "", "Output bundle path")
	main := cmd.String("main", "", "Main entry point file")
	srcsFlag := cmd.String("srcs", "", "Comma-separated sources")
	depsFlag := cmd.String("deps", "", "Comma-separated dependency paths")
	deno := cmd.String("deno", "deno", "Path to deno binary")
	bundlerTool := cmd.String("bundler-tool", "", "External bundler executable (e.g. esbuild)")
	moduleName := cmd.String("module-name", "", "Module name alias")
	format := cmd.String("format", "esm", "Bundle format (esm or iife)")
	minify := cmd.Bool("minify", false, "Minify bundled output")
	sourcemap := cmd.Bool("sourcemap", false, "Generate source map")
	flags := cmd.String("flags", "", "Additional bundler flags")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	srcs := compile.ExpandCommaSeparated([]string{*srcsFlag})
	srcs = append(srcs, cmd.Args()...)
	deps := compile.ExpandCommaSeparated([]string{*depsFlag})

	var extraFlags []string
	if *flags != "" {
		extraFlags = strings.Fields(*flags)
	}

	opts := bundle.Options{
		Deno:        *deno,
		BundlerTool: *bundlerTool,
		Out:         *out,
		Main:        *main,
		Srcs:        srcs,
		Deps:        deps,
		ModuleName:  *moduleName,
		Format:      *format,
		Minify:      *minify,
		Sourcemap:   *sourcemap,
		Flags:       extraFlags,
	}
	return bundle.Run(opts)
}

func handleBinary(args []string) error {
	cmd := flag.NewFlagSet("binary", flag.ContinueOnError)
	cmd.Usage = func() {
		fmt.Fprintf(cmd.Output(), "Usage:\n  please_ts binary [options] [-- sources...]\n\nOptions:\n")
		cmd.PrintDefaults()
	}
	out := cmd.String("out", "", "Output binary executable path")
	main := cmd.String("main", "", "Main entry point file")
	srcsFlag := cmd.String("srcs", "", "Comma-separated sources")
	depsFlag := cmd.String("deps", "", "Comma-separated dependency paths")
	deno := cmd.String("deno", "deno", "Path to deno binary")
	moduleName := cmd.String("module-name", "", "Module name alias")
	flags := cmd.String("flags", "", "Additional deno compile flags")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	srcs := compile.ExpandCommaSeparated([]string{*srcsFlag})
	srcs = append(srcs, cmd.Args()...)
	deps := compile.ExpandCommaSeparated([]string{*depsFlag})

	var extraFlags []string
	if *flags != "" {
		extraFlags = strings.Fields(*flags)
	}

	opts := binary.Options{
		Deno:       *deno,
		Out:        *out,
		Main:       *main,
		Srcs:       srcs,
		Deps:       deps,
		ModuleName: *moduleName,
		Flags:      extraFlags,
	}
	return binary.Run(opts)
}

func handleTestRunner(args []string) error {
	cmd := flag.NewFlagSet("testrunner", flag.ContinueOnError)
	cmd.Usage = func() {
		fmt.Fprintf(cmd.Output(), "Usage:\n  please_ts testrunner [options] [-- sources...]\n\nOptions:\n")
		cmd.PrintDefaults()
	}
	deno := cmd.String("deno", "deno", "Path to deno binary")
	runner := cmd.String("runner", "deno", "Test runner to use (deno or vitest)")
	srcsFlag := cmd.String("srcs", "", "Comma-separated sources")
	depsFlag := cmd.String("deps", "", "Comma-separated dependency paths")
	moduleName := cmd.String("module-name", "", "Module name alias")
	resultsFile := cmd.String("results-file", "test.results", "Path to write JUnit XML test results")
	coverageActive := cmd.Bool("coverage", false, "Whether coverage collection is enabled")
	coverageFile := cmd.String("coverage-file", "", "Path to write coverage output file")
	browser := cmd.String("browser", "", "Browser engine for browser tests (chromium, firefox, webkit)")
	browserBinary := cmd.String("browser-binary", "", "Path to hermetic browser executable")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	srcs := compile.ExpandCommaSeparated([]string{*srcsFlag})
	srcs = append(srcs, cmd.Args()...)
	deps := compile.ExpandCommaSeparated([]string{*depsFlag})

	opts := testrunner.RunOptions{
		Deno:          *deno,
		Runner:        *runner,
		Srcs:          srcs,
		Deps:          deps,
		ModuleName:    *moduleName,
		ResultsFile:   *resultsFile,
		Coverage:      *coverageActive,
		CoverageFile:  *coverageFile,
		Browser:       *browser,
		BrowserBinary: *browserBinary,
	}
	return testrunner.Run(opts)
}

func handleUnpack(args []string) error {
	cmd := flag.NewFlagSet("unpack", flag.ContinueOnError)
	tarball := cmd.String("tarball", "", "Path to npm tarball")
	archive := cmd.String("archive", "", "Path to tarball or zip archive")
	out := cmd.String("out", "", "Destination output directory")
	name := cmd.String("name", "", "Module/package name")
	binary := cmd.String("binary", "", "Binary executable to extract in toolchain mode")
	symlink := cmd.String("symlink", "", "Optional symlink name for extracted binary")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	archivePath := *archive
	if archivePath == "" {
		archivePath = *tarball
	}

	opts := unpack.Options{
		Archive: archivePath,
		Tarball: *tarball,
		Out:     *out,
		Name:    *name,
		Binary:  *binary,
		Symlink: *symlink,
	}
	return unpack.Run(opts)
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, "please_ts - Orchestrator tool for Please TypeScript/Deno rules\n\n")
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  please_ts compile [options] [-- sources...]\n")
	fmt.Fprintf(w, "  please_ts bundle [options]\n")
	fmt.Fprintf(w, "  please_ts binary [options]\n")
	fmt.Fprintf(w, "  please_ts testrunner [options] [-- sources...]\n")
	fmt.Fprintf(w, "  please_ts unpack [options]\n")
}

func runCommand(args []string) error {
	if len(args) < 1 {
		printUsage(os.Stderr)
		return fmt.Errorf("no subcommand specified")
	}

	switch args[0] {
	case "compile":
		return handleCompile(args[1:])
	case "bundle":
		return handleBundle(args[1:])
	case "binary":
		return handleBinary(args[1:])
	case "testrunner":
		return handleTestRunner(args[1:])
	case "unpack":
		return handleUnpack(args[1:])
	case "help", "--help", "-h":
		printUsage(os.Stdout)
		return nil
	default:
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"tools/please_swift/binary"
	"tools/please_swift/compile"
	"tools/please_swift/testrunner"
)

func handleCompile(args []string) error {
	cmd := flag.NewFlagSet("compile", flag.ContinueOnError)
	swiftc := cmd.String("swiftc", "swiftc", "Path to swiftc binary")
	moduleName := cmd.String("module-name", "", "Swift module name")
	srcsFlag := cmd.String("srcs", "", "Comma-separated source files")
	depsFlag := cmd.String("deps", "", "Comma-separated dependency paths")
	out := cmd.String("out", "", "Output directory for module and library")
	outDir := cmd.String("out-dir", "", "Output directory for module and library")
	outLib := cmd.String("out-lib", "", "Output static library .a path")
	outModule := cmd.String("out-module", "", "Output .swiftmodule path")
	swiftVersion := cmd.String("swift-version", "", "Swift language version")
	staticStdlib := cmd.Bool("static-stdlib", false, "Statically link Swift runtime")
	coverage := cmd.Bool("coverage", false, "Enable code coverage instrumentation")
	flags := cmd.String("flags", "", "Additional swiftc flags")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	finalOutDir := *outDir
	if finalOutDir == "" {
		finalOutDir = *out
	}

	srcs := compile.ExpandCommaSeparated([]string{*srcsFlag})
	srcs = append(srcs, cmd.Args()...)
	deps := compile.ExpandCommaSeparated([]string{*depsFlag})

	var extraFlags []string
	if *flags != "" {
		extraFlags = strings.Fields(*flags)
	}

	opts := compile.Options{
		Swiftc:       *swiftc,
		ModuleName:   *moduleName,
		Srcs:         srcs,
		Deps:         deps,
		OutDir:       finalOutDir,
		OutLib:       *outLib,
		OutModule:    *outModule,
		SwiftVersion: *swiftVersion,
		Flags:        extraFlags,
		StaticStdlib: *staticStdlib,
		Coverage:     *coverage,
	}
	return compile.Run(opts)
}

func handleBinary(args []string) error {
	cmd := flag.NewFlagSet("binary", flag.ContinueOnError)
	swiftc := cmd.String("swiftc", "swiftc", "Path to swiftc binary")
	out := cmd.String("out", "", "Output executable binary path")
	mainFile := cmd.String("main", "", "Main entrypoint Swift file")
	srcsFlag := cmd.String("srcs", "", "Comma-separated source files")
	depsFlag := cmd.String("deps", "", "Comma-separated dependency paths")
	moduleName := cmd.String("module-name", "", "Module name")
	staticStdlib := cmd.Bool("static-stdlib", false, "Statically link Swift runtime")
	flags := cmd.String("flags", "", "Additional swiftc flags")

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
		Swiftc:       *swiftc,
		Out:          *out,
		Main:         *mainFile,
		Srcs:         srcs,
		Deps:         deps,
		ModuleName:   *moduleName,
		Flags:        extraFlags,
		StaticStdlib: *staticStdlib,
	}
	return binary.Run(opts)
}

func handleTestRunner(args []string) error {
	cmd := flag.NewFlagSet("testrunner", flag.ContinueOnError)
	swiftc := cmd.String("swiftc", "swiftc", "Path to swiftc binary")
	llvmProfdata := cmd.String("llvm-profdata", "llvm-profdata", "Path to llvm-profdata")
	llvmCov := cmd.String("llvm-cov", "llvm-cov", "Path to llvm-cov")
	framework := cmd.String("framework", "swift-testing", "Testing framework (swift-testing or xctest)")
	srcsFlag := cmd.String("srcs", "", "Comma-separated test source files")
	depsFlag := cmd.String("deps", "", "Comma-separated dependency paths")
	out := cmd.String("out", "", "Output test executable path")
	binary := cmd.String("binary", "", "Path to precompiled test executable")
	compileOnly := cmd.Bool("compile-only", false, "Only compile without running tests")
	resultsFile := cmd.String("results-file", "test.results", "Path to write JUnit XML test results")
	coverageActive := cmd.Bool("coverage", false, "Whether coverage collection is enabled")
	coverageFile := cmd.String("coverage-file", "", "Path to write coverage output file")
	testArgsFlag := cmd.String("test-args", "", "Arguments to pass to test runner executable")
	flags := cmd.String("flags", "", "Additional swiftc flags")

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

	var testArgs []string
	if *testArgsFlag != "" {
		testArgs = strings.Fields(*testArgsFlag)
	}

	opts := testrunner.RunOptions{
		Swiftc:       *swiftc,
		LlvmProfdata: *llvmProfdata,
		LlvmCov:      *llvmCov,
		Framework:    *framework,
		Srcs:         srcs,
		Deps:         deps,
		Out:          *out,
		Binary:       *binary,
		CompileOnly:  *compileOnly,
		ResultsFile:  *resultsFile,
		Coverage:     *coverageActive,
		CoverageFile: *coverageFile,
		TestArgs:     testArgs,
		Flags:        extraFlags,
	}
	return testrunner.Run(opts)
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, "please_swift - Orchestrator tool for Please Swift rules\n\n")
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  please_swift compile [options] [-- sources...]\n")
	fmt.Fprintf(w, "  please_swift binary [options] [-- sources...]\n")
	fmt.Fprintf(w, "  please_swift testrunner [options] [-- sources...]\n")
}

func runCommand(args []string) error {
	if len(args) < 1 {
		printUsage(os.Stderr)
		return fmt.Errorf("no subcommand specified")
	}

	switch args[0] {
	case "compile":
		return handleCompile(args[1:])
	case "binary":
		return handleBinary(args[1:])
	case "testrunner":
		return handleTestRunner(args[1:])
	case "help", "--help", "-h":
		printUsage(os.Stdout)
		return nil
	default:
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

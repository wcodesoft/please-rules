package main

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"tools/please_rust/compile"
	"tools/please_rust/compilec"
	"tools/please_rust/download"
	"tools/please_rust/testrunner"
)

func handleCompile(args []string) error {
	cmd := flag.NewFlagSet("compile", flag.ContinueOnError)
	out := cmd.String("out", "", "Output artifact path")
	crateName := cmd.String("crate-name", "", "Name of the crate")
	crateType := cmd.String("crate-type", "", "Type of crate: rlib, bin, proc-macro, test")
	edition := cmd.String("edition", "2021", "Rust edition (e.g. 2021, 2024)")
	mainSrc := cmd.String("main-src", "", "Main source entrypoint file")
	version := cmd.String("version", "", "Crate version")
	flags := cmd.String("flags", "", "Additional rustc flags")
	rustc := cmd.String("rustc", "", "Path to rustc executable")
	meta := cmd.String("meta", "", "Path to crate_meta.json (from download rule)")
	nativeLib := cmd.String("native-lib", "", "Path to a .a static lib to link")
	featuresFlag := cmd.String("features", "", "Comma-separated feature flags")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	var features []string
	if *featuresFlag != "" {
		features = strings.Split(*featuresFlag, ",")
	}

	opts := compile.Options{
		Out:       *out,
		CrateName: *crateName,
		CrateType: *crateType,
		Edition:   *edition,
		MainSrc:   *mainSrc,
		Version:   *version,
		Flags:     *flags,
		Rustc:     *rustc,
		Inputs:    cmd.Args(),
		Meta:      *meta,
		NativeLib: *nativeLib,
		Features:  features,
	}
	return compile.Run(opts)
}

func handleDownload(args []string) error {
	cmd := flag.NewFlagSet("download", flag.ContinueOnError)
	crate := cmd.String("crate", "", "Crate name")
	version := cmd.String("version", "", "Crate version")
	sha256 := cmd.String("sha256", "", "Expected SHA-256 hex digest of the tarball")
	out := cmd.String("out", "", "Output path for the repacked tar")

	if err := cmd.Parse(args); err != nil {
		return err
	}
	if *crate == "" {
		return fmt.Errorf("--crate is required")
	}
	if *version == "" {
		return fmt.Errorf("--version is required")
	}
	if *sha256 == "" {
		return fmt.Errorf("--sha256 is required")
	}
	if *out == "" {
		return fmt.Errorf("--out is required")
	}
	return download.DownloadCrate(*crate, *version, *sha256, *out)
}

func handleHash(args []string, stdout io.Writer) error {
	cmd := flag.NewFlagSet("hash", flag.ContinueOnError)
	crate := cmd.String("crate", "", "Crate name")
	version := cmd.String("version", "", "Crate version")

	if err := cmd.Parse(args); err != nil {
		return err
	}
	if *crate == "" {
		return fmt.Errorf("--crate is required")
	}
	if *version == "" {
		return fmt.Errorf("--version is required")
	}
	h, err := download.HashCrate(*crate, *version)
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, h)
	return nil
}

func handleCompileC(args []string) error {
	cmd := flag.NewFlagSet("compile-c", flag.ContinueOnError)
	out := cmd.String("out", "", "Output static archive path")
	var includes []string
	cmd.Func("include", "Include directory (repeatable)", func(s string) error {
		includes = append(includes, s)
		return nil
	})

	if err := cmd.Parse(args); err != nil {
		return err
	}
	if *out == "" {
		return fmt.Errorf("--out is required")
	}
	sources := cmd.Args()
	return compilec.CompileC(*out, includes, sources)
}

func handleTestRunner(args []string) error {
	cmd := flag.NewFlagSet("test-runner", flag.ContinueOnError)
	pkg := cmd.String("pkg", "", "Package / target name")
	resultsFile := cmd.String("results-file", "test.results", "Path to write JUnit XML results")
	coverageFile := cmd.String("coverage-file", "", "Path to write coverage report (default: $COVERAGE_FILE or test.coverage)")
	llvmProfdata := cmd.String("llvm-profdata", "", "Path to llvm-profdata executable")
	llvmCov := cmd.String("llvm-cov", "", "Path to llvm-cov executable")
	coverage := cmd.Bool("coverage", false, "Force coverage collection")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	cmdArgs := cmd.Args()
	if len(cmdArgs) == 0 {
		return fmt.Errorf("test binary argument is required")
	}

	testBinary := cmdArgs[0]
	extraArgs := cmdArgs[1:]
	return testrunner.RunWithOptions(testrunner.RunOptions{
		PkgName:        *pkg,
		TestBinary:     testBinary,
		ExtraArgs:      extraArgs,
		ResultsFile:    *resultsFile,
		CoverageActive: *coverage,
		CoverageFile:   *coverageFile,
		LlvmProfdata:   *llvmProfdata,
		LlvmCov:        *llvmCov,
	})
}

func runCLI(args []string, stdout, stderr io.Writer) error {
	if len(args) < 1 {
		printUsage(stderr)
		return fmt.Errorf("no command specified")
	}

	command := args[0]
	cmdArgs := args[1:]

	switch command {
	case "compile":
		return handleCompile(cmdArgs)
	case "download":
		return handleDownload(cmdArgs)
	case "hash":
		return handleHash(cmdArgs, stdout)
	case "compile-c":
		return handleCompileC(cmdArgs)
	case "test-runner":
		return handleTestRunner(cmdArgs)
	default:
		fmt.Fprintf(stderr, "Unknown command: %s\n\n", command)
		printUsage(stderr)
		return fmt.Errorf("unknown command: %s", command)
	}
}

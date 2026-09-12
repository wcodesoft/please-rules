package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"tools/please_rust/compile"
	"tools/please_rust/compilec"
	"tools/please_rust/download"
	"tools/please_rust/testrunner"
)

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: plz_rust <command> [arguments]\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  compile       Compile Rust code with rustc\n")
	fmt.Fprintf(os.Stderr, "  download      Download a crate from crates.io and verify its checksum\n")
	fmt.Fprintf(os.Stderr, "  hash          Print the SHA-256 of a crate tarball from crates.io\n")
	fmt.Fprintf(os.Stderr, "  compile-c     Compile C sources into a static archive\n")
	fmt.Fprintf(os.Stderr, "  test-runner   Execute test binary and output JUnit results\n")
}

func handleCompile(args []string) error {
	cmd := flag.NewFlagSet("compile", flag.ExitOnError)
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
	cmd := flag.NewFlagSet("download", flag.ExitOnError)
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

func handleHash(args []string) error {
	cmd := flag.NewFlagSet("hash", flag.ExitOnError)
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
	fmt.Println(h)
	return nil
}

func handleCompileC(args []string) error {
	cmd := flag.NewFlagSet("compile-c", flag.ExitOnError)
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
	cmd := flag.NewFlagSet("test-runner", flag.ExitOnError)
	pkg := cmd.String("pkg", "", "Package / target name")
	resultsFile := cmd.String("results-file", "test.results", "Path to write JUnit XML results")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	cmdArgs := cmd.Args()
	if len(cmdArgs) == 0 {
		return fmt.Errorf("test binary argument is required")
	}

	testBinary := cmdArgs[0]
	extraArgs := cmdArgs[1:]
	return testrunner.Run(*pkg, testBinary, extraArgs, *resultsFile)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	var err error
	switch command {
	case "compile":
		err = handleCompile(args)
	case "download":
		err = handleDownload(args)
	case "hash":
		err = handleHash(args)
	case "compile-c":
		err = handleCompileC(args)
	case "test-runner":
		err = handleTestRunner(args)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

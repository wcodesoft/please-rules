package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"tools/please_rust/compile"
	"tools/please_rust/fetch"
	"tools/please_rust/testrunner"
)

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: plz_rust <command> [arguments]\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  compile       Compile Rust code with rustc\n")
	fmt.Fprintf(os.Stderr, "  fetch         Fetch and build third-party crates with cargo\n")
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

	if err := cmd.Parse(args); err != nil {
		return err
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
	}
	return compile.Run(opts)
}

func handleFetch(args []string) error {
	cmd := flag.NewFlagSet("fetch", flag.ExitOnError)
	cargo := cmd.String("cargo", "", "Path to cargo executable")
	rustc := cmd.String("rustc", "", "Path to rustc executable")
	crateName := cmd.String("crate", "", "Name of the single crate to fetch")
	version := cmd.String("version", "", "Version of the single crate to fetch")
	featuresFlag := cmd.String("features", "", "Comma-separated features for the crate")
	procMacro := cmd.Bool("proc-macro", false, "Whether this crate is a proc macro")
	buildFile := cmd.String("build-file", "", "Path to BUILD file containing rust_crate declarations")
	outDir := cmd.String("out-dir", "", "Directory to output built rlibs")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	if *outDir == "" {
		return fmt.Errorf("--out-dir is required")
	}

	if *crateName != "" {
		var features []string
		if *featuresFlag != "" {
			features = strings.Split(*featuresFlag, ",")
		}
		return fetch.FetchCrate(*cargo, *rustc, *crateName, *version, features, *procMacro, *outDir)
	}

	if *buildFile != "" {
		return fetch.FetchAll(*cargo, *rustc, *buildFile, *outDir)
	}

	return fmt.Errorf("either --crate or --build-file must be specified")
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
	case "fetch":
		err = handleFetch(args)
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

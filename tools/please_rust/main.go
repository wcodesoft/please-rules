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

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "compile":
		compileCmd := flag.NewFlagSet("compile", flag.ExitOnError)
		out := compileCmd.String("out", "", "Output artifact path")
		crateName := compileCmd.String("crate-name", "", "Name of the crate")
		crateType := compileCmd.String("crate-type", "", "Type of crate: rlib, bin, proc-macro, test")
		edition := compileCmd.String("edition", "2021", "Rust edition (e.g. 2021, 2024)")
		mainSrc := compileCmd.String("main-src", "", "Main source entrypoint file")
		version := compileCmd.String("version", "", "Crate version")
		flags := compileCmd.String("flags", "", "Additional rustc flags")
		rustc := compileCmd.String("rustc", "", "Path to rustc executable")

		if err := compileCmd.Parse(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing compile args: %v\n", err)
			os.Exit(1)
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
			Inputs:    compileCmd.Args(),
		}

		if err := compile.Run(opts); err != nil {
			fmt.Fprintf(os.Stderr, "Compile error: %v\n", err)
			os.Exit(1)
		}

	case "fetch":
		fetchCmd := flag.NewFlagSet("fetch", flag.ExitOnError)
		cargo := fetchCmd.String("cargo", "", "Path to cargo executable")
		rustc := fetchCmd.String("rustc", "", "Path to rustc executable")
		crateName := fetchCmd.String("crate", "", "Name of the single crate to fetch")
		version := fetchCmd.String("version", "", "Version of the single crate to fetch")
		featuresFlag := fetchCmd.String("features", "", "Comma-separated features for the crate")
		buildFile := fetchCmd.String("build-file", "", "Path to BUILD file containing rust_crate declarations")
		outDir := fetchCmd.String("out-dir", "", "Directory to output built rlibs")

		if err := fetchCmd.Parse(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing fetch args: %v\n", err)
			os.Exit(1)
		}

		if *outDir == "" {
			fmt.Fprintf(os.Stderr, "--out-dir is required\n")
			os.Exit(1)
		}

		if *crateName != "" {
			var features []string
			if *featuresFlag != "" {
				features = strings.Split(*featuresFlag, ",")
			}
			if err := fetch.FetchCrate(*cargo, *rustc, *crateName, *version, features, *outDir); err != nil {
				fmt.Fprintf(os.Stderr, "Fetch crate error: %v\n", err)
				os.Exit(1)
			}
		} else if *buildFile != "" {
			if err := fetch.FetchAll(*cargo, *rustc, *buildFile, *outDir); err != nil {
				fmt.Fprintf(os.Stderr, "Fetch all error: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Either --crate or --build-file must be specified\n")
			os.Exit(1)
		}

	case "test-runner":
		testCmd := flag.NewFlagSet("test-runner", flag.ExitOnError)
		pkg := testCmd.String("pkg", "", "Package / target name")
		resultsFile := testCmd.String("results-file", "test.results", "Path to write JUnit XML results")

		if err := testCmd.Parse(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing test-runner args: %v\n", err)
			os.Exit(1)
		}

		args := testCmd.Args()
		if len(args) == 0 {
			fmt.Fprintf(os.Stderr, "test binary argument is required\n")
			os.Exit(1)
		}

		testBinary := args[0]
		extraArgs := args[1:]

		if err := testrunner.Run(*pkg, testBinary, extraArgs, *resultsFile); err != nil {
			fmt.Fprintf(os.Stderr, "Test execution failed: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

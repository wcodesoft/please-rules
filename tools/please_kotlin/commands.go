package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"tools/please_kotlin/compile"
	"tools/please_kotlin/testrunner"
)

func handleCompile(args []string) error {
	cmd := flag.NewFlagSet("compile", flag.ContinueOnError)
	out := cmd.String("out", "", "Output jar path")
	srcsFlag := cmd.String("srcs", "", "Comma-separated source files")
	depsFlag := cmd.String("deps", "", "Comma-separated dependency jar paths")
	kotlinc := cmd.String("kotlinc", "", "Path to kotlinc executable")
	java := cmd.String("java", "", "Path to java binary")
	jvmTarget := cmd.String("jvm-target", "21", "Target JVM bytecode version")
	moduleName := cmd.String("module-name", "", "Kotlin module name")
	mainClass := cmd.String("main-class", "", "Main class for JAR manifest")
	flags := cmd.String("flags", "", "Additional kotlinc flags")
	executable := cmd.Bool("executable", false, "Produce self-executing runnable JAR")
	mergeDeps := cmd.Bool("merge-deps", false, "Merge dependency JARs into output JAR")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	srcs := compile.ExpandCommaSeparated([]string{*srcsFlag})
	srcs = append(srcs, cmd.Args()...)
	var cleanSrcs []string
	for _, s := range srcs {
		s = strings.TrimSpace(s)
		if s != "" && s != "\\" {
			cleanSrcs = append(cleanSrcs, s)
		}
	}

	deps := compile.ExpandCommaSeparated([]string{*depsFlag})
	var cleanDeps []string
	for _, d := range deps {
		d = strings.TrimSpace(d)
		if d != "" && d != "\\" {
			cleanDeps = append(cleanDeps, d)
		}
	}

	var extraFlags []string
	if *flags != "" {
		extraFlags = strings.Fields(*flags)
	}

	opts := compile.Options{
		Kotlinc:    *kotlinc,
		Out:        *out,
		Srcs:       cleanSrcs,
		Deps:       cleanDeps,
		JvmTarget:  *jvmTarget,
		ModuleName: *moduleName,
		MainClass:  *mainClass,
		Flags:      extraFlags,
		Executable: *executable,
		MergeDeps:  *mergeDeps,
		Java:       *java,
	}
	return compile.Run(opts)
}

func handleTestRunner(args []string) error {
	cmd := flag.NewFlagSet("testrunner", flag.ContinueOnError)
	java := cmd.String("java", "", "Path to java binary")
	testJar := cmd.String("test-jar", "", "Path to test jar")
	depsFlag := cmd.String("deps", "", "Comma-separated dependency jar paths")
	testClass := cmd.String("test-class", "", "JUnit test class to execute via ConsoleLauncher (mandatory)")
	junitRunner := cmd.String("junit-runner", "", "Path to junit-platform-console-standalone.jar")
	resultsFile := cmd.String("results-file", "test.results", "Path to write JUnit results XML")
	coverageFile := cmd.String("coverage-file", "", "Path to write coverage output")
	coverageActive := cmd.Bool("coverage", false, "Whether coverage is active")
	jacocoAgent := cmd.String("jacoco-agent", "", "Path to jacocoagent.jar")
	jacocoCli := cmd.String("jacoco-cli", "", "Path to jacococli.jar")
	srcsFlag := cmd.String("srcs", "", "Comma-separated source files for coverage")
	classFiles := cmd.String("class-files", "", "Path to classfiles or jar for JaCoCo")
	repoRoot := cmd.String("repo-root", "", "Repository root directory")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	if *testClass == "" {
		return fmt.Errorf("--test-class is mandatory: specify the JUnit test class to execute")
	}

	deps := compile.ExpandCommaSeparated([]string{*depsFlag})
	srcs := compile.ExpandCommaSeparated([]string{*srcsFlag})

	opts := testrunner.RunOptions{
		Java:           *java,
		TestJar:        *testJar,
		Deps:           deps,
		TestClass:      *testClass,
		JunitRunner:    *junitRunner,
		ExtraArgs:      cmd.Args(),
		ResultsFile:    *resultsFile,
		CoverageActive: *coverageActive,
		CoverageFile:   *coverageFile,
		JacocoAgent:    *jacocoAgent,
		JacocoCli:      *jacocoCli,
		SourceFiles:    srcs,
		ClassFiles:     *classFiles,
		RepoRoot:       *repoRoot,
	}
	return testrunner.Run(opts)
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, "please_kotlin - Build helper for Please Kotlin rules\n\n")
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  please_kotlin compile [options] [-- sources...]\n")
	fmt.Fprintf(w, "  please_kotlin testrunner [options] [-- jvm-args...]\n")
}

func runCommand(args []string) error {
	if len(args) < 1 {
		printUsage(os.Stderr)
		return fmt.Errorf("no subcommand specified")
	}

	switch args[0] {
	case "compile":
		return handleCompile(args[1:])
	case "testrunner":
		return handleTestRunner(args[1:])
	case "help", "--help", "-h":
		printUsage(os.Stdout)
		return nil
	default:
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

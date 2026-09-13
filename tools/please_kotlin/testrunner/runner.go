package testrunner

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"tools/please_kotlin/compile"
	"tools/please_kotlin/toolchain"
)

// DefaultResultsFile is the standard Please test results XML path.
const DefaultResultsFile = "test.results"

// RunOptions configures the execution of a test runner and coverage collection.
type RunOptions struct {
	Java           string
	TestJar        string
	Deps           []string
	TestClass      string
	JunitRunner    string
	ExtraArgs      []string
	ResultsFile    string
	CoverageActive bool
	CoverageFile   string
	JacocoAgent    string
	JacocoCli      string
	SourceFiles    []string
	ClassFiles     string
	RepoRoot       string
}

func resolveCoverage(opts RunOptions) (bool, string) {
	active := opts.CoverageActive || os.Getenv("COVERAGE") == "true" || os.Getenv("COVERAGE_FILE") != ""
	file := opts.CoverageFile
	if file == "" {
		file = os.Getenv("COVERAGE_FILE")
	}
	if file == "" && active {
		file = "test.coverage"
	}
	return active, file
}

func buildJvmArgs(opts RunOptions, jacocoExec, reportDir string) []string {
	var args []string
	if jacocoExec != "" && opts.JacocoAgent != "" {
		args = append(args, fmt.Sprintf("-javaagent:%s=destfile=%s,includes=*", opts.JacocoAgent, jacocoExec))
	}

	seenCp := make(map[string]bool)
	var cpParts []string
	addCp := func(p string) {
		clean := filepath.Clean(p)
		if clean == "" || clean == "." {
			return
		}
		if !seenCp[clean] {
			seenCp[clean] = true
			cpParts = append(cpParts, clean)
		}
	}

	if opts.JunitRunner != "" {
		addCp(opts.JunitRunner)
	}
	if opts.TestJar != "" {
		addCp(opts.TestJar)
	}
	allJars := compile.DiscoverJars(opts.Deps, "")
	for _, jar := range allJars {
		if opts.TestJar != "" && filepath.Clean(jar) == filepath.Clean(opts.TestJar) {
			continue
		}
		addCp(jar)
	}
	if stdlib := toolchain.FindKotlinStdlib(opts.Java); stdlib != "" {
		addCp(stdlib)
	}

	cp := strings.Join(cpParts, string(os.PathListSeparator))
	if cp != "" {
		args = append(args, "-cp", cp)
	}

	args = append(args, "org.junit.platform.console.ConsoleLauncher", "execute",
		"--reports-dir", reportDir,
		"--select-class", opts.TestClass,
	)
	args = append(args, opts.ExtraArgs...)
	return args
}

// Run executes the test JVM via JUnit 5 Platform and optionally collects coverage.
func Run(opts RunOptions) error {
	if opts.TestClass == "" {
		return fmt.Errorf("test-class is mandatory: specify the JUnit test class to execute")
	}

	javaBin, err := toolchain.ResolveJava(opts.Java)
	if err != nil {
		return fmt.Errorf("failed to resolve java binary: %w", err)
	}

	activeCov, covFile := resolveCoverage(opts)
	tmpDir, err := os.MkdirTemp("", "kotlin-test-*")
	if err != nil {
		return fmt.Errorf("failed to create temp test dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	jacocoExec := ""
	if activeCov && opts.JacocoAgent != "" {
		jacocoExec = filepath.Join(tmpDir, "jacoco.exec")
	}

	jvmArgs := buildJvmArgs(opts, jacocoExec, tmpDir)
	cmd := exec.Command(javaBin, jvmArgs...)
	cmd.Dir = "."

	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf

	start := time.Now()
	runErr := cmd.Run()
	duration := time.Since(start)

	output := outBuf.String()
	fmt.Print(output)

	resultsFile := opts.ResultsFile
	if resultsFile == "" {
		resultsFile = DefaultResultsFile
	}

	written := false
	entries, _ := os.ReadDir(tmpDir)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".xml") {
			xmlBytes, err := os.ReadFile(filepath.Join(tmpDir, e.Name()))
			if err == nil && len(xmlBytes) > 0 {
				_ = os.WriteFile(resultsFile, xmlBytes, 0644)
				written = true
				break
			}
		}
	}

	if !written {
		suite := ParseTestOutput(output, opts.TestClass, opts.TestClass, duration, runErr == nil)
		_ = WriteJUnitResults(resultsFile, suite)
	}

	if activeCov && jacocoExec != "" {
		_ = processJacocoCoverage(javaBin, opts, tmpDir, jacocoExec, covFile)
	}

	return runErr
}

func processJacocoCoverage(javaBin string, opts RunOptions, tmpDir, jacocoExec, covFile string) error {
	if _, err := os.Stat(jacocoExec); os.IsNotExist(err) {
		return nil
	}
	if opts.JacocoCli == "" {
		return nil
	}

	classTarget := opts.ClassFiles
	if classTarget == "" {
		classTarget = opts.TestJar
	}
	if classTarget == "" {
		return nil
	}

	reportXml := filepath.Join(tmpDir, "jacoco_report.xml")
	repCmd := exec.Command(javaBin, "-jar", opts.JacocoCli, "report", jacocoExec,
		"--classfiles", classTarget, "--xml", reportXml)
	if err := repCmd.Run(); err != nil {
		return err
	}

	xmlData, err := os.ReadFile(reportXml)
	if err != nil {
		return err
	}

	parsed, err := ParseJacocoXml(xmlData)
	if err != nil {
		return err
	}

	normalized := NormalizeCoveragePaths(parsed, opts.RepoRoot, opts.SourceFiles)
	gcovData := FormatGcov(normalized, opts.RepoRoot)

	if err := os.MkdirAll(filepath.Dir(covFile), 0755); err != nil {
		return err
	}
	return os.WriteFile(covFile, gcovData, 0644)
}

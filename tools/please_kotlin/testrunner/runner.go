package testrunner

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
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
	LcovFile       string // raw lcov export (functions and branches), written empty without coverage
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

	// The raw lcov export is a declared test output: create it up front so it exists
	// (empty) when coverage is off or the report cannot be produced.
	if opts.LcovFile != "" {
		if err := os.MkdirAll(filepath.Dir(opts.LcovFile), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(opts.LcovFile, nil, 0644); err != nil {
			return fmt.Errorf("failed to reset lcov file: %w", err)
		}
	}

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

	suites, err := ReadJUnitReports(tmpDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: unreadable JUnit report: %v\n", err)
	}
	// Without a report, or when the process failed but no test case did (the JVM crashed
	// at exit, no test was found), say so with a test case of its own.
	if len(suites) == 0 {
		suites = []JUnitTestSuite{ProcessSuite(opts.TestClass, output, duration, runErr == nil)}
	} else if runErr != nil && !Failed(suites) {
		suites = append(suites, ProcessSuite(opts.TestClass+" (process)", output, duration, false))
	}
	if err := WriteJUnitResults(resultsFile, suites...); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write test results: %v\n", err)
	}

	if activeCov && jacocoExec != "" {
		if err := processJacocoCoverage(javaBin, opts, tmpDir, jacocoExec, covFile); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to process coverage: %v\n", err)
		}
	}

	return runErr
}

func processJacocoCoverage(javaBin string, opts RunOptions, tmpDir, jacocoExec, covFile string) error {
	if _, err := os.Stat(jacocoExec); os.IsNotExist(err) {
		return fmt.Errorf("JaCoCo agent wrote no execution data to %s", jacocoExec)
	}
	if opts.JacocoCli == "" {
		return fmt.Errorf("no JaCoCo CLI configured (set JacocoCli)")
	}

	classTarget := opts.ClassFiles
	if classTarget == "" {
		classTarget = opts.TestJar
		// The test jar bundles third-party classes that JaCoCo cannot analyze (and
		// that do not belong in the report); restrict it to the project's classes.
		if projectDir, err := extractProjectClasses(opts.TestJar, filepath.Join(tmpDir, "project-classes")); err != nil {
			return err
		} else if projectDir != "" {
			classTarget = projectDir
		}
	}
	if classTarget == "" {
		return fmt.Errorf("no class files to report coverage on")
	}

	reportXml := filepath.Join(tmpDir, "jacoco_report.xml")
	repCmd := exec.Command(javaBin, "-jar", opts.JacocoCli, "report", jacocoExec,
		"--classfiles", classTarget, "--xml", reportXml)
	if out, err := repCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("jacococli report: %w: %s", err, strings.TrimSpace(string(out)))
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
	if err := os.WriteFile(covFile, gcovData, 0644); err != nil {
		return err
	}
	return writeLcovFile(opts.LcovFile, xmlData, opts.RepoRoot, opts.SourceFiles)
}

// writeLcovFile exports the JaCoCo report as lcov. An empty path disables the export.
func writeLcovFile(path string, jacocoXML []byte, repoRoot string, knownSrcs []string) error {
	if path == "" {
		return nil
	}
	report, err := JacocoToLcov(jacocoXML, repoRoot, knownSrcs)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := report.Write(&buf); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

// extractProjectClasses unpacks the project classes listed in jar into dir and
// returns dir, or "" if the jar does not list its project classes.
func extractProjectClasses(jar, dir string) (string, error) {
	names := compile.ReadProjectClasses(jar)
	if len(names) == 0 {
		return "", nil
	}
	r, err := zip.OpenReader(jar)
	if err != nil {
		return "", err
	}
	defer r.Close()
	wanted := make(map[string]bool, len(names))
	for _, n := range names {
		wanted[n] = true
	}
	for _, f := range r.File {
		if !wanted[f.Name] || !strings.HasSuffix(f.Name, ".class") {
			continue
		}
		dest := filepath.Join(dir, filepath.FromSlash(f.Name))
		if !strings.HasPrefix(dest, filepath.Clean(dir)+string(os.PathSeparator)) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return "", err
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(dest, data, 0644); err != nil {
			return "", err
		}
	}
	return dir, nil
}

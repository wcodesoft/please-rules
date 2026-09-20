package testrunner

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// RunOptions configures the execution of the Swift test runner.
type RunOptions struct {
	Swiftc       string
	LlvmProfdata string
	LlvmCov      string
	Framework    string
	Srcs         []string
	Deps         []string
	Out          string
	Binary       string
	CompileOnly  bool
	ResultsFile  string
	Coverage     bool
	CoverageFile string
	TestArgs     []string
	Flags        []string
}

// JUnit XML structs
type JUnitTestSuites struct {
	XMLName  xml.Name         `xml:"testsuites"`
	Suites   []JUnitTestSuite `xml:"testsuite"`
	Tests    int              `xml:"tests,attr"`
	Failures int              `xml:"failures,attr"`
	Errors   int              `xml:"errors,attr"`
	Skipped  int              `xml:"skipped,attr,omitempty"`
	Time     string           `xml:"time,attr"`
}

type JUnitTestSuite struct {
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Skipped   int             `xml:"skipped,attr,omitempty"`
	Time      string          `xml:"time,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

type JUnitTestCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Skipped   *JUnitSkipped `xml:"skipped,omitempty"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
}

type JUnitSkipped struct {
	Message string `xml:"message,attr,omitempty"`
}

type JUnitFailure struct {
	Message  string `xml:"message,attr"`
	Type     string `xml:"type,attr,omitempty"`
	Contents string `xml:",chardata"`
}

type ParsedTestCase struct {
	Suite    string
	Name     string
	Duration float64
	Passed   bool
	Skipped  bool
	Failure  string
}

func resolveTool(tool string) string {
	if tool == "" {
		return tool
	}
	if filepath.IsAbs(tool) {
		return tool
	}
	if p, err := exec.LookPath(tool); err == nil {
		return p
	}
	commonDirs := []string{
		"/home/linuxbrew/.linuxbrew/bin",
		"/usr/local/bin",
		"/usr/bin",
		"/opt/swift/usr/bin",
	}
	for _, dir := range commonDirs {
		candidate := filepath.Join(dir, tool)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return tool
}

// Run coordinates building, executing, and reporting tests.
func Run(opts RunOptions) error {
	if opts.Swiftc == "" {
		opts.Swiftc = "swiftc"
	}
	opts.Swiftc = resolveTool(opts.Swiftc)
	if opts.LlvmProfdata == "" {
		opts.LlvmProfdata = "llvm-profdata"
	}
	opts.LlvmProfdata = resolveTool(opts.LlvmProfdata)
	if opts.LlvmCov == "" {
		opts.LlvmCov = "llvm-cov"
	}
	opts.LlvmCov = resolveTool(opts.LlvmCov)
	if opts.ResultsFile == "" {
		opts.ResultsFile = "test.results"
	}
	if opts.Framework == "" {
		opts.Framework = "swift-testing"
	}

	coverageActive := opts.Coverage || os.Getenv("COVERAGE") != "" || os.Getenv("COVERAGE_FILE") != ""
	fmt.Fprintf(os.Stderr, "DEBUG ENV: COVERAGE=%q COVERAGE_FILE=%q coverageActive=%v opts.Coverage=%v\n", os.Getenv("COVERAGE"), os.Getenv("COVERAGE_FILE"), coverageActive, opts.Coverage)
	if opts.CoverageFile == "" {
		if covEnv := os.Getenv("COVERAGE_FILE"); covEnv != "" {
			opts.CoverageFile = covEnv
		} else {
			opts.CoverageFile = "test.coverage"
		}
	}

	tmpDir, err := os.MkdirTemp("", "please_swift_test_*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	testBinary := opts.Out
	if testBinary == "" {
		testBinary = filepath.Join(tmpDir, "test_binary")
	}

	if opts.Binary != "" && !coverageActive {
		testBinary = opts.Binary
	} else {
		// 1. Check for @main in source files
		hasMain := false
		for _, src := range opts.Srcs {
			if data, err := os.ReadFile(src); err == nil {
				if strings.Contains(string(data), "@main") {
					hasMain = true
					break
				}
			}
		}

		compileSrcs := append([]string{}, opts.Srcs...)
		extraFlags := append([]string{}, opts.Flags...)

		if !hasMain && opts.Framework == "swift-testing" {
			runnerMainPath := filepath.Join(tmpDir, "__runner_main.swift")
			runnerCode := `import Testing
#if canImport(Glibc)
import Glibc
#elseif canImport(Darwin)
import Darwin
#endif

@main
struct __PleaseTestRunner {
    static func main() async {
        var args = Testing.__CommandLineArguments_v0()
        if let env = getenv("PLEASE_SWIFT_XUNIT_OUTPUT") {
            args.xunitOutput = String(cString: env)
        }
        let exitCode: CInt = await Testing.__swiftPMEntryPoint(passing: args)
        exit(exitCode)
    }
}
`
			if err := os.WriteFile(runnerMainPath, []byte(runnerCode), 0644); err != nil {
				return fmt.Errorf("failed to write test runner main: %w", err)
			}
			compileSrcs = append(compileSrcs, runnerMainPath)
			extraFlags = append(extraFlags, "-parse-as-library", "-lTesting")
		}

		if coverageActive {
			extraFlags = append(extraFlags, "-profile-generate", "-profile-coverage-mapping")
		}

		// 2. Discover dependency paths
		incDirs, archives := discoverDependencies(opts.Deps)

		var compileArgs []string
		compileArgs = append(compileArgs, "-o", testBinary)
		for _, inc := range incDirs {
			compileArgs = append(compileArgs, "-I", inc)
		}
		compileArgs = append(compileArgs, extraFlags...)
		compileArgs = append(compileArgs, compileSrcs...)

		if runtime.GOOS == "linux" && len(archives) > 0 {
			compileArgs = append(compileArgs, "-Xlinker", "--start-group")
			for _, arch := range archives {
				compileArgs = append(compileArgs, arch)
			}
			compileArgs = append(compileArgs, "-Xlinker", "--end-group")
		} else {
			for _, arch := range archives {
				compileArgs = append(compileArgs, arch)
			}
		}

		var compileErr bytes.Buffer
		cCmd := exec.Command(opts.Swiftc, compileArgs...)
		cCmd.Stderr = &compileErr
		if err := cCmd.Run(); err != nil {
			return fmt.Errorf("failed to compile test executable: %v\n%s", err, compileErr.String())
		}
	}

	if opts.CompileOnly {
		return nil
	}

	// 3. Execute test binary
	profDir := filepath.Join(tmpDir, "profiles")
	_ = os.MkdirAll(profDir, 0755)
	profPattern := filepath.Join(profDir, "test_%p.profraw")

	testCmd := exec.Command(testBinary, opts.TestArgs...)
	testCmd.Env = append(os.Environ(), "LLVM_PROFILE_FILE="+profPattern)
	if absResults, err := filepath.Abs(opts.ResultsFile); err == nil {
		testCmd.Env = append(testCmd.Env, "PLEASE_SWIFT_XUNIT_OUTPUT="+absResults)
	} else {
		testCmd.Env = append(testCmd.Env, "PLEASE_SWIFT_XUNIT_OUTPUT="+opts.ResultsFile)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	testCmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	testCmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	startTime := time.Now()
	testErr := testCmd.Run()
	elapsed := time.Since(startTime).Seconds()

	fullOutput := stdoutBuf.String() + "\n" + stderrBuf.String()

	// 4. Check if native xUnit XML was produced; otherwise fall back to output parsing
	var cases []ParsedTestCase
	hasResultsFile := false
	if info, err := os.Stat(opts.ResultsFile); err == nil && info.Size() > 0 {
		hasResultsFile = true
	}

	if !hasResultsFile {
		cases = parseTestOutput(fullOutput, opts.Framework)
		if len(cases) == 0 {
			// Fallback single testcase
			passed := testErr == nil
			failMsg := ""
			if !passed {
				failMsg = fmt.Sprintf("Test execution failed: %v", testErr)
			}
			cases = append(cases, ParsedTestCase{
				Suite:    "SwiftTestSuite",
				Name:     "Execution",
				Duration: elapsed,
				Passed:   passed,
				Failure:  failMsg,
			})
		}

		if err := writeJUnitResults(opts.ResultsFile, cases, elapsed); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to write JUnit test results: %v\n", err)
		}
	}

	// 5. Handle coverage if enabled
	if coverageActive {
		if err := processCoverage(opts, testBinary, profDir, tmpDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to process coverage: %v\n", err)
		}
	}

	if testErr != nil {
		return fmt.Errorf("test execution failed: %w", testErr)
	}

	for _, c := range cases {
		if !c.Passed && !c.Skipped {
			return fmt.Errorf("one or more tests failed")
		}
	}

	return nil
}

func cleanSwiftTestName(raw string) string {
	name := strings.TrimSpace(raw)
	name = strings.Trim(name, `"'`+"`")
	name = strings.TrimSuffix(name, "()")
	name = strings.Trim(name, `"'`+"`")
	return strings.TrimSpace(name)
}

func isTestRunSummary(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	return trimmed == "run" || strings.HasPrefix(trimmed, "run with ") || strings.HasPrefix(trimmed, "run ")
}

func parseTestOutput(output, framework string) []ParsedTestCase {
	var results []ParsedTestCase

	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	cleanOutput := ansiRegex.ReplaceAllString(output, "")

	// Detect suite name if present (e.g. "◇ Suite Foo started" or "✔ Suite Foo passed")
	suiteName := "SwiftTesting"
	suiteRe := regexp.MustCompile(`(?:◇|✔)\s+Suite\s+(.+?)\s+(?:started|passed)`)
	allSuites := suiteRe.FindAllStringSubmatch(cleanOutput, -1)
	uniqueSuites := make(map[string]bool)
	for _, m := range allSuites {
		if len(m) > 1 {
			s := cleanSwiftTestName(m[1])
			if s != "" {
				uniqueSuites[s] = true
			}
		}
	}
	if len(uniqueSuites) == 1 {
		for s := range uniqueSuites {
			suiteName = s
		}
	}

	// swift-testing format:
	// ✔ Test <name> passed after X.XXX seconds.
	// ✘ Test <name> failed after X.XXX seconds with N issues.
	// ✘ Test <name> recorded an issue at <file>:<line>:<col>: <msg>
	// ➜ Test <name> skipped: <reason>
	swiftTestingPassedRe := regexp.MustCompile(`✔\s+Test\s+(.+?)\s+passed after\s+([0-9.]+)\s+seconds`)
	swiftTestingFailedRe := regexp.MustCompile(`✘\s+Test\s+(.+?)\s+failed after\s+([0-9.]+)\s+seconds`)
	swiftTestingIssueRe := regexp.MustCompile(`✘\s+Test\s+(.+?)\s+recorded an issue at\s+(.*)`)
	swiftTestingSkippedRe := regexp.MustCompile(`➜\s+Test\s+(.+?)\s+skipped(?::\s*(.*))?`)

	issuesByTest := make(map[string][]string)
	for _, match := range swiftTestingIssueRe.FindAllStringSubmatch(cleanOutput, -1) {
		if isTestRunSummary(match[1]) {
			continue
		}
		testName := cleanSwiftTestName(match[1])
		issuesByTest[testName] = append(issuesByTest[testName], match[2])
	}

	for _, match := range swiftTestingPassedRe.FindAllStringSubmatch(cleanOutput, -1) {
		if isTestRunSummary(match[1]) {
			continue
		}
		name := cleanSwiftTestName(match[1])
		duration, _ := strconv.ParseFloat(match[2], 64)
		results = append(results, ParsedTestCase{
			Suite:    suiteName,
			Name:     name,
			Duration: duration,
			Passed:   true,
		})
	}

	for _, match := range swiftTestingFailedRe.FindAllStringSubmatch(cleanOutput, -1) {
		if isTestRunSummary(match[1]) {
			continue
		}
		name := cleanSwiftTestName(match[1])
		duration, _ := strconv.ParseFloat(match[2], 64)
		issueDetails := strings.Join(issuesByTest[name], "\n")
		if issueDetails == "" {
			issueDetails = "Test failed"
		}
		results = append(results, ParsedTestCase{
			Suite:    suiteName,
			Name:     name,
			Duration: duration,
			Passed:   false,
			Failure:  issueDetails,
		})
	}

	for _, match := range swiftTestingSkippedRe.FindAllStringSubmatch(cleanOutput, -1) {
		if isTestRunSummary(match[1]) {
			continue
		}
		name := cleanSwiftTestName(match[1])
		reason := "Test skipped"
		if len(match) > 2 && strings.TrimSpace(match[2]) != "" {
			reason = strings.Trim(strings.TrimSpace(match[2]), `"`)
		}
		results = append(results, ParsedTestCase{
			Suite:    suiteName,
			Name:     name,
			Duration: 0.0,
			Passed:   true,
			Skipped:  true,
			Failure:  reason,
		})
	}

	if len(results) > 0 {
		return results
	}

	// XCTest format fallback:
	// Test Case '-[Suite.Test testMethod]' passed (0.001 seconds).
	// Test Case '-[Suite.Test testMethod]' failed (0.002 seconds).
	xctestPassedRe := regexp.MustCompile(`Test Case '-\[([^ ]+) ([^\]]+)\]' passed \(([0-9.]+) seconds\)`)
	xctestFailedRe := regexp.MustCompile(`Test Case '-\[([^ ]+) ([^\]]+)\]' failed \(([0-9.]+) seconds\)`)

	for _, match := range xctestPassedRe.FindAllStringSubmatch(cleanOutput, -1) {
		duration, _ := strconv.ParseFloat(match[3], 64)
		results = append(results, ParsedTestCase{
			Suite:    match[1],
			Name:     match[2],
			Duration: duration,
			Passed:   true,
		})
	}

	for _, match := range xctestFailedRe.FindAllStringSubmatch(cleanOutput, -1) {
		duration, _ := strconv.ParseFloat(match[3], 64)
		results = append(results, ParsedTestCase{
			Suite:    match[1],
			Name:     match[2],
			Duration: duration,
			Passed:   false,
			Failure:  "XCTest failed",
		})
	}

	return results
}

func writeJUnitResults(path string, cases []ParsedTestCase, totalTime float64) error {
	_ = os.MkdirAll(filepath.Dir(path), 0755)

	suitesMap := make(map[string][]ParsedTestCase)
	for _, c := range cases {
		suitesMap[c.Suite] = append(suitesMap[c.Suite], c)
	}

	var suites []JUnitTestSuite
	totalTests := 0
	totalFailures := 0
	totalSkipped := 0

	for suiteName, suiteCases := range suitesMap {
		s := JUnitTestSuite{
			Name: suiteName,
		}
		suiteTime := 0.0
		for _, c := range suiteCases {
			tc := JUnitTestCase{
				Name:      c.Name,
				ClassName: c.Suite,
				Time:      fmt.Sprintf("%.4f", c.Duration),
			}
			suiteTime += c.Duration
			totalTests++
			if c.Skipped {
				tc.Skipped = &JUnitSkipped{
					Message: c.Failure,
				}
				s.Skipped++
				totalSkipped++
			} else if !c.Passed {
				tc.Failure = &JUnitFailure{
					Message:  c.Failure,
					Contents: c.Failure,
				}
				s.Failures++
				totalFailures++
			}
			s.TestCases = append(s.TestCases, tc)
		}
		s.Tests = len(suiteCases)
		s.Time = fmt.Sprintf("%.4f", suiteTime)
		suites = append(suites, s)
	}

	ts := JUnitTestSuites{
		Suites:   suites,
		Tests:    totalTests,
		Failures: totalFailures,
		Errors:   0,
		Skipped:  totalSkipped,
		Time:     fmt.Sprintf("%.4f", totalTime),
	}

	data, err := xml.MarshalIndent(ts, "", "  ")
	if err != nil {
		return err
	}
	content := []byte(xml.Header + string(data) + "\n")
	return os.WriteFile(path, content, 0644)
}

func processCoverage(opts RunOptions, testBinary, profDir, tmpDir string) error {
	entries, err := os.ReadDir(profDir)
	if err != nil {
		return err
	}

	var profFiles []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".profraw") {
			profFiles = append(profFiles, filepath.Join(profDir, entry.Name()))
		}
	}

	if len(profFiles) == 0 {
		return fmt.Errorf("no .profraw files found in %s", profDir)
	}

	mergedProf := filepath.Join(tmpDir, "test.profdata")
	mergeArgs := append([]string{"merge", "-sparse"}, profFiles...)
	mergeArgs = append(mergeArgs, "-o", mergedProf)

	if err := exec.Command(opts.LlvmProfdata, mergeArgs...).Run(); err != nil {
		return fmt.Errorf("llvm-profdata merge failed: %w", err)
	}

	exportCmd := exec.Command(opts.LlvmCov, "export", "-format=lcov", testBinary, "-instr-profile="+mergedProf)
	var lcovOut bytes.Buffer
	exportCmd.Stdout = &lcovOut
	if err := exportCmd.Run(); err != nil {
		return fmt.Errorf("llvm-cov export failed: %w", err)
	}

	cwd, _ := os.Getwd()
	coberturaXML := LcovToCoberturaXML(lcovOut.Bytes(), cwd)

	if err := os.MkdirAll(filepath.Dir(opts.CoverageFile), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(opts.CoverageFile, coberturaXML, 0644); err != nil {
		return err
	}

	// Also write to $COVERAGE_FILE if set
	if covEnv := os.Getenv("COVERAGE_FILE"); covEnv != "" && covEnv != opts.CoverageFile {
		_ = os.MkdirAll(filepath.Dir(covEnv), 0755)
		_ = os.WriteFile(covEnv, coberturaXML, 0644)
	}

	return nil
}

type coverageLine struct {
	Number int
	Hits   int
}

type coverageFileRecord struct {
	Filename string
	Lines    []coverageLine
}

// LcovToCoberturaXML transforms LCOV data into standard Cobertura XML format.
func LcovToCoberturaXML(lcovData []byte, cwd string) []byte {
	lines := strings.Split(string(lcovData), "\n")
	cleanCwd := filepath.Clean(cwd)

	var files []coverageFileRecord
	var currentFile *coverageFileRecord

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if strings.HasPrefix(line, "SF:") {
			if currentFile != nil && len(currentFile.Lines) > 0 {
				files = append(files, *currentFile)
			}
			path := strings.TrimPrefix(line, "SF:")
			path = strings.TrimPrefix(path, "file://")

			// Ignore synthetic test runner files
			if strings.Contains(path, "__runner_main.swift") {
				currentFile = nil
				continue
			}

			// Normalize paths to repository-relative
			if idx := strings.Index(path, "._test/"); idx != -1 {
				path = path[idx+len("._test/"):]
				if strings.HasPrefix(path, "run_") {
					if slashIdx := strings.Index(path, "/"); slashIdx != -1 {
						path = path[slashIdx+1:]
					}
				}
			} else if idx := strings.Index(path, "._build/"); idx != -1 {
				path = path[idx+len("._build/"):]
			}

			if strings.HasPrefix(path, cleanCwd) {
				path = strings.TrimPrefix(path, cleanCwd)
				path = strings.TrimPrefix(path, "/")
			}
			path = strings.TrimPrefix(path, "./")

			currentFile = &coverageFileRecord{Filename: path}
		} else if strings.HasPrefix(line, "DA:") && currentFile != nil {
			da := strings.TrimPrefix(line, "DA:")
			parts := strings.Split(da, ",")
			if len(parts) >= 2 {
				lineNum, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
				hits, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err1 == nil && err2 == nil {
					currentFile.Lines = append(currentFile.Lines, coverageLine{
						Number: lineNum,
						Hits:   hits,
					})
				}
			}
		} else if line == "end_of_record" && currentFile != nil {
			if len(currentFile.Lines) > 0 {
				files = append(files, *currentFile)
			}
			currentFile = nil
		}
	}
	if currentFile != nil && len(currentFile.Lines) > 0 {
		files = append(files, *currentFile)
	}

	if len(files) == 0 {
		return []byte("<?xml version=\"1.0\" ?>\n<coverage>\n  <packages/>\n</coverage>\n")
	}

	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" ?>\n")
	sb.WriteString("<coverage>\n")
	sb.WriteString("  <packages>\n")
	sb.WriteString("    <package name=\"swift\">\n")
	sb.WriteString("      <classes>\n")

	for _, f := range files {
		sb.WriteString(fmt.Sprintf("        <class name=%q filename=%q>\n", filepath.Base(f.Filename), f.Filename))
		sb.WriteString("          <lines>\n")
		sort.Slice(f.Lines, func(i, j int) bool {
			return f.Lines[i].Number < f.Lines[j].Number
		})
		for _, l := range f.Lines {
			sb.WriteString(fmt.Sprintf("            <line number=\"%d\" hits=\"%d\"/>\n", l.Number, l.Hits))
		}
		sb.WriteString("          </lines>\n")
		sb.WriteString("        </class>\n")
	}

	sb.WriteString("      </classes>\n")
	sb.WriteString("    </package>\n")
	sb.WriteString("  </packages>\n")
	sb.WriteString("</coverage>\n")

	return []byte(sb.String())
}

func discoverDependencies(deps []string) (includeDirs []string, archives []string) {
	incSet := make(map[string]bool)
	archSet := make(map[string]bool)

	searchRoots := deps
	if len(searchRoots) == 0 {
		searchRoots = []string{"."}
	}

	for _, root := range searchRoots {
		info, err := os.Stat(root)
		if err == nil && info.IsDir() {
			if !incSet[root] {
				incSet[root] = true
				includeDirs = append(includeDirs, root)
			}
		}

		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if strings.HasSuffix(path, ".swiftmodule") {
				dir := filepath.Dir(path)
				if !incSet[dir] {
					incSet[dir] = true
					includeDirs = append(includeDirs, dir)
				}
			}
			if strings.HasSuffix(path, ".a") {
				if !archSet[path] {
					archSet[path] = true
					archives = append(archives, path)
				}
			}
			return nil
		})
	}
	return includeDirs, archives
}

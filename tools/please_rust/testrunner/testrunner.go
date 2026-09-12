package testrunner

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"tools/please_rust/toolchain"
)

// JUnitTestSuites matches standard JUnit XML output format.
type JUnitTestSuites struct {
	XMLName   xml.Name         `xml:"testsuites"`
	TestSuite []JUnitTestSuite `xml:"testsuite"`
}

type JUnitTestSuite struct {
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Time      string          `xml:"time,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

type JUnitTestCase struct {
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
}

type JUnitFailure struct {
	Message  string `xml:"message,attr"`
	Type     string `xml:"type,attr"`
	Contents string `xml:",chardata"`
}

var (
	testLineRegex = regexp.MustCompile(`test\s+([^\s]+)\s+\.\.\.\s+(ok|FAILED|ignored)`)
	summaryRegex  = regexp.MustCompile(`test result:\s+(ok|FAILED)\.\s+(\d+)\s+passed;\s+(\d+)\s+failed;\s+(\d+)\s+ignored;`)
)

// ParseTestOutput parses Rust's default test runner output into JUnit format.
func ParseTestOutput(pkgName string, output string, duration time.Duration) *JUnitTestSuites {
	suite := JUnitTestSuite{
		Name: pkgName,
		Time: fmt.Sprintf("%.3f", duration.Seconds()),
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if m := testLineRegex.FindStringSubmatch(line); len(m) >= 3 {
			tName := m[1]
			status := m[2]
			tc := JUnitTestCase{
				Name:      tName,
				Classname: pkgName,
				Time:      "0.000",
			}
			if status == "FAILED" {
				suite.Failures++
				tc.Failure = &JUnitFailure{
					Message: "Test failed",
					Type:    "Failure",
				}
			}
			suite.TestCases = append(suite.TestCases, tc)
			suite.Tests++
		}
	}

	if suite.Tests == 0 {
		suite.Tests = 1
		suite.TestCases = append(suite.TestCases, JUnitTestCase{
			Name:      "execution",
			Classname: pkgName,
			Time:      fmt.Sprintf("%.3f", duration.Seconds()),
		})
	}

	return &JUnitTestSuites{
		TestSuite: []JUnitTestSuite{suite},
	}
}

// ConvertOutputToJUnit reads raw test output from a file/string and writes test.results JUnit XML.
func ConvertOutputToJUnit(pkgName string, rawOutputFile string, resultsFile string) error {
	if resultsFile == "" {
		resultsFile = "test.results"
	}

	data, err := os.ReadFile(rawOutputFile)
	if err != nil {
		return fmt.Errorf("failed to read raw test output: %w", err)
	}

	suites := ParseTestOutput(pkgName, string(data), 0)
	xmlData, err := xml.MarshalIndent(suites, "", "  ")
	if err != nil {
		return err
	}

	xmlWithHeader := append([]byte(xml.Header), xmlData...)
	if outDir := filepath.Dir(resultsFile); outDir != "" && outDir != "." {
		os.MkdirAll(outDir, 0755)
	}

	return os.WriteFile(resultsFile, xmlWithHeader, 0644)
}

// FileCoverage represents line hit counts for a single source file.
type FileCoverage struct {
	Path     string
	LineHits map[int]int
}

// ParseLcov parses raw LCOV bytes into a slice of FileCoverage.
func ParseLcov(data []byte) ([]FileCoverage, error) {
	var files []FileCoverage
	var curFile *FileCoverage

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "SF:") {
			sfPath := strings.TrimPrefix(line, "SF:")
			curFile = &FileCoverage{
				Path:     sfPath,
				LineHits: make(map[int]int),
			}
		} else if strings.HasPrefix(line, "DA:") && curFile != nil {
			parts := strings.Split(strings.TrimPrefix(line, "DA:"), ",")
			if len(parts) >= 2 {
				lineNum, err1 := strconv.Atoi(parts[0])
				hitCount, err2 := strconv.Atoi(parts[1])
				if err1 == nil && err2 == nil {
					curFile.LineHits[lineNum] = hitCount
				}
			}
		} else if line == "end_of_record" && curFile != nil {
			files = append(files, *curFile)
			curFile = nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return files, nil
}

// NormalizeLcovPaths converts paths in FileCoverage to repository-relative paths and excludes external files.
func NormalizeLcovPaths(files []FileCoverage, repoRoot string) []FileCoverage {
	var res []FileCoverage
	for _, f := range files {
		rel := normalizePath(f.Path, repoRoot)
		if rel != "" {
			res = append(res, FileCoverage{
				Path:     rel,
				LineHits: f.LineHits,
			})
		}
	}
	return res
}

func normalizePath(rawPath string, repoRoot string) string {
	cleaned := filepath.Clean(rawPath)
	if strings.Contains(cleaned, "/rustc/") || strings.Contains(cleaned, "/.cargo/") || strings.Contains(cleaned, "/lib/rustlib/") {
		return ""
	}

	// If path contains Please build directory indicator
	if idx := strings.Index(cleaned, "._build/"); idx != -1 {
		rel := cleaned[idx+len("._build/"):]
		return rel
	}
	if idx := strings.Index(cleaned, "._test/run_"); idx != -1 {
		rest := cleaned[idx+len("._test/run_"):]
		if slashIdx := strings.Index(rest, "/"); slashIdx != -1 {
			return rest[slashIdx+1:]
		}
	}

	// Try relative to repoRoot
	if repoRoot != "" {
		absRepo, err := filepath.Abs(repoRoot)
		if err == nil {
			if strings.HasPrefix(cleaned, absRepo) {
				rel, err := filepath.Rel(absRepo, cleaned)
				if err == nil && !strings.HasPrefix(rel, "..") {
					if strings.HasPrefix(rel, "plz-out/tmp/") {
						if bIdx := strings.Index(rel, "._build/"); bIdx != -1 {
							return rel[bIdx+len("._build/"):]
						}
					}
					return rel
				}
			}
		}
	}

	// Try relative to current working directory
	if cwd, err := os.Getwd(); err == nil {
		if strings.HasPrefix(cleaned, cwd) {
			rel, err := filepath.Rel(cwd, cleaned)
			if err == nil && !strings.HasPrefix(rel, "..") {
				return rel
			}
		}
	}

	// Check if path contains plz-out or tmp, try to match subpath in repo
	if repoRoot != "" && (strings.Contains(cleaned, "plz-out") || strings.Contains(cleaned, "tmp")) {
		parts := strings.Split(cleaned, string(filepath.Separator))
		for i := 1; i < len(parts); i++ {
			candidate := filepath.Join(parts[i:]...)
			if _, err := os.Stat(filepath.Join(repoRoot, candidate)); err == nil {
				return candidate
			}
		}
	}

	// If it's already a relative path, return it cleaned
	if !filepath.IsAbs(cleaned) {
		return strings.TrimPrefix(cleaned, "./")
	}

	return strings.TrimPrefix(cleaned, "/")
}

// FormatGcov converts FileCoverage to Please-compatible GCOV format.
func FormatGcov(files []FileCoverage, repoRoot string) []byte {
	var buf bytes.Buffer
	for _, f := range files {
		fmt.Fprintf(&buf, "        -:    0:Source:%s\n", f.Path)

		lineCount := 0
		for l := range f.LineHits {
			if l > lineCount {
				lineCount = l
			}
		}

		sourcePath := f.Path
		if repoRoot != "" && !filepath.IsAbs(sourcePath) {
			sourcePath = filepath.Join(repoRoot, f.Path)
		}
		if data, err := os.ReadFile(sourcePath); err == nil {
			cnt := bytes.Count(data, []byte{'\n'})
			if len(data) > 0 && !bytes.HasSuffix(data, []byte{'\n'}) {
				cnt++
			}
			if cnt > lineCount {
				lineCount = cnt
			}
		}

		for l := 1; l <= lineCount; l++ {
			if hits, ok := f.LineHits[l]; ok {
				if hits > 0 {
					fmt.Fprintf(&buf, "        %d:  %3d:code\n", hits, l)
				} else {
					fmt.Fprintf(&buf, "    #####:  %3d:code\n", l)
				}
			} else {
				fmt.Fprintf(&buf, "        -:  %3d:code\n", l)
			}
		}
	}
	return buf.Bytes()
}

// FormatLcov formats FileCoverage into standard LCOV tracefile format.
func FormatLcov(files []FileCoverage) []byte {
	var buf bytes.Buffer
	for _, f := range files {
		fmt.Fprintf(&buf, "SF:%s\n", f.Path)
		var lines []int
		for l := range f.LineHits {
			lines = append(lines, l)
		}
		sort.Ints(lines)
		for _, l := range lines {
			fmt.Fprintf(&buf, "DA:%d,%d\n", l, f.LineHits[l])
		}
		fmt.Fprintf(&buf, "end_of_record\n")
	}
	return buf.Bytes()
}

// ProcessCoverage parses, normalizes, and generates both LCOV and GCOV coverage reports.
func ProcessCoverage(lcovData []byte, repoRoot string) ([]byte, []byte, error) {
	parsed, err := ParseLcov(lcovData)
	if err != nil {
		return nil, nil, err
	}

	normalized := NormalizeLcovPaths(parsed, repoRoot)
	normLcov := FormatLcov(normalized)
	gcovData := FormatGcov(normalized, repoRoot)
	return normLcov, gcovData, nil
}

// RunOptions configures the execution of a test binary and coverage collection.
type RunOptions struct {
	PkgName        string
	TestBinary     string
	ExtraArgs      []string
	ResultsFile    string
	CoverageActive bool
	CoverageFile   string
	LlvmProfdata   string
	LlvmCov        string
}

// RunWithOptions executes a Rust test binary with coverage support.
func RunWithOptions(opts RunOptions) error {
	resultsFile := opts.ResultsFile
	if resultsFile == "" {
		resultsFile = "test.results"
	}

	coverageActive := opts.CoverageActive || os.Getenv("COVERAGE") == "true" || os.Getenv("COVERAGE_FILE") != ""
	coverageFile := opts.CoverageFile
	if coverageFile == "" {
		coverageFile = os.Getenv("COVERAGE_FILE")
	}
	if coverageFile == "" && coverageActive {
		coverageFile = "test.coverage"
	}

	tmpDir := os.Getenv("TMP_DIR")
	if tmpDir == "" {
		tmpDir = os.Getenv("TMPDIR")
	}
	if tmpDir == "" {
		tmpDir = os.TempDir()
	}

	var cleanArgs []string
	for _, arg := range opts.ExtraArgs {
		if arg != "--" {
			cleanArgs = append(cleanArgs, arg)
		}
	}
	cmd := exec.Command(opts.TestBinary, cleanArgs...)

	var buf bytes.Buffer
	multiOut := io.MultiWriter(os.Stdout, &buf)
	multiErr := io.MultiWriter(os.Stderr, &buf)

	cmd.Stdout = multiOut
	cmd.Stderr = multiErr
	cmd.Env = os.Environ()

	if coverageActive {
		if existing, err := filepath.Glob(filepath.Join(tmpDir, "*.profraw")); err == nil {
			for _, f := range existing {
				_ = os.Remove(f)
			}
		}
		if cwdExisting, err := filepath.Glob("*.profraw"); err == nil {
			for _, f := range cwdExisting {
				_ = os.Remove(f)
			}
		}

		profPattern := filepath.Join(tmpDir, "default_%m_%p.profraw")
		var newEnv []string
		for _, e := range cmd.Env {
			if !strings.HasPrefix(e, "LLVM_PROFILE_FILE=") {
				newEnv = append(newEnv, e)
			}
		}
		cmd.Env = append(newEnv, "LLVM_PROFILE_FILE="+profPattern)
	}

	startTime := time.Now()
	runErr := cmd.Run()
	duration := time.Since(startTime)

	testSuites := ParseTestOutput(opts.PkgName, buf.String(), duration)
	xmlData, err := xml.MarshalIndent(testSuites, "", "  ")
	if err == nil {
		xmlWithHeader := append([]byte(xml.Header), xmlData...)
		if outDir := filepath.Dir(resultsFile); outDir != "" && outDir != "." {
			os.MkdirAll(outDir, 0755)
		}
		_ = os.WriteFile(resultsFile, xmlWithHeader, 0644)
	}

	if coverageActive {
		covErr := collectAndProcessCoverage(opts, tmpDir, coverageFile)
		if covErr != nil {
			fmt.Fprintf(os.Stderr, "coverage error: %v\n", covErr)
		}
	}

	return runErr
}

func collectAndProcessCoverage(opts RunOptions, tmpDir string, coverageFile string) error {
	profrawFiles, _ := filepath.Glob(filepath.Join(tmpDir, "*.profraw"))
	cwdFiles, _ := filepath.Glob("*.profraw")
	profrawFiles = append(profrawFiles, cwdFiles...)

	if len(profrawFiles) == 0 {
		return nil
	}

	profdataPath := opts.LlvmProfdata
	if profdataPath == "" {
		profdataPath = os.Getenv("TOOLS_LLVM_PROFDATA")
	}
	if profdataPath == "" {
		profdataPath = os.Getenv("LLVM_PROFDATA")
	}
	resolvedProfdata, err := toolchain.FindLlvmProfdata(profdataPath)
	if err != nil {
		return fmt.Errorf("failed to locate llvm-profdata: %w", err)
	}

	mergedProfdata := filepath.Join(tmpDir, "merged.profdata")
	mergeArgs := append([]string{"merge", "-sparse", "-o", mergedProfdata}, profrawFiles...)
	mergeCmd := exec.Command(resolvedProfdata, mergeArgs...)
	mergeCmd.Stdout = os.Stdout
	mergeCmd.Stderr = os.Stderr
	if err := mergeCmd.Run(); err != nil {
		return fmt.Errorf("llvm-profdata merge failed: %w", err)
	}

	covPath := opts.LlvmCov
	if covPath == "" {
		covPath = os.Getenv("TOOLS_LLVM_COV")
	}
	if covPath == "" {
		covPath = os.Getenv("LLVM_COV")
	}
	resolvedCov, err := toolchain.FindLlvmCov(covPath)
	if err != nil {
		return fmt.Errorf("failed to locate llvm-cov: %w", err)
	}

	exportArgs := []string{
		"export",
		"--format=lcov",
		"--instr-profile=" + mergedProfdata,
		opts.TestBinary,
		"--ignore-filename-regex=/rustc/|/.cargo/",
	}
	exportCmd := exec.Command(resolvedCov, exportArgs...)
	var lcovBuf bytes.Buffer
	var errBuf bytes.Buffer
	exportCmd.Stdout = &lcovBuf
	exportCmd.Stderr = &errBuf
	if err := exportCmd.Run(); err != nil {
		return fmt.Errorf("llvm-cov export failed: %w (stderr: %s)", err, errBuf.String())
	}

	repoRoot := os.Getenv("REPO_ROOT")
	if repoRoot == "" {
		repoRoot = os.Getenv("PLZ_REPO_ROOT")
	}
	if repoRoot == "" {
		cwd, _ := os.Getwd()
		if idx := strings.Index(cwd, "/plz-out/"); idx != -1 {
			repoRoot = cwd[:idx]
		} else {
			repoRoot = cwd
		}
	}

	normLcov, gcovData, err := ProcessCoverage(lcovBuf.Bytes(), repoRoot)
	if err != nil {
		return fmt.Errorf("failed to process coverage output: %w", err)
	}

	lcovOutPath := "test.lcov"
	if coverageFile != "" {
		lcovOutPath = filepath.Join(filepath.Dir(coverageFile), "test.lcov")
	}
	if outDir := filepath.Dir(lcovOutPath); outDir != "" && outDir != "." {
		_ = os.MkdirAll(outDir, 0755)
	}
	_ = os.WriteFile(lcovOutPath, normLcov, 0644)

	if coverageFile != "" {
		if outDir := filepath.Dir(coverageFile); outDir != "" && outDir != "." {
			_ = os.MkdirAll(outDir, 0755)
		}
		dataToWrite := gcovData
		if os.Getenv("RUST_COVERAGE_FORMAT") == "lcov" {
			dataToWrite = normLcov
		}
		if err := os.WriteFile(coverageFile, dataToWrite, 0644); err != nil {
			return fmt.Errorf("failed to write coverage file %s: %w", coverageFile, err)
		}
	}

	return nil
}

// Run executes a Rust test binary, streaming stdout/stderr while capturing output for JUnit results.
func Run(pkgName string, testBinary string, extraArgs []string, resultsFile string) error {
	return RunWithOptions(RunOptions{
		PkgName:     pkgName,
		TestBinary:  testBinary,
		ExtraArgs:   extraArgs,
		ResultsFile: resultsFile,
	})
}


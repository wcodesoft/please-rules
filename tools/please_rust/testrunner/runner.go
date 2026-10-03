package testrunner

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"tools/please_rust/toolchain"
)

// RunOptions configures the execution of a test binary and coverage collection.
type RunOptions struct {
	PkgName        string
	TestBinary     string
	ExtraArgs      []string
	ResultsFile    string
	CoverageActive bool
	CoverageFile   string
	LcovFile       string // raw lcov export (functions and branches), written empty without coverage
	LlvmProfdata   string
	LlvmCov        string
}

func resolveResultsFile(path string) string {
	if path == "" {
		return DefaultResultsFile
	}
	return path
}

func resolveCoverageConfig(opts RunOptions) (bool, string) {
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

func resolveTmpDir() string {
	if dir := os.Getenv("TMP_DIR"); dir != "" {
		return dir
	}
	if dir := os.Getenv("TMPDIR"); dir != "" {
		return dir
	}
	return os.TempDir()
}

// findProfrawFiles lists the raw profiles in tmpDir and the working directory, once
// each. Under Please the two are the same directory, and merging a profile twice
// doubles its execution counts.
func findProfrawFiles(tmpDir string) []string {
	var files []string
	seen := make(map[string]bool)
	for _, pattern := range []string{filepath.Join(tmpDir, "*.profraw"), "*.profraw"} {
		matches, _ := filepath.Glob(pattern)
		for _, m := range matches {
			key := m
			if abs, err := filepath.Abs(m); err == nil {
				key = abs
			}
			if !seen[key] {
				seen[key] = true
				files = append(files, m)
			}
		}
	}
	return files
}

func cleanProfrawFiles(patterns ...string) {
	for _, pat := range patterns {
		if files, err := filepath.Glob(pat); err == nil {
			for _, f := range files {
				_ = os.Remove(f)
			}
		}
	}
}

func prepareCommand(opts RunOptions, tmpDir string, coverageActive bool, captureBuf *bytes.Buffer) *exec.Cmd {
	var cleanArgs []string
	for _, arg := range opts.ExtraArgs {
		if arg != "--" {
			cleanArgs = append(cleanArgs, arg)
		}
	}
	cmd := exec.Command(opts.TestBinary, cleanArgs...)
	cmd.Stdout = io.MultiWriter(os.Stdout, captureBuf)
	cmd.Stderr = io.MultiWriter(os.Stderr, captureBuf)
	cmd.Env = os.Environ()

	if coverageActive {
		cleanProfrawFiles(filepath.Join(tmpDir, "*.profraw"), "*.profraw")
		profPattern := filepath.Join(tmpDir, "default_%m_%p.profraw")
		var newEnv []string
		for _, e := range cmd.Env {
			if !strings.HasPrefix(e, "LLVM_PROFILE_FILE=") {
				newEnv = append(newEnv, e)
			}
		}
		cmd.Env = append(newEnv, "LLVM_PROFILE_FILE="+profPattern)
	}

	return cmd
}

// RunWithOptions executes a Rust test binary with coverage support.
func RunWithOptions(opts RunOptions) error {
	resultsFile := resolveResultsFile(opts.ResultsFile)
	coverageActive, coverageFile := resolveCoverageConfig(opts)
	tmpDir := resolveTmpDir()

	// The raw lcov export is a declared test output: create it up front so it exists
	// (empty) when coverage is off or the report cannot be produced.
	if err := writeRawLcov(opts.LcovFile, nil); err != nil {
		return fmt.Errorf("failed to reset lcov file: %w", err)
	}

	var buf bytes.Buffer
	cmd := prepareCommand(opts, tmpDir, coverageActive, &buf)

	startTime := time.Now()
	runErr := cmd.Run()
	duration := time.Since(startTime)

	testSuites := ParseTestOutput(opts.PkgName, buf.String(), duration)
	_ = writeJUnitResults(testSuites, resultsFile)

	if coverageActive {
		if covErr := collectAndProcessCoverage(opts, tmpDir, coverageFile); covErr != nil {
			fmt.Fprintf(os.Stderr, "coverage error: %v\n", covErr)
		}
	}

	return runErr
}

func mergeProfdata(profdataPath, tmpDir string, profrawFiles []string) (string, error) {
	if profdataPath == "" {
		profdataPath = os.Getenv("TOOLS_LLVM_PROFDATA")
	}
	if profdataPath == "" {
		profdataPath = os.Getenv("LLVM_PROFDATA")
	}
	resolved, err := toolchain.FindLlvmProfdata(profdataPath)
	if err != nil {
		return "", fmt.Errorf("failed to locate llvm-profdata: %w", err)
	}

	mergedPath := filepath.Join(tmpDir, "merged.profdata")
	args := append([]string{"merge", "-sparse", "-o", mergedPath}, profrawFiles...)
	cmd := exec.Command(resolved, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		version, _ := exec.Command(resolved, "--version").CombinedOutput()
		return "", fmt.Errorf("llvm-profdata merge failed using %s (%s): %w", resolved, strings.Join(strings.Fields(string(version)), " "), err)
	}
	return mergedPath, nil
}

func exportLcov(covPath, testBinary, mergedProfdata string) ([]byte, error) {
	return exportCoverage("lcov", covPath, testBinary, mergedProfdata)
}

// exportCoverage runs `llvm-cov export` in the given format ("lcov" or "text" for
// the JSON export).
func exportCoverage(format, covPath, testBinary, mergedProfdata string) ([]byte, error) {
	if covPath == "" {
		covPath = os.Getenv("TOOLS_LLVM_COV")
	}
	if covPath == "" {
		covPath = os.Getenv("LLVM_COV")
	}
	resolved, err := toolchain.FindLlvmCov(covPath)
	if err != nil {
		return nil, fmt.Errorf("failed to locate llvm-cov: %w", err)
	}

	args := []string{
		"export",
		"--format=" + format,
		"--instr-profile=" + mergedProfdata,
		testBinary,
		"--ignore-filename-regex=/rustc/|/.cargo/",
	}
	cmd := exec.Command(resolved, args...)
	var lcovBuf, errBuf bytes.Buffer
	cmd.Stdout = &lcovBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("llvm-cov export failed: %w (stderr: %s)", err, errBuf.String())
	}
	return lcovBuf.Bytes(), nil
}

// exportRawLcov writes the raw lcov report. The JSON export adds branch records for
// partly executed lines; if it fails the report is still written without them.
func exportRawLcov(opts RunOptions, lcovBytes []byte, mergedProfdata, repoRoot string) error {
	llvmJSON, err := exportCoverage("text", opts.LlvmCov, opts.TestBinary, mergedProfdata)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: no sub-line regions for the raw lcov export: %v\n", err)
		llvmJSON = nil
	}
	data, err := buildRawLcov(lcovBytes, llvmJSON, repoRoot)
	if err != nil {
		return err
	}
	return writeRawLcov(opts.LcovFile, data)
}

func detectRepoRoot() string {
	if root := os.Getenv("REPO_ROOT"); root != "" {
		return root
	}
	if root := os.Getenv("PLZ_REPO_ROOT"); root != "" {
		return root
	}
	cwd, _ := os.Getwd()
	if idx := strings.Index(cwd, "/plz-out/"); idx != -1 {
		return cwd[:idx]
	}
	return cwd
}

func writeCoverageFiles(normLcov, gcovData []byte, coverageFile string) error {
	lcovOutPath := "test.lcov"
	if coverageFile != "" {
		lcovOutPath = filepath.Join(filepath.Dir(coverageFile), "test.lcov")
	}
	_ = ensureDir(lcovOutPath)
	_ = os.WriteFile(lcovOutPath, normLcov, 0644)

	if coverageFile == "" {
		return nil
	}
	if err := ensureDir(coverageFile); err != nil {
		return fmt.Errorf("creating coverage dir: %w", err)
	}
	dataToWrite := gcovData
	if os.Getenv("RUST_COVERAGE_FORMAT") == "lcov" {
		dataToWrite = normLcov
	}
	if err := os.WriteFile(coverageFile, dataToWrite, 0644); err != nil {
		return fmt.Errorf("failed to write coverage file %s: %w", coverageFile, err)
	}
	return nil
}

func collectAndProcessCoverage(opts RunOptions, tmpDir string, coverageFile string) error {
	profrawFiles := findProfrawFiles(tmpDir)

	if len(profrawFiles) == 0 {
		return nil
	}

	mergedProfdata, err := mergeProfdata(opts.LlvmProfdata, tmpDir, profrawFiles)
	if err != nil {
		return err
	}

	lcovBytes, err := exportLcov(opts.LlvmCov, opts.TestBinary, mergedProfdata)
	if err != nil {
		return err
	}

	repoRoot := detectRepoRoot()
	normLcov, gcovData, err := ProcessCoverage(lcovBytes, repoRoot)
	if err != nil {
		return fmt.Errorf("failed to process coverage output: %w", err)
	}

	if opts.LcovFile != "" {
		if err := exportRawLcov(opts, lcovBytes, mergedProfdata, repoRoot); err != nil {
			fmt.Fprintf(os.Stderr, "raw lcov export error: %v\n", err)
		}
	}

	return writeCoverageFiles(normLcov, gcovData, coverageFile)
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

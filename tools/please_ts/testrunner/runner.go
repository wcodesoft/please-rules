package testrunner

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"tools/please_ts/importmap"
)

// RunOptions configures test execution.
type RunOptions struct {
	Deno          string
	Runner        string // "deno" | "vitest"
	Srcs          []string
	Deps          []string
	ModuleName    string
	ResultsFile   string
	Coverage      bool
	CoverageFile  string
	Browser       string
	BrowserBinary string
	ExtraArgs     []string
}

// Run executes the tests and generates JUnit XML and optional coverage reports.
func Run(opts RunOptions) error {
	if len(opts.Srcs) == 0 {
		return fmt.Errorf("no test source files specified")
	}

	resultsFile := opts.ResultsFile
	if resultsFile == "" {
		resultsFile = "test.results"
	}
	// Ensure results file dir exists
	if err := os.MkdirAll(filepath.Dir(resultsFile), 0755); err != nil {
		return err
	}

	if opts.Runner == "vitest" {
		return runVitest(opts, resultsFile)
	}

	return runDeno(opts, resultsFile)
}

func runDeno(opts RunOptions, resultsFile string) error {
	denoBin := opts.Deno
	if denoBin == "" {
		denoBin = "deno"
	}

	coverageActive, coverageFile := resolveCoverage(opts)

	tmpDir, err := os.MkdirTemp("", "please_ts_test_*")
	if err != nil {
		return fmt.Errorf("failed creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	denoCacheDir := filepath.Join(tmpDir, ".deno_cache")
	if err := os.MkdirAll(denoCacheDir, 0755); err != nil {
		return fmt.Errorf("failed creating DENO_DIR: %w", err)
	}

	// 1. Synthesize target-local import map
	importMapPath := ".import_map.json"
	im, err := importmap.Synthesize(opts.ModuleName, opts.Srcs, opts.Deps, ".")
	if err != nil {
		return fmt.Errorf("failed synthesizing import map: %w", err)
	}
	if err := im.WriteToFile(importMapPath); err != nil {
		return fmt.Errorf("failed writing import map: %w", err)
	}
	defer os.Remove(importMapPath)

	// 2. Prepare test arguments
	var resolvedSrcs []string
	for _, s := range opts.Srcs {
		if _, err := os.Stat(s); err == nil {
			resolvedSrcs = append(resolvedSrcs, s)
			continue
		}
		found := false
		_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			if !found && !info.IsDir() && (path == s || filepath.Base(path) == s || strings.HasSuffix(path, "/"+s)) {
				resolvedSrcs = append(resolvedSrcs, path)
				found = true
			}
			return nil
		})
		if !found {
			resolvedSrcs = append(resolvedSrcs, s)
		}
	}

	args := []string{
		"test",
		"--no-remote",
		"--import-map", importMapPath,
		"--junit-path", resultsFile,
	}

	covDir := ""
	if coverageActive {
		covDir = filepath.Join(tmpDir, "cov_profile")
		args = append(args, "--coverage="+covDir)
	}

	args = append(args, opts.ExtraArgs...)
	args = append(args, resolvedSrcs...)

	cmd := exec.Command(denoBin, args...)
	cmd.Env = append(os.Environ(), "DENO_DIR="+denoCacheDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	testErr := cmd.Run()

	// 3. Ensure JUnit XML exists even if test failed early
	if _, err := os.Stat(resultsFile); err != nil {
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, testErr)
	}

	// 4. Generate coverage report if requested
	if coverageActive && coverageFile != "" && covDir != "" {
		_ = generateDenoCoverage(denoBin, covDir, coverageFile, denoCacheDir)
	}

	if testErr != nil {
		return fmt.Errorf("tests failed: %w", testErr)
	}
	return nil
}

func runVitest(opts RunOptions, resultsFile string) error {
	args := []string{
		"vitest",
		"run",
		"--reporter=junit",
		"--outputFile=" + resultsFile,
	}

	if opts.Browser != "" {
		args = append(args, "--browser.name="+opts.Browser, "--browser.headless")
	}

	if opts.Coverage && opts.CoverageFile != "" {
		args = append(args, "--coverage.enabled", "--coverage.reporter=lcov", "--coverage.reportsDirectory="+filepath.Dir(opts.CoverageFile))
	}

	args = append(args, opts.ExtraArgs...)
	args = append(args, opts.Srcs...)

	cmd := exec.Command(args[0], args[1:]...)
	env := os.Environ()
	if opts.BrowserBinary != "" {
		env = append(env, "PLAYWRIGHT_CHROMIUM_EXECUTABLE_PATH="+opts.BrowserBinary)
	}
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	testErr := cmd.Run()
	if _, err := os.Stat(resultsFile); err != nil {
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, testErr)
	}
	return testErr
}

func resolveCoverage(opts RunOptions) (bool, string) {
	active := opts.Coverage || os.Getenv("COVERAGE") == "true" || os.Getenv("COVERAGE_FILE") != ""
	file := opts.CoverageFile
	if file == "" {
		file = os.Getenv("COVERAGE_FILE")
	}
	if file == "" && active {
		file = "test.coverage"
	}
	return active, file
}

type coverageLine struct {
	Number int
	Hits   int
}

type coverageFileRecord struct {
	Filename string
	Lines    []coverageLine
}

func generateDenoCoverage(denoBin, covDir, outputFile, denoCacheDir string) error {
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return err
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(denoBin, "coverage", "--lcov", covDir)
	cmd.Env = append(os.Environ(), "DENO_DIR="+denoCacheDir)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		_ = os.WriteFile(outputFile, []byte("<?xml version=\"1.0\" ?>\n<coverage>\n  <packages/>\n</coverage>\n"), 0644)
		return fmt.Errorf("deno coverage (%v): %s", err, stderr.String())
	}

	cwd, _ := os.Getwd()
	xmlData := lcovToCoberturaXML(stdout.Bytes(), cwd)
	return os.WriteFile(outputFile, xmlData, 0644)
}

func lcovToCoberturaXML(lcovData []byte, cwd string) []byte {
	lines := strings.Split(string(lcovData), "\n")
	cleanCwd := filepath.Clean(cwd)

	var files []coverageFileRecord
	var currentFile *coverageFileRecord

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if strings.HasPrefix(line, "SF:") {
			if currentFile != nil {
				files = append(files, *currentFile)
			}
			path := strings.TrimPrefix(line, "SF:")
			path = strings.TrimPrefix(path, "file://")
			if strings.HasPrefix(path, cleanCwd) {
				path = strings.TrimPrefix(path, cleanCwd)
				path = strings.TrimPrefix(path, "/")
			}
			path = strings.TrimPrefix(path, "./")

			dir := filepath.Dir(path)
			base := filepath.Base(path)
			parentDir := filepath.Dir(dir)
			if parentDir != "." && parentDir != "" {
				if filepath.Base(dir) == strings.TrimSuffix(base, filepath.Ext(base)) {
					path = filepath.Join(parentDir, base)
				} else if _, err := os.Stat(filepath.Join(dir, "ts_metadata.json")); err == nil {
					path = filepath.Join(parentDir, base)
				}
			}

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
			files = append(files, *currentFile)
			currentFile = nil
		}
	}
	if currentFile != nil {
		files = append(files, *currentFile)
	}

	if len(files) == 0 {
		return []byte("<?xml version=\"1.0\" ?>\n<coverage>\n  <packages/>\n</coverage>\n")
	}

	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" ?>\n")
	sb.WriteString("<coverage>\n")
	sb.WriteString("  <packages>\n")
	sb.WriteString("    <package name=\"ts\">\n")
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

func writeFallbackJUnit(resultsFile string, srcs []string, testErr error) error {
	msg := "test execution finished"
	failureXML := ""
	if testErr != nil {
		msg = testErr.Error()
		failureXML = fmt.Sprintf("\n    <failure message=%q>%s</failure>", msg, msg)
	}

	xmlContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<testsuites name="TypeScript Tests">
  <testsuite name="ts_test" tests="%d" failures="%d" errors="0">
    <testcase name="execution" classname="ts_test">%s
    </testcase>
  </testsuite>
</testsuites>
`, len(srcs), boolToInt(testErr != nil), failureXML)

	return os.WriteFile(resultsFile, []byte(xmlContent), 0644)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

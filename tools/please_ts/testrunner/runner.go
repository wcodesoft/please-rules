package testrunner

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"tools/please_ts/bundle"
	"tools/please_ts/importmap"
)

// RunOptions configures test execution.
type RunOptions struct {
	Deno          string
	Runner        string // "deno" | "vitest" | "browser"
	Srcs          []string
	Deps          []string
	ModuleName    string
	ResultsFile   string
	Coverage      bool
	CoverageFile  string
	Browser       string
	BrowserBinary string
	VitestDir     string
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

	if opts.Browser != "" && opts.Runner != "vitest" {
		return runBrowserTest(opts, resultsFile)
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
	tmpDir, err := os.MkdirTemp("", "please_ts_vitest_*")
	if err != nil {
		return fmt.Errorf("failed creating temp dir for vitest: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 1. Synthesize target-local import map for alias resolution
	im, err := importmap.Synthesize(opts.ModuleName, opts.Srcs, opts.Deps, ".")
	if err != nil {
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, err)
		return fmt.Errorf("failed synthesizing import map: %w", err)
	}

	aliasMap := make(map[string]string)
	if im != nil {
		for k, target := range im.Imports {
			if strings.HasSuffix(k, "/") {
				continue
			}
			if k == "vitest" || strings.HasPrefix(k, "vitest/") || k == "chai" || strings.HasPrefix(k, "chai/") {
				continue
			}
			tClean := strings.TrimSuffix(target, "/")
			if tClean != "" {
				absTarget, err := filepath.Abs(tClean)
				if err == nil {
					aliasMap[k] = absTarget
				} else {
					aliasMap[k] = tClean
				}
			}
		}
	}

	// Resolve sources relative to current working directory
	var resolvedSrcs []string
	for _, s := range opts.Srcs {
		found := false
		_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			if !found && !info.IsDir() && (path == s || filepath.Base(path) == s || strings.HasSuffix(path, "/"+s)) {
				absPath, err := filepath.Abs(path)
				if err == nil {
					resolvedSrcs = append(resolvedSrcs, absPath)
				} else {
					resolvedSrcs = append(resolvedSrcs, path)
				}
				found = true
			}
			return nil
		})
		if !found {
			absPath, err := filepath.Abs(s)
			if err == nil {
				resolvedSrcs = append(resolvedSrcs, absPath)
			} else {
				resolvedSrcs = append(resolvedSrcs, s)
			}
		}
	}

	aliasBytes, _ := json.MarshalIndent(aliasMap, "    ", "  ")
	srcsBytes, _ := json.Marshal(resolvedSrcs)

	configPath := filepath.Join(tmpDir, "vitest.config.mjs")
	configContent := fmt.Sprintf(`export default {
  test: {
    globals: true,
    include: %s,
    watch: false,
  },
  resolve: {
    alias: %s,
  },
};
`, string(srcsBytes), string(aliasBytes))

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, err)
		return fmt.Errorf("failed writing vitest config: %w", err)
	}

	absResults, err := filepath.Abs(resultsFile)
	if err != nil {
		absResults = resultsFile
	}

	args := []string{
		"run",
		"--config", configPath,
		"--reporter=junit",
		"--outputFile=" + absResults,
	}

	if opts.Browser != "" {
		args = append(args, "--browser.name="+opts.Browser, "--browser.headless")
	}

	coverageActive, coverageFile := resolveCoverage(opts)
	if coverageActive && coverageFile != "" {
		args = append(args, "--coverage.enabled", "--coverage.reporter=lcov", "--coverage.reportsDirectory="+filepath.Dir(coverageFile))
	}

	args = append(args, opts.ExtraArgs...)
	args = append(args, resolvedSrcs...)

	denoBin := opts.Deno
	if denoBin == "" {
		denoBin = "deno"
	}
	denoArgs := []string{"run"}
	if opts.VitestDir != "" {
		denoArgs = append(denoArgs, "--no-remote")
	}
	denoArgs = append(denoArgs, "-A", "npm:vitest")
	denoArgs = append(denoArgs, args...)
	cmd := exec.Command(denoBin, denoArgs...)

	env := os.Environ()
	if opts.VitestDir != "" {
		if absV, err := filepath.Abs(opts.VitestDir); err == nil {
			env = append(env, "DENO_DIR="+absV)
		} else {
			env = append(env, "DENO_DIR="+opts.VitestDir)
		}
	}
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

type browserTestResult struct {
	Name     string  `json:"name"`
	Duration float64 `json:"duration"`
	Error    string  `json:"error"`
}

func runBrowserTest(opts RunOptions, resultsFile string) error {
	browserBin := opts.BrowserBinary
	if browserBin == "" {
		return fmt.Errorf("no browser binary specified for browser test")
	}

	tmpDir, err := os.MkdirTemp("", "please_ts_browser_test_*")
	if err != nil {
		return fmt.Errorf("failed creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 1. Bundle test files for the browser environment
	bundleOut := filepath.Join(tmpDir, "test_bundle.js")
	mainSrc := opts.Srcs[0]
	bundleOpts := bundle.Options{
		Deno:       opts.Deno,
		Out:        bundleOut,
		Main:       mainSrc,
		Srcs:       opts.Srcs,
		Deps:       opts.Deps,
		ModuleName: opts.ModuleName,
		Format:     "iife",
	}
	if err := bundle.Run(bundleOpts); err != nil {
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, err)
		return fmt.Errorf("failed bundling browser test: %w", err)
	}

	bundleBytes, err := os.ReadFile(bundleOut)
	if err != nil {
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, err)
		return err
	}

	profileDir := filepath.Join(tmpDir, "profile")
	_ = os.MkdirAll(profileDir, 0755)

	htmlPath := filepath.Join(tmpDir, "index.html")
	_ = os.WriteFile(htmlPath, []byte("<!DOCTYPE html><html><head><meta charset=\"utf-8\"></head><body></body></html>"), 0644)

	// 2. Launch headless Chromium
	browserCmd := exec.Command(browserBin,
		"--headless",
		"--no-sandbox",
		"--disable-gpu",
		"--disable-dev-shm-usage",
		"--remote-debugging-port=0",
		"--user-data-dir="+profileDir,
		"--allow-file-access-from-files",
		"--disable-web-security",
		"file://"+htmlPath,
	)
	stderr, err := browserCmd.StderrPipe()
	if err != nil {
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, err)
		return fmt.Errorf("failed opening browser stderr pipe: %w", err)
	}

	if err := browserCmd.Start(); err != nil {
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, err)
		return fmt.Errorf("failed starting browser %s: %w", browserBin, err)
	}
	defer func() {
		_ = browserCmd.Process.Kill()
		_ = browserCmd.Wait()
	}()

	// Read stderr to capture DevTools WebSocket URL
	wsURLChan := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if idx := strings.Index(line, "DevTools listening on ws://"); idx != -1 {
				wsURLChan <- strings.TrimSpace(line[idx+len("DevTools listening on "):])
				return
			}
		}
	}()

	var wsURL string
	select {
	case wsURL = <-wsURLChan:
	case <-time.After(15 * time.Second):
		err := fmt.Errorf("timeout waiting for browser DevTools WebSocket")
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, err)
		return err
	}

	// Query /json/list for target page WebSocket
	httpURL := strings.Replace(wsURL, "ws://", "http://", 1)
	slashIdx := strings.Index(httpURL[7:], "/")
	base := httpURL[:7+slashIdx]
	resp, err := http.Get(base + "/json/list")
	if err != nil {
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, err)
		return fmt.Errorf("failed querying browser targets: %w", err)
	}
	defer resp.Body.Close()

	var targets []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil || len(targets) == 0 {
		err := fmt.Errorf("no browser targets found")
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, err)
		return err
	}

	pageWS, _ := targets[0]["webSocketDebuggerUrl"].(string)
	if pageWS == "" {
		err := fmt.Errorf("no webSocketDebuggerUrl found for page target")
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, err)
		return err
	}

	client, err := DialCDP(pageWS)
	if err != nil {
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, err)
		return fmt.Errorf("failed dialing CDP: %w", err)
	}
	defer client.Close()

	_, _ = client.Send("Runtime.enable", nil)
	_, _ = client.Send("Page.enable", nil)

	// 3. Inject test harness into browser
	harnessJS := `(() => {
		window.__TESTS__ = [];
		window.Deno = window.Deno || {};
		window.Deno.test = function(nameOrObj, fn) {
			if (typeof nameOrObj === 'object') {
				window.__TESTS__.push({ name: nameOrObj.name, fn: fn || nameOrObj.fn });
			} else {
				window.__TESTS__.push({ name: nameOrObj, fn: fn });
			}
		};
		window.test = window.Deno.test;
		window.it = window.test;
		window.describe = function(name, fn) { fn(); };
		window.expect = function(actual) {
			return {
				toBe: function(expected) {
					if (actual !== expected) throw new Error('Expected ' + expected + ' but got ' + actual);
				},
				toEqual: function(expected) {
					if (JSON.stringify(actual) !== JSON.stringify(expected)) throw new Error('Expected ' + JSON.stringify(expected) + ' but got ' + JSON.stringify(actual));
				},
				toBeTruthy: function() {
					if (!actual) throw new Error('Expected truthy but got ' + actual);
				},
				toBeFalsy: function() {
					if (actual) throw new Error('Expected falsy but got ' + actual);
				}
			};
		};
	})()`
	if _, err := client.Evaluate(harnessJS); err != nil {
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, err)
		return fmt.Errorf("failed injecting test harness: %w", err)
	}

	// 4. Inject bundled test script
	if _, err := client.Evaluate(string(bundleBytes)); err != nil {
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, err)
		return fmt.Errorf("failed evaluating test bundle: %w", err)
	}

	// 5. Run registered tests and collect results
	runnerJS := `(async () => {
		const results = [];
		for (const t of window.__TESTS__) {
			const start = performance.now();
			let err = null;
			try {
				await t.fn();
			} catch (e) {
				err = (e && e.stack) ? e.stack : String(e);
			}
			const duration = (performance.now() - start) / 1000;
			results.push({ name: t.name, duration: duration, error: err || "" });
		}
		return results;
	})()`

	evalRes, err := client.Evaluate(runnerJS)
	if err != nil {
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, err)
		return fmt.Errorf("failed running browser tests: %w", err)
	}

	var resPayload struct {
		Result struct {
			Value []browserTestResult `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(evalRes, &resPayload); err != nil {
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, err)
		return fmt.Errorf("failed parsing browser test results: %w", err)
	}

	results := resPayload.Result.Value
	return writeBrowserJUnit(resultsFile, opts.Srcs, results)
}

func writeBrowserJUnit(resultsFile string, srcs []string, results []browserTestResult) error {
	var totalDuration float64
	failures := 0
	for _, r := range results {
		totalDuration += r.Duration
		if r.Error != "" {
			failures++
		}
	}

	suiteName := filepath.Base(srcs[0])
	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	sb.WriteString(fmt.Sprintf("<testsuites name=\"%s\" tests=\"%d\" failures=\"%d\" errors=\"0\" time=\"%.4f\">\n",
		escapeXML(suiteName), len(results), failures, totalDuration))
	sb.WriteString(fmt.Sprintf("  <testsuite name=\"%s\" tests=\"%d\" failures=\"%d\" errors=\"0\" time=\"%.4f\">\n",
		escapeXML(suiteName), len(results), failures, totalDuration))

	for _, r := range results {
		if r.Error != "" {
			fmt.Printf("FAIL: %s (%.4fs)\n%s\n", r.Name, r.Duration, r.Error)
			sb.WriteString(fmt.Sprintf("    <testcase name=\"%s\" classname=\"%s\" time=\"%.4f\">\n",
				escapeXML(r.Name), escapeXML(suiteName), r.Duration))
			sb.WriteString(fmt.Sprintf("      <failure message=\"test failed\">%s</failure>\n",
				escapeXML(r.Error)))
			sb.WriteString("    </testcase>\n")
		} else {
			fmt.Printf("PASS: %s (%.4fs)\n", r.Name, r.Duration)
			sb.WriteString(fmt.Sprintf("    <testcase name=\"%s\" classname=\"%s\" time=\"%.4f\" />\n",
				escapeXML(r.Name), escapeXML(suiteName), r.Duration))
		}
	}

	sb.WriteString("  </testsuite>\n")
	sb.WriteString("</testsuites>\n")

	if err := os.WriteFile(resultsFile, []byte(sb.String()), 0644); err != nil {
		return err
	}

	if failures > 0 {
		return fmt.Errorf("%d browser tests failed", failures)
	}
	return nil
}

func escapeXML(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
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

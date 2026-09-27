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
	"strings"
	"time"

	"tools/please_ts/bundle"
)

type browserTestResult struct {
	Name     string  `json:"name"`
	Duration float64 `json:"duration"`
	Error    string  `json:"error"`
}

func (opts RunOptions) runBrowserTest(resultsFile string) error {
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

	// Ensure DOM is fully loaded and document.body exists
	readyJS := `(async () => {
		for (let i = 0; i < 200; i++) {
			if (document.body) {
				return true;
			}
			await new Promise(r => setTimeout(r, 25));
		}
		if (!document.body) {
			if (!document.documentElement) {
				document.appendChild(document.createElement('html'));
			}
			document.documentElement.appendChild(document.createElement('body'));
		}
		return !!document.body;
	})()`
	if _, err := client.Evaluate(readyJS); err != nil {
		_ = writeFallbackJUnit(resultsFile, opts.Srcs, err)
		return fmt.Errorf("failed preparing browser DOM: %w", err)
	}

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
		for (let i = 0; i < 200 && !document.body; i++) {
			await new Promise(r => setTimeout(r, 25));
		}
		if (!document.body && document.documentElement) {
			document.documentElement.appendChild(document.createElement('body'));
		}
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

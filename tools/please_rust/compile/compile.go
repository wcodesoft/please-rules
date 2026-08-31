package compile

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/please-build/rust-rules/tools/please_rust/toolchain"
)

// Options specifies the parameters needed to compile a Rust target.
type Options struct {
	Out       string
	CrateName string
	CrateType string
	Edition   string
	MainSrc   string
	Version   string
	Flags     string
	Rustc     string
	Inputs    []string
}

// sanitizeCrateName converts hyphenated crate names to underscores for rustc extern/crate identification.
func sanitizeCrateName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

// extractCrateName derives the crate name from a library artifact filename.
func extractCrateName(filename string) string {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	trimmed := strings.TrimSuffix(base, ext)
	trimmed = strings.TrimPrefix(trimmed, "lib")
	return sanitizeCrateName(trimmed)
}

// resolveMainSrc finds the actual entrypoint source file among inputs and current directory.
func resolveMainSrc(mainSrc string, crateType string, inputs []string) string {
	// If mainSrc explicitly exists as a file, return it
	if mainSrc != "" {
		if _, err := os.Stat(mainSrc); err == nil {
			return mainSrc
		}
		for _, input := range inputs {
			if input == mainSrc || filepath.Base(input) == filepath.Base(mainSrc) || strings.HasSuffix(input, mainSrc) {
				if _, err := os.Stat(input); err == nil {
					return input
				}
			}
		}
	}

	// Look for standard root/package level files first: lib.rs, main.rs, then src/lib.rs, src/main.rs
	preferredNames := []string{"lib.rs", "main.rs", "src/lib.rs", "src/main.rs"}
	if crateType == "bin" {
		preferredNames = []string{"main.rs", "src/main.rs", "lib.rs", "src/lib.rs"}
	}

	for _, pref := range preferredNames {
		if _, err := os.Stat(pref); err == nil {
			return pref
		}
		for _, input := range inputs {
			if input == pref || filepath.Base(input) == pref || strings.HasSuffix(input, pref) {
				if _, err := os.Stat(input); err == nil {
					return input
				}
			}
		}
	}

	// If only one .rs file exists in inputs, use it directly
	for _, input := range inputs {
		if filepath.Ext(input) == ".rs" {
			if _, err := os.Stat(input); err == nil {
				return input
			}
		}
	}

	// Check any .rs file in the working directory
	if entries, err := os.ReadDir("."); err == nil {
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".rs" {
				return e.Name()
			}
		}
	}

	return mainSrc
}

// discoverDepFiles scans directories in inputs or the current build directory for .rlib and .so files.
func discoverDepFiles(inputs []string) []string {
	seen := make(map[string]bool)
	var deps []string

	addDep := func(p string) {
		if !seen[p] {
			seen[p] = true
			deps = append(deps, p)
		}
	}

	for _, input := range inputs {
		ext := filepath.Ext(input)
		if ext == ".rlib" || ext == ".so" || ext == ".dylib" || ext == ".dll" {
			addDep(input)
		}
	}

	_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if !info.IsDir() {
			ext := filepath.Ext(path)
			if ext == ".rlib" || ext == ".so" || ext == ".dylib" || ext == ".dll" {
				addDep(path)
			}
		}
		return nil
	})

	return deps
}

// BuildRustcArgs assembles the arguments for invoking rustc.
func BuildRustcArgs(opts Options, realBinaryOut string) ([]string, error) {
	if opts.CrateName == "" {
		return nil, fmt.Errorf("crate name (--crate-name) is required")
	}
	if opts.Out == "" {
		return nil, fmt.Errorf("output path (--out) is required")
	}

	mainFile := resolveMainSrc(opts.MainSrc, opts.CrateType, opts.Inputs)
	if mainFile == "" {
		return nil, fmt.Errorf("main source file could not be determined")
	}

	args := []string{}

	if opts.Edition != "" {
		args = append(args, "--edition", opts.Edition)
	}

	crateType := opts.CrateType
	if crateType == "test" {
		args = append(args, "--test")
	} else if crateType != "" {
		args = append(args, "--crate-type", crateType)
	}

	args = append(args, "--crate-name", sanitizeCrateName(opts.CrateName))

	outPath := opts.Out
	if crateType == "test" && realBinaryOut != "" {
		outPath = realBinaryOut
	}
	args = append(args, "-o", outPath)

	allDeps := discoverDepFiles(opts.Inputs)

	searchDirs := make(map[string]bool)
	type externDef struct {
		crate string
		path  string
	}
	var externs []externDef
	externMap := make(map[string]string)

	for _, depPath := range allDeps {
		dir := filepath.Dir(depPath)
		if dir != "" && dir != "." {
			searchDirs[dir] = true
		}
		cName := extractCrateName(depPath)
		if cName != "" && cName != sanitizeCrateName(opts.CrateName) {
			if _, exists := externMap[cName]; !exists {
				externMap[cName] = depPath
				externs = append(externs, externDef{crate: cName, path: depPath})
			}
		}
	}

	for dir := range searchDirs {
		args = append(args, "-L", fmt.Sprintf("dependency=%s", dir))
	}
	args = append(args, "-L", "dependency=.")

	for _, ext := range externs {
		args = append(args, "--extern", fmt.Sprintf("%s=%s", ext.crate, ext.path))
	}

	if opts.Flags != "" {
		fields := strings.Fields(opts.Flags)
		args = append(args, fields...)
	}

	args = append(args, mainFile)

	return args, nil
}

// generateTestRunnerScript creates a wrapper script that embeds the compiled test binary and converts output to JUnit test.results.
func generateTestRunnerScript(scriptPath string, realBinaryPath string, crateName string) error {
	binBytes, err := os.ReadFile(realBinaryPath)
	if err != nil {
		return fmt.Errorf("failed to read test binary: %w", err)
	}

	_ = os.Remove(realBinaryPath)

	scriptContent := fmt.Sprintf(`#!/bin/bash
set -eo pipefail

TMPDIR="$(mktemp -d)"
BIN="$TMPDIR/test_bin"
trap 'rm -rf "$TMPDIR"' EXIT

# Extract embedded test binary
sed '1,/^#__BINARY_PAYLOAD__#/d' "$0" > "$BIN"
chmod +x "$BIN"

# Run test binary, stream output, and capture into temp file
TMP_OUT="$TMPDIR/output.txt"

set +e
"$BIN" --nocapture "$@" 2>&1 | tee "$TMP_OUT"
STATUS="${PIPESTATUS[0]}"
set -e

# Parse test output into JUnit test.results
python3 -c "
import sys, re, xml.etree.ElementTree as ET

try:
    with open('$TMP_OUT') as f:
        lines = f.read().splitlines()
except Exception:
    lines = []

test_re = re.compile(r'^test\s+([^\s]+)\s+\.\.\.\s+(ok|FAILED|ignored)')
suite = ET.Element('testsuite', name='%s')

tests = 0
failures = 0

for line in lines:
    m = test_re.match(line.strip())
    if m:
        tname, status = m.groups()
        tests += 1
        tc = ET.SubElement(suite, 'testcase', name=tname, classname='%s', time='0.000')
        if status == 'FAILED':
            failures += 1
            f = ET.SubElement(tc, 'failure', message='Test failed', type='Failure')

suite.set('tests', str(tests or 1))
suite.set('failures', str(failures))
suite.set('errors', '0')
suite.set('time', '0.000')

suites = ET.Element('testsuites')
suites.append(suite)

with open('test.results', 'wb') as f:
    f.write(b'<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n')
    f.write(ET.tostring(suites))
" 2>/dev/null || true

exit "$STATUS"
#__BINARY_PAYLOAD__#
`, crateName, crateName)

	fullData := append([]byte(scriptContent), binBytes...)
	if err := os.WriteFile(scriptPath, fullData, 0755); err != nil {
		return fmt.Errorf("failed to write test runner script: %w", err)
	}
	return nil
}

// Run executes the Rust compilation with rustc.
func Run(opts Options) error {
	rustcPath, err := toolchain.FindRustc(opts.Rustc)
	if err != nil {
		return err
	}

	outDir := filepath.Dir(opts.Out)
	if outDir != "" && outDir != "." {
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return fmt.Errorf("failed to create output dir %s: %w", outDir, err)
		}
	}

	realBinaryOut := opts.Out
	if opts.CrateType == "test" {
		realBinaryOut = opts.Out + ".raw_bin"
	}

	args, err := BuildRustcArgs(opts, realBinaryOut)
	if err != nil {
		return err
	}

	cmd := exec.Command(rustcPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if opts.Version != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("CARGO_PKG_VERSION=%s", opts.Version))
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rustc compilation failed: %w", err)
	}

	if opts.CrateType == "test" {
		if err := generateTestRunnerScript(opts.Out, realBinaryOut, opts.CrateName); err != nil {
			return err
		}
	}

	return nil
}

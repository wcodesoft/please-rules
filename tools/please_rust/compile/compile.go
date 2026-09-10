package compile

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"tools/please_rust/toolchain"
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

var hashSuffixRegex = regexp.MustCompile(`-[0-9a-fA-F]{16}$`)

// extractCrateName derives the crate name from a library artifact filename.
func extractCrateName(filename string) string {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	trimmed := strings.TrimSuffix(base, ext)
	trimmed = strings.TrimPrefix(trimmed, "lib")
	trimmed = hashSuffixRegex.ReplaceAllString(trimmed, "")
	return sanitizeCrateName(trimmed)
}

// findMatchingInput checks if target exists on filesystem or matches an input path.
func findMatchingInput(target string, inputs []string) string {
	if _, err := os.Stat(target); err == nil {
		return target
	}
	targetBase := filepath.Base(target)
	for _, input := range inputs {
		if input == target || filepath.Base(input) == targetBase || strings.HasSuffix(input, target) {
			if _, err := os.Stat(input); err == nil {
				return input
			}
		}
	}
	return ""
}

// findFirstRsFile finds the first existing .rs file from inputs or current directory.
func findFirstRsFile(inputs []string) string {
	for _, input := range inputs {
		if filepath.Ext(input) == ".rs" {
			if _, err := os.Stat(input); err == nil {
				return input
			}
		}
	}

	if entries, err := os.ReadDir("."); err == nil {
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".rs" {
				return e.Name()
			}
		}
	}
	return ""
}

// resolveMainSrc finds the actual entrypoint source file among inputs and current directory.
func resolveMainSrc(mainSrc string, crateType string, inputs []string) string {
	candidates := []string{"lib.rs", "main.rs", "src/lib.rs", "src/main.rs"}
	if crateType == "bin" {
		candidates = []string{"main.rs", "src/main.rs", "lib.rs", "src/lib.rs"}
	}
	if mainSrc != "" {
		candidates = append([]string{mainSrc}, candidates...)
	}

	for _, cand := range candidates {
		if found := findMatchingInput(cand, inputs); found != "" {
			return found
		}
	}

	if fallback := findFirstRsFile(inputs); fallback != "" {
		return fallback
	}

	return mainSrc
}

// isLibFile returns true if the given filename has a shared/static library extension.
func isLibFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".rlib" || ext == ".so" || ext == ".dylib" || ext == ".dll"
}

// discoverDepFiles scans directories in inputs or the current build directory for .rlib and .so files.
func discoverDepFiles(inputs []string) []string {
	seen := make(map[string]bool)
	var deps []string

	addDep := func(p string) {
		clean := filepath.Clean(p)
		if !seen[clean] {
			seen[clean] = true
			deps = append(deps, clean)
		}
	}

	for _, input := range inputs {
		if isLibFile(input) {
			addDep(input)
		}
	}

	_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() && isLibFile(path) {
			addDep(path)
		}
		return nil
	})

	return deps
}

// updateExternMap registers depPath under its crate name if not already seen or preferred over hashed names.
func updateExternMap(externMap map[string]string, depPath, selfSanitized string) {
	cName := extractCrateName(depPath)
	if cName == "" || cName == selfSanitized {
		return
	}
	existing, exists := externMap[cName]
	isUnhashed := !strings.Contains(filepath.Base(depPath), "-")
	hasHashedExisting := exists && strings.Contains(filepath.Base(existing), "-")
	if !exists || (isUnhashed && hasHashedExisting) {
		externMap[cName] = depPath
	}
}

// resolveExternFlags builds search directory (-L) and extern library (--extern) arguments.
func resolveExternFlags(depPaths []string, selfCrate string) []string {
	searchDirs := make(map[string]bool)
	externMap := make(map[string]string)
	selfSanitized := sanitizeCrateName(selfCrate)

	for _, depPath := range depPaths {
		if dir := filepath.Dir(depPath); dir != "" && dir != "." {
			searchDirs[dir] = true
		}
		updateExternMap(externMap, depPath, selfSanitized)
	}

	var args []string
	for dir := range searchDirs {
		args = append(args, "-L", fmt.Sprintf("dependency=%s", dir))
	}
	args = append(args, "-L", "dependency=.")

	for cName, path := range externMap {
		args = append(args, "--extern", fmt.Sprintf("%s=%s", cName, path))
	}

	return args
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

	if opts.CrateType == "test" {
		args = append(args, "--test")
	} else if opts.CrateType != "" {
		args = append(args, "--crate-type", opts.CrateType)
	}

	args = append(args, "--crate-name", sanitizeCrateName(opts.CrateName))

	outPath := opts.Out
	if opts.CrateType == "test" && realBinaryOut != "" {
		outPath = realBinaryOut
	}
	args = append(args, "-o", outPath)

	allDeps := discoverDepFiles(opts.Inputs)
	args = append(args, resolveExternFlags(allDeps, opts.CrateName)...)

	if opts.Flags != "" {
		args = append(args, strings.Fields(opts.Flags)...)
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

// ensureOutputDir creates the parent directory of path if needed.
func ensureOutputDir(path string) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output dir %s: %w", dir, err)
		}
	}
	return nil
}

// invokeRustc executes rustc with the provided arguments and environment.
func invokeRustc(rustcPath string, args []string, version string) error {
	cmd := exec.Command(rustcPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if version != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("CARGO_PKG_VERSION=%s", version))
	}

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "DEBUG: rustc command failed: %s %s\n", rustcPath, strings.Join(args, " "))
		return fmt.Errorf("rustc compilation failed: %w", err)
	}
	return nil
}

// Run executes the Rust compilation with rustc.
func Run(opts Options) error {
	rustcPath, err := toolchain.FindRustc(opts.Rustc)
	if err != nil {
		return err
	}

	if err := ensureOutputDir(opts.Out); err != nil {
		return err
	}

	realBinaryOut := opts.Out
	if opts.CrateType == "test" {
		realBinaryOut = opts.Out + ".raw_bin"
	}

	args, err := BuildRustcArgs(opts, realBinaryOut)
	if err != nil {
		return err
	}

	if err := invokeRustc(rustcPath, args, opts.Version); err != nil {
		return err
	}

	if opts.CrateType == "test" {
		return generateTestRunnerScript(opts.Out, realBinaryOut, opts.CrateName)
	}

	return nil
}

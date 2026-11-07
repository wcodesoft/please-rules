package compile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tools/please_ts/npmcache"
)

func TestEmitOutput(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compile_emit_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srcFile := filepath.Join(tmpDir, "calculator.ts")
	if err := os.WriteFile(srcFile, []byte("export const x = 1;"), 0644); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(tmpDir, "out")
	opts := Options{
		Out:        outDir,
		Srcs:       []string{srcFile},
		ModuleName: "@domain/calculator",
	}

	if err := emitOutput(opts, nil); err != nil {
		t.Fatalf("emitOutput failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "calculator.ts")); err != nil {
		t.Errorf("expected copied calculator.ts in out: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "ts_metadata.json")); err != nil {
		t.Errorf("expected ts_metadata.json in out: %v", err)
	}
}

func TestExpandCommaSeparated(t *testing.T) {
	input := []string{"a, b, c", "d,e"}
	output := ExpandCommaSeparated(input)
	if len(output) != 5 {
		t.Fatalf("expected 5 items, got %d", len(output))
	}
	expected := []string{"a", "b", "c", "d", "e"}
	for i, v := range output {
		if v != expected[i] {
			t.Errorf("expected %s at index %d, got %s", expected[i], i, v)
		}
	}
}

// fakeDeno writes a `deno` that records its arguments, its DENO_DIR and the directories
// of the npm cache it sees there.
func fakeDeno(t *testing.T) (bin, log string) {
	t.Helper()
	dir := t.TempDir()
	log = filepath.Join(dir, "calls.log")
	bin = filepath.Join(dir, "deno")
	script := `#!/bin/sh
{
  echo "args: $*"
  echo "DENO_DIR=$DENO_DIR"
  (cd "$DENO_DIR" && find npm -mindepth 3 -maxdepth 3 -type d 2>/dev/null | sort | sed 's/^/pkg: /')
} >> "` + log + `"
`
	if err := os.WriteFile(bin, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return bin, log
}

func writePackage(t *testing.T, root, name, version string) {
	t.Helper()
	dir := filepath.Join(root, "npm", "registry.npmjs.org", name, version)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
}

// inWorkDir changes to a fresh directory holding the sources and, optionally, an npm slice.
func inWorkDir(t *testing.T, withSlice bool) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.ts"), []byte("export const a = 1;\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if withSlice {
		slice := filepath.Join(dir, "third_party", "debug")
		writePackage(t, slice, "debug", "4.3.7")
		if err := npmcache.Write(slice, npmcache.Slice{Name: "debug", Version: "4.3.7", Specifier: "npm:debug@4.3.7"}); err != nil {
			t.Fatal(err)
		}
	}
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

func TestRunVitestTargetWithNpmSlicesUsesAPerRunCacheAndNeverTheNetwork(t *testing.T) {
	inWorkDir(t, true)
	shared := t.TempDir()
	writePackage(t, shared, "vitest", "5.0.3")
	deno, log := fakeDeno(t)

	if err := Run(Options{Deno: deno, Srcs: []string{"a.ts"}, VitestDir: shared}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(log)
	calls := string(data)
	if !strings.Contains(calls, "--cached-only") {
		t.Errorf("deno check can reach the network:\n%s", calls)
	}
	if strings.Contains(calls, "DENO_DIR="+shared+"\n") {
		t.Errorf("deno wrote into the shared Vitest cache:\n%s", calls)
	}
	for _, want := range []string{"pkg: npm/registry.npmjs.org/vitest/5.0.3", "pkg: npm/registry.npmjs.org/debug/4.3.7"} {
		if !strings.Contains(calls, want) {
			t.Errorf("the per-run cache lacks %q:\n%s", want, calls)
		}
	}
	// the shared cache did not get the slice
	if _, err := os.Stat(filepath.Join(shared, "npm", "registry.npmjs.org", "debug")); err == nil {
		t.Error("the npm slice was merged into the shared Vitest cache")
	}
}

func TestRunVitestTargetWithoutNpmSlicesReadsTheSharedCacheFromAPrivateDirectory(t *testing.T) {
	inWorkDir(t, false)
	shared := t.TempDir()
	writePackage(t, shared, "vitest", "5.0.3")
	deno, log := fakeDeno(t)
	if err := Run(Options{Deno: deno, Srcs: []string{"a.ts"}, VitestDir: shared}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(log)
	if strings.Contains(string(data), "DENO_DIR="+shared+"\n") {
		t.Errorf("deno wrote into the shared Vitest cache, which it does on every run:\n%s", data)
	}
	if !strings.Contains(string(data), "DENO_DIR=") || strings.Contains(string(data), "--cached-only") {
		t.Errorf("unexpected invocation:\n%s", data)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestEmbedDependencies(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "pkg", "out")

	// Two sibling libraries, one npm module, and a library that already embeds a dependency of its
	// own (a nested module, as every library output now does).
	writeTestFile(t, filepath.Join(root, "a", "a", "ts_metadata.json"), "{}")
	writeTestFile(t, filepath.Join(root, "a", "a", "a.ts"), "a")
	writeTestFile(t, filepath.Join(root, "b", "b", "ts_metadata.json"), "{}")
	writeTestFile(t, filepath.Join(root, "b", "b", "b.ts"), "b")
	writeTestFile(t, filepath.Join(root, "b", "b", "c", "c", "ts_metadata.json"), "{}")
	writeTestFile(t, filepath.Join(root, "b", "b", "c", "c", "c.ts"), "c")
	writeTestFile(t, filepath.Join(root, "npm", "clsx", "ts_module.json"), "{}")
	writeTestFile(t, filepath.Join(root, "npm", "clsx", "dist", "clsx.mjs"), "clsx")
	writeTestFile(t, filepath.Join(root, "npm", "clsx", ".deps", "ms", "index.js"), "ms")
	if err := os.Symlink(".deps", filepath.Join(root, "npm", "clsx", "node_modules")); err != nil {
		t.Fatal(err)
	}
	// A stray file that is not part of any module must not be copied.
	writeTestFile(t, filepath.Join(root, "unrelated", "notes.txt"), "x")
	// The target's own output, with its own metadata, must not be copied into itself.
	writeTestFile(t, filepath.Join(out, "ts_metadata.json"), "{}")
	writeTestFile(t, filepath.Join(out, "own.ts"), "own")

	if err := embedDependencies(root, out); err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"a/a/a.ts", "a/a/ts_metadata.json",
		"b/b/b.ts", "b/b/c/c/c.ts", "b/b/c/c/ts_metadata.json",
		"npm/clsx/dist/clsx.mjs", "npm/clsx/ts_module.json",
		"own.ts",
	} {
		if _, err := os.Stat(filepath.Join(out, want)); err != nil {
			t.Errorf("%s missing from the output: %v", want, err)
		}
	}
	if link, err := os.Readlink(filepath.Join(out, "npm", "clsx", "node_modules")); err != nil || link != ".deps" {
		t.Errorf("node_modules link = %q, %v, want a relative link to .deps", link, err)
	}
	if _, err := os.Stat(filepath.Join(out, "npm", "clsx", "node_modules", "ms", "index.js")); err != nil {
		t.Errorf("a dependency is not reachable through the copied node_modules link: %v", err)
	}
	for _, unwanted := range []string{
		"unrelated", // not a module
		"c",         // the nested module travels inside b, not as a sibling
		"pkg/out",   // the output is not copied into itself
		"out/pkg/out",
	} {
		if _, err := os.Stat(filepath.Join(out, unwanted)); err == nil {
			t.Errorf("%s was copied into the output", unwanted)
		}
	}
}

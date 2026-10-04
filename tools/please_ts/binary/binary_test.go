package binary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tools/please_ts/npmcache"
)

func TestOptionsValidation(t *testing.T) {
	err := Run(Options{})
	if err == nil {
		t.Errorf("expected error when Main is empty")
	}

	err = Run(Options{Main: "main.ts"})
	if err == nil {
		t.Errorf("expected error when Out is empty")
	}
}

// fakeDeno writes a `deno` that records its arguments and the files below DENO_DIR/npm.
func fakeDeno(t *testing.T) (bin, log string) {
	t.Helper()
	dir := t.TempDir()
	log = filepath.Join(dir, "calls.log")
	bin = filepath.Join(dir, "deno")
	script := `#!/bin/sh
{
  echo "args: $*"
  (cd "$DENO_DIR" && find npm -type d 2>/dev/null | sort | sed 's/^/dir: /')
} >> "` + log + `"
out=""
while [ $# -gt 0 ]; do if [ "$1" = "-o" ]; then out="$2"; fi; shift; done
[ -n "$out" ] && echo binary > "$out"
`
	if err := os.WriteFile(bin, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return bin, log
}

func inDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	return dir
}

func TestRunUsesTheNpmCacheSlicesOfItsDependenciesOffline(t *testing.T) {
	dir := inDir(t)
	slice := filepath.Join(dir, "third_party", "debug")
	if err := os.MkdirAll(filepath.Join(slice, "npm", "registry.npmjs.org", "debug", "4.3.7"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := npmcache.Write(slice, npmcache.Slice{Name: "debug", Version: "4.3.7", Specifier: "npm:debug@4.3.7"}); err != nil {
		t.Fatal(err)
	}
	deno, log := fakeDeno(t)

	if err := Run(Options{Deno: deno, Out: "bin/app", Main: "main.ts", Srcs: []string{"main.ts"}}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(log)
	calls := string(data)
	for _, want := range []string{
		"args: compile --no-remote --unstable-detect-cjs --import-map .import_map.json -o bin/app --cached-only main.ts",
		"dir: npm/registry.npmjs.org/debug/4.3.7", // the slice was merged into the per-run DENO_DIR
	} {
		if !strings.Contains(calls, want) {
			t.Errorf("missing %q in:\n%s", want, calls)
		}
	}
}

func TestRunWithoutNpmPackagesDoesNotRestrictDenoToTheCache(t *testing.T) {
	inDir(t)
	deno, log := fakeDeno(t)
	if err := Run(Options{Deno: deno, Out: "app", Main: "main.ts", Srcs: []string{"main.ts"}}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(log)
	if strings.Contains(string(data), "--cached-only") {
		t.Errorf("--cached-only without npm packages:\n%s", data)
	}
}

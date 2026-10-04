package bundle

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tools/please_ts/npmcache"
)

// fakeDeno writes an executable that behaves like `deno`: `--version` prints version, and any
// other invocation appends its arguments, DENO_DIR and the esbuild files visible under it to
// log, then runs extra (a shell snippet) and writes the bundle given by -o.
func fakeDeno(t *testing.T, version, extra string) (bin, log string) {
	t.Helper()
	dir := t.TempDir()
	log = filepath.Join(dir, "calls.log")
	bin = filepath.Join(dir, "deno")
	script := `#!/bin/sh
if [ "$1" = "--version" ]; then echo "deno ` + version + ` (stable, release, x86_64-unknown-linux-gnu)"; exit 0; fi
{
  echo "args: $*"
  echo "DENO_DIR=$DENO_DIR"
  (cd "$DENO_DIR" && find dl -type l 2>/dev/null | sort | sed 's/^/esbuild: /')
} >> "` + log + `"
` + extra + `
out=""
while [ $# -gt 0 ]; do if [ "$1" = "-o" ]; then out="$2"; fi; shift; done
[ -n "$out" ] && echo "bundled" > "$out"
exit 0
`
	if err := os.WriteFile(bin, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	return bin, log
}

func fakeEsbuild(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "esbuild")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

// workDir makes a temporary directory holding one npm slice the current directory, as Please
// stages the slices of a target's dependencies.
func workDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	slice := filepath.Join(dir, "third_party", "debug")
	if err := os.MkdirAll(filepath.Join(slice, "npm", "registry.npmjs.org", "debug", "4.3.7"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := npmcache.Write(slice, npmcache.Slice{Name: "debug", Version: "4.3.7", Specifier: "npm:debug@4.3.7"}); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	return dir
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestEsbuildPlatform(t *testing.T) {
	for _, c := range []struct{ os, arch, want string }{
		{"linux", "amd64", "linux-x64"},
		{"linux", "arm64", "linux-arm64"},
		{"darwin", "arm64", "darwin-arm64"},
		{"darwin", "amd64", "darwin-x64"},
	} {
		if got, err := esbuildPlatform(c.os, c.arch); err != nil || got != c.want {
			t.Errorf("%s/%s = %q, %v; want %q", c.os, c.arch, got, err, c.want)
		}
	}
	for _, c := range [][2]string{{"windows", "amd64"}, {"linux", "386"}} {
		if _, err := esbuildPlatform(c[0], c[1]); err == nil {
			t.Errorf("%v: expected an error", c)
		}
	}
}

func TestDenoVersion(t *testing.T) {
	bin, _ := fakeDeno(t, "2.9.7", "")
	if v, err := denoVersion(bin); err != nil || v != "2.9.7" {
		t.Errorf("version = %q, %v", v, err)
	}
	notDeno := filepath.Join(t.TempDir(), "x")
	_ = os.WriteFile(notDeno, []byte("#!/bin/sh\necho something else\n"), 0755)
	if _, err := denoVersion(notDeno); err == nil || !strings.Contains(err.Error(), "something else") {
		t.Errorf("err = %v", err)
	}
	if _, err := denoVersion(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("expected an error for a missing deno")
	}
}

func TestPlaceEsbuild(t *testing.T) {
	bin, _ := fakeDeno(t, "2.9.7", "")
	esbuild := fakeEsbuild(t)
	platform, err := esbuildPlatform(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skip(err)
	}

	denoDir := t.TempDir()
	if err := placeEsbuild(bin, denoDir, Options{EsbuildBinary: esbuild}); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(denoDir, "dl", "esbuild-0.25.5-1", "esbuild-"+platform)
	if target, err := os.Readlink(link); err != nil || target != esbuild {
		t.Errorf("link %s -> %q, %v; want %s", link, target, err, esbuild)
	}

	// an explicit version wins over the table (and is the way out for an unknown Deno)
	other, _ := fakeDeno(t, "9.9.9", "")
	denoDir = t.TempDir()
	if err := placeEsbuild(other, denoDir, Options{EsbuildBinary: esbuild, EsbuildCacheVersion: "1.2.3-4"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(denoDir, "dl", "esbuild-1.2.3-4", "esbuild-"+platform)); err != nil {
		t.Error(err)
	}
}

func TestPlaceEsbuildErrors(t *testing.T) {
	bin, _ := fakeDeno(t, "2.9.7", "")
	unknown, _ := fakeDeno(t, "9.9.9", "")
	esbuild := fakeEsbuild(t)

	for name, c := range map[string]struct {
		deno string
		opts Options
		want string
	}{
		"no binary":      {bin, Options{}, "needs the pinned esbuild"},
		"unknown deno":   {unknown, Options{EsbuildBinary: esbuild}, "does not know where Deno 9.9.9 looks for esbuild"},
		"missing binary": {bin, Options{EsbuildBinary: filepath.Join(t.TempDir(), "gone")}, "esbuild binary"},
		"deno not found": {filepath.Join(t.TempDir(), "nodeno"), Options{EsbuildBinary: esbuild}, "--version"},
	} {
		err := placeEsbuild(c.deno, t.TempDir(), c.opts)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want one containing %q", name, err, c.want)
		}
	}
}

func TestRunBundlesNpmPackagesWithDenoBundle(t *testing.T) {
	workDir(t)
	deno, log := fakeDeno(t, "2.9.7", "")
	out := "dist/app.js"
	opts := Options{
		Deno: deno, EsbuildBinary: fakeEsbuild(t), Out: out, Main: "app.ts", Srcs: []string{"app.ts"},
		Format: "iife", Minify: true, Sourcemap: true, Flags: []string{"--quiet"},
	}
	if err := Run(opts); err != nil {
		t.Fatal(err)
	}
	if got := read(t, out); got != "bundled\n" {
		t.Errorf("bundle = %q", got)
	}
	calls := read(t, log)
	for _, want := range []string{
		"args: bundle --no-remote --import-map .import_map.json --platform browser -o dist/app.js --format iife --minify --sourcemap --quiet app.ts",
		"esbuild: dl/esbuild-0.25.5-1/esbuild-",
	} {
		if !strings.Contains(calls, want) {
			t.Errorf("deno was not run with %q:\n%s", want, calls)
		}
	}
	if _, err := os.Stat(".import_map.json"); err == nil {
		t.Error("the import map was left behind")
	}
	// the per-run DENO_DIR is a fixed name in the working directory (it ends up in the bundle's
	// comments, which must not depend on a random path), and is removed afterwards
	denoDir := strings.SplitN(strings.SplitN(calls, "DENO_DIR=", 2)[1], "\n", 2)[0]
	if filepath.Base(denoDir) != ".please_ts_deno" {
		t.Errorf("DENO_DIR = %s", denoDir)
	}
	if _, err := os.Stat(denoDir); err == nil {
		t.Errorf("DENO_DIR %s was not cleaned up", denoDir)
	}
}

func TestRunBundlesWithoutNpmPackagesThroughEsbuildAsBefore(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(old) })

	tool := filepath.Join(t.TempDir(), "esbuild-tool")
	_ = os.WriteFile(tool, []byte("#!/bin/sh\necho \"$@\" > args.txt\n"), 0755)
	if err := Run(Options{BundlerTool: tool, Out: "out.js", Main: "app.ts", Srcs: []string{"app.ts"}, Format: "esm"}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, "args.txt"); !strings.Contains(got, "app.ts --outfile=out.js --bundle --format=esm") {
		t.Errorf("esbuild args = %q", got)
	}
}

func TestRunNeedsThePinnedEsbuildForNpmPackages(t *testing.T) {
	workDir(t)
	deno, _ := fakeDeno(t, "2.9.7", "")
	err := Run(Options{Deno: deno, Out: "out.js", Main: "app.ts", Srcs: []string{"app.ts"}})
	if err == nil || !strings.Contains(err.Error(), "needs the pinned esbuild") {
		t.Errorf("err = %v", err)
	}
}

func TestRunFailsWhenDenoFetchedItsOwnEsbuild(t *testing.T) {
	workDir(t)
	// what Deno leaves behind after downloading @esbuild/<platform> itself
	deno, _ := fakeDeno(t, "2.9.7", `mkdir -p "$DENO_DIR/npm/registry.npmjs.org/@esbuild/linux-x64"`)
	err := Run(Options{Deno: deno, EsbuildBinary: fakeEsbuild(t), Out: "out.js", Main: "app.ts", Srcs: []string{"app.ts"}})
	if err == nil || !strings.Contains(err.Error(), "downloaded esbuild instead of using the pinned binary") {
		t.Errorf("err = %v", err)
	}
}

func TestRunReportsAFailingDenoBundle(t *testing.T) {
	workDir(t)
	deno, _ := fakeDeno(t, "2.9.7", "exit 3")
	err := Run(Options{Deno: deno, EsbuildBinary: fakeEsbuild(t), Out: "out.js", Main: "app.ts", Srcs: []string{"app.ts"}})
	if err == nil || !strings.Contains(err.Error(), "deno bundle failed") {
		t.Errorf("err = %v", err)
	}
}

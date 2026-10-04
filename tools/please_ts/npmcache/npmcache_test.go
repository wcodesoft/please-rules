package npmcache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func makeSlice(t *testing.T, root, name, version string, deps map[string]string) string {
	t.Helper()
	dir := filepath.Join(root, "slices", name+"-"+version)
	pkgDir := filepath.Join(dir, "npm", "registry.npmjs.org", name, version)
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "index.js"), []byte("module.exports = '"+version+"';"), 0644); err != nil {
		t.Fatal(err)
	}
	packument := map[string]any{
		"name":      name,
		"versions":  map[string]any{version: map[string]any{"name": name, "version": version}},
		"dist-tags": map[string]string{"latest": version},
	}
	data, _ := json.Marshal(packument)
	if err := os.WriteFile(filepath.Join(dir, "npm", "registry.npmjs.org", name, "registry.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	if err := Write(dir, Slice{Name: name, Version: version, Specifier: Specifier(name, version), Dependencies: deps}); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestCachedVersion(t *testing.T) {
	root := t.TempDir()
	mk := func(name, version string) {
		if err := os.MkdirAll(filepath.Join(root, "npm", "registry.npmjs.org", filepath.FromSlash(name), version), 0755); err != nil {
			t.Fatal(err)
		}
	}
	mk("vitest", "5.0.1")
	mk("@vitest/coverage-v8", "5.0.1")
	mk("debug", "4.3.7")
	mk("debug", "2.6.9")
	// a packument next to the version directories is not a version
	_ = os.WriteFile(filepath.Join(root, "npm", "registry.npmjs.org", "vitest", "registry.json"), []byte("{}"), 0644)

	if v, err := CachedVersion(root, "vitest"); err != nil || v != "5.0.1" {
		t.Errorf("vitest = %q, %v", v, err)
	}
	if v, err := CachedVersion(root, "@vitest/coverage-v8"); err != nil || v != "5.0.1" {
		t.Errorf("scoped = %q, %v", v, err)
	}
	if v, err := CachedVersion(root, "absent"); err != nil || v != "" {
		t.Errorf("absent = %q, %v", v, err)
	}
	if _, err := CachedVersion(root, "debug"); err == nil || !strings.Contains(err.Error(), "2.6.9, 4.3.7") {
		t.Errorf("several versions: err = %v", err)
	}
}

func TestWriteReadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := Slice{Name: "@scope/pkg", Version: "1.2.3", Specifier: "npm:@scope/pkg@1.2.3", Dependencies: map[string]string{"ms": "^2.1.3"}}
	if err := Write(dir, want); err != nil {
		t.Fatal(err)
	}
	got, err := Read(dir)
	if err != nil || !reflect.DeepEqual(*got, want) {
		t.Fatalf("Read = %+v, %v; want %+v", got, err, want)
	}
}

func TestReadRejectsIncompleteMetadata(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, MetadataFile), []byte(`{"name":"x"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(dir); err == nil {
		t.Error("expected an error for metadata without a version")
	}
	if _, err := Read(t.TempDir()); err == nil {
		t.Error("expected an error for a missing file")
	}
}

func TestDiscoverFindsSlicesSorted(t *testing.T) {
	root := t.TempDir()
	b := makeSlice(t, root, "b", "1.0.0", nil)
	a := makeSlice(t, root, "a", "1.0.0", nil)
	got := Discover(root)
	if !reflect.DeepEqual(got, []string{a, b}) {
		t.Errorf("Discover = %v, want [%s %s]", got, a, b)
	}
	if len(Discover(t.TempDir())) != 0 {
		t.Error("an empty tree has no slices")
	}
}

func TestMergeCopiesPackagesIntoDenoDir(t *testing.T) {
	root := t.TempDir()
	debug := makeSlice(t, root, "debug", "4.3.7", map[string]string{"ms": "^2.1.3"})
	ms := makeSlice(t, root, "ms", "2.1.3", nil)
	denoDir := filepath.Join(t.TempDir(), "deno")

	if err := Merge([]string{debug, ms}, denoDir); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		"npm/registry.npmjs.org/debug/4.3.7/index.js",
		"npm/registry.npmjs.org/debug/registry.json",
		"npm/registry.npmjs.org/ms/2.1.3/index.js",
		"npm/registry.npmjs.org/ms/registry.json",
	} {
		if _, err := os.Stat(filepath.Join(denoDir, p)); err != nil {
			t.Errorf("missing %s: %v", p, err)
		}
	}
	if _, err := os.Stat(filepath.Join(denoDir, MetadataFile)); err == nil {
		t.Error("slice metadata must not be copied into the cache")
	}
}

func TestMergeUnionsVersionsOfTheSamePackage(t *testing.T) {
	root := t.TempDir()
	old := makeSlice(t, root, "ms", "2.0.0", nil)
	cur := makeSlice(t, root, "ms", "2.1.3", nil)
	denoDir := t.TempDir()

	if err := Merge([]string{old, cur}, denoDir); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(denoDir, "npm/registry.npmjs.org/ms/registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	var packument struct {
		Versions map[string]json.RawMessage `json:"versions"`
	}
	if err := json.Unmarshal(data, &packument); err != nil {
		t.Fatal(err)
	}
	if len(packument.Versions) != 2 {
		t.Errorf("versions = %v, want both 2.0.0 and 2.1.3", packument.Versions)
	}
	for _, v := range []string{"2.0.0", "2.1.3"} {
		if _, err := os.Stat(filepath.Join(denoDir, "npm/registry.npmjs.org/ms", v, "index.js")); err != nil {
			t.Errorf("version %s missing: %v", v, err)
		}
	}
}

func TestMergeIsIdempotentAndKeepsExistingFiles(t *testing.T) {
	root := t.TempDir()
	s := makeSlice(t, root, "ms", "2.1.3", nil)
	denoDir := t.TempDir()
	for i := 0; i < 2; i++ {
		if err := Merge([]string{s}, denoDir); err != nil {
			t.Fatal(err)
		}
	}
	target := filepath.Join(denoDir, "npm/registry.npmjs.org/ms/2.1.3/index.js")
	if err := os.WriteFile(target, []byte("kept"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Merge([]string{s}, denoDir); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(target); string(got) != "kept" {
		t.Errorf("an existing file was overwritten: %q", got)
	}
}

func TestMergeSkipsSlicesWithoutCacheContent(t *testing.T) {
	dir := t.TempDir()
	if err := Merge([]string{dir}, t.TempDir()); err != nil {
		t.Errorf("a directory without npm/ must be skipped, got %v", err)
	}
}

func TestPrepareMergesAndReportsWhetherThereWereSlices(t *testing.T) {
	root := t.TempDir()
	makeSlice(t, root, "ms", "2.1.3", nil)
	denoDir := filepath.Join(t.TempDir(), "deno")

	found, err := Prepare(root, denoDir)
	if err != nil || !found {
		t.Fatalf("Prepare = %v, %v; want true", found, err)
	}
	if _, err := os.Stat(filepath.Join(denoDir, "npm/registry.npmjs.org/ms/2.1.3/index.js")); err != nil {
		t.Errorf("package not merged: %v", err)
	}

	empty := filepath.Join(t.TempDir(), "deno")
	found, err = Prepare(t.TempDir(), empty)
	if err != nil || found {
		t.Fatalf("Prepare without slices = %v, %v; want false", found, err)
	}
	if _, err := os.Stat(empty); err == nil {
		t.Error("the Deno directory must be left alone when there is nothing to merge")
	}
}

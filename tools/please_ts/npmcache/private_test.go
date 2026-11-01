package npmcache

import (
	"os"
	"path/filepath"
	"testing"
)

func sharedCache(t *testing.T) string {
	t.Helper()
	shared := t.TempDir()
	pkg := filepath.Join(shared, "npm", "registry.npmjs.org", "vitest", "5.0.1")
	if err := os.MkdirAll(pkg, 0755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(pkg, "index.js"): "export {}",
		"check_cache_v2":               "database",
		"check_cache_v2-wal":           "log of a connection",
		"check_cache_v2-shm":           "memory of a connection",
		"dep_analysis_cache_v2":        "another database",
	}
	for path, content := range files {
		if !filepath.IsAbs(path) {
			path = filepath.Join(shared, path)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return shared
}

func TestPrivateReadsTheSharedPackagesAndCopiesTheDatabases(t *testing.T) {
	shared := sharedCache(t)
	private := filepath.Join(t.TempDir(), ".deno_cache")
	if err := Private(shared, private); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(private, "npm", "registry.npmjs.org", "vitest", "5.0.1", "index.js")); err != nil || string(got) != "export {}" {
		t.Errorf("expected the shared packages to be readable through the private directory, got %q, %v", got, err)
	}
	for _, name := range []string{"check_cache_v2", "dep_analysis_cache_v2"} {
		if got, err := os.ReadFile(filepath.Join(private, name)); err != nil || len(got) == 0 {
			t.Errorf("expected a copy of %s, got %q, %v", name, got, err)
		}
	}
	for _, name := range []string{"check_cache_v2-wal", "check_cache_v2-shm"} {
		if _, err := os.Stat(filepath.Join(private, name)); err == nil {
			t.Errorf("%s belongs to a connection of the shared database and must not be copied", name)
		}
	}
}

func TestPrivateNeverWritesToTheSharedCache(t *testing.T) {
	shared := sharedCache(t)
	private := filepath.Join(t.TempDir(), ".deno_cache")
	if err := Private(shared, private); err != nil {
		t.Fatal(err)
	}
	// what a Deno run does: write to its databases and add a transpiled file
	if err := os.WriteFile(filepath.Join(private, "check_cache_v2"), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(private, "gen"), 0755); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(shared, "check_cache_v2")); string(got) != "database" {
		t.Errorf("the shared database changed: %q", got)
	}
	if _, err := os.Stat(filepath.Join(shared, "gen")); err == nil {
		t.Error("something was added to the shared cache")
	}
}

func TestPrivateOfACacheWithoutPackagesOrDatabasesIsAnEmptyDirectory(t *testing.T) {
	private := filepath.Join(t.TempDir(), "d")
	if err := Private(t.TempDir(), private); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(private)
	if err != nil || len(entries) != 0 {
		t.Errorf("expected an empty directory, got %v, %v", entries, err)
	}
}

func TestPrivateReportsASharedCacheThatIsNotThere(t *testing.T) {
	if err := Private(filepath.Join(t.TempDir(), "missing"), filepath.Join(t.TempDir(), "d")); err == nil {
		t.Error("expected an error for a shared cache that does not exist")
	}
}

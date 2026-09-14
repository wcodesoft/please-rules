package binary

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunValidation(t *testing.T) {
	// Missing output
	err := Run(Options{Out: ""})
	if err == nil {
		t.Error("expected error on empty out")
	}

	// Missing sources
	err = Run(Options{Out: "/tmp/out"})
	if err == nil {
		t.Error("expected error on missing sources and main")
	}
}

func TestDiscoverDependencies(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "swift-bin-dep-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "libFoo.a"), []byte("dummy-archive"), 0644)

	incDirs, archives := discoverDependencies([]string{tmpDir})

	if len(incDirs) != 1 || incDirs[0] != tmpDir {
		t.Errorf("unexpected incDirs: %v", incDirs)
	}
	if len(archives) != 1 || archives[0] != filepath.Join(tmpDir, "libFoo.a") {
		t.Errorf("unexpected archives: %v", archives)
	}
}

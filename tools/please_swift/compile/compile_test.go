package compile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExpandCommaSeparated(t *testing.T) {
	input := []string{"a, b ,c", "d", "e,f"}
	expected := []string{"a", "b", "c", "d", "e", "f"}
	got := ExpandCommaSeparated(input)

	if len(got) != len(expected) {
		t.Fatalf("expected %d elements, got %d", len(expected), len(got))
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("at index %d: expected %q, got %q", i, expected[i], got[i])
		}
	}
}

func TestRunValidation(t *testing.T) {
	// Missing module name
	err := Run(Options{ModuleName: ""})
	if err == nil {
		t.Error("expected error on empty module name")
	}

	// Missing sources
	err = Run(Options{ModuleName: "Foo", Srcs: nil})
	if err == nil {
		t.Error("expected error on empty srcs")
	}

	// Missing output
	err = Run(Options{ModuleName: "Foo", Srcs: []string{"foo.swift"}})
	if err == nil {
		t.Error("expected error on empty out-dir and out-lib")
	}
}

func TestDiscoverDependencies(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "swift-dep-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	meta := Metadata{
		ModuleName:   "DepModule",
		Archive:      "libDepModule.a",
		SwiftModule:  "DepModule.swiftmodule",
		ModuleDir:    tmpDir,
		Dependencies: nil,
	}
	metaBytes, _ := json.Marshal(meta)
	_ = os.WriteFile(filepath.Join(tmpDir, "swift_metadata.json"), metaBytes, 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "libDepModule.a"), []byte("dummy-archive"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "DepModule.swiftmodule"), []byte("dummy-module"), 0644)

	incDirs, archives, depNames := discoverDependencies([]string{tmpDir})

	if len(incDirs) != 1 || incDirs[0] != tmpDir {
		t.Errorf("unexpected incDirs: %v", incDirs)
	}
	if len(archives) != 1 || archives[0] != filepath.Join(tmpDir, "libDepModule.a") {
		t.Errorf("unexpected archives: %v", archives)
	}
	if len(depNames) != 1 || depNames[0] != "DepModule" {
		t.Errorf("unexpected depNames: %v", depNames)
	}
}

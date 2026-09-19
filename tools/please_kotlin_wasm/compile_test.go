package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandCommaSeparated(t *testing.T) {
	input := []string{"a,b,c", "d", " e , f "}
	expected := []string{"a", "b", "c", "d", "e", "f"}
	result := ExpandCommaSeparated(input)

	if len(result) != len(expected) {
		t.Fatalf("expected %d elements, got %d", len(expected), len(result))
	}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("at index %d: expected %q, got %q", i, v, result[i])
		}
	}
}

func TestDiscoverWasmSources(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wasm-srcs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	kt1 := filepath.Join(tmpDir, "A.kt")
	kt2 := filepath.Join(subDir, "B.kt")
	other := filepath.Join(tmpDir, "other.txt")

	_ = os.WriteFile(kt1, []byte("// A"), 0644)
	_ = os.WriteFile(kt2, []byte("// B"), 0644)
	_ = os.WriteFile(other, []byte("// other"), 0644)

	// Test passing directory
	found := DiscoverWasmSources([]string{tmpDir})
	if len(found) != 2 {
		t.Errorf("expected 2 .kt files, got %d: %v", len(found), found)
	}

	// Test passing single file
	foundSingle := DiscoverWasmSources([]string{kt1})
	if len(foundSingle) != 1 || foundSingle[0] != kt1 {
		t.Errorf("expected [%s], got %v", kt1, foundSingle)
	}
}

func TestDiscoverKlibs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wasm-klibs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	klib1 := filepath.Join(tmpDir, "lib1.klib")
	other := filepath.Join(tmpDir, "other.jar")

	_ = os.WriteFile(klib1, []byte("klib"), 0644)
	_ = os.WriteFile(other, []byte("jar"), 0644)

	found := DiscoverKlibs([]string{tmpDir})
	if len(found) != 1 || found[0] != klib1 {
		t.Errorf("expected [%s], got %v", klib1, found)
	}
}

func TestResolveKotlincWasm(t *testing.T) {
	// Explicit path that doesn't exist falls back gracefully
	path, err := ResolveKotlincWasm("custom-kotlinc-wasm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path == "" {
		t.Errorf("expected non-empty path")
	}
}

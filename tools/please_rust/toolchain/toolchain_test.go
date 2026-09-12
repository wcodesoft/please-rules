package toolchain

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRustc(t *testing.T) {
	rustcPath, err := FindRustc("")
	if err != nil {
		t.Logf("FindRustc returned error (expected if rustc not installed): %v", err)
	} else {
		t.Logf("Found rustc at: %s", rustcPath)
	}
}

func TestFindLlvmProfdata(t *testing.T) {
	path, err := FindLlvmProfdata("")
	if err != nil {
		t.Logf("FindLlvmProfdata returned error (expected if llvm-profdata not installed): %v", err)
	} else {
		t.Logf("Found llvm-profdata at: %s", path)
	}
}

func TestFindLlvmCov(t *testing.T) {
	path, err := FindLlvmCov("")
	if err != nil {
		t.Logf("FindLlvmCov returned error (expected if llvm-cov not installed): %v", err)
	} else {
		t.Logf("Found llvm-cov at: %s", path)
	}
}

func TestFindRustc_ValidOverride(t *testing.T) {
	tmpDir := t.TempDir()
	dummyBin := filepath.Join(tmpDir, "my_rustc")
	if err := os.WriteFile(dummyBin, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("writing dummy bin: %v", err)
	}

	found, err := FindRustc(dummyBin)
	if err != nil {
		t.Fatalf("FindRustc with override failed: %v", err)
	}
	absExpected, _ := filepath.Abs(dummyBin)
	if found != absExpected {
		t.Errorf("got %s, want %s", found, absExpected)
	}
}

func TestFindRustc_InvalidOverride(t *testing.T) {
	_, err := FindRustc("/nonexistent/bin/rustc")
	if err == nil {
		t.Fatal("expected error for invalid override, got nil")
	}
}

func TestCheckExecutable(t *testing.T) {
	// A directory should return ""
	tmpDir := t.TempDir()
	if path := checkExecutable(tmpDir); path != "" {
		t.Errorf("expected empty string for directory, got %s", path)
	}

	// Non-existent path should return ""
	if path := checkExecutable(filepath.Join(tmpDir, "missing")); path != "" {
		t.Errorf("expected empty string for missing path, got %s", path)
	}

	// Existing file
	file := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(file, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	if path := checkExecutable(file); path != file {
		t.Errorf("expected %s, got %s", file, path)
	}
}

func TestCandidateSearchPaths(t *testing.T) {
	paths := candidateSearchPaths("rustc")
	if len(paths) == 0 {
		t.Fatal("expected non-empty candidate search paths")
	}
}

func TestFindTool_NotFound(t *testing.T) {
	_, err := findTool("", "completely_imaginary_tool_12345")
	if err == nil {
		t.Fatal("expected error for missing tool, got nil")
	}
}

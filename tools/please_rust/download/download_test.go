package download

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildCrateMeta_DefaultEditionAndLibSrc(t *testing.T) {
	tmpDir := t.TempDir()
	dirName := "mycrate-1.2.3"
	extractedDir := filepath.Join(tmpDir, dirName)
	srcDir := filepath.Join(extractedDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write a Cargo.toml with an explicit edition.
	toml := `[package]
name = "mycrate"
version = "1.2.3"
edition = "2021"
`
	if err := os.WriteFile(filepath.Join(extractedDir, "Cargo.toml"), []byte(toml), 0644); err != nil {
		t.Fatal(err)
	}
	// Create src/lib.rs so the default path exists.
	if err := os.WriteFile(filepath.Join(srcDir, "lib.rs"), []byte("// lib"), 0644); err != nil {
		t.Fatal(err)
	}

	meta, err := buildCrateMeta("mycrate", "1.2.3", extractedDir, dirName)
	if err != nil {
		t.Fatalf("buildCrateMeta returned error: %v", err)
	}

	if meta.Edition != "2021" {
		t.Errorf("edition: got %q, want %q", meta.Edition, "2021")
	}
	wantLibSrc := fmt.Sprintf("%s/src/lib.rs", dirName)
	if meta.LibSrc != wantLibSrc {
		t.Errorf("LibSrc: got %q, want %q", meta.LibSrc, wantLibSrc)
	}
	if meta.Name != "mycrate" {
		t.Errorf("Name: got %q, want %q", meta.Name, "mycrate")
	}
	if meta.Version != "1.2.3" {
		t.Errorf("Version: got %q, want %q", meta.Version, "1.2.3")
	}
}

func TestBuildCrateMeta_LibSectionPath(t *testing.T) {
	tmpDir := t.TempDir()
	dirName := "mylib-0.1.0"
	extractedDir := filepath.Join(tmpDir, dirName)
	if err := os.MkdirAll(extractedDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Cargo.toml with a [lib] path override and no edition (should default to 2015).
	toml := `[package]
name = "mylib"
version = "0.1.0"

[lib]
name = "mylib"
path = "lib.rs"
`
	if err := os.WriteFile(filepath.Join(extractedDir, "Cargo.toml"), []byte(toml), 0644); err != nil {
		t.Fatal(err)
	}

	meta, err := buildCrateMeta("mylib", "0.1.0", extractedDir, dirName)
	if err != nil {
		t.Fatalf("buildCrateMeta returned error: %v", err)
	}

	if meta.Edition != "2015" {
		t.Errorf("edition: got %q, want %q", meta.Edition, "2015")
	}
	wantLibSrc := fmt.Sprintf("%s/lib.rs", dirName)
	if meta.LibSrc != wantLibSrc {
		t.Errorf("LibSrc: got %q, want %q", meta.LibSrc, wantLibSrc)
	}
}

func TestBuildCrateMeta_FallbackToLibRs(t *testing.T) {
	tmpDir := t.TempDir()
	dirName := "oldcrate-0.0.1"
	extractedDir := filepath.Join(tmpDir, dirName)
	if err := os.MkdirAll(extractedDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Cargo.toml with no edition and no [lib] section; only lib.rs at root.
	toml := `[package]
name = "oldcrate"
version = "0.0.1"
`
	if err := os.WriteFile(filepath.Join(extractedDir, "Cargo.toml"), []byte(toml), 0644); err != nil {
		t.Fatal(err)
	}
	// Create lib.rs at root (no src/lib.rs).
	if err := os.WriteFile(filepath.Join(extractedDir, "lib.rs"), []byte("// lib"), 0644); err != nil {
		t.Fatal(err)
	}

	meta, err := buildCrateMeta("oldcrate", "0.0.1", extractedDir, dirName)
	if err != nil {
		t.Fatalf("buildCrateMeta returned error: %v", err)
	}

	if meta.Edition != "2015" {
		t.Errorf("edition: got %q, want %q", meta.Edition, "2015")
	}
	wantLibSrc := fmt.Sprintf("%s/lib.rs", dirName)
	if meta.LibSrc != wantLibSrc {
		t.Errorf("LibSrc: got %q, want %q", meta.LibSrc, wantLibSrc)
	}
}

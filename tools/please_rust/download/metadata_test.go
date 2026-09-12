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

	toml := `[package]
name = "mycrate"
version = "1.2.3"
edition = "2021"
`
	if err := os.WriteFile(filepath.Join(extractedDir, "Cargo.toml"), []byte(toml), 0644); err != nil {
		t.Fatal(err)
	}
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

	toml := `[package]
name = "oldcrate"
version = "0.0.1"
`
	if err := os.WriteFile(filepath.Join(extractedDir, "Cargo.toml"), []byte(toml), 0644); err != nil {
		t.Fatal(err)
	}
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

func TestBuildCrateMeta_MissingCargoToml(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := buildCrateMeta("foo", "1.0.0", tmpDir, "foo-1.0.0")
	if err == nil {
		t.Fatal("expected error for missing Cargo.toml, got nil")
	}
}

func TestBuildCrateMeta_NoLibFiles(t *testing.T) {
	tmpDir := t.TempDir()
	cargoToml := `[package]
name = "nolib"
version = "1.0.0"
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(cargoToml), 0644); err != nil {
		t.Fatal(err)
	}
	meta, err := buildCrateMeta("nolib", "1.0.0", tmpDir, "nolib-1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.LibSrc != "nolib-1.0.0/src/lib.rs" {
		t.Errorf("got %s, want nolib-1.0.0/src/lib.rs", meta.LibSrc)
	}
}

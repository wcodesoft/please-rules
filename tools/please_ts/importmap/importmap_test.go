package importmap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSynthesizeWithModuleName(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "importmap_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	src1 := filepath.Join(tmpDir, "calculator.ts")
	if err := os.WriteFile(src1, []byte("export function add(a: number, b: number) { return a + b; }"), 0644); err != nil {
		t.Fatal(err)
	}

	im, err := Synthesize("@domain/calculator", []string{src1}, nil, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if entry, ok := im.Imports["@domain/calculator"]; !ok || entry != "./calculator.ts" {
		t.Errorf("expected @domain/calculator to be ./calculator.ts, got %q", entry)
	}
}

func TestSynthesizeWithMetadataDep(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "importmap_meta_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	depDir := filepath.Join(tmpDir, "third_party", "preact")
	if err := os.MkdirAll(depDir, 0755); err != nil {
		t.Fatal(err)
	}

	distDir := filepath.Join(depDir, "dist")
	if err := os.MkdirAll(distDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(distDir, "preact.mjs"), []byte("export const h = () => {};"), 0644); err != nil {
		t.Fatal(err)
	}

	metaJSON := `{
		"name": "preact",
		"entry": "dist/preact.mjs",
		"types": "dist/preact.d.ts"
	}`
	if err := os.WriteFile(filepath.Join(depDir, "ts_module.json"), []byte(metaJSON), 0644); err != nil {
		t.Fatal(err)
	}

	appDir := filepath.Join(tmpDir, "src")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	src1 := filepath.Join(appDir, "app.ts")
	if err := os.WriteFile(src1, []byte("import { h } from 'preact';"), 0644); err != nil {
		t.Fatal(err)
	}

	im, err := Synthesize("", []string{src1}, []string{depDir}, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedEntry := "./third_party/preact/dist/preact.mjs"
	if entry, ok := im.Imports["preact"]; !ok || entry != expectedEntry {
		t.Errorf("expected preact to be %q, got %q", expectedEntry, entry)
	}
}

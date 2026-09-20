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

func TestSynthesizePackageSubpaths(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "importmap_subpaths_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	badge := filepath.Join(tmpDir, "Badge.ts")
	_ = os.WriteFile(badge, []byte("export const Badge = 'badge';"), 0644)
	card := filepath.Join(tmpDir, "MetricCard.ts")
	_ = os.WriteFile(card, []byte("export const MetricCard = 'card';"), 0644)
	modal := filepath.Join(tmpDir, "Modal.ts")
	_ = os.WriteFile(modal, []byte("export const Modal = 'modal';"), 0644)

	srcs := []string{badge, card, modal}
	moduleName := "@repo/dashboard/components"

	im, err := Synthesize(moduleName, srcs, nil, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedMappings := map[string]string{
		"@repo/dashboard/components/Badge":         "./Badge.ts",
		"@repo/dashboard/components/Badge.ts":      "./Badge.ts",
		"@repo/dashboard/components/MetricCard":    "./MetricCard.ts",
		"@repo/dashboard/components/MetricCard.ts": "./MetricCard.ts",
		"@repo/dashboard/components/Modal":         "./Modal.ts",
		"@repo/dashboard/components/Modal.ts":      "./Modal.ts",
	}

	for key, want := range expectedMappings {
		got, ok := im.Imports[key]
		if !ok {
			t.Errorf("missing import key %q", key)
		} else if got != want {
			t.Errorf("import[%q] = %q, want %q", key, got, want)
		}
	}
}

func TestLoadMetadataPackageSubpaths(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "importmap_load_subpaths_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	depDir := filepath.Join(tmpDir, "plz-out", "gen", "components")
	_ = os.MkdirAll(depDir, 0755)

	_ = os.WriteFile(filepath.Join(depDir, "Badge.ts"), []byte("export const Badge = 'badge';"), 0644)
	_ = os.WriteFile(filepath.Join(depDir, "MetricCard.ts"), []byte("export const MetricCard = 'card';"), 0644)

	metaJSON := `{
		"name": "@repo/dashboard/components",
		"entry": "Badge.ts",
		"files": ["Badge.ts", "MetricCard.ts"]
	}`
	_ = os.WriteFile(filepath.Join(depDir, "ts_metadata.json"), []byte(metaJSON), 0644)

	appDir := filepath.Join(tmpDir, "src")
	_ = os.MkdirAll(appDir, 0755)
	appSrc := filepath.Join(appDir, "app.ts")
	_ = os.WriteFile(appSrc, []byte("import { Badge } from '@repo/dashboard/components/Badge';"), 0644)

	im, err := Synthesize("", []string{appSrc}, []string{depDir}, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedSubpaths := map[string]string{
		"@repo/dashboard/components/Badge":         "./plz-out/gen/components/Badge.ts",
		"@repo/dashboard/components/Badge.ts":      "./plz-out/gen/components/Badge.ts",
		"@repo/dashboard/components/MetricCard":    "./plz-out/gen/components/MetricCard.ts",
		"@repo/dashboard/components/MetricCard.ts": "./plz-out/gen/components/MetricCard.ts",
	}

	for key, want := range expectedSubpaths {
		got, ok := im.Imports[key]
		if !ok {
			t.Errorf("missing subpath import key %q", key)
		} else if got != want {
			t.Errorf("import[%q] = %q, want %q", key, got, want)
		}
	}
}

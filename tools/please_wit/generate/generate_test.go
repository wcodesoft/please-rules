package generate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverWorlds(t *testing.T) {
	tmpDir := t.TempDir()
	witContent := `
package test:demo;

interface foo {
    bar: func();
}

world my-world-1 {
    export foo;
}

world my-world-2 {
    import foo;
}
`
	witFile := filepath.Join(tmpDir, "demo.wit")
	if err := os.WriteFile(witFile, []byte(witContent), 0644); err != nil {
		t.Fatalf("failed to write test wit file: %v", err)
	}

	worlds, err := DiscoverWorlds(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverWorlds failed: %v", err)
	}

	if len(worlds) != 2 {
		t.Fatalf("expected 2 worlds, got %d: %v", len(worlds), worlds)
	}
	if worlds[0] != "my-world-1" || worlds[1] != "my-world-2" {
		t.Errorf("unexpected worlds: %v", worlds)
	}
}

func TestGeneratorForLang(t *testing.T) {
	cases := map[string]string{
		"rust":   "rust",
		"go":     "go",
		"cpp":    "cpp",
		"cc":     "cpp",
		"c":      "c",
		"swift":  "c",
		"kotlin": "c",
		"ts":     "c",
		"python": "c",
	}

	for lang, expected := range cases {
		got := GeneratorForLang(lang)
		if got != expected {
			t.Errorf("GeneratorForLang(%q) = %q; want %q", lang, got, expected)
		}
	}
}

func TestGenerateCompanions(t *testing.T) {
	tmpDir := t.TempDir()

	for _, lang := range []string{"swift", "kotlin", "ts", "python"} {
		subDir := filepath.Join(tmpDir, lang)
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("failed to mkdir: %v", err)
		}
		if err := GenerateCompanions(lang, subDir); err != nil {
			t.Fatalf("GenerateCompanions failed for %s: %v", lang, err)
		}
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "swift", "WitBridging.swift")); err != nil {
		t.Errorf("Swift companion missing")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "kotlin", "WitBindings.kt")); err != nil {
		t.Errorf("Kotlin companion missing")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "ts", "index.d.ts")); err != nil {
		t.Errorf("TypeScript companion missing")
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "python", "__init__.py")); err != nil {
		t.Errorf("Python companion missing")
	}
}

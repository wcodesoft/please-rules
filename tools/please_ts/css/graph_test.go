package css

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tools/please_ts/importmap"
)

func TestSplitSources(t *testing.T) {
	tsSrcs, cssSrcs := SplitSources([]string{"a.ts", "b.css", "c.tsx", "STYLE.CSS"})
	if !reflect.DeepEqual(tsSrcs, []string{"a.ts", "c.tsx"}) {
		t.Errorf("tsSrcs = %v, want [a.ts c.tsx]", tsSrcs)
	}
	if !reflect.DeepEqual(cssSrcs, []string{"b.css", "STYLE.CSS"}) {
		t.Errorf("cssSrcs = %v, want [b.css STYLE.CSS]", cssSrcs)
	}
}

func writeModuleFixture(t *testing.T, dir string, meta importmap.ModuleMetadata, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ts_metadata.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDiscoverGraphFilesOrdersDependenciesAndDeduplicatesDiamonds(t *testing.T) {
	root := t.TempDir()

	// Diamond graph:
	//   @ui/tokens (tokens.css)
	//    /      \
	// @ui/btn   @ui/card
	//    \      /
	//    page.css (direct source)
	tokensDir := filepath.Join(root, "ui", "tokens")
	writeModuleFixture(t, tokensDir, importmap.ModuleMetadata{
		Name:     "@ui/tokens",
		Entry:    "tokens.ts",
		Files:    []string{"tokens.ts", "tokens.css"},
		CSSFiles: []string{"tokens.css"},
	}, map[string]string{
		"tokens.ts":  "export const t = 1;",
		"tokens.css": ":root { --gap: 8px; }",
	})

	btnDir := filepath.Join(root, "ui", "btn")
	writeModuleFixture(t, btnDir, importmap.ModuleMetadata{
		Name:     "@ui/btn",
		Entry:    "btn.ts",
		Files:    []string{"btn.ts", "btn.css"},
		CSSFiles: []string{"btn.css"},
		Imports: map[string]string{
			"@ui/tokens": "./ui/tokens/tokens.ts",
		},
	}, map[string]string{
		"btn.ts":  "export const b = 1;",
		"btn.css": ".btn { gap: var(--gap); }",
	})

	// Also simulate an embedded copy of @ui/tokens inside @ui/btn (as ts_library --embed-deps produces)
	embeddedTokensDir := filepath.Join(btnDir, "ui", "tokens")
	writeModuleFixture(t, embeddedTokensDir, importmap.ModuleMetadata{
		Name:     "@ui/tokens",
		Entry:    "tokens.ts",
		Files:    []string{"tokens.ts", "tokens.css"},
		CSSFiles: []string{"tokens.css"},
	}, map[string]string{
		"tokens.ts":  "export const t = 1;",
		"tokens.css": ":root { --gap: 8px; }",
	})

	cardDir := filepath.Join(root, "ui", "card")
	writeModuleFixture(t, cardDir, importmap.ModuleMetadata{
		Name:     "@ui/card",
		Entry:    "card.ts",
		Files:    []string{"card.ts", "card.css"},
		CSSFiles: []string{"card.css"},
		Imports: map[string]string{
			"@ui/tokens": "./ui/tokens/tokens.ts",
		},
	}, map[string]string{
		"card.ts":  "export const c = 1;",
		"card.css": ".card { gap: var(--gap); }",
	})

	pageCSS := filepath.Join(root, "page.css")
	if err := os.WriteFile(pageCSS, []byte(".page { display: grid; }"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := DiscoverGraphFiles([]string{btnDir, cardDir}, []string{pageCSS}, root)
	if err != nil {
		t.Fatalf("DiscoverGraphFiles failed: %v", err)
	}

	want := []string{
		filepath.Join(tokensDir, "tokens.css"),
		filepath.Join(btnDir, "btn.css"),
		filepath.Join(cardDir, "card.css"),
		pageCSS,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DiscoverGraphFiles =\n  %v\nwant:\n  %v", got, want)
	}
}

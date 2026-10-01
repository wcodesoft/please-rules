package testrunner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tools/please_ts/importmap"
)

func TestBuildVitestAliasesMatrix(t *testing.T) {
	tests := []struct {
		name      string
		imports   map[string]string
		wantKeys  []string
		doNotWant []string
	}{
		{
			name: "filters out vitest and chai internal packages",
			imports: map[string]string{
				"vitest":                 "npm:vitest",
				"vitest/browser":         "npm:vitest/browser",
				"chai":                   "npm:chai",
				"@domain/calc":           "./lib/calc.ts",
				"@repo/components/Badge": "./components/Badge.ts",
				"prefix/":                "./prefix/",
			},
			wantKeys: []string{
				"@domain/calc",
				"@repo/components/Badge",
			},
			doNotWant: []string{
				"vitest",
				"vitest/browser",
				"chai",
				"prefix/",
			},
		},
		{
			name:      "nil import map returns empty map",
			imports:   nil,
			wantKeys:  nil,
			doNotWant: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var im *importmap.ImportMap
			if tt.imports != nil {
				im = importmap.New()
				im.Imports = tt.imports
			}
			aliases := buildVitestAliases(im)

			for _, key := range tt.wantKeys {
				if _, ok := aliases[key]; !ok {
					t.Errorf("buildVitestAliases missing key %q", key)
				}
			}
			for _, notWant := range tt.doNotWant {
				if _, exists := aliases[notWant]; exists {
					t.Errorf("buildVitestAliases should NOT contain %q", notWant)
				}
			}
		})
	}
}

func TestWriteVitestConfigMatrix(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vitest_config_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "vitest.config.mjs")
	srcs := []string{"/abs/test1.ts", "/abs/test2.ts"}
	aliases := map[string]string{
		"@domain/calc": "/abs/calc.ts",
	}

	if err := writeVitestConfig(configPath, srcs, aliases); err != nil {
		t.Fatalf("writeVitestConfig failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	wantSnippets := []string{
		`export default {`,
		`globals: true`,
		`"/abs/test1.ts"`,
		`"find": "^@domain/calc$"`,
		`"replacement": "/abs/calc.ts"`,
	}
	for _, want := range wantSnippets {
		if !strings.Contains(content, want) {
			t.Errorf("config missing %q in:\n%s", want, content)
		}
	}
}

func TestWriteVitestConfigNestedAliasOrdering(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "vitest.config.mjs")
	aliases := map[string]string{
		"@scope/app":                   "/abs/app.ts",
		"@scope/app/components/widget": "/abs/widget.ts",
	}
	if err := writeVitestConfig(configPath, []string{"/abs/t.ts"}, aliases); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(configPath)
	content := string(data)

	// Exact-match anchors prevent "@scope/app" from capturing the nested name.
	for _, want := range []string{`"^@scope/app$"`, `"^@scope/app/components/widget$"`} {
		if !strings.Contains(content, want) {
			t.Errorf("config missing %q in:\n%s", want, content)
		}
	}
	if strings.Index(content, "components/widget") > strings.Index(content, `"^@scope/app$"`) {
		t.Errorf("longer alias should precede shorter alias in:\n%s", content)
	}
}

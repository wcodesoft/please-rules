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
		`"@domain/calc": "/abs/calc.ts"`,
	}
	for _, want := range wantSnippets {
		if !strings.Contains(content, want) {
			t.Errorf("config missing %q in:\n%s", want, content)
		}
	}
}

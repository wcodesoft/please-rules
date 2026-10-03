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

func TestVitestCoverageArgsUseDedicatedDirectory(t *testing.T) {
	args := strings.Join(vitestCoverageArgs("/tmp/run/coverage"), " ")
	for _, want := range []string{"--coverage.enabled", "--coverage.reporter=lcov", "--coverage.reportsDirectory=/tmp/run/coverage"} {
		if !strings.Contains(args, want) {
			t.Errorf("vitestCoverageArgs missing %q in %q", want, args)
		}
	}
}

func TestConvertVitestCoverage(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	lcov := "SF:" + filepath.Join(cwd, "lib.ts") + "\nDA:1,1\nDA:2,0\nend_of_record\n"
	if err := os.WriteFile(filepath.Join(dir, "lcov.info"), []byte(lcov), 0644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "sub", "test.coverage")
	if err := convertVitestCoverage(dir, out, ""); err != nil {
		t.Fatalf("convertVitestCoverage: %v", err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`filename="lib.ts"`, `<line number="1" hits="1"/>`, `<line number="2" hits="0"/>`} {
		if !strings.Contains(string(got), want) {
			t.Errorf("coverage file missing %q:\n%s", want, got)
		}
	}
}

func TestConvertVitestCoverageMissingReport(t *testing.T) {
	if err := convertVitestCoverage(t.TempDir(), filepath.Join(t.TempDir(), "out"), ""); err == nil {
		t.Error("expected error when vitest wrote no lcov.info")
	}
}

func TestConvertVitestCoverageExportsRawLcov(t *testing.T) {
	dir := t.TempDir()
	cwd, _ := os.Getwd()
	lcov := "TN:\nSF:" + filepath.Join(cwd, "lib.ts") + "\nFN:1,classify\nFNDA:2,classify\nDA:2,2\nDA:3,0\nBRDA:2,0,0,2\nBRDA:2,0,1,0\nend_of_record\n"
	if err := os.WriteFile(filepath.Join(dir, "lcov.info"), []byte(lcov), 0644); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	out, raw := filepath.Join(root, "test.coverage"), filepath.Join(root, "sub", "target.lcov")
	if err := convertVitestCoverage(dir, out, raw); err != nil {
		t.Fatalf("convertVitestCoverage: %v", err)
	}
	got, err := os.ReadFile(raw)
	if err != nil {
		t.Fatalf("raw lcov not written: %v", err)
	}
	for _, want := range []string{"SF:lib.ts\n", "FN:1,classify\n", "FNDA:2,classify\n", "BRDA:2,0,1,0\n", "DA:3,0\n"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("raw lcov missing %q:\n%s", want, got)
		}
	}
}

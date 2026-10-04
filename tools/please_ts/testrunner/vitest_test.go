package testrunner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tools/please_ts/importmap"
	"tools/please_ts/npmcache"
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

func TestBuildVitestNpmDependencies(t *testing.T) {
	im := importmap.New()
	im.Imports = map[string]string{
		"debug":                     "npm:debug@4.3.7",
		"debug/":                    "npm:/debug@4.3.7/",
		"@codemirror/legacy-modes":  "npm:@codemirror/legacy-modes@6.5.1",
		"@codemirror/legacy-modes/": "npm:/@codemirror/legacy-modes@6.5.1/",
		"@test/lib":                 "/abs/lib.ts",
	}
	got := buildVitestNpmDependencies(im)
	want := map[string]string{"debug": "4.3.7", "@codemirror/legacy-modes": "6.5.1"}
	if len(got) != len(want) || got["debug"] != want["debug"] || got["@codemirror/legacy-modes"] != want["@codemirror/legacy-modes"] {
		t.Errorf("got %v, want %v", got, want)
	}
	if len(buildVitestNpmDependencies(nil)) != 0 || len(buildVitestNpmDependencies(importmap.New())) != 0 {
		t.Error("expected no dependencies")
	}
}

func TestBuildVitestAliasesLeavesNpmPackagesToTheirNodeModules(t *testing.T) {
	im := importmap.New()
	im.Imports = map[string]string{"debug": "npm:debug@4.3.7", "@test/lib": "lib/lib.ts"}
	aliases := buildVitestAliases(im)
	if _, ok := aliases["debug"]; ok {
		t.Errorf("an npm: target became a file alias: %v", aliases)
	}
	if _, ok := aliases["@test/lib"]; !ok {
		t.Errorf("first-party alias missing: %v", aliases)
	}
}

func inTempDir(t *testing.T) {
	t.Helper()
	old, _ := os.Getwd()
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

func cachedPackage(t *testing.T, denoDir, name, version string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(denoDir, "npm", "registry.npmjs.org", filepath.FromSlash(name), version), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestWriteVitestPackageJSON(t *testing.T) {
	cache := t.TempDir()
	cachedPackage(t, cache, "vitest", "5.0.1")
	cachedPackage(t, cache, "@vitest/coverage-v8", "5.0.1")

	read := func() map[string]map[string]string {
		data, err := os.ReadFile("package.json")
		if err != nil {
			t.Fatal(err)
		}
		var p map[string]map[string]string
		if err := json.Unmarshal(data, &p); err != nil {
			t.Fatal(err)
		}
		return p
	}

	t.Run("nothing to declare writes nothing", func(t *testing.T) {
		inTempDir(t)
		if err := writeVitestPackageJSON(cache, false, nil); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat("package.json"); err == nil {
			t.Error("package.json was written")
		}
	})

	t.Run("coverage declares the cached versions, not latest", func(t *testing.T) {
		inTempDir(t)
		if err := writeVitestPackageJSON(cache, true, nil); err != nil {
			t.Fatal(err)
		}
		deps := read()["dependencies"]
		if deps["vitest"] != "5.0.1" || deps["@vitest/coverage-v8"] != "5.0.1" || len(deps) != 2 {
			t.Errorf("deps = %v", deps)
		}
	})

	t.Run("npm packages are declared with their exact version", func(t *testing.T) {
		inTempDir(t)
		if err := writeVitestPackageJSON(cache, true, map[string]string{"debug": "4.3.7"}); err != nil {
			t.Fatal(err)
		}
		deps := read()["dependencies"]
		if deps["debug"] != "4.3.7" || deps["vitest"] != "5.0.1" || len(deps) != 3 {
			t.Errorf("deps = %v", deps)
		}
	})

	t.Run("a package.json the target ships is kept", func(t *testing.T) {
		inTempDir(t)
		_ = os.WriteFile("package.json", []byte(`{"name":"mine"}`), 0644)
		if err := writeVitestPackageJSON(cache, true, map[string]string{"debug": "4.3.7"}); err != nil {
			t.Fatal(err)
		}
		if data, _ := os.ReadFile("package.json"); string(data) != `{"name":"mine"}` {
			t.Errorf("package.json = %s", data)
		}
	})

	t.Run("several cached versions are an error", func(t *testing.T) {
		inTempDir(t)
		two := t.TempDir()
		cachedPackage(t, two, "vitest", "5.0.1")
		cachedPackage(t, two, "vitest", "5.0.3")
		if err := writeVitestPackageJSON(two, true, nil); err == nil || !strings.Contains(err.Error(), "several versions") {
			t.Errorf("err = %v", err)
		}
	})
}

func TestVitestDenoDir(t *testing.T) {
	shared := t.TempDir()
	cachedPackage(t, shared, "vitest", "5.0.1")
	opts := RunOptions{VitestDir: shared}

	t.Run("without npm packages it is the shared cache", func(t *testing.T) {
		got, err := opts.vitestDenoDir(t.TempDir(), false)
		if err != nil || got != shared {
			t.Errorf("got %q, %v; want %q", got, err, shared)
		}
	})

	t.Run("with npm packages it is a per-run copy that also holds the slices", func(t *testing.T) {
		dir := t.TempDir()
		old, _ := os.Getwd()
		_ = os.Chdir(dir)
		t.Cleanup(func() { _ = os.Chdir(old) })
		slice := filepath.Join(dir, "third_party", "debug")
		cachedPackage(t, slice, "debug", "4.3.7")
		if err := npmcache.Write(slice, npmcache.Slice{Name: "debug", Version: "4.3.7", Specifier: "npm:debug@4.3.7"}); err != nil {
			t.Fatal(err)
		}

		tmp := t.TempDir()
		got, err := opts.vitestDenoDir(tmp, true)
		if err != nil {
			t.Fatal(err)
		}
		if got == shared {
			t.Fatal("the shared cache would be written to")
		}
		for _, pkg := range []string{"vitest/5.0.1", "debug/4.3.7"} {
			if _, err := os.Stat(filepath.Join(got, "npm", "registry.npmjs.org", filepath.FromSlash(pkg))); err != nil {
				t.Errorf("the per-run cache lacks %s: %v", pkg, err)
			}
		}
		if _, err := os.Stat(filepath.Join(shared, "npm", "registry.npmjs.org", "debug")); err == nil {
			t.Error("the slice was merged into the shared cache")
		}
	})
}

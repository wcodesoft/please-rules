package importmap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSynthesizeTargetSourcesMatrix(t *testing.T) {
	tests := []struct {
		name        string
		moduleName  string
		files       map[string]string
		srcs        []string
		wantImports map[string]string
	}{
		{
			name:       "single entry file matches module name",
			moduleName: "@domain/calculator",
			files: map[string]string{
				"calculator.ts": "export function add(a: number, b: number) { return a + b; }",
			},
			srcs: []string{"calculator.ts"},
			wantImports: map[string]string{
				"@domain/calculator":               "./calculator.ts",
				"@domain/calculator/":              "./",
				"@domain/calculator/calculator":    "./calculator.ts",
				"@domain/calculator/calculator.ts": "./calculator.ts",
			},
		},
		{
			name:       "multiple source files with subpaths",
			moduleName: "@repo/dashboard/components",
			files: map[string]string{
				"Badge.ts":      "export const Badge = 'badge';",
				"MetricCard.ts": "export const MetricCard = 'card';",
				"Modal.ts":      "export const Modal = 'modal';",
			},
			srcs: []string{"Badge.ts", "MetricCard.ts", "Modal.ts"},
			wantImports: map[string]string{
				"@repo/dashboard/components/Badge":         "./Badge.ts",
				"@repo/dashboard/components/Badge.ts":      "./Badge.ts",
				"@repo/dashboard/components/MetricCard":    "./MetricCard.ts",
				"@repo/dashboard/components/MetricCard.ts": "./MetricCard.ts",
				"@repo/dashboard/components/Modal":         "./Modal.ts",
				"@repo/dashboard/components/Modal.ts":      "./Modal.ts",
			},
		},
		{
			name:       "empty module name skips root mapping",
			moduleName: "",
			files: map[string]string{
				"app.ts": "console.log('hello');",
			},
			srcs:        []string{"app.ts"},
			wantImports: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "importmap_target_*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			var absSrcs []string
			for relPath, content := range tt.files {
				fullPath := filepath.Join(tmpDir, relPath)
				_ = os.MkdirAll(filepath.Dir(fullPath), 0755)
				if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
			}
			for _, s := range tt.srcs {
				absSrcs = append(absSrcs, filepath.Join(tmpDir, s))
			}

			im, err := Synthesize(tt.moduleName, absSrcs, nil, tmpDir)
			if err != nil {
				t.Fatalf("Synthesize failed: %v", err)
			}

			for key, want := range tt.wantImports {
				got, ok := im.Imports[key]
				if !ok {
					t.Errorf("import[%q] missing, want %q", key, want)
				} else if got != want {
					t.Errorf("import[%q] = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestSynthesizeMetadataDepsMatrix(t *testing.T) {
	tests := []struct {
		name        string
		depRelDir   string
		metaFile    string
		metaJSON    string
		files       map[string]string
		appSrc      string
		appContent  string
		wantImports map[string]string
	}{
		{
			name:      "module metadata with entry point",
			depRelDir: "third_party/preact",
			metaFile:  "ts_module.json",
			metaJSON: `{
				"name": "preact",
				"entry": "dist/preact.mjs",
				"types": "dist/preact.d.ts"
			}`,
			files: map[string]string{
				"dist/preact.mjs": "export const h = () => {};",
			},
			appSrc:     "src/app.ts",
			appContent: "import { h } from 'preact';",
			wantImports: map[string]string{
				"preact":  "./third_party/preact/dist/preact.mjs",
				"preact/": "./third_party/preact/dist/",
			},
		},
		{
			name:      "library metadata with multiple subpath files",
			depRelDir: "plz-out/gen/components",
			metaFile:  "ts_metadata.json",
			metaJSON: `{
				"name": "@repo/dashboard/components",
				"entry": "Badge.ts",
				"files": ["Badge.ts", "MetricCard.ts"]
			}`,
			files: map[string]string{
				"Badge.ts":      "export const Badge = 'badge';",
				"MetricCard.ts": "export const MetricCard = 'card';",
			},
			appSrc:     "src/app.ts",
			appContent: "import { Badge } from '@repo/dashboard/components/Badge';",
			wantImports: map[string]string{
				"@repo/dashboard/components/Badge":         "./plz-out/gen/components/Badge.ts",
				"@repo/dashboard/components/Badge.ts":      "./plz-out/gen/components/Badge.ts",
				"@repo/dashboard/components/MetricCard":    "./plz-out/gen/components/MetricCard.ts",
				"@repo/dashboard/components/MetricCard.ts": "./plz-out/gen/components/MetricCard.ts",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "importmap_depmeta_*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			depDir := filepath.Join(tmpDir, tt.depRelDir)
			_ = os.MkdirAll(depDir, 0755)

			for relPath, content := range tt.files {
				p := filepath.Join(depDir, relPath)
				_ = os.MkdirAll(filepath.Dir(p), 0755)
				_ = os.WriteFile(p, []byte(content), 0644)
			}
			_ = os.WriteFile(filepath.Join(depDir, tt.metaFile), []byte(tt.metaJSON), 0644)

			appPath := filepath.Join(tmpDir, tt.appSrc)
			_ = os.MkdirAll(filepath.Dir(appPath), 0755)
			_ = os.WriteFile(appPath, []byte(tt.appContent), 0644)

			im, err := Synthesize("", []string{appPath}, []string{depDir}, tmpDir)
			if err != nil {
				t.Fatalf("Synthesize failed: %v", err)
			}

			for key, want := range tt.wantImports {
				got, ok := im.Imports[key]
				if !ok {
					t.Errorf("import[%q] missing, want %q", key, want)
				} else if got != want {
					t.Errorf("import[%q] = %q, want %q", key, got, want)
				}
			}
		})
	}
}

func TestSynthesizeScopesAndPrecedenceMatrix(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(tmpDir string) (srcs, deps []string)
		wantImports    map[string]string
		wantScopes     map[string]map[string]string
		wantNotImports []string
	}{
		{
			name: "explicit root target overrides internal module scope",
			setup: func(tmpDir string) ([]string, []string) {
				// 1. Explicit third_party/react
				reactDir := filepath.Join(tmpDir, "third_party", "react")
				_ = os.MkdirAll(reactDir, 0755)
				_ = os.WriteFile(filepath.Join(reactDir, "index.js"), []byte("module.exports = { v: '18' };"), 0644)
				_ = os.WriteFile(filepath.Join(reactDir, "ts_module.json"), []byte(`{"name":"react","entry":"index.js"}`), 0644)

				// 2. third_party/react-router with internal .deps/react
				routerDir := filepath.Join(tmpDir, "third_party", "react-router")
				routerInternalReact := filepath.Join(routerDir, ".deps", "react")
				_ = os.MkdirAll(routerInternalReact, 0755)
				_ = os.WriteFile(filepath.Join(routerDir, "index.js"), []byte("const r = require('react');"), 0644)
				_ = os.WriteFile(filepath.Join(routerInternalReact, "index.js"), []byte("module.exports = { internal: true };"), 0644)
				_ = os.WriteFile(filepath.Join(routerDir, "ts_module.json"), []byte(`{
					"name": "react-router",
					"entry": "index.js",
					"deps": {"react": "./.deps/react/index.js"}
				}`), 0644)

				appSrc := filepath.Join(tmpDir, "src", "app.ts")
				_ = os.MkdirAll(filepath.Dir(appSrc), 0755)
				_ = os.WriteFile(appSrc, []byte("import React from 'react';"), 0644)

				return []string{appSrc}, []string{reactDir, routerDir}
			},
			wantImports: map[string]string{
				"react":        "./third_party/react/index.js",
				"react-router": "./third_party/react-router/index.js",
			},
			wantScopes: map[string]map[string]string{
				"./third_party/react-router/": {
					"react": "./third_party/react/index.js",
				},
			},
		},
		{
			name: "conflicting internal dependencies remain isolated in distinct scopes",
			setup: func(tmpDir string) ([]string, []string) {
				// express has internal debug@2
				expressDir := filepath.Join(tmpDir, "third_party", "express")
				_ = os.MkdirAll(filepath.Join(expressDir, ".deps", "debug"), 0755)
				_ = os.WriteFile(filepath.Join(expressDir, "index.js"), []byte("require('debug');"), 0644)
				_ = os.WriteFile(filepath.Join(expressDir, ".deps", "debug", "index.js"), []byte("module.exports = 'v2';"), 0644)
				_ = os.WriteFile(filepath.Join(expressDir, "ts_module.json"), []byte(`{
					"name": "express",
					"entry": "index.js",
					"deps": {"debug": "./.deps/debug/index.js"}
				}`), 0644)

				// cors has internal debug@4
				corsDir := filepath.Join(tmpDir, "third_party", "cors")
				_ = os.MkdirAll(filepath.Join(corsDir, ".deps", "debug"), 0755)
				_ = os.WriteFile(filepath.Join(corsDir, "index.js"), []byte("require('debug');"), 0644)
				_ = os.WriteFile(filepath.Join(corsDir, ".deps", "debug", "index.js"), []byte("module.exports = 'v4';"), 0644)
				_ = os.WriteFile(filepath.Join(corsDir, "ts_module.json"), []byte(`{
					"name": "cors",
					"entry": "index.js",
					"deps": {"debug": "./.deps/debug/index.js"}
				}`), 0644)

				appSrc := filepath.Join(tmpDir, "src", "app.ts")
				_ = os.MkdirAll(filepath.Dir(appSrc), 0755)
				_ = os.WriteFile(appSrc, []byte("import express from 'express';"), 0644)

				return []string{appSrc}, []string{expressDir, corsDir}
			},
			wantImports: map[string]string{
				"express": "./third_party/express/index.js",
				"cors":    "./third_party/cors/index.js",
			},
			wantScopes: map[string]map[string]string{
				"./third_party/express/": {
					"debug": "./third_party/express/.deps/debug/index.js",
				},
				"./third_party/cors/": {
					"debug": "./third_party/cors/.deps/debug/index.js",
				},
			},
			wantNotImports: []string{"debug"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "importmap_scopes_matrix_*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			srcs, deps := tt.setup(tmpDir)

			im, err := Synthesize("", srcs, deps, tmpDir)
			if err != nil {
				t.Fatalf("Synthesize failed: %v", err)
			}

			// Validate top-level imports
			for key, want := range tt.wantImports {
				got, ok := im.Imports[key]
				if !ok {
					t.Errorf("import[%q] missing, want %q", key, want)
				} else if got != want {
					t.Errorf("import[%q] = %q, want %q", key, got, want)
				}
			}

			// Validate scopes
			for scopeKey, expectedMappings := range tt.wantScopes {
				scopeMap, ok := im.Scopes[scopeKey]
				if !ok {
					t.Fatalf("scope[%q] missing", scopeKey)
				}
				for key, want := range expectedMappings {
					got, ok := scopeMap[key]
					if !ok {
						t.Errorf("scope[%q][%q] missing, want %q", scopeKey, key, want)
					} else if got != want {
						t.Errorf("scope[%q][%q] = %q, want %q", scopeKey, key, got, want)
					}
				}
			}

			// Validate prohibited leaked imports
			for _, notWant := range tt.wantNotImports {
				if got, leaked := im.Imports[notWant]; leaked {
					t.Errorf("expected %q NOT to be in root imports, but got %q", notWant, got)
				}
			}
		})
	}
}

func TestSynthesizeSingletonHoistingMatrix(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(tmpDir string) (srcs, deps []string)
		wantImports map[string]string
		wantScopes  map[string]map[string]string
	}{
		{
			name: "undeclared peer dependency is hoisted to root imports",
			setup: func(tmpDir string) ([]string, []string) {
				libADir := filepath.Join(tmpDir, "third_party", "libA")
				_ = os.MkdirAll(filepath.Join(libADir, ".deps", "react"), 0755)
				_ = os.WriteFile(filepath.Join(libADir, "index.js"), []byte("export const a = 1;"), 0644)
				_ = os.WriteFile(filepath.Join(libADir, ".deps", "react", "index.js"), []byte("export const React = {};"), 0644)
				_ = os.WriteFile(filepath.Join(libADir, "ts_module.json"), []byte(`{
					"name": "libA",
					"entry": "index.js",
					"peer_deps": {"react": "^18.0.0"},
					"deps": {"react": "./.deps/react/index.js"}
				}`), 0644)

				libBDir := filepath.Join(tmpDir, "third_party", "libB")
				_ = os.MkdirAll(libBDir, 0755)
				_ = os.WriteFile(filepath.Join(libBDir, "index.js"), []byte("export const b = 2;"), 0644)
				_ = os.WriteFile(filepath.Join(libBDir, "ts_module.json"), []byte(`{
					"name": "libB",
					"entry": "index.js",
					"peer_deps": {"react": "^18.0.0"}
				}`), 0644)

				appSrc := filepath.Join(tmpDir, "app.ts")
				_ = os.WriteFile(appSrc, []byte("import { a } from 'libA';"), 0644)

				return []string{appSrc}, []string{libADir, libBDir}
			},
			wantImports: map[string]string{
				"react": "./third_party/libA/.deps/react/index.js",
				"libA":  "./third_party/libA/index.js",
				"libB":  "./third_party/libB/index.js",
			},
			wantScopes: map[string]map[string]string{
				"./third_party/libA/": {
					"react": "./third_party/libA/.deps/react/index.js",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, err := os.MkdirTemp("", "importmap_hoist_matrix_*")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			srcs, deps := tt.setup(tmpDir)
			im, err := Synthesize("", srcs, deps, tmpDir)
			if err != nil {
				t.Fatalf("Synthesize failed: %v", err)
			}

			for key, want := range tt.wantImports {
				got, ok := im.Imports[key]
				if !ok {
					t.Errorf("import[%q] missing, want %q", key, want)
				} else if got != want {
					t.Errorf("import[%q] = %q, want %q", key, got, want)
				}
			}

			for scopeKey, expectedMappings := range tt.wantScopes {
				scopeMap, ok := im.Scopes[scopeKey]
				if !ok {
					t.Fatalf("scope[%q] missing", scopeKey)
				}
				for key, want := range expectedMappings {
					got, ok := scopeMap[key]
					if !ok {
						t.Errorf("scope[%q][%q] missing, want %q", scopeKey, key, want)
					} else if got != want {
						t.Errorf("scope[%q][%q] = %q, want %q", scopeKey, key, got, want)
					}
				}
			}
		})
	}
}

func TestFindTargetEntryMatrix(t *testing.T) {
	tests := []struct {
		name       string
		moduleName string
		srcs       []string
		want       string
	}{
		{
			name:       "exact match with module base name",
			moduleName: "@domain/calculator",
			srcs:       []string{"add.ts", "calculator.ts", "subtract.ts"},
			want:       "calculator.ts",
		},
		{
			name:       "exact match with tsx extension",
			moduleName: "Button",
			srcs:       []string{"Button.tsx", "styles.css"},
			want:       "Button.tsx",
		},
		{
			name:       "index.ts preferred when module name file is absent",
			moduleName: "@repo/utils",
			srcs:       []string{"format.ts", "index.ts", "math.ts"},
			want:       "index.ts",
		},
		{
			name:       "mod.ts preferred when index is absent",
			moduleName: "std_wrapper",
			srcs:       []string{"helper.ts", "mod.ts"},
			want:       "mod.ts",
		},
		{
			name:       "fallback to first source file when no candidate matches",
			moduleName: "random_lib",
			srcs:       []string{"first.ts", "second.ts"},
			want:       "first.ts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findTargetEntry(tt.moduleName, tt.srcs)
			if got != tt.want {
				t.Errorf("findTargetEntry(%q, %v) = %q, want %q", tt.moduleName, tt.srcs, got, tt.want)
			}
		})
	}
}

func TestIsSourceFileMatrix(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"index.ts", true},
		{"component.tsx", true},
		{"bundle.js", true},
		{"app.jsx", true},
		{"mod.mjs", true},
		{"data.json", false},
		{"main.go", false},
		{"README.md", false},
		{"build.sh", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isSourceFile(tt.path)
			if got != tt.want {
				t.Errorf("isSourceFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestDirFromPathMatrix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"./foo/bar.js", "./foo/"},
		{"foo/bar.js", "./foo/"},
		{"./calculator.ts", "./"},
		{"calculator.ts", "./"},
		{"./third_party/react/index.js", "./third_party/react/"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := dirFromPath(tt.input)
			if got != tt.want {
				t.Errorf("dirFromPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEnsureTrailingSlashMatrix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"third_party/react", "./third_party/react/"},
		{"./third_party/react", "./third_party/react/"},
		{"./third_party/react/", "./third_party/react/"},
		{"/root/path", "/root/path/"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ensureTrailingSlash(tt.input)
			if got != tt.want {
				t.Errorf("ensureTrailingSlash(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

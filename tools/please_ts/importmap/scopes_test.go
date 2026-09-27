package importmap

import (
	"os"
	"path/filepath"
	"testing"
)

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

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

func TestSynthesizeScopesAndPrecedence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "importmap_scopes_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 1. Explicit third_party/react
	reactDir := filepath.Join(tmpDir, "third_party", "react")
	_ = os.MkdirAll(reactDir, 0755)
	_ = os.WriteFile(filepath.Join(reactDir, "index.js"), []byte("module.exports = { version: '18.3.1' };"), 0644)
	reactMeta := `{
		"name": "react",
		"entry": "index.js"
	}`
	_ = os.WriteFile(filepath.Join(reactDir, "ts_module.json"), []byte(reactMeta), 0644)

	// 2. third_party/react-router with internal .deps/react
	routerDir := filepath.Join(tmpDir, "third_party", "react-router")
	routerInternalReact := filepath.Join(routerDir, ".deps", "react")
	_ = os.MkdirAll(routerInternalReact, 0755)
	_ = os.WriteFile(filepath.Join(routerDir, "index.js"), []byte("const r = require('react');"), 0644)
	_ = os.WriteFile(filepath.Join(routerInternalReact, "index.js"), []byte("module.exports = { internal: true };"), 0644)
	routerMeta := `{
		"name": "react-router",
		"entry": "index.js",
		"deps": {
			"react": "./.deps/react/index.js"
		}
	}`
	_ = os.WriteFile(filepath.Join(routerDir, "ts_module.json"), []byte(routerMeta), 0644)

	src := filepath.Join(tmpDir, "src", "app.ts")
	_ = os.MkdirAll(filepath.Dir(src), 0755)
	_ = os.WriteFile(src, []byte("import React from 'react'; import { Router } from 'react-router';"), 0644)

	im, err := Synthesize("", []string{src}, []string{reactDir, routerDir}, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify top-level imports
	if got := im.Imports["react"]; got != "./third_party/react/index.js" {
		t.Errorf("expected imports['react'] = './third_party/react/index.js', got %q", got)
	}
	if got := im.Imports["react-router"]; got != "./third_party/react-router/index.js" {
		t.Errorf("expected imports['react-router'] = './third_party/react-router/index.js', got %q", got)
	}

	// Verify react-router scope redirects to the explicit root react!
	scopeKey := "./third_party/react-router/"
	routerScope := im.Scopes[scopeKey]
	if routerScope == nil {
		t.Fatalf("expected scope for %q", scopeKey)
	}
	if got := routerScope["react"]; got != "./third_party/react/index.js" {
		t.Errorf("expected scope['react'] redirected to root target './third_party/react/index.js', got %q", got)
	}
}

func TestSynthesizeSingletonHoisting(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "importmap_hoist_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// libA and libB both need react (peer dependency), but react is NOT explicitly declared
	libADir := filepath.Join(tmpDir, "third_party", "libA")
	libAInternalReact := filepath.Join(libADir, ".deps", "react")
	_ = os.MkdirAll(libAInternalReact, 0755)
	_ = os.WriteFile(filepath.Join(libADir, "index.js"), []byte("export const a = 1;"), 0644)
	_ = os.WriteFile(filepath.Join(libAInternalReact, "index.js"), []byte("export const React = {};"), 0644)
	libAMeta := `{
		"name": "libA",
		"entry": "index.js",
		"peer_deps": {
			"react": "^18.0.0"
		},
		"deps": {
			"react": "./.deps/react/index.js"
		}
	}`
	_ = os.WriteFile(filepath.Join(libADir, "ts_module.json"), []byte(libAMeta), 0644)

	libBDir := filepath.Join(tmpDir, "third_party", "libB")
	_ = os.MkdirAll(libBDir, 0755)
	_ = os.WriteFile(filepath.Join(libBDir, "index.js"), []byte("export const b = 2;"), 0644)
	libBMeta := `{
		"name": "libB",
		"entry": "index.js",
		"peer_deps": {
			"react": "^18.0.0"
		}
	}`
	_ = os.WriteFile(filepath.Join(libBDir, "ts_module.json"), []byte(libBMeta), 0644)

	src := filepath.Join(tmpDir, "app.ts")
	_ = os.WriteFile(src, []byte("import { a } from 'libA';"), 0644)

	im, err := Synthesize("", []string{src}, []string{libADir, libBDir}, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify react is hoisted to top-level imports!
	if got := im.Imports["react"]; got != "./third_party/libA/.deps/react/index.js" {
		t.Errorf("expected hoisted imports['react'] = './third_party/libA/.deps/react/index.js', got %q", got)
	}

	// Verify libA scope points to hoisted react
	if got := im.Scopes["./third_party/libA/"]["react"]; got != "./third_party/libA/.deps/react/index.js" {
		t.Errorf("expected libA scope['react'] = './third_party/libA/.deps/react/index.js', got %q", got)
	}
}

func TestSynthesizeConflictingInternalScopes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "importmap_conflict_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// express has debug@2
	expressDir := filepath.Join(tmpDir, "third_party", "express")
	expressDebug := filepath.Join(expressDir, ".deps", "debug")
	_ = os.MkdirAll(expressDebug, 0755)
	_ = os.WriteFile(filepath.Join(expressDir, "index.js"), []byte("const debug = require('debug');"), 0644)
	_ = os.WriteFile(filepath.Join(expressDebug, "index.js"), []byte("module.exports = 'debug-v2';"), 0644)
	expressMeta := `{
		"name": "express",
		"entry": "index.js",
		"deps": {
			"debug": "./.deps/debug/index.js"
		}
	}`
	_ = os.WriteFile(filepath.Join(expressDir, "ts_module.json"), []byte(expressMeta), 0644)

	// cors has debug@4
	corsDir := filepath.Join(tmpDir, "third_party", "cors")
	corsDebug := filepath.Join(corsDir, ".deps", "debug")
	_ = os.MkdirAll(corsDebug, 0755)
	_ = os.WriteFile(filepath.Join(corsDir, "index.js"), []byte("const debug = require('debug');"), 0644)
	_ = os.WriteFile(filepath.Join(corsDebug, "index.js"), []byte("module.exports = 'debug-v4';"), 0644)
	corsMeta := `{
		"name": "cors",
		"entry": "index.js",
		"deps": {
			"debug": "./.deps/debug/index.js"
		}
	}`
	_ = os.WriteFile(filepath.Join(corsDir, "ts_module.json"), []byte(corsMeta), 0644)

	src := filepath.Join(tmpDir, "app.ts")
	_ = os.WriteFile(src, []byte("import express from 'express'; import cors from 'cors';"), 0644)

	im, err := Synthesize("", []string{src}, []string{expressDir, corsDir}, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// debug must NOT be leaked to root imports!
	if _, leaked := im.Imports["debug"]; leaked {
		t.Errorf("private helper 'debug' should not be leaked to root imports")
	}

	// express scope gets its debug v2
	if got := im.Scopes["./third_party/express/"]["debug"]; got != "./third_party/express/.deps/debug/index.js" {
		t.Errorf("expected express debug = './third_party/express/.deps/debug/index.js', got %q", got)
	}

	// cors scope gets its debug v4
	if got := im.Scopes["./third_party/cors/"]["debug"]; got != "./third_party/cors/.deps/debug/index.js" {
		t.Errorf("expected cors debug = './third_party/cors/.deps/debug/index.js', got %q", got)
	}
}

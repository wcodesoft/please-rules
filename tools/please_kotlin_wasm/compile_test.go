package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandCommaSeparated(t *testing.T) {
	input := []string{"a,b,c", "d", " e , f "}
	expected := []string{"a", "b", "c", "d", "e", "f"}
	result := ExpandCommaSeparated(input)

	if len(result) != len(expected) {
		t.Fatalf("expected %d elements, got %d", len(expected), len(result))
	}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("at index %d: expected %q, got %q", i, v, result[i])
		}
	}
}

func TestDiscoverWasmSources(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wasm-srcs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	kt1 := filepath.Join(tmpDir, "A.kt")
	kt2 := filepath.Join(subDir, "B.kt")
	other := filepath.Join(tmpDir, "other.txt")

	_ = os.WriteFile(kt1, []byte("// A"), 0644)
	_ = os.WriteFile(kt2, []byte("// B"), 0644)
	_ = os.WriteFile(other, []byte("// other"), 0644)

	// Test passing directory
	found := DiscoverWasmSources([]string{tmpDir})
	if len(found) != 2 {
		t.Errorf("expected 2 .kt files, got %d: %v", len(found), found)
	}

	// Test passing single file
	foundSingle := DiscoverWasmSources([]string{kt1})
	if len(foundSingle) != 1 || foundSingle[0] != kt1 {
		t.Errorf("expected [%s], got %v", kt1, foundSingle)
	}
}

func TestDiscoverKlibs(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wasm-klibs-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	klib1 := filepath.Join(tmpDir, "lib1.klib")
	other := filepath.Join(tmpDir, "other.jar")

	_ = os.WriteFile(klib1, []byte("klib"), 0644)
	_ = os.WriteFile(other, []byte("jar"), 0644)

	found := DiscoverKlibs([]string{tmpDir})
	if len(found) != 1 || found[0] != klib1 {
		t.Errorf("expected [%s], got %v", klib1, found)
	}
}

func TestResolveKotlincWasm(t *testing.T) {
	// Explicit path that doesn't exist falls back gracefully
	path, err := ResolveKotlincWasm("custom-kotlinc-wasm")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path == "" {
		t.Errorf("expected non-empty path")
	}
}

func TestCasesConversion(t *testing.T) {
	if ToPascalCase("disjoint-set") != "DisjointSet" {
		t.Errorf("expected DisjointSet, got %s", ToPascalCase("disjoint-set"))
	}
	if ToCamelCase("find-root") != "findRoot" {
		t.Errorf("expected findRoot, got %s", ToCamelCase("find-root"))
	}
	if ToPascalCase("math_ops") != "MathOps" {
		t.Errorf("expected MathOps, got %s", ToPascalCase("math_ops"))
	}
}

func TestMapWitTypeToKotlin(t *testing.T) {
	tests := map[string]string{
		"s32":          "Int",
		"s64":          "Long",
		"bool":         "Boolean",
		"string":       "String",
		"list<string>": "List<String>",
		"result<s32>":  "Int?",
		"option<bool>": "Boolean?",
	}
	for wit, kt := range tests {
		res := MapWitTypeToKotlin(wit)
		if res != kt {
			t.Errorf("for WIT %s, expected Kotlin %s, got %s", wit, kt, res)
		}
	}
}

func TestParseWitAndGenerate(t *testing.T) {
	witContent := `package example:structures;

interface disjoint-set {
    find-root: func(element: s32) -> s32;
    union-sets: func(a: s32, b: s32) -> bool;
    reset: func();
}
`
	tmpDir, err := os.MkdirTemp("", "wit-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	witFile := filepath.Join(tmpDir, "structures.wit")
	if err := os.WriteFile(witFile, []byte(witContent), 0644); err != nil {
		t.Fatal(err)
	}

	ifaces, err := ParseWit(witFile)
	if err != nil {
		t.Fatalf("ParseWit failed: %v", err)
	}
	if len(ifaces) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(ifaces))
	}
	iface := ifaces[0]
	if iface.Name != "disjoint-set" {
		t.Errorf("expected disjoint-set, got %s", iface.Name)
	}
	if iface.Package != "example.structures" {
		t.Errorf("expected example.structures, got %s", iface.Package)
	}
	if len(iface.Functions) != 3 {
		t.Fatalf("expected 3 functions, got %d", len(iface.Functions))
	}

	outDir := filepath.Join(tmpDir, "gen")
	_ = os.MkdirAll(outDir, 0755)

	artifacts, err := GenerateWitArtifacts(ifaces, "CustomDisjointSetImpl", "org.test", outDir)
	if err != nil {
		t.Fatalf("GenerateWitArtifacts failed: %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("expected 2 artifacts (interface and bridge), got %d", len(artifacts))
	}

	// Verify interface file
	ifacePath := filepath.Join(outDir, "DisjointSet.kt")
	ifaceBytes, err := os.ReadFile(ifacePath)
	if err != nil {
		t.Fatalf("failed to read interface file: %v", err)
	}
	ifaceSrc := string(ifaceBytes)
	if !strings.Contains(ifaceSrc, "package example.structures") {
		t.Errorf("interface missing package: %s", ifaceSrc)
	}
	if !strings.Contains(ifaceSrc, "public interface DisjointSet {") {
		t.Errorf("interface missing declaration: %s", ifaceSrc)
	}
	if !strings.Contains(ifaceSrc, "fun findRoot(element: Int): Int") {
		t.Errorf("interface missing findRoot: %s", ifaceSrc)
	}
	if !strings.Contains(ifaceSrc, "fun reset()") {
		t.Errorf("interface missing reset: %s", ifaceSrc)
	}

	// Verify bridge file
	bridgePath := filepath.Join(outDir, "DisjointSetBridge.kt")
	bridgeBytes, err := os.ReadFile(bridgePath)
	if err != nil {
		t.Fatalf("failed to read bridge file: %v", err)
	}
	bridgeSrc := string(bridgeBytes)
	if !strings.Contains(bridgeSrc, "package org.test") {
		t.Errorf("bridge missing package: %s", bridgeSrc)
	}
	if !strings.Contains(bridgeSrc, "import example.structures.*") {
		t.Errorf("bridge missing interface package import: %s", bridgeSrc)
	}
	if !strings.Contains(bridgeSrc, "import kotlin.wasm.WasmExport") {
		t.Errorf("bridge missing WasmExport import: %s", bridgeSrc)
	}
	if !strings.Contains(bridgeSrc, "private val instance by lazy { CustomDisjointSetImpl() }") {
		t.Errorf("bridge missing instance instantiation: %s", bridgeSrc)
	}
	if !strings.Contains(bridgeSrc, "@WasmExport\nfun findRoot(element: Int): Int {\n    return instance.findRoot(element)\n}") {
		t.Errorf("bridge missing @WasmExport findRoot: %s", bridgeSrc)
	}
	if !strings.Contains(bridgeSrc, "@WasmExport\nfun reset() {\n    instance.reset()\n}") {
		t.Errorf("bridge missing @WasmExport reset: %s", bridgeSrc)
	}
}

func TestGenerateWitArtifactsAutoDetectImpl(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "wit-autodetect-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srcFile := filepath.Join(tmpDir, "DisjointSet.kt")
	srcContent := "package structures\n\nclass DisjointSet {\n}\n"
	if err := os.WriteFile(srcFile, []byte(srcContent), 0644); err != nil {
		t.Fatal(err)
	}

	ifaces := []WitInterface{
		{
			Name:    "disjoint-set",
			Package: "babel.structures",
			Functions: []WitFunc{
				{Name: "reset", ReturnType: "Unit"},
			},
		},
	}

	outDir := filepath.Join(tmpDir, "gen")
	_ = os.MkdirAll(outDir, 0755)

	artifacts, err := GenerateWitArtifacts(ifaces, "", "structures", outDir, srcFile)
	if err != nil {
		t.Fatalf("GenerateWitArtifacts failed: %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(artifacts))
	}

	bridgePath := filepath.Join(outDir, "DisjointSetBridge.kt")
	bridgeBytes, err := os.ReadFile(bridgePath)
	if err != nil {
		t.Fatalf("failed to read bridge file: %v", err)
	}
	bridgeSrc := string(bridgeBytes)
	if !strings.Contains(bridgeSrc, "private val instance by lazy { DisjointSet() }") {
		t.Errorf("expected auto-detected DisjointSet(), got %s", bridgeSrc)
	}
}

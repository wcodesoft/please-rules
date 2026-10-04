package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tools/common/wit/ast"
)

func TestExpandCommaSeparated(t *testing.T) {
	input := []string{"a,b,c", "d", " e , f ", "g h  i"}
	expected := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"}
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

// witType parses a WIT type by declaring it as the result of a function.
func witType(t *testing.T, wit string) *ast.TypeRef {
	t.Helper()
	pkg, err := ast.ParseContent("interface i { f: func() -> " + wit + "; }")
	if err != nil {
		t.Fatalf("%s: %v", wit, err)
	}
	return pkg.Interfaces[0].Functions[0].Results
}

func TestMapTypeToKotlin(t *testing.T) {
	tests := map[string]string{
		"s32":                        "Int",
		"u8":                         "Int",
		"s64":                        "Long",
		"f64":                        "Double",
		"bool":                       "Boolean",
		"string":                     "String",
		"list<string>":               "List<String>",
		"list<list<s32>>":            "List<List<Int>>",
		"result<s32>":                "Int?",
		"result<s32, string>":        "Int?",
		"result<_, string>":          "Unit",
		"result<list<u8>, string>":   "List<Int>?",
		"option<bool>":               "Boolean?",
		"option<list<string>>":       "List<String>?",
		"shape":                      "Shape",
		"two-sum":                    "TwoSum",
		"result<option<s32>, error>": "Int??",
	}
	for wit, want := range tests {
		got, err := MapTypeToKotlin(witType(t, wit))
		if err != nil || got != want {
			t.Errorf("for WIT %s, expected Kotlin %s, got %s (err %v)", wit, want, got, err)
		}
	}
	if got, err := MapTypeToKotlin(nil); err != nil || got != "Unit" {
		t.Errorf("nil type = %q, %v; want Unit", got, err)
	}
}

func TestMapTypeToKotlinRejectsWhatItCannotMap(t *testing.T) {
	_, err := MapTypeToKotlin(witType(t, "tuple<s32, string>"))
	if err == nil || !strings.Contains(err.Error(), "tuple<s32, string> is not supported") {
		t.Errorf("err = %v", err)
	}
	// ...and the error says where, once it comes out of ParseWit.
	file := filepath.Join(t.TempDir(), "x.wit")
	_ = os.WriteFile(file, []byte("package a:b;\ninterface i { pair: func(x: tuple<s32, s32>) -> s32; }\n"), 0644)
	_, err = ParseWit(file)
	if err == nil || !strings.Contains(err.Error(), "interface i: function pair: parameter x") {
		t.Errorf("err = %v", err)
	}
}

func TestParseWitReadsWitStructure(t *testing.T) {
	dir := t.TempDir()
	// The package is declared in one file only; interfaces of the others belong to it.
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("a-root.wit", "// package fake:one;\npackage example:orders@1.2.0;\n\ninterface api {\n  /// Doc\n  place: func(\n    item: string,\n    quantity: u32,\n  ) -> result<u32, string>;\n  use other.{thing};\n  variant status { open, closed(u32) }\n  flags perms { read, write }\n  resource cart {\n    constructor();\n    add: func(item: string);\n    total: static func() -> u32;\n  }\n  cancel: func(id: u32);\n}\n")
	write("b-more.wit", "interface audit { log: func(message: string) -> list<string>; }\n")
	write("notes.txt", "interface ignored { nope: func(); }\n")

	ifaces, err := ParseWit(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(ifaces) != 2 {
		t.Fatalf("interfaces = %+v", ifaces)
	}
	api, audit := ifaces[0], ifaces[1]
	if api.Name != "api" || api.Package != "example.orders" || audit.Name != "audit" || audit.Package != "example.orders" {
		t.Errorf("api = %s/%s, audit = %s/%s", api.Package, api.Name, audit.Package, audit.Name)
	}
	// resource methods are not interface functions, and use/variant/flags generate nothing
	if len(api.Functions) != 2 {
		t.Fatalf("api functions = %+v", api.Functions)
	}
	place := api.Functions[0]
	if place.Name != "place" || place.ReturnType != "Int?" || len(place.Params) != 2 ||
		place.Params[0] != (WitParam{Name: "item", Type: "String"}) || place.Params[1] != (WitParam{Name: "quantity", Type: "Int"}) {
		t.Errorf("place = %+v", place)
	}
	if api.Functions[1].Name != "cancel" || api.Functions[1].ReturnType != "Unit" {
		t.Errorf("cancel = %+v", api.Functions[1])
	}
	if audit.Functions[0].ReturnType != "List<String>" {
		t.Errorf("log = %+v", audit.Functions[0])
	}
}

func TestParseWitErrors(t *testing.T) {
	if _, err := ParseWit(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("expected an error for a missing path")
	}
	file := filepath.Join(t.TempDir(), "bad.wit")
	_ = os.WriteFile(file, []byte("interface {\n"), 0644)
	if _, err := ParseWit(file); err == nil || !strings.Contains(err.Error(), file) {
		t.Errorf("err = %v, want one naming %s", err, file)
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

func TestCompileWasmLibraryModeValidation(t *testing.T) {
	opts := WasmOptions{
		Out:  "lib.klib",
		Srcs: []string{},
	}
	err := CompileWasm(opts)
	if err == nil {
		t.Errorf("expected error for empty sources in library mode, got nil")
	}
}

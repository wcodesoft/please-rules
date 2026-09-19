package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"tools/please_wit/ast"
)

func TestGenerateCpp(t *testing.T) {
	tmpDir := t.TempDir()
	witContent := `
package test:structures;

interface two-sum {
    solve: func(nums: list<s32>, target: s32) -> list<s32>;
    reset: func();
}

interface records-and-enums {
    record point {
        x: f64,
        y: f64,
    }

    enum color {
        red,
        green,
        blue,
    }
}
`
	witFile := filepath.Join(tmpDir, "structures.wit")
	if err := os.WriteFile(witFile, []byte(witContent), 0644); err != nil {
		t.Fatal(err)
	}

	cppOut := filepath.Join(tmpDir, "cpp_out")
	_ = os.MkdirAll(cppOut, 0755)

	opts := Options{
		Lang: "cpp",
		Out:  cppOut,
		Srcs: []string{witFile},
	}
	if err := Run(opts); err != nil {
		t.Fatalf("Run(cpp) failed: %v", err)
	}

	cppHFile := filepath.Join(cppOut, "structures.h")
	cppHBytes, err := os.ReadFile(cppHFile)
	if err != nil {
		t.Fatalf("failed to read %s: %v", cppHFile, err)
	}
	cppHCode := string(cppHBytes)

	// Verify pragma once and namespace
	if !strings.Contains(cppHCode, "#pragma once") {
		t.Errorf("expected #pragma once, got: %s", cppHCode)
	}
	if !strings.Contains(cppHCode, "namespace test_structures {") {
		t.Errorf("expected namespace test_structures, got: %s", cppHCode)
	}

	// Verify class TwoSum
	if !strings.Contains(cppHCode, "class TwoSum {") {
		t.Errorf("expected class TwoSum, got: %s", cppHCode)
	}
	if !strings.Contains(cppHCode, "virtual ~TwoSum() = default;") {
		t.Errorf("expected virtual destructor, got: %s", cppHCode)
	}
	if !strings.Contains(cppHCode, "virtual std::vector<int32_t> solve(const std::vector<int32_t>& nums, int32_t target) = 0;") {
		t.Errorf("expected virtual solve signature, got: %s", cppHCode)
	}
	if !strings.Contains(cppHCode, "virtual void reset() = 0;") {
		t.Errorf("expected virtual reset signature, got: %s", cppHCode)
	}

	// Verify struct Point
	if !strings.Contains(cppHCode, "struct Point {") {
		t.Errorf("expected struct Point, got: %s", cppHCode)
	}

	// Verify enum class Color
	if !strings.Contains(cppHCode, "enum class Color {") {
		t.Errorf("expected enum class Color, got: %s", cppHCode)
	}

	// Verify companion .cpp file is generated
	cppSrcFile := filepath.Join(cppOut, "structures.cpp")
	cppSrcBytes, err := os.ReadFile(cppSrcFile)
	if err != nil {
		t.Fatalf("failed to read %s: %v", cppSrcFile, err)
	}
	if !strings.Contains(string(cppSrcBytes), `#include "structures.h"`) {
		t.Errorf("expected include in cpp source, got: %s", string(cppSrcBytes))
	}
}

func TestCppGeneratorDirect(t *testing.T) {
	pkg := &ast.Package{
		Namespace: "demo",
		Name:      "api",
		Interfaces: []ast.Interface{
			{
				Name: "greeter",
				Functions: []ast.Function{
					{
						Name: "say-hello",
						Params: []ast.Param{
							{Name: "name", Type: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "string"}},
						},
						Results: &ast.TypeRef{Kind: ast.KindPrimitive, Name: "string"},
					},
				},
			},
		},
	}

	gen := &CppGenerator{}
	if gen.Name() != "cpp" {
		t.Errorf("expected name 'cpp', got %q", gen.Name())
	}

	files, err := gen.Generate(pkg, Options{}, "api")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files (h and cpp), got %d", len(files))
	}
	if files[0].Name != "api.h" {
		t.Errorf("expected api.h, got %q", files[0].Name)
	}
	if files[1].Name != "api.cpp" {
		t.Errorf("expected api.cpp, got %q", files[1].Name)
	}
	if !strings.Contains(files[0].Content, "class Greeter {") {
		t.Errorf("expected class Greeter, got: %s", files[0].Content)
	}
	if !strings.Contains(files[1].Content, `#include "api.h"`) {
		t.Errorf("expected include in cpp companion, got: %s", files[1].Content)
	}
}

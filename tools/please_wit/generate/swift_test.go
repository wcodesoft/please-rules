package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"tools/please_wit/ast"
)

func TestGenerateSwift(t *testing.T) {
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

	swiftOut := filepath.Join(tmpDir, "swift_out")
	_ = os.MkdirAll(swiftOut, 0755)

	opts := Options{
		Lang: "swift",
		Out:  swiftOut,
		Srcs: []string{witFile},
	}
	if err := Run(opts); err != nil {
		t.Fatalf("Run(swift) failed: %v", err)
	}

	swiftFile := filepath.Join(swiftOut, "Structures.swift")
	swiftBytes, err := os.ReadFile(swiftFile)
	if err != nil {
		t.Fatalf("failed to read %s: %v", swiftFile, err)
	}
	swiftCode := string(swiftBytes)

	// Verify header
	if !strings.Contains(swiftCode, "import Foundation") {
		t.Errorf("expected import Foundation, got: %s", swiftCode)
	}

	// Verify protocol TwoSum
	if !strings.Contains(swiftCode, "public protocol TwoSum {") {
		t.Errorf("expected public protocol TwoSum, got: %s", swiftCode)
	}
	if !strings.Contains(swiftCode, "func solve(nums: [Int32], target: Int32) -> [Int32]") {
		t.Errorf("expected func solve signature, got: %s", swiftCode)
	}
	if !strings.Contains(swiftCode, "func reset()") {
		t.Errorf("expected func reset signature, got: %s", swiftCode)
	}

	// Verify struct Point
	if !strings.Contains(swiftCode, "public struct Point: Equatable, Codable {") {
		t.Errorf("expected public struct Point, got: %s", swiftCode)
	}

	// Verify enum Color
	if !strings.Contains(swiftCode, "public enum Color: String, Codable {") {
		t.Errorf("expected public enum Color, got: %s", swiftCode)
	}

	// Verify module.modulemap generated
	modMapFile := filepath.Join(swiftOut, "module.modulemap")
	modBytes, err := os.ReadFile(modMapFile)
	if err != nil {
		t.Fatalf("failed to read modulemap %s: %v", modMapFile, err)
	}
	if !strings.Contains(string(modBytes), "module Structures {") {
		t.Errorf("expected module Structures in modulemap, got: %s", string(modBytes))
	}
}

func TestSwiftGeneratorDirect(t *testing.T) {
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

	gen := &SwiftGenerator{}
	if gen.Name() != "swift" {
		t.Errorf("expected name 'swift', got %q", gen.Name())
	}

	files, err := gen.Generate(pkg, Options{}, "api")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files (swift and modulemap), got %d", len(files))
	}
	if files[0].Name != "Api.swift" {
		t.Errorf("expected filename 'Api.swift', got %q", files[0].Name)
	}
	if files[1].Name != "module.modulemap" {
		t.Errorf("expected modulemap, got %q", files[1].Name)
	}
	if !strings.Contains(files[0].Content, "public protocol Greeter") {
		t.Errorf("expected public protocol Greeter, got: %s", files[0].Content)
	}
	if !strings.Contains(files[1].Content, "module Api {") {
		t.Errorf("expected module Api in modulemap, got: %s", files[1].Content)
	}
}

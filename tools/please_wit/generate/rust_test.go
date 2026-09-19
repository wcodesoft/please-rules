package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"tools/please_wit/ast"
)

func TestGenerateRust(t *testing.T) {
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

	rustOut := filepath.Join(tmpDir, "rust_out")
	_ = os.MkdirAll(rustOut, 0755)

	opts := Options{
		Lang: "rust",
		Out:  rustOut,
		Srcs: []string{witFile},
	}
	if err := Run(opts); err != nil {
		t.Fatalf("Run(rust) failed: %v", err)
	}

	rustFile := filepath.Join(rustOut, "structures.rs")
	rustBytes, err := os.ReadFile(rustFile)
	if err != nil {
		t.Fatalf("failed to read %s: %v", rustFile, err)
	}
	rustCode := string(rustBytes)

	// Verify trait TwoSum
	if !strings.Contains(rustCode, "pub trait TwoSum {") {
		t.Errorf("expected pub trait TwoSum, got: %s", rustCode)
	}
	if !strings.Contains(rustCode, "fn solve(&mut self, nums: Vec<i32>, target: i32) -> Vec<i32>;") {
		t.Errorf("expected fn solve, got: %s", rustCode)
	}
	if !strings.Contains(rustCode, "fn reset(&mut self);") {
		t.Errorf("expected fn reset, got: %s", rustCode)
	}

	// Verify struct Point
	if !strings.Contains(rustCode, "pub struct Point {") {
		t.Errorf("expected pub struct Point, got: %s", rustCode)
	}
	if !strings.Contains(rustCode, "pub x: f64,") || !strings.Contains(rustCode, "pub y: f64,") {
		t.Errorf("expected pub x, pub y in Point, got: %s", rustCode)
	}

	// Verify enum Color
	if !strings.Contains(rustCode, "pub enum Color {") {
		t.Errorf("expected pub enum Color, got: %s", rustCode)
	}
	if !strings.Contains(rustCode, "Red,") || !strings.Contains(rustCode, "Green,") || !strings.Contains(rustCode, "Blue,") {
		t.Errorf("expected enum variants in Color, got: %s", rustCode)
	}
}

func TestRustGeneratorDirect(t *testing.T) {
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

	gen := &RustGenerator{}
	if gen.Name() != "rust" {
		t.Errorf("expected name 'rust', got %q", gen.Name())
	}

	files, err := gen.Generate(pkg, Options{}, "api")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Name != "api.rs" {
		t.Errorf("expected api.rs, got %q", files[0].Name)
	}
	if !strings.Contains(files[0].Content, "pub trait Greeter {") {
		t.Errorf("expected pub trait Greeter, got: %s", files[0].Content)
	}
}

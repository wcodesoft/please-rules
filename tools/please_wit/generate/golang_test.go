package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"tools/please_wit/ast"
)

func TestGenerateGo(t *testing.T) {
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

	goOut := filepath.Join(tmpDir, "go_out")
	_ = os.MkdirAll(goOut, 0755)

	opts := Options{
		Lang: "go",
		Out:  goOut,
		Srcs: []string{witFile},
	}
	if err := Run(opts); err != nil {
		t.Fatalf("Run(go) failed: %v", err)
	}

	goFile := filepath.Join(goOut, "structures.go")
	goBytes, err := os.ReadFile(goFile)
	if err != nil {
		t.Fatalf("failed to read %s: %v", goFile, err)
	}
	goCode := string(goBytes)

	// Verify package declaration
	if !strings.Contains(goCode, "package structures") {
		t.Errorf("expected package structures, got: %s", goCode)
	}

	// Verify interface TwoSum
	if !strings.Contains(goCode, "type TwoSum interface {") {
		t.Errorf("expected type TwoSum interface, got: %s", goCode)
	}
	if !strings.Contains(goCode, "Solve(nums []int32, target int32) []int32") {
		t.Errorf("expected Solve method signature, got: %s", goCode)
	}
	if !strings.Contains(goCode, "Reset()") {
		t.Errorf("expected Reset method signature, got: %s", goCode)
	}

	// Verify struct Point
	if !strings.Contains(goCode, "type Point struct {") {
		t.Errorf("expected type Point struct, got: %s", goCode)
	}
	if !strings.Contains(goCode, "X float64") || !strings.Contains(goCode, "Y float64") {
		t.Errorf("expected fields X, Y in Point, got: %s", goCode)
	}

	// Verify enum Color
	if !strings.Contains(goCode, "type Color int") {
		t.Errorf("expected type Color int, got: %s", goCode)
	}
	if !strings.Contains(goCode, "Color_Red Color = iota") {
		t.Errorf("expected Color_Red Color = iota, got: %s", goCode)
	}
}

func TestGoGeneratorDirect(t *testing.T) {
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

	gen := &GoGenerator{}
	if gen.Name() != "go" {
		t.Errorf("expected name 'go', got %q", gen.Name())
	}

	files, err := gen.Generate(pkg, Options{}, "api")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Name != "api.go" {
		t.Errorf("expected api.go, got %q", files[0].Name)
	}
	if !strings.Contains(files[0].Content, "type Greeter interface {") {
		t.Errorf("expected type Greeter interface, got: %s", files[0].Content)
	}
}

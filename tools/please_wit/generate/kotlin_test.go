package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"tools/please_wit/ast"
)

func TestGenerateKotlin(t *testing.T) {
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

    get-point: func() -> point;
}
`
	witFile := filepath.Join(tmpDir, "structures.wit")
	if err := os.WriteFile(witFile, []byte(witContent), 0644); err != nil {
		t.Fatal(err)
	}

	ktOut := filepath.Join(tmpDir, "kt_out")
	_ = os.MkdirAll(ktOut, 0755)

	opts := Options{
		Lang: "kotlin",
		Out:  ktOut,
		Srcs: []string{witFile},
	}
	if err := Run(opts); err != nil {
		t.Fatalf("Run(kotlin) failed: %v", err)
	}

	ktFile := filepath.Join(ktOut, "Structures.kt")
	ktBytes, err := os.ReadFile(ktFile)
	if err != nil {
		t.Fatalf("failed to read %s: %v", ktFile, err)
	}
	ktCode := string(ktBytes)

	// Verify package declaration
	if !strings.Contains(ktCode, "package test.structures") {
		t.Errorf("expected package test.structures, got: %s", ktCode)
	}

	// Verify interface TwoSum
	if !strings.Contains(ktCode, "public interface TwoSum {") {
		t.Errorf("expected public interface TwoSum, got: %s", ktCode)
	}
	if !strings.Contains(ktCode, "fun solve(nums: List<Int>, target: Int): List<Int>") {
		t.Errorf("expected fun solve signature, got: %s", ktCode)
	}
	if !strings.Contains(ktCode, "fun reset()") {
		t.Errorf("expected fun reset signature, got: %s", ktCode)
	}

	// Verify record Point
	if !strings.Contains(ktCode, "public data class Point(") {
		t.Errorf("expected data class Point, got: %s", ktCode)
	}
	if !strings.Contains(ktCode, "val x: Double,") || !strings.Contains(ktCode, "val y: Double") {
		t.Errorf("expected fields x, y in Point, got: %s", ktCode)
	}

	// Verify enum Color
	if !strings.Contains(ktCode, "public enum class Color {") {
		t.Errorf("expected enum class Color, got: %s", ktCode)
	}
	if !strings.Contains(ktCode, "RED,") || !strings.Contains(ktCode, "GREEN,") || !strings.Contains(ktCode, "BLUE;") {
		t.Errorf("expected enum cases, got: %s", ktCode)
	}
}

func TestKotlinGeneratorDirect(t *testing.T) {
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

	gen := &KotlinGenerator{}
	if gen.Name() != "kotlin" {
		t.Errorf("expected name 'kotlin', got %q", gen.Name())
	}

	files, err := gen.Generate(pkg, Options{}, "api")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Name != "Api.kt" {
		t.Errorf("expected filename 'Api.kt', got %q", files[0].Name)
	}
	if !strings.Contains(files[0].Content, "public interface Greeter") {
		t.Errorf("expected public interface Greeter, got: %s", files[0].Content)
	}
}

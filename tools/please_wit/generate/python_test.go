package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"tools/please_wit/ast"
)

func TestGeneratePython(t *testing.T) {
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

	pyOut := filepath.Join(tmpDir, "py_out")
	_ = os.MkdirAll(pyOut, 0755)

	opts := Options{
		Lang: "python",
		Out:  pyOut,
		Srcs: []string{witFile},
	}
	if err := Run(opts); err != nil {
		t.Fatalf("Run(python) failed: %v", err)
	}

	pyFile := filepath.Join(pyOut, "__init__.py")
	pyBytes, err := os.ReadFile(pyFile)
	if err != nil {
		t.Fatalf("failed to read %s: %v", pyFile, err)
	}
	pyCode := string(pyBytes)

	// Verify typing imports
	if !strings.Contains(pyCode, "from typing import") || !strings.Contains(pyCode, "Protocol") {
		t.Errorf("expected typing import with Protocol, got: %s", pyCode)
	}

	// Verify class TwoSum(Protocol)
	if !strings.Contains(pyCode, "class TwoSum(Protocol):") {
		t.Errorf("expected class TwoSum(Protocol), got: %s", pyCode)
	}
	if !strings.Contains(pyCode, "def solve(self, nums: List[int], target: int) -> List[int]:") {
		t.Errorf("expected def solve, got: %s", pyCode)
	}
	if !strings.Contains(pyCode, "def reset(self) -> None:") {
		t.Errorf("expected def reset, got: %s", pyCode)
	}

	// Verify Point dataclass
	if !strings.Contains(pyCode, "class Point:") {
		t.Errorf("expected class Point, got: %s", pyCode)
	}

	// Verify Color enum
	if !strings.Contains(pyCode, "class Color(Enum):") {
		t.Errorf("expected class Color(Enum), got: %s", pyCode)
	}

	// Verify pyi stub file also exists
	pyiFile := filepath.Join(pyOut, "__init__.pyi")
	if _, err := os.Stat(pyiFile); err != nil {
		t.Errorf("expected __init__.pyi to exist: %v", err)
	}
}

func TestPythonGeneratorDirect(t *testing.T) {
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

	gen := &PythonGenerator{}
	if gen.Name() != "python" {
		t.Errorf("expected name 'python', got %q", gen.Name())
	}

	files, err := gen.Generate(pkg, Options{}, "api")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files (__init__.py and __init__.pyi), got %d", len(files))
	}
	if files[0].Name != "__init__.py" {
		t.Errorf("expected __init__.py, got %q", files[0].Name)
	}
	if files[1].Name != "__init__.pyi" {
		t.Errorf("expected __init__.pyi, got %q", files[1].Name)
	}
	if !strings.Contains(files[0].Content, "class Greeter(Protocol):") {
		t.Errorf("expected class Greeter(Protocol), got: %s", files[0].Content)
	}
}

package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"tools/please_wit/ast"
)

func TestGenerateTypeScript(t *testing.T) {
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

	tsOut := filepath.Join(tmpDir, "ts_out")
	_ = os.MkdirAll(tsOut, 0755)

	opts := Options{
		Lang: "ts",
		Out:  tsOut,
		Srcs: []string{witFile},
	}
	if err := Run(opts); err != nil {
		t.Fatalf("Run(ts) failed: %v", err)
	}

	tsFile := filepath.Join(tsOut, "structures.d.ts")
	tsBytes, err := os.ReadFile(tsFile)
	if err != nil {
		t.Fatalf("failed to read %s: %v", tsFile, err)
	}
	tsCode := string(tsBytes)

	// Verify interface TwoSum
	if !strings.Contains(tsCode, "export interface TwoSum {") {
		t.Errorf("expected export interface TwoSum, got: %s", tsCode)
	}
	if !strings.Contains(tsCode, "solve(nums: number[], target: number): number[];") {
		t.Errorf("expected solve signature, got: %s", tsCode)
	}
	if !strings.Contains(tsCode, "reset(): void;") {
		t.Errorf("expected reset signature, got: %s", tsCode)
	}

	// Verify interface Point
	if !strings.Contains(tsCode, "export interface Point {") {
		t.Errorf("expected export interface Point, got: %s", tsCode)
	}

	// Verify enum Color
	if !strings.Contains(tsCode, "export enum Color {") {
		t.Errorf("expected export enum Color, got: %s", tsCode)
	}
}

func TestTypeScriptGeneratorDirect(t *testing.T) {
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

	gen := &TypeScriptGenerator{}
	if gen.Name() != "typescript" {
		t.Errorf("expected name 'typescript', got %q", gen.Name())
	}

	files, err := gen.Generate(pkg, Options{}, "api")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files (api.d.ts and index.d.ts), got %d", len(files))
	}
	if files[0].Name != "api.d.ts" {
		t.Errorf("expected filename 'api.d.ts', got %q", files[0].Name)
	}
	if files[1].Name != "index.d.ts" {
		t.Errorf("expected index.d.ts, got %q", files[1].Name)
	}
	if !strings.Contains(files[0].Content, "export interface Greeter") {
		t.Errorf("expected export interface Greeter, got: %s", files[0].Content)
	}
}

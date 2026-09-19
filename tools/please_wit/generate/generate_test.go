package generate

import (
	"os"
	"path/filepath"
	"testing"
	"tools/please_wit/ast"
)

func TestDiscoverWorlds(t *testing.T) {
	tmpDir := t.TempDir()
	witContent := `
package test:demo;

interface foo {
    bar: func();
}

world my-world-1 {
    export foo;
}

world my-world-2 {
    import foo;
}
`
	witFile := filepath.Join(tmpDir, "demo.wit")
	if err := os.WriteFile(witFile, []byte(witContent), 0644); err != nil {
		t.Fatalf("failed to write test wit file: %v", err)
	}

	worlds, err := DiscoverWorlds(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverWorlds failed: %v", err)
	}

	if len(worlds) != 2 {
		t.Fatalf("expected 2 worlds, got %d: %v", len(worlds), worlds)
	}
	if worlds[0] != "my-world-1" || worlds[1] != "my-world-2" {
		t.Errorf("unexpected worlds: %v", worlds)
	}
}

func TestDeriveBaseName(t *testing.T) {
	// 1. CompanionFilename set
	opts1 := Options{CompanionFilename: "custom-name.kt"}
	if base := DeriveBaseName(opts1, ""); base != "custom-name" {
		t.Errorf("expected custom-name, got %q", base)
	}

	// 2. Derive from package name in options
	opts2 := Options{Package: "test:structures"}
	if base := DeriveBaseName(opts2, ""); base != "structures" {
		t.Errorf("expected structures, got %q", base)
	}

	// 3. Fallback when package name and witPath are empty
	opts3 := Options{}
	if base := DeriveBaseName(opts3, ""); base != "Wit" {
		t.Errorf("expected Wit, got %q", base)
	}
}

func TestGenerateUnknownLanguage(t *testing.T) {
	tmpDir := t.TempDir()
	opts := Options{
		Lang: "unsupported-lang",
		Out:  tmpDir,
		Srcs: []string{"dummy.wit"},
	}
	err := Run(opts)
	if err == nil {
		t.Errorf("expected error for unsupported language, got nil")
	}
}

func TestGetGenerator(t *testing.T) {
	cases := []struct {
		lang     string
		expected string
	}{
		{"kotlin", "kotlin"},
		{"kt", "kotlin"},
		{"swift", "swift"},
		{"ts", "typescript"},
		{"typescript", "typescript"},
		{"python", "python"},
		{"py", "python"},
		{"rust", "rust"},
		{"rs", "rust"},
		{"go", "go"},
		{"golang", "go"},
		{"cpp", "cpp"},
		{"cc", "cpp"},
		{"c", "cpp"},
		{"cxx", "cpp"},
	}

	for _, tc := range cases {
		gen, err := GetGenerator(tc.lang)
		if err != nil {
			t.Fatalf("GetGenerator(%q) failed: %v", tc.lang, err)
		}
		if gen.Name() != tc.expected {
			t.Errorf("GetGenerator(%q).Name() = %q, want %q", tc.lang, gen.Name(), tc.expected)
		}
	}
}

func TestMapWitType(t *testing.T) {
	s32Type := &ast.TypeRef{Kind: ast.KindPrimitive, Name: "s32"}
	stringType := &ast.TypeRef{Kind: ast.KindPrimitive, Name: "string"}
	listS32 := &ast.TypeRef{Kind: ast.KindList, TypeArgs: []*ast.TypeRef{s32Type}}
	optString := &ast.TypeRef{Kind: ast.KindOption, TypeArgs: []*ast.TypeRef{stringType}}
	namedType := &ast.TypeRef{Kind: ast.KindNamed, Name: "custom-data"}

	tests := []struct {
		lang     string
		typeRef  *ast.TypeRef
		expected string
	}{
		// Kotlin
		{"kotlin", nil, "Unit"},
		{"kotlin", s32Type, "Int"},
		{"kotlin", stringType, "String"},
		{"kotlin", listS32, "List<Int>"},
		{"kotlin", optString, "String?"},
		{"kotlin", namedType, "CustomData"},

		// Swift
		{"swift", nil, "Void"},
		{"swift", s32Type, "Int32"},
		{"swift", stringType, "String"},
		{"swift", listS32, "[Int32]"},
		{"swift", optString, "String?"},
		{"swift", namedType, "CustomData"},

		// TypeScript
		{"typescript", nil, "void"},
		{"typescript", s32Type, "number"},
		{"typescript", stringType, "string"},
		{"typescript", listS32, "number[]"},
		{"typescript", optString, "string | null"},
		{"typescript", namedType, "CustomData"},

		// Python
		{"python", nil, "None"},
		{"python", s32Type, "int"},
		{"python", stringType, "str"},
		{"python", listS32, "List[int]"},
		{"python", optString, "Optional[str]"},
		{"python", namedType, "CustomData"},

		// Rust
		{"rust", nil, "()"},
		{"rust", s32Type, "i32"},
		{"rust", stringType, "String"},
		{"rust", listS32, "Vec<i32>"},
		{"rust", optString, "Option<String>"},
		{"rust", namedType, "CustomData"},

		// Go
		{"go", nil, ""},
		{"go", s32Type, "int32"},
		{"go", stringType, "string"},
		{"go", listS32, "[]int32"},
		{"go", optString, "*string"},
		{"go", namedType, "CustomData"},

		// C++
		{"cpp", nil, "void"},
		{"cpp", s32Type, "int32_t"},
		{"cpp", stringType, "std::string"},
		{"cpp", listS32, "std::vector<int32_t>"},
		{"cpp", optString, "std::optional<std::string>"},
		{"cpp", namedType, "CustomData"},
	}

	for _, tc := range tests {
		gen, err := GetGenerator(tc.lang)
		if err != nil {
			t.Fatalf("GetGenerator(%q) failed: %v", tc.lang, err)
		}
		got := gen.MapWitType(tc.typeRef)
		if got != tc.expected {
			t.Errorf("[%s] MapWitType(%v) = %q, want %q", tc.lang, tc.typeRef, got, tc.expected)
		}
	}
}

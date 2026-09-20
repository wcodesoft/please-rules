package generate

import (
	"os"
	"path/filepath"
	"strings"
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

// TestCasingConversions_Matrix tests identifier case conversions with a matrix of delimiters and formats.
func TestCasingConversions_Matrix(t *testing.T) {
	matrix := []struct {
		input      string
		wantPascal string
		wantCamel  string
		wantSnake  string
	}{
		{"", "", "", ""},
		{"structures", "Structures", "structures", "structures"},
		{"two_sum", "TwoSum", "twoSum", "two_sum"},
		{"two-sum", "TwoSum", "twoSum", "two_sum"},
		{"two:sum", "TwoSum", "twoSum", "two_sum"},
		{"two.sum", "TwoSum", "twoSum", "two_sum"},
		{"my_great_service", "MyGreatService", "myGreatService", "my_great_service"},
		{"hello_world-test:case.now", "HelloWorldTestCaseNow", "helloWorldTestCaseNow", "hello_world_test_case_now"},
		{"UPPER_CASE_NAME", "UPPERCASENAME", "upperCaseName", "upper_case_name"},
		{"single", "Single", "single", "single"},
		{"trailing_delim_", "TrailingDelim", "trailingDelim", "trailing_delim"},
		{"-leading-delim", "LeadingDelim", "leadingDelim", "leading_delim"},
		{"multiple___delims---here", "MultipleDelimsHere", "multipleDelimsHere", "multiple_delims_here"},
	}

	for _, tc := range matrix {
		name := tc.input
		if name == "" {
			name = "<empty>"
		}
		t.Run(name, func(t *testing.T) {
			gotPascal := ToPascalCase(tc.input)
			if gotPascal != tc.wantPascal {
				t.Errorf("ToPascalCase(%q) = %q, want %q", tc.input, gotPascal, tc.wantPascal)
			}
			gotCamel := ToCamelCase(tc.input)
			if gotCamel != tc.wantCamel {
				t.Errorf("ToCamelCase(%q) = %q, want %q", tc.input, gotCamel, tc.wantCamel)
			}
			gotSnake := ToSnakeCase(tc.input)
			if gotSnake != tc.wantSnake {
				t.Errorf("ToSnakeCase(%q) = %q, want %q", tc.input, gotSnake, tc.wantSnake)
			}
		})
	}
}

// TestDiscoverWorlds_Matrix runs a matrix of file and directory structures through DiscoverWorlds.
func TestDiscoverWorlds_Matrix(t *testing.T) {
	matrix := []struct {
		name        string
		setup       func(t *testing.T) string
		wantWorlds  []string
		expectError bool
	}{
		{
			name: "single_wit_file_with_multiple_worlds",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				file := filepath.Join(dir, "types.wit")
				content := "world primary-world {}\nworld secondary-world {}\nworld primary-world {}\n"
				_ = os.WriteFile(file, []byte(content), 0644)
				return file
			},
			wantWorlds:  []string{"primary-world", "secondary-world"},
			expectError: false,
		},
		{
			name: "directory_with_multiple_wit_and_non_wit_files",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				_ = os.WriteFile(filepath.Join(dir, "a.wit"), []byte("world world-a {}\n"), 0644)
				_ = os.WriteFile(filepath.Join(dir, "b.wit"), []byte("world world-b {}\nworld world-a {}\n"), 0644)
				_ = os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("world ignore-me {}\n"), 0644)
				_ = os.Mkdir(filepath.Join(dir, "subdir.wit"), 0755)
				return dir
			},
			wantWorlds:  []string{"world-a", "world-b"},
			expectError: false,
		},
		{
			name: "file_without_worlds",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				file := filepath.Join(dir, "empty.wit")
				_ = os.WriteFile(file, []byte("package foo:bar;\ninterface baz {}\n"), 0644)
				return file
			},
			wantWorlds:  nil,
			expectError: false,
		},
		{
			name: "non_existent_path",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "does-not-exist.wit")
			},
			wantWorlds:  nil,
			expectError: true,
		},
	}

	for _, tc := range matrix {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.setup(t)
			worlds, err := DiscoverWorlds(path)
			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error for %s, got nil", tc.name)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tc.name, err)
			}
			if len(worlds) != len(tc.wantWorlds) {
				t.Fatalf("worlds count mismatch: got %v, want %v", worlds, tc.wantWorlds)
			}
			for i := range worlds {
				if worlds[i] != tc.wantWorlds[i] {
					t.Errorf("world[%d] mismatch: got %q, want %q", i, worlds[i], tc.wantWorlds[i])
				}
			}
		})
	}
}

// TestDiscoverPackage_Matrix runs a matrix of scenarios through DiscoverPackage.
func TestDiscoverPackage_Matrix(t *testing.T) {
	matrix := []struct {
		name        string
		setup       func(t *testing.T) string
		wantPackage string
		expectError bool
	}{
		{
			name: "single_file_with_package",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				file := filepath.Join(dir, "api.wit")
				_ = os.WriteFile(file, []byte("package example:orders;\ninterface api {}\n"), 0644)
				return file
			},
			wantPackage: "example:orders",
			expectError: false,
		},
		{
			name: "directory_with_package_in_one_file",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				_ = os.WriteFile(filepath.Join(dir, "types.wit"), []byte("interface types {}\n"), 0644)
				_ = os.WriteFile(filepath.Join(dir, "root.wit"), []byte("package corp:service-core;\n"), 0644)
				_ = os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("package ignored:file;\n"), 0644)
				return dir
			},
			wantPackage: "corp:service-core",
			expectError: false,
		},
		{
			name: "file_without_package",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				file := filepath.Join(dir, "no_pkg.wit")
				_ = os.WriteFile(file, []byte("interface foo {}\n"), 0644)
				return file
			},
			wantPackage: "",
			expectError: false,
		},
		{
			name: "non_existent_path",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent")
			},
			wantPackage: "",
			expectError: true,
		},
	}

	for _, tc := range matrix {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.setup(t)
			pkg, err := DiscoverPackage(path)
			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error for %s, got nil", tc.name)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tc.name, err)
			}
			if pkg != tc.wantPackage {
				t.Errorf("package mismatch: got %q, want %q", pkg, tc.wantPackage)
			}
		})
	}
}

// TestDeriveBaseName_Matrix tests DeriveBaseName across priority order and fallbacks.
func TestDeriveBaseName_Matrix(t *testing.T) {
	matrix := []struct {
		name     string
		opts     Options
		setup    func(t *testing.T) string
		expected string
	}{
		{
			name:     "companion_filename_priority_over_all",
			opts:     Options{CompanionFilename: "custom-bindings.wit.go", Package: "ignore:pkg"},
			setup:    func(t *testing.T) string { return "" },
			expected: "custom-bindings.wit",
		},
		{
			name:     "package_with_namespace_in_options",
			opts:     Options{Package: "my-ns:service-api"},
			setup:    func(t *testing.T) string { return "" },
			expected: "service-api",
		},
		{
			name:     "package_without_namespace_in_options",
			opts:     Options{Package: "standalone_service"},
			setup:    func(t *testing.T) string { return "" },
			expected: "standalone_service",
		},
		{
			name: "package_discovered_from_wit_dir",
			opts: Options{},
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				_ = os.WriteFile(filepath.Join(dir, "api.wit"), []byte("package org:remote-db;\n"), 0644)
				return dir
			},
			expected: "remote-db",
		},
		{
			name: "world_with_dash_suffix_discovered_from_wit",
			opts: Options{},
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				_ = os.WriteFile(filepath.Join(dir, "api.wit"), []byte("world payment-gateway-world {}\n"), 0644)
				return dir
			},
			expected: "payment-gateway",
		},
		{
			name: "world_with_underscore_suffix_discovered_from_wit",
			opts: Options{},
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				_ = os.WriteFile(filepath.Join(dir, "api.wit"), []byte("world checkout_world {}\n"), 0644)
				return dir
			},
			expected: "checkout",
		},
		{
			name: "directory_path_with_wit_suffix_fallback",
			opts: Options{},
			setup: func(t *testing.T) string {
				base := t.TempDir()
				dir := filepath.Join(base, "shopping_cart_wit")
				_ = os.Mkdir(dir, 0755)
				_ = os.WriteFile(filepath.Join(dir, "empty.wit"), []byte("interface cart {}\n"), 0644)
				return dir
			},
			expected: "shopping_cart",
		},
		{
			name:     "empty_fallback",
			opts:     Options{},
			setup:    func(t *testing.T) string { return "" },
			expected: "Wit",
		},
	}

	for _, tc := range matrix {
		t.Run(tc.name, func(t *testing.T) {
			witPath := tc.setup(t)
			got := DeriveBaseName(tc.opts, witPath)
			if got != tc.expected {
				t.Errorf("DeriveBaseName() = %q, want %q", got, tc.expected)
			}
		})
	}
}

// TestGenerateFromAST_Matrix tests AST generation across various languages and configurations.
func TestGenerateFromAST_Matrix(t *testing.T) {
	witContent := `
package test:calc;

interface calculator {
    add: func(a: s32, b: s32) -> s32;
}

world calc-world {
    export calculator;
}
`
	languages := []string{"go", "rust", "typescript", "python", "kotlin", "cpp", "swift"}

	for _, lang := range languages {
		t.Run("lang_"+lang, func(t *testing.T) {
			witDir := t.TempDir()
			witFile := filepath.Join(witDir, "calc.wit")
			if err := os.WriteFile(witFile, []byte(witContent), 0644); err != nil {
				t.Fatalf("failed to write test wit file: %v", err)
			}

			outDir := t.TempDir()
			opts := Options{
				Lang: lang,
				Out:  outDir,
			}

			if err := GenerateFromAST(opts, witFile); err != nil {
				t.Fatalf("GenerateFromAST(%s) failed: %v", lang, err)
			}

			entries, err := os.ReadDir(outDir)
			if err != nil {
				t.Fatalf("failed to read outDir: %v", err)
			}
			if len(entries) == 0 {
				t.Errorf("expected generated files in %s, got none", outDir)
			}
		})
	}

	t.Run("invalid_wit_syntax", func(t *testing.T) {
		witDir := t.TempDir()
		badFile := filepath.Join(witDir, "bad.wit")
		_ = os.WriteFile(badFile, []byte("interface { invalid syntax"), 0644)
		opts := Options{Lang: "go", Out: t.TempDir()}
		if err := GenerateFromAST(opts, badFile); err == nil {
			t.Errorf("expected error for invalid WIT syntax, got nil")
		}
	})

	t.Run("unsupported_language", func(t *testing.T) {
		witDir := t.TempDir()
		witFile := filepath.Join(witDir, "calc.wit")
		_ = os.WriteFile(witFile, []byte(witContent), 0644)
		opts := Options{Lang: "unsupported", Out: t.TempDir()}
		if err := GenerateFromAST(opts, witFile); err == nil {
			t.Errorf("expected error for unsupported language, got nil")
		}
	})

	t.Run("package_override_applied_when_pkg_empty", func(t *testing.T) {
		witDir := t.TempDir()
		witFile := filepath.Join(witDir, "simple.wit")
		_ = os.WriteFile(witFile, []byte("interface math { abs: func(x: s32) -> s32; }\n"), 0644)

		outDir := t.TempDir()
		opts := Options{
			Lang:    "go",
			Out:     outDir,
			Package: "custom_ns:custom_pkg",
		}
		if err := GenerateFromAST(opts, witFile); err != nil {
			t.Fatalf("GenerateFromAST failed: %v", err)
		}
	})
}

// TestRun_Matrix tests the Run entrypoint with a matrix of source configurations and error conditions.
func TestRun_Matrix(t *testing.T) {
	witA := `
package test:bundle;
interface alpha {
    ping: func() -> string;
}
`
	witB := `
interface beta {
    pong: func() -> string;
}
`

	matrix := []struct {
		name        string
		lang        string
		setup       func(t *testing.T) (srcs []string, outDir string)
		expectError bool
		errContains string
	}{
		{
			name: "no_sources",
			lang: "go",
			setup: func(t *testing.T) ([]string, string) {
				return nil, t.TempDir()
			},
			expectError: true,
			errContains: "no WIT sources specified",
		},
		{
			name: "single_file_source",
			lang: "go",
			setup: func(t *testing.T) ([]string, string) {
				dir := t.TempDir()
				f := filepath.Join(dir, "alpha.wit")
				_ = os.WriteFile(f, []byte(witA), 0644)
				return []string{f}, t.TempDir()
			},
			expectError: false,
		},
		{
			name: "single_dir_source",
			lang: "rust",
			setup: func(t *testing.T) ([]string, string) {
				dir := t.TempDir()
				_ = os.WriteFile(filepath.Join(dir, "alpha.wit"), []byte(witA), 0644)
				_ = os.WriteFile(filepath.Join(dir, "beta.wit"), []byte(witB), 0644)
				return []string{dir}, t.TempDir()
			},
			expectError: false,
		},
		{
			name: "multiple_file_sources",
			lang: "typescript",
			setup: func(t *testing.T) ([]string, string) {
				dir := t.TempDir()
				f1 := filepath.Join(dir, "alpha.wit")
				f2 := filepath.Join(dir, "beta.wit")
				_ = os.WriteFile(f1, []byte(witA), 0644)
				_ = os.WriteFile(f2, []byte(witB), 0644)
				return []string{f1, f2}, t.TempDir()
			},
			expectError: false,
		},
		{
			name: "multiple_sources_combining_dir_and_files",
			lang: "python",
			setup: func(t *testing.T) ([]string, string) {
				dir1 := t.TempDir()
				_ = os.WriteFile(filepath.Join(dir1, "alpha.wit"), []byte(witA), 0644)
				dir2 := t.TempDir()
				f2 := filepath.Join(dir2, "beta.wit")
				_ = os.WriteFile(f2, []byte(witB), 0644)
				return []string{dir1, f2}, t.TempDir()
			},
			expectError: false,
		},
		{
			name: "multi_source_with_non_existent_source",
			lang: "go",
			setup: func(t *testing.T) ([]string, string) {
				dir := t.TempDir()
				f1 := filepath.Join(dir, "alpha.wit")
				_ = os.WriteFile(f1, []byte(witA), 0644)
				return []string{f1, filepath.Join(dir, "nonexistent.wit")}, t.TempDir()
			},
			expectError: true,
		},
		{
			name: "invalid_output_dir",
			lang: "go",
			setup: func(t *testing.T) ([]string, string) {
				dir := t.TempDir()
				f := filepath.Join(dir, "alpha.wit")
				_ = os.WriteFile(f, []byte(witA), 0644)
				outBlocker := filepath.Join(t.TempDir(), "blocker")
				_ = os.WriteFile(outBlocker, []byte("file"), 0644)
				return []string{f}, filepath.Join(outBlocker, "sub")
			},
			expectError: true,
			errContains: "failed to create output directory",
		},
	}

	for _, tc := range matrix {
		t.Run(tc.name, func(t *testing.T) {
			srcs, outDir := tc.setup(t)
			opts := Options{
				Lang: tc.lang,
				Out:  outDir,
				Srcs: srcs,
			}
			err := Run(opts)
			if tc.expectError {
				if err == nil {
					t.Fatalf("expected error for %s, got nil", tc.name)
				}
				if tc.errContains != "" && !strings.Contains(err.Error(), tc.errContains) {
					t.Errorf("error %q does not contain expected substring %q", err.Error(), tc.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tc.name, err)
			}
		})
	}
}

// TestInternalHelpers tests edge cases of lower-level helper methods.
func TestInternalHelpers(t *testing.T) {
	t.Run("resolveWitFiles_error_on_missing_dir", func(t *testing.T) {
		_, err := resolveWitFiles(filepath.Join(t.TempDir(), "nonexistent"))
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("parsePackageFromContent_various_forms", func(t *testing.T) {
		cases := []struct {
			content string
			want    string
		}{
			{"package foo:bar;", "foo:bar"},
			{"   package    foo:bar-baz;   ", "foo:bar-baz"},
			{"// package commented:out;\npackage actual:pkg;", "actual:pkg"},
			{"no package here", ""},
		}
		for _, c := range cases {
			got := parsePackageFromContent([]byte(c.content))
			if got != c.want {
				t.Errorf("parsePackageFromContent(%q) = %q, want %q", c.content, got, c.want)
			}
		}
	})

	t.Run("applyPackageOverride", func(t *testing.T) {
		pkg := &ast.Package{}
		applyPackageOverride(pkg, "ns:name")
		if pkg.Namespace != "ns" || pkg.Name != "name" {
			t.Errorf("expected ns:name, got %s:%s", pkg.Namespace, pkg.Name)
		}

		pkgSingle := &ast.Package{}
		applyPackageOverride(pkgSingle, "singlename")
		if pkgSingle.Namespace != "" || pkgSingle.Name != "singlename" {
			t.Errorf("expected :singlename, got %s:%s", pkgSingle.Namespace, pkgSingle.Name)
		}

		// When already set, override should not overwrite
		applyPackageOverride(pkgSingle, "other:ignored")
		if pkgSingle.Name != "singlename" {
			t.Errorf("expected singlename to be preserved, got %s", pkgSingle.Name)
		}
	})

	t.Run("writeOutputFiles_mkdir_error", func(t *testing.T) {
		// Target file whose parent directory cannot be created because a regular file is in the path
		dir := t.TempDir()
		blockingFile := filepath.Join(dir, "blocking")
		_ = os.WriteFile(blockingFile, []byte("file"), 0644)

		files := []OutputFile{
			{Name: filepath.Join("blocking", "sub", "output.txt"), Content: "hello"},
		}
		err := writeOutputFiles(dir, files)
		if err == nil {
			t.Errorf("expected error when directory creation fails, got nil")
		}
	})

	t.Run("copyFile_nonexistent_src", func(t *testing.T) {
		err := copyFile(filepath.Join(t.TempDir(), "missing.txt"), filepath.Join(t.TempDir(), "dest.txt"))
		if err == nil {
			t.Errorf("expected error copying non-existent file, got nil")
		}
	})

	t.Run("parseWorldsFromFile_nonexistent", func(t *testing.T) {
		res := parseWorldsFromFile(filepath.Join(t.TempDir(), "missing.wit"), make(map[string]bool))
		if len(res) != 0 {
			t.Errorf("expected empty result, got %v", res)
		}
	})

	t.Run("parsePackageFromFile_nonexistent", func(t *testing.T) {
		res := parsePackageFromFile(filepath.Join(t.TempDir(), "missing.wit"))
		if res != "" {
			t.Errorf("expected empty result, got %v", res)
		}
	})

	t.Run("writeOutputFiles_file_write_error", func(t *testing.T) {
		dir := t.TempDir()
		subDir := filepath.Join(dir, "output.txt")
		_ = os.Mkdir(subDir, 0755)
		files := []OutputFile{
			{Name: "output.txt", Content: "should fail because target is a directory"},
		}
		err := writeOutputFiles(dir, files)
		if err == nil {
			t.Errorf("expected error writing file over existing directory, got nil")
		}
	})

	t.Run("copyWitDirEntries_error", func(t *testing.T) {
		srcDir := t.TempDir()
		witFile := filepath.Join(srcDir, "valid.wit")
		_ = os.WriteFile(witFile, []byte("interface foo {}"), 0644)
		destDir := t.TempDir()
		_ = os.Mkdir(filepath.Join(destDir, "valid.wit"), 0755)
		err := copyWitDirEntries(srcDir, destDir)
		if err == nil {
			t.Errorf("expected error when copy fails, got nil")
		}
	})
}

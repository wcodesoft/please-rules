package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestGeneratorForLang(t *testing.T) {
	cases := map[string]string{
		"rust":   "rust",
		"go":     "go",
		"cpp":    "cpp",
		"cc":     "cpp",
		"c":      "c",
		"swift":  "c",
		"kotlin": "c",
		"ts":     "c",
		"python": "c",
	}

	for lang, expected := range cases {
		got := GeneratorForLang(lang)
		if got != expected {
			t.Errorf("GeneratorForLang(%q) = %q; want %q", lang, got, expected)
		}
	}
}

func TestDiscoverPackage(t *testing.T) {
	tmpDir := t.TempDir()
	witContent := `
package test:structures;

world my-world {
}
`
	witFile := filepath.Join(tmpDir, "demo.wit")
	if err := os.WriteFile(witFile, []byte(witContent), 0644); err != nil {
		t.Fatalf("failed to write test wit file: %v", err)
	}

	pkg, err := DiscoverPackage(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverPackage failed: %v", err)
	}
	if pkg != "test:structures" {
		t.Errorf("expected package 'test:structures', got %q", pkg)
	}
}

func TestToPascalCase(t *testing.T) {
	cases := map[string]string{
		"structures":      "Structures",
		"two-sum":         "TwoSum",
		"two_sum":         "TwoSum",
		"test:structures": "TestStructures",
		"example_code":    "ExampleCode",
	}
	for in, expected := range cases {
		got := ToPascalCase(in)
		if got != expected {
			t.Errorf("ToPascalCase(%q) = %q; want %q", in, got, expected)
		}
	}
}

func TestDeriveBaseName(t *testing.T) {
	opts := Options{CompanionFilename: "MyCustom.swift"}
	if got := DeriveBaseName(opts, ""); got != "MyCustom" {
		t.Errorf("expected MyCustom, got %q", got)
	}

	opts = Options{Package: "example:structures"}
	if got := DeriveBaseName(opts, ""); got != "structures" {
		t.Errorf("expected structures, got %q", got)
	}
}

func TestGenerateCompanions(t *testing.T) {
	tmpDir := t.TempDir()

	for _, lang := range []string{"swift", "kotlin", "ts", "python"} {
		subDir := filepath.Join(tmpDir, lang)
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatalf("failed to mkdir: %v", err)
		}
		opts := Options{
			Lang:    lang,
			Out:     subDir,
			Package: "test:structures",
		}
		if err := GenerateCompanions(opts, ""); err != nil {
			t.Fatalf("GenerateCompanions failed for %s: %v", lang, err)
		}
	}

	// Swift auto-discovered filename and module.modulemap
	if _, err := os.Stat(filepath.Join(tmpDir, "swift", "Structures.swift")); err != nil {
		t.Errorf("Swift auto-discovered companion missing: %v", err)
	}
	mapContent, err := os.ReadFile(filepath.Join(tmpDir, "swift", "module.modulemap"))
	if err != nil {
		t.Errorf("Swift module.modulemap missing: %v", err)
	} else if !strings.Contains(string(mapContent), "module Structures") {
		t.Errorf("expected module Structures in modulemap, got: %s", string(mapContent))
	}

	// Kotlin auto-discovered filename and package
	ktFile := filepath.Join(tmpDir, "kotlin", "Structures.kt")
	ktContent, err := os.ReadFile(ktFile)
	if err != nil {
		t.Errorf("Kotlin auto-discovered companion missing: %v", err)
	} else if !strings.Contains(string(ktContent), "package test.structures") {
		t.Errorf("expected package test.structures, got: %s", string(ktContent))
	}

	// TypeScript index.d.ts
	if _, err := os.Stat(filepath.Join(tmpDir, "ts", "index.d.ts")); err != nil {
		t.Errorf("TypeScript companion missing")
	}

	// Python __init__.py
	if _, err := os.Stat(filepath.Join(tmpDir, "python", "__init__.py")); err != nil {
		t.Errorf("Python companion missing")
	}
}

func TestGenerateCompanionsCustom(t *testing.T) {
	tmpDir := t.TempDir()

	swiftDir := filepath.Join(tmpDir, "swift")
	_ = os.MkdirAll(swiftDir, 0755)
	opts := Options{
		Lang:              "swift",
		Out:               swiftDir,
		CompanionFilename: "CustomBindings.swift",
		ModuleName:        "CustomModule",
	}
	if err := GenerateCompanions(opts, ""); err != nil {
		t.Fatalf("GenerateCompanions custom swift failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(swiftDir, "CustomBindings.swift")); err != nil {
		t.Errorf("Custom Swift companion missing")
	}
	mapContent, err := os.ReadFile(filepath.Join(swiftDir, "module.modulemap"))
	if err != nil || !strings.Contains(string(mapContent), "module CustomModule") {
		t.Errorf("expected module CustomModule, got: %s", string(mapContent))
	}
}

func TestGenerateFromAST(t *testing.T) {
	tmpDir := t.TempDir()
	witContent := `
package test:structures;

interface two-sum {
    solve: func(nums: list<s32>, target: s32) -> list<s32>;
    reset: func();
}
`
	witFile := filepath.Join(tmpDir, "two_sum.wit")
	if err := os.WriteFile(witFile, []byte(witContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 1. Kotlin
	ktOut := filepath.Join(tmpDir, "kt_out")
	_ = os.MkdirAll(ktOut, 0755)
	optsKt := Options{
		Lang: "kotlin",
		Out:  ktOut,
		Srcs: []string{witFile},
	}
	if err := Run(optsKt); err != nil {
		t.Fatalf("Run(kotlin) failed: %v", err)
	}
	ktFile := filepath.Join(ktOut, "Structures.kt")
	ktBytes, err := os.ReadFile(ktFile)
	if err != nil {
		t.Fatalf("failed to read %s: %v", ktFile, err)
	}
	ktCode := string(ktBytes)
	if !strings.Contains(ktCode, "package test.structures") {
		t.Errorf("expected package test.structures, got: %s", ktCode)
	}
	if !strings.Contains(ktCode, "public interface TwoSum {") {
		t.Errorf("expected public interface TwoSum, got: %s", ktCode)
	}
	if !strings.Contains(ktCode, "fun solve(nums: List<Int>, target: Int): List<Int>") {
		t.Errorf("expected fun solve, got: %s", ktCode)
	}

	// 2. Swift
	swiftOut := filepath.Join(tmpDir, "swift_out")
	_ = os.MkdirAll(swiftOut, 0755)
	optsSwift := Options{
		Lang: "swift",
		Out:  swiftOut,
		Srcs: []string{witFile},
	}
	if err := Run(optsSwift); err != nil {
		t.Fatalf("Run(swift) failed: %v", err)
	}
	swiftFile := filepath.Join(swiftOut, "Structures.swift")
	swiftBytes, err := os.ReadFile(swiftFile)
	if err != nil {
		t.Fatalf("failed to read %s: %v", swiftFile, err)
	}
	swiftCode := string(swiftBytes)
	if !strings.Contains(swiftCode, "public protocol TwoSum {") {
		t.Errorf("expected public protocol TwoSum, got: %s", swiftCode)
	}
	if !strings.Contains(swiftCode, "func solve(nums: [Int32], target: Int32) -> [Int32]") {
		t.Errorf("expected func solve, got: %s", swiftCode)
	}

	// 3. TypeScript
	tsOut := filepath.Join(tmpDir, "ts_out")
	_ = os.MkdirAll(tsOut, 0755)
	optsTS := Options{
		Lang: "ts",
		Out:  tsOut,
		Srcs: []string{witFile},
	}
	if err := Run(optsTS); err != nil {
		t.Fatalf("Run(ts) failed: %v", err)
	}
	tsFile := filepath.Join(tsOut, "structures.d.ts")
	tsBytes, err := os.ReadFile(tsFile)
	if err != nil {
		t.Fatalf("failed to read %s: %v", tsFile, err)
	}
	tsCode := string(tsBytes)
	if !strings.Contains(tsCode, "export interface TwoSum {") {
		t.Errorf("expected export interface TwoSum, got: %s", tsCode)
	}
	if !strings.Contains(tsCode, "solve(nums: number[], target: number): number[];") {
		t.Errorf("expected solve signature, got: %s", tsCode)
	}

	// 4. Python
	pyOut := filepath.Join(tmpDir, "py_out")
	_ = os.MkdirAll(pyOut, 0755)
	optsPy := Options{
		Lang: "python",
		Out:  pyOut,
		Srcs: []string{witFile},
	}
	if err := Run(optsPy); err != nil {
		t.Fatalf("Run(python) failed: %v", err)
	}
	pyFile := filepath.Join(pyOut, "__init__.py")
	pyBytes, err := os.ReadFile(pyFile)
	if err != nil {
		t.Fatalf("failed to read %s: %v", pyFile, err)
	}
	pyCode := string(pyBytes)
	if !strings.Contains(pyCode, "class TwoSum(Protocol):") {
		t.Errorf("expected class TwoSum(Protocol), got: %s", pyCode)
	}
	if !strings.Contains(pyCode, "def solve(self, nums: List[int], target: int) -> List[int]:") {
		t.Errorf("expected def solve, got: %s", pyCode)
	}
}


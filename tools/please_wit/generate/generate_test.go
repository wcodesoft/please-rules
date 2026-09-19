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

	// 5. Rust
	rustOut := filepath.Join(tmpDir, "rust_out")
	_ = os.MkdirAll(rustOut, 0755)
	optsRust := Options{
		Lang: "rust",
		Out:  rustOut,
		Srcs: []string{witFile},
	}
	if err := Run(optsRust); err != nil {
		t.Fatalf("Run(rust) failed: %v", err)
	}
	rustFile := filepath.Join(rustOut, "structures.rs")
	rustBytes, err := os.ReadFile(rustFile)
	if err != nil {
		t.Fatalf("failed to read %s: %v", rustFile, err)
	}
	rustCode := string(rustBytes)
	if !strings.Contains(rustCode, "pub trait TwoSum {") {
		t.Errorf("expected pub trait TwoSum, got: %s", rustCode)
	}
	if !strings.Contains(rustCode, "fn solve(&mut self, nums: Vec<i32>, target: i32) -> Vec<i32>;") {
		t.Errorf("expected fn solve, got: %s", rustCode)
	}

	// 6. Go
	goOut := filepath.Join(tmpDir, "go_out")
	_ = os.MkdirAll(goOut, 0755)
	optsGo := Options{
		Lang: "go",
		Out:  goOut,
		Srcs: []string{witFile},
	}
	if err := Run(optsGo); err != nil {
		t.Fatalf("Run(go) failed: %v", err)
	}
	goFile := filepath.Join(goOut, "structures.go")
	goBytes, err := os.ReadFile(goFile)
	if err != nil {
		t.Fatalf("failed to read %s: %v", goFile, err)
	}
	goCode := string(goBytes)
	if !strings.Contains(goCode, "type TwoSum interface {") {
		t.Errorf("expected type TwoSum interface, got: %s", goCode)
	}
	if !strings.Contains(goCode, "Solve(nums []int32, target int32) []int32") {
		t.Errorf("expected Solve method, got: %s", goCode)
	}

	// 7. C++
	cppOut := filepath.Join(tmpDir, "cpp_out")
	_ = os.MkdirAll(cppOut, 0755)
	optsCpp := Options{
		Lang: "cpp",
		Out:  cppOut,
		Srcs: []string{witFile},
	}
	if err := Run(optsCpp); err != nil {
		t.Fatalf("Run(cpp) failed: %v", err)
	}
	cppHFile := filepath.Join(cppOut, "structures.h")
	cppHBytes, err := os.ReadFile(cppHFile)
	if err != nil {
		t.Fatalf("failed to read %s: %v", cppHFile, err)
	}
	cppHCode := string(cppHBytes)
	if !strings.Contains(cppHCode, "class TwoSum {") {
		t.Errorf("expected class TwoSum, got: %s", cppHCode)
	}
	if !strings.Contains(cppHCode, "virtual std::vector<int32_t> solve(") {
		t.Errorf("expected virtual solve method, got: %s", cppHCode)
	}
}


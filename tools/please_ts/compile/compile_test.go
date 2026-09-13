package compile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEmitOutput(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "compile_emit_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	srcFile := filepath.Join(tmpDir, "calculator.ts")
	if err := os.WriteFile(srcFile, []byte("export const x = 1;"), 0644); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(tmpDir, "out")
	opts := Options{
		Out:        outDir,
		Srcs:       []string{srcFile},
		ModuleName: "@domain/calculator",
	}

	if err := emitOutput(opts, nil); err != nil {
		t.Fatalf("emitOutput failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "calculator.ts")); err != nil {
		t.Errorf("expected copied calculator.ts in out: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "ts_metadata.json")); err != nil {
		t.Errorf("expected ts_metadata.json in out: %v", err)
	}
}

func TestExpandCommaSeparated(t *testing.T) {
	input := []string{"a, b, c", "d,e"}
	output := ExpandCommaSeparated(input)
	if len(output) != 5 {
		t.Fatalf("expected 5 items, got %d", len(output))
	}
	expected := []string{"a", "b", "c", "d", "e"}
	for i, v := range output {
		if v != expected[i] {
			t.Errorf("expected %s at index %d, got %s", expected[i], i, v)
		}
	}
}

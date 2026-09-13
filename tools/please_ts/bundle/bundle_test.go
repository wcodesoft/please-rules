package bundle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuiltinBundler(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bundle_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	libFile := filepath.Join(tmpDir, "math.ts")
	if err := os.WriteFile(libFile, []byte("export const add = (a: number, b: number) => a + b;"), 0644); err != nil {
		t.Fatal(err)
	}

	mainFile := filepath.Join(tmpDir, "index.ts")
	mainContent := `import { add } from "./math.ts";
console.log(add(1, 2));
`
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	outFile := filepath.Join(tmpDir, "bundle.js")
	opts := Options{
		Main:   mainFile,
		Out:    outFile,
		Srcs:   []string{mainFile, libFile},
		Format: "iife",
		Minify: false,
	}

	if err := Run(opts); err != nil {
		t.Fatalf("bundle.Run failed: %v", err)
	}

	outBytes, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}

	content := string(outBytes)
	if !strings.Contains(content, "(function()") {
		t.Errorf("expected IIFE wrapper in output: %s", content)
	}
	if !strings.Contains(content, "export const add") {
		t.Errorf("expected bundled math module in output: %s", content)
	}
}

package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectFilesSkipsTestFilesAndExtractsSites(t *testing.T) {
	dir := t.TempDir()
	srcFile := filepath.Join(dir, "calc.go")
	testFile := filepath.Join(dir, "calc_test.go")

	if err := os.WriteFile(srcFile, []byte("package calc\nfunc Add(a, b int) int { return a + b }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testFile, []byte("package calc\nfunc TestAdd() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	files, err := collectFiles([]string{dir}, "", dir, "", nil)
	if err != nil {
		t.Fatalf("collectFiles: %v", err)
	}
	if len(files) != 1 || files[0].relPath != "calc.go" || len(files[0].sites) == 0 {
		t.Fatalf("unexpected collected files: %+v", files)
	}
}

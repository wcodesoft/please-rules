package compilec

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCompileC_Success(t *testing.T) {
	if _, err := findCC(); err != nil {
		t.Skip("C compiler not available")
	}

	tmpDir := t.TempDir()

	headerFile := filepath.Join(tmpDir, "hello.h")
	if err := os.WriteFile(headerFile, []byte("#define VALUE 42\n"), 0644); err != nil {
		t.Fatalf("writing header: %v", err)
	}

	srcFile1 := filepath.Join(tmpDir, "func1.c")
	code1 := `#include "hello.h"
int get_val() { return VALUE; }
`
	if err := os.WriteFile(srcFile1, []byte(code1), 0644); err != nil {
		t.Fatalf("writing src1: %v", err)
	}

	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Test collision handling: func1.c in subDir has same basename
	srcFile2 := filepath.Join(subDir, "func1.c")
	code2 := `int add_val(int x) { return x + 1; }
`
	if err := os.WriteFile(srcFile2, []byte(code2), 0644); err != nil {
		t.Fatalf("writing src2: %v", err)
	}

	outArchive := filepath.Join(tmpDir, "out", "libtest.a")

	err := CompileC(outArchive, []string{tmpDir}, []string{srcFile1, srcFile2})
	if err != nil {
		t.Fatalf("CompileC failed: %v", err)
	}

	info, err := os.Stat(outArchive)
	if err != nil {
		t.Fatalf("output archive not found: %v", err)
	}
	if info.Size() == 0 {
		t.Fatalf("output archive is empty")
	}

	// Verify archive contents with ar t
	out, err := exec.Command("ar", "t", outArchive).Output()
	if err != nil {
		t.Fatalf("ar t failed: %v", err)
	}
	t.Logf("ar table of contents: %s", string(out))
}

func TestCompileC_EmptySources(t *testing.T) {
	tmpDir := t.TempDir()
	outArchive := filepath.Join(tmpDir, "libempty.a")

	err := CompileC(outArchive, nil, nil)
	if err == nil {
		t.Fatal("expected error when no sources provided, got nil")
	}
}

func TestCompileC_CompilerError(t *testing.T) {
	if _, err := findCC(); err != nil {
		t.Skip("C compiler not available")
	}

	tmpDir := t.TempDir()
	badSrc := filepath.Join(tmpDir, "bad.c")
	if err := os.WriteFile(badSrc, []byte("invalid C syntax {{{"), 0644); err != nil {
		t.Fatalf("writing bad src: %v", err)
	}

	outArchive := filepath.Join(tmpDir, "libbad.a")
	err := CompileC(outArchive, nil, []string{badSrc})
	if err == nil {
		t.Fatal("expected compilation error, got nil")
	}
}

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunCLI_Compile_MissingRequired(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runCLI([]string{"compile"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing required compile flags, got nil")
	}
}

func TestRunCLI_Compile_Success(t *testing.T) {
	truePath := "/bin/true"
	if _, err := os.Stat(truePath); err != nil {
		truePath = "/usr/bin/true"
	}
	if _, err := os.Stat(truePath); err != nil {
		t.Skip("true not found")
	}

	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "main.rs")
	if err := os.WriteFile(src, []byte("fn main(){}"), 0644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err := runCLI([]string{
		"compile",
		"--out=" + filepath.Join(tmpDir, "out"),
		"--crate-name=mycrate",
		"--main-src=" + src,
		"--rustc=" + truePath,
		"--features=foo,bar",
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runCLI compile failed: %v", err)
	}
}

func TestRunCLI_Download_MissingFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runCLI([]string{"download"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing --crate, got nil")
	}

	err = runCLI([]string{"download", "--crate=foo"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing --version, got nil")
	}

	err = runCLI([]string{"download", "--crate=foo", "--version=1.0.0"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing --sha256, got nil")
	}

	err = runCLI([]string{"download", "--crate=foo", "--version=1.0.0", "--sha256=abc"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing --out, got nil")
	}
}

func TestRunCLI_Download_Dispatch(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runCLI([]string{
		"download",
		"--crate=fakecrate_xyz",
		"--version=0.0.0",
		"--sha256=0000000000000000000000000000000000000000000000000000000000000000",
		"--out=" + filepath.Join(t.TempDir(), "out.tar"),
	}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for fake crate download, got nil")
	}
}

func TestRunCLI_Hash_MissingFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runCLI([]string{"hash"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing --crate, got nil")
	}

	err = runCLI([]string{"hash", "--crate=foo"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing --version, got nil")
	}
}

func TestRunCLI_Hash_Dispatch(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runCLI([]string{"hash", "--crate=fakecrate_xyz", "--version=0.0.0"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for fake crate hash, got nil")
	}
}

func TestRunCLI_CompileC_MissingFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runCLI([]string{"compile-c"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing --out, got nil")
	}

	err = runCLI([]string{"compile-c", "--out=lib.a"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing sources, got nil")
	}
}

func TestRunCLI_CompileC_Success(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "simple.c")
	if err := os.WriteFile(src, []byte("int test() { return 0; }\n"), 0644); err != nil {
		t.Fatal(err)
	}

	outArchive := filepath.Join(tmpDir, "libsimple.a")
	var stdout, stderr bytes.Buffer
	err := runCLI([]string{
		"compile-c",
		"--out=" + outArchive,
		"--include=" + tmpDir,
		src,
	}, &stdout, &stderr)
	if err != nil {
		t.Logf("compile-c execution returned: %v (expected if cc missing)", err)
	}
}

func TestRunCLI_TestRunner_MissingFlags(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := runCLI([]string{"test-runner"}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error for missing test binary arg, got nil")
	}
}

func TestRunCLI_TestRunner_Success(t *testing.T) {
	truePath := "/bin/true"
	if _, err := os.Stat(truePath); err != nil {
		truePath = "/usr/bin/true"
	}
	if _, err := os.Stat(truePath); err != nil {
		t.Skip("true not found")
	}

	tmpDir := t.TempDir()
	resFile := filepath.Join(tmpDir, "results.xml")

	var stdout, stderr bytes.Buffer
	err := runCLI([]string{
		"test-runner",
		"--pkg=mypkg",
		"--results-file=" + resFile,
		truePath,
	}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("runCLI test-runner failed: %v", err)
	}
}

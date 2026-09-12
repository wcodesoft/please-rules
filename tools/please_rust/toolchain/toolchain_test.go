package toolchain

import (
	"testing"
)

func TestFindRustc(t *testing.T) {
	rustcPath, err := FindRustc("")
	if err != nil {
		t.Logf("FindRustc returned error (expected if rustc not installed): %v", err)
	} else {
		t.Logf("Found rustc at: %s", rustcPath)
	}
}

func TestFindLlvmProfdata(t *testing.T) {
	path, err := FindLlvmProfdata("")
	if err != nil {
		t.Logf("FindLlvmProfdata returned error (expected if llvm-profdata not installed): %v", err)
	} else {
		t.Logf("Found llvm-profdata at: %s", path)
	}
}

func TestFindLlvmCov(t *testing.T) {
	path, err := FindLlvmCov("")
	if err != nil {
		t.Logf("FindLlvmCov returned error (expected if llvm-cov not installed): %v", err)
	} else {
		t.Logf("Found llvm-cov at: %s", path)
	}
}

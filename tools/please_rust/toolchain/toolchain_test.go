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


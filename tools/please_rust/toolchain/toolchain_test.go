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

func TestFindCargo(t *testing.T) {
	cargoPath, err := FindCargo("")
	if err != nil {
		t.Logf("FindCargo returned error (expected if cargo not installed): %v", err)
	} else {
		t.Logf("Found cargo at: %s", cargoPath)
	}
}

package main

import (
	"testing"
)

func TestHandleCompileValidation(t *testing.T) {
	// Missing --out
	err := handleCompile([]string{"--srcs", "main.kt"})
	if err == nil {
		t.Errorf("expected error when --out is missing")
	}

	// Missing --srcs
	err = handleCompile([]string{"--out", "out.wasm"})
	if err == nil {
		t.Errorf("expected error when --srcs has no files")
	}
}

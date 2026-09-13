package toolchain

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveKotlincConfigured(t *testing.T) {
	tmpDir := t.TempDir()
	fakeKotlinc := filepath.Join(tmpDir, "fake_kotlinc")
	if err := os.WriteFile(fakeKotlinc, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("failed to create fake kotlinc: %v", err)
	}

	got, err := ResolveKotlinc(fakeKotlinc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != fakeKotlinc {
		t.Errorf("got %q, want %q", got, fakeKotlinc)
	}
}

func TestResolveKotlincFallback(t *testing.T) {
	got, err := ResolveKotlinc("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == "" {
		t.Errorf("expected non-empty fallback")
	}
}

func TestResolveJavaConfigured(t *testing.T) {
	tmpDir := t.TempDir()
	fakeJava := filepath.Join(tmpDir, "fake_java")
	if err := os.WriteFile(fakeJava, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("failed to create fake java: %v", err)
	}

	got, err := ResolveJava(fakeJava)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != fakeJava {
		t.Errorf("got %q, want %q", got, fakeJava)
	}
}

func TestResolveJavaFallback(t *testing.T) {
	got, err := ResolveJava("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == "" {
		t.Errorf("expected non-empty fallback")
	}
}

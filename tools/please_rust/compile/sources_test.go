package compile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveMainSrc_Explicit(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "custom.rs")
	if err := os.WriteFile(src, []byte("fn main(){}"), 0644); err != nil {
		t.Fatal(err)
	}
	found := resolveMainSrc(src, "bin", []string{src})
	if found != src {
		t.Errorf("got %s, want %s", found, src)
	}
}

func TestResolveMainSrc_Fallback(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "other.rs")
	if err := os.WriteFile(src, []byte("fn main(){}"), 0644); err != nil {
		t.Fatal(err)
	}
	found := resolveMainSrc("", "bin", []string{src})
	if found != src {
		t.Errorf("got %s, want %s", found, src)
	}
}

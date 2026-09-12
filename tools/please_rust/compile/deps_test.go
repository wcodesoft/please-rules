package compile

import (
	"testing"
)

func TestSanitizeCrateName(t *testing.T) {
	cases := map[string]string{
		"tree-sitter-rust": "tree_sitter_rust",
		"normal_name":      "normal_name",
		"foo-bar-baz":      "foo_bar_baz",
	}
	for in, expected := range cases {
		if got := sanitizeCrateName(in); got != expected {
			t.Errorf("sanitizeCrateName(%s) = %s, expected %s", in, got, expected)
		}
	}
}

func TestExtractCrateName(t *testing.T) {
	cases := map[string]string{
		"libtree_sitter_rust.rlib": "tree_sitter_rust",
		"libfoo-bar.rlib":          "foo_bar",
		"libtest.so":               "test",
	}
	for in, expected := range cases {
		if got := extractCrateName(in); got != expected {
			t.Errorf("extractCrateName(%s) = %s, expected %s", in, got, expected)
		}
	}
}

func TestIsLibFile(t *testing.T) {
	valid := []string{"foo.rlib", "libfoo.so", "bar.dylib", "baz.dll"}
	for _, path := range valid {
		if !isLibFile(path) {
			t.Errorf("expected %s to be recognized as lib file", path)
		}
	}

	invalid := []string{"foo.rs", "main.go", "archive.tar", "lib.a"}
	for _, path := range invalid {
		if isLibFile(path) {
			t.Errorf("expected %s to NOT be recognized as lib file", path)
		}
	}
}

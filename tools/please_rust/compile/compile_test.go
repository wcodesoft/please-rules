package compile

import (
	"testing"
)

func TestBuildRustcArgs(t *testing.T) {
	opts := Options{
		Out:       "plz-out/bin/my_bin",
		CrateName: "my-bin",
		CrateType: "bin",
		Edition:   "2021",
		MainSrc:   "main.rs",
		Version:   "0.1.0",
		Flags:     "-C opt-level=3",
		Inputs: []string{
			"main.rs",
			"lib.rs",
			"path/to/libfoo.rlib",
			"another/path/libbar_baz.rlib",
		},
	}

	args, err := BuildRustcArgs(opts, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedPrefix := []string{
		"--edition", "2021",
		"--crate-type", "bin",
		"--crate-name", "my_bin",
		"-o", "plz-out/bin/my_bin",
	}

	for i := 0; i < len(expectedPrefix); i += 2 {
		flag := expectedPrefix[i]
		val := expectedPrefix[i+1]
		found := false
		for j, a := range args {
			if a == flag && j+1 < len(args) && args[j+1] == val {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected flag %s %s in args %v", flag, val, args)
		}
	}

	// Verify externs
	hasFoo := false
	hasBar := false
	for _, a := range args {
		if a == "foo=path/to/libfoo.rlib" {
			hasFoo = true
		}
		if a == "bar_baz=another/path/libbar_baz.rlib" {
			hasBar = true
		}
	}
	if !hasFoo || !hasBar {
		t.Errorf("missing externs in args: %v", args)
	}

	// Main src must be the last argument
	if args[len(args)-1] != "main.rs" {
		t.Errorf("expected last arg to be main.rs, got %s", args[len(args)-1])
	}
}

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

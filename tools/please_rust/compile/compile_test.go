package compile

import (
	"os"
	"path/filepath"
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

	args, err := BuildRustcArgs(opts)
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

func TestBuildRustcArgs_WithMeta(t *testing.T) {
	tmpDir := t.TempDir()
	metaPath := filepath.Join(tmpDir, "crate_meta.json")
	metaContent := `{"lib_src":"custom/path/lib.rs","edition":"2018"}`
	if err := os.WriteFile(metaPath, []byte(metaContent), 0644); err != nil {
		t.Fatalf("writing meta file: %v", err)
	}

	opts := Options{
		Out:       "plz-out/lib/libfoo.rlib",
		CrateName: "foo",
		Meta:      metaPath,
	}

	args, err := BuildRustcArgs(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hasEdition := false
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--edition" && args[i+1] == "2018" {
			hasEdition = true
			break
		}
	}
	if !hasEdition {
		t.Errorf("expected --edition 2018 in args: %v", args)
	}

	if args[len(args)-1] != "custom/path/lib.rs" {
		t.Errorf("expected last arg to be custom/path/lib.rs, got %s", args[len(args)-1])
	}
}

func TestBuildRustcArgs_CrateTypes(t *testing.T) {
	testOpts := Options{
		Out:       "plz-out/bin/my_test",
		CrateName: "my_test",
		CrateType: "test",
		MainSrc:   "test.rs",
	}
	args, err := BuildRustcArgs(testOpts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hasTest := false
	for _, a := range args {
		if a == "--test" {
			hasTest = true
			break
		}
	}
	if !hasTest {
		t.Errorf("expected --test in args: %v", args)
	}

	pmOpts := Options{
		Out:       "plz-out/lib/libpm.so",
		CrateName: "pm",
		CrateType: "proc-macro",
		MainSrc:   "lib.rs",
	}
	args, err = BuildRustcArgs(pmOpts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	hasExternPM := false
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--extern" && args[i+1] == "proc_macro" {
			hasExternPM = true
			break
		}
	}
	if !hasExternPM {
		t.Errorf("expected --extern proc_macro in args: %v", args)
	}
}

func TestBuildRustcArgs_FeaturesAndNativeLib(t *testing.T) {
	opts := Options{
		Out:       "plz-out/lib/libmy.rlib",
		CrateName: "my",
		MainSrc:   "lib.rs",
		Features:  []string{"extra", "serde"},
		NativeLib: "path/to/libhelper.a",
	}
	args, err := BuildRustcArgs(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedFeats := map[string]bool{
		`feature="extra"`: false,
		`feature="serde"`: false,
	}
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--cfg" {
			if _, ok := expectedFeats[args[i+1]]; ok {
				expectedFeats[args[i+1]] = true
			}
		}
	}
	for f, seen := range expectedFeats {
		if !seen {
			t.Errorf("expected --cfg %s in args: %v", f, args)
		}
	}

	hasNativeDir := false
	hasNativeLib := false
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-L" && args[i+1] == "native=path/to" {
			hasNativeDir = true
		}
		if args[i] == "-l" && args[i+1] == "static=helper" {
			hasNativeLib = true
		}
	}
	if !hasNativeDir || !hasNativeLib {
		t.Errorf("missing native lib flags in args: %v", args)
	}
}

func TestBuildRustcArgs_ValidationErrors(t *testing.T) {
	_, err := BuildRustcArgs(Options{Out: "foo", MainSrc: "lib.rs"})
	if err == nil {
		t.Error("expected error for missing CrateName, got nil")
	}

	_, err = BuildRustcArgs(Options{CrateName: "foo", MainSrc: "lib.rs"})
	if err == nil {
		t.Error("expected error for missing Out, got nil")
	}

	_, err = BuildRustcArgs(Options{CrateName: "foo", Out: "foo", Meta: "/nonexistent/meta.json"})
	if err == nil {
		t.Error("expected error for nonexistent Meta file, got nil")
	}

	_, err = BuildRustcArgs(Options{CrateName: "foo", Out: "foo"})
	if err == nil {
		t.Error("expected error for unresolvable MainSrc, got nil")
	}
}

func TestBuildCargoVersionEnv(t *testing.T) {
	if env := buildCargoVersionEnv(""); len(env) != 0 {
		t.Errorf("expected empty env for empty version, got %v", env)
	}

	env := buildCargoVersionEnv("1.2.3-alpha.1+build999")
	expected := map[string]string{
		"CARGO_PKG_VERSION":       "1.2.3-alpha.1+build999",
		"CARGO_PKG_VERSION_MAJOR": "1",
		"CARGO_PKG_VERSION_MINOR": "2",
		"CARGO_PKG_VERSION_PATCH": "3",
	}
	for _, e := range env {
		for k, v := range expected {
			if e == k+"="+v {
				delete(expected, k)
			}
		}
	}
	if len(expected) > 0 {
		t.Errorf("missing expected env vars: %v in %v", expected, env)
	}
}

func TestEnsureOutputDir(t *testing.T) {
	tmpDir := t.TempDir()
	nested := filepath.Join(tmpDir, "sub1", "sub2", "out.rlib")
	if err := ensureOutputDir(nested); err != nil {
		t.Fatalf("ensureOutputDir failed: %v", err)
	}
	if fi, err := os.Stat(filepath.Dir(nested)); err != nil || !fi.IsDir() {
		t.Errorf("expected directory to exist: %v", err)
	}

	if err := ensureOutputDir("out.rlib"); err != nil {
		t.Errorf("unexpected error for simple file: %v", err)
	}
}

func TestRun_Success(t *testing.T) {
	truePath := "/bin/true"
	if _, err := os.Stat(truePath); err != nil {
		truePath = "/usr/bin/true"
	}
	if _, err := os.Stat(truePath); err != nil {
		t.Skip("true command not found")
	}

	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "main.rs")
	if err := os.WriteFile(srcFile, []byte("fn main() {}"), 0644); err != nil {
		t.Fatalf("writing dummy main.rs: %v", err)
	}

	opts := Options{
		Out:       filepath.Join(tmpDir, "bin", "out"),
		CrateName: "test_crate",
		MainSrc:   srcFile,
		Rustc:     truePath,
		Version:   "1.0.0",
	}

	if err := Run(opts); err != nil {
		t.Fatalf("Run failed: %v", err)
	}
}

func TestRun_Failure(t *testing.T) {
	falsePath := "/bin/false"
	if _, err := os.Stat(falsePath); err != nil {
		falsePath = "/usr/bin/false"
	}
	if _, err := os.Stat(falsePath); err != nil {
		t.Skip("false command not found")
	}

	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "main.rs")
	if err := os.WriteFile(srcFile, []byte("fn main() {}"), 0644); err != nil {
		t.Fatalf("writing dummy main.rs: %v", err)
	}

	opts := Options{
		Out:       filepath.Join(tmpDir, "bin", "out"),
		CrateName: "test_crate",
		MainSrc:   srcFile,
		Rustc:     falsePath,
	}

	if err := Run(opts); err == nil {
		t.Fatal("expected Run to fail with false binary, got nil")
	}
}

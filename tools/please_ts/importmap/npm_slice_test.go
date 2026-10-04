package importmap

import (
	"os"
	"path/filepath"
	"testing"

	"tools/please_ts/npmcache"
)

func writeNpmSlice(t *testing.T, root, dir string, s npmcache.Slice) string {
	t.Helper()
	d := filepath.Join(root, dir)
	if err := os.MkdirAll(d, 0755); err != nil {
		t.Fatal(err)
	}
	if err := npmcache.Write(d, s); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestSynthesizeMapsNpmSlicesToNpmSpecifiers(t *testing.T) {
	root := t.TempDir()
	writeNpmSlice(t, root, "third_party/debug", npmcache.Slice{Name: "debug", Version: "4.3.7", Specifier: "npm:debug@4.3.7"})
	writeNpmSlice(t, root, "third_party/scoped", npmcache.Slice{Name: "@codemirror/legacy-modes", Version: "6.5.1", Specifier: "npm:@codemirror/legacy-modes@6.5.1"})

	im, err := Synthesize("", nil, nil, root)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"debug":                     "npm:debug@4.3.7",
		"debug/":                    "npm:/debug@4.3.7/",
		"@codemirror/legacy-modes":  "npm:@codemirror/legacy-modes@6.5.1",
		"@codemirror/legacy-modes/": "npm:/@codemirror/legacy-modes@6.5.1/",
	}
	for k, v := range want {
		if im.Imports[k] != v {
			t.Errorf("imports[%q] = %q, want %q", k, im.Imports[k], v)
		}
	}
}

func TestSynthesizeMapsExplicitNpmSliceDirectoryAndMetadataFile(t *testing.T) {
	root := t.TempDir()
	dir := writeNpmSlice(t, root, "third_party/ms", npmcache.Slice{Name: "ms", Version: "2.1.3", Specifier: "npm:ms@2.1.3"})

	for name, dep := range map[string]string{"directory": dir, "metadata file": filepath.Join(dir, npmcache.MetadataFile)} {
		im := New()
		if _, err := im.LoadDependencies([]string{dep}, root); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if im.Imports["ms"] != "npm:ms@2.1.3" || im.Imports["ms/"] != "npm:/ms@2.1.3/" {
			t.Errorf("%s: imports = %v", name, im.Imports)
		}
	}
}

func TestSynthesizeReportsAnUnreadableNpmSlice(t *testing.T) {
	root := t.TempDir()
	d := filepath.Join(root, "third_party", "broken")
	if err := os.MkdirAll(d, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(d, npmcache.MetadataFile), []byte("{"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Synthesize("", nil, nil, root); err == nil {
		t.Error("expected an error for invalid ts_npm.json")
	}
}

func TestNpmSlicesLeaveFirstPartyAliasesAlone(t *testing.T) {
	root := t.TempDir()
	writeNpmSlice(t, root, "third_party/ms", npmcache.Slice{Name: "ms", Version: "2.1.3", Specifier: "npm:ms@2.1.3"})
	src := filepath.Join(root, "app", "app.ts")
	if err := os.MkdirAll(filepath.Dir(src), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("export {};"), 0644); err != nil {
		t.Fatal(err)
	}
	im, err := Synthesize("@app/app", []string{src}, nil, root)
	if err != nil {
		t.Fatal(err)
	}
	if im.Imports["@app/app"] == "" || im.Imports["ms"] != "npm:ms@2.1.3" {
		t.Errorf("imports = %v", im.Imports)
	}
}

func TestHasNpmSpecifiers(t *testing.T) {
	im := New()
	if im.HasNpmSpecifiers() {
		t.Error("an empty import map has no npm specifiers")
	}
	im.Imports["@app/lib"] = "./lib/lib.ts"
	if im.HasNpmSpecifiers() {
		t.Error("first-party aliases are not npm specifiers")
	}
	im.Imports["ms/"] = "npm:/ms@2.1.3/"
	if !im.HasNpmSpecifiers() {
		t.Error("an npm:/ prefix mapping is an npm specifier")
	}
}

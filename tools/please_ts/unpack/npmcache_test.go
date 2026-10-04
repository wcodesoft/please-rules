package unpack

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tools/please_ts/npmcache"
)

// makeNpmTarball writes a gzipped tarball laid out like an npm package ("package/" prefix).
func makeNpmTarball(t *testing.T, dir string, files map[string]string) string {
	t.Helper()
	path := filepath.Join(dir, "pkg.tgz")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for name, content := range files {
		if err := tw.WriteHeader(&tar.Header{Name: "package/" + name, Mode: 0644, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

func stageSlice(t *testing.T, root, name, version string) {
	t.Helper()
	dir := filepath.Join(root, "staged", name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := npmcache.Write(dir, npmcache.Slice{Name: name, Version: version, Specifier: npmcache.Specifier(name, version)}); err != nil {
		t.Fatal(err)
	}
}

func TestBuildNpmCacheLaysOutTheDenoCache(t *testing.T) {
	work := t.TempDir()
	chdir(t, work)
	tgz := makeNpmTarball(t, work, map[string]string{
		"package.json": `{"name":"ms","version":"2.1.3"}`,
		"index.js":     "module.exports = 1;",
		"lib/util.js":  "exports.x = 1;",
	})
	out := filepath.Join(work, "out")

	if err := Run(Options{Archive: tgz, Out: out, Name: "ms", NpmCache: true}); err != nil {
		t.Fatal(err)
	}

	for _, p := range []string{"index.js", "package.json", "lib/util.js"} {
		if _, err := os.Stat(filepath.Join(out, "npm/registry.npmjs.org/ms/2.1.3", p)); err != nil {
			t.Errorf("package file %s missing: %v", p, err)
		}
	}
	slice, err := npmcache.Read(out)
	if err != nil || slice.Specifier != "npm:ms@2.1.3" {
		t.Fatalf("slice = %+v, %v", slice, err)
	}

	data, err := os.ReadFile(filepath.Join(out, "npm/registry.npmjs.org/ms/registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	var packument struct {
		Name     string `json:"name"`
		Format   string `json:"_deno.packumentFormat"`
		DistTags map[string]string
		Versions map[string]struct {
			Dist struct {
				Integrity string `json:"integrity"`
				Tarball   string `json:"tarball"`
			} `json:"dist"`
		} `json:"versions"`
	}
	if err := json.Unmarshal(data, &packument); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(tgz)
	sum := sha512.Sum512(raw)
	wantIntegrity := "sha512-" + base64.StdEncoding.EncodeToString(sum[:])
	v := packument.Versions["2.1.3"]
	if packument.Name != "ms" || packument.Format != "full" || v.Dist.Integrity != wantIntegrity ||
		v.Dist.Tarball != "https://registry.npmjs.org/ms/-/ms-2.1.3.tgz" {
		t.Errorf("packument = %s", data)
	}
}

func TestBuildNpmCacheScopedPackageAndDependencyFields(t *testing.T) {
	work := t.TempDir()
	chdir(t, work)
	stageSlice(t, work, "lib", "1.0.0")
	tgz := makeNpmTarball(t, work, map[string]string{
		"package.json": `{"name":"@scope/pkg","version":"1.0.0","dependencies":{"lib":"1.0.0"},
			"peerDependencies":{"react":"^18"},"peerDependenciesMeta":{"react":{"optional":true}}}`,
		"index.js": "",
	})
	out := filepath.Join(work, "out")
	if err := Run(Options{Archive: tgz, Out: out, NpmCache: true}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "npm/registry.npmjs.org/@scope/pkg/registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"dependencies":{"lib":"1.0.0"}`, `"peerDependencies"`, `"peerDependenciesMeta"`, "pkg-1.0.0.tgz"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("registry.json lacks %s:\n%s", want, data)
		}
	}
}

func TestBuildNpmCacheDependencyClosure(t *testing.T) {
	depPkg := map[string]string{
		"package.json": `{"name":"debug","version":"4.3.7","dependencies":{"ms":"^2.1.3"}}`,
		"index.js":     "",
	}
	for name, tc := range map[string]struct {
		staged  map[string]string // name -> version
		wantErr string
	}{
		"missing":         {nil, `ms@^2.1.3`},
		"unsatisfied":     {map[string]string{"ms": "1.0.0"}, `ms@^2.1.3`},
		"different major": {map[string]string{"ms": "3.0.0"}, `ms@^2.1.3`},
		"satisfied":       {map[string]string{"ms": "2.1.3"}, ""},
		"newer minor":     {map[string]string{"ms": "2.4.0"}, ""},
	} {
		t.Run(name, func(t *testing.T) {
			work := t.TempDir()
			chdir(t, work)
			for n, v := range tc.staged {
				stageSlice(t, work, n, v)
			}
			tgz := makeNpmTarball(t, work, depPkg)
			err := Run(Options{Archive: tgz, Out: filepath.Join(work, "out"), Name: "debug", NpmCache: true})
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) || !strings.Contains(err.Error(), "ts_npm_module") {
				t.Fatalf("error = %v, want one naming %s and ts_npm_module", err, tc.wantErr)
			}
		})
	}
}

func TestBuildNpmCacheIgnoresItsOwnOutputInTheClosureCheck(t *testing.T) {
	work := t.TempDir()
	chdir(t, work)
	out := filepath.Join(work, "out")
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	// A stale slice for the dependency inside out must not count as a staged dependency.
	if err := npmcache.Write(out, npmcache.Slice{Name: "ms", Version: "2.1.3", Specifier: "npm:ms@2.1.3"}); err != nil {
		t.Fatal(err)
	}
	tgz := makeNpmTarball(t, work, map[string]string{
		"package.json": `{"name":"debug","version":"4.3.7","dependencies":{"ms":"^2.1.3"}}`,
	})
	if err := Run(Options{Archive: tgz, Out: out, Name: "debug", NpmCache: true}); err == nil {
		t.Error("the output directory must not satisfy the target's own dependencies")
	}
}

func TestBuildNpmCacheRejectsBadTarballs(t *testing.T) {
	work := t.TempDir()
	chdir(t, work)
	cases := map[string]struct {
		files map[string]string
		name  string
	}{
		"no package.json": {map[string]string{"index.js": ""}, ""},
		"no version":      {map[string]string{"package.json": `{"name":"x"}`}, ""},
		"wrong name":      {map[string]string{"package.json": `{"name":"x","version":"1.0.0"}`}, "y"},
		"invalid json":    {map[string]string{"package.json": `{`}, ""},
	}
	for n, c := range cases {
		tgz := makeNpmTarball(t, work, c.files)
		if err := Run(Options{Archive: tgz, Out: filepath.Join(work, "out-"+strings.ReplaceAll(n, " ", "-")), Name: c.name, NpmCache: true}); err == nil {
			t.Errorf("%s: expected an error", n)
		}
	}
}

// stageSliceWithPackage stages a dependency slice that holds an extracted package, as the
// output of a ts_npm_module does.
func stageSliceWithPackage(t *testing.T, root, name, version string) {
	t.Helper()
	stageSlice(t, root, name, version)
	pkgDir := filepath.Join(root, "staged", name, "npm", "registry.npmjs.org", name)
	if err := os.MkdirAll(filepath.Join(pkgDir, version), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, version, "index.js"), []byte("module.exports = 1;"), 0644); err != nil {
		t.Fatal(err)
	}
	packument := `{"name":"` + name + `","dist-tags":{"latest":"` + version + `"},"versions":{"` + version + `":{"name":"` + name + `","version":"` + version + `"}},"_deno.packumentFormat":"full"}`
	if err := os.WriteFile(filepath.Join(pkgDir, "registry.json"), []byte(packument), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildNpmCacheBundlesTheDependenciesIntoTheSlice(t *testing.T) {
	work := t.TempDir()
	chdir(t, work)
	stageSliceWithPackage(t, work, "ms", "2.1.3")
	tgz := makeNpmTarball(t, work, map[string]string{
		"package.json": `{"name":"debug","version":"4.3.7","dependencies":{"ms":"^2.1.3"}}`,
		"index.js":     "",
	})
	out := filepath.Join(work, "out")
	if err := Run(Options{Archive: tgz, Out: out, Name: "debug", NpmCache: true}); err != nil {
		t.Fatal(err)
	}

	for _, p := range []string{
		"npm/registry.npmjs.org/debug/4.3.7/index.js",
		"npm/registry.npmjs.org/ms/2.1.3/index.js", // the dependency, so a consumer only needs debug
		"npm/registry.npmjs.org/ms/registry.json",
	} {
		if _, err := os.Stat(filepath.Join(out, p)); err != nil {
			t.Errorf("slice lacks %s: %v", p, err)
		}
	}
	// The slice is still named after its own package: only debug is importable by name.
	slice, err := npmcache.Read(out)
	if err != nil || slice.Name != "debug" || slice.Version != "4.3.7" {
		t.Errorf("slice = %+v, %v", slice, err)
	}
}

func TestBuildNpmCacheSlicesWithDifferentVersionsOfADependencyStayResolvable(t *testing.T) {
	work := t.TempDir()
	chdir(t, work)
	stageSliceWithPackage(t, work, "ms", "2.1.3")
	// A second staged dependency that itself carries another version of ms.
	other := filepath.Join(work, "staged", "other")
	oldPkg := filepath.Join(other, "npm", "registry.npmjs.org", "ms")
	if err := os.MkdirAll(filepath.Join(oldPkg, "2.0.0"), 0755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(oldPkg, "2.0.0", "index.js"), []byte("x"), 0644)
	_ = os.WriteFile(filepath.Join(oldPkg, "registry.json"), []byte(`{"name":"ms","versions":{"2.0.0":{"name":"ms","version":"2.0.0"}}}`), 0644)
	if err := npmcache.Write(other, npmcache.Slice{Name: "other", Version: "1.0.0", Specifier: "npm:other@1.0.0"}); err != nil {
		t.Fatal(err)
	}

	tgz := makeNpmTarball(t, work, map[string]string{
		"package.json": `{"name":"top","version":"1.0.0","dependencies":{"ms":"^2.1.3","other":"1.0.0"}}`,
	})
	out := filepath.Join(work, "out")
	if err := Run(Options{Archive: tgz, Out: out, Name: "top", NpmCache: true}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "npm/registry.npmjs.org/ms/registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"2.0.0"`) || !strings.Contains(string(data), `"2.1.3"`) {
		t.Errorf("both ms versions must be in the merged packument: %s", data)
	}
}

func TestVersionSatisfies(t *testing.T) {
	for _, tc := range []struct {
		version, spec string
		want          bool
	}{
		{"2.1.3", "2.1.3", true},
		{"2.1.3", "^2.1.3", true},
		{"2.4.0", "^2.1.3", true},
		{"2.0.0", "^2.1.3", false},
		{"3.0.0", "^2.1.3", false},
		{"1.2.9", "~1.2.3", true},
		{"1.3.0", "~1.2.3", false},
		{"5.0.0", ">=2.0.0", true},
		{"1.0.0", ">=2.0.0", false},
		{"9.9.9", "*", true},
		{"9.9.9", "", true},
		// Ranges are interpreted completely, not approximated.
		{"2.5.0", "^1 || ^2", true},
		{"3.5.0", "^1 || ^2", false},
		{"2.0.0", ">=1 <3", true},
		{"3.0.0", ">=1 <3", false},
		{"1.9.0", "1.x", true},
		{"2.0.0", "1.x", false},
		{"1.2.3-beta.1", "^1.0.0", false},
		// Specifiers that are not version ranges cannot be judged and are accepted.
		{"9.9.9", "latest", true},
		{"9.9.9", "git+https://example.com/x.git", true},
	} {
		if got := versionSatisfies(tc.version, tc.spec); got != tc.want {
			t.Errorf("versionSatisfies(%q, %q) = %v, want %v", tc.version, tc.spec, got, tc.want)
		}
	}
}

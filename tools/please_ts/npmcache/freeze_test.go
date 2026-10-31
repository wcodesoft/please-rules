package npmcache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// packument is a registry.json of a package that lists these versions, each published at "t-<version>".
func packument(versions []string, tags map[string]string, extra map[string]any) map[string]any {
	vs := map[string]any{}
	times := map[string]any{"created": "2020", "modified": "2026"}
	for _, v := range versions {
		vs[v] = map[string]any{"version": v}
		times[v] = "t-" + v
	}
	doc := map[string]any{"name": "pkg", "versions": vs, "time": times, "dist-tags": tags}
	for k, v := range extra {
		doc[k] = v
	}
	return doc
}

// cacheWith writes a Deno cache with these packages: the versions each lists and the ones it has extracted.
func cacheWith(t *testing.T, packages map[string]struct {
	cached []string
	doc    map[string]any
}) string {
	t.Helper()
	root := t.TempDir()
	for name, p := range packages {
		dir := filepath.Join(root, filepath.FromSlash(registryDir), filepath.FromSlash(name))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(p.doc)
		if err := os.WriteFile(filepath.Join(dir, "registry.json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
		for _, v := range p.cached {
			if err := os.MkdirAll(filepath.Join(dir, v), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, v, "package.json"), []byte("{}"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	return root
}

type pkg = struct {
	cached []string
	doc    map[string]any
}

func read(t *testing.T, root, name string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(registryDir), filepath.FromSlash(name), "registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func keys(m any) []string {
	var out []string
	for k := range m.(map[string]any) {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestFreezeTakesTheVersionsNewerThanTheNewestCachedWithTheirTimesAndTags(t *testing.T) {
	root := cacheWith(t, map[string]pkg{"magic-string": {
		cached: []string{"1.4.4"},
		doc:    packument([]string{"1.4.3", "1.4.4", "1.4.5", "1.5.0"}, map[string]string{"latest": "1.5.0", "next": "1.5.0", "stable": "1.4.3"}, nil),
	}})

	res, err := Freeze(root)
	if err != nil {
		t.Fatal(err)
	}
	if res != (FreezeResult{Packages: 1, Versions: 2}) {
		t.Errorf("expected 1 package cut by 2 versions, got %+v", res)
	}
	doc := read(t, root, "magic-string")
	if got := keys(doc["versions"]); !reflect.DeepEqual(got, []string{"1.4.3", "1.4.4"}) {
		t.Errorf("expected the cached version and the older one to stay, got %v", got)
	}
	if got := keys(doc["time"]); !reflect.DeepEqual(got, []string{"1.4.3", "1.4.4", "created", "modified"}) {
		t.Errorf("expected the times of the versions that are gone to go, and created and modified to stay, got %v", got)
	}
	if got := doc["dist-tags"]; !reflect.DeepEqual(got, map[string]any{"latest": "1.4.4", "stable": "1.4.3"}) {
		t.Errorf("expected a tag to a version that is gone to go and latest to be the newest left, got %v", got)
	}
}

func TestFreezeKeepsWhatItDoesNotKnowAboutThePackage(t *testing.T) {
	root := cacheWith(t, map[string]pkg{"pkg": {
		cached: []string{"1.0.0"},
		doc:    packument([]string{"1.0.0", "2.0.0"}, map[string]string{"latest": "2.0.0"}, map[string]any{"description": "a package", "_deno.etag": "abc"}),
	}})
	if _, err := Freeze(root); err != nil {
		t.Fatal(err)
	}
	doc := read(t, root, "pkg")
	if doc["name"] != "pkg" || doc["description"] != "a package" || doc["_deno.etag"] != "abc" {
		t.Errorf("expected the other fields to stay, got %v", doc)
	}
}

func TestFreezeOrdersVersionsByTheirNumbersAndAPrereleaseBelowItsRelease(t *testing.T) {
	root := cacheWith(t, map[string]pkg{
		"numbers": {cached: []string{"1.9.0"}, doc: packument([]string{"1.2.0", "1.9.0", "1.10.0"}, map[string]string{"latest": "1.10.0"}, nil)},
		"pre":     {cached: []string{"1.4.4"}, doc: packument([]string{"1.4.4-rc.1", "1.4.4", "1.5.0-rc.1"}, map[string]string{"latest": "1.4.4"}, nil)},
	})
	if _, err := Freeze(root); err != nil {
		t.Fatal(err)
	}
	if got := keys(read(t, root, "numbers")["versions"]); !reflect.DeepEqual(got, []string{"1.2.0", "1.9.0"}) {
		t.Errorf("expected 1.10.0 to be above 1.9.0 and cut, got %v", got)
	}
	if got := keys(read(t, root, "pre")["versions"]); !reflect.DeepEqual(got, []string{"1.4.4", "1.4.4-rc.1"}) {
		t.Errorf("expected 1.4.4-rc.1 to stay (below 1.4.4) and 1.5.0-rc.1 to go, got %v", got)
	}
}

func TestFreezePointsLatestAtTheNewestReleaseNotAPrerelease(t *testing.T) {
	root := cacheWith(t, map[string]pkg{"pkg": {
		cached: []string{"1.1.0-beta.1"},
		doc:    packument([]string{"1.0.0", "1.1.0-beta.1", "1.1.0"}, map[string]string{"latest": "1.1.0"}, nil),
	}})
	if _, err := Freeze(root); err != nil {
		t.Fatal(err)
	}
	if got := read(t, root, "pkg")["dist-tags"].(map[string]any)["latest"]; got != "1.0.0" {
		t.Errorf("expected the 1.0.0 release and not the cached beta, got %v", got)
	}
}

func TestFreezeKeepsEveryVersionUpToTheNewestOfSeveralCachedMajors(t *testing.T) {
	root := cacheWith(t, map[string]pkg{"vite": {
		cached: []string{"6.4.5", "7.1.0"},
		doc:    packument([]string{"6.4.5", "7.0.0", "7.1.0", "8.0.0"}, map[string]string{"latest": "8.0.0"}, nil),
	}})
	if _, err := Freeze(root); err != nil {
		t.Fatal(err)
	}
	if got := keys(read(t, root, "vite")["versions"]); !reflect.DeepEqual(got, []string{"6.4.5", "7.0.0", "7.1.0"}) {
		t.Errorf("expected a range on 6.x still to find 6.4.5, got %v", got)
	}
}

func TestFreezeLeavesAPackageWithNothingCachedAndOneWhoseNewestIsCached(t *testing.T) {
	optional := packument([]string{"2.3.3", "2.3.4"}, map[string]string{"latest": "2.3.4"}, nil)
	current := packument([]string{"2.0.0", "3.0.0"}, map[string]string{"latest": "3.0.0"}, nil)
	root := cacheWith(t, map[string]pkg{
		"fsevents":   {doc: optional},
		"up-to-date": {cached: []string{"3.0.0"}, doc: current},
	})
	res, err := Freeze(root)
	if err != nil {
		t.Fatal(err)
	}
	if res != (FreezeResult{}) {
		t.Errorf("expected nothing to change, got %+v", res)
	}
	if got := keys(read(t, root, "fsevents")["versions"]); !reflect.DeepEqual(got, []string{"2.3.3", "2.3.4"}) {
		t.Errorf("expected an optional package that is not cached to keep its list, got %v", got)
	}
}

func TestFreezeDoesNotCountAHalfExtractedDirectoryAsACachedVersion(t *testing.T) {
	root := cacheWith(t, map[string]pkg{"postcss": {
		cached: []string{"8.5.31", "8.5.abc123.tmp"},
		doc:    packument([]string{"8.5.31", "8.5.32"}, map[string]string{"latest": "8.5.32"}, nil),
	}})
	if _, err := Freeze(root); err != nil {
		t.Fatal(err)
	}
	if got := keys(read(t, root, "postcss")["versions"]); !reflect.DeepEqual(got, []string{"8.5.31"}) {
		t.Errorf("expected the leftover temporary directory to be no version, got %v", got)
	}
}

func TestFreezeHandlesScopedPackagesAndAnEmptyOrMissingCache(t *testing.T) {
	root := cacheWith(t, map[string]pkg{"@babel/parser": {
		cached: []string{"7.29.11"},
		doc:    packument([]string{"7.29.11", "7.29.12", "8.0.8"}, map[string]string{"latest": "8.0.8", "v7": "7.29.11"}, nil),
	}})
	res, err := Freeze(root)
	if err != nil || res.Versions != 2 {
		t.Fatalf("expected a scoped package to be cut by 2, got %+v, %v", res, err)
	}
	if got := read(t, root, "@babel/parser")["dist-tags"]; !reflect.DeepEqual(got, map[string]any{"latest": "7.29.11", "v7": "7.29.11"}) {
		t.Errorf("expected latest to move to 7.29.11 and v7 to stay, got %v", got)
	}
	if res, err := Freeze(t.TempDir()); err != nil || res != (FreezeResult{}) {
		t.Errorf("expected a cache without npm packages to be left alone, got %+v, %v", res, err)
	}
	if res, err := Freeze(filepath.Join(t.TempDir(), "missing")); err != nil || res != (FreezeResult{}) {
		t.Errorf("expected a directory that is not there to be no error, got %+v, %v", res, err)
	}
}

func TestFreezeAgainChangesNothing(t *testing.T) {
	root := cacheWith(t, map[string]pkg{"pkg": {cached: []string{"1.0.0"}, doc: packument([]string{"1.0.0", "1.1.0"}, map[string]string{"latest": "1.1.0"}, nil)}})
	if _, err := Freeze(root); err != nil {
		t.Fatal(err)
	}
	res, err := Freeze(root)
	if err != nil || res != (FreezeResult{}) {
		t.Errorf("expected nothing to change the second time, got %+v, %v", res, err)
	}
}

func TestFreezeReportsAnUnreadableRegistryDocument(t *testing.T) {
	root := cacheWith(t, map[string]pkg{"pkg": {cached: []string{"1.0.0"}, doc: packument([]string{"1.0.0"}, nil, nil)}})
	file := filepath.Join(root, filepath.FromSlash(registryDir), "pkg", "registry.json")
	if err := os.WriteFile(file, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Freeze(root); err == nil {
		t.Error("expected an error that names the file")
	}
}

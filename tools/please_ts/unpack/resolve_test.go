package unpack

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"tools/please_ts/semver"
)

// mockVersion describes one published version.
type mockVersion struct {
	deps, peers, optional map[string]string
	published             string // RFC 3339, for as-of tests
}

// mockRegistry is an npm registry that serves real tarballs with integrity data.
type mockRegistry struct {
	t           *testing.T
	srv         *httptest.Server
	mu          sync.Mutex
	versions    map[string]map[string]mockVersion // package -> version -> description
	tarballs    map[string][]byte                 // "name@version" -> tarball
	hits        map[string]int                    // request path -> count
	corrupt     map[string]bool                   // "name@version": serve other bytes than the integrity covers
	noIntegrity map[string]bool                   // "name@version": publish no integrity data
	wrongName   map[string]string                 // "name@version": the package.json name to put in the tarball
}

func newMockRegistry(t *testing.T) *mockRegistry {
	m := &mockRegistry{
		t: t, versions: map[string]map[string]mockVersion{}, tarballs: map[string][]byte{},
		hits: map[string]int{}, corrupt: map[string]bool{}, noIntegrity: map[string]bool{}, wrongName: map[string]string{},
	}
	m.srv = httptest.NewServer(http.HandlerFunc(m.serve))
	t.Cleanup(m.srv.Close)
	return m
}

func (m *mockRegistry) URL() string { return m.srv.URL }

func (m *mockRegistry) add(name, version string, v mockVersion) {
	if m.versions[name] == nil {
		m.versions[name] = map[string]mockVersion{}
	}
	m.versions[name][version] = v
	manifestName := name
	if w, ok := m.wrongName[name+"@"+version]; ok {
		manifestName = w
	}
	manifest := map[string]any{"name": manifestName, "version": version, "main": "index.js"}
	for k, deps := range map[string]map[string]string{"dependencies": v.deps, "peerDependencies": v.peers, "optionalDependencies": v.optional} {
		if len(deps) > 0 {
			manifest[k] = deps
		}
	}
	pkgJSON, _ := json.Marshal(manifest)
	m.tarballs[name+"@"+version] = createTestTarball(m.t, map[string]string{
		"package.json": string(pkgJSON), "index.js": "module.exports = '" + name + "@" + version + "';",
	})
}

func (m *mockRegistry) hitsFor(substr string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for path, c := range m.hits {
		if strings.Contains(path, substr) {
			n += c
		}
	}
	return n
}

func (m *mockRegistry) serve(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	m.mu.Lock()
	m.hits[path]++
	m.mu.Unlock()

	if name, file, ok := strings.Cut(path, "/-/"); ok {
		base := name[strings.LastIndex(name, "/")+1:]
		version := strings.TrimSuffix(strings.TrimPrefix(file, base+"-"), ".tgz")
		data, ok := m.tarballs[name+"@"+version]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if m.corrupt[name+"@"+version] {
			data = append([]byte("corrupt"), data...)
		}
		_, _ = w.Write(data)
		return
	}

	versions, ok := m.versions[path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	all := make([]string, 0, len(versions))
	for v := range versions {
		all = append(all, v)
	}
	sort.Slice(all, func(i, j int) bool {
		a, _ := semver.Parse(all[i])
		b, _ := semver.Parse(all[j])
		return semver.Compare(a, b) < 0
	})
	entries := map[string]any{}
	times := map[string]string{}
	for _, v := range all {
		sum := sha512.Sum512(m.tarballs[path+"@"+v]) // integrity of the genuine tarball
		dist := map[string]string{"tarball": fmt.Sprintf("%s/%s/-/%s-%s.tgz", m.srv.URL, path, path[strings.LastIndex(path, "/")+1:], v)}
		if !m.noIntegrity[path+"@"+v] {
			dist["integrity"] = "sha512-" + base64.StdEncoding.EncodeToString(sum[:])
		}
		entries[v] = map[string]any{"version": v, "dist": dist}
		if versions[v].published != "" {
			times[v] = versions[v].published
		}
	}
	doc := map[string]any{"name": path, "dist-tags": map[string]string{"latest": all[len(all)-1]}, "versions": entries, "time": times}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(doc)
}

// rootTarball writes a root package's tarball and returns its path.
func rootTarball(t *testing.T, dir, name, version string, v mockVersion) string {
	t.Helper()
	manifest := map[string]any{"name": name, "version": version, "main": "index.js"}
	for k, deps := range map[string]map[string]string{"dependencies": v.deps, "peerDependencies": v.peers, "optionalDependencies": v.optional} {
		if len(deps) > 0 {
			manifest[k] = deps
		}
	}
	pkgJSON, _ := json.Marshal(manifest)
	path := filepath.Join(dir, "root.tgz")
	if err := os.WriteFile(path, createTestTarball(t, map[string]string{"package.json": string(pkgJSON), "index.js": ""}), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func deps(kv ...string) map[string]string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return m
}

func versionOf(t *testing.T, dir string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		t.Fatal(err)
	}
	var p struct{ Version string }
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	return p.Version
}

// ---- ts_module: the flat .deps layout ----

func runModule(t *testing.T, m *mockRegistry, root mockVersion) (string, error) {
	t.Helper()
	work := t.TempDir()
	out := filepath.Join(work, "out")
	err := Run(Options{
		Tarball: rootTarball(t, work, "root-pkg", "1.0.0", root), Out: out, Name: "root-pkg",
		ResolveTransitive: true, Registry: m.URL(),
	})
	return out, err
}

func TestModuleResolvesAChainOfDependencies(t *testing.T) {
	m := newMockRegistry(t)
	m.add("a", "1.2.0", mockVersion{deps: deps("b", "^2.0.0")})
	m.add("b", "2.1.0", mockVersion{})
	out, err := runModule(t, m, mockVersion{deps: deps("a", "^1.0.0")})
	if err != nil {
		t.Fatal(err)
	}
	if versionOf(t, filepath.Join(out, ".deps", "a")) != "1.2.0" || versionOf(t, filepath.Join(out, ".deps", "b")) != "2.1.0" {
		t.Error("the whole chain should be in .deps at the resolved versions")
	}
}

func TestModuleResolutionIsStrict(t *testing.T) {
	m := newMockRegistry(t)
	m.add("a", "1.0.0", mockVersion{})
	_, err := runModule(t, m, mockVersion{deps: deps("a", "^9.0.0")})
	if err == nil || !strings.Contains(err.Error(), "no version of a satisfies") {
		t.Errorf("a range nothing satisfies must fail, not fall back to another version; got %v", err)
	}
}

func TestModuleResolutionFailureIsAnErrorNotAWarning(t *testing.T) {
	m := newMockRegistry(t) // the registry knows no package at all
	_, err := runModule(t, m, mockVersion{deps: deps("missing", "^1.0.0")})
	if err == nil || !strings.Contains(err.Error(), "missing@^1.0.0") || !strings.Contains(err.Error(), "root-pkg") {
		t.Errorf("an unresolvable dependency must fail and name what needed it, got %v", err)
	}
}

func TestModuleVerifiesIntegrity(t *testing.T) {
	m := newMockRegistry(t)
	m.add("a", "1.0.0", mockVersion{})
	m.corrupt["a@1.0.0"] = true
	if _, err := runModule(t, m, mockVersion{deps: deps("a", "^1.0.0")}); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Errorf("a tarball that does not match the registry's integrity must be refused, got %v", err)
	}

	m2 := newMockRegistry(t)
	m2.add("a", "1.0.0", mockVersion{})
	m2.noIntegrity["a@1.0.0"] = true
	if _, err := runModule(t, m2, mockVersion{deps: deps("a", "^1.0.0")}); err == nil || !strings.Contains(err.Error(), "no integrity") {
		t.Errorf("a version without integrity data cannot be verified, got %v", err)
	}
}

func TestModuleRejectsATarballOfAnotherPackage(t *testing.T) {
	m := newMockRegistry(t)
	m.wrongName["a@1.0.0"] = "evil"
	m.add("a", "1.0.0", mockVersion{})
	if _, err := runModule(t, m, mockVersion{deps: deps("a", "^1.0.0")}); err == nil || !strings.Contains(err.Error(), `holds package "evil"`) {
		t.Errorf("got %v", err)
	}
}

func TestModuleReportsConflictingVersionsInsteadOfPickingTheFirst(t *testing.T) {
	m := newMockRegistry(t)
	m.add("a", "1.0.0", mockVersion{deps: deps("c", "^1.0.0")})
	m.add("b", "1.0.0", mockVersion{deps: deps("c", "^2.0.0")})
	m.add("c", "1.4.0", mockVersion{})
	m.add("c", "2.0.0", mockVersion{})
	_, err := runModule(t, m, mockVersion{deps: deps("a", "^1.0.0", "b", "^1.0.0")})
	if err == nil || !strings.Contains(err.Error(), "conflicting versions") || !strings.Contains(err.Error(), "ts_npm_module") {
		t.Errorf("incompatible needs for one package must be reported, got %v", err)
	}
}

func TestModuleSharesACompatibleDependency(t *testing.T) {
	m := newMockRegistry(t)
	m.add("a", "1.0.0", mockVersion{deps: deps("c", "^1.0.0")})
	m.add("b", "1.0.0", mockVersion{deps: deps("c", "^1.2.0")})
	m.add("c", "1.5.0", mockVersion{})
	out, err := runModule(t, m, mockVersion{deps: deps("a", "^1.0.0", "b", "^1.0.0")})
	if err != nil {
		t.Fatalf("two compatible ranges are not a conflict: %v", err)
	}
	if versionOf(t, filepath.Join(out, ".deps", "c")) != "1.5.0" {
		t.Error("c should be resolved once, to a version both accept")
	}
}

func TestModulePeerDependenciesAreBestEffort(t *testing.T) {
	m := newMockRegistry(t)
	m.add("a", "1.0.0", mockVersion{})
	out, err := runModule(t, m, mockVersion{deps: deps("a", "^1.0.0"), peers: deps("not-published", "^1.0.0")})
	if err != nil {
		t.Fatalf("a peer that cannot be resolved is a warning, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, ".deps", "a", "package.json")); err != nil {
		t.Error("the real dependency must still be resolved")
	}
}

// ---- ts_npm_module: slices ----

func runNpmCache(t *testing.T, m *mockRegistry, root mockVersion, resolve bool) (string, error) {
	t.Helper()
	work := t.TempDir()
	chdir(t, work)
	out := filepath.Join(work, "out")
	err := Run(Options{
		Archive: rootTarball(t, work, "root-pkg", "1.0.0", root), Out: out, Name: "root-pkg", NpmCache: true,
		ResolveTransitive: resolve, Registry: m.URL(),
	})
	return out, err
}

func cacheVersions(t *testing.T, out, name string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(out, "npm/registry.npmjs.org", name, "registry.json"))
	if err != nil {
		t.Fatalf("no packument for %s: %v", name, err)
	}
	var p struct {
		Versions map[string]json.RawMessage `json:"versions"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	var vs []string
	for v := range p.Versions {
		if _, err := os.Stat(filepath.Join(out, "npm/registry.npmjs.org", name, v, "index.js")); err != nil {
			t.Errorf("%s@%s is in the packument but not extracted: %v", name, v, err)
		}
		vs = append(vs, v)
	}
	sort.Strings(vs)
	return vs
}

func TestNpmCacheResolvesAndBundlesTransitiveDependencies(t *testing.T) {
	m := newMockRegistry(t)
	m.add("a", "1.2.0", mockVersion{deps: deps("b", "^2.0.0")})
	m.add("b", "2.1.0", mockVersion{})
	out, err := runNpmCache(t, m, mockVersion{deps: deps("a", "^1.0.0")}, true)
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"root-pkg": "1.0.0", "a": "1.2.0", "b": "2.1.0"} {
		if got := cacheVersions(t, out, name); len(got) != 1 || got[0] != want {
			t.Errorf("%s versions = %v, want [%s]", name, got, want)
		}
	}
}

func TestNpmCacheKeepsConflictingVersionsSideBySide(t *testing.T) {
	m := newMockRegistry(t)
	m.add("a", "1.0.0", mockVersion{deps: deps("c", "^1.0.0")})
	m.add("b", "1.0.0", mockVersion{deps: deps("c", "^2.0.0")})
	m.add("c", "1.4.0", mockVersion{})
	m.add("c", "2.0.0", mockVersion{})
	out, err := runNpmCache(t, m, mockVersion{deps: deps("a", "^1.0.0", "b", "^1.0.0")}, true)
	if err != nil {
		t.Fatalf("unlike ts_module, a slice can hold both: %v", err)
	}
	if got := cacheVersions(t, out, "c"); strings.Join(got, ",") != "1.4.0,2.0.0" {
		t.Errorf("c versions = %v, want 1.4.0 and 2.0.0 (Deno picks per dependent)", got)
	}
}

func TestNpmCacheReusesAVersionThatAlreadySatisfies(t *testing.T) {
	m := newMockRegistry(t)
	m.add("a", "1.0.0", mockVersion{deps: deps("c", "^1.0.0")})
	m.add("b", "1.0.0", mockVersion{deps: deps("c", "^1.2.0")})
	m.add("c", "1.2.0", mockVersion{})
	m.add("c", "1.5.0", mockVersion{})
	out, err := runNpmCache(t, m, mockVersion{deps: deps("a", "^1.0.0", "b", "^1.0.0")}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := cacheVersions(t, out, "c"); len(got) != 1 || got[0] != "1.5.0" {
		t.Errorf("c should be resolved once, got %v", got)
	}
	if n := m.hitsFor("c-1."); n != 1 {
		t.Errorf("c was downloaded %d times, want 1", n)
	}
}

func TestNpmCacheUsesStagedSlicesBeforeTheRegistry(t *testing.T) {
	m := newMockRegistry(t)
	m.add("b", "1.0.0", mockVersion{})
	work := t.TempDir()
	chdir(t, work)
	stageSliceWithPackage(t, work, "a", "1.0.0") // explicitly provided; the registry has no "a"
	out := filepath.Join(work, "out")
	err := Run(Options{
		Archive: rootTarball(t, work, "root-pkg", "1.0.0", mockVersion{deps: deps("a", "^1.0.0", "b", "^1.0.0")}),
		Out:     out, Name: "root-pkg", NpmCache: true, ResolveTransitive: true, Registry: m.URL(),
	})
	if err != nil {
		t.Fatalf("a declared dependency must not be resolved again: %v", err)
	}
	if m.hitsFor("a") != 0 {
		t.Error("the registry must not be asked for a package a slice provides")
	}
	if len(cacheVersions(t, out, "a")) != 1 || len(cacheVersions(t, out, "b")) != 1 {
		t.Error("both the staged and the resolved dependency must end up in the slice")
	}
}

func TestNpmCacheDoesNotResolveOptionalOrPeerDependencies(t *testing.T) {
	m := newMockRegistry(t) // knows nothing: any request would fail
	out, err := runNpmCache(t, m, mockVersion{optional: deps("fsevents", "^2.0.0"), peers: deps("react", "^18.0.0")}, true)
	if err != nil {
		t.Fatalf("optional and peer dependencies are not fetched: %v", err)
	}
	if got := cacheVersions(t, out, "root-pkg"); len(got) != 1 {
		t.Errorf("root-pkg = %v", got)
	}
	if n := m.hitsFor(""); n != 0 {
		t.Errorf("the registry was contacted %d times, want 0", n)
	}
}

func TestNpmCacheResolutionErrors(t *testing.T) {
	t.Run("no version satisfies", func(t *testing.T) {
		m := newMockRegistry(t)
		m.add("a", "1.0.0", mockVersion{})
		_, err := runNpmCache(t, m, mockVersion{deps: deps("a", "^9.0.0")}, true)
		if err == nil || !strings.Contains(err.Error(), "no version of a satisfies") || !strings.Contains(err.Error(), "root-pkg@1.0.0") {
			t.Errorf("got %v", err)
		}
	})
	t.Run("integrity mismatch", func(t *testing.T) {
		m := newMockRegistry(t)
		m.add("a", "1.0.0", mockVersion{})
		m.corrupt["a@1.0.0"] = true
		if _, err := runNpmCache(t, m, mockVersion{deps: deps("a", "^1.0.0")}, true); err == nil || !strings.Contains(err.Error(), "does not match") {
			t.Errorf("got %v", err)
		}
	})
	t.Run("a transitive dependency fails", func(t *testing.T) {
		m := newMockRegistry(t)
		m.add("a", "1.0.0", mockVersion{deps: deps("gone", "^1.0.0")})
		_, err := runNpmCache(t, m, mockVersion{deps: deps("a", "^1.0.0")}, true)
		if err == nil || !strings.Contains(err.Error(), "gone@^1.0.0") || !strings.Contains(err.Error(), "a@1.0.0") {
			t.Errorf("the error must name the package and who required it, got %v", err)
		}
	})
}

func TestNpmCacheWithoutResolveStillRequiresDeclaredDependencies(t *testing.T) {
	m := newMockRegistry(t)
	m.add("a", "1.0.0", mockVersion{})
	_, err := runNpmCache(t, m, mockVersion{deps: deps("a", "^1.0.0")}, false)
	if err == nil || !strings.Contains(err.Error(), "resolve_transitive") {
		t.Errorf("strict mode must fail and point at resolve_transitive, got %v", err)
	}
	if m.hitsFor("") != 0 {
		t.Error("strict mode must not contact the registry")
	}
}

// ---- the as-of date defaults to the day the root version was published ----

// publishedAround registers the root and a dependency whose versions straddle the root's
// publish day (2021-03-10).
func publishedAround(t *testing.T) *mockRegistry {
	m := newMockRegistry(t)
	m.add("root-pkg", "1.0.0", mockVersion{published: "2021-03-10T15:00:00.000Z"})
	m.add("a", "1.0.0", mockVersion{published: "2021-03-01T09:00:00.000Z"})
	m.add("a", "1.1.0", mockVersion{published: "2021-03-10T23:30:00.000Z"}) // later that day: still counts
	m.add("a", "1.2.0", mockVersion{published: "2021-03-11T00:10:00.000Z"}) // the next day: too new
	return m
}

func TestAutomaticPinUsesTheDayTheRootWasPublished(t *testing.T) {
	root := mockVersion{deps: deps("a", "^1.0.0")}

	m := publishedAround(t)
	out, err := runNpmCache(t, m, root, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := cacheVersions(t, out, "a"); len(got) != 1 || got[0] != "1.1.0" {
		t.Errorf("npm cache: a = %v, want 1.1.0 (the newest published by the end of the root's day)", got)
	}

	m = publishedAround(t)
	modOut, err := runModule(t, m, root)
	if err != nil {
		t.Fatal(err)
	}
	if got := versionOf(t, filepath.Join(modOut, ".deps", "a")); got != "1.1.0" {
		t.Errorf("ts_module: a = %s, want 1.1.0", got)
	}
}

func TestPinningIsSkippedWhenTheRegistryDoesNotKnowTheRoot(t *testing.T) {
	m := newMockRegistry(t) // root-pkg is not published here, say a private tarball
	m.add("a", "1.0.0", mockVersion{published: "2021-03-01T09:00:00.000Z"})
	m.add("a", "1.2.0", mockVersion{published: "2024-03-11T00:10:00.000Z"})
	out, err := runNpmCache(t, m, mockVersion{deps: deps("a", "^1.0.0")}, true)
	if err != nil {
		t.Fatalf("an unknown root must not fail the build: %v", err)
	}
	if got := cacheVersions(t, out, "a"); len(got) != 1 || got[0] != "1.2.0" {
		t.Errorf("a = %v, want the newest when there is no date to pin to", got)
	}
}

func TestPinningIsSkippedWhenTheRootHasNoPublishTime(t *testing.T) {
	m := newMockRegistry(t)
	m.add("root-pkg", "1.0.0", mockVersion{}) // no time recorded
	m.add("a", "1.0.0", mockVersion{published: "2021-03-01T09:00:00.000Z"})
	m.add("a", "1.2.0", mockVersion{published: "2024-03-11T00:10:00.000Z"})
	out, err := runNpmCache(t, m, mockVersion{deps: deps("a", "^1.0.0")}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := cacheVersions(t, out, "a"); len(got) != 1 || got[0] != "1.2.0" {
		t.Errorf("a = %v", got)
	}
}

func TestResolverPinsOnlyOnce(t *testing.T) {
	m := publishedAround(t)
	r := newResolver(m.URL())
	r.pinToRoot("root-pkg", "1.0.0")
	first := r.asOf
	if first.IsZero() || first.Format("2006-01-02T15:04:05") != "2021-03-10T23:59:59" {
		t.Fatalf("asOf = %v, want the end of 2021-03-10", first)
	}
	m.add("root-pkg", "2.0.0", mockVersion{published: "2024-01-01T00:00:00.000Z"})
	r.pinToRoot("root-pkg", "2.0.0")
	if !r.asOf.Equal(first) {
		t.Error("a resolver keeps the date it was pinned to")
	}
}

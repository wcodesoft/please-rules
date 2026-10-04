package unpack

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tools/please_ts/npmcache"
)

// buildNpmCache turns a hash-pinned npm tarball into a slice of a Deno npm cache, with no
// network access: the package is extracted into npm/registry.npmjs.org/<name>/<version>/
// and described by a minimal registry.json, so that Deno resolves npm:<name>@<version>
// from its cache. Deno does not verify the cache, so the integrity of the package is the
// sha256 Please checked on the tarball.
//
// Every dependency the package declares must be provided by another slice staged in the
// build directory (a ts_npm_module in deps); otherwise Deno would only notice at run time,
// by trying the network.
func buildNpmCache(opts Options) error {
	tmp, err := os.MkdirTemp("", "please_ts_npm_*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	if err := extractArchive(opts.ArchivePath(), tmp); err != nil {
		return fmt.Errorf("failed extracting %s: %w", opts.ArchivePath(), err)
	}
	pkg, raw, err := readPackageManifest(tmp)
	if err != nil {
		return err
	}
	if pkg.Name == "" || pkg.Version == "" {
		return fmt.Errorf("package.json has no name or version")
	}
	if opts.Name != "" && opts.Name != pkg.Name {
		return fmt.Errorf("tarball holds package %q, expected %q", pkg.Name, opts.Name)
	}

	staged, err := stagedSlices(opts.Out)
	if err != nil {
		return err
	}
	if err := checkDependencyClosure(pkg, staged); err != nil {
		return err
	}

	pkgDir := filepath.Join(opts.Out, "npm", "registry.npmjs.org", filepath.FromSlash(pkg.Name))
	versionDir := filepath.Join(pkgDir, pkg.Version)
	if err := os.MkdirAll(filepath.Dir(versionDir), 0755); err != nil {
		return err
	}
	if err := copyPath(tmp, versionDir); err != nil {
		return fmt.Errorf("failed staging the package: %w", err)
	}

	integrity, err := tarballIntegrity(opts.ArchivePath())
	if err != nil {
		return err
	}
	if err := writePackument(pkgDir, pkg, raw, integrity); err != nil {
		return err
	}

	// Bundle the dependencies' packages into this slice, so that a target only needs to list
	// this module, like ts_module: Please stages the direct dependencies of a target, not
	// the transitive ones. Each staged slice already contains its own dependencies.
	dirs := make([]string, 0, len(staged))
	for _, s := range staged {
		dirs = append(dirs, s.dir)
	}
	if err := npmcache.Merge(dirs, opts.Out); err != nil {
		return fmt.Errorf("failed bundling dependencies: %w", err)
	}

	return npmcache.Write(opts.Out, npmcache.Slice{
		Name:         pkg.Name,
		Version:      pkg.Version,
		Specifier:    npmcache.Specifier(pkg.Name, pkg.Version),
		Dependencies: pkg.Dependencies,
	})
}

// readPackageManifest reads package.json, returning both the typed fields and the raw
// object (to carry dependency fields into registry.json unchanged).
func readPackageManifest(dir string) (PackageJSON, map[string]json.RawMessage, error) {
	var pkg PackageJSON
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return pkg, nil, fmt.Errorf("package.json not found in the tarball: %w", err)
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return pkg, nil, fmt.Errorf("invalid package.json: %w", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return pkg, nil, err
	}
	return pkg, raw, nil
}

func tarballIntegrity(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha512.Sum512(data)
	return "sha512-" + base64.StdEncoding.EncodeToString(sum[:]), nil
}

// writePackument writes the registry.json Deno reads from its cache. The
// _deno.packumentFormat key tells Deno that the packument is complete; without it Deno
// tries to download the real one.
func writePackument(pkgDir string, pkg PackageJSON, raw map[string]json.RawMessage, integrity string) error {
	version := map[string]any{
		"name":    pkg.Name,
		"version": pkg.Version,
		"dist": map[string]string{
			"tarball":   fmt.Sprintf("https://registry.npmjs.org/%s/-/%s-%s.tgz", pkg.Name, tarballBase(pkg.Name), pkg.Version),
			"integrity": integrity,
		},
	}
	for _, key := range []string{"dependencies", "peerDependencies", "peerDependenciesMeta", "optionalDependencies", "bin"} {
		if v, ok := raw[key]; ok {
			version[key] = v
		}
	}
	packument := map[string]any{
		"name":                  pkg.Name,
		"dist-tags":             map[string]string{"latest": pkg.Version},
		"versions":              map[string]any{pkg.Version: version},
		"_deno.packumentFormat": "full",
	}
	data, err := json.Marshal(packument)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(pkgDir, "registry.json"), data, 0644)
}

func tarballBase(name string) string {
	if i := strings.LastIndex(name, "/"); i >= 0 {
		return name[i+1:]
	}
	return name
}

// stagedSlice is a slice found in the build directory.
type stagedSlice struct {
	dir string
	npmcache.Slice
}

// stagedSlices returns the slices staged under the working directory, which are the
// ts_npm_module targets in deps; outDir itself is skipped.
func stagedSlices(outDir string) ([]stagedSlice, error) {
	absOut, _ := filepath.Abs(outDir)
	var staged []stagedSlice
	for _, dir := range npmcache.Discover(".") {
		if abs, _ := filepath.Abs(dir); abs == absOut {
			continue
		}
		s, err := npmcache.Read(dir)
		if err != nil {
			return nil, err
		}
		staged = append(staged, stagedSlice{dir: dir, Slice: *s})
	}
	return staged, nil
}

// checkDependencyClosure verifies that each required dependency of pkg is provided by a
// staged slice. Optional dependencies and peer dependencies, which the consumer provides,
// are not required.
func checkDependencyClosure(pkg PackageJSON, staged []stagedSlice) error {
	if len(pkg.Dependencies) == 0 {
		return nil
	}
	versions := map[string][]string{}
	for _, s := range staged {
		versions[s.Name] = append(versions[s.Name], s.Version)
	}

	names := make([]string, 0, len(pkg.Dependencies))
	for name := range pkg.Dependencies {
		names = append(names, name)
	}
	sort.Strings(names)

	var missing []string
	for _, name := range names {
		rng := pkg.Dependencies[name]
		if !anySatisfies(versions[name], rng) {
			missing = append(missing, fmt.Sprintf("%s@%s", name, rng))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%s@%s depends on %s, which no ts_npm_module in deps provides; "+
			"add a ts_npm_module for each (with its tarball hash) to deps",
			pkg.Name, pkg.Version, strings.Join(missing, ", "))
	}
	return nil
}

func anySatisfies(versions []string, constraint string) bool {
	for _, v := range versions {
		if satisfies(v, constraint) {
			return true
		}
	}
	return false
}

// satisfies reports whether version meets an npm range. It understands exact versions,
// "*", "x"-less ranges with ^, ~ and >= prefixes; any other range form (||, spaces, <,
// hyphen ranges, x-ranges) is accepted when a version is staged at all, since the
// closure check only has to catch a dependency that was forgotten, not pick versions.
func satisfies(version, constraint string) bool {
	c := strings.TrimSpace(constraint)
	if c == "" || c == "*" || c == "latest" || version == c {
		return true
	}
	prefix := ""
	switch {
	case strings.HasPrefix(c, "^"), strings.HasPrefix(c, "~"):
		prefix, c = c[:1], c[1:]
	case strings.HasPrefix(c, ">="):
		prefix, c = ">=", strings.TrimSpace(c[2:])
	}
	if strings.ContainsAny(c, " |<>=xX-") || c == "" {
		return true
	}
	return matchesConstraint(parseSemver(version), parseSemver(c), prefix)
}

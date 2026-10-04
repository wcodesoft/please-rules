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

// buildNpmCache turns an npm tarball into a slice of a Deno npm cache: the package is
// extracted into npm/registry.npmjs.org/<name>/<version>/ and described by a minimal
// registry.json, so that Deno resolves npm:<name>@<version> from its cache. Deno does not
// verify the cache, so the integrity of the package is the sha256 Please checked on the
// tarball.
//
// The dependencies of the package come from other slices staged in the build directory (a
// ts_npm_module in deps), whose packages are bundled into this slice. With
// ResolveTransitive, dependencies that no staged slice provides are resolved from the
// registry instead, each verified against the registry's integrity data, and kept as
// separate versions when dependents need different ones. Without it every dependency must
// be provided, and the build fails naming those that are not, since Deno would only notice
// at run time, by trying the network.
func buildNpmCache(opts Options) error {
	data, err := os.ReadFile(opts.ArchivePath())
	if err != nil {
		return err
	}
	pkg, _, err := inspectTarball(data, opts.Name)
	if err != nil {
		return fmt.Errorf("%s: %w", opts.ArchivePath(), err)
	}

	staged, err := stagedSlices(opts.Out)
	if err != nil {
		return err
	}

	if opts.ResolveTransitive && len(pkg.Dependencies) > 0 {
		tmp, err := os.MkdirTemp("", "please_ts_resolved_*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp)
		resolved, err := resolveIntoSlices(pkg, staged, newResolver(opts.RegistryURL()), tmp)
		if err != nil {
			return fmt.Errorf("failed resolving the dependencies of %s: %w", pkg.Name, err)
		}
		staged = append(staged, resolved...)
	}

	if err := checkDependencyClosure(pkg, staged); err != nil {
		return err
	}
	if _, err := writeSlice(data, opts.Name, opts.Out); err != nil {
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

// inspectTarball extracts a tarball to a scratch directory and reads its package.json,
// returning the typed fields and the raw object (to carry dependency fields into
// registry.json unchanged). wantName, when set, must be the package the tarball holds.
func inspectTarball(data []byte, wantName string) (PackageJSON, map[string]json.RawMessage, error) {
	var pkg PackageJSON
	tmp, err := os.MkdirTemp("", "please_ts_npm_*")
	if err != nil {
		return pkg, nil, err
	}
	defer os.RemoveAll(tmp)
	if err := extractTarballBytes(data, tmp); err != nil {
		return pkg, nil, fmt.Errorf("failed extracting the tarball: %w", err)
	}
	manifest, err := os.ReadFile(filepath.Join(tmp, "package.json"))
	if err != nil {
		return pkg, nil, fmt.Errorf("package.json not found in the tarball: %w", err)
	}
	if err := json.Unmarshal(manifest, &pkg); err != nil {
		return pkg, nil, fmt.Errorf("invalid package.json: %w", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(manifest, &raw); err != nil {
		return pkg, nil, err
	}
	if pkg.Name == "" || pkg.Version == "" {
		return pkg, nil, fmt.Errorf("package.json has no name or version")
	}
	if wantName != "" && wantName != pkg.Name {
		return pkg, nil, fmt.Errorf("the tarball holds package %q, expected %q", pkg.Name, wantName)
	}
	return pkg, raw, nil
}

// writeSlice lays a tarball out as a slice in outDir: the package in the Deno cache layout,
// its registry.json and ts_npm.json.
func writeSlice(tarball []byte, wantName, outDir string) (PackageJSON, error) {
	tmp, err := os.MkdirTemp("", "please_ts_npm_*")
	if err != nil {
		return PackageJSON{}, err
	}
	defer os.RemoveAll(tmp)
	if err := extractTarballBytes(tarball, tmp); err != nil {
		return PackageJSON{}, fmt.Errorf("failed extracting the tarball: %w", err)
	}
	pkg, raw, err := inspectTarball(tarball, wantName)
	if err != nil {
		return pkg, err
	}

	pkgDir := filepath.Join(outDir, "npm", "registry.npmjs.org", filepath.FromSlash(pkg.Name))
	versionDir := filepath.Join(pkgDir, pkg.Version)
	if err := os.MkdirAll(filepath.Dir(versionDir), 0755); err != nil {
		return pkg, err
	}
	if err := copyPath(tmp, versionDir); err != nil {
		return pkg, fmt.Errorf("failed staging the package: %w", err)
	}
	sum := sha512.Sum512(tarball)
	integrity := "sha512-" + base64.StdEncoding.EncodeToString(sum[:])
	if err := writePackument(pkgDir, pkg, raw, integrity); err != nil {
		return pkg, err
	}
	return pkg, npmcache.Write(outDir, npmcache.Slice{
		Name:         pkg.Name,
		Version:      pkg.Version,
		Specifier:    npmcache.Specifier(pkg.Name, pkg.Version),
		Dependencies: pkg.Dependencies,
	})
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

// resolveIntoSlices resolves the dependencies of pkg that no staged slice provides, and
// theirs in turn, writing each package as a slice under tmpDir. Versions that are already
// available (staged, or resolved for another dependent) are reused when they satisfy the
// specifier; otherwise another version is resolved, so dependents with incompatible needs
// each get theirs. Dependencies come from the package.json inside the verified tarball, not
// from registry metadata. Optional and peer dependencies are not resolved: a peer is for the
// consumer to provide.
func resolveIntoSlices(pkg PackageJSON, staged []stagedSlice, r *resolver, tmpDir string) ([]stagedSlice, error) {
	r.pinToRoot(pkg.Name, pkg.Version)
	have := map[string][]string{} // package -> versions available
	for _, s := range staged {
		have[s.Name] = append(have[s.Name], s.Version)
	}

	var queue []pendingDep
	enqueue := func(deps map[string]string, requiredBy string) {
		for _, name := range sortedKeys(deps) {
			queue = append(queue, pendingDep{name: name, spec: deps[name], requiredBy: requiredBy})
		}
	}
	enqueue(pkg.Dependencies, pkg.Name+"@"+pkg.Version)

	var out []stagedSlice
	for len(queue) > 0 {
		d := queue[0]
		queue = queue[1:]
		if anySatisfies(have[d.name], d.spec) {
			continue
		}
		info, data, err := r.fetch(d.name, d.spec)
		if err != nil {
			return nil, fmt.Errorf("%s@%s (required by %s): %w", d.name, d.spec, d.requiredBy, err)
		}
		dir := filepath.Join(tmpDir, strings.ReplaceAll(d.name, "/", "+")+"@"+info.Version)
		resolved, err := writeSlice(data, d.name, dir)
		if err != nil {
			return nil, fmt.Errorf("%s@%s: %w", d.name, info.Version, err)
		}
		have[d.name] = append(have[d.name], resolved.Version)
		out = append(out, stagedSlice{dir: dir, Slice: npmcache.Slice{
			Name: resolved.Name, Version: resolved.Version,
			Specifier: npmcache.Specifier(resolved.Name, resolved.Version), Dependencies: resolved.Dependencies,
		}})
		enqueue(resolved.Dependencies, resolved.Name+"@"+resolved.Version)
	}
	return out, nil
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
			"add a ts_npm_module for each (with its tarball hash) to deps, or set resolve_transitive",
			pkg.Name, pkg.Version, strings.Join(missing, ", "))
	}
	return nil
}

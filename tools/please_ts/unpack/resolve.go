package unpack

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"tools/please_ts/registry"
	"tools/please_ts/semver"
)

// resolver resolves dependency specifiers against an npm registry and downloads the
// tarballs, verified against the integrity data the registry publishes.
//
// Resolution is reproducible: only versions published by the end of the day the root package
// version was published are considered (see pinToRoot), so a range such as ^2.1.3 resolves to
// the same version next year, and the version pinned by its hash is all that fixes the result.
type resolver struct {
	client     *registry.Client
	asOf       time.Time
	packuments map[string]*registry.Packument
}

func newResolver(registryURL string) *resolver {
	return &resolver{client: registry.New(registryURL), packuments: map[string]*registry.Packument{}}
}

// pinToRoot sets the as-of date to the end of the day (UTC) the root package version was
// published. The whole day counts so that a dependency released a few minutes after the
// package that needs it is still found. If the registry does not know the root version or
// its publish time (a tarball from another URL, a private registry without times),
// resolution is not pinned, with a warning.
func (r *resolver) pinToRoot(name, version string) {
	if !r.asOf.IsZero() {
		return
	}
	p, err := r.client.Packument(name, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: dependencies of %s@%s are not pinned to a date: %v\n", name, version, err)
		return
	}
	raw, ok := p.Time[version]
	if !ok {
		fmt.Fprintf(os.Stderr, "Warning: dependencies of %s@%s are not pinned to a date: the registry has no publish time for it\n", name, version)
		return
	}
	published, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: dependencies of %s@%s are not pinned to a date: bad publish time %q\n", name, version, raw)
		return
	}
	day := time.Date(published.Year(), published.Month(), published.Day(), 0, 0, 0, 0, time.UTC)
	r.asOf = day.Add(24*time.Hour - time.Nanosecond)
	r.packuments[name] = p // the full packument is reusable
}

// fetch resolves spec for a package to one version and downloads and verifies its tarball.
// The full packument (with publish times) is only fetched when there is an as-of date.
func (r *resolver) fetch(name, spec string) (registry.VersionInfo, []byte, error) {
	p, ok := r.packuments[name]
	if !ok {
		var err error
		if p, err = r.client.Packument(name, !r.asOf.IsZero()); err != nil {
			return registry.VersionInfo{}, nil, err
		}
		r.packuments[name] = p
	}
	info, err := registry.Resolve(p, spec, r.asOf)
	if err != nil {
		return registry.VersionInfo{}, nil, err
	}
	data, err := r.client.Download(name, info)
	if err != nil {
		return registry.VersionInfo{}, nil, err
	}
	return info, data, nil
}

// extractTarballBytes extracts a verified npm tarball into destDir, without the leading
// "package/" directory.
func extractTarballBytes(data []byte, destDir string) error {
	f, err := os.CreateTemp("", "please_ts_dep_*.tgz")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return extractTarGz(f.Name(), destDir)
}

// versionSatisfies reports whether a version meets a dependency specifier. Specifiers that
// are not version ranges (a dist-tag, say) cannot be judged here and count as satisfied.
func versionSatisfies(version, spec string) bool {
	v, err := semver.Parse(version)
	if err != nil {
		return true
	}
	r, err := semver.ParseRange(spec)
	if err != nil {
		return true
	}
	return r.Satisfies(v)
}

func anySatisfies(versions []string, spec string) bool {
	for _, v := range versions {
		if versionSatisfies(v, spec) {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// pendingDep is a dependency waiting to be resolved.
type pendingDep struct {
	name, spec, requiredBy string
	peer                   bool
}

// resolveTransitiveDependencies fetches the dependencies of a ts_module into .deps/<name>,
// one directory per package (the layout holds a single version of each).
//
// Resolution is strict: a specifier that no registry version satisfies is an error, and so
// is a download that does not match the registry's integrity data, a tarball that holds a
// different package than asked for, and two dependents that need versions of one package
// that cannot both be the one in .deps (ts_npm_module keeps both). Peer dependencies are
// fetched too, as a default for consumers that do not provide them, but only as a best
// effort: a peer that cannot be resolved or conflicts is a warning.
func resolveTransitiveDependencies(outDir string, rootPkg PackageJSON, r *resolver, depsMap map[string]string) error {
	r.pinToRoot(rootPkg.Name, rootPkg.Version)
	resolved := map[string]string{} // package -> version in .deps
	var queue []pendingDep
	for _, name := range sortedKeys(rootPkg.Dependencies) {
		queue = append(queue, pendingDep{name, rootPkg.Dependencies[name], rootPkg.Name, false})
	}
	for _, name := range sortedKeys(rootPkg.PeerDependencies) {
		queue = append(queue, pendingDep{name, rootPkg.PeerDependencies[name], rootPkg.Name, true})
	}

	for len(queue) > 0 {
		d := queue[0]
		queue = queue[1:]
		if d.name == rootPkg.Name {
			continue
		}

		if have, ok := resolved[d.name]; ok {
			if versionSatisfies(have, d.spec) {
				continue
			}
			msg := fmt.Sprintf("%s requires %s@%s, but %s@%s is already in .deps; a ts_module holds one version of a package (use ts_npm_module, which keeps several)",
				d.requiredBy, d.name, d.spec, d.name, have)
			if d.peer {
				fmt.Fprintf(os.Stderr, "Warning: %s\n", msg)
				continue
			}
			return fmt.Errorf("conflicting versions: %s", msg)
		}

		destDir := filepath.Join(outDir, ".deps", d.name)
		if _, err := os.Stat(filepath.Join(destDir, "package.json")); err == nil {
			existing := readPackageJSON(destDir)
			resolved[d.name] = existing.Version
			if entry := findLocalEntry(destDir); entry != "" {
				depsMap[d.name] = cleanRelativePath(filepath.Join(".deps", d.name, entry))
			}
			continue
		}

		info, data, err := r.fetch(d.name, d.spec)
		if err != nil {
			if d.peer {
				fmt.Fprintf(os.Stderr, "Warning: peer dependency %s@%s of %s was not resolved: %v\n", d.name, d.spec, d.requiredBy, err)
				continue
			}
			return fmt.Errorf("resolving %s@%s (required by %s): %w", d.name, d.spec, d.requiredBy, err)
		}
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}
		if err := extractTarballBytes(data, destDir); err != nil {
			return fmt.Errorf("extracting %s@%s: %w", d.name, info.Version, err)
		}
		pkg := readPackageJSON(destDir)
		if pkg.Name != "" && pkg.Name != d.name {
			return fmt.Errorf("the tarball of %s@%s holds package %q", d.name, info.Version, pkg.Name)
		}
		if pkg.Version == "" {
			pkg.Version = info.Version
		}
		ensureCommonJSType(destDir)
		resolved[d.name] = pkg.Version

		if entry := determineEntry(destDir, pkg); entry != "" {
			depsMap[d.name] = cleanRelativePath(filepath.Join(".deps", d.name, entry))
		}
		for _, name := range sortedKeys(pkg.Dependencies) {
			queue = append(queue, pendingDep{name, pkg.Dependencies[name], d.name + "@" + pkg.Version, false})
		}
	}
	return nil
}

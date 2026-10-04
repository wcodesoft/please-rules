// Package npmcache handles npm packages as slices of a Deno npm cache.
//
// A slice is the output of a ts_npm_module target: a directory holding the package
// extracted into Deno's cache layout (npm/registry.npmjs.org/<name>/<version>/ plus a
// registry.json), and a ts_npm.json describing it. Slices are built offline from a
// hash-pinned tarball; the runners merge the slices a target depends on into the
// per-run DENO_DIR, so Deno resolves npm: specifiers without network access, without a
// node_modules directory and without a lockfile.
package npmcache

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// MetadataFile is the name of the file that marks a directory as a slice.
const MetadataFile = "ts_npm.json"

// registryDir is the directory under the Deno cache that holds npm packages.
const registryDir = "npm/registry.npmjs.org"

// Slice describes one npm package of the cache.
type Slice struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Specifier    string            `json:"specifier"`
	Dependencies map[string]string `json:"dependencies,omitempty"`
}

// Specifier returns the npm: specifier of an exact package version.
func Specifier(name, version string) string {
	return "npm:" + name + "@" + version
}

// Read loads the slice described by dir/ts_npm.json.
func Read(dir string) (*Slice, error) {
	data, err := os.ReadFile(filepath.Join(dir, MetadataFile))
	if err != nil {
		return nil, err
	}
	var s Slice
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Join(dir, MetadataFile), err)
	}
	if s.Name == "" || s.Version == "" {
		return nil, fmt.Errorf("%s: name and version are required", filepath.Join(dir, MetadataFile))
	}
	return &s, nil
}

// Write stores the slice metadata in dir/ts_npm.json.
func Write(dir string, s Slice) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, MetadataFile), data, 0644)
}

// Discover returns the slice directories below root, sorted.
func Discover(root string) []string {
	var dirs []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == MetadataFile {
			dirs = append(dirs, filepath.Dir(path))
		}
		return nil
	})
	sort.Strings(dirs)
	return dirs
}

// Merge copies the npm cache content of every slice into denoDir (a DENO_DIR). Files
// that already exist are kept, except registry.json, whose versions are unioned so that
// two slices holding different versions of one package both stay resolvable.
func Merge(slices []string, denoDir string) error {
	for _, slice := range slices {
		src := filepath.Join(slice, "npm")
		if _, err := os.Stat(src); err != nil {
			continue
		}
		err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(slice, path)
			if err != nil {
				return err
			}
			dst := filepath.Join(denoDir, rel)
			if d.IsDir() {
				return os.MkdirAll(dst, 0755)
			}
			if _, err := os.Stat(dst); err == nil {
				if d.Name() == "registry.json" && strings.Contains(filepath.ToSlash(dst), registryDir) {
					return mergePackument(path, dst)
				}
				return nil
			}
			return copyFile(path, dst)
		})
		if err != nil {
			return fmt.Errorf("merging npm slice %s: %w", slice, err)
		}
	}
	return nil
}

// CachedVersion returns the version of a package that the Deno cache at denoDir holds, or ""
// when it holds none of it. An unversioned npm: specifier resolves to the registry's latest
// release, which is not necessarily the cached one (Deno then downloads it, into a cache that
// is meant to be read-only), so callers pin the cached version. More than one cached version is
// an error: there is no telling which one is meant.
func CachedVersion(denoDir, name string) (string, error) {
	entries, err := os.ReadDir(filepath.Join(denoDir, registryDir, filepath.FromSlash(name)))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var versions []string
	for _, e := range entries {
		if e.IsDir() {
			versions = append(versions, e.Name())
		}
	}
	switch len(versions) {
	case 0:
		return "", nil
	case 1:
		return versions[0], nil
	}
	sort.Strings(versions)
	return "", fmt.Errorf("%s holds several versions of %s (%s); cannot tell which one to use", denoDir, name, strings.Join(versions, ", "))
}

// Prepare merges the slices found below root into denoDir and reports whether there were
// any. Callers add --cached-only to the Deno command when it returns true, so a package
// missing from the cache fails at once instead of reaching for the network.
func Prepare(root, denoDir string) (bool, error) {
	slices := Discover(root)
	if len(slices) == 0 {
		return false, nil
	}
	if err := os.MkdirAll(denoDir, 0755); err != nil {
		return false, err
	}
	return true, Merge(slices, denoDir)
}

func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm()|0200)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// mergePackument adds the versions of the registry.json at src to the one at dst.
func mergePackument(src, dst string) error {
	var a, b map[string]json.RawMessage
	for path, target := range map[string]*map[string]json.RawMessage{dst: &a, src: &b} {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, target); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
	}
	var va, vb map[string]json.RawMessage
	if err := json.Unmarshal(a["versions"], &va); err != nil {
		return err
	}
	if err := json.Unmarshal(b["versions"], &vb); err != nil {
		return err
	}
	for v, raw := range vb {
		if _, ok := va[v]; !ok {
			va[v] = raw
		}
	}
	merged, err := json.Marshal(va)
	if err != nil {
		return err
	}
	a["versions"] = merged
	out, err := json.Marshal(a)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, out, 0644)
}

package npmcache

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"tools/please_ts/semver"
)

// FreezeResult says what Freeze changed.
type FreezeResult struct {
	// Packages is the number of registry.json files that were cut.
	Packages int
	// Versions is the number of versions that were taken out of them.
	Versions int
}

// Freeze makes the Deno cache at denoDir say only what it holds.
//
// `deno cache` chooses the versions to cache by a window of time that ends when it runs
// (it leaves out the ones published in the last day or so), and keeps the registry's whole
// list of versions of each package, in its registry.json, beside them. Used later, Deno
// chooses again with a window that ends later, sees newer versions in that list, and either
// fetches them into the cache while a build runs, which races with whatever copies the cache,
// or, with --cached-only, stops with `npm package not found in cache`. A cache then works
// only on the day it was built.
//
// Freeze takes from each registry.json the versions newer than the newest one cached, so
// that what Deno can choose is what is there, on any day. A package with nothing cached (an
// optional one that is not for this platform) is left as it is: Deno still has to read its
// list. Calling it again changes nothing.
func Freeze(denoDir string) (FreezeResult, error) {
	var result FreezeResult
	root := filepath.Join(denoDir, filepath.FromSlash(registryDir))
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return result, nil
	}
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Name() == "registry.json" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return result, err
	}
	sort.Strings(files)
	for _, file := range files {
		removed, err := freezePackument(file)
		if err != nil {
			return result, fmt.Errorf("freezing %s: %w", file, err)
		}
		if removed > 0 {
			result.Packages++
			result.Versions += removed
		}
	}
	return result, nil
}

// cachedVersions are the versions that are extracted next to a registry.json.
func cachedVersions(dir string) ([]semver.Version, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var versions []semver.Version
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// A directory that is not a version (a half-extracted `8.5.<hash>.tmp`) is not cached.
		if v, err := semver.Parse(e.Name()); err == nil {
			versions = append(versions, v)
		}
	}
	return versions, nil
}

func newest(versions []semver.Version) semver.Version {
	top := versions[0]
	for _, v := range versions[1:] {
		if semver.Compare(v, top) > 0 {
			top = v
		}
	}
	return top
}

// freezePackument cuts one registry.json, and says how many versions it took out of it.
func freezePackument(file string) (int, error) {
	cached, err := cachedVersions(filepath.Dir(file))
	if err != nil {
		return 0, err
	}
	if len(cached) == 0 {
		return 0, nil
	}
	top := newest(cached)

	data, err := os.ReadFile(file)
	if err != nil {
		return 0, err
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return 0, err
	}
	var versions map[string]json.RawMessage
	if raw, ok := doc["versions"]; !ok || json.Unmarshal(raw, &versions) != nil {
		return 0, nil
	}

	removed := 0
	for name := range versions {
		// A version that cannot be read is kept: there is no telling that it is newer.
		if v, err := semver.Parse(name); err == nil && semver.Compare(v, top) > 0 {
			delete(versions, name)
			removed++
		}
	}
	if removed == 0 {
		return 0, nil
	}
	if err := setField(doc, "versions", versions); err != nil {
		return 0, err
	}
	if err := dropTimes(doc, versions); err != nil {
		return 0, err
	}
	if err := fixTags(doc, versions); err != nil {
		return 0, err
	}
	out, err := json.Marshal(doc)
	if err != nil {
		return 0, err
	}
	return removed, os.WriteFile(file, out, 0o644)
}

func setField(doc map[string]json.RawMessage, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	doc[key] = raw
	return nil
}

// dropTimes takes the publication times of the versions that are gone; created and modified stay.
func dropTimes(doc map[string]json.RawMessage, kept map[string]json.RawMessage) error {
	raw, ok := doc["time"]
	if !ok {
		return nil
	}
	var times map[string]json.RawMessage
	if err := json.Unmarshal(raw, &times); err != nil {
		return nil
	}
	for name := range times {
		if _, isVersion := semver.Parse(name); isVersion != nil {
			continue
		}
		if _, ok := kept[name]; !ok {
			delete(times, name)
		}
	}
	return setField(doc, "time", times)
}

// fixTags takes the dist-tags that point to a version that is gone, and points latest at the
// newest release that is left when it was one of them.
func fixTags(doc map[string]json.RawMessage, kept map[string]json.RawMessage) error {
	var tags map[string]string
	if raw, ok := doc["dist-tags"]; ok {
		if err := json.Unmarshal(raw, &tags); err != nil {
			return nil
		}
	}
	if tags == nil {
		tags = map[string]string{}
	}
	for tag, version := range tags {
		if _, ok := kept[version]; !ok {
			delete(tags, tag)
		}
	}
	if _, ok := tags["latest"]; !ok {
		if latest, ok := newestRelease(kept); ok {
			tags["latest"] = latest
		}
	}
	return setField(doc, "dist-tags", tags)
}

// newestRelease is the newest of the versions that is not a prerelease, or the newest of all
// when every one is.
func newestRelease(versions map[string]json.RawMessage) (string, bool) {
	var releases, all []semver.Version
	for name := range versions {
		v, err := semver.Parse(name)
		if err != nil {
			continue
		}
		all = append(all, v)
		if len(v.Pre) == 0 {
			releases = append(releases, v)
		}
	}
	pool := releases
	if len(pool) == 0 {
		pool = all
	}
	if len(pool) == 0 {
		return "", false
	}
	return newest(pool).String(), true
}

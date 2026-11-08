package css

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tools/please_ts/importmap"
)

// IsCSSFile reports whether path has a .css extension.
func IsCSSFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".css")
}

// SplitSources separates a mixed list of source files into non-CSS (TS/JS) and CSS slices.
func SplitSources(srcs []string) (tsSrcs, cssSrcs []string) {
	for _, s := range srcs {
		if IsCSSFile(s) {
			cssSrcs = append(cssSrcs, s)
		} else {
			tsSrcs = append(tsSrcs, s)
		}
	}
	return tsSrcs, cssSrcs
}

// DiscoverGraphFiles returns all unique CSS files across the transitive dependency graph
// in deterministic topological order (dependencies before dependents), followed by directSrcs.
// Diamond dependencies (either reached via multiple paths or identical content at canonical module paths)
// are deduplicated automatically.
func DiscoverGraphFiles(deps []string, directSrcs []string, workingDir string) ([]string, error) {
	if workingDir == "" {
		workingDir = "."
	}

	mods, err := discoverAllModules(deps, workingDir)
	if err != nil {
		return nil, err
	}

	orderedMods := topoSortModules(mods)

	var result []string
	seenLogical := make(map[string]bool)
	seenContentKey := make(map[string]bool)

	addFile := func(logicalKey, path string) {
		cleanPath := filepath.Clean(path)
		if seenLogical[logicalKey] || seenLogical[cleanPath] {
			return
		}
		if _, err := os.Stat(cleanPath); err != nil {
			return
		}
		if hash, err := fileContentKey(filepath.Base(cleanPath), cleanPath); err == nil {
			if seenContentKey[hash] && logicalKey == cleanPath {
				return
			}
			seenContentKey[hash] = true
		}
		seenLogical[logicalKey] = true
		seenLogical[cleanPath] = true
		result = append(result, cleanPath)
	}

	for _, mod := range orderedMods {
		for _, f := range mod.cssFiles {
			logical := f
			if mod.meta.Name != "" {
				logical = mod.meta.Name + "/" + f
			}
			addFile(logical, filepath.Join(mod.baseDir, f))
		}
	}

	for _, src := range directSrcs {
		if IsCSSFile(src) {
			addFile(filepath.Clean(src), src)
		}
	}

	return result, nil
}

type discoveredModule struct {
	key      string
	meta     importmap.ModuleMetadata
	baseDir  string
	cssFiles []string
	depKeys  []string
}

func discoverAllModules(deps []string, workingDir string) (map[string]*discoveredModule, error) {
	metaFiles := collectMetadataPaths(deps, workingDir)
	sort.Strings(metaFiles)

	// Prefer outermost module directories so embedded copies inside another library's output
	// do not shadow the canonical module directory when both exist.
	sort.SliceStable(metaFiles, func(i, j int) bool {
		return strings.Count(metaFiles[i], string(filepath.Separator)) < strings.Count(metaFiles[j], string(filepath.Separator))
	})

	byKey := make(map[string]*discoveredModule)
	byDir := make(map[string]*discoveredModule)

	for _, metaPath := range metaFiles {
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta importmap.ModuleMetadata
		if err := json.Unmarshal(data, &meta); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", metaPath, err)
		}

		baseDir := filepath.Clean(filepath.Dir(metaPath))
		key := meta.Name
		if key == "" {
			key = baseDir
		}
		if _, exists := byKey[key]; exists {
			continue
		}

		cssFiles := extractModuleCSSFiles(meta, baseDir)
		mod := &discoveredModule{
			key:      key,
			meta:     meta,
			baseDir:  baseDir,
			cssFiles: cssFiles,
		}
		byKey[key] = mod
		byDir[baseDir] = mod
	}

	// Wire dependency edges from each module's recorded Imports map.
	for _, mod := range byKey {
		seenDep := make(map[string]bool)
		for spec, target := range mod.meta.Imports {
			depKey := resolveModuleDependencyKey(mod, spec, target, byKey, byDir)
			if depKey != "" && depKey != mod.key && !seenDep[depKey] {
				seenDep[depKey] = true
				mod.depKeys = append(mod.depKeys, depKey)
			}
		}
		sort.Strings(mod.depKeys)
	}

	return byKey, nil
}

func resolveModuleDependencyKey(
	mod *discoveredModule,
	spec, target string,
	byKey map[string]*discoveredModule,
	byDir map[string]*discoveredModule,
) string {
	cleanSpec := strings.TrimSuffix(spec, "/")
	if _, ok := byKey[cleanSpec]; ok {
		return cleanSpec
	}
	cleanTarget := filepath.Clean(strings.TrimPrefix(target, "./"))
	targetDir := filepath.Dir(cleanTarget)
	if depMod, ok := byDir[targetDir]; ok {
		return depMod.key
	}
	return ""
}

func extractModuleCSSFiles(meta importmap.ModuleMetadata, baseDir string) []string {
	seen := make(map[string]bool)
	var cssFiles []string

	for _, f := range meta.CSSFiles {
		if IsCSSFile(f) && !seen[f] {
			seen[f] = true
			cssFiles = append(cssFiles, f)
		}
	}
	for _, f := range meta.Files {
		if IsCSSFile(f) && !seen[f] {
			seen[f] = true
			cssFiles = append(cssFiles, f)
		}
	}
	if len(meta.Files) == 0 && len(meta.CSSFiles) == 0 {
		if entries, err := os.ReadDir(baseDir); err == nil {
			for _, e := range entries {
				if !e.IsDir() && IsCSSFile(e.Name()) && !seen[e.Name()] {
					seen[e.Name()] = true
					cssFiles = append(cssFiles, e.Name())
				}
			}
		}
	}
	return cssFiles
}

func collectMetadataPaths(deps []string, workingDir string) []string {
	seen := make(map[string]bool)
	var paths []string

	addPath := func(p string) {
		clean := filepath.Clean(p)
		if !seen[clean] {
			seen[clean] = true
			paths = append(paths, clean)
		}
	}

	for _, d := range deps {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if filepath.Base(d) == "ts_metadata.json" {
			addPath(d)
			continue
		}
		metaCandidate := filepath.Join(d, "ts_metadata.json")
		if _, err := os.Stat(metaCandidate); err == nil {
			addPath(metaCandidate)
		}
	}

	_ = filepath.WalkDir(workingDir, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == "ts_metadata.json" {
			addPath(path)
		}
		return nil
	})

	return paths
}

func topoSortModules(mods map[string]*discoveredModule) []*discoveredModule {
	keys := make([]string, 0, len(mods))
	for k := range mods {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	visited := make(map[string]bool, len(mods))
	var ordered []*discoveredModule

	var visit func(k string)
	visit = func(k string) {
		if visited[k] {
			return
		}
		visited[k] = true
		mod, ok := mods[k]
		if !ok {
			return
		}
		for _, depKey := range mod.depKeys {
			visit(depKey)
		}
		ordered = append(ordered, mod)
	}

	for _, k := range keys {
		visit(k)
	}
	return ordered
}

func fileContentKey(baseName, path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return baseName + ":" + hex.EncodeToString(sum[:]), nil
}

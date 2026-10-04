package importmap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tools/please_ts/npmcache"
)

// discoverDepPaths collects explicit dependencies and auto-discovers module metadata files in workingDir.
func discoverDepPaths(deps []string, workingDir string) []string {
	allDeps := make([]string, 0, len(deps))
	allDeps = append(allDeps, deps...)

	_ = filepath.Walk(workingDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			if info.Name() == "ts_metadata.json" || info.Name() == "ts_module.json" || info.Name() == npmcache.MetadataFile {
				allDeps = append(allDeps, path)
			}
		}
		return nil
	})

	return allDeps
}

// LoadDependencies processes all discovered dependency paths and returns loaded modules.
func (im *ImportMap) LoadDependencies(depPaths []string, workingDir string) ([]*loadedModule, error) {
	var loadedModules []*loadedModule
	seenDeps := make(map[string]bool)

	for _, dep := range depPaths {
		dep = strings.TrimSpace(dep)
		if dep == "" || seenDeps[dep] {
			continue
		}
		seenDeps[dep] = true

		mod, err := im.LoadDepItem(dep, workingDir)
		if err != nil {
			return nil, err
		}
		if mod != nil {
			loadedModules = append(loadedModules, mod)
		}
	}

	return loadedModules, nil
}

// LoadDepItem handles a single dependency path: metadata file, directory, or source file.
func (im *ImportMap) LoadDepItem(dep string, workingDir string) (*loadedModule, error) {
	if filepath.Base(dep) == npmcache.MetadataFile {
		return nil, im.registerNpmSlice(filepath.Dir(dep))
	}
	if isMetadataFile(dep) {
		mod, err := loadMetadataFile(dep, workingDir)
		if err != nil {
			return nil, fmt.Errorf("failed loading metadata from %s: %w", dep, err)
		}
		return mod, nil
	}

	info, err := os.Stat(dep)
	if err == nil && info.IsDir() {
		return im.LoadDirectoryDep(dep, workingDir)
	}

	if isSourceFile(dep) {
		im.MapDirectSourceFile(dep, workingDir)
	}

	return nil, nil
}

// LoadDirectoryDep checks for metadata inside a directory or maps it directly.
func (im *ImportMap) LoadDirectoryDep(dir, workingDir string) (*loadedModule, error) {
	if _, err := os.Stat(filepath.Join(dir, npmcache.MetadataFile)); err == nil {
		return nil, im.registerNpmSlice(dir)
	}

	metaPath := filepath.Join(dir, "ts_module.json")
	if _, err := os.Stat(metaPath); err == nil {
		return loadMetadataFile(metaPath, workingDir)
	}

	metaLibPath := filepath.Join(dir, "ts_metadata.json")
	if _, err := os.Stat(metaLibPath); err == nil {
		return loadMetadataFile(metaLibPath, workingDir)
	}

	im.MapDirectory(dir, workingDir)
	return nil, nil
}

// registerNpmSlice maps an npm package provided by a ts_npm_module to npm: specifiers that
// Deno resolves from its cache (see package npmcache): the bare name for the package
// entry, and name/ for subpaths, which Deno resolves with the package's own exports map.
func (im *ImportMap) registerNpmSlice(dir string) error {
	slice, err := npmcache.Read(dir)
	if err != nil {
		return fmt.Errorf("failed loading npm slice %s: %w", dir, err)
	}
	im.Imports[slice.Name] = slice.Specifier
	im.Imports[slice.Name+"/"] = "npm:/" + slice.Name + "@" + slice.Version + "/"
	return nil
}

// MapDirectSourceFile maps a single source file to imports.
func (im *ImportMap) MapDirectSourceFile(filePath, workingDir string) {
	relPath, err := relativeTo(workingDir, filePath)
	if err != nil {
		return
	}
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	nameWithoutExt := strings.TrimSuffix(base, ext)
	im.Imports[nameWithoutExt] = relPath
	im.Imports[base] = relPath
	im.Imports[filePath] = relPath
}

// MapDirectory maps all source files in a directory.
func (im *ImportMap) MapDirectory(dir string, workingDir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	dirBase := filepath.Base(dir)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if isSourceFile(name) {
			path := filepath.Join(dir, name)
			relPath, err := relativeTo(workingDir, path)
			if err == nil {
				im.Imports[dirBase+"/"+name] = relPath
				ext := filepath.Ext(name)
				im.Imports[dirBase+"/"+strings.TrimSuffix(name, ext)] = relPath
			}
		}
	}
}

// MapModuleFiles maps all declared and discovered files of a module.
func (im *ImportMap) MapModuleFiles(mod *loadedModule, workingDir string) {
	if mod.meta.Name == "" {
		return
	}

	for _, f := range mod.meta.Files {
		filePath := filepath.Join(mod.baseDir, f)
		if rel, err := relativeTo(workingDir, filePath); err == nil {
			ext := filepath.Ext(f)
			nameWithoutExt := strings.TrimSuffix(f, ext)
			im.Imports[mod.meta.Name+"/"+f] = rel
			im.Imports[mod.meta.Name+"/"+nameWithoutExt] = rel
			im.Imports["./"+f] = rel
			im.Imports[f] = rel
		}
	}

	if entries, err := os.ReadDir(mod.baseDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && isSourceFile(e.Name()) {
				filePath := filepath.Join(mod.baseDir, e.Name())
				if rel, err := relativeTo(workingDir, filePath); err == nil {
					name := e.Name()
					ext := filepath.Ext(name)
					nameWithoutExt := strings.TrimSuffix(name, ext)
					im.Imports[mod.meta.Name+"/"+name] = rel
					im.Imports[mod.meta.Name+"/"+nameWithoutExt] = rel
					im.Imports["./"+name] = rel
					im.Imports[name] = rel
				}
			}
		}
	}
}

func loadMetadataFile(metaFile, workingDir string) (*loadedModule, error) {
	data, err := os.ReadFile(metaFile)
	if err != nil {
		return nil, err
	}

	var meta ModuleMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	baseDir := filepath.Dir(metaFile)
	relBaseDir, err := relativeTo(workingDir, baseDir)
	if err != nil {
		relBaseDir = baseDir
	}

	relEntry := ""
	if meta.Entry != "" {
		entryPath := filepath.Join(baseDir, meta.Entry)
		if rel, err := relativeTo(workingDir, entryPath); err == nil {
			relEntry = rel
		}
	}

	mod := &loadedModule{
		meta:         meta,
		baseDir:      baseDir,
		relBaseDir:   relBaseDir,
		relEntry:     relEntry,
		internalDeps: make(map[string]string),
	}

	for k, v := range meta.Deps {
		depPath := v
		if !filepath.IsAbs(depPath) && !strings.HasPrefix(depPath, "http://") && !strings.HasPrefix(depPath, "https://") && !strings.HasPrefix(depPath, "npm:") {
			depPath = filepath.Join(baseDir, v)
		}
		if rel, err := relativeTo(workingDir, depPath); err == nil {
			mod.internalDeps[k] = rel
		} else {
			mod.internalDeps[k] = v
		}
	}

	depsDir := filepath.Join(baseDir, ".deps")
	if info, err := os.Stat(depsDir); err == nil && info.IsDir() {
		discoverInternalDeps(depsDir, workingDir, mod.internalDeps)
	}

	return mod, nil
}

func discoverInternalDeps(depsDir, workingDir string, result map[string]string) {
	entries, err := os.ReadDir(depsDir)
	if err != nil {
		return
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pkgName := e.Name()
		pkgDir := filepath.Join(depsDir, pkgName)

		if strings.HasPrefix(pkgName, "@") {
			subEntries, err := os.ReadDir(pkgDir)
			if err == nil {
				for _, sub := range subEntries {
					if sub.IsDir() {
						scopedName := pkgName + "/" + sub.Name()
						subDir := filepath.Join(pkgDir, sub.Name())
						entry := findPackageEntry(subDir)
						if entry != "" {
							if rel, err := relativeTo(workingDir, filepath.Join(subDir, entry)); err == nil {
								if _, exists := result[scopedName]; !exists {
									result[scopedName] = rel
								}
							}
						}
					}
				}
			}
			continue
		}

		entry := findPackageEntry(pkgDir)
		if entry != "" {
			if rel, err := relativeTo(workingDir, filepath.Join(pkgDir, entry)); err == nil {
				if _, exists := result[pkgName]; !exists {
					result[pkgName] = rel
				}
			}
		}
	}
}

func findPackageEntry(pkgDir string) string {
	pkgJSONPath := filepath.Join(pkgDir, "package.json")
	if data, err := os.ReadFile(pkgJSONPath); err == nil {
		var p struct {
			Main   string `json:"main"`
			Module string `json:"module"`
		}
		if json.Unmarshal(data, &p) == nil {
			if p.Module != "" {
				if _, err := os.Stat(filepath.Join(pkgDir, p.Module)); err == nil {
					return p.Module
				}
			}
			if p.Main != "" {
				if _, err := os.Stat(filepath.Join(pkgDir, p.Main)); err == nil {
					return p.Main
				}
				if _, err := os.Stat(filepath.Join(pkgDir, p.Main+".js")); err == nil {
					return p.Main + ".js"
				}
			}
		}
	}

	candidates := []string{
		"index.mjs", "index.js", "mod.ts", "index.ts",
		"dist/index.mjs", "dist/index.js",
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(pkgDir, c)); err == nil {
			return c
		}
	}
	return ""
}

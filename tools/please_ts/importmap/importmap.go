package importmap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ImportMap represents a standard W3C / Deno import map.
type ImportMap struct {
	Imports map[string]string            `json:"imports"`
	Scopes  map[string]map[string]string `json:"scopes,omitempty"`
}

// New creates an empty ImportMap.
func New() *ImportMap {
	return &ImportMap{
		Imports: make(map[string]string),
		Scopes:  make(map[string]map[string]string),
	}
}

// ModuleMetadata represents metadata emitted by ts_module or ts_library.
type ModuleMetadata struct {
	Name       string            `json:"name"`
	Version    string            `json:"version,omitempty"`
	Entry      string            `json:"entry"`
	Types      string            `json:"types,omitempty"`
	Imports    map[string]string `json:"imports,omitempty"`
	Files      []string          `json:"files,omitempty"`
	SourcePath string            `json:"source_path,omitempty"`
	Deps       map[string]string `json:"deps,omitempty"`
	PeerDeps   map[string]string `json:"peer_deps,omitempty"`
}

type loadedModule struct {
	meta         ModuleMetadata
	baseDir      string
	relBaseDir   string
	relEntry     string
	internalDeps map[string]string
}

// Synthesize generates an ImportMap for a target given its moduleName, srcs, and deps.
func Synthesize(moduleName string, srcs []string, deps []string, workingDir string) (*ImportMap, error) {
	im := New()

	if err := mapTargetSources(im, moduleName, srcs, workingDir); err != nil {
		return nil, err
	}

	depPaths := discoverDepPaths(deps, workingDir)
	loadedModules, err := loadDependencies(im, depPaths, workingDir)
	if err != nil {
		return nil, err
	}

	rootModules := registerRootModules(im, loadedModules, workingDir)
	hoistPeerDependencies(im, loadedModules, rootModules)
	populateModuleScopes(im, loadedModules, workingDir)

	return im, nil
}

// mapTargetSources maps the target's own module_name and source file subpaths to im.Imports.
func mapTargetSources(im *ImportMap, moduleName string, srcs []string, workingDir string) error {
	if moduleName == "" || len(srcs) == 0 {
		return nil
	}

	entry := findTargetEntry(moduleName, srcs)
	relEntry, err := relativeTo(workingDir, entry)
	if err == nil {
		im.Imports[moduleName] = relEntry
		if !strings.HasSuffix(moduleName, "/") {
			im.Imports[moduleName+"/"] = dirFromPath(relEntry)
		}
	}

	for _, src := range srcs {
		relPath, err := relativeTo(workingDir, src)
		if err != nil {
			continue
		}
		base := filepath.Base(src)
		ext := filepath.Ext(base)
		nameWithoutExt := strings.TrimSuffix(base, ext)

		im.Imports[moduleName+"/"+base] = relPath
		im.Imports[moduleName+"/"+nameWithoutExt] = relPath

		// Map relative subpath if source is in a subfolder relative to target entry
		entryDir := filepath.Dir(entry)
		if relToEntry, err := filepath.Rel(entryDir, src); err == nil && !strings.HasPrefix(relToEntry, "..") {
			subClean := strings.TrimSuffix(relToEntry, filepath.Ext(relToEntry))
			if subClean != base && subClean != nameWithoutExt {
				im.Imports[moduleName+"/"+relToEntry] = relPath
				im.Imports[moduleName+"/"+subClean] = relPath
			}
		}
	}

	return nil
}

// discoverDepPaths collects explicit dependencies and auto-discovers module metadata files in workingDir.
func discoverDepPaths(deps []string, workingDir string) []string {
	allDeps := make([]string, 0, len(deps))
	allDeps = append(allDeps, deps...)

	_ = filepath.Walk(workingDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			if info.Name() == "ts_metadata.json" || info.Name() == "ts_module.json" {
				allDeps = append(allDeps, path)
			}
		}
		return nil
	})

	return allDeps
}

// loadDependencies processes all discovered dependency paths and returns loaded modules.
func loadDependencies(im *ImportMap, depPaths []string, workingDir string) ([]*loadedModule, error) {
	var loadedModules []*loadedModule
	seenDeps := make(map[string]bool)

	for _, dep := range depPaths {
		dep = strings.TrimSpace(dep)
		if dep == "" || seenDeps[dep] {
			continue
		}
		seenDeps[dep] = true

		mod, err := loadDepItem(im, dep, workingDir)
		if err != nil {
			return nil, err
		}
		if mod != nil {
			loadedModules = append(loadedModules, mod)
		}
	}

	return loadedModules, nil
}

// loadDepItem handles a single dependency path: metadata file, directory, or source file.
func loadDepItem(im *ImportMap, dep string, workingDir string) (*loadedModule, error) {
	if isMetadataFile(dep) {
		mod, err := loadMetadataFile(dep, workingDir)
		if err != nil {
			return nil, fmt.Errorf("failed loading metadata from %s: %w", dep, err)
		}
		return mod, nil
	}

	info, err := os.Stat(dep)
	if err == nil && info.IsDir() {
		return loadDirectoryDep(im, dep, workingDir)
	}

	if isSourceFile(dep) {
		mapDirectSourceFile(im, dep, workingDir)
	}

	return nil, nil
}

func isMetadataFile(path string) bool {
	return strings.HasSuffix(path, "ts_module.json") || strings.HasSuffix(path, "ts_metadata.json")
}

func loadDirectoryDep(im *ImportMap, dir, workingDir string) (*loadedModule, error) {
	metaPath := filepath.Join(dir, "ts_module.json")
	if _, err := os.Stat(metaPath); err == nil {
		return loadMetadataFile(metaPath, workingDir)
	}

	metaLibPath := filepath.Join(dir, "ts_metadata.json")
	if _, err := os.Stat(metaLibPath); err == nil {
		return loadMetadataFile(metaLibPath, workingDir)
	}

	mapDirectory(im, dir, workingDir)
	return nil, nil
}

func mapDirectSourceFile(im *ImportMap, filePath, workingDir string) {
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

// registerRootModules registers top-level imports for all loaded root modules.
func registerRootModules(im *ImportMap, loadedModules []*loadedModule, workingDir string) map[string]*loadedModule {
	rootModules := make(map[string]*loadedModule)
	for _, mod := range loadedModules {
		if mod.meta.Name == "" {
			continue
		}
		rootModules[mod.meta.Name] = mod
		if mod.relEntry != "" {
			im.Imports[mod.meta.Name] = mod.relEntry
			im.Imports[mod.meta.Entry] = mod.relEntry
			im.Imports["./"+mod.meta.Entry] = mod.relEntry
			if !strings.HasSuffix(mod.meta.Name, "/") {
				im.Imports[mod.meta.Name+"/"] = dirFromPath(mod.relEntry)
			}
		}
		mapModuleFiles(im, mod, workingDir)
	}
	return rootModules
}

// hoistPeerDependencies hoists undeclared peer dependencies across modules to the root imports map.
func hoistPeerDependencies(im *ImportMap, loadedModules []*loadedModule, rootModules map[string]*loadedModule) {
	depCanonical := make(map[string]string)
	isPeer := make(map[string]bool)

	for _, mod := range loadedModules {
		for peer := range mod.meta.PeerDeps {
			isPeer[peer] = true
		}
		for depName, depRel := range mod.internalDeps {
			if _, ok := depCanonical[depName]; !ok {
				depCanonical[depName] = depRel
			}
		}
	}

	for depName, canonicalRel := range depCanonical {
		if _, inRoot := rootModules[depName]; inRoot {
			continue
		}
		if _, inImports := im.Imports[depName]; inImports {
			continue
		}
		if isPeer[depName] {
			im.Imports[depName] = canonicalRel
			im.Imports[depName+"/"] = dirFromPath(canonicalRel)
		}
	}
}

// populateModuleScopes sets up W3C Import Map scopes for each module.
func populateModuleScopes(im *ImportMap, loadedModules []*loadedModule, workingDir string) {
	for _, mod := range loadedModules {
		if mod.relBaseDir == "" {
			continue
		}
		populateSingleModuleScope(im, mod, workingDir)
	}
}

// populateSingleModuleScope configures the scope block for one module.
func populateSingleModuleScope(im *ImportMap, mod *loadedModule, workingDir string) {
	scopeKey := ensureTrailingSlash(mod.relBaseDir)

	if len(mod.internalDeps) > 0 {
		if im.Scopes[scopeKey] == nil {
			im.Scopes[scopeKey] = make(map[string]string)
		}

		for depName, depRel := range mod.internalDeps {
			// Explicit Target Precedence & Canonical Hoisting Rule:
			// If depName exists in im.Imports (explicitly declared or hoisted singleton),
			// route internal imports of this dependency to the canonical import!
			if rootRel, ok := im.Imports[depName]; ok {
				im.Scopes[scopeKey][depName] = rootRel
				if rootDir, ok := im.Imports[depName+"/"]; ok {
					im.Scopes[scopeKey][depName+"/"] = rootDir
				}
			} else {
				// Isolated private helper
				im.Scopes[scopeKey][depName] = depRel
				im.Scopes[scopeKey][depName+"/"] = dirFromPath(depRel)
			}
		}
	}

	// Backward compatibility: map meta.Imports
	for k, v := range mod.meta.Imports {
		if mod.meta.Name != "" && (k == mod.meta.Name || strings.HasPrefix(k, mod.meta.Name+"/")) {
			continue
		}
		if k == mod.meta.Entry || k == "./"+mod.meta.Entry {
			continue
		}
		resolved := resolveImportPath(v, mod.baseDir, workingDir)
		if strings.HasSuffix(k, "/") && !strings.HasSuffix(resolved, "/") {
			resolved += "/"
		}
		if strings.HasPrefix(v, "./") && len(mod.internalDeps) > 0 {
			if im.Scopes[scopeKey] == nil {
				im.Scopes[scopeKey] = make(map[string]string)
			}
			im.Scopes[scopeKey][k] = resolved
		} else {
			if _, ok := im.Imports[k]; !ok {
				im.Imports[k] = resolved
			}
		}
	}
}

// WriteToFile serializes the ImportMap to the specified path.
func (im *ImportMap) WriteToFile(filePath string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(im, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
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

	// 1. Process meta.Deps
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

	// 2. Discover .deps directory if present
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

func mapModuleFiles(im *ImportMap, mod *loadedModule, workingDir string) {
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

func resolveImportPath(v, baseDir, workingDir string) string {
	if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") || strings.HasPrefix(v, "jsr:") || strings.HasPrefix(v, "npm:") {
		return v
	}
	cleanV := strings.TrimPrefix(v, "./")
	if _, err := os.Stat(cleanV); err == nil {
		if rel, err := relativeTo(workingDir, cleanV); err == nil {
			return rel
		}
	} else {
		fullPath := filepath.Join(baseDir, v)
		if rel, err := relativeTo(workingDir, fullPath); err == nil {
			return rel
		}
	}
	return v
}

func findTargetEntry(moduleName string, srcs []string) string {
	cleanName := filepath.Base(moduleName)
	if strings.HasPrefix(cleanName, "@") {
		cleanName = strings.TrimPrefix(cleanName, "@")
	}

	for _, src := range srcs {
		base := filepath.Base(src)
		if base == cleanName+".ts" || base == cleanName+".tsx" || base == cleanName+".js" {
			return src
		}
	}
	for _, src := range srcs {
		base := filepath.Base(src)
		if base == "index.ts" || base == "index.tsx" || base == "mod.ts" || base == "index.js" {
			return src
		}
	}
	return srcs[0]
}

func mapDirectory(im *ImportMap, dir string, workingDir string) {
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

func isSourceFile(path string) bool {
	return strings.HasSuffix(path, ".ts") ||
		strings.HasSuffix(path, ".tsx") ||
		strings.HasSuffix(path, ".js") ||
		strings.HasSuffix(path, ".jsx") ||
		strings.HasSuffix(path, ".mjs")
}

func dirFromPath(filePath string) string {
	dir := filepath.Dir(filePath)
	return ensureTrailingSlash(dir)
}

func ensureTrailingSlash(dir string) string {
	if !strings.HasPrefix(dir, ".") && !strings.HasPrefix(dir, "/") {
		dir = "./" + dir
	}
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	return dir
}

func relativeTo(base, target string) (string, error) {
	if base == "" {
		base = "."
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		absBase = base
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		absTarget = target
	}
	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return target, err
	}
	if !strings.HasPrefix(rel, ".") {
		rel = "./" + rel
	}
	return rel, nil
}

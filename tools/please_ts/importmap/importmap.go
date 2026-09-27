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

	// 1. Map target's own module_name if specified
	if moduleName != "" && len(srcs) > 0 {
		entry := findTargetEntry(moduleName, srcs)
		relEntry, err := relativeTo(workingDir, entry)
		if err == nil {
			im.Imports[moduleName] = relEntry
			if !strings.HasSuffix(moduleName, "/") {
				dir := filepath.Dir(relEntry)
				if !strings.HasPrefix(dir, ".") && !strings.HasPrefix(dir, "/") {
					dir = "./" + dir
				}
				if !strings.HasSuffix(dir, "/") {
					dir += "/"
				}
				im.Imports[moduleName+"/"] = dir
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

			// Also map relative subpath if source is in a subfolder relative to target entry
			entryDir := filepath.Dir(entry)
			if relToEntry, err := filepath.Rel(entryDir, src); err == nil && !strings.HasPrefix(relToEntry, "..") {
				subClean := strings.TrimSuffix(relToEntry, filepath.Ext(relToEntry))
				if subClean != base && subClean != nameWithoutExt {
					im.Imports[moduleName+"/"+relToEntry] = relPath
					im.Imports[moduleName+"/"+subClean] = relPath
				}
			}
		}
	}

	// 2. Discover all dependencies
	allDeps := make([]string, 0, len(deps))
	allDeps = append(allDeps, deps...)

	// Auto-discover dependency metadata in the sandbox
	_ = filepath.Walk(workingDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			if info.Name() == "ts_metadata.json" || info.Name() == "ts_module.json" {
				allDeps = append(allDeps, path)
			}
		}
		return nil
	})

	var loadedModules []*loadedModule
	seenDeps := make(map[string]bool)

	for _, dep := range allDeps {
		dep = strings.TrimSpace(dep)
		if dep == "" || seenDeps[dep] {
			continue
		}
		seenDeps[dep] = true

		// Check if dep points to a module metadata JSON file
		if strings.HasSuffix(dep, "ts_module.json") || strings.HasSuffix(dep, "ts_metadata.json") {
			mod, err := loadMetadataFile(dep, workingDir)
			if err != nil {
				return nil, fmt.Errorf("failed loading metadata from %s: %w", dep, err)
			}
			if mod != nil {
				loadedModules = append(loadedModules, mod)
			}
			continue
		}

		// Check if dep is a directory containing ts_module.json or ts_metadata.json
		info, err := os.Stat(dep)
		if err == nil && info.IsDir() {
			metaPath := filepath.Join(dep, "ts_module.json")
			if _, err := os.Stat(metaPath); err == nil {
				mod, err := loadMetadataFile(metaPath, workingDir)
				if err != nil {
					return nil, fmt.Errorf("failed loading %s: %w", metaPath, err)
				}
				if mod != nil {
					loadedModules = append(loadedModules, mod)
				}
				continue
			}

			metaLibPath := filepath.Join(dep, "ts_metadata.json")
			if _, err := os.Stat(metaLibPath); err == nil {
				mod, err := loadMetadataFile(metaLibPath, workingDir)
				if err != nil {
					return nil, fmt.Errorf("failed loading %s: %w", metaLibPath, err)
				}
				if mod != nil {
					loadedModules = append(loadedModules, mod)
				}
				continue
			}

			// Directory with source files: find potential entry or map all .ts files
			mapDirectory(im, dep, workingDir)
			continue
		}

		// If dep is a direct source file (.ts, .tsx, .js, .mjs)
		if isSourceFile(dep) {
			relPath, err := relativeTo(workingDir, dep)
			if err == nil {
				base := filepath.Base(dep)
				ext := filepath.Ext(base)
				nameWithoutExt := strings.TrimSuffix(base, ext)
				im.Imports[nameWithoutExt] = relPath
				im.Imports[base] = relPath
				im.Imports[dep] = relPath
			}
		}
	}

	// 3. Collect root modules and register top-level imports
	rootModules := make(map[string]*loadedModule)
	for _, mod := range loadedModules {
		if mod.meta.Name != "" {
			rootModules[mod.meta.Name] = mod
			if mod.relEntry != "" {
				im.Imports[mod.meta.Name] = mod.relEntry
				im.Imports[mod.meta.Entry] = mod.relEntry
				im.Imports["./"+mod.meta.Entry] = mod.relEntry
				if !strings.HasSuffix(mod.meta.Name, "/") {
					dir := filepath.Dir(mod.relEntry)
					if !strings.HasPrefix(dir, ".") && !strings.HasPrefix(dir, "/") {
						dir = "./" + dir
					}
					if !strings.HasSuffix(dir, "/") {
						dir += "/"
					}
					im.Imports[mod.meta.Name+"/"] = dir
				}
			}
			mapModuleFiles(im, mod, workingDir)
		}
	}

	// 4. Detect shared singletons / peer dependencies across modules to hoist to root imports
	depCounts := make(map[string]int)
	depCanonical := make(map[string]string)
	isPeer := make(map[string]bool)

	for _, mod := range loadedModules {
		for peer := range mod.meta.PeerDeps {
			isPeer[peer] = true
		}
		for depName, depRel := range mod.internalDeps {
			depCounts[depName]++
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
		// Only hoist if explicitly marked as a peer dependency
		if isPeer[depName] {
			im.Imports[depName] = canonicalRel
			dir := filepath.Dir(canonicalRel)
			if !strings.HasPrefix(dir, ".") && !strings.HasPrefix(dir, "/") {
				dir = "./" + dir
			}
			if !strings.HasSuffix(dir, "/") {
				dir += "/"
			}
			im.Imports[depName+"/"] = dir
		}
	}

	// 5. Populate scopes for each module
	for _, mod := range loadedModules {
		if mod.relBaseDir == "" {
			continue
		}
		scopeKey := mod.relBaseDir
		if !strings.HasPrefix(scopeKey, ".") && !strings.HasPrefix(scopeKey, "/") {
			scopeKey = "./" + scopeKey
		}
		if !strings.HasSuffix(scopeKey, "/") {
			scopeKey += "/"
		}

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
					dir := filepath.Dir(depRel)
					if !strings.HasPrefix(dir, ".") && !strings.HasPrefix(dir, "/") {
						dir = "./" + dir
					}
					if !strings.HasSuffix(dir, "/") {
						dir += "/"
					}
					im.Scopes[scopeKey][depName+"/"] = dir
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

	return im, nil
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

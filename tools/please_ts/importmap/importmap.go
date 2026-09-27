package importmap

import (
	"encoding/json"
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

	if err := im.MapTargetSources(moduleName, srcs, workingDir); err != nil {
		return nil, err
	}

	depPaths := discoverDepPaths(deps, workingDir)
	loadedModules, err := im.LoadDependencies(depPaths, workingDir)
	if err != nil {
		return nil, err
	}

	rootModules := im.RegisterRootModules(loadedModules, workingDir)
	im.HoistPeerDependencies(loadedModules, rootModules)
	im.PopulateModuleScopes(loadedModules, workingDir)

	return im, nil
}

// MapTargetSources maps the target's own module_name and source file subpaths to im.Imports.
func (im *ImportMap) MapTargetSources(moduleName string, srcs []string, workingDir string) error {
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

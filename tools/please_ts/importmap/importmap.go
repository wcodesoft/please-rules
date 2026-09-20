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
	Entry      string            `json:"entry"`
	Types      string            `json:"types,omitempty"`
	Imports    map[string]string `json:"imports,omitempty"`
	Files      []string          `json:"files,omitempty"`
	SourcePath string            `json:"source_path,omitempty"`
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

	// 2. Map dependencies
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

	seenDeps := make(map[string]bool)
	for _, dep := range allDeps {
		dep = strings.TrimSpace(dep)
		if dep == "" || seenDeps[dep] {
			continue
		}
		seenDeps[dep] = true

		// Check if dep points to a module metadata JSON file
		if strings.HasSuffix(dep, "ts_module.json") || strings.HasSuffix(dep, "ts_metadata.json") {
			if err := loadMetadata(im, dep, workingDir); err != nil {
				return nil, fmt.Errorf("failed loading metadata from %s: %w", dep, err)
			}
			continue
		}

		// Check if dep is a directory containing ts_module.json or ts_metadata.json
		info, err := os.Stat(dep)
		if err == nil && info.IsDir() {
			metaPath := filepath.Join(dep, "ts_module.json")
			if _, err := os.Stat(metaPath); err == nil {
				if err := loadMetadata(im, metaPath, workingDir); err != nil {
					return nil, fmt.Errorf("failed loading %s: %w", metaPath, err)
				}
				continue
			}

			metaLibPath := filepath.Join(dep, "ts_metadata.json")
			if _, err := os.Stat(metaLibPath); err == nil {
				if err := loadMetadata(im, metaLibPath, workingDir); err != nil {
					return nil, fmt.Errorf("failed loading %s: %w", metaLibPath, err)
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

func findTargetEntry(moduleName string, srcs []string) string {
	cleanName := filepath.Base(moduleName)
	if strings.HasPrefix(cleanName, "@") {
		cleanName = strings.TrimPrefix(cleanName, "@")
	}

	// Look for exact match name.ts, mod.ts, index.ts, or first src
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

func loadMetadata(im *ImportMap, metaFile string, workingDir string) error {
	data, err := os.ReadFile(metaFile)
	if err != nil {
		return err
	}

	var meta ModuleMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return err
	}

	baseDir := filepath.Dir(metaFile)
	if meta.Name != "" && meta.Entry != "" {
		entryPath := filepath.Join(baseDir, meta.Entry)
		relEntry, err := relativeTo(workingDir, entryPath)
		if err == nil {
			im.Imports[meta.Name] = relEntry
			im.Imports[meta.Entry] = relEntry
			im.Imports["./"+meta.Entry] = relEntry
			if !strings.HasSuffix(meta.Name, "/") {
				dir := filepath.Dir(relEntry)
				if !strings.HasPrefix(dir, ".") && !strings.HasPrefix(dir, "/") {
					dir = "./" + dir
				}
				if !strings.HasSuffix(dir, "/") {
					dir += "/"
				}
				im.Imports[meta.Name+"/"] = dir
			}
		}
	}

	// Map any source files in baseDir and meta.Files under meta.Name and relative names
	if meta.Name != "" {
		for _, f := range meta.Files {
			filePath := filepath.Join(baseDir, f)
			if rel, err := relativeTo(workingDir, filePath); err == nil {
				ext := filepath.Ext(f)
				nameWithoutExt := strings.TrimSuffix(f, ext)
				im.Imports[meta.Name+"/"+f] = rel
				im.Imports[meta.Name+"/"+nameWithoutExt] = rel
				im.Imports["./"+f] = rel
				im.Imports[f] = rel
			}
		}
	}

	if entries, err := os.ReadDir(baseDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && isSourceFile(e.Name()) {
				filePath := filepath.Join(baseDir, e.Name())
				if rel, err := relativeTo(workingDir, filePath); err == nil {
					name := e.Name()
					ext := filepath.Ext(name)
					nameWithoutExt := strings.TrimSuffix(name, ext)
					if meta.Name != "" {
						im.Imports[meta.Name+"/"+name] = rel
						im.Imports[meta.Name+"/"+nameWithoutExt] = rel
					}
					im.Imports["./"+name] = rel
					im.Imports[name] = rel
				}
			}
		}
	}

	for k, v := range meta.Imports {
		if meta.Name != "" && (k == meta.Name || strings.HasPrefix(k, meta.Name+"/")) {
			continue
		}
		if k == meta.Entry || k == "./"+meta.Entry {
			continue
		}
		resolved := v
		if !strings.HasPrefix(v, "http://") && !strings.HasPrefix(v, "https://") && !strings.HasPrefix(v, "jsr:") && !strings.HasPrefix(v, "npm:") {
			cleanV := strings.TrimPrefix(v, "./")
			if _, err := os.Stat(cleanV); err == nil {
				rel, err := relativeTo(workingDir, cleanV)
				if err == nil {
					resolved = rel
				}
			} else {
				fullPath := filepath.Join(baseDir, v)
				rel, err := relativeTo(workingDir, fullPath)
				if err == nil {
					resolved = rel
				}
			}
		}
		if strings.HasSuffix(k, "/") && !strings.HasSuffix(resolved, "/") {
			resolved += "/"
		}
		im.Imports[k] = resolved
	}

	return nil
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

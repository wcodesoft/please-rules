package importmap

import (
	"os"
	"path/filepath"
	"strings"
)

// RegisterRootModules registers top-level imports for all loaded root modules.
func (im *ImportMap) RegisterRootModules(loadedModules []*loadedModule, workingDir string) map[string]*loadedModule {
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
		im.MapModuleFiles(mod, workingDir)
	}
	return rootModules
}

// HoistPeerDependencies hoists undeclared peer dependencies across modules to the root imports map.
func (im *ImportMap) HoistPeerDependencies(loadedModules []*loadedModule, rootModules map[string]*loadedModule) {
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

// PopulateModuleScopes sets up W3C Import Map scopes for each module.
func (im *ImportMap) PopulateModuleScopes(loadedModules []*loadedModule, workingDir string) {
	for _, mod := range loadedModules {
		if mod.relBaseDir == "" {
			continue
		}
		im.PopulateSingleModuleScope(mod, workingDir)
	}
}

// PopulateSingleModuleScope configures the scope block for one module.
func (im *ImportMap) PopulateSingleModuleScope(mod *loadedModule, workingDir string) {
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

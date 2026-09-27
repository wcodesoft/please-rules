package importmap

import (
	"path/filepath"
	"strings"
)

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

func isSourceFile(path string) bool {
	return strings.HasSuffix(path, ".ts") ||
		strings.HasSuffix(path, ".tsx") ||
		strings.HasSuffix(path, ".js") ||
		strings.HasSuffix(path, ".jsx") ||
		strings.HasSuffix(path, ".mjs")
}

func isMetadataFile(path string) bool {
	return strings.HasSuffix(path, "ts_module.json") || strings.HasSuffix(path, "ts_metadata.json")
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

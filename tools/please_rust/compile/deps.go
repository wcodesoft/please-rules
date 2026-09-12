package compile

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// sanitizeCrateName converts hyphenated crate names to underscores for rustc extern/crate identification.
func sanitizeCrateName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

var hashSuffixRegex = regexp.MustCompile(`-[0-9a-fA-F]{16}$`)

// extractCrateName derives the crate name from a library artifact filename.
func extractCrateName(filename string) string {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	trimmed := strings.TrimSuffix(base, ext)
	trimmed = strings.TrimPrefix(trimmed, "lib")
	trimmed = hashSuffixRegex.ReplaceAllString(trimmed, "")
	return sanitizeCrateName(trimmed)
}

// isLibFile returns true if the given filename has a shared/static library extension.
func isLibFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".rlib" || ext == ".so" || ext == ".dylib" || ext == ".dll"
}

// discoverDepFiles scans directories in inputs or the current build directory for .rlib and .so files.
func discoverDepFiles(inputs []string) []string {
	seen := make(map[string]bool)
	var deps []string

	addDep := func(p string) {
		clean := filepath.Clean(p)
		if !seen[clean] {
			seen[clean] = true
			deps = append(deps, clean)
		}
	}

	for _, input := range inputs {
		if isLibFile(input) {
			addDep(input)
		}
	}

	_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() && isLibFile(path) {
			addDep(path)
		}
		return nil
	})

	return deps
}

// updateExternMap registers depPath under its crate name if not already seen or preferred over hashed names.
func updateExternMap(externMap map[string]string, depPath, selfSanitized string) {
	cName := extractCrateName(depPath)
	if cName == "" || cName == selfSanitized {
		return
	}
	existing, exists := externMap[cName]
	isUnhashed := !strings.Contains(filepath.Base(depPath), "-")
	hasHashedExisting := exists && strings.Contains(filepath.Base(existing), "-")
	if !exists || (isUnhashed && hasHashedExisting) {
		externMap[cName] = depPath
	}
}

// resolveExternFlags builds search directory (-L) and extern library (--extern) arguments.
func resolveExternFlags(depPaths []string, selfCrate string) []string {
	searchDirs := make(map[string]bool)
	externMap := make(map[string]string)
	selfSanitized := sanitizeCrateName(selfCrate)

	for _, depPath := range depPaths {
		if dir := filepath.Dir(depPath); dir != "" && dir != "." {
			searchDirs[dir] = true
		}
		updateExternMap(externMap, depPath, selfSanitized)
	}

	var args []string
	for dir := range searchDirs {
		args = append(args, "-L", fmt.Sprintf("dependency=%s", dir))
	}
	args = append(args, "-L", "dependency=.")

	for cName, path := range externMap {
		args = append(args, "--extern", fmt.Sprintf("%s=%s", cName, path))
	}

	return args
}

package mutation

import (
	"path/filepath"
	"strings"
)

// NormalizePaths rewrites every mutant file path with fn and recomputes the report.
func (r *Report) NormalizePaths(fn func(string) string) {
	for i := range r.Mutants {
		r.Mutants[i].File = fn(r.Mutants[i].File)
	}
	r.Recompute()
}

// RelPath turns a file path into a clean, slash-separated path relative to the
// first matching root directory.
func RelPath(path string, roots ...string) string {
	path = strings.TrimPrefix(path, "file://")
	if filepath.IsAbs(path) {
		for _, root := range roots {
			if root == "" {
				continue
			}
			if rel, err := filepath.Rel(root, path); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				path = rel
				break
			}
		}
	}
	return filepath.ToSlash(filepath.Clean(path))
}

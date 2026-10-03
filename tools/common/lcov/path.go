package lcov

import (
	"path/filepath"
	"strings"
)

// NormalizePaths rewrites every file path with fn, merging records that end up with
// the same path.
func (r *Report) NormalizePaths(fn func(string) string) {
	m := Merge(r)
	for _, f := range m.Files {
		f.Path = fn(f.Path)
	}
	*r = *Merge(m)
}

// RelPath turns the path found in a coverage report into a clean, slash-separated
// path relative to root: it drops a file:// scheme and, when the path is absolute and
// below one of the given roots (tried in order, for example the working directory
// and a sandbox directory), strips that root. Other paths are only cleaned.
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

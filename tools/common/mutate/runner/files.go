package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tools/common/mutation"
	"tools/common/treesitter"
)

type fileInput struct {
	absPath string
	relPath string
	langCfg *treesitter.LanguageConfig
	orig    []byte
	mode    os.FileMode
	sites   []treesitter.Site
}

func collectFiles(
	srcs []string,
	explicitLang string,
	workDir string,
	stripPrefix string,
	allowedMutators map[treesitter.MutatorCategory]bool,
) ([]fileInput, error) {
	rawPaths, err := expandSourcePaths(srcs, explicitLang, workDir)
	if err != nil {
		return nil, err
	}
	uniquePaths, err := deduplicateSourcePaths(rawPaths)
	if err != nil {
		return nil, err
	}

	out := make([]fileInput, 0, len(uniquePaths))
	for _, absPath := range uniquePaths {
		fi, err := loadFileInput(absPath, explicitLang, workDir, stripPrefix, allowedMutators)
		if err != nil {
			return nil, err
		}
		out = append(out, fi)
	}
	return out, nil
}

func expandSourcePaths(srcs []string, explicitLang, workDir string) ([]string, error) {
	var paths []string
	for _, item := range splitCommaSpecs(srcs) {
		expanded, err := expandSourceItem(item, explicitLang, workDir)
		if err != nil {
			return nil, err
		}
		paths = append(paths, expanded...)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no source files specified (use -srcs)")
	}
	return paths, nil
}

func expandSourceItem(item, explicitLang, workDir string) ([]string, error) {
	p := item
	if !filepath.IsAbs(p) {
		p = filepath.Join(workDir, p)
	}
	info, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return walkSourceDir(p, explicitLang)
	}
	return []string{p}, nil
}

func walkSourceDir(dir, explicitLang string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(dir, func(walkPath string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if isCandidateSourceEntry(walkPath, d, explicitLang) {
			paths = append(paths, walkPath)
		}
		return nil
	})
	return paths, err
}

func isCandidateSourceEntry(walkPath string, d os.DirEntry, explicitLang string) bool {
	if d.IsDir() || treesitter.IsTestFile(walkPath) {
		return false
	}
	_, err := treesitter.ResolveLanguage(walkPath, explicitLang)
	return err == nil
}

func deduplicateSourcePaths(paths []string) ([]string, error) {
	var unique []string
	seen := make(map[string]bool, len(paths))
	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			return nil, err
		}
		if seen[absPath] || treesitter.IsTestFile(absPath) {
			continue
		}
		seen[absPath] = true
		unique = append(unique, absPath)
	}
	return unique, nil
}

func loadFileInput(
	absPath string,
	explicitLang string,
	workDir string,
	stripPrefix string,
	allowedMutators map[treesitter.MutatorCategory]bool,
) (fileInput, error) {
	langCfg, err := treesitter.ResolveLanguage(absPath, explicitLang)
	if err != nil {
		return fileInput{}, err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return fileInput{}, err
	}
	orig, err := os.ReadFile(absPath)
	if err != nil {
		return fileInput{}, err
	}
	rel := relativeReportPath(absPath, workDir, stripPrefix)
	sites, err := langCfg.Discover(rel, orig, allowedMutators)
	if err != nil {
		return fileInput{}, fmt.Errorf("%s: %w", rel, err)
	}
	return fileInput{
		absPath: absPath,
		relPath: rel,
		langCfg: langCfg,
		orig:    orig,
		mode:    info.Mode(),
		sites:   sites,
	}, nil
}

func relativeReportPath(absPath, workDir, stripPrefix string) string {
	rel := mutation.RelPath(absPath, workDir)
	if stripPrefix != "" {
		rel = strings.TrimPrefix(rel, strings.TrimSuffix(stripPrefix, "/")+"/")
	}
	return rel
}

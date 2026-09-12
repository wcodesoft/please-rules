package testrunner

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// FileCoverage represents line hit counts for a single source file.
type FileCoverage struct {
	Path     string
	LineHits map[int]int
}

// parseDaLine extracts line number and hit count from an LCOV DA record.
func parseDaLine(line string) (int, int, bool) {
	parts := strings.Split(strings.TrimPrefix(line, "DA:"), ",")
	if len(parts) < 2 {
		return 0, 0, false
	}
	lineNum, err1 := strconv.Atoi(parts[0])
	hitCount, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return lineNum, hitCount, true
}

// ParseLcov parses raw LCOV bytes into a slice of FileCoverage.
func ParseLcov(data []byte) ([]FileCoverage, error) {
	var files []FileCoverage
	var curFile *FileCoverage

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "SF:") {
			curFile = &FileCoverage{
				Path:     strings.TrimPrefix(line, "SF:"),
				LineHits: make(map[int]int),
			}
		} else if strings.HasPrefix(line, "DA:") && curFile != nil {
			if lineNum, hits, ok := parseDaLine(line); ok {
				curFile.LineHits[lineNum] = hits
			}
		} else if line == "end_of_record" && curFile != nil {
			files = append(files, *curFile)
			curFile = nil
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return files, nil
}

// NormalizeLcovPaths converts paths in FileCoverage to repository-relative paths and excludes external files.
func NormalizeLcovPaths(files []FileCoverage, repoRoot string) []FileCoverage {
	var res []FileCoverage
	for _, f := range files {
		if rel := normalizePath(f.Path, repoRoot); rel != "" {
			res = append(res, FileCoverage{
				Path:     rel,
				LineHits: f.LineHits,
			})
		}
	}
	return res
}

func isExternalOrToolchainPath(cleaned string) bool {
	return strings.Contains(cleaned, "/rustc/") ||
		strings.Contains(cleaned, "/.cargo/") ||
		strings.Contains(cleaned, "/lib/rustlib/")
}

func stripBuildDir(p string) string {
	if idx := strings.Index(p, "._build/"); idx != -1 {
		return p[idx+len("._build/"):]
	}
	return ""
}

func stripPleaseBuildPrefix(cleaned string) string {
	if s := stripBuildDir(cleaned); s != "" {
		return s
	}
	if idx := strings.Index(cleaned, "._test/run_"); idx != -1 {
		rest := cleaned[idx+len("._test/run_"):]
		if slashIdx := strings.Index(rest, "/"); slashIdx != -1 {
			return rest[slashIdx+1:]
		}
	}
	return ""
}

func resolveRepoRelative(cleaned, repoRoot string) string {
	if repoRoot == "" {
		return ""
	}
	absRepo, err := filepath.Abs(repoRoot)
	if err != nil {
		return ""
	}
	absCleaned, err := filepath.Abs(cleaned)
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(absRepo, absCleaned)
	if err == nil && !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel) {
		if strings.HasPrefix(rel, "plz-out/tmp/") {
			if s := stripBuildDir(rel); s != "" {
				return s
			}
		}
		return rel
	}
	return ""
}

func resolveCwdRelative(cleaned string) string {
	cwd, err := os.Getwd()
	if err != nil || cwd == "" {
		return ""
	}
	absCleaned, err := filepath.Abs(cleaned)
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(cwd, absCleaned)
	if err == nil && !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel) {
		return rel
	}
	return ""
}

func resolveCandidateInRepo(cleaned, repoRoot string) string {
	if repoRoot == "" || (!strings.Contains(cleaned, "plz-out") && !strings.Contains(cleaned, "tmp")) {
		return ""
	}
	parts := strings.Split(cleaned, string(filepath.Separator))
	for i := 1; i < len(parts); i++ {
		candidate := filepath.Join(parts[i:]...)
		if _, err := os.Stat(filepath.Join(repoRoot, candidate)); err == nil {
			return candidate
		}
	}
	return ""
}

func normalizePath(rawPath string, repoRoot string) string {
	cleaned := filepath.Clean(rawPath)
	if isExternalOrToolchainPath(cleaned) {
		return ""
	}
	resolvers := []func() string{
		func() string { return stripPleaseBuildPrefix(cleaned) },
		func() string { return resolveRepoRelative(cleaned, repoRoot) },
		func() string { return resolveCwdRelative(cleaned) },
		func() string { return resolveCandidateInRepo(cleaned, repoRoot) },
	}
	for _, r := range resolvers {
		if rel := r(); rel != "" {
			return rel
		}
	}
	if !filepath.IsAbs(cleaned) {
		return strings.TrimPrefix(cleaned, "./")
	}
	return strings.TrimPrefix(cleaned, "/")
}

func determineTotalLines(f FileCoverage, repoRoot string) int {
	lineCount := 0
	for l := range f.LineHits {
		if l > lineCount {
			lineCount = l
		}
	}

	sourcePath := f.Path
	if repoRoot != "" && !filepath.IsAbs(sourcePath) {
		sourcePath = filepath.Join(repoRoot, f.Path)
	}
	if data, err := os.ReadFile(sourcePath); err == nil {
		cnt := bytes.Count(data, []byte{'\n'})
		if len(data) > 0 && !bytes.HasSuffix(data, []byte{'\n'}) {
			cnt++
		}
		if cnt > lineCount {
			lineCount = cnt
		}
	}
	return lineCount
}

func formatGcovFileLine(buf *bytes.Buffer, lineHits map[int]int, line int) {
	if hits, ok := lineHits[line]; ok {
		if hits > 0 {
			fmt.Fprintf(buf, "        %d:  %3d:code\n", hits, line)
		} else {
			fmt.Fprintf(buf, "    #####:  %3d:code\n", line)
		}
	} else {
		fmt.Fprintf(buf, "        -:  %3d:code\n", line)
	}
}

// FormatGcov converts FileCoverage to Please-compatible GCOV format.
func FormatGcov(files []FileCoverage, repoRoot string) []byte {
	var buf bytes.Buffer
	for _, f := range files {
		fmt.Fprintf(&buf, "        -:    0:Source:%s\n", f.Path)
		lineCount := determineTotalLines(f, repoRoot)
		for l := 1; l <= lineCount; l++ {
			formatGcovFileLine(&buf, f.LineHits, l)
		}
	}
	return buf.Bytes()
}

// FormatLcov formats FileCoverage into standard LCOV tracefile format.
func FormatLcov(files []FileCoverage) []byte {
	var buf bytes.Buffer
	for _, f := range files {
		fmt.Fprintf(&buf, "SF:%s\n", f.Path)
		var lines []int
		for l := range f.LineHits {
			lines = append(lines, l)
		}
		sort.Ints(lines)
		for _, l := range lines {
			fmt.Fprintf(&buf, "DA:%d,%d\n", l, f.LineHits[l])
		}
		fmt.Fprintf(&buf, "end_of_record\n")
	}
	return buf.Bytes()
}

// ProcessCoverage parses, normalizes, and generates both LCOV and GCOV coverage reports.
func ProcessCoverage(lcovData []byte, repoRoot string) ([]byte, []byte, error) {
	parsed, err := ParseLcov(lcovData)
	if err != nil {
		return nil, nil, err
	}

	normalized := NormalizeLcovPaths(parsed, repoRoot)
	normLcov := FormatLcov(normalized)
	gcovData := FormatGcov(normalized, repoRoot)
	return normLcov, gcovData, nil
}

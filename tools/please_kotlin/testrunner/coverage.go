package testrunner

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileCoverage holds line execution counts for a source file.
type FileCoverage struct {
	Path     string
	LineHits map[int]int
}

// JacocoReport mirrors the XML structure produced by jacococli report.
type JacocoReport struct {
	XMLName  xml.Name        `xml:"report"`
	Packages []JacocoPackage `xml:"package"`
}

type JacocoPackage struct {
	Name        string             `xml:"name,attr"`
	SourceFiles []JacocoSourceFile `xml:"sourcefile"`
}

type JacocoSourceFile struct {
	Name  string       `xml:"name,attr"`
	Lines []JacocoLine `xml:"line"`
}

type JacocoLine struct {
	Nr int `xml:"nr,attr"`
	Mi int `xml:"mi,attr"`
	Ci int `xml:"ci,attr"`
}

// ParseJacocoXml parses JaCoCo XML report bytes into a slice of FileCoverage.
func ParseJacocoXml(data []byte) ([]FileCoverage, error) {
	var report JacocoReport
	if err := xml.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("failed to parse JaCoCo XML: %w", err)
	}

	var results []FileCoverage
	for _, pkg := range report.Packages {
		pkgPath := strings.Trim(pkg.Name, "/")
		for _, sf := range pkg.SourceFiles {
			filePath := sf.Name
			if pkgPath != "" {
				filePath = filepath.Join(pkgPath, sf.Name)
			}

			hits := make(map[int]int)
			for _, l := range sf.Lines {
				if l.Ci > 0 {
					hits[l.Nr] = l.Ci
				} else if l.Mi > 0 {
					hits[l.Nr] = 0
				}
			}

			results = append(results, FileCoverage{
				Path:     filePath,
				LineHits: hits,
			})
		}
	}
	return results, nil
}

// NormalizeCoveragePaths matches source files in the coverage report against known source files.
func NormalizeCoveragePaths(files []FileCoverage, repoRoot string, knownSrcs []string) []FileCoverage {
	knownMap := make(map[string]string)
	for _, src := range knownSrcs {
		base := filepath.Base(src)
		knownMap[base] = src
		knownMap[src] = src
	}

	var normalized []FileCoverage
	for _, f := range files {
		matchedPath := f.Path
		if full, ok := knownMap[f.Path]; ok {
			matchedPath = full
		} else if full, ok := knownMap[filepath.Base(f.Path)]; ok {
			matchedPath = full
		}

		if repoRoot != "" {
			if rel, err := filepath.Rel(repoRoot, matchedPath); err == nil && !strings.HasPrefix(rel, "..") {
				matchedPath = rel
			}
		}

		normalized = append(normalized, FileCoverage{
			Path:     filepath.ToSlash(matchedPath),
			LineHits: f.LineHits,
		})
	}
	return normalized
}

// FormatGcov formats FileCoverage into Please-compatible GCOV format.
func FormatGcov(files []FileCoverage, repoRoot string) []byte {
	var buf bytes.Buffer
	for _, f := range files {
		fmt.Fprintf(&buf, "        -:    0:Source:%s\n", f.Path)
		maxLine := determineTotalLines(f, repoRoot)
		for l := 1; l <= maxLine; l++ {
			if hits, ok := f.LineHits[l]; ok {
				if hits > 0 {
					fmt.Fprintf(&buf, "        %d:  %3d:code\n", hits, l)
				} else {
					fmt.Fprintf(&buf, "    #####:  %3d:code\n", l)
				}
			} else {
				fmt.Fprintf(&buf, "        -:  %3d:code\n", l)
			}
		}
	}
	return buf.Bytes()
}

// FormatLcov formats FileCoverage into LCOV tracefile format.
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

func determineTotalLines(fc FileCoverage, repoRoot string) int {
	maxLine := 0
	for l := range fc.LineHits {
		if l > maxLine {
			maxLine = l
		}
	}

	searchPath := fc.Path
	if repoRoot != "" && !filepath.IsAbs(searchPath) {
		searchPath = filepath.Join(repoRoot, fc.Path)
	}

	if f, err := os.Open(searchPath); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		count := 0
		for scanner.Scan() {
			count++
		}
		if count > maxLine {
			return count
		}
	}
	return maxLine
}

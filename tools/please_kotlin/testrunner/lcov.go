package testrunner

import (
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"

	"tools/common/lcov"
)

// jacocoReport mirrors the parts of a JaCoCo XML report that carry method and
// branch data, which ParseJacocoXml does not need.
type jacocoReport struct {
	Packages []struct {
		Name    string `xml:"name,attr"`
		Classes []struct {
			Name           string `xml:"name,attr"`
			SourceFileName string `xml:"sourcefilename,attr"`
			Methods        []struct {
				Name     string `xml:"name,attr"`
				Desc     string `xml:"desc,attr"`
				Line     int    `xml:"line,attr"`
				Counters []struct {
					Type    string `xml:"type,attr"`
					Covered int    `xml:"covered,attr"`
				} `xml:"counter"`
			} `xml:"method"`
		} `xml:"class"`
		SourceFiles []struct {
			Name  string `xml:"name,attr"`
			Lines []struct {
				Nr int `xml:"nr,attr"`
				Mi int `xml:"mi,attr"`
				Ci int `xml:"ci,attr"`
				Mb int `xml:"mb,attr"`
				Cb int `xml:"cb,attr"`
			} `xml:"line"`
		} `xml:"sourcefile"`
	} `xml:"package"`
}

// JacocoToLcov converts a JaCoCo XML report to an lcov report with lines, functions
// and branches. JaCoCo only records whether code ran, not how often, so execution
// counts are 0 or 1. It reports branches as counts per line (missed and covered),
// without saying which arm was taken: the arms of a line are numbered 0..n-1 with
// the covered ones first, so the arm numbers carry no meaning beyond "this many
// arms ran and this many did not". Lines that never ran have their branches marked
// as not evaluated.
func JacocoToLcov(data []byte, repoRoot string, knownSrcs []string) (*lcov.Report, error) {
	var in jacocoReport
	if err := xml.Unmarshal(data, &in); err != nil {
		return nil, fmt.Errorf("failed to parse JaCoCo XML: %w", err)
	}

	report := &lcov.Report{}
	files := make(map[string]*lcov.File)
	file := func(pkg, name string) *lcov.File {
		path := name
		if p := strings.Trim(pkg, "/"); p != "" {
			path = p + "/" + name
		}
		f, ok := files[path]
		if !ok {
			f = &lcov.File{Path: path, Lines: make(map[int]int)}
			files[path] = f
			report.Files = append(report.Files, f)
		}
		return f
	}

	for _, pkg := range in.Packages {
		for _, sf := range pkg.SourceFiles {
			f := file(pkg.Name, sf.Name)
			for _, l := range sf.Lines {
				if l.Ci == 0 && l.Mi == 0 {
					continue
				}
				hits := 0
				if l.Ci > 0 {
					hits = 1
				}
				f.Lines[l.Nr] = hits
				for arm := 0; arm < l.Cb+l.Mb; arm++ {
					taken := 0
					switch {
					case arm < l.Cb:
						taken = 1
					case hits == 0:
						taken = lcov.NotEvaluated
					}
					f.Branches = append(f.Branches, lcov.Branch{Line: l.Nr, Block: 0, Arm: arm, Taken: taken})
				}
			}
		}
		for _, c := range pkg.Classes {
			if c.SourceFileName == "" {
				continue
			}
			f := file(pkg.Name, c.SourceFileName)
			class := strings.ReplaceAll(c.Name, "/", ".")
			for _, m := range c.Methods {
				if m.Line == 0 {
					continue
				}
				hits := 0
				for _, ctr := range m.Counters {
					if ctr.Type == "METHOD" && ctr.Covered > 0 {
						hits = 1
					}
				}
				f.Functions = append(f.Functions, lcov.Function{Name: class + "." + m.Name + m.Desc, Line: m.Line, Hits: hits})
			}
		}
	}

	report.NormalizePaths(func(p string) string {
		n := NormalizeCoveragePaths([]FileCoverage{{Path: p}}, repoRoot, knownSrcs)
		return filepath.ToSlash(n[0].Path)
	})
	return report, nil
}

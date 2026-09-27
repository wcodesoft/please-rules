package testrunner

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func (opts RunOptions) resolveCoverage() (bool, string) {
	active := opts.Coverage || os.Getenv("COVERAGE") == "true" || os.Getenv("COVERAGE_FILE") != ""
	file := opts.CoverageFile
	if file == "" {
		file = os.Getenv("COVERAGE_FILE")
	}
	if file == "" && active {
		file = "test.coverage"
	}
	return active, file
}

type coverageLine struct {
	Number int
	Hits   int
}

type coverageFileRecord struct {
	Filename string
	Lines    []coverageLine
}

func generateDenoCoverage(denoBin, covDir, outputFile, denoCacheDir string) error {
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		return err
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(denoBin, "coverage", "--lcov", covDir)
	cmd.Env = append(os.Environ(), "DENO_DIR="+denoCacheDir)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		_ = os.WriteFile(outputFile, []byte("<?xml version=\"1.0\" ?>\n<coverage>\n  <packages/>\n</coverage>\n"), 0644)
		return fmt.Errorf("deno coverage (%v): %s", err, stderr.String())
	}

	cwd, _ := os.Getwd()
	xmlData := lcovToCoberturaXML(stdout.Bytes(), cwd)
	return os.WriteFile(outputFile, xmlData, 0644)
}

func lcovToCoberturaXML(lcovData []byte, cwd string) []byte {
	lines := strings.Split(string(lcovData), "\n")
	cleanCwd := filepath.Clean(cwd)

	var files []coverageFileRecord
	var currentFile *coverageFileRecord

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if strings.HasPrefix(line, "SF:") {
			if currentFile != nil {
				files = append(files, *currentFile)
			}
			path := strings.TrimPrefix(line, "SF:")
			path = strings.TrimPrefix(path, "file://")
			if strings.HasPrefix(path, cleanCwd) {
				path = strings.TrimPrefix(path, cleanCwd)
				path = strings.TrimPrefix(path, "/")
			}
			path = strings.TrimPrefix(path, "./")

			dir := filepath.Dir(path)
			base := filepath.Base(path)
			parentDir := filepath.Dir(dir)
			if parentDir != "." && parentDir != "" {
				if filepath.Base(dir) == strings.TrimSuffix(base, filepath.Ext(base)) {
					path = filepath.Join(parentDir, base)
				} else if _, err := os.Stat(filepath.Join(dir, "ts_metadata.json")); err == nil {
					path = filepath.Join(parentDir, base)
				}
			}

			currentFile = &coverageFileRecord{Filename: path}
		} else if strings.HasPrefix(line, "DA:") && currentFile != nil {
			da := strings.TrimPrefix(line, "DA:")
			parts := strings.Split(da, ",")
			if len(parts) >= 2 {
				lineNum, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
				hits, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err1 == nil && err2 == nil {
					currentFile.Lines = append(currentFile.Lines, coverageLine{
						Number: lineNum,
						Hits:   hits,
					})
				}
			}
		} else if line == "end_of_record" && currentFile != nil {
			files = append(files, *currentFile)
			currentFile = nil
		}
	}
	if currentFile != nil {
		files = append(files, *currentFile)
	}

	if len(files) == 0 {
		return []byte("<?xml version=\"1.0\" ?>\n<coverage>\n  <packages/>\n</coverage>\n")
	}

	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" ?>\n")
	sb.WriteString("<coverage>\n")
	sb.WriteString("  <packages>\n")
	sb.WriteString("    <package name=\"ts\">\n")
	sb.WriteString("      <classes>\n")
	for _, f := range files {
		sb.WriteString(fmt.Sprintf("        <class name=%q filename=%q>\n", filepath.Base(f.Filename), f.Filename))
		sb.WriteString("          <lines>\n")
		sort.Slice(f.Lines, func(i, j int) bool {
			return f.Lines[i].Number < f.Lines[j].Number
		})
		for _, l := range f.Lines {
			sb.WriteString(fmt.Sprintf("            <line number=\"%d\" hits=\"%d\"/>\n", l.Number, l.Hits))
		}
		sb.WriteString("          </lines>\n")
		sb.WriteString("        </class>\n")
	}
	sb.WriteString("      </classes>\n")
	sb.WriteString("    </package>\n")
	sb.WriteString("  </packages>\n")
	sb.WriteString("</coverage>\n")

	return []byte(sb.String())
}

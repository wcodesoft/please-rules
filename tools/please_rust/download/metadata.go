package download

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CrateMeta holds metadata about a downloaded crate, serialisable to JSON.
type CrateMeta struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	LibSrc  string `json:"lib_src"`
	Edition string `json:"edition"`
}

// writeCrateMetaFile generates and writes crate_meta.json in the extracted crate directory.
func writeCrateMetaFile(extractedDir, dirName, name, version string) error {
	meta, err := buildCrateMeta(name, version, extractedDir, dirName)
	if err != nil {
		return fmt.Errorf("building crate meta for %s-%s: %w", name, version, err)
	}

	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling crate meta: %w", err)
	}
	metaPath := filepath.Join(extractedDir, "crate_meta.json")
	if err := os.WriteFile(metaPath, metaBytes, 0644); err != nil {
		return fmt.Errorf("writing crate_meta.json: %w", err)
	}
	return nil
}

// parseCargoToml scans a Cargo.toml document for [package].edition and [lib].path
// without regular expressions, respecting section headers, comments, and quoted strings.
func parseCargoToml(content string) (edition, libPath string) {
	section := ""
	for _, rawLine := range strings.Split(content, "\n") {
		line := strings.TrimSpace(stripTomlComment(rawLine))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = unquoteTomlString(strings.TrimSpace(val))
		switch {
		case (section == "package" || section == "") && key == "edition" && val != "":
			edition = val
		case section == "lib" && key == "path" && val != "":
			libPath = val
		case section == "" && key == "lib.path" && val != "":
			libPath = val
		}
	}
	return edition, libPath
}

// stripTomlComment removes a '#' comment outside of quoted strings.
func stripTomlComment(line string) string {
	var quote byte
	escaped := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && quote == '"' {
			escaped = true
			continue
		}
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
			continue
		}
		if c == '#' {
			return line[:i]
		}
	}
	return line
}

func unquoteTomlString(val string) string {
	if len(val) >= 2 {
		if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
			return val[1 : len(val)-1]
		}
	}
	return ""
}

// buildCrateMeta reads Cargo.toml from the extracted crate directory and
// produces a CrateMeta value. dirName is the top-level directory name inside
// the tarball (e.g. "serde-1.0.0").
func buildCrateMeta(name, version, extractedDir, dirName string) (*CrateMeta, error) {
	cargoTomlPath := filepath.Join(extractedDir, "Cargo.toml")
	tomlBytes, err := os.ReadFile(cargoTomlPath)
	if err != nil {
		return nil, fmt.Errorf("reading Cargo.toml: %w", err)
	}
	parsedEdition, parsedLibPath := parseCargoToml(string(tomlBytes))

	// Parse edition from the [package] section.
	edition := "2015" // safe default for old crates
	if parsedEdition != "" {
		edition = parsedEdition
	}

	// Determine the lib source path.
	// Priority: [lib] path = "...", then src/lib.rs, then lib.rs.
	libRelPath := "src/lib.rs" // default
	if parsedLibPath != "" {
		libRelPath = parsedLibPath
	} else {
		// Check which default exists.
		if _, err := os.Stat(filepath.Join(extractedDir, "src", "lib.rs")); err != nil {
			if _, err2 := os.Stat(filepath.Join(extractedDir, "lib.rs")); err2 == nil {
				libRelPath = "lib.rs"
			}
		}
	}

	// The LibSrc field is relative to the tar root: "{name}-{version}/src/lib.rs".
	libSrc := dirName + "/" + libRelPath

	return &CrateMeta{
		Name:    name,
		Version: version,
		LibSrc:  libSrc,
		Edition: edition,
	}, nil
}

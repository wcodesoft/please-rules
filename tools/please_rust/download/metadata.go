package download

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// CrateMeta holds metadata about a downloaded crate, serialisable to JSON.
type CrateMeta struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	LibSrc  string `json:"lib_src"`
	Edition string `json:"edition"`
}

var (
	editionRegex = regexp.MustCompile(`(?m)^\s*edition\s*=\s*"(\d+)"`)
	libPathRegex = regexp.MustCompile(`(?m)^\[lib\][^\[]*path\s*=\s*"([^"]+)"`)
)

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

// buildCrateMeta reads Cargo.toml from the extracted crate directory and
// produces a CrateMeta value. dirName is the top-level directory name inside
// the tarball (e.g. "serde-1.0.0").
func buildCrateMeta(name, version, extractedDir, dirName string) (*CrateMeta, error) {
	cargoTomlPath := filepath.Join(extractedDir, "Cargo.toml")
	tomlBytes, err := os.ReadFile(cargoTomlPath)
	if err != nil {
		return nil, fmt.Errorf("reading Cargo.toml: %w", err)
	}
	tomlContent := string(tomlBytes)

	// Parse edition from the [package] section.
	edition := "2015" // safe default for old crates
	if m := editionRegex.FindStringSubmatch(tomlContent); m != nil {
		edition = m[1]
	}

	// Determine the lib source path.
	// Priority: [lib] path = "...", then src/lib.rs, then lib.rs.
	libRelPath := "src/lib.rs" // default
	if m := libPathRegex.FindStringSubmatch(tomlContent); m != nil {
		libRelPath = m[1]
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

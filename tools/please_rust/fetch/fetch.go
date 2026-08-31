package fetch

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/please-build/rust-rules/tools/please_rust/toolchain"
)

// CrateReq represents a crate dependency defined in BUILD files.
type CrateReq struct {
	Name     string
	Version  string
	Features []string
}

var crateRegex = regexp.MustCompile(`rust_crate\(\s*name\s*=\s*"([^"]+)",\s*version\s*=\s*"([^"]+)"`)

// ParseBuildFile extracts rust_crate calls from BUILD file content.
func ParseBuildFile(content string) ([]CrateReq, error) {
	matches := crateRegex.FindAllStringSubmatch(content, -1)
	var crates []CrateReq
	for _, m := range matches {
		if len(m) >= 3 {
			crates = append(crates, CrateReq{
				Name:    m[1],
				Version: m[2],
			})
		}
	}
	return crates, nil
}

// GenerateCargoToml creates a dummy Cargo.toml for building specified crates.
func GenerateCargoToml(crates []CrateReq) string {
	var sb strings.Builder
	sb.WriteString("[package]\nname = \"third_party_fetch\"\nversion = \"0.1.0\"\nedition = \"2021\"\n\n[lib]\npath = \"lib.rs\"\n\n[dependencies]\n")
	for _, c := range crates {
		if len(c.Features) > 0 {
			feats := fmt.Sprintf(`["%s"]`, strings.Join(c.Features, `", "`))
			sb.WriteString(fmt.Sprintf("%s = { version = \"%s\", features = %s }\n", c.Name, c.Version, feats))
		} else {
			sb.WriteString(fmt.Sprintf("%s = \"%s\"\n", c.Name, c.Version))
		}
	}
	return sb.String()
}

// Fetch runs cargo build to fetch and build third_party crates into outDir.
func Fetch(cargoOverride string, buildFilePath string, outDir string) error {
	cargoPath, err := toolchain.FindCargo(cargoOverride)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(buildFilePath)
	if err != nil {
		return fmt.Errorf("failed to read build file %s: %w", buildFilePath, err)
	}

	crates, err := ParseBuildFile(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse build file: %w", err)
	}

	if len(crates) == 0 {
		return nil
	}

	tmpDir, err := os.MkdirTemp("", "plz_rust_fetch_*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cargoToml := GenerateCargoToml(crates)
	if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(cargoToml), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "lib.rs"), []byte("// dummy\n"), 0644); err != nil {
		return err
	}

	cmd := exec.Command(cargoPath, "build", "--release")
	cmd.Dir = tmpDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cargo build failed: %w", err)
	}

	// Copy resulting rlibs to outDir
	targetDeps := filepath.Join(tmpDir, "target", "release", "deps")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create outDir: %w", err)
	}

	entries, err := os.ReadDir(targetDeps)
	if err != nil {
		return fmt.Errorf("failed to read deps dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".rlib") {
			src := filepath.Join(targetDeps, entry.Name())
			data, err := os.ReadFile(src)
			if err != nil {
				return err
			}
			dest := filepath.Join(outDir, entry.Name())
			if err := os.WriteFile(dest, data, 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

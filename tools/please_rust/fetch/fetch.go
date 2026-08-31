package fetch

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"tools/please_rust/toolchain"
)

// CrateReq represents a crate dependency defined in BUILD files.
type CrateReq struct {
	Name     string
	Version  string
	Features []string
}

// GenerateCargoToml creates a Cargo.toml for building specified crates.
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

// sanitizeEnv ensures PATH and HOME/CARGO_HOME are available when cargo runs in sandbox.
func getCargoEnv(cargoPath string, rustcOverride string) []string {
	env := os.Environ()
	cargoDir := filepath.Dir(cargoPath)

	var rustcDir string
	if rustcPath, err := toolchain.FindRustc(rustcOverride); err == nil {
		rustcDir = filepath.Dir(rustcPath)
	}

	pathVar := os.Getenv("PATH")
	var extraPaths []string
	if cargoDir != "" && cargoDir != "." {
		extraPaths = append(extraPaths, cargoDir)
	}
	if rustcDir != "" && rustcDir != "." && rustcDir != cargoDir {
		extraPaths = append(extraPaths, rustcDir)
	}
	if home := os.Getenv("HOME"); home != "" {
		cargoBin := filepath.Join(home, ".cargo", "bin")
		extraPaths = append(extraPaths, cargoBin)
	}

	updatedPath := strings.Join(append(extraPaths, pathVar), string(os.PathListSeparator))

	// Replace or append PATH
	pathSet := false
	for i, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			env[i] = "PATH=" + updatedPath
			pathSet = true
			break
		}
	}
	if !pathSet {
		env = append(env, "PATH="+updatedPath)
	}

	return env
}

// FetchCrate downloads and compiles a single third-party crate into outDir.
func FetchCrate(cargoOverride string, rustcOverride string, name string, version string, features []string, outDir string) error {
	cargoPath, err := toolchain.FindCargo(cargoOverride)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("plz_rust_crate_%s_*", name))
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	crates := []CrateReq{
		{
			Name:     name,
			Version:  version,
			Features: features,
		},
	}

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
	cmd.Env = getCargoEnv(cargoPath, rustcOverride)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cargo build failed for crate %s: %w", name, err)
	}

	targetDeps := filepath.Join(tmpDir, "target", "release", "deps")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create outDir: %w", outDir, err)
	}

	entries, err := os.ReadDir(targetDeps)
	if err != nil {
		return fmt.Errorf("failed to read deps dir: %w", err)
	}

	sanitizedName := strings.ReplaceAll(name, "-", "_")
	prefix := fmt.Sprintf("lib%s-", sanitizedName)
	exact := fmt.Sprintf("lib%s.rlib", sanitizedName)

	copied := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".rlib") {
			src := filepath.Join(targetDeps, entry.Name())
			data, err := os.ReadFile(src)
			if err != nil {
				return err
			}

			dest := filepath.Join(outDir, entry.Name())
			_ = os.WriteFile(dest, data, 0644)

			if entry.Name() == exact || strings.HasPrefix(entry.Name(), prefix) {
				canonical := filepath.Join(outDir, exact)
				_ = os.WriteFile(canonical, data, 0644)
				copied++
			}
		}
	}

	if copied == 0 {
		return fmt.Errorf("could not find built .rlib for crate %s in %s", name, targetDeps)
	}

	return nil
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

// FetchAll reads a BUILD file containing rust_crate declarations and builds all of them.
func FetchAll(cargoOverride string, rustcOverride string, buildFilePath string, outDir string) error {
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

	cargoPath, err := toolchain.FindCargo(cargoOverride)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "plz_rust_fetchall_*")
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
	cmd.Env = getCargoEnv(cargoPath, rustcOverride)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cargo build failed: %w", err)
	}

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
			_ = os.WriteFile(dest, data, 0644)
		}
	}

	return nil
}

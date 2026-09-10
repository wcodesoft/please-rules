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

// collectExtraPaths builds additional directory paths to prepend to PATH.
func collectExtraPaths(cargoPath string, rustcOverride string) []string {
	var extraPaths []string
	cargoDir := filepath.Dir(cargoPath)
	if cargoDir != "" && cargoDir != "." {
		extraPaths = append(extraPaths, cargoDir)
	}

	if rustcPath, err := toolchain.FindRustc(rustcOverride); err == nil {
		rustcDir := filepath.Dir(rustcPath)
		if rustcDir != "" && rustcDir != "." && rustcDir != cargoDir {
			extraPaths = append(extraPaths, rustcDir)
		}
	}

	if home := os.Getenv("HOME"); home != "" {
		extraPaths = append(extraPaths, filepath.Join(home, ".cargo", "bin"))
	}
	return extraPaths
}

// getCargoEnv ensures PATH and HOME/CARGO_HOME are available when cargo runs in sandbox.
func getCargoEnv(cargoPath string, rustcOverride string) []string {
	env := os.Environ()
	extraPaths := collectExtraPaths(cargoPath, rustcOverride)
	pathVar := os.Getenv("PATH")
	updatedPath := strings.Join(append(extraPaths, pathVar), string(os.PathListSeparator))

	for i, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			env[i] = "PATH=" + updatedPath
			return env
		}
	}
	return append(env, "PATH="+updatedPath)
}

// buildCratesInSandbox generates a cargo project in a temp directory and builds the specified crates.
func buildCratesInSandbox(cargoPath, rustcOverride string, crates []CrateReq) (string, func(), error) {
	tmpDir, err := os.MkdirTemp("", "plz_rust_build_*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	cleanup := func() { os.RemoveAll(tmpDir) }

	cargoToml := GenerateCargoToml(crates)
	if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(cargoToml), 0644); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "lib.rs"), []byte("// dummy\n"), 0644); err != nil {
		cleanup()
		return "", nil, err
	}

	cmd := exec.Command(cargoPath, "build", "--release")
	cmd.Dir = tmpDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = getCargoEnv(cargoPath, rustcOverride)

	if err := cmd.Run(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("cargo build failed: %w", err)
	}

	return tmpDir, cleanup, nil
}

// isArtifactFile checks if a file entry is a compiled Rust artifact (.rlib, .so, or .dylib).
func isArtifactFile(e os.DirEntry) bool {
	if e.IsDir() {
		return false
	}
	name := e.Name()
	return strings.HasSuffix(name, ".rlib") || strings.HasSuffix(name, ".so") || strings.HasSuffix(name, ".dylib")
}

// copySingleCrateArtifacts copies compiled dependencies and produces the canonical artifact for a crate.
func copySingleCrateArtifacts(targetDeps, outDir, name string, procMacro bool) error {
	entries, err := os.ReadDir(targetDeps)
	if err != nil {
		return fmt.Errorf("failed to read deps dir: %w", err)
	}

	sanitizedName := strings.ReplaceAll(name, "-", "_")
	ext := ".rlib"
	if procMacro {
		ext = ".so"
	}

	exact := fmt.Sprintf("lib%s%s", sanitizedName, ext)
	prefix := fmt.Sprintf("lib%s-", sanitizedName)

	copied := 0
	for _, entry := range entries {
		if !isArtifactFile(entry) {
			continue
		}
		eName := entry.Name()
		data, err := os.ReadFile(filepath.Join(targetDeps, eName))
		if err != nil {
			return err
		}

		_ = os.WriteFile(filepath.Join(outDir, eName), data, 0644)
		if (strings.HasPrefix(eName, prefix) || eName == exact) && strings.HasSuffix(eName, ext) {
			_ = os.WriteFile(filepath.Join(outDir, exact), data, 0644)
			copied++
		}
	}

	if copied == 0 {
		return fmt.Errorf("could not find built %s for crate %s in %s", ext, name, targetDeps)
	}
	return nil
}

// FetchCrate downloads and compiles a single third-party crate into outDir.
func FetchCrate(cargoOverride string, rustcOverride string, name string, version string, features []string, procMacro bool, outDir string) error {
	cargoPath, err := toolchain.FindCargo(cargoOverride)
	if err != nil {
		return err
	}

	crates := []CrateReq{
		{
			Name:     name,
			Version:  version,
			Features: features,
		},
	}

	tmpDir, cleanup, err := buildCratesInSandbox(cargoPath, rustcOverride, crates)
	if err != nil {
		return fmt.Errorf("cargo build failed for crate %s: %w", name, err)
	}
	defer cleanup()

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create outDir: %w", err)
	}

	targetDeps := filepath.Join(tmpDir, "target", "release", "deps")
	return copySingleCrateArtifacts(targetDeps, outDir, name, procMacro)
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

// copyAllArtifacts copies all compiled Rust artifacts from targetDeps to outDir.
func copyAllArtifacts(targetDeps, outDir string) error {
	entries, err := os.ReadDir(targetDeps)
	if err != nil {
		return fmt.Errorf("failed to read deps dir: %w", err)
	}

	for _, entry := range entries {
		if isArtifactFile(entry) {
			data, err := os.ReadFile(filepath.Join(targetDeps, entry.Name()))
			if err != nil {
				return err
			}
			_ = os.WriteFile(filepath.Join(outDir, entry.Name()), data, 0644)
		}
	}
	return nil
}

// parseBuildCrates reads and parses crate declarations from a BUILD file path.
func parseBuildCrates(buildFilePath string) ([]CrateReq, error) {
	content, err := os.ReadFile(buildFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read build file %s: %w", buildFilePath, err)
	}
	return ParseBuildFile(string(content))
}

// FetchAll reads a BUILD file containing rust_crate declarations and builds all of them.
func FetchAll(cargoOverride string, rustcOverride string, buildFilePath string, outDir string) error {
	crates, err := parseBuildCrates(buildFilePath)
	if err != nil {
		return err
	}
	if len(crates) == 0 {
		return nil
	}

	cargoPath, err := toolchain.FindCargo(cargoOverride)
	if err != nil {
		return err
	}

	tmpDir, cleanup, err := buildCratesInSandbox(cargoPath, rustcOverride, crates)
	if err != nil {
		return err
	}
	defer cleanup()

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create outDir: %w", err)
	}

	targetDeps := filepath.Join(tmpDir, "target", "release", "deps")
	return copyAllArtifacts(targetDeps, outDir)
}

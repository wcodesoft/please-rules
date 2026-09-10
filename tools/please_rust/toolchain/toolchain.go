package toolchain

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// checkExecutable returns path if target is an existing non-directory file or found in PATH.
func checkExecutable(target string) string {
	if path, err := exec.LookPath(target); err == nil {
		return path
	}
	if fi, err := os.Stat(target); err == nil && !fi.IsDir() {
		return target
	}
	return ""
}

// candidateSearchPaths returns common directories where tool binaries may be installed.
func candidateSearchPaths(name string) []string {
	homeDir, _ := os.UserHomeDir()
	return []string{
		filepath.Join(homeDir, ".cargo", "bin", name),
		filepath.Join("/home/linuxbrew/.linuxbrew/bin", name),
		filepath.Join("/usr/local/bin", name),
		filepath.Join("/usr/bin", name),
		filepath.Join("/opt/homebrew/bin", name),
	}
}

// findTool resolves the path to an executable tool (e.g. rustc, cargo).
func findTool(override, name string) (string, error) {
	candidates := []string{override}
	if override == "" {
		candidates = append([]string{name}, candidateSearchPaths(name)...)
	}

	for _, c := range candidates {
		if c != "" {
			if found := checkExecutable(c); found != "" {
				return found, nil
			}
		}
	}

	if override != "" {
		return "", fmt.Errorf("specified %s '%s' not found", name, override)
	}
	return "", fmt.Errorf("%s not found in PATH or standard locations (~/.cargo/bin, linuxbrew, /usr/local/bin). Please install %s or specify --%s", name, name, name)
}

// FindRustc resolves the path to the rustc binary.
// It checks the provided override, PATH, and common standard installation paths.
func FindRustc(override string) (string, error) {
	return findTool(override, "rustc")
}

// FindCargo resolves the path to the cargo binary.
func FindCargo(override string) (string, error) {
	return findTool(override, "cargo")
}

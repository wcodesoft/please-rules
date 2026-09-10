package toolchain

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// findTool resolves the path to an executable tool (e.g. rustc, cargo).
func findTool(override, name string) (string, error) {
	checkPath := func(target string) string {
		if path, err := exec.LookPath(target); err == nil {
			return path
		}
		if fi, err := os.Stat(target); err == nil && !fi.IsDir() {
			return target
		}
		return ""
	}

	if override != "" {
		if found := checkPath(override); found != "" {
			return found, nil
		}
		return "", fmt.Errorf("specified %s '%s' not found", name, override)
	}

	if found := checkPath(name); found != "" {
		return found, nil
	}

	homeDir, _ := os.UserHomeDir()
	candidatePaths := []string{
		filepath.Join(homeDir, ".cargo", "bin", name),
		filepath.Join("/home/linuxbrew/.linuxbrew/bin", name),
		filepath.Join("/usr/local/bin", name),
		filepath.Join("/usr/bin", name),
		filepath.Join("/opt/homebrew/bin", name),
	}

	for _, p := range candidatePaths {
		if p == "" {
			continue
		}
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, nil
		}
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

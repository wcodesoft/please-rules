package toolchain

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// FindRustc resolves the path to the rustc binary.
// It checks the provided override, PATH, and common standard installation paths.
func FindRustc(override string) (string, error) {
	if override != "" {
		if path, err := exec.LookPath(override); err == nil {
			return path, nil
		}
		if _, err := os.Stat(override); err == nil {
			return override, nil
		}
		return "", fmt.Errorf("specified rustc '%s' not found", override)
	}

	if path, err := exec.LookPath("rustc"); err == nil {
		return path, nil
	}

	homeDir, _ := os.UserHomeDir()
	candidatePaths := []string{
		filepath.Join(homeDir, ".cargo", "bin", "rustc"),
		"/home/linuxbrew/.linuxbrew/bin/rustc",
		"/usr/local/bin/rustc",
		"/usr/bin/rustc",
		"/opt/homebrew/bin/rustc",
	}

	for _, p := range candidatePaths {
		if p == "" {
			continue
		}
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, nil
		}
	}

	return "", fmt.Errorf("rustc not found in PATH or standard locations (~/.cargo/bin, linuxbrew, /usr/local/bin). Please install rustc or specify --rustc")
}

// FindCargo resolves the path to the cargo binary.
func FindCargo(override string) (string, error) {
	if override != "" {
		if path, err := exec.LookPath(override); err == nil {
			return path, nil
		}
		if _, err := os.Stat(override); err == nil {
			return override, nil
		}
		return "", fmt.Errorf("specified cargo '%s' not found", override)
	}

	if path, err := exec.LookPath("cargo"); err == nil {
		return path, nil
	}

	homeDir, _ := os.UserHomeDir()
	candidatePaths := []string{
		filepath.Join(homeDir, ".cargo", "bin", "cargo"),
		"/home/linuxbrew/.linuxbrew/bin/cargo",
		"/usr/local/bin/cargo",
		"/usr/bin/cargo",
		"/opt/homebrew/bin/cargo",
	}

	for _, p := range candidatePaths {
		if p == "" {
			continue
		}
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, nil
		}
	}

	return "", fmt.Errorf("cargo not found in PATH or standard locations (~/.cargo/bin, linuxbrew, /usr/local/bin). Please install cargo or specify --cargo")
}

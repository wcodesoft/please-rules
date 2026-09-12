package toolchain

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	paths := []string{
		filepath.Join(homeDir, ".cargo", "bin", name),
		filepath.Join("/home/linuxbrew/.linuxbrew/bin", name),
		filepath.Join("/usr/local/bin", name),
		filepath.Join("/usr/bin", name),
		filepath.Join("/bin", name),
		filepath.Join("/opt/homebrew/bin", name),
	}

	// Check LLVM system directories
	llvmBins, _ := filepath.Glob("/usr/lib/llvm-*/bin/" + name)
	paths = append(paths, llvmBins...)

	// Check rustup toolchains
	rustupBins, _ := filepath.Glob(filepath.Join(homeDir, ".rustup", "toolchains", "*", "lib", "rustlib", "*", "bin", name))
	paths = append(paths, rustupBins...)

	// Try rustc sysroot if available
	if rustcPath, err := exec.LookPath("rustc"); err == nil {
		if out, err := exec.Command(rustcPath, "--print", "sysroot").Output(); err == nil {
			sysroot := strings.TrimSpace(string(out))
			paths = append(paths, filepath.Join(sysroot, "bin", name))
			sysrootBins, _ := filepath.Glob(filepath.Join(sysroot, "lib", "rustlib", "*", "bin", name))
			paths = append(paths, sysrootBins...)
		}
	}

	return paths
}

// findTool resolves the path to an executable tool (e.g. rustc, cargo, llvm-profdata, llvm-cov).
func findTool(override, name string) (string, error) {
	candidates := []string{override}
	if override == "" {
		candidates = append([]string{name}, candidateSearchPaths(name)...)
	}

	for _, c := range candidates {
		if c != "" {
			if found := checkExecutable(c); found != "" {
				if abs, err := filepath.Abs(found); err == nil {
					return abs, nil
				}
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
// FindLlvmProfdata resolves the path to the llvm-profdata binary.
func FindLlvmProfdata(override string) (string, error) {
	return findTool(override, "llvm-profdata")
}

// FindLlvmCov resolves the path to the llvm-cov binary.
func FindLlvmCov(override string) (string, error) {
	return findTool(override, "llvm-cov")
}

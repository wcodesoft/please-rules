package binary

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Options contains parameters for building a native Swift executable.
type Options struct {
	Swiftc       string
	Out          string
	Main         string
	Srcs         []string
	Deps         []string
	ModuleName   string
	Flags        []string
	StaticStdlib bool
}

func resolveTool(tool string) string {
	if tool == "" {
		return tool
	}
	if filepath.IsAbs(tool) {
		return tool
	}
	if p, err := exec.LookPath(tool); err == nil {
		return p
	}
	commonDirs := []string{
		"/home/linuxbrew/.linuxbrew/bin",
		"/usr/local/bin",
		"/usr/bin",
		"/opt/swift/usr/bin",
	}
	for _, dir := range commonDirs {
		candidate := filepath.Join(dir, tool)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return tool
}

// Run compiles Swift sources and links dependencies into an executable.
func Run(opts Options) error {
	if opts.Swiftc == "" {
		opts.Swiftc = "swiftc"
	}
	opts.Swiftc = resolveTool(opts.Swiftc)
	if opts.Out == "" {
		return fmt.Errorf("out binary path must not be empty")
	}

	allSrcs := make([]string, 0, len(opts.Srcs)+1)
	if opts.Main != "" {
		allSrcs = append(allSrcs, opts.Main)
	}
	for _, src := range opts.Srcs {
		if src != opts.Main {
			allSrcs = append(allSrcs, src)
		}
	}

	if len(allSrcs) == 0 {
		return fmt.Errorf("at least one source file or main entrypoint must be specified")
	}

	if err := os.MkdirAll(filepath.Dir(opts.Out), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	includeDirs, archives := discoverDependencies(opts.Deps)

	var args []string
	args = append(args, "-o", opts.Out)

	if opts.ModuleName != "" {
		args = append(args, "-module-name", opts.ModuleName)
	}
	if opts.StaticStdlib {
		args = append(args, "-static-stdlib")
	}

	for _, inc := range includeDirs {
		args = append(args, "-I", inc)
	}
	for _, arch := range archives {
		args = append(args, arch)
	}

	args = append(args, opts.Flags...)
	args = append(args, allSrcs...)

	var stderr bytes.Buffer
	cmd := exec.Command(opts.Swiftc, args...)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("swift binary linking failed (%w): %s", err, stderr.String())
	}

	return nil
}

func discoverDependencies(deps []string) (includeDirs []string, archives []string) {
	incSet := make(map[string]bool)
	archSet := make(map[string]bool)

	searchRoots := deps
	if len(searchRoots) == 0 {
		searchRoots = []string{"."}
	}

	for _, root := range searchRoots {
		info, err := os.Stat(root)
		if err == nil && info.IsDir() {
			if !incSet[root] {
				incSet[root] = true
				includeDirs = append(includeDirs, root)
			}
		}

		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if strings.HasSuffix(path, ".swiftmodule") {
				dir := filepath.Dir(path)
				if !incSet[dir] {
					incSet[dir] = true
					includeDirs = append(includeDirs, dir)
				}
			}
			if strings.HasSuffix(path, ".a") {
				if !archSet[path] {
					archSet[path] = true
					archives = append(archives, path)
				}
			}
			return nil
		})
	}
	return includeDirs, archives
}

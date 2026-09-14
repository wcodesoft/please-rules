package compile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Metadata describes a compiled Swift module and its artifacts.
type Metadata struct {
	ModuleName   string   `json:"module_name"`
	Archive      string   `json:"archive"`
	SwiftModule  string   `json:"swiftmodule"`
	ModuleDir    string   `json:"module_dir"`
	Dependencies []string `json:"dependencies,omitempty"`
}

// Options contains parameters for compiling a Swift library.
type Options struct {
	Swiftc       string
	ModuleName   string
	Srcs         []string
	Deps         []string
	OutDir       string
	OutLib       string
	OutModule    string
	SwiftVersion string
	Flags        []string
	StaticStdlib bool
	Coverage     bool
}

// ExpandCommaSeparated splits comma- or whitespace-separated strings into a slice of trimmed strings.
func ExpandCommaSeparated(items []string) []string {
	var result []string
	for _, item := range items {
		for _, part := range strings.Fields(item) {
			for _, sub := range strings.Split(part, ",") {
				trimmed := strings.TrimSpace(sub)
				if trimmed != "" {
					result = append(result, trimmed)
				}
			}
		}
	}
	return result
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

// Run compiles Swift sources into a static library archive and module interface.
func Run(opts Options) error {
	if opts.Swiftc == "" {
		opts.Swiftc = "swiftc"
	}
	opts.Swiftc = resolveTool(opts.Swiftc)
	if opts.ModuleName == "" {
		return fmt.Errorf("module name must not be empty")
	}
	if len(opts.Srcs) == 0 {
		return fmt.Errorf("at least one source file must be specified")
	}
	if opts.OutLib == "" && opts.OutDir == "" {
		return fmt.Errorf("either out-lib or out-dir must be specified")
	}

	if opts.OutDir == "" {
		opts.OutDir = filepath.Dir(opts.OutLib)
	}
	if opts.OutLib == "" {
		opts.OutLib = filepath.Join(opts.OutDir, fmt.Sprintf("lib%s.a", opts.ModuleName))
	}
	if opts.OutModule == "" {
		opts.OutModule = filepath.Join(opts.OutDir, fmt.Sprintf("%s.swiftmodule", opts.ModuleName))
	}

	if err := os.MkdirAll(opts.OutDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Discover include dirs (-I) and library archives from deps
	includeDirs, archives, depNames := discoverDependencies(opts.Deps)

	var args []string
	args = append(args, "-emit-module", "-emit-library", "-static")
	args = append(args, "-module-name", opts.ModuleName)
	args = append(args, "-emit-module-path", opts.OutModule)
	args = append(args, "-o", opts.OutLib)

	if opts.SwiftVersion != "" {
		args = append(args, "-swift-version", opts.SwiftVersion)
	}
	if opts.StaticStdlib {
		args = append(args, "-static-stdlib")
	}
	if opts.Coverage || os.Getenv("COVERAGE") == "true" {
		args = append(args, "-profile-generate", "-profile-coverage-mapping")
	}

	for _, inc := range includeDirs {
		args = append(args, "-I", inc)
	}

	for _, arch := range archives {
		args = append(args, arch)
	}

	args = append(args, opts.Flags...)
	args = append(args, opts.Srcs...)

	var stderr bytes.Buffer
	cmd := exec.Command(opts.Swiftc, args...)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("swift compilation failed (%w): %s", err, stderr.String())
	}

	// Write swift_metadata.json
	meta := Metadata{
		ModuleName:   opts.ModuleName,
		Archive:      filepath.Base(opts.OutLib),
		SwiftModule:  filepath.Base(opts.OutModule),
		ModuleDir:    opts.OutDir,
		Dependencies: depNames,
	}
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal swift metadata: %w", err)
	}
	metaPath := filepath.Join(opts.OutDir, "swift_metadata.json")
	if err := os.WriteFile(metaPath, metaBytes, 0644); err != nil {
		return fmt.Errorf("failed to write swift_metadata.json: %w", err)
	}

	return nil
}

func discoverDependencies(deps []string) (includeDirs []string, archives []string, depNames []string) {
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
			if info.IsDir() {
				metaFile := filepath.Join(path, "swift_metadata.json")
				if data, err := os.ReadFile(metaFile); err == nil {
					var meta Metadata
					if err := json.Unmarshal(data, &meta); err == nil {
						depNames = append(depNames, meta.ModuleName)
					}
					if !incSet[path] {
						incSet[path] = true
						includeDirs = append(includeDirs, path)
					}
				}
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
	return includeDirs, archives, depNames
}

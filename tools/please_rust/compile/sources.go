package compile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// crateMeta mirrors the JSON produced by the download package's crate_meta.json.
type crateMeta struct {
	LibSrc  string `json:"lib_src"`
	Edition string `json:"edition"`
}

// readCrateMeta reads and JSON-decodes a crate_meta.json file produced by the download subcommand.
func readCrateMeta(path string) (*crateMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading crate_meta.json %s: %w", path, err)
	}
	var m crateMeta
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing crate_meta.json %s: %w", path, err)
	}
	return &m, nil
}

// applyCrateMeta updates opts with LibSrc and Edition from opts.Meta if set.
func applyCrateMeta(opts *Options) error {
	if opts.Meta == "" {
		return nil
	}
	meta, err := readCrateMeta(opts.Meta)
	if err != nil {
		return err
	}
	if meta.LibSrc != "" {
		opts.MainSrc = meta.LibSrc
	}
	if meta.Edition != "" {
		opts.Edition = meta.Edition
	}
	return nil
}

// fileExists returns true if path exists on disk and is a regular file.
func fileExists(path string) bool {
	if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
		return true
	}
	return false
}

// findMatchingInput checks if target exists on filesystem or matches an input path.
func findMatchingInput(target string, inputs []string) string {
	if fileExists(target) {
		return target
	}
	targetBase := filepath.Base(target)
	for _, input := range inputs {
		if input == target || filepath.Base(input) == targetBase || stringsHasSuffix(input, target) {
			if fileExists(input) {
				return input
			}
		}
	}
	return ""
}

func stringsHasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// findFirstRsFile finds the first existing .rs file from inputs or current directory.
func findFirstRsFile(inputs []string) string {
	for _, input := range inputs {
		if filepath.Ext(input) == ".rs" && fileExists(input) {
			return input
		}
	}

	if entries, err := os.ReadDir("."); err == nil {
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".rs" {
				return e.Name()
			}
		}
	}
	return ""
}

// resolveMainSrc finds the actual entrypoint source file among inputs and current directory.
func resolveMainSrc(mainSrc string, crateType string, inputs []string) string {
	candidates := []string{"lib.rs", "main.rs", "src/lib.rs", "src/main.rs"}
	if crateType == "bin" {
		candidates = []string{"main.rs", "src/main.rs", "lib.rs", "src/lib.rs"}
	}
	if mainSrc != "" {
		candidates = append([]string{mainSrc}, candidates...)
	}

	for _, cand := range candidates {
		if found := findMatchingInput(cand, inputs); found != "" {
			return found
		}
	}

	if fallback := findFirstRsFile(inputs); fallback != "" {
		return fallback
	}

	return mainSrc
}

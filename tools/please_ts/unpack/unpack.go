package unpack

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tools/please_ts/importmap"
)

// Options holds configuration for unpacking an npm package tarball.
type Options struct {
	Tarball string
	Out     string
	Name    string
}

// PackageJSON models relevant fields from npm package.json.
type PackageJSON struct {
	Name    string          `json:"name"`
	Version string          `json:"version"`
	Main    string          `json:"main"`
	Module  string          `json:"module"`
	Types   string          `json:"types"`
	Typings string          `json:"typings"`
	Exports json.RawMessage `json:"exports"`
}

// Run unpacks the npm tarball and creates ts_module.json metadata.
func Run(opts Options) error {
	if opts.Tarball == "" {
		return fmt.Errorf("tarball path must be specified")
	}
	if opts.Out == "" {
		return fmt.Errorf("output directory must be specified")
	}

	if err := os.MkdirAll(opts.Out, 0755); err != nil {
		return err
	}

	if err := extractTarGz(opts.Tarball, opts.Out); err != nil {
		return fmt.Errorf("failed extracting tarball %s: %w", opts.Tarball, err)
	}

	// Read package.json if present
	pkgJSONPath := filepath.Join(opts.Out, "package.json")
	var pkg PackageJSON
	if data, err := os.ReadFile(pkgJSONPath); err == nil {
		_ = json.Unmarshal(data, &pkg)
	}

	modName := opts.Name
	if modName == "" {
		modName = pkg.Name
	}

	entry := determineEntry(opts.Out, pkg)
	types := pkg.Types
	if types == "" {
		types = pkg.Typings
	}

	meta := importmap.ModuleMetadata{
		Name:  modName,
		Entry: entry,
		Types: types,
	}

	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(opts.Out, "ts_module.json"), metaBytes, 0644)
}

func extractTarGz(tarGzPath, destDir string) error {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Strip standard "package/" prefix in npm tarballs
		cleanName := strings.TrimPrefix(header.Name, "package/")
		cleanName = strings.TrimPrefix(cleanName, "./package/")
		if cleanName == "" || cleanName == "package" {
			continue
		}

		targetPath := filepath.Join(destDir, cleanName)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
	return nil
}

func determineEntry(destDir string, pkg PackageJSON) string {
	// 1. Check "module" field
	if pkg.Module != "" {
		if _, err := os.Stat(filepath.Join(destDir, pkg.Module)); err == nil {
			return pkg.Module
		}
	}

	// 2. Check "main" field
	if pkg.Main != "" {
		if _, err := os.Stat(filepath.Join(destDir, pkg.Main)); err == nil {
			return pkg.Main
		}
		// Some packages omit .js
		if _, err := os.Stat(filepath.Join(destDir, pkg.Main+".js")); err == nil {
			return pkg.Main + ".js"
		}
	}

	// 3. Fallback candidates
	candidates := []string{
		"index.mjs", "index.js", "mod.ts", "index.ts",
		"dist/index.mjs", "dist/index.js",
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(destDir, c)); err == nil {
			return c
		}
	}

	return ""
}

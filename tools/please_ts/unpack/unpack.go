package unpack

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"tools/please_ts/importmap"
)

// Options holds configuration for unpacking an archive or package.
type Options struct {
	Archive string // path to tarball or zip file
	Tarball string // legacy flag alias for archive
	Out     string // destination output directory
	Name    string // module name
	Binary  string // optional binary name to extract in toolchain mode
	Symlink string // optional symlink name for extracted binary
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

// Run unpacks the archive. In toolchain mode (opts.Binary != ""), it locates
// the binary, stages its runtime dependencies, and sets permissions.
// Otherwise, it unpacks an npm module and creates ts_module.json metadata.
func Run(opts Options) error {
	archivePath := opts.Archive
	if archivePath == "" {
		archivePath = opts.Tarball
	}
	if archivePath == "" {
		return fmt.Errorf("archive path must be specified")
	}
	if opts.Out == "" {
		return fmt.Errorf("output directory must be specified")
	}

	if opts.Binary != "" {
		return unpackToolchain(archivePath, opts.Out, opts.Binary, opts.Symlink)
	}

	if err := os.MkdirAll(opts.Out, 0755); err != nil {
		return err
	}

	if err := extractArchive(archivePath, opts.Out); err != nil {
		return fmt.Errorf("failed extracting archive %s: %w", archivePath, err)
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

func extractArchive(archivePath, destDir string) error {
	if strings.HasSuffix(archivePath, ".zip") {
		return extractZip(archivePath, destDir)
	}
	return extractTarGz(archivePath, destDir)
}

func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		clean := filepath.Clean(f.Name)
		if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
			continue
		}
		targetPath := filepath.Join(destDir, clean)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		mode := f.Mode()
		if mode == 0 {
			mode = 0644
		}
		outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, mode)
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func unpackToolchain(archivePath, outDir, binaryName, symlinkName string) error {
	tmpDir, err := os.MkdirTemp("", "please_ts_toolchain_*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if err := extractArchive(archivePath, tmpDir); err != nil {
		return fmt.Errorf("failed extracting toolchain archive %s: %w", archivePath, err)
	}

	var binaryPath string
	_ = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && (info.Name() == binaryName || filepath.Base(path) == binaryName) {
			binaryPath = path
			return io.EOF
		}
		return nil
	})

	if binaryPath == "" {
		return fmt.Errorf("binary %q not found in archive %s", binaryName, archivePath)
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	binDir := filepath.Dir(binaryPath)
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		src := filepath.Join(binDir, entry.Name())
		dst := filepath.Join(outDir, entry.Name())
		if err := copyPath(src, dst); err != nil {
			return fmt.Errorf("failed copying %s to %s: %w", src, dst, err)
		}
	}

	targetBin := filepath.Join(outDir, binaryName)
	_ = os.Chmod(targetBin, 0755)

	if symlinkName != "" && symlinkName != binaryName {
		symPath := filepath.Join(outDir, symlinkName)
		_ = os.Remove(symPath)
		if err := os.Symlink(binaryName, symPath); err != nil {
			_ = copyFile(targetBin, symPath)
			_ = os.Chmod(symPath, 0755)
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_RDWR|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func copyPath(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		linkTarget, err := os.Readlink(src)
		if err != nil {
			return err
		}
		_ = os.Remove(dst)
		return os.Symlink(linkTarget, dst)
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, 0755); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyPath(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	return copyFile(src, dst)
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

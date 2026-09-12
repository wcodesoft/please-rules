package download

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// extractRegFile writes a regular file from the tar reader to target.
func extractRegFile(tr *tar.Reader, hdr *tar.Header, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return fmt.Errorf("creating parent dir for %s: %w", target, err)
	}
	mode := hdr.FileInfo().Mode()
	if mode == 0 {
		mode = 0644
	}
	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("creating file %s: %w", target, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, tr); err != nil {
		return fmt.Errorf("writing file %s: %w", target, err)
	}
	return nil
}

// extractTarEntry processes a single tar header and extracts it to destDir.
func extractTarEntry(tr *tar.Reader, hdr *tar.Header, destDir string) error {
	if filepath.IsAbs(hdr.Name) || strings.Contains(hdr.Name, "..") {
		return nil
	}

	target := filepath.Join(destDir, hdr.Name)
	switch hdr.Typeflag {
	case tar.TypeDir:
		if err := os.MkdirAll(target, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", target, err)
		}
	case tar.TypeReg, tar.TypeRegA:
		return extractRegFile(tr, hdr, target)
	case tar.TypeSymlink:
		// Skip symlinks to avoid path-traversal via link target.
	}
	return nil
}

// extractGzTar decompresses and extracts a gzipped tar stream into destDir.
// Paths that are absolute or contain ".." components are skipped for security.
func extractGzTar(r io.Reader, destDir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar entry: %w", err)
		}
		if err := extractTarEntry(tr, hdr, destDir); err != nil {
			return err
		}
	}
	return nil
}

// copyFileToTar copies a file from path into tw.
func copyFileToTar(tw *tar.Writer, path string) error {
	fh, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	defer fh.Close()
	if _, err := io.Copy(tw, fh); err != nil {
		return fmt.Errorf("copying %s into tar: %w", path, err)
	}
	return nil
}

// writeTarEntry writes a single filesystem entry into the tar archive.
func writeTarEntry(tw *tar.Writer, baseDir, path string, info os.FileInfo) error {
	rel, err := filepath.Rel(baseDir, path)
	if err != nil {
		return fmt.Errorf("computing relative path for %s: %w", path, err)
	}

	hdr, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return fmt.Errorf("building tar header for %s: %w", path, err)
	}
	hdr.Name = rel
	if info.IsDir() {
		hdr.Name += "/"
	}

	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("writing tar header for %s: %w", rel, err)
	}

	if !info.IsDir() {
		return copyFileToTar(tw, path)
	}
	return nil
}

// packTar creates a plain (non-compressed) tar archive at outPath containing
// all files under filepath.Join(baseDir, subDir), with paths relative to baseDir.
func packTar(baseDir, subDir, outPath string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return fmt.Errorf("creating output parent dir: %w", err)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating output tar %s: %w", outPath, err)
	}
	defer f.Close()

	tw := tar.NewWriter(f)
	defer tw.Close()

	srcDir := filepath.Join(baseDir, subDir)
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return writeTarEntry(tw, baseDir, path, info)
	})
}

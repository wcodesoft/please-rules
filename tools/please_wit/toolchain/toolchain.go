package toolchain

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Options struct {
	Out           string
	WitBindgenTar string
	WasmToolsTar  string
}

// ExtractBinary extracts a specific binary named binaryName from a tar.gz archive to destPath.
func ExtractBinary(tarGzPath, binaryName, destPath string) error {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return fmt.Errorf("failed to open archive %s: %w", tarGzPath, err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to read gzip header for %s: %w", tarGzPath, err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tar archive %s: %w", tarGzPath, err)
		}

		if filepath.Base(hdr.Name) == binaryName && (hdr.Typeflag == tar.TypeReg || hdr.Typeflag == tar.TypeRegA) {
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}
			outF, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return fmt.Errorf("failed to create destination binary %s: %w", destPath, err)
			}
			if _, err := io.Copy(outF, tr); err != nil {
				outF.Close()
				return fmt.Errorf("failed to write %s: %w", destPath, err)
			}
			outF.Close()
			return nil
		}
	}
	return fmt.Errorf("binary %s not found in archive %s", binaryName, tarGzPath)
}

// Run executes the toolchain extraction.
func Run(opts Options) error {
	binDir := filepath.Join(opts.Out, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}

	if opts.WitBindgenTar != "" {
		dest := filepath.Join(binDir, "wit-bindgen")
		if err := ExtractBinary(opts.WitBindgenTar, "wit-bindgen", dest); err != nil {
			return err
		}
	}

	if opts.WasmToolsTar != "" {
		dest := filepath.Join(binDir, "wasm-tools")
		if err := ExtractBinary(opts.WasmToolsTar, "wasm-tools", dest); err != nil {
			return err
		}
	}

	return nil
}

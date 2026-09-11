package download

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CrateMeta holds metadata about a downloaded crate, serialisable to JSON.
type CrateMeta struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	LibSrc  string `json:"lib_src"`
	Edition string `json:"edition"`
}

// crateURL returns the canonical crates.io download URL for the given crate.
func crateURL(name, version string) string {
	return fmt.Sprintf("https://static.crates.io/crates/%s/%s-%s.crate", name, name, version)
}

// HashCrate downloads the crate tarball and returns its hex-encoded SHA-256 digest.
// This is used by the `hash` subcommand to pre-compute the expected checksum.
func HashCrate(name, version string) (string, error) {
	url := crateURL(name, version)
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return "", fmt.Errorf("downloading crate %s-%s: %w", name, version, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("downloading crate %s-%s: HTTP %d", name, version, resp.StatusCode)
	}

	h := sha256.New()
	if _, err := io.Copy(h, resp.Body); err != nil {
		return "", fmt.Errorf("hashing crate %s-%s: %w", name, version, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// DownloadCrate downloads the crate tarball, verifies its SHA-256, extracts it,
// augments it with a crate_meta.json file, and repacks it as a plain (non-gzipped)
// tar archive at outPath.
func DownloadCrate(name, version, expectedSHA256, outPath string) error {
	url := crateURL(name, version)
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return fmt.Errorf("downloading crate %s-%s: %w", name, version, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading crate %s-%s: HTTP %d", name, version, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading crate %s-%s body: %w", name, version, err)
	}

	// SHA-256 verification.
	sum := sha256.Sum256(data)
	gotHex := hex.EncodeToString(sum[:])
	if !strings.EqualFold(gotHex, expectedSHA256) {
		return fmt.Errorf(
			"crate %s-%s checksum mismatch — possible supply chain attack: expected %s, got %s",
			name, version, expectedSHA256, gotHex,
		)
	}

	// Extract into a temporary directory.
	tmpDir, err := os.MkdirTemp("", "please_rust_download_*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := extractGzTar(bytes.NewReader(data), tmpDir); err != nil {
		return fmt.Errorf("extracting crate %s-%s: %w", name, version, err)
	}

	dirName := fmt.Sprintf("%s-%s", name, version)
	extractedDir := filepath.Join(tmpDir, dirName)

	// Build and write crate_meta.json.
	meta, err := buildCrateMeta(name, version, extractedDir, dirName)
	if err != nil {
		return fmt.Errorf("building crate meta for %s-%s: %w", name, version, err)
	}

	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling crate meta: %w", err)
	}
	metaPath := filepath.Join(extractedDir, "crate_meta.json")
	if err := os.WriteFile(metaPath, metaBytes, 0644); err != nil {
		return fmt.Errorf("writing crate_meta.json: %w", err)
	}

	// Repack as a plain tar (not gzipped).
	if err := packTar(tmpDir, dirName, outPath); err != nil {
		return fmt.Errorf("packing crate %s-%s: %w", name, version, err)
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

		// Security: reject absolute paths and path-traversal attempts.
		if filepath.IsAbs(hdr.Name) || strings.Contains(hdr.Name, "..") {
			continue
		}

		target := filepath.Join(destDir, hdr.Name)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("creating directory %s: %w", target, err)
			}
		case tar.TypeReg, tar.TypeRegA:
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
			_, copyErr := io.Copy(f, tr)
			f.Close()
			if copyErr != nil {
				return fmt.Errorf("writing file %s: %w", target, copyErr)
			}
		case tar.TypeSymlink:
			// Skip symlinks to avoid path-traversal via link target.
		}
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
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Compute the path relative to baseDir so entries appear as "{subDir}/...".
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
			fh, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("opening %s: %w", path, err)
			}
			defer fh.Close()
			if _, err := io.Copy(tw, fh); err != nil {
				return fmt.Errorf("copying %s into tar: %w", path, err)
			}
		}
		return nil
	})
	return err
}

var (
	editionRegex = regexp.MustCompile(`(?m)^\s*edition\s*=\s*"(\d+)"`)
	libPathRegex = regexp.MustCompile(`(?m)^\[lib\][^\[]*path\s*=\s*"([^"]+)"`)
)

// buildCrateMeta reads Cargo.toml from the extracted crate directory and
// produces a CrateMeta value. dirName is the top-level directory name inside
// the tarball (e.g. "serde-1.0.0").
func buildCrateMeta(name, version, extractedDir, dirName string) (*CrateMeta, error) {
	cargoTomlPath := filepath.Join(extractedDir, "Cargo.toml")
	tomlBytes, err := os.ReadFile(cargoTomlPath)
	if err != nil {
		return nil, fmt.Errorf("reading Cargo.toml: %w", err)
	}
	tomlContent := string(tomlBytes)

	// Parse edition from the [package] section.
	edition := "2015" // safe default for old crates
	if m := editionRegex.FindStringSubmatch(tomlContent); m != nil {
		edition = m[1]
	}

	// Determine the lib source path.
	// Priority: [lib] path = "...", then src/lib.rs, then lib.rs.
	libRelPath := "src/lib.rs" // default
	if m := libPathRegex.FindStringSubmatch(tomlContent); m != nil {
		libRelPath = m[1]
	} else {
		// Check which default exists.
		if _, err := os.Stat(filepath.Join(extractedDir, "src", "lib.rs")); err != nil {
			if _, err2 := os.Stat(filepath.Join(extractedDir, "lib.rs")); err2 == nil {
				libRelPath = "lib.rs"
			}
		}
	}

	// The LibSrc field is relative to the tar root: "{name}-{version}/src/lib.rs".
	libSrc := dirName + "/" + libRelPath

	return &CrateMeta{
		Name:    name,
		Version: version,
		LibSrc:  libSrc,
		Edition: edition,
	}, nil
}

package download

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// crateBaseURL allows overriding the crate download base URL for testing.
var crateBaseURL = "https://static.crates.io/crates"

// crateURL returns the canonical crates.io download URL for the given crate.
func crateURL(name, version string) string {
	return fmt.Sprintf("%s/%s/%s-%s.crate", crateBaseURL, name, name, version)
}

// fetchCrate downloads the crate bytes from url.
func fetchCrate(url string) ([]byte, error) {
	resp, err := httpClient.Get(url) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("downloading: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// HashCrate downloads the crate tarball and returns its hex-encoded SHA-256 digest.
// This is used by the `hash` subcommand to pre-compute the expected checksum.
func HashCrate(name, version string) (string, error) {
	data, err := fetchCrate(crateURL(name, version))
	if err != nil {
		return "", fmt.Errorf("fetching crate %s-%s: %w", name, version, err)
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

// verifySHA256 checks whether data matches the expected hex-encoded SHA-256 digest.
func verifySHA256(data []byte, expected string) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, expected) {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, got)
	}
	return nil
}

// unpackAndRepackCrate extracts a downloaded crate tarball, adds metadata, and repacks it as a plain tar.
func unpackAndRepackCrate(data []byte, name, version, expectedSHA256, outPath string) error {
	if err := verifySHA256(data, expectedSHA256); err != nil {
		return fmt.Errorf("crate %s-%s: %w", name, version, err)
	}

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

	if err := writeCrateMetaFile(extractedDir, dirName, name, version); err != nil {
		return err
	}

	if err := packTar(tmpDir, dirName, outPath); err != nil {
		return fmt.Errorf("packing crate %s-%s: %w", name, version, err)
	}
	return nil
}

// DownloadCrate downloads the crate tarball, verifies its SHA-256, extracts it,
// augments it with a crate_meta.json file, and repacks it as a plain (non-gzipped)
// tar archive at outPath.
func DownloadCrate(name, version, expectedSHA256, outPath string) error {
	data, err := fetchCrate(crateURL(name, version))
	if err != nil {
		return fmt.Errorf("downloading crate %s-%s: %w", name, version, err)
	}
	return unpackAndRepackCrate(data, name, version, expectedSHA256, outPath)
}

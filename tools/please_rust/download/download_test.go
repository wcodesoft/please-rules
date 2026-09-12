package download

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// createSampleCrateArchive creates an in-memory .crate (tar.gz) archive for testing.
func createSampleCrateArchive(t *testing.T, dirName, cargoToml, libContent string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	entries := []struct {
		name    string
		content string
		isDir   bool
	}{
		{name: dirName + "/", isDir: true},
		{name: dirName + "/Cargo.toml", content: cargoToml},
		{name: dirName + "/src/", isDir: true},
		{name: dirName + "/src/lib.rs", content: libContent},
	}

	for _, e := range entries {
		hdr := &tar.Header{
			Name:     e.name,
			Mode:     0644,
			Typeflag: tar.TypeReg,
		}
		if e.isDir {
			hdr.Typeflag = tar.TypeDir
			hdr.Mode = 0755
		} else {
			hdr.Size = int64(len(e.content))
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("writing header %s: %v", e.name, err)
		}
		if !e.isDir {
			if _, err := tw.Write([]byte(e.content)); err != nil {
				t.Fatalf("writing content %s: %v", e.name, err)
			}
		}
	}

	tw.Close()
	gw.Close()
	return buf.Bytes()
}

func TestDownloadCrate_SuccessAndVerify(t *testing.T) {
	dirName := "testpkg-0.1.0"
	cargoToml := `[package]
name = "testpkg"
version = "0.1.0"
edition = "2021"
`
	crateBytes := createSampleCrateArchive(t, dirName, cargoToml, "pub fn hello() {}")
	sum := sha256.Sum256(crateBytes)
	expectedHex := hex.EncodeToString(sum[:])

	// Mock crates.io server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(crateBytes)
	}))
	defer ts.Close()

	oldURL := crateBaseURL
	crateBaseURL = ts.URL
	defer func() { crateBaseURL = oldURL }()

	outDir := t.TempDir()
	outTar := filepath.Join(outDir, "testpkg.tar")

	err := DownloadCrate("testpkg", "0.1.0", expectedHex, outTar)
	if err != nil {
		t.Fatalf("DownloadCrate failed: %v", err)
	}

	if fi, err := os.Stat(outTar); err != nil || fi.Size() == 0 {
		t.Fatalf("repacked tar missing or empty: %v", err)
	}

	// Test HashCrate against the same mock server
	gotHash, err := HashCrate("testpkg", "0.1.0")
	if err != nil {
		t.Fatalf("HashCrate failed: %v", err)
	}
	if gotHash != expectedHex {
		t.Errorf("HashCrate got %s, want %s", gotHash, expectedHex)
	}
}

func TestDownloadCrate_ChecksumMismatch(t *testing.T) {
	crateBytes := []byte("dummy crate content")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(crateBytes)
	}))
	defer ts.Close()

	oldURL := crateBaseURL
	crateBaseURL = ts.URL
	defer func() { crateBaseURL = oldURL }()

	outTar := filepath.Join(t.TempDir(), "out.tar")
	err := DownloadCrate("testpkg", "0.1.0", "0000000000000000000000000000000000000000000000000000000000000000", outTar)
	if err == nil {
		t.Fatal("expected checksum mismatch error, got nil")
	}
}

func TestDownloadCrate_HttpError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	oldURL := crateBaseURL
	crateBaseURL = ts.URL
	defer func() { crateBaseURL = oldURL }()

	outTar := filepath.Join(t.TempDir(), "out.tar")
	err := DownloadCrate("notfound", "1.0.0", "abc", outTar)
	if err == nil {
		t.Fatal("expected HTTP error, got nil")
	}

	_, err = HashCrate("notfound", "1.0.0")
	if err == nil {
		t.Fatal("expected HashCrate HTTP error, got nil")
	}
}

func TestDownloadCrate_CorruptTarGz(t *testing.T) {
	corruptBytes := []byte("this is not a valid tar.gz archive")
	sum := sha256.Sum256(corruptBytes)
	hexHash := hex.EncodeToString(sum[:])

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(corruptBytes)
	}))
	defer ts.Close()

	oldURL := crateBaseURL
	crateBaseURL = ts.URL
	defer func() { crateBaseURL = oldURL }()

	outTar := filepath.Join(t.TempDir(), "out.tar")
	err := DownloadCrate("corrupt", "1.0.0", hexHash, outTar)
	if err == nil {
		t.Fatal("expected error for corrupt tar.gz payload, got nil")
	}
}

func TestFetchCrate_NetworkError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()
	_, err := fetchCrate(ts.URL)
	if err == nil {
		t.Fatal("expected network error, got nil")
	}
}

func TestUnpackAndRepackCrate_Errors(t *testing.T) {
	validBytes := createSampleCrateArchive(t, "pkg-1.0.0", "[package]\nname=\"pkg\"\nversion=\"1.0.0\"\n", "pub fn a(){}")
	sum := sha256.Sum256(validBytes)
	hexHash := hex.EncodeToString(sum[:])

	// Checksum mismatch
	err := unpackAndRepackCrate(validBytes, "pkg", "1.0.0", "wronghex", t.TempDir()+"/out.tar")
	if err == nil {
		t.Error("expected checksum mismatch error")
	}

	// PackTar error (invalid outPath)
	err = unpackAndRepackCrate(validBytes, "pkg", "1.0.0", hexHash, "/dev/null/cannot/create/out.tar")
	if err == nil {
		t.Error("expected packTar error for invalid outPath")
	}
}

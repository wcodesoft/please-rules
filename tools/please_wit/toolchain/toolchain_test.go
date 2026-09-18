package toolchain

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func createTestTarGz(t *testing.T, archivePath, binaryName, content string) {
	t.Helper()
	f, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("failed to create archive: %v", err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	hdr := &tar.Header{
		Name:     "subfolder/" + binaryName,
		Mode:     0755,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("failed to write header: %v", err)
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		t.Fatalf("failed to write content: %v", err)
	}
}

func TestExtractBinary(t *testing.T) {
	tmpDir := t.TempDir()
	tarPath := filepath.Join(tmpDir, "test.tar.gz")
	destPath := filepath.Join(tmpDir, "out", "bin", "mytool")

	createTestTarGz(t, tarPath, "mytool", "echo hello")

	if err := ExtractBinary(tarPath, "mytool", destPath); err != nil {
		t.Fatalf("ExtractBinary failed: %v", err)
	}

	data, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read extracted binary: %v", err)
	}
	if string(data) != "echo hello" {
		t.Errorf("got content %q, want %q", string(data), "echo hello")
	}
}

func TestToolchainRun(t *testing.T) {
	tmpDir := t.TempDir()
	bindgenTar := filepath.Join(tmpDir, "wit-bindgen.tar.gz")
	wasmTar := filepath.Join(tmpDir, "wasm-tools.tar.gz")
	outDir := filepath.Join(tmpDir, "toolchain")

	createTestTarGz(t, bindgenTar, "wit-bindgen", "bindgen-binary")
	createTestTarGz(t, wasmTar, "wasm-tools", "wasm-tools-binary")

	opts := Options{
		Out:           outDir,
		WitBindgenTar: bindgenTar,
		WasmToolsTar:  wasmTar,
	}

	if err := Run(opts); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "bin", "wit-bindgen")); err != nil {
		t.Errorf("wit-bindgen not extracted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "bin", "wasm-tools")); err != nil {
		t.Errorf("wasm-tools not extracted: %v", err)
	}
}

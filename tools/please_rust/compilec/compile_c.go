package compilec

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// findCC returns the C compiler to use. It checks the CC environment variable
// first, then falls back to searching for cc, gcc, and clang in PATH.
func findCC() (string, error) {
	if cc := os.Getenv("CC"); cc != "" {
		return cc, nil
	}
	for _, name := range []string{"cc", "gcc", "clang"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("no C compiler found; set CC or install cc/gcc/clang")
}

// compileSource compiles a single C source file into an object file at objPath.
func compileSource(cc, src, objPath string, iFlags []string) error {
	ccArgs := append([]string{"-c", "-O2", "-fPIC"}, iFlags...)
	ccArgs = append(ccArgs, "-o", objPath, src)

	cmd := exec.Command(cc, ccArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compiling %s: %w", src, err)
	}
	return nil
}

// compileAllSources compiles each source file in sources to an object file in tmpDir.
func compileAllSources(cc, tmpDir string, sources, includes []string) ([]string, error) {
	var iFlags []string
	for _, inc := range includes {
		iFlags = append(iFlags, "-I", inc)
	}

	var objFiles []string
	for i, src := range sources {
		base := filepath.Base(src)
		ext := filepath.Ext(base)
		objName := fmt.Sprintf("%d_%s.o", i, strings.TrimSuffix(base, ext))
		objPath := filepath.Join(tmpDir, objName)

		if err := compileSource(cc, src, objPath, iFlags); err != nil {
			return nil, err
		}
		objFiles = append(objFiles, objPath)
	}
	return objFiles, nil
}

// createArchive packages object files into a static library archive using ar.
func createArchive(outFile string, objFiles []string) error {
	if dir := filepath.Dir(outFile); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
	}

	arArgs := append([]string{"rcs", outFile}, objFiles...)
	arCmd := exec.Command("ar", arArgs...)
	arCmd.Stdout = os.Stdout
	arCmd.Stderr = os.Stderr
	if err := arCmd.Run(); err != nil {
		return fmt.Errorf("ar rcs %s: %w", outFile, err)
	}
	return nil
}

// CompileC compiles the given C source files into a static archive at outFile.
// includes is a list of directories to add as -I flags.
// sources must be non-empty.
func CompileC(outFile string, includes []string, sources []string) error {
	if len(sources) == 0 {
		return fmt.Errorf("compilec: no source files provided")
	}

	cc, err := findCC()
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "please_rust_compilec_*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	objFiles, err := compileAllSources(cc, tmpDir, sources, includes)
	if err != nil {
		return err
	}

	return createArchive(outFile, objFiles)
}

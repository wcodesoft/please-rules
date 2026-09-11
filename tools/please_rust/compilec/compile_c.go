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

	// Build the shared -I flag list.
	var iFlags []string
	for _, inc := range includes {
		iFlags = append(iFlags, "-I", inc)
	}

	// Compile each source file to an object file.
	var objFiles []string
	for i, src := range sources {
		base := filepath.Base(src)
		// Strip extension, use .o with index prefix to avoid collisions across directories.
		ext := filepath.Ext(base)
		objName := fmt.Sprintf("%d_%s.o", i, strings.TrimSuffix(base, ext))
		objPath := filepath.Join(tmpDir, objName)

		ccArgs := []string{"-c", "-O2", "-fPIC"}
		ccArgs = append(ccArgs, iFlags...)
		ccArgs = append(ccArgs, "-o", objPath, src)


		cmd := exec.Command(cc, ccArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("compiling %s: %w", src, err)
		}
		objFiles = append(objFiles, objPath)
	}

	// Ensure output directory exists.
	if dir := filepath.Dir(outFile); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
	}

	// Run: ar rcs outFile <.o files>
	arArgs := append([]string{"rcs", outFile}, objFiles...)
	arCmd := exec.Command("ar", arArgs...)
	arCmd.Stdout = os.Stdout
	arCmd.Stderr = os.Stderr
	if err := arCmd.Run(); err != nil {
		return fmt.Errorf("ar rcs %s: %w", outFile, err)
	}
	return nil
}

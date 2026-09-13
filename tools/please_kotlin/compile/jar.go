package compile

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// JarOptions configures the creation of a JAR archive.
type JarOptions struct {
	SourceDir      string
	OutJar         string
	MainClass      string
	MergeJars      []string
	ExecutableStub bool
	Java           string
}

// CreateJar packages the files in SourceDir into OutJar with an optional Main-Class in MANIFEST.MF.
func CreateJar(opts JarOptions) error {
	if err := os.MkdirAll(filepath.Dir(opts.OutJar), 0755); err != nil {
		return fmt.Errorf("failed to create output jar directory: %w", err)
	}

	outFile, err := os.Create(opts.OutJar)
	if err != nil {
		return fmt.Errorf("failed to create jar file %s: %w", opts.OutJar, err)
	}
	defer outFile.Close()

	if opts.ExecutableStub {
		javaFallback := opts.Java
		if javaFallback == "" {
			javaFallback = "java"
		}
		stub := fmt.Sprintf("#!/usr/bin/env bash\nJAVA_EXEC=\"${TOOLS_JAVA:-%s}\"\nif ! command -v \"$JAVA_EXEC\" >/dev/null 2>&1 && command -v java >/dev/null 2>&1; then\n  JAVA_EXEC=\"java\"\nfi\nexec \"$JAVA_EXEC\" -jar \"$0\" \"$@\"\n", javaFallback)
		if _, err := outFile.WriteString(stub); err != nil {
			return fmt.Errorf("failed to write executable stub: %w", err)
		}
	}

	zw := zip.NewWriter(outFile)
	defer zw.Close()

	// Write MANIFEST.MF first
	if err := writeManifest(zw, opts.MainClass); err != nil {
		return err
	}

	seenEntries := make(map[string]bool)
	seenEntries["META-INF/MANIFEST.MF"] = true

	// Walk SourceDir and write all files
	if err := walkAndWriteFiles(zw, opts.SourceDir, seenEntries); err != nil {
		return err
	}

	// Merge external jars if requested
	for _, jarPath := range opts.MergeJars {
		if err := mergeJar(zw, jarPath, seenEntries); err != nil {
			return err
		}
	}

	if err := zw.Close(); err != nil {
		return err
	}

	if opts.ExecutableStub {
		_ = os.Chmod(opts.OutJar, 0755)
	}

	return nil
}

func writeManifest(zw *zip.Writer, mainClass string) error {
	w, err := zw.Create("META-INF/MANIFEST.MF")
	if err != nil {
		return fmt.Errorf("failed to create manifest entry: %w", err)
	}
	manifest := "Manifest-Version: 1.0\nCreated-By: please_kotlin\n"
	if mainClass != "" {
		manifest += fmt.Sprintf("Main-Class: %s\n", mainClass)
	}
	manifest += "\n"
	_, err = io.WriteString(w, manifest)
	return err
}

func walkAndWriteFiles(zw *zip.Writer, sourceDir string, seenEntries map[string]bool) error {
	if sourceDir == "" {
		return nil
	}

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Normalize to forward slashes for zip entries
		zipPath := filepath.ToSlash(relPath)

		// Don't overwrite manifest if already created
		if strings.ToUpper(zipPath) == "META-INF/MANIFEST.MF" || seenEntries[zipPath] {
			return nil
		}
		seenEntries[zipPath] = true

		w, err := zw.Create(zipPath)
		if err != nil {
			return fmt.Errorf("failed to create zip entry %s: %w", zipPath, err)
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open source file %s: %w", path, err)
		}
		defer f.Close()

		if _, err := io.Copy(w, f); err != nil {
			return fmt.Errorf("failed to copy file content to zip entry %s: %w", zipPath, err)
		}
		return nil
	})
}

func mergeJar(zw *zip.Writer, jarPath string, seenEntries map[string]bool) error {
	r, err := zip.OpenReader(jarPath)
	if err != nil {
		return fmt.Errorf("failed to open jar %s for merging: %w", jarPath, err)
	}
	defer r.Close()

	for _, f := range r.File {
		name := filepath.ToSlash(f.Name)
		upper := strings.ToUpper(name)
		if upper == "META-INF/MANIFEST.MF" || strings.HasPrefix(upper, "META-INF/SIG-") ||
			strings.HasSuffix(upper, ".SF") || strings.HasSuffix(upper, ".RSA") || strings.HasSuffix(upper, ".DSA") {
			continue
		}
		if seenEntries[name] {
			continue
		}
		seenEntries[name] = true

		if f.FileInfo().IsDir() {
			continue
		}

		w, err := zw.Create(name)
		if err != nil {
			return fmt.Errorf("failed to create entry %s from jar %s: %w", name, jarPath, err)
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open entry %s in jar %s: %w", name, jarPath, err)
		}
		_, err = io.Copy(w, rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("failed to copy entry %s from jar %s: %w", name, jarPath, err)
		}
	}
	return nil
}

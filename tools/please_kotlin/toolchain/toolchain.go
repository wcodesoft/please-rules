package toolchain

import (
	"os"
	"os/exec"
	"path/filepath"
)

// ResolveKotlinc determines the path to the kotlinc binary.
func ResolveKotlinc(configured string) (string, error) {
	return resolveExecutable(configured, "kotlinc")
}

// ResolveJava determines the path to the java binary.
func ResolveJava(configured string) (string, error) {
	return resolveExecutable(configured, "java")
}

func resolveExecutable(configured, fallbackName string) (string, error) {
	if configured != "" {
		if filepath.IsAbs(configured) || fileExists(configured) {
			return configured, nil
		}
		if path, err := exec.LookPath(configured); err == nil {
			return path, nil
		}
		return configured, nil
	}
	if path, err := exec.LookPath(fallbackName); err == nil {
		return path, nil
	}
	return fallbackName, nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// FindKotlinStdlib searches for kotlin-stdlib.jar based on kotlinc or java path or environment.
func FindKotlinStdlib(kotlincOrJava string) string {
	var candidates []string
	if kotlincOrJava != "" {
		dir := filepath.Dir(kotlincOrJava)
		candidates = append(candidates,
			filepath.Join(dir, "..", "kotlinc", "lib", "kotlin-stdlib.jar"),
			filepath.Join(dir, "..", "lib", "kotlin-stdlib.jar"),
			filepath.Join(dir, "kotlinc", "lib", "kotlin-stdlib.jar"),
			filepath.Join(dir, "lib", "kotlin-stdlib.jar"),
		)
	}
	if envKotlinc := os.Getenv("TOOLS_KOTLINC"); envKotlinc != "" {
		dir := filepath.Dir(envKotlinc)
		candidates = append(candidates,
			filepath.Join(dir, "..", "kotlinc", "lib", "kotlin-stdlib.jar"),
			filepath.Join(dir, "..", "lib", "kotlin-stdlib.jar"),
		)
	}
	if envJava := os.Getenv("TOOLS_JAVA"); envJava != "" {
		dir := filepath.Dir(envJava)
		candidates = append(candidates,
			filepath.Join(dir, "..", "kotlinc", "lib", "kotlin-stdlib.jar"),
			filepath.Join(dir, "..", "..", "kotlinc", "lib", "kotlin-stdlib.jar"),
		)
	}
	for _, c := range candidates {
		clean := filepath.Clean(c)
		if fileExists(clean) {
			return clean
		}
	}
	return ""
}

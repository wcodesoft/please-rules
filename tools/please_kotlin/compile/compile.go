package compile

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"tools/please_kotlin/toolchain"
)

// Options configures a kotlinc compilation invocation.
type Options struct {
	Kotlinc    string
	Out        string
	Srcs       []string
	Deps       []string
	JvmTarget  string
	ModuleName string
	MainClass  string
	Flags      []string
	Executable bool
	MergeDeps  bool
	Java       string
}

// DiscoverJars combines explicit dependency JARs with any .jar files found in the current working directory.
func DiscoverJars(explicitDeps []string, outJar string) []string {
	seen := make(map[string]bool)
	var result []string

	addJar := func(p string) {
		clean := filepath.Clean(p)
		if clean == "" || clean == "." {
			return
		}
		if outJar != "" && (clean == filepath.Clean(outJar) || filepath.Base(clean) == filepath.Base(outJar)) {
			return
		}
		if !seen[clean] {
			seen[clean] = true
			result = append(result, clean)
		}
	}

	for _, dep := range explicitDeps {
		addJar(dep)
	}

	_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() && strings.HasSuffix(path, ".jar") {
			addJar(path)
		}
		return nil
	})

	return result
}

// BuildKotlincArgs constructs the CLI arguments for kotlinc.
func BuildKotlincArgs(classesDir string, opts Options) []string {
	var args []string
	args = append(args, "-d", classesDir)

	if opts.JvmTarget != "" {
		args = append(args, "-jvm-target", opts.JvmTarget)
	}
	if opts.ModuleName != "" {
		args = append(args, "-module-name", opts.ModuleName)
	}

	allDeps := DiscoverJars(opts.Deps, opts.Out)
	if len(allDeps) > 0 {
		cp := strings.Join(allDeps, string(os.PathListSeparator))
		args = append(args, "-cp", cp)
	}

	if len(opts.Flags) > 0 {
		args = append(args, opts.Flags...)
	}

	args = append(args, opts.Srcs...)
	return args
}

// Run executes the kotlinc compiler and packages the output into a JAR.
func Run(opts Options) error {
	if opts.Out == "" {
		return fmt.Errorf("output jar path (--out) is required")
	}
	if len(opts.Srcs) == 0 {
		return fmt.Errorf("at least one source file (--srcs) is required")
	}

	kotlinc, err := toolchain.ResolveKotlinc(opts.Kotlinc)
	if err != nil {
		return fmt.Errorf("failed to resolve kotlinc compiler: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "kotlinc-classes-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary classes directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	args := BuildKotlincArgs(tmpDir, opts)
	cmd := exec.Command(kotlinc, args...)
	cmd.Dir = "."

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		output := stderr.String()
		if output == "" {
			output = stdout.String()
		}
		return fmt.Errorf("kotlinc compilation failed: %w\n%s", err, output)
	}

	var mergeJars []string
	if opts.MergeDeps || opts.Executable {
		mergeJars = DiscoverJars(opts.Deps, opts.Out)
		if opts.Executable {
			if stdlib := toolchain.FindKotlinStdlib(opts.Kotlinc); stdlib != "" {
				mergeJars = append(mergeJars, stdlib)
			}
		}
	}

	mainClass := opts.MainClass
	if mainClass == "" && opts.Executable {
		mainClass = discoverMainFromDir(tmpDir)
	}

	jarOpts := JarOptions{
		SourceDir:      tmpDir,
		OutJar:         opts.Out,
		MainClass:      mainClass,
		MergeJars:      mergeJars,
		ExecutableStub: opts.Executable,
		Java:           opts.Java,
	}
	return CreateJar(jarOpts)
}

func discoverMainFromDir(dir string) string {
	var candidates []string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() && strings.HasSuffix(path, ".class") && !strings.Contains(path, "$") {
			rel, _ := filepath.Rel(dir, path)
			cls := strings.TrimSuffix(rel, ".class")
			cls = filepath.ToSlash(cls)
			cls = strings.ReplaceAll(cls, "/", ".")
			candidates = append(candidates, cls)
		}
		return nil
	})
	for _, c := range candidates {
		if strings.HasSuffix(c, "MainKt") {
			return c
		}
	}
	for _, c := range candidates {
		if strings.HasSuffix(c, "Kt") {
			return c
		}
	}
	if len(candidates) > 0 {
		return candidates[0]
	}
	return ""
}

// ExpandCommaSeparated splits comma- or whitespace-separated strings into a flat slice.
func ExpandCommaSeparated(items []string) []string {
	var result []string
	for _, item := range items {
		normalized := strings.ReplaceAll(item, ",", " ")
		for _, part := range strings.Fields(normalized) {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
	}
	return result
}

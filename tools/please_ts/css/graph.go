package css

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// MetadataFile is the JSON metadata file emitted in every css_library output directory.
const MetadataFile = "css_metadata.json"

// Metadata describes a compiled css_library module and its direct dependencies.
type Metadata struct {
	Name  string   `json:"name"`
	Files []string `json:"files"`
	Deps  []string `json:"deps,omitempty"`
}

// CompileOptions holds flags for `please_ts css compile`.
type CompileOptions struct {
	Name string
	Out  string
	Srcs []string
	Deps []string
}

// BundleOptions holds flags for `please_ts css bundle`.
type BundleOptions struct {
	Out           string
	Srcs          []string
	Deps          []string
	Minify        bool
	Sourcemap     bool
	AllowOverride bool
	EsbuildBinary string
	Flags         []string
}

// Compile validates the CSS sources of a css_library, writes css_metadata.json,
// and embeds transitive css_library dependencies so downstream targets only need direct deps.
func Compile(opts CompileOptions) error {
	if len(opts.Srcs) == 0 {
		return fmt.Errorf("no CSS source files provided for css_library")
	}
	if opts.Out == "" {
		return fmt.Errorf("output directory must be specified for css_library")
	}

	if err := CheckFiles(opts.Srcs); err != nil {
		return err
	}

	if err := os.MkdirAll(opts.Out, 0755); err != nil {
		return err
	}

	var files []string
	for _, src := range opts.Srcs {
		base := filepath.Base(src)
		dst := filepath.Join(opts.Out, base)
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("copying CSS source %s: %w", src, err)
		}
		files = append(files, base)
	}

	depNames := discoverDirectDepNames(opts.Deps, ".")
	name := opts.Name
	if name == "" {
		name = filepath.Base(opts.Out)
	}

	meta := Metadata{
		Name:  name,
		Files: files,
		Deps:  depNames,
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(opts.Out, MetadataFile), data, 0644); err != nil {
		return err
	}

	return embedDependencies(".", opts.Out)
}

// Bundle collects all transitive css_library files and direct srcs in topological order,
// checks for cross-file selector/@keyframes/:root variable collisions, and bundles/minifies via esbuild.
func Bundle(opts BundleOptions) error {
	if opts.Out == "" {
		return fmt.Errorf("output path must be specified for css_bundle")
	}

	graphFiles, err := DiscoverGraphFiles(opts.Deps, opts.Srcs, ".")
	if err != nil {
		return err
	}
	if len(graphFiles) == 0 {
		return fmt.Errorf("no CSS files found in srcs or deps for css_bundle %s", opts.Out)
	}

	if !opts.AllowOverride {
		if err := CheckFiles(graphFiles); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Dir(opts.Out), 0755); err != nil {
		return err
	}

	entryFile, cleanup, err := writeBundleEntryFile(graphFiles)
	if err != nil {
		return err
	}
	defer cleanup()

	args := []string{entryFile, "--outfile=" + opts.Out, "--bundle"}
	if opts.Minify {
		args = append(args, "--minify")
	}
	if opts.Sourcemap {
		args = append(args, "--sourcemap")
	}
	args = append(args, opts.Flags...)

	esbuildBin := opts.EsbuildBinary
	if esbuildBin == "" {
		esbuildBin = "esbuild"
	}
	cmd := exec.Command(esbuildBin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("esbuild CSS bundle failed: %w", err)
	}
	return nil
}

func writeBundleEntryFile(graphFiles []string) (string, func(), error) {
	var sb strings.Builder
	for _, f := range graphFiles {
		rel := f
		if !strings.HasPrefix(rel, "./") && !strings.HasPrefix(rel, "../") && !filepath.IsAbs(rel) {
			rel = "./" + filepath.ToSlash(rel)
		} else {
			rel = filepath.ToSlash(rel)
		}
		sb.WriteString(fmt.Sprintf("@import %q;\n", rel))
	}

	tmpFile, err := os.CreateTemp(".", ".please_ts_css_entry_*.css")
	if err != nil {
		return "", nil, fmt.Errorf("creating temporary CSS entry: %w", err)
	}
	name := tmpFile.Name()
	if _, err := tmpFile.WriteString(sb.String()); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(name)
		return "", nil, err
	}
	_ = tmpFile.Close()
	return name, func() { _ = os.Remove(name) }, nil
}

// DiscoverGraphFiles returns all unique CSS files across transitive css_library dependencies
// in deterministic topological order (dependencies before dependents), followed by directSrcs.
func DiscoverGraphFiles(deps []string, directSrcs []string, workingDir string) ([]string, error) {
	if workingDir == "" {
		workingDir = "."
	}

	mods, err := discoverAllLibraries(deps, workingDir)
	if err != nil {
		return nil, err
	}

	orderedMods := topoSortLibraries(mods)

	var result []string
	seenLogical := make(map[string]bool)
	seenContentKey := make(map[string]bool)

	addFile := func(logicalKey, path string) {
		cleanPath := filepath.Clean(path)
		if seenLogical[logicalKey] || seenLogical[cleanPath] {
			return
		}
		if _, err := os.Stat(cleanPath); err != nil {
			return
		}
		if hash, err := fileContentKey(filepath.Base(cleanPath), cleanPath); err == nil {
			if seenContentKey[hash] && logicalKey == cleanPath {
				return
			}
			seenContentKey[hash] = true
		}
		seenLogical[logicalKey] = true
		seenLogical[cleanPath] = true
		result = append(result, cleanPath)
	}

	for _, mod := range orderedMods {
		for _, f := range mod.meta.Files {
			logical := mod.meta.Name + "/" + f
			addFile(logical, filepath.Join(mod.baseDir, f))
		}
	}

	for _, src := range directSrcs {
		if strings.HasSuffix(strings.ToLower(src), ".css") {
			addFile(filepath.Clean(src), src)
		}
	}

	return result, nil
}

type discoveredLibrary struct {
	key     string
	meta    Metadata
	baseDir string
}

func discoverAllLibraries(deps []string, workingDir string) (map[string]*discoveredLibrary, error) {
	metaFiles := collectMetadataPaths(deps, workingDir)
	sort.Strings(metaFiles)

	// Prefer outermost module directories so embedded copies inside another library's output
	// do not shadow the canonical module directory when both exist.
	sort.SliceStable(metaFiles, func(i, j int) bool {
		return strings.Count(metaFiles[i], string(filepath.Separator)) < strings.Count(metaFiles[j], string(filepath.Separator))
	})

	byKey := make(map[string]*discoveredLibrary)
	for _, metaPath := range metaFiles {
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta Metadata
		if err := json.Unmarshal(data, &meta); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", metaPath, err)
		}

		baseDir := filepath.Clean(filepath.Dir(metaPath))
		key := meta.Name
		if key == "" {
			key = baseDir
		}
		if _, exists := byKey[key]; exists {
			continue
		}

		sort.Strings(meta.Deps)
		byKey[key] = &discoveredLibrary{
			key:     key,
			meta:    meta,
			baseDir: baseDir,
		}
	}

	return byKey, nil
}

func discoverDirectDepNames(deps []string, workingDir string) []string {
	seen := make(map[string]bool)
	var names []string

	for _, metaPath := range collectMetadataPaths(deps, workingDir) {
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta Metadata
		if err := json.Unmarshal(data, &meta); err == nil && meta.Name != "" && !seen[meta.Name] {
			seen[meta.Name] = true
			names = append(names, meta.Name)
		}
	}
	sort.Strings(names)
	return names
}

func collectMetadataPaths(deps []string, workingDir string) []string {
	seen := make(map[string]bool)
	var paths []string

	addPath := func(p string) {
		clean := filepath.Clean(p)
		if !seen[clean] {
			seen[clean] = true
			paths = append(paths, clean)
		}
	}

	for _, d := range deps {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if filepath.Base(d) == MetadataFile {
			addPath(d)
			continue
		}
		metaCandidate := filepath.Join(d, MetadataFile)
		if _, err := os.Stat(metaCandidate); err == nil {
			addPath(metaCandidate)
		}
	}

	_ = filepath.WalkDir(workingDir, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && d.Name() == MetadataFile {
			addPath(path)
		}
		return nil
	})

	return paths
}

func topoSortLibraries(mods map[string]*discoveredLibrary) []*discoveredLibrary {
	keys := make([]string, 0, len(mods))
	for k := range mods {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	visited := make(map[string]bool, len(mods))
	var ordered []*discoveredLibrary

	var visit func(k string)
	visit = func(k string) {
		if visited[k] {
			return
		}
		visited[k] = true
		mod, ok := mods[k]
		if !ok {
			return
		}
		for _, depKey := range mod.meta.Deps {
			visit(depKey)
		}
		ordered = append(ordered, mod)
	}

	for _, k := range keys {
		visit(k)
	}
	return ordered
}

func embedDependencies(root, outDir string) error {
	absOut, err := filepath.Abs(outDir)
	if err != nil {
		return err
	}

	var libDirs []string
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		abs, absErr := filepath.Abs(path)
		if absErr != nil {
			return nil
		}
		if d.IsDir() && abs == absOut {
			return filepath.SkipDir
		}
		if !d.IsDir() && d.Name() == MetadataFile {
			libDirs = append(libDirs, filepath.Dir(path))
		}
		return nil
	})
	if err != nil {
		return err
	}

	sort.Strings(libDirs)
	var outermost []string
	for _, dir := range libDirs {
		if n := len(outermost); n > 0 && strings.HasPrefix(dir+string(filepath.Separator), outermost[n-1]+string(filepath.Separator)) {
			continue
		}
		outermost = append(outermost, dir)
	}

	for _, dir := range outermost {
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return err
		}
		dst := filepath.Join(outDir, rel)
		if err := copyLibraryFiles(dir, dst); err != nil {
			return fmt.Errorf("embedding CSS dependency %s: %w", dir, err)
		}
	}
	return nil
}

func copyLibraryFiles(src, dst string) error {
	data, err := os.ReadFile(filepath.Join(src, MetadataFile))
	if err != nil {
		return err
	}
	var meta Metadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return err
	}
	for _, name := range append([]string{MetadataFile}, meta.Files...) {
		if err := linkOrCopy(filepath.Join(src, name), filepath.Join(dst, name)); err != nil {
			return err
		}
	}
	return nil
}

func linkOrCopy(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	if err := os.Link(src, dst); err == nil {
		return nil
	}
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func fileContentKey(baseName, path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return baseName + ":" + hex.EncodeToString(sum[:]), nil
}

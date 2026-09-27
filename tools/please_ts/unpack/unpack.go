package unpack

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"tools/please_ts/importmap"
)

// Options holds configuration for unpacking an archive or package.
type Options struct {
	Archive           string // path to tarball or zip file
	Tarball           string // legacy flag alias for archive
	Out               string // destination output directory
	Name              string // module name
	Binary            string // optional binary name to extract in toolchain mode
	Symlink           string // optional symlink name for extracted binary
	ResolveTransitive bool   // recursively resolve and unpack transitive dependencies
	Registry          string // npm registry URL (default https://registry.npmjs.org)
}

// Validate checks whether the required options are provided.
func (o Options) Validate() error {
	if o.ArchivePath() == "" {
		return fmt.Errorf("archive path must be specified")
	}
	if o.Out == "" {
		return fmt.Errorf("output directory must be specified")
	}
	return nil
}

// ArchivePath returns the primary archive path or fallback tarball path.
func (o Options) ArchivePath() string {
	if o.Archive != "" {
		return o.Archive
	}
	return o.Tarball
}

// RegistryURL returns the configured npm registry URL or default.
func (o Options) RegistryURL() string {
	if o.Registry != "" {
		return o.Registry
	}
	return "https://registry.npmjs.org"
}

// PackageJSON models relevant fields from npm package.json.
type PackageJSON struct {
	Name             string            `json:"name"`
	Version          string            `json:"version"`
	Main             string            `json:"main"`
	Module           string            `json:"module"`
	Types            string            `json:"types"`
	Typings          string            `json:"typings"`
	Exports          json.RawMessage   `json:"exports"`
	Dependencies     map[string]string `json:"dependencies"`
	PeerDependencies map[string]string `json:"peerDependencies"`
}

type npmVersionData struct {
	Version      string            `json:"version"`
	Dependencies map[string]string `json:"dependencies"`
	Dist         struct {
		Tarball string `json:"tarball"`
	} `json:"dist"`
}

type npmManifest struct {
	Name     string `json:"name"`
	DistTags struct {
		Latest string `json:"latest"`
	} `json:"dist-tags"`
	Versions map[string]npmVersionData `json:"versions"`
}

// Run executes the unpacking operation based on provided Options.
func Run(opts Options) error {
	if err := opts.Validate(); err != nil {
		return err
	}
	if opts.Binary != "" {
		return unpackToolchain(opts.ArchivePath(), opts.Out, opts.Binary, opts.Symlink)
	}
	return unpackModule(opts)
}

// unpackModule unpacks an npm module, resolves dependencies, and creates ts_module.json metadata.
func unpackModule(opts Options) error {
	if err := os.MkdirAll(opts.Out, 0755); err != nil {
		return err
	}

	if err := extractArchive(opts.ArchivePath(), opts.Out); err != nil {
		return fmt.Errorf("failed extracting archive %s: %w", opts.ArchivePath(), err)
	}

	pkg := readPackageJSON(opts.Out)
	ensureCommonJSType(opts.Out)

	depsMap := make(map[string]string)
	peerDepsMap := copyStringMap(pkg.PeerDependencies)

	if opts.ResolveTransitive && hasDependencies(pkg) {
		if err := resolveTransitiveDependencies(opts.Out, pkg, opts.RegistryURL(), depsMap); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: transitive dependency resolution failed: %v\n", err)
		}
	}

	linkAndScanLocalDeps(opts.Out, depsMap)

	return writeModuleMetadata(opts.Out, opts.Name, pkg, depsMap, peerDepsMap)
}

func readPackageJSON(dir string) PackageJSON {
	var pkg PackageJSON
	pkgJSONPath := filepath.Join(dir, "package.json")
	if data, err := os.ReadFile(pkgJSONPath); err == nil {
		_ = json.Unmarshal(data, &pkg)
	}
	return pkg
}

func hasDependencies(pkg PackageJSON) bool {
	return len(pkg.Dependencies) > 0 || len(pkg.PeerDependencies) > 0
}

func copyStringMap(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func linkAndScanLocalDeps(outDir string, depsMap map[string]string) {
	depsDir := filepath.Join(outDir, ".deps")
	info, err := os.Stat(depsDir)
	if err != nil || !info.IsDir() {
		return
	}

	nodeModulesLink := filepath.Join(outDir, "node_modules")
	_ = os.Remove(nodeModulesLink)
	_ = os.Symlink(".deps", nodeModulesLink)
	scanLocalDeps(depsDir, outDir, depsMap)
}

func writeModuleMetadata(outDir, customName string, pkg PackageJSON, depsMap, peerDepsMap map[string]string) error {
	modName := customName
	if modName == "" {
		modName = pkg.Name
	}

	entry := determineEntry(outDir, pkg)
	types := pkg.Types
	if types == "" {
		types = pkg.Typings
	}

	meta := importmap.ModuleMetadata{
		Name:     modName,
		Version:  pkg.Version,
		Entry:    entry,
		Types:    types,
		Deps:     depsMap,
		PeerDeps: peerDepsMap,
	}

	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(outDir, "ts_module.json"), metaBytes, 0644)
}

func cleanRelativePath(path string) string {
	clean := filepath.Clean(path)
	clean = filepath.ToSlash(clean)
	if !strings.HasPrefix(clean, "./") && !strings.HasPrefix(clean, "../") && !strings.HasPrefix(clean, "/") {
		clean = "./" + clean
	}
	return clean
}

func scanLocalDeps(depsDir, outDir string, depsMap map[string]string) {
	entries, err := os.ReadDir(depsDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pkgName := e.Name()
		pkgDir := filepath.Join(depsDir, pkgName)
		if strings.HasPrefix(pkgName, "@") {
			scanScopedDeps(pkgDir, pkgName, depsMap)
			continue
		}
		entry := findLocalEntry(pkgDir)
		if entry != "" {
			depsMap[pkgName] = cleanRelativePath(filepath.Join(".deps", pkgName, entry))
		}
	}
}

func scanScopedDeps(pkgDir, pkgName string, depsMap map[string]string) {
	subEntries, err := os.ReadDir(pkgDir)
	if err != nil {
		return
	}
	for _, sub := range subEntries {
		if !sub.IsDir() {
			continue
		}
		scopedPkg := pkgName + "/" + sub.Name()
		scopedDir := filepath.Join(pkgDir, sub.Name())
		entry := findLocalEntry(scopedDir)
		if entry != "" {
			depsMap[scopedPkg] = cleanRelativePath(filepath.Join(".deps", scopedPkg, entry))
		}
	}
}

func findLocalEntry(pkgDir string) string {
	pkgJSONPath := filepath.Join(pkgDir, "package.json")
	if data, err := os.ReadFile(pkgJSONPath); err == nil {
		var p PackageJSON
		if json.Unmarshal(data, &p) == nil {
			return determineEntry(pkgDir, p)
		}
	}
	candidates := []string{"index.mjs", "index.js", "mod.ts", "index.ts", "dist/index.mjs", "dist/index.js"}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(pkgDir, c)); err == nil {
			return c
		}
	}
	return ""
}

func resolveTransitiveDependencies(outDir string, rootPkg PackageJSON, registry string, depsMap map[string]string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	visited := make(map[string]bool)
	visited[rootPkg.Name] = true

	queue := make(map[string]string)
	for k, v := range rootPkg.Dependencies {
		queue[k] = v
	}
	for k, v := range rootPkg.PeerDependencies {
		queue[k] = v
	}

	for len(queue) > 0 {
		var currentPkg, currentConstraint string
		for k, v := range queue {
			currentPkg = k
			currentConstraint = v
			delete(queue, k)
			break
		}

		if visited[currentPkg] {
			continue
		}
		visited[currentPkg] = true

		destDir := filepath.Join(outDir, ".deps", currentPkg)
		if _, err := os.Stat(filepath.Join(destDir, "package.json")); err == nil {
			entry := findLocalEntry(destDir)
			if entry != "" {
				depsMap[currentPkg] = cleanRelativePath(filepath.Join(".deps", currentPkg, entry))
			}
			continue
		}

		pkgInfo, err := fetchAndExtractPackage(client, registry, currentPkg, currentConstraint, destDir)
		if err != nil {
			return fmt.Errorf("failed fetching package %s: %w", currentPkg, err)
		}

		entry := determineEntry(destDir, *pkgInfo)
		if entry != "" {
			depsMap[currentPkg] = cleanRelativePath(filepath.Join(".deps", currentPkg, entry))
		}

		for nextDep, nextVer := range pkgInfo.Dependencies {
			if !visited[nextDep] {
				queue[nextDep] = nextVer
			}
		}
	}

	return nil
}

func fetchAndExtractPackage(client *http.Client, registry, pkgName, constraint, destDir string) (*PackageJSON, error) {
	manifest, err := fetchPackageManifest(client, registry, pkgName)
	if err != nil {
		return nil, err
	}

	resolvedVer := resolveVersion(manifest.DistTags.Latest, manifest.Versions, constraint)
	verData, ok := manifest.Versions[resolvedVer]
	if !ok {
		return nil, fmt.Errorf("version %s not found in manifest for %s", resolvedVer, pkgName)
	}

	tarballURL := verData.Dist.Tarball
	if tarballURL == "" {
		base := pkgName
		if strings.Contains(base, "/") {
			base = base[strings.LastIndex(base, "/")+1:]
		}
		tarballURL = fmt.Sprintf("%s/%s/-/%s-%s.tgz", strings.TrimSuffix(registry, "/"), pkgName, base, resolvedVer)
	}

	tarballFile, err := os.CreateTemp("", "please_ts_dep_*.tgz")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tarballFile.Name())
	defer tarballFile.Close()

	if err := downloadTarball(client, tarballURL, tarballFile); err != nil {
		return nil, err
	}
	_ = tarballFile.Close()

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, err
	}

	if err := extractTarGz(tarballFile.Name(), destDir); err != nil {
		return nil, fmt.Errorf("failed extracting %s: %w", tarballFile.Name(), err)
	}

	pkg := readPackageJSON(destDir)
	if pkg.Version == "" {
		pkg.Version = resolvedVer
	}
	ensureCommonJSType(destDir)

	return &pkg, nil
}

func fetchPackageManifest(client *http.Client, registry, pkgName string) (*npmManifest, error) {
	manifestURL := strings.TrimSuffix(registry, "/") + "/" + url.PathEscape(pkgName)
	req, err := http.NewRequest("GET", manifestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.npm.install-v1+json; q=1.0, application/json; q=0.8, */*")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned status %d for %s", resp.StatusCode, manifestURL)
	}

	var manifest npmManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed decoding manifest for %s: %w", pkgName, err)
	}
	return &manifest, nil
}

func downloadTarball(client *http.Client, tarballURL string, destFile *os.File) error {
	dlResp, err := client.Get(tarballURL)
	if err != nil {
		return fmt.Errorf("failed downloading tarball from %s: %w", tarballURL, err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed downloading tarball from %s: status %d", tarballURL, dlResp.StatusCode)
	}

	_, err = io.Copy(destFile, dlResp.Body)
	return err
}

func ensureCommonJSType(dir string) {
	pkgJSONPath := filepath.Join(dir, "package.json")
	data, err := os.ReadFile(pkgJSONPath)
	if err != nil {
		return
	}
	var rawPkg map[string]interface{}
	if err := json.Unmarshal(data, &rawPkg); err != nil {
		return
	}
	if t, ok := rawPkg["type"].(string); !ok || t == "" {
		rawPkg["type"] = "commonjs"
		if updated, err := json.MarshalIndent(rawPkg, "", "  "); err == nil {
			_ = os.WriteFile(pkgJSONPath, updated, 0644)
		}
	}
}

func resolveVersion(latest string, versions map[string]npmVersionData, constraint string) string {
	clean := strings.TrimSpace(constraint)
	if clean == "" || clean == "*" || clean == "latest" {
		if latest != "" {
			return latest
		}
	}
	if _, ok := versions[clean]; ok {
		return clean
	}

	prefix := ""
	if strings.HasPrefix(clean, "^") || strings.HasPrefix(clean, "~") {
		prefix = clean[:1]
		clean = clean[1:]
	} else if strings.HasPrefix(clean, ">=") {
		prefix = ">="
		clean = strings.TrimSpace(clean[2:])
	}

	targetParts := parseSemver(clean)
	var bestMatch string
	var bestParts [3]int

	for ver := range versions {
		parts := parseSemver(ver)
		if matchesConstraint(parts, targetParts, prefix) {
			if bestMatch == "" || compareSemver(parts, bestParts) > 0 {
				bestMatch = ver
				bestParts = parts
			}
		}
	}

	if bestMatch != "" {
		return bestMatch
	}
	if latest != "" {
		return latest
	}
	for ver := range versions {
		return ver
	}
	return constraint
}

func matchesConstraint(parts, targetParts [3]int, prefix string) bool {
	switch prefix {
	case "^":
		return parts[0] == targetParts[0] && compareSemver(parts, targetParts) >= 0
	case "~":
		return parts[0] == targetParts[0] && parts[1] == targetParts[1] && compareSemver(parts, targetParts) >= 0
	case ">=":
		return compareSemver(parts, targetParts) >= 0
	default:
		return compareSemver(parts, targetParts) == 0
	}
}

func parseSemver(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	if idx := strings.IndexAny(v, "-+"); idx != -1 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	var res [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		n, _ := strconv.Atoi(parts[i])
		res[i] = n
	}
	return res
}

func compareSemver(a, b [3]int) int {
	for i := 0; i < 3; i++ {
		if a[i] > b[i] {
			return 1
		}
		if a[i] < b[i] {
			return -1
		}
	}
	return 0
}

func extractArchive(archivePath, destDir string) error {
	if strings.HasSuffix(archivePath, ".zip") {
		return extractZip(archivePath, destDir)
	}
	return extractTarGz(archivePath, destDir)
}

func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		clean := filepath.Clean(f.Name)
		if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
			continue
		}
		targetPath := filepath.Join(destDir, clean)

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		mode := f.Mode()
		if mode == 0 {
			mode = 0644
		}
		outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, mode)
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func unpackToolchain(archivePath, outDir, binaryName, symlinkName string) error {
	tmpDir, err := os.MkdirTemp("", "please_ts_toolchain_*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	if err := extractArchive(archivePath, tmpDir); err != nil {
		return fmt.Errorf("failed extracting toolchain archive %s: %w", archivePath, err)
	}

	binaryPath, err := findBinaryInDir(tmpDir, binaryName)
	if err != nil {
		return fmt.Errorf("binary %q not found in archive %s: %w", binaryName, archivePath, err)
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	binDir := filepath.Dir(binaryPath)
	if err := stageToolchainFiles(binDir, outDir, binaryName, symlinkName); err != nil {
		return err
	}

	return nil
}

func findBinaryInDir(rootDir, binaryName string) (string, error) {
	var binaryPath string
	_ = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && (info.Name() == binaryName || filepath.Base(path) == binaryName) {
			binaryPath = path
			return io.EOF
		}
		return nil
	})
	if binaryPath == "" {
		return "", fmt.Errorf("binary not found")
	}
	return binaryPath, nil
}

func stageToolchainFiles(binDir, outDir, binaryName, symlinkName string) error {
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		src := filepath.Join(binDir, entry.Name())
		dst := filepath.Join(outDir, entry.Name())
		if err := copyPath(src, dst); err != nil {
			return fmt.Errorf("failed copying %s to %s: %w", src, dst, err)
		}
	}

	targetBin := filepath.Join(outDir, binaryName)
	_ = os.Chmod(targetBin, 0755)

	if symlinkName != "" && symlinkName != binaryName {
		symPath := filepath.Join(outDir, symlinkName)
		_ = os.Remove(symPath)
		if err := os.Symlink(binaryName, symPath); err != nil {
			_ = copyFile(targetBin, symPath)
			_ = os.Chmod(symPath, 0755)
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_RDWR|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func copyPath(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		linkTarget, err := os.Readlink(src)
		if err != nil {
			return err
		}
		_ = os.Remove(dst)
		return os.Symlink(linkTarget, dst)
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, 0755); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyPath(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	return copyFile(src, dst)
}

func extractTarGz(tarGzPath, destDir string) error {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		cleanName := strings.TrimPrefix(header.Name, "package/")
		cleanName = strings.TrimPrefix(cleanName, "./package/")
		if cleanName == "" || cleanName == "package" {
			continue
		}

		targetPath := filepath.Join(destDir, cleanName)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return err
			}
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, header.FileInfo().Mode())
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}
	return nil
}

func determineEntry(destDir string, pkg PackageJSON) string {
	if pkg.Module != "" {
		if _, err := os.Stat(filepath.Join(destDir, pkg.Module)); err == nil {
			return pkg.Module
		}
	}

	if pkg.Main != "" {
		if _, err := os.Stat(filepath.Join(destDir, pkg.Main)); err == nil {
			return pkg.Main
		}
		if _, err := os.Stat(filepath.Join(destDir, pkg.Main+".js")); err == nil {
			return pkg.Main + ".js"
		}
	}

	candidates := []string{
		"index.mjs", "index.js", "mod.ts", "index.ts",
		"dist/index.mjs", "dist/index.js",
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(destDir, c)); err == nil {
			return c
		}
	}

	return ""
}

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"tools/common/mutation"
	"tools/common/treesitter"
)

// Options configures a mutation testing run.
type Options struct {
	Srcs              []string
	Lang              string
	Target            string
	TestCmd           string
	WorkDir           string
	StripPrefix       string
	OutReport         string
	ResultsFile       string
	MinScore          float64
	TimeoutMultiplier float64
	MutantTimeout     time.Duration
	Mutators          []string
	ListOnly          bool
	Stdout            io.Writer
	Stderr            io.Writer
}

type fileInput struct {
	absPath string
	relPath string
	langCfg *treesitter.LanguageConfig
	orig    []byte
	mode    os.FileMode
	sites   []treesitter.Site
}

// Run executes the mutation testing workflow configured by opts.
func Run(ctx context.Context, opts Options) (*mutation.Report, error) {
	opts, workDir, err := normalizeOptions(opts)
	if err != nil {
		return nil, err
	}

	allowedMutators, err := parseMutatorFilter(opts.Mutators)
	if err != nil {
		return nil, err
	}

	files, err := collectFiles(opts.Srcs, opts.Lang, workDir, opts.StripPrefix, allowedMutators)
	if err != nil {
		return nil, err
	}

	rep := initReport(opts, files)
	if opts.ListOnly {
		return runListMode(rep, files, opts)
	}

	perMutantTimeout, err := runBaseline(ctx, opts, workDir)
	if err != nil {
		return nil, err
	}

	mutants, err := evaluateAllMutants(ctx, files, opts.TestCmd, workDir, perMutantTimeout)
	if err != nil {
		return nil, err
	}
	rep.Mutants = mutants

	return finalizeReport(rep, opts)
}

func normalizeOptions(opts Options) (Options, string, error) {
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}
	if opts.TimeoutMultiplier <= 0 {
		opts.TimeoutMultiplier = 5.0
	}
	workDir, err := resolveWorkDir(opts.WorkDir)
	if err != nil {
		return opts, "", err
	}
	return opts, workDir, nil
}

func resolveWorkDir(workDir string) (string, error) {
	if workDir == "" {
		dir, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getwd: %w", err)
		}
		return dir, nil
	}
	dir, err := filepath.Abs(workDir)
	if err != nil {
		return "", fmt.Errorf("resolve work-dir: %w", err)
	}
	return dir, nil
}

func initReport(opts Options, files []fileInput) *mutation.Report {
	reportLang := opts.Lang
	if reportLang == "" && len(files) > 0 {
		reportLang = files[0].langCfg.ReportName
	}
	return &mutation.Report{
		SchemaVersion: mutation.SchemaVersion,
		Language:      reportLang,
		Target:        opts.Target,
		MinScore:      opts.MinScore,
	}
}

func runListMode(rep *mutation.Report, files []fileInput, opts Options) (*mutation.Report, error) {
	idx := 0
	for _, f := range files {
		for _, s := range f.sites {
			idx++
			m := siteToMutant(fmt.Sprintf("m%d", idx), s, mutation.Skipped, 0)
			rep.Mutants = append(rep.Mutants, m)
			fmt.Fprintf(opts.Stdout, "[%s] %s:%d:%d (%s) %s -> %s\n",
				m.ID, m.File, m.Line, m.Col, m.Mutator, m.Original, m.Replacement)
		}
	}
	rep.Recompute()
	if err := writeOutputs(rep, opts.OutReport, opts.ResultsFile); err != nil {
		return rep, err
	}
	return rep, nil
}

func runBaseline(ctx context.Context, opts Options, workDir string) (time.Duration, error) {
	if strings.TrimSpace(opts.TestCmd) == "" {
		return 0, fmt.Errorf("-test-cmd is required unless -list is set")
	}
	baseStart := time.Now()
	baseOut, baseStatus, err := runTestCommand(ctx, opts.TestCmd, workDir, 0)
	baseDur := time.Since(baseStart)
	if err != nil || baseStatus != mutation.Survived {
		return 0, fmt.Errorf("baseline test command failed before applying any mutations (%v):\n%s", err, baseOut)
	}
	return computeMutantTimeout(baseDur, opts), nil
}

func computeMutantTimeout(baseDur time.Duration, opts Options) time.Duration {
	if opts.MutantTimeout > 0 {
		return opts.MutantTimeout
	}
	scaled := time.Duration(float64(baseDur) * opts.TimeoutMultiplier)
	if scaled < 5*time.Second {
		return 5 * time.Second
	}
	return scaled
}

func evaluateAllMutants(
	ctx context.Context,
	files []fileInput,
	testCmd string,
	workDir string,
	timeout time.Duration,
) ([]mutation.Mutant, error) {
	var all []mutation.Mutant
	for _, f := range files {
		fileMutants, err := evaluateFileMutants(ctx, f, len(all), testCmd, workDir, timeout)
		if err != nil {
			return nil, err
		}
		all = append(all, fileMutants...)
	}
	return all, nil
}

func evaluateFileMutants(
	ctx context.Context,
	f fileInput,
	startIdx int,
	testCmd string,
	workDir string,
	timeout time.Duration,
) ([]mutation.Mutant, error) {
	out := make([]mutation.Mutant, 0, len(f.sites))
	for i, s := range f.sites {
		id := fmt.Sprintf("m%d", startIdx+i+1)
		mutatedSrc := treesitter.Apply(f.orig, s)
		status, durMs, err := evaluateMutant(ctx, f, mutatedSrc, testCmd, workDir, timeout)
		if err != nil {
			return nil, err
		}
		out = append(out, siteToMutant(id, s, status, durMs))
	}
	return out, nil
}

func finalizeReport(rep *mutation.Report, opts Options) (*mutation.Report, error) {
	rep.Recompute()
	_ = rep.WriteSummary(opts.Stdout)
	if err := writeOutputs(rep, opts.OutReport, opts.ResultsFile); err != nil {
		return rep, err
	}
	if err := rep.CheckThreshold(opts.MinScore); err != nil {
		return rep, err
	}
	return rep, nil
}

func evaluateMutant(
	ctx context.Context,
	f fileInput,
	mutatedSrc []byte,
	testCmd string,
	workDir string,
	timeout time.Duration,
) (mutation.Status, int64, error) {
	if err := os.WriteFile(f.absPath, mutatedSrc, f.mode); err != nil {
		return "", 0, fmt.Errorf("write mutated %s: %w", f.relPath, err)
	}
	defer func() {
		_ = os.WriteFile(f.absPath, f.orig, f.mode)
	}()

	start := time.Now()
	_, status, _ := runTestCommand(ctx, testCmd, workDir, timeout)
	durMs := time.Since(start).Milliseconds()

	if err := os.WriteFile(f.absPath, f.orig, f.mode); err != nil {
		return "", durMs, fmt.Errorf("restore original %s: %w", f.relPath, err)
	}
	return status, durMs, nil
}

func runTestCommand(parentCtx context.Context, testCmd, workDir string, timeout time.Duration) (string, mutation.Status, error) {
	ctx := parentCtx
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(parentCtx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", testCmd)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(out), mutation.Timeout, ctx.Err()
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return string(out), mutation.Killed, err
		}
		return string(out), mutation.Unviable, err
	}
	return string(out), mutation.Survived, nil
}

func siteToMutant(id string, s treesitter.Site, status mutation.Status, durMs int64) mutation.Mutant {
	return mutation.Mutant{
		ID:          id,
		File:        s.File,
		Line:        s.Line,
		Col:         s.Col,
		EndLine:     s.EndLine,
		EndCol:      s.EndCol,
		Mutator:     string(s.Mutator),
		Function:    s.Function,
		Original:    s.Original,
		Replacement: s.Replacement,
		Status:      status,
		DurationMs:  durMs,
	}
}

func parseMutatorFilter(specs []string) (map[treesitter.MutatorCategory]bool, error) {
	if len(specs) == 0 {
		return nil, nil
	}
	valid := validMutatorSet()
	allowed := make(map[treesitter.MutatorCategory]bool)
	for _, part := range splitCommaSpecs(specs) {
		cat := treesitter.MutatorCategory(part)
		if !valid[cat] {
			return nil, fmt.Errorf("unknown mutator category %q", part)
		}
		allowed[cat] = true
	}
	return allowed, nil
}

func validMutatorSet() map[treesitter.MutatorCategory]bool {
	valid := make(map[treesitter.MutatorCategory]bool, len(treesitter.AllMutatorCategories))
	for _, c := range treesitter.AllMutatorCategories {
		valid[c] = true
	}
	return valid
}

func splitCommaSpecs(specs []string) []string {
	var out []string
	for _, spec := range specs {
		for _, part := range strings.Split(spec, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				out = append(out, trimmed)
			}
		}
	}
	return out
}

func collectFiles(
	srcs []string,
	explicitLang string,
	workDir string,
	stripPrefix string,
	allowedMutators map[treesitter.MutatorCategory]bool,
) ([]fileInput, error) {
	rawPaths, err := expandSourcePaths(srcs, explicitLang, workDir)
	if err != nil {
		return nil, err
	}
	uniquePaths, err := deduplicateSourcePaths(rawPaths)
	if err != nil {
		return nil, err
	}

	out := make([]fileInput, 0, len(uniquePaths))
	for _, absPath := range uniquePaths {
		fi, err := loadFileInput(absPath, explicitLang, workDir, stripPrefix, allowedMutators)
		if err != nil {
			return nil, err
		}
		out = append(out, fi)
	}
	return out, nil
}

func expandSourcePaths(srcs []string, explicitLang, workDir string) ([]string, error) {
	var paths []string
	for _, item := range splitCommaSpecs(srcs) {
		expanded, err := expandSourceItem(item, explicitLang, workDir)
		if err != nil {
			return nil, err
		}
		paths = append(paths, expanded...)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no source files specified (use -srcs)")
	}
	return paths, nil
}

func expandSourceItem(item, explicitLang, workDir string) ([]string, error) {
	p := item
	if !filepath.IsAbs(p) {
		p = filepath.Join(workDir, p)
	}
	info, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return walkSourceDir(p, explicitLang)
	}
	return []string{p}, nil
}

func walkSourceDir(dir, explicitLang string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(dir, func(walkPath string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if isCandidateSourceEntry(walkPath, d, explicitLang) {
			paths = append(paths, walkPath)
		}
		return nil
	})
	return paths, err
}

func isCandidateSourceEntry(walkPath string, d os.DirEntry, explicitLang string) bool {
	if d.IsDir() || treesitter.IsTestFile(walkPath) {
		return false
	}
	_, err := treesitter.ResolveLanguage(walkPath, explicitLang)
	return err == nil
}

func deduplicateSourcePaths(paths []string) ([]string, error) {
	var unique []string
	seen := make(map[string]bool, len(paths))
	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			return nil, err
		}
		if seen[absPath] || treesitter.IsTestFile(absPath) {
			continue
		}
		seen[absPath] = true
		unique = append(unique, absPath)
	}
	return unique, nil
}

func loadFileInput(
	absPath string,
	explicitLang string,
	workDir string,
	stripPrefix string,
	allowedMutators map[treesitter.MutatorCategory]bool,
) (fileInput, error) {
	langCfg, err := treesitter.ResolveLanguage(absPath, explicitLang)
	if err != nil {
		return fileInput{}, err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return fileInput{}, err
	}
	orig, err := os.ReadFile(absPath)
	if err != nil {
		return fileInput{}, err
	}
	rel := relativeReportPath(absPath, workDir, stripPrefix)
	sites, err := langCfg.Discover(rel, orig, allowedMutators)
	if err != nil {
		return fileInput{}, fmt.Errorf("%s: %w", rel, err)
	}
	return fileInput{
		absPath: absPath,
		relPath: rel,
		langCfg: langCfg,
		orig:    orig,
		mode:    info.Mode(),
		sites:   sites,
	}, nil
}

func relativeReportPath(absPath, workDir, stripPrefix string) string {
	rel := mutation.RelPath(absPath, workDir)
	if stripPrefix != "" {
		rel = strings.TrimPrefix(rel, strings.TrimSuffix(stripPrefix, "/")+"/")
	}
	return rel
}

func writeOutputs(rep *mutation.Report, outReport, resultsFile string) error {
	if err := writeFileWith(outReport, rep.WriteJSON); err != nil {
		return err
	}
	return writeFileWith(resultsFile, rep.WriteJUnitXML)
}

func writeFileWith(path string, fn func(io.Writer) error) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	writeErr := fn(f)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

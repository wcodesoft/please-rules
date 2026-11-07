package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"tools/common/mutation"
)

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

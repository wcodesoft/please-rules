package lcov

import (
	"fmt"
	"io"
	"sort"
)

// Write writes the report as an lcov tracefile. Output is deterministic: files,
// lines, functions and branches are sorted, and the summary counts (FNF, FNH, BRF,
// BRH, LF, LH) are computed from the records.
func (r *Report) Write(w io.Writer) error {
	files := append([]*File(nil), r.Files...)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })

	for _, f := range files {
		f.sortRecords()
		fmt.Fprintf(w, "TN:\nSF:%s\n", f.Path)
		for _, fn := range f.Functions {
			fmt.Fprintf(w, "FN:%d,%s\n", fn.Line, fn.Name)
		}
		for _, fn := range f.Functions {
			fmt.Fprintf(w, "FNDA:%d,%s\n", fn.Hits, fn.Name)
		}
		fmt.Fprintf(w, "FNF:%d\nFNH:%d\n", len(f.Functions), f.FunctionsHit())
		for _, b := range f.Branches {
			if b.Taken == NotEvaluated {
				fmt.Fprintf(w, "BRDA:%d,%d,%d,-\n", b.Line, b.Block, b.Arm)
			} else {
				fmt.Fprintf(w, "BRDA:%d,%d,%d,%d\n", b.Line, b.Block, b.Arm, b.Taken)
			}
		}
		fmt.Fprintf(w, "BRF:%d\nBRH:%d\n", len(f.Branches), f.BranchesHit())
		lines := make([]int, 0, len(f.Lines))
		for n := range f.Lines {
			lines = append(lines, n)
		}
		sort.Ints(lines)
		for _, n := range lines {
			fmt.Fprintf(w, "DA:%d,%d\n", n, f.Lines[n])
		}
		if _, err := fmt.Fprintf(w, "LF:%d\nLH:%d\nend_of_record\n", f.LinesFound(), f.LinesHit()); err != nil {
			return err
		}
	}
	return nil
}

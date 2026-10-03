package lcov

import (
	"encoding/json"
	"fmt"
	"io"
)

// AddRegionBranches adds synthesized branch records to report from the sub-line
// regions in the JSON export of llvm-cov (`llvm-cov export -format=text`).
//
// llvm-cov's lcov export only has BRDA records when the compiler emitted branch
// regions, which stable Rust and Swift do not. The JSON export always has
// segments, which show that part of a line never ran: the right-hand side of
// `a && b`, the body of a one-line `if`, a closure that was never called.
//
// A region entry that begins on a line where another region is already open (or
// that is not the first region of the line) is a sub-line region. Each becomes one
// arm of the line, numbered in column order with its execution count. Records are
// only added to lines that ran and have at least one arm that did not: a line whose
// sub-line regions all ran needs no branch record to be classified as covered, and
// branch-level information the compiler already provided (existing BRDA records on
// the line) is left alone. The arms are therefore a heuristic, not source-level
// branches: a branch whose arms are on separate lines shows up as an uncovered
// line, not as a partial one.
//
// File names in the export are mapped with normalize (nil keeps them) before
// being matched against the report.
func AddRegionBranches(report *Report, llvmJSON io.Reader, normalize func(string) string) error {
	var export struct {
		Data []struct {
			Files []struct {
				Filename string  `json:"filename"`
				Segments [][]any `json:"segments"`
			} `json:"files"`
		} `json:"data"`
	}
	if err := json.NewDecoder(llvmJSON).Decode(&export); err != nil {
		return fmt.Errorf("parsing llvm-cov export: %w", err)
	}

	for _, d := range export.Data {
		for _, ef := range d.Files {
			name := ef.Filename
			if normalize != nil {
				name = normalize(name)
			}
			file := report.File(name)
			if file == nil {
				continue
			}
			segs, err := parseSegments(ef.Segments)
			if err != nil {
				return fmt.Errorf("%s: %w", ef.Filename, err)
			}
			addFileRegionBranches(file, segs)
		}
	}
	for _, f := range report.Files {
		f.sortRecords()
	}
	return nil
}

type segment struct {
	line, col  int
	count      int
	hasCount   bool
	regionHead bool
	gap        bool
}

func parseSegments(raw [][]any) ([]segment, error) {
	out := make([]segment, 0, len(raw))
	for _, r := range raw {
		if len(r) < 6 {
			return nil, fmt.Errorf("segment has %d fields, want 6", len(r))
		}
		num := func(v any) int { f, _ := v.(float64); return int(f) }
		flag := func(v any) bool { b, _ := v.(bool); return b }
		out = append(out, segment{
			line: num(r[0]), col: num(r[1]), count: num(r[2]),
			hasCount: flag(r[3]), regionHead: flag(r[4]), gap: flag(r[5]),
		})
	}
	return out, nil
}

func addFileRegionBranches(file *File, segs []segment) {
	hasBranches := make(map[int]bool)
	for _, b := range file.Branches {
		hasBranches[b.Line] = true
	}

	// open reports whether a region with a count is open at the start of a line:
	// the last segment before the line has one.
	open := false
	for i := 0; i < len(segs); {
		line := segs[i].line
		j := i
		for j < len(segs) && segs[j].line == line {
			j++
		}

		var entries []segment
		for k := i; k < j; k++ {
			s := segs[k]
			if s.regionHead && s.hasCount && !s.gap {
				// The first region head of a line that no open region covers is the
				// line's own region, not a sub-line one.
				if !open && k == i {
					continue
				}
				entries = append(entries, s)
			}
		}

		if hits, ran := file.Lines[line]; ran && hits > 0 && !hasBranches[line] {
			missed := false
			for _, e := range entries {
				if e.count == 0 {
					missed = true
				}
			}
			if missed {
				for arm, e := range entries {
					file.Branches = append(file.Branches, Branch{Line: line, Block: 0, Arm: arm, Taken: e.count})
				}
			}
		}

		open = segs[j-1].hasCount
		i = j
	}
}

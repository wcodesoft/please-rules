// Package lcov reads, merges and writes lcov tracefiles with line, function and
// branch data.
//
// It is the shared exchange format for the language rule sets: each test runner
// can export the raw report of its coverage tool, and consumers (dashboards,
// annotated source views) read it back, merging reports from several test targets.
package lcov

import "sort"

// Report is a set of per-file coverage records.
type Report struct {
	Files []*File
}

// File holds the coverage of one source file. Paths are slash-separated.
type File struct {
	Path      string
	Lines     map[int]int // line number -> execution count
	Functions []Function
	Branches  []Branch
}

// Function is a function record (FN and FNDA).
type Function struct {
	Name string
	Line int
	Hits int
}

// NotEvaluated is the Branch.Taken value for a branch whose enclosing block never
// ran (written as "-" in lcov).
const NotEvaluated = -1

// Branch is one arm of a branch point (BRDA).
type Branch struct {
	Line  int
	Block int
	Arm   int
	Taken int // execution count, or NotEvaluated
}

// Status is the coverage state of a single source line.
type Status int

const (
	// NotExecutable lines have no coverage record (blank lines, comments, ...).
	NotExecutable Status = iota
	// Uncovered lines are executable but never ran.
	Uncovered
	// Partial lines ran but have a branch arm that never did.
	Partial
	// Covered lines ran and every recorded branch arm was taken.
	Covered
)

func (s Status) String() string {
	switch s {
	case Uncovered:
		return "uncovered"
	case Partial:
		return "partial"
	case Covered:
		return "covered"
	}
	return "not-executable"
}

// newFile returns an empty file record for path.
func newFile(path string) *File {
	return &File{Path: path, Lines: make(map[int]int)}
}

// File returns the record for path, or nil.
func (r *Report) File(path string) *File {
	for _, f := range r.Files {
		if f.Path == path {
			return f
		}
	}
	return nil
}

// LineStatus classifies a line. A line with a count of zero is Uncovered even if it
// has branch records; a line that ran is Partial when any of its branch arms was not
// taken, and Covered otherwise.
func (f *File) LineStatus(line int) Status {
	hits, ok := f.Lines[line]
	if !ok {
		return NotExecutable
	}
	if hits == 0 {
		return Uncovered
	}
	for _, b := range f.Branches {
		if b.Line == line && b.Taken <= 0 {
			return Partial
		}
	}
	return Covered
}

// LinesFound is the number of executable lines (LF).
func (f *File) LinesFound() int { return len(f.Lines) }

// LinesHit is the number of executable lines that ran (LH).
func (f *File) LinesHit() int {
	n := 0
	for _, h := range f.Lines {
		if h > 0 {
			n++
		}
	}
	return n
}

// FunctionsHit is the number of functions that ran (FNH).
func (f *File) FunctionsHit() int {
	n := 0
	for _, fn := range f.Functions {
		if fn.Hits > 0 {
			n++
		}
	}
	return n
}

// BranchesHit is the number of branch arms taken at least once (BRH).
func (f *File) BranchesHit() int {
	n := 0
	for _, b := range f.Branches {
		if b.Taken > 0 {
			n++
		}
	}
	return n
}

func (f *File) sortRecords() {
	sort.Slice(f.Functions, func(i, j int) bool {
		a, b := f.Functions[i], f.Functions[j]
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.Name < b.Name
	})
	sort.Slice(f.Branches, func(i, j int) bool {
		a, b := f.Branches[i], f.Branches[j]
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Block != b.Block {
			return a.Block < b.Block
		}
		return a.Arm < b.Arm
	})
}

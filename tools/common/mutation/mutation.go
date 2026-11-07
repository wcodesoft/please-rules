// Package mutation defines the common mutation testing report schema, JUnit XML
// output, console summary formatter, and score threshold checks shared across
// all language rule sets.
package mutation

import (
	"fmt"
	"math"
	"sort"
)

// SchemaVersion is the current version of the .mutation.json schema.
const SchemaVersion = 1

// Status classifies the outcome of evaluating a single mutant.
type Status string

const (
	// Killed mutants caused at least one test to fail.
	Killed Status = "killed"
	// Survived mutants left all tests passing.
	Survived Status = "survived"
	// Timeout mutants exceeded the allowed per-mutant test execution duration.
	Timeout Status = "timeout"
	// Unviable mutants failed to compile or load.
	Unviable Status = "unviable"
	// Skipped mutants were not evaluated (e.g. filtered or dry-run).
	Skipped Status = "skipped"
)

// ValidStatus reports whether s is a known mutant status.
func ValidStatus(s Status) bool {
	switch s {
	case Killed, Survived, Timeout, Unviable, Skipped:
		return true
	}
	return false
}

// Mutant represents a single source mutation and its test evaluation outcome.
type Mutant struct {
	ID          string `json:"id"`
	File        string `json:"file"`
	Line        int    `json:"line"`
	Col         int    `json:"col"`
	EndLine     int    `json:"end_line,omitempty"`
	EndCol      int    `json:"end_col,omitempty"`
	Mutator     string `json:"mutator"`
	Function    string `json:"function,omitempty"`
	Original    string `json:"original"`
	Replacement string `json:"replacement"`
	Status      Status `json:"status"`
	DurationMs  int64  `json:"duration_ms"`
}

// Report is the shared JSON exchange format (<target>.mutation.json) produced by
// mutation test targets across all language rule sets.
type Report struct {
	SchemaVersion int      `json:"schema_version"`
	Language      string   `json:"language"`
	Target        string   `json:"target"`
	MinScore      float64  `json:"min_score"`
	Total         int      `json:"total"`
	Killed        int      `json:"killed"`
	Survived      int      `json:"survived"`
	Timeout       int      `json:"timeout"`
	Unviable      int      `json:"unviable"`
	Skipped       int      `json:"skipped,omitempty"`
	Score         float64  `json:"score"`
	Mutants       []Mutant `json:"mutants"`
}

// Recompute sorts the mutants deterministically, assigns sequential IDs if empty,
// and recalculates summary counts and the mutation score percentage.
//
// Following standard mutation testing semantics (and issue #106):
//
//	detected = killed + timeout
//	evaluated = killed + survived + timeout
//	score = (detected / evaluated) * 100
//
// If no mutants were evaluated (evaluated == 0), Score is 100.0.
func (r *Report) Recompute() {
	if r.SchemaVersion == 0 {
		r.SchemaVersion = SchemaVersion
	}
	if r.Mutants == nil {
		r.Mutants = []Mutant{}
	}

	sort.Slice(r.Mutants, func(i, j int) bool {
		a, b := r.Mutants[i], r.Mutants[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.Col != b.Col {
			return a.Col < b.Col
		}
		if a.Mutator != b.Mutator {
			return a.Mutator < b.Mutator
		}
		if a.Original != b.Original {
			return a.Original < b.Original
		}
		return a.Replacement < b.Replacement
	})

	r.Total = 0
	r.Killed = 0
	r.Survived = 0
	r.Timeout = 0
	r.Unviable = 0
	r.Skipped = 0

	for i := range r.Mutants {
		if r.Mutants[i].ID == "" {
			r.Mutants[i].ID = fmt.Sprintf("m%d", i+1)
		}
		switch r.Mutants[i].Status {
		case Killed:
			r.Killed++
		case Survived:
			r.Survived++
		case Timeout:
			r.Timeout++
		case Unviable:
			r.Unviable++
		case Skipped:
			r.Skipped++
		}
	}

	evaluated := r.Killed + r.Survived + r.Timeout
	r.Total = evaluated
	if evaluated == 0 {
		r.Score = 100.0
	} else {
		raw := float64(r.Killed+r.Timeout) / float64(evaluated) * 100.0
		r.Score = math.Round(raw*100) / 100
	}
}

// CheckSurvived returns an error if any evaluated mutant survived.
func (r *Report) CheckSurvived() error {
	if r.Survived > 0 {
		return fmt.Errorf("%d mutant(s) survived out of %d evaluated (mutation score %.2f%%)",
			r.Survived, r.Total, r.Score)
	}
	return nil
}

// CheckThreshold verifies that r.Score meets or exceeds minScore.
func (r *Report) CheckThreshold(minScore float64) error {
	if minScore <= 0 {
		return nil
	}
	if r.Score+1e-9 < minScore {
		return fmt.Errorf("mutation score %.2f%% is below minimum threshold %.2f%% (%d survived out of %d evaluated mutants)",
			r.Score, minScore, r.Survived, r.Total)
	}
	return nil
}

// FileMutants returns all mutants belonging to the given slash-separated path.
func (r *Report) FileMutants(path string) []Mutant {
	var out []Mutant
	for _, m := range r.Mutants {
		if m.File == path {
			out = append(out, m)
		}
	}
	return out
}

// Merge combines multiple mutation reports into a single report. If the same
// mutant (matching File, Line, Col, Mutator, Original, Replacement) appears in
// multiple reports, Killed takes precedence over Timeout, which takes precedence
// over Survived, which takes precedence over Unviable/Skipped.
func Merge(reports ...*Report) *Report {
	type key struct {
		file        string
		line, col   int
		mutator     string
		original    string
		replacement string
	}
	byKey := make(map[key]Mutant)
	out := &Report{SchemaVersion: SchemaVersion}

	for _, rep := range reports {
		if rep == nil {
			continue
		}
		if out.Language == "" {
			out.Language = rep.Language
		}
		if out.Target == "" {
			out.Target = rep.Target
		}
		if rep.MinScore > out.MinScore {
			out.MinScore = rep.MinScore
		}
		for _, m := range rep.Mutants {
			k := key{
				file:        m.File,
				line:        m.Line,
				col:         m.Col,
				mutator:     m.Mutator,
				original:    m.Original,
				replacement: m.Replacement,
			}
			prev, exists := byKey[k]
			if !exists || statusRank(m.Status) > statusRank(prev.Status) {
				m.ID = ""
				byKey[k] = m
			}
		}
	}

	out.Mutants = make([]Mutant, 0, len(byKey))
	for _, m := range byKey {
		out.Mutants = append(out.Mutants, m)
	}
	out.Recompute()
	return out
}

func statusRank(s Status) int {
	switch s {
	case Killed:
		return 5
	case Timeout:
		return 4
	case Survived:
		return 3
	case Unviable:
		return 2
	case Skipped:
		return 1
	default:
		return 0
	}
}

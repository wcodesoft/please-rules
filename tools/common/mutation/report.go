package mutation

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
)

// ParseJSON decodes a mutation Report from r.
func ParseJSON(r io.Reader) (*Report, error) {
	var rep Report
	dec := json.NewDecoder(r)
	if err := dec.Decode(&rep); err != nil {
		return nil, err
	}
	return &rep, nil
}

// WriteJSON writes the report as indented JSON to w after recomputing summary counts.
func (r *Report) WriteJSON(w io.Writer) error {
	r.Recompute()
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

type junitTestSuites struct {
	XMLName xml.Name         `xml:"testsuites"`
	Suites  []junitTestSuite `xml:"testsuite"`
}

type junitTestSuite struct {
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Skipped  int             `xml:"skipped,attr,omitempty"`
	Time     string          `xml:"time,attr"`
	Cases    []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
	Skipped   *junitSkipped `xml:"skipped,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}

type junitSkipped struct {
	Message string `xml:"message,attr,omitempty"`
}

// WriteJUnitXML writes a JUnit XML report suitable for Please's test.results.
// Every surviving mutant is emitted as an individual JUnit failure.
func (r *Report) WriteJUnitXML(w io.Writer) error {
	r.Recompute()

	suiteName := r.Target
	if suiteName == "" {
		suiteName = "mutation"
	}

	var totalMs int64
	var cases []junitTestCase
	failures := 0
	skipped := 0

	for _, m := range r.Mutants {
		totalMs += m.DurationMs
		tcName := fmt.Sprintf("%s:%d:%d [%s] %s -> %s", m.File, m.Line, m.Col, m.Mutator, m.Original, m.Replacement)
		className := m.File
		if m.Function != "" {
			className = fmt.Sprintf("%s.%s", m.File, m.Function)
		}
		tc := junitTestCase{
			Name:      tcName,
			ClassName: className,
			Time:      fmt.Sprintf("%.3f", float64(m.DurationMs)/1000.0),
		}

		switch m.Status {
		case Survived:
			failures++
			fnCtx := ""
			if m.Function != "" {
				fnCtx = fmt.Sprintf(" in %s", m.Function)
			}
			msg := fmt.Sprintf("condition at %s:%d:%d%s (%s) mutated %q -> %q and no test failed (no test covering this condition)",
				m.File, m.Line, m.Col, fnCtx, m.Mutator, m.Original, m.Replacement)
			tc.Failure = &junitFailure{
				Message: msg,
				Type:    "MutantSurvived",
				Body:    fmt.Sprintf("%s\nFile: %s:%d:%d\nMutator: %s\nFunction: %s\nOriginal: %s\nReplacement: %s", msg, m.File, m.Line, m.Col, m.Mutator, m.Function, m.Original, m.Replacement),
			}
		case Unviable, Skipped:
			skipped++
			tc.Skipped = &junitSkipped{Message: string(m.Status)}
		}
		cases = append(cases, tc)
	}

	if err := r.CheckThreshold(r.MinScore); err != nil {
		failures++
		cases = append(cases, junitTestCase{
			Name:      "mutation_score_threshold",
			ClassName: suiteName,
			Time:      "0.000",
			Failure: &junitFailure{
				Message: err.Error(),
				Type:    "MutationScoreBelowThreshold",
				Body:    err.Error(),
			},
		})
	}

	doc := junitTestSuites{
		Suites: []junitTestSuite{
			{
				Name:     suiteName,
				Tests:    len(cases),
				Failures: failures,
				Skipped:  skipped,
				Time:     fmt.Sprintf("%.3f", float64(totalMs)/1000.0),
				Cases:    cases,
			},
		},
	}

	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return err
	}
	_, err := io.WriteString(w, "\n")
	return err
}

// WriteSummary formats a human-readable mutation summary to w.
func (r *Report) WriteSummary(w io.Writer) error {
	r.Recompute()
	target := r.Target
	if target == "" {
		target = "(unnamed)"
	}
	if _, err := fmt.Fprintf(w, "Mutation Report for %s (%s)\n", target, r.Language); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  Total evaluated: %d (killed: %d, survived: %d, timeout: %d, unviable: %d)\n",
		r.Total, r.Killed, r.Survived, r.Timeout, r.Unviable); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "  Mutation score:  %.2f%% (min_score: %.2f%%)\n", r.Score, r.MinScore); err != nil {
		return err
	}

	if r.Survived > 0 {
		if _, err := fmt.Fprintln(w, "  Surviving mutants (code mutated, but no test failed — missing test coverage):"); err != nil {
			return err
		}
		for _, m := range r.Mutants {
			if m.Status != Survived {
				continue
			}
			fnInfo := ""
			if m.Function != "" {
				fnInfo = fmt.Sprintf(" in %s", m.Function)
			}
			if _, err := fmt.Fprintf(w, "    - [%s] %s:%d:%d%s (%s): mutated %q -> %q and no test failed\n",
				m.ID, m.File, m.Line, m.Col, fnInfo, m.Mutator, m.Original, m.Replacement); err != nil {
				return err
			}
		}
	}
	return nil
}

package mutation

import (
	"bytes"
	"strings"
	"testing"
)

func TestJSONRoundTripJUnitXMLAndSummary(t *testing.T) {
	r := &Report{
		Language: "rust",
		Target:   "//pkg:rs_mutate_test",
		MinScore: 80.0,
		Mutants: []Mutant{
			{
				File:        "src/lib.rs",
				Line:        10,
				Col:         5,
				Mutator:     "equality",
				Function:    "is_valid",
				Original:    "==",
				Replacement: "!=",
				Status:      Killed,
				DurationMs:  50,
			},
			{
				File:        "src/lib.rs",
				Line:        14,
				Col:         9,
				Mutator:     "boundary",
				Function:    "is_valid",
				Original:    "<",
				Replacement: "<=",
				Status:      Survived,
				DurationMs:  45,
			},
		},
	}

	var jsonBuf bytes.Buffer
	if err := r.WriteJSON(&jsonBuf); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	parsed, err := ParseJSON(&jsonBuf)
	if err != nil {
		t.Fatalf("ParseJSON: %v", err)
	}
	if parsed.Score != 50.0 || len(parsed.Mutants) != 2 {
		t.Fatalf("parsed = %+v", parsed)
	}

	var xmlBuf bytes.Buffer
	if err := r.WriteJUnitXML(&xmlBuf); err != nil {
		t.Fatalf("WriteJUnitXML: %v", err)
	}
	xmlStr := xmlBuf.String()
	if !strings.Contains(xmlStr, "MutantSurvived") || !strings.Contains(xmlStr, "MutationScoreBelowThreshold") {
		t.Fatalf("expected failure tags in JUnit XML, got:\n%s", xmlStr)
	}

	var summaryBuf bytes.Buffer
	if err := r.WriteSummary(&summaryBuf); err != nil {
		t.Fatalf("WriteSummary: %v", err)
	}
	if !strings.Contains(summaryBuf.String(), "Surviving mutants (code mutated, but no test failed") {
		t.Fatalf("expected surviving mutants section in summary, got:\n%s", summaryBuf.String())
	}
}

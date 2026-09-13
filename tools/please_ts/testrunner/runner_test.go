package testrunner

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFallbackJUnit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "testrunner_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	resultsFile := filepath.Join(tmpDir, "test.results")
	if err := writeFallbackJUnit(resultsFile, []string{"foo_test.ts"}, errors.New("sample error")); err != nil {
		t.Fatalf("failed to write fallback junit: %v", err)
	}

	data, err := os.ReadFile(resultsFile)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "<failure message=\"sample error\">") {
		t.Errorf("expected failure message in junit xml: %s", content)
	}
}

func TestLcovToCoberturaXML(t *testing.T) {
	lcov := `SF:/home/user/repo/test/ts/lib/calculator.ts
DA:1,1
DA:2,2
DA:5,0
end_of_record
`
	xmlBytes := lcovToCoberturaXML([]byte(lcov), "/home/user/repo")
	xmlStr := string(xmlBytes)

	if !strings.Contains(xmlStr, `<coverage>`) {
		t.Fatalf("expected coverage tag, got: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, `filename="test/ts/lib/calculator.ts"`) {
		t.Errorf("expected relative filename in class tag: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, `<line number="1" hits="1"/>`) {
		t.Errorf("expected line 1 hit in XML: %s", xmlStr)
	}
	if !strings.Contains(xmlStr, `<line number="5" hits="0"/>`) {
		t.Errorf("expected line 5 uncovered in XML: %s", xmlStr)
	}

	// Empty LCOV should yield valid empty Cobertura XML
	emptyXML := string(lcovToCoberturaXML([]byte(""), "/home/user/repo"))
	if !strings.Contains(emptyXML, "<packages/>") {
		t.Errorf("expected empty packages element, got: %s", emptyXML)
	}
}

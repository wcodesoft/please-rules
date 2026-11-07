package python

import "testing"

func TestConfig(t *testing.T) {
	src := []byte(`def can_access(is_admin, active):
    if is_admin and not active:
        return True
    return False

def test_can_access():
    assert 1 + 1 == 2
`)

	cfg := Config()
	if !cfg.IsTestFile("test_calc.py") || !cfg.IsTestFile("calc_test.py") || cfg.IsTestFile("calc.py") {
		t.Fatal("unexpected Python IsTestFile result")
	}

	sites, err := cfg.Discover("calc.py", src, nil)
	if err != nil {
		t.Fatalf("Config().Discover: %v", err)
	}
	foundLogical := false
	foundBool := false
	for _, s := range sites {
		if s.Function == "test_can_access" {
			t.Fatalf("unexpected mutation inside test_can_access: %+v", s)
		}
		if s.Function == "can_access" && s.Original == "and" && s.Replacement == "or" {
			foundLogical = true
		}
		if s.Function == "can_access" && s.Original == "True" && s.Replacement == "False" {
			foundBool = true
		}
	}
	if !foundLogical || !foundBool {
		t.Fatalf("expected Python logical and boolean mutations in can_access, got: %+v", sites)
	}
}

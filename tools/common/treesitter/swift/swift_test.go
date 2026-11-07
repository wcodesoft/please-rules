package swift

import "testing"

func TestConfig(t *testing.T) {
	src := []byte(`func discount(price: Int) -> Int {
    if price >= 100 {
        return price - 10
    }
    return price
}

func testDiscount() {
    let x = 100 - 10
}
`)

	cfg := Config()
	if !cfg.IsTestFile("calctests.swift") || cfg.IsTestFile("calc.swift") {
		t.Fatal("unexpected Swift IsTestFile result")
	}

	sites, err := cfg.Discover("Calc.swift", src, nil)
	if err != nil {
		t.Fatalf("Config().Discover: %v", err)
	}
	matched := false
	for _, s := range sites {
		if s.Function == "testDiscount" {
			t.Fatalf("unexpected mutation inside testDiscount: %+v", s)
		}
		if s.Function == "discount" && s.Original == "-" && s.Replacement == "+" {
			matched = true
		}
	}
	if !matched {
		t.Fatalf("expected '-' -> '+' in discount, got: %+v", sites)
	}
}

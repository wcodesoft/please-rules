package rust

import "testing"

func TestConfig(t *testing.T) {
	src := []byte(`pub fn in_range(x: i32, max: i32) -> bool {
    x >= 0 && x < max
}

#[cfg(test)]
mod tests {
    #[test]
    fn test_in_range() {
        assert!(1 + 1 == 2);
    }
}
`)

	cfg := Config()
	if !cfg.IsTestFile("foo_test.rs") || cfg.IsTestFile("lib.rs") {
		t.Fatal("unexpected Rust IsTestFile result")
	}

	sites, err := cfg.Discover("lib.rs", src, nil)
	if err != nil {
		t.Fatalf("Config().Discover: %v", err)
	}
	matched := false
	for _, s := range sites {
		if s.Function == "test_in_range" {
			t.Fatalf("unexpected mutation inside #[cfg(test)]: %+v", s)
		}
		if s.Function == "in_range" && s.Original == "&&" && s.Replacement == "||" {
			matched = true
		}
	}
	if !matched {
		t.Fatalf("expected '&&' -> '||' in in_range, got: %+v", sites)
	}
}

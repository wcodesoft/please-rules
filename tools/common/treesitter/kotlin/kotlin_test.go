package kotlin

import "testing"

func TestConfig(t *testing.T) {
	src := []byte(`package demo

fun addPositive(a: Int, b: Int): Int {
    if (a > 0 && b > 0) {
        return a + b
    }
    return 0
}

@Test
fun testAddPositive() {
    val x = 1 + 2
}
`)

	cfg := Config()
	if !cfg.IsTestFile("calctest.kt") || cfg.IsTestFile("calc.kt") {
		t.Fatal("unexpected Kotlin IsTestFile result")
	}

	sites, err := cfg.Discover("Calc.kt", src, nil)
	if err != nil {
		t.Fatalf("Config().Discover: %v", err)
	}
	matched := false
	for _, s := range sites {
		if s.Function == "testAddPositive" {
			t.Fatalf("unexpected mutation inside @Test function: %+v", s)
		}
		if s.Function == "addPositive" && s.Original == "+" && s.Replacement == "-" {
			matched = true
		}
	}
	if !matched {
		t.Fatalf("expected '+' -> '-' in addPositive, got: %+v", sites)
	}
}

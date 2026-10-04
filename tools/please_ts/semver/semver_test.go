package semver

import "testing"

func TestParse(t *testing.T) {
	for in, want := range map[string]string{
		"1.2.3":            "1.2.3",
		"v1.2.3":           "1.2.3",
		"=1.2.3":           "1.2.3",
		" 1.2.3 ":          "1.2.3",
		"1.2.3-beta.1":     "1.2.3-beta.1",
		"1.2.3-beta.1+b.5": "1.2.3-beta.1",
		"1.2.3+build":      "1.2.3",
		"0.0.0":            "0.0.0",
	} {
		v, err := Parse(in)
		if err != nil || v.String() != want {
			t.Errorf("Parse(%q) = %v, %v; want %s", in, v, err, want)
		}
	}
	for _, in := range []string{"", "1.2", "1.2.3.4", "a.b.c", "01.2.3", "1.2.3-", "1.2.3-a..b", "1.x.3", ">1.2.3", "-1.2.3"} {
		if _, err := Parse(in); err == nil {
			t.Errorf("Parse(%q): expected an error", in)
		}
	}
}

func TestComparePrecedence(t *testing.T) {
	// The ordering example of the semver specification, lowest first.
	ordered := []string{"1.0.0-alpha", "1.0.0-alpha.1", "1.0.0-alpha.beta", "1.0.0-beta", "1.0.0-beta.2", "1.0.0-beta.11", "1.0.0-rc.1", "1.0.0", "1.0.1", "1.1.0", "2.0.0", "10.0.0"}
	for i := range ordered {
		for j := range ordered {
			a, _ := Parse(ordered[i])
			b, _ := Parse(ordered[j])
			want := 0
			if i < j {
				want = -1
			} else if i > j {
				want = 1
			}
			if got := Compare(a, b); got != want {
				t.Errorf("Compare(%s, %s) = %d, want %d", ordered[i], ordered[j], got, want)
			}
		}
	}
	a, _ := Parse("1.0.0+a")
	b, _ := Parse("1.0.0+b")
	if Compare(a, b) != 0 {
		t.Error("build metadata must not affect precedence")
	}
}

func TestRangeIncludes(t *testing.T) {
	for _, tc := range [][2]string{
		{"1.0.0 - 2.0.0", "1.2.3"}, {"1.0.0 - 2.0.0", "2.0.0"}, {"1.2.3 - 2.3", "2.3.9"}, {"1.2 - 2.3.4", "1.2.0"}, {"1.2.3 - 2", "2.9.9"},
		{"^1.2.3", "1.2.3"}, {"^1.2.3", "1.9.9"}, {"^1.2", "1.9.0"}, {"^1.2.x", "1.2.0"}, {"^1", "1.0.0"},
		{"^0.2.3", "0.2.9"}, {"^0.0.3", "0.0.3"}, {"^0.0", "0.0.9"}, {"^0", "0.9.9"}, {"^0.x", "0.5.0"},
		{"~1.2.3", "1.2.9"}, {"~1.2", "1.2.0"}, {"~1", "1.9.9"}, {"~>1.2.3", "1.2.4"},
		{"1.2.x", "1.2.3"}, {"1.x", "1.9.0"}, {"1.X", "1.0.0"}, {"*", "1.2.3"}, {"x", "9.9.9"}, {"", "1.0.0"}, {"^", "1.0.0"},
		{"1", "1.5.0"}, {"1.2", "1.2.9"}, {"=1.2.3", "1.2.3"}, {"v1.2.3", "1.2.3"}, {"1.2.3", "1.2.3"},
		{">=1.0.0", "1.0.0"}, {">1.0.0", "1.0.1"}, {"<2.0.0", "1.9.9"}, {"<=2.0.0", "2.0.0"}, {"<=2", "2.9.9"}, {"<=1.2", "1.2.9"},
		{">1.2", "1.3.0"}, {">1", "2.0.0"}, {">=1.2", "1.2.0"}, {"<1.2", "1.1.9"},
		{">=1.2.3 <2", "1.9.9"}, {">= 1.2.3", "1.2.3"}, {"> 1.2.3 < 2.0.0", "1.5.0"},
		{"1.2.3 || 2.0.0", "2.0.0"}, {"^1 || ^2", "2.5.0"}, {"<1 || >=3", "3.1.0"},
		{"^1.2.3-beta.2", "1.2.3-beta.4"}, {"^1.2.3-beta.2", "1.3.0"}, {">=1.2.3-beta.2", "1.2.3-beta.4"}, {"1.2.3-beta.2", "1.2.3-beta.2"},
	} {
		r, err := ParseRange(tc[0])
		if err != nil {
			t.Errorf("ParseRange(%q): %v", tc[0], err)
			continue
		}
		v, _ := Parse(tc[1])
		if !r.Satisfies(v) {
			t.Errorf("%q should include %s", tc[0], tc[1])
		}
	}
}

func TestRangeExcludes(t *testing.T) {
	for _, tc := range [][2]string{
		{"^1.2.3", "2.0.0"}, {"^1.2.3", "1.2.2"}, {"^0.2.3", "0.3.0"}, {"^0.0.3", "0.0.4"}, {"^0.0", "0.1.0"}, {"^0", "1.0.0"},
		{"~1.2.3", "1.3.0"}, {"~1.2", "1.3.0"}, {"~1", "2.0.0"},
		{"1.2.x", "1.3.0"}, {"1.x", "2.0.0"}, {"1", "2.0.0"}, {"1.2", "1.3.0"}, {"1.2.3", "1.2.4"},
		{">1.0.0", "1.0.0"}, {"<1.0.0", "1.0.0"}, {">1.2", "1.2.9"}, {">1", "1.9.9"}, {"<=1.2", "1.3.0"},
		{"1.2.3 - 2.3.4", "2.3.5"}, {"1.2.3 - 2.3", "2.4.0"}, {"1.2.3 - 2", "3.0.0"}, {"1.0.0 - 2.0.0", "0.9.9"},
		{"1.2.3 || 2.0.0", "1.2.4"}, {">=1 <3", "3.0.0"}, {"<1 || >=3", "2.0.0"},
		// Prereleases only match a comparator that names a prerelease of the same version.
		{"^1.2.3", "1.2.4-beta.1"}, {"^1.2.3-beta.2", "1.2.4-beta.1"}, {"*", "1.2.3-beta.1"}, {">=1.0.0", "2.0.0-alpha"},
		{"<2.0.0", "2.0.0-alpha"}, {"^1.2.3-beta.2", "1.2.3-beta.1"}, {"1.x", "1.2.3-beta.1"},
	} {
		r, err := ParseRange(tc[0])
		if err != nil {
			t.Errorf("ParseRange(%q): %v", tc[0], err)
			continue
		}
		v, _ := Parse(tc[1])
		if r.Satisfies(v) {
			t.Errorf("%q should not include %s", tc[0], tc[1])
		}
	}
}

func TestParseRangeErrors(t *testing.T) {
	for _, in := range []string{"1.2.3.4", "a", "1.x.3", ">=a", "^a.b", "1.2.3 - ", "||x.y"} {
		if _, err := ParseRange(in); err == nil {
			t.Errorf("ParseRange(%q): expected an error", in)
		}
	}
}

func TestRangeNothingMatchesStarComparators(t *testing.T) {
	v, _ := Parse("1.0.0")
	for _, in := range []string{">*", "<*"} {
		r, err := ParseRange(in)
		if err != nil || r.Satisfies(v) {
			t.Errorf("%q: err=%v includes=%v; want a range that matches nothing", in, err, r.Satisfies(v))
		}
	}
}

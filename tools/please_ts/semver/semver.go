// Package semver implements the parts of npm's semantic versioning that dependency
// resolution needs: versions with prereleases, and ranges with ^, ~, comparators,
// x-ranges, hyphen ranges and ||.
package semver

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is a parsed semantic version. Build metadata is ignored, as it is for
// precedence.
type Version struct {
	Major, Minor, Patch uint64
	Pre                 []string // prerelease identifiers
}

// Parse parses "1.2.3", "v1.2.3" or "1.2.3-beta.1+build". All three numbers are required.
func Parse(s string) (Version, error) {
	orig := s
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(strings.TrimPrefix(s, "="), "v")
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	var pre string
	if i := strings.IndexByte(s, '-'); i >= 0 {
		s, pre = s[:i], s[i+1:]
		if pre == "" {
			return Version{}, fmt.Errorf("invalid version %q", orig)
		}
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid version %q", orig)
	}
	var nums [3]uint64
	for i, p := range parts {
		n, err := parseNum(p)
		if err != nil {
			return Version{}, fmt.Errorf("invalid version %q", orig)
		}
		nums[i] = n
	}
	v := Version{Major: nums[0], Minor: nums[1], Patch: nums[2]}
	if pre != "" {
		v.Pre = strings.Split(pre, ".")
		for _, id := range v.Pre {
			if id == "" {
				return Version{}, fmt.Errorf("invalid version %q", orig)
			}
		}
	}
	return v, nil
}

func parseNum(s string) (uint64, error) {
	if s == "" || (len(s) > 1 && s[0] == '0') {
		return 0, fmt.Errorf("bad number")
	}
	return strconv.ParseUint(s, 10, 64)
}

func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if len(v.Pre) > 0 {
		s += "-" + strings.Join(v.Pre, ".")
	}
	return s
}

// Compare returns -1, 0 or 1 following semver precedence: a prerelease sorts before its
// release, and prerelease identifiers compare numerically when numeric.
func Compare(a, b Version) int {
	for _, d := range [][2]uint64{{a.Major, b.Major}, {a.Minor, b.Minor}, {a.Patch, b.Patch}} {
		if d[0] != d[1] {
			if d[0] < d[1] {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(a.Pre) == 0 && len(b.Pre) == 0:
		return 0
	case len(a.Pre) == 0:
		return 1
	case len(b.Pre) == 0:
		return -1
	}
	for i := 0; i < len(a.Pre) && i < len(b.Pre); i++ {
		if c := comparePre(a.Pre[i], b.Pre[i]); c != 0 {
			return c
		}
	}
	switch {
	case len(a.Pre) < len(b.Pre):
		return -1
	case len(a.Pre) > len(b.Pre):
		return 1
	}
	return 0
}

func comparePre(a, b string) int {
	an, aerr := strconv.ParseUint(a, 10, 64)
	bn, berr := strconv.ParseUint(b, 10, 64)
	switch {
	case aerr == nil && berr == nil:
		if an < bn {
			return -1
		} else if an > bn {
			return 1
		}
		return 0
	case aerr == nil: // numeric identifiers sort before alphanumeric ones
		return -1
	case berr == nil:
		return 1
	}
	return strings.Compare(a, b)
}

type comparator struct {
	op string // "<", "<=", ">", ">=", "="
	v  Version
}

func (c comparator) test(v Version) bool {
	cmp := Compare(v, c.v)
	switch c.op {
	case "<":
		return cmp < 0
	case "<=":
		return cmp <= 0
	case ">":
		return cmp > 0
	case ">=":
		return cmp >= 0
	}
	return cmp == 0
}

// Range is a parsed npm version range: alternatives (||) of comparator sets.
type Range struct {
	sets [][]comparator
}

// ParseRange parses an npm range such as "^1.2.3", "~1.2", ">=1 <3", "1.x || ^2", "1.2.3 - 2.3.4"
// or "*". The empty string means "*".
func ParseRange(s string) (Range, error) {
	var r Range
	for _, alt := range strings.Split(s, "||") {
		set, err := parseSet(strings.TrimSpace(alt))
		if err != nil {
			return Range{}, fmt.Errorf("invalid range %q: %w", s, err)
		}
		r.sets = append(r.sets, set)
	}
	return r, nil
}

// Satisfies reports whether v is in the range. A prerelease version only satisfies a
// comparator set that has a comparator with a prerelease on the same major.minor.patch.
func (r Range) Satisfies(v Version) bool {
	for _, set := range r.sets {
		if satisfiesSet(set, v) {
			return true
		}
	}
	return false
}

func satisfiesSet(set []comparator, v Version) bool {
	for _, c := range set {
		if !c.test(v) {
			return false
		}
	}
	if len(v.Pre) == 0 {
		return true
	}
	for _, c := range set {
		if len(c.v.Pre) > 0 && c.v.Major == v.Major && c.v.Minor == v.Minor && c.v.Patch == v.Patch {
			return true
		}
	}
	return false
}

// partial is a version with optional minor and patch ("1", "1.2", "1.2.x", "1.2.3-beta").
type partial struct {
	major, minor, patch int64 // -1 when missing or a wildcard
	pre                 []string
}

func parsePartial(s string) (partial, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(strings.TrimPrefix(s, "="), "v")
	if s == "" || s == "*" || s == "x" || s == "X" {
		return partial{-1, -1, -1, nil}, nil
	}
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	var pre []string
	if i := strings.IndexByte(s, '-'); i >= 0 {
		pre = strings.Split(s[i+1:], ".")
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) > 3 {
		return partial{}, fmt.Errorf("bad version %q", s)
	}
	p := partial{-1, -1, -1, pre}
	slots := []*int64{&p.major, &p.minor, &p.patch}
	wildcard := false
	for i, part := range parts {
		if part == "x" || part == "X" || part == "*" {
			wildcard = true // a wildcard opens everything after it
			continue
		}
		if wildcard {
			return partial{}, fmt.Errorf("bad version %q: a number follows a wildcard", s)
		}
		n, err := strconv.ParseInt(part, 10, 64)
		if err != nil || n < 0 {
			return partial{}, fmt.Errorf("bad version %q", s)
		}
		*slots[i] = n
	}
	if (p.minor < 0 && p.patch >= 0) || (p.major < 0 && p.minor >= 0) {
		return partial{}, fmt.Errorf("bad version %q", s)
	}
	return p, nil
}

func (p partial) version(fill uint64) Version {
	v := Version{Pre: p.pre}
	nums := []*uint64{&v.Major, &v.Minor, &v.Patch}
	for i, n := range []int64{p.major, p.minor, p.patch} {
		if n >= 0 {
			*nums[i] = uint64(n)
		} else {
			*nums[i] = fill
		}
	}
	return v
}

// floor is the lowest version the partial denotes ("1.2" is 1.2.0).
func (p partial) floor() Version { return p.version(0) }

// ceiling returns the exclusive upper bound of the partial ("1.2" is <1.3.0-0).
func (p partial) ceiling() (Version, bool) {
	switch {
	case p.major < 0:
		return Version{}, false
	case p.minor < 0:
		return Version{Major: uint64(p.major) + 1, Pre: []string{"0"}}, true
	case p.patch < 0:
		return Version{Major: uint64(p.major), Minor: uint64(p.minor) + 1, Pre: []string{"0"}}, true
	}
	return Version{}, false
}

func parseSet(s string) ([]comparator, error) {
	if s == "" {
		return nil, nil // "*"
	}
	fields := strings.Fields(s)
	// Hyphen range: "A - B".
	if len(fields) == 3 && fields[1] == "-" {
		lo, err := parsePartial(fields[0])
		if err != nil {
			return nil, err
		}
		hi, err := parsePartial(fields[2])
		if err != nil {
			return nil, err
		}
		var set []comparator
		if lo.major >= 0 {
			set = append(set, comparator{">=", lo.floor()})
		}
		if c, ok := hi.ceiling(); ok {
			set = append(set, comparator{"<", c})
		} else if hi.major >= 0 {
			set = append(set, comparator{"<=", hi.floor()})
		}
		return set, nil
	}

	// "> 1.2.3" is written with a space in some manifests: glue a lone operator to the next field.
	var glued []string
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if isOperator(f) && i+1 < len(fields) {
			f += fields[i+1]
			i++
		}
		glued = append(glued, f)
	}

	var set []comparator
	for _, f := range glued {
		cs, err := parseComparator(f)
		if err != nil {
			return nil, err
		}
		set = append(set, cs...)
	}
	return set, nil
}

func isOperator(s string) bool {
	switch s {
	case "<", "<=", ">", ">=", "=", "^", "~", "~>":
		return true
	}
	return false
}

func parseComparator(f string) ([]comparator, error) {
	op := ""
	for _, o := range []string{"~>", "<=", ">=", "<", ">", "=", "^", "~"} {
		if strings.HasPrefix(f, o) {
			op = o
			f = f[len(o):]
			break
		}
	}
	p, err := parsePartial(f)
	if err != nil {
		return nil, err
	}

	switch op {
	case "^":
		return caret(p), nil
	case "~", "~>":
		return tilde(p), nil
	case ">":
		if c, ok := p.ceiling(); ok {
			return []comparator{{">=", withoutPre(c)}}, nil
		}
		if p.major < 0 {
			return []comparator{{"<", Version{}}}, nil // ">*" matches nothing
		}
		return []comparator{{">", p.floor()}}, nil
	case ">=":
		if p.major < 0 {
			return nil, nil
		}
		return []comparator{{">=", p.floor()}}, nil
	case "<":
		if p.major < 0 {
			return []comparator{{"<", Version{}}}, nil // "<*" matches nothing
		}
		return []comparator{{"<", p.floor()}}, nil
	case "<=":
		if c, ok := p.ceiling(); ok {
			return []comparator{{"<", c}}, nil
		}
		if p.major < 0 {
			return nil, nil
		}
		return []comparator{{"<=", p.floor()}}, nil
	}

	// No operator or "=": an exact version or an x-range.
	if p.major < 0 {
		return nil, nil
	}
	if c, ok := p.ceiling(); ok {
		return []comparator{{">=", p.floor()}, {"<", c}}, nil
	}
	return []comparator{{"=", p.floor()}}, nil
}

func withoutPre(v Version) Version { v.Pre = nil; return v }

func caret(p partial) []comparator {
	if p.major < 0 {
		return nil
	}
	lo := p.floor()
	var hi Version
	switch {
	case p.major > 0 || (p.minor < 0):
		hi = Version{Major: lo.Major + 1}
	case p.minor > 0 || p.patch < 0:
		hi = Version{Major: 0, Minor: lo.Minor + 1}
	default:
		hi = Version{Major: 0, Minor: 0, Patch: lo.Patch + 1}
	}
	hi.Pre = []string{"0"}
	return []comparator{{">=", lo}, {"<", hi}}
}

func tilde(p partial) []comparator {
	if p.major < 0 {
		return nil
	}
	lo := p.floor()
	var hi Version
	if p.minor < 0 {
		hi = Version{Major: lo.Major + 1}
	} else {
		hi = Version{Major: lo.Major, Minor: lo.Minor + 1}
	}
	hi.Pre = []string{"0"}
	return []comparator{{">=", lo}, {"<", hi}}
}

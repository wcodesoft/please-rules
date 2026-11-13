package css

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// VarDef is the definition of a custom property: a `--name: value` declaration in any rule, or an
// `@property --name` rule.
type VarDef struct {
	Name string
	File string
	Line int
}

// VarUse is one `var(--name)` or `var(--name, fallback)` reference.
type VarUse struct {
	Name     string
	File     string
	Line     int
	Fallback bool // the reference gives a fallback value
}

// Variables holds the custom properties a stylesheet defines and the ones it references.
type Variables struct {
	Defs []VarDef
	Uses []VarUse
}

// ParseVariables scans CSS source text and returns the custom property definitions and `var()`
// references it contains, in source order. Comments and string contents are not looked at.
func ParseVariables(filename, src string) (Variables, error) {
	s := newScanner(filename, src)
	if _, err := s.parse(); err != nil {
		return Variables{}, err
	}
	return s.vars, nil
}

// recordVariables notes the definition made by one declaration and the var() references in its
// value. token is the whole declaration text; the value starts at valueStart.
func (s *scanner) recordVariables(prop, token string, valueStart, line int) {
	if strings.HasPrefix(prop, "--") {
		s.vars.Defs = append(s.vars.Defs, VarDef{Name: prop, File: s.file, Line: line})
	}
	for _, m := range scanVarUses(token[valueStart:]) {
		s.vars.Uses = append(s.vars.Uses, VarUse{
			Name:     m.name,
			File:     s.file,
			Line:     line + strings.Count(token[:valueStart+m.offset], "\n"),
			Fallback: m.fallback,
		})
	}
}

type varMatch struct {
	name     string
	fallback bool
	offset   int // byte offset of the `var(` in the scanned text
}

// scanVarUses finds the `var(--name ...)` references in a declaration value. A reference inside
// another one's fallback is found too. String contents are skipped.
func scanVarUses(value string) []varMatch {
	var found []varMatch
	for i := 0; i < len(value); {
		c := value[i]
		if c == '"' || c == '\'' {
			i = skipQuoted(value, i)
			continue
		}
		if (c == 'v' || c == 'V') && isVarCall(value, i) {
			j := skipSpaces(value, i+4)
			if strings.HasPrefix(value[j:], "--") {
				k := j + 2
				for k < len(value) && isNameByte(value[k]) {
					k++
				}
				if k > j+2 {
					after := skipSpaces(value, k)
					found = append(found, varMatch{
						name:     value[j:k],
						fallback: after < len(value) && value[after] == ',',
						offset:   i,
					})
					i = k
					continue
				}
			}
			i = j
			continue
		}
		i++
	}
	return found
}

// isVarCall reports whether value has `var(` at i, not as the tail of a longer function name.
func isVarCall(value string, i int) bool {
	if i+4 > len(value) || !strings.EqualFold(value[i:i+3], "var") || value[i+3] != '(' {
		return false
	}
	return i == 0 || !isNameByte(value[i-1])
}

func isNameByte(b byte) bool {
	return b == '-' || b == '_' || b >= 0x80 ||
		(b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func skipSpaces(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	return i
}

// skipQuoted returns the index just after the string literal that starts at i.
func skipQuoted(s string, i int) int {
	quote := s[i]
	for i++; i < len(s); i++ {
		if s[i] == '\\' {
			i++
			continue
		}
		if s[i] == quote {
			return i + 1
		}
	}
	return len(s)
}

// UndefinedVar is a custom property that is referenced with var() and defined nowhere.
type UndefinedVar struct {
	Name string
	Uses []VarUse
}

// FindUndefinedVars returns the custom properties that vars references without a fallback and
// never defines, sorted by name. A reference that gives a fallback is not reported: the fallback
// states what is meant when the property is missing. Names matched by allow are skipped; an entry
// is an exact name (`--x`) or a prefix ending in `*` (`--chart-*`).
//
// The check does not follow the cascade: a property defined anywhere (a :root, a theme block, a
// component rule, an @property) counts as defined everywhere.
func FindUndefinedVars(vars Variables, allow []string) []UndefinedVar {
	defined := make(map[string]bool, len(vars.Defs))
	for _, d := range vars.Defs {
		defined[d.Name] = true
	}
	byName := map[string]*UndefinedVar{}
	for _, u := range vars.Uses {
		if u.Fallback || defined[u.Name] || isAllowedVar(u.Name, allow) {
			continue
		}
		v := byName[u.Name]
		if v == nil {
			v = &UndefinedVar{Name: u.Name}
			byName[u.Name] = v
		}
		v.Uses = append(v.Uses, u)
	}
	out := make([]UndefinedVar, 0, len(byName))
	for _, v := range byName {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func isAllowedVar(name string, allow []string) bool {
	for _, a := range allow {
		if strings.HasSuffix(a, "*") {
			if strings.HasPrefix(name, strings.TrimSuffix(a, "*")) {
				return true
			}
		} else if name == a {
			return true
		}
	}
	return false
}

// maxListedUses is how many places of use are printed for each undefined property.
const maxListedUses = 5

// CheckVariables parses the CSS files of a bundle and returns an error that names every custom
// property used with var() but defined in none of them, with the places of use.
func CheckVariables(files []string, allow []string) error {
	for _, a := range allow {
		if !strings.HasPrefix(a, "--") || strings.TrimSuffix(strings.TrimPrefix(a, "--"), "*") == "" {
			return fmt.Errorf("invalid allow_undefined_vars entry %q: want a custom property name such as --x, or a prefix such as --chart-*", a)
		}
	}

	var all Variables
	seen := make(map[string]bool, len(files))
	for _, file := range files {
		if seen[file] {
			continue
		}
		seen[file] = true
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("reading CSS file %s: %w", file, err)
		}
		vars, err := ParseVariables(file, string(data))
		if err != nil {
			return err
		}
		all.Defs = append(all.Defs, vars.Defs...)
		all.Uses = append(all.Uses, vars.Uses...)
	}

	undefined := FindUndefinedVars(all, allow)
	if len(undefined) == 0 {
		return nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "CSS undefined-variable check failed: %d custom %s used with var() but defined in none of the bundle's files:\n",
		len(undefined), plural(len(undefined), "property is", "properties are"))
	for _, v := range undefined {
		fmt.Fprintf(&sb, "  %s: %d %s without a fallback\n", v.Name, len(v.Uses), plural(len(v.Uses), "use", "uses"))
		for i, u := range v.Uses {
			if i == maxListedUses {
				fmt.Fprintf(&sb, "      ... and %d more\n", len(v.Uses)-maxListedUses)
				break
			}
			fmt.Fprintf(&sb, "      %s:%d\n", u.File, u.Line)
		}
	}
	sb.WriteString("Define the property, give each use a fallback (var(--x, value)), or list it in allow_undefined_vars when it is set at runtime.")
	return fmt.Errorf("%s", sb.String())
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

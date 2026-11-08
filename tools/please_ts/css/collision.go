package css

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// RuleKind identifies the category of a CSS definition tracked for cross-file collisions.
type RuleKind string

const (
	// KindSelector represents a top-level or at-rule-scoped CSS style selector (e.g. ".card").
	KindSelector RuleKind = "selector"
	// KindKeyframes represents an @keyframes animation definition (e.g. "@keyframes spin").
	KindKeyframes RuleKind = "keyframes"
	// KindVariable represents a CSS custom property declared on :root (e.g. "--primary-color").
	KindVariable RuleKind = "variable"
)

// Definition records a single CSS selector, @keyframes, or :root custom property occurrence.
type Definition struct {
	Kind  RuleKind
	Name  string
	Scope string // enclosing conditional at-rules (e.g. "@media (max-width:600px)")
	File  string
	Line  int
}

// Key returns the canonical collision key for this definition.
func (d Definition) Key() string {
	if d.Scope != "" {
		return string(d.Kind) + "|" + d.Scope + "|" + d.Name
	}
	return string(d.Kind) + "||" + d.Name
}

// Collision represents a duplicate CSS definition across two distinct files.
type Collision struct {
	First  Definition
	Second Definition
}

// Error formats a human-readable collision diagnostic with file and line numbers.
func (c Collision) Error() string {
	scopeDesc := ""
	if c.First.Scope != "" {
		scopeDesc = fmt.Sprintf(" (in %s)", c.First.Scope)
	}
	return fmt.Sprintf(
		"duplicate CSS %s %q%s defined in multiple files:\n  - %s:%d\n  - %s:%d",
		c.First.Kind,
		c.First.Name,
		scopeDesc,
		c.First.File,
		c.First.Line,
		c.Second.File,
		c.Second.Line,
	)
}

// CheckFiles parses the given CSS files and returns an error if any two distinct files
// define the same selector (within the same conditional at-rule scope), @keyframes name,
// or :root custom property.
func CheckFiles(files []string) error {
	var allDefs []Definition
	seenFiles := make(map[string]bool, len(files))

	for _, file := range files {
		if seenFiles[file] {
			continue
		}
		seenFiles[file] = true

		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("reading CSS file %s: %w", file, err)
		}
		defs, err := ParseDefinitions(file, string(data))
		if err != nil {
			return err
		}
		allDefs = append(allDefs, defs...)
	}

	collisions := FindCollisions(allDefs)
	if len(collisions) == 0 {
		return nil
	}

	msgs := make([]string, 0, len(collisions))
	for _, col := range collisions {
		msgs = append(msgs, col.Error())
	}
	return fmt.Errorf("CSS collision check failed:\n%s", strings.Join(msgs, "\n"))
}

// FindCollisions detects definitions that share the same canonical key across different files.
// Multiple definitions of the same selector or variable within the *same* file are allowed.
func FindCollisions(defs []Definition) []Collision {
	firstByFile := make(map[string]Definition)
	var collisions []Collision
	reported := make(map[string]bool)

	for _, d := range defs {
		key := d.Key()
		prev, exists := firstByFile[key]
		if !exists {
			firstByFile[key] = d
			continue
		}
		if prev.File == d.File {
			continue
		}
		reportKey := key + "|" + prev.File + "|" + d.File
		if !reported[reportKey] {
			reported[reportKey] = true
			collisions = append(collisions, Collision{First: prev, Second: d})
		}
	}

	sort.Slice(collisions, func(i, j int) bool {
		if collisions[i].First.File != collisions[j].First.File {
			return collisions[i].First.File < collisions[j].First.File
		}
		return collisions[i].First.Line < collisions[j].First.Line
	})
	return collisions
}

// ParseDefinitions scans CSS source text, validates balanced braces/strings/comments,
// and extracts top-level and conditional-group style selectors, @keyframes names, and :root variables.
func ParseDefinitions(filename, src string) ([]Definition, error) {
	s := newScanner(filename, src)
	return s.parse()
}

type scanner struct {
	file string
	src  string
	pos  int
	line int
}

func newScanner(file, src string) *scanner {
	return &scanner{
		file: file,
		src:  src,
		line: 1,
	}
}

func (s *scanner) parse() ([]Definition, error) {
	defs, err := s.parseBlock(nil, false)
	if err != nil {
		return nil, err
	}
	return defs, nil
}

func (s *scanner) parseBlock(scopes []string, expectClosingBrace bool) ([]Definition, error) {
	var defs []Definition

	for {
		if err := s.skipTrivia(); err != nil {
			return nil, err
		}
		if s.pos >= len(s.src) {
			if expectClosingBrace {
				return nil, fmt.Errorf("%s:%d: unexpected end of CSS file (unclosed '{')", s.file, s.line)
			}
			return defs, nil
		}

		if s.src[s.pos] == '}' {
			if !expectClosingBrace {
				return nil, fmt.Errorf("%s:%d: unexpected '}' in CSS file", s.file, s.line)
			}
			s.advance()
			return defs, nil
		}

		startLine := s.line
		prelude, term, err := s.readPrelude()
		if err != nil {
			return nil, err
		}

		prelude = normalizeWhitespace(prelude)
		if term == ';' || term == 0 {
			continue
		}

		// term == '{'
		if strings.HasPrefix(prelude, "@") {
			blockDefs, err := s.handleAtRule(prelude, startLine, scopes)
			if err != nil {
				return nil, err
			}
			defs = append(defs, blockDefs...)
			continue
		}

		// Style rule: extract selectors and optional :root custom properties
		scopeStr := strings.Join(scopes, " > ")
		selectors := splitSelectors(prelude)
		isRoot := false
		for _, sel := range selectors {
			normSel := normalizeSelector(sel)
			if normSel == "" {
				continue
			}
			if normSel == ":root" {
				isRoot = true
			}
			defs = append(defs, Definition{
				Kind:  KindSelector,
				Name:  normSel,
				Scope: scopeStr,
				File:  s.file,
				Line:  startLine,
			})
		}

		bodyDefs, err := s.readDeclarationBlock(isRoot, scopeStr)
		if err != nil {
			return nil, err
		}
		defs = append(defs, bodyDefs...)
	}
}

func (s *scanner) handleAtRule(prelude string, startLine int, scopes []string) ([]Definition, error) {
	lower := strings.ToLower(prelude)

	switch {
	case strings.HasPrefix(lower, "@keyframes ") || strings.HasPrefix(lower, "@-webkit-keyframes "):
		fields := strings.Fields(prelude)
		var defs []Definition
		if len(fields) >= 2 {
			name := strings.Trim(fields[1], `"'`)
			defs = append(defs, Definition{
				Kind: KindKeyframes,
				Name: name,
				File: s.file,
				Line: startLine,
			})
		}
		if err := s.skipBalancedBlock(); err != nil {
			return nil, err
		}
		return defs, nil

	case isConditionalGroupAtRule(lower):
		nextScopes := append(append([]string(nil), scopes...), normalizeAtRuleScope(prelude))
		return s.parseBlock(nextScopes, true)

	default:
		// @font-face, @page, @property, @counter-style, etc.
		if err := s.skipBalancedBlock(); err != nil {
			return nil, err
		}
		return nil, nil
	}
}

func isConditionalGroupAtRule(lower string) bool {
	for _, prefix := range []string{"@media", "@supports", "@container", "@layer", "@scope"} {
		if lower == prefix || strings.HasPrefix(lower, prefix+" ") || strings.HasPrefix(lower, prefix+"(") {
			return true
		}
	}
	return false
}

// readDeclarationBlock reads inside a style rule's `{ ... }`, extracting `--*` custom properties
// when `extractVars` is true, and skipping any nested blocks safely.
func (s *scanner) readDeclarationBlock(extractVars bool, scopeStr string) ([]Definition, error) {
	var defs []Definition
	for {
		if err := s.skipTrivia(); err != nil {
			return nil, err
		}
		if s.pos >= len(s.src) {
			return nil, fmt.Errorf("%s:%d: unexpected end of CSS file inside rule block", s.file, s.line)
		}
		if s.src[s.pos] == '}' {
			s.advance()
			return defs, nil
		}

		propLine := s.line
		token, term, err := s.readDeclarationOrNested()
		if err != nil {
			return nil, err
		}
		if term == '{' {
			// Nested block inside style rule: skip balanced braces
			if err := s.skipBalancedBlock(); err != nil {
				return nil, err
			}
			continue
		}
		if !extractVars {
			continue
		}
		colonIdx := strings.IndexByte(token, ':')
		if colonIdx <= 0 {
			continue
		}
		propName := strings.TrimSpace(token[:colonIdx])
		if strings.HasPrefix(propName, "--") {
			defs = append(defs, Definition{
				Kind:  KindVariable,
				Name:  propName,
				Scope: scopeStr,
				File:  s.file,
				Line:  propLine,
			})
		}
	}
}

func (s *scanner) readDeclarationOrNested() (string, byte, error) {
	var sb strings.Builder
	parenDepth := 0
	for s.pos < len(s.src) {
		ch := s.src[s.pos]
		if ch == '/' && s.pos+1 < len(s.src) && s.src[s.pos+1] == '*' {
			if err := s.skipComment(); err != nil {
				return "", 0, err
			}
			sb.WriteByte(' ')
			continue
		}
		if ch == '"' || ch == '\'' {
			lit, err := s.readString()
			if err != nil {
				return "", 0, err
			}
			sb.WriteString(lit)
			continue
		}
		if ch == '(' {
			parenDepth++
			sb.WriteByte(ch)
			s.advance()
			continue
		}
		if ch == ')' && parenDepth > 0 {
			parenDepth--
			sb.WriteByte(ch)
			s.advance()
			continue
		}
		if parenDepth == 0 {
			if ch == ';' || ch == '{' {
				s.advance()
				return sb.String(), ch, nil
			}
			if ch == '}' {
				return sb.String(), '}', nil
			}
		}
		sb.WriteByte(ch)
		s.advance()
	}
	return sb.String(), 0, nil
}

func (s *scanner) readPrelude() (string, byte, error) {
	var sb strings.Builder
	parenDepth := 0
	bracketDepth := 0

	for s.pos < len(s.src) {
		ch := s.src[s.pos]
		if ch == '/' && s.pos+1 < len(s.src) && s.src[s.pos+1] == '*' {
			if err := s.skipComment(); err != nil {
				return "", 0, err
			}
			sb.WriteByte(' ')
			continue
		}
		if ch == '"' || ch == '\'' {
			lit, err := s.readString()
			if err != nil {
				return "", 0, err
			}
			sb.WriteString(lit)
			continue
		}
		switch ch {
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case '{', ';':
			if parenDepth == 0 && bracketDepth == 0 {
				s.advance()
				return sb.String(), ch, nil
			}
		case '}':
			if parenDepth == 0 && bracketDepth == 0 {
				return "", 0, fmt.Errorf("%s:%d: unexpected '}' in CSS selector prelude", s.file, s.line)
			}
		}
		sb.WriteByte(ch)
		s.advance()
	}
	return sb.String(), 0, nil
}

func (s *scanner) skipBalancedBlock() error {
	depth := 1
	for s.pos < len(s.src) && depth > 0 {
		ch := s.src[s.pos]
		if ch == '/' && s.pos+1 < len(s.src) && s.src[s.pos+1] == '*' {
			if err := s.skipComment(); err != nil {
				return err
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			if _, err := s.readString(); err != nil {
				return err
			}
			continue
		}
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
		}
		s.advance()
	}
	if depth != 0 {
		return fmt.Errorf("%s:%d: unclosed '{' in CSS block", s.file, s.line)
	}
	return nil
}

func (s *scanner) skipTrivia() error {
	for s.pos < len(s.src) {
		ch := s.src[s.pos]
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			s.advance()
			continue
		}
		if ch == '/' && s.pos+1 < len(s.src) && s.src[s.pos+1] == '*' {
			if err := s.skipComment(); err != nil {
				return err
			}
			continue
		}
		break
	}
	return nil
}

func (s *scanner) skipComment() error {
	startLine := s.line
	s.advance() // '/'
	s.advance() // '*'
	for s.pos+1 < len(s.src) {
		if s.src[s.pos] == '*' && s.src[s.pos+1] == '/' {
			s.advance()
			s.advance()
			return nil
		}
		s.advance()
	}
	return fmt.Errorf("%s:%d: unclosed CSS comment", s.file, startLine)
}

func (s *scanner) readString() (string, error) {
	quote := s.src[s.pos]
	startLine := s.line
	var sb strings.Builder
	sb.WriteByte(quote)
	s.advance()

	for s.pos < len(s.src) {
		ch := s.src[s.pos]
		if ch == '\\' && s.pos+1 < len(s.src) {
			sb.WriteByte(ch)
			s.advance()
			sb.WriteByte(s.src[s.pos])
			s.advance()
			continue
		}
		if ch == quote {
			sb.WriteByte(ch)
			s.advance()
			return sb.String(), nil
		}
		if ch == '\n' {
			return "", fmt.Errorf("%s:%d: unterminated string literal in CSS", s.file, startLine)
		}
		sb.WriteByte(ch)
		s.advance()
	}
	return "", fmt.Errorf("%s:%d: unterminated string literal in CSS", s.file, startLine)
}

func (s *scanner) advance() {
	if s.pos < len(s.src) {
		if s.src[s.pos] == '\n' {
			s.line++
		}
		s.pos++
	}
}

// splitSelectors splits a comma-separated selector list, ignoring commas inside `(...)` or `[...]`.
func splitSelectors(prelude string) []string {
	var result []string
	var cur strings.Builder
	parenDepth := 0
	bracketDepth := 0
	var inQuote byte

	for i := 0; i < len(prelude); i++ {
		ch := prelude[i]
		if inQuote != 0 {
			if ch == '\\' && i+1 < len(prelude) {
				cur.WriteByte(ch)
				i++
				cur.WriteByte(prelude[i])
				continue
			}
			if ch == inQuote {
				inQuote = 0
			}
			cur.WriteByte(ch)
			continue
		}
		switch ch {
		case '"', '\'':
			inQuote = ch
		case '(':
			parenDepth++
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
		case '[':
			bracketDepth++
		case ']':
			if bracketDepth > 0 {
				bracketDepth--
			}
		case ',':
			if parenDepth == 0 && bracketDepth == 0 {
				result = append(result, cur.String())
				cur.Reset()
				continue
			}
		}
		cur.WriteByte(ch)
	}
	if cur.Len() > 0 {
		result = append(result, cur.String())
	}
	return result
}

// normalizeSelector canonicalizes whitespace around combinators (`>`, `+`, `~`) and collapses runs of spaces.
func normalizeSelector(sel string) string {
	s := normalizeWhitespace(sel)
	for _, comb := range []string{">", "+", "~"} {
		s = strings.ReplaceAll(s, " "+comb+" ", comb)
		s = strings.ReplaceAll(s, " "+comb, comb)
		s = strings.ReplaceAll(s, comb+" ", comb)
	}
	return s
}

func normalizeAtRuleScope(prelude string) string {
	s := normalizeWhitespace(prelude)
	s = strings.ReplaceAll(s, ": ", ":")
	s = strings.ReplaceAll(s, " :", ":")
	return s
}

func normalizeWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

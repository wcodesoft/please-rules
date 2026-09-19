package ast

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// TokenType represents lexical token categories in WIT.
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdent
	TokenColon
	TokenSemicolon
	TokenComma
	TokenLBrace
	TokenRBrace
	TokenLParen
	TokenRParen
	TokenLAngle
	TokenRAngle
	TokenArrow
	TokenEquals
	TokenAt
)

type Token struct {
	Type  TokenType
	Value string
	Line  int
}

// Lexer tokenizes WIT source code.
type Lexer struct {
	src    []rune
	pos    int
	length int
	line   int
}

func NewLexer(input string) *Lexer {
	src := []rune(input)
	return &Lexer{
		src:    src,
		pos:    0,
		length: len(src),
		line:   1,
	}
}

func (l *Lexer) peek() rune {
	if l.pos >= l.length {
		return 0
	}
	return l.src[l.pos]
}

func (l *Lexer) next() rune {
	if l.pos >= l.length {
		return 0
	}
	r := l.src[l.pos]
	l.pos++
	if r == '\n' {
		l.line++
	}
	return r
}

func (l *Lexer) skipWhitespaceAndComments() {
	for l.pos < l.length {
		r := l.peek()
		if unicode.IsSpace(r) {
			l.next()
			continue
		}
		if r == '/' && l.pos+1 < l.length {
			nextR := l.src[l.pos+1]
			if nextR == '/' {
				// Single line comment
				for l.pos < l.length && l.peek() != '\n' {
					l.next()
				}
				continue
			} else if nextR == '*' {
				// Multi line comment
				l.next() // consume '/'
				l.next() // consume '*'
				for l.pos < l.length {
					if l.peek() == '*' && l.pos+1 < l.length && l.src[l.pos+1] == '/' {
						l.next()
						l.next()
						break
					}
					l.next()
				}
				continue
			}
		}
		break
	}
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespaceAndComments()

	if l.pos >= l.length {
		return Token{Type: TokenEOF, Value: "", Line: l.line}
	}

	startLine := l.line
	r := l.next()

	switch r {
	case ':':
		return Token{Type: TokenColon, Value: ":", Line: startLine}
	case ';':
		return Token{Type: TokenSemicolon, Value: ";", Line: startLine}
	case ',':
		return Token{Type: TokenComma, Value: ",", Line: startLine}
	case '{':
		return Token{Type: TokenLBrace, Value: "{", Line: startLine}
	case '}':
		return Token{Type: TokenRBrace, Value: "}", Line: startLine}
	case '(':
		return Token{Type: TokenLParen, Value: "(", Line: startLine}
	case ')':
		return Token{Type: TokenRParen, Value: ")", Line: startLine}
	case '<':
		return Token{Type: TokenLAngle, Value: "<", Line: startLine}
	case '>':
		return Token{Type: TokenRAngle, Value: ">", Line: startLine}
	case '=':
		return Token{Type: TokenEquals, Value: "=", Line: startLine}
	case '@':
		return Token{Type: TokenAt, Value: "@", Line: startLine}
	case '-':
		if l.peek() == '>' {
			l.next()
			return Token{Type: TokenArrow, Value: "->", Line: startLine}
		}
	}

	// Identifiers (including kebab-case, dots, and colons in package paths or versions)
	if unicode.IsLetter(r) || r == '_' || r == '%' {
		var val []rune
		val = append(val, r)
		for l.pos < l.length {
			peeked := l.peek()
			if unicode.IsLetter(peeked) || unicode.IsDigit(peeked) || peeked == '_' || peeked == '-' {
				val = append(val, l.next())
			} else {
				break
			}
		}
		return Token{Type: TokenIdent, Value: string(val), Line: startLine}
	}

	// Digits (e.g. in versions)
	if unicode.IsDigit(r) {
		var val []rune
		val = append(val, r)
		for l.pos < l.length {
			peeked := l.peek()
			if unicode.IsDigit(peeked) || peeked == '.' {
				val = append(val, l.next())
			} else {
				break
			}
		}
		return Token{Type: TokenIdent, Value: string(val), Line: startLine}
	}

	return Token{Type: TokenIdent, Value: string(r), Line: startLine}
}

// Parser parses WIT tokens into an AST.
type Parser struct {
	tokens []Token
	pos    int
}

func NewParser(tokens []Token) *Parser {
	return &Parser{tokens: tokens, pos: 0}
}

func (p *Parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) next() Token {
	tok := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *Parser) expect(tt TokenType) (Token, error) {
	tok := p.next()
	if tok.Type != tt {
		return tok, fmt.Errorf("line %d: expected token type %v, got %v (%q)", tok.Line, tt, tok.Type, tok.Value)
	}
	return tok, nil
}

func (p *Parser) expectIdent() (string, error) {
	tok := p.next()
	if tok.Type != TokenIdent {
		return "", fmt.Errorf("line %d: expected identifier, got %q", tok.Line, tok.Value)
	}
	return tok.Value, nil
}

// Parse parses the whole WIT file into a Package AST.
func (p *Parser) Parse() (*Package, error) {
	pkg := &Package{}

	for p.peek().Type != TokenEOF {
		tok := p.peek()
		if tok.Type == TokenIdent {
			switch tok.Value {
			case "package":
				p.next()
				if err := p.parsePackage(pkg); err != nil {
					return nil, err
				}
			case "interface":
				p.next()
				iface, err := p.parseInterface()
				if err != nil {
					return nil, err
				}
				pkg.Interfaces = append(pkg.Interfaces, *iface)
			case "world":
				p.next()
				w, err := p.parseWorld()
				if err != nil {
					return nil, err
				}
				pkg.Worlds = append(pkg.Worlds, *w)
			default:
				// Skip unexpected token at top level
				p.next()
			}
		} else {
			p.next()
		}
	}

	return pkg, nil
}

func (p *Parser) parsePackage(pkg *Package) error {
	ns, err := p.expectIdent()
	if err != nil {
		return err
	}
	pkg.Namespace = ns

	if p.peek().Type == TokenColon {
		p.next()
		name, err := p.expectIdent()
		if err != nil {
			return err
		}
		pkg.Name = name
	} else {
		pkg.Name = pkg.Namespace
		pkg.Namespace = ""
	}

	if p.peek().Type == TokenAt {
		p.next()
		ver, err := p.expectIdent()
		if err == nil {
			pkg.Version = ver
		}
	}

	if p.peek().Type == TokenSemicolon {
		p.next()
	}
	return nil
}

func (p *Parser) parseInterface() (*Interface, error) {
	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}

	iface := &Interface{Name: name}

	for p.peek().Type != TokenRBrace && p.peek().Type != TokenEOF {
		tok := p.peek()
		if tok.Type == TokenIdent {
			switch tok.Value {
			case "record":
				p.next()
				rec, err := p.parseRecord()
				if err != nil {
					return nil, err
				}
				iface.Records = append(iface.Records, *rec)
			case "enum":
				p.next()
				enm, err := p.parseEnum()
				if err != nil {
					return nil, err
				}
				iface.Enums = append(iface.Enums, *enm)
			case "type":
				p.next()
				tdef, err := p.parseTypeDef()
				if err != nil {
					return nil, err
				}
				iface.TypeDefs = append(iface.TypeDefs, *tdef)
			default:
				// Function declaration: <name>: func(...) [-> ...] ;
				fn, err := p.parseFunction()
				if err != nil {
					return nil, err
				}
				iface.Functions = append(iface.Functions, *fn)
			}
		} else {
			p.next()
		}
	}

	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}

	return iface, nil
}

func (p *Parser) parseFunction() (*Function, error) {
	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}

	if _, err := p.expect(TokenColon); err != nil {
		return nil, err
	}

	funcKeyword, err := p.expectIdent()
	if err != nil || funcKeyword != "func" {
		return nil, fmt.Errorf("expected 'func', got %q", funcKeyword)
	}

	if _, err := p.expect(TokenLParen); err != nil {
		return nil, err
	}

	var params []Param
	for p.peek().Type != TokenRParen && p.peek().Type != TokenEOF {
		pName, err := p.expectIdent()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenColon); err != nil {
			return nil, err
		}
		pType, err := p.parseTypeRef()
		if err != nil {
			return nil, err
		}
		params = append(params, Param{Name: pName, Type: pType})

		if p.peek().Type == TokenComma {
			p.next()
		} else {
			break
		}
	}

	if _, err := p.expect(TokenRParen); err != nil {
		return nil, err
	}

	var resultType *TypeRef
	if p.peek().Type == TokenArrow {
		p.next()
		rt, err := p.parseTypeRef()
		if err != nil {
			return nil, err
		}
		resultType = rt
	}

	if p.peek().Type == TokenSemicolon {
		p.next()
	}

	return &Function{
		Name:    name,
		Params:  params,
		Results: resultType,
	}, nil
}

func (p *Parser) parseTypeRef() (*TypeRef, error) {
	tok := p.peek()
	if tok.Type != TokenIdent {
		return nil, fmt.Errorf("line %d: expected type name, got %q", tok.Line, tok.Value)
	}
	p.next()
	typeName := tok.Value

	switch typeName {
	case "list":
		if p.peek().Type == TokenLAngle {
			p.next()
			inner, err := p.parseTypeRef()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(TokenRAngle); err != nil {
				return nil, err
			}
			return &TypeRef{Kind: KindList, Name: "list", TypeArgs: []*TypeRef{inner}}, nil
		}
		return &TypeRef{Kind: KindList, Name: "list"}, nil

	case "option":
		if p.peek().Type == TokenLAngle {
			p.next()
			inner, err := p.parseTypeRef()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(TokenRAngle); err != nil {
				return nil, err
			}
			return &TypeRef{Kind: KindOption, Name: "option", TypeArgs: []*TypeRef{inner}}, nil
		}
		return &TypeRef{Kind: KindOption, Name: "option"}, nil

	case "result":
		var typeArgs []*TypeRef
		if p.peek().Type == TokenLAngle {
			p.next()
			// Handle result<_, E> or result<T, E> or result<T>
			if p.peek().Type == TokenIdent && p.peek().Value == "_" {
				p.next()
				typeArgs = append(typeArgs, &TypeRef{Kind: KindPrimitive, Name: "_"})
			} else {
				t1, err := p.parseTypeRef()
				if err != nil {
					return nil, err
				}
				typeArgs = append(typeArgs, t1)
			}

			if p.peek().Type == TokenComma {
				p.next()
				t2, err := p.parseTypeRef()
				if err != nil {
					return nil, err
				}
				typeArgs = append(typeArgs, t2)
			}

			if _, err := p.expect(TokenRAngle); err != nil {
				return nil, err
			}
		}
		return &TypeRef{Kind: KindResult, Name: "result", TypeArgs: typeArgs}, nil

	case "tuple":
		var typeArgs []*TypeRef
		if p.peek().Type == TokenLAngle {
			p.next()
			for p.peek().Type != TokenRAngle && p.peek().Type != TokenEOF {
				t, err := p.parseTypeRef()
				if err != nil {
					return nil, err
				}
				typeArgs = append(typeArgs, t)
				if p.peek().Type == TokenComma {
					p.next()
				} else {
					break
				}
			}
			if _, err := p.expect(TokenRAngle); err != nil {
				return nil, err
			}
		}
		return &TypeRef{Kind: KindTuple, Name: "tuple", TypeArgs: typeArgs}, nil

	case "u8", "u16", "u32", "u64", "s8", "s16", "s32", "s64", "f32", "f64", "char", "bool", "string":
		return &TypeRef{Kind: KindPrimitive, Name: typeName}, nil

	default:
		return &TypeRef{Kind: KindNamed, Name: typeName}, nil
	}
}

func (p *Parser) parseRecord() (*Record, error) {
	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}

	var fields []Field
	for p.peek().Type != TokenRBrace && p.peek().Type != TokenEOF {
		fName, err := p.expectIdent()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(TokenColon); err != nil {
			return nil, err
		}
		fType, err := p.parseTypeRef()
		if err != nil {
			return nil, err
		}
		fields = append(fields, Field{Name: fName, Type: fType})

		if p.peek().Type == TokenComma {
			p.next()
		} else {
			break
		}
	}

	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}
	return &Record{Name: name, Fields: fields}, nil
}

func (p *Parser) parseEnum() (*Enum, error) {
	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}

	var cases []EnumCase
	for p.peek().Type != TokenRBrace && p.peek().Type != TokenEOF {
		cName, err := p.expectIdent()
		if err != nil {
			return nil, err
		}
		cases = append(cases, EnumCase{Name: cName})

		if p.peek().Type == TokenComma {
			p.next()
		} else {
			break
		}
	}

	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}
	return &Enum{Name: name, Cases: cases}, nil
}

func (p *Parser) parseTypeDef() (*TypeDef, error) {
	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenEquals); err != nil {
		return nil, err
	}
	tRef, err := p.parseTypeRef()
	if err != nil {
		return nil, err
	}
	if p.peek().Type == TokenSemicolon {
		p.next()
	}
	return &TypeDef{Name: name, Type: tRef}, nil
}

func (p *Parser) parseWorld() (*World, error) {
	name, err := p.expectIdent()
	if err != nil {
		return nil, err
	}
	if _, err := p.expect(TokenLBrace); err != nil {
		return nil, err
	}

	w := &World{Name: name}

	for p.peek().Type != TokenRBrace && p.peek().Type != TokenEOF {
		tok := p.peek()
		if tok.Type == TokenIdent {
			if tok.Value == "export" {
				p.next()
				item, err := p.expectIdent()
				if err == nil {
					w.Exports = append(w.Exports, item)
				}
				if p.peek().Type == TokenSemicolon {
					p.next()
				}
			} else if tok.Value == "import" {
				p.next()
				item, err := p.expectIdent()
				if err == nil {
					w.Imports = append(w.Imports, item)
				}
				if p.peek().Type == TokenSemicolon {
					p.next()
				}
			} else {
				p.next()
			}
		} else {
			p.next()
		}
	}

	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}
	return w, nil
}

// ParseContent parses WIT content from a string.
func ParseContent(content string) (*Package, error) {
	lexer := NewLexer(content)
	var tokens []Token
	for {
		tok := lexer.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	parser := NewParser(tokens)
	return parser.Parse()
}

// ParseFile parses a single .wit file.
func ParseFile(path string) (*Package, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseContent(string(data))
}

// ParsePath parses .wit files in a file or directory, merging into a single Package AST.
func ParsePath(witPath string) (*Package, error) {
	var files []string
	fi, err := os.Stat(witPath)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		err := filepath.Walk(witPath, func(path string, info os.FileInfo, err error) error {
			if err == nil && info != nil && !info.IsDir() && strings.HasSuffix(path, ".wit") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		files = append(files, witPath)
	}

	merged := &Package{}
	for _, file := range files {
		pkg, err := ParseFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to parse %s: %w", file, err)
		}
		if merged.Namespace == "" && pkg.Namespace != "" {
			merged.Namespace = pkg.Namespace
		}
		if merged.Name == "" && pkg.Name != "" {
			merged.Name = pkg.Name
		}
		if merged.Version == "" && pkg.Version != "" {
			merged.Version = pkg.Version
		}
		merged.Interfaces = append(merged.Interfaces, pkg.Interfaces...)
		merged.Worlds = append(merged.Worlds, pkg.Worlds...)
	}

	return merged, nil
}

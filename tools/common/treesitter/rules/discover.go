package rules

import (
	"slices"
	"strings"

	"github.com/odvcencio/gotreesitter"
)

// Site describes a single candidate source mutation at an exact byte span.
type Site struct {
	File        string
	Line        int // 1-based start line
	Col         int // 1-based start column (bytes + 1)
	EndLine     int // 1-based end line
	EndCol      int // 1-based end column
	StartByte   uint32
	EndByte     uint32
	Mutator     MutatorCategory
	Function    string
	Original    string
	Replacement string
}

// Apply returns a new byte slice with the site's [StartByte:EndByte] replaced by Replacement.
func Apply(src []byte, s Site) []byte {
	out := make([]byte, 0, len(src)-int(s.EndByte-s.StartByte)+len(s.Replacement))
	out = append(out, src[:s.StartByte]...)
	out = append(out, s.Replacement...)
	out = append(out, src[s.EndByte:]...)
	return out
}

// Discover parses src using the language config's grammar and returns all candidate
// mutation sites inside non-test executable function bodies.
func (c *LanguageConfig) Discover(file string, src []byte, allowedMutators map[MutatorCategory]bool) ([]Site, error) {
	tree, lang, err := c.Parse(src)
	if err != nil {
		return nil, err
	}
	d := &discoverer{
		cfg:             c,
		lang:            lang,
		file:            file,
		src:             src,
		allowedMutators: allowedMutators,
		seen:            make(map[siteKey]bool),
	}
	d.walk(tree.RootNode(), "", false)
	return d.sites, nil
}

type siteKey struct {
	start, end  uint32
	mutator     MutatorCategory
	replacement string
}

type discoverer struct {
	cfg             *LanguageConfig
	lang            *gotreesitter.Language
	file            string
	src             []byte
	allowedMutators map[MutatorCategory]bool
	seen            map[siteKey]bool
	sites           []Site
}

func (d *discoverer) walk(n *gotreesitter.Node, currentFunc string, inFuncBody bool) {
	if n == nil {
		return
	}
	ntype := n.Type(d.lang)
	if isSkippedSubtree(ntype) || d.isTestSubtree(n, ntype) {
		return
	}
	if slices.Contains(d.cfg.Function.NodeTypes, ntype) {
		d.walkFunctionNode(n, currentFunc)
		return
	}
	if inFuncBody {
		d.collectNodeSites(n, currentFunc)
	}
	d.walkChildren(n, currentFunc, inFuncBody)
}

func (d *discoverer) walkFunctionNode(n *gotreesitter.Node, parentFunc string) {
	if d.hasSkippedParent(n) {
		return
	}
	fnName := d.extractFunctionName(n)
	if fnName != "" && d.isTestFuncName(fnName) {
		return
	}
	if fnName == "" {
		fnName = parentFunc
	}
	body := d.findFunctionBody(n)
	if body != nil {
		d.walk(body, fnName, true)
	}
}

func (d *discoverer) walkChildren(n *gotreesitter.Node, currentFunc string, inFuncBody bool) {
	count := n.ChildCount()
	for i := 0; i < count; i++ {
		d.walk(n.Child(i), currentFunc, inFuncBody)
	}
}

func (d *discoverer) collectNodeSites(n *gotreesitter.Node, currentFunc string) {
	for _, c := range d.extractSites(n, currentFunc) {
		if !d.isAllowedMutator(c.Mutator) {
			continue
		}
		k := siteKey{
			start:       c.StartByte,
			end:         c.EndByte,
			mutator:     c.Mutator,
			replacement: c.Replacement,
		}
		if !d.seen[k] {
			d.seen[k] = true
			d.sites = append(d.sites, c)
		}
	}
}

func (d *discoverer) isAllowedMutator(cat MutatorCategory) bool {
	return len(d.allowedMutators) == 0 || d.allowedMutators[cat]
}

var skippedSubtreeTypes = map[string]bool{
	"import_declaration": true, "import_spec": true, "import_statement": true,
	"import_from_statement": true, "import_header": true, "import_list": true,
	"use_declaration": true, "package_clause": true, "package_header": true,
	"type_declaration": true, "type_spec": true, "type_alias_declaration": true,
	"interface_declaration": true, "struct_item": true, "enum_item": true,
	"trait_item": true, "type_item": true, "protocol_declaration": true,
	"attribute_item": true, "inner_attribute_item": true, "annotation": true,
	"Modifiers": true, "modifiers": true, "decorator": true,
	"comment": true, "line_comment": true, "block_comment": true, "multiline_comment": true,
	"type_annotation": true, "type_arguments": true, "type_parameters": true, "generic_type": true,
}

func isSkippedSubtree(ntype string) bool {
	return skippedSubtreeTypes[ntype]
}

func (d *discoverer) hasSkippedParent(n *gotreesitter.Node) bool {
	skipParents := d.cfg.Function.SkipParents
	if len(skipParents) == 0 {
		return false
	}
	for p := n.Parent(); p != nil; p = p.Parent() {
		if slices.Contains(skipParents, p.Type(d.lang)) {
			return true
		}
	}
	return false
}

func (d *discoverer) isTestSubtree(n *gotreesitter.Node, ntype string) bool {
	if d.hasTestAttribute(n, ntype) {
		return true
	}
	return d.isTestCall(n, ntype)
}

func (d *discoverer) hasTestAttribute(n *gotreesitter.Node, ntype string) bool {
	markers := d.cfg.Test.AttrMarkers
	if len(markers) == 0 {
		return false
	}
	if ntype != "mod_item" && !slices.Contains(d.cfg.Function.NodeTypes, ntype) {
		return false
	}
	return d.hasPrecedingTestAttr(n, markers) || d.hasChildTestAttr(n, markers)
}

func (d *discoverer) isTestCall(n *gotreesitter.Node, ntype string) bool {
	if len(d.cfg.Test.CallNames) == 0 || (ntype != "call_expression" && ntype != "call") {
		return false
	}
	fnNode := n.ChildByFieldName("function", d.lang)
	if fnNode == nil && n.ChildCount() > 0 {
		fnNode = n.Child(0)
	}
	if fnNode == nil {
		return false
	}
	callee := strings.TrimSpace(fnNode.Text(d.src))
	return slices.Contains(d.cfg.Test.CallNames, callee)
}

var attrNodeTypes = map[string]bool{
	"attribute_item": true,
	"decorator":      true,
	"annotation":     true,
	"comment":        true,
	"line_comment":   true,
	"modifiers":      true,
	"attribute":      true,
}

func containsAnyMarker(text string, markers []string) bool {
	for _, m := range markers {
		if strings.Contains(text, m) {
			return true
		}
	}
	return false
}

func (d *discoverer) hasPrecedingTestAttr(n *gotreesitter.Node, markers []string) bool {
	for prev := n.PrevSibling(); prev != nil; prev = prev.PrevSibling() {
		if !attrNodeTypes[prev.Type(d.lang)] {
			return false
		}
		if containsAnyMarker(prev.Text(d.src), markers) {
			return true
		}
	}
	return false
}

func (d *discoverer) hasChildTestAttr(n *gotreesitter.Node, markers []string) bool {
	count := n.ChildCount()
	for i := 0; i < count; i++ {
		c := n.Child(i)
		if attrNodeTypes[c.Type(d.lang)] && containsAnyMarker(c.Text(d.src), markers) {
			return true
		}
	}
	return false
}

func (d *discoverer) isTestFuncName(name string) bool {
	for _, pref := range d.cfg.Test.FuncNamePrefs {
		if strings.HasPrefix(name, pref) {
			return true
		}
	}
	return false
}

var identifierNodeTypes = map[string]bool{
	"identifier":          true,
	"simple_identifier":   true,
	"field_identifier":    true,
	"property_identifier": true,
}

var varDeclParentTypes = map[string]bool{
	"variable_declarator":   true,
	"short_var_declaration": true,
	"property_declaration":  true,
}

func (d *discoverer) extractFunctionName(n *gotreesitter.Node) string {
	if name := d.fieldText(n, d.cfg.Function.NameField); name != "" {
		return name
	}
	if name := d.firstIdentifierChildText(n); name != "" {
		return name
	}
	return d.parentDeclaratorName(n)
}

func (d *discoverer) fieldText(n *gotreesitter.Node, field string) string {
	if field == "" {
		return ""
	}
	child := n.ChildByFieldName(field, d.lang)
	if child == nil {
		return ""
	}
	return strings.TrimSpace(child.Text(d.src))
}

func (d *discoverer) firstIdentifierChildText(n *gotreesitter.Node) string {
	count := n.ChildCount()
	for i := 0; i < count; i++ {
		c := n.Child(i)
		if identifierNodeTypes[c.Type(d.lang)] {
			return strings.TrimSpace(c.Text(d.src))
		}
	}
	return ""
}

func (d *discoverer) parentDeclaratorName(n *gotreesitter.Node) string {
	p := n.Parent()
	if p == nil || !varDeclParentTypes[p.Type(d.lang)] {
		return ""
	}
	return d.fieldText(p, "name")
}

func (d *discoverer) findFunctionBody(n *gotreesitter.Node) *gotreesitter.Node {
	if d.cfg.Function.BodyField != "" {
		if body := n.ChildByFieldName(d.cfg.Function.BodyField, d.lang); body != nil {
			return body
		}
	}
	count := n.ChildCount()
	for i := 0; i < count; i++ {
		c := n.Child(i)
		if slices.Contains(d.cfg.Function.BodyTypes, c.Type(d.lang)) {
			return c
		}
	}
	return nil
}

func (d *discoverer) extractSites(n *gotreesitter.Node, currentFunc string) []Site {
	ntype := n.Type(d.lang)
	repl := d.cfg.Replacement
	var out []Site

	if slices.Contains(repl.BinaryNodes, ntype) {
		out = append(out, d.extractBinarySites(n, currentFunc)...)
	}
	if slices.Contains(repl.UnaryNodes, ntype) {
		out = append(out, d.extractUnarySites(n, currentFunc)...)
	}
	if slices.Contains(repl.UpdateNodes, ntype) {
		out = append(out, d.extractUpdateSites(n, currentFunc)...)
	}
	if slices.Contains(repl.AssignNodes, ntype) {
		out = append(out, d.extractAssignSites(n, currentFunc)...)
	}
	if d.isBoolLiteralNode(n, ntype) {
		out = append(out, d.extractBoolSite(n, currentFunc)...)
	}
	return out
}

func (d *discoverer) extractBinarySites(n *gotreesitter.Node, currentFunc string) []Site {
	opNodes, hasStringOperand := d.scanBinaryChildren(n)
	var sites []Site
	for _, opNode := range opNodes {
		opText := strings.TrimSpace(opNode.Text(d.src))
		if opText == "+" && hasStringOperand {
			continue
		}
		spec := d.cfg.Replacement.BinaryOps[opText]
		for _, r := range spec.Replacements {
			sites = append(sites, makeSite(opNode, d.file, currentFunc, spec.Category, opText, r))
		}
	}
	return sites
}

func (d *discoverer) scanBinaryChildren(n *gotreesitter.Node) ([]*gotreesitter.Node, bool) {
	repl := d.cfg.Replacement
	var opNodes []*gotreesitter.Node
	hasString := false

	count := n.ChildCount()
	for i := 0; i < count; i++ {
		c := n.Child(i)
		ctype := c.Type(d.lang)
		if slices.Contains(repl.StringNodes, ctype) {
			hasString = true
		}
		if d.isBinaryOperatorChild(c, ctype) {
			opNodes = append(opNodes, c)
		}
	}
	return opNodes, hasString
}

func (d *discoverer) isBinaryOperatorChild(c *gotreesitter.Node, ctype string) bool {
	if c.IsNamed() && ctype != "custom_operator" && ctype != "comparison_operator" {
		return false
	}
	text := strings.TrimSpace(c.Text(d.src))
	_, ok := d.cfg.Replacement.BinaryOps[text]
	return ok
}

func (d *discoverer) extractUnarySites(n *gotreesitter.Node, currentFunc string) []Site {
	if n.ChildCount() < 2 {
		return nil
	}
	opText := strings.TrimSpace(n.Child(0).Text(d.src))
	if !slices.Contains(d.cfg.Replacement.UnaryOps, opText) {
		return nil
	}
	operandText := strings.TrimSpace(n.Child(n.ChildCount() - 1).Text(d.src))
	fullText := strings.TrimSpace(n.Text(d.src))
	if operandText == "" || fullText == operandText {
		return nil
	}
	return []Site{makeSite(n, d.file, currentFunc, MutatorUnary, fullText, operandText)}
}

func (d *discoverer) extractUpdateSites(n *gotreesitter.Node, currentFunc string) []Site {
	if len(d.cfg.Replacement.UpdateOps) == 0 {
		return nil
	}
	count := n.ChildCount()
	for i := 0; i < count; i++ {
		c := n.Child(i)
		text := strings.TrimSpace(c.Text(d.src))
		if repl, ok := d.cfg.Replacement.UpdateOps[text]; ok {
			return []Site{makeSite(c, d.file, currentFunc, MutatorUnary, text, repl)}
		}
	}
	return nil
}

func (d *discoverer) extractAssignSites(n *gotreesitter.Node, currentFunc string) []Site {
	if len(d.cfg.Replacement.AssignOps) == 0 {
		return nil
	}
	opNode, opText, rhsIsString := d.scanAssignChildren(n)
	if opNode == nil || (opText == "+=" && rhsIsString) {
		return nil
	}
	return []Site{makeSite(opNode, d.file, currentFunc, MutatorAssignment, opText, d.cfg.Replacement.AssignOps[opText])}
}

func (d *discoverer) scanAssignChildren(n *gotreesitter.Node) (*gotreesitter.Node, string, bool) {
	var opNode *gotreesitter.Node
	var opText string
	rhsIsString := false

	count := n.ChildCount()
	for i := 0; i < count; i++ {
		c := n.Child(i)
		if strings.Contains(c.Type(d.lang), "string") {
			rhsIsString = true
		}
		text := strings.TrimSpace(c.Text(d.src))
		if _, ok := d.cfg.Replacement.AssignOps[text]; ok {
			opNode = c
			opText = text
		}
	}
	return opNode, opText, rhsIsString
}

func (d *discoverer) isBoolLiteralNode(n *gotreesitter.Node, ntype string) bool {
	if !n.IsNamed() {
		return false
	}
	repl := d.cfg.Replacement
	if ntype != repl.BoolTrueNode && ntype != repl.BoolFalseNode && ntype != "boolean_literal" {
		return false
	}
	if p := n.Parent(); p != nil && p.Type(d.lang) == "boolean_literal" {
		return false
	}
	_, ok := repl.BoolLiterals[strings.TrimSpace(n.Text(d.src))]
	return ok
}

func (d *discoverer) extractBoolSite(n *gotreesitter.Node, currentFunc string) []Site {
	text := strings.TrimSpace(n.Text(d.src))
	repl, ok := d.cfg.Replacement.BoolLiterals[text]
	if !ok {
		return nil
	}
	return []Site{makeSite(n, d.file, currentFunc, MutatorBooleanLiteral, text, repl)}
}

func makeSite(n *gotreesitter.Node, file, currentFunc string, cat MutatorCategory, orig, repl string) Site {
	sp := n.StartPoint()
	ep := n.EndPoint()
	return Site{
		File:        file,
		Line:        int(sp.Row) + 1,
		Col:         int(sp.Column) + 1,
		EndLine:     int(ep.Row) + 1,
		EndCol:      int(ep.Column) + 1,
		StartByte:   n.StartByte(),
		EndByte:     n.EndByte(),
		Mutator:     cat,
		Function:    currentFunc,
		Original:    orig,
		Replacement: repl,
	}
}

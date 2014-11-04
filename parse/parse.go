// Copyright 2014 Joel Scoble (github:mohae). All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// This code is based on code originally written by The Go Authors.
// Their copyright notice immediately follows this one.

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package parse builds parse trees for templates as defined by text/template
// and html/template. Clients should use those packages to construct templates
// rather than this one, which provides shared internal data structures not
// intended for general use.
package parse

import (
	"bytes"
	"fmt"
	_"log"
	"runtime"
	_"strconv"
	"strings"
)

// Tree is the representation of a single parsed template.
type Tree struct {
	Name      string    // name of the template represented by the tree.
	ParseName string    // name of the top-level template during parsing, for error messages.
	Root      *ListNode // top-level root of the tree.
	text      string    // text parsed to create the template (or its parent)
	// Parsing only; cleared after parse.
	funcs     []map[string]interface{}
	lex       *lexer
	token     [3]item // three-token lookahead for parser.
	peekCount int
	variables      []string // variables defined at the moment.
	isStandalone bool  // for Standalone Comment determination
	history    [3]item // two-token history for parser (not include current.
	glanceCount int    // glancCount is to history as peekCount is to token
}

// Copy returns a copy of the Tree. Any parsing state is discarded.
func (t *Tree) Copy() *Tree {
	if t == nil {
		return nil
	}
	return &Tree{
		Name:      t.Name,
		ParseName: t.ParseName,
		Root:      t.Root.CopyList(),
		text:      t.text,
	}
}

// Parse returns a map from template name to parse.Tree, created by parsing the
// templates described in the argument string. The top-level template will be
// given the specified name. If an error is encountered, parsing stops and an
// empty map is returned with the error.
func Parse(name, text, oTag string, cTag string, funcs ...map[string]interface{}) (treeSet map[string]*Tree, err error) {
	treeSet = make(map[string]*Tree)
	t := New(name)
	t.text = text
	_, err = t.Parse(text, OTag, CTag, treeSet, funcs...)
	return
}

// next returns the next token.
func (t *Tree) next() item {
	if t.peekCount > 0 {
		t.peekCount--
	} else {
		t.token[0] = t.lex.nextItem()
	}
	return t.token[t.peekCount]
}

// backup backs the input stream up one token.
func (t *Tree) backup() {
	t.peekCount++
}

// backup2 backs the input stream up two tokens.
// The zeroth token is already there.
func (t *Tree) backup2(t1 item) {
	t.token[1] = t1
	t.peekCount = 2
}

// backup3 backs the input stream up three tokens
// The zeroth token is already there.
func (t *Tree) backup3(t2, t1 item) { // Reverse order: we're pushing back.
	t.token[1] = t1
	t.token[2] = t2
	t.peekCount = 3
}

// peek returns but does not consume the next token.
func (t *Tree) peek() item {
	if t.peekCount > 0 {
		return t.token[t.peekCount-1]
	}
	t.peekCount = 1
	t.token[0] = t.lex.nextItem()
	return t.token[0]
}

// nextNonSpace returns the next non-space token.
func (t *Tree) nextNonSpace() (token item) {
	for {
		token = t.next()
		if token.typ != itemSpace {
			break
		}
	}
	return token
}

// peekNonSpace returns but does not consume the next non-space token.
func (t *Tree) peekNonSpace() (token item) {
	for {
		token = t.next()
		if token.typ != itemSpace {
			break
		}
	}
	t.backup()
	return token
}

// nextCTag returns the next CTag, everything in between is skipped
func (t *Tree) nextCTag() (token item) {
	for {
		token = t.next()
		logger.Debugf("nextCTag: %s %q\n", ItemStrings[token.typ], token.value)
//		t.addHistory(token)
		if token.typ == itemCTag {
			// move beyond it and then break
//			t.addHistory(token)
			break
		}
	}
	return token
}

// Parsing.

// New allocates a new parse tree with the given name.
func New(name string, funcs ...map[string]interface{}) *Tree {
	return &Tree{
		Name:  name,
		funcs: funcs,
	}
}

// ErrorContext returns a textual representation of the location of the node in the input text.
func (t *Tree) ErrorContext(n Node) (location, context string) {
	pos := int(n.Position())
	text := t.text[:pos]
	byteNum := strings.LastIndex(text, "\n")
	if byteNum == -1 {
		byteNum = pos // On first line.
	} else {
		byteNum++ // After the newline.
		byteNum = pos - byteNum
	}
	lineNum := 1 + strings.Count(text, "\n")
	context = n.String()
	if len(context) > 20 {
		context = fmt.Sprintf("%.20s...", context)
	}
	return fmt.Sprintf("%s:%d:%d", t.ParseName, lineNum, byteNum), context 
}

// errorf formats the error and terminates processing.
func (t *Tree) errorf(format string, args ...interface{}) {
	t.Root = nil
	format = fmt.Sprintf("template: %s:%d: %s\n", t.ParseName, t.lex.lineNumber(), format)
	panic(fmt.Errorf(format, args...))
}

// error terminates processing.
func (t *Tree) error(err error) {
	t.errorf("%s\n", err)
}

// expect consumes the next token and guarantees it has the required type.
func (t *Tree) expect(expected itemType, context string) item {
	token := t.nextNonSpace()
	if token.typ != expected {
		t.unexpected(token, context)
	}
	return token
}

// expectOneOf consumes the next token and guarantees it has one of the required types.
func (t *Tree) expectOneOf(expected1, expected2 itemType, context string) item {
	token := t.nextNonSpace()
	if token.typ != expected1 && token.typ != expected2{
		t.unexpected(token, context)
	}
	return token
}

// unexpected complains about the token and terminates processing.
func (t *Tree) unexpected(token item, context string) {
	t.errorf("unexpected %s in %s", token, context)
}

// recover is the handler that turns panics into returns from the top level of Parse.
func (t *Tree) recover(errp *error) {
	e := recover()
	if e != nil {
		if _, ok := e.(runtime.Error); ok {
			panic(e)
		}
		if t != nil {
			t.stopParse()
		}
		*errp = e.(error)
	}
	return
}

// startParse initializes the parser, using the lexer.
func (t *Tree) startParse(funcs []map[string]interface{}, lex *lexer) {
	t.Root = nil
	t.lex = lex
	t.funcs = funcs
}

// stopParse terminates parsing.
func (t *Tree) stopParse() {
	t.lex = nil
	t.variables = nil
	t.funcs = nil
}

// Parse parses the template definition string to construct a representation of
// the template for execution. If either action delimiter string is empty, the
// default ("{{" or "}}") is used. Embedded template definitions are added to
// the treeSet map.
func (t *Tree) Parse(text, leftDelim, rightDelim string, treeSet map[string]*Tree, funcs ...map[string]interface{}) (tree *Tree, err error) {
	defer t.recover(&err)
	t.ParseName = t.Name
	t.startParse(funcs, lex(t.Name, text, leftDelim, rightDelim))
	t.text = text
	t.parse(treeSet)
	t.add(treeSet)
	t.stopParse()
	return t, nil
}

// add adds tree to the treeSet.
func (t *Tree) add(treeSet map[string]*Tree) {
	for k, v := range treeSet {
		logger.Debugf("addTreeSet %s: %+v\n", k, v)
	}
	tree := treeSet[t.Name]
	if tree == nil || IsEmptyTree(tree.Root) {
		treeSet[t.Name] = t
		return
	}
	if !IsEmptyTree(t.Root) {
		t.errorf("template: multiple definition of template %q", t.Name)
	}
}

// IsEmptyTree reports whether this tree (node) is empty of everything but space.
func IsEmptyTree(n Node) bool {
	switch n := n.(type) {
	case nil:
		return true
	case *ActionNode:
	case *IfNode:
	case *ListNode:
		for _, node := range n.Nodes {
			if !IsEmptyTree(node) {
				return false
			}
		}
		return true
	case *RangeNode:
	case *TemplateNode:
	case *TextNode:
		return len(bytes.TrimSpace(n.Text)) == 0
	case *WithNode:
	default:
		panic("unknown node: " + n.String())
	}
	return false
}

// parse is the top-level parser for a template, essentially the same
// as itemList except it also parses {{}} stuff, or  the {{}} equiv.
// It runs to EOF.
func (t *Tree) parse(treeSet map[string]*Tree) (next Node) {
	t.Root = newList(t.peek().pos)
	for t.peek().typ != EOF {
		it := t.peek()
		if isOTag(it.typ) {
			tag := t.next() // move beyond oTag
			if tag.typ == tagInverted || tag.typ == tagSection {
				newT := New("section") // name will be updated once we know it
				newT.text = t.text
				newT.startParse(t.funcs, t.lex)
				newT.parseSection(treeSet)
				continue
			}
			t.backup()
		}
		n := t.textOrAction()
		if n == nil {
			continue // skip nils, lexed items that are not to be emitted
		}
		logger.Debugf("n: %s %q\n", NodeStrings[n.Type()], n.String())
		if n.Type() == NodeEnd {
			t.errorf("zunexpected %s\n", n)
		}
		t.Root.append(n)
	}
	return nil
}

// parseSection parses a {{#section}} ...  {{/section}} or {{^section}} ..
// {{/section}} and installs the definition in the treeSet map.  The "section" 
// keyword has already been scanned.
func (t *Tree) parseSection(treeSet map[string]*Tree) {
	const context = "section"
	t.Name = t.expectOneOf(itemDiscard, itemIdentifier, context).value
	t.expect(itemCTag, context)
	var end Node
	t.Root, end = t.itemList()
	if end.Type() != NodeEnd {
		t.errorf("unexpected %s in %s", end, context)
	}
	t.add(treeSet)
	t.stopParse()
}

// itemList:
//	textOrAction*
// Terminates at {{/section}}
func (t *Tree) itemList() (list *ListNode, next Node) {
	list = newList(t.peekNonSpace().pos)
	for t.peekNonSpace().typ != EOF {
		n := t.textOrAction()
		logger.Debugf("%s %s\n", NodeStrings[n.Type()], n.String())
		switch n.Type() {
		case NodeEnd:
//			t.nextCTag()
			return list, n
		}
		list.append(n)
	}
	t.errorf("unexpected EOF")
	return
}

// textOrAction 
func (t *Tree) textOrAction() Node {
	token := t.next()
	t.addHistory(token)
	logger.Debugf("textOrAction %s %q\n", ItemStrings[token.typ], token.value)
	switch token.typ {
	case itemText:
		return newText(token.pos, token.value)
	case itemSpace:
		return newSpace(token.pos, token.value)
	case itemNL:
		return newNL(token.pos, token.value)
	case itemCR:
		return newCR(token.pos, token.value)
	case itemDiscard:
		return nil
	case itemCTag:
		return nil
	default: // everything else goes to action, unexepected tokens raise errors there
		// backup because it gets reconsumed 
		return t.action()
	}
	return nil
}


// Action:
//	control
//	command ("|" command)*
// Left delim is past. Now get actions.
// First word could be a keyword such as range.
func (t *Tree) action() (n Node) {
	token := t.history[0]  // current element
	switch  token.typ {
	case itemDiscard:
		return nil
	case tagComment:
		logger.Debugf("action: %s\n", ItemStrings[token.typ])
		t.processComment()
		return nil
	case tagEndSection:
		return t.endSectionControl()
	case tagPartial:
		n = t.partialControl()
		return n
//	case itemCTag:
//		t.nextCTag()
//		return nil
	case tagΔDelimiter: 
		t.processΔDelimiter() // skip ctag
		return nil
	}
	t.backup()
	logger.Infof("action post Backup: %s %q\npeek(): %d %s %q\n", ItemStrings[token.typ], token.value, int(t.peek().pos), ItemStrings[t.peek().typ], t.peek().value)
	// Do not pop variables; they persist until "end".
	return newAction(token.typ, t.peek().pos, t.lex.lineNumber(), t.pipeline("command"))
}

// adds an item to the history and returns the number of items in the history.
// only 3 items max, including current are in history.
func (t *Tree) addHistory(i item) int {
	if t.glanceCount == 2 {
		t.history[2] = t.history[1]
	}
	if t.glanceCount >= 1 {
		t.history[1] = t.history[0]
	}
	t.history[0] = i
	if t.glanceCount < 2 {
		t.glanceCount++
	}
	return t.glanceCount
}

// newEscaped is an escaped variable: {{variable}}
func (t *Tree) newEscaped() (n Node) {
	// Do not pop variables; they persist until "end".
//	i := t.next()
	i := t.peek()
	logger.Debugf("newUnescaped %d %q\n", int(t.peek().pos), i.value)
 	n = newVariable(tagEscaped, t.peek().pos, i.value)
//	n = newIdentifier(Node, s)
	// skip the cTag
	t.next()
	return 
}

// newUnescaped is an unescaped variable: {{{variable}}} | {{&variable}}
func (t *Tree) newUnescaped() (n Node) {
	// Do not pop variables; they persist until "end".
//	i := t.next()
	i := t.peek()
	logger.Debugf("newUnescaped %d %q\n", int(t.peek().pos), i.value)
 	n = newVariable(tagUnescaped, t.peek().pos, i.value)
//	n = newIdentifier(NodeUnescaped, s)
	t.next()
	return
}

// processComment processes comments.  Comments can be inline or stand-alone.
// Stand-alone comments have one of the following properties:
//      \n   {{!comment}}  EOF
//      \n   {{!comment}}  \n
//      FileBegin   {{!comment}}  \n
// for comments, spaces are ignored
func (t *Tree) processComment() {
	// preTagNL/postTagNL carry information about
	var standalone, popRoot bool
	// look back at the root stack to see if this is a standalone comment
	// All we can do here is eliminate whether it is a standalone or not,
	// the final determination depends on other factors too:
	// A standalone begins a text, only has spaces prior to the comment, 
	// has a newline immediately before the comment, or only has spaces
	// between a newline preceeding the comment and the comment begin.
	if t.glanceCount == 0 {
		standalone = true 	// could be either at this point
		goto skipInternal
	}
	if t.history[1].typ == itemNL {
		standalone = true
		popRoot = false
	}
	if t.history[1].typ == itemSpace {
		if t.history[2].typ != itemNL && t.history[2] != itemEmpty {
			standalone = false
			popRoot = false
			goto skipInternal
		}
		standalone = true
		popRoot = true
	}
skipInternal:
	// Anything within the brackets gets skipped.
	_ = t.nextCTag()
	// Now look forward to see if its a standalone: i.e., comment ends the 
	// text, the close tag is only followed by spaces. the close tag is only
	// followed by a newline, or the close tage is only followed by spaces 
	// and a newline. Anything after the next newline does not affect it.
	// if there are non-characters immediately following, it might be inlin
	if standalone {
		// peek ahead up to two
		nxt1 := t.peek()
		if nxt1.typ == EOF {
			// skip to standalone elision
			goto elide
		}
		if nxt1.typ == itemNL {
			// this needs to be consumed
			_ = t.next()
			goto elide
		}
		if nxt1.typ == itemCR {
			// this needs to be consumed, along with the itemLF
			_ = t.next()
			if t.peek().typ == itemNL {
				_ = t.next()
			}
			goto elide
		}
		t.next()
		nxt2 := t.peek()
		if nxt1.typ == itemSpace && nxt2.typ == itemNL {
			t.next()
			goto elide
		}
		t.backup()
	}
	// If its a standalone, clean up around it
elide:
	if popRoot {
 		t.popRoot()
	}
}

//
func (t *Tree) processΔDelimiter() {
	// preTagNL/postTagNL carry information about
	var standalone, popRoot bool
	// look back at the root stack to see if this is a standalone comment
	// All we can do here is eliminate whether it is a standalone or not,
	// the final determination depends on other factors too:
	// A standalone begins a text, only has spaces prior to the comment, 
	// has a newline immediately before the comment, or only has spaces
	// between a newline preceeding the comment and the comment begin.
	if t.glanceCount == 0 {
		standalone = true 	// could be either at this point
		goto skipInternal
	}
	if t.history[1].typ == itemSpace {
		standalone = true
		popRoot = true
		var itemEmpty item
		if t.history[2].typ != itemNL && t.history[2] != itemEmpty {
			standalone = false
			popRoot = false
			goto skipInternal
		}
	}
	if t.history[1].typ == itemNL {
		standalone = true
		popRoot = false
	}
skipInternal:
	// Anything within the brackets gets skipped.
	_ = t.nextCTag()
	// Now look forward to see if its a standalone: i.e., comment ends the 
	// text, the close tag is only followed by spaces. the close tag is only
	// followed by a newline, or the close tage is only followed by spaces 
	// and a newline. Anything after the next newline does not affect it.
	// if there are non-characters immediately following, it might be inlin
	if standalone {
		// peek ahead up to two
		nxt1 := t.peek()
		if nxt1.typ == EOF {
			// skip to standalone elision
			goto elide
		}
		if nxt1.typ == itemNL {
			// this needs to be consumed
			_ = t.next()
			goto elide
		}
		if nxt1.typ == itemCR {
			// this needs to be consumed, along with the itemLF
			_ = t.next()
			if t.peek().typ == itemNL {
				_ = t.next()
			}
			goto elide
		}
		t.next()
		nxt2 := t.peek()
		if nxt1.typ == itemSpace && nxt2.typ == itemNL {
			t.next()
			goto elide
		}
		t.backup()
	}
	// If its a standalone, clean up around it
elide:
	if popRoot {
 		t.popRoot()
	}
/*
	var atSOF, atEOF, preStandalone, postStandalone bool
	// preTagNL/postTagNL carry information about
	atSOF = t.isSOF()	// see if we are at start of file
	preStandalone = t.preStandalone() 
	// skip everything within brackets
	t.nextCTag()
	logger.Debugf("postCTag: %s %q\n", ItemStrings[t.peek().typ], t.peek().value)
	// If its a standalone, clean up around it
	if  atSOF || preStandalone {
		atEOF = t.isEOF() // see if we are at eof
		postStandalone = t.postStandalone()		
	}
	// Probably doesn't normally happen, but if this tree is just an elided tag,
	// we're done
	if atSOF && atEOF {
		return nil
	}
	logger.Debugf("atSOF: %s\tatEOF: %s\tpreStandalone: %s\tpostStandalone: %s\n", strconv.FormatBool(atSOF), strconv.FormatBool(atEOF), strconv.FormatBool(preStandalone), strconv.FormatBool(postStandalone))
	// see about processing the lead stuff first
	if preStandalone && (postStandalone || atEOF) {
		var i int
		// rewind until history is done or NL is encountered.
		for i = 1; i < t.glanceCount; i++ {
			logger.Debugf("rewind %d %s %q\n", i, ItemStrings[t.history[i].typ], t.history[i].value)
			// remove up until previous nl
			if t.history[i].typ == itemNL {
				i--
				break
			}
		}
		if i > 0 {
			t.popVars(i) // discard prior items
		}
	}
	logger.Debugf("ΔDelimiter peek: %d %q\n", int(t.peek().typ), t.peek().value)
	// if its a standalone, do process {{}} element processing
	if postStandalone && (preStandalone || atSOF) {
		logger.Debugf("ΔDelimiter Post process\n")
		// next nonSpace should be nl; otherwise error
		item := t.nextNonSpace()
		logger.Debugf("nextNonSpace %s %s\n", ItemStrings[item.typ], item.value)
		if item.typ != itemNL && t.nextNonSpace().typ != itemCR {
			if item.typ == ERROR {
				return nil
			}
			err := fmt.Errorf("rollie ΔDelimiter: expected newline, got %s\n", ItemStrings[item.typ])
			logger.Error(err)
			return err
		}
	}
	return nil
*/
}

// isEOF determines whether or not we are at EOF
func (t *Tree) isEOF() bool {
	if t.peek().typ == EOF {
		return true
	}
	return false
}

// isSOF determines whether or not we are at the SOF
func (t *Tree) isSOF() bool {
	if t.glanceCount <= 1 { // history[0] == current item, so glanceCount==1 (TODO I should rework this)
		return true
	}
	return false
}

func (t *Tree) preStandalone() bool {
	// look back at the root stack to see if this is a standalone comment
	// All we can do here is eliminate whether it is a standalone or not,
	// the final determination depends on other factors too:
	// A standalone begins a text, only has spaces prior to the comment, 
	// has a newline immediately before the comment, or only has spaces
	// between a newline preceeding the comment and the comment begin.
	if t.glanceCount == 1 {
		return true
	}
	if t.history[1].typ == itemNL { //if previos item is nl is true
		return true
	}
	if t.history[1].typ == itemSpace {
		if t.glanceCount <= 2 {
			return false
		}
		if  t.history[2].typ != itemNL && t.history[2] != itemEmpty {
			return false
		}
		return true
	}
	return false
//	logger.Debugf("%s:%s %s:%s %s:%s\n", ItemStrings[t.history[2].typ], t.history[2].value, ItemStrings[t.history[1].typ], t.history[1].value, ItemStrings[t.history[0].typ], t.history[0].value)
}

// postStandalone i.e., comment ends the 
// text, the close tag is only followed by spaces. the close tag is only
// followed by a newline, or the close tage is only followed by spaces 
// and a newline. Anything after the next newline does not affect it.
// if there are non-characters immediately following, it might be inlin
func (t *Tree) postStandalone() bool {
	// return to original position
	defer t.backup()
	typ := t.peek().typ
	logger.Debugf("postStandalone %s %q\n", ItemStrings[typ], t.peek().value)
	if typ == EOF {
		return true
	}
	if  typ == itemNL || typ == itemCR {
		return true
	}
	if typ == itemSpace {
		t.next()
		defer t.backup()
		if t.peek().typ == itemNL || t.peek().typ == itemCR {
			return true
		}
	}
	return false
}

// pops one element from the root.
func (t *Tree) popRoot() {
	t.Root.Nodes = t.Root.Nodes[0:len(t.Root.Nodes)-1]
}

// Pipeline:
//	declarations? command ('|' command)*
func (t *Tree) pipeline(context string) (pipe *PipeNode) {
	var decl []*VariableNode
	t.next() // consume the oTag
	pos := t.peekNonSpace().pos
	logger.Debugf("context: %+v\tp:%s %d v: %s\n", context, ItemStrings[t.peekNonSpace().typ], int(pos), t.peekNonSpace().value)
	// Are there declarations?
	for {
		if v := t.peekNonSpace(); v.typ ==  identEscaped || v.typ == identUnescaped {
			_ = t.nextNonSpace()
			variable := newVariable(v.typ, v.pos, v.value)
			decl = append(decl, variable)
			t.variables = append(t.variables, v.value)
			t.backup()
//			t.variables()
		}
		break
	}
	pipe = newPipeline(pos, t.lex.lineNumber(), decl)
	tok := t.peekNonSpace()
	logger.Debugf("%s %s\n", ItemStrings[tok.typ], tok.value)
	for {
		switch token := t.nextNonSpace(); token.typ {
		case itemCTag:
			if len(pipe.Cmds) == 0 {
				t.errorf("missing value for %s", context)
			}
			if token.typ == itemCTag {
				t.backup()
			}
			return
		case identEscaped, identUnescaped, itemIdentifier:
			t.backup()
			pipe.append(t.command())
		default:
			logger.Criticalf("%s %s\n", ItemStrings[tok.typ], tok.value)
			t.unexpected(token, context)
		}
	}
}

func (t *Tree) commentControl() Node{
	logger.Debugf("commentControl: %+v\n", t)
	return newComment(t.expect(itemCTag, "end").pos, t.lex.lineNumber(), t.pipeline("tag"))
}

// End:
//	{{/section}} where section is the value corresponding with the
//                   open tag: {{!section}} |  {{^section}}
func (t *Tree) endSectionControl() Node {
	// consume the end
	item := t.next()
	logger.Debugf("end section: %+v\n", item)
	return newEnd(t.expect(itemCTag, t.Name).pos, item.value)
}

// Escaped variable:
//	{{variable}}
func (t *Tree) escapedControl() Node {
	return newVariable(tagEscaped, t.peek().pos, t.peek().value)
}

// Unescaped variable:
//	{{{variable}}}
// or
//      {{&variable}}
func (t *Tree) unescapedControl() Node {
	return newVariable(tagUnescaped, t.peek().pos, t.peek().value)
}

// partial
//	{{>partial}}
func (t *Tree) partialControl() Node {
	return newPartial(t.peek().pos, t.peek().value)
}


// command:
//	operand (space operand)*
// space-separated arguments up to a pipeline character or right delimiter.
// we consume the pipe character but leave the right delim to terminate the action.
func (t *Tree) command() *CommandNode {
	cmd := newCommand(t.peekNonSpace().pos)
	for {
		tok := t.peekNonSpace() // skip leading spaces.
		logger.Infof("command: %s %d %s\n", ItemStrings[tok.typ], int(tok.pos), tok.value)
		operand := t.operand()
		if operand != nil {
			cmd.append(operand)
		}
		switch token := t.next(); token.typ {
		case itemSpace:
			continue
		case ERROR:
			t.errorf("%s", token.value)
		case itemCTag:
			t.backup()
		case itemPipe:
			t.errorf("itemPipe in command, not implemented: parse.goL708")
		case identEscaped:
			
		case identUnescaped:
		default:
			t.errorf("unexpected %s in operand; missing space?", token)
		}
		break
	}
	if len(cmd.Args) == 0 {
		t.errorf("empty command")
	}
	return cmd
}

/*
// Template:
//	{{template stringValue pipeline}}
// Template keyword is past.  The name must be something that can evaluate
// to a string.
func (t *Tree) templateControl() Node {
	var name string
	token := t.nextNonSpace()
	switch token.typ {
	case itemString, itemRawString:
		s, err := strconv.Unquote(token.value)
		if err != nil {
			t.error(err)
		}
		name = s
	default:
		t.unexpected(token, "template invocation")
	}
	var pipe *PipeNode
	if t.nextNonSpace().typ != itemCTag {
		t.backup()
		// Do not pop variables; they persist until "end".
		pipe = t.pipeline("template")
	}
	return newTemplate(token.pos, t.lex.lineNumber(), name, pipe)
}

// Else:
//	{{else}}
// Else keyword is past.
func (t *Tree) elseControl() Node {
	// Special case for "else if".
	peek := t.peekNonSpace()
	if peek.typ == itemIf {
		// We see "{{else if ... " but in effect rewrite it to {{else}}{{if ... ".
		return newElse(peek.pos, t.lex.lineNumber())
	}
	return newElse(t.expect(itemCTag, "else").pos, t.lex.lineNumber())
}
*/
func isOTag(i itemType) bool {
	logger.Debugf("%s\n", ItemStrings[i])
	if i == tagEscaped || i == tagUnescaped || i == tagSection || i == tagInverted || i == tagPartial || i ==  tagParent || i == tagΔDelimiter {
		return true
	}
	return false
}

// operand:
//	term .Field*
// An operand is a space-separated component of a command,
// a term possibly followed by field accesses.
// A nil return means the next item is not an operand.
func (t *Tree) operand() Node {
	node := t.term()
	if node == nil {
		return nil
	}
	logger.Debugf("operand node: %s %+v\n", NodeStrings[node.Type()], node)
	token := t.peek()
	logger.Debugf("%+v\n", token)
	if t.peek().typ == identEscaped || t.peek().typ == identUnescaped {
		lastTyp := t.peek().typ
		chain := newChain(t.peek().pos, node)
		for t.peek().typ == identEscaped || t.peek().typ == identUnescaped {
			lastTyp = t.peek().typ
			chain.Add(t.next().value)
		}
		// Compatibility with original API: If the term is of type NodeField
		// or NodeVariable, just put more fields on the original.
		// Otherwise, keep the Chain node.
		// TODO: Switch to Chains always when we can.
		logger.Debugf("%s\n", ItemStrings[lastTyp])
		switch node.Type() {
		case NodeVariable:
			node = newVariable(identEscaped, chain.Position(), chain.String())
		default:
			node = chain
		}
	}

	return node
}

// term:
//	literal (number, string, nil, boolean)
//	function (identifier)
//	.
//	.Field
//	$
//	'(' pipeline ')'
// A term is a simple "expression".
// A nil return means the next item is not a term.
func (t *Tree) term() Node {
	token := t.nextNonSpace()
	logger.Debugf("term: %s %s\n", ItemStrings[token.typ], token.value)
	switch token.typ {
	case ERROR:
		t.errorf("%s", token.value)
//	case itemIdentifier:
//		if !t.hasFunction(token.value) {
//			t.errorf("function %q not defined", token.value)
//		}
//		return NewIdentifier(NodeIdentifier, token.value)
	case identEscaped:
		logger.Debugf("call useVar: %s %s\n", ItemStrings[token.typ], token.value)
		return t.useVar(tagEscaped, token.pos, token.value)
	case identUnescaped:
		return t.useVar(tagUnescaped, token.pos, token.value)
	}
	t.backup()
	return nil
}
/*
// hasFunction reports if a function name exists in the Tree's maps.
func (t *Tree) hasFunction(name string) bool {
	for _, funcMap := range t.funcs {
		if funcMap == nil {
			continue
		}
		if funcMap[name] != nil {
			return true
		}
	}
	return false
}
*/

// popVars trims the variables by the specified amount
func (t *Tree) popVars(n int) {
	t.variables = t.variables[:len(t.variables)-1-n]
}

// useVar returns a node for a variable reference. It errors
// if the variable is not defined.
func (t *Tree) useVar(typ itemType, pos Pos, name string) Node {
	v := newVariable(typ, pos, name)
	for _, varName := range t.variables {
		logger.Infof("variable %s %s\n", varName, v.Ident[0])
		if varName == v.Ident[0] {
			return v
		}
	}
	t.errorf("undefined variable %q", v.Ident[0])
	return nil
}

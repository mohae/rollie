// Copyright 2014 Joel Scoble (github:mohae). All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// This code is based on code originally written by The Go Authors.
// Their copyright notice immediately follows this one.

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parse

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// item represents a item and the corresponding string.
type item struct {
	typ   itemType
	pos   Pos
	value string
}

func (i item) String() string {
	switch {
	case i.typ == EOF:
		return "EOF"
	case i.typ == ERROR:
		return i.value
	}
	return fmt.Sprintf("%q", i.value)
}

// itemType is a set of lexical items for Mustache
type itemType int

// The list of items.
const (
	// Special items
	ERROR itemType = iota
	EOF
	itemText       // main
	itemSpace      // 1 or more spaces
	itemTag        // {{ }}
	itemNL         // \n
	itemCR         // \r
	itemIdentifier //
	itemDiscard    // stuff that gets discarded

	itemOTag // {{
	itemCTag // }}

	// variable tags
	tagEscaped    //   : {{variable}}
	tagUnescaped  // { : {{{ variable}}
	tagUnescaped2 // & : {{& variable}}
	// section tags
	tagSection    // # : {{#section}}
	tagInverted   // ^ : {{^section}}
	tagEndSection // / : {{/section}}
	tagComment    //! : {{!comment}}
	tagPartial    // > : {{>partial
	tagΔDelimiter // = : {{=| |=}} : | is the new oTag and cTag

	identEscaped
	identUnescaped

	markerDot   // . : {{.}}
	markerIndex // -index : {{-index}}
	markerFirst // #-first : {{#-first}} {{/first}}
	markerLast  // /-last : {{#-first}} {{/first}}
	markerOdd   // #-odd : {{#-odd}} {{/-odd}}
	markerText  // " : {{"sometext}}

	//residual stuff should be replaced with correct itemType from above
	itemEnd
	itemRange
	itemTemplate
	itemPipe
)

const (
	OTag  = "{{"
	OLen  = 2
	ORune = '{'
	CTag  = "}}"
	CLen  = 2
	CRune = '}'
)

var itemEmpty item

// itemStrings provides string descriptions to the item.
var ItemStrings = [...]string{
	ERROR:     "ERROR",
	EOF:       "EOF",
	itemText:  "text",
	itemSpace: "space",
	itemTag:   "tag",
	itemNL:    "nl",
	itemCR:    "cr",
	//	itemChar:           "char",
	//	itemNil:            "nil",
	itemIdentifier: "identifier",
	itemDiscard:    "discard",
	itemOTag:       "otag",
	itemCTag:       "ctag",
	tagEscaped:     "escapedVarTag",
	tagUnescaped:   "unescapedVarTag",
	tagUnescaped2:  "unescapedVarTag2",
	tagSection:     "sectionTag",
	tagInverted:    "invertedTag",
	tagEndSection:  "endSectionTag",
	tagComment:     "commentTag",
	tagPartial:     "partialTag",
	tagΔDelimiter:  "ΔDelimiterTag",
	identEscaped:   "escapedVar",
	identUnescaped: "unescapedVar",
	markerDot:      "markerDot",
	markerIndex:    "markerIndex",
	markerFirst:    "markerFirst",
	markerLast:     "markerLast",
	markerOdd:      "markerOdd",
	markerText:     "markerText",
}

const eof = -1

type stateFn func(*lexer) stateFn

type lexer struct {
	// name and mustach template
	name  string // the name of the input; used only for error reports
	input string // the string being scanned

	// tag information, each lexer has their own because Mustache templates
	// can modify tags.
	oTag string // open tag
	oLen int
	cTag string // close tag info
	cLen int

	state       stateFn   // the next lexing function to enter
	pos         Pos       // current position in the input
	start       Pos       // start position of this item
	width       Pos       // width of last rune read from input
	lastPos     Pos       // position of most recent item returned by nextItem
	items       chan item // channel of scanned items
	parentDepth int       // nesting depth of ( ) exprs
}

// Returns a new, initialized lexer with the tag defaults set to {{}}.
func NewLexerFromString(name, data string, initState stateFn) *lexer {
	l := &lexer{name: name, input: data, oTag: OTag, oLen: OLen, cTag: CTag, cLen: CLen, state: initState, items: make(chan item, 2)}
	return l
}

// next returns the next rune in input.
func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[int(l.pos):])
	l.width = Pos(w)
	l.pos += l.width
	return r
}

// current returns the current rune without consuming it
func (l *lexer) current() (r rune, w int) {
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return eof, 0
	}
	r, w = utf8.DecodeRuneInString(l.input[int(l.pos):])
	return r, w
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// glance is like peek, except it looks back one rune
func (l *lexer) glance() rune {
	l.backup()
	return l.next()
}

// backup goes back one rune. Can be called once per next call.
func (l *lexer) backup() {
	l.pos -= l.width
}

// emit the current item information
func (l *lexer) emit(t itemType) {
	l.items <- item{typ: t, pos: l.start, value: l.input[l.start:l.pos]}
	l.start = l.pos
}

// ignore skips to the current post
func (l *lexer) ignore() {
	l.start = l.pos
}

// ingorePrior skips to the point prior to current.
func (l *lexer) ignorePrior() {
	l.backup()
	l.ignore()
	l.next()
}

// lineNumber reports which line we're on, based on the position of
// the previous item returned by nextItem. Doing it this way
// means we don't have to worry about peek double counting.
func (l *lexer) lineNumber() int {
	return 1 + strings.Count(l.input[:l.lastPos], "\n")
}

// error returns an error token and terminates the scan by passing back
// a nil pointer that will be the next state, terminating l.run.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{typ: ERROR, value: fmt.Sprintf(format, args...), pos: l.start}
	return nil
}

// nextItem returns the next item from the input.
func (l *lexer) nextItem() item {
	item := <-l.items
	l.lastPos = item.pos
	return item
}

// lex creates a new scanner for the input string.
func lex(name, input, oTag, cTag string) *lexer {
	if oTag == "" {
		oTag = OTag
	}
	if cTag == "" {
		cTag = CTag
	}
	l := &lexer{
		name:  name,
		oTag:  oTag,
		oLen:  len(oTag),
		cTag:  cTag,
		cLen:  len(cTag),
		input: input,
		items: make(chan item, 2), // Two item ring buffer
	}
	go l.run()
	return l
}

// run runs the state machine for the lexer.
func (l *lexer) run() {
	for l.state = lexText; l.state != nil; {
		l.state = l.state(l)
	}
}

// state functions

// Change approach to maintain a stack of recent history.
// Stack will only be used when new lines and groups of spaces
// are encountered.
// When a non stack-adding item is encountered, the stack will
// be reset, unless the next items accumulate into a comment.
// At which point the stack will be evaluated.
// Stacks are reset at the end of each comment.

// lexText checks for the basic block types and has them handled.
func lexText(l *lexer) stateFn {
	for {
		c, _ := l.current()
		if strings.HasPrefix(l.input[l.pos:], l.oTag) {
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexOTag // Next state.
		}
		// if a CRLF occurs, this sequential processing will handle it.
		if isCR(c) { // \r are handled separately
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexCR(l)
		}
		if isNL(c) { // \n are handled separately
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexNL(l)
		}
		if isSpace(c) { // spaces are separate because of standalone comments
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexSpace(l)
		}
		/* TODO proper dot handling
		if c == '.' {
			if l.pos > l.start {
				l.emit(itemText)
			}
			return lexDot(l)
		}
		*/
		if l.next() == eof {
			break
		}
	}
	// Correctly reached EOF.
	if l.pos > l.start {
		l.emit(itemText)
	}
	l.emit(EOF)
	return nil // Stop the run loop.
}

// lexOTag checks to see what kind of tag this is, with an undecorated tag
// being an escaped variable {{}}. Each handler takes care of its own CTag.
func lexOTag(l *lexer) stateFn {
	// move pointer to next pos beyond oTag to see what kind it is
	l.pos += Pos(l.oLen)
	r := l.next()
	switch r {
	case '!': // comment
		// comments get elided so we don't emit anything
		l.emit(tagComment)
		return lexComment
	case '{', '&': // unescaped comment
		l.emit(tagUnescaped)
		return lexUnescaped
	case '#': // section
		l.emit(tagSection)
		return lexSection
	case '/': // close tag
		l.emit(tagEndSection)
		return lexSection
	case '^': // inverted
		l.emit(tagInverted)
		return lexSection
	case '>': // partial
		l.emit(tagPartial)
		return lexPartial
	case '=': // delimiter change
		l.emit(tagΔDelimiter)
		return lexΔDelimiter
	}

	// default is escaped varialble: {{variable}}
	l.parentDepth = 0
	l.backup()
	l.emit(tagEscaped)
	return lexEscaped
}

// lexCTag scans the right delimiter, which is known to be present.
// We also check the character after the delimiter to see what type it
// is, and dispatch accordingly.
func lexCTag(l *lexer) stateFn {
	l.pos += Pos(l.cLen)
	l.emit(itemCTag)
	return lexText
}

// lexComment handles comment lexing. The ! has already been consumed.
// This only creates a token, item, of the comment, as the actual handling
// of its elision is determined by what surrounds it.
func lexComment(l *lexer) stateFn {
	l.ignore()
	i := strings.Index(l.input[l.pos:], l.cTag)
	switch true {
	case i < 0:
		return l.errorf("unclosed comment tag")
	case i == 0:
		return lexCTag
	}
	l.pos += Pos(i)
	l.emit(itemDiscard)
	return lexCTag
}

// lexEscaped handles escaped variable lexing. This only creates a token,
// item, of the variable.
func lexEscaped(l *lexer) stateFn {
	i := strings.Index(l.input[l.pos:], l.cTag)
	switch true {
	case i < 0:
		return l.errorf("unclosed escaped variable tag")
	case i == 0:
		return lexCTag
	}
	l.pos += Pos(i)
	l.emit(identEscaped)
	return lexCTag
}

// lexUnescaped handles unescaped variable lexing. This only creates a token,
// item, of the variable.
func lexUnescaped(l *lexer) stateFn {
	i := strings.Index(l.input[l.pos:], l.cTag)
	switch true {
	case i < 0:
		return l.errorf("unclosed escaped variable tag")
	case i == 0:
		return lexCTag
	}
	l.pos += Pos(i)
	l.emit(identUnescaped)
	// see if there is a }}}, elide the first if there is
	if strings.HasPrefix(l.input[l.pos:], "}"+l.cTag) {
		l.pos += Pos(1)
		l.start = l.pos
	}
	return lexCTag
}

// lexΔDelimiter handles the update of current oTag and cTag info
// with the new delimiter, e.g. {{=| |=}} {{=%% %%=}} {{= | | =}}
// TODO clean this up with some sane code
func lexΔDelimiter(l *lexer) stateFn {
	// cache cTag info because it will be replace during the course of execution
	origCTagPos := strings.Index(l.input[l.pos:], l.cTag)
	if origCTagPos < 0 {
		return l.errorf("rollie: unclosed delimiter tag")
	}
	origCTagPos += int(l.pos)
	cLenOrig := l.cLen
	// skip any spaces between = and Otag
	pos := skipWhitespace(l)
	if pos < 0 {
		return l.errorf("rollie: unexpected EOF encountered while changing delimiters")
	}
	endPos := nextWhitespace(l)
	if endPos < 0 {
		return l.errorf("rollie: unable to find end of new delimiter, check that there is a space following it")
	}
	// Extract the new otag
	l.oTag = strings.TrimSpace(l.input[pos:endPos])
	l.oLen = len(l.oTag)
	l.start = Pos(endPos)
	l.pos = Pos(endPos)

	//	nxt := l.next() // skip the =
	// skip the whitespace that separates delims
	_ = skipWhitespace(l)
	l.cTag = ""
	// get everything before the end delim
	for l.peek() != '=' {
		r := l.next()
		if r == ' ' {
			continue
		}

		l.cTag += string(r)
	}
	l.cLen = len(l.cTag)
	skipWhitespace(l)
	// check for the second = sign since it should be there.
	if l.peek() != '=' {
		return l.errorf("rollie: expected '=' got %q while trying to close a change delimiter tag", l.peek())
	}
	// skip to the original cTag
	l.start = Pos(origCTagPos)
	l.pos = Pos(origCTagPos + cLenOrig)
	l.emit(itemCTag)
	return lexText
}

// lexSection processes a section {{#section}} stuff {{\section}}
func lexSection(l *lexer) stateFn {
	i := strings.Index(l.input[l.pos:], l.cTag)
	if i < 0 {
		return l.errorf("unclosed tag")
	}
	l.pos += Pos(i)
	l.emit(itemIdentifier)
	return lexCTag
}

// lexPartial
func lexPartial(l *lexer) stateFn {
	i := strings.Index(l.input[l.pos:], l.cTag)
	switch true {
	case i < 0:
		return l.errorf("unclosed escaped variable tag")
	case i == 0:
		return lexCTag
	}
	l.pos += Pos(i)
	l.emit(itemIdentifier)
	return lexCTag
}

// lexSpace
func lexSpace(l *lexer) stateFn {
	for isSpace(l.peek()) {
		l.next()
	}
	l.emit(itemSpace)
	return lexText
}

// lexCR: \r
func lexCR(l *lexer) stateFn {
	l.next()
	l.emit(itemCR)
	return lexText
}

// lexDot: .
func lexDot(l *lexer) stateFn {
	l.next()
	l.emit(markerDot)
	return lexText
}

// lexNL, aka LF: \n
func lexNL(l *lexer) stateFn {
	l.next()
	l.emit(itemNL)
	return lexText
}

func nextWhitespace(l *lexer) int {
	pos := l.pos
	var r rune
	for {
		r = l.next()
		if isSpace(r) {
			// set pointer to its original pos
			ret := int(l.pos)
			l.pos = pos
			return ret
		}

	}
	// if we got here, no space was found
	return -1
}

// skips whitespace
func skipWhitespace(l *lexer) int {
	var r rune
	for {
		r = l.peek()
		if isSpace(r) {
			l.next()
			continue
		}
		l.ignore()
		return int(l.pos)
	}
	// shouldn't get here
	return -1
}

// isSpace reports whether r is a space character.
func isSpace(r rune) bool {
	return r == ' ' || r == '\t'
}

// isCR reports whether r is the cr character
func isCR(r rune) bool {
	return r == '\r'
}

// isNL reports whether r is an end-of-line character.
func isNL(r rune) bool {
	return r == '\n'
}

// isAlphaNumeric reports whether r is an alphabetic, digit, or underscore.
func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// exposed for dev
func PrintCollection(items []item) {
	for _, i := range items {
		fmt.Printf("%d\t%s\t%q\n", int(i.pos), ItemStrings[i.typ], i.value)
	}
}

// collect gathers the emitted items into a slice.-- for development
func Collect(name, src, left, right string) (items []item) {
	l := lex(name, src, left, right)
	for {
		item := l.nextItem()
		items = append(items, item)
		if item.typ == EOF || item.typ == ERROR {
			break
		}
	}
	return
}

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
	"testing"
)

// Make the types prettyprint.
var itemName = map[itemType]string{
	EOF:       "EOF",
	itemOTag:  "oTag",
	itemCTag:  "cTag",
	itemSpace: "space",
	itemNL: "nl",
	tagEscaped: "escaped",
	tagUnescaped: "unescaped",
	itemIdentifier: "identifier",
	itemVariable: "variable",
	tagSection: "section",
	tagEndSection: "\\section",
	tagInverted: "inverted",
	tagPartial: "partial",
	tagParent: "parent",
	tagΔDelimiter: "ΔDelimiter",
}

func (i itemType) String() string {
	s := itemName[i]
	if s == "" {
		return fmt.Sprintf("item%d", int(i))
	}
	return s
}

// collect gathers the emitted items into a slice.-- for development
func collect(t *lexTest, left, right string) (items []item) {
	l := lex(t.name, t.input, left, right)
	for {
		item := l.nextItem()
		items = append(items, item)
		if item.typ == EOF || item.typ == ERROR {
			break
		}
	}
	return
}

type lexTest struct {
	name  string
	input string
	items []item
}

var (
	tEOF   = item{EOF, 0, ""}
	tLeft  = item{itemOTag, 0, "{{"}
	tRight = item{itemCTag, 0, "}}"}
	tSpace = item{itemSpace, 0, " "}
)

var lexTests = []lexTest{
	{"empty", "", []item{tEOF}},
	{"spaces", " \t\n", []item{{itemText, 0, " \t\n"}, tEOF}},
	{"text", `now is the time`, []item{{itemText, 0, "now is the time"}, tEOF}},
	{"text with comment", "hello-{{/* this is a comment */}}-world", []item{
		{itemText, 0, "hello-"},
		{itemText, 0, "-world"},
		tEOF,
	}},
	{"punctuation", "{{,@% }}", []item{
		tLeft,
		{itemText, 0, ","},
		{itemText, 0, "@"},
		{itemText, 0, "%"},
		tSpace,
		tRight,
		tEOF,
	}},
	/*
		{"pipeline", `intro {{echo hi 1.2 |noargs|args 1 "hi"}} outro`, []item{
			{itemText, "intro "},
			tLeft,
			tSpace,
			tPipe,
			tSpace,
			{itemText, " outro"},
			tEOF,
		}},
	*/
	// errors
	{"badchar", "#{{\x01}}", []item{
		{itemText, 0, "#"},
		tLeft,
		{ERROR, 0, "unrecognized character in action: U+0001"},
	}},
	{"unclosed tag", "{{\n}}", []item{
		tLeft,
		{ERROR, 0, "unclosed action"},
	}},
	{"EOF in tag", "{{ stuff", []item{
		tLeft,
		{ERROR, 0, "unclosed action"},
	}},
	{"unclosed quote", "{{\"\n\"}}", []item{
		tLeft,
		{ERROR, 0, "unterminated quoted string"},
	}},
	/*
		// Fixed bugs
		// Many elements in an action blew the lookahead until
		// we made lexInsideAction not loop.
		{"long pipeline deadlock", "{{|||||}}", []item{
			tLeft,
			tPipe,
			tPipe,
			tPipe,
			tPipe,
			tPipe,
			tRight,
			tEOF,
		}},
	*/
	{"text with bad comment", "hello-{{/*/}}-world", []item{
		{itemText, 0, "hello-"},
		{ERROR, 0, `unclosed comment`},
	}},
	{"text with comment close separted from delim", "hello-{{/* */ }}-world", []item{
		{itemText, 0, "hello-"},
		{ERROR, 0, `comment ends before closing delimiter`},
	}},
}

func equal(i1, i2 []item, checkPos bool) bool {
	if len(i1) != len(i2) {
		return false
	}
	for k := range i1 {
		if i1[k].typ != i2[k].typ {
			return false
		}
		if i1[k].value != i2[k].value {
			return false
		}
		if checkPos && i1[k].pos != i2[k].pos {
			return false
		}
	}
	return true
}

/*
func TestLex(t *testing.T) {
	for _, test := range lexTests {
		items := collect(&test, "", "")
		if !equal(items, test.items, false) {
			t.Errorf("%s: got\n\t%+v\nexpected\n\t%v", test.name, items, test.items)
		}
	}
}
*/

var (
	tLeftDelims = item{tagEscaped, 0, "$$"}
	tRighDelims = item{itemCTag, 0, "@@"}
)

// Some easy cases from above, but with delimiters $$ and @@
var lexDelimTests = []lexTest{
	{"punctuation", "$$,@%{{}}@@", []item{
		tLeftDelims,
		{itemVariable, 0, ",@%{{}}"},
		tRighDelims,
		tEOF,
	}},
	{"empty action", "$$@@", []item{tLeftDelims, tRighDelims, tEOF}},
}

func TestDelims(t *testing.T) {
	for _, test := range lexDelimTests {
		items := collect(&test, "$$", "@@")
		if !equal(items, test.items, false) {
			t.Errorf("%s: got\n\t%+V\nexpected\n\t%+V", test.name, items, test.items)
		}
	}
}

var lexPosTests = []lexTest{
	{"empty", "", []item{tEOF}},
	{"punctuation", "{{,@%#}}", []item{
		{tagEscaped, 0, "{{"},
		{itemVariable, 2, ",@%#"},
		{itemCTag, 6, "}}"},
		{EOF, 8, ""},
	}},
	{"sample", "0123{{hello}}xyz", []item{
		{itemText, 0, "0123"},
		{tagEscaped, 4, "{{"},
		{itemVariable, 6, "hello"},
		{itemCTag, 11, "}}"},
		{itemText, 13, "xyz"},
		{EOF, 16, ""},
	}},
}

// The other tests don't check position, to make the test cases easier to construct.
// This one does.
func TestPos(t *testing.T) {
	for _, test := range lexPosTests {
		items := collect(&test, "", "")
		if !equal(items, test.items, true) {
			t.Errorf("%s: got\n\t%v\nexpected\n\t%v", test.name, items, test.items)
			if len(items) == len(test.items) {
				// Detailed print; avoid item.String() to expose the position value.
				for i := range items {
					if !equal(items[i:i+1], test.items[i:i+1], true) {
						i1 := items[i]
						i2 := test.items[i]
						t.Errorf("\t#%d: got {%v %d %q} expected  {%v %d %q}", i, i1.typ, i1.pos, i1.value, i2.typ, i2.pos, i2.value)
					}
				}
			}
		}
	}
}

// Simple Mustache tests: a few mustache tests, Full tests and spec tests are at parent level.
func TestSimpleStache(t *testing.T) {
	simpleStacheTests := []lexTest{
	{
		"sectionList",
		"\"{{#list}}{{item}}{{/list}}\"",
		[]item{
			{itemText, Pos(0), "\""},
			{tagSection, Pos(1), "{{#"},
			{itemIdentifier, Pos(4), "list"},
			{itemCTag, Pos(8), "}}"},
			{tagEscaped, Pos(10), "{{"},
			{itemVariable, Pos(12), "item"},
			{itemCTag, Pos(16), "}}"},
			{tagEndSection, Pos(18), "{{/"},
			{itemIdentifier, Pos(21), "list"},
			{itemCTag, Pos(25), "}}"},
			{itemText, Pos(27), "\""},
			{EOF, Pos(28), ""},
		},
	},
	{
		"invertedContext",
		"\"{{^context}}Hi {{name}}.{{/context}}\"",
		[]item{
			{itemText, Pos(0), "\""},
			{tagInverted, Pos(1), "{{^"},
			{itemIdentifier, Pos(4), "context"},
			{itemCTag, Pos(11), "}}"},
			{itemText, Pos(13), "Hi"},
			{itemSpace, Pos(15), " "},
			{tagEscaped, Pos(16), "{{"},
			{itemVariable, Pos(18), "name"},
			{itemCTag, Pos(22), "}}"},
			{itemText, Pos(24), "."},
			{tagEndSection, Pos(25), "{{/"},
			{itemIdentifier, Pos(28), "context"},
			{itemCTag, Pos(35), "}}"},
			{itemText, Pos(37), "\""},
			{EOF, Pos(38), ""},
		},
	},
	{
		"invertedList",
		"\"{{^list}}{{n}}{{/list}}\"",
		[]item{
			{itemText, Pos(0), "\""},
			{tagInverted, Pos(1), "{{^"},
			{itemIdentifier, Pos(4), "list"},
			{itemCTag, Pos(8), "}}"},
			{tagEscaped, Pos(10), "{{"},
			{itemVariable, Pos(12), "n"},
			{itemCTag, Pos(13), "}}"},
			{tagEndSection, Pos(15), "{{/"},
			{itemIdentifier, Pos(18), "list"},
			{itemCTag, Pos(22), "}}"},
			{itemText, Pos(24), "\""},
			{EOF, Pos(25), ""},
		},
	}}

	for _, test := range simpleStacheTests {
		items := collect(&test, "", "")
		if !equal(items, test.items, true) {
			t.Errorf("%s: got\n\t%v\nexpected\n\t%v", test.name, items, test.items)
			if len(items) == len(test.items) {
				// Detailed print; avoid item.String() to expose the position value.
				for i := range items {
					if !equal(items[i:i+1], test.items[i:i+1], true) {
						i1 := items[i]
						i2 := test.items[i]
						t.Errorf("\t#%d: got {%v %d %q} expected  {%v %d %q}", i, i1.typ, i1.pos, i1.value, i2.typ, i2.pos, i2.value)
					}
				}
			}
		}
	}
}

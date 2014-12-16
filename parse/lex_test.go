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

import "testing"

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

var lexTests = []lexTest{
	/*
			{"empty", "", []item{{EOF, 0, ""}}},
			{"spaces", " \t\n", []item{
				{itemSpace, 0, " \t"},
				{itemNL, 2, "\n"},
				{EOF, 3, ""},
			}},
			{"text", `now is the time`, []item{
				{itemText, 0, "now"},
				{itemSpace, 3, " "},
				{itemText, 4, "is"},
				{itemSpace, 6, " "},
				{itemText, 7, "the"},
				{itemSpace, 10, " "},
				{itemText, 11, "time"},
				{EOF, 15, ""},
			}},
			{"tag escaped", "This is {{escaped }}\n", []item{
				{itemText, 0, "This"},
				{itemSpace, 4, " "},
				{itemText, 5, "is"},
				{itemSpace, 7, " "},
				{tagEscaped, 8, "{{"},
				{identEscaped, 10, "escaped "},
				{itemCTag, 18, "}}"},
				{itemNL, 20, "\n"},
				{EOF, 21, ""},
			}},
			{"tag unescaped", "This isn't {{{escaped}}}", []item{
				{itemText, 0, "This"},
				{itemSpace, 4, " "},
				{itemText, 5, "isn't"},
				{itemSpace, 10, " "},
				{tagUnescaped, 11, "{{{"},
				{identUnescaped, 14, "escaped"},
				{itemCTag, 22, "}}"},
				{EOF, 24, ""},
			}},
			{"tag unescaped2", "This\n isn't\n{{^escaped }}", []item{
				{itemText, 0, "This"},
				{itemNL, 4, "\n"},
				{itemSpace, 5, " "},
				{itemText, 6, "isn't"},
				{itemNL, 11, "\n"},
				{tagInverted, 12, "{{^"},
				{itemIdentifier, 15, "escaped "},
				{itemCTag, 23, "}}"},
				{EOF, 25, ""},
			}},
			{"tag section", "A section {{#section}}", []item{
				{itemText, 0, "A"},
				{itemSpace, 1, " "},
				{itemText, 2, "section"},
				{itemSpace, 9, " "},
				{tagSection, 10, "{{#"},
				{itemIdentifier, 13, "section"},
				{itemCTag, 20, "}}"},
				{EOF, 22, ""},
			}},
		{"tag inverted", "  {{^inverted}}\n\n", []item{
			{itemSpace, 0, "  "},
			{tagInverted, 2, "{{^"},
			{itemIdentifier, 5, "inverted"},
			{itemCTag, 13, "}}"},
			{itemNL, 15, "\n"},
			{itemNL, 16, "\n"},
			{EOF, 17, ""},
		}},
		{"tag comment", "{{!comment}}", []item{
			{tagComment, 0, "{{!"},
			{itemDiscard, 3, "comment"},
			{itemCTag, 10, "}}"},
			{EOF, 12, ""},
		}},
	*/
	{"tag partial", "I'm {{>partial}}", []item{
		{itemText, 0, "I'm"},
		{itemSpace, 3, " "},
		{tagPartial, 4, "{{>"},
		{itemIdentifier, 7, "partial"},
		{itemCTag, 14, "}}"},
		{EOF, 16, ""},
	}},
	{"tag ΔDelimiter", "change delimiter {{=| |=}}", []item{
		{itemText, 0, "change"},
		{itemSpace, 6, " "},
		{itemText, 7, "delimiter"},
		{itemSpace, 16, " "},
		{tagΔDelimiter, 17, "{{="},
		{itemCTag, 24, "}}"},
		{EOF, 26, ""},
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
		if i1[k].pos != i2[k].pos {
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

func TestLex(t *testing.T) {
	for _, test := range lexTests {
		items := collect(&test, "", "")
		if !equal(items, test.items, false) {
			t.Errorf("%s: got\n\t%+v\nexpected\n\t%v", test.name, items, test.items)
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
				{identEscaped, Pos(12), "item"},
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
				{identEscaped, Pos(18), "name"},
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
				{identEscaped, Pos(12), "n"},
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

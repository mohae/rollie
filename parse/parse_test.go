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
	"flag"
	"fmt"
	"strings"
	"testing"
)

var debug = flag.Bool("debug", false, "show the errors produced by the main tests")

type parseTest struct {
	name   string
	input  string
	ok     bool
	result string // what the user would see in an error message.
}

const (
	noError  = true
	hasError = false
)

var parseTests = []parseTest{
	{"empty", "", noError,
		``},
	{"spaces", " \t\n", noError,
` 	
`},
	{"text", "some text", noError,
		`"some" "text"`},
	{"emptyTag", "{{}}", hasError,
		`{{}}`},
//	{"$ invocation", "{{$}}", noError,
//		"{{$}}"},
//	{"pipeline", "{{.X|.Y}}", noError,
//		`{{.X | .Y}}`},
//	{"pipeline with decl", "{{$x := .X|.Y}}", noError,
//		`{{$x := .X | .Y}}`},
//	{"nested pipeline", "{{.X (.Y .Z) (.A | .B .C) (.E)}}", noError,
//		`{{.X (.Y .Z) (.A | .B .C) (.E)}}`},
//	{"template", "{{template `x`}}", noError,
//		`{{template "x"}}`},
//	{"template with arg", "{{template `x` .Y}}", noError,
//		`{{template "x" .Y}}`},
	// Errors.
	{"unclosed action", "hello{{range", hasError, ""},
//	{"unmatched end", "{{end}}", hasError, ""},
//	{"missing end", "hello{{range .x}}", hasError, ""},//
//	{"missing end after else", "hello{{range .x}}{{else}}", hasError, ""},
//	{"undefined function", "hello{{undefined}}", hasError, ""},
//	{"undefined variable", "{{$x}}", hasError, ""},
//	{"variable undefined after end", "{{with $x := 4}}{{end}}{{$x}}", hasError, ""},
//	{"variable undefined in template", "{{template $v}}", hasError, ""},
//	{"declare with field", "{{with $x.Y := 4}}{{end}}", hasError, ""},
//	{"template with field ref", "{{template .X}}", hasError, ""},
//	{"template with var", "{{template $v}}", hasError, ""},
//	{"invalid punctuation", "{{printf 3, 4}}", hasError, ""},
//	{"multidecl outside range", "{{with $v, $u := 3}}{{end}}", hasError, ""},
//	{"too many decls in range", "{{range $u, $v, $w := 3}}{{end}}", hasError, ""},
//	{"dot applied to parentheses", "{{printf (printf .).}}", hasError, ""},
//	{"adjacent args", "{{printf 3`x`}}", hasError, ""},
//	{"adjacent args with .", "{{printf `x`.}}", hasError, ""},
//	{"extra end after if", "{{if .X}}a{{else if .Y}}b{{end}}{{end}}", hasError, ""},
	// Equals (and other chars) do not assignments make (yet).
//	{"bug0a", "{{$x := 0}}{{$x}}", noError, "{{$x := 0}}{{$x}}"},
//	{"bug0b", "{{$x = 1}}{{$x}}", hasError, ""},
//	{"bug0c", "{{$x ! 2}}{{$x}}", hasError, ""},
//	{"bug0d", "{{$x % 3}}{{$x}}", hasError, ""},
	// Check the parse fails for := rather than comma.
//	{"bug0e", "{{range $x := $y := 3}}{{end}}", hasError, ""},
	// Another bug: variable read must ignore following punctuation.
//	{"bug1a", "{{$x:=.}}{{$x!2}}", hasError, ""},                     // ! is just illegal here.
//	{"bug1b", "{{$x:=.}}{{$x+2}}", hasError, ""},                     // $x+2 should not parse as ($x) (+2).
//	{"bug1c", "{{$x:=.}}{{$x +2}}", noError, "{{$x := .}}{{$x +2}}"}, // It's OK with a space.
}

var builtins = map[string]interface{}{
	"printf": fmt.Sprintf,
}

func testParse(doCopy bool, t *testing.T) {
	textFormat = "%q"
	defer func() { textFormat = "%s" }()
	for _, test := range parseTests {
		tmpl, err := New(test.name).Parse(test.input, "{{", "}}", make(map[string]*Tree), builtins)
		switch {
		case err == nil && !test.ok:
			t.Errorf("%q: expected error; got none", test.name)
			continue
		case err != nil && test.ok:
			t.Errorf("%q: unexpected error: %v", test.name, err)
			continue
		case err != nil && !test.ok:
			// expected error, got one
			if *debug {
				fmt.Printf("%s: %s\n\t%s\n", test.name, test.input, err)
			}
			continue
		}
		var result string
		if doCopy {
			result = tmpl.Root.Copy().String()
		} else {
			result = tmpl.Root.String()
		}
		if result != test.result {
			t.Errorf("%s=(%q): got\n\t%q\nexpected\n\t%q", test.name, test.input, result, test.result)
		}
	}
}

func TestParse(t *testing.T) {
	testParse(false, t)
}

// Same as TestParse, but we copy the node first
func TestParseCopy(t *testing.T) {
	testParse(true, t)
}

type isEmptyTest struct {
	name  string
	input string
	empty bool
}

var isEmptyTests = []isEmptyTest{
	{"empty", "", true},
	{"nonempty", "hello", false},
	{"spaces only", " \t\n \t\n", false},
	{"tag", "{{!comment}}", true},
}

func TestIsEmpty(t *testing.T) {
	if !IsEmptyTree(nil) {
		t.Errorf("nil tree is not empty")
	}
	for _, test := range isEmptyTests {
		tree, err := New("root").Parse(test.input, "{{", "}}", make(map[string]*Tree), nil)
		if err != nil {
			t.Errorf("%q: unexpected error: %v", test.name, err)
			continue
		}
		if empty := IsEmptyTree(tree.Root); empty != test.empty {
			t.Errorf("%q: expected %t got %t", test.name, test.empty, empty)
		}
	}
}
/*
func TestErrorContextWithTreeCopy(t *testing.T) {
	tree, err := New("root").Parse("{{if true}}{{end}}", "", "", make(map[string]*Tree), nil)
	if err != nil {
		t.Fatalf("unexpected tree parse failure: %v", err)
	}
	treeCopy := tree.Copy()
	wantLocation, wantContext := tree.ErrorContext(tree.Root.Nodes[0])
	gotLocation, gotContext := treeCopy.ErrorContext(treeCopy.Root.Nodes[0])
	if wantLocation != gotLocation {
		t.Errorf("wrong error location want %q got %q", wantLocation, gotLocation)
	}
	if wantContext != gotContext {
		t.Errorf("wrong error location want %q got %q", wantContext, gotContext)
	}
}
*/

// All failures, and the result is a string that must appear in the error message.
var errorTests = []parseTest{
	// Check line numbers are accurate.
	{"unclosed1",
		"line1\n{{",
		hasError, `template: unclosed1:2: unexpected "{{" in input`},
	{"space",
		"{{`x`3}}",
		hasError, `template: space:1: unexpected "{{" in input`},
}

func TestErrors(t *testing.T) {
	for _, test := range errorTests {
		_, err := New(test.name).Parse(test.input, "", "", make(map[string]*Tree))
		if err == nil {
			t.Errorf("%q: expected error", test.name)
			continue
		}
		if !strings.Contains(err.Error(), test.result) {
			t.Errorf("%q: error %q does not contain %q", test.name, err, test.result)
		}
	}
}


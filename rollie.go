// Copyright 2014 Joel Scoble (github:mohae). All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// This code is based on code originally written by The Go Authors.
// Their copyright notice immediately follows this one.

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rollie

import (
	"bytes"
	"io/ioutil"
)

// RenderFile
func RenderFile(s string, context ...interface{}) string {
	file, err := ioutil.ReadFile(s)
	if err != nil {
		return err.Error()
	}
	return Render(string(file), context)
}

func Render(data string, context ...interface{}) string {
	t, err := Parse(data)
	if err != nil {
		logger.Errorf("%s\n",err)
		return err.Error()
	}
	var b bytes.Buffer
	logger.Debug("\n\n\n")
	t.Execute(&b, context)
	return b.String()
}

// Parse is a wrapper for parse(). It takes a string, uses "parse" as the
// string's name, and calls parse(), returning the results
func Parse(text string) (*Template, error) {
	return mparse(text)
}

// ParseFile is a wrapper for parse(). It take s a filename, reads its
// contents and parses that as a Mustache template.
func ParseFile(s string) (*Template, error) {
	text, err := ioutil.ReadFile(s)
	if err != nil {
		return nil, err
	}

	return mparse(string(text))
}

// mparse does the actual work of parsing the Mustache template and returning
// the results.
func mparse(text string) (*Template, error) {
	tpl := Template{}
	t, err := tpl.Parse(text)
	if err != nil {
		return nil, err
	}
	return t, nil
}


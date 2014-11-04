// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Parse nodes.

package parse

import (
	"bytes"
	"fmt"
	"strings"
)

var textFormat = "%s" // Changed to "%q" in tests for better error messages.

// A Node is an element in the parse tree. The interface is trivial.
// The interface contains an unexported method so that only
// types local to this package can satisfy it.
type Node interface {
	Type() NodeType
	String() string
	// Copy does a deep copy of the Node and all its components.
	// To avoid type assertions, some XxxNodes also have specialized
	// CopyXxx methods that return *XxxNode.
	Copy() Node
	Position() Pos // byte position of start of node in full original input string
	// Make sure only functions in this package can create Nodes.
	unexported()
}

// NodeType identifies the type of a parse tree node.
type NodeType int

// Pos represents a byte position in the original input text from which
// this template was parsed.
type Pos int

func (p Pos) Position() Pos {
	return p
}

// unexported keeps Node implementations local to the package.
// All implementations embed Pos, so this takes care of it.
func (Pos) unexported() {
}

// Type returns itself and provides an easy default implementation
// for embedding in a Node. Embedded in all non-trivial Nodes.
func (t NodeType) Type() NodeType {
	return t
}

const (
	NodeText       NodeType = iota // Plain text.
	NodeNL                        // NL are evaluated separately
	NodeCR			    // CRs need to be handled too
	NodeSpace                     // Space..we classify that differently
	NodeComment
	NodeEscaped
	NodeUnescaped
	NodeSection
	NodeInverted
	NodePartial
	NodeParent
	NodeList			// A list of nodes
	NodePipe                       // A pipeline of commands.
	NodeTemplate                   // A template invocation action.
	NodeIdentifier
	NodeCommand
	NodeVariable
	NodeCTag
	NodeBranch
	NodeIf
	NodeElse
	NodeRange
	NodeWith
	NodeAction                     // A non-control action such as a field evaluation.
	NodeChain
	NodeField
	NodeDot
	NodeEnd
)

// NodeStrings gives a string description for the NodeType
var NodeStrings map[NodeType]string
var ItemToNode map[itemType]NodeType   // itemTypes to NodeTypes (for ident declarations)

func init() {
	NodeStrings = make(map[NodeType]string)
	NodeStrings[NodeText] = "NodeText"
	NodeStrings[NodeNL] = "NodeNL"
	NodeStrings[NodeCR] = "NodeCR"
	NodeStrings[NodeSpace] = "NodeSpace"
	NodeStrings[NodeComment] = "NodeComment"
	NodeStrings[NodeEscaped] = "NodeEscaped"
	NodeStrings[NodeUnescaped] = "NodeUnescaped"
	NodeStrings[NodeSection] = "NodeSection"
	NodeStrings[NodeInverted] = "NodeInverted"
	NodeStrings[NodePartial] = "NodePartial"
	NodeStrings[NodeParent] = "NodeParent"
	NodeStrings[NodeList] = "NodeList"
	NodeStrings[NodePipe] = "NodePipe"
	NodeStrings[NodeTemplate] = "NodeTemplate"
	NodeStrings[NodeIdentifier] = "NodeIdentifier"
	NodeStrings[NodeCommand] = "NodeCommand"
	NodeStrings[NodeVariable] = "NodeVariable"
	NodeStrings[NodeCTag] = "NodeCTag"
	NodeStrings[NodeBranch] = "NodeBranch"
	NodeStrings[NodeIf] = "NodeIf"
	NodeStrings[NodeRange] = "NodeRange"
	NodeStrings[NodeWith] = "NodeWith"
	NodeStrings[NodeChain] = "NodeChain"
	NodeStrings[NodeField] = "NodeField"	
	NodeStrings[NodeDot] = "NodeDot"	
	NodeStrings[NodeEnd] = "NodeEnd"
	ItemToNode = make(map[itemType]NodeType)
	ItemToNode[identUnescaped] = NodeUnescaped
	ItemToNode[identEscaped] = NodeEscaped
}
// Nodes.

// TextNode holds plain text.
type TextNode struct {
	NodeType
	Pos
	Text []byte // The text; may span newlines.
}


func newText(pos Pos, text string) *TextNode {
	return &TextNode{NodeType: NodeText, Pos: pos, Text: []byte(text)}
}

func (t *TextNode) String() string {
	return fmt.Sprintf(textFormat, t.Text)
}

func (t *TextNode) Copy() Node {
	return &TextNode{NodeType: NodeText, Text: append([]byte{}, t.Text...)}
}

// NLNode holds newline char
type NLNode struct {
	NodeType
	Pos
	Text []byte
}

func newNL(pos Pos, text string) *NLNode {
	return &NLNode{NodeType: NodeNL, Pos: pos, Text: []byte(text)}
}

func (nl *NLNode) String() string {
	return string(nl.Text)
}

func (nl *NLNode) Copy() Node {
	return &NLNode{NodeType: NodeNL, Text: append([]byte{}, nl.Text...)}
}

// CRNode holds newline char
type CRNode struct {
	NodeType
	Pos
	Text []byte
}

func newCR(pos Pos, text string) *CRNode {
	return &CRNode{NodeType: NodeCR, Pos: pos, Text: []byte(text)}
}

func (cr *CRNode) String() string {
	return string(cr.Text)
}

func (cr *CRNode) Copy() Node {
	return &NLNode{NodeType: NodeCR, Text: append([]byte{}, cr.Text...)}
}

// SpaceNode holds a series of spaces, this sometimes matter
type SpaceNode struct {
	NodeType
	Pos
	Text []byte
}

func newSpace(pos Pos, text string) *SpaceNode {
	return &SpaceNode{NodeType: NodeSpace, Pos: pos, Text: []byte(text)}
}

func (t *SpaceNode) String() string {
	return string(t.Text)
}

func (t *SpaceNode) Copy() Node {
	return &SpaceNode{NodeType: NodeSpace, Text: append([]byte{}, t.Text...)}
}

// CTagNode holds a closing tag
type CTagNode struct {
	NodeType
	Pos
	Text []byte
}

func newCTag(pos Pos, text string) *CTagNode {
	return &CTagNode{NodeType: NodeCTag, Pos: pos, Text: []byte(text)}
}

func (t *CTagNode) String() string {
	return string(t.Text)
}

func (t *CTagNode) Copy() Node {
	return &CTagNode{NodeType: NodeCTag, Text: append([]byte{}, t.Text...)}
}

// CommentNode holds a tag
type CommentNode struct {
	NodeType
	TagType itemType
	Pos
//	Width int
	Line	int
	Pipe *PipeNode
}

func newComment(pos Pos, line int, pipe *PipeNode) *CommentNode {
	return &CommentNode{NodeType: NodeComment, Pos: pos, Line: line, Pipe: pipe}
}

func (t *CommentNode) String() string {
	return  fmt.Sprintf("{{%s}}", t.Pipe)
}

func (t *CommentNode) Copy() Node {
	return &CommentNode{NodeType: t.NodeType, TagType: t.TagType, Pos: t.Pos, Line: t.Line, Pipe: t.Pipe.CopyPipe()}
}

// VariableNode holds an escaped variable
type VariableNode struct {
	NodeType
	Typ itemType // Variables can be escaped or unescaped; this is for that.
	Pos
	Ident []string // Variable name and fields in lexical order
}

func newVariable(typ itemType, pos Pos, ident string) *VariableNode {
	return &VariableNode{NodeType: NodeVariable, Typ: typ, Pos: pos, Ident: strings.Split(ident, ".")}
}

func (v *VariableNode) String() string {
	s := ""
	for i, id := range v.Ident {
		if i > 0 {
			s += "."
		}
		s += id
	}
	return s
}

func (v *VariableNode) Copy() Node {
	return &VariableNode{NodeType: NodeVariable, Typ: v.Typ, Pos: v.Pos, Ident: append([]string{}, v.Ident...)}
}

// DotNode holds the special identifier '.'.
type DotNode struct {
	Pos
}

func newDot(pos Pos) *DotNode {
	return &DotNode{Pos: pos}
}

func (d *DotNode) Type() NodeType {
	return NodeDot
}

func (d *DotNode) String() string {
	return "."
}

func (d *DotNode) Copy() Node {
	return newDot(d.Pos)
}

// InvertedNode holds an identifier.
type InvertedNode struct {
	NodeType
	Pos
	Ident string // The identifier's name.
}

// NewInverted returns a new InvertedNode with the given identifier name.
func newInverted(ident string) *InvertedNode {
	return &InvertedNode{NodeType: NodeInverted, Ident: ident}
}

// SetPos sets the position. ParentNode is a public method so we can't modify its signature.
// Chained for convenience.
// TODO: fix one day?
func (i *InvertedNode) SetPos(pos Pos) *InvertedNode {
	i.Pos = pos
	return i
}

func (i *InvertedNode) String() string {
	return i.Ident
}

func (i *InvertedNode) Copy() Node {
	return newInverted(i.Ident).SetPos(i.Pos)
}

// PartialNode holds an identifier.
type PartialNode struct {
	NodeType
	Pos
	Ident string // The identifier's name.
}

// NewPartial returns a new PartialNode with the given identifier name.
func newPartial(pos Pos, ident string) *PartialNode {
	return &PartialNode{NodeType: NodePartial, Pos: pos, Ident: ident}
}

// SetPos sets the position. NewIdentifier is a public method so we can't modify its signature.
// Chained for convenience.
// TODO: fix one day?
func (i *PartialNode) SetPos(pos Pos) *PartialNode {
	i.Pos = pos
	return i
}

func (i *PartialNode) String() string {
	return i.Ident
}

func (i *PartialNode) Copy() Node {
	return newPartial(i.Pos, i.Ident)
}

// ParentNode holds an identifier.
type ParentNode struct {
	NodeType
	Pos
	Ident string // The identifier's name.
}

// NewParent returns a new ParentNode with the given identifier name.
func newParent(ident string) *ParentNode {
	return &ParentNode{NodeType: NodeParent, Ident: ident}
}

// SetPos sets the position. ParentNode is a public method so we can't modify its signature.
// Chained for convenience.
// TODO: fix one day?
func (i *ParentNode) SetPos(pos Pos) *ParentNode {
	i.Pos = pos
	return i
}

func (i *ParentNode) String() string {
	return i.Ident
}

func (i *ParentNode) Copy() Node {
	return newParent(i.Ident).SetPos(i.Pos)
}

// ListNode holds a sequence of nodes.
type ListNode struct {
	NodeType
	Pos
	Nodes []Node // The element nodes in lexical order.
}

func newList(pos Pos) *ListNode {
	return &ListNode{NodeType: NodeList, Pos: pos}
}

func (l *ListNode) append(n Node) {
	l.Nodes = append(l.Nodes, n)
}

func (l *ListNode) String() string {
	b := new(bytes.Buffer)
	for _, n := range l.Nodes {
		fmt.Fprint(b, n)
	}
	return b.String()
}

func (l *ListNode) CopyList() *ListNode {
	if l == nil {
		return l
	}
	n := newList(l.Pos)
	for _, elem := range l.Nodes {
		n.append(elem.Copy())
	}
	return n
}

func (l *ListNode) Copy() Node {
	return l.CopyList()
}
// ActionNode holds an action (something bounded by delimiters).
// Control actions have their own nodes; ActionNode represents simple
// ones such as field evaluations and parenthesized pipelines.
type ActionNode struct {
	NodeType
	Typ itemType
	Pos
	Line int
	Pipe *PipeNode // The pipeline in the action.
}

func newAction(typ itemType, pos Pos, line int, pipe *PipeNode) *ActionNode {
	return &ActionNode{NodeType: NodeAction, Typ: typ, Pos: pos, Line: line, Pipe: pipe}
}

func (a *ActionNode) String() string {
	return fmt.Sprintf("{{%s}}", a.Pipe)

}

func (a *ActionNode) Copy() Node {
	return newAction(a.Typ, a.Pos, a.Line, a.Pipe.CopyPipe())

}

// PipeNode holds a pipeline with optional declaration
type PipeNode struct {
	NodeType
	Pos
	Line int             // The line number in the input (deprecated; kept for compatibility)
	Decl []*VariableNode
	Cmds []*CommandNode  // The commands in lexical order.
}

func newPipeline(pos Pos, line int, decl []*VariableNode) *PipeNode {
	return &PipeNode{NodeType: NodePipe, Pos: pos, Line: line, Decl: decl}
}

func (p *PipeNode) append(command *CommandNode) {
	p.Cmds = append(p.Cmds, command)
}

func (p *PipeNode) String() string {
	s := ""
	if len(p.Decl) > 0 {
		for i, v := range p.Decl {
			if i > 0 {
				s += ", "
			}
			s += v.String()
		}
		s += " := "
	}
	for i, c := range p.Cmds {
		if i > 0 {
			s += " | "
		}
		s += c.String()
	}
	return s
}

func (p *PipeNode) CopyPipe() *PipeNode {
	if p == nil {
		return p
	}
	var decl []*VariableNode
	for _, e := range p.Decl {
		decl = append(decl, e.Copy().(*VariableNode))
	}
	n := newPipeline(p.Pos, p.Line, decl)
	for _, c := range p.Cmds {
		n.append(c.Copy().(*CommandNode))
	}
	return n
}

func (p *PipeNode) Copy() Node {
	return p.CopyPipe()
}

// TemplateNode represents a {{template}} action.
type TemplateNode struct {
	NodeType
	Pos
	Line int       // The line number in the input (deprecated; kept for compatibility)
	Name string    // The name of the template (unquoted).
	Pipe *PipeNode // The command to evaluate as dot for the template.
}

func newTemplate(pos Pos, line int, name string, pipe *PipeNode) *TemplateNode {
	return &TemplateNode{NodeType: NodeTemplate, Line: line, Pos: pos, Name: name, Pipe: pipe}
}

func (t *TemplateNode) String() string {
	if t.Pipe == nil {
		return fmt.Sprintf("{{template %q}}", t.Name)
	}
	return fmt.Sprintf("{{template %q %s}}", t.Name, t.Pipe)
}

func (t *TemplateNode) Copy() Node {
	return newTemplate(t.Pos, t.Line, t.Name, t.Pipe.CopyPipe())
}


// IdentifierNode holds an identifier.
type IdentifierNode struct {
	NodeType
	Pos
	Ident string // The identifier's name.
}

// NewIdentifier returns a new IdentifierNode with the given identifier name.
func NewIdentifier(typ NodeType, ident string) *IdentifierNode {
	return &IdentifierNode{NodeType: typ, Ident: ident}
}

// SetPos sets the position. NewIdentifier is a public method so we can't modify its signature.
// Chained for convenience.
// TODO: fix one day?
func (i *IdentifierNode) SetPos(pos Pos) *IdentifierNode {
	i.Pos = pos
	return i
}

func (i *IdentifierNode) String() string {
	return i.Ident
}

func (i *IdentifierNode) Copy() Node {
	return NewIdentifier(i.NodeType, i.Ident).SetPos(i.Pos)
}

// CommandNode holds a command (a pipeline inside an evaluating action).
type CommandNode struct {
	NodeType
	Pos
	Args []Node // Arguments in lexical order: Identifier, field, or constant.
}

func newCommand(pos Pos) *CommandNode {
	return &CommandNode{NodeType: NodeCommand, Pos: pos}
}

func (c *CommandNode) append(arg Node) {
	c.Args = append(c.Args, arg)
}

func (c *CommandNode) String() string {
	s := ""
	for i, arg := range c.Args {
		if i > 0 {
			s += " "
		}
		if arg, ok := arg.(*PipeNode); ok {
			s += "(" + arg.String() + ")"
			continue
		}
		s += arg.String()
	}
	return s
}

func (c *CommandNode) Copy() Node {
	if c == nil {
		return c
	}
	n := newCommand(c.Pos)
	for _, c := range c.Args {
		n.append(c.Copy())
	}
	return n
}

// BranchNode is the common representation of if, range, and with.
type BranchNode struct {
	NodeType
	Pos
	Line     int       // The line number in the input (deprecated; kept for compatibility)
	Pipe     *PipeNode // The pipeline to be evaluated.
	List     *ListNode // What to execute if the value is non-empty.
	ElseList *ListNode // What to execute if the value is empty (nil if absent).
}

func (b *BranchNode) String() string {
	name := ""
	switch b.NodeType {
	case NodeIf:
		name = "if"
	case NodeRange:
		name = "range"
	case NodeWith:
		name = "with"
	default:
		panic("unknown branch type")
	}
	if b.ElseList != nil {
		return fmt.Sprintf("{{%s %s}}%s{{else}}%s{{end}}", name, b.Pipe, b.List, b.ElseList)
	}
	return fmt.Sprintf("{{%s %s}}%s{{end}}", name, b.Pipe, b.List)
}

// endNode represents an {{end}} action.
// It does not appear in the final parse tree.
type endNode struct {
	Pos
	Name string
}

func newEnd(pos Pos, name string) *endNode {
	return &endNode{Pos: pos, Name: name}
}

func (e *endNode) Type() NodeType {
	return NodeEnd
}

func (e *endNode) String() string {
	return fmt.Sprintf("{{\\%s}}", e.Name)
}

func (e *endNode) Copy() Node {
	return newEnd(e.Pos, e.Name)
}

// elseNode represents an {{else}} action. Does not appear in the final tree.
type elseNode struct {
	NodeType
	Pos
	Line int // The line number in the input (deprecated; kept for compatibility)
}

func newElse(pos Pos, line int) *elseNode {
	return &elseNode{NodeType: NodeElse, Pos: pos, Line: line}
}

func (e *elseNode) Type() NodeType {
	return NodeElse
}

func (e *elseNode) String() string {
	return "{{else}}"
}

func (e *elseNode) Copy() Node {
	return newElse(e.Pos, e.Line)
}
// IfNode represents an {{if}} action and its commands.
type IfNode struct {
	BranchNode
}

func newIf(pos Pos, line int, pipe *PipeNode, list, elseList *ListNode) *IfNode {
	return &IfNode{BranchNode{NodeType: NodeIf, Pos: pos, Line: line, Pipe: pipe, List: list, ElseList: elseList}}
}

func (i *IfNode) Copy() Node {
	return newIf(i.Pos, i.Line, i.Pipe.CopyPipe(), i.List.CopyList(), i.ElseList.CopyList())
}

// RangeNode represents a {{range}} action and its commands.
type RangeNode struct {
	BranchNode
}

func newRange(pos Pos, line int, pipe *PipeNode, list, elseList *ListNode) *RangeNode {
	return &RangeNode{BranchNode{NodeType: NodeRange, Pos: pos, Line: line, Pipe: pipe, List: list, ElseList: elseList}}
}

func (r *RangeNode) Copy() Node {
	return newRange(r.Pos, r.Line, r.Pipe.CopyPipe(), r.List.CopyList(), r.ElseList.CopyList())
}

// WithNode represents a {{with}} action and its commands.
type WithNode struct {
	BranchNode
}

func newWith(pos Pos, line int, pipe *PipeNode, list, elseList *ListNode) *WithNode {
	return &WithNode{BranchNode{NodeType: NodeWith, Pos: pos, Line: line, Pipe: pipe, List: list, ElseList: elseList}}
}

func (w *WithNode) Copy() Node {
	return newWith(w.Pos, w.Line, w.Pipe.CopyPipe(), w.List.CopyList(), w.ElseList.CopyList())
}

// FieldNode holds a field (identifier starting with '.').
// The names may be chained ('.x.y').
// The period is dropped from each ident.
type FieldNode struct {
	NodeType
	Pos
	Ident []string // The identifiers in lexical order.
}

func newField(pos Pos, ident string) *FieldNode {
	return &FieldNode{NodeType: NodeField, Pos: pos, Ident: strings.Split(ident[1:], ".")} // [1:] to drop leading period
}

func (f *FieldNode) String() string {
	s := ""
	for _, id := range f.Ident {
		s += "." + id
	}
	return s
}

func (f *FieldNode) Copy() Node {
	return &FieldNode{NodeType: NodeField, Pos: f.Pos, Ident: append([]string{}, f.Ident...)}
}

// ChainNode holds a term followed by a chain of field accesses (identifier starting with '.').
// The names may be chained ('.x.y').
// The periods are dropped from each ident.
type ChainNode struct {
	NodeType
	Pos
	Node  Node
	Field []string // The identifiers in lexical order.
}

func newChain(pos Pos, node Node) *ChainNode {
	return &ChainNode{NodeType: NodeChain, Pos: pos, Node: node}
}

// Add adds the named field (which should start with a period) to the end of the chain.
func (c *ChainNode) Add(field string) {
	if len(field) == 0 || field[0] != '.' {
		panic("no dot in field")
	}
	field = field[1:] // Remove leading dot.
	if field == "" {
		panic("empty field")
	}
	c.Field = append(c.Field, field)
}

func (c *ChainNode) String() string {
	s := c.Node.String()
	if _, ok := c.Node.(*PipeNode); ok {
		s = "(" + s + ")"
	}
	for _, field := range c.Field {
		s += "." + field
	}
	return s
}

func (c *ChainNode) Copy() Node {
	return &ChainNode{NodeType: NodeChain, Pos: c.Pos, Node: c.Node, Field: append([]string{}, c.Field...)}
}

/*
// TagNode holds a tag
type TagNode struct {
	NodeType
	TagType itemType
	Pos
	LineNumber int
	Text string
}

func newTag(pos Pos, ln int, text string) *TagNode {
	return &TagNode{NodeType: NodeTag, Pos: pos, LineNumber: ln, Text: text}
}

func (t *TagNode) String() string {
	return  string(t.Text)
}

func (t *TagNode) Copy() Node {
	return &TagNode{NodeType: t.NodeType, TagType: t.TagType, Pos: t.Pos, LineNumber: t.LineNumber, Text: t.Text}
}





*/



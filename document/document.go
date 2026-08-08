// Package document defines the stable, renderer-agnostic Markdown document
// model. It is the public contract between the parser and all renderers and
// must import nothing outside the standard library.
package document

type Kind int

const (
	KindDocument Kind = iota
	KindHeading
	KindParagraph
	KindBlockQuote
	KindAdmonition
	KindList
	KindListItem
	KindCodeBlock
	KindDiagram
	KindMathBlock
	KindTable
	KindTableRow
	KindTableCell
	KindThematicBreak
	KindHTMLBlock
	KindDefinitionList
	KindDefinitionTerm
	KindDefinitionDesc
	KindFootnoteDef
	KindText
	KindSoftBreak
	KindHardBreak
	KindEmphasis
	KindStrong
	KindStrikethrough
	KindCodeSpan
	KindLink
	KindImage
	KindWikiLink
	KindMathInline
	KindHTMLInline
	KindFootnoteRef
)

type Node interface {
	Kind() Kind
	Children() []Node
	AppendChild(Node)
}

// Container is the embeddable base for all nodes.
type Container struct{ kids []Node }

func (c *Container) Children() []Node   { return c.kids }
func (c *Container) AppendChild(n Node) { c.kids = append(c.kids, n) }

type Alignment int

const (
	AlignNone Alignment = iota
	AlignLeft
	AlignCenter
	AlignRight
)

// Block nodes.
type Document struct {
	Container
	Meta      map[string]any // front-matter, nil if absent
	Footnotes []*FootnoteDef
}
type Heading struct {
	Container
	Level    int
	AnchorID string
}
type Paragraph struct{ Container }
type BlockQuote struct{ Container }
type Admonition struct {
	Container
	Variant string // note|tip|important|warning|caution
}
type List struct {
	Container
	Ordered bool
	Start   int
	Tight   bool
}
type ListItem struct {
	Container
	Task    bool
	Checked bool
}
type CodeBlock struct {
	Container
	Language string
	Code     string
}
type Diagram struct {
	Container
	Engine string // "mermaid"
	Source string
}
type MathBlock struct {
	Container
	Source string
}
type Table struct {
	Container
	Alignments []Alignment
}
type TableRow struct {
	Container
	Header bool
}
type TableCell struct{ Container }
type ThematicBreak struct{ Container }
type HTMLBlock struct {
	Container
	HTML string
}
type DefinitionList struct{ Container }
type DefinitionTerm struct{ Container }
type DefinitionDesc struct{ Container }
type FootnoteDef struct {
	Container
	Index int
}

// Inline nodes.
type Text struct {
	Container
	Value string
}
type SoftBreak struct{ Container }
type HardBreak struct{ Container }
type Emphasis struct{ Container }
type Strong struct{ Container }
type Strikethrough struct{ Container }
type CodeSpan struct {
	Container
	Value string
}
type Link struct {
	Container
	Destination string
	Title       string
}
type Image struct {
	Container
	Destination string
	Title       string
	Alt         string
}
type WikiLink struct {
	Container
	Target string
}
type MathInline struct {
	Container
	Source  string
	Display bool
}
type HTMLInline struct {
	Container
	HTML string
}
type FootnoteRef struct {
	Container
	Index int
}

func (*Document) Kind() Kind       { return KindDocument }
func (*Heading) Kind() Kind        { return KindHeading }
func (*Paragraph) Kind() Kind      { return KindParagraph }
func (*BlockQuote) Kind() Kind     { return KindBlockQuote }
func (*Admonition) Kind() Kind     { return KindAdmonition }
func (*List) Kind() Kind           { return KindList }
func (*ListItem) Kind() Kind       { return KindListItem }
func (*CodeBlock) Kind() Kind      { return KindCodeBlock }
func (*Diagram) Kind() Kind        { return KindDiagram }
func (*MathBlock) Kind() Kind      { return KindMathBlock }
func (*Table) Kind() Kind          { return KindTable }
func (*TableRow) Kind() Kind       { return KindTableRow }
func (*TableCell) Kind() Kind      { return KindTableCell }
func (*ThematicBreak) Kind() Kind  { return KindThematicBreak }
func (*HTMLBlock) Kind() Kind      { return KindHTMLBlock }
func (*DefinitionList) Kind() Kind { return KindDefinitionList }
func (*DefinitionTerm) Kind() Kind { return KindDefinitionTerm }
func (*DefinitionDesc) Kind() Kind { return KindDefinitionDesc }
func (*FootnoteDef) Kind() Kind    { return KindFootnoteDef }
func (*Text) Kind() Kind           { return KindText }
func (*SoftBreak) Kind() Kind      { return KindSoftBreak }
func (*HardBreak) Kind() Kind      { return KindHardBreak }
func (*Emphasis) Kind() Kind       { return KindEmphasis }
func (*Strong) Kind() Kind         { return KindStrong }
func (*Strikethrough) Kind() Kind  { return KindStrikethrough }
func (*CodeSpan) Kind() Kind       { return KindCodeSpan }
func (*Link) Kind() Kind           { return KindLink }
func (*Image) Kind() Kind          { return KindImage }
func (*WikiLink) Kind() Kind       { return KindWikiLink }
func (*MathInline) Kind() Kind     { return KindMathInline }
func (*HTMLInline) Kind() Kind     { return KindHTMLInline }
func (*FootnoteRef) Kind() Kind    { return KindFootnoteRef }

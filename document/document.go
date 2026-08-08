// Package document defines the stable, renderer-agnostic Markdown document
// model. It is the public contract between the parser and all renderers and
// must import nothing outside the standard library.
package document

// Kind identifies a Node's concrete type without requiring a type switch.
// Each concrete node type below has exactly one corresponding Kind* value.
type Kind int

const (
	// Values are pinned — a serialization/FFI compatibility contract.
	// Never renumber; append only.
	KindDocument       Kind = 0
	KindHeading        Kind = 1
	KindParagraph      Kind = 2
	KindBlockQuote     Kind = 3
	KindAdmonition     Kind = 4
	KindList           Kind = 5
	KindListItem       Kind = 6
	KindCodeBlock      Kind = 7
	KindDiagram        Kind = 8
	KindMathBlock      Kind = 9
	KindTable          Kind = 10
	KindTableRow       Kind = 11
	KindTableCell      Kind = 12
	KindThematicBreak  Kind = 13
	KindHTMLBlock      Kind = 14
	KindDefinitionList Kind = 15
	KindDefinitionTerm Kind = 16
	KindDefinitionDesc Kind = 17
	KindFootnoteDef    Kind = 18
	KindText           Kind = 19
	KindSoftBreak      Kind = 20
	KindHardBreak      Kind = 21
	KindEmphasis       Kind = 22
	KindStrong         Kind = 23
	KindStrikethrough  Kind = 24
	KindCodeSpan       Kind = 25
	KindLink           Kind = 26
	KindImage          Kind = 27
	KindWikiLink       Kind = 28
	KindMathInline     Kind = 29
	KindHTMLInline     Kind = 30
	KindFootnoteRef    Kind = 31
)

// Node is a member of the document tree. Every concrete type in this
// package implements it by embedding Container.
type Node interface {
	Kind() Kind
	Children() []Node
	AppendChild(Node)
}

// Container is the embeddable base for all nodes; it implements Children
// and AppendChild via a backing slice.
type Container struct {
	kids []Node
	span Span
}

func (c *Container) Children() []Node   { return c.kids }
func (c *Container) AppendChild(n Node) { c.kids = append(c.kids, n) }

// Span returns the node's source location (zero if unknown; see Span).
func (c *Container) Span() Span { return c.span }

// SetSpan records the node's source location.
func (c *Container) SetSpan(s Span) { c.span = s }

// Span locates a node in the original Markdown source. Lines are 1-based;
// offsets are 0-based byte offsets forming the half-open range
// [StartOffset, EndOffset). The zero Span means "position unknown". In
// v0.2 only block-level nodes are populated; inline spans are reserved
// for future use. Offsets are relative to the source AFTER any leading
// UTF-8 BOM has been stripped.
type Span struct {
	StartLine, EndLine     int
	StartOffset, EndOffset int
}

// IsZero reports whether the span carries no position information.
func (s Span) IsZero() bool { return s == Span{} }

// Alignment is a table column's horizontal alignment, taken from its
// `:---:`-style delimiter-row marker.
type Alignment int

const (
	AlignNone Alignment = iota
	AlignLeft
	AlignCenter
	AlignRight
)

// Block nodes.

// Document is the root of a parsed tree: its children are the top-level
// block nodes, alongside decoded front matter and collected footnote
// definitions.
type Document struct {
	Container
	Meta      map[string]any // decoded front matter, nil if the document has none
	Footnotes []*FootnoteDef // footnote definitions, in first-reference order
}

// Heading is a section heading with level 1-6 and a pre-computed,
// deduplicated anchor slug.
type Heading struct {
	Container
	Level    int    // heading level, 1-6
	AnchorID string // slugified, per-document-unique anchor id, or "" if anchors are disabled
}

// Paragraph is a run of inline content set off by blank lines.
type Paragraph struct{ Container }

// BlockQuote is quoted content, rendered as a <blockquote>.
type BlockQuote struct{ Container }

// Admonition is a callout box promoted from a GitHub-style [!VARIANT]
// blockquote marker.
type Admonition struct {
	Container
	Variant string // note|tip|important|warning|caution
}

// List is an ordered or unordered list of ListItem children.
type List struct {
	Container
	Ordered bool // true for a numbered list, false for a bulleted list
	Start   int  // first item's number, for an ordered list
	Tight   bool // true when items render without wrapping <p> tags
}

// ListItem is one entry in a List, optionally a GFM task-list item.
type ListItem struct {
	Container
	Task    bool // true for a task-list item ("- [ ]" / "- [x]")
	Checked bool // task item's checked state; meaningful only when Task is true
}

// CodeBlock is a fenced or indented code block with an optional language
// tag.
type CodeBlock struct {
	Container
	Language string // fence info-string language tag, or "" if none
	Code     string // literal source text
}

// Diagram is a fenced code block whose language names a supported diagram
// engine, rendered as diagram markup instead of a code block.
type Diagram struct {
	Container
	Engine string // "mermaid"
	Source string // diagram definition text
}

// MathBlock is standalone display math, promoted from either a `$$ ... $$`
// paragraph or a ```math fenced block.
type MathBlock struct {
	Container
	Source string // math expression source, without delimiters
}

// Table is a GFM table with one Alignment per column.
type Table struct {
	Container
	Alignments []Alignment // one entry per column, in column order
}

// TableRow is one row of a Table.
type TableRow struct {
	Container
	Header bool // true for the header row
}

// TableCell is one cell of a TableRow.
type TableCell struct{ Container }

// ThematicBreak is a horizontal rule (<hr>).
type ThematicBreak struct{ Container }

// HTMLBlock is a raw block of HTML markup, sanitized at render time unless
// Options.Unsafe is set.
type HTMLBlock struct {
	Container
	HTML string // raw source markup
}

// DefinitionList is a list of DefinitionTerm/DefinitionDesc pairs.
type DefinitionList struct{ Container }

// DefinitionTerm is the term half of a DefinitionList entry.
type DefinitionTerm struct{ Container }

// DefinitionDesc is the description half of a DefinitionList entry.
type DefinitionDesc struct{ Container }

// FootnoteDef is a footnote's defined body, numbered by Index to match its
// FootnoteRef reference sites.
type FootnoteDef struct {
	Container
	Index int // 1-based reference order
}

// Inline nodes.

// Text is a run of literal, already-unescaped text content.
type Text struct {
	Container
	Value string // literal text content
}

// SoftBreak is a line break within a paragraph that renders as whitespace
// (a newline in HTML).
type SoftBreak struct{ Container }

// HardBreak is an explicit line break (trailing double-space or backslash)
// that renders as <br>.
type HardBreak struct{ Container }

// Emphasis is *italic* text.
type Emphasis struct{ Container }

// Strong is **bold** text.
type Strong struct{ Container }

// Strikethrough is ~~struck-through~~ text (GFM extension).
type Strikethrough struct{ Container }

// CodeSpan is inline `code`.
type CodeSpan struct {
	Container
	Value string // literal code content
}

// Link is an inline [text](destination "title") link or autolink.
type Link struct {
	Container
	Destination string // link target URL
	Title       string // optional link title, "" if none
}

// Image is an inline ![alt](destination "title") image.
type Image struct {
	Container
	Destination string // image source URL
	Title       string // optional image title, "" if none
	Alt         string // plain-text rendering of the image's children
}

// WikiLink is a [[Target]] wiki-style link, resolved to a URL at render
// time.
type WikiLink struct {
	Container
	Target string // raw page reference, as written between [[ and ]]
}

// MathInline is inline math.
type MathInline struct {
	Container
	Source  string // math expression source, without delimiters
	Display bool   // true when a $$...$$ span renders as a block-level element rather than inline
}

// HTMLInline is a raw inline HTML span, sanitized at render time unless
// Options.Unsafe is set.
type HTMLInline struct {
	Container
	HTML string // raw source markup
}

// FootnoteRef is a [^label] reference site, numbered by Index to match its
// FootnoteDef.
type FootnoteRef struct {
	Container
	Index int // 1-based reference order, matching the target FootnoteDef
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

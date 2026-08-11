// Package tree builds the version-1 native render tree: a layout-free,
// fully RESOLVED semantic tree that native hosts (Flutter, SwiftUI,
// Compose, …) render as platform widgets, with everything policy-heavy
// — URL resolution and filtering, raw-HTML sanitizing, admonition
// titles, footnote pairing, math/mermaid fallbacks — already applied
// library-side by [Build], through the same shared derivations the HTML
// renderer uses (render/internal/derive and the resolve package), so
// the two renderers cannot drift.
//
// # Evolution within version 1
//
// The version stays 1 across additive growth: new OPTIONAL fields may
// appear on existing nodes, and new block/inline KINDS may appear as the
// document model grows. Consumers must therefore default-case on unknown
// kinds (render nothing, or a host-chosen placeholder) rather than treat
// the kind set as closed — the Flutter model's Unknown nodes and any
// TypeScript switch's default arm are the intended paths. A version bump
// is reserved for changes that break existing fields.
//
// # Wire schema (version 1, strict)
//
// [Tree.MarshalJSON] produces the envelope
//
//	{"version":1, "blocks":[…], "footnotes":[…]}
//
// where "blocks" and "footnotes" are always arrays (empty, never null).
// Every block object carries, in order: "kind", "span" (omitted when
// the node has no source position — same rule and shape as the document
// codec's spans: {startLine,endLine,startOffset,endOffset}), "id" (see
// "Block identity" below), its kind-specific fields, and its children —
// named "children" when they are inlines, "blocks" when they are nested
// blocks. Inline objects carry "kind", "span" (same omission rule), and
// kind-specific fields.
//
// Kind names are exactly the document codec's wire names (document.Kind
// String values): heading, paragraph, blockQuote, admonition, list,
// codeBlock, mathBlock, diagram, table, thematicBreak, htmlBlock,
// definitionList, definitionTerm, definitionDesc, footnoteDef, text,
// softBreak, hardBreak, emphasis, strong, strikethrough, codeSpan,
// link, image, mathInline, htmlInline, footnoteRef. One naming universe
// across the library: a kind in the doc JSON and the same kind in the
// render tree spell identically.
//
// Field presence is strict and documented per type below: fields that
// distinguish states are always emitted (codeBlock "runs" is null vs an
// array; list item "task" is null vs a boolean; link/image "url" is
// present even when empty), while purely-optional decorations use
// omit-when-empty ("anchorId", "title", "blocked", "display",
// wiki-link "source").
//
// # Block identity
//
// A block's "id" is hex(sha256(source bytes of the block's span))[:16]
// — the first 8 hash bytes, 16 hex characters — computed over
// Options.Source[span.StartOffset:span.EndOffset]. It is a CONTENT
// hash: stable across edits elsewhere in the document (the basis for
// host-side diffing / itemized rebuilds), which also means two blocks
// with byte-identical source share an id — hosts needing unique keys
// disambiguate by position among equal ids. When no source bytes are
// available for a block (Options.Source is empty or nil — e.g.
// building from decoded doc JSON — or the node carries the zero span),
// the id falls back to
// hex(sha256("\x00mdv-fallback\x00" + kind + ":" + ordinal))[:16],
// where ordinal is the block's 0-based position in a document-order
// count of every tree-emitted block (list items and footnotes
// included). The NUL-delimited prefix domain-separates the fallback
// preimage from the content-hash form — source bytes can never spell
// it, since decoded markdown contains no NUL — so a block whose source
// literally reads e.g. "list:2" cannot collide with a fallback id.
// Fallback ids are deterministic for a given document but positional,
// not content-stable.
package tree

import "github.com/sriannamalai/markdownviewer/document"

// Version is the render-tree wire-schema version emitted in the
// envelope's "version" field.
const Version = 1

// Tree is the built render tree. Its wire form is produced by
// [Tree.MarshalJSON]; see the package documentation for the schema.
type Tree struct {
	Blocks    []Block    // top-level blocks, in document order
	Footnotes []Footnote // footnote definitions, in first-reference order
}

// Block is a block-level node of the render tree. Concrete types:
// [Heading], [Paragraph], [BlockQuote], [Admonition], [List],
// [CodeBlock], [MathBlock], [Diagram], [Table], [ThematicBreak],
// [HTMLBlock], [DefinitionList], [DefinitionTerm], [DefinitionDesc].
type Block interface{ isBlock() }

// Inline is an inline node of the render tree. Concrete types: [Text],
// [SoftBreak], [HardBreak], [Emphasis], [Strong], [Strikethrough],
// [CodeSpan], [Link], [Image], [MathInline], [HTMLInline],
// [FootnoteRef].
type Inline interface{ isInline() }

// Heading is a section heading.
// Wire: {"kind":"heading",span,id,"level",("anchorId"),"children"}.
type Heading struct {
	Span     document.Span
	ID       string
	Level    int      // 1-6
	AnchorID string   // slugified per-document-unique anchor, "" (omitted on the wire) when Options.HeadingAnchors is off or the document carries none
	Children []Inline // heading content
}

// Paragraph is a run of inline content.
// Wire: {"kind":"paragraph",span,id,"children"}.
type Paragraph struct {
	Span     document.Span
	ID       string
	Children []Inline
}

// BlockQuote is quoted block content.
// Wire: {"kind":"blockQuote",span,id,"blocks"}.
type BlockQuote struct {
	Span   document.Span
	ID     string
	Blocks []Block
}

// Admonition is a callout box. Variant and Title come from the shared
// derivation (derive.AdmonitionTitle): Variant is normalized (empty →
// "note") and Title is its title-cased display form — the SAME title
// the HTML renderer shows.
// Wire: {"kind":"admonition",span,id,"variant","title","blocks"}.
type Admonition struct {
	Span    document.Span
	ID      string
	Variant string // note|tip|important|warning|caution (normalized)
	Title   string // derived display title ("Note", …)
	Blocks  []Block
}

// List is an ordered or unordered list.
// Wire: {"kind":"list",span,id,"ordered","start","tight","items"}.
type List struct {
	Span    document.Span
	ID      string
	Ordered bool
	Start   int // first item's number; meaningful when Ordered
	Tight   bool
	Items   []ListItem
}

// ListItem is one list entry. Task is nil for an ordinary item; for a
// task-list item it points at the checked state, so the wire "task"
// field is null | false | true.
// Wire: {span,id,"task","blocks"} — "task" always present.
type ListItem struct {
	Span   document.Span
	ID     string
	Task   *bool // nil: not a task item; else the checked state
	Blocks []Block
}

// TokenRun is one syntax-highlight token of a code block: a slice of
// the code text tagged with chroma's canonical token-type name.
type TokenRun struct {
	Text      string
	TokenType string // chroma TokenType.String() name, e.g. "Keyword"
}

// CodeBlock is a fenced or indented code block. Text always carries the
// full literal source. Runs is nil (wire: null) when highlighting is
// off or no token runs are available (unknown/missing language, chroma
// failure), an array otherwise — whose Text fields concatenate to
// exactly Text, so a host styling runs and a host printing Text show
// the same characters.
//
// Wire: {"kind":"codeBlock",span,id,"language","label","runs","text"}
// — "runs" and "text" always present.
type CodeBlock struct {
	Span     document.Span
	ID       string
	Language string     // fence info-string language tag, "" if none
	Label    string     // display label (derive.CodeLabel: tag or "code")
	Runs     []TokenRun // nil = no runs (highlighting off / unavailable)
	Text     string     // literal source text, always present
}

// MathBlock is display math for the host's math engine.
// Wire: {"kind":"mathBlock",span,id,"source","engine":"katex"}.
type MathBlock struct {
	Span   document.Span
	ID     string
	Source string // TeX source, without delimiters
}

// Diagram is a diagram definition for the host's diagram engine.
// Wire: {"kind":"diagram",span,id,"source","engine":"mermaid"}.
type Diagram struct {
	Span   document.Span
	ID     string
	Engine string // "mermaid"
	Source string
}

// Cell is one table cell: its inline content.
type Cell []Inline

// Table is a pipe table. Alignments has one entry per column; Header
// is the header row's cells (nil only for hand-built documents without
// a header row); Rows are the body rows.
// Wire: {"kind":"table",span,id,"alignments" (array of
// "none"|"left"|"center"|"right"),"header","rows"} — header and rows
// always arrays.
type Table struct {
	Span       document.Span
	ID         string
	Alignments []document.Alignment
	Header     []Cell
	Rows       [][]Cell
}

// ThematicBreak is a horizontal rule.
// Wire: {"kind":"thematicBreak",span,id}.
type ThematicBreak struct {
	Span document.Span
	ID   string
}

// HTMLBlock is a raw block of HTML markup. Unless Options.AllowRawHTML
// is set, HTML is the bluemonday-sanitized form (the shared
// derive.SanitizeHTML policy) and Unsafe is false; with AllowRawHTML,
// HTML is the raw source markup and Unsafe is true, telling the host
// the content was NOT sanitized.
// Wire: {"kind":"htmlBlock",span,id,"html","unsafe"} — both always
// present.
type HTMLBlock struct {
	Span   document.Span
	ID     string
	HTML   string
	Unsafe bool
}

// DefinitionList is a list of [DefinitionTerm]/[DefinitionDesc] blocks,
// in source order (a term may be followed by several descriptions and
// vice versa, so the pairing stays positional, mirroring the document
// model).
// Wire: {"kind":"definitionList",span,id,"blocks"}.
type DefinitionList struct {
	Span   document.Span
	ID     string
	Blocks []Block // DefinitionTerm / DefinitionDesc, in order
}

// DefinitionTerm is the term half of a definition-list entry.
// Wire: {"kind":"definitionTerm",span,id,"children"}.
type DefinitionTerm struct {
	Span     document.Span
	ID       string
	Children []Inline
}

// DefinitionDesc is the description half of a definition-list entry.
// Wire: {"kind":"definitionDesc",span,id,"blocks"}.
type DefinitionDesc struct {
	Span   document.Span
	ID     string
	Blocks []Block
}

// Footnote is one resolved footnote definition in the envelope's
// "footnotes" array, paired with its reference sites: Index matches the
// [FootnoteRef] inlines that cite it, and RefCount is the number of
// such sites (for hosts rendering per-reference backlinks). Pairing and
// order come from the shared derivation (derive.Footnotes).
// Wire: {"kind":"footnoteDef",span,id,"index","count","blocks"}.
type Footnote struct {
	Span     document.Span
	ID       string
	Index    int // 1-based, matches FootnoteRef.Index
	RefCount int
	Blocks   []Block
}

// Text is a literal text run (emoji shortcodes already substituted by
// the parser).
// Wire: {"kind":"text",span,"value"}.
type Text struct {
	Span  document.Span
	Value string
}

// SoftBreak is an intra-paragraph line break that renders as
// whitespace. It never carries a span (the parser records none).
// Wire: {"kind":"softBreak"}.
type SoftBreak struct{}

// HardBreak is an explicit line break. It never carries a span.
// Wire: {"kind":"hardBreak"}.
type HardBreak struct{}

// Emphasis is emphasized (italic) content.
// Wire: {"kind":"emphasis",span,"children"}.
type Emphasis struct {
	Span     document.Span
	Children []Inline
}

// Strong is strongly-emphasized (bold) content.
// Wire: {"kind":"strong",span,"children"}.
type Strong struct {
	Span     document.Span
	Children []Inline
}

// Strikethrough is struck-through content.
// Wire: {"kind":"strikethrough",span,"children"}.
type Strikethrough struct {
	Span     document.Span
	Children []Inline
}

// CodeSpan is inline code.
// Wire: {"kind":"codeSpan",span,"value"}.
type CodeSpan struct {
	Span  document.Span
	Value string
}

// Link is a hyperlink with its destination fully resolved: the shared
// pipeline (derive.Href — Resolver trust, resolve.DefaultResolution,
// resolve.SafeURL filtering, percent-encoding) has already run. A
// policy-blocked destination yields URL "" with Blocked true — distinct
// from an empty-but-allowed destination (URL "" and Blocked false).
// Resolver-returned URLs are host-trusted and carried verbatim.
//
// A wiki link resolves INTO a Link node (via the same pipeline, with
// the resolve.ResolveWikiLink kind and the default ".md" fallback);
// Source is then "wikiLink" so hosts can style it differently, and ""
// (omitted on the wire) for an ordinary link.
// Wire: {"kind":"link",span,"url",("blocked"),("title"),("source"),
// "children"} — "url" always present.
type Link struct {
	Span     document.Span
	URL      string
	Blocked  bool   // destination was blocked by URL policy
	Title    string // "" omitted on the wire
	Source   string // "wikiLink" when resolved from [[…]], else ""
	Children []Inline
}

// Image is an image with its destination resolved by the same pipeline
// as [Link] (a blocked destination yields URL "" + Blocked true, and a
// declined/absent resolver takes the default resolution path).
// Wire: {"kind":"image",span,"url",("blocked"),"alt",("title")} —
// "url" and "alt" always present.
type Image struct {
	Span    document.Span
	URL     string
	Blocked bool
	Alt     string
	Title   string
}

// MathInline is inline math for the host's math engine. Display marks a
// $$…$$ span that renders as display math despite sitting inline.
// Wire: {"kind":"mathInline",span,"source",("display")}.
type MathInline struct {
	Span    document.Span
	Source  string
	Display bool
}

// HTMLInline is a raw inline HTML span, sanitized/flagged exactly like
// [HTMLBlock].
// Wire: {"kind":"htmlInline",span,"html","unsafe"}.
type HTMLInline struct {
	Span   document.Span
	HTML   string
	Unsafe bool
}

// FootnoteRef is a footnote reference site; Index pairs it with the
// envelope [Footnote] carrying the same index. It never carries a span
// (the parser records none).
// Wire: {"kind":"footnoteRef","index"}.
type FootnoteRef struct {
	Index int
}

func (*Heading) isBlock()        {}
func (*Paragraph) isBlock()      {}
func (*BlockQuote) isBlock()     {}
func (*Admonition) isBlock()     {}
func (*List) isBlock()           {}
func (*CodeBlock) isBlock()      {}
func (*MathBlock) isBlock()      {}
func (*Diagram) isBlock()        {}
func (*Table) isBlock()          {}
func (*ThematicBreak) isBlock()  {}
func (*HTMLBlock) isBlock()      {}
func (*DefinitionList) isBlock() {}
func (*DefinitionTerm) isBlock() {}
func (*DefinitionDesc) isBlock() {}

func (*Text) isInline()          {}
func (*SoftBreak) isInline()     {}
func (*HardBreak) isInline()     {}
func (*Emphasis) isInline()      {}
func (*Strong) isInline()        {}
func (*Strikethrough) isInline() {}
func (*CodeSpan) isInline()      {}
func (*Link) isInline()          {}
func (*Image) isInline()         {}
func (*MathInline) isInline()    {}
func (*HTMLInline) isInline()    {}
func (*FootnoteRef) isInline()   {}

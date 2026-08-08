package document

import (
	"encoding/json"
	"fmt"
)

const jsonVersion = 1

type jsonSpan struct {
	StartLine   int `json:"startLine"`
	EndLine     int `json:"endLine"`
	StartOffset int `json:"startOffset"`
	EndOffset   int `json:"endOffset"`
}

type jsonNode struct {
	Kind        string      `json:"kind"`
	Level       int         `json:"level,omitempty"`
	AnchorID    string      `json:"anchorId,omitempty"`
	Variant     string      `json:"variant,omitempty"`
	Ordered     bool        `json:"ordered,omitempty"`
	Start       int         `json:"start,omitempty"`
	Tight       bool        `json:"tight,omitempty"`
	Task        bool        `json:"task,omitempty"`
	Checked     bool        `json:"checked,omitempty"`
	Language    string      `json:"language,omitempty"`
	Code        string      `json:"code,omitempty"`
	Engine      string      `json:"engine,omitempty"`
	Source      string      `json:"source,omitempty"`
	Alignments  []string    `json:"alignments,omitempty"`
	Header      bool        `json:"header,omitempty"`
	HTML        string      `json:"html,omitempty"`
	Index       int         `json:"index,omitempty"`
	Value       string      `json:"value,omitempty"`
	Destination string      `json:"destination,omitempty"`
	Title       string      `json:"title,omitempty"`
	Alt         string      `json:"alt,omitempty"`
	Target      string      `json:"target,omitempty"`
	Display     bool        `json:"display,omitempty"`
	Span        *jsonSpan   `json:"span,omitempty"`
	Children    []*jsonNode `json:"children,omitempty"`
}

type jsonDoc struct {
	Version   int            `json:"version"`
	Kind      string         `json:"kind"`
	Meta      map[string]any `json:"meta,omitempty"`
	Span      *jsonSpan      `json:"span,omitempty"`
	Footnotes []*jsonNode    `json:"footnotes,omitempty"`
	Children  []*jsonNode    `json:"children,omitempty"`
}

var alignNames = map[Alignment]string{
	AlignNone: "none", AlignLeft: "left", AlignCenter: "center", AlignRight: "right",
}
var alignByName = map[string]Alignment{
	"none": AlignNone, "left": AlignLeft, "center": AlignCenter, "right": AlignRight,
}

// MarshalJSON encodes a document tree in the stable v1 wire format: nodes
// as {"kind":"<name>",...} objects with lowerCamel fields, zero spans and
// empty children omitted. Meta values must be JSON-marshalable; YAML
// timestamps round-trip as RFC 3339 strings.
func MarshalJSON(doc *Document) ([]byte, error) {
	env := &jsonDoc{Version: jsonVersion, Kind: KindDocument.String(), Meta: doc.Meta}
	env.Span = spanOut(doc.Span())
	for _, f := range doc.Footnotes {
		env.Footnotes = append(env.Footnotes, nodeOut(f))
	}
	for _, c := range doc.Children() {
		env.Children = append(env.Children, nodeOut(c))
	}
	return json.Marshal(env)
}

func spanOut(s Span) *jsonSpan {
	if s.IsZero() {
		return nil
	}
	return &jsonSpan{s.StartLine, s.EndLine, s.StartOffset, s.EndOffset}
}

func nodeOut(n Node) *jsonNode {
	j := &jsonNode{Kind: n.Kind().String()}
	if sp, ok := n.(interface{ Span() Span }); ok {
		j.Span = spanOut(sp.Span())
	}
	switch n := n.(type) {
	case *Heading:
		j.Level, j.AnchorID = n.Level, n.AnchorID
	case *Admonition:
		j.Variant = n.Variant
	case *List:
		j.Ordered, j.Start, j.Tight = n.Ordered, n.Start, n.Tight
	case *ListItem:
		j.Task, j.Checked = n.Task, n.Checked
	case *CodeBlock:
		j.Language, j.Code = n.Language, n.Code
	case *Diagram:
		j.Engine, j.Source = n.Engine, n.Source
	case *MathBlock:
		j.Source = n.Source
	case *Table:
		for _, a := range n.Alignments {
			j.Alignments = append(j.Alignments, alignNames[a])
		}
	case *TableRow:
		j.Header = n.Header
	case *HTMLBlock:
		j.HTML = n.HTML
	case *FootnoteDef:
		j.Index = n.Index
	case *Text:
		j.Value = n.Value
	case *CodeSpan:
		j.Value = n.Value
	case *Link:
		j.Destination, j.Title = n.Destination, n.Title
	case *Image:
		j.Destination, j.Title, j.Alt = n.Destination, n.Title, n.Alt
	case *WikiLink:
		j.Target = n.Target
	case *MathInline:
		j.Source, j.Display = n.Source, n.Display
	case *HTMLInline:
		j.HTML = n.HTML
	case *FootnoteRef:
		j.Index = n.Index
	}
	for _, c := range n.Children() {
		j.Children = append(j.Children, nodeOut(c))
	}
	return j
}

// UnmarshalJSON decodes the v1 wire format produced by MarshalJSON.
func UnmarshalJSON(data []byte) (*Document, error) {
	var env jsonDoc
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("document: invalid JSON: %w", err)
	}
	if env.Version != jsonVersion {
		return nil, fmt.Errorf("document: unsupported version %d (want %d)", env.Version, jsonVersion)
	}
	doc := &Document{Meta: env.Meta}
	if env.Span != nil {
		doc.SetSpan(Span{env.Span.StartLine, env.Span.EndLine, env.Span.StartOffset, env.Span.EndOffset})
	}
	for _, jf := range env.Footnotes {
		n, err := nodeIn(jf)
		if err != nil {
			return nil, err
		}
		def, ok := n.(*FootnoteDef)
		if !ok {
			return nil, fmt.Errorf("document: footnote entry has kind %q", jf.Kind)
		}
		doc.Footnotes = append(doc.Footnotes, def)
	}
	for _, jc := range env.Children {
		n, err := nodeIn(jc)
		if err != nil {
			return nil, err
		}
		doc.AppendChild(n)
	}
	return doc, nil
}

func nodeIn(j *jsonNode) (Node, error) {
	k, ok := KindFromString(j.Kind)
	if !ok {
		return nil, fmt.Errorf("document: unknown node kind %q", j.Kind)
	}
	var n Node
	switch k {
	case KindHeading:
		n = &Heading{Level: j.Level, AnchorID: j.AnchorID}
	case KindParagraph:
		n = &Paragraph{}
	case KindBlockQuote:
		n = &BlockQuote{}
	case KindAdmonition:
		n = &Admonition{Variant: j.Variant}
	case KindList:
		n = &List{Ordered: j.Ordered, Start: j.Start, Tight: j.Tight}
	case KindListItem:
		n = &ListItem{Task: j.Task, Checked: j.Checked}
	case KindCodeBlock:
		n = &CodeBlock{Language: j.Language, Code: j.Code}
	case KindDiagram:
		n = &Diagram{Engine: j.Engine, Source: j.Source}
	case KindMathBlock:
		n = &MathBlock{Source: j.Source}
	case KindTable:
		t := &Table{}
		for _, a := range j.Alignments {
			al, ok := alignByName[a]
			if !ok {
				return nil, fmt.Errorf("document: unknown alignment %q", a)
			}
			t.Alignments = append(t.Alignments, al)
		}
		n = t
	case KindTableRow:
		n = &TableRow{Header: j.Header}
	case KindTableCell:
		n = &TableCell{}
	case KindThematicBreak:
		n = &ThematicBreak{}
	case KindHTMLBlock:
		n = &HTMLBlock{HTML: j.HTML}
	case KindDefinitionList:
		n = &DefinitionList{}
	case KindDefinitionTerm:
		n = &DefinitionTerm{}
	case KindDefinitionDesc:
		n = &DefinitionDesc{}
	case KindFootnoteDef:
		n = &FootnoteDef{Index: j.Index}
	case KindText:
		n = &Text{Value: j.Value}
	case KindSoftBreak:
		n = &SoftBreak{}
	case KindHardBreak:
		n = &HardBreak{}
	case KindEmphasis:
		n = &Emphasis{}
	case KindStrong:
		n = &Strong{}
	case KindStrikethrough:
		n = &Strikethrough{}
	case KindCodeSpan:
		n = &CodeSpan{Value: j.Value}
	case KindLink:
		n = &Link{Destination: j.Destination, Title: j.Title}
	case KindImage:
		n = &Image{Destination: j.Destination, Title: j.Title, Alt: j.Alt}
	case KindWikiLink:
		n = &WikiLink{Target: j.Target}
	case KindMathInline:
		n = &MathInline{Source: j.Source, Display: j.Display}
	case KindHTMLInline:
		n = &HTMLInline{HTML: j.HTML}
	case KindFootnoteRef:
		n = &FootnoteRef{Index: j.Index}
	case KindDocument:
		return nil, fmt.Errorf("document: nested document node not allowed")
	default:
		return nil, fmt.Errorf("document: kind %q not constructible", j.Kind)
	}
	if j.Span != nil {
		if sp, ok := n.(interface{ SetSpan(Span) }); ok {
			sp.SetSpan(Span{j.Span.StartLine, j.Span.EndLine, j.Span.StartOffset, j.Span.EndOffset})
		}
	}
	for _, jc := range j.Children {
		c, err := nodeIn(jc)
		if err != nil {
			return nil, err
		}
		n.AppendChild(c)
	}
	return n, nil
}

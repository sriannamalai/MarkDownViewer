package tree

import (
	"encoding/json"
	"fmt"

	"github.com/sriannamalai/markdownviewer/document"
)

// MarshalJSON encodes the tree in the strict version-1 wire format
// documented on the package: {"version":1,"blocks":[…],"footnotes":[…]}
// with per-kind block/inline objects. Field order within each object is
// fixed (kind, span, id, kind fields, children/blocks) and presence
// rules are strict — see each node type's Wire comment.
func (t *Tree) MarshalJSON() ([]byte, error) {
	blocks, err := blocksOut(t.Blocks)
	if err != nil {
		return nil, err
	}
	footnotes := make([]any, 0, len(t.Footnotes))
	for _, fn := range t.Footnotes {
		bs, err := blocksOut(fn.Blocks)
		if err != nil {
			return nil, err
		}
		footnotes = append(footnotes, &wFootnote{
			wHead:    head(document.KindFootnoteDef, fn.Span, fn.ID),
			Index:    fn.Index,
			RefCount: fn.RefCount,
			Blocks:   bs,
		})
	}
	return json.Marshal(&struct {
		Version   int   `json:"version"`
		Blocks    []any `json:"blocks"`
		Footnotes []any `json:"footnotes"`
	}{Version: Version, Blocks: blocks, Footnotes: footnotes})
}

// wSpan is the wire span, identical to the document codec's shape.
type wSpan struct {
	StartLine   int `json:"startLine"`
	EndLine     int `json:"endLine"`
	StartOffset int `json:"startOffset"`
	EndOffset   int `json:"endOffset"`
}

func spanOut(s document.Span) *wSpan {
	if s.IsZero() {
		return nil
	}
	return &wSpan{s.StartLine, s.EndLine, s.StartOffset, s.EndOffset}
}

// wHead is the leading triple every block object carries.
type wHead struct {
	Kind string `json:"kind"`
	Span *wSpan `json:"span,omitempty"`
	ID   string `json:"id"`
}

func head(k document.Kind, span document.Span, id string) wHead {
	return wHead{Kind: k.String(), Span: spanOut(span), ID: id}
}

// wIHead is the leading pair every inline object carries.
type wIHead struct {
	Kind string `json:"kind"`
	Span *wSpan `json:"span,omitempty"`
}

func ihead(k document.Kind, span document.Span) wIHead {
	return wIHead{Kind: k.String(), Span: spanOut(span)}
}

type (
	wHeading struct {
		wHead
		Level    int    `json:"level"`
		AnchorID string `json:"anchorId,omitempty"`
		Children []any  `json:"children"`
	}
	wParagraph struct {
		wHead
		Children []any `json:"children"`
	}
	wBlockQuote struct {
		wHead
		Blocks []any `json:"blocks"`
	}
	wAdmonition struct {
		wHead
		Variant string `json:"variant"`
		Title   string `json:"title"`
		Blocks  []any  `json:"blocks"`
	}
	wList struct {
		wHead
		Ordered bool    `json:"ordered"`
		Start   int     `json:"start"`
		Tight   bool    `json:"tight"`
		Items   []wItem `json:"items"`
	}
	wItem struct {
		Span   *wSpan `json:"span,omitempty"`
		ID     string `json:"id"`
		Task   *bool  `json:"task"` // null | false | true — always present
		Blocks []any  `json:"blocks"`
	}
	wRun struct {
		Text      string `json:"text"`
		TokenType string `json:"tokenType"`
	}
	wCodeBlock struct {
		wHead
		Language string `json:"language"`
		Label    string `json:"label"`
		Runs     []wRun `json:"runs"` // null when no runs — always present
		Text     string `json:"text"`
	}
	wMathBlock struct {
		wHead
		Source string `json:"source"`
		Engine string `json:"engine"`
	}
	wDiagram struct {
		wHead
		Source string `json:"source"`
		Engine string `json:"engine"`
	}
	wTable struct {
		wHead
		Alignments []string  `json:"alignments"`
		Header     [][]any   `json:"header"`
		Rows       [][][]any `json:"rows"`
	}
	wThematicBreak struct {
		wHead
	}
	wHTMLBlock struct {
		wHead
		HTML   string `json:"html"`
		Unsafe bool   `json:"unsafe"`
	}
	wDefinitionList struct {
		wHead
		Blocks []any `json:"blocks"`
	}
	wDefinitionTerm struct {
		wHead
		Children []any `json:"children"`
	}
	wDefinitionDesc struct {
		wHead
		Blocks []any `json:"blocks"`
	}
	wFootnote struct {
		wHead
		Index    int   `json:"index"`
		RefCount int   `json:"refCount"`
		Blocks   []any `json:"blocks"`
	}

	wText struct {
		wIHead
		Value string `json:"value"`
	}
	wBreak struct {
		wIHead
	}
	wContainer struct {
		wIHead
		Children []any `json:"children"`
	}
	wCodeSpan struct {
		wIHead
		Value string `json:"value"`
	}
	wLink struct {
		wIHead
		URL      string `json:"url"`
		Blocked  bool   `json:"blocked,omitempty"`
		Title    string `json:"title,omitempty"`
		Source   string `json:"source,omitempty"`
		Children []any  `json:"children"`
	}
	wImage struct {
		wIHead
		URL     string `json:"url"`
		Blocked bool   `json:"blocked,omitempty"`
		Alt     string `json:"alt"`
		Title   string `json:"title,omitempty"`
	}
	wMathInline struct {
		wIHead
		Source  string `json:"source"`
		Display bool   `json:"display,omitempty"`
	}
	wHTMLInline struct {
		wIHead
		HTML   string `json:"html"`
		Unsafe bool   `json:"unsafe"`
	}
	wFootnoteRef struct {
		Kind  string `json:"kind"`
		Index int    `json:"index"`
	}
)

var alignNames = map[document.Alignment]string{
	document.AlignNone: "none", document.AlignLeft: "left",
	document.AlignCenter: "center", document.AlignRight: "right",
}

func blocksOut(bs []Block) ([]any, error) {
	out := make([]any, 0, len(bs))
	for _, b := range bs {
		w, err := blockOut(b)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, nil
}

func blockOut(b Block) (any, error) {
	switch b := b.(type) {
	case *Heading:
		children, err := inlinesOut(b.Children)
		if err != nil {
			return nil, err
		}
		return &wHeading{wHead: head(document.KindHeading, b.Span, b.ID),
			Level: b.Level, AnchorID: b.AnchorID, Children: children}, nil
	case *Paragraph:
		children, err := inlinesOut(b.Children)
		if err != nil {
			return nil, err
		}
		return &wParagraph{wHead: head(document.KindParagraph, b.Span, b.ID), Children: children}, nil
	case *BlockQuote:
		blocks, err := blocksOut(b.Blocks)
		if err != nil {
			return nil, err
		}
		return &wBlockQuote{wHead: head(document.KindBlockQuote, b.Span, b.ID), Blocks: blocks}, nil
	case *Admonition:
		blocks, err := blocksOut(b.Blocks)
		if err != nil {
			return nil, err
		}
		return &wAdmonition{wHead: head(document.KindAdmonition, b.Span, b.ID),
			Variant: b.Variant, Title: b.Title, Blocks: blocks}, nil
	case *List:
		items := make([]wItem, 0, len(b.Items))
		for _, it := range b.Items {
			blocks, err := blocksOut(it.Blocks)
			if err != nil {
				return nil, err
			}
			items = append(items, wItem{Span: spanOut(it.Span), ID: it.ID, Task: it.Task, Blocks: blocks})
		}
		return &wList{wHead: head(document.KindList, b.Span, b.ID),
			Ordered: b.Ordered, Start: b.Start, Tight: b.Tight, Items: items}, nil
	case *CodeBlock:
		var runs []wRun // nil marshals as the schema's null
		if b.Runs != nil {
			runs = make([]wRun, 0, len(b.Runs))
			for _, r := range b.Runs {
				runs = append(runs, wRun(r))
			}
		}
		return &wCodeBlock{wHead: head(document.KindCodeBlock, b.Span, b.ID),
			Language: b.Language, Label: b.Label, Runs: runs, Text: b.Text}, nil
	case *MathBlock:
		return &wMathBlock{wHead: head(document.KindMathBlock, b.Span, b.ID),
			Source: b.Source, Engine: "katex"}, nil
	case *Diagram:
		return &wDiagram{wHead: head(document.KindDiagram, b.Span, b.ID),
			Source: b.Source, Engine: b.Engine}, nil
	case *Table:
		aligns := make([]string, 0, len(b.Alignments))
		for _, a := range b.Alignments {
			aligns = append(aligns, alignNames[a])
		}
		header, err := cellsOut(b.Header)
		if err != nil {
			return nil, err
		}
		rows := make([][][]any, 0, len(b.Rows))
		for _, row := range b.Rows {
			cells, err := cellsOut(row)
			if err != nil {
				return nil, err
			}
			rows = append(rows, cells)
		}
		return &wTable{wHead: head(document.KindTable, b.Span, b.ID),
			Alignments: aligns, Header: header, Rows: rows}, nil
	case *ThematicBreak:
		return &wThematicBreak{wHead: head(document.KindThematicBreak, b.Span, b.ID)}, nil
	case *HTMLBlock:
		return &wHTMLBlock{wHead: head(document.KindHTMLBlock, b.Span, b.ID),
			HTML: b.HTML, Unsafe: b.Unsafe}, nil
	case *DefinitionList:
		blocks, err := blocksOut(b.Blocks)
		if err != nil {
			return nil, err
		}
		return &wDefinitionList{wHead: head(document.KindDefinitionList, b.Span, b.ID), Blocks: blocks}, nil
	case *DefinitionTerm:
		children, err := inlinesOut(b.Children)
		if err != nil {
			return nil, err
		}
		return &wDefinitionTerm{wHead: head(document.KindDefinitionTerm, b.Span, b.ID), Children: children}, nil
	case *DefinitionDesc:
		blocks, err := blocksOut(b.Blocks)
		if err != nil {
			return nil, err
		}
		return &wDefinitionDesc{wHead: head(document.KindDefinitionDesc, b.Span, b.ID), Blocks: blocks}, nil
	default:
		return nil, fmt.Errorf("tree: unencodable block type %T", b)
	}
}

func cellsOut(cells []Cell) ([][]any, error) {
	out := make([][]any, 0, len(cells))
	for _, c := range cells {
		inls, err := inlinesOut(c)
		if err != nil {
			return nil, err
		}
		out = append(out, inls)
	}
	return out, nil
}

func inlinesOut(is []Inline) ([]any, error) {
	out := make([]any, 0, len(is))
	for _, i := range is {
		w, err := inlineOut(i)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, nil
}

func inlineOut(i Inline) (any, error) {
	switch i := i.(type) {
	case *Text:
		return &wText{wIHead: ihead(document.KindText, i.Span), Value: i.Value}, nil
	case *SoftBreak:
		return &wBreak{wIHead: ihead(document.KindSoftBreak, document.Span{})}, nil
	case *HardBreak:
		return &wBreak{wIHead: ihead(document.KindHardBreak, document.Span{})}, nil
	case *Emphasis:
		children, err := inlinesOut(i.Children)
		if err != nil {
			return nil, err
		}
		return &wContainer{wIHead: ihead(document.KindEmphasis, i.Span), Children: children}, nil
	case *Strong:
		children, err := inlinesOut(i.Children)
		if err != nil {
			return nil, err
		}
		return &wContainer{wIHead: ihead(document.KindStrong, i.Span), Children: children}, nil
	case *Strikethrough:
		children, err := inlinesOut(i.Children)
		if err != nil {
			return nil, err
		}
		return &wContainer{wIHead: ihead(document.KindStrikethrough, i.Span), Children: children}, nil
	case *CodeSpan:
		return &wCodeSpan{wIHead: ihead(document.KindCodeSpan, i.Span), Value: i.Value}, nil
	case *Link:
		children, err := inlinesOut(i.Children)
		if err != nil {
			return nil, err
		}
		return &wLink{wIHead: ihead(document.KindLink, i.Span), URL: i.URL,
			Blocked: i.Blocked, Title: i.Title, Source: i.Source, Children: children}, nil
	case *Image:
		return &wImage{wIHead: ihead(document.KindImage, i.Span), URL: i.URL,
			Blocked: i.Blocked, Alt: i.Alt, Title: i.Title}, nil
	case *MathInline:
		return &wMathInline{wIHead: ihead(document.KindMathInline, i.Span),
			Source: i.Source, Display: i.Display}, nil
	case *HTMLInline:
		return &wHTMLInline{wIHead: ihead(document.KindHTMLInline, i.Span),
			HTML: i.HTML, Unsafe: i.Unsafe}, nil
	case *FootnoteRef:
		return &wFootnoteRef{Kind: document.KindFootnoteRef.String(), Index: i.Index}, nil
	default:
		return nil, fmt.Errorf("tree: unencodable inline type %T", i)
	}
}

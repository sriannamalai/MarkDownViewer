package parser

import (
	"strings"

	emast "github.com/yuin/goldmark-emoji/ast"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	gparser "github.com/yuin/goldmark/parser"
	"go.abhg.dev/goldmark/frontmatter"
	"go.abhg.dev/goldmark/wikilink"

	"github.com/sriannamalai/markdownviewer/document"
)

type transformer struct {
	src       []byte
	cfg       Config
	slugs     map[string]int
	item      *document.ListItem // innermost list item, for task checkboxes
	footnotes []*document.FootnoteDef
}

func (t *transformer) document(root ast.Node, ctx gparser.Context) *document.Document {
	doc := &document.Document{}
	t.appendChildren(doc, root)
	doc.Footnotes = t.footnotes
	if d := frontmatter.Get(ctx); d != nil {
		var m map[string]any
		if err := d.Decode(&m); err == nil {
			doc.Meta = m
		}
	}
	return doc
}

func (t *transformer) appendChildren(parent document.Node, n ast.Node) {
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if out := t.convert(c); out != nil {
			// The Linkify inline parser triggers on plain spaces and, when
			// no URL/email matches, leaves the surrounding prose split
			// across sibling ast.Text nodes instead of one merged run.
			// Coalesce adjacent document.Text nodes so output stays
			// identical to the non-Linkify tree. Scoped to ast.Text inputs
			// only, so it doesn't also swallow synthesized Text nodes from
			// other node kinds (e.g. emoji shortcodes) that must stay
			// distinct siblings.
			if _, isASTText := c.(*ast.Text); isASTText {
				if txt, ok := out.(*document.Text); ok {
					if kids := parent.Children(); len(kids) > 0 {
						if last, ok := kids[len(kids)-1].(*document.Text); ok {
							last.Value += txt.Value
							out = nil
						}
					}
				}
			}
			if out != nil {
				parent.AppendChild(out)
			}
		}
		// ast.Text carries trailing break flags; emit them as siblings.
		if txt, ok := c.(*ast.Text); ok {
			if txt.HardLineBreak() {
				parent.AppendChild(&document.HardBreak{})
			} else if txt.SoftLineBreak() {
				parent.AppendChild(&document.SoftBreak{})
			}
		}
	}
}

func (t *transformer) convert(n ast.Node) document.Node {
	switch n := n.(type) {
	case *ast.Heading:
		h := &document.Heading{Level: n.Level}
		t.appendChildren(h, n)
		h.AnchorID = t.slug(document.PlainText(h))
		return h
	case *ast.Paragraph:
		p := &document.Paragraph{}
		t.appendChildren(p, n)
		return p
	case *ast.TextBlock: // tight-list item content; List.Tight drives <p> omission
		p := &document.Paragraph{}
		t.appendChildren(p, n)
		return p
	case *ast.Blockquote:
		bq := &document.BlockQuote{}
		t.appendChildren(bq, n)
		if t.cfg.Admonitions {
			if adm := promoteAdmonition(bq); adm != nil {
				return adm
			}
		}
		return bq
	case *ast.List:
		l := &document.List{Ordered: n.IsOrdered(), Start: 1, Tight: n.IsTight}
		if n.IsOrdered() {
			l.Start = n.Start
		}
		t.appendChildren(l, n)
		return l
	case *ast.ListItem:
		li := &document.ListItem{}
		prev := t.item
		t.item = li
		t.appendChildren(li, n)
		t.item = prev
		return li
	case *ast.FencedCodeBlock:
		lang := string(n.Language(t.src))
		return t.codeOrSpecial(lang, blockLines(n, t.src))
	case *ast.CodeBlock:
		return &document.CodeBlock{Code: blockLines(n, t.src)}
	case *ast.ThematicBreak:
		return &document.ThematicBreak{}
	case *ast.HTMLBlock:
		html := blockLines(n, t.src)
		if n.HasClosure() {
			html += string(n.ClosureLine.Value(t.src))
		}
		return &document.HTMLBlock{HTML: html}
	case *ast.Text:
		return &document.Text{Value: string(n.Segment.Value(t.src))}
	case *ast.String:
		return &document.Text{Value: string(n.Value)}
	case *ast.Emphasis:
		var out document.Node
		if n.Level >= 2 {
			out = &document.Strong{}
		} else {
			out = &document.Emphasis{}
		}
		t.appendChildren(out, n)
		return out
	case *ast.CodeSpan:
		return &document.CodeSpan{Value: spanText(n, t.src)}
	case *ast.Link:
		l := &document.Link{Destination: string(n.Destination), Title: string(n.Title)}
		t.appendChildren(l, n)
		return l
	case *ast.AutoLink:
		url := string(n.URL(t.src))
		if n.AutoLinkType == ast.AutoLinkEmail && !strings.HasPrefix(url, "mailto:") {
			url = "mailto:" + url
		}
		l := &document.Link{Destination: url}
		l.AppendChild(&document.Text{Value: string(n.Label(t.src))})
		return l
	case *ast.Image:
		img := &document.Image{Destination: string(n.Destination), Title: string(n.Title)}
		t.appendChildren(img, n)
		img.Alt = document.PlainText(img)
		return img
	case *ast.RawHTML:
		var b strings.Builder
		for i := 0; i < n.Segments.Len(); i++ {
			seg := n.Segments.At(i)
			b.Write(seg.Value(t.src))
		}
		return &document.HTMLInline{HTML: b.String()}
	case *east.Table:
		tbl := &document.Table{}
		for _, a := range n.Alignments {
			tbl.Alignments = append(tbl.Alignments, map[east.Alignment]document.Alignment{
				east.AlignNone: document.AlignNone, east.AlignLeft: document.AlignLeft,
				east.AlignCenter: document.AlignCenter, east.AlignRight: document.AlignRight,
			}[a])
		}
		t.appendChildren(tbl, n)
		return tbl
	case *east.TableHeader:
		row := &document.TableRow{Header: true}
		t.appendChildren(row, n)
		return row
	case *east.TableRow:
		row := &document.TableRow{}
		t.appendChildren(row, n)
		return row
	case *east.TableCell:
		cell := &document.TableCell{}
		t.appendChildren(cell, n)
		return cell
	case *east.Strikethrough:
		s := &document.Strikethrough{}
		t.appendChildren(s, n)
		return s
	case *east.TaskCheckBox:
		if t.item != nil {
			t.item.Task = true
			t.item.Checked = n.IsChecked
		}
		return nil
	case *east.FootnoteLink:
		return &document.FootnoteRef{Index: n.Index}
	case *east.FootnoteBacklink:
		return nil // renderer generates backlinks
	case *east.FootnoteList:
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if def, ok := t.convert(c).(*document.FootnoteDef); ok {
				t.footnotes = append(t.footnotes, def)
			}
		}
		return nil
	case *east.Footnote:
		def := &document.FootnoteDef{Index: n.Index}
		t.appendChildren(def, n)
		return def
	case *emast.Emoji:
		return &document.Text{Value: string(n.Value.Unicode)}
	case *east.DefinitionList:
		dl := &document.DefinitionList{}
		t.appendChildren(dl, n)
		return dl
	case *east.DefinitionTerm:
		dt := &document.DefinitionTerm{}
		t.appendChildren(dt, n)
		return dt
	case *east.DefinitionDescription:
		dd := &document.DefinitionDesc{}
		t.appendChildren(dd, n)
		return dd
	case *wikilink.Node:
		wl := &document.WikiLink{Target: string(n.Target)}
		t.appendChildren(wl, n)
		return wl
	default:
		return nil // extension nodes handled in Tasks 3-8
	}
}

var admonitionVariants = map[string]bool{
	"NOTE": true, "TIP": true, "IMPORTANT": true, "WARNING": true, "CAUTION": true,
}

// promoteAdmonition returns an Admonition if bq's first paragraph starts
// with a GitHub-style [!VARIANT] marker; the marker (and a following
// SoftBreak) is stripped. Returns nil when bq is a plain quote.
func promoteAdmonition(bq *document.BlockQuote) *document.Admonition {
	kids := bq.Children()
	if len(kids) == 0 {
		return nil
	}
	p, ok := kids[0].(*document.Paragraph)
	if !ok {
		return nil
	}
	inl := p.Children()
	if len(inl) == 0 {
		return nil
	}
	txt, ok := inl[0].(*document.Text)
	if !ok || len(txt.Value) < 4 || !strings.HasPrefix(txt.Value, "[!") || !strings.HasSuffix(txt.Value, "]") {
		return nil
	}
	name := txt.Value[2 : len(txt.Value)-1]
	if !admonitionVariants[name] {
		return nil
	}
	adm := &document.Admonition{Variant: strings.ToLower(name)}
	rest := inl[1:]
	if len(rest) > 0 {
		if _, isBreak := rest[0].(*document.SoftBreak); isBreak {
			rest = rest[1:]
		}
	}
	if len(rest) > 0 {
		np := &document.Paragraph{}
		for _, c := range rest {
			np.AppendChild(c)
		}
		adm.AppendChild(np)
	}
	for _, c := range kids[1:] {
		adm.AppendChild(c)
	}
	return adm
}

// codeOrSpecial maps special fence languages to dedicated nodes.
func (t *transformer) codeOrSpecial(lang, code string) document.Node {
	switch lang {
	case "mermaid":
		return &document.Diagram{Engine: "mermaid", Source: code}
	case "math":
		return &document.MathBlock{Source: code}
	}
	return &document.CodeBlock{Language: lang, Code: code}
}

func blockLines(n ast.Node, src []byte) string {
	var b strings.Builder
	l := n.Lines()
	for i := 0; i < l.Len(); i++ {
		seg := l.At(i)
		b.Write(seg.Value(src))
	}
	return b.String()
}

func spanText(n ast.Node, src []byte) string {
	var b strings.Builder
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if txt, ok := c.(*ast.Text); ok {
			b.Write(txt.Segment.Value(src))
		}
	}
	return b.String()
}

package document

import (
	"fmt"
	"strings"
)

// Dump renders a node tree as an indented text outline for tests.
func Dump(n Node) string {
	var b strings.Builder
	dump(&b, n, 0)
	return b.String()
}

func dump(b *strings.Builder, n Node, depth int) {
	b.WriteString(strings.Repeat("  ", depth))
	b.WriteString(label(n))
	b.WriteByte('\n')
	for _, c := range n.Children() {
		dump(b, c, depth+1)
	}
}

func label(n Node) string {
	switch n := n.(type) {
	case *Document:
		return "Document"
	case *Heading:
		return fmt.Sprintf("Heading[%d] id=%q", n.Level, n.AnchorID)
	case *Paragraph:
		return "Paragraph"
	case *BlockQuote:
		return "BlockQuote"
	case *Admonition:
		return fmt.Sprintf("Admonition[%s]", n.Variant)
	case *List:
		return fmt.Sprintf("List ordered=%t start=%d tight=%t", n.Ordered, n.Start, n.Tight)
	case *ListItem:
		if n.Task {
			return fmt.Sprintf("ListItem task checked=%t", n.Checked)
		}
		return "ListItem"
	case *CodeBlock:
		return fmt.Sprintf("CodeBlock lang=%q %q", n.Language, n.Code)
	case *Diagram:
		return fmt.Sprintf("Diagram[%s] %q", n.Engine, n.Source)
	case *MathBlock:
		return fmt.Sprintf("MathBlock %q", n.Source)
	case *Table:
		parts := make([]string, len(n.Alignments))
		for i, a := range n.Alignments {
			parts[i] = [...]string{"none", "left", "center", "right"}[a]
		}
		return fmt.Sprintf("Table aligns=[%s]", strings.Join(parts, ","))
	case *TableRow:
		return fmt.Sprintf("TableRow header=%t", n.Header)
	case *TableCell:
		return "TableCell"
	case *ThematicBreak:
		return "ThematicBreak"
	case *HTMLBlock:
		return fmt.Sprintf("HTMLBlock %q", n.HTML)
	case *DefinitionList:
		return "DefinitionList"
	case *DefinitionTerm:
		return "DefinitionTerm"
	case *DefinitionDesc:
		return "DefinitionDesc"
	case *FootnoteDef:
		return fmt.Sprintf("FootnoteDef[%d]", n.Index)
	case *Text:
		return fmt.Sprintf("Text %q", n.Value)
	case *SoftBreak:
		return "SoftBreak"
	case *HardBreak:
		return "HardBreak"
	case *Emphasis:
		return "Emphasis"
	case *Strong:
		return "Strong"
	case *Strikethrough:
		return "Strikethrough"
	case *CodeSpan:
		return fmt.Sprintf("CodeSpan %q", n.Value)
	case *Link:
		return fmt.Sprintf("Link dest=%q title=%q", n.Destination, n.Title)
	case *Image:
		return fmt.Sprintf("Image dest=%q alt=%q title=%q", n.Destination, n.Alt, n.Title)
	case *WikiLink:
		return fmt.Sprintf("WikiLink target=%q", n.Target)
	case *MathInline:
		return fmt.Sprintf("MathInline display=%t %q", n.Display, n.Source)
	case *HTMLInline:
		return fmt.Sprintf("HTMLInline %q", n.HTML)
	case *FootnoteRef:
		return fmt.Sprintf("FootnoteRef[%d]", n.Index)
	default:
		return fmt.Sprintf("UNKNOWN(%T)", n)
	}
}

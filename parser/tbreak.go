package parser

import (
	"github.com/yuin/goldmark/ast"
	gparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// thematicBreakParser replicates goldmark's stock thematic-break block
// parser (goldmark/parser/thematic_break.go) with one addition: it appends
// the break's line segment to the node's Lines(), which the stock parser
// leaves empty, so the transformer can derive a real Span for
// document.ThematicBreak instead of the zero value.
//
// It is registered at priority 150 — after the setext-heading parser (100),
// before the stock thematic-break parser (200) — so parse precedence is
// byte-for-byte the stock ordering: setext headings still win over "---"
// under a paragraph, the front-matter parser (priority 0) still claims a
// line-1 fence first, and the stock parser simply never fires because this
// one matches the identical predicate first. The CommonMark suite
// (652/652, rendered-HTML equality) pins that equivalence.
type thematicBreakParser struct{}

var _ gparser.BlockParser = thematicBreakParser{}

func (thematicBreakParser) Trigger() []byte {
	return []byte{'-', '*', '_'}
}

func (thematicBreakParser) Open(parent ast.Node, reader text.Reader, pc gparser.Context) (ast.Node, gparser.State) {
	line, segment := reader.PeekLine()
	if isThematicBreakLine(line, reader.LineOffset()) {
		reader.AdvanceToEOL()
		node := ast.NewThematicBreak()
		node.Lines().Append(segment)
		return node, gparser.NoChildren
	}
	return nil, gparser.NoChildren
}

func (thematicBreakParser) Continue(node ast.Node, reader text.Reader, pc gparser.Context) gparser.State {
	return gparser.Close
}

func (thematicBreakParser) Close(node ast.Node, reader text.Reader, pc gparser.Context) {}

func (thematicBreakParser) CanInterruptParagraph() bool { return true }

func (thematicBreakParser) CanAcceptIndentedLine() bool { return false }

// isThematicBreakLine is goldmark's unexported isThematicBreak predicate
// (goldmark/parser/thematic_break.go), copied verbatim: at most 3 columns
// of indent, then 3+ repetitions of one of '-', '*', '_' with only spaces
// between, and nothing else on the line.
func isThematicBreakLine(line []byte, offset int) bool {
	w, pos := util.IndentWidth(line, offset)
	if w > 3 {
		return false
	}
	mark := byte(0)
	count := 0
	for i := pos; i < len(line); i++ {
		c := line[i]
		if util.IsSpace(c) {
			continue
		}
		if mark == 0 {
			mark = c
			count = 1
			if mark == '*' || mark == '-' || mark == '_' {
				continue
			}
			return false
		}
		if c != mark {
			return false
		}
		count++
	}
	return count > 2
}

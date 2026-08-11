package parser

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	gparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

var mathKind = ast.NewNodeKind("Math")

type mathNode struct {
	ast.BaseInline
	Display bool
	Source  []byte
	// Segment is the full delimiter-inclusive source range ("$x$" /
	// "$$x$$"), recorded at parse time for the node's Span.
	Segment text.Segment
}

func (n *mathNode) Kind() ast.NodeKind { return mathKind }
func (n *mathNode) Dump(src []byte, level int) {
	ast.DumpHelper(n, src, level, nil, nil)
}

type mathExt struct{}

func (mathExt) Extend(md goldmark.Markdown) {
	md.Parser().AddOptions(gparser.WithInlineParsers(
		util.Prioritized(mathParser{}, 150),
	))
}

type mathParser struct{}

func (mathParser) Trigger() []byte { return []byte{'$'} }

// Parse recognizes $x$ and $$x$$ on a single line. Multi-line $$ blocks are
// promoted from paragraphs in the transformer. Rules: no space just inside
// either delimiter; content non-empty.
func (mathParser) Parse(parent ast.Node, block text.Reader, pc gparser.Context) ast.Node {
	line, seg := block.PeekLine()
	delim := 1
	if len(line) >= 2 && line[1] == '$' {
		delim = 2
	}
	rest := line[delim:]
	if len(rest) == 0 || rest[0] == ' ' || rest[0] == '\t' {
		return nil
	}
	end := -1
	for i := 1; i+delim <= len(rest); i++ {
		if rest[i] != '$' {
			continue
		}
		if delim == 2 && (i+1 >= len(rest) || rest[i+1] != '$') {
			continue
		}
		if rest[i-1] == ' ' || rest[i-1] == '\t' {
			continue
		}
		end = i
		break
	}
	if end <= 0 {
		return nil
	}
	src := make([]byte, end)
	copy(src, rest[:end])
	total := delim + end + delim
	block.Advance(total)
	return &mathNode{
		Display: delim == 2, Source: src,
		Segment: text.NewSegment(seg.Start, seg.Start+total),
	}
}

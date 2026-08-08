package document

// WalkStatus is a Visitor's instruction to Walk about how to proceed.
type WalkStatus int

const (
	Continue     WalkStatus = iota // proceed normally
	SkipChildren                   // (enter only) skip the current node's subtree
	Stop                           // abort the walk entirely
)

// Visitor is called once on entering and once on leaving each node visited
// by Walk; entering distinguishes the two calls.
type Visitor func(n Node, entering bool) WalkStatus

// Walk visits n and its descendants depth-first, calling visit on enter and
// exit. SkipChildren (on enter) skips the subtree; Stop aborts the walk.
func Walk(n Node, visit Visitor) {
	walk(n, visit)
}

func walk(n Node, visit Visitor) WalkStatus {
	switch visit(n, true) {
	case Stop:
		return Stop
	case SkipChildren:
		return visit(n, false)
	}
	for _, c := range n.Children() {
		if walk(c, visit) == Stop {
			return Stop
		}
	}
	return visit(n, false)
}

// PlainText returns the concatenated Text and CodeSpan content of a subtree.
func PlainText(n Node) string {
	var out []byte
	Walk(n, func(n Node, entering bool) WalkStatus {
		if !entering {
			return Continue
		}
		switch n := n.(type) {
		case *Text:
			out = append(out, n.Value...)
		case *CodeSpan:
			out = append(out, n.Value...)
		}
		return Continue
	})
	return string(out)
}

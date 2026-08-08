package markdownviewer_test

import (
	"fmt"

	markdownviewer "github.com/sriannamalai/markdownviewer"
	"github.com/sriannamalai/markdownviewer/document"
)

// Render parses and renders Markdown to HTML. Fragment() is used here to
// keep the example's output small and deterministic; without it Render
// produces a full HTML page (doctype, head, embedded theme CSS) which is
// too large to pin down with an // Output: comment.
func ExampleRender() {
	out, err := markdownviewer.Render([]byte("# Hello\n\nWorld.\n"), markdownviewer.Fragment())
	if err != nil {
		panic(err)
	}
	fmt.Print(string(out))
	// Output:
	// <h1 id="hello">Hello</h1>
	// <p>World.</p>
}

// Fragment emits body-only HTML: no <html>/<head>/<style>, just the
// rendered markup. Hosts embedding output into an existing page use this
// instead of the default full-page output.
func ExampleFragment() {
	out, err := markdownviewer.Render([]byte("**bold**\n"), markdownviewer.Fragment())
	if err != nil {
		panic(err)
	}
	fmt.Print(string(out))
	// Output:
	// <p><strong>bold</strong></p>
}

// WithResolver installs a callback that rewrites link/image/wiki-link
// targets — here, routing an image through a CDN. Returning ok=false falls
// back to default resolution for anything the resolver doesn't handle.
func ExampleWithResolver() {
	resolver := func(kind markdownviewer.ResolveKind, target string) (string, bool) {
		if kind == markdownviewer.ResolveImage {
			return "https://cdn.example.com/" + target, true
		}
		return "", false
	}
	out, err := markdownviewer.Render([]byte("![a cat](cat.png)\n"),
		markdownviewer.Fragment(), markdownviewer.WithResolver(resolver))
	if err != nil {
		panic(err)
	}
	fmt.Print(string(out))
	// Output:
	// <p><img src="https://cdn.example.com/cat.png" alt="a cat" /></p>
}

// Parse returns the document model directly, for callers that want to walk
// or otherwise process the tree themselves rather than render it to HTML.
func ExampleParse() {
	src := "# One\n\nSome text.\n\n## Two\n\nMore text.\n\n### Three\n"
	doc, err := markdownviewer.Parse([]byte(src))
	if err != nil {
		panic(err)
	}
	headings := 0
	document.Walk(doc, func(n document.Node, entering bool) document.WalkStatus {
		if entering {
			if _, ok := n.(*document.Heading); ok {
				headings++
			}
		}
		return document.Continue
	})
	fmt.Println(headings)
	// Output:
	// 3
}

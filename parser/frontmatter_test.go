package parser

import (
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/frontmatter"

	"github.com/sriannamalai/markdownviewer/document"
)

// TestUnterminatedFrontMatterFenceRecovered covers the silent-data-loss bug:
// with the default Config (FrontMatter: true), a document whose first line
// is "---" and that never has a closing "---" is claimed entirely by
// go.abhg.dev/goldmark/frontmatter's block parser as an unterminated
// front-matter block, discarding the whole document. ParseWith detects the
// swallow (no Meta extracted, zero children, non-blank source) and
// reparses with FrontMatter disabled so "---" falls back to its correct
// CommonMark meaning: a thematic break.
func TestUnterminatedFrontMatterFenceRecovered(t *testing.T) {
	assertDoc(t, "---\n", `
Document
  ThematicBreak
`)
}

func TestUnterminatedFrontMatterFenceWithBodyRecovered(t *testing.T) {
	assertDoc(t, "---\nbody after\n", `
Document
  ThematicBreak
  Paragraph
    Text "body after"
`)
}

// TestTerminatedFrontMatterStillWorks guards against a regression where the
// unterminated-fence fallback misfires on properly closed front matter.
func TestTerminatedFrontMatterStillWorks(t *testing.T) {
	doc, err := Parse([]byte("---\ntitle: Hi\n---\n\nbody\n"))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Meta["title"] != "Hi" {
		t.Fatalf("meta: %#v", doc.Meta)
	}
	if len(doc.Children()) != 1 {
		t.Fatalf("want 1 child (the body paragraph), got %d", len(doc.Children()))
	}
}

// TestFrontMatterOnlyDocumentStaysEmpty covers the meta-bearing-but-
// content-free case: terminated front matter with no body must NOT trigger
// the unterminated-fence fallback, even though the resulting document has
// zero children.
func TestFrontMatterOnlyDocumentStaysEmpty(t *testing.T) {
	doc, err := Parse([]byte("---\ntitle: Hi\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Meta["title"] != "Hi" {
		t.Fatalf("meta: %#v", doc.Meta)
	}
	if len(doc.Children()) != 0 {
		t.Fatalf("want 0 children, got %d", len(doc.Children()))
	}
}

// TestTerminatedEmptyBodyFrontMatterStaysEmpty covers the regression found
// in round-1 review: "---\n---\n" is a genuinely closed, empty-body
// front-matter block. frontmatter's Close() runs unconditionally (even at
// EOF for a truly unterminated block), so frontmatter.Get(ctx) is non-nil
// and doc.Meta decodes to nil either way — doc.Meta == nil alone cannot
// distinguish "closed, nothing between the fences" from "never closed".
// The fallback must key off whether a matching closing line actually
// exists in the source (frontMatterFenceTerminated), not off doc.Meta.
func TestTerminatedEmptyBodyFrontMatterStaysEmpty(t *testing.T) {
	doc, err := Parse([]byte("---\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Meta) != 0 {
		t.Fatalf("meta: %#v", doc.Meta)
	}
	if len(doc.Children()) != 0 {
		t.Fatalf("want 0 children (no fallback misfire), got %d: %s", len(doc.Children()), document.Dump(doc))
	}
}

// TestTerminatedWhitespaceBodyFrontMatterStaysEmpty is the same case with a
// whitespace-only interior line, which also decodes to a nil/empty Meta.
func TestTerminatedWhitespaceBodyFrontMatterStaysEmpty(t *testing.T) {
	doc, err := Parse([]byte("---\n \n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Meta) != 0 {
		t.Fatalf("meta: %#v", doc.Meta)
	}
	if len(doc.Children()) != 0 {
		t.Fatalf("want 0 children (no fallback misfire), got %d: %s", len(doc.Children()), document.Dump(doc))
	}
}

// TestBlankSourceStaysEmpty covers the other guard: an all-blank source
// must not be treated as a swallowed document and must not error.
func TestBlankSourceStaysEmpty(t *testing.T) {
	doc, err := Parse([]byte("\n\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Children()) != 0 {
		t.Fatalf("want 0 children, got %d", len(doc.Children()))
	}
}

// TestCommonMarkOnlyUnaffectedByFallback verifies the fallback logic is
// gated on cfg.FrontMatter: with CommonMarkOnly() there is no frontmatter
// extension in play at all, so "---" is a thematic break on the first
// (unmodified) parse, and the ParseWith code path guarding the reparse
// must not be exercised.
func TestCommonMarkOnlyUnaffectedByFallback(t *testing.T) {
	doc, err := ParseWith([]byte("---\n"), CommonMarkOnly())
	if err != nil {
		t.Fatal(err)
	}
	kids := doc.Children()
	if len(kids) != 1 {
		t.Fatalf("want 1 child, got %d", len(kids))
	}
	if _, ok := kids[0].(*document.ThematicBreak); !ok {
		t.Fatalf("want ThematicBreak, got %T", kids[0])
	}
}

// TestUpstreamFrontmatterStillSwallowsUnterminated is an upstream canary,
// not a test of our own code: it builds a goldmark instance directly with
// go.abhg.dev/goldmark/frontmatter's &frontmatter.Extender{} (bypassing our
// ParseWith entirely) and parses an unterminated fence via goldmark's own
// Parser().Parse API. It asserts upstream still swallows the whole
// document into the unterminated front-matter block (zero children on the
// resulting AST document) — the exact behavior frontMatterFenceTerminated
// and its fallback trigger in parser.go exist to work around.
//
// If this test fails after a dependency bump, upstream changed its
// swallow-on-unterminated behavior: re-verify frontMatterFenceTerminated
// AND the fallback trigger conditions in parser.go against the new
// parse.go before assuming anything else is safe. Today's other tests
// only exercise our mirror of that grammar, not upstream's actual
// behavior, so they would not catch this on their own.
func TestUpstreamFrontmatterStillSwallowsUnterminated(t *testing.T) {
	md := goldmark.New(
		goldmark.WithExtensions(
			&frontmatter.Extender{},
		),
	)
	doc := md.Parser().Parse(text.NewReader([]byte("---\nbody\n")))
	if n := doc.ChildCount(); n != 0 {
		t.Fatalf("upstream frontmatter no longer swallows an unterminated "+
			"fence: got %d children, want 0 (re-verify frontMatterFenceTerminated "+
			"and the ParseWith fallback trigger against the new parse.go)", n)
	}
}

func TestFrontMatterFenceTerminated(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want bool
	}{
		{"unterminated bare", "---\n", false},
		{"unterminated with body", "---\nbody after\n", false},
		{"terminated empty body", "---\n---\n", true},
		{"terminated whitespace body", "---\n \n---\n", true},
		{"terminated with data", "---\ntitle: Hi\n---\n\nbody\n", true},
		{"toml delimiter", "+++\ntitle = 1\n+++\n", true},
		{"toml unterminated", "+++\n", false},
		{"mismatched count", "---\n----\n", false},
		{"not an opener", "hi\n---\n", true},
		{"no trailing newline", "---", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := frontMatterFenceTerminated([]byte(c.src))
			if got != c.want {
				t.Fatalf("frontMatterFenceTerminated(%q) = %v, want %v", c.src, got, c.want)
			}
		})
	}
}

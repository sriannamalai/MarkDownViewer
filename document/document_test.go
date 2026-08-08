package document

import (
	"strings"
	"testing"
)

func sample() *Document {
	doc := &Document{}
	h := &Heading{Level: 2, AnchorID: "hello"}
	h.AppendChild(&Text{Value: "Hello"})
	p := &Paragraph{}
	em := &Emphasis{}
	em.AppendChild(&Text{Value: "world"})
	p.AppendChild(em)
	doc.AppendChild(h)
	doc.AppendChild(p)
	return doc
}

func TestWalkOrder(t *testing.T) {
	var got []string
	Walk(sample(), func(n Node, entering bool) WalkStatus {
		if entering {
			got = append(got, label(n))
		}
		return Continue
	})
	want := []string{`Document`, `Heading[2] id="hello"`, `Text "Hello"`, `Paragraph`, `Emphasis`, `Text "world"`}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("walk order:\n got %v\nwant %v", got, want)
	}
}

func TestWalkSkipChildren(t *testing.T) {
	var got []string
	Walk(sample(), func(n Node, entering bool) WalkStatus {
		if entering {
			got = append(got, label(n))
			if n.Kind() == KindHeading {
				return SkipChildren
			}
		}
		return Continue
	})
	for _, l := range got {
		if l == `Text "Hello"` {
			t.Fatal("SkipChildren did not skip heading text")
		}
	}
}

func TestDump(t *testing.T) {
	want := "Document\n  Heading[2] id=\"hello\"\n    Text \"Hello\"\n  Paragraph\n    Emphasis\n      Text \"world\"\n"
	if got := Dump(sample()); got != want {
		t.Fatalf("dump:\n got %q\nwant %q", got, want)
	}
}

func TestPlainText(t *testing.T) {
	if got := PlainText(sample()); got != "Helloworld" {
		t.Fatalf("plaintext: %q", got)
	}
}

package htmlrender

import (
	"bytes"
	"strings"
	"testing"

	"github.com/sriannamalai/markdownviewer/parser"
)

func render(t *testing.T, md string, mutate func(*Options)) string {
	t.Helper()
	doc, err := parser.Parse([]byte(md))
	if err != nil {
		t.Fatal(err)
	}
	opts := DefaultOptions()
	opts.Fragment = true
	opts.Highlight = false
	if mutate != nil {
		mutate(&opts)
	}
	var buf bytes.Buffer
	if err := Render(&buf, doc, opts); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestCoreBlocks(t *testing.T) {
	got := render(t, "# Title\n\npara *em* **st**\n\n- a\n- b\n", nil)
	want := "<h1 id=\"title\">Title</h1>\n<p>para <em>em</em> <strong>st</strong></p>\n<ul>\n<li>a</li>\n<li>b</li>\n</ul>\n"
	if got != want {
		t.Fatalf("\n got %q\nwant %q", got, want)
	}
}

func TestAnchorsOff(t *testing.T) {
	got := render(t, "# Title\n", func(o *Options) { o.HeadingAnchors = false })
	if got != "<h1>Title</h1>\n" {
		t.Fatalf("got %q", got)
	}
}

func TestEscaping(t *testing.T) {
	got := render(t, "a < b & \"c\"\n", nil)
	if !strings.Contains(got, "a &lt; b &amp; &quot;c&quot;") {
		t.Fatalf("got %q", got)
	}
}

func TestUnsafeURLsStripped(t *testing.T) {
	got := render(t, "[x](javascript:alert(1)) ![y](data:text/html;base64,xx)\n", nil)
	if strings.Contains(got, "javascript:") || strings.Contains(got, "data:") {
		t.Fatalf("unsafe URL survived: %q", got)
	}
}

func TestUnsafeURLsKeptWhenUnsafe(t *testing.T) {
	got := render(t, "[x](javascript:alert(1))\n", func(o *Options) { o.Unsafe = true })
	if !strings.Contains(got, `href="javascript:alert(1)"`) {
		t.Fatalf("got %q", got)
	}
}

func TestWikiLinkDefaultResolution(t *testing.T) {
	got := render(t, "[[Some Page]]\n", nil)
	if !strings.Contains(got, `<a href="Some Page.md" class="wikilink">Some Page</a>`) {
		t.Fatalf("got %q", got)
	}
}

func TestUnsafeWikiLinkStripped(t *testing.T) {
	got := render(t, "[[javascript:alert(1)]]\n", nil)
	if strings.Contains(got, "href=") {
		t.Fatalf("blocked wikilink should have no href attribute: %q", got)
	}
	if !strings.Contains(got, `<a class="wikilink">javascript:alert(1)</a>`) {
		t.Fatalf("blocked wikilink should render without href but keep class: %q", got)
	}
}

func TestUnsafeWikiLinkKeptWhenUnsafe(t *testing.T) {
	got := render(t, "[[javascript:alert(1)]]\n", func(o *Options) { o.Unsafe = true })
	if !strings.Contains(got, `href="javascript:alert(1).md"`) {
		t.Fatalf("got %q", got)
	}
}

func TestResolverCallback(t *testing.T) {
	// Uses an allowlisted https scheme: safeURL's scheme allowlist (Task
	// 11 hardening) applies uniformly to resolver output too, so a custom
	// scheme like "vfs://" would now be blocked.
	got := render(t, "![i](pic.png) [[P]]\n", func(o *Options) {
		o.Resolver = func(kind ResolveKind, target string) (string, bool) {
			return "https://vfs.example/" + target, true
		}
	})
	if !strings.Contains(got, `src="https://vfs.example/pic.png"`) || !strings.Contains(got, `href="https://vfs.example/P"`) {
		t.Fatalf("got %q", got)
	}
}

func TestTaskListCheckboxes(t *testing.T) {
	got := render(t, "- [x] done\n", nil)
	if !strings.Contains(got, `<input type="checkbox" checked disabled>`) {
		t.Fatalf("got %q", got)
	}
}

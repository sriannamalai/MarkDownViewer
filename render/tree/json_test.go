package tree_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sriannamalai/markdownviewer/document"
	"github.com/sriannamalai/markdownviewer/render/tree"
)

func marshal(t *testing.T, md string) string {
	t.Helper()
	out, err := json.Marshal(buildDefault(t, md))
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

func TestEnvelopeStrictness(t *testing.T) {
	// Empty document: version pinned, blocks/footnotes are arrays, never
	// null.
	tr, err := tree.Build(&document.Document{}, tree.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(tr)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(out); got != `{"version":1,"blocks":[],"footnotes":[]}` {
		t.Fatalf("empty envelope: %s", got)
	}
}

func TestWireFieldPresence(t *testing.T) {
	// runs is null (not omitted) while token runs are stubbed; text is
	// always present.
	code := marshal(t, "```go\npackage x\n```\n")
	if !strings.Contains(code, `"runs":null`) {
		t.Fatalf("codeBlock runs not null: %s", code)
	}
	if !strings.Contains(code, `"label":"go"`) || !strings.Contains(code, `"text":"package x\n"`) {
		t.Fatalf("codeBlock fields: %s", code)
	}

	// task tri-state: null | false | true, always present.
	list := marshal(t, "- plain\n- [ ] todo\n- [x] done\n")
	for _, want := range []string{`"task":null`, `"task":false`, `"task":true`} {
		if !strings.Contains(list, want) {
			t.Fatalf("missing %s in %s", want, list)
		}
	}

	// link url is present even when empty; blocked appears only when
	// true.
	links := marshal(t, "[empty]() and [bad](javascript:x) and [ok](https://a)\n")
	if strings.Count(links, `"url":""`) != 2 {
		t.Fatalf("empty urls: %s", links)
	}
	if strings.Count(links, `"blocked":true`) != 1 || strings.Contains(links, `"blocked":false`) {
		t.Fatalf("blocked marks: %s", links)
	}

	// engines pinned.
	if s := marshal(t, "$$\nx\n$$\n"); !strings.Contains(s, `"engine":"katex"`) {
		t.Fatalf("mathBlock engine: %s", s)
	}
	if s := marshal(t, "```mermaid\ng\n```\n"); !strings.Contains(s, `"engine":"mermaid"`) {
		t.Fatalf("diagram engine: %s", s)
	}

	// htmlBlock: html + unsafe always present.
	if s := marshal(t, "<div><em>x</em></div>\n"); !strings.Contains(s, `"unsafe":false`) {
		t.Fatalf("htmlBlock unsafe flag: %s", s)
	}

	// wiki links surface as link nodes tagged source:"wikiLink";
	// ordinary links omit source.
	wiki := marshal(t, "[[Page]] and [plain](https://a)\n")
	if strings.Count(wiki, `"source":"wikiLink"`) != 1 {
		t.Fatalf("wiki source tag: %s", wiki)
	}

	// footnotes envelope: index + count pairing.
	fn := marshal(t, "x[^a] y[^a]\n\n[^a]: Body.\n")
	if !strings.Contains(fn, `"index":1`) || !strings.Contains(fn, `"count":2`) {
		t.Fatalf("footnote pairing: %s", fn)
	}

	// table header/rows always arrays; alignments as names.
	tbl := marshal(t, "| a |\n|:-:|\n")
	if !strings.Contains(tbl, `"alignments":["center"]`) || !strings.Contains(tbl, `"rows":[]`) {
		t.Fatalf("table wire: %s", tbl)
	}
}

func TestWireKindNamesMatchDocumentCodec(t *testing.T) {
	// One naming universe: every kind string the tree emits must be a
	// document codec wire name.
	out := marshal(t, "# h\n\npara *em* **st** ~~del~~ `c` [l](https://a) ![i](b) [[w]]\n\n> q\n\n- x\n\n| a |\n|---|\n\n---\n\n$$m$$\n\nfoo[^1]\n\n[^1]: fn\n")
	var env map[string]any
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatal(err)
	}
	var check func(v any)
	check = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			if k, ok := v["kind"].(string); ok {
				if _, known := document.KindFromString(k); !known {
					t.Errorf("kind %q is not a document codec name", k)
				}
			}
			for _, c := range v {
				check(c)
			}
		case []any:
			for _, c := range v {
				check(c)
			}
		}
	}
	check(env)
}

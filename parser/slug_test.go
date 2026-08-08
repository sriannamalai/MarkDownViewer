package parser

import "testing"

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Hello World":        "hello-world",
		"What's New in 2.0?": "whats-new-in-20",
		"  spaces  ":         "spaces",
		"émoji ✨ ok":         "émoji--ok",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSlugDedupe(t *testing.T) {
	tr := &transformer{slugs: map[string]int{}}
	if got := tr.slug("Intro"); got != "intro" {
		t.Fatalf("first: %q", got)
	}
	if got := tr.slug("Intro"); got != "intro-1" {
		t.Fatalf("second: %q", got)
	}
	if got := tr.slug("Intro"); got != "intro-2" {
		t.Fatalf("third: %q", got)
	}
}

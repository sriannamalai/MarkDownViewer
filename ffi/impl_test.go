package main

import (
	"bytes"
	"strings"
	"testing"
)

const sampleMD = "# Hello *world*\n\nSome `code` here.\n"

func TestRenderImpl(t *testing.T) {
	html, err := renderImpl([]byte(sampleMD), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<h1", "Hello", "<em>world</em>", "<code>code</code>"} {
		if !bytes.Contains(html, []byte(want)) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestRenderImplFragment(t *testing.T) {
	html, err := renderImpl([]byte(sampleMD), []byte(`{"fragment": true}`))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(html, []byte("<html")) {
		t.Error("fragment output contains <html")
	}
}

func TestRenderImplBadOptions(t *testing.T) {
	if _, err := renderImpl([]byte(sampleMD), []byte(`{"bogus": 1}`)); err == nil ||
		!strings.Contains(err.Error(), "bogus") {
		t.Fatalf("want unknown-field error, got %v", err)
	}
}

func TestParseImpl(t *testing.T) {
	doc, err := parseImpl([]byte(sampleMD), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"version":1`, `"heading"`} {
		if !bytes.Contains(doc, []byte(want)) {
			t.Errorf("document JSON missing %q in %s", want, doc)
		}
	}
}

func TestParseImplValidatesOptions(t *testing.T) {
	if _, err := parseImpl([]byte(sampleMD), []byte(`{"version": 2}`)); err == nil {
		t.Fatal("want version error, got nil")
	}
}

func TestRenderDocRoundTrip(t *testing.T) {
	opts := []byte(`{"theme": "dark"}`)
	direct, err := renderImpl([]byte(sampleMD), opts)
	if err != nil {
		t.Fatal(err)
	}
	docJSON, err := parseImpl([]byte(sampleMD), nil)
	if err != nil {
		t.Fatal(err)
	}
	viaDoc, err := renderDocImpl(docJSON, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(direct, viaDoc) {
		t.Error("render and parse->renderDoc outputs differ")
	}
}

func TestRenderDocImplBadJSON(t *testing.T) {
	if _, err := renderDocImpl([]byte(`{`), nil); err == nil {
		t.Fatal("want document JSON error, got nil")
	}
}

func TestEmptyInputs(t *testing.T) {
	if _, err := renderImpl(nil, nil); err != nil {
		t.Errorf("renderImpl(nil): %v", err)
	}
	if _, err := parseImpl(nil, nil); err != nil {
		t.Errorf("parseImpl(nil): %v", err)
	}
}

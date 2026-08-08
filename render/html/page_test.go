package htmlrender

import (
	"strings"
	"testing"
)

func TestFullPage(t *testing.T) {
	got := render(t, "# Hi\n", func(o *Options) { o.Fragment = false; o.ThemeName = "auto" })
	for _, want := range []string{
		"<!doctype html>", "<meta charset=\"utf-8\">", "<style>",
		"markdown-body", "--md-bg:", "@media (prefers-color-scheme: dark)",
		"<h1 id=\"hi\">Hi</h1>",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("page missing %q", want)
		}
	}
	if strings.Contains(got, "http://") || strings.Contains(got, "https://cdn") {
		t.Error("page must not reference the network")
	}
}

func TestFullPageLightHasNoDarkBlock(t *testing.T) {
	got := render(t, "# Hi\n", func(o *Options) { o.Fragment = false; o.ThemeName = "light" })
	if strings.Contains(got, "prefers-color-scheme") {
		t.Error("explicit light theme should not embed dark override")
	}
}

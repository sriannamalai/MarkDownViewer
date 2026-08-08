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

func TestThemeOverridesInLightMode(t *testing.T) {
	got := render(t, "# Hi\n", func(o *Options) {
		o.Fragment = false
		o.ThemeName = "light"
		o.ThemeOverrides = map[string]string{"--md-bg": "#f0f0f0"}
	})
	if !strings.Contains(got, ":root{--md-bg:#f0f0f0;") {
		t.Error("light mode should emit theme override block")
	}
}

func TestThemeOverridesAppearsAfterThemeAndInDarkBlock(t *testing.T) {
	got := render(t, "# Hi\n", func(o *Options) {
		o.Fragment = false
		o.ThemeName = "auto"
		o.ThemeOverrides = map[string]string{"--md-fg": "#000000"}
	})
	// Check that override appears after light theme :root block
	rootIdx := strings.Index(got, ":root{--md-")
	if rootIdx == -1 {
		t.Fatalf("light theme :root block not found")
	}
	overrideIdx := strings.Index(got, ":root{--md-fg:#000000;}")
	if overrideIdx == -1 || overrideIdx <= rootIdx {
		t.Error("override should appear after light theme block")
	}
	// Check that override appears inside dark media query
	darkMediaIdx := strings.Index(got, "@media (prefers-color-scheme: dark){")
	darkMediaEndIdx := strings.LastIndex(got[darkMediaIdx:], "}")
	if darkMediaIdx == -1 || darkMediaEndIdx == -1 {
		t.Fatalf("dark media block not found")
	}
	darkMediaContent := got[darkMediaIdx : darkMediaIdx+darkMediaEndIdx]
	if !strings.Contains(darkMediaContent, ":root{--md-fg:#000000;}") {
		t.Error("override should appear inside dark media block")
	}
}

func TestCustomStylesheetReplacesBaseCSS(t *testing.T) {
	customCSS := ".custom-class { color: red; }"
	got := render(t, "# Hi\n", func(o *Options) {
		o.Fragment = false
		o.ThemeName = "light"
		o.Stylesheet = customCSS
	})
	if !strings.Contains(got, customCSS) {
		t.Error("custom stylesheet should be present")
	}
	if strings.Contains(got, "max-width: 860px") {
		t.Error("base.css should not be present when custom stylesheet is used")
	}
}

func TestFragmentModeIgnoresCSS(t *testing.T) {
	got := render(t, "# Hi\n", func(o *Options) {
		o.Fragment = true
		o.ThemeOverrides = map[string]string{"--md-bg": "#f0f0f0"}
	})
	if strings.Contains(got, "<style>") {
		t.Error("fragment mode should not include style tags")
	}
	if strings.Contains(got, ":root{") {
		t.Error("fragment mode should not include theme overrides")
	}
}

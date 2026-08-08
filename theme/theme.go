// Package theme provides the built-in visual themes as CSS custom-property
// sets over one base stylesheet.
package theme

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed base.css
var baseCSS string

// BaseCSS returns the theme-agnostic base stylesheet shared by all themes;
// a Theme layers its CSS custom properties on top of it.
func BaseCSS() string { return baseCSS }

// Theme is a named set of CSS custom-property values, paired with a chroma
// syntax-highlight style, that together give a rendered page its visual
// appearance over BaseCSS.
type Theme struct {
	Name        string            // theme name, e.g. "light" or "dark"
	Vars        map[string]string // CSS custom-property values, keyed by property name (e.g. "--md-bg")
	ChromaStyle string            // paired chroma syntax-highlight style
}

// Light returns the built-in light theme.
func Light() Theme {
	return Theme{Name: "light", ChromaStyle: "github", Vars: map[string]string{
		"--md-bg": "#ffffff", "--md-fg": "#1f2328", "--md-accent": "#0969da",
		"--md-code-bg": "#f6f8fa", "--md-border": "#d1d9e0", "--md-quote-fg": "#59636e",
	}}
}

// Dark returns the built-in dark theme.
func Dark() Theme {
	return Theme{Name: "dark", ChromaStyle: "github-dark", Vars: map[string]string{
		"--md-bg": "#0d1117", "--md-fg": "#e6edf3", "--md-accent": "#4493f8",
		"--md-code-bg": "#161b22", "--md-border": "#30363d", "--md-quote-fg": "#8d96a0",
	}}
}

// Get resolves a theme name. "auto" returns Light; page assembly layers the
// dark override in a prefers-color-scheme media query.
func Get(name string) (Theme, error) {
	switch name {
	case "light", "auto", "":
		return Light(), nil
	case "dark":
		return Dark(), nil
	}
	return Theme{}, fmt.Errorf("theme: unknown theme %q", name)
}

// CSS emits the variable declarations under the given selector.
func (t Theme) CSS(selector string) string {
	keys := make([]string, 0, len(t.Vars))
	for k := range t.Vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(selector + "{")
	for _, k := range keys {
		b.WriteString(k + ":" + t.Vars[k] + ";")
	}
	b.WriteString("}")
	return b.String()
}

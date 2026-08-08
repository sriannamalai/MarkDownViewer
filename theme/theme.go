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

func BaseCSS() string { return baseCSS }

type Theme struct {
	Name        string
	Vars        map[string]string
	ChromaStyle string // paired chroma syntax-highlight style
}

func Light() Theme {
	return Theme{Name: "light", ChromaStyle: "github", Vars: map[string]string{
		"--md-bg": "#ffffff", "--md-fg": "#1f2328", "--md-accent": "#0969da",
		"--md-code-bg": "#f6f8fa", "--md-border": "#d1d9e0", "--md-quote-fg": "#59636e",
	}}
}

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

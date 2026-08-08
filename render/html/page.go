package htmlrender

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/sriannamalai/markdownviewer/assets"
	"github.com/sriannamalai/markdownviewer/document"
	"github.com/sriannamalai/markdownviewer/theme"
)

// errWriter wraps an io.Writer and tracks the first write error,
// no-opping subsequent writes after an error occurs.
type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) write(s string) {
	if ew.err != nil {
		return
	}
	_, ew.err = io.WriteString(ew.w, s)
}

var validCustomPropertyRe = regexp.MustCompile(`^--[a-zA-Z0-9_-]+$`)

// usesFeatures reports whether doc contains any diagram or math nodes, so
// the page only pays for mermaid/KaTeX assets when it actually needs them.
func usesFeatures(doc *document.Document) (mermaid, math bool) {
	document.Walk(doc, func(n document.Node, entering bool) document.WalkStatus {
		if !entering {
			return document.Continue
		}
		switch n.(type) {
		case *document.Diagram:
			mermaid = true
		case *document.MathBlock, *document.MathInline:
			math = true
		}
		return document.Continue
	})
	return mermaid, math
}

// sanitizeCSS removes </style sequences case-insensitively from host-supplied CSS
// to prevent breakout from the page's style element.
func sanitizeCSS(s string) string {
	lower := strings.ToLower(s)
	for strings.Contains(lower, "</style") {
		i := strings.Index(lower, "</style")
		s = s[:i] + s[i+7:]
		lower = strings.ToLower(s)
	}
	return s
}

// emitThemeOverrides emits CSS custom-property overrides in sorted key order.
// Keys must match the pattern --[a-zA-Z0-9_-]+; non-conforming keys are silently dropped.
func emitThemeOverrides(overrides map[string]string) string {
	if len(overrides) == 0 {
		return ""
	}
	keys := make([]string, 0, len(overrides))
	for k := range overrides {
		if validCustomPropertyRe.MatchString(k) {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return ""
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString(":root{")
	for _, k := range keys {
		sanitized := sanitizeCSS(overrides[k])
		b.WriteString(k + ":" + sanitized + ";")
	}
	b.WriteString("}")
	return b.String()
}

func renderPage(w io.Writer, doc *document.Document, opts Options) error {
	th, err := theme.Get(opts.ThemeName)
	if err != nil {
		return err
	}
	lightChroma, err := chromaCSS(theme.Light().ChromaStyle)
	if err != nil {
		return err
	}
	mermaidUsed, mathUsed := usesFeatures(doc)

	ew := &errWriter{w: w}
	ew.write("<!doctype html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n")
	ew.write("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n<style>\n")
	if opts.ThemeName == "dark" {
		darkChroma, err := chromaCSS(theme.Dark().ChromaStyle)
		if err != nil {
			return err
		}
		ew.write(theme.Dark().CSS(":root") + "\n" + darkChroma)
		if overrides := emitThemeOverrides(opts.ThemeOverrides); overrides != "" {
			ew.write("\n" + overrides)
		}
	} else {
		ew.write(th.CSS(":root") + "\n" + lightChroma)
		if overrides := emitThemeOverrides(opts.ThemeOverrides); overrides != "" {
			ew.write("\n" + overrides)
		}
	}
	if opts.ThemeName == "auto" || opts.ThemeName == "" {
		darkChroma, err := chromaCSS(theme.Dark().ChromaStyle)
		if err != nil {
			return err
		}
		// The light chroma CSS above is unconditional; any token class it
		// styles but the dark style leaves unstyled needs an explicit
		// "inherit" rule here so it doesn't survive, unreadable, into a
		// dark-preferring viewer. See neutralizeMissingClasses.
		darkChroma += neutralizeMissingClasses(darkChroma, lightChroma)
		ew.write("\n@media (prefers-color-scheme: dark){\n" + theme.Dark().CSS(":root") + "\n" + darkChroma)
		if overrides := emitThemeOverrides(opts.ThemeOverrides); overrides != "" {
			ew.write("\n" + overrides)
		}
		ew.write("}\n")
	}
	if opts.Stylesheet != "" {
		ew.write(sanitizeCSS(opts.Stylesheet))
	} else {
		ew.write(theme.BaseCSS())
	}
	if mathUsed && opts.Math {
		ew.write(assets.KatexCSS())
	}
	ew.write("</style>\n</head>\n<body class=\"markdown-body\">\n")
	if ew.err != nil {
		return ew.err
	}
	if err := renderFragment(w, doc, opts); err != nil {
		return err
	}
	if mermaidUsed && opts.Mermaid {
		mtheme := "default"
		if opts.ThemeName == "dark" {
			mtheme = "dark"
		}
		ew.write("<script>" + assets.MermaidJS() + "</script>\n")
		if ew.err != nil {
			return ew.err
		}
		_, ew.err = fmt.Fprintf(w, "<script>mermaid.initialize({startOnLoad:true,theme:%q});</script>\n", mtheme)
		if ew.err != nil {
			return ew.err
		}
	}
	if mathUsed && opts.Math {
		ew.write("<script>" + assets.KatexJS() + "</script>\n")
		ew.write("<script>document.querySelectorAll('.math').forEach(function(el){" +
			"katex.render(el.textContent,el,{displayMode:el.classList.contains('math-display'),throwOnError:false});});</script>\n")
	}
	ew.write("</body>\n</html>\n")
	return ew.err
}

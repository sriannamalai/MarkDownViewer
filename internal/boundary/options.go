package boundary

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	markdownviewer "github.com/sriannamalai/markdownviewer"
	htmlrender "github.com/sriannamalai/markdownviewer/render/html"
)

// options is the version-1 options JSON accepted by every exported
// function. Fields irrelevant to an operation (e.g. theme for mdv_parse)
// are decoded and ignored by that operation.
type options struct {
	Version        *int              `json:"version"`
	Theme          string            `json:"theme"`
	Fragment       bool              `json:"fragment"`
	AllowRawHTML   bool              `json:"allowRawHTML"`
	Mermaid        bool              `json:"mermaid"`
	Math           bool              `json:"math"`
	Highlighting   bool              `json:"highlighting"`
	MaxWidth       string            `json:"maxWidth"`
	SourceMap      bool              `json:"sourceMap"`
	ThemeOverrides map[string]string `json:"themeOverrides"`
	Stylesheet     string            `json:"stylesheet"`
}

func defaultOptions() options {
	return options{Theme: "auto", Mermaid: true, Math: true, Highlighting: true}
}

// decodeOptions strictly decodes the options JSON over the defaults.
// nil, empty, or all-whitespace input yields the defaults.
func decodeOptions(data []byte) (options, error) {
	o := defaultOptions()
	if len(bytes.TrimSpace(data)) == 0 {
		return o, nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&o); err != nil {
		return options{}, fmt.Errorf("options: %w", err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return options{}, fmt.Errorf("options: trailing data after the options object")
	}
	if o.Version != nil && *o.Version != 1 {
		return options{}, fmt.Errorf("options: unsupported version %d (want 1)", *o.Version)
	}
	return o, nil
}

func (o options) toFacadeOptions(resolver htmlrender.Resolver) []markdownviewer.Option {
	var opts []markdownviewer.Option
	if resolver != nil {
		opts = append(opts, markdownviewer.WithResolver(resolver))
	}
	if o.Theme != "" {
		opts = append(opts, markdownviewer.WithTheme(o.Theme))
	}
	if o.Fragment {
		opts = append(opts, markdownviewer.Fragment())
	}
	if o.AllowRawHTML {
		opts = append(opts, markdownviewer.AllowRawHTML())
	}
	if !o.Mermaid {
		opts = append(opts, markdownviewer.DisableMermaid())
	}
	if !o.Math {
		opts = append(opts, markdownviewer.DisableMath())
	}
	if !o.Highlighting {
		opts = append(opts, markdownviewer.DisableHighlighting())
	}
	if o.SourceMap {
		opts = append(opts, markdownviewer.WithSourceMap())
	}
	if len(o.ThemeOverrides) > 0 {
		opts = append(opts, markdownviewer.WithThemeOverrides(o.ThemeOverrides))
	}
	// After WithThemeOverrides: it replaces the whole override map, and
	// WithMaxWidth sets a key inside it (see the facade godoc).
	if o.MaxWidth != "" {
		opts = append(opts, markdownviewer.WithMaxWidth(o.MaxWidth))
	}
	if o.Stylesheet != "" {
		opts = append(opts, markdownviewer.WithStylesheet(o.Stylesheet))
	}
	return opts
}

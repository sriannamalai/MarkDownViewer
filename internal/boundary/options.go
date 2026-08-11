package boundary

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	markdownviewer "github.com/sriannamalai/markdownviewer"
	"github.com/sriannamalai/markdownviewer/parser"
)

// options is the version-1 options JSON accepted by every exported
// function. Fields irrelevant to an operation (e.g. theme for mdv_parse,
// or parser for mdv_render_doc, which consumes an already-parsed
// document) are decoded and ignored by that operation.
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
	ExtraCSS       string            `json:"extraCss"`
	CodeHeader     bool              `json:"codeHeader"`
	HeadingAnchors bool              `json:"headingAnchors"`
	Parser         *parserOptions    `json:"parser"`
}

// parserOptions is the nested "parser" object: a 1:1 JSON mirror of
// parser.Config's syntax-extension toggles. Absent (or JSON null) means
// "library default" — parser.Default(), every extension on — with
// byte-identical output to older payloads that had no such field.
//
// commonmarkOnly selects parser.CommonMarkOnly() (the zero Config, no
// extensions) as the base instead of parser.Default(). The per-extension
// fields are tristate overrides applied on top of that base: absent/null
// keeps the base's setting, true enables, false disables — so
// {"commonmarkOnly": true, "tables": true} is pure CommonMark plus GFM
// tables, and {"wikiLinks": false} is the full default set minus
// wiki-links. Note the nested "math" toggle is parse-side ([$x$] syntax
// recognition); the top-level "math" option is render-side (KaTeX).
type parserOptions struct {
	CommonmarkOnly  bool  `json:"commonmarkOnly"`
	Tables          *bool `json:"tables"`
	Strikethrough   *bool `json:"strikethrough"`
	TaskLists       *bool `json:"taskLists"`
	Linkify         *bool `json:"linkify"`
	Footnotes       *bool `json:"footnotes"`
	DefinitionLists *bool `json:"definitionLists"`
	FrontMatter     *bool `json:"frontMatter"`
	Emoji           *bool `json:"emoji"`
	WikiLinks       *bool `json:"wikiLinks"`
	Math            *bool `json:"math"`
	Admonitions     *bool `json:"admonitions"`
}

// toConfig folds the nested parser object into a parser.Config.
func (p *parserOptions) toConfig() parser.Config {
	cfg := parser.Default()
	if p.CommonmarkOnly {
		cfg = parser.CommonMarkOnly()
	}
	for dst, v := range map[*bool]*bool{
		&cfg.Tables:          p.Tables,
		&cfg.Strikethrough:   p.Strikethrough,
		&cfg.TaskLists:       p.TaskLists,
		&cfg.Linkify:         p.Linkify,
		&cfg.Footnotes:       p.Footnotes,
		&cfg.DefinitionLists: p.DefinitionLists,
		&cfg.FrontMatter:     p.FrontMatter,
		&cfg.Emoji:           p.Emoji,
		&cfg.WikiLinks:       p.WikiLinks,
		&cfg.Math:            p.Math,
		&cfg.Admonitions:     p.Admonitions,
	} {
		if v != nil {
			*dst = *v
		}
	}
	return cfg
}

// jsonKeys returns the exact JSON key of every field of the struct type
// v, derived from the struct tags so it cannot drift from the decoder.
func jsonKeys(v any) map[string]bool {
	keys := map[string]bool{}
	t := reflect.TypeOf(v)
	for i := 0; i < t.NumField(); i++ {
		name, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
		keys[name] = true
	}
	return keys
}

// knownOptionKeys / knownParserOptionKeys hold the exact JSON keys of
// options and its nested parser object. encoding/json matches field names
// case-insensitively even with DisallowUnknownFields, so strict decoding
// additionally requires an exact-case key match against these sets (e.g.
// "extraCSS" must be rejected, not silently folded into "extraCss" —
// likewise "parser.wikilinks" vs "parser.wikiLinks").
var (
	knownOptionKeys       = jsonKeys(options{})
	knownParserOptionKeys = jsonKeys(parserOptions{})
)

func defaultOptions() options {
	return options{Theme: "auto", Mermaid: true, Math: true, Highlighting: true, HeadingAnchors: true}
}

// decodeOptions strictly decodes the options JSON over the defaults.
// nil, empty, or all-whitespace input yields the defaults.
func decodeOptions(data []byte) (options, error) {
	o := defaultOptions()
	if len(bytes.TrimSpace(data)) == 0 {
		return o, nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err == nil {
		// Malformed/trailing input is left for the strict decode below to
		// report; this pass only enforces exact-case key names.
		for k := range raw {
			if !knownOptionKeys[k] {
				return options{}, fmt.Errorf("options: unknown field %q", k)
			}
		}
		// The nested "parser" object gets the same exact-case treatment.
		// Its value arrives as a json.RawMessage here; re-unmarshalling it
		// as a key map succeeds for an object (and for JSON null, which
		// yields a nil map — same as an absent field). Any other value
		// kind is rejected up front with a clearer message than the strict
		// decoder's type error.
		if p, ok := raw["parser"]; ok {
			var nested map[string]json.RawMessage
			if err := json.Unmarshal(p, &nested); err != nil {
				return options{}, fmt.Errorf("options: \"parser\" must be a JSON object")
			}
			for k := range nested {
				if !knownParserOptionKeys[k] {
					return options{}, fmt.Errorf("options: unknown field %q", "parser."+k)
				}
			}
		}
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

func (o options) toFacadeOptions(resolver markdownviewer.Resolver) []markdownviewer.Option {
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
	if !o.HeadingAnchors {
		opts = append(opts, markdownviewer.DisableHeadingAnchors())
	}
	// Parse-time only: Render/RenderTo parse with this config; RenderDoc
	// receives an already-parsed tree, so WithParserConfig is a no-op
	// there (the field is decoded and ignored, like theme for mdv_parse).
	if o.Parser != nil {
		opts = append(opts, markdownviewer.WithParserConfig(o.Parser.toConfig()))
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
	if o.ExtraCSS != "" {
		opts = append(opts, markdownviewer.WithExtraCSS(o.ExtraCSS))
	}
	if o.CodeHeader {
		opts = append(opts, markdownviewer.WithCodeHeader())
	}
	return opts
}

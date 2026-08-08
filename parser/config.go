package parser

// Config selects which syntax extensions are enabled, each independently
// toggleable. The zero value is pure CommonMark; Default() enables
// everything.
type Config struct {
	// Tables enables GFM pipe tables; Strikethrough enables ~~text~~;
	// TaskLists enables "- [ ]"/"- [x]" list items; Linkify auto-links bare
	// URLs and email addresses.
	Tables, Strikethrough, TaskLists, Linkify bool
	// Footnotes enables [^label] references and definitions;
	// DefinitionLists enables term/description lists; FrontMatter enables
	// leading YAML/TOML/JSON metadata blocks.
	Footnotes, DefinitionLists, FrontMatter bool
	// Emoji enables :shortcode: substitution; WikiLinks enables [[Target]]
	// links; Math enables $inline$ and $$display$$ math; Admonitions
	// promotes GitHub-style [!NOTE]-marked blockquotes.
	Emoji, WikiLinks, Math, Admonitions bool
}

// Default returns a Config with every syntax extension enabled.
func Default() Config {
	return Config{
		Tables: true, Strikethrough: true, TaskLists: true, Linkify: true,
		Footnotes: true, DefinitionLists: true, FrontMatter: true,
		Emoji: true, WikiLinks: true, Math: true, Admonitions: true,
	}
}

// CommonMarkOnly returns the zero Config, enabling no extensions beyond
// plain CommonMark.
func CommonMarkOnly() Config { return Config{} }

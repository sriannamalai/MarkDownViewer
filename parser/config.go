package parser

// Config selects which syntax extensions are enabled. The zero value is
// pure CommonMark; Default() enables everything.
type Config struct {
	Tables, Strikethrough, TaskLists, Linkify bool
	Footnotes, DefinitionLists, FrontMatter   bool
	Emoji, WikiLinks, Math, Admonitions       bool
}

func Default() Config {
	return Config{
		Tables: true, Strikethrough: true, TaskLists: true, Linkify: true,
		Footnotes: true, DefinitionLists: true, FrontMatter: true,
		Emoji: true, WikiLinks: true, Math: true, Admonitions: true,
	}
}

func CommonMarkOnly() Config { return Config{} }

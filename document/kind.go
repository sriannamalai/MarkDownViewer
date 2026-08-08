package document

var kindNames = [...]string{
	KindDocument: "document", KindHeading: "heading", KindParagraph: "paragraph",
	KindBlockQuote: "blockQuote", KindAdmonition: "admonition", KindList: "list",
	KindListItem: "listItem", KindCodeBlock: "codeBlock", KindDiagram: "diagram",
	KindMathBlock: "mathBlock", KindTable: "table", KindTableRow: "tableRow",
	KindTableCell: "tableCell", KindThematicBreak: "thematicBreak",
	KindHTMLBlock: "htmlBlock", KindDefinitionList: "definitionList",
	KindDefinitionTerm: "definitionTerm", KindDefinitionDesc: "definitionDesc",
	KindFootnoteDef: "footnoteDef", KindText: "text", KindSoftBreak: "softBreak",
	KindHardBreak: "hardBreak", KindEmphasis: "emphasis", KindStrong: "strong",
	KindStrikethrough: "strikethrough", KindCodeSpan: "codeSpan", KindLink: "link",
	KindImage: "image", KindWikiLink: "wikiLink", KindMathInline: "mathInline",
	KindHTMLInline: "htmlInline", KindFootnoteRef: "footnoteRef",
}

var kindByName = func() map[string]Kind {
	m := make(map[string]Kind, len(kindNames))
	for i, n := range kindNames {
		m[n] = Kind(i)
	}
	return m
}()

// String returns the stable lowerCamel wire name of the kind ("unknown"
// for out-of-range values). The names are a serialization contract.
func (k Kind) String() string {
	if k < 0 || int(k) >= len(kindNames) {
		return "unknown"
	}
	return kindNames[k]
}

// KindFromString resolves a wire name back to its Kind.
func KindFromString(s string) (Kind, bool) {
	k, ok := kindByName[s]
	return k, ok
}

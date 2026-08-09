package main

import (
	markdownviewer "github.com/sriannamalai/markdownviewer"
	"github.com/sriannamalai/markdownviewer/document"
)

// renderImpl is mdv_render behind the cgo boundary: markdown -> HTML.
func renderImpl(md, optsJSON []byte) ([]byte, error) {
	o, err := decodeOptions(optsJSON)
	if err != nil {
		return nil, err
	}
	return markdownviewer.Render(md, o.toFacadeOptions()...)
}

// parseImpl is mdv_parse: markdown -> version-1 document JSON. Options are
// validated for consistency with the other calls, but no current field
// affects parsing.
func parseImpl(md, optsJSON []byte) ([]byte, error) {
	if _, err := decodeOptions(optsJSON); err != nil {
		return nil, err
	}
	doc, err := markdownviewer.Parse(md)
	if err != nil {
		return nil, err
	}
	return document.MarshalJSON(doc)
}

// renderDocImpl is mdv_render_doc: version-1 document JSON -> HTML.
func renderDocImpl(docJSON, optsJSON []byte) ([]byte, error) {
	o, err := decodeOptions(optsJSON)
	if err != nil {
		return nil, err
	}
	doc, err := document.UnmarshalJSON(docJSON)
	if err != nil {
		return nil, err
	}
	return markdownviewer.RenderDoc(doc, o.toFacadeOptions()...)
}

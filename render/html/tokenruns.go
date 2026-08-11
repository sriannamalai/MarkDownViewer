package htmlrender

import (
	"crypto/sha256"
	"strings"
)

// TokenRun is one syntax-highlight token of a code block: a slice of the
// code text tagged with chroma's canonical token-type name
// (chroma.TokenType.String(), e.g. "Keyword", "LiteralString",
// "NameFunction"). It is the data form of the tokenise half of the v0.9
// tokenise/format split — what a native renderer styles directly,
// without the HTML formatter.
type TokenRun struct {
	Text      string
	TokenType string
}

// TokenRuns returns the syntax-highlight token runs for (code, lang),
// consulting the bounded package-level runs cache first (the tokenise
// step dominates render CPU; see runsCache). Returns (nil, false) when
// lang is empty/unknown or chroma fails — mirroring highlightHTML's
// fallback-to-plain semantics — and failures are never cached.
//
// Invariant: on ok, the concatenation of the returned runs' Text is
// exactly code — the parity guarantee native renderers rely on. Chroma
// lexers configured with EnsureNL append a trailing newline the source
// lacks; that byte is trimmed back off the final run. Any other
// divergence (e.g. EnsureLF rewriting CRLF line endings) returns
// (nil, false) so the caller falls back to plain text rather than
// styling text that differs from the source.
//
// The returned slice may be shared with the cache and other callers:
// treat it as immutable.
func TokenRuns(code, lang string) ([]TokenRun, bool) {
	if lang == "" {
		return nil, false
	}
	key := highlightKey{lang: lang, sum: sha256.Sum256([]byte(code))}
	if runs, ok := runsCache.get(key); ok {
		return runs, true
	}
	it := tokenise(code, lang)
	if it == nil {
		return nil, false
	}
	runs := []TokenRun{} // non-nil: empty code yields ok with zero runs
	var joined strings.Builder
	for _, tok := range it.Tokens() {
		if tok.Value == "" {
			continue
		}
		runs = append(runs, TokenRun{Text: tok.Value, TokenType: tok.Type.String()})
		joined.WriteString(tok.Value)
	}
	runs, ok := normalizeRuns(runs, joined.String(), code)
	if !ok {
		return nil, false
	}
	runsCache.put(key, runs)
	return runs, true
}

// normalizeRuns enforces the exact-concatenation invariant: joined (the
// concatenation of the runs' Text) must spell code byte-for-byte. A
// single trailing "\n" the lexer appended (EnsureNL) is trimmed off the
// final run; any other divergence reports !ok.
func normalizeRuns(runs []TokenRun, joined, code string) ([]TokenRun, bool) {
	if joined == code {
		return runs, true
	}
	if joined != code+"\n" || len(runs) == 0 {
		return nil, false
	}
	last := runs[len(runs)-1].Text // ends in the appended '\n' by construction
	if len(last) == 1 {
		return runs[:len(runs)-1], true
	}
	runs[len(runs)-1].Text = last[:len(last)-1]
	return runs, true
}

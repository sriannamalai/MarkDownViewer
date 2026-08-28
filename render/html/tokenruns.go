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
// lexers tokenise with EnsureLF on, rewriting CRLF/CR line endings to
// LF before lexing; when code contains \r, the runs it returns are
// re-expanded back to code's original CRLF/CR bytes (see crlfToLF) so
// this invariant holds against the real source rather than chroma's
// normalized view of it. Chroma lexers configured with EnsureNL also
// append a trailing newline the source lacks; that byte is trimmed back
// off the final run.
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
	if !strings.ContainsRune(code, '\r') {
		// Fast path: chroma's EnsureLF is a no-op with no \r present, so
		// joined is already in code's own byte space.
		runs, ok := normalizeRuns(runs, joined.String(), code)
		if !ok {
			return nil, false
		}
		runsCache.put(key, runs)
		return runs, true
	}
	// code has \r: chroma's lexer normalized CRLF/CR to LF before
	// tokenising, so joined is in LF-normalized space, not code's. Redo
	// the same normalization locally (crlfToLF) to know what chroma
	// actually tokenised, validate the runs against that, then re-expand
	// each run's text back into code's original bytes (origOffset) so
	// the exact-concatenation invariant holds against the real source
	// instead of silently declining CRLF fences.
	norm, origOffset := crlfToLF(code)
	runs, ok := normalizeRuns(runs, joined.String(), norm)
	if !ok {
		return nil, false
	}
	runs = restoreCRLF(runs, origOffset, code)
	runsCache.put(key, runs)
	return runs, true
}

// normalizeRuns enforces the exact-concatenation invariant: joined (the
// concatenation of the runs' Text) must spell target byte-for-byte
// (code itself when code has no \r, or its LF-normalized form
// otherwise — see TokenRuns). A single trailing "\n" the lexer appended
// (EnsureNL) is trimmed off the final run; any other divergence reports
// !ok.
func normalizeRuns(runs []TokenRun, joined, target string) ([]TokenRun, bool) {
	if joined == target {
		return runs, true
	}
	if joined != target+"\n" || len(runs) == 0 {
		return nil, false
	}
	last := runs[len(runs)-1].Text // ends in the appended '\n' by construction
	if len(last) == 1 {
		return runs[:len(runs)-1], true
	}
	runs[len(runs)-1].Text = last[:len(last)-1]
	return runs, true
}

// crlfToLF mirrors chroma's internal EnsureLF normalization exactly
// (every "\r\n" pair and every lone "\r" becomes "\n"; see
// alecthomas/chroma/v2's regexp.go ensureLF), while also recording,
// for each byte of the returned norm, the offset in code where that
// byte's source segment starts. origOffset has one extra trailing
// entry (origOffset[len(norm)] == len(code)), so a run spanning
// norm[start:end] corresponds to exactly code[origOffset[start]:
// origOffset[end]] — a 2-byte "\r\n" collapse re-expands to both
// bytes, a lone "\r" re-expands to itself, and an unchanged byte
// re-expands to itself.
func crlfToLF(code string) (norm string, origOffset []int) {
	buf := make([]byte, 0, len(code))
	offs := make([]int, 0, len(code)+1)
	for i := 0; i < len(code); i++ {
		c := code[i]
		if c == '\r' && i+1 < len(code) && code[i+1] == '\n' {
			buf = append(buf, '\n')
			offs = append(offs, i)
			i++ // the paired '\n' collapses into the same output byte
			continue
		}
		if c == '\r' {
			c = '\n'
		}
		buf = append(buf, c)
		offs = append(offs, i)
	}
	offs = append(offs, len(code))
	return string(buf), offs
}

// restoreCRLF re-expands runs — whose Text is exactly norm from
// crlfToLF, byte for byte — back into code's original CRLF/CR bytes
// using origOffset, preserving every run's token boundaries.
func restoreCRLF(runs []TokenRun, origOffset []int, code string) []TokenRun {
	pos := 0
	for i := range runs {
		n := len(runs[i].Text)
		start, end := origOffset[pos], origOffset[pos+n]
		runs[i].Text = code[start:end]
		pos += n
	}
	return runs
}

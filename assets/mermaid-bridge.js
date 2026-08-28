// mdvRenderMermaid(id, source, theme) renders Mermaid diagram source to
// an SVG string using the "mermaid.js" asset bundle, without requiring
// the diagram to be attached to a visible on-screen element the way the
// full-page HTML renderer's "startOnLoad" scan does. It is meant for
// hosts that only have an offscreen/headless webview available — e.g. a
// native app rendering a render-tree Diagram block's Source into real
// SVG for display outside a browser tab. See the design note
// ".superpowers/specs/2026-08-28 Mermaid Offscreen-SVG Direction.md"
// for why this stays a host-side (webview) responsibility instead of a
// Go-side rendering pipeline.
//
// Usage: inject the "mermaid.js" asset FIRST, then this script, into
// the same JS context (an offscreen WebView, a hidden iframe, etc.),
// then call:
//
//   mdvRenderMermaid(id, source, theme).then(function (result) { ... })
//
// - id: a unique, HTML-id-safe string. Callers should pass the render
//   tree's Diagram.ID (already unique per document) so repeated calls
//   for the same document are stable and collision-free.
// - source: the diagram's raw Mermaid source (Diagram.Source).
// - theme: one of mermaid's theme names ("default", "dark", "forest",
//   "neutral", "base") — mirrors the theme argument the full-page HTML
//   renderer passes to mermaid.initialize. Optional; defaults to
//   "default".
//
// Resolves to {ok:true, svg:"<svg ...>...</svg>"} on success, or
// {ok:false, error:"..."} on failure (invalid diagram source, a
// mermaid exception, etc.) — this function never throws and its
// returned promise never rejects, so a host can treat every outcome as
// data.
(function () {
  "use strict";
  var initializedTheme = null;
  window.mdvRenderMermaid = function (id, source, theme) {
    try {
      theme = theme || "default";
      // mermaid.initialize sets global config; only re-run it when the
      // requested theme actually changes, so repeated calls with the
      // same theme don't pay redundant re-init cost, while a genuine
      // theme switch is always picked up before the next render.
      if (initializedTheme !== theme) {
        mermaid.initialize({ startOnLoad: false, theme: theme });
        initializedTheme = theme;
      }
      return mermaid
        .render(id, source)
        .then(function (result) {
          return { ok: true, svg: result.svg };
        })
        .catch(function (err) {
          return { ok: false, error: String((err && err.message) || err) };
        });
    } catch (err) {
      return Promise.resolve({ ok: false, error: String((err && err.message) || err) });
    }
  };
})();

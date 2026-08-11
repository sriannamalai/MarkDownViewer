/* Smoke-test harness for libmdviewer. Exercises every exported symbol and
 * exits non-zero on the first failure. Build/run: see README.md. */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include "libmdviewer.h"

static int failures = 0;

#define CHECK(cond, msg) do { \
    if (!(cond)) { fprintf(stderr, "FAIL: %s\n", (msg)); failures++; } \
    else { printf("ok: %s\n", (msg)); } \
} while (0)

/* ---- resolver test support ---- */
struct resolver_state {
    int calls;
    int saw_kind[3];          /* index by kind 0/1/2 */
    char last_target[256];
    void *seen_userdata;
    char *owned;               /* mode 7: buffer the host retains ownership of */
    int mode;                 /* 0: resolve images, decline rest
                                 1: resolve javascript: link (trust contract)
                                 2: invalid return code
                                 3: return 1 with NULL out_url
                                 4: resolve image to empty URL
                                 5: resolve wiki-links, decline rest
                                 6: return 1 with an oversized out_url_len
                                    after a real allocation (MaxInt guard)
                                 7: invalid return code after allocating;
                                    host retains ownership and frees its
                                    own buffer (no-touch rule) */
};

static int test_resolver(int kind, const char *target, size_t target_len,
                         void *userdata, char **out_url, size_t *out_url_len) {
    struct resolver_state *st = (struct resolver_state *)userdata;
    st->calls++;
    if (kind >= 0 && kind < 3) st->saw_kind[kind] = 1;
    size_t n = target_len < sizeof(st->last_target) - 1 ? target_len : sizeof(st->last_target) - 1;
    memcpy(st->last_target, target, n);
    st->last_target[n] = '\0';
    st->seen_userdata = userdata;

    if (st->mode == 2) return 7;
    if (st->mode == 3) { *out_url = NULL; return 1; }
    if (st->mode == 4 && kind == 1) {
        *out_url = (char *)mdv_alloc(0);
        *out_url_len = 0;
        return 1;
    }
    if (st->mode == 1 && kind == 0 && target_len >= 11 &&
        memcmp(target, "javascript:", 11) == 0) {
        const char *u = "javascript:alert(1)";
        size_t ul = strlen(u);
        char *buf = (char *)mdv_alloc(ul);
        if (!buf) return 0;
        memcpy(buf, u, ul);
        *out_url = buf; *out_url_len = ul;
        return 1;
    }
    if (st->mode == 0 && kind == 1) {
        const char *u = "asset://img/photo.png";
        size_t ul = strlen(u);
        char *buf = (char *)mdv_alloc(ul);
        if (!buf) return 0;
        memcpy(buf, u, ul);
        *out_url = buf; *out_url_len = ul;
        return 1;
    }
    if (st->mode == 5 && kind == 2) {
        const char *u = "notes/Wiki Page.html";
        size_t ul = strlen(u);
        char *buf = (char *)mdv_alloc(ul);
        if (!buf) return 0;
        memcpy(buf, u, ul);
        *out_url = buf; *out_url_len = ul;
        return 1;
    }
    if (st->mode == 6 && kind == 1) {
        char *buf = (char *)mdv_alloc(4);
        if (!buf) return 0;
        memcpy(buf, "abcd", 4);
        *out_url = buf;
        *out_url_len = (size_t)-1;  /* lie: SIZE_MAX */
        return 1;
    }
    if (st->mode == 7 && kind == 1) {
        st->owned = (char *)mdv_alloc(8);
        if (st->owned) memcpy(st->owned, "leakless", 8);
        *out_url = st->owned;
        *out_url_len = 8;
        return 42;  /* contract violation; library must NOT free out_url */
    }
    return 0;
}

int main(void) {
    const char *md = "# Hello *world*\n\n- [x] done\n";
    size_t md_len = strlen(md);

    const char *ver = mdv_version();
    CHECK(ver != NULL && ver[0] != '\0', "mdv_version returns a version");
    printf("libmdviewer %s\n", ver ? ver : "(null)");

    /* Render with defaults. */
    char *html = NULL, *err = NULL; size_t html_len = 0;
    int rc = mdv_render((char *)md, md_len, NULL, &html, &html_len, &err);
    CHECK(rc == 0 && err == NULL, "mdv_render succeeds with NULL options");
    CHECK(html != NULL && html_len > 0, "mdv_render returns output");
    CHECK(html != NULL && html[html_len] == '\0', "output has trailing NUL");
    CHECK(html != NULL && strstr(html, "<h1") != NULL, "output contains <h1");
    CHECK(html != NULL && strstr(html, "Hello") != NULL, "output contains text");

    /* Parse -> render_doc must match direct render byte-for-byte. */
    char *doc = NULL, *err2 = NULL; size_t doc_len = 0;
    rc = mdv_parse((char *)md, md_len, NULL, &doc, &doc_len, &err2);
    CHECK(rc == 0 && doc != NULL, "mdv_parse succeeds");
    CHECK(doc != NULL && strstr(doc, "\"version\":1") != NULL, "document JSON is version 1");

    char *html2 = NULL, *err3 = NULL; size_t html2_len = 0;
    rc = mdv_render_doc(doc, doc_len, NULL, &html2, &html2_len, &err3);
    CHECK(rc == 0 && html2 != NULL, "mdv_render_doc succeeds");
    CHECK(html2 != NULL && html != NULL && html2_len == html_len &&
          memcmp(html, html2, html_len) == 0,
          "render and parse->render_doc agree byte-for-byte");

    /* Fragment option changes output. */
    char *frag = NULL, *err4 = NULL; size_t frag_len = 0;
    rc = mdv_render((char *)md, md_len, (char *)"{\"fragment\": true}", &frag, &frag_len, &err4);
    CHECK(rc == 0 && frag != NULL, "mdv_render with options succeeds");
    CHECK(frag != NULL && strstr(frag, "<html") == NULL, "fragment output has no <html");

    /* Error paths. */
    char *out = NULL, *bad_err = NULL; size_t out_len = 0;
    rc = mdv_render((char *)md, md_len, (char *)"{\"bogus\": 1}", &out, &out_len, &bad_err);
    CHECK(rc != 0 && out == NULL, "unknown option field fails");
    CHECK(bad_err != NULL && strstr(bad_err, "bogus") != NULL, "error names the bad field");
    mdv_free(bad_err); bad_err = NULL;

    rc = mdv_render(NULL, 5, NULL, &out, &out_len, &bad_err);
    CHECK(rc != 0, "NULL input with non-zero length fails");
    mdv_free(bad_err); bad_err = NULL;

    rc = mdv_render(NULL, 0, NULL, &out, &out_len, &bad_err);
    CHECK(rc == 0 && out != NULL, "NULL input with zero length renders empty doc");
    mdv_free(out); out = NULL;

    char *doc_bad_err = NULL;
    rc = mdv_render_doc((char *)"{", 1, NULL, &out, &out_len, &doc_bad_err);
    CHECK(rc != 0 && doc_bad_err != NULL, "invalid document JSON fails");
    mdv_free(doc_bad_err);

    /* Assets (v0.5). */
    char *asset = NULL, *asset_err = NULL; size_t asset_len = 0;
    rc = mdv_asset((char *)"mermaid.js", &asset, &asset_len, &asset_err);
    CHECK(rc == 0 && asset != NULL && asset_len > 1000, "mdv_asset returns mermaid.js");
    CHECK(asset != NULL && strstr(asset, "mermaid") != NULL, "mermaid.js has content marker");
    mdv_free(asset); asset = NULL;

    rc = mdv_asset((char *)"theme-dark.css", &asset, &asset_len, &asset_err);
    CHECK(rc == 0 && asset != NULL && strstr(asset, ".chroma") != NULL,
          "theme-dark.css includes chroma highlighting CSS");
    mdv_free(asset); asset = NULL;

    rc = mdv_asset((char *)"theme-light.json", &asset, &asset_len, &asset_err);
    CHECK(rc == 0 && asset != NULL && strstr(asset, "--md-bg") != NULL,
          "theme-light.json carries the --md-bg palette token");
    CHECK(asset != NULL && strstr(asset, "\"version\":1") != NULL &&
          strstr(asset, "\"mode\":\"light\"") != NULL,
          "theme-light.json is version-1 JSON with mode light");
    mdv_free(asset); asset = NULL;

    rc = mdv_asset((char *)"bogus.js", &asset, &asset_len, &asset_err);
    CHECK(rc != 0 && asset_err != NULL && strstr(asset_err, "mermaid.js") != NULL,
          "unknown asset error lists valid names");
    mdv_free(asset_err); asset_err = NULL;

    rc = mdv_asset(NULL, &asset, &asset_len, &asset_err);
    CHECK(rc != 0, "NULL asset name fails");
    mdv_free(asset_err); asset_err = NULL;

    /* ---- Resolver callback (mdv_render_r / mdv_render_doc_r) ---- */
    const char *rmd =
        "![alt](img/photo.png)\n\n"
        "[click](docs/guide.md)\n\n"
        "[[Wiki Page]]\n\n"
        "[evil](javascript:alert(1))\n";
    size_t rmd_len = strlen(rmd);

    /* Mode 0: images resolved, links/wiki declined -> defaults. */
    struct resolver_state st0; memset(&st0, 0, sizeof st0); st0.mode = 0;
    char *rh = NULL, *re = NULL; size_t rl = 0;
    rc = mdv_render_r((char *)rmd, rmd_len, (char *)"{\"fragment\": true}",
                      test_resolver, &st0, &rh, &rl, &re);
    CHECK(rc == 0 && rh != NULL, "mdv_render_r succeeds with resolver");
    CHECK(rh && strstr(rh, "src=\"asset://img/photo.png\"") != NULL,
          "resolved image URL emitted verbatim");
    CHECK(rh && strstr(rh, "href=\"docs/guide.md\"") != NULL,
          "declined link takes default resolution");
    CHECK(rh && strstr(rh, "Wiki Page.md") != NULL,
          "declined wiki-link gets default .md resolution");
    CHECK(rh && strstr(rh, "javascript:") == NULL,
          "declined unsafe link is filtered by safeURL");
    CHECK(st0.calls >= 4, "resolver called for every target");
    CHECK(st0.saw_kind[0] && st0.saw_kind[1] && st0.saw_kind[2],
          "resolver saw kinds 0 (link), 1 (image), 2 (wiki-link)");
    CHECK(st0.seen_userdata == &st0, "userdata passed through untouched");
    mdv_free(rh); rh = NULL;

    /* Trust contract: a resolved URL bypasses safeURL. */
    struct resolver_state st1; memset(&st1, 0, sizeof st1); st1.mode = 1;
    rc = mdv_render_r((char *)rmd, rmd_len, (char *)"{\"fragment\": true}",
                      test_resolver, &st1, &rh, &rl, &re);
    CHECK(rc == 0 && rh && strstr(rh, "javascript:alert(1)") != NULL,
          "resolved URL is trusted verbatim (bypasses safeURL)");
    mdv_free(rh); rh = NULL;

    /* Contract violations -> error, not crash. */
    struct resolver_state st2; memset(&st2, 0, sizeof st2); st2.mode = 2;
    rc = mdv_render_r((char *)rmd, rmd_len, NULL, test_resolver, &st2, &rh, &rl, &re);
    CHECK(rc != 0 && rh == NULL, "invalid resolver return code fails render");
    CHECK(re != NULL && strstr(re, "resolver") != NULL,
          "invalid-return error mentions the resolver");
    mdv_free(re); re = NULL;

    struct resolver_state st3; memset(&st3, 0, sizeof st3); st3.mode = 3;
    rc = mdv_render_r((char *)rmd, rmd_len, NULL, test_resolver, &st3, &rh, &rl, &re);
    CHECK(rc != 0 && re != NULL && strstr(re, "resolver") != NULL,
          "return 1 with NULL out_url fails render");
    mdv_free(re); re = NULL;

    /* Empty URL via mdv_alloc(0). */
    void *z = mdv_alloc(0);
    CHECK(z != NULL, "mdv_alloc(0) returns non-NULL");
    mdv_free((char *)z);
    struct resolver_state st4; memset(&st4, 0, sizeof st4); st4.mode = 4;
    rc = mdv_render_r((char *)rmd, rmd_len, (char *)"{\"fragment\": true}",
                      test_resolver, &st4, &rh, &rl, &re);
    CHECK(rc == 0 && rh && strstr(rh, "src=\"\"") != NULL,
          "empty resolved URL is representable");
    mdv_free(rh); rh = NULL;

    /* NULL resolver === plain variant, byte for byte. */
    char *plain = NULL; size_t plain_len = 0; char *pe = NULL;
    rc = mdv_render((char *)rmd, rmd_len, NULL, &plain, &plain_len, &pe);
    char *viaR = NULL; size_t viaR_len = 0; char *ve = NULL;
    int rc2 = mdv_render_r((char *)rmd, rmd_len, NULL, NULL, NULL, &viaR, &viaR_len, &ve);
    CHECK(rc == 0 && rc2 == 0 && plain_len == viaR_len &&
          memcmp(plain, viaR, plain_len) == 0,
          "NULL resolver is byte-identical to mdv_render");
    mdv_free(plain); mdv_free(viaR);

    /* mdv_render_doc_r threads the resolver too. */
    char *rdoc = NULL; size_t rdoc_len = 0; char *de = NULL;
    rc = mdv_parse((char *)rmd, rmd_len, NULL, &rdoc, &rdoc_len, &de);
    CHECK(rc == 0 && rdoc != NULL, "mdv_parse for render_doc_r succeeds");
    struct resolver_state st5; memset(&st5, 0, sizeof st5); st5.mode = 0;
    rc = mdv_render_doc_r(rdoc, rdoc_len, (char *)"{\"fragment\": true}",
                          test_resolver, &st5, &rh, &rl, &re);
    CHECK(rc == 0 && rh && strstr(rh, "src=\"asset://img/photo.png\"") != NULL,
          "mdv_render_doc_r resolves via callback");
    mdv_free(rh); mdv_free(rdoc);

    /* Wiki-link RESOLVE path (kind 2), not just decline. */
    struct resolver_state st6; memset(&st6, 0, sizeof st6); st6.mode = 5;
    rc = mdv_render_r((char *)rmd, rmd_len, (char *)"{\"fragment\": true}",
                      test_resolver, &st6, &rh, &rl, &re);
    CHECK(rc == 0 && rh != NULL, "mdv_render_r succeeds with wiki-resolving resolver");
    CHECK(rh && strstr(rh, "href=\"notes/Wiki Page.html\"") != NULL,
          "resolved wiki-link URL emitted verbatim");
    CHECK(rh && strstr(rh, "Wiki Page.md") == NULL,
          "resolved wiki-link does not fall back to .md default");
    mdv_free(rh); rh = NULL;

    /* Mode 6: return 1 with an oversized out_url_len after a real
     * allocation. Pins the MaxInt guard: the library must reject this
     * as a contract violation (not read out of bounds), and since
     * return code was 1, ownership DID transfer — the library frees
     * the buffer itself. */
    struct resolver_state st7_6; memset(&st7_6, 0, sizeof st7_6); st7_6.mode = 6;
    rc = mdv_render_r((char *)rmd, rmd_len, NULL, test_resolver, &st7_6, &rh, &rl, &re);
    CHECK(rc != 0 && rh == NULL, "oversized out_url_len after allocation fails render");
    CHECK(re != NULL && strstr(re, "resolver") != NULL,
          "oversized out_url_len error mentions the resolver");
    mdv_free(re); re = NULL;

    /* Mode 7: invalid return code after allocating. Pins the no-touch
     * rule: no ownership transfer happened (return code wasn't 1), so
     * the library must NOT free *out_url — the host retains ownership
     * and frees its own buffer below. Under the old code (which freed
     * a non-NULL out_url on any violation path) this sequence would
     * double-free; under ASan that is the check that matters. */
    struct resolver_state st7; memset(&st7, 0, sizeof st7); st7.mode = 7;
    rc = mdv_render_r((char *)rmd, rmd_len, NULL, test_resolver, &st7, &rh, &rl, &re);
    CHECK(rc != 0 && rh == NULL, "invalid return code after allocation fails render");
    CHECK(re != NULL && strstr(re, "resolver") != NULL,
          "invalid-return-after-allocation error mentions the resolver");
    mdv_free(re); re = NULL;
    mdv_free(st7.owned); /* host frees its own buffer; must not double-free */

    /* ---- v0.8 options: extraCss + codeHeader ---- */

    /* extraCss through mdv_render_doc: present, and appended AFTER the
     * base styling (theme tokens like --md-bg come first in <style>). */
    char *xh = NULL, *xe = NULL; size_t xl = 0;
    rc = mdv_render_doc(doc, doc_len,
                        (char *)"{\"extraCss\":\"body{font-size:117%}\"}",
                        &xh, &xl, &xe);
    CHECK(rc == 0 && xh != NULL, "mdv_render_doc with extraCss succeeds");
    CHECK(xh && strstr(xh, "body{font-size:117%}") != NULL,
          "extraCss appears in the output");
    CHECK(xh && strstr(xh, "--md-bg") != NULL &&
          strstr(xh, "body{font-size:117%}") != NULL &&
          strstr(xh, "--md-bg") < strstr(xh, "body{font-size:117%}"),
          "extraCss is appended after the base styling tokens");
    mdv_free(xh); xh = NULL;

    /* codeHeader wraps fenced code with header markup + copy button. */
    const char *cmd_md = "```shell\nls -la\n```\n";
    rc = mdv_render((char *)cmd_md, strlen(cmd_md),
                    (char *)"{\"codeHeader\":true}", &xh, &xl, &xe);
    CHECK(rc == 0 && xh != NULL, "mdv_render with codeHeader succeeds");
    CHECK(xh && strstr(xh, "class=\"md-code-lang\">shell") != NULL,
          "codeHeader emits the fence language label");
    /* Assert the markup form: bare "md-code-copy" also appears in the
     * embedded base.css of every full page, opted in or not. */
    CHECK(xh && strstr(xh, "class=\"md-code-copy\"") != NULL,
          "codeHeader emits the copy button");
    mdv_free(xh); xh = NULL;

    /* Wrong-case key is rejected loudly (exact-case boundary). */
    rc = mdv_render((char *)md, md_len, (char *)"{\"extraCSS\":\"x\"}",
                    &xh, &xl, &xe);
    CHECK(rc != 0 && xh == NULL, "wrong-case extraCSS key fails");
    CHECK(xe != NULL && strstr(xe, "extraCSS") != NULL,
          "wrong-case error names the offending key");
    mdv_free(xe); xe = NULL;

    /* ---- v0.9 options: parser config + headingAnchors ---- */

    /* parser.wikiLinks=false renders [[x]] as literal text. */
    const char *wiki_md = "[[Wiki Page]]\n";
    rc = mdv_render((char *)wiki_md, strlen(wiki_md),
                    (char *)"{\"fragment\":true,\"parser\":{\"wikiLinks\":false}}",
                    &xh, &xl, &xe);
    CHECK(rc == 0 && xh != NULL && strstr(xh, "[[Wiki Page]]") != NULL &&
          strstr(xh, "<a ") == NULL,
          "parser.wikiLinks=false renders wiki syntax literally");
    mdv_free(xh); xh = NULL;

    /* Wrong-case NESTED key is rejected, named with its path. */
    rc = mdv_render((char *)md, md_len,
                    (char *)"{\"parser\":{\"wikilinks\":false}}", &xh, &xl, &xe);
    CHECK(rc != 0 && xh == NULL && xe != NULL &&
          strstr(xe, "parser.wikilinks") != NULL,
          "wrong-case nested parser key fails naming parser.wikilinks");
    mdv_free(xe); xe = NULL;

    /* headingAnchors=false drops the heading id attributes. */
    const char *h_md = "# Hi\n";
    rc = mdv_render((char *)h_md, strlen(h_md),
                    (char *)"{\"fragment\":true,\"headingAnchors\":false}",
                    &xh, &xl, &xe);
    CHECK(rc == 0 && xh != NULL && strstr(xh, "<h1>") != NULL &&
          strstr(xh, "id=") == NULL,
          "headingAnchors=false omits heading id attributes");
    mdv_free(xh); xh = NULL;

    /* Cleanup: every returned buffer through mdv_free; NULL is a no-op. */
    mdv_free(html); mdv_free(doc); mdv_free(html2); mdv_free(frag);
    mdv_free(NULL);

    if (failures) { fprintf(stderr, "%d failure(s)\n", failures); return 1; }
    printf("all checks passed\n");
    return 0;
}

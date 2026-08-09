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

    rc = mdv_asset((char *)"bogus.js", &asset, &asset_len, &asset_err);
    CHECK(rc != 0 && asset_err != NULL && strstr(asset_err, "mermaid.js") != NULL,
          "unknown asset error lists valid names");
    mdv_free(asset_err); asset_err = NULL;

    rc = mdv_asset(NULL, &asset, &asset_len, &asset_err);
    CHECK(rc != 0, "NULL asset name fails");
    mdv_free(asset_err); asset_err = NULL;

    /* Cleanup: every returned buffer through mdv_free; NULL is a no-op. */
    mdv_free(html); mdv_free(doc); mdv_free(html2); mdv_free(frag);
    mdv_free(NULL);

    if (failures) { fprintf(stderr, "%d failure(s)\n", failures); return 1; }
    printf("all checks passed\n");
    return 0;
}

// CI browser smoke for the wasm playground (examples/web/index.html).
//
// Prerequisites:
//   ./scripts/build-wasm.sh                 # dist/wasm/npm must exist
//   (cd examples/web && npm ci)             # playwright, pinned in package.json
//   npx playwright install chromium         # browser binary
//   python3 -m http.server 8000             # serve the REPO ROOT
//
// Run from the repo root: node examples/web/ci-playwright.mjs
// BASE_URL overrides the server address (default http://127.0.0.1:8000).
//
// Mirrors the manual verification sweep: the page must reach its E2E
// ready hook (body.dataset.ready = '1'), and the sandboxed preview
// iframe must contain the fully rendered default document — a mermaid
// diagram executed to an <svg>, KaTeX-rendered math, and the demo
// resolver's data-URI image.
import { chromium } from 'playwright';

const base = process.env.BASE_URL ?? 'http://127.0.0.1:8000';
const url = `${base}/examples/web/index.html`;
const TIMEOUT = 30_000;

const browser = await chromium.launch();
let failed = false;
try {
  const page = await browser.newPage();

  // Uncaught page errors are always a failure; console errors are
  // surfaced for diagnostics but only fail via the assertions below.
  const pageErrors = [];
  page.on('pageerror', (e) => pageErrors.push(String(e)));
  page.on('console', (m) => {
    if (m.type() === 'error') console.error(`[console.error] ${m.text()}`);
  });

  await page.goto(url, { waitUntil: 'domcontentloaded', timeout: TIMEOUT });

  // E2E hook: index.html sets body.dataset.ready = '1' after the wasm
  // module has loaded and the first render was kicked off.
  await page.waitForSelector('body[data-ready="1"]', { timeout: TIMEOUT });
  console.log('ok: page ready (body[data-ready="1"])');

  const status = await page.textContent('#status');
  if (!/^libmdviewer \S+/.test(status ?? '')) {
    throw new Error(`status line did not report a wasm version: ${JSON.stringify(status)}`);
  }
  console.log(`ok: wasm loaded (${status})`);

  // The preview is a sandboxed srcdoc iframe; Playwright can still
  // reach into it. The default document exercises mermaid + KaTeX.
  const frame = page.frameLocator('#out');
  await frame.locator('pre.mermaid svg').first().waitFor({ timeout: TIMEOUT });
  console.log('ok: mermaid diagram rendered to <svg>');
  await frame.locator('.math .katex').first().waitFor({ timeout: TIMEOUT });
  console.log('ok: KaTeX markup rendered');
  await frame.locator('img[src^="data:image/svg+xml"]').first().waitFor({ timeout: TIMEOUT });
  console.log('ok: resolver-provided image present');

  if (pageErrors.length > 0) {
    throw new Error(`uncaught page error(s):\n${pageErrors.join('\n')}`);
  }
  console.log('all checks passed');
} catch (e) {
  failed = true;
  console.error(`FAIL: ${e instanceof Error ? e.message : e}`);
} finally {
  await browser.close();
}
process.exit(failed ? 1 : 0);

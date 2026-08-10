// Smoke-test harness for the libmdviewer wasm build — the Node twin of
// examples/c/harness.c. Build first: ./scripts/build-wasm.sh
// Run from the repo root: node examples/node/harness.mjs
import { loadMdviewer } from '../../dist/wasm/npm/index.js';
import { spawnSync } from 'node:child_process';

let failures = 0;
function check(cond, msg) {
  if (cond) console.log(`ok: ${msg}`);
  else { console.error(`FAIL: ${msg}`); failures++; }
}
function throws(fn, substr, msg) {
  try { fn(); check(false, `${msg} (did not throw)`); }
  catch (e) { check(String(e.message).includes(substr), `${msg} [${e.message}]`); }
}

const mdv = await loadMdviewer();
const md = '# Hello *world*\n\n- [x] done\n';

check(typeof mdv.version() === 'string' && mdv.version().length > 0, 'version() returns a version');
console.log(`libmdviewer wasm ${mdv.version()}`);

// Render with defaults: full page.
const page = mdv.render(md);
check(page.includes('<html'), 'default render is a full page');
check(page.includes('<h1') && page.includes('Hello'), 'render output has content');

// Fragment option.
const frag = mdv.render(md, { fragment: true });
check(!frag.includes('<html'), 'fragment output has no <html');

// parse -> renderDoc agrees with direct render byte-for-byte.
const doc = mdv.parse(md);
check(doc && doc.version === 1, 'parse returns a version-1 document object');
check(mdv.renderDoc(doc) === page, 'render and parse->renderDoc agree');
check(mdv.renderDoc(JSON.stringify(doc)) === page, 'renderDoc accepts a JSON string');

// Strict options: unknown field, bad version, trailing garbage via string? (object API only)
throws(() => mdv.render(md, { bogus: 1 }), 'bogus', 'unknown option field throws, naming the field');
throws(() => mdv.render(md, { version: 2 }), 'version', 'unsupported options version throws');
throws(() => mdv.parse(md, { resolver: () => null }), 'resolver', 'parse rejects a resolver');

// Assets.
const mermaid = mdv.asset('mermaid.js');
check(mermaid instanceof Uint8Array && mermaid.length > 0, 'asset(mermaid.js) returns bytes');
check(new TextDecoder().decode(mermaid.slice(0, 200_000)).includes('mermaid'), 'mermaid.js looks like mermaid');
const themeCSS = new TextDecoder().decode(mdv.asset('theme-dark.css'));
check(themeCSS.includes('--md-bg'), 'theme-dark.css has theme tokens');
check(themeCSS.includes('.chroma'), 'theme-dark.css includes chroma highlight CSS');
throws(() => mdv.asset('nope.css'), 'valid:', 'unknown asset error lists valid names');

// Resolver: resolve, decline, kinds, trust contract, throw.
const rmd = '![alt](img/photo.png)\n\n[click](docs/guide.md)\n\n[[Wiki Page]]\n';
const seen = [];
const resolved = mdv.render(rmd, {
  fragment: true,
  resolver: (kind, target) => {
    seen.push([kind, target]);
    return kind === 1 ? `asset://${target}` : null;
  },
});
check(resolved.includes('src="asset://img/photo.png"'), 'resolved image URL emitted verbatim');
check(resolved.includes('href="docs/guide.md"'), 'declined link takes default resolution');
check(resolved.includes('Wiki Page.md'), 'declined wiki-link gets default .md resolution');
check(seen.some(([k]) => k === 0) && seen.some(([k]) => k === 1) && seen.some(([k]) => k === 2),
  'resolver saw kinds 0, 1, 2');

const trusted = mdv.render('[e](javascript:x())', {
  fragment: true,
  resolver: () => 'javascript:alert(1)',
});
check(trusted.includes('javascript:alert(1)'), 'resolved URL is trusted verbatim');
const filtered = mdv.render('[e](javascript:x())', { fragment: true });
check(!filtered.includes('javascript:'), 'without resolver, unsafe URL is filtered');

throws(() => mdv.render(rmd, { fragment: true, resolver: () => { throw new Error('host boom'); } }),
  'host boom', 'throwing resolver fails the render with the host error');
throws(() => mdv.render(rmd, { fragment: true, resolver: () => 42 }),
  'resolver', 'non-string resolver return fails the render');

// Singleton.
check(await loadMdviewer() === mdv, 'loadMdviewer is a singleton');

// Non-serializable option values throw instead of being silently dropped.
throws(() => mdv.render(md, { bogusField: undefined }), 'bogusField',
  'undefined option value throws instead of vanishing');
throws(() => mdv.render(md, { onClick: () => {} }), 'onClick',
  'function option value throws instead of vanishing');

// Wiki-link RESOLVE path (kind 2), not just decline.
const wikiResolved = mdv.render('[[Wiki Page]]', {
  fragment: true,
  resolver: (kind, target) => (kind === 2 ? `notes/${target}.html` : null),
});
check(wikiResolved.includes('href="notes/Wiki Page.html"'), 'resolved wiki-link URL emitted verbatim');
check(!wikiResolved.includes('Wiki Page.md'), 'resolved wiki-link does not fall back to .md default');

// Corrupt-wasm rejection + retry-after-failure, in a subprocess (the
// singleton in THIS process already holds a good load).
const sub = spawnSync(process.execPath, ['--input-type=module', '-e', `
  import { loadMdviewer } from ${JSON.stringify(new URL('../../dist/wasm/npm/index.js', import.meta.url).href)};
  const bad = new Uint8Array([0, 97, 115, 109, 1, 0, 0, 0]); // magic+version only
  let rejected = false;
  try { await loadMdviewer(bad); } catch (e) {
    rejected = /wasm start failed|exited before/i.test(String(e.message));
  }
  if (!rejected) { console.error('corrupt wasm did not reject usefully'); process.exit(1); }
  // Retry with the real binary must now succeed (rejection is not cached).
  const mdv = await loadMdviewer();
  if (!mdv.render('# retry', { fragment: true }).includes('<h1')) process.exit(1);
  console.log('subprocess ok');
  process.exit(0);
`], { encoding: 'utf8', timeout: 60_000 });
check(sub.status === 0 && sub.stdout.includes('subprocess ok'),
  `corrupt-wasm rejects + retry succeeds in subprocess [status ${sub.status}: ${(sub.stderr || '').trim()}]`);

if (failures > 0) { console.error(`${failures} failure(s)`); process.exit(1); }
console.log('all checks passed');

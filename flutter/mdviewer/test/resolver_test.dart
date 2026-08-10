import 'package:flutter_test/flutter_test.dart';
import 'package:mdviewer/mdviewer.dart';

// Resolver coverage against the host dylib (mdv_render_r / mdv_render_doc_r),
// ported from examples/node/harness.mjs and examples/c/harness.c's resolver
// sections. A few C-harness cases pin FFI-level contract violations that a
// typed Dart `MdvResolver` (String? Function(MdvResolveKind, String)) simply
// cannot produce from the Dart side — there is no way to make Dart return an
// invalid code, a `1` with no URL, or an oversized length, so those cases
// (modes 2/3/6/7 in the C harness) have no Dart-side counterpart. Likewise
// the JS harness's "non-string resolver return fails the render" test does
// not apply: Dart's type system rejects a non-`String?` return at compile
// time.
void main() {
  final mdv = Mdviewer.instance;

  // Same 3-target document as both native harnesses.
  const rmd =
      '![alt](img/photo.png)\n\n[click](docs/guide.md)\n\n[[Wiki Page]]\n';

  test('resolved image URL emitted verbatim', () {
    final page = mdv.render(
      rmd,
      options: MdvOptions(
        fragment: true,
        resolver: (kind, target) =>
            kind == MdvResolveKind.image ? 'asset://$target' : null,
      ),
    );
    expect(page, contains('src="asset://img/photo.png"'));
  });

  test('declined link keeps default resolution', () {
    final page = mdv.render(
      rmd,
      options: MdvOptions(
        fragment: true,
        resolver: (kind, target) =>
            kind == MdvResolveKind.image ? 'asset://$target' : null,
      ),
    );
    expect(page, contains('href="docs/guide.md"'));
  });

  test('declined wiki-link gets .md default', () {
    final page = mdv.render(
      rmd,
      options: MdvOptions(
        fragment: true,
        resolver: (kind, target) =>
            kind == MdvResolveKind.image ? 'asset://$target' : null,
      ),
    );
    expect(page, contains('Wiki Page.md'));
  });

  test(
    'resolver sees kinds link, image, and wikiLink for the 3-target doc',
    () {
      final seen = <MdvResolveKind>{};
      mdv.render(
        rmd,
        options: MdvOptions(
          fragment: true,
          resolver: (kind, target) {
            seen.add(kind);
            return null;
          },
        ),
      );
      expect(seen, {
        MdvResolveKind.link,
        MdvResolveKind.image,
        MdvResolveKind.wikiLink,
      });
    },
  );

  test('resolved wiki-link URL emitted verbatim, no .md fallback', () {
    final page = mdv.render(
      '[[Wiki Page]]',
      options: MdvOptions(
        fragment: true,
        resolver: (kind, target) =>
            kind == MdvResolveKind.wikiLink ? 'notes/$target.html' : null,
      ),
    );
    expect(page, contains('href="notes/Wiki Page.html"'));
    expect(page, isNot(contains('Wiki Page.md')));
  });

  test(
    'trust contract: resolved URL is emitted verbatim, bypassing safeURL',
    () {
      final page = mdv.render(
        '[e](javascript:x())',
        options: MdvOptions(
          fragment: true,
          resolver: (kind, target) => 'javascript:alert(1)',
        ),
      );
      expect(page, contains('javascript:alert(1)'));
    },
  );

  test('trust contract: without a resolver, unsafe scheme is filtered', () {
    final page = mdv.render(
      '[e](javascript:x())',
      options: const MdvOptions(fragment: true),
    );
    expect(page, isNot(contains('javascript:')));
  });

  test(
    'throwing resolver: render throws MdviewerException with the host '
    'error text, and a subsequent render on the same instance still works',
    () {
      expect(
        () => mdv.render(
          rmd,
          options: MdvOptions(
            fragment: true,
            resolver: (kind, target) => throw StateError('host boom'),
          ),
        ),
        throwsA(
          isA<MdviewerException>().having(
            (e) => e.message,
            'message',
            contains('host boom'),
          ),
        ),
      );

      // The throwing resolver must not have poisoned the instance (leaked
      // NativeCallable, stuck state, etc.) — a normal render right after
      // must succeed exactly as if nothing had happened.
      final page = mdv.render(rmd, options: const MdvOptions(fragment: true));
      expect(page, contains('href="docs/guide.md"'));
    },
  );

  test('empty-string URL resolves to src=""', () {
    final page = mdv.render(
      rmd,
      options: MdvOptions(
        fragment: true,
        resolver: (kind, target) => kind == MdvResolveKind.image ? '' : null,
      ),
    );
    expect(page, contains('src=""'));
  });

  test('resolver works through renderDoc too', () {
    final doc = mdv.parse(rmd);
    final page = mdv.renderDoc(
      doc,
      options: MdvOptions(
        fragment: true,
        resolver: (kind, target) =>
            kind == MdvResolveKind.image ? 'asset://$target' : null,
      ),
    );
    expect(page, contains('src="asset://img/photo.png"'));
    expect(page, contains('href="docs/guide.md"'));
    expect(page, contains('Wiki Page.md'));
  });

  test('a null-returning resolver is byte-identical to no resolver', () {
    final plain = mdv.render(rmd, options: const MdvOptions(fragment: true));
    final viaResolver = mdv.render(
      rmd,
      options: MdvOptions(fragment: true, resolver: (kind, target) => null),
    );
    expect(viaResolver, plain);
  });
}

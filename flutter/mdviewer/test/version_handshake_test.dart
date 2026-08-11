import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:mdviewer/mdviewer.dart';

void main() {
  test('mdviewerPluginVersion matches pubspec.yaml version', () {
    // flutter test runs with the package root as CWD.
    final pubspec = File('pubspec.yaml').readAsStringSync();
    final version = RegExp(
      r'^version:\s*(\S+)\s*$',
      multiLine: true,
    ).firstMatch(pubspec)?[1];
    expect(version, mdviewerPluginVersion);
    // The handshake null-asserts that the plugin's own version parses as a
    // clean X.Y.Z release; a pre-release marker (e.g. `1.0.0-dev`) in the
    // pubspec would turn every handshake into a bare TypeError. Pin it.
    expect(
      RegExp(r'^\d+\.\d+\.\d+$').hasMatch(mdviewerPluginVersion),
      isTrue,
      reason:
          'mdviewerPluginVersion must stay a clean X.Y.Z release string '
          '(the handshake null-asserts this)',
    );
  });

  test('handshake passes against the real dylib on first library call', () {
    // Mdviewer.instance runs the handshake before caching; reaching a
    // successful render proves the loaded library passed it. The host
    // dylib is a source build (git-describe stamp), which the handshake
    // exempts by design; re-running the check on its reported version
    // must stay a no-op.
    expect(Mdviewer.instance.render('# hi'), contains('<h1'));
    expect(
      () => checkVersionHandshake(Mdviewer.instance.version),
      returnsNormally,
    );
  });

  test('clean release with matching major.minor passes', () {
    final parts = mdviewerPluginVersion.split('.');
    final samePatchBumped = '${parts[0]}.${parts[1]}.99';
    expect(() => checkVersionHandshake(mdviewerPluginVersion), returnsNormally);
    expect(() => checkVersionHandshake(samePatchBumped), returnsNormally);
  });

  test('minor skew throws, naming both versions and the fix', () {
    expect(
      () => checkVersionHandshake('0.8.1'),
      throwsA(
        isA<MdviewerException>().having(
          (e) => e.message,
          'message',
          allOf(
            contains('0.8.1'),
            contains(mdviewerPluginVersion),
            contains('fetch_binaries.sh'),
            contains('build-ffi.sh'),
            contains('MDVIEWER_SKIP_VERSION_CHECK'),
          ),
        ),
      ),
    );
  });

  test('major skew throws too', () {
    expect(
      () => checkVersionHandshake('1.9.0'),
      throwsA(isA<MdviewerException>()),
    );
  });

  test('source-build (non-release) versions are exempt', () {
    for (final v in ['0.8.1-6-gae98975', 'dev', 'ae98975', '']) {
      expect(() => checkVersionHandshake(v), returnsNormally, reason: v);
    }
  });

  test('MDVIEWER_SKIP_VERSION_CHECK=1 skips a real mismatch', () {
    expect(
      () => checkVersionHandshake(
        '0.1.0',
        environment: {'MDVIEWER_SKIP_VERSION_CHECK': '1'},
      ),
      returnsNormally,
    );
    // Any other value does not skip.
    expect(
      () => checkVersionHandshake(
        '0.1.0',
        environment: {'MDVIEWER_SKIP_VERSION_CHECK': 'yes'},
      ),
      throwsA(isA<MdviewerException>()),
    );
  });
}

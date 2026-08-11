import 'exceptions.dart';

/// This Dart plugin's own version. MUST stay in sync with `pubspec.yaml`'s
/// `version:` field — pubspec.yaml is not readable at plugin runtime, so
/// the version is mirrored here as a compile-time constant, and a test
/// (`test/version_handshake_test.dart`) asserts the two match.
const mdviewerPluginVersion = '0.10.0';

/// Matches a clean release version (`X.Y.Z`), capturing major and minor.
/// Source builds are stamped by `git describe` (e.g. `0.8.1-6-gae98975`,
/// or a bare commit hash on tagless checkouts) and deliberately do not
/// match — see [checkVersionHandshake].
final _releaseVersion = RegExp(r'^(\d+)\.(\d+)\.(\d+)$');

/// The plugin↔library version handshake, run once on the first library
/// call (from `Mdviewer.instance`) with the loaded library's
/// `mdv_version()` string.
///
/// Throws [MdviewerException] naming both versions when [libraryVersion]
/// is a clean `X.Y.Z` release whose major.minor differs from
/// [mdviewerPluginVersion] — a clear skew diagnosis (stale prebuilt
/// binaries next to a newer plugin, or vice versa) instead of the
/// confusing downstream "unknown field"-style boundary errors skew
/// otherwise produces.
///
/// Two deliberate escapes:
///
/// - **Source builds are exempt.** A [libraryVersion] that is not a clean
///   `X.Y.Z` release (git-describe suffixed like `0.8.1-6-gae98975`,
///   `dev`, or a bare commit hash) was built from source alongside the
///   checkout — its git-describe stamp lags between tag events, so
///   strict matching would reject every pre-tag development build. The
///   handshake targets *release-artifact* skew (`tool/fetch_binaries.sh`
///   downloads), which is always stamped clean `X.Y.Z`.
/// - **`MDVIEWER_SKIP_VERSION_CHECK=1`** in [environment] (the process
///   environment in production) skips the check entirely.
void checkVersionHandshake(
  String libraryVersion, {
  Map<String, String> environment = const {},
}) {
  if (environment['MDVIEWER_SKIP_VERSION_CHECK'] == '1') return;
  final lib = _releaseVersion.firstMatch(libraryVersion);
  if (lib == null) return; // source build: exempt (see doc comment)
  final plugin = _releaseVersion.firstMatch(mdviewerPluginVersion)!;
  if (lib[1] == plugin[1] && lib[2] == plugin[2]) return;
  throw MdviewerException(
    'version skew: the loaded libmdviewer reports $libraryVersion but the '
    'mdviewer Dart plugin is $mdviewerPluginVersion (major.minor must '
    'match). Likely fix: refresh the native binaries — '
    'tool/fetch_binaries.sh (or tool/build_binaries.sh) for the mobile '
    'artifacts, scripts/build-ffi.sh for the host dylib. To bypass at '
    'your own risk, set MDVIEWER_SKIP_VERSION_CHECK=1 in the environment.',
  );
}

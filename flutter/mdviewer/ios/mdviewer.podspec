Pod::Spec.new do |s|
  s.name             = 'mdviewer'
  s.version          = '0.0.1'
  s.summary          = 'libmdviewer FFI binaries for the mdviewer Flutter plugin.'
  s.description      = 'Bundles the libmdviewer XCFramework; the Dart API talks to it over dart:ffi.'
  s.homepage         = 'https://github.com/sriannamalai/markdownviewer'
  s.license          = { :type => 'Apache-2.0', :file => '../LICENSE' }
  s.author           = { 'Sri Annamalai' => 'noreply@example.com' }
  s.source           = { :path => '.' }
  s.platform         = :ios, '13.0'
  s.vendored_frameworks = 'Frameworks/libmdviewer.xcframework'
  s.dependency 'Flutter'
  # CocoaPods builds the Pods project's Debug config with
  # ONLY_ACTIVE_ARCH=NO (unlike the app target, which respects it), so
  # this pod target's own ARCHS is "arm64 x86_64" on an Apple Silicon
  # host even when the running destination is a single arm64 simulator.
  # The xcframework only ships an arm64 simulator slice (build-mobile.sh
  # cross-compiles Go for GOARCH=arm64 only) — CocoaPods' xcframework
  # copy script requires one slice to cover every requested arch
  # together, so without this exclusion it fails to find a match and
  # the link step errors with "Library 'mdviewer' not found".
  s.pod_target_xcconfig = {
    'DEFINES_MODULE' => 'YES',
    'EXCLUDED_ARCHS[sdk=iphonesimulator*]' => 'x86_64',
  }
  # The Dart side resolves this library via DynamicLibrary.process() (see
  # lib/src/library_loader.dart) and looks up C symbols with
  # dlsym(RTLD_DEFAULT, ...) — that only finds symbols that actually made
  # it into the app's linked binary. A plain `-lmdviewer` (which is what
  # CocoaPods generates from vendored_frameworks) only pulls object files
  # out of the static archive that resolve an *existing* undefined
  # reference, and nothing in Runner's Swift/ObjC source references
  # mdv_render et al. by name, so none of libmdviewer.a would actually be
  # linked in and every dlsym lookup would fail at runtime. -force_load
  # pulls in the whole archive unconditionally. This must be
  # `user_target_xcconfig`, not `pod_target_xcconfig`: the latter only
  # affects this pod's own (unused, sourceless) build product, not the
  # consuming app target that actually needs the symbols linked in.
  s.user_target_xcconfig = {
    'OTHER_LDFLAGS' =>
      '$(inherited) -force_load $(PODS_XCFRAMEWORKS_BUILD_DIR)/mdviewer/libmdviewer.a',
  }
end

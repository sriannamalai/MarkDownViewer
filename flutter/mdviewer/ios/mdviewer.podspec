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
  s.pod_target_xcconfig = { 'DEFINES_MODULE' => 'YES' }
end

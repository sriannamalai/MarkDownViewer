/// Error surfaced by the libmdviewer boundary (exact boundary message).
class MdviewerException implements Exception {
  MdviewerException(this.message);
  final String message;
  @override
  String toString() => 'MdviewerException: $message';
}

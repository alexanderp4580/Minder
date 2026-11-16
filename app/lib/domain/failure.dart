import 'entry.dart';

class RepositoryFailure implements Exception {
  const RepositoryFailure(
    this.message, {
    this.unauthorized = false,
    this.replacement,
    this.transport = false,
    this.conflict = false,
  });
  final String message;
  final bool unauthorized;

  /// True when the request never reached the server: no network, a timeout or a dropped stream.
  final bool transport;
  final Entry? replacement;

  /// True when the server answered 409: the item changed since the client read it.
  final bool conflict;
  @override
  String toString() => message;
}

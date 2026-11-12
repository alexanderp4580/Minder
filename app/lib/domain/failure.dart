import 'entry.dart';

class RepositoryFailure implements Exception {
  const RepositoryFailure(
    this.message, {
    this.unauthorized = false,
    this.replacement,
  });
  final String message;
  final bool unauthorized;
  final Entry? replacement;
  @override
  String toString() => message;
}

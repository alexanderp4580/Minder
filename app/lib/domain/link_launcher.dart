/// Opens a link outside the app: a browser, the dialer or a mail app.
abstract interface class LinkLauncher {
  /// Returns false when nothing could open [uri].
  Future<bool> open(Uri uri);
}

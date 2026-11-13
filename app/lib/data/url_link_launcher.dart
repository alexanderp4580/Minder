import 'package:url_launcher/url_launcher.dart';

import '../domain/link_launcher.dart';

class UrlLinkLauncher implements LinkLauncher {
  UrlLinkLauncher({Future<bool> Function(Uri)? launch})
    : _launch = launch ?? _external;
  final Future<bool> Function(Uri) _launch;

  static Future<bool> _external(Uri uri) =>
      launchUrl(uri, mode: LaunchMode.externalApplication);

  @override
  Future<bool> open(Uri uri) async {
    if (!{'https', 'tel', 'mailto'}.contains(uri.scheme)) return false;
    if (uri.scheme == 'https'
        ? uri.host.isEmpty || uri.userInfo.isNotEmpty
        : uri.path.isEmpty) {
      return false;
    }
    try {
      return await _launch(uri);
    } catch (_) {
      return false;
    }
  }
}

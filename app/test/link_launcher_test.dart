import 'package:flutter_test/flutter_test.dart';
import 'package:minder/data/url_link_launcher.dart';

void main() {
  test('rejects unsafe schemes and malformed targets before opening', () async {
    final opened = <Uri>[];
    final launcher = UrlLinkLauncher(
      launch: (uri) async {
        opened.add(uri);
        return true;
      },
    );
    for (final raw in [
      'javascript:alert(1)',
      'file:///private',
      'http://example.com',
      'https:missing-host',
      'tel:',
      'mailto:',
    ]) {
      expect(await launcher.open(Uri.parse(raw)), isFalse);
    }
    expect(opened, isEmpty);
    for (final raw in [
      'https://example.com',
      'tel:+31612345678',
      'mailto:a@example.com',
    ]) {
      expect(await launcher.open(Uri.parse(raw)), isTrue);
    }
    expect(opened.map((u) => u.toString()), [
      'https://example.com',
      'tel:+31612345678',
      'mailto:a@example.com',
    ]);
  });
  test('an unavailable app and a platform exception return false', () async {
    expect(
      await UrlLinkLauncher(launch: (_) async => false)
          .open(Uri.parse('tel:0201234567')),
      isFalse,
    );
    expect(
      await UrlLinkLauncher(launch: (_) async => throw Exception('unavailable'))
          .open(Uri.parse('https://example.com')),
      isFalse,
    );
  });
}

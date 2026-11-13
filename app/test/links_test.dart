import 'package:flutter_test/flutter_test.dart';
import 'package:minder/domain/links.dart';

List<LinkSpan> links(String text) =>
    findLinks(text).where((s) => s.kind != LinkKind.text).toList();

void main() {
  test('plain HTTP links stay text under the approved HTTPS-only rule', () {
    expect(
      findLinks('http://example.com').where((s) => s.target != null),
      isEmpty,
    );
  });
  group('phone numbers', () {
    for (final number in [
      '+31 6 1234 5678',
      '020-1234567',
      '(020) 123 4567',
      '0031201234567',
      '06 12345678',
      '+1 (555) 123-4567',
    ]) {
      test('$number is a phone number', () {
        final found = links('Call $number today');
        expect(found, hasLength(1), reason: number);
        expect(found.single.kind, LinkKind.phone);
        expect(found.single.text, number);
        expect(found.single.target!.scheme, 'tel');
      });
    }

    test('the tel target keeps only digits and a leading plus', () {
      expect(
        links('+31 (0)20 123 4567').single.target.toString(),
        'tel:+310201234567',
      );
    });

    for (final text in [
      'Starts 2026-11-20',
      'Reference ABC123456789',
      'Time 14:30',
      'Room 12345',
      'Booking 123456789',
      'Ticket 9876-5432',
      'Starts 2026-11-20 14:30',
      'Starts 20-11-2026 14:30',
      'Account +31 6 1234 5678 9012 3456',
      'Order 0123456789012345678',
    ]) {
      test('"$text" has no phone number', () {
        expect(links(text), isEmpty);
      });
    }
  });

  test('a number before a colon is still a number', () {
    final found = links('Call 020-1234567: ask for Jan');
    expect(found.single.text, '020-1234567');
  });

  group('email and web links', () {
    test('an email gets a mailto target', () {
      final found = links('Write to info@clinic.nl.');
      expect(found.single.kind, LinkKind.email);
      expect(found.single.text, 'info@clinic.nl');
      expect(found.single.target.toString(), 'mailto:info@clinic.nl');
    });

    test('trailing punctuation stays outside a web link', () {
      expect(links('See https://x.nl/a.').single.text, 'https://x.nl/a');
      expect(
        links('Visit (https://x.nl/a), then').single.text,
        'https://x.nl/a',
      );
      expect(links('Go to www.x.nl, please').single.text, 'www.x.nl');
      expect(
        links('Go to www.x.nl, please').single.target.toString(),
        'https://www.x.nl',
      );
    });

    test('a web link wins over the email and phone inside it', () {
      final found = links('Open https://x.nl/call/0201234567?u=a@b.nl now');
      expect(found, hasLength(1));
      expect(found.single.kind, LinkKind.url);
    });

    test('other schemes never become links', () {
      expect(links('javascript:alert(1)'), isEmpty);
      expect(links('file:///etc/passwd'), isEmpty);
      expect(links('data:text/html,hi'), isEmpty);
    });
  });

  test('the spans rebuild the original text', () {
    const text =
        'Bring ID. Call 020-1234567 or mail a@b.nl, see https://x.nl/a. Done.';
    expect(findLinks(text).map((s) => s.text).join(), text);
    expect(links(text).map((s) => s.kind), [
      LinkKind.phone,
      LinkKind.email,
      LinkKind.url,
    ]);
  });

  test('empty text gives no spans', () {
    expect(findLinks(''), isEmpty);
  });

  test('the maps link encodes the address', () {
    final uri = mapsUri('Damrak 1 & Co, Zürich');
    expect(uri.scheme, 'https');
    expect(uri.host, 'www.google.com');
    expect(uri.path, '/maps/search/');
    expect(uri.queryParameters['api'], '1');
    expect(uri.queryParameters['query'], 'Damrak 1 & Co, Zürich');
    expect(uri.toString(), contains('Damrak%201%20%26%20Co'));
  });
}

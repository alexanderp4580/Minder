/// What a piece of text is: plain, or something a person can tap.
enum LinkKind { text, phone, email, url }

class LinkSpan {
  const LinkSpan(this.kind, this.text, [this.target]);
  final LinkKind kind;
  final String text;

  /// Where a tap goes. Null for plain text. Only http, https, tel and mailto appear here.
  final Uri? target;
}

final _url = RegExp(r'''(?:https?://|www\.)[^\s<>"']+''', caseSensitive: false);
final _email = RegExp(
  r'[A-Za-z0-9._%+\-]+@[A-Za-z0-9\-]+(?:\.[A-Za-z0-9\-]+)*\.[A-Za-z]{2,}',
);
final _phone = RegExp(r'\+?\(?\d[\d \t().\-]{5,}\d');
final _isoDate = RegExp(r'\d{4}-\d{2}-\d{2}');

const _endings = '.,;:!?\'\"';

bool _alnum(String c) => RegExp(r'[A-Za-z0-9]').hasMatch(c);

/// Splits [text] into plain, phone, email and web spans. Joining the span texts gives
/// [text] back. Web links win over emails, and emails over phone numbers.
List<LinkSpan> findLinks(String text) {
  final found = <({int start, int end, LinkSpan span})>[];
  bool free(int start, int end) =>
      found.every((f) => end <= f.start || start >= f.end);

  for (final m in _url.allMatches(text)) {
    final raw = _trimLink(m.group(0)!);
    final target = Uri.tryParse(
      raw.toLowerCase().startsWith('www.') ? 'https://$raw' : raw,
    );
    if (target == null || !target.hasAuthority || target.host.isEmpty) continue;
    if (target.scheme != 'http' && target.scheme != 'https') continue;
    found.add((
      start: m.start,
      end: m.start + raw.length,
      span: LinkSpan(LinkKind.url, raw, target),
    ));
  }
  for (final m in _email.allMatches(text)) {
    if (!free(m.start, m.end)) continue;
    final address = m.group(0)!;
    found.add((
      start: m.start,
      end: m.end,
      span: LinkSpan(
        LinkKind.email,
        address,
        Uri(scheme: 'mailto', path: address),
      ),
    ));
  }
  for (final m in _phone.allMatches(text)) {
    final number = m.group(0)!;
    if (!free(m.start, m.end) || !_isPhone(text, m.start, m.end, number)) {
      continue;
    }
    final digits = number.replaceAll(RegExp(r'\D'), '');
    found.add((
      start: m.start,
      end: m.end,
      span: LinkSpan(
        LinkKind.phone,
        number,
        Uri(scheme: 'tel', path: '${number.startsWith('+') ? '+' : ''}$digits'),
      ),
    ));
  }

  found.sort((a, b) => a.start.compareTo(b.start));
  final spans = <LinkSpan>[];
  var at = 0;
  for (final f in found) {
    if (f.start > at) {
      spans.add(LinkSpan(LinkKind.text, text.substring(at, f.start)));
    }
    spans.add(f.span);
    at = f.end;
  }
  if (at < text.length) spans.add(LinkSpan(LinkKind.text, text.substring(at)));
  return spans;
}

/// Drops the punctuation that ends a sentence, and a bracket the link did not open.
String _trimLink(String raw) {
  var link = raw;
  while (link.isNotEmpty) {
    final last = link[link.length - 1];
    final closesUnopened =
        (last == ')' &&
            ')'.allMatches(link).length > '('.allMatches(link).length) ||
        (last == ']' &&
            ']'.allMatches(link).length > '['.allMatches(link).length);
    if (_endings.contains(last) || closesUnopened) {
      link = link.substring(0, link.length - 1);
    } else {
      break;
    }
  }
  return link;
}

/// A phone number has 7 to 15 digits and is not glued to letters or digits. It starts
/// with +, 0 or a bracket, or has separators and at least 10 digits. Dates are not numbers.
bool _isPhone(String text, int start, int end, String number) {
  if (start > 0 && (_alnum(text[start - 1]) || text[start - 1] == ':')) {
    return false;
  }
  if (end < text.length) {
    final next = text[end];
    if (_alnum(next)) return false;
    if (next == ':' && end + 1 < text.length && _alnum(text[end + 1])) {
      return false;
    }
  }
  if (_isoDate.hasMatch(number)) return false;
  final digits = number.replaceAll(RegExp(r'\D'), '').length;
  if (digits < 7 || digits > 15) return false;
  final startsLikeANumber =
      number.startsWith('+') ||
      number.startsWith('0') ||
      number.startsWith('(');
  final separated = RegExp(r'[ .()\-]').hasMatch(number);
  return startsLikeANumber || (separated && digits >= 10);
}

/// A Google Maps search for [address].
Uri mapsUri(String address) => Uri.parse(
  'https://www.google.com/maps/search/?api=1&query=${Uri.encodeComponent(address)}',
);

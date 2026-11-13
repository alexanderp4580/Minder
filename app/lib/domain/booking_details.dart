import 'links.dart';

/// Read-only compatibility for older items with explicitly labelled note fields.
class BookingDetails {
  const BookingDetails({
    this.phone = '',
    this.email = '',
    this.website = '',
    this.reference = '',
    this.notes = '',
  });
  final String phone, email, website, reference, notes;

  factory BookingDetails.fromNotes({
    String phone = '',
    String email = '',
    String website = '',
    String reference = '',
    String notes = '',
  }) {
    final values = {
      'phone': phone,
      'email': email,
      'website': website,
      'reference': reference,
    };
    final remaining = <String>[];
    for (final raw in notes.split('\n')) {
      final line = raw.trim().replaceFirst(RegExp(r'^[-*•]\s*'), '');
      var match = RegExp(r'^([^:=]+)\s*[:=]\s*(.+)$').firstMatch(line);
      final spaced = match == null;
      match ??= RegExp(
        r'^(Phone|Tel\.?|Telephone|Call|Booking reference|Booking ref\.?|Confirmation number|Reservation number)\s+(.+)$',
        caseSensitive: false,
      ).firstMatch(line);
      var key = match == null ? null : _labels[match[1]!.trim().toLowerCase()];
      final value = match?[2]?.trim() ?? '';
      if (match != null && match[1]!.trim().toLowerCase() == 'contact') {
        final links = findLinks(value).where((s) => s.target != null).toList();
        if (links.length == 1 && links.single.text == value) {
          key = switch (links.single.kind) {
            LinkKind.phone => 'phone',
            LinkKind.email => 'email',
            LinkKind.url => 'website',
            _ => null,
          };
        }
      }
      if (spaced &&
          key == 'phone' &&
          detailTarget(value, LinkKind.phone) == null) {
        remaining.add(raw);
        continue;
      }
      if (key != null && (values[key]!.isEmpty || values[key] == value)) {
        values[key] = value;
      } else {
        remaining.add(raw);
      }
    }
    return BookingDetails(
      phone: values['phone']!,
      email: values['email']!,
      website: values['website']!,
      reference: values['reference']!,
      notes: remaining.join('\n').trim(),
    );
  }
}

// Bare Ref/Reference labels can be calendar IDs, so they stay in Notes.
const _labels = {
  'phone': 'phone',
  'call': 'phone',
  'contact phone': 'phone',
  'phone number': 'phone',
  'telephone': 'phone',
  'tel': 'phone',
  'tel.': 'phone',
  'telefoon': 'phone',
  'telefoonnummer': 'phone',
  'email': 'email',
  'e-mail': 'email',
  'contact email': 'email',
  'email address': 'email',
  'e-mailadres': 'email',
  'website': 'website',
  'web': 'website',
  'url': 'website',
  'booking reference': 'reference',
  'booking ref': 'reference',
  'booking ref.': 'reference',
  'booking number': 'reference',
  'confirmation number': 'reference',
  'confirmation code': 'reference',
  'reservation number': 'reference',
  'reservation reference': 'reference',
  'boekingsnummer': 'reference',
  'reserveringsnummer': 'reference',
};

Uri? detailTarget(String value, LinkKind kind) {
  final text = kind == LinkKind.phone ? 'Phone:$value' : value;
  for (final span in findLinks(text)) {
    if (span.kind == kind &&
        value.startsWith(span.text) &&
        RegExp(r'^[.,;!?)]*$').hasMatch(value.substring(span.text.length))) {
      return span.target;
    }
  }
  return null;
}

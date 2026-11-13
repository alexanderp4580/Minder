import 'package:flutter_test/flutter_test.dart';
import 'package:minder/domain/booking_details.dart';
import 'package:minder/domain/links.dart';

void main() {
  test('explicit fields keep conflicting older values visible', () {
    final details = BookingDetails.fromNotes(
      phone: '020-1111111',
      notes: '- Phone: 020-2222222\n- Bring ID',
    );
    expect(details.phone, '020-1111111');
    expect(details.notes, '- Phone: 020-2222222\n- Bring ID');
  });
  test('calendar UID refs and conditional call instructions stay in Notes', () {
    final details = BookingDetails.fromNotes(
      notes: '- Ref: UID hotel-123 sequence 2\n- Call 020-1234567 if late',
    );
    expect(details.reference, '');
    expect(details.phone, '');
    expect(
      details.notes,
      '- Ref: UID hotel-123 sequence 2\n- Call 020-1234567 if late',
    );
  });
  test('booking reference keeps leading zeros beyond the note display cap', () {
    final details = BookingDetails.fromNotes(
      notes: '${'x' * 700}\n- Booking reference: 00123456789',
    );
    expect(details.reference, '00123456789');
    expect(details.notes, 'x' * 700);
  });
  test(
    'typed phone target supports local numbers and sentence punctuation',
    () {
      expect(detailTarget('1234567', LinkKind.phone).toString(), 'tel:1234567');
      expect(
        detailTarget('020-1234567.', LinkKind.phone).toString(),
        'tel:0201234567',
      );
      expect(detailTarget('Reference 00123456789', LinkKind.phone), isNull);
    },
  );
}

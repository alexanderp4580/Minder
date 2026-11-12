import 'package:flutter_test/flutter_test.dart';
import 'package:minder/application/calendar_controller.dart';
import 'package:minder/domain/item.dart';
import 'package:minder/domain/item_repository.dart';
import 'package:minder/data/wire.dart';

class CalendarItems implements ItemRepository {
  CalendarItems(this.items);
  final List<Item> items;
  @override
  Future<List<Item>> itemsBetween(DateTime from, DateTime to) async => items;
}

void main() {
  test('hotel stays occupy nights and exclude checkout', () async {
    final item = itemFromJson({
      'id': 'hotel',
      'title': 'Hotel',
      'status': 'open',
      'kind': 'event',
      'start_date': '2026-11-01',
      'end_date': '2026-11-15',
    });
    final controller = CalendarController(
      CalendarItems([item]),
      now: () => DateTime(2026, 11, 2),
    );
    await controller.load();
    expect(controller.itemsOn(DateTime(2026, 11, 2)).single.title, 'Hotel');
    expect(controller.itemsOn(DateTime(2026, 11, 14)), hasLength(1));
    expect(controller.itemsOn(DateTime(2026, 11, 15)), isEmpty);
    controller.dispose();
  });
  test('capture wire retains questions, evidence and original attachment', () {
    final capture = captureFromJson({
      'id': 'source',
      'status': 'needs-answer',
      'revision': 2,
      'sender': 'owner@example.com',
      'subject': 'Booking',
      'questions': ['What date?'],
      'attachments': [
        {
          'id': 'original',
          'name': 'original.eml',
          'media_type': 'message/rfc822',
          'size': 40,
        },
      ],
      'candidates': [
        {
          'id': 'candidate',
          'revision': 2,
          'state': 'needs-answer',
          'question': 'What date?',
          'proposal': {'title': 'Flight'},
          'evidence': [
            {'part': '1', 'excerpt': 'Flight'},
          ],
        },
      ],
    });
    expect(capture.questions.single, 'What date?');
    expect(capture.attachments.single.name, 'original.eml');
    expect(capture.candidates.single.evidence.single, '1: Flight');
  });
}

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:minder/data/wire.dart';
import 'package:minder/domain/entry.dart';
import 'package:minder/domain/link_launcher.dart';
import 'package:minder/ui/calendar_item_details.dart';
import 'package:minder/ui/format.dart';
import 'package:minder/ui/linked_text.dart';

class RecordingLauncher implements LinkLauncher {
  Uri? opened;
  @override
  Future<bool> open(Uri uri) async {
    opened = uri;
    return true;
  }
}

void main() {
  testWidgets(
    'task details distinguish scheduled work, deadline and explicit notification',
    (tester) async {
      final item = itemFromJson({
        'id': 'task',
        'title': 'Questionnaire',
        'status': 'open',
        'kind': 'task',
        'start_at': '2026-11-23T12:00:00Z',
        'due_at': '2026-11-26T12:00:00Z',
        'remind_at': '2026-11-23T12:00:00Z',
      });
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(body: CalendarItemDetails(item: item)),
        ),
      );
      expect(find.text('Scheduled'), findsOneWidget);
      expect(find.text('Deadline'), findsOneWidget);
      expect(find.text('Notification'), findsOneWidget);
      expect(item.placedAt, DateTime.parse('2026-11-23T12:00:00Z').toLocal());
    },
  );
  testWidgets('detail pairs survive the wire and render with web links', (
    tester,
  ) async {
    final launcher = RecordingLauncher();
    final item = itemFromJson({
      'id': 'x',
      'title': 'Follow-up',
      'status': 'open',
      'details': [
        {'label': 'Meeting link', 'value': 'https://teams.example/join'},
        {'label': 'Dress code', 'value': 'Casual'},
      ],
    });
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: CalendarItemDetails(item: item, launcher: launcher),
        ),
      ),
    );
    expect(find.text('Meeting link'), findsOneWidget);
    expect(find.text('Dress code'), findsOneWidget);
    expect(
      find.byWidgetPredicate(
        (w) => w is LinkedText && w.text == 'https://teams.example/join',
      ),
      findsOneWidget,
    );
    expect(find.text('Website'), findsNothing);
    await tester.tap(
      find.text('https://teams.example/join', findRichText: true),
    );
    await tester.pump();
    expect(launcher.opened, Uri.parse('https://teams.example/join'));
    expect(
      itemFromJson({'id': 'old', 'title': 'Old', 'status': 'open'}).details,
      isEmpty,
    );
  });

  test('preview details retain literal values and label', () {
    final rows = previewRows(
      Change(
        id: 'p',
        action: 'create_item',
        itemTitle: 'x',
        state: ChangeState.pending,
        fields: const [
          FieldChange(
            name: 'details',
            oldValue: '',
            newValue:
                'Meeting link: https://teams.example/join\nDress code: Casual',
          ),
        ],
      ),
    );
    expect(rows.single.label, 'Details');
    expect(
      rows.single.newValue,
      'Meeting link: https://teams.example/join\nDress code: Casual',
    );
  });
}

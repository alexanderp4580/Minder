import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:minder/application/chat_controller.dart';
import 'package:minder/domain/chat_repository.dart';
import 'package:minder/domain/entry.dart';
import 'package:minder/ui/entry_view.dart';
import 'package:minder/ui/format.dart';
import 'package:minder/ui/tokens.dart';

class RecordingRepository extends Fake implements ChatRepository {
  final reminderCalls = <List<int>>[];
  @override
  Future<List<Entry>> history(int after) async => [];
  @override
  Stream<Entry> watch(int after) => const Stream.empty();
  @override
  Future<Entry> setChangeReminders(String changeId, List<int> offsets) async {
    reminderCalls.add(offsets);
    return _entry(
      Change(
        id: 'p',
        action: 'create_item',
        itemTitle: 'Fitting',
        fields: const [],
        state: ChangeState.pending,
        reminderOffsets: offsets,
        reminderChoices: _choices,
      ),
    );
  }
}

const _choices = [10, 30, 60, 120, 1440, 2880, 4320, 10080];

Entry _entry(Change change) => Entry(
  id: 'entry',
  cursor: 1,
  kind: EntryKind.preview,
  text: 'Review this change.',
  createdAt: DateTime.utc(2026, 11, 13, 9),
  change: change,
);

Change _change(
  List<FieldChange> fields, {
  ChangeState state = ChangeState.pending,
  List<int> offsets = const [],
  List<int> choices = const [],
}) => Change(
  id: 'p',
  action: 'create_item',
  itemTitle: 'Fitting',
  fields: fields,
  state: state,
  reminderOffsets: offsets,
  reminderChoices: choices,
);

FieldChange _f(String name, String value, [String old = '']) =>
    FieldChange(name: name, oldValue: old, newValue: value);

Future<RecordingRepository> _show(WidgetTester tester, Change change) async {
  final repository = RecordingRepository();
  final controller = ChatController(
    repository,
    pollInterval: const Duration(days: 1),
  );
  addTearDown(controller.dispose);
  await tester.pumpWidget(
    MaterialApp(
      theme: Tokens.theme,
      home: Scaffold(
        body: SingleChildScrollView(
          child: EntryView(entry: _entry(change), controller: controller),
        ),
      ),
    ),
  );
  return repository;
}

void main() {
  test('formatters read naturally', () {
    expect(durationText(45), '45 minutes');
    expect(durationText(1), '1 minute');
    expect(durationText(60), '1 hour');
    expect(durationText(90), '1 hour 30 minutes');
    expect(durationText(120), '2 hours');
    expect(durationText(1440), '1 day');
    expect(offsetLabel(10), '10 minutes');
    expect(offsetLabel(120), '2 hours');
    expect(offsetLabel(2880), '2 days');
    expect(offsetLabel(10080), '1 week');
    expect(kindLabel('event'), 'Appointment');
    expect(kindLabel('task'), 'Task');
    expect(kindLabel('reminder'), 'Reminder');
    expect(noteLines('- Bring glasses\n* Ref: AB12\n\nplain line'), [
      'Bring glasses',
      'Ref: AB12',
      'plain line',
    ]);
    final capped = noteLines('- ${'x' * 5000}');
    expect(capped.join().length, lessThanOrEqualTo(601));
    expect(capped.last.endsWith('…'), isTrue);
  });

  testWidgets('previews show the title once', (tester) async {
    await _show(tester, _change([_f('title', 'Fitting'), _f('kind', 'event')]));
    expect(find.text('Fitting'), findsOneWidget);
    expect(find.text('Title:'), findsNothing);
  });

  testWidgets('starts and ends carry the timezone', (tester) async {
    await _show(
      tester,
      _change([
        _f('start_at', '2026-11-20T14:30:00+01:00'),
        _f('start_timezone', 'Europe/Amsterdam'),
        _f('end_at', '2026-11-20T09:15:00-05:00'),
        _f('end_timezone', 'America/New_York'),
      ]),
    );
    expect(find.text('Fri 20 Nov 14:30 (Europe/Amsterdam)'), findsOneWidget);
    expect(find.text('Fri 20 Nov 09:15 (America/New_York)'), findsOneWidget);
    expect(find.textContaining('timezone'), findsNothing);
  });

  testWidgets('end falls back to the start timezone and item timezone', (
    tester,
  ) async {
    await _show(
      tester,
      _change([
        _f('start_at', '2026-11-20T14:30:00+01:00'),
        _f('end_at', '2026-11-20T15:15:00+01:00'),
        _f('timezone', 'Europe/Amsterdam'),
      ]),
    );
    expect(find.text('Fri 20 Nov 14:30 (Europe/Amsterdam)'), findsOneWidget);
    expect(find.text('Fri 20 Nov 15:15 (Europe/Amsterdam)'), findsOneWidget);
    expect(find.text('Timezone:'), findsNothing);
  });

  testWidgets('duration reads with its unit', (tester) async {
    await _show(
      tester,
      _change([_f('duration_minutes', '45'), _f('kind', 'event')]),
    );
    expect(find.text('Duration:'), findsOneWidget);
    expect(find.text('45 minutes'), findsOneWidget);
  });

  testWidgets('address and appointment labels', (tester) async {
    await _show(
      tester,
      _change([
        _f('location', 'Kinkerstraat 123, Amsterdam'),
        _f('kind', 'event'),
      ]),
    );
    expect(find.text('Address:'), findsOneWidget);
    expect(find.text('Kinkerstraat 123, Amsterdam'), findsOneWidget);
    expect(find.text('Type:'), findsOneWidget);
    expect(find.text('Appointment'), findsOneWidget);
    expect(find.text('Location:'), findsNothing);
  });

  testWidgets('notes render as an inert bullet list', (tester) async {
    await _show(
      tester,
      _change([
        _f(
          'notes',
          '- Bring glasses\n- [link](http://x.example)\n- <b>bold</b>',
        ),
      ]),
    );
    expect(find.text('Notes:'), findsOneWidget);
    expect(find.text('Bring glasses'), findsOneWidget);
    expect(find.text('[link](http://x.example)'), findsOneWidget);
    expect(find.text('<b>bold</b>'), findsOneWidget);
    expect(find.text('•'), findsNWidgets(3));
  });

  testWidgets('reminder chips toggle and send the new offsets', (tester) async {
    final repository = await _show(
      tester,
      _change(
        [_f('reminders', '1 day, 1 hour')],
        offsets: const [1440, 60],
        choices: _choices,
      ),
    );
    expect(find.text('Reminders:'), findsNothing);
    await tester.tap(find.text('10 minutes'));
    await tester.pumpAndSettle();
    expect(repository.reminderCalls.single, [1440, 60, 10]);
    await tester.tap(find.text('Clear'));
    await tester.pumpAndSettle();
    expect(repository.reminderCalls.last, isEmpty);
    expect(
      tester.getSize(find.text('1 week').hitTestable().first).height,
      greaterThan(0),
    );
  });

  testWidgets('chips are disabled once confirmed', (tester) async {
    final repository = await _show(
      tester,
      _change(
        const [],
        state: ChangeState.confirmed,
        offsets: const [1440],
        choices: _choices,
      ),
    );
    await tester.tap(find.text('10 minutes'));
    await tester.pumpAndSettle();
    expect(repository.reminderCalls, isEmpty);
  });
}

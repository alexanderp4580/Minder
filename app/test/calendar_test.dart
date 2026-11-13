import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:minder/application/calendar_controller.dart';
import 'package:minder/data/wire.dart';
import 'package:minder/domain/entry.dart';
import 'package:minder/domain/failure.dart';
import 'package:minder/domain/item.dart';
import 'package:minder/domain/item_repository.dart';
import 'package:minder/domain/link_launcher.dart';
import 'package:minder/domain/links.dart';
import 'package:minder/ui/calendar_screen.dart';
import 'package:minder/ui/tokens.dart';

String utc(int y, int m, int d, [int h = 0, int min = 0]) =>
    DateTime(y, m, d, h, min).toUtc().toIso8601String();

Item item(
  String id,
  String title, {
  String? start,
  String? due,
  String status = 'open',
  int duration = 0,
  int priority = 0,
  String location = '',
}) => itemFromJson({
  'id': id,
  'title': title,
  'status': status,
  'start_at': ?start,
  'due_at': ?due,
  'duration_minutes': duration,
  'priority': priority,
  'location': location,
});

Entry deletePreview(Item item) => entryFromJson({
  'id': 'e-${item.id}',
  'cursor': 1,
  'kind': 'preview',
  'text': 'Delete this item and its data.',
  'created_at': '2026-11-12T10:00:00Z',
  'change': {
    'id': 'c-${item.id}',
    'action': 'delete_item',
    'item_title': item.title,
    'state': 'pending',
    'fields': [
      {'name': 'item', 'old': item.title, 'new': 'Deleted'},
    ],
  },
});

class FakeLauncher implements LinkLauncher {
  final opened = <Uri>[];
  @override
  Future<bool> open(Uri uri) async {
    opened.add(uri);
    return true;
  }
}

Item appointment({
  String endZone = 'Europe/Amsterdam',
  String location = '',
  String notes = '',
  List<int> offsets = const [],
}) => itemFromJson({
  'id': 'a',
  'title': 'Dentist',
  'status': 'open',
  'kind': 'event',
  'start_at': utc(2026, 11, 12, 9),
  'start_local': '2026-11-12T09:00',
  'start_timezone': 'Europe/Amsterdam',
  'end_at': utc(2026, 11, 12, 9, 30),
  'end_local': '2026-11-12T09:30',
  'end_timezone': endZone,
  'duration_minutes': 30,
  'location': location,
  'notes': notes,
  'reminder_offsets_minutes': offsets,
});

class FakeItems implements ItemRepository {
  final requests = <(DateTime, DateTime)>[];
  final confirmed = <String>[], discarded = <String>[];
  List<Item> items = [];
  bool offline = false;
  @override
  Future<Entry> previewDelete(Item item) async => deletePreview(item);
  @override
  Future<Entry> confirm(String changeId) async {
    confirmed.add(changeId);
    items.removeWhere((i) => 'c-${i.id}' == changeId);
    return deletePreview(item('x', 'x'));
  }

  @override
  Future<Entry> discard(String changeId) async {
    discarded.add(changeId);
    return deletePreview(item('x', 'x'));
  }

  @override
  Future<List<Item>> itemsBetween(DateTime from, DateTime to) async {
    requests.add((from, to));
    if (offline) throw const RepositoryFailure('Check your connection.');
    return items.where((i) {
      final at = i.placedAt;
      return at != null && !at.isBefore(from) && at.isBefore(to);
    }).toList();
  }
}

CalendarController controller(FakeItems repo) =>
    CalendarController(repo, now: () => DateTime(2026, 11, 12, 10));

class PendingItems extends Fake implements ItemRepository {
  final requests = <Completer<List<Item>>>[];
  @override
  Future<List<Item>> itemsBetween(DateTime from, DateTime to) {
    final result = Completer<List<Item>>();
    requests.add(result);
    return result.future;
  }
}

void main() {
  setUp(() {
    TestWidgetsFlutterBinding.ensureInitialized()
        .platformDispatcher
        .accessibilityFeaturesTestValue = const FakeAccessibilityFeatures(
      disableAnimations: true,
    );
  });
  tearDown(() {
    TestWidgetsFlutterBinding.instance.platformDispatcher
        .clearAccessibilityFeaturesTestValue();
  });
  setUpAll(() async {
    TestWidgetsFlutterBinding.ensureInitialized();
    final icons = FontLoader('MaterialIcons')
      ..addFont(rootBundle.load('fonts/MaterialIcons-Regular.otf'));
    await icons.load();
    for (final name in ['Michroma', 'Chakra Petch']) {
      final loader = FontLoader(name);
      for (final file
          in name == 'Michroma'
              ? ['Michroma-Regular.ttf']
              : [
                  'ChakraPetch-Regular.ttf',
                  'ChakraPetch-Medium.ttf',
                  'ChakraPetch-SemiBold.ttf',
                  'ChakraPetch-Bold.ttf',
                ]) {
        loader.addFont(rootBundle.load('assets/fonts/$file'));
      }
      await loader.load();
    }
  });

  test('an item keeps its notes, kind and reminders', () {
    final full = itemFromJson({
      'id': 'a',
      'title': 'Dentist',
      'status': 'open',
      'kind': 'event',
      'notes': '- Bring ID\n- Call 020-1234567',
      'reminder_offsets_minutes': [1440, 60],
    });
    expect(full.kind, 'event');
    expect(full.notes, '- Bring ID\n- Call 020-1234567');
    expect(full.reminderOffsets, [1440, 60]);
    final bare = itemFromJson({'id': 'b', 'title': 'x', 'status': 'open'});
    expect(bare.kind, '');
    expect(bare.notes, '');
    expect(bare.reminderOffsets, isEmpty);
  });

  test('load asks for six weeks from the Monday before the month', () async {
    final repo = FakeItems();
    final c = controller(repo);
    await c.load();
    expect(c.gridStart, DateTime(2026, 10, 26));
    expect(repo.requests.single.$1, DateTime(2026, 10, 26));
    expect(repo.requests.single.$2, DateTime(2026, 12, 7));
    expect(c.selected, DateTime(2026, 11, 12));
    c.dispose();
  });

  test('items sit on their start day, else their deadline day', () async {
    final repo = FakeItems()
      ..items = [
        item(
          'a',
          'Dentist',
          start: utc(2026, 11, 13, 9),
          due: utc(2026, 11, 30, 9),
        ),
        item('b', 'Taxes', due: utc(2026, 11, 13, 8)),
        item('c', 'Other', due: utc(2026, 11, 20, 8)),
      ];
    final c = controller(repo);
    await c.load();
    c.select(DateTime(2026, 11, 13));
    expect(c.selectedItems.map((i) => i.title), ['Taxes', 'Dentist']);
    expect(c.countOn(DateTime(2026, 11, 20)), 1);
    expect(c.countOn(DateTime(2026, 11, 30)), 0);
    c.dispose();
  });

  test('moving the month reloads and a failure keeps the last items', () async {
    final repo = FakeItems()
      ..items = [item('a', 'Dentist', start: utc(2026, 11, 13, 9))];
    final c = controller(repo);
    await c.load();
    await c.next();
    expect(c.month, DateTime(2026, 12));
    expect(repo.requests.length, 2);
    await c.previous();
    repo.offline = true;
    await c.load();
    expect(c.error, 'Check your connection.');
    expect(c.countOn(DateTime(2026, 11, 13)), 1);
    c.dispose();
  });

  test('a slow earlier month cannot replace the latest month', () async {
    final repo = PendingItems();
    final c = CalendarController(repo, now: () => DateTime(2026, 11, 12));
    final november = c.load();
    final december = c.next();
    repo.requests[1].complete([
      item('dec', 'December', start: utc(2026, 12, 13)),
    ]);
    await december;
    repo.requests[0].complete([
      item('nov', 'November', start: utc(2026, 11, 13)),
    ]);
    await november;
    expect(c.month, DateTime(2026, 12));
    expect(c.itemsOn(DateTime(2026, 12, 13)).map((i) => i.id), ['dec']);
    expect(c.loading, false);
    c.dispose();
  });

  test('an earlier failure cannot end a newer load or set its error', () async {
    final repo = PendingItems();
    final c = CalendarController(repo);
    final first = c.load();
    final latest = c.next();
    repo.requests[0].completeError(const RepositoryFailure('Old failure'));
    await first;
    expect(c.loading, true);
    expect(c.error, isNull);
    repo.requests[1].complete([]);
    await latest;
    expect(c.loading, false);
    expect(c.error, isNull);
    c.dispose();
  });

  for (final width in [320.0, 360.0]) {
    testWidgets(
      'a ${width.toInt()} px calendar scrolls its agenda without clipping enlarged text',
      (tester) async {
        tester.view.physicalSize = Size(width, 640);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.reset);
        final repo = FakeItems()
          ..items = [
            item(
              'long',
              'Dentist appointment with Dr Smit',
              start: utc(2026, 11, 12, 9),
              location: 'Damrak 1, Amsterdam',
            ),
          ];
        final c = controller(repo);
        await c.load();
        await tester.pumpWidget(
          MaterialApp(
            theme: Tokens.theme,
            builder: (context, child) => MediaQuery(
              data: MediaQuery.of(context)
                  .copyWith(textScaler: const TextScaler.linear(1.4)),
              child: child!,
            ),
            home: CalendarScreen(controller: c),
          ),
        );
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
        final dayTarget = tester.getRect(
          find.byKey(const ValueKey('day-2026-11-12')),
        );
        expect(dayTarget.width, greaterThanOrEqualTo(44));
        await tester.ensureVisible(
          find.text('Dentist appointment with Dr Smit'),
        );
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
        final title = tester.getRect(
          find.text('Dentist appointment with Dr Smit'),
        );
        expect(title.left, greaterThanOrEqualTo(0));
        expect(title.right, lessThanOrEqualTo(width));
        expect(title.top, greaterThanOrEqualTo(0));
        expect(title.bottom, lessThanOrEqualTo(640));
        await tester.ensureVisible(
          find.widgetWithText(OutlinedButton, 'Today'),
        );
        await tester.tap(find.widgetWithText(OutlinedButton, 'Today'));
        await tester.pumpAndSettle();
        expect(c.selected, DateTime(2026, 11, 12));
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox());
        c.dispose();
      },
    );
  }

  testWidgets('screen shows the month, markers and the day agenda', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(412, 892);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final repo = FakeItems()
      ..items = [
        item(
          'a',
          'Dentist',
          start: utc(2026, 11, 13, 9),
          duration: 30,
          priority: 2,
          location: 'Damrak 1',
        ),
        item('b', 'Taxes', due: utc(2026, 11, 13, 17)),
        item('c', 'Old task', due: utc(2026, 11, 13, 12), status: 'done'),
      ];
    final c = controller(repo);
    await c.load();
    await tester.pumpWidget(
      MaterialApp(
        theme: Tokens.theme,
        home: CalendarScreen(controller: c),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('November'), findsOneWidget);
    expect(find.text('Nothing planned.'), findsOneWidget);
    await tester.tap(find.byKey(const ValueKey('day-2026-11-13')));
    await tester.pumpAndSettle();
    expect(find.text('Dentist'), findsOneWidget);
    expect(find.text('09:00'), findsOneWidget);
    expect(find.text('30 min'), findsOneWidget);
    expect(find.text('P2'), findsOneWidget);
    expect(find.text('Damrak 1'), findsOneWidget);
    expect(find.text('Due 17:00'), findsOneWidget);
    await expectLater(
      find.byType(CalendarScreen),
      matchesGoldenFile('goldens/calendar.png'),
    );
    await tester.tap(find.byTooltip('Next month'));
    await tester.pumpAndSettle();
    expect(find.text('December'), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
    c.dispose();
  });

  group('Delete', () {
    Future<(FakeItems, CalendarController)> open(WidgetTester tester) async {
      tester.view.physicalSize = const Size(412, 892);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      final repo = FakeItems()
        ..items = [item('a', 'Dentist', start: utc(2026, 11, 12, 9))];
      final c = controller(repo);
      await c.load();
      await tester.pumpWidget(
        MaterialApp(
          theme: Tokens.theme,
          home: CalendarScreen(controller: c),
        ),
      );
      await tester.pumpAndSettle();
      return (repo, c);
    }

    testWidgets('has a 44 px target on each agenda card', (tester) async {
      final (_, c) = await open(tester);
      expect(tester.getSize(find.byTooltip('Delete')).height, 44);
      expect(tester.getSize(find.byTooltip('Delete')).width, 44);
      await tester.pumpWidget(const SizedBox());
      c.dispose();
    });

    testWidgets('shows what goes, then removes the item on confirm', (
      tester,
    ) async {
      final (repo, c) = await open(tester);
      await tester.tap(find.byTooltip('Delete'));
      await tester.pumpAndSettle();
      expect(find.text('Delete this item and its data.'), findsOneWidget);
      expect(find.text('Deleted'), findsOneWidget);
      await tester.tap(find.widgetWithText(FilledButton, 'Delete'));
      await tester.pumpAndSettle();
      expect(repo.confirmed, ['c-a']);
      expect(find.text('Dentist'), findsNothing);
      await tester.pumpWidget(const SizedBox());
      c.dispose();
    });

    testWidgets('leaves no pending preview when the dialog closes any way', (
      tester,
    ) async {
      final (repo, c) = await open(tester);
      await tester.tap(find.byTooltip('Delete'));
      await tester.pumpAndSettle();
      await tester.binding.handlePopRoute();
      await tester.pumpAndSettle();
      expect(find.text('Delete item'), findsNothing);
      expect(repo.discarded, ['c-a']);
      expect(repo.confirmed, isEmpty);
      await tester.pumpWidget(const SizedBox());
      c.dispose();
    });

    testWidgets('keeps the item when the owner backs out', (tester) async {
      final (repo, c) = await open(tester);
      await tester.tap(find.byTooltip('Delete'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Keep it'));
      await tester.pumpAndSettle();
      expect(repo.discarded, ['c-a']);
      expect(repo.confirmed, isEmpty);
      expect(find.text('Dentist'), findsOneWidget);
      await tester.pumpWidget(const SizedBox());
      c.dispose();
    });
  });

  group('Item details', () {
    Future<(FakeLauncher, CalendarController)> show(
      WidgetTester tester,
      Item shown,
    ) async {
      tester.view.physicalSize = const Size(412, 892);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      final launcher = FakeLauncher();
      final c = controller(FakeItems()..items = [shown]);
      await c.load();
      await tester.pumpWidget(
        MaterialApp(
          theme: Tokens.theme,
          home: CalendarScreen(controller: c, linkLauncher: launcher),
        ),
      );
      await tester.pumpAndSettle();
      return (launcher, c);
    }

    Future<void> finish(WidgetTester tester, CalendarController c) async {
      await tester.pumpWidget(const SizedBox());
      c.dispose();
    }

    testWidgets(
      'tapping the card expands it and enlarged details scroll on a narrow phone',
      (tester) async {
        final (_, c) = await show(
          tester,
          appointment(
            location: 'Damrak 1, Amsterdam, Netherlands',
            notes: 'Bring ID and booking ABC123456789',
            offsets: [60],
          ),
        );
        tester.view.physicalSize = const Size(320, 640);
        await tester.pumpWidget(
          MaterialApp(
            theme: Tokens.theme,
            home: MediaQuery(
              data: const MediaQueryData(
                textScaler: TextScaler.linear(1.4),
                disableAnimations: true,
              ),
              child: CalendarScreen(
                controller: c,
                linkLauncher: FakeLauncher(),
              ),
            ),
          ),
        );
        await tester.pumpAndSettle();
        await tester.ensureVisible(find.text('Dentist'));
        await tester.tap(find.text('Dentist'));
        await tester.pumpAndSettle();
        expect(find.text('Notes'), findsOneWidget);
        await tester.ensureVisible(find.text('1 hour before'));
        await tester.pumpAndSettle();
        expect(tester.takeException(), isNull);
        await finish(tester, c);
      },
    );

    testWidgets('a timezone without an end does not invent an Ends row', (
      tester,
    ) async {
      final (_, c) = await show(
        tester,
        Item(
          id: 'a',
          title: 'Dentist',
          status: 'open',
          startAt: DateTime(2026, 11, 12, 9),
          startTimezone: 'Europe/Amsterdam',
          endTimezone: 'Europe/Amsterdam',
        ),
      );
      await tester.ensureVisible(find.text('Details'));
      await tester.tap(find.text('Details'));
      await tester.pumpAndSettle();
      expect(find.text('Starts'), findsOneWidget);
      expect(find.text('Ends'), findsNothing);
      await finish(tester, c);
    });

    testWidgets('an appointment in one zone does not say Departs or Arrives', (
      tester,
    ) async {
      final (_, c) = await show(tester, appointment());
      expect(find.textContaining('Departs'), findsNothing);
      expect(find.textContaining('Arrives'), findsNothing);
      await finish(tester, c);
    });

    testWidgets('a trip across zones says Departs and Arrives', (tester) async {
      final (_, c) = await show(
        tester,
        appointment(endZone: 'America/New_York'),
      );
      expect(
        find.text('Departs 2026-11-12 09:00 Europe/Amsterdam'),
        findsOneWidget,
      );
      expect(
        find.text('Arrives 2026-11-12 09:30 America/New_York'),
        findsOneWidget,
      );
      await finish(tester, c);
    });

    testWidgets('an item with no details has no Details button', (
      tester,
    ) async {
      final (_, c) = await show(
        tester,
        item('t', 'Taxes', due: utc(2026, 11, 12, 17)),
      );
      expect(find.text('Details'), findsNothing);
      await finish(tester, c);
    });

    testWidgets('Details shows every row and Hide details hides them', (
      tester,
    ) async {
      final (_, c) = await show(
        tester,
        appointment(
          location: 'Damrak 1',
          notes: '- Bring ID\n- Call 020-1234567',
          offsets: [1440, 60],
        ),
      );
      expect(tester.getSize(find.text('Details')).height, greaterThan(0));
      expect(
        tester.getSize(find.widgetWithText(TextButton, 'Details')).height,
        44,
      );
      await tester.tap(find.text('Details'));
      await tester.pumpAndSettle();
      for (final label in [
        'Starts',
        'Ends',
        'Duration',
        'Type',
        'Address',
        'Notes',
        'Reminders',
      ]) {
        expect(find.text(label), findsOneWidget, reason: label);
      }
      expect(find.text('Appointment'), findsOneWidget);
      expect(find.text('Bring ID'), findsOneWidget);
      expect(find.text('1 day, 1 hour before'), findsOneWidget);
      tester.view.physicalSize = const Size(412, 1600);
      await tester.pumpAndSettle();
      await expectLater(
        find.byType(CalendarScreen),
        matchesGoldenFile('goldens/calendar-details.png'),
      );

      await tester.ensureVisible(find.text('Hide details'));
      await tester.tap(find.text('Hide details'));
      await tester.pumpAndSettle();
      expect(find.text('Notes'), findsNothing);
      await finish(tester, c);
    });

    testWidgets('the address opens Google Maps', (tester) async {
      final (launcher, c) = await show(
        tester,
        appointment(location: 'Damrak 1'),
      );
      await tester.tap(find.text('Details'));
      await tester.pumpAndSettle();
      await tester.ensureVisible(find.text('Damrak 1'));
      await tester.tap(find.text('Damrak 1'));
      await tester.pump();
      expect(launcher.opened, [mapsUri('Damrak 1')]);
      await finish(tester, c);
    });

    testWidgets('a phone number and an email in the notes open their apps', (
      tester,
    ) async {
      final (launcher, c) = await show(
        tester,
        appointment(notes: '- Call 020-1234567\n- Mail info@clinic.nl'),
      );
      await tester.tap(find.text('Details'));
      await tester.pumpAndSettle();
      await tester.ensureVisible(find.text('Call 020-1234567'));
      await tester.tapOnText(find.textRange.ofSubstring('020-1234567'));
      await tester.ensureVisible(find.text('Mail info@clinic.nl'));
      await tester.tapOnText(find.textRange.ofSubstring('info@clinic.nl'));
      expect(launcher.opened.map((u) => u.toString()), [
        'tel:0201234567',
        'mailto:info@clinic.nl',
      ]);
      await finish(tester, c);
    });
  });
}

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:minder/application/calendar_controller.dart';
import 'package:minder/application/use_cases.dart';
import 'package:minder/data/wire.dart';
import 'package:minder/domain/chat_repository.dart';
import 'package:minder/domain/item.dart';
import 'package:minder/domain/item_repository.dart';
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

class FakeItems implements ItemRepository {
  final requests = <(DateTime, DateTime)>[];
  List<Item> items = [];
  bool offline = false;
  @override
  Future<List<Item>> itemsBetween(DateTime from, DateTime to) async {
    requests.add((from, to));
    if (offline) throw const ApiException('Check your connection.');
    return items.where((i) {
      final at = i.placedAt;
      return at != null && !at.isBefore(from) && at.isBefore(to);
    }).toList();
  }
}

CalendarController controller(FakeItems repo) =>
    CalendarController(LoadItems(repo), now: () => DateTime(2026, 11, 12, 10));

class PendingItems implements ItemRepository {
  final requests = <Completer<List<Item>>>[];
  @override
  Future<List<Item>> itemsBetween(DateTime from, DateTime to) {
    final result = Completer<List<Item>>();
    requests.add(result);
    return result.future;
  }
}

void main() {
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
    final c = CalendarController(
      LoadItems(repo),
      now: () => DateTime(2026, 11, 12),
    );
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
    final c = CalendarController(LoadItems(repo));
    final first = c.load();
    final latest = c.next();
    repo.requests[0].completeError(const ApiException('Old failure'));
    await first;
    expect(c.loading, true);
    expect(c.error, isNull);
    repo.requests[1].complete([]);
    await latest;
    expect(c.loading, false);
    expect(c.error, isNull);
    c.dispose();
  });

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
    expect(find.text('NOVEMBER 2026'), findsOneWidget);
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
    expect(find.text('DECEMBER 2026'), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
    c.dispose();
  });
}

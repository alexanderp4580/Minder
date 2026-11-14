import 'package:minder/ui/item_colours.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:minder/application/calendar_controller.dart';
import 'package:minder/data/wire.dart';
import 'package:minder/domain/item.dart';
import 'package:minder/domain/item_repository.dart';
import 'package:minder/ui/calendar_screen.dart';
import 'package:minder/ui/calendar_item_details.dart';
import 'package:minder/ui/tokens.dart';

Item sample(Map<String, dynamic> extra) => itemFromJson({
  'id': 'workshop',
  'title': 'Workshop',
  'status': 'open',
  'kind': 'event',
  'start_date': '2026-11-20',
  'end_date': '2026-11-23',
  ...extra,
});

class Items extends Fake implements ItemRepository {
  Items(this.values);
  final List<Item> values;
  @override
  Future<List<Item>> itemsBetween(DateTime from, DateTime to) async => values;
}

void main() {
  testWidgets(
    'detail colour choice waits for Apply, Cancel preserves it and failures remain visible',
    (tester) async {
      final calls = <String>[];
      await tester.pumpWidget(
        MaterialApp(
          theme: Tokens.theme,
          home: Scaffold(
            body: SingleChildScrollView(
              child: ItemColourEditor(
                value: 'blue',
                onApply: (v) async {
                  calls.add(v);
                  throw Exception('offline');
                },
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.byType(TextButton));
      await tester.pump();
      await tester.tap(find.byKey(const ValueKey('colour-amber')));
      await tester.pump();
      expect(calls, isEmpty);
      await tester.tap(find.text('Cancel'));
      await tester.pump();
      expect(calls, isEmpty);
      expect(find.text('Apply colour'), findsNothing);
      await tester.tap(find.byType(TextButton));
      await tester.pump();
      await tester.tap(find.byKey(const ValueKey('colour-teal')));
      await tester.tap(find.text('Apply colour'));
      await tester.pumpAndSettle();
      expect(calls, ['teal']);
      expect(
        find.text('Could not change the colour. Try again.'),
        findsOneWidget,
      );
      expect(find.text('Cancel'), findsOneWidget);
    },
  );

  test('ordinary inclusive dates include final day; checkout and legacy exclude it', () {
    expect(
      sample({'date_end': 'inclusive'}).occursOn(DateTime(2026, 11, 23)),
      isTrue,
    );
    expect(
      sample({'date_end': 'checkout'}).occursOn(DateTime(2026, 11, 23)),
      isFalse,
    );
    expect(sample({}).occursOn(DateTime(2026, 11, 23)), isFalse);
    final instant = itemFromJson({
      'id': 'timed',
      'title': 'Party',
      'status': 'open',
      'start_at': '2026-11-20T17:00:00+01:00',
      'end_at': '2026-11-21T00:00:00Z',
    });
    expect(instant.occursOn(instant.endAt!), isFalse);
  });
  testWidgets(
    'Home and unknown labels stay plain; only classified addresses link',
    (tester) async {
      Future<void> show(Map<String, dynamic> extra) async {
        await tester.pumpWidget(
          MaterialApp(
            theme: Tokens.theme,
            home: Scaffold(
              body: SingleChildScrollView(
                child: CalendarItemDetails(item: sample(extra)),
              ),
            ),
          ),
        );
      }

      await show({'location': 'Home', 'location_kind': 'label'});
      expect(find.text('Location'), findsOneWidget);
      expect(find.text('Opens Google Maps'), findsNothing);
      expect(find.text('Event'), findsOneWidget);
      await show({'location': 'Unclassified place'});
      expect(find.text('Opens Google Maps'), findsNothing);
      await show({
        'location': 'Damrak 1, Amsterdam',
        'location_kind': 'address',
      });
      expect(find.text('Opens Google Maps'), findsOneWidget);
    },
  );
  testWidgets(
    'named spans continue across weeks and retain all overlapping titles',
    (tester) async {
      final c = CalendarController(
        Items([
          sample({'color': 'blue', 'date_end': 'inclusive'}),
          sample({
            'id': 'trip',
            'title': 'Trip',
            'color': 'teal',
            'date_end': 'inclusive',
          }),
          sample({
            'id': 'party',
            'title': 'Party',
            'start_date': '2026-11-21',
            'end_date': '2026-11-21',
            'date_end': 'inclusive',
            'color': 'lilac',
          }),
        ]),
        now: () => DateTime(2026, 11, 14),
      );
      await c.load();
      addTearDown(c.dispose);
      tester.view.physicalSize = const Size(412, 1200);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await tester.pumpWidget(
        MaterialApp(
          theme: Tokens.theme,
          home: CalendarScreen(controller: c),
        ),
      );
      await tester.pump();
      expect(find.byKey(const ValueKey('eventbar-workshop-3')), findsOneWidget);
      expect(find.byKey(const ValueKey('eventbar-workshop-4')), findsOneWidget);
      expect(find.byKey(const ValueKey('eventbar-trip-3')), findsOneWidget);
      expect(find.byKey(const ValueKey('eventbar-party-3')), findsOneWidget);
      c.select(DateTime(2026, 11, 23));
      await tester.pump();
      expect(find.text('Workshop'), findsWidgets);
      expect(find.text('Trip'), findsWidgets);
      expect(tester.takeException(), isNull);
    },
  );
}

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:minder/application/calendar_controller.dart';
import 'package:minder/application/chat_controller.dart';
import 'package:minder/data/wire.dart';
import 'package:minder/domain/chat_repository.dart';
import 'package:minder/domain/entry.dart';
import 'package:minder/domain/item.dart';
import 'package:minder/domain/item_repository.dart';
import 'package:minder/ui/calendar_screen.dart';
import 'package:minder/ui/chat_screen.dart';
import 'package:minder/ui/tokens.dart';

class EmptyItems implements ItemRepository {
  @override
  Future<List<Item>> itemsBetween(DateTime from, DateTime to) async => [];
}

class Repo implements ChatRepository {
  Repo(this.saved);
  final List<Entry> saved;
  @override
  Future<List<Entry>> history(int after) async =>
      saved.where((e) => e.cursor > after).toList();
  @override
  Stream<Entry> watch(int after) => const Stream.empty();
  @override
  Future<Entry> send(String text) async => saved.first;
  @override
  Future<Entry> confirm(String id) async => saved.first;
  @override
  Future<Entry> discard(String id) async => saved.first;
  @override
  Future<Entry> undo(String id) async => saved.first;
  @override
  Future<Entry> actOnReminder(String id, ReminderAction a) async => saved.first;
}

Entry entry(
  int n,
  String kind, {
  String text = '',
  Map<String, dynamic>? change,
  Map<String, dynamic>? reminder,
  double cost = 0,
}) => entryFromJson({
  'id': 'e$n',
  'cursor': n,
  'kind': kind,
  'text': text,
  'created_at': DateTime(2026, 11, 12, 15, 5).toUtc().toIso8601String(),
  'change': ?change,
  'reminder': ?reminder,
  'cost_eur': cost,
});

void main() {
  testWidgets(
    'calendar keeps separate day targets, selection frames and week gaps',
    (tester) async {
      tester.view.physicalSize = const Size(412, 892);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      final c = CalendarController(
        EmptyItems(),
        now: () => DateTime(2026, 11, 12, 10),
      );
      await c.load();
      await tester.pumpWidget(
        MaterialApp(
          theme: Tokens.theme,
          home: CalendarScreen(controller: c),
        ),
      );
      await tester.pumpAndSettle();
      final first = tester.getRect(
        find.byKey(const ValueKey('day-2026-11-02')),
      );
      final next = tester.getRect(find.byKey(const ValueKey('day-2026-11-09')));
      expect(first.height, 56);
      expect(next.top - first.top, 64);
      final selection = tester.getRect(
        find.byKey(const ValueKey('calendar-selection')),
      );
      expect(selection.size, const Size(44, 44));
      final lastWeek = tester.getRect(
        find.byKey(const ValueKey('day-2026-12-06')),
      );
      final today = tester.getRect(
        find.widgetWithText(OutlinedButton, 'Today'),
      );
      expect(today.top - lastWeek.bottom, greaterThanOrEqualTo(32));
      expect(today.height, greaterThanOrEqualTo(44));
      expect(find.text('November'), findsOneWidget);
      expect(find.text('2026'), findsOneWidget);
      await tester.pumpWidget(const SizedBox());
      c.dispose();
    },
  );

  setUpAll(() async {
    TestWidgetsFlutterBinding.ensureInitialized();
    await (FontLoader(
      'MaterialIcons',
    )..addFont(rootBundle.load('fonts/MaterialIcons-Regular.otf'))).load();
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

  testWidgets('the theme sizes buttons and fields at 44 px', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: Tokens.theme,
        home: Scaffold(
          body: Column(
            children: [
              FilledButton(onPressed: () {}, child: const Text('Confirm')),
              OutlinedButton(onPressed: () {}, child: const Text('Discard')),
              IconButton(
                tooltip: 'Calendar',
                onPressed: () {},
                icon: const Icon(Icons.calendar_month),
              ),
              const TextField(),
            ],
          ),
        ),
      ),
    );
    expect(tester.getSize(find.byType(FilledButton)).height, 44);
    expect(tester.getSize(find.byType(OutlinedButton)).height, 44);
    expect(
      tester.getSize(find.byType(IconButton)),
      const Size(Tokens.minTarget, Tokens.minTarget),
    );
    expect(tester.getSize(find.byType(TextField)).height, 44);
  });

  testWidgets('chat layout follows the mockup spacing', (tester) async {
    tester.view.physicalSize = const Size(412, 892);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final repo = Repo([
      entry(1, 'user', text: 'Move the dentist to Friday.'),
      entry(
        2,
        'preview',
        cost: 0.0008,
        change: {
          'id': 'p',
          'action': 'update_item',
          'item_title': 'Dentist',
          'fields': [
            {'name': 'title', 'new': 'Dentist'},
          ],
          'state': 'pending',
        },
      ),
      entry(
        3,
        'reminder',
        text: 'Reminder: Pay tax bill.',
        reminder: {'id': 'r1', 'state': 'active'},
      ),
    ]);
    final c = ChatController(repo, pollInterval: const Duration(days: 1));
    await c.start();
    await tester.pumpWidget(
      MaterialApp(
        theme: Tokens.theme,
        home: ChatScreen(controller: c, onCalendar: () {}),
      ),
    );
    await tester.pumpAndSettle();
    // Your message starts 64 px from the screen edge: 20 px gutter and 44 px margin.
    expect(
      tester
          .getTopLeft(
            find
                .descendant(
                  of: find.byKey(const ValueKey('e1')),
                  matching: find.byType(DecoratedBox),
                )
                .first,
          )
          .dx,
      64,
    );
    // The time follows the sender. Minder's line starts at the gutter,
    // and yours ends at the right gutter.
    expect(
      tester.getTopLeft(find.text('MINDER · 15:05 · €0.0008')).dx,
      Tokens.gutter,
    );
    expect(
      tester.getTopRight(find.text('YOU · 15:05')).dx,
      412 - Tokens.gutter,
    );
    // Confirm and Discard share one row, 8 px apart. Confirm is last, so its
    // cut corner sits at the bottom right of the card.
    final confirm = tester.getRect(find.byType(FilledButton).first);
    final discard = tester.getRect(find.byType(OutlinedButton).first);
    expect(confirm.height, 44);
    expect(discard.height, 44);
    expect(confirm.left - discard.right, Tokens.gap);
    expect(confirm.width / discard.width, closeTo(1.3, 0.02));
    expect(confirm.right, 412 - Tokens.gutter - 12 - 1);
    // The reminder buttons share one row, 8 px apart, 44 px high. Done is last,
    // at the right edge.
    final done = tester.getRect(find.widgetWithText(FilledButton, 'Done'));
    final snooze = tester.getRect(
      find.widgetWithText(OutlinedButton, 'Snooze'),
    );
    final open = tester.getRect(find.widgetWithText(OutlinedButton, 'Open'));
    expect(snooze.top, open.top);
    expect(open.top, done.top);
    expect(done.height, 44);
    expect(open.left - snooze.right, Tokens.gap);
    expect(done.left - open.right, Tokens.gap);
    expect(done.right, 412 - Tokens.gutter);
    // The header has no divider. The first sender line sits 38 px under the
    // status line: 22 px of list padding and 16 px of entry margin.
    final status = tester.getBottomLeft(find.text('MINDER IS HERE'));
    final firstSender = tester.getTopLeft(find.text('YOU · 15:05'));
    expect(firstSender.dy - status.dy, closeTo(38, 4));
    // Your message pads 10 px above and 12 px at the sides.
    final box = tester.getRect(
      find
          .descendant(
            of: find.byKey(const ValueKey('e1')),
            matching: find.byType(DecoratedBox),
          )
          .first,
    );
    final text = tester.getRect(find.text('Move the dentist to Friday.'));
    expect(text.top - box.top, closeTo(10 + 1, 3));
    expect(text.left - box.left, closeTo(12 + 1, 1));
    // The composer is 56 px high with a 44 px send button.
    expect(tester.getSize(find.byKey(const ValueKey('composer'))).height, 56);
    expect(tester.getSize(find.byTooltip('Send')).height, 44);
    // The text field is as tall as the Send button and shares its centre line.
    final field = tester.getRect(find.byType(TextField));
    final send = tester.getRect(find.byTooltip('Send'));
    expect(field.height, 44);
    expect(field.center.dy, send.center.dy);
    expect(send.right, 412 - 14);
    // Every button has a 44 px hit area.
    for (final f in [
      find.byType(FilledButton),
      find.byType(OutlinedButton),
      find.byType(IconButton),
    ]) {
      for (final e in f.evaluate()) {
        expect(
          tester.getSize(find.byWidget(e.widget)).height,
          greaterThanOrEqualTo(44),
        );
      }
    }
    await tester.pumpWidget(const SizedBox());
    c.dispose();
  });
}

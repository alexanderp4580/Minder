import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:minder/application/chat_controller.dart';
import 'package:minder/domain/chat_repository.dart';
import 'package:minder/domain/entry.dart';
import 'package:minder/ui/chat_screen.dart';
import 'package:minder/ui/components.dart';
import 'package:minder/ui/tokens.dart';

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
  @override
  void close() {}
}

Entry entry(
  int n,
  String kind, {
  String text = '',
  Map<String, dynamic>? change,
  Map<String, dynamic>? reminder,
  double cost = 0,
}) => Entry.fromJson({
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
  setUpAll(() async {
    TestWidgetsFlutterBinding.ensureInitialized();
    await (FontLoader('MaterialIcons')
          ..addFont(rootBundle.load('fonts/MaterialIcons-Regular.otf')))
        .load();
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

  testWidgets('shared buttons follow the brand sizes', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        theme: Tokens.theme,
        home: Scaffold(
          body: Column(
            children: [
              PrimaryButton(label: 'Confirm', onPressed: () {}),
              SecondaryButton(label: 'Discard', onPressed: () {}),
              SquareIconButton(
                icon: Icons.calendar_month,
                tooltip: 'Calendar',
                onPressed: () {},
              ),
            ],
          ),
        ),
      ),
    );
    expect(tester.getSize(find.byType(PrimaryButton)).height, 44);
    expect(tester.getSize(find.byType(SecondaryButton)).height, 44);
    expect(
      tester.getSize(find.byType(SquareIconButton)),
      const Size(Tokens.minTarget, Tokens.minTarget),
    );
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
    // Confirm and Discard share one row of equal width, 8 px apart.
    final confirm = tester.getRect(find.byType(PrimaryButton).first);
    final discard = tester.getRect(find.byType(SecondaryButton).first);
    expect(confirm.height, 44);
    expect(discard.height, 44);
    expect(discard.left - confirm.right, Tokens.gap);
    expect(confirm.width / discard.width, closeTo(1.3, 0.02));
    // The reminder buttons share one row, 8 px apart, 44 px high.
    final done = tester.getRect(find.widgetWithText(PrimaryButton, 'Done'));
    final snooze = tester.getRect(find.byType(SecondaryFace));
    final open = tester.getRect(find.widgetWithText(SecondaryButton, 'Open'));
    expect(done.top, snooze.top);
    expect(snooze.top, open.top);
    expect(done.height, 44);
    expect(snooze.left - done.right, Tokens.gap);
    expect(open.left - snooze.right, Tokens.gap);
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
    // Every button has a 44 px hit area.
    for (final f in [
      find.byType(PrimaryButton),
      find.byType(SecondaryButton),
      find.byType(SquareIconButton),
    ]) {
      for (final e in f.evaluate()) {
        expect(tester.getSize(find.byWidget(e.widget)).height, greaterThanOrEqualTo(44));
      }
    }
    await tester.pumpWidget(const SizedBox());
    c.dispose();
  });
}

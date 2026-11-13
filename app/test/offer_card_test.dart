import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:minder/application/chat_controller.dart';
import 'package:minder/data/wire.dart';
import 'package:minder/domain/chat_repository.dart';
import 'package:minder/domain/entry.dart';
import 'package:minder/ui/entry_view.dart';
import 'package:minder/ui/tokens.dart';

const _offerText =
    'Look up "SnowWorld Landgraaf" on Google for its address and website?';

class _Repository extends Fake implements ChatRepository {
  _Repository(this.stored);
  final List<Entry> stored;
  final sent = <String>[];
  Completer<Entry>? pending;
  @override
  Future<List<Entry>> history(int after) async => stored;
  @override
  Stream<Entry> watch(int after) => const Stream.empty();
  @override
  Future<Entry> send(String text) {
    sent.add(text);
    return pending?.future ?? Future.value(_user(text, 9));
  }
}

Entry _user(String text, int cursor) => Entry(
  id: 'u$cursor',
  cursor: cursor,
  kind: EntryKind.user,
  text: text,
  createdAt: DateTime.utc(2026, 11, 13, 14),
);

Entry _offer({List<String> options = const ['Skip', 'Look it up']}) => Entry(
  id: 'offer',
  cursor: 2,
  kind: EntryKind.minder,
  text: _offerText,
  createdAt: DateTime.utc(2026, 11, 13, 14),
  options: options,
);

Future<(_Repository, ChatController)> _show(
  WidgetTester tester,
  List<Entry> stored,
  Entry shown, {
  Completer<Entry>? pending,
}) async {
  final repository = _Repository(stored)..pending = pending;
  final controller = ChatController(
    repository,
    pollInterval: const Duration(days: 1),
  );
  await controller.start();
  await tester.pumpWidget(
    MaterialApp(
      theme: Tokens.theme,
      home: Scaffold(
        body: AnimatedBuilder(
          animation: controller,
          builder: (_, _) => SingleChildScrollView(
            child: EntryView(entry: shown, controller: controller),
          ),
        ),
      ),
    ),
  );
  return (repository, controller);
}

/// Disposes the controller inside the test, so its poll timer is gone before the timer check.
Future<void> _done(WidgetTester tester, ChatController controller) async {
  await tester.pumpWidget(const SizedBox());
  controller.dispose();
}

void main() {
  test('an entry reads its quick replies from the wire', () {
    final entry = entryFromJson({
      'id': 'e',
      'cursor': 2,
      'kind': 'minder',
      'text': _offerText,
      'created_at': '2026-11-13T14:00:00Z',
      'options': ['Skip', 'Look it up'],
    });
    expect(entry.options, ['Skip', 'Look it up']);
    expect(
      entryFromJson({
        'id': 'e',
        'kind': 'minder',
        'created_at': '2026-11-13T14:00:00Z',
      }).options,
      isEmpty,
    );
  });

  testWidgets('the newest offer is a card with Skip and Look it up', (
    tester,
  ) async {
    final offer = _offer();
    final (_, controller) = await _show(tester, [offer], offer);
    expect(find.text('LOOK UP · ASKS FIRST'), findsOneWidget);
    expect(find.text(_offerText), findsOneWidget);
    expect(find.widgetWithText(OutlinedButton, 'Skip'), findsOneWidget);
    final primary = find.widgetWithText(FilledButton, 'Look it up');
    expect(primary, findsOneWidget);
    expect(tester.getSize(primary).height, greaterThanOrEqualTo(44));
    await _done(tester, controller);
  });

  testWidgets('tapping Look it up sends exactly that text', (tester) async {
    final offer = _offer();
    final (repository, controller) = await _show(tester, [offer], offer);
    await tester.tap(find.widgetWithText(FilledButton, 'Look it up'));
    await tester.pump();
    expect(repository.sent, ['Look it up']);
    await _done(tester, controller);
  });

  testWidgets('tapping Skip sends Skip', (tester) async {
    final offer = _offer();
    final (repository, controller) = await _show(tester, [offer], offer);
    await tester.tap(find.widgetWithText(OutlinedButton, 'Skip'));
    await tester.pump();
    expect(repository.sent, ['Skip']);
    await _done(tester, controller);
  });

  testWidgets('an answered offer keeps its text and loses its buttons', (
    tester,
  ) async {
    final offer = _offer();
    final (_, controller) = await _show(tester, [
      offer,
      _user('Look it up', 3),
    ], offer);
    expect(find.text(_offerText), findsOneWidget);
    expect(find.byType(FilledButton), findsNothing);
    expect(find.byType(OutlinedButton), findsNothing);
    await _done(tester, controller);
  });

  testWidgets('the buttons wait while a message is sending', (tester) async {
    final offer = _offer();
    final pending = Completer<Entry>();
    final (_, controller) = await _show(
      tester,
      [offer],
      offer,
      pending: pending,
    );
    await tester.tap(find.widgetWithText(OutlinedButton, 'Skip'));
    await tester.pump();
    expect(controller.waiting, isTrue);
    expect(
      tester
          .widget<FilledButton>(find.widgetWithText(FilledButton, 'Look it up'))
          .onPressed,
      isNull,
    );
    pending.complete(_user('Skip', 9));
    await tester.pump();
    await _done(tester, controller);
  });

  testWidgets('a minder entry without options renders as before', (
    tester,
  ) async {
    final plain = _offer(options: const []);
    final (_, controller) = await _show(tester, [plain], plain);
    expect(find.text(_offerText), findsOneWidget);
    expect(find.text('LOOK UP · ASKS FIRST'), findsNothing);
    expect(find.byType(FilledButton), findsNothing);
    await _done(tester, controller);
  });
}

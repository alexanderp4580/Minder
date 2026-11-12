import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:minder/application/chat_controller.dart';
import 'package:minder/domain/chat_repository.dart';
import 'package:minder/domain/entry.dart';
import 'package:minder/ui/chat_screen.dart';
import 'package:minder/ui/tokens.dart';

Entry entry(
  String id,
  int cursor,
  String kind, {
  String text = '',
  Map<String, dynamic>? change,
  double? cost,
}) => Entry.fromJson({
  'id': id,
  'cursor': cursor,
  'kind': kind,
  'text': text,
  'created_at': '2026-11-11T09:00:00Z',
  'change': ?change,
  'cost_eur': ?cost,
});
Map<String, dynamic> preview({String state = 'pending', String? undo}) => {
  'id': 'p',
  'action': 'create_item',
  'item_title': 'Call plumber',
  'fields': [
    {'name': 'title', 'new': 'Call plumber'},
  ],
  'state': state,
  'undo_id': ?undo,
};

class FakeRepository implements ChatRepository {
  final events = StreamController<Entry>.broadcast();
  final saved = <Entry>[];
  bool offline = false;
  int sends = 0;
  @override
  Future<List<Entry>> history(int after) async {
    if (offline) throw const ApiException('Check your connection.');
    return List.of(saved.where((e) => e.cursor > after));
  }

  @override
  Stream<Entry> watch(int after) => events.stream;
  @override
  Future<Entry> send(String text) async {
    if (offline) throw const ApiException('Check your connection.');
    sends++;
    return entry('u$sends', sends, 'user', text: text);
  }

  @override
  Future<Entry> confirm(String changeId) async => entry(
    'result',
    3,
    'result',
    text: 'Saved: Call plumber.',
    change: preview(state: 'confirmed', undo: 'audit'),
  );
  @override
  Future<Entry> discard(String changeId) async => entry(
    'p',
    4,
    'preview',
    text: 'Change discarded.',
    change: preview(state: 'discarded'),
  );
  @override
  Future<Entry> undo(String undoId) async =>
      entry('undo', 5, 'result', text: 'Undone: Call plumber.');
  @override
  Future<Entry> actOnReminder(String reminderId, ReminderAction action) async =>
      confirm(reminderId);

  @override
  void close() {
    events.close();
  }
}

void main() {
  setUpAll(() async {
    TestWidgetsFlutterBinding.ensureInitialized();
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
  test('send waits for reply, confirms and undoes', () async {
    final repo = FakeRepository();
    final c = ChatController(repo, pollInterval: const Duration(days: 1));
    await c.start();
    await c.send('Call plumber');
    expect(c.waiting, isTrue);
    expect(c.entries.single.text, 'Call plumber');
    repo.events.add(entry('p', 2, 'preview', change: preview()));
    await Future<void>.delayed(Duration.zero);
    expect(c.waiting, isFalse);
    await c.confirm('p');
    expect(c.entries.last.kind, 'result');
    await c.undo('audit');
    expect(c.entries.last.text, 'Undone: Call plumber.');
    c.dispose();
  });
  test('failed send can retry and discard leaves no active buttons', () async {
    final repo = FakeRepository();
    final c = ChatController(repo, pollInterval: const Duration(days: 1));
    await c.start();
    repo.offline = true;
    await c.send('Call plumber');
    expect(c.error, isNotNull);
    repo.offline = false;
    await c.retry();
    expect(repo.sends, 1);
    repo.events.add(entry('p', 2, 'preview', change: preview()));
    await Future<void>.delayed(Duration.zero);
    await c.discard('p');
    expect(c.entries.last.change?['state'], 'discarded');
    c.dispose();
  });
  test('poll fills gaps and replaces older entries by cursor', () async {
    final repo = FakeRepository();
    final c = ChatController(repo, pollInterval: const Duration(days: 1));
    await c.start();
    repo.events.add(entry('p', 1, 'preview', change: preview()));
    await Future<void>.delayed(Duration.zero);
    repo.saved.add(
      entry('p', 3, 'preview', change: preview(state: 'confirmed')),
    );
    repo.saved.add(entry('r', 4, 'result', text: 'Saved.'));
    await c.sync();
    expect(c.entries.length, 2);
    expect(c.entries.first.change?['state'], 'confirmed');
    expect(c.cursor, 4);
    c.dispose();
  });
  testWidgets('chat renders preview and sends text at phone width', (
    tester,
  ) async {
    tester.view.reset();
    tester.view.physicalSize = const Size(412, 892);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final repo = FakeRepository();
    repo.saved.add(entry('p', 1, 'preview', change: preview()));
    final c = ChatController(repo, pollInterval: const Duration(days: 1));
    await c.start();
    await tester.pumpWidget(
      MaterialApp(
        theme: Tokens.theme,
        home: ChatScreen(controller: c),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Confirm'), findsOneWidget);
    expect(find.text('Discard'), findsOneWidget);
    expect(find.text('Edit'), findsNothing);
    await tester.enterText(find.byType(TextField), 'Call plumber');
    await tester.tap(find.byTooltip('Send'));
    await tester.pump();
    expect(repo.sends, 1);
    expect(find.text('Working'), findsOneWidget);
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
    c.dispose();
  });
  testWidgets('active reminder shows Done, Snooze and Open and routes Done', (
    tester,
  ) async {
    tester.view.reset();
    tester.view.physicalSize = const Size(412, 892);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final repo = FakeRepository();
    repo.saved.add(
      Entry.fromJson({
        'id': 'r',
        'cursor': 1,
        'kind': 'minder',
        'text': 'Reminder: Call plumber.',
        'created_at': '2026-11-11T09:00:00Z',
        'reminder': {'id': 'occ1', 'state': 'active'},
      }),
    );
    final c = ChatController(repo, pollInterval: const Duration(days: 1));
    await c.start();
    await tester.pumpWidget(
      MaterialApp(
        theme: Tokens.theme,
        home: ChatScreen(controller: c),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Done'), findsOneWidget);
    expect(find.text('Snooze'), findsOneWidget);
    expect(find.text('Open'), findsOneWidget);
    await tester.tap(find.text('Done'));
    await tester.pumpAndSettle();
    expect(find.text('Saved: Call plumber.'), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
    c.dispose();
  });
  testWidgets('profile preview shows its action and field labels', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(412, 892);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final repo = FakeRepository();
    repo.saved.add(
      entry(
        'p',
        1,
        'preview',
        change: {
          'id': 'p',
          'action': 'update_profile',
          'item_title': 'Chore',
          'fields': [
            {'name': 'remind_before_minutes', 'old': '0', 'new': '30'},
          ],
          'state': 'pending',
        },
      ),
    );
    final c = ChatController(repo, pollInterval: const Duration(days: 1));
    await c.start();
    await tester.pumpWidget(
      MaterialApp(
        theme: Tokens.theme,
        home: ChatScreen(controller: c),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('PREVIEW · CHANGE PROFILE'), findsOneWidget);
    expect(find.text('Remind before (minutes):'), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
    c.dispose();
  });
  testWidgets('a reply shows what its request cost', (tester) async {
    tester.view.physicalSize = const Size(412, 892);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final repo = FakeRepository();
    repo.saved.add(entry('a', 1, 'minder', text: 'Where is it?', cost: 0.0008));
    repo.saved.add(entry('b', 2, 'minder', text: 'Free reply.'));
    final c = ChatController(repo, pollInterval: const Duration(days: 1));
    await c.start();
    await tester.pumpWidget(
      MaterialApp(
        theme: Tokens.theme,
        home: ChatScreen(controller: c),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.textContaining('€0.0008'), findsOneWidget);
    expect(find.textContaining('€'), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
    c.dispose();
  });
  testWidgets('the header opens the calendar', (tester) async {
    tester.view.physicalSize = const Size(412, 892);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final repo = FakeRepository();
    final c = ChatController(repo, pollInterval: const Duration(days: 1));
    await c.start();
    var opened = 0;
    await tester.pumpWidget(
      MaterialApp(
        theme: Tokens.theme,
        home: ChatScreen(controller: c, onCalendar: () => opened++),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('Calendar'));
    expect(opened, 1);
    await tester.pumpWidget(const SizedBox());
    c.dispose();
  });
  testWidgets('entry types match M0 golden', (tester) async {
    tester.view.physicalSize = const Size(412, 892);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final repo = FakeRepository();
    repo.saved.addAll([
      entry('u', 1, 'user', text: 'Call the plumber on Thursday.'),
      entry('m', 2, 'minder', text: 'Review the date and time.'),
      entry('p', 3, 'preview', change: preview()),
      entry(
        'r',
        4,
        'result',
        text: 'Saved: Call plumber.',
        change: preview(state: 'confirmed', undo: 'audit'),
      ),
      entry('e', 5, 'error', text: 'Check your connection.'),
    ]);
    final c = ChatController(repo, pollInterval: const Duration(days: 1));
    await c.start();
    await tester.pumpWidget(
      MaterialApp(
        theme: Tokens.theme,
        home: ChatScreen(controller: c),
      ),
    );
    await tester.pumpAndSettle();
    await expectLater(
      find.byType(ChatScreen),
      matchesGoldenFile('goldens/m0-chat.png'),
    );
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
    c.dispose();
  });
}

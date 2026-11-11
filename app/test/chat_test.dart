import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:minder/application/chat_controller.dart';
import 'package:minder/data/chat_api.dart';
import 'package:minder/domain/entry.dart';
import 'package:minder/ui/chat_screen.dart';
import 'package:minder/ui/tokens.dart';

Entry entry(
  String id,
  int cursor,
  String kind, {
  String text = '',
  Map<String, dynamic>? change,
}) => Entry.fromJson({
  'id': id,
  'cursor': cursor,
  'kind': kind,
  'text': text,
  'created_at': '2026-11-11T09:00:00Z',
  if (change != null) 'change': change,
});
Map<String, dynamic> preview({String state = 'pending', String? undo}) => {
  'id': 'p',
  'action': 'create_item',
  'item_title': 'Call plumber',
  'fields': [
    {'name': 'title', 'new': 'Call plumber'},
  ],
  'state': state,
  if (undo != null) 'undo_id': undo,
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
  Future<Entry> act(String route) async {
    if (route.endsWith('discard'))
      return entry(
        'p',
        4,
        'preview',
        text: 'Change discarded.',
        change: preview(state: 'discarded'),
      );
    if (route.startsWith('/v1/undo'))
      return entry('undo', 5, 'result', text: 'Undone: Call plumber.');
    return entry(
      'result',
      3,
      'result',
      text: 'Saved: Call plumber.',
      change: preview(state: 'confirmed', undo: 'audit'),
    );
  }

  @override
  void close() {
    events.close();
  }
}

void main() {
  setUpAll(() async {
    TestWidgetsFlutterBinding.ensureInitialized();
    for (final name in ['Michroma','Chakra Petch']) {
      final loader=FontLoader(name);
      for(final file in name=='Michroma'?['Michroma-Regular.ttf']:['ChakraPetch-Regular.ttf','ChakraPetch-Medium.ttf','ChakraPetch-SemiBold.ttf','ChakraPetch-Bold.ttf']) {loader.addFont(rootBundle.load('assets/fonts/$file'));}
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
    await c.act('/v1/changes/p/confirm');
    expect(c.entries.last.kind, 'result');
    await c.act('/v1/undo/audit');
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
    await c.act('/v1/changes/p/discard');
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

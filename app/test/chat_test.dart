import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:minder/application/chat_controller.dart';
import 'package:minder/data/wire.dart';
import 'package:minder/domain/chat_repository.dart';
import 'package:minder/domain/entry.dart';
import 'package:minder/domain/failure.dart';
import 'package:minder/ui/chat_screen.dart';
import 'package:minder/ui/tokens.dart';

Entry entry(
  String id,
  int cursor,
  String kind, {
  String text = '',
  Map<String, dynamic>? change,
  double? cost,
}) => entryFromJson({
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
  RepositoryFailure? sendFailure, historyFailure;
  Future<void>? sendGate;
  int sends = 0;
  @override
  Future<List<Entry>> history(int after) async {
    if (historyFailure != null) throw historyFailure!;
    if (offline) {
      throw const RepositoryFailure('Check your connection.', transport: true);
    }
    return List.of(saved.where((e) => e.cursor > after));
  }

  @override
  Stream<Entry> watch(int after) => events.stream;
  @override
  Future<Entry> send(String text) async {
    if (sendGate != null) await sendGate;
    if (offline) {
      throw const RepositoryFailure('Check your connection.', transport: true);
    }
    if (sendFailure != null) throw sendFailure!;
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
  Future<Entry> setChangeReminders(String changeId, List<int> offsets) async =>
      entry('p', 6, 'preview', change: preview());
  @override
  Future<Entry> undo(String undoId) async =>
      entry('undo', 5, 'result', text: 'Undone: Call plumber.');
  @override
  Future<Entry> actOnReminder(String reminderId, ReminderAction action) async =>
      confirm(reminderId);

  void close() {
    events.close();
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
  test(
    'disposing chat preserves a repository shared with other controllers',
    () async {
      final repository = FakeRepository();
      final controller = ChatController(
        repository,
        pollInterval: const Duration(days: 1),
      );
      await controller.start();
      controller.dispose();
      expect(
        () => repository.events.add(entry('later', 99, 'result')),
        returnsNormally,
      );
      repository.close();
    },
  );

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
  test(
    'a failed send hands the text back and sends again on request',
    () async {
      final repo = FakeRepository();
      final c = ChatController(repo, pollInterval: const Duration(days: 1));
      await c.start();
      repo.offline = true;
      await c.send('Call plumber');
      expect(c.error, isNull);
      expect(c.takeUnsentText(), 'Call plumber');
      expect(c.takeUnsentText(), isNull);
      repo.offline = false;
      await c.send('Call plumber');
      expect(repo.sends, 1);
      repo.events.add(entry('p', 2, 'preview', change: preview()));
      await Future<void>.delayed(Duration.zero);
      await c.discard('p');
      expect(c.entries.last.change?.state, ChangeState.discarded);
      c.dispose();
    },
  );
  test('a delete result drops the cards of the deleted item', () async {
    final repo = FakeRepository();
    final c = ChatController(repo, pollInterval: const Duration(days: 1));
    await c.start();
    repo.saved.addAll([
      entry('p', 1, 'preview', change: preview()),
      entry('r', 2, 'result', text: 'Saved: Call plumber.'),
      entry('other', 3, 'minder', text: 'Hello'),
    ]);
    await c.sync();
    expect(c.entries.map((e) => e.id), ['p', 'r', 'other']);
    repo.saved.add(
      entry(
        'gone',
        4,
        'result',
        text: 'Deleted: Call plumber.',
        change: {
          'id': 'd',
          'action': 'delete_item',
          'item_title': 'Call plumber',
          'fields': [],
          'state': 'confirmed',
          'removed_entry_ids': ['p', 'r'],
        },
      ),
    );
    await c.sync();
    expect(c.entries.map((e) => e.id), ['other', 'gone']);
    c.dispose();
  });
  test('a dropped event stream shows only the header icon', () async {
    final repo = FakeRepository();
    final c = ChatController(
      repo,
      pollInterval: const Duration(days: 1),
      offlineGrace: const Duration(milliseconds: 20),
    );
    await c.start();
    repo.events.addError(Exception('Connection reset by peer'));
    await Future<void>.delayed(const Duration(milliseconds: 60));
    expect(c.error, isNull);
    expect(c.showOffline, isTrue);
    c.dispose();
  });

  test(
    'a message from a background failure clears when sync succeeds',
    () async {
      final repo = FakeRepository();
      final c = ChatController(repo, pollInterval: const Duration(days: 1));
      await c.start();
      repo.historyFailure = const RepositoryFailure('The server is busy.');
      await c.sync();
      expect(c.error, 'The server is busy.');
      repo.historyFailure = null;
      await c.sync();
      expect(c.error, isNull);
      c.dispose();
    },
  );

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
    expect(c.entries.first.change?.state, ChangeState.confirmed);
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
  testWidgets('only the newest failed reply offers Retry', (tester) async {
    tester.view.physicalSize = const Size(412, 892);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    final repo = FakeRepository();
    final c = ChatController(repo, pollInterval: const Duration(days: 1));
    repo.saved.addAll([
      entry('u1', 1, 'user', text: 'Call plumber'),
      entry('e1', 2, 'error', text: 'The model failed.'),
      entry('u2', 3, 'user', text: 'Call dentist'),
      entry('e2', 4, 'error', text: 'The model failed again.'),
    ]);
    await c.start();
    await tester.pumpWidget(
      MaterialApp(
        theme: Tokens.theme,
        home: ChatScreen(controller: c),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Retry'), findsOneWidget);
    repo.saved.add(entry('m', 5, 'minder', text: 'Done.'));
    await c.sync();
    await tester.pumpAndSettle();
    expect(find.text('Retry'), findsNothing);
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
      entryFromJson({
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
  testWidgets('the bottom bar opens the calendar', (tester) async {
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
    expect(find.byTooltip('Calendar'), findsNothing);
    await tester.tap(find.text('Calendar'));
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
        home: ChatScreen(controller: c, onCalendar: () {}),
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

  testWidgets(
    'the list stays at the bottom after sending with the keyboard open',
    (tester) async {
      tester.view.physicalSize = const Size(412, 892);
      tester.view.devicePixelRatio = 1;
      addTearDown(() {
        tester.view.reset();
      });
      final repo = GrowingRepository();
      for (var i = 1; i <= 30; i++) {
        repo.saved.add(entry('m$i', i, 'minder', text: 'Earlier message $i.'));
      }
      final c = ChatController(repo, pollInterval: const Duration(days: 1));
      await c.start();
      await tester.pumpWidget(
        MaterialApp(
          theme: Tokens.theme,
          home: ChatScreen(controller: c, onCalendar: () {}),
        ),
      );
      await tester.pumpAndSettle();
      ScrollPosition position() => tester
          .state<ScrollableState>(
            find.descendant(
              of: find.byType(ListView),
              matching: find.byType(Scrollable),
            ),
          )
          .position;
      expect(position().pixels, position().maxScrollExtent);
      // The keyboard opens. The bottom bar hides.
      tester.view.viewInsets = const FakeViewPadding(bottom: 300);
      await tester.pump(const Duration(milliseconds: 600));
      await tester.pumpAndSettle();
      expect(position().pixels, closeTo(position().maxScrollExtent, 1));
      await tester.enterText(find.byType(TextField), 'Call the plumber');
      await tester.tap(find.byTooltip('Send'));
      await tester.pump(const Duration(milliseconds: 300));
      // The keyboard closes in steps, then the bar comes back.
      for (final inset in [200.0, 100.0, 1.0, 0.0]) {
        tester.view.viewInsets = FakeViewPadding(bottom: inset);
        await tester.pump(const Duration(milliseconds: 100));
      }
      await tester.pump(const Duration(milliseconds: 600));
      expect(find.text('Call the plumber'), findsOneWidget);
      expect(position().pixels, closeTo(position().maxScrollExtent, 1));

      repo.events.add(entry('first-reply', 200, 'minder', text: 'Saved.'));
      await tester.pumpAndSettle();

      // Reading older messages must survive another keyboard resize.
      await tester.drag(find.byType(ListView), const Offset(0, 400));
      await tester.pumpAndSettle();
      final readingOffset = position().pixels;
      expect(position().extentAfter, greaterThan(200));
      tester.view.viewInsets = const FakeViewPadding(bottom: 300);
      await tester.pumpAndSettle();
      expect(position().pixels, closeTo(readingOffset, 1));
      tester.view.viewInsets = const FakeViewPadding();
      await tester.pumpAndSettle();
      expect(position().pixels, closeTo(readingOffset, 1));

      // Scrolling back to the bottom resumes following the viewport.
      await tester.drag(find.byType(ListView), const Offset(0, -1200));
      await tester.pumpAndSettle();
      expect(position().pixels, closeTo(position().maxScrollExtent, 1));
      tester.view.viewInsets = const FakeViewPadding(bottom: 300);
      await tester.pumpAndSettle();
      expect(position().pixels, closeTo(position().maxScrollExtent, 1));

      // A new reply retains the existing automatic scroll behaviour.
      await tester.drag(find.byType(ListView), const Offset(0, 400));
      await tester.pumpAndSettle();
      repo.events.add(entry('reply', 201, 'minder', text: 'Message received.'));
      await tester.pumpAndSettle();
      expect(position().pixels, closeTo(position().maxScrollExtent, 1));

      await tester.pumpWidget(const SizedBox());
      c.dispose();
      repo.close();
    },
  );

  group('connection status', () {
    Future<void> finish(WidgetTester tester, ChatController c) async {
      await tester.pumpWidget(const SizedBox());
      c.dispose();
    }

    Future<(FakeRepository, ChatController)> screen(WidgetTester tester) async {
      tester.view.physicalSize = const Size(412, 892);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.reset);
      final repo = FakeRepository();
      final c = ChatController(
        repo,
        pollInterval: const Duration(days: 1),
        offlineGrace: const Duration(seconds: 2),
      );
      await c.start();
      await tester.pumpWidget(
        MaterialApp(
          theme: Tokens.theme,
          home: ChatScreen(controller: c),
        ),
      );
      return (repo, c);
    }

    testWidgets('a one second drop shows nothing', (tester) async {
      final (repo, c) = await screen(tester);
      repo.offline = true;
      await c.sync();
      await tester.pump(const Duration(seconds: 1));
      repo.offline = false;
      await c.sync();
      await tester.pump(const Duration(seconds: 3));
      expect(c.showOffline, isFalse);
      expect(find.byIcon(Icons.cloud_off_outlined), findsNothing);
      await finish(tester, c);
    });

    testWidgets('a three second drop shows the header icon', (tester) async {
      final (repo, c) = await screen(tester);
      repo.offline = true;
      await c.sync();
      await tester.pump(const Duration(seconds: 3));
      expect(c.showOffline, isTrue);
      expect(find.byIcon(Icons.cloud_off_outlined), findsOneWidget);
      expect(find.bySemanticsLabel('Offline. Tap to retry'), findsOneWidget);
      await finish(tester, c);
    });

    testWidgets('recovery hides the icon', (tester) async {
      final (repo, c) = await screen(tester);
      repo.offline = true;
      await c.sync();
      await tester.pump(const Duration(seconds: 3));
      repo.offline = false;
      await tester.tap(find.byIcon(Icons.cloud_off_outlined));
      await tester.pump();
      await tester.pump();
      expect(c.showOffline, isFalse);
      expect(find.byIcon(Icons.cloud_off_outlined), findsNothing);
      await finish(tester, c);
    });

    testWidgets('no banner or Retry block for transport failures', (
      tester,
    ) async {
      final (repo, c) = await screen(tester);
      repo.offline = true;
      await c.sync();
      await tester.pump(const Duration(seconds: 5));
      expect(c.error, isNull);
      expect(find.text('Check your connection. Then try again.'), findsNothing);
      expect(find.text('Retry'), findsNothing);
      await finish(tester, c);
    });

    testWidgets('a failed send puts the text back and shows no Retry block', (
      tester,
    ) async {
      final (repo, c) = await screen(tester);
      repo.offline = true;
      await tester.enterText(find.byType(TextField), 'Call plumber');
      await tester.tap(find.byTooltip('Send'));
      await tester.pump();
      await tester.pump();
      expect(c.error, isNull);
      expect(find.text('Retry'), findsNothing);
      expect(
        tester.widget<TextField>(find.byType(TextField)).controller!.text,
        'Call plumber',
      );
      await finish(tester, c);
    });

    testWidgets('restored text goes in front of what was typed meanwhile', (
      tester,
    ) async {
      final (repo, c) = await screen(tester);
      repo.offline = true;
      final gate = Completer<void>();
      repo.sendGate = gate.future;
      await tester.enterText(find.byType(TextField), 'Call plumber');
      await tester.tap(find.byTooltip('Send'));
      await tester.pump();
      await tester.enterText(find.byType(TextField), 'and the dentist');
      gate.complete();
      await tester.pump();
      await tester.pump();
      expect(
        tester.widget<TextField>(find.byType(TextField)).controller!.text,
        'Call plumber and the dentist',
      );
      await finish(tester, c);
    });

    testWidgets(
      'a server refusal shows its message without Retry until the next send',
      (tester) async {
        final (repo, c) = await screen(tester);
        repo.sendFailure = const RepositoryFailure('Monthly limit reached.');
        await tester.enterText(find.byType(TextField), 'Call plumber');
        await tester.tap(find.byTooltip('Send'));
        await tester.pump();
        await tester.pump();
        expect(find.text('Monthly limit reached.'), findsOneWidget);
        expect(find.text('Retry'), findsNothing);
        repo.sendFailure = null;
        await c.sync();
        await tester.pump();
        expect(find.text('Monthly limit reached.'), findsOneWidget);
        await tester.enterText(find.byType(TextField), 'Call plumber');
        await tester.tap(find.byTooltip('Send'));
        await tester.pump();
        await tester.pump();
        expect(find.text('Monthly limit reached.'), findsNothing);
        await finish(tester, c);
      },
    );
  });
}

class GrowingRepository extends FakeRepository {
  @override
  Future<Entry> send(String text) async {
    sends++;
    return entry('u$sends', 100 + sends, 'user', text: text);
  }
}

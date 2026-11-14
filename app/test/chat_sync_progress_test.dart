import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:minder/application/chat_controller.dart';
import 'package:minder/data/wire.dart';
import 'package:minder/domain/chat_repository.dart';
import 'package:minder/domain/entry.dart';
import 'package:minder/ui/chat_screen.dart';
import 'package:minder/ui/tokens.dart';

Entry _entry(
  String id,
  int cursor,
  String kind, {
  String text = '',
  String? state,
  String? captureStatus,
}) => entryFromJson({
  'id': id,
  'cursor': cursor,
  'kind': kind,
  'text': text,
  'created_at': '2026-11-14T16:00:00Z',
  if (state != null)
    'change': {
      'id': 'change',
      'action': 'create_item',
      'item_title': 'First appointment',
      'state': state,
      'fields': [],
    },
  if (captureStatus != null) ...{
    'capture_id': 'email',
    'capture': {'id': 'email', 'revision': 1, 'status': captureStatus},
  },
});

class ReviewRepository extends Fake implements ChatRepository {
  final saved = <Entry>[];
  final events = StreamController<Entry>.broadcast();
  final historyAfter = <int>[];
  Completer<void>? historyGate;
  @override
  Future<List<Entry>> history(int after) async {
    historyAfter.add(after);
    final snapshot = saved.where((e) => e.cursor > after).toList();
    await historyGate?.future;
    return snapshot;
  }

  @override
  Stream<Entry> watch(int after) => events.stream;
  @override
  Future<Entry> confirm(String changeId) async {
    final result = _entry(
      'result',
      12,
      'result',
      text: 'Saved: First appointment.',
    );
    saved.addAll([
      _entry('preview', 11, 'preview', state: 'confirmed'),
      result,
      _entry('next', 13, 'minder', text: 'Item 2 of 4'),
    ]);
    return result; // HTTP wins the race; the event stream has not delivered anything.
  }

  @override
  Future<Entry> send(String text) async =>
      _entry('user', 30, 'user', text: text);
}

ChatController _controller(ReviewRepository repo) =>
    ChatController(repo, pollInterval: const Duration(days: 1));

Future<void> _show(WidgetTester tester, ChatController chat) =>
    tester.pumpWidget(
      MaterialApp(
        theme: Tokens.theme,
        home: ChatScreen(controller: chat),
      ),
    );

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
  testWidgets(
    'Confirm resolves the original card when HTTP arrives before its events',
    (tester) async {
      final repo = ReviewRepository()
        ..saved.add(_entry('preview', 10, 'preview', state: 'pending'));
      final chat = _controller(repo);
      addTearDown(repo.events.close);
      await chat.start();
      await _show(tester, chat);
      await tester.pumpAndSettle();
      await tester.tap(find.text('Confirm'));
      await tester.pumpAndSettle();
      expect(find.text('Confirmed'), findsOneWidget);
      expect(find.text('Confirm'), findsNothing);
      expect(find.text('Item 2 of 4'), findsOneWidget);
      chat.dispose();
      await tester.pumpWidget(const SizedBox());
    },
  );

  test(
    'an answer response cannot skip the earlier closed-question event',
    () async {
      final repo = ReviewRepository()
        ..saved.add(
          _entry('question', 20, 'question', captureStatus: 'needs-answer'),
        );
      final chat = _controller(repo);
      addTearDown(chat.dispose);
      addTearDown(repo.events.close);
      await chat.start();
      final processing = _entry(
        'capture',
        22,
        'capture',
        captureStatus: 'received',
      );
      repo.saved.addAll([
        _entry('question', 21, 'minder', text: 'Questions answered.'),
        processing,
      ]);
      chat.receiveCaptureEntry(processing);
      await chat.sync();
      expect(chat.entries.first.kind, EntryKind.minder);
      expect(chat.entries.first.text, 'Questions answered.');
    },
  );

  test('an action during an in-flight poll leaves earlier events available for the next sync', () async {
    final repo = ReviewRepository()
      ..saved.add(_entry('preview', 10, 'preview', state: 'pending'));
    final chat = _controller(repo);
    addTearDown(chat.dispose);
    addTearDown(repo.events.close);
    await chat.start();
    final gate = repo.historyGate = Completer<void>();
    final polling = chat.sync();
    await chat.confirm('change');
    gate.complete();
    await polling;
    repo.historyGate = null;
    await chat.sync();
    expect(chat.entries.first.change!.state, ChangeState.confirmed);
  });

  testWidgets(
    'Working covers initial extraction and answers until ready or failed',
    (tester) async {
      final repo = ReviewRepository();
      final chat = _controller(repo);
      addTearDown(repo.events.close);
      await chat.start();
      await _show(tester, chat);
      expect(find.text('Working'), findsNothing);
      var cursor = 1;
      for (final status in [
        'received',
        'processing',
        'ready-for-review',
        'received',
        'failed',
      ]) {
        chat.receiveCaptureEntry(
          _entry('capture', cursor++, 'capture', captureStatus: status),
        );
        await tester.pump();
        expect(
          find.text('Working'),
          status == 'received' || status == 'processing'
              ? findsOneWidget
              : findsNothing,
          reason: status,
        );
      }
      chat.dispose();
      await tester.pumpWidget(const SizedBox());
    },
  );

  testWidgets(
    'Working stays visible while reading earlier messages in a long review',
    (tester) async {
      final repo = ReviewRepository()
        ..saved.addAll([
          _entry('capture', 1, 'capture', captureStatus: 'received'),
          for (var n = 2; n < 32; n++)
            _entry('message$n', n, 'minder', text: 'Review note $n. ' * 12),
        ]);
      final chat = _controller(repo);
      addTearDown(repo.events.close);
      await chat.start();
      await _show(tester, chat);
      await tester.pump(const Duration(milliseconds: 500));
      await tester.drag(find.byType(ListView), const Offset(0, 5000));
      await tester.pump(const Duration(milliseconds: 500));
      expect(find.text('Working'), findsOneWidget);
      final status = tester.getRect(find.text('Working'));
      final composer = tester.getRect(find.byKey(const ValueKey('composer')));
      expect(status.bottom, lessThanOrEqualTo(composer.top));
      expect(status.top, greaterThanOrEqualTo(0));
      chat.dispose();
      await tester.pumpWidget(const SizedBox());
    },
  );

  testWidgets('a finished email does not hide normal chat thinking', (
    tester,
  ) async {
    final repo = ReviewRepository();
    final chat = _controller(repo);
    addTearDown(repo.events.close);
    await chat.start();
    await chat.send('Add a task');
    chat.receiveCaptureEntry(
      _entry('capture', 31, 'capture', captureStatus: 'ready-for-review'),
    );
    await _show(tester, chat);
    expect(find.text('Working'), findsOneWidget);
    chat.dispose();
    await tester.pumpWidget(const SizedBox());
  });
}

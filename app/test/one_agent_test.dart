import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:minder/application/capture_controller.dart';
import 'package:minder/application/chat_controller.dart';
import 'package:minder/data/wire.dart';
import 'package:minder/domain/capture.dart';
import 'package:minder/domain/capture_repository.dart';
import 'package:minder/domain/entry.dart';
import 'package:minder/domain/failure.dart';
import 'package:minder/ui/capture_view.dart';
import 'package:minder/ui/chat_screen.dart';
import 'package:minder/ui/tokens.dart';

import 'chat_test.dart' show FakeRepository, entry;

/// A chat repository that also answers reviews, like the real API.
class ReviewRepository extends FakeRepository implements CaptureRepository {
  final answers = <String>[];
  @override
  Future<Entry> answerCapture(
    String id,
    int revision,
    String text, {
    bool acknowledgeCoverage = false,
    String? candidateId,
  }) async {
    answers.add('$candidateId:$text');
    return entry(
      'answer${answers.length}',
      50 + answers.length,
      'user',
      text: text,
    );
  }

  @override
  Future<Capture> getCapture(String id) => throw UnimplementedError();
  @override
  Future<Entry> dismissCapture(String id, int revision) =>
      throw UnimplementedError();
  @override
  Future<Entry> retryCapture(String id) => throw UnimplementedError();
  @override
  Future<Entry> deleteCapture(String id, int revision) =>
      throw UnimplementedError();
  @override
  Future<AttachmentDownload> downloadAttachment(
    String id,
    CaptureAttachment attachment,
  ) => throw UnimplementedError();
  @override
  Future<void> saveAttachment(AttachmentDownload file) =>
      throw UnimplementedError();
}

const _openReview = Capture(
  id: 'source',
  status: 'needs-answer',
  revision: 3,
  subject: 'Checkups',
  candidates: [
    CaptureCandidate(
      id: 'a',
      state: 'needs-answer',
      title: 'Dentist',
      question: 'How long will it take?',
      options: ['30 minutes', '1 hour'],
    ),
  ],
);

void main() {
  group('long messages', () {
    test('a paste up to 400,000 bytes is sent whole', () async {
      final repo = FakeRepository();
      final c = ChatController(repo, pollInterval: const Duration(days: 1));
      await c.start();
      final paste = 'é' * 200000; // 400,000 bytes
      await c.send(paste);
      expect(repo.sends, 1);
      expect(c.entries.single.text, paste);
      expect(c.error, isNull);
      c.dispose();
    });

    test('a paste over 400,000 bytes is refused, visibly, and kept', () async {
      final repo = FakeRepository();
      final c = ChatController(repo, pollInterval: const Duration(days: 1));
      await c.start();
      final paste = '${'é' * 200000}x';
      await c.send(paste);
      expect(repo.sends, 0);
      expect(c.error, contains('400,000'));
      expect(c.waiting, isFalse);
      expect(c.takeUnsentText(), paste);
      c.dispose();
    });

    testWidgets(
      'the composer does not cut a pasted email at 4,000 characters',
      (tester) async {
        tester.view.physicalSize = const Size(412, 892);
        tester.view.devicePixelRatio = 1;
        addTearDown(tester.view.reset);
        final repo = FakeRepository();
        final c = ChatController(repo, pollInterval: const Duration(days: 1));
        await c.start();
        await tester.pumpWidget(
          MaterialApp(
            theme: Tokens.theme,
            home: ChatScreen(controller: c),
          ),
        );
        final paste = 'Dentist on Friday. ' * 400; // 7,600 characters
        await tester.enterText(find.byType(TextField), paste);
        await tester.tap(find.byTooltip('Send'));
        await tester.pump();
        expect(c.entries.single.text, paste.trim());
        await tester.pumpWidget(const SizedBox());
        c.dispose();
      },
    );
  });

  group('Working', () {
    test('a quick reply keeps Working until the agent answers', () async {
      final repo = ReviewRepository();
      final c = ChatController(repo, pollInterval: const Duration(days: 1));
      await c.start();
      final controller = CaptureController(repo, c.receiveCaptureEntry);
      await controller.answer(_openReview, '1 hour', candidateId: 'a');
      expect(repo.answers, ['a:1 hour']);
      expect(c.entries.last.kind, EntryKind.user);
      expect(c.thinking, isTrue, reason: 'the reply is queued, not answered');
      repo.events.add(entry('m', 60, 'minder', text: 'Got it.'));
      await Future<void>.delayed(Duration.zero);
      expect(c.thinking, isFalse);
      controller.dispose();
      c.dispose();
    });

    test(
      'Working survives a reconnect while the owner is the last to speak',
      () async {
        final repo = FakeRepository()
          ..saved.add(entry('u', 1, 'user', text: 'Where is it?'));
        final c = ChatController(repo, pollInterval: const Duration(days: 1));
        await c.start();
        expect(c.waiting, isFalse);
        expect(c.thinking, isTrue);
        c.dispose();
      },
    );

    test('a failed turn posts an error entry that ends Working', () async {
      final repo = FakeRepository()
        ..saved.add(entry('u', 1, 'user', text: 'Hello'));
      final c = ChatController(repo, pollInterval: const Duration(days: 1));
      await c.start();
      expect(c.thinking, isTrue);
      repo.events.add(
        entry(
          'u-reply',
          2,
          'error',
          text: 'Minder could not finish. Try again.',
        ),
      );
      await Future<void>.delayed(Duration.zero);
      expect(c.thinking, isFalse);
      expect(c.entries.last.kind, EntryKind.error);
      c.dispose();
    });

    test(
      'a typed follow-up during an open review is an ordinary message',
      () async {
        final repo = ReviewRepository();
        final c = ChatController(repo, pollInterval: const Duration(days: 1));
        await c.start();
        await c.send('Where is the dentist?');
        expect(repo.sends, 1);
        expect(repo.answers, isEmpty);
        c.dispose();
      },
    );
  });

  group('source cards', () {
    test('an older card without a kind is a forwarded email', () {
      final capture = captureFromJson({
        'id': 'c',
        'status': 'reviewed',
        'revision': 1,
      });
      expect(capture.pasted, isFalse);
    });

    test('a review of pasted text says so', () {
      final capture = captureFromJson({
        'id': 'c',
        'kind': 'chat',
        'status': 'reviewed',
        'revision': 1,
      });
      expect(capture.pasted, isTrue);
    });

    Future<void> pump(WidgetTester tester, Capture capture) async {
      final repo = ReviewRepository();
      final c = ChatController(repo, pollInterval: const Duration(days: 1));
      await tester.pumpWidget(
        MaterialApp(
          theme: Tokens.theme,
          home: Scaffold(
            body: SingleChildScrollView(
              child: CaptureView(capture: capture, chat: c),
            ),
          ),
        ),
      );
      addTearDown(() async {
        await tester.pumpWidget(const SizedBox());
        c.dispose();
      });
    }

    testWidgets('a forwarded email keeps its email wording', (tester) async {
      await pump(tester, _openReview);
      expect(find.text('EMAIL'), findsOneWidget);
      expect(find.text('Email review'), findsOneWidget);
      expect(find.text('Delete email source…'), findsOneWidget);
    });

    testWidgets('pasted text is not called an email', (tester) async {
      const pasted = Capture(
        id: 'chat',
        kind: 'chat',
        status: 'needs-answer',
        revision: 1,
        candidates: [
          CaptureCandidate(id: 'a', state: 'pending', title: 'Dentist'),
        ],
      );
      await pump(tester, pasted);
      expect(find.text('PASTED TEXT'), findsOneWidget);
      expect(find.text('Review of pasted text'), findsOneWidget);
      expect(find.text('Delete source…'), findsOneWidget);
      expect(find.textContaining('email'), findsNothing);
    });
  });

  test('the answer to a quick reply decodes as the owner entry', () {
    final e = entryFromJson(
      jsonDecode(
        '{"id":"x","cursor":5,"kind":"user","text":"1 hour","created_at":"2026-11-14T10:00:00Z"}',
      ) as Map<String, dynamic>,
    );
    expect(e.kind, EntryKind.user);
  });

  test('a failure the server reports keeps Working off', () async {
    final repo = FakeRepository()
      ..sendFailure = const RepositoryFailure('Wait for Minder to finish.');
    final c = ChatController(repo, pollInterval: const Duration(days: 1));
    await c.start();
    await c.send('Hello');
    expect(c.waiting, isFalse);
    expect(c.error, 'Wait for Minder to finish.');
    c.dispose();
  });
}

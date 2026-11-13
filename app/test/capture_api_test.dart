import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:minder/application/chat_controller.dart';
import 'package:minder/data/chat_api.dart';
import 'package:minder/domain/capture.dart';
import 'package:minder/ui/capture_view.dart';
import 'package:minder/ui/question_view.dart';
import 'package:minder/ui/tokens.dart';
import 'package:minder/ui/item_sources.dart';

void main() {
  testWidgets('a targeted email answer preserves unrelated chat waiting', (
    tester,
  ) async {
    final api = ChatApi(
      'https://example.test',
      client: MockClient((request) async {
        expect(request.url.path, '/v1/captures/source/answers');
        final input = jsonDecode(request.body) as Map<String, dynamic>;
        expect(input['revision'], 2);
        expect(input['text'], '21 November');
        return http.Response(
          jsonEncode({
            'id': 'capture-entry',
            'cursor': 9,
            'kind': 'capture',
            'text': 'Processing',
            'created_at': '2026-11-12T10:00:00Z',
            'capture': {
              'id': 'source',
              'revision': 3,
              'status': 'received',
              'candidates': [],
              'attachments': [],
            },
          }),
          200,
        );
      }),
    );
    final chat = ChatController(api)..waiting = true;
    await tester.pumpWidget(
      MaterialApp(
        theme: Tokens.theme,
        home: Scaffold(
          body: SingleChildScrollView(
            child: QuestionView(
              capture: const Capture(
                id: 'source',
                status: 'needs-answer',
                revision: 2,
                questions: [CaptureQuestion(text: 'What date?')],
              ),
              chat: chat,
            ),
          ),
        ),
      ),
    );
    await tester.enterText(find.byType(TextField), '21 November');
    await tester.tap(find.text('Send answer'));
    await tester.pumpAndSettle();
    expect(chat.entries.single.capture!.revision, 3);
    expect(chat.waiting, isTrue);
    await tester.pumpWidget(const SizedBox());
    chat.dispose();
    api.close();
  });
  testWidgets(
    'Do nothing closes an open email and zero-item emails read clearly',
    (tester) async {
      final api = ChatApi(
        'https://example.test',
        client: MockClient((request) async {
          expect(request.url.path, '/v1/captures/source/dismiss');
          expect(jsonDecode(request.body), {'revision': 2});
          return http.Response(
            jsonEncode({
              'id': 'capture-entry',
              'cursor': 9,
              'kind': 'capture',
              'text': 'Nothing to do. The email is kept.',
              'created_at': '2026-11-12T10:00:00Z',
              'capture': {
                'id': 'source',
                'revision': 2,
                'status': 'reviewed',
                'candidates': [],
                'attachments': [],
              },
            }),
            200,
          );
        }),
      );
      final chat = ChatController(api);
      Future<void> show(Capture capture) => tester.pumpWidget(
        MaterialApp(
          theme: Tokens.theme,
          home: Scaffold(
            body: SingleChildScrollView(
              child: CaptureView(capture: capture, chat: chat),
            ),
          ),
        ),
      );
      await show(
        const Capture(id: 'source', status: 'ready-for-review', revision: 2),
      );
      expect(find.text('Review the proposals in chat.'), findsNothing);
      expect(
        find.text('No actionable items found. Tell Minder what to capture.'),
        findsOneWidget,
      );
      expect(tester.getSize(find.text('Do nothing')).height, greaterThan(0));
      await tester.ensureVisible(find.text('Do nothing'));
      await tester.tap(find.text('Do nothing'));
      await tester.pumpAndSettle();
      expect(chat.entries.single.capture!.status, 'reviewed');
      await show(const Capture(id: 'source', status: 'reviewed', revision: 2));
      expect(find.text('Do nothing'), findsNothing);
      await tester.pumpWidget(const SizedBox());
      chat.dispose();
      api.close();
    },
  );
  test(
    'attachments use owner authentication and preserve binary bytes',
    () async {
      final api = ChatApi(
        'https://example.test',
        authorization: () async => {'Authorization': 'Bearer fixture'},
        client: MockClient((request) async {
          expect(request.url.path, '/v1/captures/source/attachments/binary');
          expect(request.headers['Authorization'], 'Bearer fixture');
          return http.Response.bytes([0, 255, 42], 200);
        }),
      );
      final file = await api.downloadAttachment(
        'source',
        const CaptureAttachment(
          id: 'binary',
          name: 'ticket.bin',
          mediaType: 'application/octet-stream',
          size: 3,
        ),
      );
      expect(file.bytes, [0, 255, 42]);
      expect(file.name, 'ticket.bin');
      api.close();
    },
  );
  testWidgets('calendar sources show the original email and attachments', (
    tester,
  ) async {
    final api = ChatApi(
      'https://example.test',
      client: MockClient((request) async {
        expect(request.url.path, '/v1/captures/source');
        return http.Response(
          jsonEncode({
            'id': 'source',
            'status': 'reviewed',
            'revision': 1,
            'subject': 'Hotel confirmation',
            'attachments': [
              {
                'id': 'original',
                'name': 'original.eml',
                'media_type': 'message/rfc822',
                'size': 32,
              },
            ],
            'candidates': [],
          }),
          200,
        );
      }),
    );
    await tester.pumpWidget(
      MaterialApp(
        theme: Tokens.theme,
        home: Scaffold(
          body: ItemSources(repository: api, sourceIds: const ['source']),
        ),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('Hotel confirmation'), findsOneWidget);
    expect(find.text('original.eml'), findsOneWidget);
    await tester.pumpWidget(const SizedBox());
    api.close();
  });
}

import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:minder/application/chat_controller.dart';
import 'package:minder/data/chat_api.dart';
import 'package:minder/data/wire.dart';
import 'package:minder/ui/chat_screen.dart';
import 'package:minder/ui/tokens.dart';

Map<String, dynamic> question(int revision) => {
  'id': 'question-$revision',
  'cursor': revision * 10,
  'kind': 'question',
  'text': 'Questions about the email.',
  'capture_id': 'source',
  'created_at': '2026-11-14T16:00:00Z',
  'capture': {
    'id': 'source',
    'revision': revision,
    'status': 'needs-answer',
    'candidates': [
      {
        'id': 'task',
        'state': 'needs-answer',
        'question': 'When will you do it?',
        'options': ['1 day before', '3 days before'],
        'proposal': {'title': 'Complete questionnaire', 'kind': 'task'},
      },
    ],
  },
};

http.Response accepted() => http.Response(
  jsonEncode({
    'id': 'answer',
    'cursor': 15,
    'kind': 'user',
    'text': '3 days before',
    'created_at': '2026-11-14T16:01:00Z',
  }),
  200,
);

Future<ChatController> showChat(WidgetTester tester, http.Client client) async {
  final api = ChatApi('https://example.test', client: client);
  final chat = ChatController(api)..loading = false;
  chat.receiveCaptureEntry(entryFromJson(question(1)));
  addTearDown(() {
    chat.dispose();
    api.close();
  });
  await tester.pumpWidget(
    MaterialApp(
      theme: Tokens.theme,
      home: ChatScreen(controller: chat),
    ),
  );
  await tester.pump();
  return chat;
}

Finder choice(String text) => find.widgetWithText(OutlinedButton, text);

void main() {
  setUp(() {
    TestWidgetsFlutterBinding.ensureInitialized()
        .platformDispatcher
        .accessibilityFeaturesTestValue = const FakeAccessibilityFeatures(
      disableAnimations: true,
    );
  });
  tearDown(
    () => TestWidgetsFlutterBinding.instance.platformDispatcher
        .clearAccessibilityFeaturesTestValue(),
  );

  testWidgets(
    'custom review replies use the one neutral composer and hide choices immediately',
    (tester) async {
      final response = Completer<http.Response>();
      final requests = <http.Request>[];
      final chat = await showChat(
        tester,
        MockClient((request) {
          requests.add(request);
          return response.future;
        }),
      );
      expect(find.byType(TextField), findsOneWidget);
      expect(find.text('Send answer'), findsNothing);
      expect(find.text('Or give a custom response below.'), findsOneWidget);
      await tester.enterText(find.byType(TextField), 'Actually, 3 days before');
      expect(
        tester.widget<EditableText>(find.byType(EditableText)).style.color,
        Tokens.text,
      );
      await tester.tap(find.widgetWithText(FilledButton, 'Send'));
      await tester.pump();
      expect(choice('1 day before'), findsNothing);
      expect(choice('3 days before'), findsNothing);
      expect(requests.single.url.path, '/v1/captures/source/answers');
      expect(
        jsonDecode(requests.single.body),
        containsPair('candidate_id', 'task'),
      );
      expect(
        jsonDecode(requests.single.body),
        containsPair('text', 'Actually, 3 days before'),
      );
      response.complete(accepted());
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 250));
      expect(choice('1 day before'), findsNothing);
      chat.receiveCaptureEntry(entryFromJson(question(2)));
      await tester.pumpAndSettle();
      expect(choice('1 day before'), findsOneWidget);
      expect(chat.waiting, isFalse);
      expect(
        tester.widget<OutlinedButton>(choice('1 day before')).onPressed,
        isNotNull,
      );
    },
  );

  testWidgets('tapping a choice hides it before the answer request finishes', (
    tester,
  ) async {
    final response = Completer<http.Response>();
    final requests = <http.Request>[];
    await showChat(
      tester,
      MockClient((request) {
        requests.add(request);
        return response.future;
      }),
    );
    await tester.ensureVisible(choice('3 days before'));
    await tester.tap(choice('3 days before'));
    await tester.pump();
    expect(choice('1 day before'), findsNothing);
    expect(choice('3 days before'), findsNothing);
    expect(requests.length, 1);
    response.complete(accepted());
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 250));
    expect(choice('3 days before'), findsNothing);
  });

  testWidgets('a refused custom reply restores the choices and composer text', (
    tester,
  ) async {
    final response = Completer<http.Response>();
    await showChat(tester, MockClient((_) => response.future));
    await tester.enterText(find.byType(TextField), '3 days before');
    await tester.tap(find.widgetWithText(FilledButton, 'Send'));
    await tester.pump();
    response.complete(http.Response('{"error":"Try again"}', 409));
    await tester.pumpAndSettle();
    expect(choice('1 day before'), findsOneWidget);
    expect(
      tester.widget<EditableText>(find.byType(EditableText)).controller.text,
      '3 days before',
    );
  });
}

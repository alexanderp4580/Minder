import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:minder/application/chat_controller.dart';
import 'package:minder/data/chat_api.dart';
import 'package:minder/data/wire.dart';
import 'package:minder/ui/capture_view.dart';
import 'package:minder/ui/tokens.dart';

Future<void> _pump(
  WidgetTester tester,
  Widget Function(ChatController) build,
) async {
  final api = ChatApi(
    'https://example.test',
    client: MockClient((_) async => http.Response(jsonEncode({}), 200)),
  );
  final chat = ChatController(api, pollInterval: const Duration(days: 1));
  addTearDown(() {
    chat.dispose();
    api.close();
  });
  await tester.pumpWidget(
    MaterialApp(
      theme: Tokens.theme,
      home: Scaffold(body: SingleChildScrollView(child: build(chat))),
    ),
  );
}

void main() {
  test(
    'the capture wire carries the unverified count and queued candidates',
    () {
      final capture = captureFromJson({
        'id': 'mail',
        'status': 'needs-answer',
        'unverified': 2,
        'candidates': [
          {
            'id': 'a',
            'state': 'pending',
            'proposal': {'title': 'Dentist'},
          },
          {
            'id': 'b',
            'state': 'queued',
            'proposal': {'title': 'Checkup'},
          },
        ],
      });
      expect(capture.unverified, 2);
      expect(capture.candidates.last.state, 'queued');
      expect(captureFromJson({'id': 'x', 'status': 'reviewed'}).unverified, 0);
    },
  );

  testWidgets(
    'the email card shows what waits and what could not be verified',
    (tester) async {
      final capture = captureFromJson({
        'id': 'mail',
        'status': 'ready-for-review',
        'revision': 2,
        'unverified': 2,
        'candidates': [
          {
            'id': 'a',
            'state': 'pending',
            'proposal': {'title': 'Dentist'},
          },
          {
            'id': 'b',
            'state': 'queued',
            'proposal': {'title': 'Checkup'},
          },
        ],
      });
      await _pump(tester, (chat) => CaptureView(capture: capture, chat: chat));
      expect(find.text('Dentist'), findsOneWidget);
      expect(find.textContaining('Waiting'), findsOneWidget);
      expect(
        find.text('I could not verify 2 appointments from this email.'),
        findsOneWidget,
      );
    },
  );

  testWidgets('a single unverified appointment reads in the singular', (
    tester,
  ) async {
    final capture = captureFromJson({
      'id': 'mail',
      'status': 'ready-for-review',
      'unverified': 1,
      'candidates': [
        {
          'id': 'a',
          'state': 'pending',
          'proposal': {'title': 'Dentist'},
        },
      ],
    });
    await _pump(tester, (chat) => CaptureView(capture: capture, chat: chat));
    expect(
      find.text('I could not verify 1 appointment from this email.'),
      findsOneWidget,
    );
  });
}

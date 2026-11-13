import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:minder/application/chat_controller.dart';
import 'package:minder/data/chat_api.dart';
import 'package:minder/data/wire.dart';
import 'package:minder/domain/capture.dart';
import 'package:minder/domain/capture_questions.dart';
import 'package:minder/ui/capture_view.dart';
import 'package:minder/ui/question_view.dart';
import 'package:minder/ui/tokens.dart';

const _fitting = CaptureCandidate(
  id: 'a',
  state: 'needs-answer',
  title: 'Fitting',
  startAt: '2026-11-20T14:30:00+01:00',
  location: 'Kinkerstraat 123, Amsterdam',
  question: 'How long will it take?',
  options: ['30 minutes', '1 hour'],
);
const _open = Capture(
  id: 'source',
  status: 'needs-answer',
  revision: 3,
  subject: 'Fwd: Fitting',
  candidates: [_fitting],
  questions: [
    CaptureQuestion(
      id: 'q1',
      text: 'Is the location important?',
      options: ['Skip'],
    ),
  ],
);

Future<ChatController> _pump(
  WidgetTester tester,
  Widget Function(ChatController) build,
  http.Client client,
) async {
  final api = ChatApi('https://example.test', client: client);
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
  return chat;
}

http.Response _reply() => http.Response(
  jsonEncode({
    'id': 'capture-entry',
    'cursor': 9,
    'kind': 'capture',
    'text': 'Processing',
    'created_at': '2026-11-13T10:00:00Z',
    'capture': {
      'id': 'source',
      'revision': 4,
      'status': 'received',
      'candidates': [],
      'attachments': [],
    },
  }),
  200,
);

void main() {
  test('duration choices preserve days and nights', () {
    expect(
      questionOptions('How long is your stay?', ['2 nights', '3 nights']),
      ['2 nights', '3 nights'],
    );
  });

  test('questions decode from objects and from the older bare strings', () {
    final capture = captureFromJson({
      'id': 'c',
      'status': 'needs-answer',
      'questions': [
        'What date?',
        {
          'id': 'q2',
          'text': 'Where?',
          'options': ['Skip'],
        },
      ],
      'candidates': [
        {
          'id': 'a',
          'state': 'needs-answer',
          'proposal': {'title': 'Fitting', 'location': 'Amsterdam'},
          'options': ['1 hour'],
          'question': 'How long?',
        },
      ],
    });
    expect(capture.questions.first.text, 'What date?');
    expect(capture.questions.last.options, ['Skip']);
    expect(capture.candidates.single.location, 'Amsterdam');
    expect(capture.candidates.single.options, ['1 hour']);
  });

  testWidgets(
    'expanded email shows parsed English details and duration suggestions',
    (tester) async {
      final semantics = tester.ensureSemantics();
      final capture = captureFromJson({
        'id': 'source',
        'status': 'needs-answer',
        'revision': 3,
        'subject': 'Afspraak bevestiging',
        'candidates': [
          {
            'id': 'a',
            'state': 'needs-answer',
            'question': 'How long will it take?',
            'proposal': {
              'title': 'Fitting',
              'kind': 'event',
              'location': 'Clinic address',
              'phone': '020-1234567',
              'booking_reference': '001234',
              'notes': '- Bring your glasses',
            },
            'evidence': [
              {'part': '1_1', 'excerpt': 'Neem je bril mee'},
            ],
          },
        ],
      });
      final answers = <Map<String, dynamic>>[];
      await _pump(
        tester,
        (chat) => CaptureView(capture: capture, chat: chat),
        MockClient((request) async {
          expect(request.url.path, '/v1/captures/source/answers');
          answers.add(jsonDecode(request.body) as Map<String, dynamic>);
          return _reply();
        }),
      );
      await tester.tap(find.text('Fitting'));
      await tester.pumpAndSettle();
      expect(find.text('Phone'), findsOneWidget);
      expect(find.text('020-1234567'), findsOneWidget);
      expect(find.text('Booking reference'), findsOneWidget);
      expect(find.text('001234'), findsOneWidget);
      expect(find.bySemanticsLabel(RegExp('001234')), findsOneWidget);
      expect(find.text('Bring your glasses'), findsOneWidget);
      expect(find.textContaining('1_1:'), findsNothing);
      expect(find.textContaining('Neem je bril'), findsNothing);
      expect(find.text('Afspraak bevestiging'), findsNothing);
      for (final option in ['30 minutes', '45 minutes', '1 hour']) {
        expect(find.text(option), findsOneWidget);
      }
      await tester.ensureVisible(find.text('30 minutes'));
      await tester.tap(find.text('30 minutes'));
      await tester.pumpAndSettle();
      expect(answers.single['text'], '30 minutes');
      expect(answers.single['candidate_id'], 'a');
      expect(answers.single['revision'], 3);
      semantics.dispose();
    },
  );

  testWidgets(
    'duration questions get useful replies when older data has no options',
    (tester) async {
      final capture = Capture(
        id: 'source',
        status: 'needs-answer',
        revision: 3,
        candidates: [
          CaptureCandidate(
            id: 'a',
            state: 'needs-answer',
            title: 'Fitting',
            question: 'How long will it take?',
          ),
        ],
      );
      await _pump(
        tester,
        (chat) => QuestionView(capture: capture, chat: chat),
        MockClient((_) async => _reply()),
      );
      for (final option in ['30 minutes', '45 minutes', '1 hour']) {
        expect(find.text(option), findsOneWidget);
      }
    },
  );

  testWidgets('email card lists found items without questions', (tester) async {
    await _pump(
      tester,
      (chat) => CaptureView(capture: _open, chat: chat),
      MockClient((_) async => _reply()),
    );
    expect(find.text('Fitting'), findsOneWidget);
    expect(find.textContaining('Fri 20 Nov 14:30'), findsOneWidget);
    expect(find.textContaining('Kinkerstraat 123, Amsterdam'), findsOneWidget);
    expect(find.text('How long will it take?'), findsNothing);
    expect(find.text('Is the location important?'), findsNothing);
    expect(find.text('Send answer'), findsNothing);
    expect(find.text('Do nothing'), findsOneWidget);
  });

  testWidgets('question entry shows each question with its quick replies', (
    tester,
  ) async {
    await _pump(
      tester,
      (chat) => QuestionView(capture: _open, chat: chat),
      MockClient((_) async => _reply()),
    );
    expect(find.text('How long will it take?'), findsOneWidget);
    expect(find.text('30 minutes'), findsOneWidget);
    expect(find.text('1 hour'), findsOneWidget);
    expect(find.text('Is the location important?'), findsOneWidget);
    expect(find.text('Skip'), findsOneWidget);
  });

  testWidgets('a quick reply is sent as the answer with the revision', (
    tester,
  ) async {
    final bodies = <Map<String, dynamic>>[];
    await _pump(
      tester,
      (chat) => QuestionView(capture: _open, chat: chat),
      MockClient((request) async {
        expect(request.url.path, '/v1/captures/source/answers');
        bodies.add(jsonDecode(request.body) as Map<String, dynamic>);
        return _reply();
      }),
    );
    await tester.tap(find.text('1 hour'));
    await tester.pumpAndSettle();
    expect(bodies.single['revision'], 3);
    expect(bodies.single['text'], '1 hour');
    expect(bodies.single['candidate_id'], 'a');
    await tester.tap(find.text('Skip'));
    await tester.pumpAndSettle();
    expect(bodies.last['text'], 'Skip');
    expect(bodies.last.containsKey('candidate_id'), isFalse);
  });

  testWidgets('one typed reply can answer several questions', (tester) async {
    final bodies = <Map<String, dynamic>>[];
    await _pump(
      tester,
      (chat) => QuestionView(capture: _open, chat: chat),
      MockClient((request) async {
        bodies.add(jsonDecode(request.body) as Map<String, dynamic>);
        return _reply();
      }),
    );
    await tester.enterText(find.byType(TextField), '45 minutes, at the shop');
    await tester.tap(find.text('Send answer'));
    await tester.pumpAndSettle();
    expect(bodies.single['text'], '45 minutes, at the shop');
    expect(bodies.single.containsKey('candidate_id'), isFalse);
  });
}

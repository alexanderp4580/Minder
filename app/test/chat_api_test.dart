import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:minder/data/chat_api.dart';
import 'package:minder/domain/entry.dart';
import 'package:minder/domain/failure.dart';

const preview = {
  'id': 'preview',
  'cursor': 42,
  'kind': 'preview',
  'text': 'Review this change.',
  'created_at': '2026-11-12T09:00:00Z',
  'cost_eur': 0.001,
  'change': {
    'id': 'change',
    'action': 'update_item',
    'item_title': 'Call mum',
    'fields': [
      {'name': 'title', 'old': 'Call', 'new': 'Call mum'},
    ],
    'state': 'pending',
    'undo_id': 'undo',
  },
  'reminder': {'id': 'reminder', 'state': 'active'},
};

void main() {
  test(
    'history decodes previews, reminders and costs at the API boundary',
    () async {
      final api = ChatApi(
        'https://test',
        client: MockClient((request) async {
          expect(request.url.path, '/v1/chat');
          expect(request.url.queryParameters['after'], '10');
          return http.Response(jsonEncode([preview]), 200);
        }),
      );
      addTearDown(api.close);
      final entry = (await api.history(10)).single;
      expect(entry.id, 'preview');
      expect(entry.cursor, 42);
      expect(entry.createdAt, DateTime.utc(2026, 11, 12, 9));
      expect(entry.costEur, 0.001);
      expect(entry.change!.state, ChangeState.pending);
      expect(entry.change!.undoId, 'undo');
      expect(entry.change!.fields.single.oldValue, 'Call');
      expect(entry.change!.fields.single.newValue, 'Call mum');
      expect(entry.reminder!.active, true);
    },
  );

  test('conflict exposes the replacement preview for the controller', () async {
    final api = ChatApi(
      'https://test',
      client: MockClient(
        (request) async => http.Response(
          jsonEncode({'message': 'Changed.', 'entry': preview}),
          409,
        ),
      ),
    );
    addTearDown(api.close);
    await expectLater(
      api.confirm('change'),
      throwsA(
        isA<RepositoryFailure>().having(
          (e) => e.replacement?.id,
          'replacement',
          'preview',
        ),
      ),
    );
  });

  test('calendar decodes placement, recurrence and optional fields', () async {
    final api = ChatApi(
      'https://test',
      client: MockClient((request) async {
        expect(request.url.path, '/v1/items');
        expect(request.url.queryParameters['from'], '2026-11-01T00:00:00.000Z');
        return http.Response(
          jsonEncode([
            {
              'id': 'item',
              'title': 'Dentist',
              'status': 'open',
              'start_at': '2026-11-12T09:00:00Z',
              'due_at': '',
              'duration_minutes': 30,
              'priority': 2,
              'location': 'Damrak 1',
              'list': 'Health',
              'recurrence': {
                'every': 'monthly',
                'interval': 2,
                'after_completion': true,
              },
            },
            {'id': 'empty', 'title': 'Unscheduled', 'status': 'done'},
          ]),
          200,
        );
      }),
    );
    addTearDown(api.close);
    final items = await api.itemsBetween(
      DateTime.utc(2026, 11),
      DateTime.utc(2026, 12),
    );
    expect(items.first.placedAt!.toUtc(), DateTime.utc(2026, 11, 12, 9));
    expect(items.first.durationMinutes, 30);
    expect(items.first.priority, 2);
    expect(items.first.location, 'Damrak 1');
    expect(items.first.list, 'Health');
    expect(items.first.repeat, 'every 2 months after completion');
    expect(items.last.placedAt, isNull);
    expect(items.last.done, true);
    expect(items.last.repeat, '');
  });

  test('reminder choices travel to the server and back', () async {
    final api = ChatApi(
      'https://example.test',
      client: MockClient((request) async {
        expect(request.url.path, '/v1/changes/change/reminders');
        expect(jsonDecode(request.body), {
          'offsets': [1440, 60],
        });
        return http.Response(
          jsonEncode({
            ...preview,
            'change': {
              ...(preview['change']! as Map<String, dynamic>),
              'reminder_offsets_minutes': [1440, 60],
              'reminder_choices_minutes': [10, 60, 1440],
            },
          }),
          200,
        );
      }),
    );
    final entry = await api.setChangeReminders('change', [1440, 60]);
    expect(entry.change!.reminderOffsets, [1440, 60]);
    expect(entry.change!.reminderChoices, [10, 60, 1440]);
    api.close();
  });
}

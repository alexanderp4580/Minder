import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:minder/data/chat_api.dart';
import 'package:minder/data/wire.dart';
import 'package:minder/domain/failure.dart';

Map<String, dynamic> task(String id, String status, {int version = 1}) => {
  'id': id,
  'title': 'Task $id',
  'status': status,
  'version': version,
  'priority': 2,
  'list': 'House',
  'color': 'teal',
  'notes': 'Ask for **two** quotes',
  'remind_at': '2026-11-20T08:00:00Z',
  'created_at': '2026-11-10T10:00:00Z',
};

void main() {
  test('boardFromJson maps the three columns and tolerates missing ones', () {
    final board = boardFromJson({
      'open': [task('a', 'open')],
      'in_progress': [task('b', 'in_progress')],
    });
    expect(board.open.single.id, 'a');
    expect(board.open.single.priority, 2);
    expect(board.open.single.list, 'House');
    expect(board.open.single.remindAt, DateTime.utc(2026, 11, 20, 8).toLocal());
    expect(board.inProgress.single.id, 'b');
    expect(board.done, isEmpty);
    expect(board.isEmpty, isFalse);
    expect(boardFromJson({}).isEmpty, isTrue);
  });

  test('the board loads from GET /v1/board', () async {
    final api = ChatApi(
      'https://test',
      client: MockClient((request) async {
        expect(request.method, 'GET');
        expect(request.url.path, '/v1/board');
        return http.Response(
          jsonEncode({
            'open': [task('a', 'open')],
            'in_progress': [],
            'done': [task('c', 'done')],
          }),
          200,
        );
      }),
    );
    addTearDown(api.close);
    final board = await api.board();
    expect(board.open.single.id, 'a');
    expect(board.done.single.id, 'c');
  });

  test('addTask posts the title and returns the item', () async {
    final api = ChatApi(
      'https://test',
      client: MockClient((request) async {
        expect(request.method, 'POST');
        expect(request.url.path, '/v1/items');
        expect(jsonDecode(request.body), {'title': 'Call plumber'});
        return http.Response(jsonEncode(task('new', 'open')), 201);
      }),
    );
    addTearDown(api.close);
    final item = await api.addTask('Call plumber');
    expect(item.id, 'new');
    expect(item.status, 'open');
  });

  test('moveTask posts the version and new status', () async {
    final api = ChatApi(
      'https://test',
      client: MockClient((request) async {
        expect(request.url.path, '/v1/items/a%20b/status');
        expect(jsonDecode(request.body), {
          'expected_version': 3,
          'status': 'in_progress',
        });
        return http.Response(
          jsonEncode(task('a b', 'in_progress', version: 4)),
          200,
        );
      }),
    );
    addTearDown(api.close);
    final moved = await api.moveTask(
      itemFromJson(task('a b', 'open', version: 3)),
      'in_progress',
    );
    expect(moved.status, 'in_progress');
    expect(moved.version, 4);
  });

  test(
    'a conflict and a calendar refusal reach the caller as messages',
    () async {
      for (final reply in [
        (409, 'The item changed. Request a new preview.'),
        (400, 'This task is on the calendar.'),
      ]) {
        final api = ChatApi(
          'https://test',
          client: MockClient(
            (request) async => http.Response(
              jsonEncode({'code': 'x', 'message': reply.$2}),
              reply.$1,
            ),
          ),
        );
        addTearDown(api.close);
        await expectLater(
          api.moveTask(itemFromJson(task('a', 'open')), 'done'),
          throwsA(
            isA<RepositoryFailure>()
                .having((e) => e.message, 'message', reply.$2)
                .having((e) => e.conflict, 'conflict', reply.$1 == 409),
          ),
        );
      }
    },
  );
}

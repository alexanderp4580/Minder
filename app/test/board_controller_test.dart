import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:minder/application/board_controller.dart';
import 'package:minder/domain/failure.dart';

import 'board_fakes.dart';

void main() {
  late FakeBoard repo;
  late BoardController c;
  setUp(() {
    repo = FakeBoard()
      ..items.addAll([
        task('a', 'open'),
        task('b', 'in_progress'),
        task('c', 'done'),
      ]);
    c = BoardController(repo);
    addTearDown(c.dispose);
  });

  test('load fills the columns and clears loading', () async {
    final f = c.load();
    expect(c.loading, isTrue);
    await f;
    expect(c.loading, isFalse);
    expect(c.board.open.single.id, 'a');
    expect(c.board.inProgress.single.id, 'b');
    expect(c.board.done.single.id, 'c');
    expect(c.error, isNull);
  });

  test('a failed load keeps the old board and shows the message', () async {
    await c.load();
    repo.failNext = const RepositoryFailure('Check your connection.');
    await c.load();
    expect(c.error, 'Check your connection.');
    expect(c.board.open, hasLength(1));
    repo.failNext = Exception('boom');
    await c.load();
    expect(c.error, 'Check your connection. Then try again.');
    await c.load();
    expect(c.error, isNull);
  });

  test('a move reloads the board and remembers it for Undo', () async {
    await c.load();
    await c.move(c.board.open.single, 'in_progress');
    expect(c.board.open, isEmpty);
    expect(c.board.inProgress.map((i) => i.id), containsAll(['a', 'b']));
    expect(c.lastMove, isNotNull);
    expect(c.lastMove!.to, 'in_progress');
    expect(c.lastMove!.from, 'open');
    expect(c.busy, isEmpty);
  });

  test('Undo moves the task back using the new version', () async {
    await c.load();
    await c.move(c.board.open.single, 'done');
    await c.undoMove();
    expect(repo.items.firstWhere((i) => i.id == 'a').status, 'open');
    expect(repo.calls, contains('move a open v2'));
    expect(c.lastMove, isNull);
  });

  test(
    'a task is busy while its move runs and a second move is ignored',
    () async {
      await c.load();
      repo.gate = Completer<void>();
      final item = c.board.open.single;
      final first = c.move(item, 'in_progress');
      expect(c.busy, {'a'});
      await c.move(item, 'done');
      repo.gate!.complete();
      await first;
      expect(repo.calls.where((x) => x.startsWith('move')).length, 1);
      expect(c.busy, isEmpty);
    },
  );

  test('a conflict reloads the board and explains it', () async {
    await c.load();
    final stale = c.board.open.single;
    repo.items[0] = task('a', 'in_progress', version: 2);
    await c.move(stale, 'done');
    expect(c.error, 'This task changed. The board is updated.');
    expect(c.board.inProgress.map((i) => i.id), contains('a'));
    expect(c.lastMove, isNull);
  });

  test('a slow earlier load cannot overwrite a newer board', () async {
    await c.load();
    final gate = Completer<void>();
    repo.boardGate = gate;
    final slow = c.load();
    await Future<void>.delayed(Duration.zero);
    await c.move(c.board.open.single, 'in_progress');
    gate.complete();
    await slow;
    expect(c.board.open, isEmpty);
    expect(c.board.inProgress.map((i) => i.id), contains('a'));
  });

  test(
    'a server refusal reloads the board; a dropped connection does not',
    () async {
      await c.load();
      var before = repo.loads;
      repo.failNext = const RepositoryFailure('This task is on the calendar.');
      await c.move(c.board.open.single, 'done');
      expect(repo.loads, before + 1);
      before = repo.loads;
      repo.failNext = const RepositoryFailure(
        'Check your connection. Then try again.',
        transport: true,
      );
      await c.move(c.board.open.single, 'done');
      expect(repo.loads, before);
    },
  );

  test('another failure keeps the board and shows its message', () async {
    await c.load();
    repo.failNext = const RepositoryFailure('This task is on the calendar.');
    await c.move(c.board.open.single, 'done');
    expect(c.error, 'This task is on the calendar.');
    expect(c.board.open, hasLength(1));
  });

  test('add trims the title, reloads and ignores blank input', () async {
    await c.load();
    expect(await c.add('   '), isFalse);
    expect(repo.calls.where((x) => x.startsWith('add')), isEmpty);
    expect(await c.add('  Call plumber '), isTrue);
    expect(repo.calls, contains('add Call plumber'));
    expect(c.board.open.map((i) => i.title), contains('Call plumber'));
    expect(c.adding, isFalse);
  });

  test('a failed add returns false and keeps the message', () async {
    await c.load();
    repo.failNext = const RepositoryFailure(
      'Use a title with 1 to 200 characters.',
    );
    expect(await c.add('x'), isFalse);
    expect(c.error, 'Use a title with 1 to 200 characters.');
  });

  test('delete confirms through the repository and reloads', () async {
    await c.load();
    final before = repo.loads;
    await c.confirmDelete('change');
    expect(repo.calls, contains('confirm change'));
    expect(repo.loads, before + 1);
    await c.discardDelete('change');
    expect(repo.calls, contains('discard change'));
  });

  test('a disposed controller ignores late results', () async {
    final late = BoardController(repo);
    repo.gate = Completer<void>();
    await late.load();
    final f = late.move(late.board.open.single, 'done');
    late.dispose();
    repo.gate!.complete();
    await f;
  });
}

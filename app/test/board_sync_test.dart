import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:minder/application/board_chat_sync.dart';
import 'package:minder/application/board_controller.dart';
import 'package:minder/domain/failure.dart';
import 'package:minder/ui/format.dart';

import 'board_fakes.dart';

class Cursor extends ChangeNotifier {
  int value = 5;
  void changed() => notifyListeners();
}

void main() {
  late FakeBoard repo;
  late BoardController c;
  setUp(() {
    repo = FakeBoard()..items.add(task('a', 'open'));
    c = BoardController(repo);
    addTearDown(c.dispose);
  });

  test('refresh does nothing until the board has loaded once', () async {
    await c.refresh();
    expect(repo.loads, 0);
    await c.load();
    await c.refresh();
    expect(repo.loads, 2);
  });

  test(
    'refresh is silent: no loading flag and no error on a failure',
    () async {
      await c.load();
      final seenLoading = <bool>[];
      c.addListener(() => seenLoading.add(c.loading));
      repo.items.add(task('b', 'open'));
      await c.refresh();
      expect(c.board.open.map((i) => i.id), ['a', 'b']);
      expect(seenLoading.every((l) => !l), isTrue);
      repo.failNext = const RepositoryFailure(
        'Check your connection.',
        transport: true,
      );
      await c.refresh();
      expect(c.error, isNull);
      expect(c.board.open, hasLength(2));
    },
  );

  test(
    'a new chat entry refreshes the board; a repeated cursor does not',
    () async {
      await c.load();
      final chat = Cursor();
      final sync = BoardChatSync(
        source: chat,
        cursor: () => chat.value,
        board: c,
      );
      addTearDown(sync.dispose);
      final before = repo.loads;
      chat.changed();
      await Future<void>.delayed(Duration.zero);
      expect(repo.loads, before);
      repo.items.add(task('b', 'open'));
      chat.value = 6;
      chat.changed();
      await Future<void>.delayed(Duration.zero);
      expect(repo.loads, before + 1);
      expect(c.board.open.map((i) => i.id), ['a', 'b']);
      sync.dispose();
      chat.value = 7;
      chat.changed();
      await Future<void>.delayed(Duration.zero);
      expect(repo.loads, before + 1, reason: 'a disposed sync stops listening');
    },
  );

  test('the preview names the shown_on field', () {
    expect(fieldLabel('shown_on'), 'Shown on');
  });
}

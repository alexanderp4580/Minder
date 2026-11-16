import 'dart:async';

import 'package:minder/data/wire.dart';
import 'package:minder/domain/board.dart';
import 'package:minder/domain/entry.dart';
import 'package:minder/domain/failure.dart';
import 'package:minder/domain/item.dart';

Item task(
  String id,
  String status, {
  int version = 1,
  String title = '',
  int priority = 0,
  String list = '',
  String notes = '',
}) => itemFromJson({
  'id': id,
  'title': title.isEmpty ? 'Task $id' : title,
  'status': status,
  'version': version,
  'priority': priority,
  'list': list,
  'notes': notes,
});

class FakeBoard implements BoardRepository {
  final items = <Item>[];
  final calls = <String>[];
  Object? failNext;
  Completer<void>? gate;

  /// Holds the next board() call until completed. The reply is the board as it was when called.
  Completer<void>? boardGate;
  int loads = 0;

  Board get snapshot => Board(
    open: items.where((i) => i.status == 'open').toList(),
    inProgress: items.where((i) => i.status == 'in_progress').toList(),
    done: items.where((i) => i.status == 'done').toList(),
  );

  void _maybeFail() {
    final e = failNext;
    if (e != null) {
      failNext = null;
      throw e;
    }
  }

  @override
  Future<Board> board() async {
    loads++;
    calls.add('board');
    _maybeFail();
    final asOfNow = snapshot;
    final held = boardGate;
    boardGate = null;
    await held?.future;
    return asOfNow;
  }

  @override
  Future<Item> addTask(String title) async {
    calls.add('add $title');
    _maybeFail();
    final item = task('n${items.length}', 'open', title: title);
    items.add(item);
    return item;
  }

  @override
  Future<Item> moveTask(Item item, String status) async {
    calls.add('move ${item.id} $status v${item.version}');
    await gate?.future;
    _maybeFail();
    final i = items.indexWhere((x) => x.id == item.id);
    if (items[i].version != item.version) {
      throw const RepositoryFailure(
        'The item changed. Request a new preview.',
        conflict: true,
      );
    }
    items[i] = task(item.id, status, version: item.version + 1);
    return items[i];
  }

  @override
  Future<Entry> previewDelete(Item item) async {
    calls.add('previewDelete ${item.id}');
    return entryFromJson({
      'id': 'e-${item.id}',
      'cursor': 1,
      'kind': 'preview',
      'text': 'Delete this item and its data.',
      'created_at': '2026-11-12T10:00:00Z',
      'change': {
        'id': 'c-${item.id}',
        'action': 'delete_item',
        'item_title': item.title,
        'state': 'pending',
        'fields': [
          {'name': 'item', 'old': item.title, 'new': 'Deleted'},
        ],
      },
    });
  }

  @override
  Future<Entry> confirm(String changeId) async {
    calls.add('confirm $changeId');
    return Entry(
      id: 'e',
      cursor: 1,
      kind: 'result',
      text: 'Deleted.',
      createdAt: DateTime.utc(2026),
    );
  }

  @override
  Future<Entry> discard(String changeId) async {
    calls.add('discard $changeId');
    return confirm(changeId);
  }
}

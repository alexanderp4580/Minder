import 'item.dart';
import 'item_repository.dart';

/// The Kanban board: tasks without a date, grouped by status.
class Board {
  const Board({
    this.open = const [],
    this.inProgress = const [],
    this.done = const [],
  });
  final List<Item> open, inProgress, done;
  bool get isEmpty => open.isEmpty && inProgress.isEmpty && done.isEmpty;
}

/// The statuses a board task can take, and the column names the owner reads.
abstract final class BoardStatus {
  static const open = 'open', inProgress = 'in_progress', done = 'done';
  static const all = [open, inProgress, done];
  static String label(String status) => switch (status) {
    open => 'To pick up',
    inProgress => 'In progress',
    done => 'Done',
    _ => status,
  };
}

abstract interface class BoardRepository implements ItemDeletion {
  Future<Board> board();

  /// Adds an open task with no date.
  Future<Item> addTask(String title);

  /// Moves [item] to [status]. The server checks the item's version.
  Future<Item> moveTask(Item item, String status);
}

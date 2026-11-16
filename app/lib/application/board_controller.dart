import 'package:flutter/foundation.dart';

import '../domain/board.dart';
import '../domain/entry.dart';
import '../domain/failure.dart';
import '../domain/item.dart';
import 'item_deleter.dart';

/// A finished move, kept so the owner can undo it.
class BoardMove {
  const BoardMove({required this.item, required this.from, required this.to});

  /// The task as the server returned it after the move.
  final Item item;
  final String from, to;
}

class BoardController extends ChangeNotifier implements ItemDeleter {
  BoardController(this._repository);
  final BoardRepository _repository;
  Board board = const Board();
  bool loading = false, adding = false;
  String? error;
  BoardMove? lastMove;
  final _busy = <String>{};
  bool _disposed = false;
  bool _loaded = false;
  int _request = 0;

  /// Tasks whose move is running.
  Set<String> get busy => Set.unmodifiable(_busy);

  String _message(Object e) => e is RepositoryFailure
      ? e.message
      : 'Check your connection. Then try again.';

  void _notify() {
    if (!_disposed) notifyListeners();
  }

  Future<void> load() async {
    if (_disposed) return;
    final request = ++_request;
    loading = true;
    error = null;
    _notify();
    try {
      final loaded = await _repository.board();
      if (_disposed || request != _request) return;
      board = loaded;
      _loaded = true;
      error = null;
    } catch (e) {
      if (_disposed || request != _request) return;
      error = _message(e);
    }
    loading = false;
    _notify();
  }

  Future<void> move(Item item, String status) =>
      _move(item, status, remember: true);

  Future<void> _move(Item item, String status, {required bool remember}) async {
    if (_busy.contains(item.id)) return;
    _busy.add(item.id);
    error = null;
    _notify();
    try {
      final moved = await _repository.moveTask(item, status);
      if (remember) {
        lastMove = BoardMove(item: moved, from: item.status, to: status);
      }
      await _reload();
    } on RepositoryFailure catch (e) {
      lastMove = null;
      error = e.conflict
          ? 'This task changed. The board is updated.'
          : e.message;
      // The server answered, so the card may be stale. A dropped connection tells us nothing.
      if (!e.transport) await _reload();
    } catch (e) {
      lastMove = null;
      error = _message(e);
    }
    _busy.remove(item.id);
    _notify();
  }

  /// Moves the last moved task back to where it was.
  Future<void> undoMove() async {
    final move = lastMove;
    if (move == null) return;
    lastMove = null;
    await _move(move.item, move.from, remember: false);
  }

  Future<bool> add(String title) async {
    final text = title.trim();
    if (text.isEmpty || adding) return false;
    adding = true;
    error = null;
    _notify();
    var ok = false;
    try {
      await _repository.addTask(text);
      ok = true;
    } catch (e) {
      error = _message(e);
    }
    adding = false;
    if (ok) await _reload();
    _notify();
    return ok;
  }

  /// Reloads without the loading bar or an error message, for a change made elsewhere such as
  /// chat. It does nothing before the first load.
  Future<void> refresh() async {
    if (!_loaded) return;
    await _reload(quiet: true);
    _notify();
  }

  /// Reloads after an action. It supersedes any older load still in flight.
  Future<void> _reload({bool quiet = false}) async {
    final request = ++_request;
    try {
      final loaded = await _repository.board();
      if (_disposed || request != _request) return;
      board = loaded;
    } catch (e) {
      if (!quiet && !_disposed && request == _request && error == null) {
        error = _message(e);
      }
    }
  }

  @override
  Future<Entry> previewDelete(Item item) => _repository.previewDelete(item);

  @override
  Future<void> confirmDelete(String changeId) async {
    await _repository.confirm(changeId);
    await load();
  }

  @override
  Future<void> discardDelete(String changeId) async {
    await _repository.discard(changeId);
  }

  @override
  void dispose() {
    _disposed = true;
    super.dispose();
  }
}

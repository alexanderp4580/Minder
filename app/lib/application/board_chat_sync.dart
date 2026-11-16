import 'dart:async';

import 'package:flutter/foundation.dart';

import 'board_controller.dart';

/// Refreshes the board when chat gets a new entry, so a task the agent created or moved shows
/// up without reopening the tab. [cursor] reads the latest chat entry cursor.
class BoardChatSync {
  BoardChatSync({
    required this.source,
    required this.cursor,
    required this.board,
  }) : _last = cursor() {
    source.addListener(_changed);
  }
  final Listenable source;
  final int Function() cursor;
  final BoardController board;
  int _last;

  void _changed() {
    final now = cursor();
    if (now == _last) return;
    _last = now;
    unawaited(board.refresh());
  }

  void dispose() => source.removeListener(_changed);
}

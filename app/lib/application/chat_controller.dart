import 'dart:async';

import 'package:flutter/foundation.dart';

import '../domain/chat_repository.dart';
import '../domain/entry.dart';
import 'use_cases.dart';

class ChatController extends ChangeNotifier {
  ChatController(
    this.repository, {
    this.pollInterval = const Duration(seconds: 5),
  }) : _send = SendMessage(repository),
       _confirm = ConfirmChange(repository),
       _discard = DiscardChange(repository),
       _undo = UndoChange(repository),
       _remind = ActOnReminder(repository);
  final ChatRepository repository;
  final SendMessage _send;
  final ConfirmChange _confirm;
  final DiscardChange _discard;
  final UndoChange _undo;
  final ActOnReminder _remind;
  final Duration pollInterval;
  final List<Entry> _entries = [];
  List<Entry> get entries => List.unmodifiable(_entries);
  int cursor = 0;
  bool waiting = false,
      loading = true,
      acting = false,
      connected = false,
      needsSignIn = false;
  String? error;
  String _lastText = '';
  Future<Entry> Function()? _retryAction;
  bool _disposed = false, _syncing = false;
  Timer? _poll, _reconnect;
  StreamSubscription<Entry>? _subscription;
  void _notify() {
    if (!_disposed) notifyListeners();
  }

  void _merge(Entry e) {
    if (_disposed) return;
    final index = _entries.indexWhere((v) => v.id == e.id);
    if (index < 0) {
      _entries.add(e);
    } else if (e.cursor >= _entries[index].cursor) {
      _entries[index] = e;
    }
    if (e.cursor > cursor) cursor = e.cursor;
    if (e.kind != EntryKind.user) {
      waiting = false;
    }
    connected = true;
    needsSignIn = false;
  }

  void _failed(Object e) {
    error = e is ApiException
        ? e.message
        : 'Check your connection. Then try again.';
    if (e is ApiException) {
      if (e.replacement != null) _merge(e.replacement!);
      needsSignIn = e.unauthorized;
    }
    connected = false;
    waiting = false;
  }

  Future<void> start() async {
    await sync();
    if (_disposed) return;
    loading = false;
    _watch();
    _poll = Timer.periodic(pollInterval, (_) => sync());
    _notify();
  }

  Future<void> sync() async {
    if (_disposed || _syncing) return;
    _syncing = true;
    try {
      final incoming = await repository.history(cursor);
      for (final e in incoming) {
        _merge(e);
      }
      connected = true;
      needsSignIn = false;
      if (_retryAction == null && _lastText.isEmpty) error = null;
    } catch (e) {
      _failed(e);
    } finally {
      _syncing = false;
      _notify();
    }
  }

  void _watch() {
    if (_disposed) return;
    _subscription = repository
        .watch(cursor)
        .listen(
          (e) {
            _merge(e);
            _notify();
          },
          onError: (Object e) {
            if (_disposed) return;
            _failed(e);
            _scheduleReconnect();
            _notify();
          },
          onDone: _scheduleReconnect,
          cancelOnError: true,
        );
  }

  void _scheduleReconnect() {
    if (_disposed) return;
    _reconnect?.cancel();
    _reconnect = Timer(const Duration(seconds: 2), () async {
      await sync();
      _watch();
    });
  }

  Future<void> send(String text) async {
    if (_disposed || waiting || text.trim().isEmpty) return;
    _lastText = text.trim();
    _retryAction = null;
    error = null;
    waiting = true;
    _notify();
    try {
      final e = await _send(_lastText);
      _merge(e);
    } catch (e) {
      _failed(e);
    }
    _notify();
  }

  Future<void> confirm(String changeId) => _act(() => _confirm(changeId));
  Future<void> discard(String changeId) => _act(() => _discard(changeId));
  Future<void> undo(String undoId) => _act(() => _undo(undoId));
  Future<void> remind(String reminderId, ReminderAction action) =>
      _act(() => _remind(reminderId, action));

  Future<void> _act(Future<Entry> Function() action) async {
    if (_disposed || acting) return;
    acting = true;
    _retryAction = action;
    error = null;
    _notify();
    try {
      _merge(await action());
      _retryAction = null;
      await sync();
    } catch (e) {
      _failed(e);
    } finally {
      acting = false;
      _notify();
    }
  }

  Future<void> retry({String? text}) async {
    if (text != null) {
      await send(text);
      return;
    }
    if (_retryAction != null) {
      await _act(_retryAction!);
    } else if (_lastText.isNotEmpty) {
      waiting = false;
      await send(_lastText);
    } else {
      await sync();
    }
  }

  String retryText(Entry entry) {
    String text = '';
    for (final e in _entries) {
      if (e.id == entry.id) break;
      if (e.kind == EntryKind.user) text = e.text;
    }
    return text;
  }

  Future<void> signedIn() async {
    needsSignIn = false;
    error = null;
    await sync();
    await _subscription?.cancel();
    _watch();
  }

  @override
  void dispose() {
    _disposed = true;
    _poll?.cancel();
    _reconnect?.cancel();
    _subscription?.cancel();
    repository.close();
    super.dispose();
  }
}

import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';

import '../domain/chat_repository.dart';
import '../domain/capture.dart';
import '../domain/capture_repository.dart';
import '../domain/entry.dart';
import '../domain/failure.dart';

class ChatController extends ChangeNotifier {
  ChatController(
    this.repository, {
    this.pollInterval = const Duration(seconds: 5),
    this.offlineGrace = const Duration(seconds: 2),
  });
  final ChatRepository repository;
  final Duration pollInterval;

  /// How long the connection must stay down before the header shows it.
  final Duration offlineGrace;
  final List<Entry> _entries = [];
  final Set<String> _answeredQuestions = {};
  ({Capture capture, String? candidateId})? _pendingReview;
  List<Entry> get entries => List.unmodifiable(_entries);
  int cursor = 0;
  // Only history and the ordered event stream establish which events we have read.
  // A direct action response may arrive ahead of earlier events in its transaction.
  int _syncedCursor = 0;

  /// True while the agent works: a message waits for its reply, the owner spoke last (a quick
  /// reply, or a reply still queued after a reconnect), or an email card is still being read.
  bool get thinking =>
      waiting ||
      (_entries.isNotEmpty && _entries.last.kind == EntryKind.user) ||
      _entries.any((e) => e.kind == 'capture' && e.capture?.processing == true);
  bool waiting = false,
      loading = true,
      acting = false,
      connected = false,
      needsSignIn = false;
  String? error;
  String? _unsent;
  bool _backgroundError = false;
  bool _disposed = false, _syncing = false;
  Timer? _poll, _reconnect, _offlineTimer;

  /// True once the connection has been down for [offlineGrace].
  bool showOffline = false;
  StreamSubscription<Entry>? _subscription;
  void _notify() {
    if (!_disposed) notifyListeners();
  }

  void _merge(Entry e, {bool synced = false}) {
    if (_disposed) return;
    final index = _entries.indexWhere((v) => v.id == e.id);
    if (index < 0) {
      _entries.add(e);
    } else if (e.cursor >= _entries[index].cursor) {
      _entries[index] = e;
    }
    final removed = e.change?.removedEntryIds ?? const [];
    if (removed.isNotEmpty) _entries.removeWhere((v) => removed.contains(v.id));
    if (e.cursor > cursor) cursor = e.cursor;
    if (synced && e.cursor > _syncedCursor) _syncedCursor = e.cursor;
    final pending = _pendingReview;
    if ((e.kind != EntryKind.user && e.captureId.isEmpty) ||
        (pending != null &&
            e.capture?.id == pending.capture.id &&
            e.capture!.revision > pending.capture.revision)) {
      waiting = false;
      _pendingReview = null;
    }
    connected = true;
    needsSignIn = false;
    _connectionOk();
  }

  /// The text of a send that never completed, once. The screen puts it back in the composer.
  String? takeUnsentText() {
    final text = _unsent;
    _unsent = null;
    return text;
  }

  bool get hasUnsentText => _unsent != null;

  /// True when nothing was added to the chat after [entry].
  bool isLatest(Entry entry) =>
      _entries.isNotEmpty && _entries.last.id == entry.id;

  void _connectionOk() {
    _offlineTimer?.cancel();
    _offlineTimer = null;
    showOffline = false;
    if (_backgroundError) {
      error = null;
      _backgroundError = false;
    }
  }

  void _connectionLost() {
    connected = false;
    _offlineTimer ??= Timer(offlineGrace, () {
      showOffline = true;
      _notify();
    });
  }

  /// A background failure. A dropped connection only feeds the header icon, and so does
  /// any error the data layer did not turn into a [RepositoryFailure], such as a reset
  /// socket on the event stream. Other messages clear when the connection works again.
  void _backgroundFailed(Object e) {
    if (e is! RepositoryFailure || e.transport) {
      _connectionLost();
    } else {
      _failed(e);
      _backgroundError = true;
    }
  }

  void _failed(Object e) {
    // A dropped connection shows only in the header icon, never as a message in the chat.
    final transport = e is! RepositoryFailure || e.transport;
    if (transport) _connectionLost();
    error = transport ? null : e.message;
    _backgroundError = false;
    if (e is RepositoryFailure) {
      if (e.replacement != null) _merge(e.replacement!);
      needsSignIn = e.unauthorized;
    }
    connected = false;
    waiting = false;
    _pendingReview = null;
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
      final incoming = await repository.history(_syncedCursor);
      for (final e in incoming) {
        _merge(e, synced: true);
      }
      connected = true;
      if (needsSignIn) error = null;
      needsSignIn = false;
      _connectionOk();
    } catch (e) {
      _backgroundFailed(e);
    } finally {
      _syncing = false;
      _notify();
    }
  }

  void _watch() {
    if (_disposed) return;
    _subscription = repository
        .watch(_syncedCursor)
        .listen(
          (e) {
            _merge(e, synced: true);
            _notify();
          },
          onError: (Object e) {
            if (_disposed) return;
            _backgroundFailed(e);
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

  String _questionKey(Capture c, String? candidateId) =>
      '${c.id}:${c.revision}:${candidateId ?? '*'}';

  Capture _currentCapture(Capture original) {
    var current = original;
    var cursor = -1;
    for (final entry in _entries) {
      final capture = entry.capture;
      if (capture != null &&
          capture.id == original.id &&
          (capture.revision > current.revision ||
              (capture.revision == current.revision &&
                  entry.cursor > cursor))) {
        current = capture;
        cursor = entry.cursor;
      }
    }
    return current;
  }

  /// Suppress the answered suggestions in both copies of a question. The owner
  /// entry also preserves this state when history reloads before the agent replies.
  bool questionAnswered(Capture capture, {String? candidateId}) {
    final current = _currentCapture(capture);
    if (current.revision > capture.revision ||
        ['failed', 'reviewed', 'dismissed'].contains(current.status)) {
      return true;
    }
    if (_answeredQuestions.contains(_questionKey(capture, null)) ||
        _answeredQuestions.contains(_questionKey(capture, candidateId))) {
      return true;
    }
    if (candidateId != null &&
        current.candidates.any(
          (c) =>
              c.id == candidateId &&
              (c.question.isEmpty ||
                  !['needs-answer', 'pending'].contains(c.state)),
        )) {
      return true;
    }
    final open = capture.candidates.where(
      (c) =>
          c.question.isNotEmpty &&
          (c.state == 'needs-answer' || c.state == 'pending'),
    );
    if (open.length + capture.questions.length != 1) return false;
    for (final question in _entries) {
      if (question.kind == EntryKind.question &&
          question.capture?.id == capture.id &&
          question.capture?.revision == capture.revision &&
          _entries.any(
            (entry) =>
                entry.kind == EntryKind.user && entry.cursor > question.cursor,
          )) {
        return true;
      }
    }
    return false;
  }

  ({Capture capture, String? candidateId})? get _openQuestion {
    final questions =
        _entries
            .where((e) => e.kind == EntryKind.question && e.capture != null)
            .toList()
          ..sort((a, b) => b.cursor.compareTo(a.cursor));
    for (final entry in questions) {
      final capture = _currentCapture(entry.capture!);
      if (['failed', 'reviewed', 'dismissed'].contains(capture.status)) {
        continue;
      }
      final open = capture.candidates
          .where(
            (c) =>
                c.question.isNotEmpty &&
                (c.state == 'needs-answer' || c.state == 'pending'),
          )
          .toList();
      if (open.isNotEmpty || capture.questions.isNotEmpty) {
        return (
          capture: capture,
          candidateId: open.length == 1 && capture.questions.isEmpty
              ? open.single.id
              : null,
        );
      }
    }
    return null;
  }

  Future<void> send(String text) => _send(text, restoreOnFailure: true);

  Future<void> answerReview(
    Capture capture,
    String text, {
    String? candidateId,
  }) => _send(text, review: (capture: capture, candidateId: candidateId));

  Future<void> _send(
    String text, {
    ({Capture capture, String? candidateId})? review,
    bool restoreOnFailure = false,
  }) async {
    final body = text.trim();
    if (_disposed || waiting || body.isEmpty) return;
    if (utf8.encode(body).length > maxMessageBytes) {
      _unsent = body; // nothing is cut; the text goes back to the composer
      error = 'That message is too long. Keep it under 400,000 bytes.';
      _notify();
      return;
    }
    error = null;
    final target = repository is CaptureRepository
        ? review ?? _openQuestion
        : null;
    final questionKey = target == null
        ? null
        : _questionKey(target.capture, target.candidateId);
    if (questionKey != null) _answeredQuestions.add(questionKey);
    _pendingReview = target;
    waiting = true;
    _notify();
    try {
      final e = target == null
          ? await repository.send(body)
          : await (repository as CaptureRepository).answerCapture(
              target.capture.id,
              target.capture.revision,
              body,
              candidateId: target.candidateId,
            );
      _merge(e);
    } catch (e) {
      if (questionKey != null) _answeredQuestions.remove(questionKey);
      if (restoreOnFailure) _unsent = body;
      _failed(e);
    }
    _notify();
  }

  void receiveCaptureEntry(Entry entry) {
    _merge(entry);
    _notify();
  }

  Future<void> confirm(String changeId) =>
      _act(() => repository.confirm(changeId));
  Future<void> discard(String changeId) =>
      _act(() => repository.discard(changeId));
  Future<void> setReminders(String changeId, List<int> offsets) async {
    await _act(() => repository.setChangeReminders(changeId, offsets));
    if (error != null) await sync();
  }

  Future<void> setColor(String changeId, String color) =>
      _act(() => repository.setChangeColor(changeId, color));
  Future<void> undo(String undoId) => _act(() => repository.undo(undoId));
  Future<void> remind(String reminderId, ReminderAction action) =>
      _act(() => repository.actOnReminder(reminderId, action));

  Future<void> _act(Future<Entry> Function() action) async {
    if (_disposed || acting) return;
    acting = true;
    error = null;
    _notify();
    try {
      _merge(await action());
      await sync();
    } catch (e) {
      _failed(e);
    } finally {
      acting = false;
      _notify();
    }
  }

  Future<void> retry(String text) => send(text);

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
    _offlineTimer?.cancel();
    _subscription?.cancel();
    super.dispose();
  }
}

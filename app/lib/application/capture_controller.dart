import 'package:flutter/foundation.dart';

import '../domain/capture.dart';
import '../domain/capture_repository.dart';
import '../domain/entry.dart';
import '../domain/failure.dart';

class CaptureController extends ChangeNotifier {
  CaptureController(this.repository, this.onEntry);
  final CaptureRepository repository;
  final void Function(Entry) onEntry;
  bool busy = false, _disposed = false;
  String? error;
  Future<void> _act(Future<void> Function() action) async {
    if (busy || _disposed) return;
    busy = true;
    error = null;
    notifyListeners();
    try {
      await action();
    } catch (e) {
      if (!_disposed) {
        error = e is RepositoryFailure
            ? e.message
            : 'The email action failed. Try again.';
      }
    } finally {
      busy = false;
      if (!_disposed) notifyListeners();
    }
  }

  Future<void> answer(
    Capture c,
    String text, {
    bool acknowledgeCoverage = false,
    String? candidateId,
  }) => _act(() async {
    final entry = await repository.answerCapture(
      c.id,
      c.revision,
      text,
      acknowledgeCoverage: acknowledgeCoverage,
      candidateId: candidateId,
    );
    if (!_disposed) onEntry(entry);
  });
  Future<void> dismiss(Capture c) => _act(() async {
    final entry = await repository.dismissCapture(c.id, c.revision);
    if (!_disposed) onEntry(entry);
  });
  Future<void> retry(Capture c) => _act(() async {
    final entry = await repository.retryCapture(c.id);
    if (!_disposed) onEntry(entry);
  });
  Future<void> delete(Capture c) => _act(() async {
    final entry = await repository.deleteCapture(c.id, c.revision);
    if (!_disposed) onEntry(entry);
  });
  Future<void> download(Capture c, CaptureAttachment attachment) =>
      _act(() async {
        final file = await repository.downloadAttachment(c.id, attachment);
        if (!_disposed) await repository.saveAttachment(file);
      });
  @override
  void dispose() {
    _disposed = true;
    super.dispose();
  }
}

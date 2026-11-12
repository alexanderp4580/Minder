import 'entry.dart';

class ApiException implements Exception {
  const ApiException(
    this.message, {
    this.unauthorized = false,
    this.replacement,
  });
  final String message;
  final bool unauthorized;
  final Entry? replacement;
  @override
  String toString() => message;
}

enum ReminderAction { done, acknowledge, snooze15, snooze60, tonight, tomorrow }

abstract interface class ChatRepository {
  Future<List<Entry>> history(int after);
  Stream<Entry> watch(int after);
  Future<Entry> send(String text);
  Future<Entry> confirm(String changeId);
  Future<Entry> discard(String changeId);
  Future<Entry> undo(String undoId);
  Future<Entry> actOnReminder(String reminderId, ReminderAction action);
  void close();
}

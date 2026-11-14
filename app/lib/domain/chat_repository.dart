import 'entry.dart';

enum ReminderAction { done, acknowledge, snooze15, snooze60, tonight, tomorrow }

/// The longest message the server accepts, in UTF-8 bytes. It covers a pasted email.
const maxMessageBytes = 400000;

abstract interface class ChatRepository {
  Future<List<Entry>> history(int after);
  Stream<Entry> watch(int after);
  Future<Entry> send(String text);
  Future<Entry> confirm(String changeId);
  Future<Entry> discard(String changeId);
  Future<Entry> setChangeColor(String changeId, String color);
  Future<Entry> setChangeReminders(String changeId, List<int> offsets);
  Future<Entry> undo(String undoId);
  Future<Entry> actOnReminder(String reminderId, ReminderAction action);
}

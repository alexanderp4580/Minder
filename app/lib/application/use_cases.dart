import '../domain/chat_repository.dart';
import '../domain/entry.dart';

class SendMessage {
  const SendMessage(this._repository);
  final ChatRepository _repository;
  Future<Entry> call(String text) => _repository.send(text);
}

class ConfirmChange {
  const ConfirmChange(this._repository);
  final ChatRepository _repository;
  Future<Entry> call(String changeId) => _repository.confirm(changeId);
}

class DiscardChange {
  const DiscardChange(this._repository);
  final ChatRepository _repository;
  Future<Entry> call(String changeId) => _repository.discard(changeId);
}

class UndoChange {
  const UndoChange(this._repository);
  final ChatRepository _repository;
  Future<Entry> call(String undoId) => _repository.undo(undoId);
}

class ActOnReminder {
  const ActOnReminder(this._repository);
  final ChatRepository _repository;
  Future<Entry> call(String reminderId, ReminderAction action) =>
      _repository.actOnReminder(reminderId, action);
}

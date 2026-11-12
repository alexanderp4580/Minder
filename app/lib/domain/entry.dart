/// The kinds of chat entry the server sends.
abstract final class EntryKind {
  static const user = 'user',
      minder = 'minder',
      preview = 'preview',
      result = 'result',
      error = 'error',
      reminder = 'reminder';
}

enum ChangeState { pending, confirmed, discarded, failed }

/// One field of a proposed change. [oldValue] is empty for a new value.
class FieldChange {
  const FieldChange({
    required this.name,
    this.oldValue = '',
    this.newValue = '',
  });

  final String name, oldValue, newValue;
}

/// A change the assistant proposed, or the result of one.
class Change {
  const Change({
    required this.id,
    required this.action,
    required this.itemTitle,
    required this.fields,
    required this.state,
    this.undoId,
  });

  final String id, action, itemTitle;
  final List<FieldChange> fields;
  final ChangeState state;
  final String? undoId;
}

/// One firing of a reminder. [state] is active until you act on it.
class ReminderInfo {
  const ReminderInfo({required this.id, required this.state});

  final String id, state;
  bool get active => state == 'active';
}

class Entry {
  const Entry({
    required this.id,
    required this.cursor,
    required this.kind,
    required this.text,
    required this.createdAt,
    this.change,
    this.reminder,
    this.costEur = 0,
  });

  final String id, kind, text;
  final int cursor;
  final DateTime createdAt;
  final Change? change;
  final ReminderInfo? reminder;
  final double costEur;
}

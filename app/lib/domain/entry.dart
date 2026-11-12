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
  factory FieldChange.fromJson(Map<String, dynamic> j) => FieldChange(
    name: j['name'] as String,
    oldValue: j['old'] as String? ?? '',
    newValue: j['new'] as String? ?? '',
  );
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
  factory Change.fromJson(Map<String, dynamic> j) => Change(
    id: j['id'] as String? ?? '',
    action: j['action'] as String,
    itemTitle: j['item_title'] as String? ?? '',
    fields: [
      for (final f in j['fields'] as List? ?? [])
        FieldChange.fromJson(f as Map<String, dynamic>),
    ],
    state: ChangeState.values.asNameMap()[j['state']] ?? ChangeState.failed,
    undoId: j['undo_id'] as String?,
  );
  final String id, action, itemTitle;
  final List<FieldChange> fields;
  final ChangeState state;
  final String? undoId;
}

/// One firing of a reminder. [state] is active until you act on it.
class ReminderInfo {
  const ReminderInfo({required this.id, required this.state});
  factory ReminderInfo.fromJson(Map<String, dynamic> j) =>
      ReminderInfo(id: j['id'] as String, state: j['state'] as String);
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
  factory Entry.fromJson(Map<String, dynamic> j) => Entry(
    id: j['id'] as String,
    cursor: (j['cursor'] as num?)?.toInt() ?? 0,
    kind: j['kind'] as String,
    text: j['text'] as String? ?? '',
    createdAt: DateTime.parse(j['created_at'] as String),
    change: j['change'] == null
        ? null
        : Change.fromJson(j['change'] as Map<String, dynamic>),
    reminder: j['reminder'] == null
        ? null
        : ReminderInfo.fromJson(j['reminder'] as Map<String, dynamic>),
    costEur: (j['cost_eur'] as num?)?.toDouble() ?? 0,
  );
  final String id, kind, text;
  final int cursor;
  final DateTime createdAt;
  final Change? change;
  final ReminderInfo? reminder;
  final double costEur;
}

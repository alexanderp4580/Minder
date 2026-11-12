import '../domain/entry.dart';
import '../domain/item.dart';
import '../domain/push_repository.dart';

FieldChange fieldChangeFromJson(Map<String, dynamic> j) => FieldChange(
  name: j['name'] as String,
  oldValue: j['old'] as String? ?? '',
  newValue: j['new'] as String? ?? '',
);

Change changeFromJson(Map<String, dynamic> j) => Change(
  id: j['id'] as String? ?? '',
  action: j['action'] as String,
  itemTitle: j['item_title'] as String? ?? '',
  fields: [
    for (final f in j['fields'] as List? ?? [])
      fieldChangeFromJson(f as Map<String, dynamic>),
  ],
  state: ChangeState.values.asNameMap()[j['state']] ?? ChangeState.failed,
  undoId: j['undo_id'] as String?,
);

ReminderInfo reminderInfoFromJson(Map<String, dynamic> j) =>
    ReminderInfo(id: j['id'] as String, state: j['state'] as String);

Entry entryFromJson(Map<String, dynamic> j) => Entry(
  id: j['id'] as String,
  cursor: (j['cursor'] as num?)?.toInt() ?? 0,
  kind: j['kind'] as String,
  text: j['text'] as String? ?? '',
  createdAt: DateTime.parse(j['created_at'] as String),
  change: j['change'] == null
      ? null
      : changeFromJson(j['change'] as Map<String, dynamic>),
  reminder: j['reminder'] == null
      ? null
      : reminderInfoFromJson(j['reminder'] as Map<String, dynamic>),
  costEur: (j['cost_eur'] as num?)?.toDouble() ?? 0,
);

Item itemFromJson(Map<String, dynamic> j) {
  DateTime? time(String key) {
    final raw = j[key] as String?;
    return raw == null || raw.isEmpty ? null : DateTime.parse(raw).toLocal();
  }

  return Item(
    id: j['id'] as String,
    title: j['title'] as String,
    status: j['status'] as String,
    startAt: time('start_at'),
    dueAt: time('due_at'),
    durationMinutes: (j['duration_minutes'] as num?)?.toInt() ?? 0,
    priority: (j['priority'] as num?)?.toInt() ?? 0,
    location: j['location'] as String? ?? '',
    list: j['list'] as String? ?? '',
    repeat: _repeat(j['recurrence'] as Map<String, dynamic>?),
  );
}

String _repeat(Map<String, dynamic>? r) {
  if (r == null) return '';
  final unit = {
    'daily': 'day',
    'weekly': 'week',
    'monthly': 'month',
    'yearly': 'year',
  }[r['every']];
  final n = (r['interval'] as num?)?.toInt() ?? 1;
  final base = n > 1 ? 'every $n ${unit}s' : 'every $unit';
  return r['after_completion'] == true ? '$base after completion' : base;
}

PushConfiguration pushConfigurationFromJson(Map<String, dynamic> json) =>
    PushConfiguration(
      appId: json['app_id'] as String? ?? '',
      apiKey: json['api_key'] as String? ?? '',
      projectId: json['project_id'] as String? ?? '',
      senderId: json['sender_id'] as String? ?? '',
    );

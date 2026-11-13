import '../domain/capture.dart';
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
  reminderOffsets: [
    for (final m in j['reminder_offsets_minutes'] as List? ?? []) m as int,
  ],
  reminderChoices: [
    for (final m in j['reminder_choices_minutes'] as List? ?? []) m as int,
  ],
  removedEntryIds: [
    for (final id in j['removed_entry_ids'] as List? ?? []) id as String,
  ],
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
  capture: j['capture'] is Map<String, dynamic>
      ? captureFromJson(j['capture'] as Map<String, dynamic>)
      : null,
  captureId:
      j['capture_id'] as String? ??
      ((j['capture'] as Map<String, dynamic>?)?['id'] as String? ?? ''),
  costEur: (j['cost_eur'] as num?)?.toDouble() ?? 0,
);

Item itemFromJson(Map<String, dynamic> j) {
  DateTime? time(String key) {
    final raw = j[key] as String?;
    return raw == null || raw.isEmpty
        ? null
        : (raw.contains('T')
              ? DateTime.parse(raw).toLocal()
              : DateTime.parse(raw));
  }

  return Item(
    id: j['id'] as String,
    title: j['title'] as String,
    status: j['status'] as String,
    startAt: time('start_at'),
    endAt: time('end_at'),
    sourceIds: [for (final id in j['source_ids'] as List? ?? []) id as String],
    startDate: time('start_date'),
    endDate: time('end_date'),
    dueDate: time('due_date'),
    startLocal: j['start_local'] as String? ?? '',
    endLocal: j['end_local'] as String? ?? '',
    startTimezone: j['start_timezone'] as String? ?? '',
    endTimezone: j['end_timezone'] as String? ?? '',
    dueAt: time('due_at'),
    durationMinutes: (j['duration_minutes'] as num?)?.toInt() ?? 0,
    priority: (j['priority'] as num?)?.toInt() ?? 0,
    version: (j['version'] as num?)?.toInt() ?? 0,
    location: j['location'] as String? ?? '',
    list: j['list'] as String? ?? '',
    repeat: _repeat(j['recurrence'] as Map<String, dynamic>?),
    kind: j['kind'] as String? ?? '',
    notes: j['notes'] as String? ?? '',
    reminderOffsets: [
      for (final m in j['reminder_offsets_minutes'] as List? ?? [])
        (m as num).toInt(),
    ],
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

Capture captureFromJson(Map<String, dynamic> j) => Capture(
  id: j['id'] as String,
  status: j['status'] as String,
  revision: (j['revision'] as num?)?.toInt() ?? 0,
  sender: j['sender'] as String? ?? '',
  subject: j['subject'] as String? ?? '',
  error: j['error'] as String? ?? '',
  deleted: j['deleted'] == true,
  coverageAcknowledged: j['coverage_acknowledged'] == true,
  questions: [
    for (final q in j['questions'] as List? ?? [])
      if (q is String)
        CaptureQuestion(text: q)
      else
        CaptureQuestion(
          id: (q as Map<String, dynamic>)['id'] as String? ?? '',
          text: q['text'] as String? ?? '',
          options: [for (final o in q['options'] as List? ?? []) o as String],
        ),
  ],
  coverage: [
    for (final p in j['coverage'] as List? ?? [])
      if (p['status'] != 'parsed' && p['status'] != 'not-actionable')
        '${p['name'] ?? p['part']}: ${p['detail'] ?? p['status']}',
  ],
  attachments: [
    for (final a in j['attachments'] as List? ?? [])
      CaptureAttachment(
        id: a['id'] as String,
        name: a['name'] as String,
        mediaType: a['media_type'] as String,
        size: (a['size'] as num).toInt(),
      ),
  ],
  candidates: [
    for (final c in j['candidates'] as List? ?? [])
      if (c['state'] != 'superseded')
        CaptureCandidate(
          id: c['id'] as String,
          state: c['state'] as String,
          title: c['proposal']['title'] as String? ?? '',
          changeId: c['change_id'] as String? ?? '',
          question: c['question'] as String? ?? '',
          options: [for (final o in c['options'] as List? ?? []) o as String],
          startAt: c['proposal']['start_at'] as String? ?? '',
          startDate: c['proposal']['start_date'] as String? ?? '',
          dueAt: c['proposal']['due_at'] as String? ?? '',
          dueDate: c['proposal']['due_date'] as String? ?? '',
          location: c['proposal']['location'] as String? ?? '',
          evidence: [
            for (final e in c['evidence'] as List? ?? [])
              '${e['part']}: ${e['excerpt']}',
          ],
        ),
  ],
);

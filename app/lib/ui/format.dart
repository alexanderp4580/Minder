import '../domain/capture.dart';
import '../domain/entry.dart';

const _days = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
const _months = [
  'Jan',
  'Feb',
  'Mar',
  'Apr',
  'May',
  'Jun',
  'Jul',
  'Aug',
  'Sep',
  'Oct',
  'Nov',
  'Dec',
];

String _two(int n) => n.toString().padLeft(2, '0');

/// 24-hour time, as in the mockups: 07:30.
String clock(DateTime t) => '${_two(t.hour)}:${_two(t.minute)}';

/// A date and time as the mockups show them: Fri 13 Nov 09:00.
String dateTime(DateTime t) =>
    '${_days[t.weekday - 1]} ${t.day} ${_months[t.month - 1]} ${clock(t)}';

/// The model cost of a reply. Amounts under a hundredth of a cent show as a bound.
String cost(double eur) =>
    eur < 0.0001 ? '<€0.0001' : '€${eur.toStringAsFixed(4)}';

/// The heading of a preview card.
String actionLabel(String action) => switch (action) {
  'create_item' => 'CREATE ITEM',
  'delete_capture' => 'DELETE EMAIL SOURCE',
  'cancel_item' => 'CANCEL ITEM',
  'delete_item' => 'DELETE ITEM',
  'complete_item' => 'COMPLETE ITEM',
  'create_profile' => 'ADD PROFILE',
  'update_profile' => 'CHANGE PROFILE',
  _ => 'CHANGE ITEM',
};

/// The name of a changed field, as the preview shows it.
String fieldLabel(String name) => switch (name) {
  'start_at' => 'Starts',
  'end_at' => 'Ends',
  'start_date' => 'Start date',
  'end_date' => 'End date / checkout',
  'due_date' => 'Due date',
  'kind' => 'Type',
  'start_timezone' => 'Departure / start timezone',
  'end_timezone' => 'Arrival / end timezone',
  'duration_minutes' => 'Duration',
  'priority' => 'Priority',
  'location' => 'Address',
  'list' => 'List',
  'profile' => 'Profile',
  'recurrence' => 'Repeats',
  'name' => 'Name',
  'remind_before_minutes' => 'Remind before (minutes)',
  'due_at' => 'Due',
  'remind_at' => 'Remind at',
  'reminders' => 'Reminders',
  'nag_minutes' => 'Nag interval (minutes)',
  'title' => 'Title',
  'notes' => 'Notes',
  'status' => 'Status',
  'timezone' => 'Timezone',
  'item' => 'Item',
  'email source' => 'Email source',
  _ => name,
};

final _offset = RegExp(r'([+-])(\d\d):(\d\d)$');

/// A field value: a time as "Fri 13 Nov 09:00", an empty value as "None".
/// A time with an explicit offset shows the wall clock at that offset, as written.
String fieldValue(String value) {
  if (value.isEmpty) return 'None';
  final date = DateTime.tryParse(value);
  if (date == null || !value.contains('T')) return value;
  final match = _offset.firstMatch(value);
  if (match != null) {
    final minutes =
        (int.parse(match[2]!) * 60 + int.parse(match[3]!)) *
        (match[1] == '-' ? -1 : 1);
    return dateTime(date.toUtc().add(Duration(minutes: minutes)));
  }
  return dateTime(date.toLocal());
}

/// "45 minutes", "1 hour 30 minutes", "2 days".
String durationText(int minutes) {
  String unit(int n, String name) => '$n $name${n == 1 ? '' : 's'}';
  if (minutes >= 1440 && minutes % 1440 == 0) {
    return unit(minutes ~/ 1440, 'day');
  }
  final hours = minutes ~/ 60, rest = minutes % 60;
  return [
    if (hours > 0) unit(hours, 'hour'),
    if (rest > 0 || hours == 0) unit(rest, 'minute'),
  ].join(' ');
}

/// A reminder offset as a chip label: "10 minutes", "1 day", "1 week".
String offsetLabel(int minutes) {
  if (minutes % 10080 == 0) {
    final weeks = minutes ~/ 10080;
    return '$weeks week${weeks == 1 ? '' : 's'}';
  }
  return durationText(minutes);
}

String kindLabel(String kind) => switch (kind) {
  'event' => 'Appointment',
  'task' => 'Task',
  'reminder' => 'Reminder',
  _ => kind,
};

/// Notes as plain lines: list markers are dropped, nothing is interpreted.
/// At most [maxChars] characters show; a cut ends with an ellipsis.
List<String> noteLines(String notes, {int maxChars = 600}) {
  final lines = <String>[];
  var used = 0;
  for (final raw in notes.split('\n')) {
    var line = raw.trim();
    if (line.startsWith('- ') || line.startsWith('* ')) {
      line = line.substring(2).trim();
    }
    if (line.isEmpty) continue;
    if (used + line.length > maxChars) {
      lines.add('${line.substring(0, maxChars - used)}…');
      return lines;
    }
    lines.add(line);
    used += line.length;
  }
  return lines;
}

/// One row of a preview card.
class PreviewRow {
  const PreviewRow(
    this.label, {
    this.oldValue = '',
    this.newValue = '',
    this.bullets = const [],
  });
  final String label, oldValue, newValue;
  final List<String> bullets;
}

/// The rows a preview card shows: the title once, timezones inline, units in values.
List<PreviewRow> previewRows(Change change) {
  String? value(String name) {
    for (final f in change.fields) {
      if (f.name == name && f.newValue.isNotEmpty) return f.newValue;
    }
    return null;
  }

  final itemZone = value('timezone');
  final startZone = value('start_timezone') ?? itemZone;
  final zones = {
    'start_at': startZone,
    'end_at': value('end_timezone') ?? startZone,
    'due_at': itemZone,
    'remind_at': itemZone,
  };
  final hasTime = zones.keys.any((n) => change.fields.any((f) => f.name == n));
  final hasStart = change.fields.any((f) => f.name == 'start_at');
  final hasEnd = change.fields.any((f) => f.name == 'end_at');
  final rows = <PreviewRow>[];
  for (final f in change.fields) {
    switch (f.name) {
      case 'title' when f.oldValue.isEmpty:
      case 'start_timezone' when hasStart:
      case 'end_timezone' when hasEnd:
      case 'timezone' when hasTime:
      case 'status' when f.oldValue.isEmpty && f.newValue == 'open':
      case 'reminders' when change.reminderChoices.isNotEmpty:
        continue;
    }
    String show(String raw) {
      if (raw.isEmpty) return '';
      switch (f.name) {
        case 'duration_minutes':
          final minutes = int.tryParse(raw);
          return minutes == null ? raw : durationText(minutes);
        case 'kind':
          return kindLabel(raw);
      }
      final text = fieldValue(raw);
      final zone = zones[f.name];
      return zone == null ? text : '$text ($zone)';
    }

    rows.add(
      f.name == 'notes'
          ? PreviewRow(fieldLabel(f.name), bullets: noteLines(f.newValue))
          : PreviewRow(
              fieldLabel(f.name),
              oldValue: show(f.oldValue),
              newValue: f.newValue.isEmpty ? 'None' : show(f.newValue),
            ),
    );
  }
  return rows;
}

/// When an email candidate happens, as one short phrase: "Fri 20 Nov 14:30" or "Due 20 Nov".
String candidateWhen(CaptureCandidate c) {
  if (c.startAt.isNotEmpty) return fieldValue(c.startAt);
  if (c.startDate.isNotEmpty) return c.startDate;
  if (c.dueAt.isNotEmpty) return 'Due ${fieldValue(c.dueAt)}';
  if (c.dueDate.isNotEmpty) return 'Due ${c.dueDate}';
  return '';
}

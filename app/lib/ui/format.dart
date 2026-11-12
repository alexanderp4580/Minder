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
  'complete_item' => 'COMPLETE ITEM',
  'create_profile' => 'ADD PROFILE',
  'update_profile' => 'CHANGE PROFILE',
  _ => 'CHANGE ITEM',
};

/// The name of a changed field, as the preview shows it.
String fieldLabel(String name) => switch (name) {
  'start_at' => 'Starts',
  'duration_minutes' => 'Duration (minutes)',
  'priority' => 'Priority',
  'location' => 'Location',
  'list' => 'List',
  'profile' => 'Profile',
  'recurrence' => 'Repeats',
  'name' => 'Name',
  'remind_before_minutes' => 'Remind before (minutes)',
  'due_at' => 'Due',
  'remind_at' => 'Remind at',
  'nag_minutes' => 'Nag interval (minutes)',
  'title' => 'Title',
  'notes' => 'Notes',
  'status' => 'Status',
  'timezone' => 'Timezone',
  _ => name,
};

/// A field value: a time as "Fri 13 Nov 09:00", an empty value as "None".
String fieldValue(String value) {
  if (value.isEmpty) return 'None';
  final date = DateTime.tryParse(value);
  if (date == null || !value.contains('T')) return value;
  return dateTime(date.toLocal());
}

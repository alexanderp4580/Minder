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

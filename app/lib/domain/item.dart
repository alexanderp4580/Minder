class Item {
  const Item({
    required this.id,
    required this.title,
    required this.status,
    this.startAt,
    this.dueAt,
    this.durationMinutes = 0,
    this.priority = 0,
    this.location = '',
    this.list = '',
    this.repeat = '',
  });
  final String id, title, status, location, list, repeat;
  final DateTime? startAt, dueAt;
  final int durationMinutes, priority;

  /// The time an item shows on the calendar: the start, else the deadline.
  DateTime? get placedAt => startAt ?? dueAt;
  bool get done => status == 'done' || status == 'cancelled';
}

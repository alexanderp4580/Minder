class Item {
  const Item({
    required this.id,
    required this.title,
    required this.status,
    this.startAt,
    this.endAt,
    this.startDate,
    this.endDate,
    this.dueDate,
    this.sourceIds = const [],
    this.startLocal = '',
    this.endLocal = '',
    this.startTimezone = '',
    this.endTimezone = '',

    this.dueAt,
    this.durationMinutes = 0,
    this.priority = 0,
    this.version = 0,
    this.location = '',
    this.list = '',
    this.repeat = '',
    this.kind = '',
    this.notes = '',
    this.reminderOffsets = const [],
  });
  final String id, title, status, location, list, repeat, kind, notes;

  /// Minutes before the start or deadline at which a reminder fires.
  final List<int> reminderOffsets;
  final DateTime? startAt, dueAt, endAt, startDate, endDate, dueDate;
  final List<String> sourceIds;
  final String startLocal, endLocal, startTimezone, endTimezone;
  final int durationMinutes, priority, version;

  /// The time an item shows on the calendar: the start, else the deadline.
  DateTime? get placedAt => startAt ?? dueAt ?? startDate ?? dueDate;
  bool occursOn(DateTime day) {
    final begin = startDate ?? startAt ?? dueDate ?? dueAt;
    if (begin == null) return false;
    final low = DateTime(day.year, day.month, day.day),
        high = DateTime(day.year, day.month, day.day + 1);
    final end = endDate ?? endAt;
    return end == null
        ? !begin.isBefore(low) && begin.isBefore(high)
        : begin.isBefore(high) && end.isAfter(low);
  }

  bool get done => status == 'done' || status == 'cancelled';
}

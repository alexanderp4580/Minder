class Detail {
  const Detail({required this.label, required this.value});
  final String label, value;
}

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
    this.remindAt,
    this.durationMinutes = 0,
    this.priority = 0,
    this.version = 0,
    this.location = '',
    this.locationKind = '',
    this.color = 'slate',
    this.dateEnd = '',
    this.phone = '',
    this.email = '',
    this.website = '',
    this.bookingReference = '',
    this.details = const [],
    this.list = '',
    this.repeat = '',
    this.kind = '',
    this.notes = '',
    this.reminderOffsets = const [],
  });
  final String id, title, status, location, list, repeat, kind, notes;
  final String phone, email, website, bookingReference;
  final String color, locationKind, dateEnd;
  final List<Detail> details;

  /// Minutes before the start or deadline at which a reminder fires.
  final List<int> reminderOffsets;
  final DateTime? startAt, dueAt, endAt, startDate, endDate, dueDate, remindAt;
  final List<String> sourceIds;
  final String startLocal, endLocal, startTimezone, endTimezone;
  final int durationMinutes, priority, version;

  /// The time an item shows on the calendar: the start, else the deadline.
  DateTime? get placedAt => startAt ?? startDate ?? dueAt ?? dueDate;
  bool occursOn(DateTime day) {
    final begin = startDate ?? startAt ?? dueDate ?? dueAt;
    if (begin == null) return false;
    final low = DateTime(day.year, day.month, day.day),
        high = DateTime(day.year, day.month, day.day + 1);
    final end = endDate == null
        ? endAt
        : dateEnd == 'inclusive'
        ? DateTime(endDate!.year, endDate!.month, endDate!.day + 1)
        : endDate;
    return end == null
        ? !begin.isBefore(low) && begin.isBefore(high)
        : begin.isBefore(high) && end.isAfter(low);
  }

  bool get done => status == 'done' || status == 'cancelled';
}

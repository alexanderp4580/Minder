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
  factory Item.fromJson(Map<String, dynamic> j) {
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
  static String _repeat(Map<String, dynamic>? r) {
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

  final String id, title, status, location, list, repeat;
  final DateTime? startAt, dueAt;
  final int durationMinutes, priority;

  /// The time an item shows on the calendar: the start, else the deadline.
  DateTime? get placedAt => startAt ?? dueAt;
  bool get done => status == 'done' || status == 'cancelled';
}

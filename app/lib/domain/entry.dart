class Entry {
  const Entry({
    required this.id,
    required this.cursor,
    required this.kind,
    required this.text,
    required this.createdAt,
    this.change,
  });
  factory Entry.fromJson(Map<String, dynamic> j) => Entry(
    id: j['id'] as String,
    cursor: (j['cursor'] as num?)?.toInt() ?? 0,
    kind: j['kind'] as String,
    text: j['text'] as String? ?? '',
    createdAt: DateTime.parse(j['created_at'] as String),
    change: j['change'] as Map<String, dynamic>?,
  );
  final String id, kind, text;
  final int cursor;
  final DateTime createdAt;
  final Map<String, dynamic>? change;
}

import 'package:flutter/foundation.dart';

import '../domain/chat_repository.dart';
import '../domain/item.dart';
import 'use_cases.dart';

class CalendarController extends ChangeNotifier {
  CalendarController(this._load, {DateTime Function()? now})
    : _now = now ?? DateTime.now {
    final today = _day(_now());
    selected = today;
    month = DateTime(today.year, today.month);
  }
  final LoadItems _load;
  final DateTime Function() _now;
  late DateTime month, selected;
  bool loading = false;
  String? error;
  bool _disposed = false;
  List<Item> _items = [];

  static DateTime _day(DateTime t) => DateTime(t.year, t.month, t.day);

  /// The Monday on or before the first of the month.
  DateTime get gridStart {
    final first = DateTime(month.year, month.month);
    return DateTime(first.year, first.month, first.day - (first.weekday - 1));
  }

  DateTime get today => _day(_now());
  List<Item> itemsOn(DateTime day) {
    final out = _items.where((i) => _day(i.placedAt!) == _day(day)).toList()
      ..sort((a, b) => a.placedAt!.compareTo(b.placedAt!));
    return out;
  }

  int countOn(DateTime day) => itemsOn(day).length;
  List<Item> get selectedItems => itemsOn(selected);

  Future<void> load() async {
    if (_disposed) return;
    loading = true;
    notifyListeners();
    try {
      final start = gridStart;
      final end = DateTime(start.year, start.month, start.day + 42);
      _items = await _load(start, end);
      error = null;
    } catch (e) {
      error = e is ApiException
          ? e.message
          : 'Check your connection. Then try again.';
    }
    loading = false;
    if (!_disposed) notifyListeners();
  }

  Future<void> _moveMonth(DateTime to) {
    month = DateTime(to.year, to.month);
    selected = month;
    return load();
  }

  Future<void> next() => _moveMonth(DateTime(month.year, month.month + 1));
  Future<void> previous() => _moveMonth(DateTime(month.year, month.month - 1));
  Future<void> goToday() async {
    final t = today;
    month = DateTime(t.year, t.month);
    selected = t;
    await load();
  }

  void select(DateTime day) {
    selected = _day(day);
    notifyListeners();
  }

  @override
  void dispose() {
    _disposed = true;
    super.dispose();
  }
}

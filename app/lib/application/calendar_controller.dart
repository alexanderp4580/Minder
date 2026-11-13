import 'package:flutter/foundation.dart';

import '../domain/failure.dart';
import '../domain/entry.dart';
import '../domain/item.dart';
import '../domain/item_repository.dart';

class CalendarController extends ChangeNotifier {
  CalendarController(this._repository, {DateTime Function()? now})
    : _now = now ?? DateTime.now {
    final today = _day(_now());
    selected = today;
    month = DateTime(today.year, today.month);
  }
  final ItemRepository _repository;
  final DateTime Function() _now;
  late DateTime month, selected;
  bool loading = false;
  String? error;
  bool _disposed = false;
  int _request = 0;
  List<Item> _items = [];

  static DateTime _day(DateTime t) => DateTime(t.year, t.month, t.day);

  /// The Monday on or before the first of the month.
  DateTime get gridStart {
    final first = DateTime(month.year, month.month);
    return DateTime(first.year, first.month, first.day - (first.weekday - 1));
  }

  DateTime get today => _day(_now());
  List<Item> itemsOn(DateTime day) {
    final out = _items.where((i) => i.occursOn(day)).toList()
      ..sort((a, b) => a.placedAt!.compareTo(b.placedAt!));
    return out;
  }

  int countOn(DateTime day) => itemsOn(day).length;
  List<Item> get selectedItems => itemsOn(selected);

  Future<Entry> previewDelete(Item item) => _repository.previewDelete(item);

  Future<void> confirmDelete(String changeId) async {
    await _repository.confirm(changeId);
    await load();
  }

  Future<void> discardDelete(String changeId) async {
    await _repository.discard(changeId);
  }

  Future<void> load() async {
    if (_disposed) return;
    final request = ++_request;
    loading = true;
    error = null;
    notifyListeners();
    try {
      final start = gridStart;
      final end = DateTime(start.year, start.month, start.day + 42);
      final items = await _repository.itemsBetween(start, end);
      if (_disposed || request != _request) return;
      _items = items;
      error = null;
    } catch (e) {
      if (_disposed || request != _request) return;
      error = e is RepositoryFailure
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

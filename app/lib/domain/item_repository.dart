import 'item.dart';

abstract interface class ItemRepository {
  /// Items placed in [from, to).
  Future<List<Item>> itemsBetween(DateTime from, DateTime to);
}

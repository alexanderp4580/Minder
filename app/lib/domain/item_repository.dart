import 'entry.dart';
import 'item.dart';

abstract interface class ItemRepository {
  /// Items placed in [from, to).
  Future<List<Item>> itemsBetween(DateTime from, DateTime to);

  /// Asks the server what deleting [item] would remove. Nothing is deleted yet.
  Future<Entry> previewDelete(Item item);
  Future<Entry> previewColor(Item item, String color);

  /// Deletes what [previewDelete] described.
  Future<Entry> confirm(String changeId);

  /// Keeps the item.
  Future<Entry> discard(String changeId);
}

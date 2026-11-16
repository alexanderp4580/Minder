import 'entry.dart';
import 'item.dart';

/// Deleting an item: the server previews what goes, then the owner confirms or keeps it.
abstract interface class ItemDeletion {
  /// Asks the server what deleting [item] would remove. Nothing is deleted yet.
  Future<Entry> previewDelete(Item item);

  /// Deletes what [previewDelete] described.
  Future<Entry> confirm(String changeId);

  /// Keeps the item.
  Future<Entry> discard(String changeId);
}

abstract interface class ItemRepository implements ItemDeletion {
  /// Items placed in [from, to).
  Future<List<Item>> itemsBetween(DateTime from, DateTime to);

  Future<Entry> previewColor(Item item, String color);
}

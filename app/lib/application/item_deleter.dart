import '../domain/entry.dart';
import '../domain/item.dart';

/// What the delete dialog needs: a preview of the removal, then Confirm or Keep it.
abstract interface class ItemDeleter {
  Future<Entry> previewDelete(Item item);
  Future<void> confirmDelete(String changeId);
  Future<void> discardDelete(String changeId);
}

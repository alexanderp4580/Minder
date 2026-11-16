import '../domain/board.dart';
import '../domain/entry.dart';
import '../domain/item.dart';
import '../domain/item_repository.dart';
import 'chat_api.dart';
import 'wire.dart';

/// The Board routes. Delete uses the shared item delete flow, so it goes through [deletion].
class BoardApi implements BoardRepository {
  BoardApi(this._request, this._deletion);
  final ApiRequest _request;
  final ItemDeletion _deletion;

  @override
  Future<Board> board() async =>
      boardFromJson(await _request('GET', '/v1/board') as Map<String, dynamic>);

  @override
  Future<Item> addTask(String title) async => itemFromJson(
    await _request('POST', '/v1/items', body: {'title': title})
        as Map<String, dynamic>,
  );

  @override
  Future<Item> moveTask(Item item, String status) async => itemFromJson(
    await _request(
      'POST',
      '/v1/items/${Uri.encodeComponent(item.id)}/status',
      body: {'expected_version': item.version, 'status': status},
    ) as Map<String, dynamic>,
  );

  @override
  Future<Entry> previewDelete(Item item) => _deletion.previewDelete(item);
  @override
  Future<Entry> confirm(String changeId) => _deletion.confirm(changeId);
  @override
  Future<Entry> discard(String changeId) => _deletion.discard(changeId);
}

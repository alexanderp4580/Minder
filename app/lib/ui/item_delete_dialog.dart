import 'package:flutter/material.dart';

import '../application/calendar_controller.dart';
import '../domain/entry.dart';
import '../domain/failure.dart';
import '../domain/item.dart';
import 'format.dart';
import 'tokens.dart';

/// Asks the server what a delete removes, then deletes or keeps the item.
/// Closing the dialog any other way discards the preview.
class ItemDeleteDialog extends StatefulWidget {
  const ItemDeleteDialog({
    super.key,
    required this.controller,
    required this.item,
  });
  final CalendarController controller;
  final Item item;
  @override
  State<ItemDeleteDialog> createState() => _ItemDeleteDialogState();
}

class _ItemDeleteDialogState extends State<ItemDeleteDialog> {
  late final Future<Entry> _preview;
  bool _decided = false, _busy = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _preview = widget.controller.previewDelete(widget.item);
  }

  String _message(Object e) => e is RepositoryFailure
      ? e.message
      : 'Check your connection. Then try again.';

  Future<void> _finish(String changeId, {required bool delete}) async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      if (delete) {
        await widget.controller.confirmDelete(changeId);
      } else {
        await widget.controller.discardDelete(changeId);
      }
      _decided = true;
      if (mounted) Navigator.of(context).pop();
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _busy = false;
        _error = _message(e);
      });
    }
  }

  /// Runs when the dialog closes. A preview nobody answered would stay in the chat.
  void _discardIfOpen() {
    if (_decided) return;
    _decided = true;
    _preview
        .then((e) => widget.controller.discardDelete(e.change!.id))
        .ignore();
  }

  Widget _content(AsyncSnapshot<Entry> snapshot) {
    if (snapshot.hasError) return Text(_message(snapshot.error!));
    final entry = snapshot.data;
    if (entry == null) {
      return const SizedBox(
        height: 60,
        child: Center(child: CircularProgressIndicator()),
      );
    }
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(entry.text),
        const SizedBox(height: 12),
        for (final row in previewRows(entry.change!))
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Wrap(
              spacing: 8,
              children: [
                Text(
                  '${row.label}:',
                  style: const TextStyle(color: Tokens.muted),
                ),
                if (row.oldValue.isNotEmpty) Text(row.oldValue),
                const Text('→'),
                Text(row.newValue),
              ],
            ),
          ),
        if (_error != null)
          Text(_error!, style: const TextStyle(color: Tokens.alert)),
      ],
    );
  }

  List<Widget> _actions(Entry? entry) {
    if (entry == null) {
      return [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Close'),
        ),
      ];
    }
    final id = entry.change!.id;
    return [
      TextButton(
        onPressed: _busy ? null : () => _finish(id, delete: false),
        child: const Text('Keep it'),
      ),
      FilledButton(
        onPressed: _busy ? null : () => _finish(id, delete: true),
        child: const Text('Delete'),
      ),
    ];
  }

  @override
  Widget build(BuildContext context) => PopScope(
    canPop: !_busy,
    onPopInvokedWithResult: (didPop, _) {
      if (didPop) _discardIfOpen();
    },
    child: FutureBuilder<Entry>(
      future: _preview,
      builder: (context, snapshot) => AlertDialog(
        title: const Text('Delete item'),
        content: SizedBox(width: 420, child: _content(snapshot)),
        actions: _actions(snapshot.data),
      ),
    ),
  );
}

import 'package:flutter/material.dart';

import '../application/calendar_controller.dart';
import '../domain/entry.dart';
import '../domain/failure.dart';
import '../domain/item.dart';
import 'tokens.dart';

/// Asks the server what a delete removes, then deletes or keeps the item.
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
  late Future<Entry> _preview;
  bool _busy = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _preview = widget.controller.previewDelete(widget.item);
  }

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
      if (mounted) Navigator.of(context).pop();
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _busy = false;
        _error = e is RepositoryFailure
            ? e.message
            : 'Check your connection. Then try again.';
      });
    }
  }

  @override
  Widget build(BuildContext context) => FutureBuilder<Entry>(
    future: _preview,
    builder: (context, snapshot) {
      final change = snapshot.data?.change;
      final failed = snapshot.hasError;
      return AlertDialog(
        title: const Text('Delete item'),
        content: SizedBox(
          width: 420,
          child: failed
              ? Text(
                  snapshot.error is RepositoryFailure
                      ? (snapshot.error as RepositoryFailure).message
                      : 'Check your connection. Then try again.',
                )
              : change == null
              ? const SizedBox(
                  height: 60,
                  child: Center(child: CircularProgressIndicator()),
                )
              : Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(snapshot.data!.text),
                    const SizedBox(height: 12),
                    for (final f in change.fields)
                      Padding(
                        padding: const EdgeInsets.only(bottom: 8),
                        child: Wrap(
                          spacing: 8,
                          children: [
                            Text(
                              f.oldValue,
                              style: const TextStyle(color: Tokens.muted),
                            ),
                            const Text('→'),
                            Text(f.newValue),
                          ],
                        ),
                      ),
                    if (_error != null)
                      Text(
                        _error!,
                        style: const TextStyle(color: Tokens.alert),
                      ),
                  ],
                ),
        ),
        actions: [
          if (change == null)
            TextButton(
              onPressed: () => Navigator.of(context).pop(),
              child: const Text('Close'),
            )
          else ...[
            TextButton(
              onPressed: _busy ? null : () => _finish(change.id, delete: false),
              child: const Text('Keep it'),
            ),
            FilledButton(
              onPressed: _busy ? null : () => _finish(change.id, delete: true),
              child: const Text('Delete'),
            ),
          ],
        ],
      );
    },
  );
}

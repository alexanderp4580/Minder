import 'package:flutter/material.dart';

import '../domain/board.dart';
import '../domain/item.dart';
import 'format.dart';
import 'item_colours.dart';
import 'tokens.dart';

/// The column a task moves to when its card button is pressed, and that button's label.
({String to, String label}) forwardMove(String status) => switch (status) {
  BoardStatus.open => (to: BoardStatus.inProgress, label: 'Start'),
  BoardStatus.inProgress => (to: BoardStatus.done, label: 'Done'),
  _ => (to: BoardStatus.open, label: 'Reopen'),
};

/// One task on the board: a colour stripe, the title, a few facts and one forward button.
class BoardTaskCard extends StatelessWidget {
  const BoardTaskCard({
    super.key,
    required this.item,
    required this.busy,
    required this.onOpen,
    required this.onMove,
  });
  final Item item;
  final bool busy;
  final VoidCallback onOpen;
  final ValueChanged<String> onMove;

  @override
  Widget build(BuildContext context) {
    final done = item.status == BoardStatus.done;
    final move = forwardMove(item.status);
    final facts = <Widget>[
      if (item.priority > 0) Text('Priority ${item.priority}'),
      if (item.list.isNotEmpty) Text(item.list),
      if (item.remindAt != null)
        Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.notifications_none, size: 14),
            const SizedBox(width: 4),
            Text(dateTime(item.remindAt!)),
          ],
        ),
    ];
    return Material(
      color: Tokens.panel,
      shape: const Border.fromBorderSide(BorderSide(color: Tokens.line)),
      child: IntrinsicHeight(
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            SizedBox(
              key: ValueKey('stripe-${item.id}'),
              width: 4,
              child: ColoredBox(color: itemColour(item.color)),
            ),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  InkWell(
                    onTap: onOpen,
                    child: Padding(
                      padding: const EdgeInsets.fromLTRB(16, 14, 16, 4),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          SizedBox(
                            width: double.infinity,
                            child: Text(
                              item.title,
                              maxLines: 3,
                              overflow: TextOverflow.ellipsis,
                              style: TextStyle(
                                fontSize: 15,
                                fontWeight: FontWeight.w600,
                                color: done ? Tokens.muted : Tokens.text,
                                decoration: done
                                    ? TextDecoration.lineThrough
                                    : null,
                              ),
                            ),
                          ),
                          if (facts.isNotEmpty) ...[
                            const SizedBox(height: 6),
                            DefaultTextStyle.merge(
                              style: const TextStyle(
                                fontSize: 12,
                                color: Tokens.muted,
                              ),
                              child: IconTheme.merge(
                                data: const IconThemeData(color: Tokens.muted),
                                child: Wrap(
                                  spacing: 12,
                                  runSpacing: 4,
                                  children: facts,
                                ),
                              ),
                            ),
                          ],
                        ],
                      ),
                    ),
                  ),
                  Padding(
                    padding: const EdgeInsets.fromLTRB(16, 4, 12, 10),
                    child: Align(
                      alignment: Alignment.centerRight,
                      child: OutlinedButton(
                        key: ValueKey('move-${item.id}'),
                        onPressed: busy ? null : () => onMove(move.to),
                        child: Text(move.label),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

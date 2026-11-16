import 'package:flutter/material.dart';

import '../domain/board.dart';
import '../domain/item.dart';
import '../domain/link_launcher.dart';
import 'calendar_item_details.dart';
import 'carbon_icon.dart';
import 'tokens.dart';

/// Opens the details of one task. [onMove] receives the chosen column; [onDelete] starts the delete preview.
Future<void> showBoardTaskSheet(
  BuildContext context, {
  required Item item,
  required LinkLauncher? launcher,
  required ValueChanged<String> onMove,
  required VoidCallback onDelete,
}) => showModalBottomSheet<void>(
  context: context,
  isScrollControlled: true,
  backgroundColor: Tokens.panel,
  shape: const RoundedRectangleBorder(),
  builder: (sheet) => SafeArea(
    child: ConstrainedBox(
      constraints: BoxConstraints(
        maxHeight: MediaQuery.sizeOf(sheet).height * .85,
      ),
      child: SingleChildScrollView(
        key: const ValueKey('task-sheet'),
        padding: const EdgeInsets.fromLTRB(20, 20, 20, 24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Expanded(
                  child: Text(
                    item.title,
                    style: const TextStyle(
                      fontSize: 18,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                ),
                IconButton(
                  key: const ValueKey('sheet-delete'),
                  tooltip: 'Delete',
                  onPressed: () {
                    Navigator.of(sheet).pop();
                    onDelete();
                  },
                  icon: const CarbonIcon(
                    CarbonGlyph.delete,
                    color: Tokens.muted,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 4),
            Text(
              BoardStatus.label(item.status),
              style: const TextStyle(color: Tokens.muted, fontSize: 12),
            ),
            if (item.list.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 4),
                child: Text(
                  item.list,
                  style: const TextStyle(color: Tokens.muted, fontSize: 12),
                ),
              ),
            CalendarItemDetails(
              item: item,
              launcher: launcher,
              showTiming: false,
            ),
            const SizedBox(height: 24),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                for (final status in BoardStatus.all)
                  if (status != item.status)
                    OutlinedButton(
                      key: ValueKey('sheet-move-$status'),
                      onPressed: () {
                        Navigator.of(sheet).pop();
                        onMove(status);
                      },
                      child: Text('Move to ${BoardStatus.label(status)}'),
                    ),
              ],
            ),
          ],
        ),
      ),
    ),
  ),
);

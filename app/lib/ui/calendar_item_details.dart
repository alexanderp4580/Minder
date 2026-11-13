import 'package:flutter/material.dart';

import '../domain/item.dart';
import '../domain/link_launcher.dart';
import '../domain/links.dart';
import 'format.dart';
import 'linked_text.dart';
import 'tokens.dart';

class CalendarItemDetails extends StatelessWidget {
  const CalendarItemDetails({super.key, required this.item, this.launcher});
  final Item item;
  final LinkLauncher? launcher;

  @override
  Widget build(BuildContext context) {
    final trip =
        item.startTimezone.isNotEmpty &&
        item.endTimezone.isNotEmpty &&
        item.startTimezone != item.endTimezone;
    String when(String local, DateTime? instant, DateTime? date, String zone) {
      if (local.isEmpty && instant == null && date == null) return '';
      if (date != null) return date.toIso8601String().split('T').first;
      final value = local.isNotEmpty
          ? local.replaceFirst('T', ' ')
          : instant == null
          ? ''
          : dateTime(instant.toLocal());
      return [value, if (zone.isNotEmpty) zone].join(' ');
    }

    Widget row(String label, Widget value) => Padding(
      padding: const EdgeInsets.only(top: 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label,
            style: const TextStyle(color: Tokens.muted, fontSize: 12),
          ),
          const SizedBox(height: 4),
          value,
        ],
      ),
    );
    final start = when(
      item.startLocal,
      item.startAt,
      item.startDate,
      item.startTimezone,
    );
    final end = when(item.endLocal, item.endAt, item.endDate, item.endTimezone);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (start.isNotEmpty) row(trip ? 'Departs' : 'Starts', Text(start)),
        if (end.isNotEmpty) row(trip ? 'Arrives' : 'Ends', Text(end)),
        if (item.dueAt != null || item.dueDate != null)
          row('Due', Text(when('', item.dueAt, item.dueDate, ''))),
        if (item.durationMinutes > 0)
          row('Duration', Text(durationText(item.durationMinutes))),
        if (item.kind.isNotEmpty) row('Type', Text(kindLabel(item.kind))),
        if (item.location.isNotEmpty)
          row(
            'Address',
            TextButton(
              style: TextButton.styleFrom(
                padding: EdgeInsets.zero,
                alignment: Alignment.centerLeft,
                minimumSize: const Size(44, 44),
              ),
              onPressed: launcher == null
                  ? null
                  : () async {
                      var opened = false;
                      try {
                        opened = await launcher!.open(mapsUri(item.location));
                      } catch (_) {}
                      if (!opened && context.mounted) {
                        ScaffoldMessenger.of(context).showSnackBar(
                          const SnackBar(
                            content: Text('Could not open this link.'),
                          ),
                        );
                      }
                    },
              child: Text(
                item.location,
                style: const TextStyle(decoration: TextDecoration.underline),
              ),
            ),
          ),
        if (item.notes.isNotEmpty)
          row(
            'Notes',
            Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                for (final line in noteLines(item.notes))
                  Padding(
                    padding: const EdgeInsets.only(bottom: 6),
                    child: LinkedText(text: line, launcher: launcher),
                  ),
              ],
            ),
          ),
        if (item.reminderOffsets.isNotEmpty)
          row(
            'Reminders',
            Text('${item.reminderOffsets.map(offsetLabel).join(', ')} before'),
          ),
      ],
    );
  }
}

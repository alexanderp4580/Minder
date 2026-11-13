import 'package:flutter/material.dart';

import '../domain/item.dart';
import '../domain/booking_details.dart';
import '../domain/link_launcher.dart';
import '../domain/links.dart';
import 'content_action.dart';
import 'format.dart';
import 'linked_text.dart';
import 'tokens.dart';

class CalendarItemDetails extends StatelessWidget {
  const CalendarItemDetails({super.key, required this.item, this.launcher});
  final Item item;
  final LinkLauncher? launcher;

  Widget? _when(String local, DateTime? instant, DateTime? date, String zone) {
    if (date != null) return Text(date.toIso8601String().split('T').first);
    if (local.isEmpty && instant == null) return null;
    final stamp = local.isNotEmpty
        ? fieldValue(local)
        : dateTime(instant!.toLocal());
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          stamp.replaceFirstMapped(
            RegExp(r' (\d{2}:\d{2})$'),
            (m) => ' · ${m[1]}',
          ),
        ),
        if (local.isNotEmpty && zone.isNotEmpty)
          Text(zone, style: const TextStyle(color: Tokens.muted)),
      ],
    );
  }

  Widget _row(String label, Widget value) => Padding(
    padding: const EdgeInsets.only(top: 16),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label, style: const TextStyle(color: Tokens.muted, fontSize: 12)),
        const SizedBox(height: 4),
        value,
      ],
    ),
  );

  Future<void> _open(BuildContext context, Uri uri) async {
    var opened = false;
    try {
      opened = await launcher!.open(uri);
    } catch (_) {}
    if (!opened && context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Could not open this link.')),
      );
    }
  }

  Widget _contact(
    BuildContext context,
    String label,
    String value,
    LinkKind kind,
  ) {
    final target = detailTarget(value, kind);
    if (target == null) return _row(label, Text(value));
    return _row(
      label,
      ContentAction(
        onPressed: launcher == null ? null : () => _open(context, target),
        child: Text(
          value,
          style: const TextStyle(
            fontSize: 15,
            fontWeight: FontWeight.w500,
            color: Tokens.text,
            decoration: TextDecoration.underline,
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final details = BookingDetails.fromNotes(
      phone: item.phone,
      email: item.email,
      website: item.website,
      reference: item.bookingReference,
      notes: item.notes,
    );
    final trip =
        item.startTimezone.isNotEmpty &&
        item.endTimezone.isNotEmpty &&
        item.startTimezone != item.endTimezone;
    final start = _when(
      item.startLocal,
      item.startAt,
      item.startDate,
      item.startTimezone,
    );
    final end = _when(
      item.endLocal,
      item.endAt,
      item.endDate,
      item.endTimezone,
    );
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (start != null) _row(trip ? 'Departs' : 'Starts', start),
        if (end != null) _row(trip ? 'Arrives' : 'Ends', end),
        if (item.dueAt != null || item.dueDate != null)
          _row('Due', _when('', item.dueAt, item.dueDate, '')!),
        if (item.durationMinutes > 0)
          _row('Duration', Text(durationText(item.durationMinutes))),
        if (item.kind.isNotEmpty) _row('Type', Text(kindLabel(item.kind))),
        if (item.location.isNotEmpty)
          _row(
            'Address',
            ContentAction(
              onPressed: launcher == null
                  ? null
                  : () => _open(context, mapsUri(item.location)),
              child: Padding(
                padding: const EdgeInsets.symmetric(vertical: 8),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            item.location,
                            style: const TextStyle(
                              fontSize: 15,
                              fontWeight: FontWeight.w500,
                              decoration: TextDecoration.underline,
                              decorationThickness: 1,
                            ),
                          ),
                          const SizedBox(height: 4),
                          const Text(
                            'Opens Google Maps',
                            style: TextStyle(
                              fontSize: 12,
                              fontWeight: FontWeight.w500,
                              color: Tokens.muted,
                            ),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(width: 16),
                    const Padding(
                      padding: EdgeInsets.only(top: 3),
                      child: Icon(Icons.open_in_new, size: 16),
                    ),
                  ],
                ),
              ),
            ),
          ),
        if (details.phone.isNotEmpty)
          _contact(context, 'Phone', details.phone, LinkKind.phone),
        if (details.email.isNotEmpty)
          _contact(context, 'Email', details.email, LinkKind.email),
        if (details.website.isNotEmpty)
          _contact(context, 'Website', details.website, LinkKind.url),
        if (details.reference.isNotEmpty)
          _row('Booking reference', SelectableText(details.reference)),
        if (details.notes.isNotEmpty)
          _row(
            'Notes',
            Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                for (final line in noteLines(details.notes))
                  Padding(
                    padding: const EdgeInsets.only(bottom: 8),
                    child: Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        const Text('•'),
                        const SizedBox(width: 8),
                        Expanded(
                          child: LinkedText(text: line, launcher: launcher),
                        ),
                      ],
                    ),
                  ),
              ],
            ),
          ),
        if (item.reminderOffsets.isNotEmpty)
          _row(
            'Reminders',
            Text('${item.reminderOffsets.map(offsetLabel).join(', ')} before'),
          ),
      ],
    );
  }
}

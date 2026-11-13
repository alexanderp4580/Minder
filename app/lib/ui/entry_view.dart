import 'package:flutter/material.dart';

import '../application/chat_controller.dart';
import '../domain/chat_repository.dart';
import '../domain/entry.dart';
import 'format.dart';
import 'tokens.dart';
import 'capture_view.dart';
import 'question_view.dart';
import 'reminder_chips.dart';

const _meta = TextStyle(fontSize: 12, color: Tokens.muted, letterSpacing: 1.2);

/// One chat entry: a sender line, then the message in its box.
class EntryView extends StatelessWidget {
  const EntryView({super.key, required this.entry, required this.controller});
  final Entry entry;
  final ChatController controller;

  @override
  Widget build(BuildContext context) {
    final mine = entry.kind == EntryKind.user;
    final boxed = mine || entry.kind == EntryKind.preview;
    final time =
        clock(entry.createdAt.toLocal()) +
        (entry.costEur > 0 ? ' · ${cost(entry.costEur)}' : '');
    return Container(
      key: ValueKey(entry.id),
      margin: const EdgeInsets.only(top: 16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Align(
            alignment: mine ? Alignment.centerRight : Alignment.centerLeft,
            child: Text('${mine ? 'YOU' : 'MINDER'} · $time', style: _meta),
          ),
          const SizedBox(height: 6),
          Container(
            margin: EdgeInsets.only(left: mine ? 44 : 0),
            padding: !boxed
                ? EdgeInsets.zero
                : mine
                ? const EdgeInsets.symmetric(horizontal: 12, vertical: 10)
                : const EdgeInsets.all(12),
            decoration: boxed
                ? BoxDecoration(
                    color: mine ? Tokens.panel : Tokens.voidColor,
                    border: Border.all(color: Tokens.edge),
                  )
                : null,
            child: _body(),
          ),
        ],
      ),
    );
  }

  Widget _body() {
    final change = entry.change;
    final reminder = entry.reminder;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (entry.capture != null && entry.kind == EntryKind.question)
          QuestionView(capture: entry.capture!, chat: controller)
        else if (entry.capture != null)
          CaptureView(capture: entry.capture!, chat: controller)
        else if (change != null && entry.kind == EntryKind.preview)
          _Preview(change: change, controller: controller)
        else
          Text(
            entry.text,
            style: TextStyle(
              color: entry.kind == EntryKind.error ? Tokens.alert : Tokens.text,
            ),
          ),
        if (reminder != null)
          reminder.active
              ? _ReminderActions(reminder: reminder, controller: controller)
              : Text(_handled(reminder), style: _muted),
        if (entry.kind == EntryKind.result && change?.undoId != null)
          _Action(
            child: OutlinedButton(
              onPressed: controller.acting
                  ? null
                  : () => controller.undo(change!.undoId!),
              child: const Text('Undo'),
            ),
          ),
        if (entry.kind == EntryKind.error)
          _Action(
            child: OutlinedButton(
              onPressed: controller.waiting
                  ? null
                  : () => controller.retry(text: controller.retryText(entry)),
              child: const Text('Retry'),
            ),
          ),
      ],
    );
  }

  String _handled(ReminderInfo r) => switch (r.state) {
    'superseded' => 'See the latest reminder.',
    'cancelled' => 'Reminder cancelled.',
    _ => 'Reminder handled.',
  };
}

const _muted = TextStyle(color: Tokens.muted);

/// A button row under a message, aligned left with 12 px above it.
class _Action extends StatelessWidget {
  const _Action({required this.child});
  final Widget child;
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(top: 12),
    child: Align(alignment: Alignment.centerLeft, child: child),
  );
}

class _Preview extends StatelessWidget {
  const _Preview({required this.change, required this.controller});
  final Change change;
  final ChatController controller;

  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      Text(
        'PREVIEW · ${actionLabel(change.action)}',
        style: const TextStyle(
          fontSize: 12,
          color: Tokens.muted,
          letterSpacing: 1,
        ),
      ),
      const SizedBox(height: 8),
      Text(
        change.itemTitle,
        style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
      ),
      const SizedBox(height: 8),
      for (final row in previewRows(change))
        Padding(
          padding: const EdgeInsets.only(bottom: 5),
          child: row.bullets.isNotEmpty
              ? Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text('${row.label}:', style: _muted),
                    for (final line in row.bullets)
                      Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          const Text('•'),
                          const SizedBox(width: 8),
                          Expanded(child: Text(line)),
                        ],
                      ),
                  ],
                )
              : Wrap(
                  spacing: 6,
                  children: [
                    Text('${row.label}:', style: _muted),
                    if (row.oldValue.isNotEmpty)
                      Text(
                        row.oldValue,
                        style: const TextStyle(
                          color: Tokens.muted,
                          decoration: TextDecoration.lineThrough,
                        ),
                      ),
                    Text(row.newValue),
                  ],
                ),
        ),
      if (change.reminderChoices.isNotEmpty) ...[
        const SizedBox(height: 4),
        const Text('Remind before', style: _muted),
        const SizedBox(height: 6),
        ReminderChips(
          choices: change.reminderChoices,
          selected: change.reminderOffsets,
          enabled: change.state == ChangeState.pending && !controller.acting,
          onChanged: (offsets) => controller.setReminders(change.id, offsets),
        ),
      ],
      if (change.state == ChangeState.pending)
        _decision()
      else
        Text(switch (change.state) {
          ChangeState.confirmed => 'Confirmed',
          ChangeState.discarded => 'Discarded',
          _ => 'Item changed. See the new preview.',
        }, style: _muted),
    ],
  );

  /// Discard, then Confirm last: the primary button sits at the row's bottom right.
  Widget _decision() => Padding(
    padding: const EdgeInsets.only(top: 12),
    child: Row(
      children: [
        Expanded(
          flex: 10,
          child: OutlinedButton(
            onPressed: controller.acting
                ? null
                : () => controller.discard(change.id),
            child: const Text('Discard'),
          ),
        ),
        const SizedBox(width: Tokens.gap),
        Expanded(
          flex: 13,
          child: FilledButton(
            onPressed: controller.acting
                ? null
                : () => controller.confirm(change.id),
            child: const Text('Confirm'),
          ),
        ),
      ],
    ),
  );
}

class _ReminderActions extends StatelessWidget {
  const _ReminderActions({required this.reminder, required this.controller});
  final ReminderInfo reminder;
  final ChatController controller;

  static const _snoozes = [
    (ReminderAction.snooze15, '15 minutes'),
    (ReminderAction.snooze60, '1 hour'),
    (ReminderAction.tonight, 'Tonight'),
    (ReminderAction.tomorrow, 'Tomorrow'),
  ];

  /// Snooze, Open, then Done last: the primary button sits at the row's bottom right.
  @override
  Widget build(BuildContext context) => Padding(
    padding: const EdgeInsets.only(top: 12),
    child: Row(
      children: [
        Expanded(
          flex: 10,
          child: MenuAnchor(
            menuChildren: [
              for (final (action, label) in _snoozes)
                MenuItemButton(
                  onPressed: () => controller.remind(reminder.id, action),
                  child: Text(label),
                ),
            ],
            builder: (context, menu, _) => OutlinedButton(
              onPressed: controller.acting
                  ? null
                  : () => menu.isOpen ? menu.close() : menu.open(),
              child: const Text('Snooze'),
            ),
          ),
        ),
        const SizedBox(width: Tokens.gap),
        Expanded(
          flex: 10,
          child: OutlinedButton(
            onPressed: controller.acting
                ? null
                : () => controller.remind(
                    reminder.id,
                    ReminderAction.acknowledge,
                  ),
            child: const Text('Open'),
          ),
        ),
        const SizedBox(width: Tokens.gap),
        Expanded(
          flex: 11,
          child: FilledButton(
            onPressed: controller.acting
                ? null
                : () => controller.remind(reminder.id, ReminderAction.done),
            child: const Text('Done'),
          ),
        ),
      ],
    ),
  );
}

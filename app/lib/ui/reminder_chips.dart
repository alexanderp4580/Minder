import 'package:flutter/material.dart';

import 'format.dart';
import 'tokens.dart';

/// Reminder offsets as toggle chips, with Clear to switch every one off.
class ReminderChips extends StatelessWidget {
  const ReminderChips({
    super.key,
    required this.choices,
    required this.selected,
    required this.onChanged,
    this.enabled = true,
  });

  final List<int> choices, selected;
  final ValueChanged<List<int>> onChanged;
  final bool enabled;

  void _toggle(int minutes) {
    final next = {...selected};
    next.contains(minutes) ? next.remove(minutes) : next.add(minutes);
    onChanged(next.toList()..sort((a, b) => b.compareTo(a)));
  }

  @override
  Widget build(BuildContext context) => Wrap(
    spacing: Tokens.gap,
    runSpacing: Tokens.gap,
    children: [
      for (final minutes in choices)
        _Chip(
          label: offsetLabel(minutes),
          on: selected.contains(minutes),
          onTap: enabled ? () => _toggle(minutes) : null,
        ),
      if (selected.isNotEmpty)
        _Chip(
          label: 'Clear',
          on: false,
          quiet: true,
          onTap: enabled ? () => onChanged(const []) : null,
        ),
    ],
  );
}

class _Chip extends StatelessWidget {
  const _Chip({
    required this.label,
    required this.on,
    required this.onTap,
    this.quiet = false,
  });
  final String label;
  final bool on, quiet;
  final VoidCallback? onTap;

  @override
  Widget build(BuildContext context) => Semantics(
    button: true,
    selected: on,
    child: InkWell(
      onTap: onTap,
      child: Container(
        constraints: const BoxConstraints(minHeight: Tokens.minTarget),
        padding: const EdgeInsets.symmetric(horizontal: 12),
        alignment: Alignment.center,
        decoration: BoxDecoration(
          color: on ? Tokens.panel : null,
          border: Border.all(
            color: on ? Tokens.signal : Tokens.line,
            width: on ? 1.5 : 1,
          ),
        ),
        child: Text(
          label,
          style: TextStyle(
            color: onTap == null || quiet ? Tokens.muted : Tokens.text,
          ),
        ),
      ),
    ),
  );
}

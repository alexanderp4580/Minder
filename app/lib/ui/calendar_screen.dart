import 'package:flutter/material.dart';

import '../application/calendar_controller.dart';
import '../domain/item.dart';
import 'format.dart';
import 'tokens.dart';

const _months = [
  'JANUARY',
  'FEBRUARY',
  'MARCH',
  'APRIL',
  'MAY',
  'JUNE',
  'JULY',
  'AUGUST',
  'SEPTEMBER',
  'OCTOBER',
  'NOVEMBER',
  'DECEMBER',
];
const _weekdays = ['MON', 'TUE', 'WED', 'THU', 'FRI', 'SAT', 'SUN'];

String _duration(int minutes) {
  if (minutes < 60) return '$minutes min';
  final h = minutes ~/ 60, m = minutes % 60;
  return m == 0 ? '$h h' : '$h h $m min';
}

String two(int n) => n.toString().padLeft(2, '0');

class CalendarScreen extends StatefulWidget {
  const CalendarScreen({super.key, required this.controller});
  final CalendarController controller;
  @override
  State<CalendarScreen> createState() => _CalendarScreenState();
}

class _CalendarScreenState extends State<CalendarScreen> {
  CalendarController get c => widget.controller;
  @override
  void initState() {
    super.initState();
    c.addListener(_changed);
  }

  void _changed() {
    if (mounted) setState(() {});
  }

  @override
  void dispose() {
    c.removeListener(_changed);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Scaffold(
    body: SafeArea(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            padding: const EdgeInsets.fromLTRB(
              Tokens.gutter,
              18,
              Tokens.gutter,
              0,
            ),
            child: Row(
              children: [
                const Expanded(
                  child: Text(
                    'CALENDAR',
                    style: TextStyle(
                      fontFamily: 'Michroma',
                      fontSize: 22,
                      letterSpacing: 4.4,
                    ),
                  ),
                ),
                IconButton(tooltip: 'Chat', onPressed: () => Navigator.of(context).maybePop(), icon: const Icon(Icons.chat_bubble_outline)),
              ],
            ),
          ),
          const SizedBox(height: 16),
          _monthBar(),
          const SizedBox(height: 8),
          _weekdayRow(),
          for (var w = 0; w < 6; w++) _week(w),
          const Divider(height: 1),
          Expanded(child: _agenda()),
        ],
      ),
    ),
  );

  Widget _monthBar() => Padding(
    padding: const EdgeInsets.symmetric(horizontal: Tokens.gutter),
    child: Row(
      children: [
        IconButton(tooltip: 'Previous month', onPressed: c.previous, icon: const Icon(Icons.chevron_left)),
        Expanded(
          child: Text(
            '${_months[c.month.month - 1]} ${c.month.year}',
            textAlign: TextAlign.center,
            style: const TextStyle(
              fontSize: 14,
              letterSpacing: 1.4,
              fontWeight: FontWeight.w700,
            ),
          ),
        ),
        IconButton(tooltip: 'Next month', onPressed: c.next, icon: const Icon(Icons.chevron_right)),
        const SizedBox(width: Tokens.gap),
        OutlinedButton(onPressed: c.goToday, child: Text('Today')),
      ],
    ),
  );

  Widget _weekdayRow() => Padding(
    padding: const EdgeInsets.symmetric(horizontal: Tokens.gutter - 4),
    child: Row(
      children: [
        for (final d in _weekdays)
          Expanded(
            child: Center(
              child: Text(
                d,
                style: const TextStyle(
                  fontSize: 12,
                  color: Tokens.muted,
                  letterSpacing: 1,
                ),
              ),
            ),
          ),
      ],
    ),
  );

  Widget _week(int w) => Padding(
    padding: const EdgeInsets.symmetric(horizontal: Tokens.gutter - 4),
    child: Row(
      children: [
        for (var d = 0; d < 7; d++)
          Expanded(
            child: _cell(
              DateTime(
                c.gridStart.year,
                c.gridStart.month,
                c.gridStart.day + w * 7 + d,
              ),
            ),
          ),
      ],
    ),
  );

  Widget _cell(DateTime day) {
    final inMonth = day.month == c.month.month;
    final selected = day == c.selected;
    final isToday = day == c.today;
    final count = c.countOn(day);
    return Semantics(
      button: true,
      selected: selected,
      label: '${_months[day.month - 1].substring(0, 3)} ${day.day}, '
          '$count ${count == 1 ? 'item' : 'items'}',
      excludeSemantics: true,
      child: InkWell(
        key: ValueKey('day-${day.year}-${two(day.month)}-${two(day.day)}'),
        overlayColor: const WidgetStatePropertyAll(Colors.transparent),
        onTap: () => c.select(day),
        child: Container(
          height: 56,
          decoration: BoxDecoration(
            border: Border.all(
              color: selected ? Tokens.signal : Colors.transparent,
              width: 2,
            ),
          ),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Text(
                '${day.day}',
                style: TextStyle(
                  fontSize: 15,
                  fontWeight: isToday ? FontWeight.w800 : FontWeight.w500,
                  color: inMonth ? Tokens.text : Tokens.dim,
                  decoration: isToday ? TextDecoration.underline : null,
                  decorationColor: Tokens.text,
                ),
              ),
              const SizedBox(height: 3),
              Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  for (var i = 0; i < count.clamp(0, 3); i++)
                    Container(
                      width: 5,
                      height: 5,
                      margin: const EdgeInsets.symmetric(horizontal: 1.5),
                      color: inMonth ? Tokens.text : Tokens.dim,
                    ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _agenda() {
    final day = c.selected;
    final items = c.selectedItems;
    return ListView(
      padding: const EdgeInsets.fromLTRB(Tokens.gutter, 16, Tokens.gutter, 24),
      children: [
        Text(
          '${_weekdays[day.weekday - 1]} ${day.day} '
          '${_months[day.month - 1].substring(0, 3)}',
          style: const TextStyle(
            fontSize: 12,
            color: Tokens.muted,
            letterSpacing: 1.2,
          ),
        ),
        const SizedBox(height: 12),
        if (c.error != null) ...[
          Text(c.error!, style: const TextStyle(color: Tokens.alert)),
          const SizedBox(height: 8),
          Align(
            alignment: Alignment.centerLeft,
            child: OutlinedButton(onPressed: c.loading ? null : c.load, child: Text('Retry')),
          ),
          const SizedBox(height: 12),
        ],
        if (items.isEmpty && c.error == null && !c.loading)
          const Text('Nothing planned.', style: TextStyle(color: Tokens.muted)),
        for (final item in items) _entry(item),
      ],
    );
  }

  Widget _entry(Item item) {
    final at = item.placedAt!;
    final time = item.startAt != null ? clock(at) : 'Due ${clock(at)}';
    final chips = [
      if (item.durationMinutes > 0) _duration(item.durationMinutes),
      if (item.priority > 0) 'P${item.priority}',
      if (item.location.isNotEmpty) item.location,
      if (item.list.isNotEmpty) item.list,
      if (item.repeat.isNotEmpty) item.repeat,
    ];
    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Tokens.panel,
        border: Border.all(color: Tokens.edge),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            time,
            style: const TextStyle(
              fontSize: 12,
              color: Tokens.muted,
              letterSpacing: 1,
            ),
          ),
          const SizedBox(height: 4),
          Text(
            item.title,
            style: TextStyle(
              fontSize: 16,
              fontWeight: FontWeight.w700,
              color: item.done ? Tokens.dim : Tokens.text,
              decoration: item.done ? TextDecoration.lineThrough : null,
            ),
          ),
          if (chips.isNotEmpty) ...[
            const SizedBox(height: 8),
            Wrap(
              spacing: 8,
              runSpacing: 6,
              children: [
                for (final chip in chips)
                  Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 8,
                      vertical: 3,
                    ),
                    decoration: BoxDecoration(
                      border: Border.all(color: Tokens.line),
                    ),
                    child: Text(
                      chip,
                      style: const TextStyle(
                        fontSize: 12,
                        color: Tokens.muted,
                      ),
                    ),
                  ),
              ],
            ),
          ],
        ],
      ),
    );
  }
}

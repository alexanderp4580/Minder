import '../domain/capture_repository.dart';
import '../domain/link_launcher.dart';
import 'calendar_item_details.dart';
import 'content_action.dart';
import 'item_delete_dialog.dart';
import 'item_sources.dart';

import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../application/calendar_controller.dart';
import '../domain/item.dart';
import 'bottom_nav.dart';
import 'format.dart';
import 'tokens.dart';
import 'screen_header.dart';
import 'orbital_background.dart';

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
const _dayNames = [
  'Monday',
  'Tuesday',
  'Wednesday',
  'Thursday',
  'Friday',
  'Saturday',
  'Sunday',
];
String _monthName(int month) {
  final upper = _months[month - 1];
  return upper[0] + upper.substring(1).toLowerCase();
}

String _duration(int minutes) {
  if (minutes < 60) return '$minutes min';
  final h = minutes ~/ 60, m = minutes % 60;
  return m == 0 ? '$h h' : '$h h $m min';
}

String _two(int n) => n.toString().padLeft(2, '0');

class CalendarScreen extends StatefulWidget {
  const CalendarScreen({
    super.key,
    required this.controller,
    this.sourceRepository,
    this.linkLauncher,
  });
  final CaptureRepository? sourceRepository;
  final LinkLauncher? linkLauncher;
  final CalendarController controller;
  @override
  State<CalendarScreen> createState() => _CalendarScreenState();
}

class _CalendarScreenState extends State<CalendarScreen> {
  CalendarController get c => widget.controller;
  final _expanded = <String>{};
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
    bottomNavigationBar: BottomNav(
      current: NavDestination.calendar,
      onSelect: (_) => Navigator.of(context).maybePop(),
    ),
    body: OrbitalBackground(
      scene: OrbitScene.calendar,
      child: SafeArea(
        child: LayoutBuilder(
          builder: (context, constraints) {
            final width = math.min(
              constraints.maxWidth,
              CalendarTokens.maxWidth,
            );
            final gutter = math.min(
              CalendarTokens.gutter,
              math.max(0.0, (width - 7 * Tokens.minTarget) / 2),
            );
            return SingleChildScrollView(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const ScreenHeader(title: 'CALENDAR'),
                  Center(
                    child: SizedBox(
                      width: width,
                      child: Padding(
                        padding: EdgeInsets.fromLTRB(
                          gutter,
                          Tokens.relatedGap,
                          gutter,
                          Tokens.blockGap,
                        ),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.stretch,
                          children: [
                            _monthBar(),
                            const SizedBox(height: Tokens.sectionGap),
                            _weekdayRow(),
                            const SizedBox(height: 12),
                            for (var w = 0; w < 6; w++) ...[
                              if (w > 0) const SizedBox(height: Tokens.gap),
                              _week(w),
                            ],
                            const SizedBox(height: Tokens.blockGap),
                            _agenda(),
                          ],
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            );
          },
        ),
      ),
    ),
  );

  Widget _monthBar() => ConstrainedBox(
    constraints: const BoxConstraints(minHeight: 54),
    child: Row(
      children: [
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                _monthName(c.month.month),
                style: const TextStyle(
                  fontSize: 24,
                  fontWeight: FontWeight.w600,
                  height: 1.15,
                ),
              ),
              const SizedBox(height: 6),
              Text(
                '${c.month.year}',
                style: const TextStyle(
                  fontSize: 14,
                  color: Tokens.muted,
                  letterSpacing: 1.12,
                  height: 1.4,
                ),
              ),
            ],
          ),
        ),
        const SizedBox(width: Tokens.sectionGap),
        IconButton(
          tooltip: 'Previous month',
          onPressed: c.previous,
          icon: const Icon(Icons.chevron_left),
        ),
        const SizedBox(width: Tokens.gap),
        IconButton(
          tooltip: 'Next month',
          onPressed: c.next,
          icon: const Icon(Icons.chevron_right),
        ),
      ],
    ),
  );

  Widget _weekdayRow() => Row(
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
                height: 1.5,
              ),
            ),
          ),
        ),
    ],
  );

  Widget _week(int w) => Row(
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
  );

  Widget _cell(DateTime day) {
    final inMonth = day.month == c.month.month;
    final selected = day == c.selected;
    final isToday = day == c.today;
    final count = c.countOn(day);
    final faceHeight = math.max(
      CalendarTokens.selectionSize,
      MediaQuery.textScalerOf(context).scale(16) * 1.2 + 16,
    );
    return Semantics(
      button: true,
      selected: selected,
      label:
          '${_months[day.month - 1].substring(0, 3)} ${day.day}, $count ${count == 1 ? 'item' : 'items'}',
      excludeSemantics: true,
      child: SizedBox(
        height: math.max(CalendarTokens.dayHeight, faceHeight + 12),
        child: InkWell(
          key: ValueKey('day-${day.year}-${_two(day.month)}-${_two(day.day)}'),
          overlayColor: const WidgetStatePropertyAll(Colors.transparent),
          onTap: () => c.select(day),
          child: Center(
            child: Container(
              key: selected ? const ValueKey('calendar-selection') : null,
              width: CalendarTokens.selectionSize,
              height: faceHeight,
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
                      fontSize: 16,
                      height: 1.2,
                      fontWeight: isToday ? FontWeight.w700 : FontWeight.w500,
                      color: inMonth ? Tokens.text : Tokens.dim,
                      decoration: isToday ? TextDecoration.underline : null,
                      decorationColor: Tokens.text,
                    ),
                  ),
                  const SizedBox(height: 6),
                  SizedBox(
                    height: 4,
                    child: Row(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        for (var i = 0; i < count.clamp(0, 3); i++)
                          Container(
                            width: 4,
                            height: 4,
                            margin: const EdgeInsets.symmetric(horizontal: 1.5),
                            color: inMonth ? Tokens.text : Tokens.dim,
                          ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _agenda() {
    final day = c.selected;
    final items = c.selectedItems;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          children: [
            Expanded(
              child: Text(
                '${_dayNames[day.weekday - 1]}, ${day.day} ${_monthName(day.month).substring(0, 3)}',
                style: const TextStyle(
                  fontSize: 16,
                  fontWeight: FontWeight.w600,
                  height: 1.5,
                ),
              ),
            ),
            const SizedBox(width: Tokens.relatedGap),
            OutlinedButton(onPressed: c.goToday, child: const Text('Today')),
          ],
        ),
        const SizedBox(height: Tokens.relatedGap),
        if (c.error != null) ...[
          Text(c.error!, style: const TextStyle(color: Tokens.alert)),
          const SizedBox(height: 8),
          Align(
            alignment: Alignment.centerLeft,
            child: OutlinedButton(
              onPressed: c.loading ? null : c.load,
              child: Text('Retry'),
            ),
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
    final time = item.startDate != null
        ? 'All day'
        : item.dueDate != null
        ? 'Due today'
        : item.startAt != null
        ? clock(at)
        : 'Due ${clock(at)}';
    // Departs and Arrives suit a trip across zones; an appointment starts and ends.
    final trip =
        item.startTimezone.isNotEmpty &&
        item.endTimezone.isNotEmpty &&
        item.startTimezone != item.endTimezone;
    final expanded = _expanded.contains(item.id);
    final hasDetails =
        item.phone.isNotEmpty ||
        item.email.isNotEmpty ||
        item.website.isNotEmpty ||
        item.bookingReference.isNotEmpty ||
        item.notes.isNotEmpty ||
        item.location.isNotEmpty ||
        item.reminderOffsets.isNotEmpty ||
        item.endLocal.isNotEmpty ||
        item.kind.isNotEmpty ||
        item.durationMinutes > 0 ||
        item.startAt != null ||
        item.startDate != null;
    final chips = [
      if (!expanded && trip && item.startLocal.isNotEmpty)
        'Departs ${item.startLocal.replaceFirst('T', ' ')} ${item.startTimezone}',
      if (!expanded && trip && item.endLocal.isNotEmpty)
        'Arrives ${item.endLocal.replaceFirst('T', ' ')} ${item.endTimezone}',
      if (item.endDate != null)
        'Checkout ${item.endDate!.year}-${item.endDate!.month.toString().padLeft(2, '0')}-${item.endDate!.day.toString().padLeft(2, '0')}',
      if (!expanded && item.durationMinutes > 0)
        _duration(item.durationMinutes),
      if (item.priority > 0) 'P${item.priority}',
      if (!expanded && item.location.isNotEmpty) item.location,
      if (item.list.isNotEmpty) item.list,
      if (item.repeat.isNotEmpty) item.repeat,
    ];
    void toggle() => setState(() {
      if (expanded) {
        _expanded.remove(item.id);
      } else {
        _expanded.add(item.id);
      }
    });
    return GestureDetector(
      onTap: hasDetails ? toggle : null,
      child: Container(
        margin: const EdgeInsets.only(bottom: Tokens.relatedGap),
        padding: const EdgeInsets.all(CalendarTokens.cardPadding),
        decoration: BoxDecoration(
          color: Tokens.panel,
          border: Border.all(color: Tokens.line),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Text(
                    time,
                    style: const TextStyle(
                      fontSize: 12,
                      color: Tokens.muted,
                      letterSpacing: 1,
                    ),
                  ),
                ),
                IconButton(
                  tooltip: 'Delete',
                  onPressed: () => showDialog<void>(
                    context: context,
                    barrierDismissible: false,
                    builder: (_) => ItemDeleteDialog(controller: c, item: item),
                  ),
                  constraints: const BoxConstraints.tightFor(
                    width: Tokens.minTarget,
                    height: Tokens.minTarget,
                  ),
                  icon: const Icon(Icons.delete_outline, color: Tokens.muted),
                ),
              ],
            ),
            const SizedBox(height: Tokens.gap),
            Text(
              item.title,
              style: TextStyle(
                fontSize: 18,
                fontWeight: FontWeight.w600,
                color: item.done ? Tokens.dim : Tokens.text,
                decoration: item.done ? TextDecoration.lineThrough : null,
              ),
            ),
            if (item.sourceIds.isNotEmpty && widget.sourceRepository != null)
              ContentAction(
                onPressed: () => showDialog<void>(
                  context: context,
                  builder: (_) => ItemSources(
                    repository: widget.sourceRepository!,
                    sourceIds: item.sourceIds,
                  ),
                ),
                child: const Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Flexible(child: Text('Email and attachments')),
                    SizedBox(width: 8),
                    Icon(Icons.chevron_right, size: 16),
                  ],
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
            if (hasDetails) ...[
              Container(
                width: double.infinity,
                margin: const EdgeInsets.only(top: 16),
                decoration: const BoxDecoration(
                  border: Border(top: BorderSide(color: Tokens.line)),
                ),
                child: Semantics(
                  expanded: expanded,
                  child: ContentAction(
                    onPressed: toggle,
                    child: Padding(
                      padding: const EdgeInsets.only(top: 12),
                      child: Row(
                        children: [
                          Expanded(
                            child: Text(expanded ? 'Hide details' : 'Details'),
                          ),
                          const SizedBox(width: 12),
                          Icon(
                            expanded ? Icons.expand_less : Icons.expand_more,
                            size: 16,
                          ),
                        ],
                      ),
                    ),
                  ),
                ),
              ),
              if (expanded)
                CalendarItemDetails(item: item, launcher: widget.linkLauncher),
            ],
          ],
        ),
      ),
    );
  }
}

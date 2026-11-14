import 'dart:math' as math;

import 'package:flutter/material.dart';

import '../domain/item.dart';
import 'tokens.dart';
import 'item_colours.dart';

class _Span {
  _Span(this.item, this.first, this.last, this.before, this.after);
  final Item item;
  final int first, last;
  final bool before, after;
  int lane = 0;
}

/// A full week owns event lanes so a bar can cross day columns without gaps.
class CalendarWeek extends StatelessWidget {
  const CalendarWeek({
    super.key,
    required this.start,
    required this.index,
    required this.items,
    required this.month,
    required this.selected,
    required this.today,
    required this.onSelect,
  });
  final DateTime start, month, selected, today;
  final int index;
  final List<Item> items;
  final ValueChanged<DateTime> onSelect;
  DateTime _day(int d) => DateTime(start.year, start.month, start.day + d);
  @override
  Widget build(BuildContext context) {
    final spans = <_Span>[];
    for (final item in items) {
      final days = [
        for (var d = 0; d < 7; d++)
          if (item.occursOn(_day(d))) d,
      ];
      if (days.isNotEmpty) {
        spans.add(
          _Span(
            item,
            days.first,
            days.last,
            item.occursOn(_day(-1)),
            item.occursOn(_day(7)),
          ),
        );
      }
    }
    spans.sort((a, b) {
      final byLength = (b.last - b.first).compareTo(a.last - a.first);
      return byLength != 0 ? byLength : a.item.id.compareTo(b.item.id);
    });
    final lanes = <List<_Span>>[];
    for (final s in spans) {
      var lane = 0;
      while (lane < lanes.length &&
          lanes[lane].any((t) => s.first <= t.last && s.last >= t.first)) {
        lane++;
      }
      if (lane == lanes.length) lanes.add([]);
      s.lane = lane;
      lanes[lane].add(s);
    }
    final dateHeight = math.max(
      44.0,
      MediaQuery.textScalerOf(context).scale(16) * 1.2 + 16,
    );
    final dayHeight = math.max(56.0, dateHeight + 12);
    final barHeight = math.max(
      20.0,
      MediaQuery.textScalerOf(context).scale(12) * 1.4 + 4,
    );
    final laneHeight = barHeight + 4;
    final height = dayHeight + lanes.length * laneHeight;
    return LayoutBuilder(
      builder: (context, constraints) {
        final width = constraints.maxWidth / 7;
        return SizedBox(
          height: height,
          child: Stack(
            children: [
              Row(
                children: [
                  for (var d = 0; d < 7; d++)
                    Expanded(
                      child: _cell(context, _day(d), height, dateHeight),
                    ),
                ],
              ),
              for (final s in spans)
                Positioned(
                  left: s.first * width + 1,
                  top: dayHeight + s.lane * laneHeight,
                  width: (s.last - s.first + 1) * width - 2,
                  height: barHeight,
                  child: IgnorePointer(
                    child: ExcludeSemantics(
                      child: ClipPath(
                        clipper: _Continuation(s.before, s.after),
                        child: Container(
                          key: ValueKey('eventbar-${s.item.id}-$index'),
                          alignment: Alignment.centerLeft,
                          padding: EdgeInsets.only(
                            left: s.before ? 6 : 4,
                            right: 4,
                          ),
                          decoration: BoxDecoration(
                            color: Color.alphaBlend(
                              itemColour(s.item.color).withValues(alpha: .28),
                              Tokens.panel,
                            ),
                            border: Border(
                              left: BorderSide(
                                width: 2,
                                color: itemColour(s.item.color),
                              ),
                            ),
                          ),
                          child: Text(
                            s.item.title,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: TextStyle(
                              fontSize: 12,
                              fontWeight: FontWeight.w600,
                              height: 1.4,
                              color: s.item.done ? Tokens.muted : Tokens.text,
                            ),
                          ),
                        ),
                      ),
                    ),
                  ),
                ),
            ],
          ),
        );
      },
    );
  }

  Widget _cell(BuildContext context, DateTime day, double height, double face) {
    final active = day == selected;
    final current = day == today;
    final count = items.where((i) => i.occursOn(day)).length;
    final names = items
        .where((i) => i.occursOn(day))
        .map((i) => i.title)
        .join(', ');
    return Semantics(
      button: true,
      selected: active,
      label:
          '${const ['JAN', 'FEB', 'MAR', 'APR', 'MAY', 'JUN', 'JUL', 'AUG', 'SEP', 'OCT', 'NOV', 'DEC'][day.month - 1]} ${day.day}, $count ${count == 1 ? 'item' : 'items'}${names.isEmpty ? '' : ', $names'}',
      excludeSemantics: true,
      child: InkWell(
        key: ValueKey(
          'day-${day.year}-${day.month.toString().padLeft(2, '0')}-${day.day.toString().padLeft(2, '0')}',
        ),
        onTap: () => onSelect(day),
        overlayColor: const WidgetStatePropertyAll(Colors.transparent),
        child: SizedBox(
          height: height,
          child: Align(
            alignment: Alignment.topCenter,
            child: Padding(
              padding: const EdgeInsets.only(top: 6),
              child: Container(
                key: active ? const ValueKey('calendar-selection') : null,
                width: 44,
                height: face,
                alignment: Alignment.center,
                decoration: BoxDecoration(
                  border: Border.all(
                    color: active ? Tokens.signal : Colors.transparent,
                    width: 2,
                  ),
                ),
                child: Text(
                  '${day.day}',
                  style: TextStyle(
                    fontSize: 16,
                    height: 1.2,
                    fontWeight: current ? FontWeight.w700 : FontWeight.w500,
                    color: day.month == month.month ? Tokens.text : Tokens.dim,
                    decoration: current ? TextDecoration.underline : null,
                    decorationColor: Tokens.text,
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class _Continuation extends CustomClipper<Path> {
  const _Continuation(this.before, this.after);
  final bool before, after;
  @override
  Path getClip(Size s) => Path()
    ..moveTo(before ? 4 : 0, 0)
    ..lineTo(after ? s.width - 4 : s.width, 0)
    ..lineTo(s.width, after ? s.height / 2 : 0)
    ..lineTo(after ? s.width - 4 : s.width, s.height)
    ..lineTo(before ? 4 : 0, s.height)
    ..lineTo(0, before ? s.height / 2 : s.height)
    ..close();
  @override
  bool shouldReclip(_Continuation old) =>
      before != old.before || after != old.after;
}

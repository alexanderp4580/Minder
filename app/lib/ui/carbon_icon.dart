// Carbon icons 11.92.0, IBM. Apache-2.0; licence: assets/carbon-LICENSE.txt.
// Geometry from https://cdn.jsdelivr.net/npm/@carbon/icons@11.92.0/svg/32/.
import 'package:flutter/material.dart';

enum CarbonGlyph {
  chat,
  calendar,
  board,
  left,
  right,
  down,
  delete,
  check,
  send,
}

class CarbonIcon extends StatelessWidget {
  const CarbonIcon(
    this.glyph, {
    super.key,
    this.size = 22,
    this.color,
    this.turns = 0,
  });
  final CarbonGlyph glyph;
  final double size;
  final Color? color;
  final int turns;
  @override
  Widget build(BuildContext context) => ExcludeSemantics(
    child: RotatedBox(
      quarterTurns: turns,
      child: SizedBox(
        width: size,
        height: size,
        child: CustomPaint(
          painter: _CarbonPainter(
            glyph,
            color ?? IconTheme.of(context).color ?? const Color(0xfff2f2f2),
          ),
        ),
      ),
    ),
  );
}

class _CarbonPainter extends CustomPainter {
  const _CarbonPainter(this.glyph, this.color);
  final CarbonGlyph glyph;
  final Color color;
  @override
  void paint(Canvas canvas, Size size) {
    canvas.scale(size.width / 32, size.height / 32);
    final paint = Paint()..color = color;
    for (final path in _paths[glyph]!) {
      canvas.drawPath(path, paint);
    }
  }

  @override
  bool shouldRepaint(_CarbonPainter old) =>
      glyph != old.glyph || color != old.color;
}

final _paths = <CarbonGlyph, List<Path>>{
  CarbonGlyph.chat: [
    Path()
      ..moveTo(17.74, 30.0)
      ..lineTo(16.0, 29.0)
      ..lineTo(20.0, 22.0)
      ..lineTo(26.0, 22.0)
      ..arcToPoint(
        const Offset(28.0, 20.0),
        radius: const Radius.elliptical(2.0, 2.0),
        rotation: 0.0,
        largeArc: false,
        clockwise: false,
      )
      ..lineTo(28.0, 8.0)
      ..arcToPoint(
        const Offset(26.0, 6.0),
        radius: const Radius.elliptical(2.0, 2.0),
        rotation: 0.0,
        largeArc: false,
        clockwise: false,
      )
      ..lineTo(6.0, 6.0)
      ..arcToPoint(
        const Offset(4.0, 8.0),
        radius: const Radius.elliptical(2.0, 2.0),
        rotation: 0.0,
        largeArc: false,
        clockwise: false,
      )
      ..lineTo(4.0, 20.0)
      ..arcToPoint(
        const Offset(6.0, 22.0),
        radius: const Radius.elliptical(2.0, 2.0),
        rotation: 0.0,
        largeArc: false,
        clockwise: false,
      )
      ..lineTo(15.0, 22.0)
      ..lineTo(15.0, 24.0)
      ..lineTo(6.0, 24.0)
      ..arcToPoint(
        const Offset(2.0, 20.0),
        radius: const Radius.elliptical(4.0, 4.0),
        rotation: 0.0,
        largeArc: false,
        clockwise: true,
      )
      ..lineTo(2.0, 8.0)
      ..arcToPoint(
        const Offset(6.0, 4.0),
        radius: const Radius.elliptical(4.0, 4.0),
        rotation: 0.0,
        largeArc: false,
        clockwise: true,
      )
      ..lineTo(26.0, 4.0)
      ..arcToPoint(
        const Offset(30.0, 8.0),
        radius: const Radius.elliptical(4.0, 4.0),
        rotation: 0.0,
        largeArc: false,
        clockwise: true,
      )
      ..lineTo(30.0, 20.0)
      ..arcToPoint(
        const Offset(26.0, 24.0),
        radius: const Radius.elliptical(4.0, 4.0),
        rotation: 0.0,
        largeArc: false,
        clockwise: true,
      )
      ..lineTo(21.16, 24.0)
      ..close(),
    Path()
      ..moveTo(8.0, 10.0)
      ..lineTo(24.0, 10.0)
      ..lineTo(24.0, 12.0)
      ..lineTo(8.0, 12.0)
      ..close(),
    Path()
      ..moveTo(8.0, 16.0)
      ..lineTo(18.0, 16.0)
      ..lineTo(18.0, 18.0)
      ..lineTo(8.0, 18.0)
      ..close(),
  ],
  // Three columns of different heights, drawn in the Carbon 32 px grid.
  CarbonGlyph.board: [
    for (final c in const [(3.0, 22.0), (12.0, 15.0), (21.0, 19.0)])
      Path()
        ..fillType = PathFillType.evenOdd
        ..addRect(Rect.fromLTWH(c.$1, 5, 8, c.$2))
        ..addRect(Rect.fromLTWH(c.$1 + 2, 7, 4, c.$2 - 4)),
  ],
  CarbonGlyph.calendar: [
    Path()
      ..moveTo(26.0, 4.0)
      ..lineTo(22.0, 4.0)
      ..lineTo(22.0, 2.0)
      ..lineTo(20.0, 2.0)
      ..lineTo(20.0, 4.0)
      ..lineTo(12.0, 4.0)
      ..lineTo(12.0, 2.0)
      ..lineTo(10.0, 2.0)
      ..lineTo(10.0, 4.0)
      ..lineTo(6.0, 4.0)
      ..cubicTo(4.9, 4.0, 4.0, 4.9, 4.0, 6.0)
      ..lineTo(4.0, 26.0)
      ..cubicTo(4.0, 27.1, 4.9, 28.0, 6.0, 28.0)
      ..lineTo(26.0, 28.0)
      ..cubicTo(27.1, 28.0, 28.0, 27.1, 28.0, 26.0)
      ..lineTo(28.0, 6.0)
      ..cubicTo(28.0, 4.9, 27.1, 4.0, 26.0, 4.0)
      ..close()
      ..moveTo(26.0, 26.0)
      ..lineTo(6.0, 26.0)
      ..lineTo(6.0, 12.0)
      ..lineTo(26.0, 12.0)
      ..lineTo(26.0, 26.0)
      ..close()
      ..moveTo(26.0, 10.0)
      ..lineTo(6.0, 10.0)
      ..lineTo(6.0, 6.0)
      ..lineTo(10.0, 6.0)
      ..lineTo(10.0, 8.0)
      ..lineTo(12.0, 8.0)
      ..lineTo(12.0, 6.0)
      ..lineTo(20.0, 6.0)
      ..lineTo(20.0, 8.0)
      ..lineTo(22.0, 8.0)
      ..lineTo(22.0, 6.0)
      ..lineTo(26.0, 6.0)
      ..lineTo(26.0, 10.0)
      ..close(),
  ],
  CarbonGlyph.left: [
    Path()
      ..moveTo(10.0, 16.0)
      ..lineTo(20.0, 6.0)
      ..lineTo(21.4, 7.4)
      ..lineTo(12.8, 16.0)
      ..lineTo(21.4, 24.6)
      ..lineTo(20.0, 26.0)
      ..close(),
  ],
  CarbonGlyph.right: [
    Path()
      ..moveTo(22.0, 16.0)
      ..lineTo(12.0, 26.0)
      ..lineTo(10.6, 24.6)
      ..lineTo(19.2, 16.0)
      ..lineTo(10.6, 7.4)
      ..lineTo(12.0, 6.0)
      ..close(),
  ],
  CarbonGlyph.down: [
    Path()
      ..moveTo(16.0, 22.0)
      ..lineTo(6.0, 12.0)
      ..lineTo(7.4, 10.6)
      ..lineTo(16.0, 19.2)
      ..lineTo(24.6, 10.6)
      ..lineTo(26.0, 12.0)
      ..close(),
  ],
  CarbonGlyph.delete: [
    Path()
      ..moveTo(12.0, 12.0)
      ..lineTo(14.0, 12.0)
      ..lineTo(14.0, 24.0)
      ..lineTo(12.0, 24.0)
      ..close(),
    Path()
      ..moveTo(18.0, 12.0)
      ..lineTo(20.0, 12.0)
      ..lineTo(20.0, 24.0)
      ..lineTo(18.0, 24.0)
      ..close(),
    Path()
      ..moveTo(4.0, 6.0)
      ..lineTo(4.0, 8.0)
      ..lineTo(6.0, 8.0)
      ..lineTo(6.0, 28.0)
      ..arcToPoint(
        const Offset(8.0, 30.0),
        radius: const Radius.elliptical(2.0, 2.0),
        rotation: 0.0,
        largeArc: false,
        clockwise: false,
      )
      ..lineTo(24.0, 30.0)
      ..arcToPoint(
        const Offset(26.0, 28.0),
        radius: const Radius.elliptical(2.0, 2.0),
        rotation: 0.0,
        largeArc: false,
        clockwise: false,
      )
      ..lineTo(26.0, 8.0)
      ..lineTo(28.0, 8.0)
      ..lineTo(28.0, 6.0)
      ..close()
      ..moveTo(8.0, 28.0)
      ..lineTo(8.0, 8.0)
      ..lineTo(24.0, 8.0)
      ..lineTo(24.0, 28.0)
      ..close(),
    Path()
      ..moveTo(12.0, 2.0)
      ..lineTo(20.0, 2.0)
      ..lineTo(20.0, 4.0)
      ..lineTo(12.0, 4.0)
      ..close(),
  ],
  CarbonGlyph.check: [
    Path()
      ..moveTo(13.0, 24.0)
      ..lineTo(4.0, 15.0)
      ..lineTo(5.414, 13.586)
      ..lineTo(13.0, 21.171)
      ..lineTo(26.586, 7.586)
      ..lineTo(28.0, 9.0)
      ..lineTo(13.0, 24.0)
      ..close(),
  ],
  CarbonGlyph.send: [
    Path()
      ..moveTo(27.71, 4.29)
      ..arcToPoint(
        const Offset(26.66, 4.06),
        radius: const Radius.elliptical(1.0, 1.0),
        rotation: 0.0,
        largeArc: false,
        clockwise: false,
      )
      ..lineTo(4.66, 12.059999999999999)
      ..arcToPoint(
        const Offset(4.66, 13.93),
        radius: const Radius.elliptical(1.0, 1.0),
        rotation: 0.0,
        largeArc: false,
        clockwise: false,
      )
      ..lineTo(14.26, 17.77)
      ..lineTo(18.1, 27.369999999999997)
      ..arcToPoint(
        const Offset(19.0, 28.0),
        radius: const Radius.elliptical(1.0, 1.0),
        rotation: 0.0,
        largeArc: false,
        clockwise: false,
      )
      ..lineTo(19.0, 28.0)
      ..arcToPoint(
        const Offset(19.92, 27.34),
        radius: const Radius.elliptical(1.0, 1.0),
        rotation: 0.0,
        largeArc: false,
        clockwise: false,
      )
      ..lineTo(27.92, 5.34)
      ..arcToPoint(
        const Offset(27.71, 4.29),
        radius: const Radius.elliptical(1.0, 1.0),
        rotation: 0.0,
        largeArc: false,
        clockwise: false,
      )
      ..close()
      ..moveTo(19.0, 24.2)
      ..lineTo(16.21, 17.2)
      ..lineTo(21.0, 12.41)
      ..lineTo(19.59, 11.0)
      ..lineTo(14.76, 15.83)
      ..lineTo(7.8, 13.0)
      ..lineTo(25.33, 6.67)
      ..close(),
  ],
};

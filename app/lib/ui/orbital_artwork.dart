import 'dart:math' as math;
import 'dart:ui' as ui;

import 'package:flutter/material.dart';

import 'orbital_artwork_data.dart';
import 'tokens.dart';

enum OrbitScene { chat, calendar }

/// Authored geometry and cached path metrics shared by the two painting layers.
class OrbitalArtwork {
  OrbitalArtwork(OrbitScene scene)
    : _source = scene == OrbitScene.calendar
          ? calendarArtwork()
          : chatArtwork();

  final OrbitalData _source;
  Size? _size;
  double _scale = 1;
  Offset _offset = Offset.zero;
  List<ui.PathMetric> _metrics = [];

  void _layout(Size size) {
    if (_size == size) return;
    _size = size;
    final width = math.min(size.width, 760.0);
    _scale = math.max(width / 462, size.height / 868);
    _offset = Offset((size.width - 462 * _scale) / 2, 0);
    final transform = Matrix4.diagonal3Values(_scale, _scale, 1)
      ..setTranslationRaw(_offset.dx, _offset.dy, 0);
    _metrics = [
      for (final path in _source.paths)
        ...path.transform(transform.storage).computeMetrics(),
    ];
  }

  Paint _edgePaint(OrbitalEdge edge, {double opacity = 1}) => Paint()
    ..style = PaintingStyle.stroke
    ..strokeWidth = .85
    ..shader = ui.Gradient.linear(
      edge.start,
      edge.end,
      [
        for (final stop in edge.stops)
          Tokens.text.withValues(alpha: stop.$2 * opacity),
      ],
      [for (final stop in edge.stops) stop.$1],
    );

  void paintStill(Canvas canvas, Size size) {
    _layout(size);
    canvas.drawColor(Tokens.voidColor, BlendMode.srcOver);
    canvas.save();
    canvas.translate(_offset.dx, _offset.dy);
    canvas.scale(_scale);
    for (final light in _source.lights) {
      canvas.save();
      canvas.translate(light.x, light.y);
      canvas.rotate(light.tilt);
      canvas.scale(light.rx, light.ry);
      canvas.drawCircle(
        Offset.zero,
        1,
        Paint()
          ..shader = ui.Gradient.radial(
            Offset.zero,
            1,
            [
              Tokens.text.withValues(alpha: light.alpha),
              Tokens.text.withValues(alpha: light.alpha * .22),
              Colors.transparent,
            ],
            [0, .4, 1],
          ),
      );
      canvas.restore();
    }
    for (var i = 0; i < _source.paths.length; i++) {
      final edge = _source.edges[i];
      canvas.drawPath(
        _source.paths[i],
        _edgePaint(edge, opacity: .16)
          ..strokeWidth = 2
          ..maskFilter = const MaskFilter.blur(BlurStyle.normal, 3.5),
      );
      canvas.drawPath(_source.paths[i], _edgePaint(edge));
    }
    canvas.restore();
  }

  void paintHighlights(Canvas canvas, Size size, double phase) {
    _layout(size);
    for (var i = 0; i < _metrics.length; i++) {
      final metric = _metrics[i];
      final start = ((phase + i * .17) % 1) * metric.length;
      final length = math.min(90 * _scale, metric.length * .12);
      _paintSegment(
        canvas,
        metric,
        start,
        math.min(start + length, metric.length),
      );
      if (start + length > metric.length) {
        _paintSegment(canvas, metric, 0, start + length - metric.length);
      }
    }
  }

  void _paintSegment(
    Canvas canvas,
    ui.PathMetric metric,
    double start,
    double end,
  ) {
    final from = metric.getTangentForOffset(start)?.position;
    final to = metric.getTangentForOffset(end)?.position;
    if (from == null || to == null || from == to) return;
    final segment = metric.extractPath(start, end);
    for (final layer in [(width: 3.0, alpha: .035), (width: 1.0, alpha: .32)]) {
      canvas.drawPath(
        segment,
        Paint()
          ..style = PaintingStyle.stroke
          ..strokeWidth = layer.width
          ..shader = ui.Gradient.linear(
            from,
            to,
            [
              Colors.transparent,
              Tokens.text.withValues(alpha: layer.alpha),
              Colors.transparent,
            ],
            [0, .55, 1],
          ),
      );
    }
  }
}

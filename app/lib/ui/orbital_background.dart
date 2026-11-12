import 'dart:math' as math;
import 'dart:typed_data';
import 'dart:ui' as ui;

import 'package:flutter/material.dart';

import 'tokens.dart';

enum OrbitScene { chat, calendar }

/// Paints decoration independently from the screen's layout and state.
class OrbitalBackground extends StatefulWidget {
  const OrbitalBackground({
    super.key,
    required this.scene,
    required this.child,
  });

  final OrbitScene scene;
  final Widget child;

  @override
  State<OrbitalBackground> createState() => _OrbitalBackgroundState();
}

class _OrbitalBackgroundState extends State<OrbitalBackground>
    with SingleTickerProviderStateMixin, WidgetsBindingObserver {
  late final AnimationController _motion;
  bool _visible = true;
  bool _reduceMotion = false;
  bool _foreground = true;

  @override
  void initState() {
    super.initState();
    _motion = AnimationController(
      vsync: this,
      duration: const Duration(seconds: 180),
    );
    _foreground =
        WidgetsBinding.instance.lifecycleState == null ||
        WidgetsBinding.instance.lifecycleState == AppLifecycleState.resumed;
    WidgetsBinding.instance.addObserver(this);
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _visible = TickerMode.valuesOf(context).enabled;
    _reduceMotion = MediaQuery.disableAnimationsOf(context);
    _syncMotion();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    _foreground = state == AppLifecycleState.resumed;
    _syncMotion();
  }

  void _syncMotion() {
    final animate = _visible && !_reduceMotion && _foreground;
    if (animate && !_motion.isAnimating) {
      _motion.repeat();
    } else if (!animate && _motion.isAnimating) {
      _motion.stop();
    }
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _motion.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Stack(
    fit: StackFit.expand,
    children: [
      Positioned.fill(
        child: IgnorePointer(
          child: ExcludeSemantics(
            child: ClipRect(
              child: Stack(
                fit: StackFit.expand,
                children: [
                  RepaintBoundary(
                    child: CustomPaint(painter: const _HaloPainter()),
                  ),
                  RepaintBoundary(
                    child: CustomPaint(
                      key: const ValueKey('orbital-arcs'),
                      painter: _OrbitPainter(
                        scene: widget.scene,
                        motion: _motion,
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
      RepaintBoundary(child: widget.child),
    ],
  );
}

class _HaloPainter extends CustomPainter {
  const _HaloPainter();

  @override
  void paint(Canvas canvas, Size size) {
    final rect = Offset.zero & size;
    final width = math.min(size.width, 760.0);
    final top = Offset(size.width / 2 + width * .48, size.height * .025);
    final bottom = Offset(size.width / 2 + width * .30, size.height * .98);
    for (final light in [
      (center: top, radius: width * .80, alpha: .26),
      (center: bottom, radius: width * .85, alpha: .16),
    ]) {
      canvas.drawRect(
        rect,
        Paint()
          ..shader = ui.Gradient.radial(
            light.center,
            light.radius,
            [
              Tokens.text.withValues(alpha: light.alpha),
              Tokens.text.withValues(alpha: light.alpha * .20),
              Colors.transparent,
            ],
            [0, .4, 1],
          ),
      );
    }
  }

  @override
  bool shouldRepaint(_HaloPainter oldDelegate) => false;
}

class _OrbitPainter extends CustomPainter {
  _OrbitPainter({required this.scene, required this.motion})
    : super(repaint: motion);
  final OrbitScene scene;
  final Animation<double> motion;

  @override
  void paint(Canvas canvas, Size size) {
    final calendar = scene == OrbitScene.calendar;
    final width = math.min(size.width, 760.0);
    final orbits = calendar
        ? const [
            (x: .80, y: .20, width: 1.9, height: .58, tilt: -.35),
            (x: .63, y: .43, width: 1.9, height: .84, tilt: -.12),
            (x: .69, y: .56, width: 2.0, height: .94, tilt: .11),
          ]
        : const [
            (x: .95, y: .07, width: 1.5, height: .38, tilt: -.45),
            (x: .80, y: .20, width: 1.8, height: .50, tilt: -.18),
            (x: .62, y: .37, width: 1.9, height: .66, tilt: .18),
          ];
    final paint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = .85
      ..shader = LinearGradient(
        begin: Alignment.topCenter,
        end: Alignment.bottomCenter,
        colors: [
          Tokens.text.withValues(alpha: .60),
          Tokens.text.withValues(alpha: .16),
          Tokens.text.withValues(alpha: .08),
          Tokens.text.withValues(alpha: .42),
        ],
        stops: const [0, .28, .60, 1],
      ).createShader(Offset.zero & size);
    for (final orbit in orbits) {
      final angle = orbit.tilt + motion.value * math.pi * 2;
      final cos = math.cos(angle);
      final sin = math.sin(angle);
      final centerX = size.width / 2 + width * (orbit.x - .5);
      final path = Path()
        ..addOval(
          Rect.fromCenter(
            center: Offset.zero,
            width: width * orbit.width,
            height: size.height * orbit.height,
          ),
        );
      canvas.drawPath(
        path.transform(
          Float64List.fromList([
            cos,
            sin,
            0,
            0,
            -sin,
            cos,
            0,
            0,
            0,
            0,
            1,
            0,
            centerX,
            size.height * orbit.y,
            0,
            1,
          ]),
        ),
        paint,
      );
    }
  }

  @override
  bool shouldRepaint(_OrbitPainter oldDelegate) =>
      oldDelegate.scene != scene || oldDelegate.motion != motion;
}

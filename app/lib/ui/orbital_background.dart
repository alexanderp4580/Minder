import 'dart:math' as math;

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
                    child: CustomPaint(
                      painter: _HaloPainter(scene: widget.scene),
                    ),
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
  const _HaloPainter({required this.scene});
  final OrbitScene scene;

  @override
  void paint(Canvas canvas, Size size) {
    final rect = Offset.zero & size;
    final halo = RadialGradient(
      center: scene == OrbitScene.calendar
          ? const Alignment(.85, -.9)
          : const Alignment(1, -.7),
      radius: .95,
      colors: [
        Tokens.text.withValues(alpha: .075),
        Tokens.text.withValues(alpha: .015),
        Colors.transparent,
      ],
      stops: const [0, .45, 1],
    );
    canvas.drawRect(rect, Paint()..shader = halo.createShader(rect));
  }

  @override
  bool shouldRepaint(_HaloPainter oldDelegate) => oldDelegate.scene != scene;
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
    final center = Offset(
      size.width / 2 + width * .35,
      size.height * (calendar ? .18 : .08),
    );
    final paint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = .7
      ..color = Tokens.text.withValues(alpha: .12);
    for (var i = 0; i < 3; i++) {
      canvas.save();
      canvas.translate(center.dx, center.dy);
      canvas.rotate(-.35 + i * .23 + motion.value * math.pi * 2);
      canvas.drawOval(
        Rect.fromCenter(
          center: Offset.zero,
          width: width * ((calendar ? 1.5 : .9) + i * .45),
          height: size.height * ((calendar ? .65 : .35) + i * .2),
        ),
        paint,
      );
      canvas.restore();
    }
  }

  @override
  bool shouldRepaint(_OrbitPainter oldDelegate) =>
      oldDelegate.scene != scene || oldDelegate.motion != motion;
}

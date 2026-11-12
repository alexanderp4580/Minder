import 'package:flutter/material.dart';

import 'orbital_artwork.dart';

export 'orbital_artwork.dart' show OrbitScene;

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
  late OrbitalArtwork _artwork;
  bool _visible = true;
  bool _reduceMotion = false;
  bool _foreground = true;

  @override
  void initState() {
    super.initState();
    _artwork = OrbitalArtwork(widget.scene);
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
  void didUpdateWidget(OrbitalBackground oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.scene != widget.scene) {
      _artwork = OrbitalArtwork(widget.scene);
    }
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
                    key: const ValueKey('orbital-artwork'),
                    child: CustomPaint(
                      key: const ValueKey('orbital-arcs'),
                      isComplex: true,
                      painter: _ArtworkPainter(_artwork),
                    ),
                  ),
                  RepaintBoundary(
                    child: CustomPaint(
                      painter: _HighlightPainter(
                        artwork: _artwork,
                        enabled: !_reduceMotion,
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

class _ArtworkPainter extends CustomPainter {
  const _ArtworkPainter(this.artwork);
  final OrbitalArtwork artwork;

  @override
  void paint(Canvas canvas, Size size) => artwork.paintStill(canvas, size);

  @override
  bool shouldRepaint(_ArtworkPainter oldDelegate) =>
      oldDelegate.artwork != artwork;
}

class _HighlightPainter extends CustomPainter {
  _HighlightPainter({
    required this.artwork,
    required this.motion,
    required this.enabled,
  }) : super(repaint: motion);
  final OrbitalArtwork artwork;
  final Animation<double> motion;
  final bool enabled;

  @override
  void paint(Canvas canvas, Size size) {
    if (enabled) artwork.paintHighlights(canvas, size, motion.value);
  }

  @override
  bool shouldRepaint(_HighlightPainter oldDelegate) =>
      oldDelegate.artwork != artwork ||
      oldDelegate.motion != motion ||
      oldDelegate.enabled != enabled;
}

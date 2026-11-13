import 'package:flutter/material.dart';

import 'tokens.dart';

/// A neutral content action with a square keyboard focus ring.
class ContentAction extends StatefulWidget {
  const ContentAction({
    super.key,
    required this.onPressed,
    required this.child,
  });
  final VoidCallback? onPressed;
  final Widget child;

  @override
  State<ContentAction> createState() => _ContentActionState();
}

class _ContentActionState extends State<ContentAction> {
  final _focus = FocusNode();

  @override
  void initState() {
    super.initState();
    _focus.addListener(_changed);
    FocusManager.instance.addHighlightModeListener(_modeChanged);
  }

  void _changed() {
    if (mounted) setState(() {});
  }

  void _modeChanged(FocusHighlightMode _) => _changed();

  @override
  void dispose() {
    FocusManager.instance.removeHighlightModeListener(_modeChanged);
    _focus.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => CustomPaint(
    foregroundPainter: _FocusRing(
      _focus.hasFocus &&
          FocusManager.instance.highlightMode == FocusHighlightMode.traditional,
    ),
    child: TextButton(
      focusNode: _focus,
      onPressed: widget.onPressed,
      child: widget.child,
    ),
  );
}

class _FocusRing extends CustomPainter {
  const _FocusRing(this.visible);
  final bool visible;

  @override
  void paint(Canvas canvas, Size size) {
    if (visible) {
      canvas.drawRect(
        (Offset.zero & size).inflate(3),
        Paint()
          ..color = Tokens.text
          ..style = PaintingStyle.stroke
          ..strokeWidth = 2,
      );
    }
  }

  @override
  bool shouldRepaint(_FocusRing old) => old.visible != visible;
}

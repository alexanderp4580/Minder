import 'package:flutter/material.dart';

import 'tokens.dart';

// Sizes and states follow docs/foundation.md and the brand guide:
// the chat mockup sets primary and secondary buttons to 44 px: Panel fill and Edge
// outline for secondary, a cut corner for primary. Every control is at least 44 px.

class PrimaryButton extends StatelessWidget {
  const PrimaryButton({
    super.key,
    required this.label,
    required this.onPressed,
    this.height = 44,
    this.cut = 12,
  });
  final String label;
  final VoidCallback? onPressed;
  final double height, cut;
  @override
  Widget build(BuildContext context) => SizedBox(
    height: height,
    child: FilledButton(
      onPressed: onPressed,
      style: FilledButton.styleFrom(
        minimumSize: Size(Tokens.minTarget, height),
        padding: const EdgeInsets.symmetric(horizontal: 18),
        shape: BeveledRectangleBorder(
          borderRadius: BorderRadius.only(bottomRight: Radius.circular(cut)),
        ),
        textStyle: const TextStyle(
          fontFamily: 'Chakra Petch',
          fontSize: 16,
          fontWeight: FontWeight.w700,
          letterSpacing: 0.64,
        ),
      ),
      child: Text(label),
    ),
  );
}

const _secondaryText = TextStyle(
  fontFamily: 'Chakra Petch',
  fontSize: 14,
  fontWeight: FontWeight.w700,
  letterSpacing: 0.56,
);

class SecondaryButton extends StatelessWidget {
  const SecondaryButton({
    super.key,
    required this.label,
    required this.onPressed,
  });
  final String label;
  final VoidCallback? onPressed;
  @override
  Widget build(BuildContext context) => SizedBox(
    height: Tokens.minTarget,
    child: OutlinedButton(
      onPressed: onPressed,
      style: OutlinedButton.styleFrom(
        backgroundColor: Tokens.panel,
        foregroundColor: Tokens.text,
        disabledForegroundColor: Tokens.dim,
        side: const BorderSide(color: Tokens.edge),
        minimumSize: const Size(Tokens.minTarget, Tokens.minTarget),
        padding: const EdgeInsets.symmetric(horizontal: 18),
        shape: const RoundedRectangleBorder(),
        textStyle: _secondaryText,
      ),
      child: Text(label),
    ),
  );
}

/// The face of a secondary button, for a menu that supplies the tap handling.
class SecondaryFace extends StatelessWidget {
  const SecondaryFace({super.key, required this.label, this.enabled = true});
  final String label;
  final bool enabled;
  @override
  Widget build(BuildContext context) => Container(
    height: Tokens.minTarget,
    alignment: Alignment.center,
    padding: const EdgeInsets.symmetric(horizontal: 18),
    decoration: BoxDecoration(
      color: Tokens.panel,
      border: Border.all(color: Tokens.edge),
    ),
    child: Text(
      label,
      style: _secondaryText.copyWith(
        color: enabled ? Tokens.text : Tokens.dim,
      ),
    ),
  );
}

class SquareIconButton extends StatelessWidget {
  const SquareIconButton({
    super.key,
    required this.icon,
    required this.tooltip,
    required this.onPressed,
  });
  final IconData icon;
  final String tooltip;
  final VoidCallback? onPressed;
  @override
  Widget build(BuildContext context) => SizedBox(
    width: Tokens.minTarget,
    height: Tokens.minTarget,
    child: DecoratedBox(
      decoration: BoxDecoration(
        color: Tokens.panel,
        border: Border.all(color: Tokens.edge),
      ),
      child: IconButton(
        tooltip: tooltip,
        onPressed: onPressed,
        padding: EdgeInsets.zero,
        icon: Icon(icon, size: 20),
      ),
    ),
  );
}

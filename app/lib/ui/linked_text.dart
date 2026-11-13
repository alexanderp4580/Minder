import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';

import '../domain/link_launcher.dart';
import '../domain/links.dart';
import 'tokens.dart';

/// Plain text with phone numbers, emails and web links made tappable. The visible text
/// is the link: nothing is hidden or interpreted, and a tap opens an app, never a call.
class LinkedText extends StatefulWidget {
  const LinkedText({super.key, required this.text, this.launcher, this.style});
  final String text;
  final LinkLauncher? launcher;
  final TextStyle? style;
  @override
  State<LinkedText> createState() => _LinkedTextState();
}

class _LinkedTextState extends State<LinkedText> {
  final _recognizers = <TapGestureRecognizer>[];
  late List<InlineSpan> _spans;

  @override
  void initState() {
    super.initState();
    _spans = _build();
  }

  @override
  void didUpdateWidget(LinkedText old) {
    super.didUpdateWidget(old);
    if (old.text != widget.text ||
        old.launcher != widget.launcher ||
        old.style != widget.style) {
      _clear();
      _spans = _build();
    }
  }

  void _clear() {
    for (final r in _recognizers) {
      r.dispose();
    }
    _recognizers.clear();
  }

  List<InlineSpan> _build() => [
    for (final span in findLinks(widget.text))
      if (span.kind == LinkKind.text || widget.launcher == null)
        TextSpan(text: span.text)
      else
        TextSpan(
          text: span.text,
          style: const TextStyle(decoration: TextDecoration.underline),
          recognizer: _recognizer(span.target!),
        ),
  ];

  TapGestureRecognizer _recognizer(Uri uri) {
    final recognizer = TapGestureRecognizer()
      ..onTap = () async {
        var opened = false;
        try {
          opened = await widget.launcher!.open(uri);
        } catch (_) {}
        if (!opened && mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Could not open this link.')),
          );
        }
      };
    _recognizers.add(recognizer);
    return recognizer;
  }

  @override
  void dispose() {
    _clear();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => Text.rich(
    TextSpan(children: _spans),
    style: widget.style ?? const TextStyle(color: Tokens.text),
  );
}

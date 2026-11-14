import 'carbon_icon.dart';

import 'package:flutter/material.dart';

import '../domain/failure.dart';
import 'tokens.dart';

const itemColours = <String, Color>{
  'blue': Color(0xff3b86f0),
  'teal': Color(0xff34d1bf),
  'green': Color(0xff9be38a),
  'amber': Color(0xffe9a020),
  'lilac': Color(0xffc3b0ff),
  'slate': Color(0xff6f879f),
};
Color itemColour(String value) => itemColours[value] ?? itemColours['slate']!;
String colourName(String value) => itemColours.containsKey(value)
    ? '${value[0].toUpperCase()}${value.substring(1)}'
    : 'Slate';

class ItemColourValue extends StatelessWidget {
  const ItemColourValue({super.key, required this.value});
  final String value;
  @override
  Widget build(BuildContext context) => Row(
    mainAxisSize: MainAxisSize.min,
    children: [
      SizedBox(
        width: 12,
        height: 12,
        child: ColoredBox(color: itemColour(value)),
      ),
      const SizedBox(width: 8),
      Flexible(child: Text(colourName(value))),
    ],
  );
}

/// The same named palette is used by details and pending chat previews.
class ItemColourPicker extends StatelessWidget {
  const ItemColourPicker({
    super.key,
    required this.selected,
    required this.onChanged,
  });
  final String selected;
  final ValueChanged<String>? onChanged;
  @override
  Widget build(BuildContext context) => LayoutBuilder(
    builder: (context, constraints) {
      final scale = MediaQuery.textScalerOf(context).scale(12) / 12;
      final columns = constraints.maxWidth / (105 * scale) >= 3 ? 3 : 2;
      final width = (constraints.maxWidth - 8 * (columns - 1)) / columns;
      return Wrap(
        spacing: 8,
        runSpacing: 8,
        children: [
          for (final c in itemColours.keys)
            SizedBox(
              width: width,
              child: Semantics(
                selected: c == selected,
                child: OutlinedButton(
                  key: ValueKey('colour-$c'),
                  onPressed: onChanged == null ? null : () => onChanged!(c),
                  style: OutlinedButton.styleFrom(
                    padding: const EdgeInsets.symmetric(horizontal: 8),
                    side: BorderSide(
                      color: c == selected ? Tokens.text : Tokens.edge,
                      width: c == selected ? 2 : 1,
                    ),
                  ),
                  child: Row(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      SizedBox(
                        width: 12,
                        height: 12,
                        child: ColoredBox(color: itemColour(c)),
                      ),
                      const SizedBox(width: 6),
                      Flexible(
                        child: Text(
                          colourName(c),
                          style: const TextStyle(fontSize: 12),
                        ),
                      ),
                      if (c == selected) ...[
                        const SizedBox(width: 4),
                        const CarbonIcon(CarbonGlyph.check, size: 14),
                      ],
                    ],
                  ),
                ),
              ),
            ),
        ],
      );
    },
  );
}

class ItemColourEditor extends StatefulWidget {
  const ItemColourEditor({
    super.key,
    required this.value,
    required this.onApply,
  });
  final String value;
  final Future<void> Function(String) onApply;
  @override
  State<ItemColourEditor> createState() => _ItemColourEditorState();
}

class _ItemColourEditorState extends State<ItemColourEditor> {
  bool _open = false, _busy = false;
  String? _error;
  late String _selected;
  @override
  void initState() {
    super.initState();
    _selected = widget.value;
  }

  Future<void> _apply() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await widget.onApply(_selected);
      if (mounted) setState(() => _open = false);
    } catch (e) {
      if (mounted) {
        setState(
          () => _error = e is RepositoryFailure
              ? e.message
              : 'Could not change the colour. Try again.',
        );
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) => Column(
    crossAxisAlignment: CrossAxisAlignment.stretch,
    children: [
      const SizedBox(height: 16),
      const Divider(height: 1),
      const SizedBox(height: 16),
      const Text('Colour', style: TextStyle(color: Tokens.muted, fontSize: 12)),
      TextButton(
        onPressed: _busy
            ? null
            : () => setState(() {
                _open = !_open;
                _selected = widget.value;
                _error = null;
              }),
        child: Row(
          children: [
            Expanded(child: ItemColourValue(value: widget.value)),
            Icon(_open ? Icons.expand_less : Icons.expand_more, size: 18),
          ],
        ),
      ),
      if (_open) ...[
        const SizedBox(height: 16),
        ItemColourPicker(
          selected: _selected,
          onChanged: _busy ? null : (v) => setState(() => _selected = v),
        ),
        const SizedBox(height: 16),
        Row(
          children: [
            Expanded(
              child: OutlinedButton(
                onPressed: _busy ? null : () => setState(() => _open = false),
                child: const Text('Cancel'),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: OutlinedButton(
                onPressed: _busy ? null : _apply,
                child: Text(_busy ? 'Applying…' : 'Apply colour'),
              ),
            ),
          ],
        ),
        if (_error != null)
          Padding(
            padding: const EdgeInsets.only(top: 8),
            child: Text(_error!, style: const TextStyle(color: Tokens.alert)),
          ),
      ],
    ],
  );
}

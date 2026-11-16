import 'package:flutter/material.dart';

import 'tokens.dart';

/// The text field of a composer: the chat message field and the Board's Add a task field.
/// Both share the text colour, cursor, hint style and the Send key, so typing feels the same.
class ComposerField extends StatelessWidget {
  const ComposerField({
    super.key,
    required this.controller,
    required this.hint,
    required this.onSubmitted,
    this.minLines = 1,
    this.maxLines = 4,
    this.maxLength,
    this.readOnly = false,
    this.keepFocusOnSubmit = false,
  });
  final TextEditingController controller;
  final String hint;
  final VoidCallback onSubmitted;
  final int minLines, maxLines;

  /// Limits the text without showing a counter.
  final int? maxLength;
  final bool readOnly;

  /// Keeps the keyboard open after the Send key, for adding several entries in a row.
  final bool keepFocusOnSubmit;

  @override
  Widget build(BuildContext context) => TextField(
    controller: controller,
    readOnly: readOnly,
    style: const TextStyle(color: Tokens.text),
    cursorColor: Tokens.text,
    minLines: minLines,
    maxLines: maxLines,
    maxLength: maxLength,
    buildCounter: maxLength == null
        ? null
        : (_, {required currentLength, required isFocused, maxLength}) => null,
    textInputAction: TextInputAction.send,
    onEditingComplete: keepFocusOnSubmit ? () {} : null,
    onSubmitted: (_) => onSubmitted(),
    decoration: InputDecoration(hintText: hint),
  );
}

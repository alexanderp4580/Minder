import 'package:flutter/material.dart';

const _body = TextStyle(
  fontSize: 15,
  height: 1.4,
  color: Tokens.text,
  fontWeight: FontWeight.w500,
);

abstract final class Tokens {
  static const voidColor = Color(0xff0a0a0a),
      panel = Color(0xff161616),
      line = Color(0xff3a3a3a),
      edge = Color(0xff6b6b6b),
      text = Color(0xfff2f2f2),
      muted = Color(0xffb5b5b5),
      dim = Color(0xff8f8f8f),
      signal = Color(0xffe0121a),
      alert = Color(0xffff4448),
      white = Color(0xffffffff);
  static const gutter = 20.0, gap = 8.0, minTarget = 44.0;
  static const pageWidth = 760.0;
  static const relatedGap = 16.0, sectionGap = 24.0, blockGap = 32.0;
  static ThemeData get theme => ThemeData(
    brightness: Brightness.dark,
    // Desktop browsers default to compact density, which shrinks fields below 44 px.
    visualDensity: VisualDensity.standard,
    scaffoldBackgroundColor: voidColor,
    fontFamily: 'Chakra Petch',
    colorScheme: const ColorScheme.dark(
      primary: signal,
      surface: panel,
      onSurface: text,
      onPrimary: white,
      error: alert,
    ),
    textTheme: const TextTheme(bodyMedium: _body, bodyLarge: _body),
    dividerColor: line,
    inputDecorationTheme: const InputDecorationTheme(
      isDense: true,
      filled: true,
      fillColor: panel,
      hintStyle: TextStyle(color: dim),
      contentPadding: EdgeInsets.symmetric(horizontal: 14, vertical: 11.5),
      constraints: BoxConstraints(minHeight: minTarget),
      border: OutlineInputBorder(
        borderRadius: BorderRadius.zero,
        borderSide: BorderSide(color: edge),
      ),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.zero,
        borderSide: BorderSide(color: edge),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.zero,
        borderSide: BorderSide(color: text),
      ),
    ),
    // Primary: Signal fill, cut bottom-right corner. Secondary: Panel fill, Edge outline.
    // Both are 44 px high. The primary button goes last in its row.
    textButtonTheme: TextButtonThemeData(
      style:
          TextButton.styleFrom(
            foregroundColor: text,
            disabledForegroundColor: dim,
            backgroundColor: Colors.transparent,
            minimumSize: const Size(minTarget, minTarget),
            padding: EdgeInsets.zero,
            alignment: Alignment.centerLeft,
            tapTargetSize: MaterialTapTargetSize.shrinkWrap,
            shape: const RoundedRectangleBorder(),
            textStyle: const TextStyle(
              fontFamily: 'Chakra Petch',
              fontSize: 14,
              fontWeight: FontWeight.w600,
            ),
          ).copyWith(
            overlayColor: WidgetStateProperty.resolveWith((states) {
              if (states.contains(WidgetState.pressed)) {
                return text.withValues(alpha: .10);
              }
              if (states.contains(WidgetState.hovered)) {
                return text.withValues(alpha: .06);
              }
              return Colors.transparent;
            }),
          ),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: signal,
        foregroundColor: white,
        minimumSize: const Size(minTarget, minTarget),
        padding: const EdgeInsets.symmetric(horizontal: 18),
        tapTargetSize: MaterialTapTargetSize.shrinkWrap,
        shape: const BeveledRectangleBorder(
          borderRadius: BorderRadius.only(bottomRight: Radius.circular(12)),
        ),
        textStyle: const TextStyle(
          fontFamily: 'Chakra Petch',
          fontSize: 16,
          fontWeight: FontWeight.w700,
          letterSpacing: 0.64,
        ),
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        backgroundColor: panel,
        foregroundColor: text,
        disabledForegroundColor: dim,
        side: const BorderSide(color: edge),
        minimumSize: const Size(minTarget, minTarget),
        padding: const EdgeInsets.symmetric(horizontal: 18),
        tapTargetSize: MaterialTapTargetSize.shrinkWrap,
        shape: const RoundedRectangleBorder(),
        textStyle: const TextStyle(
          fontFamily: 'Chakra Petch',
          fontSize: 14,
          fontWeight: FontWeight.w700,
          letterSpacing: 0.56,
        ),
      ),
    ),
    iconButtonTheme: IconButtonThemeData(
      style: IconButton.styleFrom(
        backgroundColor: panel,
        foregroundColor: text,
        side: const BorderSide(color: edge),
        fixedSize: const Size(minTarget, minTarget),
        padding: EdgeInsets.zero,
        iconSize: 20,
        tapTargetSize: MaterialTapTargetSize.shrinkWrap,
        shape: const RoundedRectangleBorder(),
      ),
    ),
    progressIndicatorTheme: const ProgressIndicatorThemeData(color: muted),
  );
}

abstract final class CalendarTokens {
  static const gutter = Tokens.sectionGap,
      maxWidth = 520.0,
      dayHeight = 56.0,
      selectionSize = Tokens.minTarget,
      cardPadding = 20.0;
}

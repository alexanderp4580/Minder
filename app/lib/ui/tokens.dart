import 'package:flutter/material.dart';

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
  static ThemeData get theme => ThemeData(
    brightness: Brightness.dark,
    scaffoldBackgroundColor: voidColor,
    fontFamily: 'Chakra Petch',
    colorScheme: const ColorScheme.dark(
      primary: signal,
      surface: panel,
      onSurface: text,
      onPrimary: white,
      error: alert,
    ),
    textTheme: const TextTheme(
      bodyMedium: TextStyle(
        fontSize: 15,
        height: 1.4,
        color: text,
        fontWeight: FontWeight.w500,
      ),
    ),
    dividerColor: line,
    inputDecorationTheme: const InputDecorationTheme(
      filled: true,
      fillColor: panel,
      hintStyle: TextStyle(color: dim),
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
    textButtonTheme: TextButtonThemeData(
      style: TextButton.styleFrom(
        foregroundColor: text,
        minimumSize: const Size(44, 44),
        shape: const RoundedRectangleBorder(),
        textStyle: const TextStyle(
          fontFamily: 'Chakra Petch',
          fontSize: 14,
          fontWeight: FontWeight.w700,
        ),
      ),
    ),
    filledButtonTheme: FilledButtonThemeData(
      style: FilledButton.styleFrom(
        backgroundColor: signal,
        foregroundColor: white,
        minimumSize: const Size(44, 44),
        shape: const BeveledRectangleBorder(
          borderRadius: BorderRadius.only(bottomRight: Radius.circular(12)),
        ),
        textStyle: const TextStyle(
          fontFamily: 'Chakra Petch',
          fontSize: 16,
          fontWeight: FontWeight.w700,
        ),
      ),
    ),
    progressIndicatorTheme: const ProgressIndicatorThemeData(color: muted),
  );
}

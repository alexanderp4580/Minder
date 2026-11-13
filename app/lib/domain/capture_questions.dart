/// English fallbacks for older duration questions without usable suggestions.
bool isDurationQuestion(String text) =>
    RegExp(r'how long|duration|hoe lang', caseSensitive: false).hasMatch(text);

String questionText(String text) =>
    isDurationQuestion(text) &&
        RegExp(r'hoe lang', caseSensitive: false).hasMatch(text)
    ? 'How long will it take?'
    : text;

List<String> questionOptions(String text, List<String> options) {
  if (!isDurationQuestion(text)) return options;
  if (options.isEmpty) return const ['30 minutes', '45 minutes', '1 hour'];
  final english = options.map((option) {
    return option.replaceAllMapped(
      RegExp(
        r'(\d+)\s*(minuten|min|minutes|uur|hours?)\b',
        caseSensitive: false,
      ),
      (match) {
        final number = int.parse(match[1]!);
        final hours = RegExp(
          r'uur|hour',
          caseSensitive: false,
        ).hasMatch(match[2]!);
        final unit = hours ? 'hour' : 'minute';
        return '$number $unit${number == 1 ? '' : 's'}';
      },
    );
  }).toList();
  final hasMinuteOrHour = english.any(
    (option) => RegExp(
      r'\b\d+\s*(?:minutes?|hours?)\b',
      caseSensitive: false,
    ).hasMatch(option),
  );
  if (!hasMinuteOrHour) return english;
  if (english.any(
    (o) =>
        RegExp(r'\b30\s*(?:min|minutes)\b', caseSensitive: false).hasMatch(o),
  )) {
    return english;
  }
  return ['30 minutes', ...english.take(2)];
}

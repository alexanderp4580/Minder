import 'package:flutter/material.dart';

import '../application/chat_controller.dart';
import '../domain/capture.dart';
import '../domain/capture_questions.dart';
import 'chat_markdown.dart';
import 'tokens.dart';

/// Review questions share the chat composer; choices are contextual shortcuts.
class QuestionView extends StatelessWidget {
  const QuestionView({super.key, required this.capture, required this.chat});
  final Capture capture;
  final ChatController chat;

  Widget _ask(String text, List<String> options, String? candidateId) =>
      QuestionPrompt(
        text: text,
        options: options,
        showChoices: !chat.questionAnswered(capture, candidateId: candidateId),
        onAnswer: chat.waiting || chat.acting
            ? null
            : (answer) =>
                  chat.answerReview(capture, answer, candidateId: candidateId),
      );

  @override
  Widget build(BuildContext context) => AnimatedBuilder(
    animation: chat,
    builder: (context, _) => Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        for (final candidate in capture.candidates)
          if (candidate.question.isNotEmpty &&
              (candidate.state == 'needs-answer' ||
                  candidate.state == 'pending'))
            _ask(candidate.question, candidate.options, candidate.id),
        for (final question in capture.questions)
          _ask(question.text, question.options, null),
      ],
    ),
  );
}

/// Shared question and suggested replies in the email details and question message.
class QuestionPrompt extends StatelessWidget {
  const QuestionPrompt({
    super.key,
    required this.text,
    required this.options,
    this.onAnswer,
    this.showChoices = true,
  });
  final String text;
  final List<String> options;
  final ValueChanged<String>? onAnswer;
  final bool showChoices;

  @override
  Widget build(BuildContext context) {
    final choices = questionOptions(text, options);
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          ChatMarkdown(text: questionText(text)),
          if (showChoices && choices.isNotEmpty) ...[
            const SizedBox(height: 8),
            Wrap(
              spacing: Tokens.gap,
              runSpacing: Tokens.gap,
              children: [
                for (final choice in choices)
                  OutlinedButton(
                    onPressed: onAnswer == null
                        ? null
                        : () => onAnswer!(choice),
                    child: Text(choice),
                  ),
              ],
            ),
            const SizedBox(height: 8),
            const Text(
              'Or give a custom response below.',
              style: TextStyle(color: Tokens.muted),
            ),
          ],
        ],
      ),
    );
  }
}

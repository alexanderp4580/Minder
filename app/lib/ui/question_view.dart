import 'package:flutter/material.dart';

import '../application/capture_controller.dart';
import '../application/chat_controller.dart';
import '../domain/capture.dart';
import '../domain/capture_repository.dart';
import 'tokens.dart';

/// The assistant's open questions about an email, each with quick replies.
/// One typed reply can answer several questions at once.
class QuestionView extends StatefulWidget {
  const QuestionView({super.key, required this.capture, required this.chat});
  final Capture capture;
  final ChatController chat;
  @override
  State<QuestionView> createState() => _QuestionViewState();
}

class _QuestionViewState extends State<QuestionView> {
  final _reply = TextEditingController();
  CaptureController? _controller;

  @override
  void initState() {
    super.initState();
    final repository = widget.chat.repository;
    if (repository is CaptureRepository) {
      _controller = CaptureController(
        repository as CaptureRepository,
        widget.chat.receiveCaptureEntry,
      );
    }
  }

  @override
  void dispose() {
    _controller?.dispose();
    _reply.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final controller = _controller;
    if (controller == null) return const Text('Questions about the email.');
    return AnimatedBuilder(
      animation: controller,
      builder: (context, _) => _content(controller),
    );
  }

  Widget _ask(
    CaptureController controller,
    String text,
    List<String> options,
    String? candidateId,
  ) => Padding(
    padding: const EdgeInsets.only(bottom: 12),
    child: Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(text),
        if (options.isNotEmpty) ...[
          const SizedBox(height: 8),
          Wrap(
            spacing: Tokens.gap,
            runSpacing: Tokens.gap,
            children: [
              for (final option in options)
                OutlinedButton(
                  onPressed: controller.busy
                      ? null
                      : () => controller.answer(
                          widget.capture,
                          option,
                          candidateId: candidateId,
                        ),
                  child: Text(option),
                ),
            ],
          ),
        ],
      ],
    ),
  );

  Widget _content(CaptureController controller) {
    final capture = widget.capture;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        for (final candidate in capture.candidates)
          if (candidate.question.isNotEmpty &&
              (candidate.state == 'needs-answer' ||
                  candidate.state == 'pending'))
            _ask(
              controller,
              candidate.question,
              candidate.options,
              candidate.id,
            ),
        for (final question in capture.questions)
          _ask(controller, question.text, question.options, null),
        TextField(
          controller: _reply,
          minLines: 1,
          maxLines: 4,
          decoration: const InputDecoration(labelText: 'Reply'),
        ),
        const SizedBox(height: 8),
        FilledButton(
          onPressed: controller.busy
              ? null
              : () async {
                  if (_reply.text.trim().isEmpty) return;
                  await controller.answer(capture, _reply.text);
                  if (mounted && controller.error == null) _reply.clear();
                },
          child: const Text('Send answer'),
        ),
        if (controller.error != null)
          Text(controller.error!, style: const TextStyle(color: Tokens.alert)),
      ],
    );
  }
}

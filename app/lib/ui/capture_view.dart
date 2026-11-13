import 'package:flutter/material.dart';

import '../application/capture_controller.dart';
import '../application/chat_controller.dart';
import '../domain/capture.dart';
import '../domain/item.dart';
import '../domain/link_launcher.dart';
import 'calendar_item_details.dart';
import 'content_action.dart';
import 'question_view.dart';
import '../domain/capture_repository.dart';
import 'format.dart';
import 'tokens.dart';

class CaptureView extends StatefulWidget {
  const CaptureView({
    super.key,
    required this.capture,
    required this.chat,
    this.linkLauncher,
  });
  final Capture capture;
  final ChatController chat;
  final LinkLauncher? linkLauncher;
  @override
  State<CaptureView> createState() => _CaptureViewState();
}

class _CaptureViewState extends State<CaptureView> {
  final _expanded = <String>{};
  final _answer = TextEditingController();
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
    _answer.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final controller = _controller;
    if (controller == null) {
      return Text(
        widget.capture.deleted ? 'Email source deleted.' : 'Email review',
      );
    }
    return AnimatedBuilder(
      animation: controller,
      builder: (context, _) => _content(controller),
    );
  }

  Widget _candidate(
    CaptureController controller,
    Capture capture,
    CaptureCandidate candidate,
  ) {
    final expanded = _expanded.contains(candidate.id);
    final details =
        candidate.details ??
        Item(
          id: 'proposal:${candidate.id}',
          title: candidate.title,
          status: candidate.state,
          location: candidate.location,
        );
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Semantics(
          expanded: expanded,
          child: ContentAction(
            onPressed: () => setState(() {
              if (expanded) {
                _expanded.remove(candidate.id);
              } else {
                _expanded.add(candidate.id);
              }
            }),
            child: Padding(
              padding: const EdgeInsets.symmetric(vertical: 12),
              child: Row(
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          candidate.title.isEmpty
                              ? 'Proposed item'
                              : candidate.title,
                          style: const TextStyle(
                            fontSize: 15,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                        Text(
                          [
                            candidateWhen(candidate),
                            if (!expanded) candidate.location,
                            if (candidate.state == 'confirmed') 'Confirmed',
                            if (candidate.state == 'discarded') 'Discarded',
                          ].where((part) => part.isNotEmpty).join(' · '),
                          style: const TextStyle(
                            fontSize: 13,
                            color: Tokens.muted,
                            fontWeight: FontWeight.w500,
                          ),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(width: 12),
                  Icon(
                    expanded ? Icons.expand_less : Icons.expand_more,
                    size: 16,
                  ),
                ],
              ),
            ),
          ),
        ),
        if (expanded) ...[
          CalendarItemDetails(
            item: details,
            launcher: widget.linkLauncher,
            showTiming: false,
          ),
          if (candidate.question.isNotEmpty &&
              (candidate.state == 'needs-answer' ||
                  candidate.state == 'pending'))
            Padding(
              padding: const EdgeInsets.only(top: 16),
              child: QuestionPrompt(
                text: candidate.question,
                options: candidate.options,
                onAnswer: controller.busy
                    ? null
                    : (answer) => controller.answer(
                        capture,
                        answer,
                        candidateId: candidate.id,
                      ),
              ),
            ),
          if (candidate.question.isEmpty && capture.candidates.length == 1)
            for (final question in capture.questions)
              Padding(
                padding: const EdgeInsets.only(top: 16),
                child: QuestionPrompt(
                  text: question.text,
                  options: question.options,
                  onAnswer: controller.busy
                      ? null
                      : (answer) => controller.answer(capture, answer),
                ),
              ),
          const SizedBox(height: 16),
        ],
      ],
    );
  }

  Widget _content(CaptureController controller) {
    final capture = widget.capture;
    if (capture.deleted) {
      return const Text('Email source deleted. Confirmed items remain.');
    }
    final noItems =
        !capture.processing &&
        capture.candidates.isEmpty &&
        capture.status == 'ready-for-review';
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: Tokens.panel,
        border: Border.all(color: Tokens.edge),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            'EMAIL',
            style: TextStyle(
              fontSize: 12,
              color: Tokens.muted,
              letterSpacing: 1,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            'Email review',
            style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
          ),
          if (capture.sender.isNotEmpty)
            Text(capture.sender, style: const TextStyle(color: Tokens.muted)),
          const SizedBox(height: 8),
          if (!noItems)
            Text(switch (capture.status) {
              'received' => 'Received. Waiting to process.',
              'processing' => 'Reading email and attachments…',
              'needs-answer' =>
                capture.candidates.isEmpty
                    ? 'Some details need your answer.'
                    : 'Found in this email. I have a few questions.',
              'failed' =>
                capture.error.isEmpty
                    ? 'Processing failed. The original is available.'
                    : capture.error,
              'reviewed' => 'Review complete.',
              _ => 'Found in this email:',
            }),
          if (noItems)
            const Padding(
              padding: EdgeInsets.only(top: 8),
              child: Text(
                'No actionable items found. Tell Minder what to capture.',
              ),
            ),
          for (final candidate in capture.candidates)
            _candidate(controller, capture, candidate),
          for (final warning in capture.coverage)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text(warning, style: const TextStyle(color: Tokens.muted)),
            ),
          if (!capture.processing && capture.status != 'reviewed') ...[
            if (capture.candidates.isEmpty) ...[
              const SizedBox(height: 12),
              TextField(
                controller: _answer,
                minLines: 1,
                maxLines: 4,
                decoration: const InputDecoration(
                  labelText: 'Answer or clarify this email',
                ),
              ),
              const SizedBox(height: 8),
              FilledButton(
                onPressed: controller.busy
                    ? null
                    : () async {
                        if (_answer.text.trim().isEmpty) return;
                        await controller.answer(capture, _answer.text);
                        if (mounted && controller.error == null) {
                          _answer.clear();
                        }
                      },
                child: const Text('Send answer'),
              ),
            ],
            if (capture.status != 'failed')
              OutlinedButton(
                onPressed: controller.busy
                    ? null
                    : () => controller.dismiss(capture),
                child: const Text('Do nothing'),
              ),
          ],
          if (capture.needsCoverageReview && !capture.processing)
            OutlinedButton(
              onPressed: controller.busy
                  ? null
                  : () => controller.answer(
                      capture,
                      'Continue with readable content.',
                      acknowledgeCoverage: true,
                    ),
              child: const Text('Continue with readable content'),
            ),
          if (capture.status == 'failed')
            OutlinedButton(
              onPressed: controller.busy
                  ? null
                  : () => controller.retry(capture),
              child: const Text('Retry email'),
            ),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [
              for (final attachment in capture.attachments)
                OutlinedButton(
                  onPressed: controller.busy
                      ? null
                      : () => controller.download(capture, attachment),
                  child: Text(attachment.name),
                ),
            ],
          ),
          TextButton(
            onPressed: controller.busy
                ? null
                : () => controller.delete(capture),
            child: const Text('Delete email source…'),
          ),
          if (controller.error != null)
            Text(
              controller.error!,
              style: const TextStyle(color: Tokens.alert),
            ),
        ],
      ),
    );
  }
}

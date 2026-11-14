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
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final controller = _controller;
    if (controller == null) {
      return Text(
        widget.capture.deleted
            ? '${widget.capture.pasted ? 'Source' : 'Email source'} deleted.'
            : widget.capture.pasted
            ? 'Review of pasted text'
            : 'Email review',
      );
    }
    return AnimatedBuilder(
      animation: Listenable.merge([controller, widget.chat]),
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
                            if (candidate.state == 'queued') 'Waiting',
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
                showChoices: !widget.chat.questionAnswered(
                  capture,
                  candidateId: candidate.id,
                ),
                onAnswer: controller.busy || widget.chat.waiting
                    ? null
                    : (answer) => widget.chat.answerReview(
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
                  showChoices: !widget.chat.questionAnswered(capture),
                  onAnswer: controller.busy || widget.chat.waiting
                      ? null
                      : (answer) => widget.chat.answerReview(capture, answer),
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
      return Text(
        '${capture.pasted ? 'Source' : 'Email source'} deleted. Confirmed items remain.',
      );
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
          Text(
            capture.pasted ? 'PASTED TEXT' : 'EMAIL',
            style: const TextStyle(
              fontSize: 12,
              color: Tokens.muted,
              letterSpacing: 1,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            capture.pasted ? 'Review of pasted text' : 'Email review',
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
                    : 'Found in this ${capture.pasted ? 'text' : 'email'}. I have a few questions.',
              'failed' =>
                capture.error.isEmpty
                    ? 'Processing failed. The original is available.'
                    : capture.error,
              'reviewed' => 'Review complete.',
              _ => 'Found in this ${capture.pasted ? 'text' : 'email'}:',
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
          if (capture.unverified > 0)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text(
                'I could not verify ${capture.unverified} '
                '${capture.unverified == 1 ? 'appointment' : 'appointments'} '
                'from this ${capture.pasted ? 'text' : 'email'}.',
                style: const TextStyle(color: Tokens.muted),
              ),
            ),
          for (final warning in capture.coverage)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text(warning, style: const TextStyle(color: Tokens.muted)),
            ),
          if (!capture.processing && capture.status != 'reviewed') ...[
            if (capture.candidates.isEmpty) ...[
              const SizedBox(height: 12),
              const Text(
                'Give a custom response below.',
                style: TextStyle(color: Tokens.muted),
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
            child: Text(
              capture.pasted ? 'Delete source…' : 'Delete email source…',
            ),
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
